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
		{"gpt", ModelSpec{Provider: "openai", ModelID: "gpt-4o"}},
		{"gpt4o", ModelSpec{Provider: "openai", ModelID: "gpt-4o"}},
		{"gpt-4o", ModelSpec{Provider: "openai", ModelID: "gpt-4o"}},
		{"GPT4O", ModelSpec{Provider: "openai", ModelID: "gpt-4o"}},
		{"gpt4o-mini", ModelSpec{Provider: "openai", ModelID: "gpt-4o-mini"}},
		{"gpt-4o-mini", ModelSpec{Provider: "openai", ModelID: "gpt-4o-mini"}},
		{"gpt-5", ModelSpec{Provider: "openai", ModelID: "gpt-5.2"}},
		{"gpt-5.1", ModelSpec{Provider: "openai", ModelID: "gpt-5.1"}},
		{"gpt-5.2", ModelSpec{Provider: "openai", ModelID: "gpt-5.2"}},
		{"gpt-5.4", ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}},
		{"gpt5-latest", ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}},
		{"gpt5-mini", ModelSpec{Provider: "openai", ModelID: "gpt-5.1-codex-mini"}},
		{"o3", ModelSpec{Provider: "openai", ModelID: "o3"}},

		// Codex aliases (GPT Pro OAuth path)
		{"codex", ModelSpec{Provider: "openai", ModelID: "gpt-5.2-codex"}},
		{"codex-mini", ModelSpec{Provider: "openai", ModelID: "gpt-5.1-codex-mini"}},
		{"codex-max", ModelSpec{Provider: "openai", ModelID: "gpt-5.1-codex-max"}},
		{"codex-latest", ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}},
		{"codex-5.1", ModelSpec{Provider: "openai", ModelID: "gpt-5.1-codex"}},
		{"codex-5.2", ModelSpec{Provider: "openai", ModelID: "gpt-5.2"}},
		{"codex-5.4", ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}},
		{"CODEX", ModelSpec{Provider: "openai", ModelID: "gpt-5.2-codex"}},

		// DeepSeek aliases
		{"deepseek", ModelSpec{Provider: "deepseek", ModelID: "deepseek-chat"}},
		{"reasoning", ModelSpec{Provider: "deepseek", ModelID: "deepseek-reasoner"}},
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

func TestResolveWithConfig_ConfigAliasOverride(t *testing.T) {
	configModels := map[string]string{
		"default": "openai/gpt-4o",
		"fast":    "google/gemini-2.5-flash",
	}

	// Config alias should resolve
	result := ResolveWithConfig("default", configModels)
	expected := ModelSpec{Provider: "openai", ModelID: "gpt-4o"}
	if result != expected {
		t.Errorf("ResolveWithConfig('default', config) = %v, want %v", result, expected)
	}

	// Config alias takes precedence over built-in
	result = ResolveWithConfig("fast", configModels)
	expected = ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}
	if result != expected {
		t.Errorf("ResolveWithConfig('fast', config) = %v, want %v", result, expected)
	}

	// Empty spec still returns DefaultModel
	result = ResolveWithConfig("", configModels)
	if result != DefaultModel {
		t.Errorf("ResolveWithConfig('', config) = %v, want DefaultModel %v", result, DefaultModel)
	}

	// Built-in alias still works when not overridden in config
	result = ResolveWithConfig("opus", configModels)
	expected = ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}
	if result != expected {
		t.Errorf("ResolveWithConfig('opus', config) = %v, want %v", result, expected)
	}
}

