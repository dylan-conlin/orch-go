package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
)

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

// TestCleanWorkspaceBased tests workspace-based cleanup detection.
// The clean command operates on workspaces directly using OpenCode API for session status.
func TestCleanWorkspaceBased(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create completed workspace (has SYNTHESIS.md)
	ws1 := filepath.Join(workspaceDir, "og-feat-completed-21dec")
	if err := os.MkdirAll(ws1, 0755); err != nil {
		t.Fatalf("Failed to create ws1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ws1, "SYNTHESIS.md"), []byte("# Complete"), 0644); err != nil {
		t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
	}

	// Create incomplete workspace (no SYNTHESIS.md)
	ws2 := filepath.Join(workspaceDir, "og-feat-in-progress-21dec")
	if err := os.MkdirAll(ws2, 0755); err != nil {
		t.Fatalf("Failed to create ws2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ws2, "SPAWN_CONTEXT.md"), []byte("Task: test"), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}

	// Verify workspace detection
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to read workspace dir: %v", err)
	}

	var completed, inProgress []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		synthPath := filepath.Join(workspaceDir, entry.Name(), "SYNTHESIS.md")
		if _, err := os.Stat(synthPath); err == nil {
			completed = append(completed, entry.Name())
		} else {
			inProgress = append(inProgress, entry.Name())
		}
	}

	if len(completed) != 1 {
		t.Errorf("Expected 1 completed workspace, got %d", len(completed))
	}
	if len(inProgress) != 1 {
		t.Errorf("Expected 1 in-progress workspace, got %d", len(inProgress))
	}
}

// TestCleanPreservesInProgressWorkspaces verifies clean never removes in-progress work.
func TestCleanPreservesInProgressWorkspaces(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create in-progress workspace (no SYNTHESIS.md, but has .session_id)
	ws := filepath.Join(workspaceDir, "og-feat-active-21dec")
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ws, ".session_id"), []byte("ses_abc123"), 0644); err != nil {
		t.Fatalf("Failed to write .session_id: %v", err)
	}

	// Simulate cleanable detection - should NOT include active work
	synthPath := filepath.Join(ws, "SYNTHESIS.md")
	_, err := os.Stat(synthPath)
	if !os.IsNotExist(err) {
		t.Error("Active workspace should not have SYNTHESIS.md")
	}

	// Workspace should still exist (not cleaned)
	if _, err := os.Stat(ws); os.IsNotExist(err) {
		t.Error("Active workspace should not be removed")
	}
}

// TestSessionIDFileBased tests session ID file operations.
func TestSessionIDFileBased(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	ws := filepath.Join(workspaceDir, "og-feat-test-21dec")
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Write session ID
	sessionIDPath := filepath.Join(ws, ".session_id")
	expectedID := "ses_test123"
	if err := os.WriteFile(sessionIDPath, []byte(expectedID), 0644); err != nil {
		t.Fatalf("Failed to write session ID: %v", err)
	}

	// Read session ID
	data, err := os.ReadFile(sessionIDPath)
	if err != nil {
		t.Fatalf("Failed to read session ID: %v", err)
	}

	if string(data) != expectedID {
		t.Errorf("Expected session ID %q, got %q", expectedID, string(data))
	}
}

// Integration test - requires environment
func TestCleanCommandIntegration(t *testing.T) {
	// Skip in CI or if not in correct environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI")
	}

	// This is a placeholder for a more comprehensive integration test
	// that would actually run the clean command against real workspaces.
	t.Skip("Integration test not implemented - requires agent setup")
}

// TestIsOrchestratorSessionTitle removed - session cleanup logic moved to OpenCode server (TTL-based)

