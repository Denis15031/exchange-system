package idempotency

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestStore_CheckAndSet(t *testing.T) {
	t.Parallel()

	store := NewStore(
		1*time.Hour,
		1*time.Hour,
		1000,
		zaptest.NewLogger(t),
	)
	defer store.Close()

	ctx := context.Background()
	key := "order-123"

	isDuplicate, err := store.CheckAndSet(ctx, key)
	if err != nil {
		t.Fatalf("CheckAndSet() unexpected error: %v", err)
	}
	if isDuplicate {
		t.Error("expected isDuplicate=false for new key")
	}

	isDuplicate, err = store.CheckAndSet(ctx, key)
	if err != nil {
		t.Fatalf("CheckAndSet() unexpected error: %v", err)
	}
	if !isDuplicate {
		t.Error("expected isDuplicate=true for existing key")
	}
}

func TestStore_TTL(t *testing.T) {
	t.Parallel()

	store := NewStore(
		100*time.Millisecond,
		50*time.Millisecond,
		1000,
		zaptest.NewLogger(t),
	)
	defer store.Close()

	ctx := context.Background()
	key := "order-ttl"

	store.CheckAndSet(ctx, key)

	time.Sleep(120 * time.Millisecond)

	isDuplicate, err := store.CheckAndSet(ctx, key)
	if err != nil {
		t.Fatalf("CheckAndSet() unexpected error: %v", err)
	}
	if isDuplicate {
		t.Error("expected key to expire, so isDuplicate should be false")
	}
}

func TestStore_MaxKeys(t *testing.T) {
	t.Parallel()

	store := NewStore(
		1*time.Hour,
		1*time.Hour,
		2,
		zaptest.NewLogger(t),
	)
	defer store.Close()

	ctx := context.Background()

	store.CheckAndSet(ctx, "key-1")
	store.CheckAndSet(ctx, "key-2")

	total, _ := store.Stats()
	if total != 2 {
		t.Errorf("expected 2 keys in store, got %d", total)
	}

	store.CheckAndSet(ctx, "key-3")

	total, _ = store.Stats()
	if total != 2 {
		t.Errorf("expected store size to remain 2 after eviction, got %d", total)
	}
}

func TestStore_Stats(t *testing.T) {
	t.Parallel()

	store := NewStore(1*time.Hour, 1*time.Hour, 10, zaptest.NewLogger(t))
	defer store.Close()

	ctx := context.Background()
	store.CheckAndSet(ctx, "a")
	store.CheckAndSet(ctx, "b")
	store.CheckAndSet(ctx, "c")

	total, max := store.Stats()

	if total != 3 {
		t.Errorf("expected total keys 3, got %d", total)
	}
	if max != 10 {
		t.Errorf("expected max keys 10, got %d", max)
	}
}

func TestStore_ContextCancellation(t *testing.T) {
	t.Parallel()

	store := NewStore(1*time.Hour, 1*time.Hour, 10, zaptest.NewLogger(t))
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.CheckAndSet(ctx, "key-cancel")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
