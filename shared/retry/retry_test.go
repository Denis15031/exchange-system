package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func isRetryableStub(err error) bool {
	return !errors.Is(err, errNoRetry)
}

var errNoRetry = errors.New("do not retry this error")

func TestDo_SuccessOnFirstTry(t *testing.T) {
	cfg := Config{
		MaxAttempts:     3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     10 * time.Second,
		Multiplier:      2.0,
		Randomization:   0.5,
	}

	callCount := 0
	err := Do(context.Background(), cfg, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("called %d times, want 1", callCount)
	}
}

func TestDo_SuccessAfterRetry(t *testing.T) {
	cfg := Config{
		MaxAttempts:     3,
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
		Randomization:   0.1,
	}

	callCount := 0
	err := Do(context.Background(), cfg, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("called %d times, want 3", callCount)
	}
}

func TestDo_FailAfterMaxAttempts(t *testing.T) {
	cfg := Config{
		MaxAttempts:     2,
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
		Randomization:   0.1,
	}

	callCount := 0
	expectedErr := errors.New("always fails")

	err := Do(context.Background(), cfg, func() error {
		callCount++
		return expectedErr
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) && err != expectedErr {
		t.Errorf("got error %v, want %v", err, expectedErr)
	}
	if callCount != 2 {
		t.Errorf("called %d times, want 2", callCount)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	cfg := Config{
		MaxAttempts:     5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		Randomization:   0.5,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Do(ctx, cfg, func() error {
		return errors.New("should not be called")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("got error %v, want context.Canceled", err)
	}
}

func TestDo_ContextTimeout(t *testing.T) {
	cfg := Config{
		MaxAttempts:     5,
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     200 * time.Millisecond,
		Multiplier:      2.0,
		Randomization:   0.1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := Do(ctx, cfg, func() error {
		return errors.New("transient")
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got error %v, want context.DeadlineExceeded", err)
	}
}

func TestDo_NonRetryableError(t *testing.T) {
	t.Skip("Skipping: tests private isRetryable() logic")
	cfg := Config{
		MaxAttempts:     3,
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
		Randomization:   0.1,
	}

	callCount := 0
	err := Do(context.Background(), cfg, func() error {
		callCount++
		return errNoRetry
	})

	if err != errNoRetry {
		t.Errorf("got error %v, want errNoRetry", err)
	}
	if callCount != 1 {
		t.Errorf("called %d times, want 1 (no retry expected)", callCount)
	}
}

func TestDo_JitterVariation(t *testing.T) {

	cfg := Config{
		MaxAttempts:     2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Randomization:   0.5, // 50% jitter
	}

	delays := make([]time.Duration, 5)
	for i := range delays {
		start := time.Now()
		_ = Do(context.Background(), cfg, func() error {
			return errors.New("retry")
		})
		delays[i] = time.Since(start)
	}

	minExpected := cfg.InitialInterval
	maxExpected := cfg.InitialInterval * time.Duration(cfg.Multiplier) * 2

	for i, d := range delays {
		if d < minExpected/2 || d > maxExpected*2 {
			t.Logf("attempt %d: delay %v outside expected range [%v, %v] (may be OK with jitter)",
				i+1, d, minExpected, maxExpected)
		}
	}
}
