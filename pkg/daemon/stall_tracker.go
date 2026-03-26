// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/execution"
)

// TokenSnapshot represents a point-in-time snapshot of an agent's token usage.
type TokenSnapshot struct {
	TotalTokens int       // Total token count at this snapshot
	Timestamp   time.Time // When this snapshot was taken
}

// StallTracker detects agents that are running but making no token progress.
// This catches agents stuck in infinite loops, crashed during tool execution,
// or otherwise hung while still appearing "active" in the session status.
type StallTracker struct {
	mu sync.Mutex
	// snapshots maps session ID -> most recent token snapshot
	snapshots map[string]TokenSnapshot
	// stallThreshold is how long to wait before flagging as stalled
	// Default: 3 minutes (configurable)
	stallThreshold time.Duration
}

// NewStallTracker creates a new stall tracker with the given threshold.
// threshold determines how long an agent must have unchanged tokens before being flagged.
func NewStallTracker(threshold time.Duration) *StallTracker {
	if threshold == 0 {
		threshold = 3 * time.Minute // Default: 3 minutes
	}
	return &StallTracker{
		snapshots:      make(map[string]TokenSnapshot),
		stallThreshold: threshold,
	}
}

// Update records a new token snapshot for the given session.
// Returns true if the agent is stalled (no token progress for threshold duration).
func (st *StallTracker) Update(sessionID string, tokens *execution.TokenStats) bool {
	if st == nil || tokens == nil {
		return false
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	totalTokens := tokens.InputTokens + tokens.OutputTokens
	now := time.Now()

	// Check if we have a previous snapshot
	prev, exists := st.snapshots[sessionID]

	// Store current snapshot
	st.snapshots[sessionID] = TokenSnapshot{
		TotalTokens: totalTokens,
		Timestamp:   now,
	}

	// If no previous snapshot, agent is not stalled (first time seeing it)
	if !exists {
		return false
	}

	// If tokens have increased, agent is making progress (not stalled)
	if totalTokens > prev.TotalTokens {
		return false
	}

	// Tokens unchanged - check how long it's been stalled
	timeSinceLastChange := now.Sub(prev.Timestamp)
	return timeSinceLastChange >= st.stallThreshold
}

// IsStalled checks if an agent is stalled without updating the snapshot.
// This is useful for read-only checks (e.g., in status displays).
func (st *StallTracker) IsStalled(sessionID string, tokens *execution.TokenStats) bool {
	if st == nil || tokens == nil {
		return false
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	totalTokens := tokens.InputTokens + tokens.OutputTokens
	now := time.Now()

	prev, exists := st.snapshots[sessionID]
	if !exists {
		return false
	}

	// If tokens have increased since last snapshot, not stalled
	if totalTokens > prev.TotalTokens {
		return false
	}

	// Tokens unchanged - check duration
	timeSinceLastChange := now.Sub(prev.Timestamp)
	return timeSinceLastChange >= st.stallThreshold
}

// CleanStale removes snapshots for sessions that haven't been seen recently.
// This prevents unbounded memory growth for long-running daemons.
// Call this periodically (e.g., every 15 minutes).
func (st *StallTracker) CleanStale(maxAge time.Duration) {
	if st == nil {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if maxAge == 0 {
		maxAge = 1 * time.Hour // Default: clean snapshots older than 1 hour
	}

	now := time.Now()
	for sessionID, snapshot := range st.snapshots {
		if now.Sub(snapshot.Timestamp) > maxAge {
			delete(st.snapshots, sessionID)
		}
	}
}

// GetStallDuration returns how long an agent has been stalled, or 0 if not stalled.
func (st *StallTracker) GetStallDuration(sessionID string, tokens *execution.TokenStats) time.Duration {
	if st == nil || tokens == nil {
		return 0
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	totalTokens := tokens.InputTokens + tokens.OutputTokens
	now := time.Now()

	prev, exists := st.snapshots[sessionID]
	if !exists {
		return 0
	}

	// If tokens have increased, not stalled
	if totalTokens > prev.TotalTokens {
		return 0
	}

	// Return how long it's been stalled
	return now.Sub(prev.Timestamp)
}
