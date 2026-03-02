package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
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
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"},
			expectWarning: false,
		},
		{
			name:          "invalid: opencode + opus",
			backend:       "opencode",
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"},
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

func TestLoadUserConfigWithMetaAndWarningMalformedConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, ".orch")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("backend: ["), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	userCfg, userMeta, warning := loadUserConfigWithMetaAndWarning()
	if warning == "" {
		t.Fatalf("expected warning for malformed config")
	}
	if !strings.Contains(warning, "config.yaml") {
		t.Fatalf("warning missing config path: %q", warning)
	}
	if !strings.Contains(warning, "backend/default_model") {
		t.Fatalf("warning missing preference hint: %q", warning)
	}
	if userCfg != nil || userMeta != nil {
		t.Fatalf("expected nil config and meta on error")
	}
}

func TestLoadUserConfigAndWarningMalformedConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, ".orch")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("backend: ["), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	userCfg, warning := loadUserConfigAndWarning()
	if warning == "" {
		t.Fatalf("expected warning for malformed config")
	}
	if userCfg != nil {
		t.Fatalf("expected nil config on error")
	}
}

// --- hotspotFilesFromResult tests ---

func TestHotspotFilesFromResult_NilResult(t *testing.T) {
	files := hotspotFilesFromResult(nil)
	if files != nil {
		t.Errorf("expected nil for nil result, got %v", files)
	}
}

func TestHotspotFilesFromResult_EmptyMatchedFiles(t *testing.T) {
	result := &gates.HotspotResult{
		HasHotspots:   true,
		MatchedFiles:  nil,
	}
	files := hotspotFilesFromResult(result)
	if files != nil {
		t.Errorf("expected nil for nil MatchedFiles, got %v", files)
	}
}

func TestHotspotFilesFromResult_WithMatchedFiles(t *testing.T) {
	result := &gates.HotspotResult{
		HasHotspots:  true,
		MatchedFiles: []string{"cmd/orch/spawn_cmd.go", "pkg/daemon/daemon.go"},
	}
	files := hotspotFilesFromResult(result)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0] != "cmd/orch/spawn_cmd.go" {
		t.Errorf("files[0] = %q, want 'cmd/orch/spawn_cmd.go'", files[0])
	}
	if files[1] != "pkg/daemon/daemon.go" {
		t.Errorf("files[1] = %q, want 'pkg/daemon/daemon.go'", files[1])
	}
}

func TestHotspotFilesFromResult_NoHotspotsStillReturnsFiles(t *testing.T) {
	// Even if HasHotspots is false, we return MatchedFiles as-is
	result := &gates.HotspotResult{
		HasHotspots:  false,
		MatchedFiles: []string{"some/file.go"},
	}
	files := hotspotFilesFromResult(result)
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}
