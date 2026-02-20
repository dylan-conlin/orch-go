package spawn

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// SettingSource indicates where a resolved setting came from.
type SettingSource string

const (
	SourceCLI           SettingSource = "cli-flag"
	SourceBeadsLabel    SettingSource = "beads-label"
	SourceProjectConfig SettingSource = "project-config"
	SourceUserConfig    SettingSource = "user-config"
	SourceHeuristic     SettingSource = "heuristic"
	SourceDefault       SettingSource = "default"
	SourceDerived       SettingSource = "derived"
)

const (
	BackendClaude   = "claude"
	BackendOpenCode = "opencode"

	SpawnModeHeadless = "headless"
	SpawnModeTmux     = "tmux"
	SpawnModeInline   = "inline"
)

// ResolvedSetting captures a resolved value with its source.
type ResolvedSetting struct {
	Value  string
	Source SettingSource
	Detail string
}

// ResolvedSpawnSettings contains resolved spawn config values with provenance.
type ResolvedSpawnSettings struct {
	Backend    ResolvedSetting
	Model      ResolvedSetting
	Tier       ResolvedSetting
	SpawnMode  ResolvedSetting
	MCP        ResolvedSetting
	Mode       ResolvedSetting
	Validation ResolvedSetting
	Warnings   []string
}

// CLISettings captures CLI flags with explicitness indicators.
type CLISettings struct {
	Backend       string
	Model         string
	Mode          string
	ModeSet       bool
	Validation    string
	ValidationSet bool
	MCP           string
	Light         bool
	Full          bool
	Headless      bool
	Tmux          bool
	Inline        bool
}

// ProjectConfigMeta tracks explicit project config keys.
type ProjectConfigMeta struct {
	SpawnMode     bool
	ClaudeModel   bool
	OpenCodeModel bool
	Models        bool
}

// UserConfigMeta tracks explicit user config keys.
type UserConfigMeta struct {
	Backend                bool
	DefaultModel           bool
	DefaultTier            bool
	Models                 bool
	AllowAnthropicOpenCode bool
}

// ResolveInput captures all inputs needed to resolve spawn settings.
type ResolveInput struct {
	CLI                    CLISettings
	BeadsLabels            []string
	ProjectConfig          *config.Config
	ProjectConfigMeta      ProjectConfigMeta
	UserConfig             *userconfig.Config
	UserConfigMeta         UserConfigMeta
	Task                   string
	BeadsID                string
	SkillName              string
	IsOrchestrator         bool
	InfrastructureDetected bool
}

