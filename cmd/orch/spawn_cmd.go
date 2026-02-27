// Package main provides spawn and work commands for the orch CLI.
// This file contains all spawn-related functionality including:
// - spawn command with all flags and modes (headless, tmux)
// - work command for daemon-driven spawns
// - beads issue creation and tracking
// - gap analysis and context gathering
// - concurrency limiting and account switching
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
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
	spawnMaxAgents          int    // Maximum concurrent agents (0 = use default or env var)
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
	spawnBypassVerification bool   // Bypass verification gate for independent parallel work
	spawnBypassReason       string // Justification for bypassing verification gate
	spawnForceHotspot       bool   // Bypass CRITICAL hotspot blocking gate
	spawnArchitectRef       string // Architect issue reference (required with --force-hotspot)
	spawnAccount            string // Account name for Claude CLI spawns (overrides auto-selection)
	spawnVerifyLevel        string // Verification level override (V0-V3)
	spawnScope              string // Session scope: small, medium, large
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

Concurrency Limiting:
  By default, limits concurrent agents to 5. This prevents runaway agent spawning.
  Configure via --max-agents flag or ORCH_MAX_AGENTS environment variable.
  Set to 0 to disable the limit (not recommended).

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
  orch spawn --bypass-triage --no-track investigation "exploratory work"
  orch spawn --bypass-triage --mcp playwright feature-impl "add UI feature"
  orch spawn --bypass-triage --workdir ~/other-project investigation "task"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		task := strings.Join(args[1:], " ")
		spawnModeSet = cmd.Flags().Changed("mode")
		spawnValidationSet = cmd.Flags().Changed("validation")

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
	spawnCmd.Flags().BoolVar(&spawnNoTrack, "no-track", false, "Opt-out of beads issue tracking (ad-hoc work)")
	spawnCmd.Flags().StringVar(&spawnMCP, "mcp", "", "MCP server config (e.g., 'playwright' for browser automation)")
	spawnCmd.Flags().BoolVar(&spawnSkipArtifactCheck, "skip-artifact-check", false, "Bypass pre-spawn kb context check")
	spawnCmd.Flags().IntVar(&spawnMaxAgents, "max-agents", 0, "Maximum concurrent agents (default 5, 0 to disable limit, or use ORCH_MAX_AGENTS env var)")
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
	spawnCmd.Flags().BoolVar(&spawnBypassVerification, "bypass-verification", false, "Bypass verification gate for independent parallel work (requires --bypass-reason)")
	spawnCmd.Flags().StringVar(&spawnBypassReason, "bypass-reason", "", "Justification for bypassing verification gate (required with --bypass-verification)")
	spawnCmd.Flags().BoolVar(&spawnForceHotspot, "force-hotspot", false, "Bypass CRITICAL hotspot blocking gate (requires --architect-ref)")
	spawnCmd.Flags().StringVar(&spawnArchitectRef, "architect-ref", "", "Architect issue ID proving area was reviewed (required with --force-hotspot)")
	spawnCmd.Flags().StringVar(&spawnScope, "scope", "", "Session scope: small, medium, large (parsed from task if not set)")
	spawnCmd.Flags().StringVar(&spawnAccount, "account", "", "Account name for Claude CLI spawns (e.g., 'work', 'personal')")
	spawnCmd.Flags().StringVar(&spawnVerifyLevel, "verify-level", "", "Verification level override (V0=acknowledge, V1=artifacts, V2=evidence, V3=behavioral)")
}

var (
	// Work command flags
	workInline bool // Run inline (blocking) with TUI

	// spawnOrientationFrame holds separate context from the task title.
	// Set by runWork from the beads issue description, rendered as
	// ORIENTATION_FRAME: section in SPAWN_CONTEXT.md (separate from TASK:).
	spawnOrientationFrame string
)

