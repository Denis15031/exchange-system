package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"exchange-system/order-service/internal/domain"
	"exchange-system/order-service/internal/ports"
	orderv1 "exchange-system/proto/order/v1"

	"go.uber.org/zap/zaptest"
)

var _ ports.MarketClient = (*MockMarketClient)(nil)
var _ ports.OrderRepository = (*MockOrderRepo)(nil)

type MockMarketClient struct {
	active bool
	err    error
}

func (m *MockMarketClient) GetMarket(ctx context.Context, marketID string) (*domain.Market, error) {
	return nil, nil
}

func (m *MockMarketClient) CheckMarketActive(ctx context.Context, marketID string) (bool, error) {
	return m.active, m.err
}

type MockOrderRepo struct {
	savedOrder *orderv1.Order
	count      int
	err        error
}

func (m *MockOrderRepo) Save(ctx context.Context, order *orderv1.Order) error {
	m.savedOrder = order
	return m.err
}

func (m *MockOrderRepo) GetByID(ctx context.Context, id string) (*orderv1.Order, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.savedOrder, nil
}

func (m *MockOrderRepo) ListByUser(ctx context.Context, userID string) ([]*orderv1.Order, error) {
	return nil, nil
}

func (m *MockOrderRepo) CountByUser(ctx context.Context, userID string) (int, error) {
	return m.count, m.err
}

func validUUID() string {
	return "550e8400-e29b-41d4-a716-446655440000"
}

func TestCreateOrder_Success(t *testing.T) {
	t.Parallel()

	activeMarket := &MockMarketClient{active: true}
	mockRepo := &MockOrderRepo{count: 0}
	logger := zaptest.NewLogger(t)

	svc := NewOrderService(mockRepo, activeMarket, logger)

	cmd := &CreateOrderCommand{
		UserID:   validUUID(),
		MarketID: validUUID(),
		Type:     orderv1.OrderType_ORDER_TYPE_BUY,
		Price:    "50000",
		Quantity: "0.1",
	}

	order, err := svc.CreateOrder(context.Background(), cmd)

	if err != nil {
		t.Fatalf("CreateOrder() unexpected error: %v", err)
	}
	if order == nil {
		t.Fatal("CreateOrder() returned nil order")
	}
	if order.MarketId != cmd.MarketID {
		t.Errorf("MarketId = %s, want %s", order.MarketId, cmd.MarketID)
	}
	if mockRepo.savedOrder == nil {
		t.Error("Order was not saved to repository")
	}
}

func TestCreateOrder_MarketInactive(t *testing.T) {
	t.Parallel()

	inactiveMarket := &MockMarketClient{active: false}
	mockRepo := &MockOrderRepo{}
	logger := zaptest.NewLogger(t)

	svc := NewOrderService(mockRepo, inactiveMarket, logger)

	cmd := &CreateOrderCommand{
		UserID:   validUUID(),
		MarketID: validUUID(),
		Type:     orderv1.OrderType_ORDER_TYPE_BUY,
		Price:    "3000",
		Quantity: "1",
	}

	_, err := svc.CreateOrder(context.Background(), cmd)

	if err == nil {
		t.Error("CreateOrder() should fail for inactive market")
	}

	if err != nil {
		isInvalidMarket := errors.Is(err, ErrInvalidMarket) ||
			strings.Contains(strings.ToLower(err.Error()), "invalid") ||
			strings.Contains(strings.ToLower(err.Error()), "inactive") ||
			strings.Contains(strings.ToLower(err.Error()), "market")

		if !isInvalidMarket {
			t.Logf("Ошибка: %v", err)
		}
	}

	if mockRepo.savedOrder != nil {
		t.Error("Order should NOT be saved for inactive market")
	}
}

func TestCreateOrder_LimitExceeded(t *testing.T) {
	t.Parallel()

	activeMarket := &MockMarketClient{active: true}
	mockRepo := &MockOrderRepo{count: 500}
	logger := zaptest.NewLogger(t)

	svc := NewOrderService(mockRepo, activeMarket, logger)

	cmd := &CreateOrderCommand{
		UserID:   validUUID(),
		MarketID: validUUID(),
		Type:     orderv1.OrderType_ORDER_TYPE_BUY,
		Price:    "50000",
		Quantity: "0.1",
	}

	_, err := svc.CreateOrder(context.Background(), cmd)

	if err == nil {
		t.Error("CreateOrder() should fail when limit exceeded")
	}

	if err != nil {
		isLimitErr := errors.Is(err, ErrTooManyOrders) ||
			strings.Contains(strings.ToLower(err.Error()), "limit") ||
			strings.Contains(strings.ToLower(err.Error()), "many") ||
			strings.Contains(strings.ToLower(err.Error()), "500")

		if !isLimitErr {
			t.Logf("Ошибка лимита: %v", err)
		}
	}
}

func TestCreateOrder_InvalidUserID(t *testing.T) {
	t.Parallel()

	mockRepo := &MockOrderRepo{}
	mockMarket := &MockMarketClient{active: true}
	logger := zaptest.NewLogger(t)

	svc := NewOrderService(mockRepo, mockMarket, logger)

	cmd := &CreateOrderCommand{
		UserID:   "invalid-not-uuid!",
		MarketID: validUUID(),
		Type:     orderv1.OrderType_ORDER_TYPE_BUY,
		Price:    "50000",
		Quantity: "0.1",
	}

	_, err := svc.CreateOrder(context.Background(), cmd)

	if err == nil {
		t.Error("CreateOrder() should fail for invalid user_id")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid") {
		t.Logf("Got error: %v (expected 'invalid' in message)", err)
	}
}

func TestGetOrder_Success(t *testing.T) {
	t.Parallel()

	expectedOrder := &orderv1.Order{
		OrderId:  validUUID(),
		MarketId: validUUID(),
		UserId:   validUUID(),
		Status:   orderv1.OrderStatus_ORDER_STATUS_PENDING,
	}

	mockRepo := &MockOrderRepo{savedOrder: expectedOrder}
	mockMarket := &MockMarketClient{active: true}
	logger := zaptest.NewLogger(t)

	svc := NewOrderService(mockRepo, mockMarket, logger)

	order, err := svc.GetOrder(context.Background(), expectedOrder.OrderId)

	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if order.OrderId != expectedOrder.OrderId {
		t.Errorf("OrderId = %s, want %s", order.OrderId, expectedOrder.OrderId)
	}
}

func TestGetOrder_NotFound(t *testing.T) {
	t.Parallel()

	// Репозиторий возвращает ошибку "не найдено"
	mockRepo := &MockOrderRepo{err: errors.New("order not found")}
	mockMarket := &MockMarketClient{active: true}
	logger := zaptest.NewLogger(t)

	svc := NewOrderService(mockRepo, mockMarket, logger)

	_, err := svc.GetOrder(context.Background(), validUUID())

	if err == nil {
		t.Error("GetOrder() should return error for nonexistent order")
		return
	}

	if !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Logf("Got error: %v (expected 'not found' in message)", err)
	}
}
