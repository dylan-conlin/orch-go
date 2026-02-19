package spawn

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

func baseResolveInput() ResolveInput {
	return ResolveInput{
		SkillName: "feature-impl",
	}
}

func containsWarning(warnings []string, target string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, target) {
			return true
		}
	}
	return false
}

func TestResolve_PrecedenceLayers(t *testing.T) {
	t.Run("cli backend", func(t *testing.T) {
		input := baseResolveInput()
		input.CLI.Backend = BackendClaude

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
		}
		if settings.Backend.Source != SourceCLI {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceCLI)
		}
	})

	t.Run("project config backend", func(t *testing.T) {
		input := baseResolveInput()
		input.ProjectConfig = &config.Config{SpawnMode: BackendClaude}
		input.ProjectConfigMeta = ProjectConfigMeta{SpawnMode: true}

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
		}
		if settings.Backend.Source != SourceProjectConfig {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceProjectConfig)
		}
	})

	t.Run("user config backend", func(t *testing.T) {
		input := baseResolveInput()
		input.UserConfig = &userconfig.Config{Backend: BackendClaude}
		input.UserConfigMeta = UserConfigMeta{Backend: true}

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
		}
		if settings.Backend.Source != SourceUserConfig {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceUserConfig)
		}
	})

	t.Run("heuristic backend", func(t *testing.T) {
		input := baseResolveInput()
		input.InfrastructureDetected = true

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
		}
		if settings.Backend.Source != SourceHeuristic {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceHeuristic)
		}
		if settings.Backend.Detail != "infra-escape-hatch" {
			t.Fatalf("Backend.Detail = %q, want %q", settings.Backend.Detail, "infra-escape-hatch")
		}
	})

	t.Run("default backend", func(t *testing.T) {
		input := baseResolveInput()

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendOpenCode {
			t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendOpenCode)
		}
		if settings.Backend.Source != SourceDefault {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDefault)
		}
	})

	t.Run("beads label mcp", func(t *testing.T) {
		input := baseResolveInput()
		input.BeadsLabels = []string{"needs:playwright"}

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.MCP.Value != "playwright" {
			t.Fatalf("MCP.Value = %q, want %q", settings.MCP.Value, "playwright")
		}
		if settings.MCP.Source != SourceBeadsLabel {
			t.Fatalf("MCP.Source = %q, want %q", settings.MCP.Source, SourceBeadsLabel)
		}
	})
}

func TestResolve_BugClass01_ProjectConfigDefaultsDoNotOverrideUserBackend(t *testing.T) {
	input := baseResolveInput()
	input.ProjectConfig = &config.Config{SpawnMode: BackendOpenCode}
	input.ProjectConfigMeta = ProjectConfigMeta{SpawnMode: false}
	input.UserConfig = &userconfig.Config{Backend: BackendClaude}
	input.UserConfigMeta = UserConfigMeta{Backend: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Backend.Value != BackendClaude {
		t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
	}
	if settings.Backend.Source != SourceUserConfig {
		t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceUserConfig)
	}
}

func TestResolve_BugClass02_UserDefaultModelDoesNotOverrideProjectBackend(t *testing.T) {
	input := baseResolveInput()
	input.ProjectConfig = &config.Config{SpawnMode: BackendClaude}
	input.ProjectConfigMeta = ProjectConfigMeta{SpawnMode: true}
	input.UserConfig = &userconfig.Config{DefaultModel: "gpt-4o"}
	input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

	_, err := Resolve(input)
	if err == nil {
		t.Fatal("Resolve() error = nil, want compatibility error")
	}
	if !strings.Contains(err.Error(), "backend claude does not support provider openai") {
		t.Fatalf("Resolve() error = %q, want compatibility error", err)
	}
}

func TestResolve_AnthropicModelBlockedOnOpenCodeByDefault(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendOpenCode
	input.CLI.Model = "sonnet"

	_, err := Resolve(input)
	if err == nil {
		t.Fatal("Resolve() error = nil, want compatibility error")
	}
	if !strings.Contains(err.Error(), "allow_anthropic_opencode") {
		t.Fatalf("Resolve() error = %q, want allow_anthropic_opencode hint", err)
	}
}

