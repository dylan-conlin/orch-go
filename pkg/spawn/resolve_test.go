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
		// Default backend is now claude since default model is Anthropic (sonnet).
		// Decision kb-2d62ef: Anthropic banned subscription OAuth in third-party tools (Feb 19 2026).
		input := baseResolveInput()

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
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

// TestResolve_BugClass02_NonAnthropicModelAutoRoutesBackend verifies that
// when project config says backend=claude but user config model is non-anthropic,
// model-aware routing auto-switches backend to opencode (model determines backend).
// This replaces the old behavior where the model was overridden to match the backend.
func TestResolve_BugClass02_NonAnthropicModelAutoRoutesBackend(t *testing.T) {
	input := baseResolveInput()
	input.ProjectConfig = &config.Config{SpawnMode: BackendClaude}
	input.ProjectConfigMeta = ProjectConfigMeta{SpawnMode: true}
	input.UserConfig = &userconfig.Config{DefaultModel: "gpt-4o"}
	input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Model-aware routing: model determines backend. Backend auto-routes to opencode.
	if settings.Backend.Value != BackendOpenCode {
		t.Fatalf("Backend.Value = %q, want %q (model-aware routing: openai model → opencode)", settings.Backend.Value, BackendOpenCode)
	}
	if settings.Backend.Source != SourceDerived {
		t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDerived)
	}
	// Model should NOT be overridden (it's the user's preference)
	if settings.Model.Value != "openai/gpt-4o" {
		t.Fatalf("Model.Value = %q, want %q (model should be preserved)", settings.Model.Value, "openai/gpt-4o")
	}
	if !containsWarning(settings.Warnings, "Auto-routed backend to opencode") {
		t.Fatalf("Warnings = %v, want auto-route warning", settings.Warnings)
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
	// Test that non-explicit project config model (OpenCodeModel: false) is ignored.
	// Uses gpt-4o as user default since opencode backend + anthropic model is now blocked.
	input := baseResolveInput()
	input.CLI.Backend = BackendOpenCode
	input.ProjectConfig = &config.Config{OpenCode: config.OpenCodeConfig{Model: "flash"}}
	input.ProjectConfigMeta = ProjectConfigMeta{OpenCodeModel: false}
	// Need a non-Anthropic default model since opencode backend is explicit
	input.UserConfig = &userconfig.Config{DefaultModel: "gpt-4o"}
	input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Falls through to user config model (gpt-4o) since project config is not explicit
	if settings.Model.Value != "openai/gpt-4o" {
		t.Fatalf("Model.Value = %q, want %q", settings.Model.Value, "openai/gpt-4o")
	}
	if settings.Model.Source != SourceUserConfig {
		t.Fatalf("Model.Source = %q, want %q", settings.Model.Source, SourceUserConfig)
	}
}

func TestResolve_BugClass06_InfraEscapeHatchDoesNotOverrideExplicitBackend(t *testing.T) {
	// Test that explicit user config backend is honored even when infrastructure is detected.
	// Uses gpt-4o as model since opencode backend + anthropic model is now blocked.
	input := baseResolveInput()
	input.InfrastructureDetected = true
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
	// Explicit backend required for opencode.model to be used (default is now claude)
	input.CLI.Backend = BackendOpenCode
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

// TestResolve_BugClass12_CLIBackendClaudeAutoResolvesOpenAIDefault reproduces
// orch-go-1127: --backend claude + user default_model gpt-4o should auto-resolve
// to anthropic default, not error with "backend claude does not support provider openai".
func TestResolve_BugClass12_CLIBackendClaudeAutoResolvesOpenAIDefault(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendClaude
	input.UserConfig = &userconfig.Config{DefaultModel: "gpt-4o"}
	input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v, want auto-resolve to Anthropic default", err)
	}
	if settings.Model.Value != model.DefaultModel.Format() {
		t.Fatalf("Model.Value = %q, want %q", settings.Model.Value, model.DefaultModel.Format())
	}
	if settings.Model.Source != SourceDerived {
		t.Fatalf("Model.Source = %q, want %q", settings.Model.Source, SourceDerived)
	}
	if settings.Model.Detail != "backend-compatibility" {
		t.Fatalf("Model.Detail = %q, want %q", settings.Model.Detail, "backend-compatibility")
	}
	if !containsWarning(settings.Warnings, "Auto-resolved model to") {
		t.Fatalf("Warnings = %v, want auto-resolve warning", settings.Warnings)
	}
}

