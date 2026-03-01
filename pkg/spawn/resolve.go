package spawn

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// AccountInfo2 holds the routing-relevant fields of an account.
// Named AccountInfo2 to avoid collision with account.AccountInfo.
type AccountInfo2 struct {
	Role      string // "primary", "spillover", or ""
	ConfigDir string // e.g. "~/.claude-personal"
}

// AccountConfigProvider abstracts account config access for testability.
// In production, this wraps account.Config. In tests, it's a mock.
type AccountConfigProvider interface {
	GetAccounts() map[string]AccountInfo2
	GetDefault() string
}

// liveAccountConfig wraps account.Config for production use.
type liveAccountConfig struct {
	cfg *account.Config
}

func (l *liveAccountConfig) GetAccounts() map[string]AccountInfo2 {
	result := make(map[string]AccountInfo2)
	for name, acc := range l.cfg.Accounts {
		result[name] = AccountInfo2{Role: acc.Role, ConfigDir: acc.ConfigDir}
	}
	return result
}

func (l *liveAccountConfig) GetDefault() string {
	return l.cfg.Default
}

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
	Account    ResolvedSetting
	Effort     ResolvedSetting
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
	Account       string
	Effort        string // Effort level: low, medium, high
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

	// AccountConfig provides account routing data (role, config_dir).
	// When nil, resolveAccount loads from disk via account.LoadConfig().
	AccountConfig AccountConfigProvider

	// CapacityFetcher returns cached capacity for a named account.
	// When set, resolveAccount uses heuristic routing (work-first, personal-spillover).
	// When nil, resolveAccount falls back to default (primary account without capacity check).
	// The caller (daemon) typically wraps a CapacityCache.Get here.
	CapacityFetcher func(name string) *account.CapacityInfo
}

