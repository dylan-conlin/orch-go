package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestRegisterCreatesActiveAgent verifies that registering an agent
// creates it in active state with proper timestamps.
func TestRegisterCreatesActiveAgent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{
		ID:       "agent-1",
		BeadsID:  "beads-abc",
		WindowID: "@100",
		Window:   "workers:0",
	}

	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// Verify agent is active
	found := reg.Find("agent-1")
	if found == nil {
		t.Fatal("expected to find agent")
	}
	if found.Status != StateActive {
		t.Errorf("expected status active, got %s", found.Status)
	}
	if found.WindowID != "@100" {
		t.Errorf("expected window_id @100, got %s", found.WindowID)
	}
	if found.BeadsID != "beads-abc" {
		t.Errorf("expected beads_id beads-abc, got %s", found.BeadsID)
	}
	if found.SpawnedAt == "" {
		t.Error("expected spawned_at to be set")
	}
	if found.UpdatedAt == "" {
		t.Error("expected updated_at to be set")
	}
}

// TestDuplicateRegisterRaises verifies that registering the same ID twice fails.
func TestDuplicateRegisterRaises(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "agent-1", WindowID: "@100"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	// Second register should fail
	agent2 := &Agent{ID: "agent-1", WindowID: "@200"}
	err = reg.Register(agent2)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

// TestWindowReuseAbandonsOldAgent verifies that reusing a window_id
// marks the old agent as abandoned.
func TestWindowReuseAbandonsOldAgent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent1 := &Agent{ID: "agent-1", WindowID: "@100"}
	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register agent-1: %v", err)
	}

	// Register agent-2 with same window_id
	agent2 := &Agent{ID: "agent-2", WindowID: "@100"}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register agent-2: %v", err)
	}

	// agent-1 should be abandoned
	found1 := reg.Find("agent-1")
	if found1.Status != StateAbandoned {
		t.Errorf("expected agent-1 status abandoned, got %s", found1.Status)
	}
	if found1.AbandonedAt == "" {
		t.Error("expected abandoned_at to be set")
	}

	// agent-2 should be active
	found2 := reg.Find("agent-2")
	if found2.Status != StateActive {
		t.Errorf("expected agent-2 status active, got %s", found2.Status)
	}
}

// TestFindPrefersAgentIDOverBeadsID verifies that Find prefers agent ID match.
func TestFindPrefersAgentIDOverBeadsID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// agent-1 has beads_id "shared-id"
	agent1 := &Agent{ID: "agent-1", BeadsID: "shared-id", WindowID: "@100"}
	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register agent-1: %v", err)
	}

	// agent "shared-id" has different beads_id
	agent2 := &Agent{ID: "shared-id", BeadsID: "other", WindowID: "@200"}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register shared-id: %v", err)
	}

	// Find "shared-id" should return the agent with ID "shared-id", not beads_id
	found := reg.Find("shared-id")
	if found == nil {
		t.Fatal("expected to find agent")
	}
	if found.ID != "shared-id" {
		t.Errorf("expected ID shared-id, got %s", found.ID)
	}
	if found.BeadsID != "other" {
		t.Errorf("expected beads_id other, got %s", found.BeadsID)
	}
}

// TestFindByBeadsID verifies that Find can locate agents by beads_id.
func TestFindByBeadsID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "agent-1", BeadsID: "beads-xyz", WindowID: "@100"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Find by beads_id
	found := reg.Find("beads-xyz")
	if found == nil {
		t.Fatal("expected to find agent by beads_id")
	}
	if found.ID != "agent-1" {
		t.Errorf("expected ID agent-1, got %s", found.ID)
	}
}

// TestListAgentsExcludesDeleted verifies that ListAgents excludes deleted agents.
func TestListAgentsExcludesDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register and delete an agent
	agent1 := &Agent{ID: "agent-1", WindowID: "@100"}
	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	agent2 := &Agent{ID: "agent-2", WindowID: "@200"}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	reg.Remove("agent-1")

	// ListAgents should only return agent-2
	agents := reg.ListAgents()
	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].ID != "agent-2" {
		t.Errorf("expected agent-2, got %s", agents[0].ID)
	}
}

