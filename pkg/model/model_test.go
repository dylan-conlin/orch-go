package model

import "testing"

func TestResolve_Empty(t *testing.T) {
	result := Resolve("")
	if result != DefaultModel {
		t.Errorf("Expected DefaultModel, got %v", result)
	}
}

func TestResolve_Aliases(t *testing.T) {
	tests := []struct {
		input    string
		expected ModelSpec
	}{
		// Anthropic aliases
		{"opus", ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}},
		{"Opus", ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}},
		{"OPUS", ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}},
		{"sonnet", ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}},
		{"haiku", ModelSpec{Provider: "anthropic", ModelID: "claude-haiku-4-5-20251001"}},
		{"opus-4.5", ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}},

		// Google aliases
		{"flash", ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}},
		{"flash-2.5", ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}},
		{"flash3", ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}},
		{"FLASH3", ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}},
		{"flash-3", ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}},
		{"flash-3.0", ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}},
		{"pro", ModelSpec{Provider: "google", ModelID: "gemini-2.5-pro"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Resolve(tt.input)
			if result != tt.expected {
				t.Errorf("Resolve(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolve_ProviderModelFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected ModelSpec
	}{
		{"anthropic/claude-opus-4-5-20251101", ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}},
		{"google/gemini-2.5-flash", ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}},
		{"openai/gpt-4o", ModelSpec{Provider: "openai", ModelID: "gpt-4o"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Resolve(tt.input)
			if result != tt.expected {
				t.Errorf("Resolve(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolve_ModelIDOnly(t *testing.T) {
	tests := []struct {
		input    string
		expected ModelSpec
	}{
		// Claude models default to anthropic
		{"claude-opus-4-5-20251101", ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}},
		{"claude-sonnet-4-5-20250929", ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}},

		// Gemini models default to google
		{"gemini-2.5-flash", ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}},
		{"gemini-3-flash-preview", ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Resolve(tt.input)
			if result != tt.expected {
				t.Errorf("Resolve(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestModelSpec_Format(t *testing.T) {
	spec := ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}
	if spec.Format() != "anthropic/claude-opus-4-5-20251101" {
		t.Errorf("Format() = %q, want %q", spec.Format(), "anthropic/claude-opus-4-5-20251101")
	}
}