// Resolve computes the resolved spawn settings with provenance.
// Precedence: CLI flags > beads labels > project config (explicit) > user config (explicit) > heuristics > defaults.
func Resolve(input ResolveInput) (ResolvedSpawnSettings, error) {
	var result ResolvedSpawnSettings

	if input.CLI.Light && input.CLI.Full {
		return result, fmt.Errorf("cannot set both --light and --full")
	}

	if input.CLI.Effort != "" && !IsValidEffort(input.CLI.Effort) {
		return result, fmt.Errorf("invalid --effort %q: must be low, medium, or high", input.CLI.Effort)
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

	allowAnthropicOpenCode := input.UserConfig != nil && input.UserConfigMeta.AllowAnthropicOpenCode && input.UserConfig.AllowAnthropicOpenCode

	// Model-aware backend routing (primary routing logic).
	// When backend was NOT explicitly set via CLI, the model's provider determines
	// the backend. This generalizes the BugClass14 symmetric auto-resolve to work
	// for any backend source (project config, user config, heuristic, default).
	// CLI --backend remains as hard override.
	// Decision: kb-2d62ef
	if result.Backend.Source != SourceCLI {
		if required, ok := modelBackendRequirement(resolvedModel); ok && required != result.Backend.Value {
			// Skip auto-routing if user explicitly allows anthropic on opencode
			if !(resolvedModel.Provider == "anthropic" && allowAnthropicOpenCode) {
				result.Backend = ResolvedSetting{Value: required, Source: SourceDerived, Detail: "model-provider-routing"}
				result.Warnings = append(result.Warnings, fmt.Sprintf("Auto-routed backend to %s (model %s is %s provider)", required, resolvedModel.ModelID, resolvedModel.Provider))
			}
		}
	}

	// When backend IS from CLI and is claude, but model is non-anthropic from
	// a lower precedence source, override the model to match the backend.
	// The user explicitly chose --backend claude, so the backend wins.
	if result.Backend.Source == SourceCLI && result.Backend.Value == BackendClaude &&
		resolvedModel.Provider != "anthropic" && result.Model.Source != SourceCLI {
		resolvedModel = model.DefaultModel
		result.Model = ResolvedSetting{Value: resolvedModel.Format(), Source: SourceDerived, Detail: "backend-compatibility"}
		result.Warnings = append(result.Warnings, fmt.Sprintf("Auto-resolved model to %s (claude backend requires Anthropic model)", resolvedModel.ModelID))
	}

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
	result.Account = resolveAccount(input)
	result.Effort = resolveEffort(input, result.Tier.Value)

	// When backend is claude and spawn mode is headless, override to tmux.
	// Claude backend physically requires a tmux window (SpawnClaude creates
	// tmux window + claude CLI). Headless mode uses OpenCode HTTP API which
	// is incompatible with claude backend. This is a technical requirement,
	// not a preference - headless + claude cannot work.
	// This also fixes the daemon path where orch work passes headless=true.
	if result.Backend.Value == BackendClaude && result.SpawnMode.Value == SpawnModeHeadless {
		result.SpawnMode = ResolvedSetting{Value: SpawnModeTmux, Source: SourceDerived, Detail: "claude-backend-requires-tmux"}
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

	// Default backend is now claude since the default model is Anthropic (sonnet).
	// This became mandatory when Anthropic banned subscription OAuth in third-party tools (Feb 19 2026).
	// OpenCode + Anthropic models is a dead path without allow_anthropic_opencode override.
	// Decision: kb-2d62ef
	return ResolvedSetting{Value: BackendClaude, Source: SourceDefault}, warnings, nil
}

func resolveModel(input ResolveInput, backend string, aliasMap map[string]string) (ResolvedSetting, error) {
	if input.ProjectConfig != nil {
		if backend == BackendClaude && input.ProjectConfigMeta.ClaudeModel && input.ProjectConfig.Claude.Model != "" {
			resolved := model.ResolveWithConfig(input.ProjectConfig.Claude.Model, aliasMap)
			if err := validateModel(resolved); err == nil {
				return ResolvedSetting{Value: resolved.Format(), Source: SourceProjectConfig}, nil
			}
			// Fall through: project config model rejected by validation (e.g., flash blocked for agents).
			// Try next precedence level instead of hard-failing.
		}
		if backend == BackendOpenCode && input.ProjectConfigMeta.OpenCodeModel && input.ProjectConfig.OpenCode.Model != "" {
			resolved := model.ResolveWithConfig(input.ProjectConfig.OpenCode.Model, aliasMap)
			if err := validateModel(resolved); err == nil {
				return ResolvedSetting{Value: resolved.Format(), Source: SourceProjectConfig}, nil
			}
			// Fall through: project config model rejected by validation (e.g., flash blocked for agents).
			// Try next precedence level instead of hard-failing.
		}
	}

	if input.UserConfig != nil && input.UserConfigMeta.DefaultModel && input.UserConfig.DefaultModel != "" {
		resolved := model.ResolveWithConfig(input.UserConfig.DefaultModel, aliasMap)
		if err := validateModel(resolved); err == nil {
			return ResolvedSetting{Value: resolved.Format(), Source: SourceUserConfig}, nil
		}
		// Fall through: user config model rejected by validation.
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

// Effort level constants for Claude CLI --effort flag.
const (
	EffortLow    = "low"
	EffortMedium = "medium"
	EffortHigh   = "high"
)

// IsValidEffort returns true if the effort level is valid.
func IsValidEffort(effort string) bool {
	switch strings.ToLower(effort) {
	case EffortLow, EffortMedium, EffortHigh:
		return true
	}
	return false
}

// resolveEffort determines the effort level for Claude CLI spawns.
//
// Precedence:
//  1. CLI flag: --effort high                  → Source: cli-flag
//  2. Heuristic: tier-based default            → Source: heuristic
//  3. Default: empty (no --effort flag passed) → Source: default
//
// Tier-based heuristic (skill-tier optimization):
//   - light tier → "medium" (faster/cheaper for implementation tasks)
//   - full tier  → "high"  (maximum reasoning for investigation/architecture)
func resolveEffort(input ResolveInput, resolvedTier string) ResolvedSetting {
	if input.CLI.Effort != "" {
		return ResolvedSetting{Value: strings.ToLower(input.CLI.Effort), Source: SourceCLI}
	}

	// Tier-based heuristic: optimize effort based on task complexity
	switch resolvedTier {
	case TierLight:
		return ResolvedSetting{Value: EffortMedium, Source: SourceHeuristic, Detail: "tier-light"}
	case TierFull:
		return ResolvedSetting{Value: EffortHigh, Source: SourceHeuristic, Detail: "tier-full"}
	}

	return ResolvedSetting{Value: "", Source: SourceDefault}
}

// resolveAccount determines which account to use for Claude CLI spawns.
//
// Precedence:
//  1. CLI flag: --account work                → Source: cli-flag
//  2. Heuristic: capacity-aware routing       → Source: heuristic
//  3. Default: first primary account          → Source: default
//
// The heuristic (when CapacityFetcher is set):
//   - Check primary accounts first (sorted by name for determinism)
//   - If any primary is healthy (>20% on both limits): use it
//   - If all primaries are low: check spillover accounts
//   - If a spillover is healthy: use it
//   - If all exhausted: use first primary (still has most headroom with higher tier)
//   - If capacity fetch fails (nil): use first primary (fail-open)
func resolveAccount(input ResolveInput) ResolvedSetting {
	// CLI flag has highest precedence
	if input.CLI.Account != "" {
		return ResolvedSetting{Value: input.CLI.Account, Source: SourceCLI}
	}

	// Load account config (from injected provider or disk)
	var provider AccountConfigProvider
	if input.AccountConfig != nil {
		provider = input.AccountConfig
	} else {
		cfg, err := account.LoadConfig()
		if err != nil || len(cfg.Accounts) == 0 {
			return ResolvedSetting{Value: "", Source: SourceDefault}
		}
		provider = &liveAccountConfig{cfg: cfg}
	}

	accounts := provider.GetAccounts()
	if len(accounts) == 0 {
		return ResolvedSetting{Value: "", Source: SourceDefault}
	}

	// Categorize accounts by role
	var primaries, spillovers []string
	for name, info := range accounts {
		switch info.Role {
		case "spillover":
			spillovers = append(spillovers, name)
		default:
			// "primary" or "" (backward compat: no role = primary candidate)
			primaries = append(primaries, name)
		}
	}

	// Sort for deterministic selection
	sort.Strings(primaries)
	sort.Strings(spillovers)

	// If no primaries, use default
	if len(primaries) == 0 {
		defaultName := provider.GetDefault()
		if defaultName != "" {
			return ResolvedSetting{Value: defaultName, Source: SourceDefault, Detail: "config-default"}
		}
		return ResolvedSetting{Value: "", Source: SourceDefault}
	}

	// When no CapacityFetcher, fall back to default behavior (no heuristic)
	if input.CapacityFetcher == nil {
		return ResolvedSetting{Value: primaries[0], Source: SourceDefault, Detail: "primary-account"}
	}

	// Heuristic routing: work-first, personal-spillover
	// Check primary accounts
	for _, name := range primaries {
		capacity := input.CapacityFetcher(name)
		if capacity != nil && capacity.IsHealthy() {
			detail := fmt.Sprintf("primary-healthy-5h:%.0f%%-7d:%.0f%%",
				capacity.FiveHourRemaining, capacity.SevenDayRemaining)
			return ResolvedSetting{Value: name, Source: SourceHeuristic, Detail: detail}
		}
	}

	// All primaries are low/errored — check spillover accounts
	for _, name := range spillovers {
		capacity := input.CapacityFetcher(name)
		if capacity != nil && capacity.IsHealthy() {
			detail := fmt.Sprintf("spillover-activated-5h:%.0f%%-7d:%.0f%%",
				capacity.FiveHourRemaining, capacity.SevenDayRemaining)
			return ResolvedSetting{Value: name, Source: SourceHeuristic, Detail: detail}
		}
	}

	// All exhausted: use first primary (highest tier, most headroom)
	return ResolvedSetting{Value: primaries[0], Source: SourceHeuristic, Detail: "all-exhausted-using-primary"}
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
	// Non-Anthropic providers require OpenCode backend (Claude CLI can't run them)
	if resolvedModel.Provider == "openai" || resolvedModel.Provider == "google" || resolvedModel.Provider == "deepseek" {
		return BackendOpenCode, true
	}
	// Anthropic models route to Claude CLI backend by default
	// This became mandatory when Anthropic banned subscription OAuth in third-party tools (Feb 19 2026)
	// OpenCode + Anthropic models = dead path unless allow_anthropic_opencode override is set
	if resolvedModel.Provider == "anthropic" {
		return BackendClaude, true
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
