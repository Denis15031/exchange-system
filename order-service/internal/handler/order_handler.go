package handler

import (
	"context"

	"exchange-system/order-service/internal/service"
	orderv1 "exchange-system/proto/order/v1"
	userv1 "exchange-system/proto/user/v1"
	"exchange-system/shared/jwtvalidator"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	orderv1.UnimplementedOrderServiceServer
	orderSvc *service.OrderService
	logger   *zap.Logger
}

func NewOrderHandler(orderSvc *service.OrderService, logger *zap.Logger) *GRPCHandler {
	return &GRPCHandler{
		orderSvc: orderSvc,
		logger:   logger,
	}
}

func (h *GRPCHandler) CreateOrder(
	ctx context.Context,
	req *orderv1.CreateOrderRequest,
) (*orderv1.CreateOrderResponse, error) {

	claims, ok := jwtvalidator.ClaimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	userRole := claims.ToUserRole()
	if userRole != userv1.UserRole_USER_ROLE_USER &&
		userRole != userv1.UserRole_USER_ROLE_PREMIUM {
		return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
	}

	// Validation
	if req.MarketId == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}
	if req.Price == "" || req.Quantity == "" {
		return nil, status.Error(codes.InvalidArgument, "price and quantity are required")
	}
	if req.Type == orderv1.OrderType_ORDER_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "order type is required")
	}

	orderID := uuid.New().String()

	createdOrder, err := h.orderSvc.CreateOrder(ctx, &service.CreateOrderCommand{
		OrderID:  orderID,
		UserID:   claims.UserID,
		MarketID: req.MarketId,
		Type:     req.Type,
		Price:    req.Price,
		Quantity: req.Quantity,
	})

	if err != nil {
		h.logger.Error("CreateOrder failed",
			zap.String("user_id", claims.UserID),
			zap.String("market_id", req.MarketId),
			zap.Error(err),
		)
		return nil, h.toGRPCError(err)
	}

	h.logger.Info("order created",
		zap.String("order_id", createdOrder.OrderId),
		zap.String("user_id", claims.UserID),
	)

	return &orderv1.CreateOrderResponse{
		OrderId: createdOrder.OrderId,
		Status:  createdOrder.Status,
	}, nil
}

func (h *GRPCHandler) GetOrderStatus(
	ctx context.Context,
	req *orderv1.GetOrderStatusRequest,
) (*orderv1.GetOrderStatusResponse, error) {

	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	order, err := h.orderSvc.GetOrder(ctx, req.OrderId)
	if err != nil {
		h.logger.Warn("GetOrderStatus failed",
			zap.String("order_id", req.OrderId),
			zap.Error(err),
		)
		return nil, h.toGRPCError(err)
	}

	return &orderv1.GetOrderStatusResponse{Order: order}, nil
}

func (h *GRPCHandler) ListOrders(
	ctx context.Context,
	req *orderv1.ListOrdersRequest,
) (*orderv1.ListOrdersResponse, error) {

	claims, ok := jwtvalidator.ClaimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orders, err := h.orderSvc.ListOrders(ctx, claims.UserID)
	if err != nil {
		h.logger.Error("ListOrders failed",
			zap.String("user_id", claims.UserID),
			zap.Error(err),
		)
		return nil, h.toGRPCError(err)
	}

	return &orderv1.ListOrdersResponse{Orders: orders}, nil
}

func (h *GRPCHandler) toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case service.ErrOrderNotFound:
		return status.Error(codes.NotFound, "order not found")
	case service.ErrInvalidMarket:
		return status.Error(codes.InvalidArgument, "invalid or inactive market")
	case service.ErrInsufficientBalance:
		return status.Error(codes.FailedPrecondition, "insufficient balance")
	case service.ErrTooManyOrders:
		return status.Error(codes.ResourceExhausted, "too many open orders")
	default:
		// ❗️ Все остальные ошибки — общий "internal server error"
		// Детали уже залогированы выше, клиенту не показываем
		return status.Error(codes.Internal, "internal server error")
	}
}
