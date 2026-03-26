package daemonconfig

// ModelRoutingConfig holds model routing configuration with per-skill, per-model,
// and per-combo overrides. Resolution order: combo > skill > model > default.
// Mirrors ComplianceConfig's combo-first pattern.
//
// The "model" dimension is the base model that would be selected without routing
// (e.g., from project config default_model). The "skill" dimension is the inferred
// skill for the spawn. Combos use "baseModel+skill" keys for highest precedence.
type ModelRoutingConfig struct {
	// Default is the global model alias when no override matches.
	// Empty string means no daemon-level override (resolve pipeline decides).
	Default string

	// Skills maps skill names to model aliases.
	// e.g., "systematic-debugging" → "opus"
	Skills map[string]string

	// Models maps base model aliases to override model aliases.
	// e.g., "gpt-5.4" → "opus" (globally override gpt-5.4 to opus)
	Models map[string]string

	// Combos maps "baseModel+skill" keys to model aliases (highest precedence).
	// e.g., "gpt-5.4+feature-impl" → "gpt-5.4" (keep gpt-5.4 for feature-impl)
	Combos map[string]string
}

// ModelRouteResult captures a model routing decision for observability.
type ModelRouteResult struct {
	// EffectiveModel is the final model alias after routing.
	EffectiveModel string `json:"effective_model"`
	// Source describes which config layer determined the model.
	// One of: "combo", "skill", "model", "default", "hardcoded", "none"
	Source string `json:"source"`
	// BaseModel is the model that would have been used without routing config.
	BaseModel string `json:"base_model,omitempty"`
	// ConfigKey is the specific config key that matched (for debugging).
	// e.g., "gpt-5.4+feature-impl" for combo, "feature-impl" for skill.
	ConfigKey string `json:"config_key,omitempty"`
	// Reason is a human-readable explanation of the routing decision.
	Reason string `json:"reason"`
}

// Resolve determines the effective model for a (skill, baseModel) pair.
// Resolution order: combo(baseModel+skill) > skill > model(baseModel) > default.
// Returns the effective model alias and a result describing the decision.
//
// baseModel is the model that would be used without routing config — typically
// from InferModelFromSkill (hardcoded skill→model map) or project config default_model.
func (c *ModelRoutingConfig) Resolve(skill, baseModel string) ModelRouteResult {
	// 1. Check combo (highest precedence)
	if c != nil && c.Combos != nil {
		key := baseModel + "+" + skill
		if model, ok := c.Combos[key]; ok {
			reason := "combo override"
			if model != baseModel {
				reason = reasonChange("combo", baseModel, model, key)
			}
			return ModelRouteResult{
				EffectiveModel: model,
				Source:         "combo",
				BaseModel:      baseModel,
				ConfigKey:      key,
				Reason:         reason,
			}
		}
	}

	// 2. Check skill
	if c != nil && c.Skills != nil {
		if model, ok := c.Skills[skill]; ok {
			reason := "skill default"
			if model != baseModel && baseModel != "" {
				reason = reasonChange("skill", baseModel, model, skill)
			}
			return ModelRouteResult{
				EffectiveModel: model,
				Source:         "skill",
				BaseModel:      baseModel,
				ConfigKey:      skill,
				Reason:         reason,
			}
		}
	}

	// 3. Check model (base model override)
	if c != nil && c.Models != nil && baseModel != "" {
		if model, ok := c.Models[baseModel]; ok {
			return ModelRouteResult{
				EffectiveModel: model,
				Source:         "model",
				BaseModel:      baseModel,
				ConfigKey:      baseModel,
				Reason:         reasonChange("model", baseModel, model, baseModel),
			}
		}
	}

	// 4. Global default
	if c != nil && c.Default != "" {
		reason := "global default"
		if c.Default != baseModel && baseModel != "" {
			reason = reasonChange("default", baseModel, c.Default, "default")
		}
		return ModelRouteResult{
			EffectiveModel: c.Default,
			Source:         "default",
			BaseModel:      baseModel,
			Reason:         reason,
		}
	}

	// 5. No config — passthrough
	return ModelRouteResult{
		EffectiveModel: baseModel,
		Source:         "none",
		BaseModel:      baseModel,
		Reason:         "no routing config",
	}
}

// IsConfigured returns true if any routing rules are defined.
func (c *ModelRoutingConfig) IsConfigured() bool {
	if c == nil {
		return false
	}
	return c.Default != "" || len(c.Skills) > 0 || len(c.Models) > 0 || len(c.Combos) > 0
}

// reasonChange builds a human-readable reason for a model change.
func reasonChange(layer, from, to, key string) string {
	if from == "" {
		return layer + " config: " + key + " → " + to
	}
	return layer + " config: " + from + " → " + to + " (key: " + key + ")"
}
