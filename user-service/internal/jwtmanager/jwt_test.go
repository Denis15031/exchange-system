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
	if kp.PrivateKey.Public() != kp.PublicKey {
		t.Error("PublicKey should match PrivateKey.Public()")
	}
}

func TestKeyPair_SaveAndLoad(t *testing.T) {
	t.Parallel()

	privatePath := t.TempDir() + "/private.pem"
	publicPath := t.TempDir() + "/public.pem"

	original, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if err := SaveKeyPair(original, privatePath, publicPath); err != nil {
		t.Fatalf("SaveKeyPair() error = %v", err)
	}

	loaded, err := LoadKeyPair(privatePath, publicPath)
	if err != nil {
		t.Fatalf("LoadKeyPair() error = %v", err)
	}

	if original.PrivateKey.N.Cmp(loaded.PrivateKey.N) != 0 {
		t.Error("Loaded PrivateKey should match original")
	}
	if original.PublicKey.N.Cmp(loaded.PublicKey.N) != 0 {
		t.Error("Loaded PublicKey should match original")
	}
}

func TestSigner_GenerateAndValidateTokens(t *testing.T) {
	t.Parallel()

	kp, _ := GenerateKeyPair()
	signer := NewSigner(kp, 15*time.Minute, 7*24*time.Hour)

	userID := "user-123"
	email := "test@example.com"
	role := userv1.UserRole_USER_ROLE_USER

	accessToken, refreshToken, err := signer.GenerateTokens(userID, email, role)
	if err != nil {
		t.Fatalf("GenerateTokens() error = %v", err)
	}

	if accessToken.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if refreshToken.Token == "" {
		t.Error("RefreshToken should not be empty")
	}
	if refreshToken.UserID != userID {
		t.Errorf("RefreshToken.UserID = %q, want %q", refreshToken.UserID, userID)
	}

	claims, err := signer.ValidateToken(accessToken.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Claims.UserID = %q, want %q", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("Claims.Email = %q, want %q", claims.Email, email)
	}
	if claims.Role != role {
		t.Errorf("Claims.Role = %v, want %v", claims.Role, role)
	}
}

func TestSigner_ValidateExpiredToken(t *testing.T) {
	t.Parallel()

	kp, _ := GenerateKeyPair()
	signer := NewSigner(kp, 1*time.Millisecond, 7*24*time.Hour)

	accessToken, _, _ := signer.GenerateTokens("user-123", "test@example.com", userv1.UserRole_USER_ROLE_USER)

	time.Sleep(10 * time.Millisecond)

	_, err := signer.ValidateToken(accessToken.AccessToken)
	if err == nil {
		t.Error("ValidateToken() should fail for expired token")
	}
}

func TestSigner_ValidateTamperedToken(t *testing.T) {
	t.Parallel()

	kp, _ := GenerateKeyPair()
	signer := NewSigner(kp, 15*time.Minute, 7*24*time.Hour)

	accessToken, _, _ := signer.GenerateTokens("user-123", "test@example.com", userv1.UserRole_USER_ROLE_USER)

	tampered := accessToken.AccessToken[:10] + "X" + accessToken.AccessToken[11:]

	_, err := signer.ValidateToken(tampered)
	if err == nil {
		t.Error("ValidateToken() should fail for tampered token")
	}
}

func TestValidator_ValidateWithPublicKey(t *testing.T) {
	t.Parallel()

	kp, _ := GenerateKeyPair()
	signer := NewSigner(kp, 15*time.Minute, 7*24*time.Hour)
	validator := NewValidator(kp.PublicKey)

	accessToken, _, _ := signer.GenerateTokens("user-123", "test@example.com", userv1.UserRole_USER_ROLE_USER)

	claims, err := validator.Validate(accessToken.AccessToken)
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

	accessToken, _, _ := signer.GenerateTokens("user-123", "test@example.com", userv1.UserRole_USER_ROLE_USER)

	_, err := validator.Validate(accessToken.AccessToken)
	if err == nil {
		t.Error("Validator should reject token signed with different key")
	}
}
