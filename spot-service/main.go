package main

import (
	"context"
	"net"

	spotV1 "exchange-system/proto/spot/v1"
	"exchange-system/shared/interceptors"
	"exchange-system/shared/ratelimit"
	"exchange-system/spot-service/internal/config"
	"exchange-system/spot-service/internal/di"
	"exchange-system/spot-service/internal/handler"
	"exchange-system/spot-service/internal/service"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	app := fx.New(
		di.SpotModule,

		fx.Invoke(func(
			lc fx.Lifecycle,
			logger *zap.Logger,
			cfg *config.Config,
			svc *service.SpotService,
			h *handler.SpotHandler,
		) {

			rlCfg := ratelimit.RateLimitConfig{
				RequestsPerSecond: float64(cfg.RateLimitRPS),
				MaxBurst:          cfg.RateLimitBurst,
			}

			grpcServer := grpc.NewServer(
				grpc.ChainUnaryInterceptor(
					interceptors.XRequestID(),
					interceptors.LoggerInterceptor(logger),
					interceptors.UnaryPanicRecoveryInterceptor(logger),
					ratelimit.UnaryServerInterceptor(
						ratelimit.NewRateLimiter(rlCfg),
					),
				),
			)

			spotV1.RegisterSpotInstrumentServiceServer(grpcServer, h)

			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					lis, err := net.Listen("tcp", cfg.GRPCPort)
					if err != nil {
						return err
					}
					logger.Info("Starting gRPC server", zap.String("port", cfg.GRPCPort))

					go func() {
						if err := grpcServer.Serve(lis); err != nil {
							logger.Error("gRPC server error", zap.Error(err))
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					logger.Info("Shutting down gRPC server")
					grpcServer.GracefulStop()
					return nil
				},
			})
		}),
	)

	app.Run()
}
