package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSpawnedIssueTracker_MarkAndCheck(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	if tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should not be spawned initially")
	}

	tracker.MarkSpawned("issue-1")

	if !tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should be spawned after marking")
	}

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
	tracker := NewSpawnedIssueTrackerWithTTL(50 * time.Millisecond)

	tracker.MarkSpawned("issue-1")
	if !tracker.IsSpawned("issue-1") {
		t.Error("issue-1 should be spawned immediately after marking")
	}

	time.Sleep(60 * time.Millisecond)

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

	time.Sleep(60 * time.Millisecond)

	tracker.MarkSpawned("issue-3")

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

	tracker.MarkSpawned("issue-1")
	tracker.MarkSpawned("issue-2")
	tracker.MarkSpawned("issue-3")

	openIssues := []Issue{
		{ID: "issue-2", Status: "open"},
		{ID: "issue-4", Status: "open"},
	}

	removed := tracker.ReconcileWithIssues(openIssues)
	if removed != 2 {
		t.Errorf("expected 2 removed (issue-1 and issue-3), got %d", removed)
	}

	if !tracker.IsSpawned("issue-2") {
		t.Error("issue-2 should still be tracked (still open)")
	}

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

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDaemon_SkipsRecentlySpawnedIssues(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
					{ID: "issue-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
				}, nil
			},
		},
	}

	tracker.MarkSpawned("issue-1")

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

func TestDaemon_OnceMarksSpawned(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	spawnCount := 0
	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-1", Title: "Test Issue", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnCount++
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil
			},
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should have processed an issue")
	}
	if spawnCount == 0 {
		t.Error("Spawner should have been called")
	}

	if !tracker.IsSpawned("issue-1") {
		t.Error("issue should remain marked after successful spawn")
	}
}

func TestDaemon_OnceUnmarksOnFailure(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				if !tracker.IsSpawned(beadsID) {
					t.Error("issue should be marked as spawned during Spawner call")
				}
				return errSpawnFailed
			},
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error: %v", err)
	}
	if result.Processed {
		t.Error("Once() should not have processed (spawn failed)")
	}

	if tracker.IsSpawned("issue-1") {
		t.Error("issue should be unmarked after failed spawn")
	}
}

var errSpawnFailed = &spawnError{msg: "spawn failed"}

type spawnError struct {
	msg string
}

func (e *spawnError) Error() string {
	return e.msg
}

func TestSpawnedIssueTracker_TitleDedup(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	tracker.MarkSpawnedWithTitle("issue-1", "Extract spawn flags phase 1")

	spawned, dupID := tracker.IsTitleSpawned("Extract spawn flags phase 1")
	if !spawned {
		t.Error("title should be detected as spawned")
	}
	if dupID != "issue-1" {
		t.Errorf("dupID = %q, want %q", dupID, "issue-1")
	}

	spawned, _ = tracker.IsTitleSpawned("Different task")
	if spawned {
		t.Error("different title should not be detected as spawned")
	}

	spawned, dupID = tracker.IsTitleSpawned("extract spawn flags phase 1")
	if !spawned {
		t.Error("title matching should be case-insensitive")
	}
	if dupID != "issue-1" {
		t.Errorf("dupID = %q, want %q", dupID, "issue-1")
	}
}

func TestSpawnedIssueTracker_TitleDedup_TTL(t *testing.T) {
	tracker := NewSpawnedIssueTrackerWithTTL(50 * time.Millisecond)

	tracker.MarkSpawnedWithTitle("issue-1", "Extract spawn flags")

	spawned, _ := tracker.IsTitleSpawned("Extract spawn flags")
	if !spawned {
		t.Error("title should be detected immediately after marking")
	}

	time.Sleep(60 * time.Millisecond)

	spawned, _ = tracker.IsTitleSpawned("Extract spawn flags")
	if spawned {
		t.Error("title should not be detected after TTL expires")
	}
}