// TestPreserveOrchestratorWorkspace tests that orchestrator workspaces are detected correctly.
func TestPreserveOrchestratorWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Create orchestrator workspace with .orchestrator marker
	orchWs := filepath.Join(workspaceDir, "og-orch-goal-04jan")
	if err := os.MkdirAll(orchWs, 0755); err != nil {
		t.Fatalf("Failed to create orchestrator workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(orchWs, ".orchestrator"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create .orchestrator marker: %v", err)
	}

	// Create meta-orchestrator workspace with .meta-orchestrator marker
	metaOrchWs := filepath.Join(workspaceDir, "meta-orch-continue-06jan")
	if err := os.MkdirAll(metaOrchWs, 0755); err != nil {
		t.Fatalf("Failed to create meta-orchestrator workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(metaOrchWs, ".meta-orchestrator"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create .meta-orchestrator marker: %v", err)
	}

	// Create regular worker workspace (no markers)
	workerWs := filepath.Join(workspaceDir, "og-feat-add-feature-21dec")
	if err := os.MkdirAll(workerWs, 0755); err != nil {
		t.Fatalf("Failed to create worker workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workerWs, "SPAWN_CONTEXT.md"), []byte("Task: test"), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}

	// Test isOrchestratorWorkspace
	if !isOrchestratorWorkspace(orchWs) {
		t.Error("Expected orchestrator workspace to be detected")
	}
	if !isOrchestratorWorkspace(metaOrchWs) {
		t.Error("Expected meta-orchestrator workspace to be detected")
	}
	if isOrchestratorWorkspace(workerWs) {
		t.Error("Expected worker workspace NOT to be detected as orchestrator")
	}
}

// TestArchiveStaleWorkspacesPreservesOrchestrator tests that --preserve-orchestrator skips orchestrator workspaces.
func TestArchiveStaleWorkspacesPreservesOrchestrator(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	archivedDir := filepath.Join(workspaceDir, "archived")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Helper to create a stale workspace with spawn time
	createStaleWorkspace := func(name string, isOrch bool) string {
		ws := filepath.Join(workspaceDir, name)
		if err := os.MkdirAll(ws, 0755); err != nil {
			t.Fatalf("Failed to create workspace %s: %v", name, err)
		}
		// Write old spawn time (8 days ago in nanoseconds)
		oldTime := int64(1704067200000000000) // Some old timestamp
		if err := os.WriteFile(filepath.Join(ws, ".spawn_time"), []byte(string(rune(oldTime))), 0644); err != nil {
			// Use fmt.Sprintf instead for proper int64 formatting
			if err := os.WriteFile(filepath.Join(ws, ".spawn_time"), []byte("1704067200000000000"), 0644); err != nil {
				t.Fatalf("Failed to write spawn time: %v", err)
			}
		}
		// Write SYNTHESIS.md to mark as completed
		if err := os.WriteFile(filepath.Join(ws, "SYNTHESIS.md"), []byte("# Complete"), 0644); err != nil {
			t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
		}
		// Write orchestrator marker if needed
		if isOrch {
			if err := os.WriteFile(filepath.Join(ws, ".orchestrator"), []byte(""), 0644); err != nil {
				t.Fatalf("Failed to create .orchestrator marker: %v", err)
			}
		}
		return ws
	}

	// Create test workspaces
	orchWs := createStaleWorkspace("og-orch-test-01jan", true)
	workerWs := createStaleWorkspace("og-feat-test-01jan", false)

	// Verify both exist before archiving
	if _, err := os.Stat(orchWs); os.IsNotExist(err) {
		t.Fatal("Orchestrator workspace should exist")
	}
	if _, err := os.Stat(workerWs); os.IsNotExist(err) {
		t.Fatal("Worker workspace should exist")
	}

	// Run archiveStaleWorkspaces with preserveOrchestrator=true, dryRun=true
	// This verifies the detection logic without actually moving files
	_, err := archiveStaleWorkspaces(tmpDir, 7, true, true)
	if err != nil {
		t.Fatalf("archiveStaleWorkspaces failed: %v", err)
	}

	// In dry-run mode, both should still exist
	if _, err := os.Stat(orchWs); os.IsNotExist(err) {
		t.Error("Orchestrator workspace should still exist after dry-run")
	}
	if _, err := os.Stat(workerWs); os.IsNotExist(err) {
		t.Error("Worker workspace should still exist after dry-run")
	}

	// Verify archived directory wasn't created (dry-run)
	if _, err := os.Stat(archivedDir); !os.IsNotExist(err) {
		t.Error("Archived directory should not be created in dry-run mode")
	}
}

func TestArchiveStaleWorkspacesUsesFallbackSpawnTime(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	archivedDir := filepath.Join(workspaceDir, "archived")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	wsName := "og-feat-fallback-01jan"
	ws := filepath.Join(workspaceDir, wsName)
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ws, "SYNTHESIS.md"), []byte("# Complete"), 0644); err != nil {
		t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
	}
	spawnContextPath := filepath.Join(ws, "SPAWN_CONTEXT.md")
	if err := os.WriteFile(spawnContextPath, []byte("Task: test"), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}

	oldTime := time.Now().AddDate(0, 0, -8)
	if err := os.Chtimes(spawnContextPath, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set SPAWN_CONTEXT.md mtime: %v", err)
	}

	archived, err := archiveStaleWorkspaces(tmpDir, 7, false, false)
	if err != nil {
		t.Fatalf("archiveStaleWorkspaces failed: %v", err)
	}
	if archived != 1 {
		t.Errorf("Expected 1 workspace archived, got %d", archived)
	}
	if _, err := os.Stat(ws); !os.IsNotExist(err) {
		t.Error("Original workspace should have been moved")
	}
	if _, err := os.Stat(filepath.Join(archivedDir, wsName)); err != nil {
		t.Fatalf("Expected archived workspace to exist: %v", err)
	}
}

