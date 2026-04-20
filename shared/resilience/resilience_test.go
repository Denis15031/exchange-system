package resilience

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"exchange-system/shared/ports"
)

func newTestConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:        name,
		Timeout:     100 * time.Millisecond,
		MaxRequests: 1,
		Interval:    50 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {

			return counts.TotalFailures >= 2
		},
		OnStateChange: nil,
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig("test-breaker")
	cb := NewCircuitBreaker(cfg)

	if cb == nil {
		t.Fatal("NewCircuitBreaker returned nil")
	}
	if cb.Name() != "test-breaker" {
		t.Errorf("Name() = %s, want test-breaker", cb.Name())
	}
	if cb.IsOpen() {
		t.Error("New circuit breaker should start in Closed state")
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultCircuitBreakerConfig("default-test")

	if cfg.Name != "default-test" {
		t.Errorf("Name = %s, want default-test", cfg.Name)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
	if cfg.MaxRequests != 5 {
		t.Errorf("MaxRequests = %d, want 5", cfg.MaxRequests)
	}

	if cfg.ReadyToTrip == nil {
		t.Error("ReadyToTrip should not be nil in default config")
	}
}

func TestExecute_Success(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(newTestConfig("success-test"))
	ctx := context.Background()

	result, err := Execute(ctx, cb, func() (string, error) {
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "ok" {
		t.Errorf("Result = %q, want ok", result)
	}
}

func TestExecute_Failure(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(newTestConfig("failure-test"))
	ctx := context.Background()
	expectedErr := errors.New("simulated failure")

	_, err := Execute(ctx, cb, func() (string, error) {
		return "", expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected %v, got %v", expectedErr, err)
	}
}

func TestExecute_ContextCancelled(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(newTestConfig("context-test"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Execute(ctx, cb, func() (string, error) {

		time.Sleep(100 * time.Millisecond)
		return "should-not-reach", nil
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestExecute_ContextTimeout(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(newTestConfig("timeout-test"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := Execute(ctx, cb, func() (string, error) {
		time.Sleep(100 * time.Millisecond) // Дольше чем таймаут контекста
		return "timeout", nil
	})

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestCircuitBreaker_TripsOnFailures(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig("trip-test")
	cb := NewCircuitBreaker(cfg).(*circuitBreaker)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		_, _ = Execute(ctx, cb, func() (string, error) {
			return "", errors.New("failure")
		})
	}

	if !cb.IsOpen() {
		t.Error("Circuit breaker should be OPEN after 2 failures")
	}

	_, err := Execute(ctx, cb, func() (string, error) {
		return "should-not-run", nil
	})
	if err == nil {
		t.Error("Execute should fail when circuit is OPEN")
	}
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig("halfopen-test")
	cb := NewCircuitBreaker(cfg).(*circuitBreaker)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		_, _ = Execute(ctx, cb, func() (string, error) {
			return "", errors.New("failure")
		})
	}
	if !cb.IsOpen() {
		t.Fatal("Circuit should be OPEN")
	}

	time.Sleep(150 * time.Millisecond)

	_, err := Execute(ctx, cb, func() (string, error) {
		return "recovered", nil
	})
	if err != nil {
		t.Fatalf("Execute in Half-Open should succeed: %v", err)
	}

	if cb.IsOpen() {
		t.Error("Circuit breaker should be CLOSED after successful Half-Open request")
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig("halfopen-fail-test")
	cb := NewCircuitBreaker(cfg).(*circuitBreaker)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		_, _ = Execute(ctx, cb, func() (string, error) { return "", errors.New("fail") })
	}

	time.Sleep(150 * time.Millisecond)

	_, _ = Execute(ctx, cb, func() (string, error) { return "", errors.New("fail-in-halfopen") })

	if !cb.IsOpen() {
		t.Error("Circuit should re-OPEN after failure in Half-Open state")
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig("stats-test")
	cb := NewCircuitBreaker(cfg)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _ = Execute(ctx, cb, func() (string, error) {
			if i%2 == 0 {
				return "ok", nil
			}
			return "", errors.New("error")
		})
	}

	stats := cb.Stats()

	if stats.Name != "stats-test" {
		t.Errorf("Stats.Name = %s, want stats-test", stats.Name)
	}
	if stats.TotalRequests < 3 {
		t.Errorf("Stats.TotalRequests = %d, want >= 3", stats.TotalRequests)
	}

	_ = stats.State
}

func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(newTestConfig("helper-test"))

	state := GetState(cb)
	if state == "unknown" {
		t.Error("GetState should return valid state string")
	}

	counts := GetCounts(cb)

	_ = counts

	isOpen := IsOpen(cb)
	if isOpen != cb.IsOpen() {
		t.Error("IsOpen helper should match method result")
	}
}

func TestOnStateChangeCallback(t *testing.T) {
	t.Parallel()

	var stateChanges []struct {
		from, to string
	}
	var mu sync.Mutex

	cfg := newTestConfig("callback-test")
	cfg.OnStateChange = func(name string, from, to State) {
		mu.Lock()
		defer mu.Unlock()
		stateChanges = append(stateChanges, struct{ from, to string }{
			from: from.String(),
			to:   to.String(),
		})
	}

	cb := NewCircuitBreaker(cfg).(*circuitBreaker)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _ = Execute(ctx, cb, func() (string, error) {
			return "", errors.New("fail")
		})
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(stateChanges) == 0 {
		t.Log("OnStateChange callback may not have fired yet (timing dependent)")
	} else {
		t.Logf("State changes recorded: %v", stateChanges)
	}
	mu.Unlock()
}

func TestCircuitBreaker_ConcurrentExecute(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig("concurrent-test")
	cfg.ReadyToTrip = func(counts Counts) bool {
		return counts.TotalFailures >= 10
	}
	cb := NewCircuitBreaker(cfg)
	ctx := context.Background()

	var wg sync.WaitGroup
	var successCount, failureCount int64

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Execute(ctx, cb, func() (string, error) {
				return "ok", nil
			})
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failureCount, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("Concurrent: %d success, %d rejected", successCount, failureCount)

}

func TestCircuitBreaker_ConcurrentStateChanges(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(newTestConfig("concurrent-state-test"))
	ctx := context.Background()

	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			_, _ = Execute(ctx, cb, func() (string, error) {
				if n%3 == 0 {
					return "", errors.New("fail")
				}
				return "ok", nil
			})
		}(i)
		go func() {
			defer wg.Done()
			_ = cb.Stats()
			_ = GetState(cb)
		}()
	}

	wg.Wait()

}

func TestExecute_TypeSafety(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(newTestConfig("type-test"))
	ctx := context.Background()

	intResult, err := Execute(ctx, cb, func() (int, error) {
		return 42, nil
	})
	if err != nil || intResult != 42 {
		t.Errorf("int Execute failed: result=%d, err=%v", intResult, err)
	}

	type MyStruct struct{ Value string }
	structResult, err := Execute(ctx, cb, func() (MyStruct, error) {
		return MyStruct{Value: "test"}, nil
	})
	if err != nil || structResult.Value != "test" {
		t.Errorf("struct Execute failed: result=%+v, err=%v", structResult, err)
	}

	_, err = Execute(ctx, cb, func() (string, error) {
		return "", errors.New("expected error")
	})
	if err == nil {
		t.Error("Expected error from Execute")
	}
}

func TestExecute_ZeroValueReturn(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(newTestConfig("zero-test"))
	ctx := context.Background()

	result, err := Execute(ctx, cb, func() (string, error) {
		return "", nil // zero value for string
	})

	if err != nil {
		t.Fatalf("Execute with zero value error = %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestCircuitBreaker_ImplementsPort(t *testing.T) {
	t.Parallel()

	var _ ports.CircuitBreaker = (*circuitBreaker)(nil)

	cb := NewCircuitBreaker(newTestConfig("port-test"))
	if cb == nil {
		t.Fatal("Failed to create CircuitBreaker")
	}

	_ = cb.Name()
	_ = cb.IsOpen()
	_ = cb.Stats()
}
