// Package main provides spawn and work commands for the orch CLI.
// This file contains command definitions, flag registration, and backend spawn functions
// (headless, tmux, inline, claude, docker).
// The core spawn orchestration logic is in spawn_pipeline.go (pipeline phases).
// Supporting functionality is in spawn_beads.go, spawn_validation.go, spawn_concurrency.go,
// spawn_usage.go, and spawn_helpers.go.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	statedb "github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
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
	spawnGapThreshold        int    // Custom gap quality threshold (default 20)
	spawnForce               bool   // Force spawn even if issue has blocking dependencies
	spawnBypassTriage        bool   // Explicitly bypass triage (documents conscious decision to spawn directly)
	spawnDesignWorkspace     string // Design workspace name for ui-design-session → feature-impl handoff
	spawnAcknowledgeDecision string // Acknowledge decision conflict to proceed with spawn
)

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
	spawnCmd.Flags().BoolVar(&spawnOpus, "opus", false, "Use Opus via Claude CLI in tmux (Max subscription, implies claude backend + tmux mode)")
	spawnCmd.Flags().BoolVar(&spawnInfra, "infra", false, "Infrastructure work: use claude+tmux backend (survives service crashes)")
	spawnCmd.Flags().StringVar(&spawnValidation, "validation", "tests", "Validation level: none, tests, smoke-test")
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
	spawnCmd.Flags().IntVar(&spawnGapThreshold, "gap-threshold", 0, "Custom gap quality threshold (default 20, only used with --gate-on-gap)")
	spawnCmd.Flags().BoolVar(&spawnForce, "force", false, "Force tactical spawn in hotspot areas (bypasses strategic-first gate - requires justification)")
	spawnCmd.Flags().BoolVar(&spawnBypassTriage, "bypass-triage", false, "Acknowledge manual spawn bypasses daemon-driven triage workflow (required for manual spawns)")
	spawnCmd.Flags().StringVar(&spawnDesignWorkspace, "design-workspace", "", "Design workspace name from ui-design-session for handoff to feature-impl (e.g., 'og-design-ready-queue-08jan')")
	spawnCmd.Flags().StringVar(&spawnAcknowledgeDecision, "acknowledge-decision", "", "[DEPRECATED] Decision gate disabled - this flag has no effect")
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
	// If model flag is provided, use it (existing behavior)
	if spawnModel != "" {
		return model.Resolve(spawnModel)
	}

	// Check global config for skill-specific model
	if globalCfg != nil {
		skillModel := globalCfg.GetModelForSkill(skillName)
		if skillModel != "" {
			return model.Resolve(skillModel)
		}
	}

	// No model flag provided - check project config for backend-specific default
	if projCfg != nil {
		if backend == "claude" && projCfg.Claude.Model != "" {
			return model.Resolve(projCfg.Claude.Model)
		}
		if backend == "opencode" && projCfg.OpenCode.Model != "" {
			return model.Resolve(projCfg.OpenCode.Model)
		}
	}

	// No project config - for opencode backend, default to DeepSeek (cost optimization)
	// For claude backend, default to Opus (Max subscription)
	if backend == "opencode" {
		return model.Resolve("deepseek")
	}

	// Claude backend or no backend specified - use existing DefaultModel behavior (Opus)
	return model.Resolve("")
}

// validateModeModelCombo checks for known invalid mode+model combinations.
// Returns a warning error (non-blocking) if an invalid combination is detected.
func validateModeModelCombo(backend string, resolvedModel model.ModelSpec) error {
	// Invalid combination: opencode + opus
	// Opus requires Claude Code CLI auth, opencode backend creates zombie agents
	if backend == "opencode" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "opus") {
		return fmt.Errorf(`Warning: opencode backend with opus model may fail (auth blocked).
  Recommendation: Remove --backend opencode to use claude backend (default)`)
	}

	// Note: Flash model is blocked earlier in the flow (hard error, not warning)
	// Claude backend + non-opus models work but are non-optimal (using Max sub for cheap models)

	return nil
}

