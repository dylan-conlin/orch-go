// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/atomicwrite"
)

// ProcessedIssueCache provides unified deduplication for spawned issues.
// It consolidates three fragmented dedup mechanisms:
// 1. Persistent cache (survives daemon restart) - replaces in-memory SpawnedIssueTracker
// 2. Session dedup (checks OpenCode sessions)
// 3. Phase Complete check (checks beads comments)
//
// The cache persists to ~/.orch/processed-issues.jsonl and automatically
// prunes entries older than 30 days on load.
type ProcessedIssueCache struct {
	mu         sync.RWMutex
	filePath   string
	entries    map[string]time.Time
	maxEntries int
	ttl        time.Duration

	// sessionChecker is injected for testing (defaults to HasExistingSessionForBeadsID)
	sessionChecker func(beadsID string) bool

	// phaseCompleteChecker is injected for testing (defaults to HasPhaseComplete)
	phaseCompleteChecker func(beadsID string) (bool, error)
}

// cacheEntry represents a single entry in the JSONL cache file.
type cacheEntry struct {
	BeadsID   string    `json:"beads_id"`
	Timestamp time.Time `json:"timestamp"`
}

const (
	DefaultProcessedIssueCacheMaxEntries = 5000
	DefaultProcessedIssueCacheTTL        = 30 * 24 * time.Hour
)

// NewProcessedIssueCache creates a new cache, loading from the specified file path.
// If the file doesn't exist, creates an empty cache.
// Automatically prunes entries older than ttl on load.
func NewProcessedIssueCache(filePath string, maxSize int, ttl time.Duration) (*ProcessedIssueCache, error) {
	if maxSize <= 0 {
		return nil, fmt.Errorf("processed issue cache maxSize must be > 0")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("processed issue cache ttl must be > 0")
	}

	cache := &ProcessedIssueCache{
		filePath:             filePath,
		entries:              make(map[string]time.Time),
		maxEntries:           maxSize,
		ttl:                  ttl,
		sessionChecker:       HasExistingSessionForBeadsID,
		phaseCompleteChecker: HasPhaseComplete,
	}

	// Load existing entries from file
	if err := cache.load(); err != nil {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}

	// Prune old entries (older than ttl)
	cache.prune()
	cache.enforceMaxEntries()

	return cache, nil
}

// ShouldProcess returns true if the issue should be processed (spawned).
// Returns false if ANY of the following dedup checks indicate the issue
// is already processed:
// 1. Issue is in persistent cache (was recently spawned)
// 2. Issue has an existing OpenCode session
// 3. Issue has "Phase: Complete" in beads comments
//
// This is the single entry point for all dedup checking, replacing the
// fragmented checks previously spread across daemon.Once().
func (c *ProcessedIssueCache) ShouldProcess(beadsID string) bool {
	if beadsID == "" {
		return false
	}

	// Check 1: Persistent cache (replaces SpawnedIssueTracker)
	c.mu.RLock()
	_, inCache := c.entries[beadsID]
	c.mu.RUnlock()

	if inCache {
		return false
	}

	// Check 2: Session dedup (checks OpenCode sessions)
	if c.sessionChecker(beadsID) {
		return false
	}

	// Check 3: Phase Complete (checks beads comments)
	// Fail-safe: on error, assume complete (don't spawn) to prevent duplicates
	hasComplete, err := c.phaseCompleteChecker(beadsID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: phase complete check failed for %s (assuming exists to prevent duplicate): %v\n", beadsID, err)
		return false
	}
	if hasComplete {
		return false
	}

	// All checks passed - issue should be processed
	return true
}

// MarkProcessed marks an issue as processed and persists to disk.
// Call this immediately before spawning work to prevent duplicate spawns.
func (c *ProcessedIssueCache) MarkProcessed(beadsID string) error {
	if beadsID == "" {
		return fmt.Errorf("beadsID cannot be empty")
	}

	c.mu.Lock()
	if _, exists := c.entries[beadsID]; !exists && len(c.entries) >= c.maxEntries {
		c.evictOldestLocked()
	}
	c.entries[beadsID] = time.Now()
	c.mu.Unlock()

	// Persist to disk
	return c.save()
}

// Unmark removes an issue from the cache.
// Call this when spawn fails or when the issue should be retried.
func (c *ProcessedIssueCache) Unmark(beadsID string) error {
	c.mu.Lock()
	delete(c.entries, beadsID)
	c.mu.Unlock()

	// Persist to disk
	return c.save()
}

// save persists the cache to disk as JSONL.
// Each line is a separate JSON object representing a cache entry.
// Uses atomic write (temp file + rename) to prevent partial reads.
// Previous implementation used streaming encoder with os.Create + defer Close,
// which leaked the temp file on encode errors.
func (c *ProcessedIssueCache) save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Build JSONL content in memory, then use atomic write
	var data []byte
	for beadsID, timestamp := range c.entries {
		entry := cacheEntry{
			BeadsID:   beadsID,
			Timestamp: timestamp,
		}
		line, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to encode cache entry: %w", err)
		}
		data = append(data, line...)
		data = append(data, '\n')
	}

	if err := atomicwrite.WriteFile(c.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// load reads the cache from disk.
// Returns nil if the file doesn't exist (empty cache).
func (c *ProcessedIssueCache) load() error {
	f, err := os.Open(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - start with empty cache
			return nil
		}
		return fmt.Errorf("failed to open cache file: %w", err)
	}
	defer f.Close()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Read each line as a JSON object
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry cacheEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			// Skip malformed lines (log warning but continue)
			fmt.Fprintf(os.Stderr, "warning: failed to parse cache entry: %v\n", err)
			continue
		}
		c.entries[entry.BeadsID] = entry.Timestamp
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	return nil
}

// prune removes entries older than the configured TTL.
// Should be called after load() to clean up stale entries.
func (c *ProcessedIssueCache) prune() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-c.ttl)
	for beadsID, timestamp := range c.entries {
		if timestamp.Before(cutoff) {
			delete(c.entries, beadsID)
		}
	}
}

func (c *ProcessedIssueCache) enforceMaxEntries() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for len(c.entries) > c.maxEntries {
		c.evictOldestLocked()
	}
}

func (c *ProcessedIssueCache) evictOldestLocked() {
	var oldestID string
	var oldestTime time.Time

	for beadsID, ts := range c.entries {
		if oldestID == "" || ts.Before(oldestTime) {
			oldestID = beadsID
			oldestTime = ts
		}
	}

	if oldestID != "" {
		delete(c.entries, oldestID)
	}
}

// Count returns the number of entries in the cache.
// Used for monitoring and debugging.
func (c *ProcessedIssueCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
