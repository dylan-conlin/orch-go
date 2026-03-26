// Package daemon provides autonomous overnight processing capabilities.
// This file implements one-shot retry for empty-execution classified failures.
// When an agent dies with zero output (empty-execution), the daemon retries once.
// A second empty-execution for the same issue escalates instead of looping.
package daemon

import (
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// EmptyExecutionClassifier classifies the terminal outcome of the most recent
// session for a given beads issue. Returns nil detail when no session is found.
type EmptyExecutionClassifier interface {
	ClassifyLastSession(beadsID string) (*opencode.OutcomeDetail, error)
}

// EmptyExecutionRetryTracker tracks which issues have already been retried
// after an empty-execution classification. Each issue gets exactly one
// automatic retry; a second empty-execution escalates for human review.
type EmptyExecutionRetryTracker struct {
	mu      sync.Mutex
	retried map[string]time.Time // beadsID → when retried
}

// NewEmptyExecutionRetryTracker creates a new tracker.
func NewEmptyExecutionRetryTracker() *EmptyExecutionRetryTracker {
	return &EmptyExecutionRetryTracker{
		retried: make(map[string]time.Time),
	}
}

// HasRetried returns true if this issue has already been retried for empty-execution.
func (t *EmptyExecutionRetryTracker) HasRetried(beadsID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.retried[beadsID]
	return ok
}

// MarkRetried records that this issue has been retried after empty-execution.
func (t *EmptyExecutionRetryTracker) MarkRetried(beadsID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.retried[beadsID] = time.Now()
}

// Clear removes the retry record for an issue (e.g., after successful completion).
func (t *EmptyExecutionRetryTracker) Clear(beadsID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.retried, beadsID)
}

// EmptyExecutionRetryRecord captures metadata about an empty-execution retry or escalation.
type EmptyExecutionRetryRecord struct {
	BeadsID  string
	Title    string
	Attempt  int    // 1 = first retry, 2 = escalation
	Reason   string // Classification reason from OutcomeDetail
	Action   string // "retrying" or "escalated"
}