func runSpawnWithSkill(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool) error {
	return runSpawnWithSkillInternal(serverURL, skillName, task, inline, headless, tmux, attach, false)
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

// runSpawnInline spawns the agent inline (blocking) - original behavior.
func runSpawnInline(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	return runSpawnInlineWithClient(opencode.NewClient(serverURL), serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}

func runSpawnInlineWithClient(client opencode.ClientInterface, serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	// Spawn opencode session
	// Format title with beads ID so orch status can match sessions
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)
	cmd := client.BuildSpawnCommand(minimalPrompt, sessionTitle, cfg.Model, cfg.Variant)
	cmd.Stderr = os.Stderr
	cmd.Dir = cfg.ProjectDir
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	result, err := opencode.ProcessOutput(stdout)
	if err != nil {
		return fmt.Errorf("failed to process output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("opencode exited with error: %w", err)
	}

	// Write session ID to workspace file for later lookups
	if result.SessionID != "" {
		if err := spawn.WriteSessionID(cfg.WorkspacePath(), result.SessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
		}
		// Record session ID in state DB for coherent abandon/status resolution
		if err := statedb.RecordSessionID(cfg.WorkspaceName, result.SessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record session ID in state db: %v\n", err)
		}
	}

	// Note: Inline mode is synchronous and blocks until completion,
	// so process ID tracking is not needed (process exits before cleanup)

	// Register orchestrator session in registry (workers use beads instead)
	registerOrchestratorSession(cfg, result.SessionID, task)

	// Log the session creation
	inlineLogger := events.NewLogger(events.DefaultLogPath())
	inlineEventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"spawn_mode":          "inline",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		inlineEventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(inlineEventData, cfg.GapAnalysis)
	addUsageInfoToEventData(inlineEventData, cfg.UsageInfo)
	inlineEvent := events.Event{
		Type:      "session.spawned",
		SessionID: result.SessionID,
		Timestamp: time.Now().Unix(),
		Data:      inlineEventData,
	}
	if err := inlineLogger.Log(inlineEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent:\n")
	fmt.Printf("  Session ID: %s\n", result.SessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	return nil
}

// runSpawnHeadless spawns the agent using CLI subprocess without a TUI.
// This is useful for automation and daemon-driven spawns.
// Uses opencode CLI with --format json to properly support model selection
// (the HTTP API ignores the model parameter).
// Includes retry logic for transient network failures.
func runSpawnHeadless(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	return runSpawnHeadlessWithClient(opencode.NewClient(serverURL), serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}

func runSpawnHeadlessWithClient(client opencode.ClientInterface, serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {

	// Build opencode command using CLI (like inline mode) to support model selection
	// The HTTP API ignores model parameter - only CLI mode honors --model flag
	// Format title with beads ID so orch status can match sessions
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)

	// Use retry logic for transient failures (network issues, server temporarily unavailable)
	retryCfg := spawn.DefaultRetryConfig()
	result, retryResult := spawn.Retry(retryCfg, func() (*headlessSpawnResult, error) {
		return startHeadlessSession(client, serverURL, sessionTitle, minimalPrompt, cfg)
	})

	if retryResult.LastErr != nil {
		// Wrap the error with user-friendly message and recovery guidance
		spawnErr := spawn.WrapSpawnError(retryResult.LastErr, "Headless spawn failed")
		if retryResult.Attempts > 1 {
			fmt.Fprintf(os.Stderr, "Spawn failed after %d attempts\n", retryResult.Attempts)
		}
		// Print formatted error with recovery guidance
		fmt.Fprintf(os.Stderr, "\n%s\n", spawn.FormatSpawnError(spawnErr))
		return spawnErr
	}

	if retryResult.Attempts > 1 {
		fmt.Printf("Spawn succeeded after %d attempts\n", retryResult.Attempts)
	}

	sessionID := result.SessionID

	// Write session ID to workspace file for later lookups
	if err := spawn.WriteSessionID(cfg.WorkspacePath(), sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
	}
	// Record session ID in state DB for coherent abandon/status resolution
	if err := statedb.RecordSessionID(cfg.WorkspaceName, sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record session ID in state db: %v\n", err)
	}

	// Write process ID to workspace file for explicit cleanup
	// This enables killing the process during orch complete/abandon and daemon cleanup
	if result.cmd != nil && result.cmd.Process != nil {
		if err := spawn.WriteProcessID(cfg.WorkspacePath(), result.cmd.Process.Pid); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write process ID: %v\n", err)
		}
	}

	// Start background cleanup goroutine
	result.StartBackgroundCleanup()

	// Register orchestrator session in registry (workers use beads instead)
	registerOrchestratorSession(cfg, sessionID, task)

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"session_id":          sessionID,
		"spawn_mode":          "headless",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if retryResult.Attempts > 1 {
		eventData["retry_attempts"] = retryResult.Attempts
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent (headless):\n")
	fmt.Printf("  Session ID: %s\n", sessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	return nil
}

// headlessSpawnResult contains the result of starting a headless session.
type headlessSpawnResult struct {
	SessionID string
	cmd       *exec.Cmd
	stdout    io.ReadCloser
}

// StartBackgroundCleanup starts a goroutine to drain stdout and wait for the process.
func (r *headlessSpawnResult) StartBackgroundCleanup() {
	if r.stdout == nil || r.cmd == nil {
		return
	}
	go func() {
		// Drain remaining stdout
		io.Copy(io.Discard, r.stdout)
		// Wait for process to complete (cleanup)
		r.cmd.Wait()
	}()
}

// ansiRegex matches ANSI escape sequences (colors, formatting, etc.)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape codes from a string for cleaner error messages.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// startHeadlessSession starts an opencode session and extracts the session ID.
// Returns the result with session ID and resources for cleanup.
// Note: Uses CLI mode instead of HTTP API because OpenCode's HTTP API ignores the model parameter.
// CLI mode correctly honors the --model flag.
// See: .kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md
func startHeadlessSession(client opencode.ClientInterface, serverURL, sessionTitle, minimalPrompt string, cfg *spawn.Config) (*headlessSpawnResult, error) {
	cmd := client.BuildSpawnCommand(minimalPrompt, sessionTitle, cfg.Model, cfg.Variant)
	cmd.Dir = cfg.ProjectDir
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		spawnErr := spawn.WrapSpawnError(err, "Failed to get stdout pipe")
		return nil, spawnErr
	}

	// Capture stderr to include in error messages when session ID extraction fails.
	// Previously stderr was discarded (nil), losing valuable error context like
	// "Error: Session not found" or model-specific errors.
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		spawnErr := spawn.WrapSpawnError(err, "Failed to start opencode process")
		return nil, spawnErr
	}

	// Process stdout to extract session ID, then let the process run in background
	// We need to read at least until we get the session ID
	sessionID, err := opencode.ExtractSessionIDFromReader(stdout)
	if err != nil {
		// Try to kill the process if we couldn't get session ID
		cmd.Process.Kill()
		// Include stderr content for better error context
		stderrContent := strings.TrimSpace(stderrBuf.String())
		// Strip ANSI escape codes for cleaner error messages
		stderrContent = stripANSI(stderrContent)
		errMsg := "Failed to extract session ID"
		if stderrContent != "" {
			errMsg = fmt.Sprintf("Failed to extract session ID: %s", stderrContent)
		}
		spawnErr := spawn.WrapSpawnError(err, errMsg)
		return nil, spawnErr
	}

	return &headlessSpawnResult{
		SessionID: sessionID,
		cmd:       cmd,
		stdout:    stdout,
	}, nil
}

// runSpawnTmux spawns the agent in a tmux window (interactive, returns immediately).
// Creates a tmux window in workers-{project} session (or orchestrator session for orchestrator skills).
func runSpawnTmux(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string, attach bool) error {
	return runSpawnTmuxWithClient(opencode.NewClient(serverURL), serverURL, cfg, minimalPrompt, beadsID, skillName, task, attach)
}

func runSpawnTmuxWithClient(client opencode.ClientInterface, serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string, attach bool) error {
	var sessionName string
	var err error

	// Meta-orchestrators and orchestrators go into 'orchestrator' tmux session
	// Workers go into 'workers-{project}' session
	if cfg.IsMetaOrchestrator || cfg.IsOrchestrator {
		sessionName, err = tmux.EnsureOrchestratorSession()
	} else {
		sessionName, err = tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
	}
	if err != nil {
		return fmt.Errorf("failed to ensure tmux session: %w", err)
	}

	// Build window name with emoji and beads ID
	windowName := tmux.BuildWindowName(cfg.WorkspaceName, cfg.SkillName, beadsID)

	// Create new tmux window
	windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// When a model is specified, create the session via API first (opencode attach
	// doesn't accept --model). Then attach to that session by ID.
	var preCreatedSessionID string
	if cfg.Model != "" {
		resp, createErr := client.CreateSession(cfg.WorkspaceName, cfg.ProjectDir, cfg.Model, "", !cfg.IsOrchestrator && !cfg.IsMetaOrchestrator)
		if createErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to pre-create session with model %s: %v (falling back to attach without model)\n", cfg.Model, createErr)
		} else {
			preCreatedSessionID = resp.ID
		}
	}

	// Build opencode command using tmux package
	attachCfg := &tmux.OpencodeAttachConfig{
		ServerURL:  serverURL,
		ProjectDir: cfg.ProjectDir,
		// Don't pass Model — opencode attach doesn't accept --model
		SessionID: preCreatedSessionID,
	}
	opencodeCmd := tmux.BuildOpencodeAttachCommand(attachCfg)

	// Send command and execute
	if err := tmux.SendKeys(windowTarget, opencodeCmd); err != nil {
		return fmt.Errorf("failed to send opencode command: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Wait for OpenCode TUI to be ready
	waitCfg := tmux.DefaultWaitConfig()
	if err := tmux.WaitForOpenCodeReady(windowTarget, waitCfg); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	// Use pre-created session ID if available, otherwise discover via API
	sessionID := preCreatedSessionID
	if sessionID == "" {
		// Capture session ID from API with retry (OpenCode may not have registered yet)
		sessionID, _ = client.FindRecentSessionWithRetry(cfg.ProjectDir, 3, 500*time.Millisecond)
		// Note: We silently ignore errors here since window_id is sufficient for tmux monitoring
	}

	// Send prompt
	sendCfg := tmux.DefaultSendPromptConfig()
	time.Sleep(sendCfg.PostReadyDelay)
	if err := tmux.SendKeysLiteral(windowTarget, minimalPrompt); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	// Write session ID to workspace file for later lookups
	if sessionID != "" {
		if err := spawn.WriteSessionID(cfg.WorkspacePath(), sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
		}
		// Record session ID in state DB for coherent abandon/status resolution
		if err := statedb.RecordSessionID(cfg.WorkspaceName, sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record session ID in state db: %v\n", err)
		}
	}

	// Record tmux window in state DB for liveness tracking
	if err := statedb.RecordTmuxWindow(cfg.WorkspaceName, windowTarget); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record tmux window in state db: %v\n", err)
	}

	// Register orchestrator session in registry (workers use beads instead)
	registerOrchestratorSession(cfg, sessionID, task)

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"session_id":          sessionID,
		"window":              windowTarget,
		"window_id":           windowID,
		"session_name":        sessionName,
		"spawn_mode":          "tmux",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Focus the newly created window (skip for daemon-driven spawns to avoid interrupting orchestrator)
	if !cfg.DaemonDriven {
		selectCmd := exec.Command("tmux", "select-window", "-t", windowTarget)
		if err := selectCmd.Run(); err != nil {
			// Non-fatal - window was created successfully
			fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
		}
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent in tmux:\n")
	fmt.Printf("  Session:    %s\n", sessionName)
	if sessionID != "" {
		fmt.Printf("  Session ID: %s\n", sessionID)
	}
	fmt.Printf("  Window:     %s\n", windowTarget)
	fmt.Printf("  Window ID:  %s\n", windowID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	// Attach if requested
	if attach {
		if err := tmux.Attach(windowTarget); err != nil {
			return fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return nil
}

// runSpawnClaude spawns the agent using Claude Code CLI in a tmux window.
func runSpawnClaude(serverURL string, cfg *spawn.Config, beadsID, skillName, task string, attach bool) error {
	result, err := spawn.SpawnClaude(cfg)
	if err != nil {
		return err
	}

	// Record tmux window in state DB for liveness tracking
	if err := statedb.RecordTmuxWindow(cfg.WorkspaceName, result.Window); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record tmux window in state db: %v\n", err)
	}

	// Register orchestrator session in registry if needed
	registerOrchestratorSession(cfg, "", task)

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"window":              result.Window,
		"window_id":           result.WindowID,
		"spawn_mode":          "claude",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Focus the newly created window (skip for daemon-driven spawns to avoid interrupting orchestrator)
	if !cfg.DaemonDriven {
		selectCmd := exec.Command("tmux", "select-window", "-t", result.Window)
		if err := selectCmd.Run(); err != nil {
			// Non-fatal
			fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
		}
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent in Claude mode (tmux):\n")
	fmt.Printf("  Window:     %s\n", result.Window)
	fmt.Printf("  Window ID:  %s\n", result.WindowID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	// Attach if requested
	if attach {
		if err := tmux.Attach(result.Window); err != nil {
			return fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return nil
}

// runSpawnClaudeInline spawns the agent using Claude Code CLI inline (blocking).
// This runs claude directly in the current terminal without tmux, for interactive sessions.
func runSpawnClaudeInline(serverURL string, cfg *spawn.Config, beadsID, skillName, task string) error {
	// Register orchestrator session in registry if needed (before spawn, in case it fails)
	registerOrchestratorSession(cfg, "", task)

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"spawn_mode":          "claude-inline",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawning agent in Claude mode (inline):\n")
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))
	fmt.Println()

	// Run Claude inline (blocking) - this will take over the terminal
	if err := spawn.SpawnClaudeInline(cfg); err != nil {
		return err
	}

	return nil
}

// runSpawnDocker spawns the agent using Docker for Statsig fingerprint isolation.
// This is an escape hatch for rate limit scenarios - provides fresh fingerprint per spawn.
func runSpawnDocker(serverURL string, cfg *spawn.Config, beadsID, skillName, task string, attach bool) error {
	result, err := spawn.SpawnDocker(cfg)
	if err != nil {
		return err
	}

	// Record tmux window in state DB for liveness tracking
	if err := statedb.RecordTmuxWindow(cfg.WorkspaceName, result.Window); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record tmux window in state db: %v\n", err)
	}

	// Register orchestrator session in registry if needed
	registerOrchestratorSession(cfg, "", task)

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"window":              result.Window,
		"window_id":           result.WindowID,
		"spawn_mode":          "docker",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Focus the newly created window (skip for daemon-driven spawns to avoid interrupting orchestrator)
	if !cfg.DaemonDriven {
		selectCmd := exec.Command("tmux", "select-window", "-t", result.Window)
		if err := selectCmd.Run(); err != nil {
			// Non-fatal
			fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
		}
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent in Docker mode (rate limit escape hatch):\n")
	fmt.Printf("  Window:     %s\n", result.Window)
	fmt.Printf("  Window ID:  %s\n", result.WindowID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Container:  %s\n", spawn.DockerImageName)
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	// Attach if requested
	if attach {
		if err := tmux.Attach(result.Window); err != nil {
			return fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return nil
}
