// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// spawnedTitles maps normalized title to issue ID for content-aware dedup.
	// This catches the case where multiple beads issues are created with identical
	// content (different IDs, same title). Without this, the daemon would spawn
	// each one because ID-based dedup treats them as distinct.
	spawnedTitles map[string]string

	// spawnCounts maps issue ID to the number of times it has been spawned.
	// This provides visibility into thrashing — issues that are repeatedly
	// spawned and fail. Counts survive daemon restarts via disk persistence
	// but are cleaned when the associated spawn entry expires from TTL.
	spawnCounts map[string]int

	// TTL is how long to keep entries before considering them stale.
	// Default is 6 hours - matching typical agent work duration.
	// This provides backup protection when session-level dedup fails.
	TTL time.Duration

	// filePath is the path to the JSON file for disk persistence.
	// When empty, the tracker operates in-memory only (test mode).
	filePath string
}

// spawnCacheFile is the JSON representation of the spawn tracker state on disk.
type spawnCacheFile struct {
	Spawned       map[string]time.Time `json:"spawned"`
	SpawnedTitles map[string]string    `json:"spawned_titles"`
	SpawnCounts   map[string]int       `json:"spawn_counts,omitempty"`
}

// NewSpawnedIssueTracker creates a new tracker with the default 6 hour TTL.
// The TTL was increased from 5 minutes to 6 hours to provide backup protection
// for long-running agents when session-level dedup fails (e.g., OpenCode API down).
// Primary dedup is done via session-level checking in daemon.Once().
func NewSpawnedIssueTracker() *SpawnedIssueTracker {
	return &SpawnedIssueTracker{
		spawned:       make(map[string]time.Time),
		spawnedTitles: make(map[string]string),
		spawnCounts:   make(map[string]int),
		TTL:           6 * time.Hour,
	}
}

// NewSpawnedIssueTrackerWithTTL creates a tracker with a custom TTL.
func NewSpawnedIssueTrackerWithTTL(ttl time.Duration) *SpawnedIssueTracker {
	return &SpawnedIssueTracker{
		spawned:       make(map[string]time.Time),
		spawnedTitles: make(map[string]string),
		spawnCounts:   make(map[string]int),
		TTL:           ttl,
	}
}

// NewSpawnedIssueTrackerWithFile creates a disk-backed tracker that survives daemon restarts.
// On creation, it loads any existing state from the file and cleans stale entries.
// On every mutation, it persists state to disk.
// Disk errors are logged but don't block operation (fail-open).
func NewSpawnedIssueTrackerWithFile(filePath string) *SpawnedIssueTracker {
	t := &SpawnedIssueTracker{
		spawned:       make(map[string]time.Time),
		spawnedTitles: make(map[string]string),
		spawnCounts:   make(map[string]int),
		TTL:           6 * time.Hour,
		filePath:      filePath,
	}
	t.loadFromFile()
	t.CleanStale()
	return t
}

// DefaultSpawnCachePath returns the default path for the spawn dedup cache file.
func DefaultSpawnCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".orch", "spawn_cache.json")
}

// MarkSpawned records that an issue has been spawned.
// Call this immediately before calling SpawnWork.
func (t *SpawnedIssueTracker) MarkSpawned(issueID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spawned[issueID] = time.Now()
	t.spawnCounts[issueID]++
	t.saveLocked()
}

// MarkSpawnedWithTitle records that an issue has been spawned, including its title
// for content-aware dedup. This prevents duplicate spawning when multiple beads issues
// are created with the same title (different IDs, identical content).
func (t *SpawnedIssueTracker) MarkSpawnedWithTitle(issueID, title string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spawned[issueID] = time.Now()
	t.spawnCounts[issueID]++
	count := t.spawnCounts[issueID]
	if count >= 3 {
		fmt.Fprintf(os.Stderr, "spawn-tracker: WARNING: issue %s spawned %d times (possible thrashing)\n", issueID, count)
	}
	if title != "" {
		normalized := normalizeTitle(title)
		if normalized != "" {
			t.spawnedTitles[normalized] = issueID
		}
	}
	t.saveLocked()
}

// IsTitleSpawned returns true if an issue with this title was recently spawned (within TTL).
// Returns the issue ID of the matching spawn, or empty string if not found.
// This catches content duplicates where different beads issues have the same title.
func (t *SpawnedIssueTracker) IsTitleSpawned(title string) (bool, string) {
	if title == "" {
		return false, ""
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	normalized := normalizeTitle(title)
	if normalized == "" {
		return false, ""
	}

	issueID, exists := t.spawnedTitles[normalized]
	if !exists {
		return false, ""
	}

	// Check if the associated spawn is still within TTL
	spawnTime, spawned := t.spawned[issueID]
	if !spawned || time.Since(spawnTime) > t.TTL {
		// Stale - clean up
		delete(t.spawnedTitles, normalized)
		t.saveLocked()
		return false, ""
	}

	return true, issueID
}

// loadFromFile reads the spawn cache from disk into the tracker's in-memory maps.
// Called once during construction. Fail-open: errors are logged, tracker starts empty.
func (t *SpawnedIssueTracker) loadFromFile() {
	if t.filePath == "" {
		return
	}
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "spawn-tracker: failed to read cache %s: %v\n", t.filePath, err)
		}
		return
	}
	var cache spawnCacheFile
	if err := json.Unmarshal(data, &cache); err != nil {
		fmt.Fprintf(os.Stderr, "spawn-tracker: failed to parse cache %s: %v\n", t.filePath, err)
		return
	}
	if cache.Spawned != nil {
		t.spawned = cache.Spawned
	}
	if cache.SpawnedTitles != nil {
		t.spawnedTitles = cache.SpawnedTitles
	}
	if cache.SpawnCounts != nil {
		t.spawnCounts = cache.SpawnCounts
	}
	fmt.Fprintf(os.Stderr, "spawn-tracker: loaded cache with %d entries from %s\n", len(t.spawned), t.filePath)
}

