package inmemory

import (
	"context"
	"errors"
	"sync"

	"exchange-system/order-service/internal/ports"
	orderv1 "exchange-system/proto/order/v1"
)

var (
	ErrOrderNotFound = errors.New("order not found")
)

type Repository struct {
	mu     sync.RWMutex
	orders map[string]*orderv1.Order
}

func NewOrderRepository() ports.OrderRepository {
	return &Repository{
		orders: make(map[string]*orderv1.Order),
	}
}

func (r *Repository) Save(ctx context.Context, order *orderv1.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.OrderId] = order
	return nil
}

func (r *Repository) GetByID(ctx context.Context, orderID string) (*orderv1.Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID string) ([]*orderv1.Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*orderv1.Order
	for _, order := range r.orders {
		if order.UserId == userID {
			result = append(result, order)
		}
	}
	return result, nil
}
func (r *Repository) CountByUser(ctx context.Context, userID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, o := range r.orders {
		if o.UserId == userID {
			count++
		}
	}
	return count, nil
}
