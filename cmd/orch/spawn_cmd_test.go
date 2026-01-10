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
			name:            "no flags defaults to opencode",
			modelFlag:       "",
			opusFlag:        false,
			expectedBackend: "opencode",
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
			backend := "opencode"

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
