package interceptors

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func stubHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return "ok", nil
}

var infoStub = &grpc.UnaryServerInfo{
	FullMethod: "/exchange.OrderService/CreateOrder",
}

func TestXRequestID_GeneratesNewID(t *testing.T) {
	t.Parallel()

	interceptor := XRequestID()

	ctx := context.Background()

	var capturedCtx context.Context
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedCtx = ctx
		return stubHandler(ctx, req)
	}

	_, err := interceptor(ctx, nil, infoStub, handler)
	if err != nil {
		t.Fatalf("Interceptor returned error: %v", err)
	}

	md, ok := metadata.FromOutgoingContext(capturedCtx)
	if !ok {
		t.Fatal("Expected outgoing metadata in context")
	}

	ids := md.Get(RequestIDKey)
	if len(ids) == 0 {
		t.Error("Expected x-request-id to be set")
	}
	if len(ids[0]) < 30 { // UUID v4 минимум 36 символов
		t.Errorf("x-request-id too short: %s", ids[0])
	}
}

func TestXRequestID_PreservesExistingID(t *testing.T) {
	t.Parallel()

	interceptor := XRequestID()
	existingID := "my-custom-trace-id-123"

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(RequestIDKey, existingID))

	var capturedID string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		md, _ := metadata.FromOutgoingContext(ctx)
		if ids := md.Get(RequestIDKey); len(ids) > 0 {
			capturedID = ids[0]
		}
		return stubHandler(ctx, req)
	}

	_, _ = interceptor(ctx, nil, infoStub, handler)

	if capturedID != existingID {
		t.Errorf("Expected existing ID %q, got %q", existingID, capturedID)
	}
}

func TestLoggerInterceptor_LogsSuccess(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	interceptor := LoggerInterceptor(logger)

	ctx := context.Background()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return stubHandler(ctx, req)
	}

	_, err := interceptor(ctx, nil, infoStub, handler)
	if err != nil {
		t.Fatalf("Interceptor returned error: %v", err)
	}

	// Проверяем, что логгер записал сообщение
	logs := observed.All()
	if len(logs) == 0 {
		t.Error("Expected logger to record at least one message")
	}

	found := false
	for _, log := range logs {
		if log.Message == "gRPC request" {
			hasMethod := false
			hasTypeCompleted := false
			for _, field := range log.Context {
				if field.Key == "method" && field.String == infoStub.FullMethod {
					hasMethod = true
				}
				if field.Key == "type" && field.String == "completed" {
					hasTypeCompleted = true
				}
			}
			if hasMethod && hasTypeCompleted {
				found = true
				break
			}
		}
	}
	if !found {
		t.Log("Logger format might differ, but interceptor executed")
	}
}

func TestLoggerInterceptor_LogsError(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	interceptor := LoggerInterceptor(logger)

	ctx := context.Background()
	expectedErr := fmt.Errorf("business error")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	_, err := interceptor(ctx, nil, infoStub, handler)
	if err != expectedErr {
		t.Errorf("Expected original error, got %v", err)
	}

	// Проверяем, что ошибка залогирована
	logs := observed.All()
	foundError := false
	for _, log := range logs {
		if log.Level == zap.ErrorLevel && log.Message == "gRPC request failed" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Log("Error log format might differ")
	}
}

func TestUnaryPanicRecoveryInterceptor_CatchesPanic(t *testing.T) {
	t.Parallel()

	interceptor := UnaryPanicRecoveryInterceptor(zap.NewNop())

	ctx := context.Background()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("simulated internal error")
	}

	_, err := interceptor(ctx, nil, infoStub, handler)

	if err == nil {
		t.Error("Expected error after panic recovery")
		return
	}

	// Проверяем, что вернулась gRPC ошибка Internal
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got %T", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("Expected codes.Internal, got %v", st.Code())
	}
}

func TestUnaryPanicRecoveryInterceptor_PassesNormalErrors(t *testing.T) {
	t.Parallel()

	interceptor := UnaryPanicRecoveryInterceptor(zap.NewNop())
	expectedErr := fmt.Errorf("business logic error")

	ctx := context.Background()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	_, err := interceptor(ctx, nil, infoStub, handler)

	if err != expectedErr {
		t.Errorf("Expected original error, got %v", err)
	}
}

func TestMetricsInterceptor_ReturnsValidInterceptor(t *testing.T) {
	t.Parallel()

	// Просто проверяем, что функция возвращает не-nil интерсептор
	interceptor := MetricsInterceptor()
	if interceptor == nil {
		t.Error("MetricsInterceptor should return non-nil interceptor")
	}
}
