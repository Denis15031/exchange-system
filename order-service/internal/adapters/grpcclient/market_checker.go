package grpcclient

import (
	"context"
	"fmt"
	"log"
	"time"

	"exchange-system/order-service/internal/domain"
	"exchange-system/order-service/internal/mapper"
	v1 "exchange-system/proto/spot/v1"
	"exchange-system/shared/resilience"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Реализует порт проверки рынков через gRPC с защитой
type MarketChecker struct {
	client v1.SpotInstrumentServiceClient
	cb     *resilience.CircuitBreaker
}

// Создает новый клиент с настроенным Circuit Breaker
func NewMarketChecker(conn *grpc.ClientConn) *MarketChecker {
	client := v1.NewSpotInstrumentServiceClient(conn)

	// Конфигурация из shared
	cbConfig := resilience.DefaultCircuitBreakerConfig("spot-service-checker")

	// Кастомная логика
	cbConfig.ReadyToTrip = func(counts resilience.Counts) bool {
		if counts.Requests < 5 {
			return false
		}
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return failureRatio >= 0.6
	}

	cbConfig.OnStateChange = func(name string, from resilience.State, to resilience.State) {
		log.Printf("Circuit Breaker [%s] changed state from %v to %v", name, from, to)
	}

	cb := resilience.NewCircuitBreaker(cbConfig)

	return &MarketChecker{
		client: client,
		cb:     cb,
	}
}

// Получает рынок с защитой
func (m *MarketChecker) GetMarket(ctx context.Context, marketID string) (*domain.Market, error) {

	market, err := resilience.Execute(m.cb, func() (*domain.Market, error) {

		reqCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()

		resp, err := m.client.GetMarket(reqCtx, &v1.GetMarketRequest{
			MarketId: marketID,
		})

		if err != nil {
			st, _ := status.FromError(err)
			log.Printf("gRPC call failed: code=%v, msg=%v", st.Code(), st.Message())

			// Сбои инфраструктуры — учитываем для Circuit Breaker
			if st.Code() == codes.DeadlineExceeded || st.Code() == codes.Unavailable || st.Code() == codes.Canceled {
				return nil, err
			}

			// Бизнес-ошибки (NotFound) — не считаем сбоем инфраструктуры
			if st.Code() == codes.NotFound {
				return nil, err
			}

			// Остальные ошибки — тоже сбой
			return nil, err
		}

		if resp.Market == nil {
			return nil, fmt.Errorf("received nil market from service")
		}

		market := mapper.ToDomainMarket(resp.Market)
		return market, nil
	})

	if err != nil {
		log.Printf("Circuit Breaker or Request failed: %v. Triggering fallback.", err)
		return m.getFallbackMarket(marketID)
	}

	return market, nil
}

// Возвращает безопасные данные по умолчанию, если сервис недоступен
func (m *MarketChecker) getFallbackMarket(marketID string) (*domain.Market, error) {
	log.Printf("Using fallback strategy for market: %s", marketID)
	return nil, fmt.Errorf("market validation service unavailable: cannot create order for %s", marketID)
}
