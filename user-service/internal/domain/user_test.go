package domain

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		cost     int
		wantErr  bool
	}{
		{"valid password", "SecurePass123!", 12, false},
		{"short password", "short", 12, false},
		{"empty password", "", 12, false},
		{"invalid cost (too low)", "pass", 4, true},
		{"invalid cost (too high)", "pass", 20, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash, err := HashPassword(tt.password, tt.cost)

			if tt.wantErr {
				if err == nil {
					t.Errorf("HashPassword() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if hash == tt.password {
				t.Error("Password should be hashed, not stored in plaintext")
			}

			if len(hash) < 60 {
				t.Errorf("Hash length too short: got %d, want >= 60", len(hash))
			}

			hash2, _ := HashPassword(tt.password, tt.cost)
			if hash == hash2 {
				t.Error("Same password should produce different hashes (salt)")
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	t.Parallel()

	password := "SecurePass123!"
	hash, _ := HashPassword(password, 12)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"correct password", password, true},
		{"wrong password", "WrongPass!", false},
		{"empty password", "", false},
		{"case sensitive", "securepass123!", false},
		{"extra space", password + " ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := CheckPassword(tt.input, hash); got != tt.expected {
				t.Errorf("CheckPassword(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		email    string
		expected bool
	}{
		{"user@example.com", true},
		{"test@domain.co.uk", true},
		{"a@b", false},
		{"invalid-email", false},
		{"", false},
		{"@example.com", false},
		{"user@", false},
		{"user @example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			t.Parallel()

			if got := ValidateEmail(tt.email); got != tt.expected {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, got, tt.expected)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		password string
		expected bool
	}{
		{"12345678", true},
		{"short", false},
		{"", false},
		{"1234567", false},
		{"123456789", true},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			t.Parallel()

			if got := ValidatePassword(tt.password); got != tt.expected {
				t.Errorf("ValidatePassword(%q) = %v, want %v", tt.password, got, tt.expected)
			}
		})
	}
}

func TestUser_ToProto_DoesNotExposePassword(t *testing.T) {
	t.Parallel()

	user := &User{
		UserID:   "user-123",
		Email:    "test@example.com",
		Username: "testuser",
		Password: "should-not-expose",
	}

	proto := user.ToProto()

	if proto.UserId != user.UserID {
		t.Errorf("UserId mismatch: got %q, want %q", proto.UserId, user.UserID)
	}
	if proto.Email != user.Email {
		t.Errorf("Email mismatch: got %q, want %q", proto.Email, user.Email)
	}
	if proto.Username != user.Username {
		t.Errorf("Username mismatch: got %q, want %q", proto.Username, user.Username)
	}

	t.Log("Password field does not exist in proto (by design) - this is correct!")
}
