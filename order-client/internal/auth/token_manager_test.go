package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestTokenManager_SaveAndLoad(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "tokens.json")

	tm := &TokenManager{
		storagePath:  tokenPath,
		accessToken:  "test-access-token",
		refreshToken: "test-refresh-token",
		expiresAt:    time.Now().Add(1 * time.Hour),
	}

	if err := tm.saveToDisk(); err != nil {
		t.Fatalf("saveToDisk() error = %v", err)
	}

	tm2 := &TokenManager{storagePath: tokenPath}
	if err := tm2.loadFromDisk(); err != nil {
		t.Fatalf("loadFromDisk() error = %v", err)
	}

	if tm2.accessToken != tm.accessToken {
		t.Errorf("accessToken = %q, want %q", tm2.accessToken, tm.accessToken)
	}
	if tm2.refreshToken != tm.refreshToken {
		t.Errorf("refreshToken = %q, want %q", tm2.refreshToken, tm.refreshToken)
	}
}

func TestTokenManager_LoadNonExistentFile(t *testing.T) {
	t.Parallel()

	tm := &TokenManager{storagePath: "/nonexistent/path/tokens.json"}

	err := tm.loadFromDisk()
	if err != nil {
		t.Errorf("loadFromDisk() should not error for nonexistent file, got %v", err)
	}
}

func TestTokenManager_FilePermissions(t *testing.T) {

	if runtime.GOOS == "windows" {
		t.Skip("File permissions test skipped on Windows (uses ACL)")
	}

	t.Parallel()

	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "tokens.json")

	tm := &TokenManager{
		storagePath:  tokenPath,
		accessToken:  "secret-token",
		refreshToken: "secret-refresh",
	}

	if err := tm.saveToDisk(); err != nil {
		t.Fatalf("saveToDisk() error = %v", err)
	}

	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("File permissions = %o, want 0600 (security risk!)", info.Mode().Perm())
	}
}

func TestTokenManager_GetAccessToken_Valid(t *testing.T) {
	t.Parallel()

	tm := &TokenManager{
		accessToken: "valid-token",
		expiresAt:   time.Now().Add(1 * time.Hour),
	}

	token, err := tm.GetAccessToken()
	if err != nil {
		t.Errorf("GetAccessToken() error = %v", err)
	}
	if token != "valid-token" {
		t.Errorf("token = %q, want %q", token, "valid-token")
	}
}

func TestTokenManager_HasToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		accessToken string
		expected    bool
	}{
		{"has token", "some-token", true},
		{"no token", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tm := &TokenManager{accessToken: tt.accessToken}
			if got := tm.HasToken(); got != tt.expected {
				t.Errorf("HasToken() = %v, want %v", got, tt.expected)
			}
		})
	}
}