var workCmd = &cobra.Command{
	Use:   "work [beads-id]",
	Short: "Start work on a beads issue with skill inference",
	Long: `Start work on a beads issue by inferring the skill from the issue type.

The skill is automatically determined from the issue type:
  - bug         → systematic-debugging
  - feature     → feature-impl
  - task        → feature-impl
  - investigation → investigation

The issue description becomes the task prompt for the spawned agent.

By default, spawns in a tmux window (visible, interruptible).
Use --inline to run in the current terminal (blocking with TUI).

Examples:
  orch-go work proj-123           # Start work in tmux window (default)
  orch-go work proj-123 --inline  # Start work inline (blocking TUI)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		spawnModeSet = false
		spawnValidationSet = false
		return runWork(serverURL, beadsID, workInline)
	},
}

func init() {
	workCmd.Flags().BoolVar(&workInline, "inline", false, "Run inline (blocking) with TUI")
	workCmd.Flags().StringVar(&spawnModel, "model", "", "Model alias (opus, sonnet) or provider/model format")
	workCmd.Flags().StringVar(&spawnWorkdir, "workdir", "", "Target project directory (for cross-project work)")
}

// InferSkillFromIssueType maps issue types to appropriate skills.
// Returns an error for types that cannot be spawned (e.g., epic) or unknown types.
//
// Bug handling: Defaults to "architect" (understand before fixing) rather than
// "systematic-debugging". This implements the "Premise Before Solution" principle -
// most bugs reported as vague symptoms need understanding before patching.
// Use explicit skill:systematic-debugging label for isolated bugs with clear cause.
func InferSkillFromIssueType(issueType string) (string, error) {
	switch issueType {
	case "bug":
		return "systematic-debugging", nil
	case "feature":
		return "feature-impl", nil
	case "task":
		return "feature-impl", nil
	case "investigation":
		return "investigation", nil
	case "epic":
		return "", fmt.Errorf("cannot spawn work on epic issues - epics are decomposed into sub-issues")
	case "":
		return "", fmt.Errorf("issue type is empty")
	default:
		return "", fmt.Errorf("unknown issue type: %s", issueType)
	}
}

// inferSkillFromBeadsIssue infers skill from a beads issue using labels, title, then type.
func inferSkillFromBeadsIssue(issue *beads.Issue) string {
	// Check for skill:* labels first
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "skill:") {
			return strings.TrimPrefix(label, "skill:")
		}
	}

	// Check for title patterns (e.g., synthesis issues)
	if strings.HasPrefix(issue.Title, "Synthesize ") && strings.Contains(issue.Title, " investigations") {
		return "kb-reflect"
	}

	// Fall back to type-based inference
	skill, err := InferSkillFromIssueType(issue.IssueType)
	if err != nil {
		return "feature-impl" // Default fallback
	}
	return skill
}

// inferMCPFromBeadsIssue extracts MCP server requirements from issue labels.
// Returns the MCP server name if found (e.g., "playwright" from "needs:playwright"),
// or empty string if no MCP-related label is present.
//
// This allows daemon-spawned agents to automatically get browser access when
// working on UI/CSS fixes that require visual verification.
func inferMCPFromBeadsIssue(issue *beads.Issue) string {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "needs:") {
			need := strings.TrimPrefix(label, "needs:")
			// Map needs labels to MCP servers
			switch need {
			case "playwright":
				return "playwright"
				// Future: add more mappings as needed
			}
		}
	}
	return ""
}

func loadBeadsLabels(beadsID string) []string {
	if beadsID == "" {
		return nil
	}
	socketPath, connErr := beads.FindSocketPath("")
	if connErr != nil {
		return nil
	}
	beadsClient := beads.NewClient(socketPath)
	if err := beadsClient.Connect(); err != nil {
		return nil
	}
	defer beadsClient.Close()
	beadsIssue, showErr := beadsClient.Show(beadsID)
	if showErr != nil {
		return nil
	}
	return beadsIssue.Labels
}

func projectMetaFromConfig(meta *config.ConfigMeta) spawn.ProjectConfigMeta {
	if meta == nil {
		return spawn.ProjectConfigMeta{}
	}
	return spawn.ProjectConfigMeta{
		SpawnMode:     meta.Explicit["spawn_mode"],
		ClaudeModel:   meta.ExplicitClaude["model"],
		OpenCodeModel: meta.ExplicitOpenCode["model"],
		Models:        meta.Explicit["models"],
	}
}

func userMetaFromConfig(meta *userconfig.ConfigMeta) spawn.UserConfigMeta {
	if meta == nil {
		return spawn.UserConfigMeta{}
	}
	return spawn.UserConfigMeta{
		Backend:                meta.Explicit["backend"],
		DefaultModel:           meta.Explicit["default_model"],
		DefaultTier:            meta.Explicit["default_tier"],
		Models:                 meta.Explicit["models"],
		AllowAnthropicOpenCode: meta.Explicit["allow_anthropic_opencode"],
	}
}

func formatUserConfigLoadWarning(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf(
		"Warning: failed to load user config %s: %v\n"+
			"         Using defaults; user config preferences (backend/default_model) will be ignored.\n",
		userconfig.ConfigPath(),
		err,
	)
}

func loadUserConfigAndWarning() (*userconfig.Config, string) {
	cfg, err := userconfig.Load()
	if err != nil {
		return nil, formatUserConfigLoadWarning(err)
	}
	return cfg, ""
}

func loadUserConfigWithMetaAndWarning() (*userconfig.Config, *userconfig.ConfigMeta, string) {
	cfg, meta, err := userconfig.LoadWithMeta()
	if err != nil {
		return nil, nil, formatUserConfigLoadWarning(err)
	}
	return cfg, meta, ""
}

func applyResolvedSpawnMode(input *orch.SpawnInput, spawnMode string) {
	if input == nil || input.Attach {
		return
	}
	switch spawnMode {
	case spawn.SpawnModeInline:
		input.Inline = true
		input.Headless = false
		input.Tmux = false
	case spawn.SpawnModeTmux:
		input.Tmux = true
		input.Inline = false
		input.Headless = false
	case spawn.SpawnModeHeadless:
		input.Headless = true
		input.Inline = false
		input.Tmux = false
	}
}

func runWork(serverURL, beadsID string, inline bool) error {
	// For cross-project work (--workdir set), redirect beads lookups to the target
	// project directory. Without this, verify.GetIssue() and beads.FindSocketPath("")
	// use the current directory's .beads/ database, which doesn't contain the
	// cross-project issue — causing silent spawn failures (orch-go-1230).
	if spawnWorkdir != "" {
		absWorkdir, err := filepath.Abs(spawnWorkdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir: %w", err)
		}
		prevDefaultDir := beads.DefaultDir
		beads.DefaultDir = absWorkdir
		defer func() { beads.DefaultDir = prevDefaultDir }()
	}

	// Get issue details from verify (for description)
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Infer skill and MCP from issue (labels, title pattern, then type)
	// Use beads.Issue which has Labels for full skill/MCP inference
	var skillName string
	var mcpServer string
	socketPath, connErr := beads.FindSocketPath("")
	if connErr == nil {
		beadsClient := beads.NewClient(socketPath)
		if connErr := beadsClient.Connect(); connErr == nil {
			defer beadsClient.Close()
			beadsIssue, showErr := beadsClient.Show(beadsID)
			if showErr == nil {
				skillName = inferSkillFromBeadsIssue(beadsIssue)
				mcpServer = inferMCPFromBeadsIssue(beadsIssue)
			}
		}
	}
	// Fall back to type-only inference if beads fails
	if skillName == "" {
		skillName, err = InferSkillFromIssueType(issue.IssueType)
		if err != nil {
			return fmt.Errorf("cannot work on issue %s: %w", beadsID, err)
		}
	}

	// Use issue title as the TASK (concise, drives workspace name slug).
	// Issue description goes into a separate ORIENTATION_FRAME section in SPAWN_CONTEXT.md.
	// This prevents long descriptions from polluting workspace names (e.g., "orientation-frame-...").
	task := issue.Title
	spawnOrientationFrame = issue.Description

	// Set the spawnIssue flag so runSpawnWithSkillInternal uses the existing issue
	spawnIssue = beadsID

	// NOTE: Do NOT load user config default_model into spawnModel here.
	// spawnModel maps to CLI.Model in the resolve pipeline (highest priority).
	// User config default_model is already handled at correct precedence in
	// pkg/spawn/resolve.go:resolveModel() via ResolveInput.UserConfig.

	fmt.Printf("Starting work on: %s\n", beadsID)
	fmt.Printf("  Title:  %s\n", issue.Title)
	fmt.Printf("  Type:   %s\n", issue.IssueType)
	fmt.Printf("  Skill:  %s\n", skillName)
	if spawnModel != "" {
		fmt.Printf("  Model:  %s\n", spawnModel)
	}
	if mcpServer != "" {
		fmt.Printf("  MCP:    %s\n", mcpServer)
	}

	// Work command is daemon-driven (issue already created and triaged)
	// Pass daemonDriven=true to skip triage bypass check
	return runSpawnWithSkillInternal(serverURL, skillName, task, inline, true, false, false, true)
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

	// Build input parameter struct
	input := &orch.SpawnInput{
		ServerURL:    serverURL,
		SkillName:    skillName,
		Task:         task,
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
	usageCheckResult, hotspotResult, err := orch.RunPreFlightChecks(input, preCheckDir, spawnBypassTriage, spawnBypassVerification, spawnForceHotspot, spawnArchitectRef, spawnBypassReason, spawnMaxAgents, extractBeadsIDFromTitle, hotspotCheckFunc)
	if err != nil {
		return err
	}

	// 2. Resolve project directory
	projectDir, projectName, err := orch.ResolveProjectDirectory(spawnWorkdir)
	if err != nil {
		return err
	}

	// 3. Load skill and generate workspace
	skillContent, workspaceName, isOrchestrator, isMetaOrchestrator, err := orch.LoadSkillAndGenerateWorkspace(skillName, projectName, task, projectDir, spawnAutoInit, spawnNoTrack, ensureOrchScaffolding)
	if err != nil {
		return err
	}

	// 4. Setup beads tracking
	beadsID, err := orch.SetupBeadsTracking(skillName, task, projectName, spawnIssue, isOrchestrator, isMetaOrchestrator, serverURL, spawnNoTrack, workspaceName, orch.CreateBeadsIssue)
	if err != nil {
		return err
	}

	// 4b. Detect cross-repo beads: when --workdir is set, the agent works in a
	// different directory than where the beads issue was created. The agent's bd
	// commands would fail because bd looks in .beads/ of the cwd by default.
	// Capture the source beads directory for BEADS_DIR env var injection.
	var crossRepoBeadsDir string
	if spawnWorkdir != "" && beadsID != "" {
		sourceDir := beads.DefaultDir
		if sourceDir == "" {
			sourceDir, _ = os.Getwd()
		}
		sourceBeadsDir := filepath.Join(sourceDir, ".beads")
		targetBeadsDir := filepath.Join(projectDir, ".beads")
		if sourceBeadsDir != targetBeadsDir {
			crossRepoBeadsDir = sourceBeadsDir
		}
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
	beadsLabels := loadBeadsLabels(beadsID)
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
		InfrastructureDetected: isInfrastructureWork(task, beadsID),
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
	kbContext, gapAnalysis, hasInjectedModels, primaryModelPath, crossRepoModelDir, err := orch.GatherSpawnContext(skillContent, task, beadsID, projectDir, workspaceName, skillName, spawnSkipArtifactCheck, spawnGateOnGap, spawnSkipGapGate, spawnGapThreshold)
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

	// 8. Build usage info
	usageInfo := orch.BuildUsageInfo(usageCheckResult)

	// 9. Load design artifacts
	designMockupPath, designPromptPath, designNotes := orch.LoadDesignArtifacts(spawnDesignWorkspace, projectDir)

	// 10. Build spawn context
	// Resolve account configDir from the resolved account name
	resolvedAccountName := resolved.Settings.Account.Value
	resolvedAccountConfigDir := account.GetConfigDir(resolvedAccountName)

	ctx := &orch.SpawnContext{
		Task:               task,
		OrientationFrame:   spawnOrientationFrame,
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
		UsageInfo:          usageInfo,
		Account:            resolvedAccountName,
		AccountConfigDir:   resolvedAccountConfigDir,
		SpawnBackend:       resolved.Settings.Backend.Value,
		Tier:               resolved.Settings.Tier.Value,
		VerifyLevel:        spawnVerifyLevel,
		Scope:              spawnScope,
		HotspotArea:        hotspotResult != nil && hotspotResult.HasHotspots,
		HotspotFiles:       hotspotFilesFromResult(hotspotResult),
		DesignMockupPath:   designMockupPath,
		DesignPromptPath:   designPromptPath,
		DesignNotes:        designNotes,
		BeadsDir:           crossRepoBeadsDir,
	}

	// 11. Build spawn config
	cfg := orch.BuildSpawnConfig(ctx, spawnPhases, resolved.Settings.Mode.Value, resolved.Settings.Validation.Value, resolved.Settings.MCP.Value, spawnNoTrack, spawnSkipArtifactCheck)

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
	return nil
}

// hotspotFilesFromResult extracts all matched file paths from a hotspot result.
// Returns nil if result is nil or has no matched hotspots.
func hotspotFilesFromResult(result *gates.HotspotResult) []string {
	if result == nil {
		return nil
	}
	return result.MatchedFiles
}

// dirExists returns true if the path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ensureOrchScaffolding checks for required scaffolding (.orch, .beads) and optionally auto-initializes.
// Returns nil if scaffolding exists or was successfully created.
// Returns an error with guidance if scaffolding is missing and auto-init is not enabled.
func ensureOrchScaffolding(projectDir string, autoInit bool, noTrack bool) error {
	beadsDir := filepath.Join(projectDir, ".beads")
	beadsExists := dirExists(beadsDir)

	// If beads exists or tracking is disabled, we're good
	if beadsExists || noTrack {
		return nil
	}

	// Beads is missing and tracking is enabled
	// If auto-init is enabled, run initialization
	if autoInit {
		fmt.Println("Auto-initializing orch scaffolding...")

		// Run init with appropriate flags (skip CLAUDE.md and tmuxinator for minimal init)
		result, err := initProject(projectDir, false, false, false, true, true, "", "")
		if err != nil {
			return fmt.Errorf("auto-init failed: %w", err)
		}

		// Print minimal summary
		if len(result.DirsCreated) > 0 {
			fmt.Printf("Created: %s\n", strings.Join(result.DirsCreated, ", "))
		}
		if result.BeadsInitiated {
			fmt.Println("Beads initialized (.beads/)")
		}
		if result.KBInitiated {
			fmt.Println("KB initialized (.kb/)")
		}

		return nil
	}

	// Not auto-init, provide helpful error message
	return fmt.Errorf("missing beads tracking (.beads/ not initialized)\n\nTo fix, run one of:\n  orch init           # Full initialization (recommended)\n  orch spawn --auto-init ...  # Auto-init during spawn\n  orch spawn --no-track ...   # Skip beads tracking (ad-hoc work)")
}

// dirExists returns true if the path exists and is a directory.

// Test-only wrapper functions that call pkg/orch versions.
// These exist only to support existing tests.

func formatSessionTitle(workspaceName, beadsID string) string {
	if beadsID == "" {
		return workspaceName
	}
	return fmt.Sprintf("%s [%s]", workspaceName, beadsID)
}

// ansiRegex matches ANSI escape sequences (colors, formatting, etc.)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func validateModeModelCombo(backend string, resolvedModel model.ModelSpec) error {
	if backend == "opencode" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "opus") {
		return fmt.Errorf(`Warning: opencode backend with opus model may fail (auth blocked).
  Recommendation: Use --model sonnet (default) or let auto-selection use claude backend`)
	}
	return nil
}

func determineBeadsID(projectName, skillName, task, spawnIssueFlag string, noTrack bool, createBeadsFn func(string, string, string) (string, error)) (string, error) {
	if spawnIssueFlag != "" {
		return resolveShortBeadsID(spawnIssueFlag)
	}
	if noTrack {
		return fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix()), nil
	}
	beadsID, err := createBeadsFn(projectName, skillName, task)
	if err != nil {
		return "", fmt.Errorf("failed to create beads issue: %w", err)
	}
	return beadsID, nil
}

func formatContextQualitySummary(gapAnalysis *spawn.GapAnalysis) string {
	if gapAnalysis == nil {
		return "not checked"
	}
	quality := gapAnalysis.ContextQuality
	var indicator, label string
	switch {
	case quality == 0:
		indicator = "🚨"
		label = "CRITICAL - No context"
	case quality < 20:
		indicator = "⚠️"
		label = "poor"
	case quality < 40:
		indicator = "⚠️"
		label = "limited"
	case quality < 60:
		indicator = "📊"
		label = "moderate"
	case quality < 80:
		indicator = "✓"
		label = "good"
	default:
		indicator = "✓"
		label = "excellent"
	}
	summary := fmt.Sprintf("%s %d/100 (%s)", indicator, quality, label)
	if gapAnalysis.MatchStats.TotalMatches > 0 {
		summary += fmt.Sprintf(" - %d matches", gapAnalysis.MatchStats.TotalMatches)
		if gapAnalysis.MatchStats.ConstraintCount > 0 {
			summary += fmt.Sprintf(" (%d constraints)", gapAnalysis.MatchStats.ConstraintCount)
		}
	}
	return summary
}

func addUsageInfoToEventData(eventData map[string]interface{}, usageInfo *spawn.UsageInfo) {
	if usageInfo == nil {
		return
	}
	eventData["usage_5h_used"] = usageInfo.FiveHourUsed
	eventData["usage_weekly_used"] = usageInfo.SevenDayUsed
	if usageInfo.AccountEmail != "" {
		eventData["usage_account"] = usageInfo.AccountEmail
	}
	if usageInfo.AutoSwitched {
		eventData["usage_auto_switched"] = true
		eventData["usage_switch_reason"] = usageInfo.SwitchReason
	}
}

func isInfrastructureWork(task string, beadsID string) bool {
	infrastructureKeywords := []string{
		"opencode", "orch-go", "pkg/spawn", "pkg/opencode", "pkg/verify",
		"pkg/state", "cmd/orch", "spawn_cmd.go", "serve.go", "status.go",
		"main.go", "dashboard", "agent-card", "agents.ts", "daemon.ts",
		"skillc", "skill.yaml", "SPAWN_CONTEXT", "spawn system",
		"spawn logic", "spawn template", "orchestration infrastructure",
		"orchestration system",
	}
	taskLower := strings.ToLower(task)
	for _, keyword := range infrastructureKeywords {
		if strings.Contains(taskLower, keyword) {
			return true
		}
	}
	if beadsID != "" {
		issue, err := verify.GetIssue(beadsID)
		if err == nil {
			titleLower := strings.ToLower(issue.Title)
			for _, keyword := range infrastructureKeywords {
				if strings.Contains(titleLower, keyword) {
					return true
				}
			}
			descLower := strings.ToLower(issue.Description)
			for _, keyword := range infrastructureKeywords {
				if strings.Contains(descLower, keyword) {
					return true
				}
			}
		}
	}
	return false
}

// buildSectionFilter creates a SectionFilter from spawn flags.
// Returns nil when no filtering is needed (backward compatible).
func buildSectionFilter(phases, mode string) *skills.SectionFilter {
	var phasesList []string
	if phases != "" {
		for _, p := range strings.Split(phases, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				phasesList = append(phasesList, p)
			}
		}
	}

	filter := &skills.SectionFilter{
		Phases: phasesList,
		Mode:   mode,
	}
	if filter.IsEmpty() {
		return nil
	}
	return filter
}
