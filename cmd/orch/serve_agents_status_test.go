package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckWorkspaceSynthesisForCompletion(t *testing.T) {
	// Create a temporary project directory with workspace
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Test 1: Workspace with SYNTHESIS.md should indicate completion
	t.Run("workspace with SYNTHESIS.md", func(t *testing.T) {
		workspaceName := "og-feat-test-25dec"
		workspacePath := filepath.Join(workspaceDir, workspaceName)
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("Failed to create workspace dir: %v", err)
		}

		// Create SYNTHESIS.md
		synthesisContent := `# Session Synthesis
TLDR: Test completed successfully
`
		if err := os.WriteFile(filepath.Join(workspacePath, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
			t.Fatalf("Failed to create SYNTHESIS.md: %v", err)
		}

		// Check if synthesis exists
		synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
		if _, err := os.Stat(synthesisPath); err != nil {
			t.Errorf("Expected SYNTHESIS.md to exist, got error: %v", err)
		}
	})

	// Test 2: Workspace without SYNTHESIS.md should not indicate completion
	t.Run("workspace without SYNTHESIS.md", func(t *testing.T) {
		workspaceName := "og-feat-no-synthesis-25dec"
		workspacePath := filepath.Join(workspaceDir, workspaceName)
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("Failed to create workspace dir: %v", err)
		}

		// Create only SPAWN_CONTEXT.md (no SYNTHESIS.md)
		spawnContextContent := `TASK: Test task
`
		if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContextContent), 0644); err != nil {
			t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
		}

		// Check that synthesis does NOT exist
		synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
		if _, err := os.Stat(synthesisPath); err == nil {
			t.Errorf("Expected SYNTHESIS.md to NOT exist")
		}
	})
}

func TestCheckWorkspaceSynthesis(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()

	// Test case 1: No SYNTHESIS.md
	exists := checkWorkspaceSynthesis(tmpDir)
	if exists {
		t.Error("Expected checkWorkspaceSynthesis to return false for empty workspace")
	}

	// Test case 2: With SYNTHESIS.md
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")
	if err := os.WriteFile(synthesisPath, []byte("# Synthesis\nTLDR: Test\n"), 0644); err != nil {
		t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
	}

	exists = checkWorkspaceSynthesis(tmpDir)
	if !exists {
		t.Error("Expected checkWorkspaceSynthesis to return true when SYNTHESIS.md exists")
	}

	// Test case 3: With empty SYNTHESIS.md
	if err := os.WriteFile(synthesisPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty SYNTHESIS.md: %v", err)
	}

	exists = checkWorkspaceSynthesis(tmpDir)
	if exists {
		t.Error("Expected checkWorkspaceSynthesis to return false for empty SYNTHESIS.md")
	}
}

// TestDetermineAgentStatus tests the Priority Cascade model for agent status determination.
// Priority order:
//  1. Beads issue closed -> "completed"
//  2. Phase: Complete reported AND dead -> "awaiting-cleanup"
//  3. Phase: Complete reported -> "completed"
//  4. SYNTHESIS.md exists AND dead -> "awaiting-cleanup"
//  5. SYNTHESIS.md exists -> "completed"
//  6. Session activity -> "active", "idle", or "dead"
func TestDetermineAgentStatus(t *testing.T) {
	// Create a temporary workspace with SYNTHESIS.md for testing
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	tests := []struct {
		name           string
		issueClosed    bool
		phaseComplete  bool
		hasSynthesis   bool
		sessionStatus  string // "active", "idle", or "dead" based on activity
		expectedStatus string
	}{
		// Priority 1: Beads closed overrides everything
		{
			name:           "beads_closed_overrides_all",
			issueClosed:    true,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "active",
			expectedStatus: "completed",
		},
		{
			name:           "beads_closed_even_if_idle",
			issueClosed:    true,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "idle",
			expectedStatus: "completed",
		},
		{
			name:           "beads_closed_even_if_dead",
			issueClosed:    true,
			phaseComplete:  true,
			hasSynthesis:   true,
			sessionStatus:  "dead",
			expectedStatus: "completed",
		},
		// Priority 2: Phase: Complete + dead -> awaiting-cleanup
		{
			name:           "phase_complete_dead_awaiting_cleanup",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   false,
			sessionStatus:  "dead",
			expectedStatus: "awaiting-cleanup",
		},
		// Priority 3: Phase: Complete + active/idle -> completed
		{
			name:           "phase_complete_overrides_session",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   false,
			sessionStatus:  "active",
			expectedStatus: "completed",
		},
		{
			name:           "phase_complete_overrides_idle",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   false,
			sessionStatus:  "idle",
			expectedStatus: "completed",
		},
		// Priority 4: SYNTHESIS.md + dead -> awaiting-cleanup
		{
			name:           "synthesis_dead_awaiting_cleanup",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   true,
			sessionStatus:  "dead",
			expectedStatus: "awaiting-cleanup",
		},
		// Priority 5: SYNTHESIS.md + active/idle -> completed
		{
			name:           "synthesis_overrides_session",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   true,
			sessionStatus:  "active",
			expectedStatus: "completed",
		},
		{
			name:           "synthesis_overrides_idle",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   true,
			sessionStatus:  "idle",
			expectedStatus: "completed",
		},
		// Priority 6: Session activity is the fallback
		{
			name:           "active_session",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "active",
			expectedStatus: "active",
		},
		{
			name:           "idle_session",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "idle",
			expectedStatus: "idle",
		},
		{
			name:           "dead_session_no_completion",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "dead",
			expectedStatus: "dead",
		},
		// Combined scenarios - higher priority wins
		{
			name:           "beads_closed_with_phase_complete",
			issueClosed:    true,
			phaseComplete:  true,
			hasSynthesis:   true,
			sessionStatus:  "idle",
			expectedStatus: "completed",
		},
		{
			name:           "phase_complete_with_synthesis",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   true,
			sessionStatus:  "active",
			expectedStatus: "completed",
		},
		{
			name:           "phase_complete_with_synthesis_dead",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   true,
			sessionStatus:  "dead",
			expectedStatus: "awaiting-cleanup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up or remove SYNTHESIS.md based on test case
			if tt.hasSynthesis {
				if err := os.WriteFile(synthesisPath, []byte("# Synthesis\nTLDR: Test"), 0644); err != nil {
					t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
				}
			} else {
				os.Remove(synthesisPath)
			}

			result := determineAgentStatus(tt.issueClosed, tt.phaseComplete, tmpDir, tt.sessionStatus)

			if result != tt.expectedStatus {
				t.Errorf("determineAgentStatus() = %q, want %q", result, tt.expectedStatus)
			}
		})
	}
}

// TestDetermineAgentStatusEmptyWorkspace tests that empty workspace path is handled correctly.
func TestDetermineAgentStatusEmptyWorkspace(t *testing.T) {
	// With empty workspace, SYNTHESIS.md check should be skipped
	result := determineAgentStatus(false, false, "", "idle")
	if result != "idle" {
		t.Errorf("Expected 'idle' for empty workspace, got %q", result)
	}
}

// TestDetermineAgentStatusNonExistentWorkspace tests non-existent workspace path.
func TestDetermineAgentStatusNonExistentWorkspace(t *testing.T) {
	// With non-existent workspace, SYNTHESIS.md check should return false
	result := determineAgentStatus(false, false, "/nonexistent/path/workspace", "active")
	if result != "active" {
		t.Errorf("Expected 'active' for non-existent workspace, got %q", result)
	}
}
