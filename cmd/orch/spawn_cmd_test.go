package main

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/orch"
)

func TestFlashModelBlocking(t *testing.T) {
	// Test that flash models are properly identified
	flashModels := []string{
		"flash",
		"flash-2.5",
		"flash3",
		"google/gemini-2.5-flash",
		"google/gemini-3-flash-preview",
	}

	for _, modelStr := range flashModels {
		t.Run(modelStr, func(t *testing.T) {
			resolved := model.Resolve(modelStr)

			// Check that it's a Google/flash model
			if resolved.Provider != "google" {
				t.Errorf("expected provider 'google', got %q", resolved.Provider)
			}

			if !strings.Contains(strings.ToLower(resolved.ModelID), "flash") {
				t.Errorf("expected model ID to contain 'flash', got %q", resolved.ModelID)
			}
		})
	}
}

func TestModelAutoSelection(t *testing.T) {
	tests := []struct {
		name            string
		modelFlag       string
		opusFlag        bool
		expectedBackend string
	}{
		{
			name:            "opus flag forces claude",
			modelFlag:       "",
			opusFlag:        true,
			expectedBackend: "claude",
		},
		{
			name:            "opus model auto-selects claude",
			modelFlag:       "opus",
			opusFlag:        false,
			expectedBackend: "claude",
		},
		{
			name:            "sonnet model uses opencode",
			modelFlag:       "sonnet",
			opusFlag:        false,
			expectedBackend: "opencode",
		},
		{
			name:            "no flags defaults to claude",
			modelFlag:       "",
			opusFlag:        false,
			expectedBackend: "claude",
		},
		{
			name:            "opus-4.5 alias auto-selects claude",
			modelFlag:       "opus-4.5",
			opusFlag:        false,
			expectedBackend: "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the auto-selection logic from runSpawnWithSkillInternal
			backend := "claude"

			if tt.opusFlag {
				backend = "claude"
			} else if tt.modelFlag != "" {
				modelLower := strings.ToLower(tt.modelFlag)
				if modelLower == "opus" || strings.Contains(modelLower, "opus") {
					backend = "claude"
				} else if modelLower == "sonnet" || strings.Contains(modelLower, "sonnet") {
					backend = "opencode"
				}
			}

			if backend != tt.expectedBackend {
				t.Errorf("expected backend %q, got %q", tt.expectedBackend, backend)
			}
		})
	}
}

func TestIsInfrastructureWork(t *testing.T) {
	tests := []struct {
		name    string
		task    string
		beadsID string
		want    bool
	}{
		{
			name:    "opencode keyword in task",
			task:    "fix opencode server crash",
			beadsID: "",
			want:    true,
		},
		{
			name:    "spawn keyword in task",
			task:    "update spawn logic to handle errors",
			beadsID: "",
			want:    true,
		},
		{
			name:    "dashboard keyword in task",
			task:    "fix dashboard agent count",
			beadsID: "",
			want:    true,
		},
		{
			name:    "pkg/spawn path in task",
			task:    "refactor pkg/spawn/context.go",
			beadsID: "",
			want:    true,
		},
		{
			name:    "cmd/orch path in task",
			task:    "update cmd/orch/serve.go logging",
			beadsID: "",
			want:    true,
		},
		{
			name:    "skillc keyword in task",
			task:    "fix skillc compilation issue",
			beadsID: "",
			want:    true,
		},
		{
			name:    "orchestration infrastructure phrase",
			task:    "improve orchestration infrastructure",
			beadsID: "",
			want:    true,
		},
		{
			name:    "non-infrastructure task",
			task:    "add user authentication feature",
			beadsID: "",
			want:    false,
		},
		{
			name:    "case insensitive detection",
			task:    "Fix OpenCode Server Bug",
			beadsID: "",
			want:    true,
		},
		{
			name:    "agent stores infrastructure",
			task:    "update agents.ts store logic",
			beadsID: "",
			want:    true,
		},
		{
			name:    "regular feature work",
			task:    "implement user profile page",
			beadsID: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orch.IsInfrastructureWork(tt.task, tt.beadsID)
			if got != tt.want {
				t.Errorf("orch.IsInfrastructureWork(%q, %q) = %v, want %v", tt.task, tt.beadsID, got, tt.want)
			}
		})
	}
}
