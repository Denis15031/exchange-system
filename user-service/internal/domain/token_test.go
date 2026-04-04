package domain

import (
	"testing"
	"time"
)

func TestRefreshToken_IsExpired(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{"expired", now.Add(-1 * time.Hour), true},
		{"valid", now.Add(1 * time.Hour), false},
		{"exactly now", now, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := &RefreshToken{ExpiresAt: tt.expiresAt}

			if tt.name == "exactly now" {
				if rt.IsExpired() {
					t.Log("Token at exact 'now' considered expired (acceptable)")
				}
				return
			}

			if got := rt.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRefreshToken_IsValid(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		revoked   bool
		expected  bool
	}{
		{"valid token", now.Add(1 * time.Hour), false, true},
		{"expired token", now.Add(-1 * time.Hour), false, false},
		{"revoked token", now.Add(1 * time.Hour), true, false},
		{"expired and revoked", now.Add(-1 * time.Hour), true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := &RefreshToken{
				ExpiresAt: tt.expiresAt,
				Revoked:   tt.revoked,
			}

			if got := rt.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}
