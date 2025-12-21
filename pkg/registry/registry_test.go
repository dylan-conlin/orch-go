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

	agent := &Agent{ID: "agent-1", SessionID: "ses_123"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Mark as completed
	reg.Complete("agent-1")

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

	// First instance modifies (mark completed)
	reg1.Complete("merge-test")
	if err := reg1.Save(); err != nil {
		t.Fatalf("failed to save after complete: %v", err)
	}

	newerUpdated := reg1.Find("merge-test").UpdatedAt
	if newerUpdated <= originalUpdated {
		t.Error("expected newer timestamp after complete")
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
	// agent-2 gets completed
	reg.Complete("agent-2")
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
	// agent-2 gets completed
	reg.Complete("agent-2")
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

// TestAbandonedAgentCanBeRespawned verifies that abandoned agents can be re-registered.
// This reproduces the bug where 'orch abandon' doesn't allow respawning with same ID.
func TestAbandonedAgentCanBeRespawned(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register an agent
	agent1 := &Agent{ID: "agent-1", WindowID: "@100"}
	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register agent-1: %v", err)
	}

	// Abandon it
	if !reg.Abandon("agent-1") {
		t.Fatal("failed to abandon agent-1")
	}

	// Save the registry
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Try to register a new agent with the same ID (respawn)
	// This should succeed because the old agent is abandoned
	agent2 := &Agent{ID: "agent-1", WindowID: "@200"}
	err = reg.Register(agent2)
	if err != nil {
		t.Errorf("expected to allow re-registration of abandoned agent, got error: %v", err)
	}

	// Verify the new agent is active
	found := reg.Find("agent-1")
	if found == nil {
		t.Fatal("expected to find agent-1")
	}
	if found.Status != StateActive {
		t.Errorf("expected status active, got %s", found.Status)
	}
	if found.WindowID != "@200" {
		t.Errorf("expected window_id @200, got %s", found.WindowID)
	}
}

// TestHeadlessWindowIDConstant verifies the headless window ID marker.
func TestHeadlessWindowIDConstant(t *testing.T) {
	if HeadlessWindowID != "headless" {
		t.Errorf("expected HeadlessWindowID to be 'headless', got %s", HeadlessWindowID)
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

// MockLivenessChecker is a mock implementation of LivenessChecker for testing.
type MockLivenessChecker struct {
	LiveWindows  map[string]bool // window IDs that are "alive"
	LiveSessions map[string]bool // session IDs that are "alive"
}

func NewMockLivenessChecker() *MockLivenessChecker {
	return &MockLivenessChecker{
		LiveWindows:  make(map[string]bool),
		LiveSessions: make(map[string]bool),
	}
}

func (m *MockLivenessChecker) WindowExists(windowID string) bool {
	return m.LiveWindows[windowID]
}

func (m *MockLivenessChecker) SessionExists(sessionID string) bool {
	return m.LiveSessions[sessionID]
}

// TestReconcileActiveMarksDeadTmuxWindowsAbandoned verifies that agents with dead tmux windows are abandoned.
func TestReconcileActiveMarksDeadTmuxWindowsAbandoned(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agents with window IDs
	agent1 := &Agent{ID: "agent-1", WindowID: "@100"} // window exists
	agent2 := &Agent{ID: "agent-2", WindowID: "@200"} // window dead

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register agent-1: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register agent-2: %v", err)
	}

	// Mock liveness: only @100 is alive
	checker := NewMockLivenessChecker()
	checker.LiveWindows["@100"] = true

	// Reconcile (not dry-run)
	result := reg.ReconcileActive(checker, false)

	// Should have checked 2 agents
	if result.Checked != 2 {
		t.Errorf("expected 2 checked, got %d", result.Checked)
	}

	// Should have abandoned 1 agent (agent-2)
	if result.Abandoned != 1 {
		t.Errorf("expected 1 abandoned, got %d", result.Abandoned)
	}

	// agent-1 should still be active
	found1 := reg.Find("agent-1")
	if found1.Status != StateActive {
		t.Errorf("expected agent-1 to be active, got %s", found1.Status)
	}

	// agent-2 should be abandoned
	found2 := reg.Find("agent-2")
	if found2.Status != StateAbandoned {
		t.Errorf("expected agent-2 to be abandoned, got %s", found2.Status)
	}
}

// TestReconcileActiveMarksDeadOpenCodeSessionsAbandoned verifies that headless agents with dead sessions are abandoned.
func TestReconcileActiveMarksDeadOpenCodeSessionsAbandoned(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register headless agents (no window ID, but have session ID)
	agent1 := &Agent{ID: "agent-1", WindowID: HeadlessWindowID, SessionID: "ses_alive"}
	agent2 := &Agent{ID: "agent-2", WindowID: HeadlessWindowID, SessionID: "ses_dead"}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register agent-1: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register agent-2: %v", err)
	}

	// Mock liveness: only ses_alive is alive
	checker := NewMockLivenessChecker()
	checker.LiveSessions["ses_alive"] = true

	// Reconcile
	result := reg.ReconcileActive(checker, false)

	// Should have checked 2 agents
	if result.Checked != 2 {
		t.Errorf("expected 2 checked, got %d", result.Checked)
	}

	// Should have abandoned 1 agent (agent-2)
	if result.Abandoned != 1 {
		t.Errorf("expected 1 abandoned, got %d", result.Abandoned)
	}

	// agent-2 should be abandoned
	found2 := reg.Find("agent-2")
	if found2.Status != StateAbandoned {
		t.Errorf("expected agent-2 to be abandoned, got %s", found2.Status)
	}
}

// TestReconcileActiveDryRunDoesNotModify verifies dry-run mode doesn't modify the registry.
func TestReconcileActiveDryRunDoesNotModify(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent with dead window
	agent := &Agent{ID: "agent-1", WindowID: "@dead"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Empty checker = all windows are dead
	checker := NewMockLivenessChecker()

	// Reconcile with dry-run
	result := reg.ReconcileActive(checker, true)

	// Should report 1 abandoned
	if result.Abandoned != 1 {
		t.Errorf("expected 1 abandoned, got %d", result.Abandoned)
	}

	// But agent should still be active (dry-run doesn't modify)
	found := reg.Find("agent-1")
	if found.Status != StateActive {
		t.Errorf("expected agent-1 to still be active after dry-run, got %s", found.Status)
	}
}

// TestReconcileActiveSkipsCompletedAndAbandoned verifies reconciliation only checks active agents.
func TestReconcileActiveSkipsCompletedAndAbandoned(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agents
	agent1 := &Agent{ID: "agent-1", WindowID: "@100"}
	agent2 := &Agent{ID: "agent-2", WindowID: "@200"}
	agent3 := &Agent{ID: "agent-3", WindowID: "@300"}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register agent-1: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register agent-2: %v", err)
	}
	if err := reg.Register(agent3); err != nil {
		t.Fatalf("failed to register agent-3: %v", err)
	}

	// Mark agent-1 as completed, agent-2 as abandoned
	reg.Complete("agent-1")
	reg.Abandon("agent-2")

	// All windows are dead
	checker := NewMockLivenessChecker()

	// Reconcile
	result := reg.ReconcileActive(checker, false)

	// Should only check agent-3 (the only active one)
	if result.Checked != 1 {
		t.Errorf("expected 1 checked (only active agent), got %d", result.Checked)
	}
}

// TestReconcileActiveWithBothChecks verifies that both tmux and OpenCode are checked.
func TestReconcileActiveWithBothChecks(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Agent with both window and session - window exists but session is dead
	agent := &Agent{ID: "agent-1", WindowID: "@100", SessionID: "ses_dead"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	checker := NewMockLivenessChecker()
	checker.LiveWindows["@100"] = true // Window is alive
	// Session is NOT alive

	// Reconcile
	result := reg.ReconcileActive(checker, false)

	// Should mark as abandoned because session is dead (even though window is alive)
	if result.Abandoned != 1 {
		t.Errorf("expected 1 abandoned (dead session), got %d", result.Abandoned)
	}

	found := reg.Find("agent-1")
	if found.Status != StateAbandoned {
		t.Errorf("expected agent-1 to be abandoned, got %s", found.Status)
	}
}

// TestReconcileActiveAgentWithNoWindowOrSession verifies agents without IDs are not abandoned.
func TestReconcileActiveAgentWithNoWindowOrSession(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Agent with no window or session ID (inline agent)
	agent := &Agent{ID: "inline-agent", WindowID: "", SessionID: ""}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	checker := NewMockLivenessChecker()

	// Reconcile
	result := reg.ReconcileActive(checker, false)

	// Should check the agent but not abandon it (no way to verify liveness)
	if result.Checked != 1 {
		t.Errorf("expected 1 checked, got %d", result.Checked)
	}
	if result.Abandoned != 0 {
		t.Errorf("expected 0 abandoned (no IDs to check), got %d", result.Abandoned)
	}

	found := reg.Find("inline-agent")
	if found.Status != StateActive {
		t.Errorf("expected inline-agent to remain active, got %s", found.Status)
	}
}

// TestReconcileActiveReturnsDetails verifies the result includes human-readable details.
func TestReconcileActiveReturnsDetails(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "agent-1", WindowID: "@dead"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	checker := NewMockLivenessChecker()
	result := reg.ReconcileActive(checker, false)

	if len(result.AgentIDs) != 1 || result.AgentIDs[0] != "agent-1" {
		t.Errorf("expected AgentIDs=['agent-1'], got %v", result.AgentIDs)
	}
	if len(result.Details) != 1 {
		t.Errorf("expected 1 detail, got %d", len(result.Details))
	}
}

// TestActiveCountReturnsCorrectCount verifies ActiveCount returns the correct number of active agents.
func TestActiveCountReturnsCorrectCount(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Initially no agents
	if count := reg.ActiveCount(); count != 0 {
		t.Errorf("expected 0 active agents, got %d", count)
	}

	// Register some agents
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

	// All three should be active
	if count := reg.ActiveCount(); count != 3 {
		t.Errorf("expected 3 active agents, got %d", count)
	}

	// Complete one, abandon another
	reg.Complete("agent-1")
	reg.Abandon("agent-2")

	// Only one should remain active
	if count := reg.ActiveCount(); count != 1 {
		t.Errorf("expected 1 active agent after complete/abandon, got %d", count)
	}

	// Remove the remaining active agent
	reg.Remove("agent-3")

	// None should be active
	if count := reg.ActiveCount(); count != 0 {
		t.Errorf("expected 0 active agents after remove, got %d", count)
	}
}

// MockBeadsStatusChecker is a mock implementation of BeadsStatusChecker for testing.
type MockBeadsStatusChecker struct {
	ClosedIssues map[string]bool // beads IDs that are "closed"
}

func NewMockBeadsStatusChecker() *MockBeadsStatusChecker {
	return &MockBeadsStatusChecker{
		ClosedIssues: make(map[string]bool),
	}
}

func (m *MockBeadsStatusChecker) IsIssueClosed(beadsID string) bool {
	return m.ClosedIssues[beadsID]
}

// TestReconcileWithBeadsMarksClosedIssuesAsCompleted verifies that agents with closed beads issues are marked as completed.
func TestReconcileWithBeadsMarksClosedIssuesAsCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agents with beads IDs
	agent1 := &Agent{ID: "agent-1", BeadsID: "beads-open", WindowID: "@100"}   // issue open
	agent2 := &Agent{ID: "agent-2", BeadsID: "beads-closed", WindowID: "@200"} // issue closed

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register agent-1: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register agent-2: %v", err)
	}

	// Mock beads checker: only beads-closed is closed
	checker := NewMockBeadsStatusChecker()
	checker.ClosedIssues["beads-closed"] = true

	// Reconcile (not dry-run)
	result := reg.ReconcileWithBeads(checker, false)

	// Should have checked 2 agents (both have beads IDs)
	if result.Checked != 2 {
		t.Errorf("expected 2 checked, got %d", result.Checked)
	}

	// Should have completed 1 agent (agent-2)
	if result.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Completed)
	}

	// agent-1 should still be active
	found1 := reg.Find("agent-1")
	if found1.Status != StateActive {
		t.Errorf("expected agent-1 to be active, got %s", found1.Status)
	}

	// agent-2 should be completed (not abandoned)
	found2 := reg.Find("agent-2")
	if found2.Status != StateCompleted {
		t.Errorf("expected agent-2 to be completed, got %s", found2.Status)
	}
	if found2.CompletedAt == "" {
		t.Error("expected agent-2 CompletedAt to be set")
	}
}