// TestListActiveReturnsOnlyActive verifies that ListActive returns only active agents.
func TestListActiveReturnsOnlyActive(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Create agents with different statuses
	agent1 := &Agent{ID: "agent-1", WindowID: "@100"}
	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	agent2 := &Agent{ID: "agent-2", WindowID: "@200"}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	agent3 := &Agent{ID: "agent-3", WindowID: "@300"}
	if err := reg.Register(agent3); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Abandon agent-1, delete agent-2
	reg.Abandon("agent-1")
	reg.Remove("agent-2")

	// Only agent-3 should be active
	active := reg.ListActive()
	if len(active) != 1 {
		t.Errorf("expected 1 active agent, got %d", len(active))
	}
	if active[0].ID != "agent-3" {
		t.Errorf("expected agent-3, got %s", active[0].ID)
	}
}

// TestAbandonReturnsTrue verifies Abandon returns true on success.
func TestAbandonReturnsTrue(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "agent-1", WindowID: "@100"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	if !reg.Abandon("agent-1") {
		t.Error("expected Abandon to return true")
	}

	found := reg.Find("agent-1")
	if found.Status != StateAbandoned {
		t.Errorf("expected status abandoned, got %s", found.Status)
	}
}

// TestAbandonReturnsFalseForNonActive verifies Abandon returns false for non-active.
func TestAbandonReturnsFalseForNonActive(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "agent-1", WindowID: "@100"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Reconcile to mark as completed
	reg.Reconcile([]string{}) // No active windows

	// Can't abandon a completed agent
	if reg.Abandon("agent-1") {
		t.Error("expected Abandon to return false for completed agent")
	}
}

// TestAbandonReturnsFalseForUnknown verifies Abandon returns false for unknown.
func TestAbandonReturnsFalseForUnknown(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	if reg.Abandon("nonexistent") {
		t.Error("expected Abandon to return false for unknown agent")
	}
}

// TestDeleteUsesTombstonePattern verifies that delete uses tombstone pattern.
func TestDeleteUsesTombstonePattern(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "agent-1", WindowID: "@100"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	result := reg.Remove("agent-1")
	if !result {
		t.Error("expected Remove to return true")
	}

	// Agent still exists but is deleted
	found := reg.Find("agent-1")
	if found == nil {
		t.Fatal("expected to find agent even after delete (tombstone)")
	}
	if found.Status != StateDeleted {
		t.Errorf("expected status deleted, got %s", found.Status)
	}

	// But doesn't appear in ListAgents()
	agents := reg.ListAgents()
	if len(agents) != 0 {
		t.Errorf("expected 0 agents in ListAgents, got %d", len(agents))
	}
}

// TestDeleteReturnsFalseForUnknown verifies Remove returns false for unknown.
func TestDeleteReturnsFalseForUnknown(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	if reg.Remove("nonexistent") {
		t.Error("expected Remove to return false for unknown agent")
	}
}

// TestReconcileCompletesAgentsWithClosedWindows verifies reconcile behavior.
func TestReconcileCompletesAgentsWithClosedWindows(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent1 := &Agent{ID: "agent-1", WindowID: "@100"}
	agent2 := &Agent{ID: "agent-2", WindowID: "@200"}
	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Window @100 closed, @200 still open
	count := reg.Reconcile([]string{"@200"})

	if count != 1 {
		t.Errorf("expected 1 completed, got %d", count)
	}

	found1 := reg.Find("agent-1")
	if found1.Status != StateCompleted {
		t.Errorf("expected agent-1 status completed, got %s", found1.Status)
	}
	if found1.CompletedAt == "" {
		t.Error("expected completed_at to be set")
	}

	found2 := reg.Find("agent-2")
	if found2.Status != StateActive {
		t.Errorf("expected agent-2 status active, got %s", found2.Status)
	}
}

// TestReconcileIgnoresAgentsWithoutWindowID verifies agents without window_id aren't affected.
func TestReconcileIgnoresAgentsWithoutWindowID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Agent without window_id (e.g., inline spawn)
	agent := &Agent{ID: "agent-1", WindowID: ""}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Reconcile with no active windows
	count := reg.Reconcile([]string{})

	if count != 0 {
		t.Errorf("expected 0 completed (no window_id), got %d", count)
	}

	found := reg.Find("agent-1")
	if found.Status != StateActive {
		t.Errorf("expected status active, got %s", found.Status)
	}
}

