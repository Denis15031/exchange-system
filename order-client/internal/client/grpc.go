package client

import (
	"context"

	"exchange-system/order-client/internal/auth"
	"exchange-system/order-client/internal/config"
	commonv1 "exchange-system/proto/common"
	orderv1 "exchange-system/proto/order/v1"
	"exchange-system/shared/logger" // ✅ Ваш logger, не zap

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func AuthInterceptor(tokenManager *auth.TokenManager) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		token, err := tokenManager.GetAccessToken()
		if err != nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		md := metadata.Pairs("authorization", "Bearer "+token)
		if reqID, ok := ctx.Value("x-request-id").(string); ok && reqID != "" {
			md = metadata.Join(md, metadata.Pairs("x-request-id", reqID))
		}

		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

type GRPCClient struct {
	conn         *grpc.ClientConn
	orderClient  orderv1.OrderServiceClient
	tokenManager *auth.TokenManager
	logger       *logger.Logger
}

func New(cfg *config.Config, log *logger.Logger, tokenManager *auth.TokenManager) (*GRPCClient, error) {
	interceptors := []grpc.UnaryClientInterceptor{
		AuthInterceptor(tokenManager),
	}

	conn, err := grpc.NewClient(
		cfg.OrderServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(interceptors...),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10*1024*1024),
			grpc.MaxCallSendMsgSize(10*1024*1024),
		),
	)
	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		conn:         conn,
		orderClient:  orderv1.NewOrderServiceClient(conn),
		tokenManager: tokenManager,
		logger:       log,
	}, nil
}

func (c *GRPCClient) CreateOrder(
	ctx context.Context,
	marketID string,
	orderType orderv1.OrderType,
	price string,
	quantity string,
) (*orderv1.CreateOrderResponse, error) {

	logger := c.logger.WithContext(ctx)

	logger.Info("creating order",
		zap.String("market_id", marketID),
		zap.String("order_type", orderType.String()),
	)

	resp, err := c.orderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: marketID,
		Type:     orderType,
		Price:    price,
		Quantity: quantity,
	})
	if err != nil {
		logger.Error("create order failed",
			zap.Error(err),
			zap.String("market_id", marketID),
		)
		return nil, err
	}

	logger.Info("order created",
		zap.String("order_id", resp.OrderId),
		zap.String("status", resp.Status),
	)

	return resp, nil
}

func (c *GRPCClient) GetOrderStatus(ctx context.Context, orderID string) (*orderv1.GetOrderStatusResponse, error) {
	logger := c.logger.WithContext(ctx)

	logger.Debug("getting order status",
		zap.String("order_id", orderID),
	)

	return c.orderClient.GetOrderStatus(ctx, &orderv1.GetOrderStatusRequest{
		OrderId: orderID,
	})
}

func (c *GRPCClient) ListOrders(ctx context.Context, page, pageSize int32) (*orderv1.ListOrdersResponse, error) {
	logger := c.logger.WithContext(ctx)

	logger.Debug("listing orders",
		zap.Int32("page", page),
		zap.Int32("page_size", pageSize),
	)

	req := &orderv1.ListOrdersRequest{
		Pagination: &commonv1.PaginationRequest{
			PageNumber: page,
			PageSize:   pageSize,
		},
	}

	return c.orderClient.ListOrders(ctx, req)
}

func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