// saveLocked writes the tracker's in-memory state to disk.
// Must be called while t.mu is held. Fail-open: errors are logged.
func (t *SpawnedIssueTracker) saveLocked() {
	if t.filePath == "" {
		return
	}
	cache := spawnCacheFile{
		Spawned:       t.spawned,
		SpawnedTitles: t.spawnedTitles,
		SpawnCounts:   t.spawnCounts,
	}
	data, err := json.Marshal(cache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spawn-tracker: failed to marshal cache: %v\n", err)
		return
	}
	dir := filepath.Dir(t.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "spawn-tracker: failed to create dir %s: %v\n", dir, err)
		return
	}
	// Atomic write: write to temp file then rename
	tmp := t.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "spawn-tracker: failed to write cache %s: %v\n", tmp, err)
		return
	}
	if err := os.Rename(tmp, t.filePath); err != nil {
		fmt.Fprintf(os.Stderr, "spawn-tracker: failed to rename cache %s: %v\n", t.filePath, err)
	}
}

// normalizeTitle normalizes a title for comparison.
// Lowercases and trims whitespace.
func normalizeTitle(title string) string {
	return strings.TrimSpace(strings.ToLower(title))
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
		t.saveLocked()
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
	// Clean title index entries pointing to this issue
	for title, id := range t.spawnedTitles {
		if id == issueID {
			delete(t.spawnedTitles, title)
		}
	}
	t.saveLocked()
}

// CleanStale removes entries older than TTL.
// Call this periodically (e.g., at the start of each poll cycle).
func (t *SpawnedIssueTracker) CleanStale() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	removed := 0
	now := time.Now()
	for id, spawnTime := range t.spawned {
		if now.Sub(spawnTime) > t.TTL {
			delete(t.spawned, id)
			removed++
		}
	}
	// Clean orphaned title entries
	for title, id := range t.spawnedTitles {
		if _, exists := t.spawned[id]; !exists {
			delete(t.spawnedTitles, title)
		}
	}
	// Clean spawn counts for issues no longer tracked
	for id := range t.spawnCounts {
		if _, exists := t.spawned[id]; !exists {
			delete(t.spawnCounts, id)
		}
	}
	if removed > 0 {
		t.saveLocked()
	}
	return removed
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
	if removed > 0 {
		t.saveLocked()
	}

	return removed
}

// ReconcileWithSessions cross-checks spawn cache entries against live sessions.
// For each tracked issue, it calls sessionChecker to determine if the session is alive.
// Entries where the session is confirmed dead are evicted. If sessionChecker returns
// an error, the entry is preserved (fail-closed) to avoid evicting entries for
// agents that may still be running but whose session check infrastructure is down.
//
// Call this at daemon startup to clear stale entries left by agents killed during
// reboot. This is distinct from CleanStale (TTL-based) and ReconcileWithIssues
// (issue-status-based).
func (t *SpawnedIssueTracker) ReconcileWithSessions(sessionChecker func(issueID string) (bool, error)) int {
	if sessionChecker == nil {
		return 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	evicted := 0
	for id := range t.spawned {
		alive, err := sessionChecker(id)
		if err != nil {
			// Fail-closed: can't confirm dead, keep the entry
			fmt.Fprintf(os.Stderr, "spawn-tracker: session check error for %s, keeping entry: %v\n", id, err)
			continue
		}
		if !alive {
			fmt.Fprintf(os.Stderr, "spawn-tracker: evicting dead session %s from cache\n", id)
			delete(t.spawned, id)
			evicted++
		}
	}

	if evicted > 0 {
		// Clean orphaned title entries
		for title, id := range t.spawnedTitles {
			if _, exists := t.spawned[id]; !exists {
				delete(t.spawnedTitles, title)
			}
		}
		// Clean spawn counts for evicted issues
		for id := range t.spawnCounts {
			if _, exists := t.spawned[id]; !exists {
				delete(t.spawnCounts, id)
			}
		}
		t.saveLocked()
	}

	return evicted
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

// SpawnCount returns how many times an issue has been spawned.
// This count survives daemon restarts via disk persistence.
func (t *SpawnedIssueTracker) SpawnCount(issueID string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.spawnCounts[issueID]
}
