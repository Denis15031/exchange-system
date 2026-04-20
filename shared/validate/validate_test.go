package validate

import (
	"strings"
	"testing"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid", "user@example.com", false},
		{"valid with plus", "user+tag@example.co.uk", false},
		{"empty", "", true},
		{"no @", "userexample.com", true},
		{"no domain", "user@", true},
		{"no local", "@example.com", true},
		{"invalid chars", "user name@example.com", true},
		{"too long", strings.Repeat("a", 260) + "@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Email(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("Email(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		minLen   int
		maxLen   int
		wantErr  bool
	}{
		{"valid", "SecurePass1!", 8, 128, false},
		{"min length", "Pass1!", 6, 128, false},
		{"too short", "Pass1", 8, 128, true},
		{"too long", strings.Repeat("a", 129), 8, 128, true},
		{"no letter", "12345678!", 8, 128, true},
		{"no digit/special", "Password", 8, 128, true},
		{"empty", "", 8, 128, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Password(tt.password, tt.minLen, tt.maxLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("Password() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUUID(t *testing.T) {
	tests := []struct {
		name    string
		uuid    string
		wantErr bool
	}{
		{"valid v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"empty", "", true},
		{"wrong format", "550e8400-e29b-41d4-a716", true},
		{"invalid chars", "550e8400-e29b-41d4-a716-44665544000g", true},
		{"v1 uuid (wrong version)", "550e8400-e29b-11d4-a716-446655440000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UUID(tt.uuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("UUID(%q) error = %v, wantErr %v", tt.uuid, err, tt.wantErr)
			}
		})
	}
}
