package daemon

import (
	"testing"
	"time"
)

func TestSpawnedIssueTracker_MarkAndCheck(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	// Initially not spawned
	if tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should not be spawned initially")
	}

	// Mark as spawned
	tracker.MarkSpawned("issue-1")

	// Now it should be spawned
	if !tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should be spawned after marking")
	}

	// Other issues should not be affected
	if tracker.IsSpawned("issue-2") {
		t.Error("issue-2 should not be spawned")
	}
}

func TestSpawnedIssueTracker_Unmark(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	tracker.MarkSpawned("issue-1")
	if !tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should be spawned")
	}

	tracker.Unmark("issue-1")
	if tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should not be spawned after unmark")
	}
}

func TestSpawnedIssueTracker_TTL(t *testing.T) {
	// Use short TTL for testing
	tracker := NewSpawnedIssueTrackerWithTTL(50 * time.Millisecond)

	tracker.MarkSpawned("issue-1")
	if !tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should be spawned immediately after marking")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Should now be considered not spawned (stale)
	if tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should not be spawned after TTL expires")
	}
}

func TestSpawnedIssueTracker_CleanStale(t *testing.T) {
	tracker := NewSpawnedIssueTrackerWithTTL(50 * time.Millisecond)

	tracker.MarkSpawned("issue-1")
	tracker.MarkSpawned("issue-2")

	if tracker.Count() != 2 {
		t.Errorf("expected 2 tracked issues, got %d", tracker.Count())
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Add a fresh one
	tracker.MarkSpawned("issue-3")

	// Clean stale
	removed := tracker.CleanStale()
	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	if tracker.Count() != 1 {
		t.Errorf("expected 1 tracked issue after cleanup, got %d", tracker.Count())
	}

	if !tracker.IsSpawned("issue-3") {
		t.Error("issue-3 should still be spawned (fresh)")
	}
}

func TestSpawnedIssueTracker_ReconcileWithIssues(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	// Mark some issues as spawned
	tracker.MarkSpawned("issue-1") // Will transition to in_progress
	tracker.MarkSpawned("issue-2") // Will remain open
	tracker.MarkSpawned("issue-3") // Will transition to closed (not in open list)

	// Simulate beads returning only open issues (issue-2 still open, others transitioned)
	openIssues := []Issue{
		{ID: "issue-2", Status: "open"},
		{ID: "issue-4", Status: "open"}, // A different open issue
	}

	removed := tracker.ReconcileWithIssues(openIssues)
	if removed != 2 {
		t.Errorf("expected 2 removed (issue-1 and issue-3), got %d", removed)
	}

	// issue-2 should still be tracked (still open)
	if !tracker.IsSpawned("issue-2") {
		t.Error("issue-2 should still be tracked (still open)")
	}

	// issue-1 and issue-3 should be removed (no longer open)
	if tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should be removed (transitioned to in_progress)")
	}
	if tracker.IsSpawned("issue-3") {
		t.Error("issue-3 should be removed (transitioned to closed)")
	}
}

func TestSpawnedIssueTracker_TrackedIDs(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	tracker.MarkSpawned("issue-1")
	tracker.MarkSpawned("issue-2")
	tracker.MarkSpawned("issue-3")

	ids := tracker.TrackedIDs()
	if len(ids) != 3 {
		t.Errorf("expected 3 tracked IDs, got %d", len(ids))
	}

	// Check all IDs are present (order not guaranteed)
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	for _, expected := range []string{"issue-1", "issue-2", "issue-3"} {
		if !idSet[expected] {
			t.Errorf("expected %s in tracked IDs", expected)
		}
	}
}

func TestSpawnedIssueTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			issueID := "issue-" + string(rune('a'+id))
			tracker.MarkSpawned(issueID)
			tracker.IsSpawned(issueID)
			tracker.Unmark(issueID)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should complete without race conditions
	// (race detector will catch issues if run with -race)
}

// TestDaemon_SkipsRecentlySpawnedIssues tests that NextIssue skips issues
// that have been recently spawned but status not yet updated in beads.
func TestDaemon_SkipsRecentlySpawnedIssues(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	d := &Daemon{
		SpawnedIssues: tracker,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "issue-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "issue-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		},
		// No label filter - match existing test patterns
	}

	// Mark issue-1 as recently spawned
	tracker.MarkSpawned("issue-1")

	// NextIssue should skip issue-1 and return issue-2
	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() returned nil, expected issue-2")
	}
	if issue.ID != "issue-2" {
		t.Errorf("NextIssue() = %q, want 'issue-2' (should skip recently spawned issue-1)", issue.ID)
	}
}

