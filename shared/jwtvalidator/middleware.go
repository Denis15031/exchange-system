package jwtvalidator

import (
	"context"

	userv1 "exchange-system/proto/user/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthMiddleware struct {
	validator *Validator
	logger    *zap.Logger
}

func NewAuthMiddleware(validator *Validator, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		validator: validator,
		logger:    logger,
	}
}

func (m *AuthMiddleware) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Public endpoints не требуют аутентификации
		if m.isPublicEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		// Извлечение токена
		token, err := m.extractToken(ctx)
		if err != nil {
			m.logger.Debug("failed to extract token",
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
			return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
		}

		// Валидация токена
		claims, err := m.validator.Validate(token)
		if err != nil {
			m.logger.Warn("invalid token",
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Добавление claims в контекст
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_email", claims.Email)
		ctx = context.WithValue(ctx, "user_role", claims.Role)

		return handler(ctx, req)
	}
}

func (m *AuthMiddleware) extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) > 0 {
		return m.parseAuthHeader(authHeaders[0])
	}

	tokenHeaders := md.Get("x-access-token")
	if len(tokenHeaders) > 0 {
		return tokenHeaders[0], nil
	}

	return "", status.Error(codes.Unauthenticated, "missing authorization header")
}

func (m *AuthMiddleware) parseAuthHeader(header string) (string, error) {
	const bearerPrefix = "Bearer "
	if len(header) < len(bearerPrefix) || header[:len(bearerPrefix)] != bearerPrefix {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header format")
	}
	return header[len(bearerPrefix):], nil
}

func (m *AuthMiddleware) isPublicEndpoint(method string) bool {
	publicMethods := map[string]bool{
		// Spot service public endpoints
		"/spot.v1.SpotInstrumentService/GetMarket":   true,
		"/spot.v1.SpotInstrumentService/ListMarkets": true,
		// Order service public endpoints (если есть)
		// "/order.v1.OrderService/...": true,
	}
	return publicMethods[method]
}

func RequireRole(requiredRoles ...userv1.UserRole) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		userRole, ok := ctx.Value("user_role").(userv1.UserRole)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "role not found in context")
		}

		// Проверка роли
		for _, role := range requiredRoles {
			if userRole == role {
				return handler(ctx, req)
			}
		}

		return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
	}
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("user_id").(string)
	return userID, ok
}

func GetRoleFromContext(ctx context.Context) (userv1.UserRole, bool) {
	role, ok := ctx.Value("user_role").(userv1.UserRole)
	return role, ok
}

func GetEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value("user_email").(string)
	return email, ok
}
