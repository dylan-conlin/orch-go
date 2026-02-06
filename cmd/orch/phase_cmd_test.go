package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/state"
)

// setupPhaseTestDB creates a temp DB with a test agent for phase tests.
func setupPhaseTestDB(t *testing.T, beadsID string) (string, *state.DB) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test-state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	agent := &state.Agent{
		WorkspaceName: "og-feat-phase-test",
		BeadsID:       beadsID,
		Mode:          "opencode",
		Skill:         "feature-impl",
		ProjectDir:    "/Users/test/orch-go",
		ProjectName:   "orch-go",
		SpawnTime:     time.Now().UnixMilli(),
	}
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("failed to insert test agent: %v", err)
	}

	return dbPath, db
}

func TestRunPhaseWithDB(t *testing.T) {
	beadsID := "orch-go-12345"
	dbPath, db := setupPhaseTestDB(t, beadsID)
	defer db.Close()

	// Run phase update
	err := runPhaseWithDB(beadsID, "Implementing", "Building feature", dbPath)
	if err != nil {
		t.Fatalf("runPhaseWithDB failed: %v", err)
	}

	// Verify the phase was written
	agent, err := db.GetAgentByBeadsID(beadsID)
	if err != nil {
		t.Fatalf("GetAgentByBeadsID failed: %v", err)
	}

	if agent.Phase != "Implementing" {
		t.Errorf("Phase = %q, want %q", agent.Phase, "Implementing")
	}
	if agent.PhaseSummary != "Building feature" {
		t.Errorf("PhaseSummary = %q, want %q", agent.PhaseSummary, "Building feature")
	}
	if agent.PhaseReportedAt == 0 {
		t.Error("PhaseReportedAt should be set")
	}
}

func TestRunPhaseWithDBNoSummary(t *testing.T) {
	beadsID := "orch-go-67890"
	dbPath, db := setupPhaseTestDB(t, beadsID)
	defer db.Close()

	// Run phase update without summary
	err := runPhaseWithDB(beadsID, "Planning", "", dbPath)
	if err != nil {
		t.Fatalf("runPhaseWithDB failed: %v", err)
	}

	agent, err := db.GetAgentByBeadsID(beadsID)
	if err != nil {
		t.Fatalf("GetAgentByBeadsID failed: %v", err)
	}

	if agent.Phase != "Planning" {
		t.Errorf("Phase = %q, want %q", agent.Phase, "Planning")
	}
	if agent.PhaseSummary != "" {
		t.Errorf("PhaseSummary = %q, want empty", agent.PhaseSummary)
	}
}

func TestRunPhaseWithDBMultipleUpdates(t *testing.T) {
	beadsID := "orch-go-multi"
	dbPath, db := setupPhaseTestDB(t, beadsID)
	defer db.Close()

	// Simulate phase progression
	phases := []struct {
		phase   string
		summary string
	}{
		{"Planning", "Analyzing codebase"},
		{"Implementing", "Building SQLite write path"},
		{"Testing", "Running test suite"},
		{"Complete", "All tests passing"},
	}

	for _, p := range phases {
		err := runPhaseWithDB(beadsID, p.phase, p.summary, dbPath)
		if err != nil {
			t.Fatalf("runPhaseWithDB(%q) failed: %v", p.phase, err)
		}
	}

	// Verify final phase
	agent, err := db.GetAgentByBeadsID(beadsID)
	if err != nil {
		t.Fatalf("GetAgentByBeadsID failed: %v", err)
	}

	if agent.Phase != "Complete" {
		t.Errorf("Phase = %q, want %q", agent.Phase, "Complete")
	}
	if agent.PhaseSummary != "All tests passing" {
		t.Errorf("PhaseSummary = %q, want %q", agent.PhaseSummary, "All tests passing")
	}
}

func TestRunPhaseWithDBAgentNotFound(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test-state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// Try to update phase for non-existent agent
	err = runPhaseWithDB("orch-go-nonexistent", "Planning", "test", dbPath)
	if err == nil {
		t.Error("runPhaseWithDB should fail for non-existent agent")
	}
}

func TestRunPhaseWithDBCompletePhase(t *testing.T) {
	beadsID := "orch-go-complete-test"
	dbPath, db := setupPhaseTestDB(t, beadsID)
	defer db.Close()

	err := runPhaseWithDB(beadsID, "Complete", "All tests passing, ready for review", dbPath)
	if err != nil {
		t.Fatalf("runPhaseWithDB failed: %v", err)
	}

	agent, err := db.GetAgentByBeadsID(beadsID)
	if err != nil {
		t.Fatalf("GetAgentByBeadsID failed: %v", err)
	}

	if agent.Phase != "Complete" {
		t.Errorf("Phase = %q, want %q", agent.Phase, "Complete")
	}
	if agent.PhaseSummary != "All tests passing, ready for review" {
		t.Errorf("PhaseSummary = %q, want %q", agent.PhaseSummary, "All tests passing, ready for review")
	}
}

func TestRunPhaseWithDBBlockedPhase(t *testing.T) {
	beadsID := "orch-go-blocked-test"
	dbPath, db := setupPhaseTestDB(t, beadsID)
	defer db.Close()

	err := runPhaseWithDB(beadsID, "BLOCKED", "Need clarification on API contract", dbPath)
	if err != nil {
		t.Fatalf("runPhaseWithDB failed: %v", err)
	}

	agent, err := db.GetAgentByBeadsID(beadsID)
	if err != nil {
		t.Fatalf("GetAgentByBeadsID failed: %v", err)
	}

	if agent.Phase != "BLOCKED" {
		t.Errorf("Phase = %q, want %q", agent.Phase, "BLOCKED")
	}
}

func TestPhaseCommandArgs(t *testing.T) {
	// Test cobra argument parsing
	cmd := phaseCmd

	// Should require at least 2 args
	if cmd.Args == nil {
		t.Fatal("phaseCmd.Args should not be nil")
	}

	// Test with too few args
	err := cmd.Args(cmd, []string{"only-one"})
	if err == nil {
		t.Error("Should fail with only 1 arg")
	}

	// Test with exactly 2 args (minimum)
	err = cmd.Args(cmd, []string{"beads-id", "Planning"})
	if err != nil {
		t.Errorf("Should succeed with 2 args: %v", err)
	}

	// Test with 3+ args (summary)
	err = cmd.Args(cmd, []string{"beads-id", "Planning", "some", "summary"})
	if err != nil {
		t.Errorf("Should succeed with 4 args: %v", err)
	}
}
