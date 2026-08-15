package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Memory stores cache values in process memory.
type Memory struct {
	mu    sync.RWMutex
	items map[string]memoryItem
	now   func() time.Time
}

type memoryItem struct {
	payload   []byte
	expiresAt time.Time
}

// NewMemory creates an empty in-process cache.
func NewMemory() *Memory {
	return &Memory{
		items: make(map[string]memoryItem),
		now:   time.Now,
	}
}

// Get implements Cache.
func (m *Memory) Get(ctx context.Context, key string, dst any) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return false, nil
	}
	if item.expired(m.now()) {
		m.mu.Lock()
		if current, ok := m.items[key]; ok && current.expired(m.now()) {
			delete(m.items, key)
		}
		m.mu.Unlock()
		return false, nil
	}

	return true, decode(item.payload, dst)
}

// Set implements Cache.
func (m *Memory) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateTTL(ttl); err != nil {
		return err
	}
	payload, err := encode(value)
	if err != nil {
		return err
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = m.now().Add(ttl)
	}

	m.mu.Lock()
	m.items[key] = memoryItem{payload: payload, expiresAt: expiresAt}
	m.mu.Unlock()
	return nil
}

// Delete implements Cache.
func (m *Memory) Delete(ctx context.Context, keys ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	for _, key := range keys {
		delete(m.items, key)
	}
	m.mu.Unlock()
	return nil
}

// Increment implements Cache.
func (m *Memory) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var current int64
	if item, ok := m.items[key]; ok && !item.expired(m.now()) {
		if err := json.Unmarshal(item.payload, &current); err != nil {
			return 0, fmt.Errorf("cache: increment %q: stored value is not an integer: %w", key, err)
		}
	}

	current += delta
	payload, err := encode(current)
	if err != nil {
		return 0, err
	}
	expiresAt := time.Time{}
	if item, ok := m.items[key]; ok && !item.expired(m.now()) {
		expiresAt = item.expiresAt
	}
	m.items[key] = memoryItem{payload: payload, expiresAt: expiresAt}
	return current, nil
}

func (i memoryItem) expired(now time.Time) bool {
	return !i.expiresAt.IsZero() && !now.Before(i.expiresAt)
}