// Resolve computes the resolved spawn settings with provenance.
// Precedence: CLI flags > beads labels > project config (explicit) > user config (explicit) > heuristics > defaults.
func Resolve(input ResolveInput) (ResolvedSpawnSettings, error) {
	var result ResolvedSpawnSettings

	if input.CLI.Light && input.CLI.Full {
		return result, fmt.Errorf("cannot set both --light and --full")
	}

	if countTrue(input.CLI.Inline, input.CLI.Tmux, input.CLI.Headless) > 1 {
		return result, fmt.Errorf("cannot set multiple spawn mode flags (--inline/--tmux/--headless)")
	}

	aliasMap := buildModelAliasMap(input.ProjectConfig, input.ProjectConfigMeta, input.UserConfig, input.UserConfigMeta)

	var resolvedModel model.ModelSpec
	modelSet := false

	if input.CLI.Model != "" {
		resolvedModel = model.ResolveWithConfig(input.CLI.Model, aliasMap)
		if err := validateModel(resolvedModel); err != nil {
			return result, err
		}
		result.Model = ResolvedSetting{Value: resolvedModel.Format(), Source: SourceCLI}
		modelSet = true
	}

	backendSetting, backendWarnings, err := resolveBackend(input, resolvedModel, modelSet)
	if err != nil {
		return result, err
	}
	result.Backend = backendSetting
	result.Warnings = append(result.Warnings, backendWarnings...)

	if !modelSet {
		modelSetting, err := resolveModel(input, result.Backend.Value, aliasMap)
		if err != nil {
			return result, err
		}
		result.Model = modelSetting
		resolvedModel = model.ResolveWithConfig(result.Model.Value, nil)
		modelSet = true
	} else {
		resolvedModel = model.ResolveWithConfig(result.Model.Value, nil)
	}

	// Auto-resolve: when backend is claude but resolved model is non-anthropic
	// (and model was not explicitly set via CLI), override to default Anthropic model.
	// This handles cross-project spawns where the project defaults to OpenAI but
	// the user explicitly chose --backend claude.
	if result.Backend.Value == BackendClaude && resolvedModel.Provider != "anthropic" && result.Model.Source != SourceCLI {
		resolvedModel = model.DefaultModel
		result.Model = ResolvedSetting{Value: resolvedModel.Format(), Source: SourceDerived, Detail: "backend-compatibility"}
		result.Warnings = append(result.Warnings, fmt.Sprintf("Auto-resolved model to %s (claude backend requires Anthropic model)", resolvedModel.ModelID))
	}

	allowAnthropicOpenCode := input.UserConfig != nil && input.UserConfigMeta.AllowAnthropicOpenCode && input.UserConfig.AllowAnthropicOpenCode
	if err := validateModelCompatibility(result.Backend.Value, resolvedModel, allowAnthropicOpenCode); err != nil {
		return result, err
	}
	if warn := warnOnNonOptimalCombo(result.Backend.Value, resolvedModel); warn != "" {
		result.Warnings = append(result.Warnings, warn)
	}

	tierSetting, err := resolveTier(input)
	if err != nil {
		return result, err
	}
	result.Tier = tierSetting

	spawnModeSetting, err := resolveSpawnMode(input)
	if err != nil {
		return result, err
	}
	result.SpawnMode = spawnModeSetting

	result.MCP = resolveMCP(input)
	result.Mode = resolveMode(input)
	result.Validation = resolveValidation(input)

	// When backend is claude and spawn mode was not explicitly set,
	// override to tmux because claude backend requires a terminal (tmux window).
	// This ensures --backend claude alone implies tmux visibility without
	// requiring the user to also pass --tmux.
	if result.Backend.Value == BackendClaude && result.SpawnMode.Source == SourceDefault {
		result.SpawnMode = ResolvedSetting{Value: SpawnModeTmux, Source: SourceDerived, Detail: "claude-backend-implies-tmux"}
	}

	return result, nil
}

func resolveBackend(input ResolveInput, resolvedModel model.ModelSpec, modelSet bool) (ResolvedSetting, []string, error) {
	warnings := []string{}

	if input.CLI.Backend != "" {
		backend := strings.ToLower(input.CLI.Backend)
		if backend != BackendClaude && backend != BackendOpenCode {
			return ResolvedSetting{}, nil, fmt.Errorf("invalid backend: %s", input.CLI.Backend)
		}
		if input.InfrastructureDetected && backend != BackendClaude {
			warnings = append(warnings, "infrastructure work detected; explicit backend overrides escape hatch")
		}
		return ResolvedSetting{Value: backend, Source: SourceCLI}, warnings, nil
	}

	if modelSet {
		if required, ok := modelBackendRequirement(resolvedModel); ok {
			if input.InfrastructureDetected && required != BackendClaude {
				warnings = append(warnings, "infrastructure work detected; model requirement overrides escape hatch")
			}
			return ResolvedSetting{Value: required, Source: SourceDerived, Detail: "model-requirement"}, warnings, nil
		}
	}

	if input.ProjectConfig != nil && input.ProjectConfigMeta.SpawnMode && input.ProjectConfig.SpawnMode != "" {
		backend := strings.ToLower(input.ProjectConfig.SpawnMode)
		if backend != BackendClaude && backend != BackendOpenCode {
			return ResolvedSetting{}, nil, fmt.Errorf("invalid project spawn_mode: %s", input.ProjectConfig.SpawnMode)
		}
		if input.InfrastructureDetected && backend != BackendClaude {
			warnings = append(warnings, "infrastructure work detected; project config backend overrides escape hatch")
		}
		return ResolvedSetting{Value: backend, Source: SourceProjectConfig}, warnings, nil
	}

	if input.UserConfig != nil && input.UserConfigMeta.Backend && input.UserConfig.Backend != "" {
		backend := strings.ToLower(input.UserConfig.Backend)
		if backend != BackendClaude && backend != BackendOpenCode {
			return ResolvedSetting{}, nil, fmt.Errorf("invalid user config backend: %s", input.UserConfig.Backend)
		}
		if input.InfrastructureDetected && backend != BackendClaude {
			warnings = append(warnings, "infrastructure work detected; user config backend overrides escape hatch")
		}
		return ResolvedSetting{Value: backend, Source: SourceUserConfig}, warnings, nil
	}

	if input.InfrastructureDetected {
		return ResolvedSetting{Value: BackendClaude, Source: SourceHeuristic, Detail: "infra-escape-hatch"}, warnings, nil
	}

	return ResolvedSetting{Value: BackendOpenCode, Source: SourceDefault}, warnings, nil
}

