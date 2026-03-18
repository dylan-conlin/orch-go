package daemon

import (
	"fmt"
	"testing"
)

func TestDaemon_SpawnIssue_StatusUpdateFailureReleasesSlot(t *testing.T) {
	pool := NewWorkerPool(1)
	spawnCalled := false
	d := &Daemon{
		Pool: pool,
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return fmt.Errorf("update failed")
		}},
	}

	issue := &Issue{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"}
	result, slot, err := d.spawnIssue(issue, "feature-impl", "sonnet")
	if err != nil {
		t.Fatalf("spawnIssue() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("spawnIssue() expected result on status update failure")
	}
	if result.Processed {
		t.Error("spawnIssue() expected Processed=false on status update failure")
	}
	if result.Error == nil {
		t.Error("spawnIssue() expected Error to be set on status update failure")
	}
	if spawnCalled {
		t.Error("spawnIssue() should not call spawnFunc when status update fails")
	}
	if slot != nil {
		t.Error("spawnIssue() expected nil slot on status update failure")
	}
	if pool.Active() != 0 {
		t.Errorf("Pool.Active() = %d, want 0 (slot should be released on error)", pool.Active())
	}
}

// =============================================================================
// Tests for Sticky Spawn Failure Fix
// =============================================================================

// TestOnceExcluding_NonErrorSkip_ContinuesToNextIssue verifies that when
// OnceExcluding returns a non-error skip for an issue (e.g., status already
// in_progress), adding that issue to the skip map and calling again processes
// the next issue in the queue. This is the core fix for sticky spawn failures:
// non-error dedup returns must be skippable so lower-priority issues get tried.
func TestOnceExcluding_NonErrorSkip_ContinuesToNextIssue(t *testing.T) {
	spawnedIDs := []string{}
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-A", Title: "High priority", Priority: 0, IssueType: "feature", Status: "open"},
					{ID: "issue-B", Title: "Lower priority", Priority: 1, IssueType: "task", Status: "open"},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				// Issue A is already in_progress (dedup case)
				if beadsID == "issue-A" {
					return "in_progress", nil
				}
				return "open", nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			spawnedIDs = append(spawnedIDs, beadsID)
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	// First call: issue-A should be skipped (non-error: status is in_progress)
	skip := make(map[string]bool)
	result1, err := d.OnceExcluding(skip)
	if err != nil {
		t.Fatalf("OnceExcluding() error: %v", err)
	}
	if result1.Processed {
		t.Fatal("OnceExcluding() should not process issue-A (status is in_progress)")
	}
	if result1.Issue == nil || result1.Issue.ID != "issue-A" {
		t.Fatalf("OnceExcluding() should return issue-A, got %v", result1.Issue)
	}
	if result1.Error != nil {
		t.Fatalf("OnceExcluding() non-error skip should have nil Error, got %v", result1.Error)
	}

	// Add issue-A to skip map (simulating what the daemon loop now does)
	skip[result1.Issue.ID] = true

	// Second call: issue-B should be tried and spawned
	result2, err := d.OnceExcluding(skip)
	if err != nil {
		t.Fatalf("OnceExcluding() second call error: %v", err)
	}
	if !result2.Processed {
		t.Fatalf("OnceExcluding() should process issue-B, got message: %s", result2.Message)
	}
	if result2.Issue == nil || result2.Issue.ID != "issue-B" {
		t.Fatalf("OnceExcluding() should spawn issue-B, got %v", result2.Issue)
	}
	if len(spawnedIDs) != 1 || spawnedIDs[0] != "issue-B" {
		t.Errorf("expected spawn of issue-B only, got %v", spawnedIDs)
	}
}

// TestOnceExcluding_SpawnFailure_RetriedWithFreshSkipMap verifies that issues
// that fail to spawn in one cycle are retried when called with a fresh skip map
// (simulating the start of a new poll cycle).
func TestOnceExcluding_SpawnFailure_RetriedWithFreshSkipMap(t *testing.T) {
	callCount := 0
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-1", Title: "Retry test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			callCount++
			if callCount == 1 {
				return fmt.Errorf("transient spawn failure")
			}
			return nil // succeeds on retry
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	// Cycle 1: spawn fails
	skip1 := make(map[string]bool)
	result1, err := d.OnceExcluding(skip1)
	if err != nil {
		t.Fatalf("Cycle 1: OnceExcluding() error: %v", err)
	}
	if result1.Processed {
		t.Fatal("Cycle 1: should not process (spawn failed)")
	}
	if result1.Error == nil {
		t.Fatal("Cycle 1: should have Error set")
	}

	// Cycle 2: fresh skip map (simulates new poll cycle), should retry
	skip2 := make(map[string]bool)
	result2, err := d.OnceExcluding(skip2)
	if err != nil {
		t.Fatalf("Cycle 2: OnceExcluding() error: %v", err)
	}
	if !result2.Processed {
		t.Fatalf("Cycle 2: should process on retry, got message: %s", result2.Message)
	}
	if callCount != 2 {
		t.Errorf("expected 2 spawn calls (fail + retry), got %d", callCount)
	}
}

// TestSpawnIssue_PhaseCompleteError_AttemptsAutoComplete verifies that when
// SpawnWork returns a "Phase: Complete but is not closed" error, the daemon
// attempts auto-completion instead of just rolling back and retrying every cycle.
func TestSpawnIssue_PhaseCompleteError_AttemptsAutoComplete(t *testing.T) {
	autoCompleteCalled := false
	autoCompleteBeadsID := ""
	autoCompleteWorkdir := ""

	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "toolshed-a8m", Title: "Cross-repo task", Priority: 0, IssueType: "task", Status: "open", ProjectDir: "/tmp/toolshed"},
				}, nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			return fmt.Errorf("failed to spawn work: exit status 1: issue %s has Phase: Complete but is not closed. Run 'orch complete %s' first", beadsID, beadsID)
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
		AutoCompleter: &mockAutoCompleter{
			CompleteFunc: func(beadsID, workdir string) error {
				autoCompleteCalled = true
				autoCompleteBeadsID = beadsID
				autoCompleteWorkdir = workdir
				return nil
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() error: %v", err)
	}

	if !autoCompleteCalled {
		t.Fatal("AutoCompleter.Complete should be called when SpawnWork returns Phase: Complete error")
	}
	if autoCompleteBeadsID != "toolshed-a8m" {
		t.Errorf("AutoCompleter.Complete beadsID = %q, want %q", autoCompleteBeadsID, "toolshed-a8m")
	}
	if autoCompleteWorkdir != "/tmp/toolshed" {
		t.Errorf("AutoCompleter.Complete workdir = %q, want %q", autoCompleteWorkdir, "/tmp/toolshed")
	}

	// Should not be marked as processed (it wasn't spawned, it was auto-completed)
	if result.Processed {
		t.Error("result.Processed should be false (issue was auto-completed, not spawned)")
	}
	// Error should be nil since auto-complete succeeded
	if result.Error != nil {
		t.Errorf("result.Error should be nil after successful auto-complete, got: %v", result.Error)
	}
	// Message should indicate auto-completion
	if result.Message == "" {
		t.Error("result.Message should describe auto-completion")
	}
}