func TestSpawnedIssueTracker_TitleDedup_Unmark(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	tracker.MarkSpawnedWithTitle("issue-1", "Some task")

	spawned, _ := tracker.IsTitleSpawned("Some task")
	if !spawned {
		t.Error("title should be detected")
	}

	tracker.Unmark("issue-1")

	spawned, _ = tracker.IsTitleSpawned("Some task")
	if spawned {
		t.Error("title should not be detected after unmark")
	}
}

func TestSpawnedIssueTracker_TitleDedup_EmptyTitle(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	tracker.MarkSpawnedWithTitle("issue-1", "")

	spawned, _ := tracker.IsTitleSpawned("")
	if spawned {
		t.Error("empty title should not be detected as spawned")
	}
}

func TestDaemon_ContentDedupSkipsDuplicateTitle(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	spawnCount := 0

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-dup", Title: "Extract spawn flags phase 1", Priority: 0, IssueType: "task", Status: "open"},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnCount++
				return nil
			},
		},
	}

	tracker.MarkSpawnedWithTitle("issue-original", "Extract spawn flags phase 1")

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error: %v", err)
	}
	if result.Processed {
		t.Error("Once() should not process duplicate title")
	}
	if spawnCount != 0 {
		t.Errorf("Spawner should not have been called, got %d calls", spawnCount)
	}
}

func TestSpawnedIssueTracker_FilePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "spawn_cache.json")

	// Create tracker, mark some issues
	tracker1 := NewSpawnedIssueTrackerWithFile(cachePath)
	tracker1.MarkSpawnedWithTitle("issue-1", "Build feature X")
	tracker1.MarkSpawned("issue-2")

	// Verify file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("cache file should exist after MarkSpawned")
	}

	// Create a new tracker from the same file (simulates daemon restart)
	tracker2 := NewSpawnedIssueTrackerWithFile(cachePath)

	if !tracker2.IsSpawned("issue-1") {
		t.Error("issue-1 should be loaded from disk")
	}
	if !tracker2.IsSpawned("issue-2") {
		t.Error("issue-2 should be loaded from disk")
	}
	if tracker2.IsSpawned("issue-3") {
		t.Error("issue-3 should not exist")
	}

	// Title dedup should also survive
	spawned, dupID := tracker2.IsTitleSpawned("Build feature X")
	if !spawned {
		t.Error("title dedup should survive restart")
	}
	if dupID != "issue-1" {
		t.Errorf("dupID = %q, want %q", dupID, "issue-1")
	}
}

func TestSpawnedIssueTracker_FilePersistence_Unmark(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "spawn_cache.json")

	tracker1 := NewSpawnedIssueTrackerWithFile(cachePath)
	tracker1.MarkSpawned("issue-1")
	tracker1.MarkSpawned("issue-2")
	tracker1.Unmark("issue-1")

	// Restart
	tracker2 := NewSpawnedIssueTrackerWithFile(cachePath)
	if tracker2.IsSpawned("issue-1") {
		t.Error("issue-1 should not be present after unmark + restart")
	}
	if !tracker2.IsSpawned("issue-2") {
		t.Error("issue-2 should survive restart")
	}
}

func TestSpawnedIssueTracker_FilePersistence_StaleCleanedOnLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "spawn_cache.json")

	// Write a cache file with an entry that has an old timestamp
	oldTime := time.Now().Add(-7 * time.Hour) // older than 6h TTL
	cacheData := `{"spawned":{"issue-old":"` + oldTime.Format(time.RFC3339Nano) + `","issue-fresh":"` + time.Now().Format(time.RFC3339Nano) + `"},"spawned_titles":{}}`
	if err := os.WriteFile(cachePath, []byte(cacheData), 0644); err != nil {
		t.Fatal(err)
	}

	tracker := NewSpawnedIssueTrackerWithFile(cachePath)
	if tracker.IsSpawned("issue-old") {
		t.Error("stale entry should be cleaned on load")
	}
	if !tracker.IsSpawned("issue-fresh") {
		t.Error("fresh entry should survive load")
	}
}

