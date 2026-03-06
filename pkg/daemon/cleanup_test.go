package daemon

import (
	"fmt"
	"os"
	"path/filepath"
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
	cfg := Config{
		CleanupEnabled:  true,
		CleanupInterval: time.Minute,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
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
	if d.Scheduler.LastRunTime(TaskCleanup).IsZero() {
		t.Fatal("lastCleanup should be updated after successful cleanup")
	}
}

func TestRunPeriodicCleanupSkipsWhenNotDue(t *testing.T) {
	called := 0
	cfg := Config{
		CleanupEnabled:  true,
		CleanupInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Cleaner: &mockSessionCleaner{CleanupFunc: func(config Config) (int, string, error) {
			called++
			return 1, "Closed 1 stale tmux window", nil
		}},
	}
	d.Scheduler.SetLastRun(TaskCleanup, time.Now())

	result := d.RunPeriodicCleanup()
	if result != nil {
		t.Fatal("RunPeriodicCleanup should return nil when cleanup is not due")
	}
	if called != 0 {
		t.Fatalf("RunPeriodicCleanup should not call cleanup func, got %d calls", called)
	}
}

// TestExpireArchivedWorkspaces tests TTL-based deletion of old archived workspaces.
func TestExpireArchivedWorkspaces(t *testing.T) {
	tmpDir := t.TempDir()
	archivedDir := filepath.Join(tmpDir, ".orch", "workspace", "archived")
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		t.Fatalf("Failed to create archived dir: %v", err)
	}

	// Create old workspace (40 days old via .spawn_time)
	oldWs := filepath.Join(archivedDir, "og-feat-old-01jan")
	if err := os.MkdirAll(oldWs, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	oldTime := time.Now().AddDate(0, 0, -40)
	if err := os.WriteFile(filepath.Join(oldWs, ".spawn_time"), []byte(fmt.Sprintf("%d", oldTime.UnixNano())), 0644); err != nil {
		t.Fatalf("Failed to write spawn time: %v", err)
	}

	// Create recent workspace (5 days old)
	recentWs := filepath.Join(archivedDir, "og-feat-recent-28feb")
	if err := os.MkdirAll(recentWs, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	recentTime := time.Now().AddDate(0, 0, -5)
	if err := os.WriteFile(filepath.Join(recentWs, ".spawn_time"), []byte(fmt.Sprintf("%d", recentTime.UnixNano())), 0644); err != nil {
		t.Fatalf("Failed to write spawn time: %v", err)
	}

	deleted, err := expireArchivedWorkspaces(tmpDir, 30)
	if err != nil {
		t.Fatalf("expireArchivedWorkspaces failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 deleted, got %d", deleted)
	}

	// Old workspace should be gone
	if _, err := os.Stat(oldWs); !os.IsNotExist(err) {
		t.Error("Old workspace should have been deleted")
	}
	// Recent workspace should remain
	if _, err := os.Stat(recentWs); os.IsNotExist(err) {
		t.Error("Recent workspace should still exist")
	}
}

// --- isWindowStaleBatch tests ---

func TestIsWindowStaleBatch_ActiveInOpenCode(t *testing.T) {
	openCodeIDs := map[string]bool{"proj-123": true}
	openBeadsIDs := map[string]bool{}
	if isWindowStaleBatch("proj-123", openCodeIDs, openBeadsIDs) {
		t.Error("window should not be stale when active in OpenCode")
	}
}

func TestIsWindowStaleBatch_OpenIssue(t *testing.T) {
	openCodeIDs := map[string]bool{}
	openBeadsIDs := map[string]bool{"proj-456": true}
	if isWindowStaleBatch("proj-456", openCodeIDs, openBeadsIDs) {
		t.Error("window with open beads issue should NOT be stale")
	}
}

func TestIsWindowStaleBatch_ClosedIssue(t *testing.T) {
	openCodeIDs := map[string]bool{}
	openBeadsIDs := map[string]bool{"proj-other": true} // proj-done not in set
	if !isWindowStaleBatch("proj-done", openCodeIDs, openBeadsIDs) {
		t.Error("window with closed issue (not in open set) should be stale")
	}
}

func TestIsWindowStaleBatch_NilBeadsFailSafe(t *testing.T) {
	openCodeIDs := map[string]bool{}
	// nil openBeadsIDs means batch fetch failed — fail-safe should keep window alive
	if isWindowStaleBatch("proj-unknown", openCodeIDs, nil) {
		t.Error("window should NOT be stale when beads batch fetch failed (fail-safe)")
	}
}

func TestIsWindowStaleBatch_EmptyBeadsSet(t *testing.T) {
	openCodeIDs := map[string]bool{}
	openBeadsIDs := map[string]bool{} // empty = no open issues
	if !isWindowStaleBatch("proj-done", openCodeIDs, openBeadsIDs) {
		t.Error("window should be stale when no open issues exist")
	}
}

func TestIsWindowStaleBatch_BothSources(t *testing.T) {
	openCodeIDs := map[string]bool{"proj-oc": true}
	openBeadsIDs := map[string]bool{"proj-beads": true}
	// proj-oc protected by OpenCode
	if isWindowStaleBatch("proj-oc", openCodeIDs, openBeadsIDs) {
		t.Error("proj-oc should be protected by OpenCode")
	}
	// proj-beads protected by beads
	if isWindowStaleBatch("proj-beads", openCodeIDs, openBeadsIDs) {
		t.Error("proj-beads should be protected by open beads issue")
	}
	// proj-neither in neither set
	if !isWindowStaleBatch("proj-neither", openCodeIDs, openBeadsIDs) {
		t.Error("proj-neither should be stale (not in either set)")
	}
}

// TestExpireArchivedWorkspaces_NoDir tests graceful handling when no archived dir exists.
func TestExpireArchivedWorkspaces_NoDir(t *testing.T) {
	tmpDir := t.TempDir()
	deleted, err := expireArchivedWorkspaces(tmpDir, 30)
	if err != nil {
		t.Fatalf("Should not fail when no archived dir: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted, got %d", deleted)
	}
}
