// Package orch provides orchestration-level utilities for agent spawn management.
// This file contains the spawn pipeline step functions that compose the spawn flow.
// Domain-specific logic has been extracted to dedicated files:
//   - spawn_types.go: Shared type definitions
//   - spawn_inference.go: Skill/tier/model inference
//   - spawn_preflight.go: Pre-flight validation gates
//   - spawn_kb_context.go: KB context + gap analysis
//   - spawn_backend.go: Backend routing + infra detection
//   - spawn_beads.go: Beads issue lifecycle
//   - spawn_design.go: Design artifact reading
package orch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// sessionScopeRegex moved to pkg/spawn.regexSessionScope (canonical location)

// CheckAndAutoSwitchAccount checks if the current account is over usage thresholds
// and automatically switches to a better account if available.
// Returns nil if no switch was needed or switch succeeded.
// Logs the switch action if one occurs.
func CheckAndAutoSwitchAccount() error {
	// Get thresholds from environment or use defaults
	thresholds := account.DefaultAutoSwitchThresholds()

	// Allow override via environment variables
	if envVal := os.Getenv("ORCH_AUTO_SWITCH_5H_THRESHOLD"); envVal != "" {
		if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
			thresholds.FiveHourThreshold = val
		}
	}
	if envVal := os.Getenv("ORCH_AUTO_SWITCH_WEEKLY_THRESHOLD"); envVal != "" {
		if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
			thresholds.WeeklyThreshold = val
		}
	}
	if envVal := os.Getenv("ORCH_AUTO_SWITCH_MIN_DELTA"); envVal != "" {
		if val, err := strconv.ParseFloat(envVal, 64); err == nil && val >= 0 {
			thresholds.MinHeadroomDelta = val
		}
	}

	// Check if auto-switch is explicitly disabled
	if os.Getenv("ORCH_AUTO_SWITCH_DISABLED") == "1" || os.Getenv("ORCH_AUTO_SWITCH_DISABLED") == "true" {
		return nil
	}

	result, err := account.AutoSwitchIfNeeded(thresholds)
	if err != nil {
		// Log warning but don't block spawn - continue with current account
		fmt.Fprintf(os.Stderr, "Warning: auto-switch check failed: %v\n", err)

		// Check if the underlying error is a TokenRefreshError and provide guidance
		var tokenErr *account.TokenRefreshError
		if errors.As(err, &tokenErr) {
			fmt.Fprintf(os.Stderr, "  → %s\n", tokenErr.ActionableGuidance())
		}
		return nil
	}

	if result.Switched {
		// Log the switch
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "account.auto_switched",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"from_account":     result.FromAccount,
				"to_account":       result.ToAccount,
				"reason":           result.Reason,
				"from_5h_used":     result.FromCapacity.FiveHourUsed,
				"from_weekly_used": result.FromCapacity.SevenDayUsed,
				"to_5h_used":       result.ToCapacity.FiveHourUsed,
				"to_weekly_used":   result.ToCapacity.SevenDayUsed,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log account switch: %v\n", err)
		}

		fmt.Printf("🔄 Auto-switched account: %s\n", result.Reason)
	}

	return nil
}