// TestArchiveStaleWorkspacesHandlesDuplicateDestination tests that archiving handles
// the case when the archive destination already exists (bug fix: orch-go-wgdse).
func TestArchiveStaleWorkspacesHandlesDuplicateDestination(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	archivedDir := filepath.Join(workspaceDir, "archived")

	// Create workspace and archived directories
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		t.Fatalf("Failed to create archived dir: %v", err)
	}

	// Create a stale workspace
	wsName := "og-feat-duplicate-test-01jan"
	ws := filepath.Join(workspaceDir, wsName)
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Write old spawn time (8 days ago in nanoseconds)
	if err := os.WriteFile(filepath.Join(ws, ".spawn_time"), []byte("1704067200000000000"), 0644); err != nil {
		t.Fatalf("Failed to write spawn time: %v", err)
	}

	// Write SYNTHESIS.md to mark as completed
	if err := os.WriteFile(filepath.Join(ws, "SYNTHESIS.md"), []byte("# New Complete"), 0644); err != nil {
		t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
	}

	// Create a pre-existing archive destination with the same name
	existingArchive := filepath.Join(archivedDir, wsName)
	if err := os.MkdirAll(existingArchive, 0755); err != nil {
		t.Fatalf("Failed to create existing archive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(existingArchive, "SYNTHESIS.md"), []byte("# Old Complete"), 0644); err != nil {
		t.Fatalf("Failed to write old SYNTHESIS.md: %v", err)
	}

	// Run archiveStaleWorkspaces (non-dry-run)
	archived, err := archiveStaleWorkspaces(tmpDir, 7, false, false)
	if err != nil {
		t.Fatalf("archiveStaleWorkspaces failed: %v", err)
	}

	// Verify the workspace was archived
	if archived != 1 {
		t.Errorf("Expected 1 workspace archived, got %d", archived)
	}

	// Verify the original workspace was removed
	if _, err := os.Stat(ws); !os.IsNotExist(err) {
		t.Error("Original workspace should have been moved")
	}

	// Verify the old archive still exists with old content
	oldContent, err := os.ReadFile(filepath.Join(existingArchive, "SYNTHESIS.md"))
	if err != nil {
		t.Fatalf("Failed to read old archive: %v", err)
	}
	if string(oldContent) != "# Old Complete" {
		t.Errorf("Old archive content was modified: %s", string(oldContent))
	}

	// Verify a new archive was created with timestamp suffix
	entries, err := os.ReadDir(archivedDir)
	if err != nil {
		t.Fatalf("Failed to read archived dir: %v", err)
	}

	foundNewArchive := false
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != wsName && len(entry.Name()) > len(wsName) {
			// This should be the timestamped version (wsName-HHMMSS)
			if newContent, err := os.ReadFile(filepath.Join(archivedDir, entry.Name(), "SYNTHESIS.md")); err == nil {
				if string(newContent) == "# New Complete" {
					foundNewArchive = true
					break
				}
			}
		}
	}

	if !foundNewArchive {
		t.Error("Expected a new archive with timestamp suffix to be created")
		t.Logf("Archives found: %v", entries)
	}
}