// TestPersistence verifies that data persists across registry instances.
func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	// Create and save an agent
	reg1, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "agent-1", BeadsID: "beads-xyz", WindowID: "@100"}
	if err := reg1.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if err := reg1.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load in a new registry instance
	reg2, err := New(path)
	if err != nil {
		t.Fatalf("failed to create second registry: %v", err)
	}

	found := reg2.Find("agent-1")
	if found == nil {
		t.Fatal("expected to find agent after reload")
	}
	if found.BeadsID != "beads-xyz" {
		t.Errorf("expected beads_id beads-xyz, got %s", found.BeadsID)
	}
}

// TestConcurrentRegistersNoDataLoss verifies concurrent registrations don't lose data.
func TestConcurrentRegistersNoDataLoss(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	numAgents := 20

	var wg sync.WaitGroup
	errChan := make(chan error, numAgents)

	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			reg, err := New(path)
			if err != nil {
				errChan <- err
				return
			}

			agent := &Agent{
				ID:       fmt.Sprintf("concurrent-%d", idx),
				WindowID: fmt.Sprintf("@%d", 1000+idx),
			}
			if err := reg.Register(agent); err != nil {
				errChan <- err
				return
			}
			if err := reg.Save(); err != nil {
				errChan <- err
				return
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	for err := range errChan {
		t.Errorf("concurrent operation failed: %v", err)
	}

	// Verify all agents registered
	finalReg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create final registry: %v", err)
	}

	agents := finalReg.ListActive()
	if len(agents) != numAgents {
		t.Errorf("expected %d agents, got %d", numAgents, len(agents))
		for _, a := range agents {
			t.Logf("  - %s", a.ID)
		}
	}
}

// TestMergePreservesNewerUpdatedAt verifies merge logic respects timestamps.
func TestMergePreservesNewerUpdatedAt(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	// Create agent
	reg1, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "merge-test", WindowID: "@888"}
	if err := reg1.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if err := reg1.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	originalUpdated := reg1.Find("merge-test").UpdatedAt

	// Create second instance (stale view)
	reg2, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Small delay to ensure timestamp difference
	time.Sleep(20 * time.Millisecond)

	// First instance modifies (reconcile to mark completed)
	reg1.Reconcile([]string{}) // Mark completed
	if err := reg1.Save(); err != nil {
		t.Fatalf("failed to save after reconcile: %v", err)
	}

	newerUpdated := reg1.Find("merge-test").UpdatedAt
	if newerUpdated <= originalUpdated {
		t.Error("expected newer timestamp after reconcile")
	}

	// Second instance tries to save stale data
	// The merge should detect disk has newer data and preserve it
	if err := reg2.Save(); err != nil {
		t.Fatalf("failed to save stale registry: %v", err)
	}

	// Verify newer state preserved
	finalReg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create final registry: %v", err)
	}

	found := finalReg.Find("merge-test")
	if found.Status != StateCompleted {
		t.Errorf("expected status completed (from newer write), got %s", found.Status)
	}
	if found.UpdatedAt != newerUpdated {
		t.Errorf("expected updated_at %s, got %s", newerUpdated, found.UpdatedAt)
	}
}

// TestTombstonePreventsResurrection verifies deleted agents don't come back.
func TestTombstonePreventsResurrection(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	// Create and delete agent
	reg1, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "zombie", WindowID: "@666"}
	if err := reg1.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	reg1.Remove("zombie")
	if err := reg1.SaveSkipMerge(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// New instance should see it as deleted
	reg2, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agents := reg2.ListAgents() // Excludes deleted
	for _, a := range agents {
		if a.ID == "zombie" {
			t.Error("zombie should not appear in ListAgents")
		}
	}

	// But it still exists with deleted status
	zombie := reg2.Find("zombie")
	if zombie == nil {
		t.Fatal("expected to find zombie via Find")
	}
	if zombie.Status != StateDeleted {
		t.Errorf("expected status deleted, got %s", zombie.Status)
	}
}

