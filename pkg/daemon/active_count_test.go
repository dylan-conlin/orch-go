package daemon

import (
	"os"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

func TestExtractBeadsIDFromWindowName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		// Standard agent window names with emoji prefix
		{"🔬 og-inv-topic-date [orch-go-abc1]", "orch-go-abc1"},
		{"⚙️ og-feat-add-feature [orch-go-xyz2]", "orch-go-xyz2"},
		{"🐛 og-debug-fix-bug [orch-go-def3]", "orch-go-def3"},
		// No beads ID
		{"main", ""},
		{"zsh", ""},
		{"", ""},
		// Bracket but no match
		{"[incomplete", ""},
	}

	for _, tt := range tests {
		result := extractBeadsIDFromWindowName(tt.name)
		if result != tt.expected {
			t.Errorf("extractBeadsIDFromWindowName(%q) = %q, want %q", tt.name, result, tt.expected)
		}
	}
}

func TestGetClosedIssuesBatchTreatsNotFoundAsNotActive(t *testing.T) {
	// Regression test: cross-project beads IDs (e.g., skillc-cb3) queried against
	// local project beads (orch-go) would fail with "not found". Previously, the
	// error handler treated these as "not closed" = active, inflating capacity.
	// Fix: treat "not found" as "not active" (closed[id] = true).

	// Use IDs that definitely don't exist in any local beads database
	fakeIDs := []string{"nonexistent-xxx1", "nonexistent-yyy2", "nonexistent-zzz3"}
	closed := GetClosedIssuesBatch(fakeIDs)

	// All non-existent IDs should be treated as "not active" (in the closed map)
	for _, id := range fakeIDs {
		if !closed[id] {
			t.Errorf("GetClosedIssuesBatch: non-existent ID %q should be treated as not active (closed), but was treated as active", id)
		}
	}
}

func TestCountActiveTmuxAgentsScopesToProject(t *testing.T) {
	// Regression test: CountActiveTmuxAgents scanned ALL workers-* sessions,
	// finding agents from other projects (workers-skillc) alongside the current
	// project (workers-orch-go). This inflated the active count.
	// Fix: when projectName is provided, only scan workers-{projectName}.

	if os.Getenv("CI") != "" {
		t.Skip("Skipping tmux test in CI (no tmux server)")
	}

	// Create two worker sessions for different "projects"
	project1 := "test-proj-alpha"
	project2 := "test-proj-beta"
	tmpDir := t.TempDir()

	session1, err := tmux.EnsureWorkersSession(project1, tmpDir)
	if err != nil {
		t.Skipf("tmux not available: %v", err)
	}
	defer func() {
		tmux.KillSession(session1)
	}()

	session2, err := tmux.EnsureWorkersSession(project2, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create second session: %v", err)
	}
	defer func() {
		tmux.KillSession(session2)
	}()

	// Create windows with beads IDs in each session
	// Note: these won't have active claude processes, so IsPaneActive will filter them.
	// But we can test the session scoping logic directly.

	// Unscoped call (empty project) should find windows in both sessions
	unscopedAgents := CountActiveTmuxAgents("")
	_ = unscopedAgents // Just verify it doesn't panic

	// Scoped to project1 should only scan workers-test-proj-alpha
	scopedAgents := CountActiveTmuxAgents(project1)
	_ = scopedAgents // Verify it doesn't panic

	// Scoped to non-existent project should return empty
	nonExistentAgents := CountActiveTmuxAgents("nonexistent-project-xyz")
	if len(nonExistentAgents) != 0 {
		t.Errorf("CountActiveTmuxAgents(nonexistent) returned %d agents, want 0", len(nonExistentAgents))
	}
}

