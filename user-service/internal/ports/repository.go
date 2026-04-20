package ports

import (
	"context"
	"exchange-system/user-service/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, userID string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
}

type TokenRepository interface {
	Store(ctx context.Context, token *domain.RefreshToken) error
	Get(ctx context.Context, token string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, token string) error
	RevokeAllByUser(ctx context.Context, userID string) error
}
