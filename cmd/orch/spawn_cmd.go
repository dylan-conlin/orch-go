// Package main provides the spawn command for the orch CLI.
// This file contains:
// - spawn command definition with all flags
// - runSpawnWithSkill and runSpawnWithSkillInternal (main spawn pipeline)
//
// Related files:
// - work_cmd.go: work command, skill inference, runWork
// - spawn_dryrun.go: dry-run validation, formatting helpers
// - spawn_helpers.go: config loading, scaffolding, misc utilities
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Spawn command flags
	spawnSkill              string
	spawnIssue              string
	spawnPhases             string
	spawnBackendFlag        string // Spawn backend: claude or opencode (overrides config and auto-selection)
	spawnValidation         string
	spawnHeadless           bool   // Run headless via HTTP API (automation/scripting)
	spawnTmux               bool   // Run in tmux window (opt-in, overrides default headless)
	spawnModel              string // Model to use for standalone spawns
	spawnNoTrack            bool   // Opt-out of beads tracking
	spawnMCP                string // MCP server config (e.g., "playwright")
	spawnSkipArtifactCheck  bool   // Bypass pre-spawn kb context check
	spawnAutoInit           bool   // Auto-initialize .orch and .beads if missing
	spawnLight              bool   // Light tier spawn (skips SYNTHESIS.md requirement)
	spawnFull               bool   // Full tier spawn (requires SYNTHESIS.md)
	spawnWorkdir            string // Target project directory (defaults to current directory)
	spawnGateOnGap          bool   // Block spawn if context quality is too low
	spawnSkipGapGate        bool   // Explicitly bypass gap gating (documents conscious decision)
	spawnGapThreshold       int    // Custom gap quality threshold (default 20)
	spawnForce              bool   // Force spawn even if issue has blocking dependencies
	spawnBypassTriage       bool   // Explicitly bypass triage (documents conscious decision to spawn directly)
	spawnDesignWorkspace    string // Design workspace name for ui-design-session → feature-impl handoff
	spawnAccount            string // Account name for Claude CLI spawns (overrides auto-selection)
	spawnVerifyLevel        string // Verification level override (V0-V3)
	spawnReviewTier         string // Review tier override (auto/scan/review/deep)
	spawnScope              string // Session scope: small, medium, large
	spawnEffort             string // Claude CLI effort level: low, medium, high
	spawnMaxTurns           int    // Max agentic turns for Claude CLI (0 = unlimited)
	spawnReason             string // Reason for override flag usage (--bypass-triage, --no-track)
	spawnSettings           string // Path to settings.json for Claude CLI (worker hook isolation)
	spawnIntentType         string // Orchestrator's declared outcome type (experience, produce, compare, etc.)
	spawnDryRun             bool   // Show spawn plan without executing
	spawnExplore            bool   // Exploration mode: decompose → parallelize → judge → synthesize
	spawnExploreBreadth     int    // Max parallel subproblem workers (default 3)
	spawnExploreDepth       int    // Max iteration depth for judge-triggered re-exploration (default 1)
	spawnExploreJudgeModel  string // Model for judge agent (cross-model judging experiment)
	spawnModeSet            bool   // Tracks whether --mode was explicitly set
	spawnValidationSet      bool   // Tracks whether --validation was explicitly set
)

// SpawnInput, SpawnContext, GapCheckResult, and headlessSpawnResult types
// have been moved to pkg/orch/extraction.go to reduce file size.