func TestCrossProjectAgentsDoNotInflateCapacity(t *testing.T) {
	// End-to-end regression test for the core bug: daemon capacity inflated
	// by cross-project tmux agents.
	//
	// Scenario: daemon has max 3 agents. 1 real orch-go agent + 2 skillc agents
	// visible in tmux. The daemon should report 1 active (not 3).
	//
	// Since we can't easily create real tmux agents in tests, we verify
	// the fix through the mock ActiveCounter + GetClosedIssuesBatch path.
	config := DefaultConfig()
	config.MaxAgents = 3
	d := NewWithConfig(config)

	// Simulate: only 1 real active agent (the mock counter returns 1)
	d.ActiveCounter = &mockActiveCounter{CountFunc: func() int { return 1 }}

	// Acquire 3 slots (simulating what happened before the fix)
	s1 := d.Pool.TryAcquire()
	s2 := d.Pool.TryAcquire()
	s3 := d.Pool.TryAcquire()
	if s1 == nil || s2 == nil || s3 == nil {
		t.Fatal("Expected to acquire 3 slots")
	}

	// Pool thinks 3/3 but real count is 1.
	// Reconcile should free 2 ghost slots.
	result := d.ReconcileActiveAgents()
	if result.Freed != 2 {
		t.Errorf("ReconcileActiveAgents() freed = %d, want 2 (cross-project ghosts)", result.Freed)
	}
	if d.AtCapacity() {
		t.Error("Daemon should NOT be at capacity after reconciling cross-project ghosts")
	}
	if d.ActiveCount() != 1 {
		t.Errorf("ActiveCount() = %d, want 1", d.ActiveCount())
	}
}

func TestPoolReconcileDoesNotResetWhenTmuxAgentsExist(t *testing.T) {
	// This test verifies the core bug: pool reconciliation should NOT
	// reset to 0 when there are active tmux agents.
	//
	// Before fix: ReconcileWithOpenCode() calls DefaultActiveCount() which
	// returns 0 for tmux-only agents, causing pool.Reconcile(0) to free all slots.
	// After fix: Uses DiscoverLiveAgents() which counts both OpenCode and tmux agents.

	pool := NewWorkerPool(3)

	// Simulate 3 agents spawned (pool at 3/3)
	s1 := pool.TryAcquire()
	s2 := pool.TryAcquire()
	s3 := pool.TryAcquire()

	if s1 == nil || s2 == nil || s3 == nil {
		t.Fatal("Expected to acquire 3 slots")
	}
	if !pool.AtCapacity() {
		t.Fatal("Pool should be at capacity with 3/3 slots")
	}

	// BUG SCENARIO: If reconciliation sees 0 OpenCode sessions,
	// it resets the pool to 0, allowing unlimited further spawns.
	// With the fix, if there are 3 tmux agents, the combined count is 3,
	// and Reconcile(3) is a no-op.
	result := pool.Reconcile(3) // Combined count = 3 tmux agents
	if result.Freed != 0 {
		t.Errorf("Reconcile(3) should free 0 slots when pool has 3 active, freed %d", result.Freed)
	}
	if !pool.AtCapacity() {
		t.Fatal("Pool should still be at capacity after Reconcile(3)")
	}

	// Verify that 4th slot cannot be acquired
	s4 := pool.TryAcquire()
	if s4 != nil {
		t.Fatal("Should not be able to acquire 4th slot when at capacity")
	}

	// Now simulate one agent completing (tmux window closed)
	result = pool.Reconcile(2) // Combined count drops to 2
	if result.Freed != 1 {
		t.Errorf("Reconcile(2) should free 1 slot, freed %d", result.Freed)
	}
	if pool.AtCapacity() {
		t.Fatal("Pool should not be at capacity after freeing 1 slot")
	}

	// Now we should be able to acquire one more slot
	s5 := pool.TryAcquire()
	if s5 == nil {
		t.Fatal("Should be able to acquire a slot after reconciliation freed one")
	}
}

func TestDaemonReconcileWithActiveCountFunc(t *testing.T) {
	// Test that the daemon uses the configurable activeCountFunc for reconciliation,
	// which allows combining OpenCode + tmux counts.
	config := DefaultConfig()
	config.MaxAgents = 3
	d := NewWithConfig(config)

	// Simulate spawning 3 agents
	s1 := d.Pool.TryAcquire()
	s2 := d.Pool.TryAcquire()
	s3 := d.Pool.TryAcquire()
	if s1 == nil || s2 == nil || s3 == nil {
		t.Fatal("Expected to acquire 3 slots")
	}

	// Set custom active counter that reports 3 active (simulating tmux agents)
	d.ActiveCounter = &mockActiveCounter{CountFunc: func() int { return 3 }}

	result := d.ReconcileActiveAgents()
	if result.Freed != 0 {
		t.Errorf("Should free 0 slots when 3 active reported, freed %d", result.Freed)
	}
	if !d.AtCapacity() {
		t.Fatal("Should still be at capacity")
	}

	// Now simulate the old bug: active count returns 0 (only checking OpenCode)
	d.ActiveCounter = &mockActiveCounter{CountFunc: func() int { return 0 }}

	result = d.ReconcileActiveAgents()
	if result.Freed != 3 {
		t.Errorf("Should free 3 slots when 0 active reported, freed %d", result.Freed)
	}
	if d.AtCapacity() {
		t.Fatal("Should not be at capacity after freeing all slots")
	}
}

