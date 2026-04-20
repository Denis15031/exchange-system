package adapters

import (
	"context"
	"fmt"
	"time"

	"exchange-system/shared/ports"

	"github.com/redis/go-redis/v9"
)

var _ ports.Cache = (*RedisCache)(nil)

type RedisCache struct {
	client *redis.Client
	prefix string
}

func NewRedisCache(addr, password string, db int, prefix string) (ports.Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisCache{client: client, prefix: prefix}, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := r.client.Get(ctx, r.key(key)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	ttl := time.Duration(ttlSeconds) * time.Second
	return r.client.Set(ctx, r.key(key), value, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.key(key)).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func (r *RedisCache) Stats() map[string]interface{} {
	return map[string]interface{}{
		"type":   "redis",
		"addr":   r.client.Options().Addr,
		"prefix": r.prefix,
	}
}

func (r *RedisCache) key(k string) string {
	if r.prefix == "" {
		return k
	}
	return r.prefix + ":" + k
}

func NewRedisCacheWithConfig(ctx context.Context, addr, password string, db int, prefix string) (ports.Cache, error) {
	opts := &redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		PoolSize:        25,
		MinIdleConns:    5,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
	}

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisCache{client: client, prefix: prefix}, nil
}
