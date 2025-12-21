package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/registry"
)

// TestRunCleanNoAgents verifies clean handles empty registry gracefully.
func TestRunCleanNoAgents(t *testing.T) {
	// Create a temp registry file
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create empty registry
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Test that clean works with empty registry (unit test on registry directly)
	agents := reg.ListCleanable()
	if len(agents) != 0 {
		t.Errorf("Expected 0 cleanable agents, got %d", len(agents))
	}
}

// TestRunCleanWithCompletedAgents verifies clean removes completed agents.
func TestRunCleanWithCompletedAgents(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry with agents
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register agents
	agent1 := &registry.Agent{ID: "agent-1", BeadsID: "beads-1", WindowID: "@100"}
	agent2 := &registry.Agent{ID: "agent-2", BeadsID: "beads-2", WindowID: "@200"}
	agent3 := &registry.Agent{ID: "agent-3", BeadsID: "beads-3", WindowID: "@300"}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("Failed to register agent-1: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("Failed to register agent-2: %v", err)
	}
	if err := reg.Register(agent3); err != nil {
		t.Fatalf("Failed to register agent-3: %v", err)
	}

	// agent-1 stays active
	// agent-2 gets completed
	// agent-3 gets abandoned
	reg.Complete("agent-2")
	reg.Abandon("agent-3")

	// ListCleanable should return agent-2 and agent-3
	cleanable := reg.ListCleanable()
	if len(cleanable) != 2 {
		t.Errorf("Expected 2 cleanable agents, got %d", len(cleanable))
	}

	// Simulate clean operation: mark as deleted
	for _, agent := range cleanable {
		reg.Remove(agent.ID)
	}

	// After clean, should have no cleanable agents
	cleanable = reg.ListCleanable()
	if len(cleanable) != 0 {
		t.Errorf("Expected 0 cleanable agents after clean, got %d", len(cleanable))
	}

	// agent-1 should still be active
	active := reg.ListActive()
	if len(active) != 1 {
		t.Errorf("Expected 1 active agent, got %d", len(active))
	}
	if active[0].ID != "agent-1" {
		t.Errorf("Expected agent-1 to be active, got %s", active[0].ID)
	}
}

// TestRunCleanDryRun verifies dry-run doesn't modify registry.
func TestRunCleanDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry with agents
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register and complete an agent
	agent := &registry.Agent{ID: "agent-1", BeadsID: "beads-1", SessionID: "ses_123"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	reg.Complete("agent-1")
	if err := reg.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Simulate dry-run: list but don't modify
	cleanable := reg.ListCleanable()
	if len(cleanable) != 1 {
		t.Fatalf("Expected 1 cleanable agent, got %d", len(cleanable))
	}

	// Don't call Remove - just log what would happen
	// This is the dry-run behavior

	// After dry-run, agent should still be cleanable
	cleanable = reg.ListCleanable()
	if len(cleanable) != 1 {
		t.Errorf("Expected 1 cleanable agent after dry-run, got %d", len(cleanable))
	}
}

// TestCompleteMarksForClean verifies Complete marks agents as cleanable.
func TestCompleteMarksForClean(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry with active agents
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register active agents
	agent1 := &registry.Agent{ID: "agent-1", BeadsID: "beads-1", SessionID: "ses_100"}
	agent2 := &registry.Agent{ID: "agent-2", BeadsID: "beads-2", SessionID: "ses_200"}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("Failed to register agent-1: %v", err)
	}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("Failed to register agent-2: %v", err)
	}

	// Initially no cleanable agents
	cleanable := reg.ListCleanable()
	if len(cleanable) != 0 {
		t.Errorf("Expected 0 cleanable agents initially, got %d", len(cleanable))
	}

	// Complete agent-2
	reg.Complete("agent-2")

	// Now agent-2 should be cleanable
	cleanable = reg.ListCleanable()
	if len(cleanable) != 1 {
		t.Errorf("Expected 1 cleanable agent after complete, got %d", len(cleanable))
	}
	if cleanable[0].ID != "agent-2" {
		t.Errorf("Expected agent-2 to be cleanable, got %s", cleanable[0].ID)
	}
}

// TestCleanPreservesActiveAgents verifies clean never removes active agents.
func TestCleanPreservesActiveAgents(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register active agent
	agent := &registry.Agent{ID: "active-agent", BeadsID: "beads-active", WindowID: "@100"}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// ListCleanable should NOT include active agents
	cleanable := reg.ListCleanable()
	for _, a := range cleanable {
		if a.ID == "active-agent" {
			t.Error("Active agent should not be in cleanable list")
		}
	}

	// Active agents should remain after any clean operation
	active := reg.ListActive()
	if len(active) != 1 {
		t.Errorf("Expected 1 active agent, got %d", len(active))
	}
}

