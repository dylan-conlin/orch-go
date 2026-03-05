package spawn

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// mockAccount represents an account for testing.
type mockAccount struct {
	role      string
	configDir string
	tier      string // e.g. "5x", "20x"
}

// mockAccountConfig implements AccountConfigProvider for testing.
type mockAccountConfig struct {
	accounts map[string]mockAccount
}

func (m *mockAccountConfig) GetAccounts() map[string]AccountInfo2 {
	result := make(map[string]AccountInfo2)
	for name, acc := range m.accounts {
		result[name] = AccountInfo2{Role: acc.role, ConfigDir: acc.configDir, Tier: acc.tier}
	}
	return result
}

func (m *mockAccountConfig) GetDefault() string {
	for name, acc := range m.accounts {
		if acc.role == "primary" {
			return name
		}
	}
	return ""
}

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

	t.Run("beads label browser tool", func(t *testing.T) {
		input := baseResolveInput()
		input.BeadsLabels = []string{"needs:playwright"}

		settings, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		// needs:playwright sets BrowserTool, not MCP
		if settings.BrowserTool.Value != "playwright-cli" {
			t.Fatalf("BrowserTool.Value = %q, want %q", settings.BrowserTool.Value, "playwright-cli")
		}
		if settings.BrowserTool.Source != SourceBeadsLabel {
			t.Fatalf("BrowserTool.Source = %q, want %q", settings.BrowserTool.Source, SourceBeadsLabel)
		}
		// MCP should remain empty (not set by needs: labels)
		if settings.MCP.Value != "" {
			t.Fatalf("MCP.Value = %q, want empty (needs:playwright goes to BrowserTool)", settings.MCP.Value)
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

func TestResolve_BugClass09_MCPAndBrowserToolIndependent(t *testing.T) {
	input := baseResolveInput()
	input.CLI.MCP = "playwright"
	input.BeadsLabels = []string{"needs:playwright"}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// --mcp playwright goes to MCP (MCP server override)
	if settings.MCP.Value != "playwright" {
		t.Fatalf("MCP.Value = %q, want %q", settings.MCP.Value, "playwright")
	}
	if settings.MCP.Source != SourceCLI {
		t.Fatalf("MCP.Source = %q, want %q", settings.MCP.Source, SourceCLI)
	}
	// needs:playwright goes to BrowserTool (default CLI path)
	if settings.BrowserTool.Value != "playwright-cli" {
		t.Fatalf("BrowserTool.Value = %q, want %q", settings.BrowserTool.Value, "playwright-cli")
	}
}

// TestResolve_BrowserToolEndToEnd_LabelsToContextInjection verifies the full chain:
// needs:playwright label → Resolve() → Config.BrowserTool → GenerateContext() → SPAWN_CONTEXT.md
// This is a regression test for orch-go-vv7l where cross-project spawns lost labels,
// causing 'Browser: none (source: default)' instead of 'Browser: playwright-cli'.
func TestResolve_BrowserToolEndToEnd_LabelsToContextInjection(t *testing.T) {
	// Step 1: Resolve with needs:playwright label
	input := baseResolveInput()
	input.BeadsLabels = []string{"needs:playwright"}

	resolved, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Step 2: Build Config as BuildSpawnConfig would
	cfg := &Config{
		Task:             "test task",
		SkillName:        "investigation",
		Project:          "test-project",
		ProjectDir:       "/tmp/test-project",
		WorkspaceName:    "test-workspace",
		BeadsID:          "test-123",
		Tier:             TierFull,
		BrowserTool:      resolved.BrowserTool.Value,
		ResolvedSettings: resolved,
	}

	// Step 3: Generate context
	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext() error = %v", err)
	}

	// Step 4: Verify BROWSER AUTOMATION section is present
	if !strings.Contains(content, "BROWSER AUTOMATION") {
		t.Error("BROWSER AUTOMATION section not found in generated context")
	}

	// Step 5: Verify CONFIG RESOLUTION shows correct browser tool (not "none")
	if !strings.Contains(content, "Browser: playwright-cli") {
		t.Error("CONFIG RESOLUTION does not show 'Browser: playwright-cli'")
	}
	if strings.Contains(content, "Browser: none") {
		t.Error("CONFIG RESOLUTION incorrectly shows 'Browser: none'")
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

// TestResolve_AccountCLIFlag tests that --account CLI flag is resolved correctly.
func TestResolve_AccountCLIFlag(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Account = "personal"

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q", settings.Account.Value, "personal")
	}
	if settings.Account.Source != SourceCLI {
		t.Fatalf("Account.Source = %q, want %q", settings.Account.Source, SourceCLI)
	}
}

// TestResolve_AccountDefaultEmpty tests that without CLI flag and no accounts config,
// account resolves to empty default.
func TestResolve_AccountDefaultEmpty(t *testing.T) {
	input := baseResolveInput()

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Account may or may not have a value depending on whether accounts.yaml exists
	// on the test machine. The important thing is it doesn't error.
	if settings.Account.Source != SourceDefault {
		// CLI was not set, so source must be default (or default with detail)
		if settings.Account.Source != SourceDefault {
			t.Fatalf("Account.Source = %q, want %q", settings.Account.Source, SourceDefault)
		}
	}
}

// ============================================================================
// Account Heuristic Routing Tests (tier-weighted 5h headroom)
// ============================================================================

func TestResolve_AccountHeuristic_HighestWeeklyHeadroomWins(t *testing.T) {
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			return &account.CapacityInfo{FiveHourRemaining: 87, SevenDayRemaining: 72}
		case "personal":
			// Personal has more weekly headroom (88% > 72%)
			return &account.CapacityInfo{FiveHourRemaining: 95, SevenDayRemaining: 88}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Personal wins because 95 5h > 87 5h (tier-weighted, both default 1x)
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (highest 5h headroom)", settings.Account.Value, "personal")
	}
	if settings.Account.Source != SourceHeuristic {
		t.Fatalf("Account.Source = %q, want %q", settings.Account.Source, SourceHeuristic)
	}
	if !strings.Contains(settings.Account.Detail, "5h-headroom") {
		t.Fatalf("Account.Detail = %q, want contains '5h-headroom'", settings.Account.Detail)
	}
}

func TestResolve_AccountHeuristic_WorkWinsWhenMoreFiveHourHeadroom(t *testing.T) {
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			// Work has more 5h headroom (90% > 40%)
			return &account.CapacityInfo{FiveHourRemaining: 90, SevenDayRemaining: 40}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 40, SevenDayRemaining: 70}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Work wins because 90% 5h > 40% 5h (5h headroom is primary, not weekly)
	if settings.Account.Value != "work" {
		t.Fatalf("Account.Value = %q, want %q (work has more 5h headroom)", settings.Account.Value, "work")
	}
	if settings.Account.Source != SourceHeuristic {
		t.Fatalf("Account.Source = %q, want %q", settings.Account.Source, SourceHeuristic)
	}
}

func TestResolve_AccountHeuristic_FiveHourPrimary(t *testing.T) {
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			// Same weekly headroom, but personal has more 5h headroom
			return &account.CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 60}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 80, SevenDayRemaining: 60}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Personal wins on 5h headroom (80% > 50%, both default 1x tier)
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (5h headroom primary)", settings.Account.Value, "personal")
	}
}

