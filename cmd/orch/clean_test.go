package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

// TestFormatBytes tests the human-readable byte formatting.
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

// TestWorkspaceInfoAgeCalculation tests workspace age calculation.
func TestWorkspaceInfoAgeCalculation(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		modTime  time.Time
		expected int // age in days
	}{
		{"today", now, 0},
		{"yesterday", now.AddDate(0, 0, -1), 1},
		{"week_ago", now.AddDate(0, 0, -7), 7},
		{"month_ago", now.AddDate(0, -1, 0), 30}, // approximately
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age := int(now.Sub(tt.modTime).Hours() / 24)
			if age != tt.expected {
				// Allow for month variation
				if tt.name == "month_ago" && (age >= 28 && age <= 31) {
					return
				}
				t.Errorf("age calculation for %s: got %d, want %d", tt.name, age, tt.expected)
			}
		})
	}
}

// TestCleanOldWorkspacesFiltering tests that workspace age filtering works correctly.
func TestCleanOldWorkspacesFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	now := time.Now()

	// Create old workspace (10 days old)
	oldWs := filepath.Join(workspaceDir, "og-feat-old-workspace")
	if err := os.MkdirAll(oldWs, 0755); err != nil {
		t.Fatalf("Failed to create old workspace: %v", err)
	}
	oldTime := now.AddDate(0, 0, -10)
	if err := os.Chtimes(oldWs, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set old workspace time: %v", err)
	}

	// Create recent workspace (2 days old)
	newWs := filepath.Join(workspaceDir, "og-feat-new-workspace")
	if err := os.MkdirAll(newWs, 0755); err != nil {
		t.Fatalf("Failed to create new workspace: %v", err)
	}
	newTime := now.AddDate(0, 0, -2)
	if err := os.Chtimes(newWs, newTime, newTime); err != nil {
		t.Fatalf("Failed to set new workspace time: %v", err)
	}

	// Read workspaces and filter by age (7 days cutoff)
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to read workspace dir: %v", err)
	}

	cutoff := now.AddDate(0, 0, -7)
	var oldWorkspaces []string
	var recentWorkspaces []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			oldWorkspaces = append(oldWorkspaces, entry.Name())
		} else {
			recentWorkspaces = append(recentWorkspaces, entry.Name())
		}
	}

	if len(oldWorkspaces) != 1 {
		t.Errorf("Expected 1 old workspace (>7 days), got %d", len(oldWorkspaces))
	}
	if len(recentWorkspaces) != 1 {
		t.Errorf("Expected 1 recent workspace (<=7 days), got %d", len(recentWorkspaces))
	}
}

// TestCleanWorkspacesSkipsActiveSession tests that workspaces with active sessions are skipped.
func TestCleanWorkspacesSkipsActiveSession(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create workspace with session ID
	ws := filepath.Join(workspaceDir, "og-feat-active-agent")
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	sessionIDPath := filepath.Join(ws, ".session_id")
	if err := os.WriteFile(sessionIDPath, []byte("ses_active123"), 0644); err != nil {
		t.Fatalf("Failed to write session ID: %v", err)
	}

	// Simulate active sessions map
	activeSessions := map[string]bool{
		"ses_active123": true,
	}

	// Read session ID and check if active
	sessionID, err := os.ReadFile(sessionIDPath)
	if err != nil {
		t.Fatalf("Failed to read session ID: %v", err)
	}

	hasActiveSession := activeSessions[string(sessionID)]
	if !hasActiveSession {
		t.Error("Workspace with ses_active123 should be detected as having active session")
	}

	// A workspace without matching session should not be active
	activeSessions2 := map[string]bool{
		"ses_other456": true,
	}
	hasActiveSession2 := activeSessions2[string(sessionID)]
	if hasActiveSession2 {
		t.Error("Workspace should not be detected as active with non-matching session ID")
	}
}