func TestModelSpec_IsAnthropicModel(t *testing.T) {
	tests := []struct {
		spec     ModelSpec
		expected bool
	}{
		{ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}, true},
		{ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}, true},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}, false},
		{ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}, false},
		{ModelSpec{Provider: "deepseek", ModelID: "deepseek-chat"}, false},
		{ModelSpec{Provider: "Anthropic", ModelID: "claude-opus-4-5-20251101"}, true},
		{ModelSpec{Provider: "", ModelID: "claude-opus-4-5-20251101"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.spec.Format(), func(t *testing.T) {
			if got := tt.spec.IsAnthropicModel(); got != tt.expected {
				t.Errorf("IsAnthropicModel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestModelSpec_IsOpenAI(t *testing.T) {
	tests := []struct {
		spec     ModelSpec
		expected bool
	}{
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}, true},
		{ModelSpec{Provider: "openai", ModelID: "gpt-4o"}, true},
		{ModelSpec{Provider: "openai", ModelID: "o3"}, true},
		{ModelSpec{Provider: "OpenAI", ModelID: "gpt-5.4"}, true},
		{ModelSpec{Provider: "OPENAI", ModelID: "gpt-4o"}, true},
		{ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}, false},
		{ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}, false},
		{ModelSpec{Provider: "deepseek", ModelID: "deepseek-chat"}, false},
		{ModelSpec{Provider: "", ModelID: "gpt-5.4"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.spec.Format(), func(t *testing.T) {
			if got := tt.spec.IsOpenAI(); got != tt.expected {
				t.Errorf("IsOpenAI() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestModelSpec_ProviderName(t *testing.T) {
	tests := []struct {
		spec     ModelSpec
		expected string
	}{
		{ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}, "Anthropic"},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}, "OpenAI"},
		{ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}, "Google"},
		{ModelSpec{Provider: "deepseek", ModelID: "deepseek-chat"}, "DeepSeek"},
		{ModelSpec{Provider: "Anthropic", ModelID: "claude-opus-4-5-20251101"}, "Anthropic"},
		{ModelSpec{Provider: "OPENAI", ModelID: "gpt-4o"}, "OpenAI"},
		{ModelSpec{Provider: "unknown-provider", ModelID: "some-model"}, "unknown-provider"},
		{ModelSpec{Provider: "", ModelID: "some-model"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.spec.Format(), func(t *testing.T) {
			if got := tt.spec.ProviderName(); got != tt.expected {
				t.Errorf("ProviderName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestModelSpec_ModelFamily(t *testing.T) {
	tests := []struct {
		spec     ModelSpec
		expected string
	}{
		// Claude family
		{ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}, "claude"},
		{ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}, "claude"},
		{ModelSpec{Provider: "anthropic", ModelID: "claude-haiku-4-5-20251001"}, "claude"},

		// Gemini family
		{ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}, "gemini"},
		{ModelSpec{Provider: "google", ModelID: "gemini-3-flash-preview"}, "gemini"},
		{ModelSpec{Provider: "google", ModelID: "gemini-2.5-pro"}, "gemini"},

		// GPT family (includes codex and o-series)
		{ModelSpec{Provider: "openai", ModelID: "gpt-4o"}, "gpt"},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.2"}, "gpt"},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}, "gpt"},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.1-codex-mini"}, "gpt"},
		{ModelSpec{Provider: "openai", ModelID: "o3"}, "gpt"},
		{ModelSpec{Provider: "openai", ModelID: "o3-mini"}, "gpt"},

		// DeepSeek family
		{ModelSpec{Provider: "deepseek", ModelID: "deepseek-chat"}, "deepseek"},
		{ModelSpec{Provider: "deepseek", ModelID: "deepseek-reasoner"}, "deepseek"},

		// Unknown model — falls back to provider
		{ModelSpec{Provider: "custom", ModelID: "some-model"}, "custom"},
		{ModelSpec{Provider: "", ModelID: "unknown"}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.spec.Format(), func(t *testing.T) {
			if got := tt.spec.ModelFamily(); got != tt.expected {
				t.Errorf("ModelFamily() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestModelSpec_String(t *testing.T) {
	tests := []struct {
		spec     ModelSpec
		expected string
	}{
		{ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}, "Anthropic: claude-opus-4-5-20251101"},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}, "OpenAI: gpt-5.4"},
		{ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}, "Google: gemini-2.5-flash"},
		{ModelSpec{Provider: "deepseek", ModelID: "deepseek-chat"}, "DeepSeek: deepseek-chat"},
		{ModelSpec{Provider: "unknown-provider", ModelID: "some-model"}, "unknown-provider: some-model"},
	}

	for _, tt := range tests {
		t.Run(tt.spec.Format(), func(t *testing.T) {
			if got := tt.spec.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestModelSpec_IsReasoningModel(t *testing.T) {
	tests := []struct {
		spec     ModelSpec
		expected bool
	}{
		// Reasoning models — should return true
		{ModelSpec{Provider: "openai", ModelID: "o3"}, true},
		{ModelSpec{Provider: "openai", ModelID: "o3-mini"}, true},
		{ModelSpec{Provider: "deepseek", ModelID: "deepseek-reasoner"}, true},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.2-codex"}, true},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.1-codex-mini"}, true},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.1-codex-max"}, true},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.1-codex"}, true},

		// Non-reasoning models — should return false
		{ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}, false},
		{ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"}, false},
		{ModelSpec{Provider: "openai", ModelID: "gpt-4o"}, false},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.2"}, false},
		{ModelSpec{Provider: "openai", ModelID: "gpt-5.4"}, false},
		{ModelSpec{Provider: "google", ModelID: "gemini-2.5-flash"}, false},
		{ModelSpec{Provider: "deepseek", ModelID: "deepseek-chat"}, false},
		{ModelSpec{Provider: "", ModelID: "o3"}, true},  // ModelID-based, not provider-based
	}

	for _, tt := range tests {
		t.Run(tt.spec.Format(), func(t *testing.T) {
			if got := tt.spec.IsReasoningModel(); got != tt.expected {
				t.Errorf("IsReasoningModel() = %v, want %v", got, tt.expected)
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
