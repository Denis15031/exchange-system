package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

type mockCheck struct {
	name  string
	err   error
	delay time.Duration
}

func (m *mockCheck) Name() string { return m.name }
func (m *mockCheck) Check(ctx context.Context) error {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
			return m.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return m.err
}

func TestHandler_ServeHTTP_AllHealthy(t *testing.T) {
	h := NewHandler(zap.NewNop(), "v1.0", "abc123")
	h.AddCheck(&mockCheck{name: "db", err: nil})
	h.AddCheck(&mockCheck{name: "cache", err: nil})

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var resp Status
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if resp.Checks["db"] != "ok" {
		t.Error("db check should be ok")
	}
}

func TestHandler_ServeHTTP_Degraded(t *testing.T) {
	h := NewHandler(zap.NewNop(), "v1.0", "abc123")
	h.AddCheck(&mockCheck{name: "db", err: nil})
	h.AddCheck(&mockCheck{name: "cache", err: errors.New("connection refused")})

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("got status %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp Status
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if resp.Status != "degraded" {
		t.Errorf("status = %q, want %q", resp.Status, "degraded")
	}
	if resp.Checks["cache"] != "error" {
		t.Error("cache check should be error")
	}
}

func TestHandler_ServeHTTP_ContextTimeout(t *testing.T) {
	h := NewHandler(zap.NewNop(), "v1.0", "abc123")
	h.AddCheck(&mockCheck{name: "slow", err: nil, delay: 10 * time.Second})

	req := httptest.NewRequest("GET", "/healthz", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 100*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusOK {
		t.Errorf("unexpected status: %d", w.Code)
	}
}

func TestHandler_VersionInfo(t *testing.T) {
	h := NewHandler(zap.NewNop(), "v2.1.0", "def456")
	h.AddCheck(&mockCheck{name: "db", err: nil})

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Status  string            `json:"status"`
		Version string            `json:"version"`
		Commit  string            `json:"commit"`
		Checks  map[string]string `json:"checks"`
	}

	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response JSON: %v", err)
	}

	if resp.Version != "v2.1.0" {
		t.Errorf("version = %q, want %q", resp.Version, "v2.1.0")
	}
	if resp.Commit != "def456" {
		t.Errorf("commit = %q, want %q", resp.Commit, "def456")
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
