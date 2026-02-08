// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"sync"
	"time"
)

// SpawnedIssueTracker tracks issue IDs that have been spawned to prevent
// duplicate spawns during the race window before beads status is updated.
//
// The race condition occurs because:
// 1. Daemon fetches ready issues (issue status = "open")
// 2. Daemon calls SpawnWork which runs "orch work"
// 3. Before "orch work" marks the issue as "in_progress", daemon polls again
// 4. The issue still appears as "open" so daemon spawns another agent
//
// This tracker solves the problem by tracking spawned issue IDs immediately
// when SpawnWork is called, before the async status update occurs.
type SpawnedIssueTracker struct {
	mu sync.Mutex

	// spawned maps issue ID to spawn timestamp.
	// Entries are removed during reconciliation when the issue is confirmed
	// to be in_progress or closed.
	spawned map[string]time.Time

	// MaxEntries bounds the number of tracked issues in memory.
	MaxEntries int

	// TTL is how long to keep entries before considering them stale.
	// Default is 6 hours - matching typical agent work duration.
	// This provides backup protection when session-level dedup fails.
	TTL time.Duration
}

const (
	DefaultSpawnedIssueTrackerMaxEntries = 5000
	DefaultSpawnedIssueTrackerTTL        = 6 * time.Hour
)

// NewSpawnedIssueTracker creates a new tracker with explicit max size and TTL.
// Primary dedup is done via session-level checking in daemon.Once().
func NewSpawnedIssueTracker(maxSize int, ttl time.Duration) *SpawnedIssueTracker {
	if maxSize <= 0 {
		panic("spawned issue tracker maxSize must be > 0")
	}
	if ttl <= 0 {
		panic("spawned issue tracker ttl must be > 0")
	}

	return &SpawnedIssueTracker{
		spawned:    make(map[string]time.Time),
		MaxEntries: maxSize,
		TTL:        ttl,
	}
}

// NewSpawnedIssueTrackerWithTTL creates a tracker with a custom TTL.
func NewSpawnedIssueTrackerWithTTL(maxSize int, ttl time.Duration) *SpawnedIssueTracker {
	return NewSpawnedIssueTracker(maxSize, ttl)
}

// MarkSpawned records that an issue has been spawned.
// Call this immediately before calling SpawnWork.
func (t *SpawnedIssueTracker) MarkSpawned(issueID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if _, exists := t.spawned[issueID]; !exists && len(t.spawned) >= t.MaxEntries {
		t.evictOldestLocked()
	}

	t.spawned[issueID] = now
}

// IsSpawned returns true if the issue was recently spawned (within TTL).
func (t *SpawnedIssueTracker) IsSpawned(issueID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	spawnTime, exists := t.spawned[issueID]
	if !exists {
		return false
	}

	// Check if entry is stale
	if time.Since(spawnTime) > t.TTL {
		// Stale entry - remove it and return false
		delete(t.spawned, issueID)
		return false
	}

	return true
}

// Unmark removes an issue from the tracker.
// Call this when spawn fails or when confirmed the issue is now in_progress/closed.
func (t *SpawnedIssueTracker) Unmark(issueID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.spawned, issueID)
}

// CleanStale removes entries older than TTL.
// Call this periodically (e.g., at the start of each poll cycle).
func (t *SpawnedIssueTracker) CleanStale() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cleanStaleLocked(time.Now())
}

func (t *SpawnedIssueTracker) cleanStaleLocked(now time.Time) int {
	removed := 0
	for id, spawnTime := range t.spawned {
		if now.Sub(spawnTime) > t.TTL {
			delete(t.spawned, id)
			removed++
		}
	}
	return removed
}

func (t *SpawnedIssueTracker) evictOldestLocked() {
	var oldestIssueID string
	var oldestTime time.Time

	for issueID, spawnTime := range t.spawned {
		if oldestIssueID == "" || spawnTime.Before(oldestTime) {
			oldestIssueID = issueID
			oldestTime = spawnTime
		}
	}

	if oldestIssueID != "" {
		delete(t.spawned, oldestIssueID)
	}
}

// ReconcileWithIssues removes tracked issues that are now in_progress or closed.
// Pass the list of issues that are still "open" - any tracked issue NOT in this
// list will be removed (it has transitioned to in_progress or closed).
func (t *SpawnedIssueTracker) ReconcileWithIssues(openIssues []Issue) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Build set of open issue IDs
	openSet := make(map[string]bool)
	for _, issue := range openIssues {
		if issue.Status == "open" {
			openSet[issue.ID] = true
		}
	}

	// Remove tracked issues that are no longer open (now in_progress or closed)
	removed := 0
	for id := range t.spawned {
		if !openSet[id] {
			delete(t.spawned, id)
			removed++
		}
	}

	return removed
}

// Count returns the number of currently tracked spawned issues.
func (t *SpawnedIssueTracker) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.spawned)
}

// TrackedIDs returns a copy of the currently tracked issue IDs.
func (t *SpawnedIssueTracker) TrackedIDs() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	ids := make([]string, 0, len(t.spawned))
	for id := range t.spawned {
		ids = append(ids, id)
	}
	return ids
}

// ClearAbandoned removes issues from the tracker that have been abandoned.
// This allows the daemon to respawn an issue after it was abandoned via `orch abandon`.
// Returns the number of issues cleared.
func (t *SpawnedIssueTracker) ClearAbandoned(abandonedIDs []string) int {
	if len(abandonedIDs) == 0 {
		return 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	cleared := 0
	for _, id := range abandonedIDs {
		if _, exists := t.spawned[id]; exists {
			delete(t.spawned, id)
			cleared++
		}
	}
	return cleared
}
