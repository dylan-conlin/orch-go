package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckWorkspaceSynthesisForCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	t.Run("workspace with SYNTHESIS.md", func(t *testing.T) {
		workspaceName := "og-feat-test-25dec"
		workspacePath := filepath.Join(workspaceDir, workspaceName)
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("Failed to create workspace dir: %v", err)
		}

		synthesisContent := "# Session Synthesis\nTLDR: Test completed successfully\n"
		if err := os.WriteFile(filepath.Join(workspacePath, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
			t.Fatalf("Failed to create SYNTHESIS.md: %v", err)
		}

		synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
		if _, err := os.Stat(synthesisPath); err != nil {
			t.Errorf("Expected SYNTHESIS.md to exist, got error: %v", err)
		}
	})

	t.Run("workspace without SYNTHESIS.md", func(t *testing.T) {
		workspaceName := "og-feat-no-synthesis-25dec"
		workspacePath := filepath.Join(workspaceDir, workspaceName)
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("Failed to create workspace dir: %v", err)
		}

		spawnContextContent := "TASK: Test task\n"
		if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContextContent), 0644); err != nil {
			t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
		}

		synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
		if _, err := os.Stat(synthesisPath); err == nil {
			t.Errorf("Expected SYNTHESIS.md to NOT exist")
		}
	})
}

func TestCheckWorkspaceSynthesis(t *testing.T) {
	tmpDir := t.TempDir()

	exists := checkWorkspaceSynthesis(tmpDir)
	if exists {
		t.Error("Expected checkWorkspaceSynthesis to return false for empty workspace")
	}

	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")
	if err := os.WriteFile(synthesisPath, []byte("# Synthesis\nTLDR: Test\n"), 0644); err != nil {
		t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
	}

	exists = checkWorkspaceSynthesis(tmpDir)
	if !exists {
		t.Error("Expected checkWorkspaceSynthesis to return true when SYNTHESIS.md exists")
	}

	if err := os.WriteFile(synthesisPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty SYNTHESIS.md: %v", err)
	}

	exists = checkWorkspaceSynthesis(tmpDir)
	if exists {
		t.Error("Expected checkWorkspaceSynthesis to return false for empty SYNTHESIS.md")
	}
}

func TestDetermineAgentStatus(t *testing.T) {
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	tests := []struct {
		name           string
		issueClosed    bool
		phaseComplete  bool
		hasSynthesis   bool
		sessionStatus  string
		expectedStatus string
	}{
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
		{
			name:           "phase_complete_dead_awaiting_cleanup",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   false,
			sessionStatus:  "dead",
			expectedStatus: "awaiting-cleanup",
		},
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
		{
			name:           "synthesis_dead_awaiting_cleanup",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   true,
			sessionStatus:  "dead",
			expectedStatus: "awaiting-cleanup",
		},
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

func TestDetermineAgentStatusEmptyWorkspace(t *testing.T) {
	result := determineAgentStatus(false, false, "", "idle")
	if result != "idle" {
		t.Errorf("Expected 'idle' for empty workspace, got %q", result)
	}
}

func TestDetermineAgentStatusNonExistentWorkspace(t *testing.T) {
	result := determineAgentStatus(false, false, "/nonexistent/path/workspace", "active")
	if result != "active" {
		t.Errorf("Expected 'active' for non-existent workspace, got %q", result)
	}
}
