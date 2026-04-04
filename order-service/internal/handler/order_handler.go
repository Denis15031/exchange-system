package handler

import (
	"context"

	orderv1 "exchange-system/proto/order/v1"
	"exchange-system/shared/jwtvalidator"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	orderv1.UnimplementedOrderServiceServer
	logger *zap.Logger
}

func NewOrderHandler(logger *zap.Logger) *GRPCHandler {
	return &GRPCHandler{
		logger: logger,
	}
}

func (h *GRPCHandler) CreateOrder(
	ctx context.Context,
	req *orderv1.CreateOrderRequest,
) (*orderv1.CreateOrderResponse, error) {

	userID, ok := jwtvalidator.GetUserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	if req.MarketId == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}
	if req.Price == "" || req.Quantity == "" {
		return nil, status.Error(codes.InvalidArgument, "price and quantity are required")
	}

	orderID := uuid.New().String()

	h.logger.Info("order created",
		zap.String("order_id", orderID),
		zap.String("user_id", userID),
		zap.String("market_id", req.MarketId),
		zap.String("type", req.Type.String()),
		zap.String("price", req.Price),
		zap.String("quantity", req.Quantity),
	)

	return &orderv1.CreateOrderResponse{
		OrderId: orderID,
		Status:  "PENDING",
	}, nil
}

func (h *GRPCHandler) GetOrderStatus(
	ctx context.Context,
	req *orderv1.GetOrderStatusRequest,
) (*orderv1.GetOrderStatusResponse, error) {

	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	return &orderv1.GetOrderStatusResponse{
		Order: &orderv1.Order{
			OrderId:  req.OrderId,
			Status:   orderv1.OrderStatus_ORDER_STATUS_PENDING, // ✅ enum
			Price:    "0",
			Quantity: "0",
		},
	}, nil
}

func (h *GRPCHandler) ListOrders(
	ctx context.Context,
	req *orderv1.ListOrdersRequest,
) (*orderv1.ListOrdersResponse, error) {

	return &orderv1.ListOrdersResponse{
		Orders: []*orderv1.Order{},
	}, nil
}