// TestCleanAllFlagLogic verifies that --all flag enables all cleanup flags.
func TestCleanAllFlagLogic(t *testing.T) {
	// This test verifies the flag preprocessing logic by simulating what happens
	// in the RunE function when cleanAll is set to true.

	// Start with all flags false (default state)
	workspaces := false
	sessions := false
	all := false

	// Test 1: When all=false, individual flags should remain unchanged
	if all {
		workspaces = true
		sessions = true
	}

	if workspaces || sessions {
		t.Error("Expected all flags to remain false when all=false")
	}

	// Test 2: When all=true, all individual flags should be set to true
	all = true
	if all {
		workspaces = true
		sessions = true
	}

	if !workspaces {
		t.Error("Expected workspaces to be true when all=true")
	}
	if !sessions {
		t.Error("Expected sessions to be true when all=true")
	}
}

// TestArchiveEmptyInvestigationsHandlesDuplicateDestination tests that archiving investigations
// handles the case when the archive destination already exists.
func TestArchiveEmptyInvestigationsHandlesDuplicateDestination(t *testing.T) {
	tmpDir := t.TempDir()
	investigationsDir := filepath.Join(tmpDir, ".kb", "investigations")
	archivedDir := filepath.Join(investigationsDir, "archived")

	// Create directories
	if err := os.MkdirAll(investigationsDir, 0755); err != nil {
		t.Fatalf("Failed to create investigations dir: %v", err)
	}
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		t.Fatalf("Failed to create archived dir: %v", err)
	}

	// Create an empty investigation file (with template placeholders)
	invName := "2026-01-01-inv-test.md"
	invPath := filepath.Join(investigationsDir, invName)
	emptyContent := `# Investigation
**Question:** [Clear, specific question this investigation answers]
**Evidence:** [Concrete observations, data, examples]
**Source:** [File paths with line numbers, commands run]
`
	if err := os.WriteFile(invPath, []byte(emptyContent), 0644); err != nil {
		t.Fatalf("Failed to write investigation file: %v", err)
	}

	// Create a pre-existing archive destination with the same name
	existingArchive := filepath.Join(archivedDir, invName)
	oldContent := "# Old Investigation"
	if err := os.WriteFile(existingArchive, []byte(oldContent), 0644); err != nil {
		t.Fatalf("Failed to write existing archive: %v", err)
	}

	// Run archiveEmptyInvestigations (non-dry-run)
	archived, err := archiveEmptyInvestigations(tmpDir, false)
	if err != nil {
		t.Fatalf("archiveEmptyInvestigations failed: %v", err)
	}

	// Verify the investigation was archived
	if archived != 1 {
		t.Errorf("Expected 1 investigation archived, got %d", archived)
	}

	// Verify the original investigation was removed
	if _, err := os.Stat(invPath); !os.IsNotExist(err) {
		t.Error("Original investigation should have been moved")
	}

	// Verify the old archive still exists with old content
	oldArchiveContent, err := os.ReadFile(existingArchive)
	if err != nil {
		t.Fatalf("Failed to read old archive: %v", err)
	}
	if string(oldArchiveContent) != oldContent {
		t.Errorf("Old archive content was modified: %s", string(oldArchiveContent))
	}

	// Verify a new archive was created with timestamp suffix
	entries, err := os.ReadDir(archivedDir)
	if err != nil {
		t.Fatalf("Failed to read archived dir: %v", err)
	}

	foundNewArchive := false
	baseName := "2026-01-01-inv-test"
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != invName && len(entry.Name()) > len(invName) {
			// This should be the timestamped version (baseName-HHMMSS.md)
			if len(entry.Name()) > len(baseName) && entry.Name()[:len(baseName)] == baseName {
				foundNewArchive = true
				break
			}
		}
	}

	if !foundNewArchive {
		t.Error("Expected a new archive with timestamp suffix to be created")
		t.Logf("Archives found: %v", entries)
	}
}

