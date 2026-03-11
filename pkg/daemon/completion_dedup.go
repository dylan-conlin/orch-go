// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"sync"
	"time"
)

// CompletionDedupTracker tracks which beads IDs have been processed by the
// completion loop, preventing reprocessing of the same Phase: Complete.
//
// This is defense-in-depth for cases where the daemon:ready-review label
// fails to persist (e.g., beads socket flakiness, label removed externally).
// Without this, an issue with a stale Phase: Complete comment gets
// re-completed every poll cycle.
type CompletionDedupTracker struct {
	mu      sync.Mutex
	entries map[string]completionEntry
	ttl     time.Duration
}

type completionEntry struct {
	summary   string
	timestamp time.Time
}

const defaultCompletionDedupTTL = 6 * time.Hour

// NewCompletionDedupTracker creates a tracker with default TTL.
func NewCompletionDedupTracker() *CompletionDedupTracker {
	return &CompletionDedupTracker{
		entries: make(map[string]completionEntry),
		ttl:     defaultCompletionDedupTTL,
	}
}

// MarkCompleted records that a beads ID was processed with the given summary.
func (t *CompletionDedupTracker) MarkCompleted(beadsID, summary string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries[beadsID] = completionEntry{
		summary:   summary,
		timestamp: time.Now(),
	}
}

// IsCompleted returns true if this beads ID was already processed with the
// same Phase: Complete summary (within TTL). A different summary means the
// issue was re-used for a new task, so it should be re-processed.
func (t *CompletionDedupTracker) IsCompleted(beadsID, summary string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.entries[beadsID]
	if !ok {
		return false
	}
	if time.Since(entry.timestamp) > t.ttl {
		delete(t.entries, beadsID)
		return false
	}
	return entry.summary == summary
}

// Clear removes a beads ID from the tracker (e.g., when issue is re-opened).
func (t *CompletionDedupTracker) Clear(beadsID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, beadsID)
}