var spawnCmd = &cobra.Command{
	Use:   "spawn [skill] [task]",
	Short: "Spawn a new agent with skill context (default: headless)",
	Long: `Spawn a new OpenCode session with skill context.

IMPORTANT: Manual spawn requires --bypass-triage flag.
The default workflow is: create issues with triage:ready label → daemon auto-spawns.
Manual spawning is for exceptions only (urgent single items, complex context needed).

To proceed with manual spawn, you must acknowledge this with --bypass-triage.
This creates friction to encourage the preferred daemon-driven workflow.

Backend Modes (--backend):
  claude:   Uses Claude Code CLI in tmux (Max subscription, unlimited Opus) (default)
  opencode: Uses OpenCode HTTP API

  Config can set default mode (orch config set spawn_mode claude|opencode).
  The --backend flag overrides the config setting for this spawn only.

Spawn Modes:
  Default (headless): Spawns via HTTP API - no TUI, automation-friendly, returns immediately
  --tmux:             Spawns in a tmux window - visible, interruptible, opt-in

Spawn Tiers:
  --light: Skip SYNTHESIS.md requirement (for code-focused work)
  --full:  Require SYNTHESIS.md for knowledge externalization

  Default tier is determined by skill:
    Full tier (require SYNTHESIS.md): investigation, architect, research,
      codebase-audit, design-session, systematic-debugging
    Light tier (skip SYNTHESIS.md): feature-impl, reliability-testing, issue-creation

Gap Gating (Gate Over Remind):
  --gate-on-gap:      Block spawn if context quality is too low (score < 20)
  --skip-gap-gate:    Explicitly bypass gating (documents conscious decision)
  --gap-threshold N:  Custom quality threshold (default 20)

  When gating is enabled and context quality is below threshold, spawn is blocked
  with a prominent message explaining the gap and how to fix it. This enforces
  the principle: 'gaps should be harder to ignore than to fix'.

Dependency Checking (--issue spawns only):
  When spawning with --issue, orch checks if the issue has blocking dependencies.
  If any dependent issues are still open, the spawn is blocked with an error
  showing which issues are blocking. Use --force to override this check.

  Example error:
    Error: orch-go-xyz is blocked by orch-go-abc (open)
    Use --force to override

Auto-Initialization:
  Use --auto-init to automatically run 'orch init' if .orch/ or .beads/ are missing.
  This is useful for spawning in new projects without prior setup.

Model aliases: opus, sonnet, haiku (Anthropic), flash, pro (Google)
Full format: provider/model (e.g., anthropic/claude-opus-4-5-20251101)

Examples:
  # Preferred workflow: create issue and let daemon spawn
  bd create "investigate auth" --type investigation -l triage:ready
  orch daemon run  # Daemon picks up triage:ready issues

  # Manual spawn (requires --bypass-triage)
  orch spawn --bypass-triage investigation "explore the codebase"
  orch spawn --bypass-triage feature-impl "add feature" --phases implementation,validation
  orch spawn --bypass-triage --issue proj-123 feature-impl "implement the feature"

  # Tmux mode (opt-in) - visible, interruptible
  orch spawn --bypass-triage --tmux investigation "explore codebase"

  # Gap gating - block spawn on poor context quality
  orch spawn --bypass-triage --gate-on-gap investigation "important task"

  # Other options
  orch spawn --bypass-triage --model opus investigation "analyze code"
  orch spawn --bypass-triage --light investigation "exploratory work"
  orch spawn --bypass-triage --mcp playwright feature-impl "add UI feature"  # injects playwright-cli context
  orch spawn --bypass-triage --workdir ~/other-project investigation "task"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		task := strings.Join(args[1:], " ")
		spawnModeSet = cmd.Flags().Changed("mode")
		spawnValidationSet = cmd.Flags().Changed("validation")

		if spawnDryRun {
			return runSpawnDryRun(serverURL, skillName, task)
		}
		return runSpawnWithSkill(serverURL, skillName, task, false, spawnHeadless, spawnTmux, false)
	},
}

func init() {
	spawnCmd.Flags().StringVar(&spawnIssue, "issue", "", "Beads issue ID for tracking")
	spawnCmd.Flags().StringVar(&spawnPhases, "phases", "", "Feature-impl phases (e.g., implementation,validation)")
	orch.RegisterModeFlag(spawnCmd)
	spawnCmd.Flags().StringVar(&spawnBackendFlag, "backend", "", "Spawn backend: claude (tmux + Claude CLI) or opencode (HTTP API). Overrides config and auto-selection.")
	spawnCmd.Flags().StringVar(&spawnValidation, "validation", "tests", "Validation level: none, tests, smoke-test")
	spawnCmd.Flags().BoolVar(&spawnHeadless, "headless", false, "Run headless via HTTP API (default behavior, flag is redundant)")
	spawnCmd.Flags().BoolVar(&spawnTmux, "tmux", false, "Run in tmux window (opt-in for visual monitoring)")
	spawnCmd.Flags().StringVar(&spawnModel, "model", "", "Model alias (opus, sonnet, haiku, flash, pro) or provider/model format")
	spawnCmd.Flags().BoolVar(&spawnNoTrack, "no-track", false, "Deprecated: creates lightweight beads issue instead (use --light)")
	spawnCmd.Flags().StringVar(&spawnMCP, "mcp", "", "MCP server preset (e.g., 'playwright' for Playwright MCP server). Default browser path is playwright-cli via needs:playwright label")
	spawnCmd.Flags().BoolVar(&spawnSkipArtifactCheck, "skip-artifact-check", false, "Bypass pre-spawn kb context check")
	spawnCmd.Flags().BoolVar(&spawnAutoInit, "auto-init", false, "Auto-initialize .orch and .beads if missing")
	spawnCmd.Flags().BoolVar(&spawnLight, "light", false, "Light tier spawn (skips SYNTHESIS.md requirement on completion)")
	spawnCmd.Flags().BoolVar(&spawnFull, "full", false, "Full tier spawn (requires SYNTHESIS.md for knowledge externalization)")
	spawnCmd.Flags().StringVar(&spawnWorkdir, "workdir", "", "Target project directory (defaults to current directory)")
	spawnCmd.Flags().BoolVar(&spawnGateOnGap, "gate-on-gap", false, "Block spawn if context quality is too low (enforces Gate Over Remind)")
	spawnCmd.Flags().BoolVar(&spawnSkipGapGate, "skip-gap-gate", false, "Explicitly bypass gap gating (documents conscious decision to proceed without context)")
	spawnCmd.Flags().IntVar(&spawnGapThreshold, "gap-threshold", 0, "Custom gap quality threshold (default 20, only used with --gate-on-gap)")
	spawnCmd.Flags().BoolVar(&spawnForce, "force", false, "Force overwrite of existing workspace (allows spawning into directory with existing session files)")
	spawnCmd.Flags().BoolVar(&spawnBypassTriage, "bypass-triage", false, "Acknowledge manual spawn bypasses daemon-driven triage workflow (required for manual spawns)")
	spawnCmd.Flags().StringVar(&spawnDesignWorkspace, "design-workspace", "", "Design workspace name from ui-design-session for handoff to feature-impl (e.g., 'og-design-ready-queue-08jan')")
	spawnCmd.Flags().StringVar(&spawnScope, "scope", "", "Session scope: small, medium, large (parsed from task if not set)")
	spawnCmd.Flags().StringVar(&spawnAccount, "account", "", "Account name for Claude CLI spawns (e.g., 'work', 'personal')")
	spawnCmd.Flags().StringVar(&spawnVerifyLevel, "verify-level", "", "Verification level override (V0=acknowledge, V1=artifacts, V2=evidence, V3=behavioral)")
	spawnCmd.Flags().StringVar(&spawnReviewTier, "review-tier", "", "Review tier override (auto=minimal, scan=quick, review=full, deep=behavioral)")
	spawnCmd.Flags().StringVar(&spawnReason, "reason", "", "Reason for override flags (--bypass-triage). Min 10 chars.")
	spawnCmd.Flags().StringVar(&spawnEffort, "effort", "", "Claude CLI effort level (low, medium, high). Default: auto from skill tier.")
	spawnCmd.Flags().IntVar(&spawnMaxTurns, "max-turns", 0, "Max agentic turns for Claude CLI spawns (0 = unlimited). Prevents runaway agents.")
	spawnCmd.Flags().StringVar(&spawnSettings, "settings", "", "Path to settings.json for Claude CLI (enables worker hook isolation)")
	spawnCmd.Flags().StringVar(&spawnIntentType, "intent", "", "Declared outcome type: experience, produce, compare, investigate, fix, build, explore")
	spawnCmd.Flags().BoolVar(&spawnDryRun, "dry-run", false, "Show spawn plan without executing (validates skill loading, context generation, and resolved settings)")
	spawnCmd.Flags().BoolVar(&spawnExplore, "explore", false, "Exploration mode: decompose question into parallel subproblems, judge findings, synthesize (investigation/architect only)")
	spawnCmd.Flags().IntVar(&spawnExploreBreadth, "explore-breadth", 3, "Max parallel subproblem workers for exploration mode (default 3)")
	spawnCmd.Flags().IntVar(&spawnExploreDepth, "explore-depth", 1, "Max iteration depth for exploration mode (1=single pass, N=judge triggers up to N-1 re-explorations)")
	spawnCmd.Flags().StringVar(&spawnExploreJudgeModel, "explore-judge-model", "", "Model for exploration judge agent (cross-model judging, e.g., 'sonnet' when workers use 'opus')")
}

func runSpawnWithSkill(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool) error {
	return runSpawnWithSkillInternal(serverURL, skillName, task, inline, headless, tmux, attach, false)
}

// runSpawnWithSkillInternal is the internal implementation that supports daemon-driven spawns.
// When daemonDriven is true, the triage bypass check is skipped (issue already triaged).
// runSpawnWithSkillInternal is the main spawn pipeline following complete_cmd.go pattern.
// Extracted into helper functions for better readability and maintainability.
func runSpawnWithSkillInternal(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool, daemonDriven bool) error {
	// Validate --mode flag early (before any side effects)
	if err := orch.ValidateMode(orch.Mode); err != nil {
		return err
	}

	// Validate --verify-level flag if provided
	if spawnVerifyLevel != "" && !spawn.IsValidVerifyLevel(spawnVerifyLevel) {
		return fmt.Errorf("invalid --verify-level %q: must be V0, V1, V2, or V3", spawnVerifyLevel)
	}

	// Validate --review-tier flag if provided
	if spawnReviewTier != "" && !spawn.IsValidReviewTier(spawnReviewTier) {
		return fmt.Errorf("invalid --review-tier %q: must be auto, scan, review, or deep", spawnReviewTier)
	}

	// Validate --effort flag if provided
	if spawnEffort != "" && !spawn.IsValidEffort(spawnEffort) {
		return fmt.Errorf("invalid --effort %q: must be low, medium, or high", spawnEffort)
	}

	// Validate --reason is provided for override flags (min 10 chars)
	if !daemonDriven {
		if spawnBypassTriage && spawnReason == "" {
			return fmt.Errorf("--reason is required when using --bypass-triage (min 10 chars)")
		}
		if spawnBypassTriage && len(spawnReason) < 10 {
			return fmt.Errorf("--reason must be at least 10 characters (got %d)", len(spawnReason))
		}
	}

	// Validate --explore flag: only allowed with investigation or architect skills
	exploreParentSkill := ""
	if spawnExplore {
		allowedExploreSkills := map[string]bool{"investigation": true, "architect": true}
		if !allowedExploreSkills[skillName] {
			return fmt.Errorf("--explore is only supported with investigation or architect skills (got %q)", skillName)
		}
		if spawnExploreBreadth < 1 || spawnExploreBreadth > 10 {
			return fmt.Errorf("--explore-breadth must be between 1 and 10 (got %d)", spawnExploreBreadth)
		}
		if spawnExploreDepth < 1 || spawnExploreDepth > 5 {
			return fmt.Errorf("--explore-depth must be between 1 and 5 (got %d)", spawnExploreDepth)
		}
		// Preserve original skill, swap to exploration orchestrator
		exploreParentSkill = skillName
		skillName = "exploration-orchestrator"
		if spawnExploreDepth > 1 {
			fmt.Printf("🔭 Exploration mode: decomposing %q task into %d parallel subproblems (depth %d)\n", exploreParentSkill, spawnExploreBreadth, spawnExploreDepth)
		} else {
			fmt.Printf("🔭 Exploration mode: decomposing %q task into %d parallel subproblems\n", exploreParentSkill, spawnExploreBreadth)
		}
	}

	// Build input parameter struct
	input := &orch.SpawnInput{
		ServerURL:    serverURL,
		SkillName:    skillName,
		Task:         task,
		IssueID:      spawnIssue,
		Inline:       inline,
		Headless:     headless,
		Tmux:         tmux,
		Attach:       attach,
		DaemonDriven: daemonDriven,
	}

	// Get project directory early for pre-flight checks
	var preCheckDir string
	if spawnWorkdir != "" {
		if absPath, err := filepath.Abs(spawnWorkdir); err == nil {
			preCheckDir = absPath
		}
	} else {
		preCheckDir, _ = os.Getwd()
	}

	// Remove triage labels early to prevent daemon race condition.
	// When manually spawning with --bypass-triage on an existing issue,
	// the triage:ready/triage:approved labels make the issue visible to the
	// daemon. Remove them immediately to close the race window between
	// issue creation and SetupBeadsTracking() setting in_progress status.
	if spawnBypassTriage && spawnIssue != "" && !daemonDriven {
		verify.RemoveTriageLabels(spawnIssue, preCheckDir)
	}

	// 1. Pre-flight checks
	// Create wrapper function to adapt RunHotspotCheckForSpawn to gates.HotspotResult
	hotspotCheckFunc := func(dir, t string) (*gates.HotspotResult, error) {
		result, err := RunHotspotCheckForSpawn(dir, t)
		if err != nil || result == nil {
			return nil, err
		}
		// Extract all matched file paths for context injection
		var matchedFiles []string
		for _, h := range result.MatchedHotspots {
			matchedFiles = append(matchedFiles, h.Path)
		}
		return &gates.HotspotResult{
			HasHotspots:        result.HasHotspots,
			HasCriticalHotspot: result.HasCriticalHotspot,
			Warning:            result.Warning,
			CriticalFiles:      result.CriticalFiles,
			MatchedFiles:       matchedFiles,
		}, nil
	}
	agreementsCheckFunc := buildAgreementsChecker()
	openQuestionCheckFunc := buildOpenQuestionChecker()
	concurrencyCheckFunc := buildConcurrencyCheck()
	hotspotResult, _, _, err := orch.RunPreFlightChecks(input, preCheckDir, spawnBypassTriage, spawnReason, hotspotCheckFunc, agreementsCheckFunc, openQuestionCheckFunc, concurrencyCheckFunc)
	if err != nil {
		return err
	}

	// 2. Resolve project directory
	projectDir, projectName, err := orch.ResolveProjectDirectory(spawnWorkdir)
	if err != nil {
		return err
	}

	// Note: projectDir is threaded through all beads/verify calls below.
	// No global state mutation needed for cross-project spawns.

	// 3. Load skill and generate workspace
	skillContent, workspaceName, isOrchestrator, isMetaOrchestrator, err := orch.LoadSkillAndGenerateWorkspace(skillName, projectName, task, projectDir, spawnAutoInit, spawnNoTrack, ensureOrchScaffolding)
	if err != nil {
		return err
	}

	// 4. Setup beads tracking
	beadsID, err := orch.SetupBeadsTracking(skillName, task, projectName, spawnIssue, isOrchestrator, isMetaOrchestrator, serverURL, spawnNoTrack, workspaceName, orch.CreateBeadsIssue, projectDir)
	if err != nil {
		return err
	}

	// 4a. Cross-repo auto-labeling: when --workdir targets a different project
	// and no --issue was provided, the issue was auto-created in the target project.
	// Add tier:light label and cross-repo back-reference for traceability.
	if spawnWorkdir != "" && beadsID != "" && spawnIssue == "" {
		cwd, _ := os.Getwd()
		if sourceProject := orch.DetectCrossRepo(cwd, projectDir); sourceProject != "" {
			orch.ApplyCrossRepoLabels(beadsID, sourceProject, projectDir)
			fmt.Printf("🔗 Cross-repo: created local issue %s in %s (from %s)\n", beadsID, projectName, sourceProject)
		}
	}

	// 4b. Detect cross-repo beads: when --workdir targets a different project,
	// determine which .beads/ directory owns the issue. Only inject BEADS_DIR
	// when the issue lives in CWD's beads (not the target's). Without this check,
	// daemon spawns for target-project issues get the wrong BEADS_DIR and bd
	// fails with "no issue found matching".
	var crossRepoBeadsDir string
	if spawnWorkdir != "" && beadsID != "" && spawnIssue != "" {
		cwd, _ := os.Getwd()
		crossRepoBeadsDir = orch.ResolveCrossRepoBeadsDir(beadsID, cwd, projectDir, orch.IssueExistsInProject)
	}

	// 5. Resolve spawn settings (centralized resolver)
	projectCfg, projectMeta, err := config.LoadWithMeta(projectDir)
	if err != nil {
		projectCfg = nil
		projectMeta = nil
	}
	userCfg, userMeta, warning := loadUserConfigWithMetaAndWarning()
	if warning != "" {
		fmt.Fprint(os.Stderr, warning)
		userCfg = nil
		userMeta = nil
	}
	beadsLabels := loadBeadsLabels(beadsID, projectDir)
	resolveInput := spawn.ResolveInput{
		CLI: spawn.CLISettings{
			Backend:       spawnBackendFlag,
			Model:         spawnModel,
			Mode:          orch.Mode,
			ModeSet:       spawnModeSet,
			Validation:    spawnValidation,
			ValidationSet: spawnValidationSet,
			MCP:           spawnMCP,
			Account:       spawnAccount,
			Effort:        spawnEffort,
			Light:         spawnLight,
			Full:          spawnFull,
			Headless:      input.Headless,
			Tmux:          input.Tmux,
			Inline:        input.Inline,
		},
		BeadsLabels:            beadsLabels,
		ProjectConfig:          projectCfg,
		ProjectConfigMeta:      projectMetaFromConfig(projectMeta),
		UserConfig:             userCfg,
		UserConfigMeta:         userMetaFromConfig(userMeta),
		Task:                   task,
		BeadsID:                beadsID,
		SkillName:              skillName,
		IsOrchestrator:         isOrchestrator,
		InfrastructureDetected: orch.IsInfrastructureWork(task, beadsID),
		CapacityFetcher:        buildCapacityFetcher(),
	}
	resolved, err := orch.ResolveSpawnSettings(resolveInput)
	if err != nil {
		return err
	}
	applyResolvedSpawnMode(input, resolved.Settings.SpawnMode.Value)

	// 5b. Apply progressive skill disclosure (section filtering)
	skillContent = skills.FilterSkillSections(skillContent, buildSectionFilter(spawnPhases, resolved.Settings.Mode.Value))

	// 6. Gather spawn context
	kbContext, gapAnalysis, hasInjectedModels, primaryModelPath, crossRepoModelDir, err := orch.GatherSpawnContext(skillContent, task, spawnOrientationFrame, beadsID, projectDir, workspaceName, skillName, spawnSkipArtifactCheck, spawnGateOnGap, spawnSkipGapGate, spawnGapThreshold)
	if err != nil {
		return err
	}

	// 6b. Warn orchestrator about cross-repo model situation
	if crossRepoModelDir != "" {
		fmt.Fprintf(os.Stderr, "⚠️  Cross-repo model detected: model lives in %s, agent workdir is %s\n", crossRepoModelDir, projectDir)
		fmt.Fprintf(os.Stderr, "   Agent will be instructed to create probe in model's repo, not workdir.\n")
	}

	// 7. Extract bug reproduction info
	isBug, reproSteps := orch.ExtractBugReproInfo(beadsID, spawnNoTrack || isOrchestrator || isMetaOrchestrator)

	// 8. Load design artifacts
	designMockupPath, designPromptPath, designNotes := orch.LoadDesignArtifacts(spawnDesignWorkspace, projectDir)

	// 9b. Gather prior art for overlapping work
	priorCompletions := ""
	if beadsID != "" && !spawnNoTrack {
		priorCompletions = spawn.GatherPriorArt(beadsID, projectDir, nil)
	}

	// 10. Build spawn context
	// Resolve account configDir from the resolved account name
	resolvedAccountName := resolved.Settings.Account.Value
	resolvedAccountConfigDir := account.GetConfigDir(resolvedAccountName)


	ctx := &orch.SpawnContext{
		Task:               task,
		OrientationFrame:   spawnOrientationFrame,
		IntentType:         spawnIntentType,
		SkillName:          skillName,
		ProjectDir:         projectDir,
		ProjectName:        projectName,
		WorkspaceName:      workspaceName,
		SkillContent:       skillContent,
		BeadsID:            beadsID,
		IsOrchestrator:     isOrchestrator,
		IsMetaOrchestrator: isMetaOrchestrator,
		ResolvedModel:      resolved.Model,
		ResolvedSettings:   resolved.Settings,
		KBContext:          kbContext,
		GapAnalysis:        gapAnalysis,
		HasInjectedModels:  hasInjectedModels,
		CrossRepoModelDir:  crossRepoModelDir,
		PrimaryModelPath:   primaryModelPath,
		IsBug:              isBug,
		ReproSteps:         reproSteps,
		Account:            resolvedAccountName,
		AccountConfigDir:   resolvedAccountConfigDir,
		SpawnBackend:       resolved.Settings.Backend.Value,
		Tier:               resolved.Settings.Tier.Value,
		VerifyLevel:        spawnVerifyLevel,
		ReviewTier:         spawnReviewTier,
		IssueType:          spawnIssueType,
		Scope:              spawnScope,
		ArchitectDesign:       "",
		HotspotArea:          hotspotResult != nil && hotspotResult.HasHotspots,
		HotspotFiles:         hotspotFilesFromResult(hotspotResult),
		HotspotDefectClasses: DefectClassesForHotspots(hotspotFilesFromResult(hotspotResult)),
		DesignMockupPath:   designMockupPath,
		DesignPromptPath:   designPromptPath,
		DesignNotes:        designNotes,
		BeadsDir:           crossRepoBeadsDir,
		PriorCompletions:   priorCompletions,
		MaxTurns:           spawnMaxTurns,
		Settings:           spawnSettings,
		Explore:            spawnExplore,
		ExploreBreadth:     spawnExploreBreadth,
		ExploreDepth:       spawnExploreDepth,
		ExploreParentSkill: exploreParentSkill,
		ExploreJudgeModel:  spawnExploreJudgeModel,
	}

	// 11. Build spawn config
	cfg := orch.BuildSpawnConfig(ctx, spawnPhases, resolved.Settings.Mode.Value, resolved.Settings.Validation.Value, resolved.Settings.MCP.Value, resolved.Settings.BrowserTool.Value, spawnNoTrack, spawnSkipArtifactCheck, spawnReason)

	// 12. Record which spawn gates were bypassed (for event tracking & miscalibration detection)
	if spawnBypassTriage && !daemonDriven {
		cfg.GatesBypassed = append(cfg.GatesBypassed, "triage")
	}
	// 13. Validate and write context (atomic spawn Phase 1: beads tag + workspace)
	minimalPrompt, rollback, err := orch.ValidateAndWriteContext(cfg, spawnForce)
	if err != nil {
		return err
	}

	// 14. Dispatch spawn (each backend calls atomic spawn Phase 2 after session creation)
	if err := orch.DispatchSpawn(input, cfg, minimalPrompt, beadsID, skillName, task, serverURL); err != nil {
		// Rollback Phase 1 writes (beads tag + workspace) on spawn failure
		if rollback != nil {
			rollback()
		}
		return err
	}

	// 15. Emit spawn.bypass event for direct (non-daemon) spawns
	if !daemonDriven {
		logger := events.NewLogger(events.DefaultLogPath())
		_ = logger.Log(events.Event{
			Type:      "spawn.bypass",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"beads_id": beadsID,
				"skill":    skillName,
				"reason":   spawnReason,
			},
		})
	}

	return nil
}