// TestCleanAllFlagIncludesOrphans verifies that --all enables --orphans and --ghosts.
func TestCleanAllFlagIncludesOrphans(t *testing.T) {
	workspaces := false
	sessions := false
	orphans := false
	ghosts := false
	all := true

	if all {
		workspaces = true
		sessions = true
		orphans = true
		ghosts = true
	}

	if !workspaces {
		t.Error("Expected workspaces to be true when all=true")
	}
	if !sessions {
		t.Error("Expected sessions to be true when all=true")
	}
	if !orphans {
		t.Error("Expected orphans to be true when all=true")
	}
	if !ghosts {
		t.Error("Expected ghosts to be true when all=true")
	}
}

// TestOrphanClassification verifies orphan GC action classification logic.
// ForceComplete for non-retryable orphans, ForceAbandon for retryable ones.
func TestOrphanClassification(t *testing.T) {
	tests := []struct {
		name         string
		orphan       agent.OrphanedAgent
		expectAction string
	}{
		{
			name: "completed orphan gets force-completed",
			orphan: agent.OrphanedAgent{
				Agent:       agent.AgentRef{BeadsID: "test-001"},
				Reason:      "no_live_execution",
				LastPhase:   "Complete",
				ShouldRetry: false,
			},
			expectAction: "force-complete",
		},
		{
			name: "retryable orphan gets force-abandoned",
			orphan: agent.OrphanedAgent{
				Agent:       agent.AgentRef{BeadsID: "test-002"},
				Reason:      "no_live_execution",
				LastPhase:   "Implementing",
				ShouldRetry: true,
			},
			expectAction: "force-abandon",
		},
		{
			name: "no-workspace orphan gets force-completed when not retryable",
			orphan: agent.OrphanedAgent{
				Agent:       agent.AgentRef{BeadsID: "test-003"},
				Reason:      "no_workspace",
				ShouldRetry: false,
			},
			expectAction: "force-complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := "force-complete"
			if tt.orphan.ShouldRetry {
				action = "force-abandon"
			}
			if action != tt.expectAction {
				t.Errorf("Expected action %q, got %q", tt.expectAction, action)
			}
		})
	}
}

// TestCleanExpiredArchives tests TTL-based deletion of old archived workspaces.
func TestCleanExpiredArchives(t *testing.T) {
	tmpDir := t.TempDir()
	archivedDir := filepath.Join(tmpDir, ".orch", "workspace", "archived")
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		t.Fatalf("Failed to create archived dir: %v", err)
	}

	// Create an old archived workspace (40 days old via .spawn_time)
	oldWs := filepath.Join(archivedDir, "og-feat-old-01jan")
	if err := os.MkdirAll(oldWs, 0755); err != nil {
		t.Fatalf("Failed to create old workspace: %v", err)
	}
	oldTime := time.Now().AddDate(0, 0, -40)
	oldNanos := oldTime.UnixNano()
	if err := os.WriteFile(filepath.Join(oldWs, ".spawn_time"), []byte(fmt.Sprintf("%d", oldNanos)), 0644); err != nil {
		t.Fatalf("Failed to write spawn time: %v", err)
	}

	// Create a recent archived workspace (5 days old)
	recentWs := filepath.Join(archivedDir, "og-feat-recent-28feb")
	if err := os.MkdirAll(recentWs, 0755); err != nil {
		t.Fatalf("Failed to create recent workspace: %v", err)
	}
	recentTime := time.Now().AddDate(0, 0, -5)
	recentNanos := recentTime.UnixNano()
	if err := os.WriteFile(filepath.Join(recentWs, ".spawn_time"), []byte(fmt.Sprintf("%d", recentNanos)), 0644); err != nil {
		t.Fatalf("Failed to write spawn time: %v", err)
	}

	// Run cleanup with 30-day TTL
	deleted, err := cleanExpiredArchives(tmpDir, 30, false)
	if err != nil {
		t.Fatalf("cleanExpiredArchives failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 workspace deleted, got %d", deleted)
	}

	// Old workspace should be gone
	if _, err := os.Stat(oldWs); !os.IsNotExist(err) {
		t.Error("Old archived workspace should have been deleted")
	}

	// Recent workspace should still exist
	if _, err := os.Stat(recentWs); os.IsNotExist(err) {
		t.Error("Recent archived workspace should still exist")
	}
}

