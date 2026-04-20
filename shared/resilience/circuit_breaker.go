package resilience

import (
	"context"
	"fmt"
	"time"

	"exchange-system/shared/ports"

	"github.com/sony/gobreaker"
)

type Counts = gobreaker.Counts
type State = gobreaker.State

type OnStateChangeFunc func(name string, from State, to State)

type CircuitBreakerConfig struct {
	Name          string
	Timeout       time.Duration
	MaxRequests   uint32
	Interval      time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange OnStateChangeFunc
}

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

type circuitBreaker struct {
	cb *gobreaker.CircuitBreaker
}

var _ ports.CircuitBreaker = (*circuitBreaker)(nil)

func NewCircuitBreaker(config CircuitBreakerConfig) ports.CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        config.Name,
		Timeout:     config.Timeout,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		ReadyToTrip: config.ReadyToTrip,
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			if config.OnStateChange != nil {
				config.OnStateChange(name, State(from), State(to))
			}
		},
	}
	return &circuitBreaker{
		cb: gobreaker.NewCircuitBreaker(settings),
	}
}

func Execute[T any](ctx context.Context, cb ports.CircuitBreaker, fn func() (T, error)) (T, error) {
	var zero T

	if err := ctx.Err(); err != nil {
		return zero, err
	}

	result, err := cb.(*circuitBreaker).cb.Execute(func() (interface{}, error) {
		type resultWrapper struct {
			value T
			err   error
		}
		done := make(chan resultWrapper, 1)

		go func() {
			val, err := fn()
			done <- resultWrapper{value: val, err: err}
		}()

		select {
		case res := <-done:
			return res.value, res.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	if err != nil {
		return zero, err
	}

	if result == nil {
		return zero, nil
	}
	typed, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf("circuit breaker returned unexpected type: got %T, expected %T", result, zero)
	}
	return typed, nil
}

func (cb *circuitBreaker) Name() string {
	return cb.cb.Name()
}

func (cb *circuitBreaker) IsOpen() bool {
	return cb.cb.State() == gobreaker.StateOpen
}

func (cb *circuitBreaker) Stats() ports.CircuitBreakerStats {
	counts := cb.cb.Counts()
	return ports.CircuitBreakerStats{
		Name:             cb.cb.Name(),
		State:            cb.cb.State().String(),
		FailureCount:     int(counts.TotalFailures),
		SuccessCount:     int(counts.ConsecutiveSuccesses),
		TotalRequests:    int64(counts.Requests),
		RejectedRequests: int64(counts.Requests - counts.TotalFailures),
	}
}

func GetState(cb ports.CircuitBreaker) string {
	if cbImpl, ok := cb.(*circuitBreaker); ok {
		return cbImpl.cb.State().String()
	}
	return "unknown"
}

func GetCounts(cb ports.CircuitBreaker) Counts {
	if cbImpl, ok := cb.(*circuitBreaker); ok {
		return cbImpl.cb.Counts()
	}
	return Counts{}
}

func IsOpen(cb ports.CircuitBreaker) bool {
	return cb.IsOpen()
}
