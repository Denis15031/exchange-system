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

func LoggerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := "unknown"
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md[RequestIDKey]; len(ids) > 0 {
				requestID = ids[0]
			}
		}
		start := time.Now()

		logger.Info("gRPC request",
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.String("type", "started"),
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
			logger.Info("gRPC request",
				zap.String("request_id", requestID),
				zap.String("method", info.FullMethod),
				zap.String("type", "completed"),
				zap.Duration("duration_ms", duration),
			)
		}
		return resp, err
	}
}

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

func MetricsInterceptor() grpc.UnaryServerInterceptor {
	grpc_prometheus.EnableHandlingTimeHistogram()
	return grpc_prometheus.UnaryServerInterceptor
}

func XRequestIDClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		requestID, ok := ctx.Value(RequestIDKey).(string)
		if !ok || requestID == "" {
			requestID = uuid.New().String()
			ctx = context.WithValue(ctx, RequestIDKey, requestID)
		}
		ctx = metadata.AppendToOutgoingContext(ctx, RequestIDKey, requestID)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func LoggerClientInterceptor(log *zap.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		requestID, _ := ctx.Value(RequestIDKey).(string)

		log.Debug("gRPC client call",
			zap.String("method", method),
			zap.String("request_id", requestID),
		)

		err := invoker(ctx, method, req, reply, cc, opts...)

		if err != nil {
			log.Error("gRPC client error",
				zap.String("method", method),
				zap.String("request_id", requestID),
				zap.Error(err),
			)
		} else {
			log.Debug("gRPC client call completed",
				zap.String("method", method),
				zap.String("request_id", requestID),
			)
		}

		return err
	}
}
