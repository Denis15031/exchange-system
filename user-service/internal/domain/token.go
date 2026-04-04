package domain

import (
	"time"

	userv1 "exchange-system/proto/user/v1"
)

type RefreshToken struct {
	TokenID   string
	UserID    string
	Token     string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

func (rt *RefreshToken) IsValid() bool {
	return !rt.Revoked && !rt.IsExpired()
}

func (rt *RefreshToken) ToProto() *userv1.JwtToken {
	return &userv1.JwtToken{
		AccessToken:  "",
		RefreshToken: rt.Token,
		ExpiresAt:    rt.ExpiresAt.Unix(),
		TokenType:    userv1.TokenType_TOKEN_TYPE_REFRESH,
	}
}
