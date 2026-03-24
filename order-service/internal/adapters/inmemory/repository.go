package inmemory

import (
	"context"
	"errors"
	"exchange-system/order-service/internal/domain"
	"exchange-system/order-service/internal/ports"
	"sync"
	"time"
)

var ErrOrderNotFound = errors.New("order not found")

type Repository struct {
	mu     sync.RWMutex
	orders map[string]*domain.Order
}

func NewOrderRepository() ports.OrderRepository {
	return &Repository{
		orders: make(map[string]*domain.Order),
	}
}

func (r *Repository) Save(ctx context.Context, order *domain.Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	order.UpdatedAt = time.Now()
	r.orders[order.ID] = order
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	orderCopy := *order
	return &orderCopy, nil
}