// TestReconcileWithBeadsSkipsAgentsWithoutBeadsID verifies that agents without beads IDs are skipped.
func TestReconcileWithBeadsSkipsAgentsWithoutBeadsID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent without beads ID
	agent := &Agent{ID: "agent-1", WindowID: "@100"} // No BeadsID
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	checker := NewMockBeadsStatusChecker()

	// Reconcile
	result := reg.ReconcileWithBeads(checker, false)

	// Should not check agents without beads ID
	if result.Checked != 0 {
		t.Errorf("expected 0 checked (no beads ID), got %d", result.Checked)
	}
	if result.Completed != 0 {
		t.Errorf("expected 0 completed, got %d", result.Completed)
	}

	// Agent should still be active
	found := reg.Find("agent-1")
	if found.Status != StateActive {
		t.Errorf("expected agent-1 to remain active, got %s", found.Status)
	}
}

// TestReconcileWithBeadsDryRunDoesNotModify verifies dry-run mode doesn't modify the registry.
func TestReconcileWithBeadsDryRunDoesNotModify(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent with closed beads issue
	agent := &Agent{ID: "agent-1", BeadsID: "beads-closed", WindowID: "@100"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	checker := NewMockBeadsStatusChecker()
	checker.ClosedIssues["beads-closed"] = true

	// Reconcile with dry-run
	result := reg.ReconcileWithBeads(checker, true)

	// Should report 1 completed
	if result.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Completed)
	}

	// But agent should still be active (dry-run doesn't modify)
	found := reg.Find("agent-1")
	if found.Status != StateActive {
		t.Errorf("expected agent-1 to still be active after dry-run, got %s", found.Status)
	}
}

