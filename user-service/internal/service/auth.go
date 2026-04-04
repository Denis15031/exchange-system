package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	userv1 "exchange-system/proto/user/v1"
	"exchange-system/user-service/internal/domain"
	"exchange-system/user-service/internal/jwtmanager"
	"exchange-system/user-service/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrInvalidPassword    = errors.New("password must be at least 8 characters")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTokenInvalid       = errors.New("invalid or expired token")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrPermissionDenied   = errors.New("permission denied")
)

type AuthService struct {
	userRepo  *repository.UserRepository
	tokenRepo *repository.TokenRepository
	signer    *jwtmanager.Signer
	logger    *zap.Logger
}

func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	signer *jwtmanager.Signer,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		signer:    signer,
		logger:    logger,
	}
}

func (s *AuthService) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	// Валидация email
	if !domain.ValidateEmail(req.Email) {
		return nil, ErrInvalidEmail
	}

	// Валидация пароля
	if !domain.ValidatePassword(req.Password) {
		return nil, ErrInvalidPassword
	}

	// Проверка на существующего пользователя
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		s.logger.Error("failed to check existing user", zap.Error(err))
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Хеширование пароля
	hashedPassword, err := domain.HashPassword(req.Password, 12)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создание пользователя
	user := &domain.User{
		UserID:   uuid.New().String(),
		Email:    req.Email,
		Username: req.Username,
		Password: hashedPassword,
		Role:     userv1.UserRole_USER_ROLE_USER,
		IsActive: true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Генерация токенов
	accessToken, refreshToken, err := s.signer.GenerateTokens(user.UserID, user.Email, user.Role)
	if err != nil {
		s.logger.Error("failed to generate tokens", zap.Error(err))
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Сохранение refresh токена
	if err := s.tokenRepo.Store(ctx, refreshToken); err != nil {
		s.logger.Error("failed to store refresh token", zap.Error(err))
	}

	s.logger.Info("user registered",
		zap.String("user_id", user.UserID),
		zap.String("email", user.Email),
	)

	return &userv1.RegisterResponse{
		User:  user.ToProto(),
		Token: accessToken,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	// Поиск пользователя по email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			_ = domain.CheckPassword(req.Password, "dummy_hash_for_timing")
			return nil, ErrInvalidCredentials
		}
		s.logger.Error("failed to get user by email", zap.Error(err))
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Проверка пароля
	if !domain.CheckPassword(req.Password, user.Password) {
		s.logger.Warn("invalid password attempt",
			zap.String("email", req.Email),
			zap.String("user_id", user.UserID),
		)
		return nil, ErrInvalidCredentials
	}

	// Проверка активности
	if !user.IsActive {
		s.logger.Warn("inactive user login attempt",
			zap.String("email", req.Email),
			zap.String("user_id", user.UserID),
		)
		return nil, ErrUnauthorized
	}

	accessToken, refreshToken, err := s.signer.GenerateTokens(user.UserID, user.Email, user.Role)
	if err != nil {
		s.logger.Error("failed to generate tokens", zap.Error(err))
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	if err := s.tokenRepo.Store(ctx, refreshToken); err != nil {
		s.logger.Error("failed to store refresh token", zap.Error(err))
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	s.logger.Info("user logged in",
		zap.String("user_id", user.UserID),
		zap.String("email", user.Email),
	)

	return &userv1.LoginResponse{
		User:  user.ToProto(),
		Token: accessToken,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	// Получение refresh токена из хранилища
	storedToken, err := s.tokenRepo.Get(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) ||
			errors.Is(err, repository.ErrTokenRevoked) ||
			errors.Is(err, repository.ErrTokenExpired) {
			return nil, ErrTokenInvalid
		}
		s.logger.Error("failed to get refresh token", zap.Error(err))
		return nil, fmt.Errorf("database error: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		s.logger.Error("failed to get user", zap.Error(err))
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrUnauthorized
	}

	if err := s.tokenRepo.Revoke(ctx, req.RefreshToken); err != nil {
		s.logger.Warn("failed to revoke old refresh token", zap.Error(err))
	}

	newAccessToken, newRefreshToken, err := s.signer.GenerateTokens(user.UserID, user.Email, user.Role)
	if err != nil {
		s.logger.Error("failed to generate new tokens", zap.Error(err))
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	if err := s.tokenRepo.Store(ctx, newRefreshToken); err != nil {
		s.logger.Error("failed to store new refresh token", zap.Error(err))
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &userv1.RefreshTokenResponse{
		Token: newAccessToken,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.tokenRepo.Revoke(ctx, refreshToken); err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil
		}
		s.logger.Error("failed to revoke token", zap.Error(err))
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	if err := s.tokenRepo.RevokeAllByUser(ctx, userID); err != nil {
		s.logger.Error("failed to revoke all tokens", zap.Error(err))
		return fmt.Errorf("failed to revoke all tokens: %w", err)
	}

	return nil
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*userv1.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("failed to get user", zap.Error(err))
		return nil, fmt.Errorf("database error: %w", err)
	}

	return user.ToProto(), nil
}

func (s *AuthService) UpdateRole(ctx context.Context, userID string, newRole userv1.UserRole, actorRole userv1.UserRole) (*userv1.User, error) {
	// Проверка прав
	if actorRole != userv1.UserRole_USER_ROLE_ADMIN {
		return nil, ErrPermissionDenied
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("failed to get user", zap.Error(err))
		return nil, fmt.Errorf("database error: %w", err)
	}

	user.Role = newRole
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to update user role", zap.Error(err))
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("user role updated",
		zap.String("user_id", userID),
		zap.String("new_role", newRole.String()),
	)

	return user.ToProto(), nil
}

func (s *AuthService) ValidateToken(tokenString string) (*jwtmanager.Claims, error) {
	claims, err := s.signer.ValidateToken(tokenString)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
