package jwtvalidator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	userv1 "exchange-system/proto/user/v1"

	"go.uber.org/zap"

	"github.com/golang-jwt/jwt/v5"
)

type claimsKey struct{}

type Claims struct {
	jwt.RegisteredClaims
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type Config struct {
	SecretOrPublicKey string        `envconfig:"JWT_SECRET_OR_PUBLIC_KEY" required:"true"`
	Issuer            string        `envconfig:"JWT_ISSUER" default:"exchange-system"`
	Audience          string        `envconfig:"JWT_AUDIENCE" default:"exchange-api"`
	Algorithm         string        `envconfig:"JWT_ALGORITHM" default:"HS256"`
	ClockSkew         time.Duration `envconfig:"JWT_CLOCK_SKEW" default:"30s"`
}

type Validator struct {
	config    *Config
	publicKey interface{}
	secretKey []byte
	logger    *zap.Logger
}

func NewValidator(cfg *Config, logger *zap.Logger) (*Validator, error) {
	v := &Validator{
		config: cfg,
		logger: logger,
	}

	switch strings.ToUpper(cfg.Algorithm) {
	case "HS256", "HS384", "HS512":
		v.secretKey = []byte(cfg.SecretOrPublicKey)
	case "RS256", "RS384", "RS512":
		pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cfg.SecretOrPublicKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
		}
		v.publicKey = pubKey
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", cfg.Algorithm)
	}

	return v, nil
}

func (v *Validator) ValidateAndExtract(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if !isValidAlgorithm(token.Method.Alg(), v.config.Algorithm) {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		if v.publicKey != nil {
			return v.publicKey, nil
		}
		return v.secretKey, nil
	},
		jwt.WithIssuer(v.config.Issuer),
		jwt.WithAudience(v.config.Audience),
		jwt.WithValidMethods([]string{v.config.Algorithm}),
		jwt.WithLeeway(v.config.ClockSkew),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}

		if strings.Contains(err.Error(), "issuer") || strings.Contains(err.Error(), "audience") {
			return nil, ErrTokenInvalid
		}
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(*Claims)
	return claims, ok
}

func (c *Claims) ToUserRole() userv1.UserRole {
	switch strings.ToUpper(c.Role) {
	case "USER":
		return userv1.UserRole_USER_ROLE_USER
	case "PREMIUM":
		return userv1.UserRole_USER_ROLE_PREMIUM
	case "ADMIN":
		return userv1.UserRole_USER_ROLE_ADMIN
	default:
		return userv1.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func isValidAlgorithm(method, expected string) bool {
	return strings.EqualFold(method, expected)
}

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("invalid token")
)