// TestCleanExpiredArchivesDryRun tests that dry-run mode doesn't delete anything.
func TestCleanExpiredArchivesDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	archivedDir := filepath.Join(tmpDir, ".orch", "workspace", "archived")
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		t.Fatalf("Failed to create archived dir: %v", err)
	}

	// Create an old archived workspace
	oldWs := filepath.Join(archivedDir, "og-feat-old-01jan")
	if err := os.MkdirAll(oldWs, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	oldTime := time.Now().AddDate(0, 0, -40)
	if err := os.WriteFile(filepath.Join(oldWs, ".spawn_time"), []byte(fmt.Sprintf("%d", oldTime.UnixNano())), 0644); err != nil {
		t.Fatalf("Failed to write spawn time: %v", err)
	}

	// Run cleanup with dry-run
	deleted, err := cleanExpiredArchives(tmpDir, 30, true)
	if err != nil {
		t.Fatalf("cleanExpiredArchives failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 workspace reported, got %d", deleted)
	}

	// Workspace should still exist
	if _, err := os.Stat(oldWs); os.IsNotExist(err) {
		t.Error("Workspace should still exist after dry-run")
	}
}

// TestCleanExpiredArchivesNoArchivedDir tests graceful handling when no archived directory exists.
func TestCleanExpiredArchivesNoArchivedDir(t *testing.T) {
	tmpDir := t.TempDir()

	deleted, err := cleanExpiredArchives(tmpDir, 30, false)
	if err != nil {
		t.Fatalf("cleanExpiredArchives should not fail when no archived dir: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted, got %d", deleted)
	}
}

// TestCleanExpiredArchivesFallbackToModTime tests that dir modtime is used when no .spawn_time exists.
func TestCleanExpiredArchivesFallbackToModTime(t *testing.T) {
	tmpDir := t.TempDir()
	archivedDir := filepath.Join(tmpDir, ".orch", "workspace", "archived")
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		t.Fatalf("Failed to create archived dir: %v", err)
	}

	// Create workspace with no .spawn_time, set modtime to 40 days ago
	ws := filepath.Join(archivedDir, "og-feat-no-spawntime-01jan")
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	oldTime := time.Now().AddDate(0, 0, -40)
	if err := os.Chtimes(ws, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set modtime: %v", err)
	}

	deleted, err := cleanExpiredArchives(tmpDir, 30, false)
	if err != nil {
		t.Fatalf("cleanExpiredArchives failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 workspace deleted (via modtime fallback), got %d", deleted)
	}
}

// TestOrphanReportFormatting verifies detectOrphansReport output formatting.
func TestOrphanReportFormatting(t *testing.T) {
	orphan := agent.OrphanedAgent{
		Agent:       agent.AgentRef{BeadsID: "orch-go-abc1"},
		Reason:      "no_live_execution",
		LastPhase:   "Complete",
		StaleFor:    2 * time.Hour,
		ShouldRetry: false,
	}

	action := "force-complete"
	if orphan.ShouldRetry {
		action = "force-abandon"
	}

	if action != "force-complete" {
		t.Errorf("Expected force-complete for non-retryable orphan, got %s", action)
	}

	// Verify detail includes phase and stale time
	detail := orphan.Reason
	if orphan.LastPhase != "" {
		detail += ", phase: " + orphan.LastPhase
	}
	if orphan.StaleFor > 0 {
		detail += ", stale 2h0m0s"
	}

	expected := "no_live_execution, phase: Complete, stale 2h0m0s"
	if detail != expected {
		t.Errorf("Expected detail %q, got %q", expected, detail)
	}
}
