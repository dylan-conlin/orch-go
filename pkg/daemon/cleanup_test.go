package daemon

import (
	"fmt"
	"testing"
	"time"
)

// TestIsWindowStale_ActiveInOpenCode verifies that windows with matching
// OpenCode sessions are never considered stale.
func TestIsWindowStale_ActiveInOpenCode(t *testing.T) {
	activeIDs := map[string]bool{"proj-123": true}
	// Should not be stale even if beads status would say closed
	statusFunc := func(id string) (string, error) {
		return "closed", nil
	}
	if isWindowStale("proj-123", activeIDs, statusFunc) {
		t.Error("window should not be stale when active in OpenCode")
	}
}

// TestIsWindowStale_ClaudeWorkerInProgress verifies that Claude CLI workers
// (no OpenCode session) with in_progress beads status are protected.
// This is the core bug fix: before this change, all Claude CLI workers
// appeared stale because they had no OpenCode sessions.
func TestIsWindowStale_ClaudeWorkerInProgress(t *testing.T) {
	activeIDs := map[string]bool{} // No OpenCode sessions
	statusFunc := func(id string) (string, error) {
		return "in_progress", nil
	}
	if isWindowStale("proj-456", activeIDs, statusFunc) {
		t.Error("Claude CLI worker with in_progress issue should NOT be stale")
	}
}

// TestIsWindowStale_ClaudeWorkerOpen verifies open-status issues are protected.
func TestIsWindowStale_ClaudeWorkerOpen(t *testing.T) {
	activeIDs := map[string]bool{}
	statusFunc := func(id string) (string, error) {
		return "open", nil
	}
	if isWindowStale("proj-789", activeIDs, statusFunc) {
		t.Error("window with open issue should NOT be stale")
	}
}

// TestIsWindowStale_ClosedIssue verifies that windows for closed issues ARE stale.
func TestIsWindowStale_ClosedIssue(t *testing.T) {
	activeIDs := map[string]bool{}
	statusFunc := func(id string) (string, error) {
		return "closed", nil
	}
	if !isWindowStale("proj-done", activeIDs, statusFunc) {
		t.Error("window with closed issue should be stale")
	}
}

// TestIsWindowStale_BeadsUnavailable verifies fail-safe: when beads status
// can't be determined, the window is kept alive (not killed).
func TestIsWindowStale_BeadsUnavailable(t *testing.T) {
	activeIDs := map[string]bool{}
	statusFunc := func(id string) (string, error) {
		return "", fmt.Errorf("beads unavailable")
	}
	if isWindowStale("proj-unknown", activeIDs, statusFunc) {
		t.Error("window should NOT be stale when beads status is unavailable (fail-safe)")
	}
}

// TestIsWindowStale_CrossProjectWorker verifies that workers from other projects
// (different beads databases) are protected when their issue is still active.
func TestIsWindowStale_CrossProjectWorker(t *testing.T) {
	// Simulate: daemon is in orch-go, worker is from price-watch
	// The beads ID prefix differs, but GetBeadsIssueStatus should still resolve
	activeIDs := map[string]bool{} // No OpenCode sessions for this worker
	statusFunc := func(id string) (string, error) {
		if id == "pw-123" {
			return "in_progress", nil
		}
		return "closed", nil
	}
	if isWindowStale("pw-123", activeIDs, statusFunc) {
		t.Error("cross-project worker with in_progress issue should NOT be stale")
	}
}

func TestRunPeriodicCleanupRunsWhenDue(t *testing.T) {
	called := 0
	d := &Daemon{
		Config: Config{
			CleanupEnabled:  true,
			CleanupInterval: time.Minute,
		},
		Cleaner: &mockSessionCleaner{CleanupFunc: func(config Config) (int, string, error) {
			called++
			return 2, "Closed 2 stale tmux windows", nil
		}},
	}

	result := d.RunPeriodicCleanup()
	if called != 1 {
		t.Fatalf("RunPeriodicCleanup should call cleanup func once, got %d", called)
	}
	if result == nil {
		t.Fatal("RunPeriodicCleanup should return result when due")
	}
	if result.Deleted != 2 {
		t.Fatalf("CleanupResult.Deleted = %d, want 2", result.Deleted)
	}
	if result.Message == "" {
		t.Fatal("CleanupResult.Message should not be empty")
	}
	if d.lastCleanup.IsZero() {
		t.Fatal("lastCleanup should be updated after successful cleanup")
	}
}

func TestRunPeriodicCleanupSkipsWhenNotDue(t *testing.T) {
	called := 0
	d := &Daemon{
		Config: Config{
			CleanupEnabled:  true,
			CleanupInterval: time.Hour,
		},
		lastCleanup: time.Now(),
		Cleaner: &mockSessionCleaner{CleanupFunc: func(config Config) (int, string, error) {
			called++
			return 1, "Closed 1 stale tmux window", nil
		}},
	}

	result := d.RunPeriodicCleanup()
	if result != nil {
		t.Fatal("RunPeriodicCleanup should return nil when cleanup is not due")
	}
	if called != 0 {
		t.Fatalf("RunPeriodicCleanup should not call cleanup func, got %d calls", called)
	}
}
