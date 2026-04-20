package adapters

import (
	"context"

	"exchange-system/shared/ports"
)

type RedisIdempotencyStore struct {
	cache ports.Cache
}

func NewRedisStore(cache ports.Cache) ports.IdempotencyStore {
	return &RedisIdempotencyStore{cache: cache}
}

func (s *RedisIdempotencyStore) CheckAndSet(ctx context.Context, key string) (bool, error) {
	_, found, err := s.cache.Get(ctx, key)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}
	return false, s.cache.Set(ctx, key, []byte("1"), 86400)
}

func (s *RedisIdempotencyStore) Delete(ctx context.Context, key string) error {
	return s.cache.Delete(ctx, key)
}

func (s *RedisIdempotencyStore) Close() {
	_ = s.cache.Close()
}

func (s *RedisIdempotencyStore) Stats() (int, int) {
	stats := s.cache.Stats()
	total, _ := stats["total_keys"].(int)
	max, _ := stats["max_size"].(int)
	return total, max
}
