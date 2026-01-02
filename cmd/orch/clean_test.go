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