func resolveModel(input ResolveInput, backend string, aliasMap map[string]string) (ResolvedSetting, error) {
	if input.ProjectConfig != nil {
		if backend == BackendClaude && input.ProjectConfigMeta.ClaudeModel && input.ProjectConfig.Claude.Model != "" {
			resolved := model.ResolveWithConfig(input.ProjectConfig.Claude.Model, aliasMap)
			if err := validateModel(resolved); err != nil {
				return ResolvedSetting{}, err
			}
			return ResolvedSetting{Value: resolved.Format(), Source: SourceProjectConfig}, nil
		}
		if backend == BackendOpenCode && input.ProjectConfigMeta.OpenCodeModel && input.ProjectConfig.OpenCode.Model != "" {
			resolved := model.ResolveWithConfig(input.ProjectConfig.OpenCode.Model, aliasMap)
			if err := validateModel(resolved); err != nil {
				return ResolvedSetting{}, err
			}
			return ResolvedSetting{Value: resolved.Format(), Source: SourceProjectConfig}, nil
		}
	}

	if input.UserConfig != nil && input.UserConfigMeta.DefaultModel && input.UserConfig.DefaultModel != "" {
		resolved := model.ResolveWithConfig(input.UserConfig.DefaultModel, aliasMap)
		if err := validateModel(resolved); err != nil {
			return ResolvedSetting{}, err
		}
		return ResolvedSetting{Value: resolved.Format(), Source: SourceUserConfig}, nil
	}

	resolved := model.DefaultModel
	if err := validateModel(resolved); err != nil {
		return ResolvedSetting{}, err
	}
	return ResolvedSetting{Value: resolved.Format(), Source: SourceDefault}, nil
}

func resolveTier(input ResolveInput) (ResolvedSetting, error) {
	if input.CLI.Light {
		return ResolvedSetting{Value: TierLight, Source: SourceCLI}, nil
	}
	if input.CLI.Full {
		return ResolvedSetting{Value: TierFull, Source: SourceCLI}, nil
	}

	if input.UserConfig != nil && input.UserConfigMeta.DefaultTier && input.UserConfig.DefaultTier != "" {
		tier := strings.ToLower(input.UserConfig.DefaultTier)
		if tier != TierLight && tier != TierFull {
			return ResolvedSetting{}, fmt.Errorf("invalid user config default_tier: %s", input.UserConfig.DefaultTier)
		}
		return ResolvedSetting{Value: tier, Source: SourceUserConfig}, nil
	}

	if inferred := inferTierFromTask(input.Task); inferred != "" {
		return ResolvedSetting{Value: inferred, Source: SourceHeuristic, Detail: "task-scope"}, nil
	}

	return ResolvedSetting{Value: DefaultTierForSkill(input.SkillName), Source: SourceHeuristic, Detail: "skill-default"}, nil
}

func resolveSpawnMode(input ResolveInput) (ResolvedSetting, error) {
	if input.CLI.Inline {
		return ResolvedSetting{Value: SpawnModeInline, Source: SourceCLI}, nil
	}
	if input.CLI.Tmux {
		return ResolvedSetting{Value: SpawnModeTmux, Source: SourceCLI}, nil
	}
	if input.CLI.Headless {
		return ResolvedSetting{Value: SpawnModeHeadless, Source: SourceCLI}, nil
	}

	if input.IsOrchestrator {
		return ResolvedSetting{Value: SpawnModeTmux, Source: SourceHeuristic, Detail: "orchestrator-default"}, nil
	}

	return ResolvedSetting{Value: SpawnModeHeadless, Source: SourceDefault}, nil
}

