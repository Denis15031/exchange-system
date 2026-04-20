package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"exchange-system/shared/ports"

	"go.uber.org/zap"
)

type HealthHandler struct {
	logger *zap.Logger
	cache  ports.Cache // Для проверки кэша
}

func NewHealthHandler(logger *zap.Logger, cache ports.Cache) *HealthHandler {
	return &HealthHandler{logger: logger, cache: cache}
}

type healthResponse struct {
	Status  string          `json:"status"`
	Version string          `json:"version"`
	Checks  map[string]bool `json:"checks,omitempty"`
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]bool)

	if h.cache != nil {
		_, _, err := h.cache.Get(ctx, "health:ping")
		checks["cache"] = (err == nil)
		if err != nil {
			h.logger.Warn("Health check: cache failed", zap.Error(err))
		}
	}

	status := "ok"
	statusCode := http.StatusOK
	if !checks["cache"] {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := healthResponse{
		Status:  status,
		Version: "1.0.0",
		Checks:  checks,
	}
	_ = json.NewEncoder(w).Encode(resp)

	h.logger.Debug("Health check completed",
		zap.String("status", status),
		zap.Bool("cache_ok", checks["cache"]),
	)
}

func (h *HealthHandler) ReadyHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	ready := true
	checks := make(map[string]bool)

	if h.cache != nil {
		_, _, err := h.cache.Get(ctx, "ready:ping")
		checks["cache"] = (err == nil)
		if !checks["cache"] {
			ready = false
			h.logger.Warn("Readiness check: cache not ready", zap.Error(err))
		}
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ready":  ready,
		"checks": checks,
	})
}