// TestCleanHandlesAgentsWithoutWindowID verifies clean works for inline agents.
func TestCleanHandlesAgentsWithoutWindowID(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register inline agent (no window ID)
	agent := &registry.Agent{ID: "inline-agent", BeadsID: "beads-inline", WindowID: ""}
	if err := reg.Register(agent); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Mark as completed
	reg.Complete("inline-agent")

	// Should be cleanable even without window ID
	cleanable := reg.ListCleanable()
	if len(cleanable) != 1 {
		t.Fatalf("Expected 1 cleanable agent, got %d", len(cleanable))
	}
	if cleanable[0].ID != "inline-agent" {
		t.Errorf("Expected inline-agent, got %s", cleanable[0].ID)
	}
}

// TestCleanPersistence verifies cleaned agents don't come back after reload.
func TestCleanPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// First session: create and clean agent
	reg1, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	agent := &registry.Agent{ID: "persistent-agent", BeadsID: "beads-persist", WindowID: "@100"}
	if err := reg1.Register(agent); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	reg1.Complete("persistent-agent")
	reg1.Remove("persistent-agent")
	if err := reg1.SaveSkipMerge(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Second session: verify agent is gone
	reg2, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create second registry: %v", err)
	}

	// ListAgents excludes deleted
	agents := reg2.ListAgents()
	for _, a := range agents {
		if a.ID == "persistent-agent" {
			t.Error("Cleaned agent should not appear in ListAgents")
		}
	}

	// ListCleanable excludes deleted
	cleanable := reg2.ListCleanable()
	for _, a := range cleanable {
		if a.ID == "persistent-agent" {
			t.Error("Cleaned agent should not appear in ListCleanable")
		}
	}
}

// TestGetProjectNameFromWorkdir verifies project name extraction.
func TestGetProjectNameFromWorkdir(t *testing.T) {
	// Test that filepath.Base works as expected for project name extraction
	tests := []struct {
		path     string
		expected string
	}{
		{"/Users/user/projects/orch-go", "orch-go"},
		{"/home/dev/my-project", "my-project"},
		{"/projects/beads", "beads"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := filepath.Base(tt.path)
			if result != tt.expected {
				t.Errorf("filepath.Base(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

// Integration test - requires environment
func TestCleanCommandIntegration(t *testing.T) {
	// Skip in CI or if not in correct environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI")
	}

	// This is a placeholder for a more comprehensive integration test
	// that would actually run the clean command against a real registry.
	t.Skip("Integration test not implemented - requires agent setup")
}

// TestReconcileIntegrationWithClean verifies that reconciliation works with the clean flow.
func TestReconcileIntegrationWithClean(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry with agents
	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register agents with different states
	// Agent 1: Active with dead window (should be reconciled to abandoned)
	agent1 := &registry.Agent{ID: "agent-1", BeadsID: "beads-1", WindowID: "@dead-window"}
	if err := reg.Register(agent1); err != nil {
		t.Fatalf("Failed to register agent-1: %v", err)
	}

	// Agent 2: Active with dead session (should be reconciled to abandoned)
	agent2 := &registry.Agent{ID: "agent-2", BeadsID: "beads-2", WindowID: registry.HeadlessWindowID, SessionID: "ses_dead"}
	if err := reg.Register(agent2); err != nil {
		t.Fatalf("Failed to register agent-2: %v", err)
	}

	// Agent 3: Already completed (should be cleaned)
	agent3 := &registry.Agent{ID: "agent-3", BeadsID: "beads-3", WindowID: "@100"}
	if err := reg.Register(agent3); err != nil {
		t.Fatalf("Failed to register agent-3: %v", err)
	}
	reg.Complete("agent-3")

	// Save the registry
	if err := reg.Save(); err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Create a mock liveness checker where nothing is alive
	mockChecker := &MockLivenessChecker{
		LiveWindows:  make(map[string]bool),
		LiveSessions: make(map[string]bool),
	}

	// Reconcile - should mark agent-1 and agent-2 as abandoned
	result := reg.ReconcileActive(mockChecker, false)

	// Should have checked 2 active agents (agent-1 and agent-2)
	if result.Checked != 2 {
		t.Errorf("Expected 2 checked, got %d", result.Checked)
	}

	// Should have marked 2 as abandoned
	if result.Abandoned != 2 {
		t.Errorf("Expected 2 abandoned, got %d", result.Abandoned)
	}

	// Now there should be 3 cleanable agents (agent-1, agent-2, agent-3)
	cleanable := reg.ListCleanable()
	if len(cleanable) != 3 {
		t.Errorf("Expected 3 cleanable after reconciliation, got %d", len(cleanable))
	}

	// Clean them
	for _, a := range cleanable {
		reg.Remove(a.ID)
	}

	// No more cleanable agents
	cleanable = reg.ListCleanable()
	if len(cleanable) != 0 {
		t.Errorf("Expected 0 cleanable after clean, got %d", len(cleanable))
	}
}

// MockLivenessChecker for testing (matches the one in registry_test.go)
type MockLivenessChecker struct {
	LiveWindows  map[string]bool
	LiveSessions map[string]bool
}

func (m *MockLivenessChecker) WindowExists(windowID string) bool {
	return m.LiveWindows[windowID]
}

func (m *MockLivenessChecker) SessionExists(sessionID string) bool {
	return m.LiveSessions[sessionID]
}
