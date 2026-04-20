package idempotency

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Store struct {
	mu      sync.RWMutex
	data    map[string]time.Time
	ttl     time.Duration
	logger  *zap.Logger
	stopCh  chan struct{}
	maxKeys int
}

func NewStore(keyTTL, cleanupInterval time.Duration, maxKeys int, logger *zap.Logger) *Store {
	s := &Store{
		data:    make(map[string]time.Time),
		ttl:     keyTTL,
		logger:  logger,
		stopCh:  make(chan struct{}),
		maxKeys: maxKeys,
	}
	go s.cleanupLoop(cleanupInterval)
	return s
}

func (s *Store) CheckAndSet(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key]; exists {
		return true, nil
	}

	if s.maxKeys > 0 && len(s.data) >= s.maxKeys {
		for k := range s.data {
			delete(s.data, k)
			break
		}
	}

	s.data[key] = time.Now().Add(s.ttl)
	return false, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *Store) Close() {
	close(s.stopCh)
}

func (s *Store) Stats() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data), s.maxKeys
}

func (s *Store) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for k, exp := range s.data {
				if now.After(exp) {
					delete(s.data, k)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}
