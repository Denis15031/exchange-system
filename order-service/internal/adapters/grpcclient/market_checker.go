package grpcclient

import (
	"context"
	"fmt"
	"log"
	"time"

	"exchange-system/order-service/internal/domain"
	"exchange-system/order-service/internal/mapper"
	"exchange-system/order-service/internal/ports"
	v1 "exchange-system/proto/spot/v1"
	sharedports "exchange-system/shared/ports"
	"exchange-system/shared/resilience"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Реализует порт проверки рынков через gRPC с защитой
type MarketChecker struct {
	client v1.SpotInstrumentServiceClient
	cb     sharedports.CircuitBreaker
}

// Проверяем реализацию локального интерфейса
var _ ports.MarketClient = (*MarketChecker)(nil)

// Создаёт клиент с настроенным Circuit Breaker
func NewMarketChecker(conn *grpc.ClientConn) ports.MarketClient {
	client := v1.NewSpotInstrumentServiceClient(conn)

	cbConfig := resilience.DefaultCircuitBreakerConfig("spot-service-checker")

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

func (m *MarketChecker) CheckMarketActive(ctx context.Context, marketID string) (bool, error) {
	market, err := m.GetMarket(ctx, marketID)
	if err != nil {
		return false, err
	}
	return market != nil && market.Enabled, nil
}

func (m *MarketChecker) GetMarket(ctx context.Context, marketID string) (*domain.Market, error) {

	market, err := resilience.Execute[*domain.Market](ctx, m.cb, func() (*domain.Market, error) {

		reqCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()

		resp, err := m.client.GetMarket(reqCtx, &v1.GetMarketRequest{
			MarketId: marketID,
		})

		if err != nil {
			st, _ := status.FromError(err)
			log.Printf("gRPC call failed: code=%v, msg=%v", st.Code(), st.Message())

			if st.Code() == codes.DeadlineExceeded ||
				st.Code() == codes.Unavailable ||
				st.Code() == codes.Canceled {
				return nil, err
			}

			if st.Code() == codes.NotFound {
				return nil, err
			}

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

func (m *MarketChecker) getFallbackMarket(marketID string) (*domain.Market, error) {
	log.Printf("Using fallback strategy for market: %s", marketID)
	return nil, fmt.Errorf("market validation service unavailable: cannot create order for %s", marketID)
}
