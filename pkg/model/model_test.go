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
		{"flash", ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}},
		{"flash-2.5", ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}},
		{"flash3", ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}},
		{"FLASH3", ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}},
		{"flash-3", ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}},
		{"pro", ModelSpec{Provider: "google", ModelID: "gemini-2.5-pro"}},

		// OpenAI aliases
		{"gpt", ModelSpec{Provider: "openai", ModelID: "gpt-5.2"}},
		{"GPT", ModelSpec{Provider: "openai", ModelID: "gpt-5.2"}},
		{"gpt5", ModelSpec{Provider: "openai", ModelID: "gpt-5.2"}},
		{"gpt-5", ModelSpec{Provider: "openai", ModelID: "gpt-5.2"}},
		{"gpt5-mini", ModelSpec{Provider: "openai", ModelID: "gpt-5-mini-20251130"}},
		{"gpt4o", ModelSpec{Provider: "openai", ModelID: "gpt-4o"}},
		{"gpt-4o", ModelSpec{Provider: "openai", ModelID: "gpt-4o"}},
		{"GPT-4o", ModelSpec{Provider: "openai", ModelID: "gpt-4o"}},
		{"gpt-mini", ModelSpec{Provider: "openai", ModelID: "gpt-4o-mini"}},
		{"gpt4o-mini", ModelSpec{Provider: "openai", ModelID: "gpt-4o-mini"}},
		{"gpt-4o-mini", ModelSpec{Provider: "openai", ModelID: "gpt-4o-mini"}},

		// DeepSeek aliases
		{"deepseek-v3", ModelSpec{Provider: "deepseek", ModelID: "deepseek-v3.2"}},
		{"reasoning", ModelSpec{Provider: "deepseek", ModelID: "deepseek-r1"}},

		// Alibaba (Qwen) aliases
		{"qwen", ModelSpec{Provider: "alibaba", ModelID: "qwen3-max"}},
		{"QWEN", ModelSpec{Provider: "alibaba", ModelID: "qwen3-max"}},
		{"qwen-max", ModelSpec{Provider: "alibaba", ModelID: "qwen3-max"}},
		{"qwen3", ModelSpec{Provider: "alibaba", ModelID: "qwen3-max"}},
		{"qwen3-max", ModelSpec{Provider: "alibaba", ModelID: "qwen3-max"}},
		{"qwen-thinking", ModelSpec{Provider: "alibaba", ModelID: "qwen3-max-2026-01-23"}},
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

		// GPT models default to openai
		{"gpt-5-20251215", ModelSpec{Provider: "openai", ModelID: "gpt-5-20251215"}},

		// DeepSeek models default to deepseek
		{"deepseek-v3.2", ModelSpec{Provider: "deepseek", ModelID: "deepseek-v3.2"}},

		// Qwen models default to alibaba
		{"qwen3-max-2026-01-23", ModelSpec{Provider: "alibaba", ModelID: "qwen3-max-2026-01-23"}},
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