func resolveMCP(input ResolveInput) ResolvedSetting {
	if input.CLI.MCP != "" {
		return ResolvedSetting{Value: input.CLI.MCP, Source: SourceCLI}
	}

	if value, ok := mcpFromLabels(input.BeadsLabels); ok {
		return ResolvedSetting{Value: value, Source: SourceBeadsLabel, Detail: "needs:" + value}
	}

	return ResolvedSetting{Value: "", Source: SourceDefault}
}

func resolveMode(input ResolveInput) ResolvedSetting {
	if input.CLI.ModeSet {
		return ResolvedSetting{Value: input.CLI.Mode, Source: SourceCLI}
	}
	return ResolvedSetting{Value: "tdd", Source: SourceDefault}
}

func resolveValidation(input ResolveInput) ResolvedSetting {
	if input.CLI.ValidationSet {
		return ResolvedSetting{Value: input.CLI.Validation, Source: SourceCLI}
	}
	return ResolvedSetting{Value: "tests", Source: SourceDefault}
}

func buildModelAliasMap(projectCfg *config.Config, projectMeta ProjectConfigMeta, userCfg *userconfig.Config, userMeta UserConfigMeta) map[string]string {
	aliases := map[string]string{}

	if userCfg != nil && userMeta.Models {
		for k, v := range userCfg.Models {
			aliases[strings.ToLower(k)] = v
		}
	}

	if projectCfg != nil && projectMeta.Models {
		for k, v := range projectCfg.Models {
			aliases[strings.ToLower(k)] = v
		}
	}

	if len(aliases) == 0 {
		return nil
	}
	return aliases
}

func modelBackendRequirement(resolvedModel model.ModelSpec) (string, bool) {
	if resolvedModel.Provider == "openai" || resolvedModel.Provider == "google" || resolvedModel.Provider == "deepseek" {
		return BackendOpenCode, true
	}
	return "", false
}

func validateModelCompatibility(backend string, resolvedModel model.ModelSpec, allowAnthropicOpenCode bool) error {
	if backend == BackendOpenCode && resolvedModel.Provider == "anthropic" {
		if allowAnthropicOpenCode {
			return nil
		}
		return fmt.Errorf("backend %s does not support provider %s (set allow_anthropic_opencode: true to override)", backend, resolvedModel.Provider)
	}
	if backend == BackendClaude && resolvedModel.Provider != "anthropic" {
		return fmt.Errorf("backend %s does not support provider %s", backend, resolvedModel.Provider)
	}
	return nil
}

func warnOnNonOptimalCombo(backend string, resolvedModel model.ModelSpec) string {
	if backend == BackendOpenCode && strings.Contains(strings.ToLower(resolvedModel.ModelID), "opus") {
		return "opencode backend with opus model may fail; consider --backend claude"
	}
	return ""
}

func validateModel(resolvedModel model.ModelSpec) error {
	if resolvedModel.Provider == "google" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "flash") {
		return fmt.Errorf("flash models are not supported for agent work")
	}
	return nil
}

func mcpFromLabels(labels []string) (string, bool) {
	for _, label := range labels {
		if strings.HasPrefix(label, "needs:") {
			value := strings.TrimSpace(strings.TrimPrefix(label, "needs:"))
			if value != "" {
				return value, true
			}
		}
	}
	return "", false
}

func inferTierFromTask(task string) string {
	scope := parseSessionScope(task)
	if scope != "" {
		switch scope {
		case "medium", "large", "full", "4-6h", "4-6h+", "2-4h":
			return TierFull
		}
	}

	lower := strings.ToLower(task)
	score := 0
	if containsAny(lower, []string{
		"create package",
		"new package",
		"create module",
		"new module",
		"new pkg/",
		"create pkg/",
		"new package/",
		"create package/",
	}) {
		score += 2
	}
	if containsAny(lower, []string{
		"comprehensive tests",
		"test suite",
		"integration tests",
		"unit tests",
		"tests for",
		"add tests",
	}) {
		score++
	}

	if score >= 2 {
		return TierFull
	}

	return ""
}

func parseSessionScope(task string) string {
	if task == "" {
		return ""
	}
	lowered := strings.ToLower(task)
	for _, line := range strings.Split(lowered, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "session scope:") {
			scope := strings.TrimSpace(strings.TrimPrefix(line, "session scope:"))
			if scope == "" {
				return ""
			}
			fields := strings.Fields(scope)
			if len(fields) == 0 {
				return ""
			}
			return fields[0]
		}
	}
	return ""
}

func containsAny(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func countTrue(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}
