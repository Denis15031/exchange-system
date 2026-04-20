package cache

import (
	"container/list"
	"context"
	"sync"
	"time"

	"exchange-system/shared/ports"
)

var _ ports.Cache = (*LRUCache)(nil)

type LRUCache struct {
	mu      sync.RWMutex
	data    map[string]*cacheEntry
	ll      *list.List
	maxSize int
}

type cacheEntry struct {
	key       string
	value     []byte
	expiresAt time.Time
	element   *list.Element
}

func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		data:    make(map[string]*cacheEntry, maxSize),
		ll:      list.New(),
		maxSize: maxSize,
	}
}

func (c *LRUCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.data[key]; ok {

		if time.Now().After(ent.expiresAt) {
			c.removeEntry(ent)
			return nil, false, nil
		}

		c.ll.MoveToFront(ent.element)
		return ent.value, true, nil
	}
	return nil, false, nil
}

func (c *LRUCache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.data[key]; ok {
		ent.value = value
		ent.expiresAt = c.calcExpiresAt(ttlSeconds)
		c.ll.MoveToFront(ent.element)
		return nil
	}

	if c.ll.Len() >= c.maxSize {
		c.removeOldest()
	}

	ent := &cacheEntry{
		key:       key,
		value:     value,
		expiresAt: c.calcExpiresAt(ttlSeconds),
	}

	ent.element = c.ll.PushFront(key)
	c.data[key] = ent

	return nil
}

func (c *LRUCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.data[key]; ok {
		c.removeEntry(ent)
	}
	return nil
}

func (c *LRUCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*cacheEntry)
	c.ll = list.New()
	return nil
}

func (c *LRUCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]interface{}{
		"type":       "lru-memory",
		"total_keys": len(c.data),
		"max_size":   c.maxSize,
	}
}

func (c *LRUCache) calcExpiresAt(ttlSeconds int) time.Time {
	if ttlSeconds > 0 {
		return time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	return time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
}

func (c *LRUCache) removeEntry(ent *cacheEntry) {
	c.ll.Remove(ent.element)
	delete(c.data, ent.key)
}

func (c *LRUCache) removeOldest() {
	if elem := c.ll.Back(); elem != nil {
		c.ll.Remove(elem)
		key := elem.Value.(string)
		delete(c.data, key)
	}
}
