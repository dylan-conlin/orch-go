// Package main provides spawn and work commands for the orch CLI.
// This file contains command definitions, flag registration, and pipeline entrypoints.
// The core spawn orchestration logic is in spawn_pipeline.go (pipeline phases).
// Supporting functionality is in spawn_beads.go, spawn_validation.go, spawn_concurrency.go,
// spawn_usage.go, spawn_helpers.go, and spawn_execute.go.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/spf13/cobra"
)

// DefaultMaxAgents is the default maximum number of concurrent agents.
const DefaultMaxAgents = 5

var (
	// Spawn command flags
	spawnSkill               string
	spawnIssue               string
	spawnPhases              string
	spawnMode                string // Implementation mode: tdd or direct
	spawnBackendFlag         string // Spawn backend: claude or opencode (overrides config and auto-selection)
	spawnAccount             string // Claude Max account name to switch to before spawn (saved accounts.yaml key)
	spawnOpus                bool   // Use Opus via Claude CLI in tmux (implies claude mode)
	spawnInfra               bool   // Infrastructure work: implies claude+tmux (survives service crashes)
	spawnValidation          string
	spawnInline              bool   // Run inline (blocking) with TUI
	spawnHeadless            bool   // Run headless via HTTP API (automation/scripting)
	spawnTmux                bool   // Run in tmux window (opt-in, overrides default headless)
	spawnAttach              bool   // Attach to tmux window after spawning
	spawnModel               string // Model to use for standalone spawns
	spawnVariant             string // Extended thinking variant: high, max, or none
	spawnNoTrack             bool   // Opt-out of beads tracking
	spawnMCP                 string // MCP server config (e.g., "playwright")
	spawnSkipArtifactCheck   bool   // Bypass pre-spawn kb context check
	spawnMaxAgents           int    // Maximum concurrent agents (0 = use default or env var)
	spawnAutoInit            bool   // Auto-initialize .orch and .beads if missing
	spawnLight               bool   // Light tier spawn (skips SYNTHESIS.md requirement)
	spawnFull                bool   // Full tier spawn (requires SYNTHESIS.md)
	spawnWorkdir             string // Target project directory (defaults to current directory)
	spawnGateOnGap           bool   // Block spawn if context quality is too low
	spawnSkipGapGate         bool   // Explicitly bypass gap gating (documents conscious decision)
	spawnGapThreshold        int    // Custom gap quality threshold (defaults from project config)
	spawnForce               bool   // Force spawn even if issue has blocking dependencies
	spawnBypassTriage        bool   // Explicitly bypass triage (documents conscious decision to spawn directly)
	spawnAcknowledgeHotspot  bool   // Suppress hotspot warning output for this spawn
	spawnDesignWorkspace     string // Design workspace name for ui-design-session → feature-impl handoff
	spawnAcknowledgeDecision string // Acknowledge decision conflict to proceed with spawn
	spawnContextBudget       int    // Token budget for generated SPAWN_CONTEXT.md
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [skill] [task]",
	Short: "Spawn a new agent with skill context (default: headless)",
	Long: `Spawn a new OpenCode session with skill context.

IMPORTANT: Tracked manual spawns require triage bypass acknowledgement.
The default workflow is: create issues with triage:ready label → daemon auto-spawns.
Manual spawning is for exceptions only (urgent single items, complex context needed).

To proceed with manual spawn, use either --bypass-triage (one-off)
or set ORCH_BYPASS_TRIAGE=1 for a session-level bypass.
This creates friction to encourage the preferred daemon-driven workflow.

Hotspot Warning Suppression:
  If you're intentionally iterating in a known hotspot area, you can suppress
  repeated hotspot warnings:
    - One-off: --acknowledge-hotspot
    - Session: export ORCH_SUPPRESS_HOTSPOT=1

Backend Modes (--backend):
  opencode: Uses OpenCode HTTP API (DeepSeek, etc.) - DEFAULT
            Dashboard visibility, cost tracking, headless batch work
  claude:   Uses Claude Code CLI in tmux (Max subscription, unlimited Opus)
            Survives service crashes, for infrastructure work
  docker:   Uses Claude CLI in Docker container for Statsig fingerprint isolation
            (Rate limit escape hatch - fresh fingerprint per spawn)

  Priority: --backend flag > --opus flag > --infra flag > config (spawn_mode) > default (opencode)
  Config can set default mode: spawn_mode: opencode in .orch/config.yaml

  Infrastructure Work:
    Use --infra flag for work on critical services (opencode, daemon, spawn itself).
    Implies: --backend claude --tmux (crash-resistant backend with visible TUI)
    Example: orch spawn --bypass-triage --infra investigation "fix opencode crash"

Spawn Modes:
  Default (headless): Spawns via HTTP API - no TUI, automation-friendly, returns immediately
  --tmux:             Spawns in a tmux window - visible, interruptible, opt-in
  --inline:           Runs in current terminal - blocking with TUI, for debugging
                      With --backend claude: Claude CLI runs directly (interactive orchestrator sessions)
                      Without backend: OpenCode TUI runs directly
  --attach:           Spawns in tmux and attaches immediately (implies --tmux)

Spawn Tiers:
  --light: Skip SYNTHESIS.md requirement (for code-focused work)
  --full:  Require SYNTHESIS.md for knowledge externalization
  
  Default tier is determined by skill:
    Full tier (require SYNTHESIS.md): investigation, architect, research,
      codebase-audit, design-session
    Light tier (skip SYNTHESIS.md): feature-impl, systematic-debugging,
      reliability-testing, issue-creation

Gap Gating (Gate Over Remind):
  --gate-on-gap:      Block spawn if context quality is too low (score < 20)
  --skip-gap-gate:    Explicitly bypass gating (documents conscious decision)
  --gap-threshold N:  Custom quality threshold (default from .orch/config.yaml spawn.context_quality.threshold, fallback 20)
  
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
  
  # Manual spawn (one-off bypass)
  orch spawn --bypass-triage investigation "explore the codebase"
  orch spawn --bypass-triage feature-impl "add feature" --phases implementation,validation
  orch spawn --bypass-triage --issue proj-123 feature-impl "implement the feature"

  # Session-level bypass (avoid repeating the flag)
  export ORCH_BYPASS_TRIAGE=1
  orch spawn investigation "orchestrator-directed task A"
  orch spawn feature-impl "orchestrator-directed task B"

  # Suppress repeated hotspot warnings while intentionally working the same area
  export ORCH_SUPPRESS_HOTSPOT=1
  orch spawn feature-impl "continue fixing cmd/orch/spawn_pipeline.go"
  
  # Tmux mode (opt-in) - visible, interruptible
  orch spawn --bypass-triage --tmux investigation "explore codebase"
  orch spawn --bypass-triage --attach investigation "explore codebase"
  
  # Inline mode - blocking with TUI, for debugging
  orch spawn --bypass-triage --inline investigation "explore codebase"

  # Claude CLI inline mode - interactive orchestrator session in current terminal
  orch spawn --bypass-triage --backend claude --inline orchestrator "coordinate work"

  # Infrastructure work - crash-resistant backend (implies claude+tmux)
  orch spawn --bypass-triage --infra investigation "fix opencode server crash"

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

		return runSpawnWithSkill(serverURL, skillName, task, spawnInline, spawnHeadless, spawnTmux, spawnAttach)
	},
}

func init() {
	spawnCmd.Flags().StringVar(&spawnIssue, "issue", "", "Beads issue ID for tracking")
	spawnCmd.Flags().StringVar(&spawnPhases, "phases", "", "Feature-impl phases (e.g., implementation,validation)")
	spawnCmd.Flags().StringVar(&spawnMode, "mode", "tdd", "Implementation mode: tdd or direct")
	spawnCmd.Flags().StringVar(&spawnBackendFlag, "backend", "", "Spawn backend: claude (tmux + Claude CLI), opencode (HTTP API), or docker (containerized for fingerprint isolation). Overrides config and auto-selection.")
	spawnCmd.Flags().StringVar(&spawnAccount, "account", "", "Claude account name from ~/.orch/accounts.yaml to switch to before spawn")
	spawnCmd.Flags().BoolVar(&spawnOpus, "opus", false, "Use Opus via Claude CLI in tmux (Max subscription, implies claude backend + tmux mode)")
	spawnCmd.Flags().BoolVar(&spawnInfra, "infra", false, "Infrastructure work: use claude+tmux backend (survives service crashes)")
	spawnCmd.Flags().StringVar(&spawnValidation, "validation", "tests", "Validation level: none, tests, integration, smoke-test")
	spawnCmd.Flags().BoolVar(&spawnInline, "inline", false, "Run inline (blocking) with TUI")
	spawnCmd.Flags().BoolVar(&spawnHeadless, "headless", false, "Run headless via HTTP API (default behavior, flag is redundant)")
	spawnCmd.Flags().BoolVar(&spawnTmux, "tmux", false, "Run in tmux window (opt-in for visual monitoring)")
	spawnCmd.Flags().BoolVar(&spawnAttach, "attach", false, "Attach to tmux window after spawning (implies --tmux)")
	spawnCmd.Flags().StringVar(&spawnModel, "model", "", "Model alias (opus, sonnet, haiku, flash, pro) or provider/model format")
	spawnCmd.Flags().StringVar(&spawnVariant, "variant", "", "Extended thinking variant: high (16k tokens), max (32k tokens), or none (disable). Defaults based on skill type.")
	spawnCmd.Flags().BoolVar(&spawnNoTrack, "no-track", false, "Opt-out of beads issue tracking (ad-hoc work)")
	spawnCmd.Flags().StringVar(&spawnMCP, "mcp", "", "MCP server config (e.g., 'playwright' for browser automation)")
	spawnCmd.Flags().BoolVar(&spawnSkipArtifactCheck, "skip-artifact-check", false, "Bypass pre-spawn kb context check")
	spawnCmd.Flags().IntVar(&spawnMaxAgents, "max-agents", -1, "Maximum concurrent agents (default 5, 0 disables limit, or use ORCH_MAX_AGENTS env var)")
	spawnCmd.Flags().BoolVar(&spawnAutoInit, "auto-init", false, "Auto-initialize .orch and .beads if missing")
	spawnCmd.Flags().BoolVar(&spawnLight, "light", false, "Light tier spawn (skips SYNTHESIS.md requirement on completion)")
	spawnCmd.Flags().BoolVar(&spawnFull, "full", false, "Full tier spawn (requires SYNTHESIS.md for knowledge externalization)")
	spawnCmd.Flags().StringVar(&spawnWorkdir, "workdir", "", "Target project directory (defaults to current directory)")
	spawnCmd.Flags().StringVar(&spawnWorkdir, "project", "", "Alias for --workdir")
	spawnCmd.Flags().MarkHidden("project")
	spawnCmd.Flags().BoolVar(&spawnGateOnGap, "gate-on-gap", false, "Block spawn if context quality is too low (enforces Gate Over Remind)")
	spawnCmd.Flags().BoolVar(&spawnSkipGapGate, "skip-gap-gate", false, "Explicitly bypass gap gating (documents conscious decision to proceed without context)")
	spawnCmd.Flags().IntVar(&spawnGapThreshold, "gap-threshold", 0, "Custom gap quality threshold (default from .orch/config.yaml spawn.context_quality.threshold, fallback 20; only used with --gate-on-gap)")
	spawnCmd.Flags().BoolVar(&spawnForce, "force", false, "Force tactical spawn in hotspot areas (bypasses strategic-first gate - requires justification)")
	spawnCmd.Flags().BoolVar(&spawnBypassTriage, "bypass-triage", false, "One-off triage bypass acknowledgement for manual tracked spawns (or set ORCH_BYPASS_TRIAGE=1 for session-level bypass)")
	spawnCmd.Flags().BoolVar(&spawnAcknowledgeHotspot, "acknowledge-hotspot", false, "Suppress hotspot warning output for this spawn (or set ORCH_SUPPRESS_HOTSPOT=1 for session-level suppression)")
	spawnCmd.Flags().StringVar(&spawnDesignWorkspace, "design-workspace", "", "Design workspace name from ui-design-session for handoff to feature-impl (e.g., 'og-design-ready-queue-08jan')")
	spawnCmd.Flags().StringVar(&spawnAcknowledgeDecision, "acknowledge-decision", "", "[DEPRECATED] Decision gate disabled - this flag has no effect")
	spawnCmd.Flags().IntVar(&spawnContextBudget, "context-budget", spawn.DefaultSpawnContextBudgetTokens, "Token budget for SPAWN_CONTEXT.md (default 12000). Context is deterministically truncated when over budget")
}

var (
	// Work command flags
	workInline  bool   // Run inline (blocking) with TUI
	workWorkdir string // Target project directory (defaults to current directory)
)

var workCmd = &cobra.Command{
	Use:   "work [beads-id]",
	Short: "Start work on a beads issue with skill inference",
	Long: `Start work on a beads issue by inferring the skill from the issue type.

The skill is automatically determined from the issue type:
  - bug         → systematic-debugging (direct action; use skill:architect label for complex bugs)
  - feature     → feature-impl
  - task        → feature-impl
  - investigation → investigation
  - question    → investigation

The issue description becomes the task prompt for the spawned agent.

By default, spawns in a tmux window (visible, interruptible).
Use --inline to run in the current terminal (blocking with TUI).
Use --workdir to spawn in a different project directory (for cross-project daemon).

Examples:
  orch-go work proj-123                           # Start work in tmux window (default)
  orch-go work proj-123 --inline                  # Start work inline (blocking TUI)
  orch-go work proj-123 --workdir ~/other-project # Start work in another project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runWork(serverURL, beadsID, workInline, workWorkdir)
	},
}