func TestSpawnedIssueTracker_FilePersistence_CorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "spawn_cache.json")

	// Write corrupt data
	if err := os.WriteFile(cachePath, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should not panic, starts empty (fail-open)
	tracker := NewSpawnedIssueTrackerWithFile(cachePath)
	if tracker.Count() != 0 {
		t.Errorf("tracker should start empty on corrupt file, got %d", tracker.Count())
	}

	// Should still work normally after corrupt load
	tracker.MarkSpawned("issue-1")
	if !tracker.IsSpawned("issue-1") {
		t.Error("tracker should work after corrupt file recovery")
	}
}

func TestSpawnedIssueTracker_FilePersistence_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "spawn_cache.json")

	// No file exists — should start empty without error
	tracker := NewSpawnedIssueTrackerWithFile(cachePath)
	if tracker.Count() != 0 {
		t.Errorf("tracker should start empty when no file, got %d", tracker.Count())
	}
}

func TestSpawnedIssueTracker_InMemoryNoPersistence(t *testing.T) {
	// Existing constructors should NOT write files
	tracker := NewSpawnedIssueTracker()
	tracker.MarkSpawned("issue-1")
	// No crash, no file written — this is the test (no assertions needed beyond no panic)
	if !tracker.IsSpawned("issue-1") {
		t.Error("in-memory tracker should still work")
	}
}

func TestDefaultSpawnCachePath(t *testing.T) {
	path := DefaultSpawnCachePath()
	if path == "" {
		t.Skip("cannot determine home dir")
	}
	if filepath.Base(path) != "spawn_cache.json" {
		t.Errorf("expected spawn_cache.json, got %s", filepath.Base(path))
	}
	if filepath.Base(filepath.Dir(path)) != ".orch" {
		t.Errorf("expected .orch dir, got %s", filepath.Dir(path))
	}
}

func TestDaemon_ContentDedupAllowsDifferentTitle(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	spawnCount := 0

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-new", Title: "Add new feature X", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnCount++
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil
			},
		},
	}

	tracker.MarkSpawnedWithTitle("issue-other", "Extract spawn flags phase 1")

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error: %v", err)
	}
	if !result.Processed {
		t.Errorf("Once() should process issue with different title, got message: %s", result.Message)
	}
	if spawnCount != 1 {
		t.Errorf("Spawner should have been called once, got %d", spawnCount)
	}
}

// TestDaemon_SpawnIssueRejectsRecentlySpawned verifies the defense-in-depth
// IsSpawned() check in spawnIssue(). Even if NextIssueExcluding is bypassed,
// spawnIssue() itself should reject issues in the spawn cache.
func TestDaemon_SpawnIssueRejectsRecentlySpawned(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	spawnCount := 0

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnCount++
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				return nil
			},
		},
	}

	// Pre-mark the issue as spawned (simulates daemon restart with cache loaded)
	tracker.MarkSpawned("issue-1")

	// Call spawnIssue directly (bypassing NextIssueExcluding)
	issue := &Issue{ID: "issue-1", Title: "Test", IssueType: "feature", Status: "open"}
	result, _, err := d.spawnIssue(issue, "feature-impl", "opus")
	if err != nil {
		t.Fatalf("spawnIssue() error: %v", err)
	}
	if result.Processed {
		t.Error("spawnIssue() should NOT process a recently-spawned issue")
	}
	if spawnCount != 0 {
		t.Errorf("Spawner should not have been called, got %d calls", spawnCount)
	}
}

