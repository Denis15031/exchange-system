package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Check interface {
	Name() string
	Check(ctx context.Context) error
}

type Status struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
	Version   string            `json:"version,omitempty"`
	Commit    string            `json:"commit,omitempty"`
}

type Handler struct {
	logger  *zap.Logger
	checks  []Check
	version string
	commit  string
	mu      sync.RWMutex
}

func NewHandler(logger *zap.Logger, version, commit string) *Handler {
	return &Handler{
		logger:  logger,
		checks:  make([]Check, 0),
		version: version,
		commit:  commit,
	}
}

func (h *Handler) AddCheck(check Check) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks = append(h.checks, check)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := h.evaluate(ctx)
	w.Header().Set("Content-Type", "application/json")

	if status.Status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(status)
}

func (h *Handler) evaluate(ctx context.Context) Status {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := Status{
		Status:    "ok",
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
		Version:   h.version,
		Commit:    h.commit,
	}

	degraded := false

	for _, check := range h.checks {
		err := check.Check(ctx)
		if err != nil {
			result.Checks[check.Name()] = "error"
			h.logger.Warn("health check failed",
				zap.String("check", check.Name()),
				zap.Error(err),
			)
			degraded = true
		} else {
			result.Checks[check.Name()] = "ok"
		}
	}

	if degraded {
		result.Status = "degraded"
	}

	return result
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", h.ServeHTTP)
	mux.HandleFunc("/ready", h.ServeHTTP)
}

type DBCheck struct {
	name string
	ping func(context.Context) error
}

func NewDBCheck(name string, ping func(context.Context) error) Check {
	return &DBCheck{name: name, ping: ping}
}
func (c *DBCheck) Name() string                    { return c.name }
func (c *DBCheck) Check(ctx context.Context) error { return c.ping(ctx) }

type RedisCheck struct {
	name string
	ping func(context.Context) error
}

func NewRedisCheck(name string, ping func(context.Context) error) Check {
	return &RedisCheck{name: name, ping: ping}
}
func (c *RedisCheck) Name() string                    { return c.name }
func (c *RedisCheck) Check(ctx context.Context) error { return c.ping(ctx) }
