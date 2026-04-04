package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type IdempotencyConfig struct {
	KeyTTL         time.Duration
	EnableCaching  bool
	EnabledMethods map[string]bool
}

func DefaultIdempotencyConfig() IdempotencyConfig {
	return IdempotencyConfig{
		KeyTTL:        24 * time.Hour,
		EnableCaching: true,
		EnabledMethods: map[string]bool{
			"/order.v1.OrderService/CreateOrder": true,
		},
	}
}

func validateConfig(cfg IdempotencyConfig) IdempotencyConfig {
	enabledMethods := make(map[string]bool, len(cfg.EnabledMethods))
	for k, v := range cfg.EnabledMethods {
		enabledMethods[k] = v
	}

	return IdempotencyConfig{
		KeyTTL:         cfg.KeyTTL,
		EnableCaching:  cfg.EnableCaching,
		EnabledMethods: enabledMethods,
	}
}

type IdempotencyManager struct {
	config IdempotencyConfig
	store  Store
	mu     sync.RWMutex
	closed bool
}

func NewIdempotencyManager(config IdempotencyConfig, store Store) *IdempotencyManager {
	return &IdempotencyManager{
		config: validateConfig(config), // ✅ Безопасное копирование
		store:  store,
	}
}

func (m *IdempotencyManager) CheckAndSet(
	ctx context.Context,
	idempotencyKey string,
	response interface{},
) (cached bool, err error) {
	if idempotencyKey == "" {
		return false, nil
	}

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return false, nil
	}
	m.mu.RUnlock()

	cachedResponse, exists, err := m.store.Check(idempotencyKey)
	if err != nil {
		return false, fmt.Errorf("failed to check idempotency key: %w", err)
	}

	if exists {
		if m.config.EnableCaching && len(cachedResponse) > 0 {
			if err := json.Unmarshal(cachedResponse, response); err != nil {
				return true, fmt.Errorf("failed to unmarshal cached response: %w", err)
			}
		}
		return true, nil
	}

	if m.config.EnableCaching && response != nil {
		responseBytes, err := json.Marshal(response)
		if err != nil {
			return false, fmt.Errorf("failed to marshal response: %w", err)
		}

		if err := m.store.Save(idempotencyKey, responseBytes, m.config.KeyTTL); err != nil {
			return false, fmt.Errorf("failed to save idempotency key: %w", err)
		}
	}

	return false, nil
}

func (m *IdempotencyManager) ValidateKey(key string) error {
	if key == "" {
		return status.Error(codes.InvalidArgument, "idempotency key is required")
	}

	if len(key) < 8 || len(key) > 128 {
		return status.Error(codes.InvalidArgument, "idempotency key must be between 8 and 128 characters")
	}

	return nil
}

func (m *IdempotencyManager) GetStore() Store {
	return m.store
}

func (m *IdempotencyManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	if closer, ok := m.store.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

func UnaryServerInterceptor(manager *IdempotencyManager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		manager.mu.RLock()
		enabled := manager.config.EnabledMethods[info.FullMethod]
		closed := manager.closed
		manager.mu.RUnlock()

		if !enabled || closed {
			return handler(ctx, req)
		}

		idempotencyKey := extractIdempotencyKey(req)
		if idempotencyKey == "" {
			return handler(ctx, req)
		}

		if err := manager.ValidateKey(idempotencyKey); err != nil {
			return nil, err
		}

		var cachedResp interface{}

		if manager.config.EnableCaching {
			cachedResp = createResponseForMethod(info.FullMethod)
		}

		cached, err := manager.CheckAndSet(ctx, idempotencyKey, cachedResp)
		if err != nil {
			return handler(ctx, req)
		}

		if cached {
			if cachedResp != nil {
				return cachedResp, nil
			}
			return handler(ctx, req)
		}

		resp, err := handler(ctx, req)
		if err != nil {
			return resp, err
		}
		if manager.config.EnableCaching && resp != nil {
			responseBytes, marshalErr := json.Marshal(resp)
			if marshalErr == nil {
				_ = manager.store.Save(idempotencyKey, responseBytes, manager.config.KeyTTL)
			}
		}

		return resp, nil
	}
}

func extractIdempotencyKey(req interface{}) string {
	if r, ok := req.(interface{ GetIdempotencyKey() string }); ok {
		return r.GetIdempotencyKey()
	}
	return ""
}

func createResponseForMethod(method string) interface{} {
	switch method {
	case "/order.v1.OrderService/CreateOrder":
		return &struct {
			OrderId string `json:"order_id"`
			Status  string `json:"status"`
		}{}
	default:
		return nil
	}
}

func (m *IdempotencyManager) IsEnabled(method string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return false
	}

	return m.config.EnabledMethods[method]
}
