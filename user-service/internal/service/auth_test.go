package service

import (
	"context"
	"testing"
	"time"

	userv1 "exchange-system/proto/user/v1"
	"exchange-system/user-service/internal/jwtmanager"
	"exchange-system/user-service/internal/repository"

	"go.uber.org/zap/zaptest"
)

func setupAuthService(t *testing.T) (*AuthService, *jwtmanager.KeyPair) {
	t.Helper()

	logger := zaptest.NewLogger(t)
	userRepo := repository.NewUserRepository()
	tokenRepo := repository.NewTokenRepository()
	keyPair, _ := jwtmanager.GenerateKeyPair()
	signer := jwtmanager.NewSigner(keyPair, 15*time.Minute, 7*24*time.Hour)

	return NewAuthService(userRepo, tokenRepo, signer, logger), keyPair
}

func TestAuthService_Register_Success(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	req := &userv1.RegisterRequest{
		Email:    "new@example.com",
		Password: "SecurePass123!",
		Username: "newuser",
	}

	resp, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if resp.User.Email != req.Email {
		t.Errorf("User.Email = %q, want %q", resp.User.Email, req.Email)
	}
	if resp.Token.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	req := &userv1.RegisterRequest{
		Email:    "dup@example.com",
		Password: "SecurePass123!",
		Username: "user1",
	}

	_, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	_, err = svc.Register(ctx, req)
	if err == nil {
		t.Error("Register() should fail for duplicate email")
	}
	if err != ErrUserAlreadyExists {
		t.Errorf("Error = %v, want ErrUserAlreadyExists", err)
	}
}

func TestAuthService_Register_InvalidEmail(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	req := &userv1.RegisterRequest{
		Email:    "invalid-email",
		Password: "SecurePass123!",
		Username: "user",
	}

	_, err := svc.Register(ctx, req)
	if err == nil {
		t.Error("Register() should fail for invalid email")
	}
	if err != ErrInvalidEmail {
		t.Errorf("Error = %v, want ErrInvalidEmail", err)
	}
}

func TestAuthService_Register_WeakPassword(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	req := &userv1.RegisterRequest{
		Email:    "test@example.com",
		Password: "short",
		Username: "user",
	}

	_, err := svc.Register(ctx, req)
	if err == nil {
		t.Error("Register() should fail for weak password")
	}
	if err != ErrInvalidPassword {
		t.Errorf("Error = %v, want ErrInvalidPassword", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &userv1.RegisterRequest{
		Email:    "login@example.com",
		Password: "SecurePass123!",
		Username: "loginuser",
	})

	req := &userv1.LoginRequest{
		Email:    "login@example.com",
		Password: "SecurePass123!",
	}

	resp, err := svc.Login(ctx, req)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if resp.User.Email != req.Email {
		t.Errorf("User.Email = %q, want %q", resp.User.Email, req.Email)
	}
	if resp.Token.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &userv1.RegisterRequest{
		Email:    "test@example.com",
		Password: "CorrectPass123!",
		Username: "user",
	})

	req := &userv1.LoginRequest{
		Email:    "test@example.com",
		Password: "WrongPass!",
	}

	_, err := svc.Login(ctx, req)
	if err == nil {
		t.Error("Login() should fail for wrong password")
	}
	if err != ErrInvalidCredentials {
		t.Errorf("Error = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthService_Login_NonExistentUser(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	req := &userv1.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "AnyPass123!",
	}

	_, err := svc.Login(ctx, req)
	if err == nil {
		t.Error("Login() should fail for nonexistent user")
	}
	if err != ErrInvalidCredentials {
		t.Errorf("Error = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthService_RefreshToken_Success(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &userv1.RegisterRequest{
		Email:    "refresh@example.com",
		Password: "SecurePass123!",
		Username: "refreshuser",
	})
	loginResp, _ := svc.Login(ctx, &userv1.LoginRequest{
		Email:    "refresh@example.com",
		Password: "SecurePass123!",
	})

	refreshReq := &userv1.RefreshTokenRequest{
		RefreshToken: loginResp.Token.RefreshToken,
	}

	refreshResp, err := svc.RefreshToken(ctx, refreshReq)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}

	if refreshResp.Token.AccessToken == "" {
		t.Error("New AccessToken should not be empty")
	}
	if refreshResp.Token.AccessToken == loginResp.Token.AccessToken {
		t.Error("New AccessToken should be different from old one")
	}
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	req := &userv1.RefreshTokenRequest{
		RefreshToken: "nonexistent-token",
	}

	_, err := svc.RefreshToken(ctx, req)
	if err == nil {
		t.Error("RefreshToken() should fail for invalid token")
	}
	if err != ErrTokenInvalid {
		t.Errorf("Error = %v, want ErrTokenInvalid", err)
	}
}

func TestAuthService_Logout_Success(t *testing.T) {
	t.Parallel()

	svc, _ := setupAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &userv1.RegisterRequest{
		Email:    "logout@example.com",
		Password: "SecurePass123!",
	})
	loginResp, _ := svc.Login(ctx, &userv1.LoginRequest{
		Email:    "logout@example.com",
		Password: "SecurePass123!",
	})

	err := svc.Logout(ctx, loginResp.Token.RefreshToken)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	_, err = svc.RefreshToken(ctx, &userv1.RefreshTokenRequest{
		RefreshToken: loginResp.Token.RefreshToken,
	})
	if err == nil {
		t.Error("RefreshToken() should fail after logout")
	}
}
