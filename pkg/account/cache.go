package account

import (
	"sync"
	"time"
)

// CapacityCache provides TTL-based caching for account capacity lookups.
// This avoids hitting the OAuth token refresh endpoint on every spawn,
// which was documented to cause token mismatch bugs (see investigation:
// 2025-12-24-inv-orch-go-investigation-auto-switch).
//
// Default TTL: 5 minutes. The 5-hour rate window is the finest granularity
// that matters, so 5 minutes of staleness is acceptable.
type CapacityCache struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	capacity  *CapacityInfo
	fetchedAt time.Time
}

// NewCapacityCache creates a new capacity cache with the given TTL.
func NewCapacityCache(ttl time.Duration) *CapacityCache {
	return &CapacityCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

// Get returns cached capacity for the named account, or nil if not cached or expired.
func (c *CapacityCache) Get(name string) *CapacityInfo {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[name]
	if !ok {
		return nil
	}

	if time.Since(entry.fetchedAt) > c.ttl {
		delete(c.entries, name)
		return nil
	}

	return entry.capacity
}

// Set stores capacity info for the named account.
func (c *CapacityCache) Set(name string, info *CapacityInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[name] = &cacheEntry{
		capacity:  info,
		fetchedAt: time.Now(),
	}
}

// Invalidate removes cached capacity for the named account.
func (c *CapacityCache) Invalidate(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, name)
}
