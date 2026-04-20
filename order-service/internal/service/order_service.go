package service

import (
	"context"
	"fmt"
	"time"

	"exchange-system/order-service/internal/ports"
	orderv1 "exchange-system/proto/order/v1"
	"exchange-system/shared/uid"

	"go.uber.org/zap"
)

const MaxOpenOrdersPerUser = 500

type CreateOrderCommand struct {
	OrderID  string
	UserID   string
	MarketID string
	Type     orderv1.OrderType
	Price    string
	Quantity string
}

type OrderService struct {
	repo         ports.OrderRepository
	marketClient ports.MarketClient
	logger       *zap.Logger
}

func NewOrderService(
	repo ports.OrderRepository,
	marketClient ports.MarketClient,
	logger *zap.Logger,
) *OrderService {
	return &OrderService{
		repo:         repo,
		marketClient: marketClient,
		logger:       logger,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, cmd *CreateOrderCommand) (*orderv1.Order, error) {
	if err := uid.Validate(cmd.MarketID, "market_id"); err != nil {
		return nil, err
	}
	if err := uid.Validate(cmd.UserID, "user_id"); err != nil {
		return nil, err
	}

	start := time.Now()
	s.logger.Debug("CreateOrder started",
		zap.String("user_id", cmd.UserID),
		zap.String("market_id", cmd.MarketID),
	)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	count, err := s.repo.CountByUser(ctx, cmd.UserID)
	if err != nil {
		s.logger.Error("failed to count user orders", zap.Error(err))
		return nil, fmt.Errorf("check order limit: %w", err)
	}
	if count >= MaxOpenOrdersPerUser {
		return nil, ErrTooManyOrders
	}

	if s.marketClient != nil {
		active, err := s.marketClient.CheckMarketActive(ctx, cmd.MarketID)
		if err != nil {
			s.logger.Warn("market check failed", zap.String("market_id", cmd.MarketID), zap.Error(err))
		} else if !active {
			return nil, ErrInvalidMarket
		}
	}

	orderID := uid.New()

	order := &orderv1.Order{
		OrderId:        orderID,
		MarketId:       cmd.MarketID,
		UserId:         cmd.UserID,
		Type:           cmd.Type,
		Status:         orderv1.OrderStatus_ORDER_STATUS_PENDING,
		Price:          cmd.Price,
		Quantity:       cmd.Quantity,
		FilledQuantity: "0",
	}

	if err := s.repo.Save(ctx, order); err != nil {
		s.logger.Error("failed to save order", zap.String("order_id", orderID), zap.Error(err))
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	s.logger.Info("CreateOrder completed",
		zap.String("order_id", orderID),
		zap.Duration("took", time.Since(start)),
	)

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*orderv1.Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, orderID)
}

func (s *OrderService) ListOrders(ctx context.Context, userID string) ([]*orderv1.Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.repo.ListByUser(ctx, userID)
}
