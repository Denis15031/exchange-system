//go:build integration
// +build integration

package handler_test

import (
	"context"
	"net"
	"testing"
	"time"

	spotV1 "exchange-system/proto/spot/v1"
	"exchange-system/shared/interceptors"
	"exchange-system/shared/ratelimit"
	"exchange-system/spot-service/internal/config"
	"exchange-system/spot-service/internal/di"
	"exchange-system/spot-service/internal/handler"
	"exchange-system/spot-service/internal/service"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var testAddr string

func TestGRPCServer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCfg := &config.Config{
		GRPCPort:            ":0", // Случайный порт
		MetricsPort:         ":0",
		LogLevel:            "error", // Меньше шума в логах
		LogFormat:           "console",
		RateLimitRPS:        1000,
		RateLimitBurst:      100,
		ShutdownGracePeriod: 5 * time.Second,
		UserServiceAddr:     "localhost:50053",
		OrderServiceAddr:    "localhost:50052",
	}

	app := fxtest.New(t,
		di.SpotModule,

		fx.Replace(testCfg),

		fx.Invoke(func(
			lc fx.Lifecycle,
			logger *zap.Logger,
			cfg *config.Config,
			svc *service.SpotService,
			h *handler.SpotHandler,
		) {
			grpcServer := grpc.NewServer(
				grpc.ChainUnaryInterceptor(
					interceptors.XRequestID(),
					interceptors.LoggerInterceptor(logger),
					interceptors.UnaryPanicRecoveryInterceptor(logger),
					ratelimit.UnaryServerInterceptor(
						ratelimit.NewRateLimiter(ratelimit.RateLimitConfig{
							RequestsPerSecond: float64(cfg.RateLimitRPS),
							MaxBurst:          cfg.RateLimitBurst,
						}),
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
					testAddr = lis.Addr().String()
					logger.Info("Test gRPC server", zap.String("addr", testAddr))
					go func() { _ = grpcServer.Serve(lis) }()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					grpcServer.GracefulStop()
					return nil
				},
			})
		}),
	)

	app.RequireStart()
	defer app.RequireStop()

	time.Sleep(100 * time.Millisecond)

	// Подключаемся как клиент
	conn, err := grpc.NewClient(testAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := spotV1.NewSpotInstrumentServiceClient(conn)

	t.Run("ViewMarkets returns response", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		resp, err := client.ViewMarkets(ctx, &spotV1.ViewMarketsRequest{})
		require.NoError(t, err, "ViewMarkets call should not fail")
		require.NotNil(t, resp, "Response should not be nil")

		t.Logf("ViewMarkets returned %d markets", len(resp.Markets))
	})
}
