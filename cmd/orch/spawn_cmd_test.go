package main

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/model"
)

func TestValidateModeModelCombo(t *testing.T) {
	tests := []struct {
		name          string
		backend       string
		modelSpec     model.ModelSpec
		expectWarning bool
		warningText   string
	}{
		{
			name:          "valid: opencode + sonnet",
			backend:       "opencode",
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"},
			expectWarning: false,
		},
		{
			name:          "valid: claude + opus",
			backend:       "claude",
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-6"},
			expectWarning: false,
		},
		{
			name:          "invalid: opencode + opus",
			backend:       "opencode",
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-6"},
			expectWarning: true,
			warningText:   "opencode backend with opus model may fail",
		},
		{
			name:          "valid: claude + sonnet (non-optimal but works)",
			backend:       "claude",
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"},
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModeModelCombo(tt.backend, tt.modelSpec)

			if tt.expectWarning {
				if err == nil {
					t.Errorf("expected warning but got nil")
				} else if !strings.Contains(err.Error(), tt.warningText) {
					t.Errorf("expected warning containing %q, got %q", tt.warningText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no warning but got: %v", err)
				}
			}
		})
	}
}

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
			name:            "opus alias auto-selects claude",
			modelFlag:       "opus",
			opusFlag:        false,
			expectedBackend: "claude",
		},
		{
			name:            "opus-4.5 legacy alias auto-selects claude",
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

// TestIsCriticalInfrastructureWork tests the narrowed infrastructure detection.
// Only CRITICAL infrastructure (server lifecycle) should trigger, not general orch work.
func TestIsCriticalInfrastructureWork(t *testing.T) {
	tests := []struct {
		name    string
		task    string
		beadsID string
		want    bool
	}{
		// CRITICAL infrastructure - should trigger
		{
			name:    "opencode server keyword",
			task:    "fix opencode server crash",
			beadsID: "",
			want:    true, // matches "opencode server"
		},
		{
			name:    "serve.go in task",
			task:    "update cmd/orch/serve.go logging",
			beadsID: "",
			want:    true, // matches "serve.go"
		},
		{
			name:    "pkg/opencode path",
			task:    "refactor pkg/opencode/client.go",
			beadsID: "",
			want:    true, // matches "pkg/opencode"
		},
		{
			name:    "case insensitive opencode server",
			task:    "Fix OpenCode Server Bug",
			beadsID: "",
			want:    true, // matches "opencode server"
		},
		{
			name:    "server restart",
			task:    "implement server restart handling",
			beadsID: "",
			want:    true, // matches "server restart"
		},
		{
			name:    "opencode api work",
			task:    "update opencode api endpoints",
			beadsID: "",
			want:    true, // matches "opencode api"
		},

		// NON-CRITICAL - should NOT trigger (narrowed scope)
		{
			name:    "spawn logic (not critical)",
			task:    "update spawn logic to handle errors",
			beadsID: "",
			want:    false, // spawn logic doesn't restart server
		},
		{
			name:    "dashboard (not critical)",
			task:    "fix dashboard agent count",
			beadsID: "",
			want:    false, // dashboard is frontend, not server
		},
		{
			name:    "pkg/spawn (not critical)",
			task:    "refactor pkg/spawn/context.go",
			beadsID: "",
			want:    false, // spawn context doesn't restart server
		},
		{
			name:    "skillc (not critical)",
			task:    "fix skillc compilation issue",
			beadsID: "",
			want:    false, // skill compiler is separate tool
		},
		{
			name:    "orchestration infrastructure phrase (not critical)",
			task:    "improve orchestration infrastructure",
			beadsID: "",
			want:    false, // too generic to be critical
		},
		{
			name:    "agents.ts (not critical)",
			task:    "update agents.ts store logic",
			beadsID: "",
			want:    false, // frontend component
		},
		{
			name:    "non-infrastructure task",
			task:    "add user authentication feature",
			beadsID: "",
			want:    false,
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
			got := isCriticalInfrastructureWork(tt.task, tt.beadsID)
			if got != tt.want {
				t.Errorf("isCriticalInfrastructureWork(%q, %q) = %v, want %v", tt.task, tt.beadsID, got, tt.want)
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no ANSI codes",
			input: "Error: Session not found",
			want:  "Error: Session not found",
		},
		{
			name:  "red bold error from opencode",
			input: "\x1b[91m\x1b[1mError: \x1b[0mSession not found",
			want:  "Error: Session not found",
		},
		{
			name:  "various colors",
			input: "\x1b[32mGreen\x1b[0m \x1b[33mYellow\x1b[0m",
			want:  "Green Yellow",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only ANSI codes",
			input: "\x1b[0m\x1b[1m\x1b[91m",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