// TestDaemon_OnceMarkSpawned tests that Once marks issue as spawned before
// calling spawnFunc and unmarks on failure.
func TestDaemon_OnceMarksSpawned(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	spawnCalled := false

	d := &Daemon{
		SpawnedIssues: tracker,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "issue-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			spawnCalled = true
			// Verify issue is marked as spawned DURING spawn call
			if !tracker.IsSpawned(beadsID) {
				t.Error("issue should be marked as spawned during spawnFunc call")
			}
			return nil
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should have processed an issue")
	}
	if !spawnCalled {
		t.Error("spawnFunc should have been called")
	}

	// Issue should still be marked after successful spawn
	if !tracker.IsSpawned("issue-1") {
		t.Error("issue should remain marked after successful spawn")
	}
}

// TestDaemon_OnceUnmarksOnFailure tests that Once unmarks issue if spawn fails.
func TestDaemon_OnceUnmarksOnFailure(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	d := &Daemon{
		SpawnedIssues: tracker,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "issue-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			// Verify issue is marked as spawned DURING spawn call
			if !tracker.IsSpawned(beadsID) {
				t.Error("issue should be marked as spawned during spawnFunc call")
			}
			return errSpawnFailed // Simulate failure
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error: %v", err)
	}
	if result.Processed {
		t.Error("Once() should not have processed (spawn failed)")
	}

	// Issue should be unmarked after failed spawn (can be retried)
	if tracker.IsSpawned("issue-1") {
		t.Error("issue should be unmarked after failed spawn")
	}
}

func TestSpawnedIssueTracker_ClearAbandoned(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	// Mark some issues as spawned
	tracker.MarkSpawned("issue-1")
	tracker.MarkSpawned("issue-2")
	tracker.MarkSpawned("issue-3")

	if tracker.Count() != 3 {
		t.Errorf("expected 3 tracked issues, got %d", tracker.Count())
	}

	// Clear some issues that were "abandoned"
	abandonedIDs := []string{"issue-1", "issue-3", "issue-not-tracked"}
	cleared := tracker.ClearAbandoned(abandonedIDs)

	// Should have cleared 2 (issue-1 and issue-3), not issue-not-tracked
	if cleared != 2 {
		t.Errorf("expected 2 cleared, got %d", cleared)
	}

	// issue-2 should still be tracked
	if !tracker.IsSpawned("issue-2") {
		t.Error("issue-2 should still be tracked")
	}

	// issue-1 and issue-3 should be cleared
	if tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should be cleared (abandoned)")
	}
	if tracker.IsSpawned("issue-3") {
		t.Error("issue-3 should be cleared (abandoned)")
	}

	if tracker.Count() != 1 {
		t.Errorf("expected 1 tracked issue after clearing, got %d", tracker.Count())
	}
}

func TestSpawnedIssueTracker_ClearAbandoned_EmptyList(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	tracker.MarkSpawned("issue-1")
	tracker.MarkSpawned("issue-2")

	// Clear with empty list should do nothing
	cleared := tracker.ClearAbandoned([]string{})
	if cleared != 0 {
		t.Errorf("expected 0 cleared with empty list, got %d", cleared)
	}

	// Clear with nil should also do nothing
	cleared = tracker.ClearAbandoned(nil)
	if cleared != 0 {
		t.Errorf("expected 0 cleared with nil, got %d", cleared)
	}

	// Both issues should still be tracked
	if tracker.Count() != 2 {
		t.Errorf("expected 2 tracked issues, got %d", tracker.Count())
	}
}

var errSpawnFailed = &spawnError{msg: "spawn failed"}

type spawnError struct {
	msg string
}

func (e *spawnError) Error() string {
	return e.msg
}

// TestDaemon_PreventsDuplicateSpawns is an integration test that verifies
// the entire flow prevents duplicate spawns during the race window.
func TestDaemon_PreventsDuplicateSpawns(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	spawnCount := 0

	d := &Daemon{
		SpawnedIssues: tracker,
		listIssuesFunc: func() ([]Issue, error) {
			// Same issue appears in every poll (simulating race condition)
			return []Issue{
				{ID: "issue-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			spawnCount++
			return nil
		},
	}

	// First spawn should succeed
	result1, _ := d.Once()
	if !result1.Processed {
		t.Error("First Once() should have processed")
	}
	if spawnCount != 1 {
		t.Errorf("spawnFunc should have been called once, got %d", spawnCount)
	}

	// Second spawn should be skipped (issue already spawned)
	result2, _ := d.Once()
	if result2.Processed {
		t.Error("Second Once() should not process (issue already spawned)")
	}
	if spawnCount != 1 {
		t.Errorf("spawnFunc should still be 1 call, got %d", spawnCount)
	}

	// Message should indicate no issues available
	if result2.Message != "No spawnable issues in queue" {
		t.Errorf("Expected 'No spawnable issues in queue', got %q", result2.Message)
	}
}