// TestReconcileWithBeadsReturnsDetails verifies the result includes human-readable details.
func TestReconcileWithBeadsReturnsDetails(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agent := &Agent{ID: "agent-1", BeadsID: "beads-closed", WindowID: "@100"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	checker := NewMockBeadsStatusChecker()
	checker.ClosedIssues["beads-closed"] = true

	result := reg.ReconcileWithBeads(checker, false)

	if len(result.AgentIDs) != 1 || result.AgentIDs[0] != "agent-1" {
		t.Errorf("expected AgentIDs=['agent-1'], got %v", result.AgentIDs)
	}
	if len(result.Details) != 1 {
		t.Errorf("expected 1 detail, got %d", len(result.Details))
	}
}

// MockCompletionIndicatorChecker is a mock implementation of CompletionIndicatorChecker for testing.
type MockCompletionIndicatorChecker struct {
	SynthesisPaths  map[string]bool // workspace paths where SYNTHESIS.md exists
	CompletedPhases map[string]bool // beads IDs where Phase: Complete is reported
}

func NewMockCompletionIndicatorChecker() *MockCompletionIndicatorChecker {
	return &MockCompletionIndicatorChecker{
		SynthesisPaths:  make(map[string]bool),
		CompletedPhases: make(map[string]bool),
	}
}

func (m *MockCompletionIndicatorChecker) SynthesisExists(workspacePath string) bool {
	return m.SynthesisPaths[workspacePath]
}

func (m *MockCompletionIndicatorChecker) IsPhaseComplete(beadsID string) bool {
	return m.CompletedPhases[beadsID]
}

// TestReconcileActiveWithCompletionCheckMarksCompletedWithSynthesis verifies that agents with
// dead sessions but SYNTHESIS.md present are marked as completed, not abandoned.
func TestReconcileActiveWithCompletionCheckMarksCompletedWithSynthesis(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent with session ID and project dir
	agent := &Agent{
		ID:         "agent-1",
		SessionID:  "ses_dead",
		WindowID:   HeadlessWindowID,
		ProjectDir: "/project",
	}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Mock liveness: session is dead
	livenessChecker := NewMockLivenessChecker()

	// Mock completion: SYNTHESIS.md exists
	completionChecker := NewMockCompletionIndicatorChecker()
	completionChecker.SynthesisPaths["/project/.orch/workspace/agent-1"] = true

	// Reconcile with completion check
	result := reg.ReconcileActiveWithCompletionCheck(livenessChecker, completionChecker, false)

	// Should mark as completed, not abandoned
	if result.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Completed)
	}
	if result.Abandoned != 0 {
		t.Errorf("expected 0 abandoned, got %d", result.Abandoned)
	}

	found := reg.Find("agent-1")
	if found.Status != StateCompleted {
		t.Errorf("expected agent-1 to be completed, got %s", found.Status)
	}
	if found.CompletedAt == "" {
		t.Error("expected CompletedAt to be set")
	}
}

