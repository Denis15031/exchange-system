package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimitConfig struct {
	RequestsPerSecond float64
	MaxBurst          int
	RoleLimits        map[string]float64
	BytesPerSecond    float64
}

func DefaultRateLimitConfig() RateLimitConfig {
	roleLimits := make(map[string]float64, len(map[string]float64{
		"ADMIN":     10000,
		"MODERATOR": 7500,
		"PREMIUM":   6000,
		"USER":      5000,
	}))
	for k, v := range map[string]float64{
		"ADMIN":     10000,
		"MODERATOR": 7500,
		"PREMIUM":   6000,
		"USER":      5000,
	} {
		roleLimits[k] = v
	}

	return RateLimitConfig{
		RequestsPerSecond: 5000,
		MaxBurst:          1000,
		RoleLimits:        roleLimits,
		BytesPerSecond:    10 * 1024 * 1024,
	}
}

type RateLimiter struct {
	config     RateLimitConfig
	serviceLim *rate.Limiter
	bytesLim   *rate.Limiter
	roleLims   map[string]*rate.Limiter
	mu         sync.RWMutex
}

func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:     config,
		serviceLim: rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.MaxBurst),
		bytesLim:   rate.NewLimiter(rate.Limit(config.BytesPerSecond), config.MaxBurst*1024),
		roleLims:   make(map[string]*rate.Limiter, len(config.RoleLimits)),
	}

	for role, rps := range config.RoleLimits {
		rl.roleLims[role] = rate.NewLimiter(rate.Limit(rps), config.MaxBurst)
	}

	return rl
}

func (rl *RateLimiter) Allow() bool {
	return rl.serviceLim.Allow()
}

func (rl *RateLimiter) AllowN(n int) bool {
	return rl.serviceLim.AllowN(time.Now(), n)
}

func (rl *RateLimiter) AllowRole(role string) bool {
	rl.mu.RLock()
	limiter, exists := rl.roleLims[role]
	if !exists {
		limiter = rl.roleLims["USER"]
	}
	rl.mu.RUnlock()

	return limiter.Allow()
}

func (rl *RateLimiter) AllowBytes(bytes int) bool {
	return rl.bytesLim.AllowN(time.Now(), bytes)
}

func (rl *RateLimiter) Wait() error {
	return rl.serviceLim.Wait(context.Background())
}

func (rl *RateLimiter) WaitRole(role string) error {
	rl.mu.RLock()
	limiter, exists := rl.roleLims[role]
	if !exists {
		limiter = rl.roleLims["USER"]
	}
	rl.mu.RUnlock()

	return limiter.Wait(context.Background())
}

func (rl *RateLimiter) UpdateRoleLimit(role string, rps float64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.roleLims[role] = rate.NewLimiter(rate.Limit(rps), rl.config.MaxBurst)
}

func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	roleLimits := make(map[string]float64, len(rl.config.RoleLimits))
	for k, v := range rl.config.RoleLimits {
		roleLimits[k] = v
	}

	return map[string]interface{}{
		"service_limit": rl.config.RequestsPerSecond,
		"bytes_limit":   rl.config.BytesPerSecond,
		"role_limits":   roleLimits,
	}
}

func (rl *RateLimiter) GetConfig() RateLimitConfig {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	roleLimits := make(map[string]float64, len(rl.config.RoleLimits))
	for k, v := range rl.config.RoleLimits {
		roleLimits[k] = v
	}

	return RateLimitConfig{
		RequestsPerSecond: rl.config.RequestsPerSecond,
		MaxBurst:          rl.config.MaxBurst,
		RoleLimits:        roleLimits,
		BytesPerSecond:    rl.config.BytesPerSecond,
	}
}

var (
	globalLimiter *RateLimiter
	once          sync.Once
)

func InitGlobal(config RateLimitConfig) {
	once.Do(func() {
		globalLimiter = NewRateLimiter(config)
	})
}

func GetGlobal() *RateLimiter {
	return globalLimiter
}
