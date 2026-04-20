package di

import (
	"context"
	"net"
	"os"

	"exchange-system/order-service/internal/config"
	"exchange-system/order-service/internal/handler"
	"exchange-system/order-service/internal/service"
	"exchange-system/shared/adapters"
	"exchange-system/shared/idempotency"
	"exchange-system/shared/interceptors"
	"exchange-system/shared/logger"
	"exchange-system/shared/ports"
	"exchange-system/shared/ratelimit"

	orderV1 "exchange-system/proto/order/v1"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var OrderModule = fx.Module("order-service",
	fx.Provide(
		config.Load,

		func(cfg *config.Config) (*zap.Logger, error) {
			l, err := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})
			if err != nil {
				return nil, err
			}
			return l.Zap(), nil
		},

		func(cfg *config.Config, logger *zap.Logger) (ports.IdempotencyStore, error) {
			if os.Getenv("USE_REDIS") == "true" {
				redisCache, err := adapters.NewRedisCache(
					getEnv("REDIS_ADDR", "localhost:6379"),
					getEnv("REDIS_PASSWORD", ""),
					0,
					"idempotency",
				)
				if err != nil {
					return nil, err
				}
				return adapters.NewRedisStore(redisCache), nil
			}

			return idempotency.NewStore(
				cfg.IdempotencyTTL,
				cfg.IdempotencyCleanupInterval,
				cfg.IdempotencyMaxKeys,
				logger,
			), nil
		},

		service.NewOrderService,
		handler.NewOrderHandler,
	),

	fx.Invoke(func(
		lc fx.Lifecycle,
		logger *zap.Logger,
		cfg *config.Config,
		h *handler.GRPCHandler,
	) {
		grpc_prometheus.EnableHandlingTimeHistogram()

		grpcServer := grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				interceptors.XRequestID(),
				interceptors.LoggerInterceptor(logger),
				interceptors.UnaryPanicRecoveryInterceptor(logger),
				interceptors.MetricsInterceptor(),
				ratelimit.UnaryServerInterceptor(
					ratelimit.NewRateLimiter(ratelimit.RateLimitConfig{
						RequestsPerSecond: float64(cfg.RateLimitRPS),
						MaxBurst:          cfg.RateLimitBurst,
					}),
				),
			),
			grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle: cfg.GRPCKeepaliveMaxConnIdle,
				Time:              cfg.GRPCKeepaliveTime,
				Timeout:           cfg.GRPCKeepaliveTimeout,
			}),
		)

		orderV1.RegisterOrderServiceServer(grpcServer, h)

		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				lis, err := net.Listen("tcp", cfg.GRPCPort)
				if err != nil {
					return err
				}
				logger.Info("Order-Service started", zap.String("port", cfg.GRPCPort))
				go func() {
					if err := grpcServer.Serve(lis); err != nil {
						logger.Error("gRPC fatal error", zap.Error(err))
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info("Order-Service stopping")
				grpcServer.GracefulStop()
				return nil
			},
		})
	}),
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