// TestReconcileActiveWithCompletionCheckMarksCompletedWithPhaseComplete verifies that agents with
// dead sessions but Phase: Complete in beads are marked as completed, not abandoned.
func TestReconcileActiveWithCompletionCheckMarksCompletedWithPhaseComplete(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent with session ID and beads ID (no project dir for SYNTHESIS.md)
	agent := &Agent{
		ID:        "agent-1",
		SessionID: "ses_dead",
		WindowID:  HeadlessWindowID,
		BeadsID:   "beads-123",
	}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Mock liveness: session is dead
	livenessChecker := NewMockLivenessChecker()

	// Mock completion: Phase: Complete is reported in beads
	completionChecker := NewMockCompletionIndicatorChecker()
	completionChecker.CompletedPhases["beads-123"] = true

	// Reconcile with completion check
	result := reg.ReconcileActiveWithCompletionCheck(livenessChecker, completionChecker, false)

	// Should mark as completed, not abandoned
	if result.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Completed)
	}
	if result.Abandoned != 0 {
		t.Errorf("expected 0 abandoned, got %d", result.Abandoned)
	}

	found := reg.Find("agent-1")
	if found.Status != StateCompleted {
		t.Errorf("expected agent-1 to be completed, got %s", found.Status)
	}
}

// TestReconcileActiveWithCompletionCheckAbandonsWithNoIndicators verifies that agents with
// dead sessions and NO completion indicators are still marked as abandoned.
func TestReconcileActiveWithCompletionCheckAbandonsWithNoIndicators(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent with session ID
	agent := &Agent{
		ID:         "agent-1",
		SessionID:  "ses_dead",
		WindowID:   HeadlessWindowID,
		ProjectDir: "/project",
		BeadsID:    "beads-123",
	}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Mock liveness: session is dead
	livenessChecker := NewMockLivenessChecker()

	// Mock completion: neither SYNTHESIS.md nor Phase: Complete exists
	completionChecker := NewMockCompletionIndicatorChecker()

	// Reconcile with completion check
	result := reg.ReconcileActiveWithCompletionCheck(livenessChecker, completionChecker, false)

	// Should mark as abandoned (no completion indicators)
	if result.Abandoned != 1 {
		t.Errorf("expected 1 abandoned, got %d", result.Abandoned)
	}
	if result.Completed != 0 {
		t.Errorf("expected 0 completed, got %d", result.Completed)
	}

	found := reg.Find("agent-1")
	if found.Status != StateAbandoned {
		t.Errorf("expected agent-1 to be abandoned, got %s", found.Status)
	}
}

