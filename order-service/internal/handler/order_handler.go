package handler

import (
	"context"
	"exchange-system/order-service/internal/service"
	orderV1 "exchange-system/order-service/proto/order/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderHandler struct {
	orderV1.UnimplementedOrderServiceServer
	service *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{service: svc}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *orderV1.CreateOrderRequest) (*orderV1.CreateOrderResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateOrder not implemented")
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *orderV1.GetOrderRequest) (*orderV1.GetOrderResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetOrder not implemented")
}
