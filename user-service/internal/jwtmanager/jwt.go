package jwtmanager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	userv1 "exchange-system/proto/user/v1"
	"exchange-system/user-service/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	Issuer            = "exchange-system-user-service"
	DefaultAccessTTL  = 15 * time.Minute
	DefaultRefreshTTL = 7 * 24 * time.Hour
	KeyBits           = 2048
)

type Claims struct {
	UserID string          `json:"user_id"`
	Email  string          `json:"email"`
	Role   userv1.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type KeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func GenerateKeyPair() (*KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, KeyBits)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

func LoadKeyPair(privateKeyPath, publicKeyPath string) (*KeyPair, error) {
	privData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	privBlock, _ := pem.Decode(privData)
	if privBlock == nil {
		return nil, errors.New("failed to decode private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	pubData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	pubBlock, _ := pem.Decode(pubData)
	if pubBlock == nil {
		return nil, errors.New("failed to decode public key")
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

func SaveKeyPair(kp *KeyPair, privateKeyPath, publicKeyPath string) error {
	privBytes := x509.MarshalPKCS1PrivateKey(kp.PrivateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})

	if err := os.WriteFile(privateKeyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(kp.PublicKey)
	if err != nil {
		return err
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	if err := os.WriteFile(publicKeyPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

type Signer struct {
	keyPair    *KeyPair
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewSigner(keyPair *KeyPair, accessTTL, refreshTTL time.Duration) *Signer {
	if accessTTL <= 0 {
		accessTTL = DefaultAccessTTL
	}
	if refreshTTL <= 0 {
		refreshTTL = DefaultRefreshTTL
	}

	return &Signer{
		keyPair:    keyPair,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *Signer) GenerateTokens(userID, email string, role userv1.UserRole) (*userv1.JwtToken, *domain.RefreshToken, error) {
	now := time.Now()

	accessClaims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessString, err := accessToken.SignedString(s.keyPair.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshToken := &domain.RefreshToken{
		TokenID:   uuid.New().String(),
		UserID:    userID,
		Token:     uuid.New().String(),
		ExpiresAt: now.Add(s.refreshTTL),
		Revoked:   false,
		CreatedAt: now,
	}

	return &userv1.JwtToken{
		AccessToken:  accessString,
		RefreshToken: refreshToken.Token,
		ExpiresAt:    now.Add(s.accessTTL).Unix(),
		TokenType:    userv1.TokenType_TOKEN_TYPE_ACCESS,
	}, refreshToken, nil
}

func (s *Signer) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.keyPair.PublicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	if claims.Issuer != Issuer {
		return nil, errors.New("invalid token issuer")
	}

	return claims, nil
}

type Validator struct {
	publicKey *rsa.PublicKey
}

func NewValidator(publicKey *rsa.PublicKey) *Validator {
	return &Validator{publicKey: publicKey}
}

func (v *Validator) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	if claims.Issuer != Issuer {
		return nil, errors.New("invalid token issuer")
	}

	return claims, nil
}
