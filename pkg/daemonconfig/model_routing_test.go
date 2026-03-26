package daemonconfig

import "testing"

func TestModelRoutingConfig_Resolve_ComboFirst(t *testing.T) {
	cfg := &ModelRoutingConfig{
		Default: "sonnet",
		Skills: map[string]string{
			"architect": "opus",
		},
		Models: map[string]string{
			"gpt-5.4": "opus",
		},
		Combos: map[string]string{
			"gpt-5.4+feature-impl": "gpt-5.4",
		},
	}

	// Combo has highest precedence
	result := cfg.Resolve("feature-impl", "gpt-5.4")
	if result.EffectiveModel != "gpt-5.4" {
		t.Errorf("combo: got %q, want gpt-5.4", result.EffectiveModel)
	}
	if result.Source != "combo" {
		t.Errorf("combo source: got %q, want combo", result.Source)
	}
}

func TestModelRoutingConfig_Resolve_SkillOverModel(t *testing.T) {
	cfg := &ModelRoutingConfig{
		Skills: map[string]string{
			"architect": "opus",
		},
		Models: map[string]string{
			"gpt-5.4": "sonnet",
		},
	}

	// Skill takes precedence over model
	result := cfg.Resolve("architect", "gpt-5.4")
	if result.EffectiveModel != "opus" {
		t.Errorf("skill: got %q, want opus", result.EffectiveModel)
	}
	if result.Source != "skill" {
		t.Errorf("skill source: got %q, want skill", result.Source)
	}
}

func TestModelRoutingConfig_Resolve_ModelOverDefault(t *testing.T) {
	cfg := &ModelRoutingConfig{
		Default: "sonnet",
		Models: map[string]string{
			"gpt-5.4": "opus",
		},
	}

	result := cfg.Resolve("feature-impl", "gpt-5.4")
	if result.EffectiveModel != "opus" {
		t.Errorf("model: got %q, want opus", result.EffectiveModel)
	}
	if result.Source != "model" {
		t.Errorf("model source: got %q, want model", result.Source)
	}
}

func TestModelRoutingConfig_Resolve_Default(t *testing.T) {
	cfg := &ModelRoutingConfig{
		Default: "sonnet",
	}

	result := cfg.Resolve("feature-impl", "")
	if result.EffectiveModel != "sonnet" {
		t.Errorf("default: got %q, want sonnet", result.EffectiveModel)
	}
	if result.Source != "default" {
		t.Errorf("default source: got %q, want default", result.Source)
	}
}

func TestModelRoutingConfig_Resolve_NoConfig(t *testing.T) {
	cfg := &ModelRoutingConfig{}

	result := cfg.Resolve("feature-impl", "opus")
	if result.EffectiveModel != "opus" {
		t.Errorf("none: got %q, want opus (passthrough)", result.EffectiveModel)
	}
	if result.Source != "none" {
		t.Errorf("none source: got %q, want none", result.Source)
	}
}

func TestModelRoutingConfig_Resolve_NilConfig(t *testing.T) {
	var cfg *ModelRoutingConfig
	result := cfg.Resolve("feature-impl", "opus")
	if result.EffectiveModel != "opus" {
		t.Errorf("nil: got %q, want opus (passthrough)", result.EffectiveModel)
	}
	if result.Source != "none" {
		t.Errorf("nil source: got %q, want none", result.Source)
	}
}

func TestModelRoutingConfig_Resolve_ReasonIncludesChange(t *testing.T) {
	cfg := &ModelRoutingConfig{
		Skills: map[string]string{
			"architect": "opus",
		},
	}

	result := cfg.Resolve("architect", "sonnet")
	if result.Reason == "" {
		t.Error("expected non-empty reason for model change")
	}
	// Should mention the change
	if result.BaseModel != "sonnet" {
		t.Errorf("base model: got %q, want sonnet", result.BaseModel)
	}
}

func TestModelRoutingConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  *ModelRoutingConfig
		want bool
	}{
		{"nil", nil, false},
		{"empty", &ModelRoutingConfig{}, false},
		{"default only", &ModelRoutingConfig{Default: "opus"}, true},
		{"skills only", &ModelRoutingConfig{Skills: map[string]string{"a": "b"}}, true},
		{"models only", &ModelRoutingConfig{Models: map[string]string{"a": "b"}}, true},
		{"combos only", &ModelRoutingConfig{Combos: map[string]string{"a+b": "c"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}