// TestReconcileActiveWithCompletionCheckNilCheckerBehavesLikeReconcileActive verifies that
// passing nil for completionChecker makes it behave like the original ReconcileActive.
func TestReconcileActiveWithCompletionCheckNilCheckerBehavesLikeReconcileActive(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent with session ID
	agent := &Agent{
		ID:        "agent-1",
		SessionID: "ses_dead",
		WindowID:  HeadlessWindowID,
	}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Mock liveness: session is dead
	livenessChecker := NewMockLivenessChecker()

	// Pass nil for completionChecker - should behave like ReconcileActive
	result := reg.ReconcileActiveWithCompletionCheck(livenessChecker, nil, false)

	// Should mark as abandoned (nil completion checker means no completion check)
	if result.Abandoned != 1 {
		t.Errorf("expected 1 abandoned, got %d", result.Abandoned)
	}

	found := reg.Find("agent-1")
	if found.Status != StateAbandoned {
		t.Errorf("expected agent-1 to be abandoned, got %s", found.Status)
	}
}

// TestReconcileActiveWithCompletionCheckDryRunDoesNotModify verifies dry-run mode doesn't modify registry.
func TestReconcileActiveWithCompletionCheckDryRunDoesNotModify(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent with session ID and completion indicator
	agent := &Agent{
		ID:         "agent-1",
		SessionID:  "ses_dead",
		WindowID:   HeadlessWindowID,
		ProjectDir: "/project",
	}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	livenessChecker := NewMockLivenessChecker()
	completionChecker := NewMockCompletionIndicatorChecker()
	completionChecker.SynthesisPaths["/project/.orch/workspace/agent-1"] = true

	// Reconcile with dry-run
	result := reg.ReconcileActiveWithCompletionCheck(livenessChecker, completionChecker, true)

	// Should report 1 completed
	if result.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Completed)
	}

	// But agent should still be active (dry-run doesn't modify)
	found := reg.Find("agent-1")
	if found.Status != StateActive {
		t.Errorf("expected agent-1 to still be active after dry-run, got %s", found.Status)
	}
}

