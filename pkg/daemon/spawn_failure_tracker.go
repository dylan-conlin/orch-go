// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"sync"
	"time"
)

// SpawnFailureTracker tracks spawn failures to surface them in health metrics.
// This prevents silent failure when UpdateBeadsStatus persistently fails.
type SpawnFailureTracker struct {
	mu sync.Mutex

	// consecutiveFailures counts failures since last successful spawn.
	consecutiveFailures int

	// lastFailure is the timestamp of the most recent failure.
	lastFailure time.Time

	// lastFailureReason is the error message from the most recent failure.
	lastFailureReason string

	// totalFailures is the cumulative count of all failures (lifetime).
	totalFailures int
}

// NewSpawnFailureTracker creates a new tracker.
func NewSpawnFailureTracker() *SpawnFailureTracker {
	return &SpawnFailureTracker{}
}

// RecordFailure records a spawn failure.
// Call this when UpdateBeadsStatus or spawn fails.
func (t *SpawnFailureTracker) RecordFailure(reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.consecutiveFailures++
	t.totalFailures++
	t.lastFailure = time.Now()
	t.lastFailureReason = reason
}

// RecordSuccess records a successful spawn.
// Call this when spawn completes successfully.
// Resets consecutive failure count.
func (t *SpawnFailureTracker) RecordSuccess() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.consecutiveFailures = 0
	// Don't reset totalFailures or lastFailure - keep for historical tracking
}

// ConsecutiveFailures returns the number of consecutive failures since last success.
func (t *SpawnFailureTracker) ConsecutiveFailures() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.consecutiveFailures
}

// TotalFailures returns the total number of failures (lifetime).
func (t *SpawnFailureTracker) TotalFailures() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalFailures
}

// LastFailure returns the timestamp and reason of the most recent failure.
// Returns zero time if no failures have occurred.
func (t *SpawnFailureTracker) LastFailure() (time.Time, string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastFailure, t.lastFailureReason
}

// Snapshot returns a snapshot of the current failure state.
func (t *SpawnFailureTracker) Snapshot() SpawnFailureSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	return SpawnFailureSnapshot{
		ConsecutiveFailures: t.consecutiveFailures,
		TotalFailures:       t.totalFailures,
		LastFailure:         t.lastFailure,
		LastFailureReason:   t.lastFailureReason,
	}
}

// SpawnFailureSnapshot is a point-in-time snapshot of failure tracking state.
type SpawnFailureSnapshot struct {
	ConsecutiveFailures int       `json:"consecutive_failures"`
	TotalFailures       int       `json:"total_failures"`
	LastFailure         time.Time `json:"last_failure,omitempty"`
	LastFailureReason   string    `json:"last_failure_reason,omitempty"`
}
