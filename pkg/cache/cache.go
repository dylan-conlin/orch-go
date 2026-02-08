package cache

import (
	"fmt"
	"time"
)

// Cache stores required construction bounds for in-memory caches.
// New caches should be constructed via NewCache/NewNamedCache so maxSize and ttl
// are always explicit by API design.
type Cache struct {
	maxSize int
	ttl     time.Duration
}

// NewCache validates and returns cache bounds.
// Panics on invalid input; use NewCacheE to return errors instead.
func NewCache(maxSize int, ttl time.Duration) Cache {
	bounds, err := NewCacheE(maxSize, ttl)
	if err != nil {
		panic(err)
	}

	return bounds
}

// NewCacheE validates and returns cache bounds.
func NewCacheE(maxSize int, ttl time.Duration) (Cache, error) {
	return newCache("", maxSize, ttl)
}

// NewNamedCache validates and returns cache bounds with a cache-specific prefix
// in panic messages.
func NewNamedCache(name string, maxSize int, ttl time.Duration) Cache {
	bounds, err := NewNamedCacheE(name, maxSize, ttl)
	if err != nil {
		panic(err)
	}

	return bounds
}

// NewNamedCacheE validates and returns cache bounds with a cache-specific prefix
// in error messages.
func NewNamedCacheE(name string, maxSize int, ttl time.Duration) (Cache, error) {
	return newCache(name, maxSize, ttl)
}

func newCache(name string, maxSize int, ttl time.Duration) (Cache, error) {
	prefix := ""
	if name != "" {
		prefix = name + " "
	}

	if maxSize <= 0 {
		return Cache{}, fmt.Errorf("%smaxSize must be > 0", prefix)
	}
	if ttl <= 0 {
		return Cache{}, fmt.Errorf("%sttl must be > 0", prefix)
	}

	return Cache{maxSize: maxSize, ttl: ttl}, nil
}

// MaxSize returns the configured maximum entry count.
func (c Cache) MaxSize() int {
	return c.maxSize
}

// TTL returns the configured entry time-to-live.
func (c Cache) TTL() time.Duration {
	return c.ttl
}
