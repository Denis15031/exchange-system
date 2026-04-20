package ports

import (
	"context"
	orderv1 "exchange-system/proto/order/v1"
)

type OrderServiceClient interface {
	CreateOrder(ctx context.Context, marketID string, orderType orderv1.OrderType, price, quantity string) (*orderv1.CreateOrderResponse, error)
	GetOrderStatus(ctx context.Context, orderID string) (*orderv1.GetOrderStatusResponse, error)
	ListOrders(ctx context.Context, pageToken string, pageSize int32) (*orderv1.ListOrdersResponse, error)
	Close() error
}
