package idempotency

import (
	"sync"
	"time"
)

type IdempotencyKey struct {
	Key       string
	CreatedAt time.Time
	ExpiresAt time.Time
	Used      bool
	Response  []byte
}

type Store interface {
	Check(key string) ([]byte, bool, error)
	Save(key string, response []byte, ttl time.Duration) error
	Delete(key string) error
	Cleanup()
	Close() error
	GetStats() map[string]interface{}
}

type InMemoryStore struct {
	keys   map[string]*IdempotencyKey
	mu     sync.RWMutex
	ttl    time.Duration
	stopCh chan struct{}
	wg     sync.WaitGroup
	closed bool
}

func NewInMemoryStore(ttl time.Duration) Store {
	store := &InMemoryStore{
		keys:   make(map[string]*IdempotencyKey),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}
	store.wg.Add(1)
	go store.periodicCleanup()

	return store
}

func (s *InMemoryStore) Check(key string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, false, nil
	}

	entry, exists := s.keys[key]
	if !exists {
		return nil, false, nil
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, false, nil
	}

	if !entry.Used {
		return nil, false, nil
	}

	responseCopy := make([]byte, len(entry.Response))
	copy(responseCopy, entry.Response)

	return responseCopy, true, nil
}

func (s *InMemoryStore) Save(key string, response []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	now := time.Now()
	if ttl == 0 {
		ttl = s.ttl
	}

	responseCopy := make([]byte, len(response))
	copy(responseCopy, response)

	s.keys[key] = &IdempotencyKey{
		Key:       key,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		Used:      true,
		Response:  responseCopy,
	}

	return nil
}

func (s *InMemoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	delete(s.keys, key)
	return nil
}

func (s *InMemoryStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	now := time.Now()
	for key, entry := range s.keys {
		if now.After(entry.ExpiresAt) {
			delete(s.keys, key)
		}
	}
}

func (s *InMemoryStore) periodicCleanup() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.Cleanup()
		case <-s.stopCh:
			s.Cleanup()
			return
		}
	}
}

func (s *InMemoryStore) Close() error {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		return nil
	}

	s.closed = true
	s.mu.Unlock()

	close(s.stopCh)

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		return nil
	}
}

func (s *InMemoryStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return map[string]interface{}{
			"total_keys": 0,
			"ttl":        s.ttl.String(),
			"closed":     true,
		}
	}

	return map[string]interface{}{
		"total_keys": len(s.keys),
		"ttl":        s.ttl.String(),
		"closed":     false,
	}
}
