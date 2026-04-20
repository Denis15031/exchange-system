package ports

import (
	"context"
	"exchange-system/order-service/internal/domain"
)

type MarketClient interface {
	GetMarket(ctx context.Context, marketID string) (*domain.Market, error)
	CheckMarketActive(ctx context.Context, marketID string) (bool, error)
}