// TestResolve_BugClass13_ClaudeBackendImpliesTmuxSpawnMode reproduces
// orch-go-1129: --backend claude alone should default spawn mode to tmux,
// not headless. Without this, DispatchSpawn hits the headless path before
// reaching the claude routing check.
func TestResolve_BugClass13_ClaudeBackendImpliesTmuxSpawnMode(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendClaude

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.SpawnMode.Value != SpawnModeTmux {
		t.Fatalf("SpawnMode.Value = %q, want %q (claude backend should imply tmux)", settings.SpawnMode.Value, SpawnModeTmux)
	}
	if settings.SpawnMode.Source != SourceDerived {
		t.Fatalf("SpawnMode.Source = %q, want %q", settings.SpawnMode.Source, SourceDerived)
	}
	if settings.SpawnMode.Detail != "claude-backend-requires-tmux" {
		t.Fatalf("SpawnMode.Detail = %q, want %q", settings.SpawnMode.Detail, "claude-backend-requires-tmux")
	}
}

// TestResolve_BugClass13b_ClaudeBackendOverridesHeadless verifies that
// claude backend always overrides headless to tmux, even with explicit --headless.
// Claude backend physically requires a tmux window (SpawnClaude creates tmux window
// + claude CLI). Headless mode uses OpenCode HTTP API which is incompatible.
func TestResolve_BugClass13b_ClaudeBackendOverridesHeadless(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendClaude
	input.CLI.Headless = true

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Claude backend requires tmux - headless is physically incompatible
	if settings.SpawnMode.Value != SpawnModeTmux {
		t.Fatalf("SpawnMode.Value = %q, want %q (claude backend requires tmux, headless incompatible)", settings.SpawnMode.Value, SpawnModeTmux)
	}
	if settings.SpawnMode.Source != SourceDerived {
		t.Fatalf("SpawnMode.Source = %q, want %q", settings.SpawnMode.Source, SourceDerived)
	}
}

// TestResolve_BugClass13c_ExplicitTmuxWithClaudeBackendStaysExplicit verifies that
// --backend claude --tmux results in tmux from CLI source (not derived).
func TestResolve_BugClass13c_ExplicitTmuxWithClaudeBackendStaysExplicit(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendClaude
	input.CLI.Tmux = true

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.SpawnMode.Value != SpawnModeTmux {
		t.Fatalf("SpawnMode.Value = %q, want %q", settings.SpawnMode.Value, SpawnModeTmux)
	}
	if settings.SpawnMode.Source != SourceCLI {
		t.Fatalf("SpawnMode.Source = %q, want %q (explicit --tmux should be CLI source)", settings.SpawnMode.Source, SourceCLI)
	}
}

// TestResolve_BugClass13d_InfraEscapeHatchAlsoImpliesTmux verifies that
// infrastructure-detected claude backend (heuristic) also implies tmux.
func TestResolve_BugClass13d_InfraEscapeHatchAlsoImpliesTmux(t *testing.T) {
	input := baseResolveInput()
	input.InfrastructureDetected = true

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Backend.Value != BackendClaude {
		t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
	}
	if settings.SpawnMode.Value != SpawnModeTmux {
		t.Fatalf("SpawnMode.Value = %q, want %q (infra escape hatch should imply tmux)", settings.SpawnMode.Value, SpawnModeTmux)
	}
}

// TestResolve_BugClass12b_CLIBackendClaudeWithExplicitModelStillErrors verifies
// that --backend claude + explicit --model gpt-4o still errors (user explicitly chose incompatible combo).
func TestResolve_BugClass12b_CLIBackendClaudeWithExplicitModelStillErrors(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendClaude
	input.CLI.Model = "gpt-4o"

	_, err := Resolve(input)
	if err == nil {
		t.Fatal("Resolve() error = nil, want compatibility error for explicit incompatible model")
	}
	if !strings.Contains(err.Error(), "backend claude does not support provider openai") {
		t.Fatalf("Resolve() error = %q, want compatibility error", err)
	}
}