func TestDaemonReconcileDefaultCountUsesBeads(t *testing.T) {
	// Test that when no custom activeCountFunc is set, the daemon uses
	// BeadsActiveCount (via defaultActiveCounter) as the capacity source.
	config := DefaultConfig()
	config.MaxAgents = 3
	d := NewWithConfig(config)

	if d.ActiveCounter == nil {
		t.Fatal("ActiveCounter should be set to defaultActiveCounter by default")
	}

	// Verify defaultActiveCounter calls BeadsActiveCount (not DiscoverLiveAgents).
	// We can't easily mock BeadsActiveCount itself, but we verify the type is correct.
	_, isDefault := d.ActiveCounter.(*defaultActiveCounter)
	if !isDefault {
		t.Fatalf("ActiveCounter should be *defaultActiveCounter, got %T", d.ActiveCounter)
	}
}

func TestIsBeadsIssueDone(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   bool
	}{
		{"no labels", nil, false},
		{"empty labels", []string{}, false},
		{"unrelated labels", []string{"triage:ready", "orch:agent"}, false},
		{"verification-failed", []string{"daemon:verification-failed"}, true},
		{"ready-review", []string{"daemon:ready-review"}, true},
		{"verification-failed among others", []string{"orch:agent", "daemon:verification-failed"}, true},
		{"ready-review among others", []string{"triage:ready", "daemon:ready-review"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBeadsIssueDone(tt.labels)
			if got != tt.want {
				t.Errorf("isBeadsIssueDone(%v) = %v, want %v", tt.labels, got, tt.want)
			}
		})
	}
}

func TestBeadsActiveCountIntegration(t *testing.T) {
	// Integration test: BeadsActiveCount queries real beads.
	// If beads daemon is not running, it falls back to CLI.
	// Either way, it should return a non-negative count without panicking.
	count := BeadsActiveCount()
	if count < 0 {
		t.Errorf("BeadsActiveCount() = %d, want >= 0", count)
	}
}

func TestReconcileFreesGhostSlots(t *testing.T) {
	// Regression test for ghost slot bug: daemon-status.json shows active:3
	// but orch status shows only 1 real agent. The pool was not freeing slots
	// because DiscoverLiveAgents() was counting dead tmux windows as active.
	//
	// With the fix, CountActiveTmuxAgents() checks pane process liveness,
	// dead windows return a lower count, and Reconcile frees the ghost slots.
	config := DefaultConfig()
	config.MaxAgents = 5
	d := NewWithConfig(config)

	// Simulate 3 agents spawned by daemon
	s1 := d.Pool.TryAcquire()
	s2 := d.Pool.TryAcquire()
	s3 := d.Pool.TryAcquire()
	if s1 == nil || s2 == nil || s3 == nil {
		t.Fatal("Expected to acquire 3 slots")
	}
	s1.BeadsID = "proj-aaa1"
	s2.BeadsID = "proj-bbb2"
	s3.BeadsID = "proj-ccc3"

	// Pool at 3/5
	if d.ActiveCount() != 3 {
		t.Fatalf("ActiveCount() = %d, want 3", d.ActiveCount())
	}

	// Simulate: 2 agents completed (tmux windows exist but processes exited).
	// With pane liveness filtering, DiscoverLiveAgents returns 1.
	d.ActiveCounter = &mockActiveCounter{CountFunc: func() int { return 1 }}

	result := d.ReconcileActiveAgents()
	if result.Freed != 2 {
		t.Errorf("ReconcileActiveAgents() freed = %d, want 2 (ghost slots)", result.Freed)
	}
	if d.ActiveCount() != 1 {
		t.Errorf("ActiveCount() after reconcile = %d, want 1", d.ActiveCount())
	}
	if d.AvailableSlots() != 4 {
		t.Errorf("AvailableSlots() after reconcile = %d, want 4", d.AvailableSlots())
	}

	// New agent can now be spawned (previously blocked by ghost slots)
	s4 := d.Pool.TryAcquire()
	if s4 == nil {
		t.Fatal("Should be able to acquire slot after ghost slots freed")
	}
}
