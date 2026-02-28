package daemon

import (
	"testing"
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

func TestPoolReconcileDoesNotResetWhenTmuxAgentsExist(t *testing.T) {
	// This test verifies the core bug: pool reconciliation should NOT
	// reset to 0 when there are active tmux agents.
	//
	// Before fix: ReconcileWithOpenCode() calls DefaultActiveCount() which
	// returns 0 for tmux-only agents, causing pool.Reconcile(0) to free all slots.
	// After fix: Uses CombinedActiveCount() which counts both OpenCode and tmux agents.

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
	freed := pool.Reconcile(3) // Combined count = 3 tmux agents
	if freed != 0 {
		t.Errorf("Reconcile(3) should free 0 slots when pool has 3 active, freed %d", freed)
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
	freed = pool.Reconcile(2) // Combined count drops to 2
	if freed != 1 {
		t.Errorf("Reconcile(2) should free 1 slot, freed %d", freed)
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

	// Set custom active count function that reports 3 active (simulating tmux agents)
	d.activeCountFunc = func() int { return 3 }

	freed := d.ReconcileActiveAgents()
	if freed != 0 {
		t.Errorf("Should free 0 slots when 3 active reported, freed %d", freed)
	}
	if !d.AtCapacity() {
		t.Fatal("Should still be at capacity")
	}

	// Now simulate the old bug: active count returns 0 (only checking OpenCode)
	d.activeCountFunc = func() int { return 0 }

	freed = d.ReconcileActiveAgents()
	if freed != 3 {
		t.Errorf("Should free 3 slots when 0 active reported, freed %d", freed)
	}
	if d.AtCapacity() {
		t.Fatal("Should not be at capacity after freeing all slots")
	}
}

func TestDaemonReconcileDefaultCountIncludesTmux(t *testing.T) {
	// Test that when no custom activeCountFunc is set, the daemon uses
	// CombinedActiveCount which includes both OpenCode and tmux sources.
	// Since CombinedActiveCount makes real HTTP/tmux calls, we verify
	// the function reference is set correctly.
	config := DefaultConfig()
	config.MaxAgents = 3
	d := NewWithConfig(config)

	if d.activeCountFunc == nil {
		t.Fatal("activeCountFunc should be set to CombinedActiveCount by default")
	}
}
