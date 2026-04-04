package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"exchange-system/shared/idempotency"
	"exchange-system/shared/interceptors"
	"exchange-system/shared/jwtvalidator"
	"exchange-system/shared/logger"
	"exchange-system/shared/ratelimit"
	"exchange-system/shared/shutdown"

	spotV1 "exchange-system/proto/spot/v1"
	"exchange-system/spot-service/internal/adapters/inmemory"
	"exchange-system/spot-service/internal/handler"
	"exchange-system/spot-service/internal/service"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func main() {
	grpcPort := getEnv("GRPC_PORT", ":50051")
	metricsPort := getEnv("METRICS_PORT", ":9090")
	logLevel := getEnv("LOG_LEVEL", "info")
	jwtPublicKeyPath := getEnv("JWT_PUBLIC_KEY_PATH", "../user-service/jwt_public.pem")

	logConfig := logger.DefaultConfig()
	logConfig.Format = "json"
	logConfig.Level = logLevel

	appLogger, err := logger.New(logConfig)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	shutdownHandler := shutdown.New(30 * time.Second)

	publicKey, err := jwtvalidator.LoadPublicKeyFromFile(jwtPublicKeyPath)
	if err != nil {
		appLogger.Warn("failed to load JWT public key, authentication will be disabled",
			zap.Error(err),
			zap.String("path", jwtPublicKeyPath),
		)

	}

	var jwtValidator *jwtvalidator.Validator
	if publicKey != nil {
		jwtValidator = jwtvalidator.NewValidator(publicKey)
		appLogger.Info("JWT validator initialized",
			zap.String("public_key_path", jwtPublicKeyPath),
		)
	}

	repo := inmemory.NewRepository()

	spotSvc := service.NewSpotService(repo)

	rateLimiter := ratelimit.NewRateLimiter(ratelimit.DefaultRateLimitConfig())

	idempotencyStore := idempotency.NewInMemoryStore(24 * time.Hour)
	idempotencyConfig := idempotency.DefaultIdempotencyConfig()
	idempotencyManager := idempotency.NewIdempotencyManager(idempotencyConfig, idempotencyStore)

	var authMiddleware *jwtvalidator.AuthMiddleware
	if jwtValidator != nil {
		authMiddleware = jwtvalidator.NewAuthMiddleware(jwtValidator, appLogger.Zap())
	}

	spotHandler := handler.NewSpotHandler(spotSvc)

	shutdownHandler.RegisterFunc("logger", func() error {
		return appLogger.Sync()
	})

	shutdownHandler.Register("idempotency", idempotencyManager)

	grpc_prometheus.EnableHandlingTimeHistogram()

	metricsServer := &http.Server{
		Addr:         metricsPort,
		Handler:      promhttp.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		appLogger.Info("Prometheus metrics started", zap.String("port", metricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("metrics server failed", zap.Error(err))
		}
	}()

	shutdownHandler.RegisterFunc("metrics-server", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return metricsServer.Shutdown(ctx)
	})

	baseInterceptors := []grpc.UnaryServerInterceptor{
		interceptors.XRequestID(),
		interceptors.LoggerInterceptor(appLogger.Zap()),
		interceptors.UnaryPanicRecoveryInterceptor(appLogger.Zap()),
		interceptors.MetricsInterceptor(),
		ratelimit.UnaryServerInterceptor(rateLimiter),
		idempotency.UnaryServerInterceptor(idempotencyManager),
	}

	if authMiddleware != nil {
		baseInterceptors = append(baseInterceptors, authMiddleware.UnaryServerInterceptor())
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(baseInterceptors...),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              2 * time.Minute,
			Timeout:           20 * time.Second,
		}),
	)

	spotV1.RegisterSpotInstrumentServiceServer(grpcServer, spotHandler)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		appLogger.Fatal("failed to listen", zap.Error(err))
	}

	appLogger.Info("SpotService is listening",
		zap.String("port", grpcPort),
		zap.Bool("auth_enabled", jwtValidator != nil),
	)

	shutdownHandler.RegisterFunc("grpc-server", func() error {
		grpcServer.GracefulStop()
		return nil
	})

	go func() {
		if err := shutdownHandler.Run(); err != nil {
			appLogger.Error("shutdown completed with errors", zap.Error(err))
		} else {
			appLogger.Info("shutdown completed successfully")
		}
	}()

	if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
		if !shutdownHandler.IsShuttingDown() {
			appLogger.Fatal("gRPC server failed", zap.Error(err))
		}
	}

	shutdownHandler.WaitForCompletion()
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
