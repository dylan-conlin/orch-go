package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testDB creates a temporary database for testing.
func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// testAgent returns a minimal agent for testing.
func testAgent(name string) *Agent {
	return &Agent{
		WorkspaceName: name,
		BeadsID:       "orch-go-" + name,
		Mode:          "opencode",
		Skill:         "feature-impl",
		Model:         "claude-sonnet-4-5-20250929",
		Tier:          "light",
		ProjectDir:    "/Users/test/orch-go",
		ProjectName:   "orch-go",
		SpawnTime:     time.Now().UnixMilli(),
		GitBaseline:   "abc123",
		IssueTitle:    "Test issue",
		IssueType:     "feature",
		IssuePriority: 1,
	}
}

func TestOpenAndClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if db.Path() != path {
		t.Errorf("Path() = %q, want %q", db.Path(), path)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify DB file was created
	if _, err := os.Stat(path); err != nil {
		t.Errorf("database file not created: %v", err)
	}
}

func TestOpenCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Verify parent directories were created
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Errorf("parent directory not created: %v", err)
	}
}

func TestWALModeEnabled(t *testing.T) {
	db := testDB(t)

	var journalMode string
	err := db.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}
}

func TestInsertAgent(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-test-06feb-a1b2")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Verify the agent was inserted
	got, err := db.GetAgent("og-feat-test-06feb-a1b2")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if got.WorkspaceName != agent.WorkspaceName {
		t.Errorf("WorkspaceName = %q, want %q", got.WorkspaceName, agent.WorkspaceName)
	}
	if got.BeadsID != agent.BeadsID {
		t.Errorf("BeadsID = %q, want %q", got.BeadsID, agent.BeadsID)
	}
	if got.Mode != "opencode" {
		t.Errorf("Mode = %q, want %q", got.Mode, "opencode")
	}
	if got.Skill != "feature-impl" {
		t.Errorf("Skill = %q, want %q", got.Skill, "feature-impl")
	}
	if got.Tier != "light" {
		t.Errorf("Tier = %q, want %q", got.Tier, "light")
	}
	if got.IsCompleted {
		t.Error("IsCompleted should be false")
	}
	if got.IsAbandoned {
		t.Error("IsAbandoned should be false")
	}
	if got.CreatedAt == 0 {
		t.Error("CreatedAt should be set")
	}
	if got.UpdatedAt == 0 {
		t.Error("UpdatedAt should be set")
	}
}

func TestInsertAgentDuplicate(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-dup-06feb-c3d4")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("First InsertAgent failed: %v", err)
	}

	// Second insert should fail (workspace_name is PRIMARY KEY)
	if err := db.InsertAgent(agent); err == nil {
		t.Error("Second InsertAgent should have failed for duplicate workspace_name")
	}
}

func TestInsertAgentMinimal(t *testing.T) {
	db := testDB(t)

	// Minimal agent with only required fields
	agent := &Agent{
		WorkspaceName: "og-feat-minimal",
		Mode:          "opencode",
		ProjectDir:    "/Users/test/project",
		SpawnTime:     time.Now().UnixMilli(),
	}

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent with minimal fields failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-minimal")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	// Nullable fields should be empty
	if got.BeadsID != "" {
		t.Errorf("BeadsID = %q, want empty", got.BeadsID)
	}
	if got.Skill != "" {
		t.Errorf("Skill = %q, want empty", got.Skill)
	}
}

func TestGetAgentByBeadsID(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-lookup-06feb-e5f6")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	got, err := db.GetAgentByBeadsID("orch-go-og-feat-lookup-06feb-e5f6")
	if err != nil {
		t.Fatalf("GetAgentByBeadsID failed: %v", err)
	}

	if got.WorkspaceName != agent.WorkspaceName {
		t.Errorf("WorkspaceName = %q, want %q", got.WorkspaceName, agent.WorkspaceName)
	}
}

func TestGetAgentNotFound(t *testing.T) {
	db := testDB(t)

	_, err := db.GetAgent("nonexistent")
	if err == nil {
		t.Error("GetAgent should return error for nonexistent agent")
	}
}

