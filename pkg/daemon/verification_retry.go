// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"sync"
)

const (
	// DefaultMaxVerificationAttempts is the retry budget for local project completions.
	DefaultMaxVerificationAttempts = 3

	// CrossProjectMaxVerificationAttempts is the retry budget for cross-project completions.
	// Lower because cross-project verification failures are almost always structural
	// (wrong context, missing workspace) and won't resolve by retrying.
	CrossProjectMaxVerificationAttempts = 1

	// LabelVerificationFailed is applied to issues that have exhausted their
	// verification retry budget. The completion scanner filters these out,
	// breaking the infinite retry loop.
	LabelVerificationFailed = "daemon:verification-failed"

	// LabelReadyReview is applied to issues that passed verification.
	// Already existed before this change; referenced here for consistency.
	LabelReadyReview = "daemon:ready-review"
)

// VerificationRetryTracker tracks how many times each beads ID has failed
// completion verification. After exhausting the retry budget, the completion
// is deferred for human review (labeled daemon:verification-failed).
//
// This prevents the infinite retry loop where the daemon discovers the same
// completed agents every cycle, tries to verify them, fails, and retries.
type VerificationRetryTracker struct {
	mu       sync.Mutex
	attempts map[string]int // beadsID -> failed attempt count
}

// NewVerificationRetryTracker creates a new tracker.
func NewVerificationRetryTracker() *VerificationRetryTracker {
	return &VerificationRetryTracker{
		attempts: make(map[string]int),
	}
}

// RecordFailure increments the failure count for a beads ID.
// Returns the new attempt count.
func (t *VerificationRetryTracker) RecordFailure(beadsID string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.attempts[beadsID]++
	return t.attempts[beadsID]
}

// IsExhausted returns true if the beads ID has exceeded its retry budget.
// Cross-project agents (ProjectDir != "") get a lower budget.
func (t *VerificationRetryTracker) IsExhausted(beadsID string, isCrossProject bool) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	max := DefaultMaxVerificationAttempts
	if isCrossProject {
		max = CrossProjectMaxVerificationAttempts
	}
	return t.attempts[beadsID] >= max
}

// Attempts returns the current failure count for a beads ID.
func (t *VerificationRetryTracker) Attempts(beadsID string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.attempts[beadsID]
}

// MaxAttemptsFor returns the retry budget for a given agent type.
func MaxAttemptsFor(isCrossProject bool) int {
	if isCrossProject {
		return CrossProjectMaxVerificationAttempts
	}
	return DefaultMaxVerificationAttempts
}

// Clear removes tracking for a beads ID (e.g., if it's resolved).
func (t *VerificationRetryTracker) Clear(beadsID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, beadsID)
}
