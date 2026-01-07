package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateResumePrompt(t *testing.T) {
	tests := []struct {
		name          string
		workspaceName string
		projectDir    string
		beadsID       string
		wantContains  []string
	}{
		{
			name:          "generates prompt with spawn context path",
			workspaceName: "og-inv-test-20dec",
			projectDir:    "/Users/test/project",
			beadsID:       "proj-123",
			wantContains: []string{
				"SPAWN_CONTEXT.md",
				"/Users/test/project",
				"og-inv-test-20dec",
				"continue",
			},
		},
		{
			name:          "includes beads ID for progress tracking",
			workspaceName: "og-feat-add-feature-20dec",
			projectDir:    "/home/user/my-project",
			beadsID:       "myproj-456",
			wantContains: []string{
				"SPAWN_CONTEXT.md",
				"myproj-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateResumePrompt(tt.workspaceName, tt.projectDir, tt.beadsID)
			for _, want := range tt.wantContains {
				if !stringContains(got, want) {
					t.Errorf("GenerateResumePrompt() = %q, want to contain %q", got, want)
				}
			}
		})
	}
}

func TestGenerateOrchestratorResumePrompt(t *testing.T) {
	// Create a temp directory for test workspaces
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		workspaceName string
		contextFile   string // which context file to create
		wantContains  []string
	}{
		{
			name:          "detects META_ORCHESTRATOR_CONTEXT.md",
			workspaceName: "meta-orch-test",
			contextFile:   "META_ORCHESTRATOR_CONTEXT.md",
			wantContains: []string{
				"META_ORCHESTRATOR_CONTEXT.md",
				"meta-orch-test",
				"paused",
				"continue",
			},
		},
		{
			name:          "detects ORCHESTRATOR_CONTEXT.md",
			workspaceName: "orch-session-test",
			contextFile:   "ORCHESTRATOR_CONTEXT.md",
			wantContains: []string{
				"ORCHESTRATOR_CONTEXT.md",
				"orch-session-test",
			},
		},
		{
			name:          "falls back to SPAWN_CONTEXT.md",
			workspaceName: "worker-test",
			contextFile:   "SPAWN_CONTEXT.md",
			wantContains: []string{
				"SPAWN_CONTEXT.md",
				"worker-test",
			},
		},
		{
			name:          "defaults to SPAWN_CONTEXT.md if no context file exists",
			workspaceName: "empty-workspace",
			contextFile:   "", // no file created
			wantContains: []string{
				"SPAWN_CONTEXT.md",
				"empty-workspace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create workspace directory
			workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", tt.workspaceName)
			if err := os.MkdirAll(workspaceDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Create context file if specified
			if tt.contextFile != "" {
				contextPath := filepath.Join(workspaceDir, tt.contextFile)
				if err := os.WriteFile(contextPath, []byte("test content"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := GenerateOrchestratorResumePrompt(tt.workspaceName, tmpDir)
			for _, want := range tt.wantContains {
				if !stringContains(got, want) {
					t.Errorf("GenerateOrchestratorResumePrompt() = %q, want to contain %q", got, want)
				}
			}
		})
	}
}

func TestGenerateSessionResumePrompt(t *testing.T) {
	got := GenerateSessionResumePrompt()

	wantContains := []string{
		"paused",
		"Continue",
	}

	for _, want := range wantContains {
		if !stringContains(got, want) {
			t.Errorf("GenerateSessionResumePrompt() = %q, want to contain %q", got, want)
		}
	}
}

// stringContains checks if s contains substr.
func stringContains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
