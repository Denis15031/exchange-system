package grpcclient

import (
	"context"
	"errors"

	"exchange-system/order-service/internal/domain"
	"exchange-system/order-service/internal/mapper"
	spotV1 "exchange-system/order-service/proto/spot/v1"

	"google.golang.org/grpc"
)

type MarketCheckerClient struct {
	client spotV1.SpotInstrumentServiceClient
}

func NewMarketCheckerClient(conn *grpc.ClientConn) *MarketCheckerClient {
	return &MarketCheckerClient{
		client: spotV1.NewSpotInstrumentServiceClient(conn),
	}
}

func (c *MarketCheckerClient) GetMarket(ctx context.Context, marketID string) (*domain.Market, error) {
	resp, err := c.client.GetMarket(ctx, &spotV1.GetMarketRequest{
		MarketId: marketID,
	})

	if err != nil {
		return nil, err
	}

	if resp.Market == nil {
		return nil, errors.New("market not found in response")
	}

	return mapper.ToDomainMarket(resp.Market), nil
}
