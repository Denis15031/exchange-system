package middleware

import (
	"context"
	userv1 "exchange-system/proto/user/v1"
	"strings"

	"exchange-system/shared/jwtvalidator"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var publicEndpoints = map[string]bool{
	"/user.v1.UserService/Register":     true,
	"/user.v1.UserService/Login":        true,
	"/user.v1.UserService/RefreshToken": true,
}

func IsPublicEndpoint(method string) bool {
	return publicEndpoints[method]
}

func RegisterPublicEndpoint(method string) {
	publicEndpoints[method] = true
}

type AuthMiddleware struct {
	validator *jwtvalidator.Validator
	logger    *zap.Logger
}

func NewAuthMiddleware(validator *jwtvalidator.Validator, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		validator: validator,
		logger:    logger,
	}
}

func (m *AuthMiddleware) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if m.isPublicEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			m.logger.Debug("missing metadata", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			m.logger.Debug("missing authorization header", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		authHeader := authHeaders[0]
		if !strings.HasPrefix(authHeader, "Bearer ") {
			m.logger.Debug("invalid authorization format", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := m.validator.ValidateAndExtract(tokenString)
		if err != nil {
			m.logger.Debug("invalid token",
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		roleVal, ok := userv1.UserRole_value[claims.Role]
		if !ok {
			m.logger.Warn("unknown role in token", zap.String("role", claims.Role))
			return nil, status.Error(codes.PermissionDenied, "invalid role")
		}
		roleEnum := userv1.UserRole(roleVal)

		ctx = jwtvalidator.ContextWithClaims(ctx, claims)

		m.logger.Debug("authenticated",
			zap.String("method", info.FullMethod),
			zap.String("user_id", claims.UserID),
			zap.String("email", claims.Email),
			zap.String("role", roleEnum.String()),
		)

		return handler(ctx, req)
	}
}

func (m *AuthMiddleware) isPublicEndpoint(method string) bool {
	return IsPublicEndpoint(method)
}
