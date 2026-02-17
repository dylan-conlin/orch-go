// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"sync"
	"time"
)

// CompletionFailureTracker tracks completion processing failures to surface them in health metrics.
// This prevents silent failure when CompletionOnce persistently fails (e.g., beads database issues).
type CompletionFailureTracker struct {
	mu sync.Mutex

	// consecutiveFailures counts failures since last successful completion processing.
	consecutiveFailures int

	// lastFailure is the timestamp of the most recent failure.
	lastFailure time.Time

	// lastFailureReason is the error message from the most recent failure.
	lastFailureReason string

	// totalFailures is the cumulative count of all failures (lifetime).
	totalFailures int
}

// NewCompletionFailureTracker creates a new tracker.
func NewCompletionFailureTracker() *CompletionFailureTracker {
	return &CompletionFailureTracker{}
}

// RecordFailure records a completion processing failure.
// Call this when CompletionOnce fails.
func (t *CompletionFailureTracker) RecordFailure(reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.consecutiveFailures++
	t.totalFailures++
	t.lastFailure = time.Now()
	t.lastFailureReason = reason
}

// RecordSuccess records successful completion processing.
// Call this when CompletionOnce completes successfully.
// Resets consecutive failure count.
func (t *CompletionFailureTracker) RecordSuccess() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.consecutiveFailures = 0
	// Don't reset totalFailures or lastFailure - keep for historical tracking
}

// ConsecutiveFailures returns the number of consecutive failures since last success.
func (t *CompletionFailureTracker) ConsecutiveFailures() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.consecutiveFailures
}

// TotalFailures returns the total number of failures (lifetime).
func (t *CompletionFailureTracker) TotalFailures() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalFailures
}

// LastFailure returns the timestamp and reason of the most recent failure.
// Returns zero time if no failures have occurred.
func (t *CompletionFailureTracker) LastFailure() (time.Time, string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastFailure, t.lastFailureReason
}

// Snapshot returns a snapshot of the current failure state.
func (t *CompletionFailureTracker) Snapshot() CompletionFailureSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	return CompletionFailureSnapshot{
		ConsecutiveFailures: t.consecutiveFailures,
		TotalFailures:       t.totalFailures,
		LastFailure:         t.lastFailure,
		LastFailureReason:   t.lastFailureReason,
	}
}

// CompletionFailureSnapshot is a point-in-time snapshot of failure tracking state.
type CompletionFailureSnapshot struct {
	ConsecutiveFailures int       `json:"consecutive_failures"`
	TotalFailures       int       `json:"total_failures"`
	LastFailure         time.Time `json:"last_failure,omitempty"`
	LastFailureReason   string    `json:"last_failure_reason,omitempty"`
}
