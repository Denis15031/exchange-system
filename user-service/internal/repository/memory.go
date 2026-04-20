package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"exchange-system/user-service/internal/domain"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrTokenNotFound     = errors.New("refresh token not found")
	ErrTokenRevoked      = errors.New("refresh token revoked")
	ErrTokenExpired      = errors.New("refresh token expired")
)

type UserRepository struct {
	users      map[string]*domain.User
	emailIndex map[string]string
	mu         sync.RWMutex
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users:      make(map[string]*domain.User),
		emailIndex: make(map[string]string),
	}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.emailIndex[user.Email]; exists {
		return ErrUserAlreadyExists
	}
	if _, exists := r.users[user.UserID]; exists {
		return ErrUserAlreadyExists
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	r.users[user.UserID] = user
	r.emailIndex[user.Email] = user.UserID
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}

	userCopy := *user
	return &userCopy, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, exists := r.emailIndex[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	user, exists := r.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}

	userCopy := *user
	return &userCopy, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.UserID]; !exists {
		return ErrUserNotFound
	}

	user.UpdatedAt = time.Now()
	r.users[user.UserID] = user
	return nil
}

type TokenRepository struct {
	tokens     map[string]*domain.RefreshToken
	userTokens map[string][]string
	mu         sync.RWMutex
}

func NewTokenRepository() *TokenRepository {
	return &TokenRepository{
		tokens:     make(map[string]*domain.RefreshToken),
		userTokens: make(map[string][]string),
	}
}

func (r *TokenRepository) Store(ctx context.Context, token *domain.RefreshToken) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.tokens[token.Token] = token
	r.userTokens[token.UserID] = append(r.userTokens[token.UserID], token.Token)
	return nil
}

func (r *TokenRepository) Get(ctx context.Context, token string) (*domain.RefreshToken, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	rt, exists := r.tokens[token]
	if !exists {
		return nil, ErrTokenNotFound
	}
	if rt.Revoked {
		return nil, ErrTokenRevoked
	}
	if rt.IsExpired() {
		return nil, ErrTokenExpired
	}

	tokenCopy := *rt
	return &tokenCopy, nil
}

func (r *TokenRepository) Revoke(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	rt, exists := r.tokens[token]
	if !exists {
		return ErrTokenNotFound
	}
	rt.Revoked = true
	return nil
}

func (r *TokenRepository) RevokeAllByUser(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	tokenIDs, exists := r.userTokens[userID]
	if !exists {
		return nil
	}

	for _, tokenID := range tokenIDs {
		if rt, ok := r.tokens[tokenID]; ok {
			rt.Revoked = true
		}
	}
	return nil
}
