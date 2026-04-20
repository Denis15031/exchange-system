package jwtmanager

import (
	"testing"
	"time"

	userv1 "exchange-system/proto/user/v1"
)

func TestGenerateKeyPair(t *testing.T) {
	t.Parallel()

	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if kp.PrivateKey == nil {
		t.Error("PrivateKey should not be nil")
	}
	if kp.PublicKey == nil {
		t.Error("PublicKey should not be nil")
	}
}

func TestSigner_GenerateAndValidateTokens(t *testing.T) {
	t.Parallel()

	kp, _ := GenerateKeyPair()
	signer := NewSigner(kp, 15*time.Minute, 7*24*time.Hour)

	userID := "user-123"
	email := "test@example.com"
	role := userv1.UserRole_USER_ROLE_USER

	tokenPair, err := signer.GenerateTokens(userID, email, role)
	if err != nil {
		t.Fatalf("GenerateTokens() error = %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if tokenPair.ExpiresAt == 0 {
		t.Error("ExpiresAt should be set")
	}
	if tokenPair.RefreshToken == nil {
		t.Error("RefreshToken should not be nil")
	}
	if tokenPair.RefreshToken.Token == "" {
		t.Error("RefreshToken.Token should not be empty")
	}
	if tokenPair.RefreshToken.UserID != userID {
		t.Errorf("RefreshToken.UserID = %q, want %q", tokenPair.RefreshToken.UserID, userID)
	}

	claims, err := signer.ValidateToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Claims.UserID = %q, want %q", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("Claims.Email = %q, want %q", claims.Email, email)
	}
	if claims.Role == "" {
		t.Error("Role should not be empty")
	}
}

func TestSigner_ValidateExpiredToken(t *testing.T) {
	t.Parallel()

	kp, _ := GenerateKeyPair()
	signer := NewSigner(kp, 1*time.Millisecond, 7*24*time.Hour)

	tokenPair, err := signer.GenerateTokens("user-123", "test@example.com", userv1.UserRole_USER_ROLE_USER)
	if err != nil {
		t.Fatalf("GenerateTokens() error = %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = signer.ValidateToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("ValidateToken() should fail for expired token")
	}
}

func TestSigner_ValidateTamperedToken(t *testing.T) {
	t.Parallel()

	kp, _ := GenerateKeyPair()
	signer := NewSigner(kp, 15*time.Minute, 7*24*time.Hour)

	tokenPair, err := signer.GenerateTokens("user-123", "test@example.com", userv1.UserRole_USER_ROLE_USER)
	if err != nil {
		t.Fatalf("GenerateTokens() error = %v", err)
	}

	tampered := tokenPair.AccessToken[:10] + "X" + tokenPair.AccessToken[11:]

	_, err = signer.ValidateToken(tampered)
	if err == nil {
		t.Error("ValidateToken() should fail for tampered token")
	}
}

func TestValidator_ValidateWithPublicKey(t *testing.T) {
	t.Parallel()

	kp, _ := GenerateKeyPair()
	signer := NewSigner(kp, 15*time.Minute, 7*24*time.Hour)
	validator := NewValidator(kp.PublicKey)

	tokenPair, err := signer.GenerateTokens("user-123", "test@example.com", userv1.UserRole_USER_ROLE_USER)
	if err != nil {
		t.Fatalf("GenerateTokens() error = %v", err)
	}

	claims, err := validator.Validate(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Validator.Validate() error = %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
}

func TestValidator_RejectWrongKey(t *testing.T) {
	t.Parallel()

	kp1, _ := GenerateKeyPair()
	kp2, _ := GenerateKeyPair()

	signer := NewSigner(kp1, 15*time.Minute, 7*24*time.Hour)
	validator := NewValidator(kp2.PublicKey)

	tokenPair, err := signer.GenerateTokens("user-123", "test@example.com", userv1.UserRole_USER_ROLE_USER)
	if err != nil {
		t.Fatalf("GenerateTokens() error = %v", err)
	}

	_, err = validator.Validate(tokenPair.AccessToken)
	if err == nil {
		t.Error("Validator should reject token signed with different key")
	}
}
