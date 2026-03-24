package main

import (
	"log"
	"net"
	"net/http"

	"exchange-system/spot-service/internal/adapters/inmemory"
	"exchange-system/spot-service/internal/handler"
	"exchange-system/spot-service/internal/service"

	v1 "exchange-system/spot-service/proto/spot/v1"

	"exchange-system/shared/interceptors"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting SpotInstrumentService...")

	// Инициализация слоев
	repo := inmemory.NewRepository()
	spotSVC := service.NewSpotService(repo)
	spotHandler := handler.NewSpotHandler(spotSVC)

	// Включаем метрики gRPC
	grpc_prometheus.EnableHandlingTimeHistogram()

	// Запуск сервера метрик (Prometheus)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logger.Info("Prometheus metrics server started on :9090")
		if err := http.ListenAndServe(":9090", nil); err != nil {
			logger.Fatal("failed to start metrics server", zap.Error(err))
		}
	}()

	// Создание gRPC сервера с интерсепторами из shared
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.XRequestID(),
			interceptors.LoggerInterceptor(logger),
			interceptors.UnaryPanicRecoveryInterceptor(logger),
			interceptors.MetricsInterceptor(),
		),
	)

	// Регистрация сервиса
	v1.RegisterSpotInstrumentServiceServer(grpcServer, spotHandler)

	// Запуск слушателя
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Fatal("Failed to listen on port 50051", zap.Error(err))
	}

	logger.Info("SpotInstrumentService is listening on port 50051")

	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("Failed to serve gRPC server", zap.Error(err))
	}
}