// TestResolve_BugClass11c_UserDefaultModelFallbackWhenNoProjectConfig verifies
// user config default_model is used when no project config model is set.
// Requires explicit backend=opencode since default backend is now claude.
func TestResolve_BugClass11c_UserDefaultModelFallbackWhenNoProjectConfig(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Backend = BackendOpenCode // Explicit backend to use OpenAI model
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

// TestResolve_BugClass14_FlashProjectConfigFallsThrough reproduces orch-go-1145:
// project config opencode.model=flash hits validateModel() gate, which should
// fall through to user config default_model instead of hard-failing.
// Without this fix, the daemon gets 857 consecutive spawn failures because
// resolveModel returns an error instead of trying the next precedence level.
func TestResolve_BugClass14_FlashProjectConfigFallsThrough(t *testing.T) {
	input := baseResolveInput()
	// Simulate the exact daemon config that caused 857 failures:
	// Project config: spawn_mode=opencode, opencode.model=flash
	// User config: backend=claude, default_model=opus
	input.ProjectConfig = &config.Config{
		SpawnMode: BackendOpenCode,
		OpenCode:  config.OpenCodeConfig{Model: "flash"},
	}
	input.ProjectConfigMeta = ProjectConfigMeta{SpawnMode: true, OpenCodeModel: true}
	input.UserConfig = &userconfig.Config{Backend: BackendClaude, DefaultModel: "opus"}
	input.UserConfigMeta = UserConfigMeta{Backend: false, DefaultModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v, want graceful fallthrough from flash to opus", err)
	}
	// Model should fall through from flash (rejected) to opus (user config)
	if !strings.Contains(settings.Model.Value, "opus") {
		t.Fatalf("Model.Value = %q, want opus variant (flash should fall through)", settings.Model.Value)
	}
	if settings.Model.Source != SourceUserConfig {
		t.Fatalf("Model.Source = %q, want %q (should fall through to user config)", settings.Model.Source, SourceUserConfig)
	}
	// Backend should auto-route from opencode to claude (anthropic model requires claude)
	if settings.Backend.Value != BackendClaude {
		t.Fatalf("Backend.Value = %q, want %q (should auto-route for anthropic model)", settings.Backend.Value, BackendClaude)
	}
	if settings.Backend.Source != SourceDerived {
		t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDerived)
	}
	// Should get warning about auto-routing
	if !containsWarning(settings.Warnings, "Auto-routed backend to claude") {
		t.Fatalf("Warnings = %v, want auto-route warning", settings.Warnings)
	}
	// Claude backend should require tmux spawn mode
	if settings.SpawnMode.Value != SpawnModeTmux {
		t.Fatalf("SpawnMode.Value = %q, want %q (claude backend requires tmux)", settings.SpawnMode.Value, SpawnModeTmux)
	}
}

// TestResolve_BugClass14b_FlashProjectConfigNoUserFallback verifies that
// when flash is rejected and no user config model exists, DefaultModel is used.
func TestResolve_BugClass14b_FlashProjectConfigNoUserFallback(t *testing.T) {
	input := baseResolveInput()
	input.ProjectConfig = &config.Config{
		SpawnMode: BackendOpenCode,
		OpenCode:  config.OpenCodeConfig{Model: "flash"},
	}
	input.ProjectConfigMeta = ProjectConfigMeta{SpawnMode: true, OpenCodeModel: true}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v, want fallthrough to DefaultModel", err)
	}
	// Should fall through to DefaultModel (sonnet)
	if settings.Model.Value != model.DefaultModel.Format() {
		t.Fatalf("Model.Value = %q, want %q", settings.Model.Value, model.DefaultModel.Format())
	}
	if settings.Model.Source != SourceDefault {
		t.Fatalf("Model.Source = %q, want %q", settings.Model.Source, SourceDefault)
	}
	// Backend should auto-switch to claude (anthropic model on opencode)
	if settings.Backend.Value != BackendClaude {
		t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
	}
}

// TestResolve_BugClass14c_FlashCLIModelStillErrors verifies that
// explicit --model flash still returns an error (user explicitly chose it).
func TestResolve_BugClass14c_FlashCLIModelStillErrors(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Model = "flash"

	_, err := Resolve(input)
	if err == nil {
		t.Fatal("Resolve() error = nil, want validation error for explicit flash")
	}
	if !strings.Contains(err.Error(), "flash models are not supported") {
		t.Fatalf("Resolve() error = %q, want flash validation error", err)
	}
}