// TestSpawnIssue_PhaseCompleteError_AutoCompleteFails_FallsBack verifies that
// when auto-completion fails, the daemon falls back to normal error handling
// (rollback + skip).
func TestSpawnIssue_PhaseCompleteError_AutoCompleteFails_FallsBack(t *testing.T) {
	autoCompleteCalled := false
	statusUpdates := map[string]string{}

	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "toolshed-b2c", Title: "Cross-repo task", Priority: 0, IssueType: "task", Status: "open", ProjectDir: "/tmp/toolshed"},
				}, nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			return fmt.Errorf("failed to spawn work: exit status 1: issue %s has Phase: Complete but is not closed", beadsID)
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			statusUpdates[beadsID] = status
			return nil
		}},
		AutoCompleter: &mockAutoCompleter{
			CompleteFunc: func(beadsID, workdir string) error {
				autoCompleteCalled = true
				return fmt.Errorf("orch complete failed: gate failure")
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() error: %v", err)
	}

	if !autoCompleteCalled {
		t.Fatal("AutoCompleter.Complete should be called even when it will fail")
	}

	// Should fall back to normal error handling
	if result.Processed {
		t.Error("result.Processed should be false")
	}
	if result.Error == nil {
		t.Error("result.Error should be set after auto-complete failure")
	}
}

// TestSpawnIssue_PhaseCompleteError_NoAutoCompleter_SkipsGracefully verifies
// that when AutoCompleter is nil, the daemon handles the Phase: Complete error
// gracefully without panicking.
func TestSpawnIssue_PhaseCompleteError_NoAutoCompleter_SkipsGracefully(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "toolshed-c3d", Title: "Cross-repo task", Priority: 0, IssueType: "task", Status: "open", ProjectDir: "/tmp/toolshed"},
				}, nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			return fmt.Errorf("failed to spawn work: exit status 1: issue %s has Phase: Complete but is not closed", beadsID)
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
		// AutoCompleter intentionally nil
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() error: %v", err)
	}

	// Should skip gracefully with error
	if result.Processed {
		t.Error("result.Processed should be false")
	}
	if result.Error == nil {
		t.Error("result.Error should be set")
	}
}
