package handler

import (
	"context"

	userv1 "exchange-system/proto/user/v1"
	"exchange-system/user-service/internal/service"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	userv1.UnimplementedUserServiceServer
	authService *service.AuthService
	logger      *zap.Logger
}

func NewGRPCHandler(
	authService *service.AuthService,
	logger *zap.Logger,
) *GRPCHandler {
	return &GRPCHandler{
		authService: authService,
		logger:      logger,
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

	resp, err := h.authService.RefreshToken(ctx, req)
	if err != nil {
		h.logger.Warn("RefreshToken failed",
			zap.Error(err),
		)
		return nil, h.toGRPCError(err)
	}

	return resp, nil
}

func (h *GRPCHandler) Logout(ctx context.Context, req *userv1.LogoutRequest) (*userv1.LogoutResponse, error) {
	h.logger.Debug("Logout request received")

	if err := h.authService.Logout(ctx, req.RefreshToken); err != nil {
		h.logger.Error("Logout failed", zap.Error(err))
		return nil, h.toGRPCError(err)
	}

	return &userv1.LogoutResponse{Success: true}, nil
}

func (h *GRPCHandler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.User, error) {
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

	return user, nil
}

func (h *GRPCHandler) UpdateRole(ctx context.Context, req *userv1.UpdateRoleRequest) (*userv1.User, error) {
	h.logger.Info("UpdateRole request received",
		zap.String("user_id", req.UserId),
		zap.String("role", req.Role.String()),
	)

	actorRole := getRoleFromContext(ctx)

	user, err := h.authService.UpdateRole(ctx, req.UserId, req.Role, actorRole)
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

	return user, nil
}

func (h *GRPCHandler) toGRPCError(err error) error {
	switch err {
	case service.ErrInvalidEmail, service.ErrInvalidPassword:
		return status.Error(codes.InvalidArgument, err.Error())
	case service.ErrUserNotFound:
		return status.Error(codes.NotFound, err.Error())
	case service.ErrUserAlreadyExists:
		return status.Error(codes.AlreadyExists, err.Error())
	case service.ErrInvalidCredentials:
		return status.Error(codes.Unauthenticated, err.Error())
	case service.ErrTokenInvalid:
		return status.Error(codes.Unauthenticated, err.Error())
	case service.ErrUnauthorized:
		return status.Error(codes.PermissionDenied, err.Error())
	case service.ErrPermissionDenied:
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		h.logger.Error("unexpected error in handler", zap.Error(err))
		return status.Error(codes.Internal, "internal server error")
	}
}

func getRoleFromContext(ctx context.Context) userv1.UserRole {
	role, ok := ctx.Value("user_role").(userv1.UserRole)
	if !ok {
		return userv1.UserRole_USER_ROLE_UNSPECIFIED
	}
	return role
}
