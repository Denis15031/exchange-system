package ports

import (
	"context"
	"exchange-system/spot-service/internal/domain"
)

// Определяет контракт для работы с данными рынков
type MarketRepository interface {
	GetAll(ctx context.Context) ([]domain.Market, error)

	GetByID(ctx context.Context, id string) (*domain.Market, error)
}
