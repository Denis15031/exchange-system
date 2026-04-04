package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"exchange-system/user-service/internal/domain"
)

func TestUserRepository_CreateAndGet(t *testing.T) {
	t.Parallel()

	repo := NewUserRepository()
	ctx := context.Background()

	user := &domain.User{
		UserID:   "user-123",
		Email:    "test@example.com",
		Username: "testuser",
		Password: "hashed-password",
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, "user-123")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.UserID != user.UserID {
		t.Errorf("GetByID().UserID = %q, want %q", got.UserID, user.UserID)
	}

	gotByEmail, err := repo.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if gotByEmail.UserID != user.UserID {
		t.Errorf("GetByEmail().UserID = %q, want %q", gotByEmail.UserID, user.UserID)
	}
}

func TestUserRepository_CreateDuplicate(t *testing.T) {
	t.Parallel()

	repo := NewUserRepository()
	ctx := context.Background()

	user := &domain.User{
		UserID: "user-123",
		Email:  "test@example.com",
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("First Create() error = %v", err)
	}

	duplicate := &domain.User{
		UserID: "user-456",
		Email:  "test@example.com",
	}
	err := repo.Create(ctx, duplicate)
	if err == nil {
		t.Error("Create() should fail for duplicate email")
	}
	if err != ErrUserAlreadyExists {
		t.Errorf("Error = %v, want ErrUserAlreadyExists", err)
	}
}

func TestUserRepository_GetNotFound(t *testing.T) {
	t.Parallel()

	repo := NewUserRepository()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Error("GetByID() should return error for nonexistent user")
	}
	if err != ErrUserNotFound {
		t.Errorf("Error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepository_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	repo := NewUserRepository()
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			user := &domain.User{
				UserID:   string(rune('0' + id)),
				Email:    string(rune('a'+id)) + "@example.com",
				Username: "user" + string(rune('0'+id)),
			}
			_ = repo.Create(ctx, user)
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			email := string(rune('a'+id)) + "@example.com"
			_, _ = repo.GetByEmail(ctx, email)
		}(i)
	}

	wg.Wait()
}

func TestTokenRepository_StoreAndGet(t *testing.T) {
	t.Parallel()

	repo := NewTokenRepository()
	ctx := context.Background()

	token := &domain.RefreshToken{
		TokenID:   "tok-123",
		UserID:    "user-123",
		Token:     "refresh-token-abc",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := repo.Store(ctx, token); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, err := repo.Get(ctx, "refresh-token-abc")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.TokenID != token.TokenID {
		t.Errorf("TokenID = %q, want %q", got.TokenID, token.TokenID)
	}
}

func TestTokenRepository_Revoke(t *testing.T) {
	t.Parallel()

	repo := NewTokenRepository()
	ctx := context.Background()

	token := &domain.RefreshToken{
		Token:     "refresh-token-abc",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	repo.Store(ctx, token)

	if err := repo.Revoke(ctx, "refresh-token-abc"); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	_, err := repo.Get(ctx, "refresh-token-abc")
	if err == nil {
		t.Error("Get() should fail for revoked token")
	}
	if err != ErrTokenRevoked {
		t.Errorf("Error = %v, want ErrTokenRevoked", err)
	}
}

func TestTokenRepository_GetExpired(t *testing.T) {
	t.Parallel()

	repo := NewTokenRepository()
	ctx := context.Background()

	token := &domain.RefreshToken{
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	repo.Store(ctx, token)

	_, err := repo.Get(ctx, "expired-token")
	if err == nil {
		t.Error("Get() should fail for expired token")
	}
	if err != ErrTokenExpired {
		t.Errorf("Error = %v, want ErrTokenExpired", err)
	}
}