func TestResolve_AccountHeuristic_PersonalWinsWhenMoreWeeklyHeadroom(t *testing.T) {
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			return &account.CapacityInfo{FiveHourRemaining: 15, SevenDayRemaining: 72}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 95, SevenDayRemaining: 88}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Personal wins: 95 5h > 15 5h (tier-weighted, both default 1x)
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (personal has more 5h headroom)", settings.Account.Value, "personal")
	}
	if settings.Account.Source != SourceHeuristic {
		t.Fatalf("Account.Source = %q, want %q", settings.Account.Source, SourceHeuristic)
	}
	if !strings.Contains(settings.Account.Detail, "5h-headroom") {
		t.Fatalf("Account.Detail = %q, want contains '5h-headroom'", settings.Account.Detail)
	}
}

func TestResolve_AccountHeuristic_BothLowUsesHigherWeeklyHeadroom(t *testing.T) {
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			return &account.CapacityInfo{FiveHourRemaining: 10, SevenDayRemaining: 5}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 8, SevenDayRemaining: 3}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Work wins: 10 5h > 8 5h (even when both are low, tier-weighted default 1x)
	if settings.Account.Value != "work" {
		t.Fatalf("Account.Value = %q, want %q (work has more 5h headroom even when both low)", settings.Account.Value, "work")
	}
	if settings.Account.Source != SourceHeuristic {
		t.Fatalf("Account.Source = %q, want %q", settings.Account.Source, SourceHeuristic)
	}
	if !strings.Contains(settings.Account.Detail, "5h-headroom") {
		t.Fatalf("Account.Detail = %q, want contains '5h-headroom'", settings.Account.Detail)
	}
}