// TestNewWithEmptyPath verifies that New with empty path uses default.
func TestNewWithEmptyPath(t *testing.T) {
	// This test verifies the default path logic
	defaultPath := DefaultPath()
	if defaultPath == "" {
		t.Error("DefaultPath should not be empty")
	}
}

// TestListCompleted verifies ListCompleted returns only completed agents.
func TestListCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Create agents with different statuses
	agent1 := &Agent{ID: "agent-1", WindowID: "@100"}
	agent2 := &Agent{ID: "agent-2", WindowID: "@200"}
	agent3 := &Agent{ID: "agent-3", WindowID: "@300"}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if err := reg.Register(agent3); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// agent-1 stays active
	// agent-2 gets reconciled to completed (window closed)
	reg.Reconcile([]string{"@100", "@300"}) // @200 is missing, so agent-2 is completed
	// agent-3 gets abandoned
	reg.Abandon("agent-3")

	// ListCompleted should only return agent-2
	completed := reg.ListCompleted()
	if len(completed) != 1 {
		t.Errorf("expected 1 completed agent, got %d", len(completed))
	}
	if len(completed) > 0 && completed[0].ID != "agent-2" {
		t.Errorf("expected agent-2, got %s", completed[0].ID)
	}
}

// TestListCleanable verifies ListCleanable returns completed and abandoned agents.
func TestListCleanable(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Create agents with different statuses
	agent1 := &Agent{ID: "agent-1", WindowID: "@100"}
	agent2 := &Agent{ID: "agent-2", WindowID: "@200"}
	agent3 := &Agent{ID: "agent-3", WindowID: "@300"}
	agent4 := &Agent{ID: "agent-4", WindowID: "@400"}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if err := reg.Register(agent3); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if err := reg.Register(agent4); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// agent-1 stays active
	// agent-2 gets reconciled to completed
	reg.Reconcile([]string{"@100", "@300", "@400"}) // @200 missing -> completed
	// agent-3 gets abandoned
	reg.Abandon("agent-3")
	// agent-4 gets deleted
	reg.Remove("agent-4")

	// ListCleanable should return agent-2 (completed) and agent-3 (abandoned)
	cleanable := reg.ListCleanable()
	if len(cleanable) != 2 {
		t.Errorf("expected 2 cleanable agents, got %d", len(cleanable))
	}

	ids := make(map[string]bool)
	for _, a := range cleanable {
		ids[a.ID] = true
	}

	if !ids["agent-2"] {
		t.Error("expected agent-2 to be cleanable (completed)")
	}
	if !ids["agent-3"] {
		t.Error("expected agent-3 to be cleanable (abandoned)")
	}
	if ids["agent-1"] {
		t.Error("agent-1 (active) should not be cleanable")
	}
	if ids["agent-4"] {
		t.Error("agent-4 (deleted) should not be cleanable")
	}
}

// TestReconcileIgnoresHeadlessAgents verifies reconcile skips headless agents.
// Headless agents are tracked via SSE events, not tmux windows.
func TestReconcileIgnoresHeadlessAgents(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Agent with tmux window
	agent1 := &Agent{ID: "tmux-agent", WindowID: "@100"}
	// Headless agent
	agent2 := &Agent{ID: "headless-agent", WindowID: HeadlessWindowID}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register tmux agent: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register headless agent: %v", err)
	}

	// Reconcile with no active windows
	count := reg.Reconcile([]string{})

	// Only the tmux agent should be reconciled
	if count != 1 {
		t.Errorf("expected 1 completed (tmux agent), got %d", count)
	}

	// tmux-agent should be completed
	found1 := reg.Find("tmux-agent")
	if found1.Status != StateCompleted {
		t.Errorf("expected tmux-agent status completed, got %s", found1.Status)
	}

	// headless-agent should still be active (SSE tracks it)
	found2 := reg.Find("headless-agent")
	if found2.Status != StateActive {
		t.Errorf("expected headless-agent status active, got %s", found2.Status)
	}
}

// TestLoadNonExistentFile verifies loading a non-existent file creates empty registry.
func TestLoadNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent", "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agents := reg.ListAgents()
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

// TestSaveCreatesParentDirectory verifies Save creates parent directories.
func TestSaveCreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "deeply", "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "agent-1", WindowID: "@100"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected registry file to be created")
	}
}
