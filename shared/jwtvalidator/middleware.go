package jwtvalidator

import (
	"context"
	"strings"

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
		if m.isPublicEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		token, err := m.extractToken(ctx)
		if err != nil {
			m.logger.Debug("failed to extract token",
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
			return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
		}

		claims, err := m.validator.ValidateAndExtract(token)
		if err != nil {
			m.logger.Warn("invalid token",
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = ContextWithClaims(ctx, claims)

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
	if len(header) < len(bearerPrefix) || !strings.HasPrefix(header, bearerPrefix) {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header format")
	}
	return header[len(bearerPrefix):], nil
}

func (m *AuthMiddleware) isPublicEndpoint(method string) bool {
	publicMethods := map[string]bool{
		"/spot.v1.SpotInstrumentService/GetMarket":   true,
		"/spot.v1.SpotInstrumentService/ListMarkets": true,
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
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "role not found in context")
		}

		userRole := claims.ToUserRole()
		for _, role := range requiredRoles {
			if userRole == role {
				return handler(ctx, req)
			}
		}

		return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
	}
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.UserID, true
}

func GetRoleFromContext(ctx context.Context) (userv1.UserRole, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return userv1.UserRole_USER_ROLE_UNSPECIFIED, false
	}
	return claims.ToUserRole(), true
}

func GetEmailFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.Email, true
}
