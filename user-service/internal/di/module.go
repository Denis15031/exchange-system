package di

import (
	"context"
	"net"

	"exchange-system/shared/interceptors"
	"exchange-system/shared/logger"
	"exchange-system/shared/ratelimit"
	"exchange-system/user-service/internal/config"
	"exchange-system/user-service/internal/handler"
	"exchange-system/user-service/internal/jwtmanager"
	"exchange-system/user-service/internal/middleware"
	"exchange-system/user-service/internal/repository"
	"exchange-system/user-service/internal/service"

	userV1 "exchange-system/proto/user/v1"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var UserModule = fx.Module("user-service",
	fx.Provide(
		config.Load,

		func(cfg *config.Config) (*zap.Logger, error) {
			logCfg := logger.Config{
				Level:  cfg.LogLevel,
				Format: cfg.LogFormat,
			}
			l, err := logger.New(logCfg)
			if err != nil {
				return nil, err
			}
			return l.Zap(), nil
		},

		repository.NewUserRepository,
		repository.NewTokenRepository,

		jwtmanager.NewSigner,
		jwtmanager.NewValidator,
		middleware.NewAuthMiddleware,

		service.NewAuthService,
		handler.NewGRPCHandler,
	),

	fx.Invoke(func(
		lc fx.Lifecycle,
		logger *zap.Logger,
		cfg *config.Config,
		authMiddleware *middleware.AuthMiddleware,
		h *handler.GRPCHandler,
	) {
		grpc_prometheus.EnableHandlingTimeHistogram()

		interceptorsChain := []grpc.UnaryServerInterceptor{
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
		}

		if authMiddleware != nil {
			interceptorsChain = append(interceptorsChain, authMiddleware.UnaryServerInterceptor())
		}

		grpcServer := grpc.NewServer(
			grpc.ChainUnaryInterceptor(interceptorsChain...),
			grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle: cfg.GRPCKeepaliveMaxConnIdle,
				Time:              cfg.GRPCKeepaliveTime,
				Timeout:           cfg.GRPCKeepaliveTimeout,
			}),
		)

		userV1.RegisterUserServiceServer(grpcServer, h)

		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				lis, err := net.Listen("tcp", cfg.GRPCPort)
				if err != nil {
					return err
				}
				logger.Info("User-Service started", zap.String("port", cfg.GRPCPort))
				go func() {
					if err := grpcServer.Serve(lis); err != nil {
						logger.Error("gRPC fatal error", zap.Error(err))
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info("User-Service stopping")
				grpcServer.GracefulStop()
				return nil
			},
		})
	}),
)
