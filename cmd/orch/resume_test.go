package main

import (
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
				if !containsString(got, want) {
					t.Errorf("GenerateResumePrompt() = %q, want to contain %q", got, want)
				}
			}
		})
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