func TestResolve_AccountHeuristic_CapacityFetchFailsUsesDefault(t *testing.T) {
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	// CapacityFetcher returns nil (fetch failed)
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Fail-open: first alphabetical account when all capacity unknown
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (fail-open to first alphabetical)", settings.Account.Value, "personal")
	}
	if settings.Account.Source != SourceHeuristic {
		t.Fatalf("Account.Source = %q, want %q", settings.Account.Source, SourceHeuristic)
	}
	if !strings.Contains(settings.Account.Detail, "all-capacity-unknown") {
		t.Fatalf("Account.Detail = %q, want contains 'all-capacity-unknown'", settings.Account.Detail)
	}
}

func TestResolve_AccountHeuristic_CLIOverridesHeuristic(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Account = "personal"
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		// Work is healthy, but CLI explicitly chose personal
		return &account.CapacityInfo{FiveHourRemaining: 87, SevenDayRemaining: 72}
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (CLI overrides heuristic)", settings.Account.Value, "personal")
	}
	if settings.Account.Source != SourceCLI {
		t.Fatalf("Account.Source = %q, want %q", settings.Account.Source, SourceCLI)
	}
}

func TestResolve_AccountHeuristic_NoCapacityFetcherUsesDefault(t *testing.T) {
	// When no CapacityFetcher is set, fall back to default behavior (no heuristic)
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work": {role: "primary", configDir: "~/.claude"},
		},
	}
	// No CapacityFetcher set

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Account.Value != "work" {
		t.Fatalf("Account.Value = %q, want %q", settings.Account.Value, "work")
	}
	if settings.Account.Source != SourceDefault {
		t.Fatalf("Account.Source = %q, want %q (no capacity fetcher = default)", settings.Account.Source, SourceDefault)
	}
}

func TestResolve_AccountHeuristic_PersonalWinsWhenWorkWeeklyLow(t *testing.T) {
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			return &account.CapacityInfo{FiveHourRemaining: 80, SevenDayRemaining: 15}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 95, SevenDayRemaining: 88}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Personal wins: 95% 5h > 80% 5h (tier-weighted, both default 1x)
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (personal has more 5h headroom)", settings.Account.Value, "personal")
	}
}

func TestResolve_AccountHeuristic_CapacityError(t *testing.T) {
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			return &account.CapacityInfo{Error: "token refresh failed"}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 95, SevenDayRemaining: 88}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Work has error (all remaining=0), personal wins with 95% 5h headroom
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (personal wins when work has error)", settings.Account.Value, "personal")
	}
}

// --- Tier-weighted routing tests ---

func TestResolve_AccountHeuristic_TierWeightedWorkWins(t *testing.T) {
	// THE BUG FIX TEST: 20x tier at 30% 5h remaining has more absolute capacity
	// than 5x tier at 50% 5h remaining.
	// Old behavior: personal wins (50% > 30% raw remaining)
	// New behavior: work wins (30% * 20 = 600 > 50% * 5 = 250 absolute headroom)
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude", tier: "20x"},
			"personal": {role: "spillover", configDir: "~/.claude-personal", tier: "5x"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			return &account.CapacityInfo{FiveHourRemaining: 30, SevenDayRemaining: 14}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 43}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Work: 30% * 20 = 600 absolute 5h headroom
	// Personal: 50% * 5 = 250 absolute 5h headroom
	if settings.Account.Value != "work" {
		t.Fatalf("Account.Value = %q, want %q (work has 600 vs 250 absolute 5h headroom)", settings.Account.Value, "work")
	}
	if settings.Account.Source != SourceHeuristic {
		t.Fatalf("Account.Source = %q, want %q", settings.Account.Source, SourceHeuristic)
	}
	if !strings.Contains(settings.Account.Detail, "5h-headroom") {
		t.Fatalf("Account.Detail = %q, want contains '5h-headroom'", settings.Account.Detail)
	}
	// Verify detail includes tier multiplier
	if !strings.Contains(settings.Account.Detail, "20x") {
		t.Fatalf("Account.Detail = %q, want contains '20x' tier info", settings.Account.Detail)
	}
}

