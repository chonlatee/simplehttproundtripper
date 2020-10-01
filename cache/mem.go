package cache

import (
	"errors"
	"sync"
)

// MemoryCache for set cache in map
type MemoryCache struct {
	mem map[string]string
	su  sync.Mutex
}

// Set for set cache with key Set(key, value string) error
func (m *MemoryCache) Set(key, value string) error {
	m.su.Lock()
	defer m.su.Unlock()
	m.mem[key] = value // optional: validate  duplicate key
	return nil
}

// Get for get value in cache Get(key string) string
func (m *MemoryCache) Get(key string) (string, error) {
	if val, ok := m.mem[key]; ok {
		return val, nil
	}

	return "", errors.New("Cache not found")
}

// NewMemoryCache just simple map cache ( not use in production )
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		mem: make(map[string]string),
	}
}
