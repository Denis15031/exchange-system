package ports

import (
	"context"
	orderv1 "exchange-system/proto/order/v1"
)

type OrderRepository interface {
	Save(ctx context.Context, order *orderv1.Order) error
	GetByID(ctx context.Context, id string) (*orderv1.Order, error)
	ListByUser(ctx context.Context, userID string) ([]*orderv1.Order, error)
	CountByUser(ctx context.Context, userID string) (int, error)
}