// TestSpawnedIssueTracker_SpawnCount verifies spawn count tracking.
func TestSpawnedIssueTracker_SpawnCount(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	if count := tracker.SpawnCount("issue-1"); count != 0 {
		t.Errorf("SpawnCount should be 0 initially, got %d", count)
	}

	tracker.MarkSpawned("issue-1")
	if count := tracker.SpawnCount("issue-1"); count != 1 {
		t.Errorf("SpawnCount should be 1 after first mark, got %d", count)
	}

	tracker.MarkSpawned("issue-1")
	if count := tracker.SpawnCount("issue-1"); count != 2 {
		t.Errorf("SpawnCount should be 2 after second mark, got %d", count)
	}

	tracker.MarkSpawnedWithTitle("issue-1", "Some title")
	if count := tracker.SpawnCount("issue-1"); count != 3 {
		t.Errorf("SpawnCount should be 3 after MarkSpawnedWithTitle, got %d", count)
	}
}

// TestSpawnedIssueTracker_SpawnCount_FilePersistence verifies spawn counts survive daemon restarts.
func TestSpawnedIssueTracker_SpawnCount_FilePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "spawn_cache.json")

	tracker1 := NewSpawnedIssueTrackerWithFile(cachePath)
	tracker1.MarkSpawned("issue-1")
	tracker1.MarkSpawned("issue-1")
	tracker1.MarkSpawned("issue-1")

	if count := tracker1.SpawnCount("issue-1"); count != 3 {
		t.Errorf("SpawnCount should be 3, got %d", count)
	}

	// Simulate daemon restart
	tracker2 := NewSpawnedIssueTrackerWithFile(cachePath)
	if count := tracker2.SpawnCount("issue-1"); count != 3 {
		t.Errorf("SpawnCount should survive restart, got %d (want 3)", count)
	}
}

// TestDaemon_OrphanDetectionPreservesSpawnCache is an integration test that verifies
// the full cycle: spawn → orphan detection → spawn attempt blocked by cache.
func TestDaemon_OrphanDetectionPreservesSpawnCache(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	spawnCount := 0

	// Step 1: Daemon spawns an issue
	d := &Daemon{
		Config: Config{
			OrphanDetectionEnabled:  true,
			OrphanDetectionInterval: 30 * time.Minute,
			OrphanAgeThreshold:      time.Hour,
		},
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-1", Title: "Build feature", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnCount++
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				return nil
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "issue-1", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Build feature"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool {
				return false // Agent died
			},
		},
	}

	// First spawn succeeds
	result, _ := d.Once()
	if !result.Processed {
		t.Fatalf("First spawn should succeed, got: %s", result.Message)
	}
	if spawnCount != 1 {
		t.Fatalf("Expected 1 spawn, got %d", spawnCount)
	}

	// Step 2: Orphan detection runs — resets status to open but PRESERVES spawn cache
	orphanResult := d.RunPeriodicOrphanDetection()
	if orphanResult.ResetCount != 1 {
		t.Fatalf("Expected 1 orphan reset, got %d", orphanResult.ResetCount)
	}
	if !tracker.IsSpawned("issue-1") {
		t.Fatal("Spawn cache entry should be preserved after orphan detection")
	}

	// Step 3: Daemon tries to spawn again — should be blocked by spawn cache
	result2, _ := d.Once()
	if result2.Processed {
		t.Error("Second spawn should be BLOCKED by spawn cache (preventing thrash loop)")
	}
	if spawnCount != 1 {
		t.Errorf("Spawner should still have 1 call (no duplicate), got %d", spawnCount)
	}
}

func TestDaemon_PreventsDuplicateSpawns(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	spawnCount := 0

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnCount++
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil
			},
		},
	}

	result1, _ := d.Once()
	if !result1.Processed {
		t.Error("First Once() should have processed")
	}
	if spawnCount != 1 {
		t.Errorf("Spawner should have been called once, got %d", spawnCount)
	}

	result2, _ := d.Once()
	if result2.Processed {
		t.Error("Second Once() should not process (issue already spawned)")
	}
	if spawnCount != 1 {
		t.Errorf("Spawner should still be 1 call, got %d", spawnCount)
	}

	if result2.Message != "No spawnable issues in queue" {
		t.Errorf("Expected 'No spawnable issues in queue', got %q", result2.Message)
	}
}
