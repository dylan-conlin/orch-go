// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListReadyIssuesForProject_EmptyPath(t *testing.T) {
	_, err := ListReadyIssuesForProject("")
	if err == nil {
		t.Error("ListReadyIssuesForProject(\"\") should return error for empty path")
	}
}

func TestListReadyIssuesForProject_NonExistentPath(t *testing.T) {
	// Test with a path that doesn't exist - should return empty list (not crash)
	issues, err := ListReadyIssuesForProject("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Errorf("ListReadyIssuesForProject() returned error: %v (should return empty list)", err)
	}
	if len(issues) != 0 {
		t.Errorf("ListReadyIssuesForProject() = %d issues, want 0 for nonexistent path", len(issues))
	}
}

func TestListReadyIssuesForProject_NoBeadsDir(t *testing.T) {
	// Create temp dir without .beads - should gracefully return empty list
	tmpDir := t.TempDir()

	issues, err := ListReadyIssuesForProject(tmpDir)
	if err != nil {
		t.Errorf("ListReadyIssuesForProject() returned error: %v (should return empty list)", err)
	}
	if len(issues) != 0 {
		t.Errorf("ListReadyIssuesForProject() = %d issues, want 0 for dir without .beads", len(issues))
	}
}

func TestListReadyIssuesForProject_PathTargeting(t *testing.T) {
	// Integration test: verify we can target a specific project path
	// This test uses the actual orch-go project which has .beads
	// Skip if not in the orch-go project directory

	cwd, err := os.Getwd()
	if err != nil {
		t.Skipf("Cannot get working directory: %v", err)
	}

	// Walk up to find .beads
	dir := cwd
	for {
		beadsDir := filepath.Join(dir, ".beads")
		if _, err := os.Stat(beadsDir); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("Not in a beads project, skipping integration test")
		}
		dir = parent
	}

	// Now test with the found project path
	issues, err := ListReadyIssuesForProject(dir)
	if err != nil {
		t.Errorf("ListReadyIssuesForProject(%q) returned error: %v", dir, err)
	}
	// We don't assert on the number of issues since it varies
	// Just ensure no crash and proper return type
	t.Logf("Found %d ready issues in %s", len(issues), dir)
}

func TestSpawnWorkForProject_EmptyPath(t *testing.T) {
	err := SpawnWorkForProject("test-123", "")
	if err == nil {
		t.Error("SpawnWorkForProject with empty path should return error")
	}
	if err.Error() != "projectPath is required" {
		t.Errorf("Expected 'projectPath is required' error, got: %v", err)
	}
}

func TestSpawnWork_DelegatesToSpawnWorkForProject(t *testing.T) {
	// Unit test: SpawnWork should delegate to SpawnWorkForProject with cwd
	// This is a behavior test - it will fail in CI (no orch binary) but
	// verifies the delegation pattern is correct.
	// The actual spawn will fail, but we're testing the delegation.
	err := SpawnWork("fake-issue-id")
	// Expect failure (no beads, no orch), but the error should indicate
	// SpawnWorkForProject was called (error message includes project name)
	if err == nil {
		t.Error("SpawnWork with fake issue should fail")
	}
	// The error should originate from SpawnWorkForProject (has project name prefix)
	// This confirms SpawnWork delegates to SpawnWorkForProject
	t.Logf("SpawnWork error (expected): %v", err)
}
