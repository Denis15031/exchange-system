package main

import (
	"log"
	"net"
	"net/http"

	"exchange-system/order-service/internal/handler"
	"exchange-system/order-service/internal/service"
	
	"exchange-system/shared/interceptors"

	orderV1 "exchange-system/order-service/proto/order/v1"
	spotV1 "exchange-system/order-service/proto/spot/v1"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() { _ = zapLogger.Sync() }()

	zapLogger.Info("Starting OrderService...")

	// 1. Репозиторий (пока заглушка)
	repo := initOrderRepository()

	// 2. Клиент Spot Service
	spotConn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		zapLogger.Fatal("failed to connect to Spot Service", zap.Error(err))
	}
	defer spotConn.Close()

	spotClient := spotV1.NewSpotInstrumentServiceClient(spotConn)

	orderSvc := service.NewOrderService(repo, spotClient)
	orderHandler := handler.NewOrderHandler(orderSvc)

	grpc_prometheus.EnableHandlingTimeHistogram()
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		zapLogger.Info("Prometheus metrics started on :9091")
		if err := http.ListenAndServe(":9091", nil); err != nil {
			zapLogger.Fatal("metrics server failed", zap.Error(err))
		}
	}()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.XRequestID(),
			interceptors.LoggerInterceptor(zapLogger),
			interceptors.UnaryPanicRecoveryInterceptor(zapLogger),
			interceptors.MetricsInterceptor(),
		),
	)

	orderV1.RegisterOrderServiceServer(grpcServer, orderHandler)

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		zapLogger.Fatal("failed to listen on port 50052", zap.Error(err))
	}

	zapLogger.Info("OrderService is listening on port 50052")
	if err := grpcServer.Serve(lis); err != nil {
		zapLogger.Fatal("failed to serve gRPC server", zap.Error(err))
	}
}

func initOrderRepository() interface{} {
	return nil
}
