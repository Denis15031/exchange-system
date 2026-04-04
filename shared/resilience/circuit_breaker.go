package resilience

import (
	"fmt"
	"time"

	"github.com/sony/gobreaker"
)

type Counts = gobreaker.Counts
type State = gobreaker.State
type CircuitBreaker = gobreaker.CircuitBreaker

// Тип колбэка при смене состояния
type OnStateChangeFunc func(name string, from State, to State)

type CircuitBreakerConfig struct {
	Name          string
	Timeout       time.Duration
	MaxRequests   uint32
	Interval      time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange OnStateChangeFunc
}

// Настройки по умолчанию для биржевых сервисов
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:        name,
		Timeout:     30 * time.Second,
		MaxRequests: 5,
		Interval:    30 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			// Размыкаем при >60% ошибок (минимум 5 запросов)
			if counts.Requests < 5 {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= 0.6
		},
		OnStateChange: nil,
	}
}

func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:          config.Name,
		Timeout:       config.Timeout,
		MaxRequests:   config.MaxRequests,
		Interval:      config.Interval,
		ReadyToTrip:   config.ReadyToTrip,
		OnStateChange: config.OnStateChange,
	}
	return gobreaker.NewCircuitBreaker(settings)
}

// Выполняет функцию через Circuit Breaker с типизированным результатом
func Execute[T any](cb *CircuitBreaker, fn func() (T, error)) (T, error) {
	result, err := cb.Execute(func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		var zero T
		return zero, err
	}
	if result == nil {
		var zero T
		return zero, nil
	}
	if typed, ok := result.(T); ok {
		return typed, nil
	}
	var zero T
	return zero, fmt.Errorf("unexpected result type from circuit breaker")
}

// Возвращает текущее состояние для мониторинга
func GetState(cb *CircuitBreaker) State {
	return cb.State()
}

// Возвращает статистику для метрик
func GetCounts(cb *CircuitBreaker) Counts {
	return cb.Counts()
}

// Проверяет, разомкнута ли цепь (для быстрого отказа)
func IsOpen(cb *CircuitBreaker) bool {
	return cb.State() == gobreaker.StateOpen
}

// Создаёт ошибку с сообщением (для совместимости)
func StringError(msg string) error {
	return fmt.Errorf("%s", msg)
}