func TestUpdateCompleted(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-complete-06feb-g7h8")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Mark as completed
	if err := db.UpdateCompleted("og-feat-complete-06feb-g7h8"); err != nil {
		t.Fatalf("UpdateCompleted failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-complete-06feb-g7h8")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if !got.IsCompleted {
		t.Error("IsCompleted should be true after UpdateCompleted")
	}
	if got.CompletedAt == 0 {
		t.Error("CompletedAt should be set after UpdateCompleted")
	}
	if got.IsAbandoned {
		t.Error("IsAbandoned should still be false")
	}
}

func TestUpdateCompletedByBeadsID(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-complete-bd-06feb-i9j0")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Mark as completed by beads ID
	if err := db.UpdateCompletedByBeadsID("orch-go-og-feat-complete-bd-06feb-i9j0"); err != nil {
		t.Fatalf("UpdateCompletedByBeadsID failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-complete-bd-06feb-i9j0")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if !got.IsCompleted {
		t.Error("IsCompleted should be true after UpdateCompletedByBeadsID")
	}
}

func TestUpdateCompletedNotFound(t *testing.T) {
	db := testDB(t)

	if err := db.UpdateCompleted("nonexistent"); err == nil {
		t.Error("UpdateCompleted should return error for nonexistent agent")
	}
}

func TestUpdateAbandoned(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-abandon-06feb-k1l2")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Mark as abandoned
	if err := db.UpdateAbandoned("og-feat-abandon-06feb-k1l2"); err != nil {
		t.Fatalf("UpdateAbandoned failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-abandon-06feb-k1l2")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if !got.IsAbandoned {
		t.Error("IsAbandoned should be true after UpdateAbandoned")
	}
	if got.AbandonedAt == 0 {
		t.Error("AbandonedAt should be set after UpdateAbandoned")
	}
	if got.IsCompleted {
		t.Error("IsCompleted should still be false")
	}
}

func TestUpdateAbandonedByBeadsID(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-abandon-bd-06feb-m3n4")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	if err := db.UpdateAbandonedByBeadsID("orch-go-og-feat-abandon-bd-06feb-m3n4"); err != nil {
		t.Fatalf("UpdateAbandonedByBeadsID failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-abandon-bd-06feb-m3n4")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if !got.IsAbandoned {
		t.Error("IsAbandoned should be true after UpdateAbandonedByBeadsID")
	}
}

func TestUpdateSessionID(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-session-06feb-o5p6")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	sessionID := "session-abc-123"
	if err := db.UpdateSessionID("og-feat-session-06feb-o5p6", sessionID); err != nil {
		t.Fatalf("UpdateSessionID failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-session-06feb-o5p6")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if got.SessionID != sessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, sessionID)
	}
}

func TestUpdateTmuxWindow(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-tmux-06feb-q7r8")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	if err := db.UpdateTmuxWindow("og-feat-tmux-06feb-q7r8", "workers-1:4"); err != nil {
		t.Fatalf("UpdateTmuxWindow failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-tmux-06feb-q7r8")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if got.TmuxWindow != "workers-1:4" {
		t.Errorf("TmuxWindow = %q, want %q", got.TmuxWindow, "workers-1:4")
	}
}

func TestUpdatePhase(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-phase-06feb-s9t0")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	if err := db.UpdatePhase("og-feat-phase-06feb-s9t0", "Implementing", "Building SQLite schema"); err != nil {
		t.Fatalf("UpdatePhase failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-phase-06feb-s9t0")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if got.Phase != "Implementing" {
		t.Errorf("Phase = %q, want %q", got.Phase, "Implementing")
	}
	if got.PhaseSummary != "Building SQLite schema" {
		t.Errorf("PhaseSummary = %q, want %q", got.PhaseSummary, "Building SQLite schema")
	}
	if got.PhaseReportedAt == 0 {
		t.Error("PhaseReportedAt should be set")
	}
}

func TestUpdatePhaseByBeadsID(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-phase-bd-06feb-u1v2")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	if err := db.UpdatePhaseByBeadsID("orch-go-og-feat-phase-bd-06feb-u1v2", "Complete", "All tests passing"); err != nil {
		t.Fatalf("UpdatePhaseByBeadsID failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-phase-bd-06feb-u1v2")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if got.Phase != "Complete" {
		t.Errorf("Phase = %q, want %q", got.Phase, "Complete")
	}
	if got.PhaseSummary != "All tests passing" {
		t.Errorf("PhaseSummary = %q, want %q", got.PhaseSummary, "All tests passing")
	}
}

func TestListActiveAgents(t *testing.T) {
	db := testDB(t)

	// Insert 3 agents
	for _, name := range []string{"active-1", "active-2", "completed-1"} {
		agent := testAgent(name)
		if err := db.InsertAgent(agent); err != nil {
			t.Fatalf("InsertAgent %s failed: %v", name, err)
		}
	}

	// Complete one, abandon none
	if err := db.UpdateCompleted("completed-1"); err != nil {
		t.Fatalf("UpdateCompleted failed: %v", err)
	}

	// Should only return 2 active agents
	active, err := db.ListActiveAgents()
	if err != nil {
		t.Fatalf("ListActiveAgents failed: %v", err)
	}

	if len(active) != 2 {
		t.Errorf("ListActiveAgents returned %d agents, want 2", len(active))
	}
}

func TestListActiveAgentsExcludesAbandoned(t *testing.T) {
	db := testDB(t)

	// Insert 3 agents
	for _, name := range []string{"a-active", "a-abandoned", "a-completed"} {
		agent := testAgent(name)
		if err := db.InsertAgent(agent); err != nil {
			t.Fatalf("InsertAgent %s failed: %v", name, err)
		}
	}

	if err := db.UpdateAbandoned("a-abandoned"); err != nil {
		t.Fatalf("UpdateAbandoned failed: %v", err)
	}
	if err := db.UpdateCompleted("a-completed"); err != nil {
		t.Fatalf("UpdateCompleted failed: %v", err)
	}

	active, err := db.ListActiveAgents()
	if err != nil {
		t.Fatalf("ListActiveAgents failed: %v", err)
	}

	if len(active) != 1 {
		t.Errorf("ListActiveAgents returned %d agents, want 1", len(active))
	}
	if active[0].WorkspaceName != "a-active" {
		t.Errorf("Active agent = %q, want %q", active[0].WorkspaceName, "a-active")
	}
}

func TestListAgentsByProject(t *testing.T) {
	db := testDB(t)

	// Insert agents for different projects
	for _, name := range []string{"proj-a1", "proj-a2"} {
		agent := testAgent(name)
		agent.ProjectName = "orch-go"
		if err := db.InsertAgent(agent); err != nil {
			t.Fatalf("InsertAgent %s failed: %v", name, err)
		}
	}
	other := testAgent("proj-b1")
	other.ProjectName = "other-project"
	if err := db.InsertAgent(other); err != nil {
		t.Fatalf("InsertAgent proj-b1 failed: %v", err)
	}

	agents, err := db.ListAgentsByProject("orch-go")
	if err != nil {
		t.Fatalf("ListAgentsByProject failed: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("ListAgentsByProject returned %d agents, want 2", len(agents))
	}
}

func TestListAllAgents(t *testing.T) {
	db := testDB(t)

	// Insert 3 agents with various states
	for _, name := range []string{"all-1", "all-2", "all-3"} {
		agent := testAgent(name)
		if err := db.InsertAgent(agent); err != nil {
			t.Fatalf("InsertAgent %s failed: %v", name, err)
		}
	}
	if err := db.UpdateCompleted("all-2"); err != nil {
		t.Fatalf("UpdateCompleted failed: %v", err)
	}
	if err := db.UpdateAbandoned("all-3"); err != nil {
		t.Fatalf("UpdateAbandoned failed: %v", err)
	}

	all, err := db.ListAllAgents()
	if err != nil {
		t.Fatalf("ListAllAgents failed: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("ListAllAgents returned %d agents, want 3", len(all))
	}
}

func TestUpdatedAtChangesOnUpdate(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-ts-06feb-w3x4")

	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	got1, _ := db.GetAgent("og-feat-ts-06feb-w3x4")
	origUpdatedAt := got1.UpdatedAt

	// Wait a bit to ensure time advances
	time.Sleep(2 * time.Millisecond)

	if err := db.UpdatePhase("og-feat-ts-06feb-w3x4", "Testing", "Running tests"); err != nil {
		t.Fatalf("UpdatePhase failed: %v", err)
	}

	got2, _ := db.GetAgent("og-feat-ts-06feb-w3x4")
	if got2.UpdatedAt <= origUpdatedAt {
		t.Errorf("UpdatedAt should increase after update: was %d, now %d", origUpdatedAt, got2.UpdatedAt)
	}
}

func TestConcurrentReads(t *testing.T) {
	db := testDB(t)

	// Insert an agent
	agent := testAgent("og-feat-concurrent-06feb-y5z6")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Concurrent reads should work with WAL mode
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := db.GetAgent("og-feat-concurrent-06feb-y5z6")
			done <- err
		}()
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent read %d failed: %v", i, err)
		}
	}
}

func TestOpenDefaultPath(t *testing.T) {
	// Test that DefaultDBPath returns a non-empty path
	path := DefaultDBPath()
	if path == "" {
		t.Skip("Could not determine home directory")
	}

	// Should end with .orch/state.db
	if filepath.Base(path) != "state.db" {
		t.Errorf("DefaultDBPath() = %q, want to end with 'state.db'", path)
	}
	if filepath.Base(filepath.Dir(path)) != ".orch" {
		t.Errorf("DefaultDBPath() parent = %q, want '.orch'", filepath.Dir(path))
	}
}

func TestSchemaIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// Open and close twice - createSchema should be idempotent
	db1, err := Open(path)
	if err != nil {
		t.Fatalf("First Open failed: %v", err)
	}
	db1.Close()

	db2, err := Open(path)
	if err != nil {
		t.Fatalf("Second Open failed: %v", err)
	}
	defer db2.Close()

	// Should still work
	agent := testAgent("idempotent-test")
	if err := db2.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent after reopen failed: %v", err)
	}
}

