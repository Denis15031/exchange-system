package repository

import (
	"context"
	"errors"
	"fmt"
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

type IdempotencyEntry struct {
	Response  interface{}
	ExpiresAt time.Time
}
type IdempotencyStore struct {
	mu    sync.RWMutex
	cache map[string]*IdempotencyEntry
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{
		cache: make(map[string]*IdempotencyEntry),
	}
}

func (s *IdempotencyStore) Get(ctx context.Context, key string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.cache[key]
	if !ok {
		return nil, fmt.Errorf("idempotency key not found: %s", key)
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, fmt.Errorf("idempotency key expired: %s", key)
	}

	return entry.Response, nil
}

func (s *IdempotencyStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[key] = &IdempotencyEntry{
		Response:  value,
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (s *IdempotencyStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, key)
	return nil
}

func NewUserRepository() *UserRepository {
	repo := &UserRepository{
		users:      make(map[string]*domain.User),
		emailIndex: make(map[string]string),
	}
	return repo
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
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
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tokens[token.Token] = token
	r.userTokens[token.UserID] = append(r.userTokens[token.UserID], token.Token)

	return nil
}

func (r *TokenRepository) Get(ctx context.Context, token string) (*domain.RefreshToken, error) {
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
