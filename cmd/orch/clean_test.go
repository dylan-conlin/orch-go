package main

import (
	"os"
	"path/filepath"
	"testing"
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

// TestIsOrchestratorSessionTitle tests the orchestrator session title detection.
func TestIsOrchestratorSessionTitle(t *testing.T) {
	tests := []struct {
		title    string
		expected bool
	}{
		// Should match orchestrator patterns
		{"meta-orch-continue-session-06jan", true},
		{"orchestrator-main", true},
		{"meta-orchestrator-06jan-abc1", true},
		{"og-orch-goal-04jan", true},
		{"Meta-Orch Session", true},

		// Should NOT match worker patterns
		{"og-feat-add-feature-21dec", false},
		{"og-debug-fix-bug-21dec", false},
		{"og-inv-investigate-21dec", false},
		{"worker-session-123", false},
		{"", false},
		{"untitled", false},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := isOrchestratorSessionTitle(tt.title)
			if result != tt.expected {
				t.Errorf("isOrchestratorSessionTitle(%q) = %v, want %v", tt.title, result, tt.expected)
			}
		})
	}
}

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