func TestUpdateProcessingBySessionID(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-proc-06feb-a1b2")
	agent.SessionID = "session-proc-123"
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Set processing to true
	if err := db.UpdateProcessingBySessionID("session-proc-123", true); err != nil {
		t.Fatalf("UpdateProcessingBySessionID(true) failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-proc-06feb-a1b2")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if !got.IsProcessing {
		t.Error("IsProcessing should be true")
	}
	if got.SessionUpdatedAt == 0 {
		t.Error("SessionUpdatedAt should be set")
	}

	// Set processing to false
	if err := db.UpdateProcessingBySessionID("session-proc-123", false); err != nil {
		t.Fatalf("UpdateProcessingBySessionID(false) failed: %v", err)
	}

	got2, err := db.GetAgent("og-feat-proc-06feb-a1b2")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if got2.IsProcessing {
		t.Error("IsProcessing should be false")
	}
}

func TestUpdateProcessingBySessionIDNotFound(t *testing.T) {
	db := testDB(t)

	// Should succeed even if no rows match (no agent with this session_id)
	err := db.UpdateProcessingBySessionID("nonexistent-session", true)
	if err != nil {
		t.Errorf("UpdateProcessingBySessionID should not error for unknown session: %v", err)
	}
}

func TestUpdateSessionActivity(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-activity-06feb-c3d4")
	agent.SessionID = "session-activity-456"
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Update activity
	if err := db.UpdateSessionActivity("session-activity-456"); err != nil {
		t.Fatalf("UpdateSessionActivity failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-activity-06feb-c3d4")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if got.SessionUpdatedAt == 0 {
		t.Error("SessionUpdatedAt should be set after UpdateSessionActivity")
	}
	if got.UpdatedAt == 0 {
		t.Error("UpdatedAt should be set after UpdateSessionActivity")
	}
}

func TestUpdateTokensBySessionID(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-tokens-06feb-e5f6")
	agent.SessionID = "session-tokens-789"
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Update tokens
	if err := db.UpdateTokensBySessionID("session-tokens-789", 1000, 500, 200, 300); err != nil {
		t.Fatalf("UpdateTokensBySessionID failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-tokens-06feb-e5f6")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if got.TokensInput != 1000 {
		t.Errorf("TokensInput = %d, want 1000", got.TokensInput)
	}
	if got.TokensOutput != 500 {
		t.Errorf("TokensOutput = %d, want 500", got.TokensOutput)
	}
	if got.TokensReasoning != 200 {
		t.Errorf("TokensReasoning = %d, want 200", got.TokensReasoning)
	}
	if got.TokensCacheRead != 300 {
		t.Errorf("TokensCacheRead = %d, want 300", got.TokensCacheRead)
	}
	if got.TokensTotal != 2000 {
		t.Errorf("TokensTotal = %d, want 2000 (sum)", got.TokensTotal)
	}
}

func TestGetAgentBySessionID(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-getsess-06feb-g7h8")
	agent.SessionID = "session-get-abc"
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	got, err := db.GetAgentBySessionID("session-get-abc")
	if err != nil {
		t.Fatalf("GetAgentBySessionID failed: %v", err)
	}
	if got.WorkspaceName != "og-feat-getsess-06feb-g7h8" {
		t.Errorf("WorkspaceName = %q, want %q", got.WorkspaceName, "og-feat-getsess-06feb-g7h8")
	}
}

func TestGetAgentBySessionIDNotFound(t *testing.T) {
	db := testDB(t)

	_, err := db.GetAgentBySessionID("nonexistent")
	if err == nil {
		t.Error("GetAgentBySessionID should return error for nonexistent session")
	}
}

// === UpsertAgent tests ===

func TestUpsertAgent_NewInsert(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-upsert-new-06feb-x1y2")

	if err := db.UpsertAgent(agent); err != nil {
		t.Fatalf("UpsertAgent (new) failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-upsert-new-06feb-x1y2")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if got.WorkspaceName != agent.WorkspaceName {
		t.Errorf("WorkspaceName = %q, want %q", got.WorkspaceName, agent.WorkspaceName)
	}
	if got.BeadsID != agent.BeadsID {
		t.Errorf("BeadsID = %q, want %q", got.BeadsID, agent.BeadsID)
	}
}

func TestUpsertAgent_ReplacesByBeadsID(t *testing.T) {
	db := testDB(t)

	// Insert first agent (simulates original spawn)
	agent1 := &Agent{
		WorkspaceName: "og-feat-old-workspace-06feb-a1b2",
		BeadsID:       "orch-go-21398",
		Mode:          "opencode",
		Skill:         "feature-impl",
		ProjectDir:    "/Users/test/orch-go",
		SpawnTime:     time.Now().Add(-1 * time.Hour).UnixMilli(),
	}
	if err := db.UpsertAgent(agent1); err != nil {
		t.Fatalf("UpsertAgent (first) failed: %v", err)
	}

	// Mark it as abandoned (simulating the drift scenario)
	if err := db.UpdateAbandoned(agent1.WorkspaceName); err != nil {
		t.Fatalf("UpdateAbandoned failed: %v", err)
	}

	// Upsert with same beads ID but different workspace (respawn)
	agent2 := &Agent{
		WorkspaceName: "og-feat-new-workspace-06feb-c3d4",
		BeadsID:       "orch-go-21398",
		Mode:          "opencode",
		Skill:         "feature-impl",
		ProjectDir:    "/Users/test/orch-go",
		SpawnTime:     time.Now().UnixMilli(),
		SessionID:     "ses_new_session_id",
	}
	if err := db.UpsertAgent(agent2); err != nil {
		t.Fatalf("UpsertAgent (respawn) failed: %v", err)
	}

	// Old workspace should no longer exist
	_, err := db.GetAgent("og-feat-old-workspace-06feb-a1b2")
	if err == nil {
		t.Error("Old workspace should have been deleted by upsert")
	}

	// New workspace should exist with fresh data
	got, err := db.GetAgentByBeadsID("orch-go-21398")
	if err != nil {
		t.Fatalf("GetAgentByBeadsID after upsert failed: %v", err)
	}

	if got.WorkspaceName != "og-feat-new-workspace-06feb-c3d4" {
		t.Errorf("WorkspaceName = %q, want %q", got.WorkspaceName, "og-feat-new-workspace-06feb-c3d4")
	}
	if got.SessionID != "ses_new_session_id" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "ses_new_session_id")
	}
	if got.IsAbandoned {
		t.Error("Respawned agent should NOT be abandoned")
	}
	if got.IsCompleted {
		t.Error("Respawned agent should NOT be completed")
	}
}

func TestUpsertAgent_NoBeadsID(t *testing.T) {
	db := testDB(t)

	// Upsert without beads ID should work (no deletion needed)
	agent := &Agent{
		WorkspaceName: "og-feat-no-beads-06feb-e5f6",
		Mode:          "opencode",
		ProjectDir:    "/Users/test/orch-go",
		SpawnTime:     time.Now().UnixMilli(),
	}
	if err := db.UpsertAgent(agent); err != nil {
		t.Fatalf("UpsertAgent without beads ID failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-no-beads-06feb-e5f6")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if got.BeadsID != "" {
		t.Errorf("BeadsID = %q, want empty", got.BeadsID)
	}
}

// === DriftMetrics tests ===

func TestGetDriftMetrics_Empty(t *testing.T) {
	db := testDB(t)

	metrics, err := db.GetDriftMetrics()
	if err != nil {
		t.Fatalf("GetDriftMetrics failed: %v", err)
	}

	if metrics.TotalAgents != 0 {
		t.Errorf("TotalAgents = %d, want 0", metrics.TotalAgents)
	}
	if metrics.ActiveAgents != 0 {
		t.Errorf("ActiveAgents = %d, want 0", metrics.ActiveAgents)
	}
}

func TestGetDriftMetrics_MissingSessionID(t *testing.T) {
	db := testDB(t)

	// Insert an active agent without session_id
	agent := &Agent{
		WorkspaceName: "og-feat-drift-nosess-06feb-g7h8",
		BeadsID:       "orch-go-drift-1",
		Mode:          "opencode",
		ProjectDir:    "/Users/test/orch-go",
		SpawnTime:     time.Now().UnixMilli(),
	}
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	metrics, err := db.GetDriftMetrics()
	if err != nil {
		t.Fatalf("GetDriftMetrics failed: %v", err)
	}

	if metrics.TotalAgents != 1 {
		t.Errorf("TotalAgents = %d, want 1", metrics.TotalAgents)
	}
	if metrics.ActiveAgents != 1 {
		t.Errorf("ActiveAgents = %d, want 1", metrics.ActiveAgents)
	}
	if metrics.MissingSessionID != 1 {
		t.Errorf("MissingSessionID = %d, want 1", metrics.MissingSessionID)
	}
}

func TestGetDriftMetrics_MissingTmuxWindow(t *testing.T) {
	db := testDB(t)

	// Insert a claude-mode agent without tmux_window
	agent := &Agent{
		WorkspaceName: "og-feat-drift-notmux-06feb-i9j0",
		BeadsID:       "orch-go-drift-2",
		Mode:          "claude",
		ProjectDir:    "/Users/test/orch-go",
		SpawnTime:     time.Now().UnixMilli(),
	}
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	metrics, err := db.GetDriftMetrics()
	if err != nil {
		t.Fatalf("GetDriftMetrics failed: %v", err)
	}

	if metrics.MissingTmuxWindow != 1 {
		t.Errorf("MissingTmuxWindow = %d, want 1", metrics.MissingTmuxWindow)
	}
}

func TestGetDriftMetrics_StaleActive(t *testing.T) {
	db := testDB(t)

	// Insert an agent with old updated_at (simulating stale active)
	agent := &Agent{
		WorkspaceName: "og-feat-drift-stale-06feb-k1l2",
		BeadsID:       "orch-go-drift-3",
		Mode:          "opencode",
		SessionID:     "ses_stale_test",
		ProjectDir:    "/Users/test/orch-go",
		SpawnTime:     time.Now().Add(-3 * time.Hour).UnixMilli(),
		CreatedAt:     time.Now().Add(-3 * time.Hour).UnixMilli(),
		UpdatedAt:     time.Now().Add(-3 * time.Hour).UnixMilli(), // 3 hours ago > 2h threshold
	}
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	metrics, err := db.GetDriftMetrics()
	if err != nil {
		t.Fatalf("GetDriftMetrics failed: %v", err)
	}

	if metrics.StaleActive != 1 {
		t.Errorf("StaleActive = %d, want 1", metrics.StaleActive)
	}
}

func TestGetDriftMetrics_CompletedExcluded(t *testing.T) {
	db := testDB(t)

	// Insert a completed agent without session_id — should NOT count as drift
	agent := &Agent{
		WorkspaceName: "og-feat-drift-done-06feb-m3n4",
		BeadsID:       "orch-go-drift-4",
		Mode:          "opencode",
		ProjectDir:    "/Users/test/orch-go",
		SpawnTime:     time.Now().UnixMilli(),
	}
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}
	if err := db.UpdateCompleted(agent.WorkspaceName); err != nil {
		t.Fatalf("UpdateCompleted failed: %v", err)
	}

	metrics, err := db.GetDriftMetrics()
	if err != nil {
		t.Fatalf("GetDriftMetrics failed: %v", err)
	}

	if metrics.ActiveAgents != 0 {
		t.Errorf("ActiveAgents = %d, want 0 (completed agents excluded)", metrics.ActiveAgents)
	}
	if metrics.MissingSessionID != 0 {
		t.Errorf("MissingSessionID = %d, want 0 (completed agents excluded)", metrics.MissingSessionID)
	}
}

// === Phase Source Tracking and Adoption Telemetry tests ===

func TestUpdatePhaseWithSource(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-phase-src-07feb-a1b2")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Update phase with explicit source tracking
	if err := db.UpdatePhaseWithSource("og-feat-phase-src-07feb-a1b2", "Implementing", "Building feature", PhaseSourceOrchPhase); err != nil {
		t.Fatalf("UpdatePhaseWithSource failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-phase-src-07feb-a1b2")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if got.Phase != "Implementing" {
		t.Errorf("Phase = %q, want %q", got.Phase, "Implementing")
	}
	if got.PhaseSource != PhaseSourceOrchPhase {
		t.Errorf("PhaseSource = %q, want %q", got.PhaseSource, PhaseSourceOrchPhase)
	}
}

func TestUpdatePhaseDefaultsToOrchPhaseSource(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-phase-default-07feb-c3d4")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// UpdatePhase (no explicit source) should default to orch_phase
	if err := db.UpdatePhase("og-feat-phase-default-07feb-c3d4", "Planning", "Analyzing"); err != nil {
		t.Fatalf("UpdatePhase failed: %v", err)
	}

	got, err := db.GetAgent("og-feat-phase-default-07feb-c3d4")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	if got.PhaseSource != PhaseSourceOrchPhase {
		t.Errorf("PhaseSource = %q, want %q (should default to orch_phase)", got.PhaseSource, PhaseSourceOrchPhase)
	}
}

func TestUpdatePhaseByBeadsIDWithSource(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-phase-bdsrc-07feb-e5f6")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Simulate bd_comment source (from status enrichment or reconciliation)
	if err := db.UpdatePhaseByBeadsIDWithSource("orch-go-og-feat-phase-bdsrc-07feb-e5f6", "Testing", "Running tests", PhaseSourceBdComment); err != nil {
		t.Fatalf("UpdatePhaseByBeadsIDWithSource failed: %v", err)
	}

	got, err := db.GetAgentByBeadsID("orch-go-og-feat-phase-bdsrc-07feb-e5f6")
	if err != nil {
		t.Fatalf("GetAgentByBeadsID failed: %v", err)
	}

	if got.PhaseSource != PhaseSourceBdComment {
		t.Errorf("PhaseSource = %q, want %q", got.PhaseSource, PhaseSourceBdComment)
	}
}

func TestGetPhaseAdoptionMetrics_Empty(t *testing.T) {
	db := testDB(t)

	metrics, err := db.GetPhaseAdoptionMetrics()
	if err != nil {
		t.Fatalf("GetPhaseAdoptionMetrics failed: %v", err)
	}

	if metrics.TotalWithPhase != 0 {
		t.Errorf("TotalWithPhase = %d, want 0", metrics.TotalWithPhase)
	}
}

func TestGetPhaseAdoptionMetrics_Mixed(t *testing.T) {
	db := testDB(t)

	// Agent 1: uses orch phase (modern path)
	a1 := testAgent("adoption-orch-phase")
	if err := db.InsertAgent(a1); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}
	if err := db.UpdatePhaseWithSource("adoption-orch-phase", "Implementing", "building", PhaseSourceOrchPhase); err != nil {
		t.Fatalf("UpdatePhaseWithSource failed: %v", err)
	}

	// Agent 2: uses bd comment (legacy path)
	a2 := testAgent("adoption-bd-comment")
	if err := db.InsertAgent(a2); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}
	if err := db.UpdatePhaseByBeadsIDWithSource("orch-go-adoption-bd-comment", "Planning", "analyzing", PhaseSourceBdComment); err != nil {
		t.Fatalf("UpdatePhaseByBeadsIDWithSource failed: %v", err)
	}

	// Agent 3: has phase but no source tracked (pre-telemetry)
	a3 := testAgent("adoption-unknown")
	a3.Phase = "Testing"
	if err := db.InsertAgent(a3); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	// Agent 4: no phase set at all (should not count)
	a4 := testAgent("adoption-no-phase")
	if err := db.InsertAgent(a4); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	metrics, err := db.GetPhaseAdoptionMetrics()
	if err != nil {
		t.Fatalf("GetPhaseAdoptionMetrics failed: %v", err)
	}

	if metrics.TotalWithPhase != 3 {
		t.Errorf("TotalWithPhase = %d, want 3", metrics.TotalWithPhase)
	}
	if metrics.ViaOrchPhase != 1 {
		t.Errorf("ViaOrchPhase = %d, want 1", metrics.ViaOrchPhase)
	}
	if metrics.ViaBdComment != 1 {
		t.Errorf("ViaBdComment = %d, want 1", metrics.ViaBdComment)
	}
	if metrics.ViaUnknown != 1 {
		t.Errorf("ViaUnknown = %d, want 1", metrics.ViaUnknown)
	}
}

func TestOpenRejectsRelativePath(t *testing.T) {
	// Attempt to open with relative path should fail
	_, err := Open(".orch/state.db")
	if err == nil {
		t.Fatal("expected error when opening with relative path, got nil")
	}
	if !strings.Contains(err.Error(), "must be absolute") {
		t.Errorf("expected error about absolute path, got: %v", err)
	}
}

func TestOpenAllowsTestPaths(t *testing.T) {
	// Create a temp directory to simulate a test database
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test-state.db")

	// This should succeed (absolute path in temp directory)
	db, err := Open(testPath)
	if err != nil {
		t.Fatalf("expected Open to succeed for test path in temp dir, got error: %v", err)
	}
	defer db.Close()

	// Verify database was created and is functional
	if db.Path() != testPath {
		t.Errorf("expected path %s, got %s", testPath, db.Path())
	}
}

func TestSchemaMigration_PhaseSource(t *testing.T) {
	// Test that opening a DB twice (simulating migration) is safe.
	// The ALTER TABLE ADD COLUMN should be idempotent.
	dir := t.TempDir()
	path := filepath.Join(dir, "migration-test.db")

	// Open first time — creates schema with phase_source
	db1, err := Open(path)
	if err != nil {
		t.Fatalf("Open (first) failed: %v", err)
	}
	db1.Close()

	// Open second time — migration should be idempotent
	db2, err := Open(path)
	if err != nil {
		t.Fatalf("Open (second) failed: %v", err)
	}
	defer db2.Close()

	// Should still work
	agent := &Agent{
		WorkspaceName: "migration-test-agent",
		Mode:          "opencode",
		ProjectDir:    "/tmp/test",
		SpawnTime:     time.Now().UnixMilli(),
	}
	if err := db2.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent after migration failed: %v", err)
	}
}