func TestResolve_AccountHeuristic_TierWeightedPersonalWins(t *testing.T) {
	// When personal has enough absolute headroom advantage, it wins despite lower tier
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude", tier: "20x"},
			"personal": {role: "spillover", configDir: "~/.claude-personal", tier: "5x"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			// Work is nearly exhausted
			return &account.CapacityInfo{FiveHourRemaining: 5, SevenDayRemaining: 86}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 92, SevenDayRemaining: 43}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Work: 5% * 20 = 100 absolute 5h headroom
	// Personal: 92% * 5 = 460 absolute 5h headroom
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (personal has 460 vs 100 absolute 5h headroom)", settings.Account.Value, "personal")
	}
}

func TestResolve_AccountHeuristic_TierDefaultsToOne(t *testing.T) {
	// When tier is not set, defaults to 1x — raw percentages used
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			return &account.CapacityInfo{FiveHourRemaining: 60, SevenDayRemaining: 70}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 40, SevenDayRemaining: 90}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// No tier set → both 1x → work wins (60*1=60 > 40*1=40 5h headroom)
	if settings.Account.Value != "work" {
		t.Fatalf("Account.Value = %q, want %q (no tier, work has more raw 5h)", settings.Account.Value, "work")
	}
}

func TestResolve_AccountHeuristic_FiveHourTieBreaksByWeekly(t *testing.T) {
	// When absolute 5h headroom is tied, weekly headroom (tier-weighted) breaks tie
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude", tier: "20x"},
			"personal": {role: "spillover", configDir: "~/.claude-personal", tier: "5x"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			// Absolute 5h: 25% * 20 = 500
			// Absolute weekly: 60% * 20 = 1200
			return &account.CapacityInfo{FiveHourRemaining: 25, SevenDayRemaining: 60}
		case "personal":
			// Absolute 5h: 100% * 5 = 500 (same!)
			// Absolute weekly: 90% * 5 = 450
			return &account.CapacityInfo{FiveHourRemaining: 100, SevenDayRemaining: 90}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// 5h tied at 500, work wins on weekly (1200 > 450)
	if settings.Account.Value != "work" {
		t.Fatalf("Account.Value = %q, want %q (5h tied, work wins on weekly)", settings.Account.Value, "work")
	}
}

func TestResolve_AccountHeuristic_WeeklyExhaustedLosesToAlternative(t *testing.T) {
	// THE BUG: work at 93% 5h / 0% weekly was picked over personal at 85% both.
	// Anthropic blocks at 100% weekly regardless of 5h headroom.
	// Fix: effective_headroom = min(fiveHourAbs, weeklyAbs) so exhausted weekly
	// zeroes the effective score.
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			// 93% 5h headroom but 0% weekly — Anthropic blocks this account
			return &account.CapacityInfo{FiveHourRemaining: 93, SevenDayRemaining: 0}
		case "personal":
			// 85% on both — healthy account
			return &account.CapacityInfo{FiveHourRemaining: 85, SevenDayRemaining: 85}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Personal must win: work's weekly is exhausted (0%), Anthropic blocks it
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (work weekly exhausted, must not be picked)", settings.Account.Value, "personal")
	}
}

func TestResolve_AccountHeuristic_WeeklyExhaustedTiered(t *testing.T) {
	// Even with high tier, 0% weekly should lose
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude", tier: "20x"},
			"personal": {role: "spillover", configDir: "~/.claude-personal", tier: "5x"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			// 20x tier, 93% 5h = 1860 absolute... but 0% weekly = 0 absolute weekly
			return &account.CapacityInfo{FiveHourRemaining: 93, SevenDayRemaining: 0}
		case "personal":
			// 5x tier, 85% both = 425 absolute each
			return &account.CapacityInfo{FiveHourRemaining: 85, SevenDayRemaining: 85}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Personal must win: work has 0% weekly, effective headroom should be 0
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (work weekly exhausted even with 20x tier)", settings.Account.Value, "personal")
	}
}

func TestResolve_AccountHeuristic_LowWeeklyReducesEffectiveHeadroom(t *testing.T) {
	// Account with low weekly (5%) should have reduced effective headroom
	// compared to account with balanced capacity
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			// High 5h but very low weekly — near exhaustion
			return &account.CapacityInfo{FiveHourRemaining: 90, SevenDayRemaining: 5}
		case "personal":
			// Moderate both
			return &account.CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 50}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Personal wins: work effective = min(90,5)=5, personal effective = min(50,50)=50
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (work has low weekly, personal is balanced)", settings.Account.Value, "personal")
	}
}