func TestResolve_AnthropicModelAllowedWithUserConfigOverride(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendOpenCode
	input.CLI.Model = "sonnet"
	input.UserConfig = &userconfig.Config{AllowAnthropicOpenCode: true}
	input.UserConfigMeta = UserConfigMeta{AllowAnthropicOpenCode: true}

	_, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestResolve_BugClass03_ExplicitModelForcesBackendRequirement(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Model = "gpt-4o"

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Backend.Value != BackendOpenCode {
		t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendOpenCode)
	}
	if settings.Backend.Source != SourceDerived {
		t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDerived)
	}
	if settings.Backend.Detail != "model-requirement" {
		t.Fatalf("Backend.Detail = %q, want %q", settings.Backend.Detail, "model-requirement")
	}
}

func TestResolve_BugClass04_ProjectConfigPerBackendModelUsed(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendOpenCode
	input.ProjectConfig = &config.Config{OpenCode: config.OpenCodeConfig{Model: "gpt-4o"}}
	input.ProjectConfigMeta = ProjectConfigMeta{OpenCodeModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Model.Value != "openai/gpt-4o" {
		t.Fatalf("Model.Value = %q, want %q", settings.Model.Value, "openai/gpt-4o")
	}
	if settings.Model.Source != SourceProjectConfig {
		t.Fatalf("Model.Source = %q, want %q", settings.Model.Source, SourceProjectConfig)
	}
}

func TestResolve_BugClass05_ProjectDefaultFlashNotExplicit(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendOpenCode
	input.ProjectConfig = &config.Config{OpenCode: config.OpenCodeConfig{Model: "flash"}}
	input.ProjectConfigMeta = ProjectConfigMeta{OpenCodeModel: false}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Model.Value != model.DefaultModel.Format() {
		t.Fatalf("Model.Value = %q, want %q", settings.Model.Value, model.DefaultModel.Format())
	}
	if settings.Model.Source != SourceDefault {
		t.Fatalf("Model.Source = %q, want %q", settings.Model.Source, SourceDefault)
	}
}

func TestResolve_BugClass06_InfraEscapeHatchDoesNotOverrideExplicitBackend(t *testing.T) {
	input := baseResolveInput()
	input.InfrastructureDetected = true
	input.UserConfig = &userconfig.Config{Backend: BackendOpenCode}
	input.UserConfigMeta = UserConfigMeta{Backend: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Backend.Value != BackendOpenCode {
		t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendOpenCode)
	}
	if settings.Backend.Source != SourceUserConfig {
		t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceUserConfig)
	}
	if !containsWarning(settings.Warnings, "infrastructure work detected; user config backend overrides escape hatch") {
		t.Fatalf("Warnings = %v, want infra override warning", settings.Warnings)
	}
}

func TestResolve_BugClass07_UserBackendDefaultNotExplicit(t *testing.T) {
	input := baseResolveInput()
	input.InfrastructureDetected = true
	input.UserConfig = &userconfig.Config{Backend: BackendOpenCode}
	input.UserConfigMeta = UserConfigMeta{Backend: false}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Backend.Value != BackendClaude {
		t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
	}
	if settings.Backend.Source != SourceHeuristic {
		t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceHeuristic)
	}
}

func TestResolve_BugClass08_UserDefaultTierOnlyWhenExplicit(t *testing.T) {
	input := baseResolveInput()
	input.UserConfig = &userconfig.Config{DefaultTier: TierFull}
	input.UserConfigMeta = UserConfigMeta{DefaultTier: false}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Tier.Value != TierLight {
		t.Fatalf("Tier.Value = %q, want %q", settings.Tier.Value, TierLight)
	}
	if settings.Tier.Source != SourceHeuristic {
		t.Fatalf("Tier.Source = %q, want %q", settings.Tier.Source, SourceHeuristic)
	}
	if settings.Tier.Detail != "skill-default" {
		t.Fatalf("Tier.Detail = %q, want %q", settings.Tier.Detail, "skill-default")
	}
}

