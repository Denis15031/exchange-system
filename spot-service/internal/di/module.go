package di

import (
	"context"
	"os"
	"strconv"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"exchange-system/shared/adapters"
	"exchange-system/shared/logger"
	sharedports "exchange-system/shared/ports"
	"exchange-system/spot-service/internal/adapters/inmemory"
	"exchange-system/spot-service/internal/cache"
	"exchange-system/spot-service/internal/config"
	"exchange-system/spot-service/internal/handler"
	spotports "exchange-system/spot-service/internal/ports"
	"exchange-system/spot-service/internal/service"
)

var SpotModule = fx.Module("spot-service",
	fx.Provide(
		config.Load,

		func(cfg *config.Config) (*zap.Logger, error) {
			logCfg := logger.Config{
				Level:  cfg.LogLevel,
				Format: cfg.LogFormat,
			}
			l, err := logger.New(logCfg)
			if err != nil {
				return nil, err
			}
			return l.Zap(), nil
		},

		func(cfg *config.Config) (sharedports.Cache, error) {
			// Если USE_REDIS не установлен или false — используем LRU
			if os.Getenv("USE_REDIS") != "true" {
				zap.L().Info("Redis disabled (USE_REDIS!=true), using LRU cache")
				lru := cache.NewLRUCache(1000)
				return &cacheWrapper{inner: lru}, nil
			}

			redisAddr := os.Getenv("REDIS_ADDR")
			if redisAddr == "" {
				redisAddr = "127.0.0.1:6379"
			}
			redisPassword := os.Getenv("REDIS_PASSWORD")
			redisDBStr := os.Getenv("REDIS_DB")
			redisDB := 0
			if redisDBStr != "" {
				if db, err := strconv.Atoi(redisDBStr); err == nil {
					redisDB = db
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			redisCache, err := adapters.NewRedisCacheWithConfig(ctx, redisAddr, redisPassword, redisDB, "spot")
			if err != nil {
				zap.L().Warn("Redis unavailable, falling back to LRU cache",
					zap.String("addr", redisAddr),
					zap.Error(err),
				)
				lru := cache.NewLRUCache(1000)
				return &cacheWrapper{inner: lru}, nil
			}

			zap.L().Info("Redis cache connected successfully", zap.String("addr", redisAddr))
			return redisCache, nil
		},

		func() spotports.MarketRepository {
			return inmemory.NewRepository()
		},

		service.NewSpotService,

		handler.NewSpotHandler,
	),
)

type cacheWrapper struct {
	inner *cache.LRUCache
}

func (w *cacheWrapper) Get(ctx context.Context, key string) ([]byte, bool, error) {
	return w.inner.Get(ctx, key)
}

func (w *cacheWrapper) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	return w.inner.Set(ctx, key, value, ttlSeconds)
}

func (w *cacheWrapper) Delete(ctx context.Context, key string) error {
	return w.inner.Delete(ctx, key)
}

func (w *cacheWrapper) Close() error {
	return w.inner.Close()
}

func (w *cacheWrapper) Stats() map[string]interface{} {
	return w.inner.Stats()
}
