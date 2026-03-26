package spawn

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RoutingImpact is a machine-readable report of provider-driven routing changes
// that occurred during spawn resolution. It captures when the resolver auto-routes
// the backend or model away from the configured/default values due to model-provider
// compatibility requirements.
type RoutingImpact struct {
	// Triggered is true when any routing change was applied.
	Triggered bool `json:"triggered"`

	// Trigger describes what caused the routing change.
	// Examples: "model-provider-routing", "backend-compatibility", ""
	Trigger string `json:"trigger,omitempty"`

	// PreviousBackend is what the backend would have been without routing.
	PreviousBackend string `json:"previous_backend,omitempty"`

	// ResolvedBackend is the final backend after routing.
	ResolvedBackend string `json:"resolved_backend,omitempty"`

	// PreviousModel is what the model would have been without routing.
	PreviousModel string `json:"previous_model,omitempty"`

	// ResolvedModel is the final model after routing.
	ResolvedModel string `json:"resolved_model,omitempty"`

	// Provider is the model provider that drove the routing change.
	Provider string `json:"provider,omitempty"`

	// Automatic is true when the change was auto-applied (not explicit CLI).
	Automatic bool `json:"automatic"`

	// Explanation is a human-readable summary of what happened and why.
	Explanation string `json:"explanation,omitempty"`
}

// JSON returns the routing impact as a JSON string.
func (r RoutingImpact) JSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// Summary returns a one-line human-readable summary.
// Returns empty string if no routing change was triggered.
func (r RoutingImpact) Summary() string {
	if !r.Triggered {
		return ""
	}
	return r.Explanation
}

// BuildRoutingImpact constructs a RoutingImpact from resolved spawn settings.
// It detects whether the resolver applied model-provider routing or backend-compatibility
// overrides by examining the Backend and Model sources and details.
//
// Three routing cases are detected:
//  1. model-requirement: CLI model triggers backend derivation in resolveBackend()
//  2. model-provider-routing: non-CLI model source triggers backend override post-resolution
//  3. backend-compatibility: explicit CLI backend triggers model override
func BuildRoutingImpact(settings ResolvedSpawnSettings) RoutingImpact {
	impact := RoutingImpact{}

	// Extract provider from the model value (format: "provider/model-id")
	extractProvider := func(modelValue string) string {
		if parts := strings.SplitN(modelValue, "/", 2); len(parts) == 2 {
			return parts[0]
		}
		return ""
	}

	// Case 1: model-requirement — CLI model determined the backend in resolveBackend().
	// When the user passes --model openai/gpt-4o, the backend is derived from the model's provider.
	if settings.Backend.Source == SourceDerived && settings.Backend.Detail == "model-requirement" {
		provider := extractProvider(settings.Model.Value)
		// Only report as routing impact for non-Anthropic providers.
		// Anthropic model → claude backend is the normal/default path.
		if provider != "" && provider != "anthropic" {
			impact.Triggered = true
			impact.Trigger = "model-provider-routing"
			impact.ResolvedBackend = settings.Backend.Value
			impact.ResolvedModel = settings.Model.Value
			impact.Provider = provider
			impact.Automatic = false // CLI model was explicit
			impact.PreviousBackend = BackendClaude
			impact.Explanation = fmt.Sprintf(
				"Backend derived from CLI model: %s requires %s backend (default would be %s)",
				settings.Model.Value, settings.Backend.Value, BackendClaude,
			)
			return impact
		}
	}

	// Case 2: model-provider-routing — backend was auto-routed to match a non-CLI model source.
	// This happens when the configured backend (project/user/default) doesn't match the model's provider.
	if settings.Backend.Source == SourceDerived && settings.Backend.Detail == "model-provider-routing" {
		impact.Triggered = true
		impact.Trigger = "model-provider-routing"
		impact.ResolvedBackend = settings.Backend.Value
		impact.ResolvedModel = settings.Model.Value
		impact.Automatic = true
		impact.Provider = extractProvider(settings.Model.Value)

		// Determine what the backend would have been without routing.
		if impact.ResolvedBackend == BackendOpenCode {
			impact.PreviousBackend = BackendClaude
		} else {
			impact.PreviousBackend = BackendOpenCode
		}

		impact.Explanation = fmt.Sprintf(
			"Auto-routed backend from %s to %s: model %s requires %s provider backend",
			impact.PreviousBackend, impact.ResolvedBackend,
			settings.Model.Value, impact.Provider,
		)
		return impact
	}

	// Case 3: backend-compatibility — model was auto-resolved to match an explicit CLI backend.
	// This happens when --backend claude is set but the configured model is non-Anthropic.
	if settings.Model.Source == SourceDerived && settings.Model.Detail == "backend-compatibility" {
		impact.Triggered = true
		impact.Trigger = "backend-compatibility"
		impact.ResolvedBackend = settings.Backend.Value
		impact.ResolvedModel = settings.Model.Value
		impact.Automatic = false // Backend was explicitly set via CLI
		impact.Explanation = fmt.Sprintf(
			"Model overridden to %s: explicit %s backend requires Anthropic model",
			settings.Model.Value, settings.Backend.Value,
		)
		return impact
	}

	return impact
}
