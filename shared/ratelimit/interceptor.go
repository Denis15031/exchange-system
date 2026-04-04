package ratelimit

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	// UserRoleKey ключ для извлечения роли из контекста
	UserRoleKey contextKey = "user_role"
)

func UnaryServerInterceptor(limiter *RateLimiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Проверка сервисного лимита
		if !limiter.Allow() {
			return nil, status.Error(
				codes.ResourceExhausted,
				"service rate limit exceeded",
			)
		}

		//Извлекаем роль из контекста (добавляется auth-интерсептором)
		role, ok := ctx.Value(UserRoleKey).(string)
		if ok && role != "" {
			if !limiter.AllowRole(role) {
				return nil, status.Error(
					codes.ResourceExhausted,
					"role rate limit exceeded: "+role,
				)
			}
		}

		//Продолжаем выполнение запроса
		return handler(ctx, req)
	}
}

func UnaryClientInterceptor(limiter *RateLimiter) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Проверка лимита перед отправкой запроса
		if !limiter.Allow() {
			return status.Error(
				codes.ResourceExhausted,
				"client rate limit exceeded",
			)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func StreamServerInterceptor(limiter *RateLimiter) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		//Проверка сервисного лимита
		if !limiter.Allow() {
			return status.Error(
				codes.ResourceExhausted,
				"service rate limit exceeded",
			)
		}

		//Проверка по роли (если есть)
		ctx := ss.Context()
		role, ok := ctx.Value(UserRoleKey).(string)
		if ok && role != "" {
			if !limiter.AllowRole(role) {
				return status.Error(
					codes.ResourceExhausted,
					"role rate limit exceeded: "+role,
				)
			}
		}

		return handler(srv, ss)
	}
}

func WithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, UserRoleKey, role)
}

func UserRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(UserRoleKey).(string)
	return role, ok && role != ""
}