func TestResolve_AccountHeuristic_BothWeeklyExhaustedPicksHigher5h(t *testing.T) {
	// Edge case: both accounts have very low weekly. Pick the one with more effective headroom.
	input := baseResolveInput()
	input.AccountConfig = &mockAccountConfig{
		accounts: map[string]mockAccount{
			"work":     {role: "primary", configDir: "~/.claude"},
			"personal": {role: "spillover", configDir: "~/.claude-personal"},
		},
	}
	input.CapacityFetcher = func(name string) *account.CapacityInfo {
		switch name {
		case "work":
			return &account.CapacityInfo{FiveHourRemaining: 90, SevenDayRemaining: 3}
		case "personal":
			return &account.CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 8}
		}
		return nil
	}

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Both low weekly: work effective = min(90,3)=3, personal effective = min(50,8)=8
	// Personal wins with higher effective headroom
	if settings.Account.Value != "personal" {
		t.Fatalf("Account.Value = %q, want %q (personal has higher effective headroom)", settings.Account.Value, "personal")
	}
}

// --- Effort resolution tests ---

func TestResolve_EffortCLIOverride(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Effort = "low"

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Effort.Value != "low" {
		t.Fatalf("Effort.Value = %q, want %q", settings.Effort.Value, "low")
	}
	if settings.Effort.Source != SourceCLI {
		t.Fatalf("Effort.Source = %q, want %q", settings.Effort.Source, SourceCLI)
	}
}

func TestResolve_EffortLightTierDefault(t *testing.T) {
	input := baseResolveInput()
	input.SkillName = "feature-impl" // light tier skill
	input.CLI.Light = true

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Effort.Value != EffortMedium {
		t.Fatalf("Effort.Value = %q, want %q for light tier", settings.Effort.Value, EffortMedium)
	}
	if settings.Effort.Source != SourceHeuristic {
		t.Fatalf("Effort.Source = %q, want %q", settings.Effort.Source, SourceHeuristic)
	}
	if settings.Effort.Detail != "tier-light" {
		t.Fatalf("Effort.Detail = %q, want %q", settings.Effort.Detail, "tier-light")
	}
}

func TestResolve_EffortFullTierDefault(t *testing.T) {
	input := baseResolveInput()
	input.SkillName = "investigation" // full tier skill
	input.CLI.Full = true

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Effort.Value != EffortHigh {
		t.Fatalf("Effort.Value = %q, want %q for full tier", settings.Effort.Value, EffortHigh)
	}
	if settings.Effort.Source != SourceHeuristic {
		t.Fatalf("Effort.Source = %q, want %q", settings.Effort.Source, SourceHeuristic)
	}
	if settings.Effort.Detail != "tier-full" {
		t.Fatalf("Effort.Detail = %q, want %q", settings.Effort.Detail, "tier-full")
	}
}

func TestResolve_EffortCLIOverridesTierHeuristic(t *testing.T) {
	input := baseResolveInput()
	input.SkillName = "feature-impl" // light tier = medium default
	input.CLI.Effort = "high"        // CLI override

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Effort.Value != "high" {
		t.Fatalf("Effort.Value = %q, want %q (CLI should override tier heuristic)", settings.Effort.Value, "high")
	}
	if settings.Effort.Source != SourceCLI {
		t.Fatalf("Effort.Source = %q, want %q", settings.Effort.Source, SourceCLI)
	}
}

func TestResolve_EffortInvalidValue(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Effort = "turbo"

	_, err := Resolve(input)
	if err == nil {
		t.Fatal("Resolve() error = nil, want validation error for invalid effort")
	}
	if !strings.Contains(err.Error(), "invalid --effort") {
		t.Fatalf("Resolve() error = %q, want 'invalid --effort' message", err)
	}
}

func TestResolve_EffortCaseInsensitive(t *testing.T) {
	input := baseResolveInput()
	input.CLI.Effort = "HIGH"

	settings, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.Effort.Value != "high" {
		t.Fatalf("Effort.Value = %q, want %q (should be normalized to lowercase)", settings.Effort.Value, "high")
	}
}

func TestIsValidEffort(t *testing.T) {
	tests := []struct {
		effort string
		valid  bool
	}{
		{"low", true},
		{"medium", true},
		{"high", true},
		{"LOW", true},
		{"Medium", true},
		{"HIGH", true},
		{"", false},
		{"turbo", false},
		{"max", false},
	}
	for _, tt := range tests {
		t.Run(tt.effort, func(t *testing.T) {
			if got := IsValidEffort(tt.effort); got != tt.valid {
				t.Errorf("IsValidEffort(%q) = %v, want %v", tt.effort, got, tt.valid)
			}
		})
	}
}
