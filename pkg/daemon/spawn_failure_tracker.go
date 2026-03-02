// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"sync"
	"time"
)

// DefaultMaxIssueFailures is the default number of spawn failures per issue
// before the issue is circuit-broken (skipped in future poll cycles).
const DefaultMaxIssueFailures = 3

// issueFailureEntry tracks spawn failures for a single issue.
type issueFailureEntry struct {
	Count      int
	LastReason string
	LastTime   time.Time
}

// SpawnFailureTracker tracks spawn failures to surface them in health metrics.
// This prevents silent failure when UpdateBeadsStatus persistently fails.
//
// Also tracks per-issue failures. After MaxIssueFailures consecutive failures
// for a single issue, that issue is circuit-broken and skipped in future polls.
// This prevents infinite spawn failure loops (e.g., cross-repo issues that
// can't be resolved locally).
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

	// issueFailures tracks per-issue spawn failure counts.
	// Key is beads issue ID. When count >= MaxIssueFailures, the issue
	// is circuit-broken and skipped in NextIssueExcluding().
	issueFailures map[string]*issueFailureEntry

	// MaxIssueFailures is the threshold for per-issue circuit breaking.
	// After this many consecutive failures for a single issue, it's skipped.
	// 0 means use DefaultMaxIssueFailures.
	MaxIssueFailures int
}

// NewSpawnFailureTracker creates a new tracker with default thresholds.
func NewSpawnFailureTracker() *SpawnFailureTracker {
	return &SpawnFailureTracker{
		issueFailures: make(map[string]*issueFailureEntry),
	}
}

// NewSpawnFailureTrackerWithThreshold creates a tracker with a custom per-issue
// failure threshold. If maxIssueFailures is 0, the default is used.
func NewSpawnFailureTrackerWithThreshold(maxIssueFailures int) *SpawnFailureTracker {
	return &SpawnFailureTracker{
		issueFailures:    make(map[string]*issueFailureEntry),
		MaxIssueFailures: maxIssueFailures,
	}
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

	// Count circuit-broken issues
	threshold := t.maxIssueFailures()
	circuitBroken := 0
	for _, entry := range t.issueFailures {
		if entry.Count >= threshold {
			circuitBroken++
		}
	}

	return SpawnFailureSnapshot{
		ConsecutiveFailures: t.consecutiveFailures,
		TotalFailures:       t.totalFailures,
		LastFailure:         t.lastFailure,
		LastFailureReason:   t.lastFailureReason,
		CircuitBrokenIssues: circuitBroken,
	}
}

// SpawnFailureSnapshot is a point-in-time snapshot of failure tracking state.
type SpawnFailureSnapshot struct {
	ConsecutiveFailures int       `json:"consecutive_failures"`
	TotalFailures       int       `json:"total_failures"`
	LastFailure         time.Time `json:"last_failure,omitempty"`
	LastFailureReason   string    `json:"last_failure_reason,omitempty"`
	// CircuitBrokenIssues is the number of issues currently circuit-broken
	// due to exceeding the per-issue failure threshold.
	CircuitBrokenIssues int `json:"circuit_broken_issues,omitempty"`
}

// maxIssueFailures returns the effective per-issue failure threshold.
func (t *SpawnFailureTracker) maxIssueFailures() int {
	if t.MaxIssueFailures > 0 {
		return t.MaxIssueFailures
	}
	return DefaultMaxIssueFailures
}

// RecordIssueFailure records a spawn failure for a specific issue.
// After maxIssueFailures consecutive failures, IsIssueCircuitBroken returns true.
func (t *SpawnFailureTracker) RecordIssueFailure(issueID, reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry, ok := t.issueFailures[issueID]
	if !ok {
		entry = &issueFailureEntry{}
		t.issueFailures[issueID] = entry
	}
	entry.Count++
	entry.LastReason = reason
	entry.LastTime = time.Now()
}

// ClearIssueFailures resets the failure count for a specific issue.
// Call this when an issue spawns successfully.
func (t *SpawnFailureTracker) ClearIssueFailures(issueID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.issueFailures, issueID)
}

// IsIssueCircuitBroken returns true if the issue has exceeded the per-issue
// failure threshold. Also returns the failure count and last reason for logging.
func (t *SpawnFailureTracker) IsIssueCircuitBroken(issueID string) (broken bool, count int, reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry, ok := t.issueFailures[issueID]
	if !ok {
		return false, 0, ""
	}
	threshold := t.maxIssueFailures()
	return entry.Count >= threshold, entry.Count, entry.LastReason
}

// IssueFailureCount returns the current failure count for an issue.
func (t *SpawnFailureTracker) IssueFailureCount(issueID string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	if entry, ok := t.issueFailures[issueID]; ok {
		return entry.Count
	}
	return 0
}

// CircuitBrokenIssues returns the IDs and reasons of all circuit-broken issues.
func (t *SpawnFailureTracker) CircuitBrokenIssues() map[string]string {
	t.mu.Lock()
	defer t.mu.Unlock()

	threshold := t.maxIssueFailures()
	result := make(map[string]string)
	for id, entry := range t.issueFailures {
		if entry.Count >= threshold {
			result[id] = fmt.Sprintf("%d failures, last: %s", entry.Count, entry.LastReason)
		}
	}
	return result
}
