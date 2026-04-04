package jwtvalidator

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	userv1 "exchange-system/proto/user/v1"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer = "exchange-system-user-service"
)

type Claims struct {
	UserID string          `json:"user_id"`
	Email  string          `json:"email"`
	Role   userv1.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type Validator struct {
	publicKey *rsa.PublicKey
}

func NewValidator(publicKey *rsa.PublicKey) *Validator {
	return &Validator{publicKey: publicKey}
}

func LoadPublicKeyFromFile(publicKeyPath string) (*rsa.PublicKey, error) {
	pubData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	pubBlock, _ := pem.Decode(pubData)
	if pubBlock == nil {
		return nil, errors.New("failed to decode public key PEM")
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}

	return rsaPubKey, nil
}

func (v *Validator) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
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

	// Проверка issuer
	if claims.Issuer != Issuer {
		return nil, errors.New("invalid token issuer")
	}

	return claims, nil
}

func (v *Validator) ValidateAndGetUserID(tokenString string) (string, error) {
	claims, err := v.Validate(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (v *Validator) ValidateAndGetRole(tokenString string) (userv1.UserRole, error) {
	claims, err := v.Validate(tokenString)
	if err != nil {
		return userv1.UserRole_USER_ROLE_UNSPECIFIED, err
	}
	return claims.Role, nil
}

func (v *Validator) HasRole(tokenString string, requiredRole userv1.UserRole) (bool, error) {
	role, err := v.ValidateAndGetRole(tokenString)
	if err != nil {
		return false, err
	}
	return role == requiredRole, nil
}

func (v *Validator) GetTokenExpiry(tokenString string) (time.Time, error) {
	claims, err := v.Validate(tokenString)
	if err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt.Time, nil
}
