package app

import (
	"context"
	"fmt"

	"exchange-system/order-client/internal/ports"
	orderv1 "exchange-system/proto/order/v1"
)

type OrderApp struct {
	client ports.OrderServiceClient
}

func NewOrderApp(client ports.OrderServiceClient) *OrderApp {
	return &OrderApp{client: client}
}

func (a *OrderApp) PlaceOrder(ctx context.Context, marketID, price, quantity string) (*orderv1.CreateOrderResponse, error) {
	resp, err := a.client.CreateOrder(ctx, marketID, orderv1.OrderType_ORDER_TYPE_BUY, price, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}
	return resp, nil
}
