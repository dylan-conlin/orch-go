package daemon

import (
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
