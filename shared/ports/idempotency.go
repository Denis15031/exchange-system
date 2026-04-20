package ports

import (
	"context"
)

type IdempotencyStore interface {
	CheckAndSet(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
	Close()
	Stats() (int, int)
}