// TestResolve_BugClass15_ModelAwareBackendRouting tests model-aware backend routing.
// Decision kb-2d62ef: Anthropic models auto-route to claude backend,
// non-Anthropic models auto-route to opencode backend.
// This became mandatory when Anthropic banned subscription OAuth in third-party tools (Feb 19 2026).
func TestResolve_BugClass15_ModelAwareBackendRouting(t *testing.T) {
	t.Run("explicit anthropic model routes to claude backend", func(t *testing.T) {
		input := baseResolveInput()
		input.CLI.Model = "opus"

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q (anthropic model should route to claude)", settings.Backend.Value, BackendClaude)
		}
		if settings.Backend.Source != SourceDerived {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDerived)
		}
		if settings.Backend.Detail != "model-requirement" {
			t.Fatalf("Backend.Detail = %q, want %q", settings.Backend.Detail, "model-requirement")
		}
	})

	t.Run("explicit sonnet model routes to claude backend", func(t *testing.T) {
		input := baseResolveInput()
		input.CLI.Model = "sonnet"

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q (anthropic model should route to claude)", settings.Backend.Value, BackendClaude)
		}
		if settings.Backend.Source != SourceDerived {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDerived)
		}
	})

	t.Run("explicit haiku model routes to claude backend", func(t *testing.T) {
		input := baseResolveInput()
		input.CLI.Model = "haiku"

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q (anthropic model should route to claude)", settings.Backend.Value, BackendClaude)
		}
	})

	t.Run("explicit openai model routes to opencode backend", func(t *testing.T) {
		input := baseResolveInput()
		input.CLI.Model = "gpt-4o"

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendOpenCode {
			t.Fatalf("Backend.Value = %q, want %q (openai model should route to opencode)", settings.Backend.Value, BackendOpenCode)
		}
		if settings.Backend.Source != SourceDerived {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDerived)
		}
		if settings.Backend.Detail != "model-requirement" {
			t.Fatalf("Backend.Detail = %q, want %q", settings.Backend.Detail, "model-requirement")
		}
	})

	t.Run("explicit codex model routes to opencode backend", func(t *testing.T) {
		input := baseResolveInput()
		input.CLI.Model = "codex"

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendOpenCode {
			t.Fatalf("Backend.Value = %q, want %q (openai codex model should route to opencode)", settings.Backend.Value, BackendOpenCode)
		}
	})

	t.Run("explicit deepseek model routes to opencode backend", func(t *testing.T) {
		input := baseResolveInput()
		input.CLI.Model = "deepseek"

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendOpenCode {
			t.Fatalf("Backend.Value = %q, want %q (deepseek model should route to opencode)", settings.Backend.Value, BackendOpenCode)
		}
	})

	t.Run("default backend is claude (since default model is anthropic)", func(t *testing.T) {
		input := baseResolveInput()

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q (default should be claude since default model is anthropic)", settings.Backend.Value, BackendClaude)
		}
		if settings.Backend.Source != SourceDefault {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDefault)
		}
		// Verify default model is anthropic sonnet
		if !strings.Contains(settings.Model.Value, "sonnet") {
			t.Fatalf("Model.Value = %q, want sonnet variant (default model)", settings.Model.Value)
		}
	})

	t.Run("explicit CLI backend overrides model-aware routing", func(t *testing.T) {
		// User explicitly chose opencode backend, but uses anthropic model
		// This should error (incompatible combo) unless allow_anthropic_opencode is set
		input := baseResolveInput()
		input.CLI.Backend = BackendOpenCode
		input.CLI.Model = "sonnet"

		_, err := Resolve(input)
		if err == nil {
			t.Fatal("Resolve() error = nil, want compatibility error for explicit incompatible combo")
		}
		if !strings.Contains(err.Error(), "allow_anthropic_opencode") {
			t.Fatalf("Resolve() error = %q, want hint about allow_anthropic_opencode override", err)
		}
	})

	t.Run("explicit CLI backend claude with openai model errors", func(t *testing.T) {
		// User explicitly chose claude backend with openai model - incompatible
		input := baseResolveInput()
		input.CLI.Backend = BackendClaude
		input.CLI.Model = "gpt-4o"

		_, err := Resolve(input)
		if err == nil {
			t.Fatal("Resolve() error = nil, want compatibility error")
		}
		if !strings.Contains(err.Error(), "backend claude does not support provider openai") {
			t.Fatalf("Resolve() error = %q, want compatibility error", err)
		}
	})

	t.Run("anthropic model implies tmux spawn mode", func(t *testing.T) {
		// When anthropic model routes to claude backend, spawn mode should be tmux
		input := baseResolveInput()
		input.CLI.Model = "opus"

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.SpawnMode.Value != SpawnModeTmux {
			t.Fatalf("SpawnMode.Value = %q, want %q (claude backend implies tmux)", settings.SpawnMode.Value, SpawnModeTmux)
		}
	})

	t.Run("user config opencode backend auto-routes to claude for anthropic model", func(t *testing.T) {
		// User config says backend=opencode, but model resolves to anthropic (default).
		// Model-aware routing should auto-switch backend to claude.
		// This generalizes BugClass14 (which only worked for project config).
		input := baseResolveInput()
		input.UserConfig = &userconfig.Config{Backend: BackendOpenCode}
		input.UserConfigMeta = UserConfigMeta{Backend: true}

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v, want auto-route to claude", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q (anthropic model should auto-route to claude)", settings.Backend.Value, BackendClaude)
		}
		if settings.Backend.Source != SourceDerived {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDerived)
		}
		if settings.Backend.Detail != "model-provider-routing" {
			t.Fatalf("Backend.Detail = %q, want %q", settings.Backend.Detail, "model-provider-routing")
		}
	})

	t.Run("user config default_model codex auto-routes to opencode", func(t *testing.T) {
		// User config says default_model=codex, no explicit backend.
		// Default backend is claude, but model-aware routing should auto-switch
		// backend to opencode since codex is an OpenAI model.
		input := baseResolveInput()
		input.UserConfig = &userconfig.Config{DefaultModel: "codex"}
		input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v, want auto-route to opencode", err)
		}
		if settings.Backend.Value != BackendOpenCode {
			t.Fatalf("Backend.Value = %q, want %q (codex model should auto-route to opencode)", settings.Backend.Value, BackendOpenCode)
		}
		if settings.Backend.Source != SourceDerived {
			t.Fatalf("Backend.Source = %q, want %q", settings.Backend.Source, SourceDerived)
		}
		// Model should remain codex (not overridden to anthropic)
		if !strings.Contains(settings.Model.Value, "codex") {
			t.Fatalf("Model.Value = %q, want codex variant (model should not be overridden)", settings.Model.Value)
		}
	})

	t.Run("daemon path headless with claude backend resolves to tmux", func(t *testing.T) {
		// Daemon calls orch work which passes headless=true.
		// When backend resolves to claude, spawn mode MUST be tmux
		// because claude backend physically requires tmux (SpawnClaude creates tmux window).
		input := baseResolveInput()
		input.CLI.Headless = true
		// No explicit backend or model → default is claude backend + anthropic model

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendClaude {
			t.Fatalf("Backend.Value = %q, want %q", settings.Backend.Value, BackendClaude)
		}
		// Even though headless was passed, claude backend requires tmux
		if settings.SpawnMode.Value != SpawnModeTmux {
			t.Fatalf("SpawnMode.Value = %q, want %q (claude backend requires tmux, headless incompatible)", settings.SpawnMode.Value, SpawnModeTmux)
		}
		if settings.SpawnMode.Source != SourceDerived {
			t.Fatalf("SpawnMode.Source = %q, want %q", settings.SpawnMode.Source, SourceDerived)
		}
	})

	t.Run("project config claude backend with non-anthropic model auto-routes backend", func(t *testing.T) {
		// Project config says backend=claude, user config says model=gpt-4o.
		// Model-aware routing: model determines backend (unless --backend CLI flag).
		// Project config backend is not CLI, so model wins → backend routes to opencode.
		input := baseResolveInput()
		input.ProjectConfig = &config.Config{SpawnMode: BackendClaude}
		input.ProjectConfigMeta = ProjectConfigMeta{SpawnMode: true}
		input.UserConfig = &userconfig.Config{DefaultModel: "gpt-4o"}
		input.UserConfigMeta = UserConfigMeta{DefaultModel: true}

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if settings.Backend.Value != BackendOpenCode {
			t.Fatalf("Backend.Value = %q, want %q (model-aware routing: openai model → opencode backend)", settings.Backend.Value, BackendOpenCode)
		}
		if settings.Model.Value != "openai/gpt-4o" {
			t.Fatalf("Model.Value = %q, want %q (model should not be overridden)", settings.Model.Value, "openai/gpt-4o")
		}
	})
}