func TestResolve_BugClass09_MCPPrecedenceCLIOverLabel(t *testing.T) {
	input := baseResolveInput()
	input.CLI.MCP = "playwright"
	input.BeadsLabels = []string{"needs:puppeteer"}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.MCP.Value != "playwright" {
		t.Fatalf("MCP.Value = %q, want %q", settings.MCP.Value, "playwright")
	}
	if settings.MCP.Source != SourceCLI {
		t.Fatalf("MCP.Source = %q, want %q", settings.MCP.Source, SourceCLI)
	}
}

func TestResolve_BugClass10_UserDefaultModelNotInjectedAsCLI(t *testing.T) {
	input := baseResolveInput()
	input.UserConfig = &userconfig.Config{Backend: BackendOpenCode, DefaultModel: "gpt-4o"}
	input.UserConfigMeta = UserConfigMeta{Backend: true, DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Backend.Value != BackendOpenCode {
		t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendOpenCode)
	}
	if settings.Backend.Source != SourceUserConfig {
		t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceUserConfig)
	}
	if settings.Model.Source != SourceUserConfig {
		t.Fatalf("Model.Source = %q, want %q", settings.Model.Source, SourceUserConfig)
	}
}

// TestResolve_BugClass11_ProjectConfigModelOverridesUserDefaultModel reproduces
// orch-go-1105: project config opencode.model must take precedence over user
// config default_model. Prior to fix, runWork() loaded default_model into
// CLI.Model (highest priority), silently overriding project config.
func TestResolve_BugClass11_ProjectConfigModelOverridesUserDefaultModel(t *testing.T) {
	input := baseResolveInput()
	// Simulate: project config has opencode.model = "gpt-4o"
	input.ProjectConfig = &config.Config{OpenCode: config.OpenCodeConfig{Model: "gpt-4o"}}
	input.ProjectConfigMeta = ProjectConfigMeta{OpenCodeModel: true}
	// Simulate: user config has default_model = "codex" (should be lower priority)
	input.UserConfig = &userconfig.Config{DefaultModel: "openai/codex-mini-latest"}
	input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Project config model must win over user config default_model
	if settings.Model.Value != "openai/gpt-4o" {
		t.Fatalf("Model.Value = %q, want %q (project config should override user default_model)", settings.Model.Value, "openai/gpt-4o")
	}
	if settings.Model.Source != SourceProjectConfig {
		t.Fatalf("Model.Source = %q, want %q", settings.Model.Source, SourceProjectConfig)
	}
}

// TestResolve_BugClass11b_CLIModelStillOverridesProjectConfig verifies that
// --model CLI flag maintains highest priority even with project config set.
func TestResolve_BugClass11b_CLIModelStillOverridesProjectConfig(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Model = "opus"
	input.CLI.Backend = BackendClaude
	input.ProjectConfig = &config.Config{Claude: config.ClaudeConfig{Model: "sonnet"}}
	input.ProjectConfigMeta = ProjectConfigMeta{ClaudeModel: true}
	input.UserConfig = &userconfig.Config{DefaultModel: "haiku"}
	input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// CLI flag must win over everything
	if settings.Model.Source != SourceCLI {
		t.Fatalf("Model.Source = %q, want %q", settings.Model.Source, SourceCLI)
	}
	if !strings.Contains(settings.Model.Value, "opus") {
		t.Fatalf("Model.Value = %q, want opus variant", settings.Model.Value)
	}
}

// TestResolve_BugClass11c_UserDefaultModelFallbackWhenNoProjectConfig verifies
// user config default_model is used when no project config model is set.
func TestResolve_BugClass11c_UserDefaultModelFallbackWhenNoProjectConfig(t *testing.T) {
	input := baseResolveInput()
	input.UserConfig = &userconfig.Config{DefaultModel: "gpt-4o"}
	input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Model.Value != "openai/gpt-4o" {
		t.Fatalf("Model.Value = %q, want %q", settings.Model.Value, "openai/gpt-4o")
	}
	if settings.Model.Source != SourceUserConfig {
		t.Fatalf("Model.Source = %q, want %q", settings.Model.Source, SourceUserConfig)
	}
}
