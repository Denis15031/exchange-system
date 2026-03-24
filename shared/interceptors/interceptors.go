package interceptors

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	RequestIDKey = "x-request-id"
)

// Добавляет x-request-id в контекст, если его нет, или извлекает существующий.
func XRequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		var requestID string

		if ok && len(md[RequestIDKey]) > 0 {
			requestID = md[RequestIDKey][0]
		} else {
			requestID = uuid.New().String()
		}
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(RequestIDKey, requestID))

		return handler(ctx, req)
	}
}

// Логирует начало и конец каждого gRPC запроса с использованием zap.
func LoggerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := "unknown"
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md[RequestIDKey]; len(ids) > 0 {
				requestID = ids[0]
			}
		}
		start := time.Now()

		logger.Info("gRPC request started",
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.Any("payload", req),
		)

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		if err != nil {
			logger.Error("gRPC request failed",
				zap.String("request_id", requestID),
				zap.String("method", info.FullMethod),
				zap.Duration("duration_ms", duration),
				zap.Error(err),
			)
		} else {
			logger.Info("gRPC request completed",
				zap.String("request_id", requestID),
				zap.String("method", info.FullMethod),
				zap.Duration("duration_ms", duration),
			)
		}
		return resp, err
	}
}

// Перехватывает паники, логирует стек и возвращает стандартную ошибку gRPC.
func UnaryPanicRecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic recovered in gRPC handler",
					zap.Any("recovered_value", r),
					zap.String("stack_trace", string(debug.Stack())),
				)
				err = status.Errorf(codes.Internal, "Internal server error: %v", r)
			}
		}()
		return handler(ctx, req)
	}
}

// Автоматически собирает метрики: количество запросов, длительность, ошибки.
func MetricsInterceptor() grpc.UnaryServerInterceptor {
	grpc_prometheus.EnableHandlingTimeHistogram()
	return grpc_prometheus.UnaryServerInterceptor
}