func init() {
	workCmd.Flags().BoolVar(&workInline, "inline", false, "Run inline (blocking) with TUI")
	workCmd.Flags().StringVar(&workWorkdir, "workdir", "", "Target project directory (defaults to current directory)")
	workCmd.Flags().StringVar(&workWorkdir, "project", "", "Alias for --workdir")
	workCmd.Flags().MarkHidden("project")
}

// resolveModelWithConfig resolves the model specification, checking project and global config
// for backend-specific defaults when no explicit --model flag is provided.
func resolveModelWithConfig(spawnModel, backend, skillName string, projCfg *config.Config, globalCfg *userconfig.Config) model.ModelSpec {
	if spawnModel != "" {
		return model.Resolve(spawnModel)
	}

	if globalCfg != nil {
		skillModel := globalCfg.GetModelForSkill(skillName)
		if skillModel != "" {
			return model.Resolve(skillModel)
		}
	}

	if projCfg != nil {
		if backend == "claude" && projCfg.Claude.Model != "" {
			return model.Resolve(projCfg.Claude.Model)
		}
		if backend == "opencode" && projCfg.OpenCode.Model != "" {
			return model.Resolve(projCfg.OpenCode.Model)
		}
	}

	if backend == "opencode" {
		return model.Resolve("deepseek")
	}

	return model.Resolve("")
}