// TestReconcileActiveWithCompletionCheckPrioritizesSynthesisOverPhase verifies that when both
// indicators are present, SYNTHESIS.md is checked first (as it's more definitive).
func TestReconcileActiveWithCompletionCheckPrioritizesSynthesisOverPhase(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent with both project dir and beads ID
	agent := &Agent{
		ID:         "agent-1",
		SessionID:  "ses_dead",
		WindowID:   HeadlessWindowID,
		ProjectDir: "/project",
		BeadsID:    "beads-123",
	}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	livenessChecker := NewMockLivenessChecker()
	completionChecker := NewMockCompletionIndicatorChecker()
	// Both indicators are present
	completionChecker.SynthesisPaths["/project/.orch/workspace/agent-1"] = true
	completionChecker.CompletedPhases["beads-123"] = true

	result := reg.ReconcileActiveWithCompletionCheck(livenessChecker, completionChecker, false)

	// Should mark as completed
	if result.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Completed)
	}

	// Detail should mention SYNTHESIS.md (checked first)
	found := false
	for _, detail := range result.Details {
		if contains(detail, "SYNTHESIS.md") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected detail to mention SYNTHESIS.md, got %v", result.Details)
	}
}

// TestReconcileActiveWithCompletionCheckLiveAgentNotAffected verifies that agents with live
// sessions are not checked for completion indicators.
func TestReconcileActiveWithCompletionCheckLiveAgentNotAffected(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agent with live session
	agent := &Agent{
		ID:         "agent-1",
		SessionID:  "ses_alive",
		WindowID:   HeadlessWindowID,
		ProjectDir: "/project",
	}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	livenessChecker := NewMockLivenessChecker()
	livenessChecker.LiveSessions["ses_alive"] = true

	// Even if SYNTHESIS.md exists, live agent should not be marked completed
	completionChecker := NewMockCompletionIndicatorChecker()
	completionChecker.SynthesisPaths["/project/.orch/workspace/agent-1"] = true

	result := reg.ReconcileActiveWithCompletionCheck(livenessChecker, completionChecker, false)

	// Should not be abandoned or completed (still active)
	if result.Abandoned != 0 {
		t.Errorf("expected 0 abandoned, got %d", result.Abandoned)
	}
	if result.Completed != 0 {
		t.Errorf("expected 0 completed, got %d", result.Completed)
	}

	found := reg.Find("agent-1")
	if found.Status != StateActive {
		t.Errorf("expected agent-1 to remain active, got %s", found.Status)
	}
}

// helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestReconcileWithBeadsSkipsNonActiveAgents verifies that only active agents are checked.
func TestReconcileWithBeadsSkipsNonActiveAgents(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "registry.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agents
	agent1 := &Agent{ID: "agent-1", BeadsID: "beads-1", WindowID: "@100"}
	agent2 := &Agent{ID: "agent-2", BeadsID: "beads-2", WindowID: "@200"}
	agent3 := &Agent{ID: "agent-3", BeadsID: "beads-3", WindowID: "@300"}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("failed to register agent-1: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("failed to register agent-2: %v", err)
	}
	if err := reg.Register(agent3); err != nil {
		t.Fatalf("failed to register agent-3: %v", err)
	}

	// Mark agent-1 as completed, agent-2 as abandoned
	reg.Complete("agent-1")
	reg.Abandon("agent-2")

	// All issues are closed
	checker := NewMockBeadsStatusChecker()
	checker.ClosedIssues["beads-1"] = true
	checker.ClosedIssues["beads-2"] = true
	checker.ClosedIssues["beads-3"] = true

	// Reconcile
	result := reg.ReconcileWithBeads(checker, false)

	// Should only check agent-3 (the only active one)
	if result.Checked != 1 {
		t.Errorf("expected 1 checked (only active agent), got %d", result.Checked)
	}
	if result.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Completed)
	}
}
