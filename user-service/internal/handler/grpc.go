package handler

import (
	"context"
	"strings"

	userv1 "exchange-system/proto/user/v1"
	"exchange-system/shared/config"
	"exchange-system/shared/jwtvalidator"
	"exchange-system/user-service/internal/service"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	userv1.UnimplementedUserServiceServer
	authService *service.AuthService
	logger      *zap.Logger
	cfg         *config.Config
}

func NewGRPCHandler(
	authService *service.AuthService,
	logger *zap.Logger,
	cfg *config.Config,
) *GRPCHandler {
	return &GRPCHandler{
		authService: authService,
		logger:      logger,
		cfg:         cfg,
	}
}

func (h *GRPCHandler) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	h.logger.Info("Register request received",
		zap.String("email", req.Email),
		zap.String("username", req.Username),
	)

	resp, err := h.authService.Register(ctx, req)
	if err != nil {
		h.logger.Error("Register failed",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return nil, h.toGRPCError(err)
	}

	h.logger.Info("Register successful",
		zap.String("user_id", resp.User.UserId),
		zap.String("email", resp.User.Email),
	)

	return resp, nil
}

func (h *GRPCHandler) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	h.logger.Info("Login request received",
		zap.String("email", req.Email),
	)

	resp, err := h.authService.Login(ctx, req)
	if err != nil {
		h.logger.Warn("Login failed",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return nil, h.toGRPCError(err)
	}

	h.logger.Info("Login successful",
		zap.String("user_id", resp.User.UserId),
		zap.String("email", resp.User.Email),
	)

	return resp, nil
}

func (h *GRPCHandler) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	h.logger.Debug("RefreshToken request received")

	refreshToken := h.getRefreshToken(ctx, req.RefreshToken)
	if refreshToken == "" {
		return nil, status.Error(codes.Unauthenticated, "refresh token required")
	}

	proxyReq := &userv1.RefreshTokenRequest{RefreshToken: refreshToken}

	resp, err := h.authService.RefreshToken(ctx, proxyReq)
	if err != nil {
		h.logger.Warn("RefreshToken failed", zap.Error(err))
		return nil, h.toGRPCError(err)
	}

	return resp, nil
}

func (h *GRPCHandler) Logout(ctx context.Context, req *userv1.LogoutRequest) (*userv1.LogoutResponse, error) {
	h.logger.Debug("Logout request received")

	refreshToken := h.getRefreshToken(ctx, req.RefreshToken)
	if refreshToken == "" {
		return nil, status.Error(codes.Unauthenticated, "refresh token required")
	}

	if err := h.authService.Logout(ctx, refreshToken); err != nil {
		h.logger.Error("Logout failed", zap.Error(err))
		return nil, h.toGRPCError(err)
	}

	return &userv1.LogoutResponse{}, nil
}

func (h *GRPCHandler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	h.logger.Debug("GetUser request received",
		zap.String("user_id", req.UserId),
	)

	user, err := h.authService.GetUser(ctx, req.UserId)
	if err != nil {
		h.logger.Error("GetUser failed",
			zap.String("user_id", req.UserId),
			zap.Error(err),
		)
		return nil, h.toGRPCError(err)
	}

	return &userv1.GetUserResponse{User: user}, nil
}

func (h *GRPCHandler) UpdateRole(ctx context.Context, req *userv1.UpdateRoleRequest) (*userv1.UpdateRoleResponse, error) {
	h.logger.Info("UpdateRole request received",
		zap.String("user_id", req.UserId),
		zap.String("role", req.NewRole.String()),
	)

	claims, ok := jwtvalidator.ClaimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	actorRole := stringToUserRole(claims.Role)
	if actorRole != userv1.UserRole_USER_ROLE_ADMIN {
		return nil, status.Error(codes.PermissionDenied, "admin role required")
	}

	user, err := h.authService.UpdateRole(ctx, req.UserId, req.NewRole, actorRole)
	if err != nil {
		h.logger.Error("UpdateRole failed",
			zap.String("user_id", req.UserId),
			zap.Error(err),
		)
		return nil, h.toGRPCError(err)
	}

	h.logger.Info("UpdateRole successful",
		zap.String("user_id", req.UserId),
		zap.String("role", user.Role.String()),
	)

	return &userv1.UpdateRoleResponse{User: user}, nil
}

func stringToUserRole(roleStr string) userv1.UserRole {
	switch roleStr {
	case "ADMIN", "USER_ROLE_ADMIN":
		return userv1.UserRole_USER_ROLE_ADMIN
	case "USER", "USER_ROLE_USER":
		return userv1.UserRole_USER_ROLE_USER
	case "PREMIUM", "USER_ROLE_PREMIUM":
		return userv1.UserRole_USER_ROLE_PREMIUM
	default:
		return userv1.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func (h *GRPCHandler) getRefreshToken(ctx context.Context, bodyToken string) string {
	if !h.cfg.AuthUseHttpOnlyCookies {
		return bodyToken
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return bodyToken
	}

	cookies := md.Get("cookie")
	if len(cookies) == 0 {
		return bodyToken
	}

	for _, c := range strings.Split(cookies[0], ";") {
		c = strings.TrimSpace(c)
		if strings.HasPrefix(c, h.cfg.AuthCookieName+"=") {
			return strings.TrimPrefix(c, h.cfg.AuthCookieName+"=")
		}
	}

	return bodyToken
}

type httpResponseWriterKey struct{}

func HTTPResponseWriterInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		return handler(ctx, req)
	}
}

func (h *GRPCHandler) toGRPCError(err error) error {
	if err == nil {
		return nil
	}
	switch err {
	case service.ErrInvalidEmail, service.ErrInvalidPassword:
		return status.Error(codes.InvalidArgument, "invalid request parameters")
	case service.ErrUserNotFound:
		return status.Error(codes.NotFound, "user not found")
	case service.ErrUserAlreadyExists:
		return status.Error(codes.AlreadyExists, "user already exists")
	case service.ErrInvalidCredentials:
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case service.ErrTokenInvalid:
		return status.Error(codes.Unauthenticated, "invalid or expired token")
	case service.ErrUnauthorized, service.ErrPermissionDenied:
		return status.Error(codes.PermissionDenied, "access denied")
	default:
		h.logger.Error("unexpected error in handler", zap.Error(err))
		return status.Error(codes.Internal, "internal server error")
	}
}