// validateModeModelCombo checks for known invalid mode+model combinations.
// Returns a warning error (non-blocking) if an invalid combination is detected.
func validateModeModelCombo(backend string, resolvedModel model.ModelSpec) error {
	if backend == "opencode" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "opus") {
		return fmt.Errorf(`Warning: opencode backend with opus model may fail (auth blocked).
  Recommendation: Remove --backend opencode to use claude backend (default)`)
	}

	return nil
}

func runSpawnWithSkill(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool) error {
	// Auto-bypass triage for orchestrator skills (they inherently perform triage)
	daemonDriven := false
	if bypass, reason := shouldAutoBypassTriage(skillName); bypass {
		daemonDriven = true
		fmt.Fprintf(os.Stderr, "ℹ️  Auto-bypassing triage ceremony (%s skill performs triage)\n", reason)
	}
	return runSpawnWithSkillInternal(serverURL, skillName, task, inline, headless, tmux, attach, daemonDriven)
}

// runSpawnWithSkillInternal is the internal implementation that supports daemon-driven spawns.
// When daemonDriven is true, the triage bypass check is skipped (issue already triaged).
//
// This function orchestrates a pipeline of sequential phases:
//  1. Pre-flight validation (triage, concurrency, rate limits, hotspots)
//  2. Project resolution (directory, scaffolding)
//  3. Skill loading (content, type detection, workspace name)
//  4. Issue tracking setup (beads ID, duplicate checks, status)
//  5. Context gathering (KB context, gap analysis)
//  6. Config building (backend, model, tier, spawn.Config)
//  7. Spawn execution (validate, write context, dispatch to backend)
//
// See spawn_pipeline.go for phase implementations.
func runSpawnWithSkillInternal(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool, daemonDriven bool) error {
	p := newSpawnPipeline(serverURL, skillName, task, inline, headless, tmux, attach, daemonDriven)

	if err := p.runPreFlightValidation(); err != nil {
		return err
	}
	if err := p.resolveProject(); err != nil {
		return err
	}
	if err := p.loadSkill(); err != nil {
		return err
	}
	if err := p.setupIssueTracking(); err != nil {
		return err
	}
	if err := p.gatherContext(); err != nil {
		return err
	}
	if err := p.buildSpawnConfig(); err != nil {
		return err
	}
	return p.executeSpawn()
}

// formatSessionTitle formats the session title to include beads ID for matching.
// Format: "workspace-name [beads-id]" (e.g., "og-debug-orch-status-23dec [orch-go-v4mw]")
// This allows extractBeadsIDFromTitle to find agents in orch status.
func formatSessionTitle(workspaceName, beadsID string) string {
	if beadsID == "" {
		return workspaceName
	}
	return fmt.Sprintf("%s [%s]", workspaceName, beadsID)
}
