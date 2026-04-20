package client

import (
	"context"
	"fmt"

	"exchange-system/order-client/internal/config"
	"exchange-system/order-client/internal/ports"
	common "exchange-system/proto/common"
	orderv1 "exchange-system/proto/order/v1"
	"exchange-system/shared/logger"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Реализует ports.OrderServiceClient
type GRPCClient struct {
	conn          *grpc.ClientConn
	client        orderv1.OrderServiceClient
	logger        *logger.Logger
	tokenProvider interface{ GetAccessToken() (string, error) }
}

var _ ports.OrderServiceClient = (*GRPCClient)(nil)

// Создаёт gRPC-клиент
func New(cfg *config.Config, logger *logger.Logger, tokenProvider interface{ GetAccessToken() (string, error) }) (ports.OrderServiceClient, error) {
	conn, err := grpc.NewClient(
		cfg.OrderServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10*1024*1024),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to order service: %w", err)
	}

	return &GRPCClient{
		conn:          conn,
		client:        orderv1.NewOrderServiceClient(conn),
		logger:        logger,
		tokenProvider: tokenProvider,
	}, nil
}

func (c *GRPCClient) CreateOrder(
	ctx context.Context,
	marketID string,
	orderType orderv1.OrderType,
	price, quantity string,
) (*orderv1.CreateOrderResponse, error) {

	ctx, err := c.withAuth(ctx)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("sending CreateOrder request",
		zap.String("market_id", marketID),
		zap.String("order_type", orderType.String()),
	)

	resp, err := c.client.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: marketID,
		Type:     orderType,
		Price:    price,
		Quantity: quantity,
	})

	if err != nil {
		c.logger.Error("CreateOrder failed", zap.String("market_id", marketID), zap.Error(err))
		return nil, err
	}

	c.logger.Info("CreateOrder success",
		zap.String("order_id", resp.OrderId),
		zap.String("status", resp.Status.String()),
	)

	return resp, nil
}

func (c *GRPCClient) GetOrderStatus(ctx context.Context, orderID string) (*orderv1.GetOrderStatusResponse, error) {
	ctx, err := c.withAuth(ctx)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("sending GetOrderStatus request", zap.String("order_id", orderID))

	resp, err := c.client.GetOrderStatus(ctx, &orderv1.GetOrderStatusRequest{
		OrderId: orderID,
	})

	if err != nil {
		c.logger.Error("GetOrderStatus failed", zap.String("order_id", orderID), zap.Error(err))
		return nil, err
	}

	return resp, nil
}

// ListOrders получает список ордеров (с пагинацией)
func (c *GRPCClient) ListOrders(ctx context.Context, pageToken string, pageSize int32) (*orderv1.ListOrdersResponse, error) {
	ctx, err := c.withAuth(ctx)
	if err != nil {
		return nil, err
	}

	return c.client.ListOrders(ctx, &orderv1.ListOrdersRequest{

		Pagination: &common.CursorPaginationRequest{
			PageToken: pageToken,
			PageSize:  pageSize,
		},
	})
}

func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Добавляет JWT токен в контекст
func (c *GRPCClient) withAuth(ctx context.Context) (context.Context, error) {
	if c.tokenProvider == nil {
		return ctx, nil
	}

	token, err := c.tokenProvider.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token)), nil
}
