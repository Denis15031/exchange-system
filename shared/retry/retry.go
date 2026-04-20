package retry

import (
	"context"
	"math/rand"
	"time"
)

type Config struct {
	MaxAttempts     int           `envconfig:"RETRY_MAX_ATTEMPTS" default:"3"`
	InitialInterval time.Duration `envconfig:"RETRY_INITIAL_INTERVAL" default:"100ms"`
	MaxInterval     time.Duration `envconfig:"RETRY_MAX_INTERVAL" default:"10s"`
	Multiplier      float64       `envconfig:"RETRY_MULTIPLIER" default:"2.0"`
	Randomization   float64       `envconfig:"RETRY_RANDOMIZATION" default:"0.5"`
}

func Do(ctx context.Context, cfg Config, fn func() error) error {
	var lastErr error
	interval := cfg.InitialInterval

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {

		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil // Успех
		}

		if !isRetryable(lastErr) {
			return lastErr
		}

		if attempt == cfg.MaxAttempts {
			return lastErr
		}

		delay := calculateDelay(interval, cfg.Multiplier, cfg.Randomization)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:

		}

		interval = time.Duration(float64(interval) * cfg.Multiplier)
		if interval > cfg.MaxInterval {
			interval = cfg.MaxInterval
		}
	}

	return lastErr
}

func DoWithResult[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	interval := cfg.InitialInterval

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isRetryable(err) {
			return zero, err
		}

		if attempt == cfg.MaxAttempts {
			return zero, err
		}

		delay := calculateDelay(interval, cfg.Multiplier, cfg.Randomization)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}

		interval = time.Duration(float64(interval) * cfg.Multiplier)
		if interval > cfg.MaxInterval {
			interval = cfg.MaxInterval
		}
	}

	return zero, lastErr
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	return true
}

func calculateDelay(base time.Duration, multiplier, randomization float64) time.Duration {
	delay := float64(base) * multiplier

	if randomization > 0 {
		jitter := (rand.Float64()*2 - 1) * randomization
		delay *= (1 + jitter)
	}

	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}