// ResolveProjectDirectory determines the project directory and name.
// Uses workdir if provided, otherwise current working directory.
func ResolveProjectDirectory(workdir string) (projectDir, projectName string, err error) {
	if workdir != "" {
		// User specified target directory via --workdir
		projectDir, err = filepath.Abs(workdir)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		// Verify directory exists
		if stat, err := os.Stat(projectDir); err != nil {
			return "", "", fmt.Errorf("workdir does not exist: %s", projectDir)
		} else if !stat.IsDir() {
			return "", "", fmt.Errorf("workdir is not a directory: %s", projectDir)
		}
	} else {
		// Default: use current working directory
		projectDir, err = os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Get project name from directory
	projectName = filepath.Base(projectDir)
	return projectDir, projectName, nil
}

// LoadSkillAndGenerateWorkspace loads skill content and generates workspace name.
// The ensureScaffoldingFunc is called to check/initialize scaffolding (passed from cmd/orch).
func LoadSkillAndGenerateWorkspace(skillName, projectName, task, projectDir string, autoInit, noTrack bool, ensureScaffoldingFunc func(string, bool, bool) error) (
	skillContent, workspaceName string,
	isOrchestrator, isMetaOrchestrator bool,
	err error) {

	// Check and optionally auto-initialize scaffolding
	if ensureScaffoldingFunc != nil {
		if err := ensureScaffoldingFunc(projectDir, autoInit, noTrack); err != nil {
			return "", "", false, false, err
		}
	}

	// Load skill content with dependencies (e.g., worker-base patterns)
	loader := skills.DefaultLoader()

	// First load raw skill content (without dependencies) to detect skill type
	// This is needed because LoadSkillWithDependencies puts dependency content first,
	// which means the main skill's frontmatter isn't at the start of the combined content
	rawSkillContent, err := loader.LoadSkillContent(skillName)
	if err == nil {
		if metadata, err := skills.ParseSkillMetadata(rawSkillContent); err == nil {
			isOrchestrator = metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"
		}
	}
	// Detect meta-orchestrator by skill name (not skill-type)
	// This enables tiered context templates for different orchestrator levels
	if skillName == "meta-orchestrator" {
		isMetaOrchestrator = true
	}

	// Generate workspace name
	// Meta-orchestrators use "meta-" prefix instead of project prefix for visual distinction
	// Orchestrators use "orch" skill prefix instead of "work" for visual distinction from workers
	workspaceName = spawn.GenerateWorkspaceName(projectName, skillName, task, spawn.WorkspaceNameOptions{
		IsMetaOrchestrator: isMetaOrchestrator,
		IsOrchestrator:     isOrchestrator,
	})

	// Now load full skill content with dependencies for the actual spawn
	skillContent, err = loader.LoadSkillWithDependencies(skillName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load skill '%s': %v\n", skillName, err)
		skillContent = "" // Continue without skill content
	}

	return skillContent, workspaceName, isOrchestrator, isMetaOrchestrator, nil
}

// ResolveAndValidateModel resolves model aliases and validates model choice.
// Returns error if flash model is requested (unsupported).
func ResolveAndValidateModel(modelFlag string) (model.ModelSpec, error) {
	// Load user config for custom model aliases
	cfg, _ := userconfig.Load()
	var configModels map[string]string
	if cfg != nil {
		configModels = cfg.Models
	}

	// If no model flag specified, check config default_model before hardcoded default
	effectiveSpec := modelFlag
	if effectiveSpec == "" && cfg != nil && cfg.DefaultModel != "" {
		effectiveSpec = cfg.DefaultModel
	}

	// Resolve model - convert aliases to full format
	// Config aliases take precedence over built-in aliases
	resolvedModel := model.ResolveWithConfig(effectiveSpec, configModels)

	// Validate flash model - TPM rate limits make it unusable
	if resolvedModel.Provider == "google" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "flash") {
		return resolvedModel, fmt.Errorf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│  🚫 Flash model not supported                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  Gemini Flash has TPM (tokens per minute) rate limits that make it           │
│  unsuitable for agent work. Use sonnet (default) or opus instead.            │
│                                                                             │
│  Available options:                                                         │
│    • --model sonnet  (default, pay-per-token via OpenCode)                  │
│    • --model opus    (Max subscription via claude CLI)                      │
└─────────────────────────────────────────────────────────────────────────────┘
`)
	}

	return resolvedModel, nil
}

// ResolveSpawnSettings resolves spawn settings using the centralized resolver and
// emits any warnings or infrastructure escape hatch messages.
func ResolveSpawnSettings(input spawn.ResolveInput) (ResolvedSpawnResult, error) {
	settings, err := spawn.Resolve(input)
	if err != nil {
		return ResolvedSpawnResult{}, err
	}

	for _, warning := range settings.Warnings {
		fmt.Fprintf(os.Stderr, "⚠️  %s\n", warning)
		if strings.Contains(warning, "infrastructure work detected") {
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	}

	if input.InfrastructureDetected && settings.Backend.Source == spawn.SourceHeuristic && settings.Backend.Detail == "infra-escape-hatch" {
		fmt.Println("🔧 Infrastructure work detected - auto-applying escape hatch (--backend claude --tmux)")
		fmt.Println("   This ensures the agent survives OpenCode server restarts.")

		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "spawn.infrastructure_detected",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"task":     input.Task,
				"beads_id": input.BeadsID,
				"skill":    input.SkillName,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log infrastructure detection: %v\n", err)
		}
	}

	resolvedModel := model.ResolveWithConfig(settings.Model.Value, nil)
	return ResolvedSpawnResult{Settings: settings, Model: resolvedModel}, nil
}

// ExtractBugReproInfo extracts reproduction steps if the issue is a bug.
// Returns isBug flag and reproduction steps.
func ExtractBugReproInfo(beadsID string, noTrack bool) (isBug bool, reproSteps string) {
	if noTrack || beadsID == "" {
		return false, ""
	}

	if reproResult, err := verify.GetReproForCompletion(beadsID); err == nil && reproResult != nil {
		isBug = reproResult.IsBug
		reproSteps = reproResult.Repro
		if isBug && reproSteps != "" {
			fmt.Printf("🐛 Bug issue detected - reproduction steps included in context\n")
		}
	}
	return isBug, reproSteps
}

// BuildUsageInfo converts rate limit check result to UsageInfo struct.
// Returns nil if no usage check result available.
func BuildUsageInfo(usageCheckResult *gates.UsageCheckResult) *spawn.UsageInfo {
	if usageCheckResult == nil || usageCheckResult.CapacityInfo == nil {
		return nil
	}

	return &spawn.UsageInfo{
		FiveHourUsed: usageCheckResult.CapacityInfo.FiveHourUsed,
		SevenDayUsed: usageCheckResult.CapacityInfo.SevenDayUsed,
		AccountEmail: usageCheckResult.CapacityInfo.Email,
		AutoSwitched: usageCheckResult.Switched,
		SwitchReason: usageCheckResult.SwitchReason,
	}
}

// BuildSpawnConfig constructs the spawn.Config from SpawnContext.
func BuildSpawnConfig(ctx *SpawnContext, phases, mode, validation, mcp string, noTrack, skipArtifactCheck bool, noTrackReason string) *spawn.Config {
	// Infer verify level if not explicitly set
	verifyLevel := ctx.VerifyLevel
	if verifyLevel == "" {
		issueType := ""
		if ctx.IsBug {
			issueType = "bug"
		}
		verifyLevel = spawn.DefaultVerifyLevel(ctx.SkillName, issueType)

		// Apply tier-based capping to inferred levels only.
		// Explicit --verify-level overrides are respected as-is.
		verifyLevel = spawn.VerifyLevelForTier(ctx.Tier, verifyLevel)
	}

	return &spawn.Config{
		Task:               ctx.Task,
		OrientationFrame:   ctx.OrientationFrame,
		SkillName:          ctx.SkillName,
		Project:            ctx.ProjectName,
		ProjectDir:         ctx.ProjectDir,
		WorkspaceName:      ctx.WorkspaceName,
		SkillContent:       ctx.SkillContent,
		BeadsID:            ctx.BeadsID,
		Phases:             phases,
		Mode:               mode,
		Validation:         validation,
		Model:              ctx.ResolvedModel.Format(),
		ResolvedSettings:   ctx.ResolvedSettings,
		MCP:                mcp,
		Tier:               ctx.Tier,
		VerifyLevel:        verifyLevel,
		Scope:              ctx.Scope,
		NoTrack:            noTrack || ctx.IsOrchestrator || ctx.IsMetaOrchestrator,
		NoTrackReason:      noTrackReason,
		SkipArtifactCheck:  skipArtifactCheck,
		KBContext:          ctx.KBContext,
		HasInjectedModels:  ctx.HasInjectedModels,
		PrimaryModelPath:   ctx.PrimaryModelPath,
		CrossRepoModelDir:  ctx.CrossRepoModelDir,
		IncludeServers:     spawn.DefaultIncludeServersForSkill(ctx.SkillName),
		GapAnalysis:        ctx.GapAnalysis,
		IsBug:              ctx.IsBug,
		ReproSteps:         ctx.ReproSteps,
		ReworkFeedback:     ctx.ReworkFeedback,
		ReworkNumber:       ctx.ReworkNumber,
		PriorSynthesis:     ctx.PriorSynthesis,
		PriorWorkspace:     ctx.PriorWorkspace,
		IsOrchestrator:     ctx.IsOrchestrator,
		IsMetaOrchestrator: ctx.IsMetaOrchestrator,
		UsageInfo:          ctx.UsageInfo,
		Account:            ctx.Account,
		AccountConfigDir:   ctx.AccountConfigDir,
		Effort:             ctx.ResolvedSettings.Effort.Value,
		SpawnMode:          ctx.SpawnBackend,
		HotspotArea:        ctx.HotspotArea,
		HotspotFiles:       ctx.HotspotFiles,
		DesignWorkspace:    "", // Will be set by caller if needed
		DesignMockupPath:   ctx.DesignMockupPath,
		DesignPromptPath:   ctx.DesignPromptPath,
		DesignNotes:        ctx.DesignNotes,
		BeadsDir:           ctx.BeadsDir,
		PriorCompletions:   ctx.PriorCompletions,
		MaxTurns:           ctx.MaxTurns,
		Settings:           ctx.Settings,
	}
}

// ValidateAndWriteContext validates context size, writes workspace via atomic spawn Phase 1, and generates prompt.
// Returns minimal prompt, rollback function (for undoing Phase 1 on spawn failure), or error if validation fails.
// The rollback function should be called if session creation fails to undo beads tagging and workspace writes.
func ValidateAndWriteContext(cfg *spawn.Config, force bool) (minimalPrompt string, rollback func(), err error) {
	// Pre-spawn token estimation and validation
	if err := spawn.ValidateContextSize(cfg); err != nil {
		return "", nil, fmt.Errorf("pre-spawn validation failed: %w", err)
	}

	// Warn about large contexts (but don't block)
	if shouldWarn, warning := spawn.ShouldWarnAboutSize(cfg); shouldWarn {
		fmt.Fprintf(os.Stderr, "%s", warning)
	}

	// Warn if task text references a different beads ID than the tracking issue
	if warning := spawn.ValidateBeadsIDConsistency(cfg.Task, cfg.BeadsID); warning != "" {
		fmt.Fprintf(os.Stderr, "%s\n", warning)
	}

	// Check for existing workspace before writing context
	// This prevents accidentally overwriting SESSION_HANDOFF.md from completed sessions
	// Note: With unique suffixes in workspace names (since Jan 2026), collisions are rare
	// but this provides an extra safety net and meaningful error messages
	if err := checkWorkspaceExists(cfg.WorkspacePath(), force); err != nil {
		return "", nil, err
	}

	// Atomic spawn Phase 1: tag beads with orch:agent + write workspace (SPAWN_CONTEXT.md, manifest, dotfiles)
	// Returns rollback function that undoes all Phase 1 writes on spawn failure.
	atomicOpts := &spawn.AtomicSpawnOpts{
		Config:  cfg,
		BeadsID: cfg.BeadsID,
		NoTrack: cfg.NoTrack,
	}
	rollback, atomicErr := spawn.AtomicSpawnPhase1(atomicOpts)
	if atomicErr != nil {
		return "", nil, fmt.Errorf("failed to write spawn context: %w", atomicErr)
	}

	// Record orientation frame in beads comments at spawn time.
	// Use OrientationFrame (issue description) if available for richer context;
	// fall back to Task (issue title) for manual spawns without separate framing.
	// Skip writing if a FRAME comment already exists (e.g., added by orchestrator before spawn).
	if !cfg.NoTrack && !cfg.IsOrchestrator && !cfg.IsMetaOrchestrator && cfg.BeadsID != "" {
		existingFrame := spawn.ExtractFrameFromBeadsComments(cfg.BeadsID)
		if existingFrame == "" {
			// No existing FRAME — write one from OrientationFrame or Task
			frame := strings.TrimSpace(cfg.OrientationFrame)
			if frame == "" {
				frame = strings.TrimSpace(cfg.Task)
			}
			if frame != "" {
				if err := addBeadsComment(cfg.BeadsID, fmt.Sprintf("FRAME: %s", frame)); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to add frame comment: %v\n", err)
				}
			}
		}
	}

	// Record spawn in session (if session is active)
	if sessionStore, err := session.New(""); err == nil {
		if err := sessionStore.RecordSpawn(cfg.BeadsID, cfg.SkillName, cfg.Task, cfg.ProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record spawn in session: %v\n", err)
		}
	}

	// Generate minimal prompt
	minimalPrompt = spawn.MinimalPrompt(cfg)
	return minimalPrompt, rollback, nil
}

// (ensureOrchScaffolding moved back to cmd/orch/spawn_cmd.go to avoid circular dependencies)

// dirExists returns true if the path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// checkWorkspaceExists verifies if a workspace already exists and has content.
// Returns an error if the workspace contains SPAWN_CONTEXT.md or SESSION_HANDOFF.md
// (indicating an active or completed session), unless force is true.
// This prevents accidental data loss from overwriting existing session artifacts.
func checkWorkspaceExists(workspacePath string, force bool) error {
	// Check if workspace directory exists
	if !dirExists(workspacePath) {
		return nil // Workspace doesn't exist, safe to create
	}

	// Check for critical files that indicate an active or completed session
	criticalFiles := []string{
		"SPAWN_CONTEXT.md",
		"SESSION_HANDOFF.md",
		"ORCHESTRATOR_CONTEXT.md",
	}

	for _, file := range criticalFiles {
		filePath := filepath.Join(workspacePath, file)
		if _, err := os.Stat(filePath); err == nil {
			if force {
				fmt.Fprintf(os.Stderr, "Warning: Overwriting existing workspace at %s (--force)\n", workspacePath)
				return nil
			}
			return fmt.Errorf("workspace already exists with %s at %s\n\nThis indicates an existing session. Use --force to overwrite or spawn with a different task", file, workspacePath)
		}
	}

	return nil // Directory exists but has no critical files, safe to reuse
}

// truncate truncates a string to a maximum length, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
