package spawn

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

func TestBuildRoutingImpact_NoRouting(t *testing.T) {
	// Default Anthropic model on claude backend — no routing needed.
	input := baseResolveInput()
	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	impact := BuildRoutingImpact(settings)
	if impact.Triggered {
		t.Fatalf("Expected no routing impact for default Anthropic model, got triggered=%v trigger=%q",
			impact.Triggered, impact.Trigger)
	}
	if impact.Summary() != "" {
		t.Fatalf("Summary should be empty when not triggered, got %q", impact.Summary())
	}
}

func TestBuildRoutingImpact_OpenAI(t *testing.T) {
	// OpenAI model on default (claude) backend — should auto-route to opencode.
	input := baseResolveInput()
	input.CLI.Model = "openai/gpt-4o"

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	impact := BuildRoutingImpact(settings)
	if !impact.Triggered {
		t.Fatal("Expected routing impact for OpenAI model")
	}
	if impact.Trigger != "model-provider-routing" {
		t.Fatalf("Trigger = %q, want %q", impact.Trigger, "model-provider-routing")
	}
	if impact.Provider != "openai" {
		t.Fatalf("Provider = %q, want %q", impact.Provider, "openai")
	}
	if impact.ResolvedBackend != BackendOpenCode {
		t.Fatalf("ResolvedBackend = %q, want %q", impact.ResolvedBackend, BackendOpenCode)
	}
	if impact.PreviousBackend != BackendClaude {
		t.Fatalf("PreviousBackend = %q, want %q", impact.PreviousBackend, BackendClaude)
	}
	if impact.Automatic {
		t.Fatal("Expected Automatic=false for CLI model (user explicitly chose the model)")
	}
	if impact.Summary() == "" {
		t.Fatal("Summary should not be empty when triggered")
	}
}

func TestBuildRoutingImpact_Google(t *testing.T) {
	// Google model — should auto-route backend to opencode.
	input := baseResolveInput()
	input.CLI.Model = "google/gemini-2.5-pro"

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	impact := BuildRoutingImpact(settings)
	if !impact.Triggered {
		t.Fatal("Expected routing impact for Google model")
	}
	if impact.Provider != "google" {
		t.Fatalf("Provider = %q, want %q", impact.Provider, "google")
	}
	if impact.ResolvedBackend != BackendOpenCode {
		t.Fatalf("ResolvedBackend = %q, want %q", impact.ResolvedBackend, BackendOpenCode)
	}
	if impact.Automatic {
		t.Fatal("Expected Automatic=false for CLI model")
	}
}

func TestBuildRoutingImpact_DeepSeek(t *testing.T) {
	// DeepSeek model — should auto-route backend to opencode.
	input := baseResolveInput()
	input.CLI.Model = "deepseek/deepseek-chat"

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	impact := BuildRoutingImpact(settings)
	if !impact.Triggered {
		t.Fatal("Expected routing impact for DeepSeek model")
	}
	if impact.Provider != "deepseek" {
		t.Fatalf("Provider = %q, want %q", impact.Provider, "deepseek")
	}
	if impact.ResolvedBackend != BackendOpenCode {
		t.Fatalf("ResolvedBackend = %q, want %q", impact.ResolvedBackend, BackendOpenCode)
	}
}

func TestBuildRoutingImpact_ExplicitBackendOverride(t *testing.T) {
	// Explicit --backend claude with non-Anthropic model from user config.
	// The resolver overrides the model to match the backend (backend-compatibility).
	input := baseResolveInput()
	input.CLI.Backend = BackendClaude
	input.UserConfig = &userconfig.Config{DefaultModel: "openai/gpt-4o"}
	input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	impact := BuildRoutingImpact(settings)
	if settings.Model.Source == SourceDerived && settings.Model.Detail == "backend-compatibility" {
		// Model was overridden — routing impact should be triggered
		if !impact.Triggered {
			t.Fatal("Expected routing impact for backend-compatibility")
		}
		if impact.Trigger != "backend-compatibility" {
			t.Fatalf("Trigger = %q, want %q", impact.Trigger, "backend-compatibility")
		}
		if impact.Automatic {
			t.Fatal("Expected Automatic=false for explicit backend override")
		}
	}
}

func TestBuildRoutingImpact_OpenClawBypass(t *testing.T) {
	// OpenClaw backend with non-Anthropic model — should NOT trigger routing.
	// OpenClaw supports all providers natively.
	input := baseResolveInput()
	input.CLI.Backend = BackendOpenClaw
	input.CLI.Model = "openai/gpt-4o"

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	impact := BuildRoutingImpact(settings)
	if impact.Triggered {
		t.Fatalf("Expected no routing impact for OpenClaw (supports all providers), got trigger=%q",
			impact.Trigger)
	}
}

func TestBuildRoutingImpact_UserConfigAutoRoute(t *testing.T) {
	// User config sets backend=opencode, but default model is Anthropic.
	// model-provider-routing should auto-route backend to claude.
	input := baseResolveInput()
	input.UserConfig = &userconfig.Config{Backend: BackendOpenCode}
	input.UserConfigMeta = UserConfigMeta{Backend: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	impact := BuildRoutingImpact(settings)
	if !impact.Triggered {
		t.Fatal("Expected routing impact when user-config backend is overridden by model-provider-routing")
	}
	if impact.Trigger != "model-provider-routing" {
		t.Fatalf("Trigger = %q, want %q", impact.Trigger, "model-provider-routing")
	}
	if impact.ResolvedBackend != BackendClaude {
		t.Fatalf("ResolvedBackend = %q, want %q", impact.ResolvedBackend, BackendClaude)
	}
}

func TestBuildRoutingImpact_JSON(t *testing.T) {
	// Verify JSON serialization is well-formed.
	input := baseResolveInput()
	input.CLI.Model = "openai/gpt-4o"

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	impact := BuildRoutingImpact(settings)
	jsonStr := impact.JSON()

	// Should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("JSON() produced invalid JSON: %v\nGot: %s", err, jsonStr)
	}

	// Should contain key fields
	if _, ok := parsed["triggered"]; !ok {
		t.Fatal("JSON missing 'triggered' field")
	}
	if _, ok := parsed["trigger"]; !ok {
		t.Fatal("JSON missing 'trigger' field")
	}
	if !strings.Contains(jsonStr, "openai") {
		t.Fatalf("JSON should contain provider 'openai', got: %s", jsonStr)
	}
}

func TestBuildRoutingImpact_ProjectConfigNonAnthropicModel(t *testing.T) {
	// Project config specifies an OpenAI model for opencode backend,
	// but no explicit CLI backend. Default backend is claude.
	// The model should trigger model-provider-routing.
	input := baseResolveInput()
	input.UserConfig = &userconfig.Config{DefaultModel: "openai/gpt-4o"}
	input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	impact := BuildRoutingImpact(settings)
	if !impact.Triggered {
		t.Fatal("Expected routing impact for non-Anthropic model from user config")
	}
	if impact.ResolvedBackend != BackendOpenCode {
		t.Fatalf("ResolvedBackend = %q, want %q", impact.ResolvedBackend, BackendOpenCode)
	}
}
