package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"exchange-system/shared/idempotency"
	"exchange-system/shared/interceptors"
	"exchange-system/shared/logger"
	"exchange-system/shared/ratelimit"
	"exchange-system/shared/shutdown"

	userv1 "exchange-system/proto/user/v1"
	"exchange-system/user-service/internal/config"
	"exchange-system/user-service/internal/handler"
	"exchange-system/user-service/internal/jwtmanager"
	"exchange-system/user-service/internal/middleware"
	"exchange-system/user-service/internal/repository"
	"exchange-system/user-service/internal/service"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	logConfig := logger.DefaultConfig()
	logConfig.Format = "json"
	logConfig.Level = cfg.LogLevel

	appLogger, err := logger.New(logConfig)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	shutdownHandler := shutdown.New(30 * time.Second)

	var keyPair *jwtmanager.KeyPair

	if cfg.JWTPrivateKeyPath != "" && cfg.JWTPublicKeyPath != "" {
		// Загрузка существующих ключей
		keyPair, err = jwtmanager.LoadKeyPair(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath)
		if err != nil {
			appLogger.Warn("failed to load JWT keys, generating new ones", zap.Error(err))
			keyPair, _ = jwtmanager.GenerateKeyPair()
			// Сохраняем для следующего запуска
			_ = jwtmanager.SaveKeyPair(keyPair, cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath)
		}
	} else {
		keyPair, err = jwtmanager.GenerateKeyPair()
		if err != nil {
			log.Fatalf("failed to generate JWT keys: %v", err)
		}
		appLogger.Warn("JWT keys generated in-memory (not persisted)")
	}

	userRepo := repository.NewUserRepository()
	tokenRepo := repository.NewTokenRepository()

	signer := jwtmanager.NewSigner(keyPair, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	validator := jwtmanager.NewValidator(keyPair.PublicKey)

	authService := service.NewAuthService(userRepo, tokenRepo, signer, appLogger.Zap())

	rateLimiter := ratelimit.NewRateLimiter(ratelimit.DefaultRateLimitConfig())

	idempotencyStore := idempotency.NewInMemoryStore(24 * time.Hour)
	idempotencyConfig := idempotency.DefaultIdempotencyConfig()
	idempotencyManager := idempotency.NewIdempotencyManager(idempotencyConfig, idempotencyStore)

	authMiddleware := middleware.NewAuthMiddleware(validator, appLogger.Zap())

	grpcHandler := handler.NewGRPCHandler(authService, appLogger.Zap())

	shutdownHandler.RegisterFunc("logger", func() error {
		return appLogger.Sync()
	})

	shutdownHandler.Register("idempotency", idempotencyManager)

	grpc_prometheus.EnableHandlingTimeHistogram()

	metricsServer := &http.Server{
		Addr:         cfg.MetricsPort,
		Handler:      promhttp.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		appLogger.Info("Prometheus metrics started", zap.String("port", cfg.MetricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("metrics server failed", zap.Error(err))
		}
	}()

	shutdownHandler.RegisterFunc("metrics-server", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return metricsServer.Shutdown(ctx)
	})

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.XRequestID(),
			interceptors.LoggerInterceptor(appLogger.Zap()),
			interceptors.UnaryPanicRecoveryInterceptor(appLogger.Zap()),
			interceptors.MetricsInterceptor(),
			ratelimit.UnaryServerInterceptor(rateLimiter),
			idempotency.UnaryServerInterceptor(idempotencyManager),
			authMiddleware.UnaryServerInterceptor(),
			// middleware.RequireRole(userv1.UserRole_USER_ROLE_ADMIN), // Для UpdateRole
		),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              2 * time.Minute,
			Timeout:           20 * time.Second,
		}),
	)

	userv1.RegisterUserServiceServer(grpcServer, grpcHandler)

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		appLogger.Fatal("failed to listen", zap.Error(err))
	}

	appLogger.Info("UserService is listening", zap.String("port", cfg.GRPCPort))

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
