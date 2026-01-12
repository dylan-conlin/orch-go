// Package main provides spawn and work commands for the orch CLI.
// This file contains all spawn-related functionality including:
// - spawn command with all flags and modes (headless, tmux, inline)
// - work command for daemon-driven spawns
// - beads issue creation and tracking
// - gap analysis and context gathering
// - concurrency limiting and account switching
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

// DefaultMaxAgents is the default maximum number of concurrent agents.
const DefaultMaxAgents = 5

var (
	// Spawn command flags
	spawnSkill             string
	spawnIssue             string
	spawnPhases            string
	spawnMode              string // Implementation mode: tdd or direct
	spawnBackendFlag       string // Spawn backend: claude or opencode (overrides config and auto-selection)
	spawnOpus              bool   // Use Opus via Claude CLI in tmux (implies claude mode)
	spawnValidation        string
	spawnInline            bool   // Run inline (blocking) with TUI
	spawnHeadless          bool   // Run headless via HTTP API (automation/scripting)
	spawnTmux              bool   // Run in tmux window (opt-in, overrides default headless)
	spawnAttach            bool   // Attach to tmux window after spawning
	spawnModel             string // Model to use for standalone spawns
	spawnNoTrack           bool   // Opt-out of beads tracking
	spawnMCP               string // MCP server config (e.g., "playwright")
	spawnSkipArtifactCheck bool   // Bypass pre-spawn kb context check
	spawnMaxAgents         int    // Maximum concurrent agents (0 = use default or env var)
	spawnAutoInit          bool   // Auto-initialize .orch and .beads if missing
	spawnLight             bool   // Light tier spawn (skips SYNTHESIS.md requirement)
	spawnFull              bool   // Full tier spawn (requires SYNTHESIS.md)
	spawnWorkdir           string // Target project directory (defaults to current directory)
	spawnGateOnGap         bool   // Block spawn if context quality is too low
	spawnSkipGapGate       bool   // Explicitly bypass gap gating (documents conscious decision)
	spawnGapThreshold      int    // Custom gap quality threshold (default 20)
	spawnForce             bool   // Force spawn even if issue has blocking dependencies
	spawnBypassTriage      bool   // Explicitly bypass triage (documents conscious decision to spawn directly)
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
  claude:   Uses Claude Code CLI in tmux (Max subscription, unlimited Opus)
  opencode: Uses OpenCode HTTP API (default)
  
  Config can set default mode (orch config set spawn_mode claude|opencode).
  The --backend flag overrides the config setting for this spawn only.

Spawn Modes:
  Default (headless): Spawns via HTTP API - no TUI, automation-friendly, returns immediately
  --tmux:             Spawns in a tmux window - visible, interruptible, opt-in
  --inline:           Runs in current terminal - blocking with TUI, for debugging
  --attach:           Spawns in tmux and attaches immediately (implies --tmux)

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
  orch spawn --bypass-triage --attach investigation "explore codebase"
  
  # Inline mode - blocking with TUI, for debugging
  orch spawn --bypass-triage --inline investigation "explore codebase"
  
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
	spawnCmd.Flags().StringVar(&spawnBackendFlag, "backend", "", "Spawn backend: claude (tmux + Claude CLI) or opencode (HTTP API). Overrides config and auto-selection.")
	spawnCmd.Flags().BoolVar(&spawnOpus, "opus", false, "Use Opus via Claude CLI in tmux (Max subscription, implies claude backend + tmux mode)")
	spawnCmd.Flags().StringVar(&spawnValidation, "validation", "tests", "Validation level: none, tests, smoke-test")
	spawnCmd.Flags().BoolVar(&spawnInline, "inline", false, "Run inline (blocking) with TUI")
	spawnCmd.Flags().BoolVar(&spawnHeadless, "headless", false, "Run headless via HTTP API (default behavior, flag is redundant)")
	spawnCmd.Flags().BoolVar(&spawnTmux, "tmux", false, "Run in tmux window (opt-in for visual monitoring)")
	spawnCmd.Flags().BoolVar(&spawnAttach, "attach", false, "Attach to tmux window after spawning (implies --tmux)")
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
	spawnCmd.Flags().BoolVar(&spawnForce, "force", false, "Force tactical spawn in hotspot areas (bypasses strategic-first gate - requires justification)")
	spawnCmd.Flags().BoolVar(&spawnBypassTriage, "bypass-triage", false, "Acknowledge manual spawn bypasses daemon-driven triage workflow (required for manual spawns)")
}

var (
	// Work command flags
	workInline bool // Run inline (blocking) with TUI
)

var workCmd = &cobra.Command{
	Use:   "work [beads-id]",
	Short: "Start work on a beads issue with skill inference",
	Long: `Start work on a beads issue by inferring the skill from the issue type.

The skill is automatically determined from the issue type:
  - bug         → architect (understand before fixing; use skill:systematic-debugging label for clear bugs)
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
		return runWork(serverURL, beadsID, workInline)
	},
}

func init() {
	workCmd.Flags().BoolVar(&workInline, "inline", false, "Run inline (blocking) with TUI")
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
		// Default to architect: understand before fixing
		// Use skill:systematic-debugging label for clear, isolated bugs
		return "architect", nil
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

func runWork(serverURL, beadsID string, inline bool) error {
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

	// Use issue title and description as the task for full context
	task := issue.Title
	if issue.Description != "" {
		task = issue.Title + "\n\n" + issue.Description
	}

	// Set the spawnIssue flag so runSpawnWithSkillInternal uses the existing issue
	spawnIssue = beadsID

	// Set the spawnMCP flag if the issue has a needs:* label (e.g., needs:playwright)
	// This allows daemon-spawned agents to automatically get browser access for UI work
	if mcpServer != "" {
		spawnMCP = mcpServer
	}

	fmt.Printf("Starting work on: %s\n", beadsID)
	fmt.Printf("  Title:  %s\n", issue.Title)
	fmt.Printf("  Type:   %s\n", issue.IssueType)
	fmt.Printf("  Skill:  %s\n", skillName)
	if mcpServer != "" {
		fmt.Printf("  MCP:    %s\n", mcpServer)
	}

	// Work command is daemon-driven (issue already created and triaged)
	// Pass daemonDriven=true to skip triage bypass check
	return runSpawnWithSkillInternal(serverURL, skillName, task, inline, true, false, false, true)
}

// getMaxAgents returns the effective maximum agents limit.
// Priority: --max-agents flag > ORCH_MAX_AGENTS env var > DefaultMaxAgents constant.
// Returns 0 if limit is explicitly disabled (flag set to 0).
func getMaxAgents() int {
	// If flag was explicitly set (non-zero), use it
	if spawnMaxAgents != 0 {
		return spawnMaxAgents
	}

	// Check environment variable
	if envVal := os.Getenv("ORCH_MAX_AGENTS"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil {
			return val
		}
		// Invalid value - fall through to default
		fmt.Fprintf(os.Stderr, "Warning: invalid ORCH_MAX_AGENTS value '%s', using default %d\n", envVal, DefaultMaxAgents)
	}

	return DefaultMaxAgents
}

// ensureOpenCodeRunning checks if OpenCode is reachable, and starts it if not.
// Returns nil if OpenCode is running (or was successfully started), error otherwise.
func ensureOpenCodeRunning() error {
	client := opencode.NewClient(serverURL)
	_, err := client.ListSessions("")
	if err == nil {
		return nil // Already running
	}

	// Check if it's a connection error (not running)
	if !strings.Contains(err.Error(), "connection refused") {
		return nil // Some other error, let it proceed
	}

	fmt.Fprintf(os.Stderr, "OpenCode not running, starting it...\n")

	// Start OpenCode server in background, fully detached via shell
	// This ensures the process survives even if the parent is killed
	// Set ORCH_WORKER=1 so agents spawned by this server know they are orch-managed workers
	cmd := exec.Command("sh", "-c", "ORCH_WORKER=1 opencode serve --port 4096 </dev/null >/dev/null 2>&1 &")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start OpenCode: %w", err)
	}

	// Wait for it to be ready (poll for up to 10 seconds)
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := client.ListSessions("")
		if err == nil {
			fmt.Fprintf(os.Stderr, "OpenCode started successfully\n")
			return nil
		}
	}

	return fmt.Errorf("OpenCode started but not responding after 10s")
}

// checkConcurrencyLimit checks if spawning a new agent would exceed the concurrency limit.
// Returns nil if spawning is allowed, or an error if at the limit.
func checkConcurrencyLimit() error {
	maxAgents := getMaxAgents()

	// Limit disabled (0 means unlimited)
	if maxAgents == 0 {
		return nil
	}

	// Ensure OpenCode is running before checking
	if err := ensureOpenCodeRunning(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		return nil // Allow spawn to proceed, it will fail later with better error
	}

	// Check active count via OpenCode API
	client := opencode.NewClient(serverURL)
	sessions, err := client.ListSessions("")
	if err != nil {
		// If we can't check, log a warning but allow the spawn
		fmt.Fprintf(os.Stderr, "Warning: could not check agent limit (API error): %v\n", err)
		return nil
	}

	// Filter to only count active ORCH-SPAWNED sessions:
	// 1. Updated within last 30 minutes (not stale)
	// 2. Has parseable beadsID (is orch-spawned, not manual OpenCode session)
	// 3. Has not reported Phase: Complete (completed agents are idle)
	now := time.Now()
	staleThreshold := 30 * time.Minute
	activeCount := 0
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		idleTime := now.Sub(updatedAt)
		if idleTime >= staleThreshold {
			continue // stale session
		}
		// Only count sessions with parseable beadsID (orch-spawned agents)
		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" {
			continue // not an orch-spawned agent
		}
		// Exclude completed agents (Phase: Complete) - they're idle and not consuming resources
		if isComplete, _ := verify.IsPhaseComplete(beadsID); isComplete {
			continue // completed agent, don't count against limit
		}
		activeCount++
	}

	if activeCount >= maxAgents {
		return fmt.Errorf("concurrency limit reached: %d active agents (max %d). Use 'orch status' to see active agents, 'orch complete' to finish agents, or --max-agents to increase limit", activeCount, maxAgents)
	}

	return nil
}

// determineSpawnTier determines the spawn tier based on flags, config, and skill defaults.
// Priority: --light flag > --full flag > userconfig.default_tier > skill default > TierFull (conservative)
func determineSpawnTier(skillName string, lightFlag, fullFlag bool) string {
	// Explicit flags take precedence
	if lightFlag {
		return spawn.TierLight
	}
	if fullFlag {
		return spawn.TierFull
	}
	// Check userconfig for default tier override
	cfg, err := userconfig.Load()
	if err == nil && cfg.GetDefaultTier() != "" {
		return cfg.GetDefaultTier()
	}
	// Fall back to skill default
	return spawn.DefaultTierForSkill(skillName)
}

// UsageThresholds defines the thresholds for proactive rate limit monitoring.
// These are checked BEFORE spawn to warn or block based on current usage.
type UsageThresholds struct {
	// WarnThreshold is the usage % above which to show a warning (default 80).
	WarnThreshold float64
	// BlockThreshold is the usage % above which to block spawn unless auto-switch succeeds (default 95).
	BlockThreshold float64
}

// DefaultUsageThresholds returns the default proactive monitoring thresholds.
func DefaultUsageThresholds() UsageThresholds {
	return UsageThresholds{
		WarnThreshold:  80,
		BlockThreshold: 95,
	}
}

// UsageCheckResult contains the result of a pre-spawn usage check.
type UsageCheckResult struct {
	// Warning is set if usage exceeds warning threshold.
	Warning string
	// Blocked is true if spawn should be blocked (usage critical and switch failed).
	Blocked bool
	// BlockReason explains why spawn was blocked.
	BlockReason string
	// Switched is true if account was auto-switched.
	Switched bool
	// SwitchReason explains the switch.
	SwitchReason string
	// CapacityInfo is the current account capacity (for telemetry).
	CapacityInfo *account.CapacityInfo
}

// checkUsageBeforeSpawn performs proactive rate limit monitoring.
// It checks usage BEFORE spawn and:
// 1. Warns at 80% usage (5h or weekly)
// 2. Attempts auto-switch at 95% usage
// 3. Blocks spawn at 95% if auto-switch fails
//
// Returns UsageCheckResult for telemetry and a blocking error if spawn should not proceed.
func checkUsageBeforeSpawn() (*UsageCheckResult, error) {
	result := &UsageCheckResult{}

	// Get thresholds from environment or use defaults
	thresholds := DefaultUsageThresholds()
	if envVal := os.Getenv("ORCH_USAGE_WARN_THRESHOLD"); envVal != "" {
		if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
			thresholds.WarnThreshold = val
		}
	}
	if envVal := os.Getenv("ORCH_USAGE_BLOCK_THRESHOLD"); envVal != "" {
		if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
			thresholds.BlockThreshold = val
		}
	}

	// Get current account capacity
	capacity, err := account.GetCurrentCapacity()
	if err != nil {
		// Log warning but don't block - can't check capacity
		fmt.Fprintf(os.Stderr, "Warning: could not check usage: %v\n", err)
		return result, nil
	}

	if capacity.Error != "" {
		fmt.Fprintf(os.Stderr, "Warning: usage check failed: %s\n", capacity.Error)
		return result, nil
	}

	result.CapacityInfo = capacity

	// Determine effective usage (use the tighter constraint)
	fiveHourUsed := capacity.FiveHourUsed
	weeklyUsed := capacity.SevenDayUsed
	effectiveUsage := fiveHourUsed
	usageType := "5h session"
	if weeklyUsed > fiveHourUsed {
		effectiveUsage = weeklyUsed
		usageType = "weekly"
	}

	// Check for blocking threshold (95%)
	if effectiveUsage >= thresholds.BlockThreshold {
		// Try auto-switch first
		switchResult, switchErr := tryAutoSwitchForSpawn()
		if switchErr == nil && switchResult.Switched {
			result.Switched = true
			result.SwitchReason = switchResult.Reason
			// Update capacity after switch
			newCapacity, _ := account.GetCurrentCapacity()
			if newCapacity != nil && newCapacity.Error == "" {
				result.CapacityInfo = newCapacity
			}
			fmt.Printf("🔄 Auto-switched account: %s\n", switchResult.Reason)
			return result, nil
		}

		// Switch failed or no alternate account - block spawn
		result.Blocked = true
		result.BlockReason = fmt.Sprintf("usage critical: %s at %.1f%% (threshold: %.0f%%)", usageType, effectiveUsage, thresholds.BlockThreshold)

		// Log the blocked spawn for pattern analysis
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "spawn.blocked.rate_limit",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"five_hour_used": fiveHourUsed,
				"weekly_used":    weeklyUsed,
				"threshold":      thresholds.BlockThreshold,
				"switch_failed":  switchErr != nil || !switchResult.Switched,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log blocked spawn: %v\n", err)
		}

		return result, fmt.Errorf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│  🛑 SPAWN BLOCKED: Rate Limit Critical                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  Current usage: %s at %.1f%%                                                │
│  Block threshold: %.0f%%                                                     │
│                                                                             │
│  Auto-switch failed: No alternate account with sufficient headroom.         │
│                                                                             │
│  Options:                                                                   │
│    • Wait for limit to reset (see 'orch usage' for reset time)              │
│    • Add another account: orch account add <name>                           │
│    • Override: ORCH_USAGE_BLOCK_THRESHOLD=100 orch spawn ...                │
└─────────────────────────────────────────────────────────────────────────────┘
`, usageType, effectiveUsage, thresholds.BlockThreshold)
	}

	// Check for warning threshold (80%)
	if effectiveUsage >= thresholds.WarnThreshold {
		result.Warning = fmt.Sprintf("⚠️  Usage warning: %s at %.1f%% (warn at %.0f%%, block at %.0f%%)", usageType, effectiveUsage, thresholds.WarnThreshold, thresholds.BlockThreshold)
		fmt.Fprintf(os.Stderr, "%s\n", result.Warning)

		// Log the warning for pattern analysis
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "spawn.warning.rate_limit",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"five_hour_used": fiveHourUsed,
				"weekly_used":    weeklyUsed,
				"threshold":      thresholds.WarnThreshold,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log usage warning: %v\n", err)
		}
	}

	return result, nil
}

// tryAutoSwitchForSpawn attempts to auto-switch to a better account for spawning.
// This is called when usage is at blocking threshold.
func tryAutoSwitchForSpawn() (*account.AutoSwitchResult, error) {
	// Use lower thresholds to trigger switch more aggressively
	thresholds := account.AutoSwitchThresholds{
		FiveHourThreshold: 90, // Lower than default 80 since we're already at 95
		WeeklyThreshold:   90, // Lower than default 90
		MinHeadroomDelta:  5,  // Lower delta requirement for emergency switch
	}

	return account.AutoSwitchIfNeeded(thresholds)
}

// checkAndAutoSwitchAccount checks if the current account is over usage thresholds
// and automatically switches to a better account if available.
// Returns nil if no switch was needed or switch succeeded.
// Logs the switch action if one occurs.
func checkAndAutoSwitchAccount() error {
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

// validateModeModelCombo checks for known invalid mode+model combinations.
// Returns a warning error (non-blocking) if an invalid combination is detected.
func validateModeModelCombo(backend string, resolvedModel model.ModelSpec) error {
	// Invalid combination: opencode + opus
	// Opus requires Claude Code CLI auth, opencode backend creates zombie agents
	if backend == "opencode" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "opus") {
		return fmt.Errorf(`Warning: opencode backend with opus model may fail (auth blocked).
  Recommendation: Use --model sonnet (default) or let auto-selection use claude backend`)
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
func runSpawnWithSkillInternal(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool, daemonDriven bool) error {
	// Check for --bypass-triage flag (required for manual spawns)
	// Daemon-driven spawns skip this check (issue already triaged)
	if !daemonDriven && !spawnBypassTriage {
		return showTriageBypassRequired(skillName, task)
	}

	// Log the triage bypass for Phase 2 review (only for manual bypasses, not daemon-driven)
	if !daemonDriven && spawnBypassTriage {
		logTriageBypass(skillName, task)
	}

	// Check concurrency limit before spawning
	if err := checkConcurrencyLimit(); err != nil {
		return err
	}

	// Proactive rate limit monitoring: warn at 80%, block at 95%
	// This replaces the old checkAndAutoSwitchAccount() with more aggressive monitoring
	usageCheckResult, usageErr := checkUsageBeforeSpawn()
	if usageErr != nil {
		// usageErr contains formatted blocking message
		return usageErr
	}

	// Get project directory early for hotspot check
	var preCheckDir string
	if spawnWorkdir != "" {
		if absPath, err := filepath.Abs(spawnWorkdir); err == nil {
			preCheckDir = absPath
		}
	} else {
		preCheckDir, _ = os.Getwd()
	}

	// STRATEGIC-FIRST ORCHESTRATION: Check for hotspots in task target area
	// In hotspot areas (5+ bugs, persistent failures), strategic approach is required
	// Tactical debugging only allowed with --force (requires justification)
	if preCheckDir != "" {
		if hotspotResult, err := RunHotspotCheckForSpawn(preCheckDir, task); err == nil && hotspotResult != nil {
			// Strategic-first gate: block tactical spawns to hotspot areas unless:
			// 1. --force flag provided (user explicitly overrides with justification), OR
			// 2. Skill is architect (strategic approach, not tactical)
			isStrategicSkill := skillName == "architect"

			if !spawnForce && !isStrategicSkill {
				// BLOCKING: Strategic approach required in hotspot area
				fmt.Fprint(os.Stderr, hotspotResult.Warning)
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "🚫 STRATEGIC-FIRST ORCHESTRATION")
				fmt.Fprintln(os.Stderr, "   Tactical debugging blocked in hotspot areas.")
				fmt.Fprintln(os.Stderr, "   ")
				fmt.Fprintln(os.Stderr, "   REQUIRED: Spawn architect first to understand root cause")
				fmt.Fprintf(os.Stderr, "   orch spawn architect \"%s\"\n", task)
				fmt.Fprintln(os.Stderr, "   ")
				fmt.Fprintln(os.Stderr, "   To override (requires justification):")
				fmt.Fprintf(os.Stderr, "   orch spawn --force %s \"%s\"\n", skillName, task)
				fmt.Fprintln(os.Stderr, "")
				return fmt.Errorf("strategic-first gate: architect required in hotspot area (use --force to override)")
			}

			// Strategic skill or --force used: print warning but allow
			if spawnForce {
				fmt.Fprint(os.Stderr, hotspotResult.Warning)
				fmt.Fprintln(os.Stderr, "⚠️  --force used: bypassing strategic-first gate")
				fmt.Fprintln(os.Stderr, "")
			} else {
				// Architect skill: print info message
				fmt.Fprint(os.Stderr, hotspotResult.Warning)
				fmt.Fprintln(os.Stderr, "✓ Strategic approach: architect skill in hotspot area")
				fmt.Fprintln(os.Stderr, "")
			}
		}
	}

	// Get project directory - use --workdir if provided, otherwise current directory
	var projectDir string
	var err error
	if spawnWorkdir != "" {
		// User specified target directory via --workdir
		projectDir, err = filepath.Abs(spawnWorkdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		// Verify directory exists
		if stat, err := os.Stat(projectDir); err != nil {
			return fmt.Errorf("workdir does not exist: %s", projectDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", projectDir)
		}
	} else {
		// Default: use current working directory
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Get project name from directory
	projectName := filepath.Base(projectDir)

	// Check and optionally auto-initialize scaffolding
	if err := ensureOrchScaffolding(projectDir, spawnAutoInit, spawnNoTrack); err != nil {
		return err
	}

	// Load skill content with dependencies (e.g., worker-base patterns)
	loader := skills.DefaultLoader()

	// First load raw skill content (without dependencies) to detect skill type
	// This is needed because LoadSkillWithDependencies puts dependency content first,
	// which means the main skill's frontmatter isn't at the start of the combined content
	isOrchestrator := false
	isMetaOrchestrator := false
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
	workspaceName := spawn.GenerateWorkspaceName(projectName, skillName, task, spawn.WorkspaceNameOptions{
		IsMetaOrchestrator: isMetaOrchestrator,
		IsOrchestrator:     isOrchestrator,
	})

	// Now load full skill content with dependencies for the actual spawn
	skillContent, err := loader.LoadSkillWithDependencies(skillName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load skill '%s': %v\n", skillName, err)
		skillContent = "" // Continue without skill content
	}

	// Determine beads ID - either from flag, create new issue, or skip if --no-track
	// Orchestrators skip beads tracking entirely - they're interactive sessions with Dylan,
	// not autonomous tasks. SESSION_HANDOFF.md is richer than beads comments.
	skipBeadsForOrchestrator := isOrchestrator || isMetaOrchestrator
	beadsID, err := determineBeadsID(projectName, skillName, task, spawnIssue, spawnNoTrack || skipBeadsForOrchestrator, createBeadsIssue)
	if err != nil {
		return fmt.Errorf("failed to determine beads ID: %w", err)
	}
	if skipBeadsForOrchestrator {
		fmt.Println("Skipping beads tracking (orchestrator session)")
	} else if spawnNoTrack {
		fmt.Println("Skipping beads tracking (--no-track)")
	}

	// Check for retry patterns on existing issues - surface to prevent blind respawning
	// Skip for orchestrators since they don't use beads tracking
	if !spawnNoTrack && !skipBeadsForOrchestrator && spawnIssue != "" {
		if stats, err := verify.GetFixAttemptStats(beadsID); err == nil && stats.IsRetryPattern() {
			warning := verify.FormatRetryWarning(stats)
			if warning != "" {
				fmt.Fprintf(os.Stderr, "\n%s\n", warning)
			}
		}
	}

	// DISABLED: Dependency check gate (Jan 4, 2026)
	// This was added to prevent spawning on issues with unresolved dependencies,
	// but it added friction without clear benefit. Dependencies are informational,
	// not blocking - agents can often make progress even if dependencies exist.
	// See: .kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md
	/*
		if !spawnNoTrack && spawnIssue != "" && !spawnForce {
			blockers, err := beads.CheckBlockingDependencies(beadsID)
			// ... gate logic disabled ...
		}
	*/
	_ = spawnForce // silence unused variable warning (flag still exists but doesn't gate)

	// Check if issue is already being worked on (prevent duplicate spawns)
	// Skip for orchestrators since they don't use beads tracking
	if !spawnNoTrack && !skipBeadsForOrchestrator && spawnIssue != "" {
		if issue, err := verify.GetIssue(beadsID); err == nil {
			if issue.Status == "closed" {
				return fmt.Errorf("issue %s is already closed", beadsID)
			}
			if issue.Status == "in_progress" {
				// Check if there's a truly active agent for this issue
				// OpenCode persists sessions to disk, so we must verify liveness not just existence
				client := opencode.NewClient(serverURL)
				sessions, _ := client.ListSessions("")
				for _, s := range sessions {
					if strings.Contains(s.Title, beadsID) {
						// Session exists - but is it actually active (recently updated)?
						// Use 30 minute threshold - if no activity, session is stale
						if client.IsSessionActive(s.ID, 30*time.Minute) {
							return fmt.Errorf("issue %s is already in_progress with active agent (session %s). Use 'orch send %s' to interact or 'orch abandon %s' to restart", beadsID, s.ID, s.ID, beadsID)
						}
						// Session exists but is stale - log and continue (allow respawn)
						fmt.Fprintf(os.Stderr, "Note: found stale session %s for issue %s (no activity in 30m)\n", s.ID[:12], beadsID)
					}
				}
				// No active session - check if Phase: Complete was reported
				// If so, orchestrator needs to run 'orch complete' before respawning
				if complete, err := verify.IsPhaseComplete(beadsID); err == nil && complete {
					return fmt.Errorf("issue %s has Phase: Complete but is not closed. Run 'orch complete %s' first", beadsID, beadsID)
				}
				// In progress but no active agent and not Phase: Complete - warn but allow respawn
				fmt.Fprintf(os.Stderr, "Warning: issue %s is in_progress but no active agent found. Respawning.\n", beadsID)
			}
		}
	}

	// Update beads issue status to in_progress (only if tracking a real issue)
	// Skip for orchestrators since they don't use beads tracking
	if !spawnNoTrack && !skipBeadsForOrchestrator && spawnIssue != "" {
		if err := verify.UpdateIssueStatus(beadsID, "in_progress"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
			// Continue anyway
		}
	}

	// Resolve model - convert aliases to full format
	resolvedModel := model.Resolve(spawnModel)

	// Validate flash model - TPM rate limits make it unusable
	if resolvedModel.Provider == "google" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "flash") {
		return fmt.Errorf(`
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

	// Parse skill requirements to determine what context to gather
	requires := spawn.ParseSkillRequires(skillContent)

	// Gather context based on skill requirements (or fall back to default behavior)
	var kbContext string
	var gapAnalysis *spawn.GapAnalysis
	if !spawnSkipArtifactCheck {
		if requires != nil && requires.HasRequirements() {
			// Skill-driven context gathering
			fmt.Printf("Gathering context (skill requires: %s)\n", requires.String())
			kbContext = spawn.GatherRequiredContext(requires, task, beadsID, projectDir)
			// For skill-driven context, create a basic gap analysis from the results
			// This is a placeholder - skills may provide their own gap info
			gapAnalysis = spawn.AnalyzeGaps(nil, task)
		} else {
			// Fall back to default kb context check with full gap analysis
			gapResult := runPreSpawnKBCheckFull(task)
			kbContext = gapResult.Context
			gapAnalysis = gapResult.GapAnalysis
		}

		// Check gap gating - may block spawn if context quality is too low
		if err := checkGapGating(gapAnalysis, spawnGateOnGap, spawnSkipGapGate, spawnGapThreshold); err != nil {
			return err
		}

		// Record gap for learning loop (if gaps detected)
		if gapAnalysis != nil && gapAnalysis.HasGaps {
			recordGapForLearning(gapAnalysis, skillName, task)
		}

		// Log if skip-gap-gate was used (documents conscious bypass)
		if spawnSkipGapGate && gapAnalysis != nil && gapAnalysis.ShouldBlockSpawn(spawnGapThreshold) {
			fmt.Fprintf(os.Stderr, "⚠️  Bypassing gap gate (--skip-gap-gate): context quality %d\n", gapAnalysis.ContextQuality)
			// Log the bypass for pattern detection
			logger := events.NewLogger(events.DefaultLogPath())
			event := events.Event{
				Type:      "gap.gate.bypassed",
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"task":            task,
					"context_quality": gapAnalysis.ContextQuality,
					"beads_id":        beadsID,
					"skill":           skillName,
				},
			}
			if err := logger.Log(event); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log gap bypass: %v\n", err)
			}
		}
	} else {
		fmt.Println("Skipping context check (--skip-artifact-check)")
	}

	// Determine spawn tier
	tier := determineSpawnTier(skillName, spawnLight, spawnFull)

	// Extract reproduction info for bug issues
	var isBug bool
	var reproSteps string
	if !spawnNoTrack && beadsID != "" {
		if reproResult, err := verify.GetReproForCompletion(beadsID); err == nil && reproResult != nil {
			isBug = reproResult.IsBug
			reproSteps = reproResult.Repro
			if isBug && reproSteps != "" {
				fmt.Printf("🐛 Bug issue detected - reproduction steps included in context\n")
			}
		}
	}

	// Build usage info from check result (for telemetry)
	var usageInfo *spawn.UsageInfo
	if usageCheckResult != nil && usageCheckResult.CapacityInfo != nil {
		usageInfo = &spawn.UsageInfo{
			FiveHourUsed: usageCheckResult.CapacityInfo.FiveHourUsed,
			SevenDayUsed: usageCheckResult.CapacityInfo.SevenDayUsed,
			AccountEmail: usageCheckResult.CapacityInfo.Email,
			AutoSwitched: usageCheckResult.Switched,
			SwitchReason: usageCheckResult.SwitchReason,
		}
	}

	// Load project config (used for server ports, etc.)
	projCfg, _ := config.Load(projectDir)

	// Determine spawn backend with auto-selection based on model
	// Priority:
	//   1. Explicit --backend flag (highest priority)
	//   2. Explicit --opus flag (forces claude mode)
	//   3. Auto-selection based on --model flag (opus → claude, sonnet → opencode)
	//   4. Config default (spawn_mode in project config)
	//   5. Default to opencode
	spawnBackend := "opencode"

	if spawnBackendFlag != "" {
		// Explicit --backend flag: highest priority
		spawnBackend = spawnBackendFlag
		// Validate backend value
		if spawnBackend != "claude" && spawnBackend != "opencode" {
			return fmt.Errorf("invalid --backend value: %s (must be 'claude' or 'opencode')", spawnBackend)
		}
	} else if spawnOpus {
		// Explicit --opus flag: use claude CLI
		spawnBackend = "claude"
	} else if spawnModel != "" {
		// Auto-select backend based on model
		modelLower := strings.ToLower(spawnModel)
		if modelLower == "opus" || strings.Contains(modelLower, "opus") {
			// Opus model: use claude CLI (Max subscription)
			spawnBackend = "claude"
			fmt.Println("Auto-selected claude backend for opus model")
		} else if modelLower == "sonnet" || strings.Contains(modelLower, "sonnet") {
			// Sonnet model: use opencode (pay-per-token API)
			spawnBackend = "opencode"
		}
		// Other models default to opencode (handled by default value above)
	} else if projCfg != nil && projCfg.SpawnMode == "claude" {
		// Config default: respect project spawn_mode setting
		spawnBackend = "claude"
	}

	// Validate mode+model combination
	if err := validateModeModelCombo(spawnBackend, resolvedModel); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
	}

	// Build spawn config
	cfg := &spawn.Config{
		Task:               task,
		SkillName:          skillName,
		Project:            projectName,
		ProjectDir:         projectDir,
		WorkspaceName:      workspaceName,
		SkillContent:       skillContent,
		BeadsID:            beadsID,
		Phases:             spawnPhases,
		Mode:               spawnMode,
		Validation:         spawnValidation,
		Model:              resolvedModel.Format(),
		MCP:                spawnMCP,
		Tier:               tier,
		NoTrack:            spawnNoTrack || skipBeadsForOrchestrator,
		SkipArtifactCheck:  spawnSkipArtifactCheck,
		KBContext:          kbContext,
		IncludeServers:     spawn.DefaultIncludeServersForSkill(skillName),
		GapAnalysis:        gapAnalysis,
		IsBug:              isBug,
		ReproSteps:         reproSteps,
		IsOrchestrator:     isOrchestrator,
		IsMetaOrchestrator: isMetaOrchestrator,
		UsageInfo:          usageInfo,
		SpawnMode:          spawnBackend,
	}

	// Pre-spawn token estimation and validation
	if err := spawn.ValidateContextSize(cfg); err != nil {
		return fmt.Errorf("pre-spawn validation failed: %w", err)
	}

	// Warn about large contexts (but don't block)
	if shouldWarn, warning := spawn.ShouldWarnAboutSize(cfg); shouldWarn {
		fmt.Fprintf(os.Stderr, "%s", warning)
	}

	// Check for existing workspace before writing context
	// This prevents accidentally overwriting SESSION_HANDOFF.md from completed sessions
	// Note: With unique suffixes in workspace names (since Jan 2026), collisions are rare
	// but this provides an extra safety net and meaningful error messages
	if err := checkWorkspaceExists(cfg.WorkspacePath(), spawnForce); err != nil {
		return err
	}

	// Write SPAWN_CONTEXT.md
	if err := spawn.WriteContext(cfg); err != nil {
		return fmt.Errorf("failed to write spawn context: %w", err)
	}

	// Record spawn in session (if session is active)
	if sessionStore, err := session.New(""); err == nil {
		if err := sessionStore.RecordSpawn(beadsID, skillName, task, projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record spawn in session: %v\n", err)
		}
	}

	// Generate minimal prompt
	minimalPrompt := spawn.MinimalPrompt(cfg)

	// Spawn mode: inline (blocking TUI), tmux (opt-in for workers, default for orchestrators), claude (tmux), or headless (default for workers)
	if inline {
		// Inline mode (blocking) - run in current terminal with TUI
		return runSpawnInline(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Explicit --headless flag overrides all other mode decisions
	if headless {
		return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Claude mode: Use tmux + claude CLI
	if cfg.SpawnMode == "claude" {
		return runSpawnClaude(serverURL, cfg, beadsID, skillName, task, attach)
	}

	// Orchestrator-type skills default to tmux mode (visible interaction)
	// Workers default to headless mode (automation-friendly)
	useTmux := tmux || attach || cfg.IsOrchestrator
	if useTmux {
		// Tmux mode - visible, interruptible
		// Default for orchestrator skills, opt-in for workers
		return runSpawnTmux(serverURL, cfg, minimalPrompt, beadsID, skillName, task, attach)
	}

	// Default for workers: Headless mode - spawn via HTTP API (automation-friendly, no TUI overhead)
	return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
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

// registerOrchestratorSession registers an orchestrator session in the session registry.
// This is called after successful spawn for orchestrator-type skills.
// Workers do not use the session registry - they use beads for lifecycle tracking.
func registerOrchestratorSession(cfg *spawn.Config, sessionID, task string) {
	if !cfg.IsOrchestrator && !cfg.IsMetaOrchestrator {
		return // Only register orchestrator sessions
	}

	registry := session.NewRegistry("")
	orchSession := session.OrchestratorSession{
		WorkspaceName: cfg.WorkspaceName,
		SessionID:     sessionID,
		ProjectDir:    cfg.ProjectDir,
		SpawnTime:     time.Now(),
		Goal:          task,
		Status:        "active",
	}
	if err := registry.Register(orchSession); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to register orchestrator session: %v\n", err)
	}
}

// addGapAnalysisToEventData adds gap analysis information to an event data map.
// This enables tracking of context gaps for pattern analysis and dashboard surfacing.
func addGapAnalysisToEventData(eventData map[string]interface{}, gapAnalysis *spawn.GapAnalysis) {
	if gapAnalysis == nil {
		return
	}

	eventData["gap_has_gaps"] = gapAnalysis.HasGaps
	eventData["gap_context_quality"] = gapAnalysis.ContextQuality

	if gapAnalysis.HasGaps {
		eventData["gap_should_warn"] = gapAnalysis.ShouldWarnAboutGaps()
		eventData["gap_match_total"] = gapAnalysis.MatchStats.TotalMatches
		eventData["gap_match_constraints"] = gapAnalysis.MatchStats.ConstraintCount
		eventData["gap_match_decisions"] = gapAnalysis.MatchStats.DecisionCount
		eventData["gap_match_investigations"] = gapAnalysis.MatchStats.InvestigationCount

		// Capture gap types for pattern analysis
		var gapTypes []string
		for _, gap := range gapAnalysis.Gaps {
			gapTypes = append(gapTypes, string(gap.Type))
		}
		if len(gapTypes) > 0 {
			eventData["gap_types"] = gapTypes
		}
	}
}

// addUsageInfoToEventData adds usage information to an event data map.
// This enables tracking of rate limit patterns and account utilization at spawn time.
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

// formatContextQualitySummary formats context quality for spawn summary output.
// Returns a formatted string with visual indicators for gap severity.
// This is the "prominent" surfacing that makes gaps hard to ignore.
func formatContextQualitySummary(gapAnalysis *spawn.GapAnalysis) string {
	if gapAnalysis == nil {
		return "not checked"
	}

	quality := gapAnalysis.ContextQuality

	// Determine visual indicator and label based on quality level
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

	// Format the summary line
	summary := fmt.Sprintf("%s %d/100 (%s)", indicator, quality, label)

	// Add match breakdown for transparency
	if gapAnalysis.MatchStats.TotalMatches > 0 {
		summary += fmt.Sprintf(" - %d matches", gapAnalysis.MatchStats.TotalMatches)
		if gapAnalysis.MatchStats.ConstraintCount > 0 {
			summary += fmt.Sprintf(" (%d constraints)", gapAnalysis.MatchStats.ConstraintCount)
		}
	}

	return summary
}

// printSpawnSummaryWithGapWarning prints the spawn summary with prominent gap warnings.
// This ensures gaps are visible in the final output, not just during context gathering.
func printSpawnSummaryWithGapWarning(gapAnalysis *spawn.GapAnalysis) {
	if gapAnalysis == nil || !gapAnalysis.ShouldWarnAboutGaps() {
		return
	}

	// Print a prominent warning box for critical gaps
	if gapAnalysis.HasCriticalGaps() || gapAnalysis.ContextQuality < 20 {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "┌─────────────────────────────────────────────────────────────┐\n")
		fmt.Fprintf(os.Stderr, "│  ⚠️  GAP WARNING: Agent spawned with limited context         │\n")
		fmt.Fprintf(os.Stderr, "├─────────────────────────────────────────────────────────────┤\n")
		fmt.Fprintf(os.Stderr, "│  Agent may compensate by guessing patterns/conventions.    │\n")
		fmt.Fprintf(os.Stderr, "│  Consider: kn decide / kn constrain / kb create            │\n")
		fmt.Fprintf(os.Stderr, "└─────────────────────────────────────────────────────────────┘\n")
	}
}

// runSpawnInline spawns the agent inline (blocking) - original behavior.
func runSpawnInline(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	// Spawn opencode session
	client := opencode.NewClient(serverURL)
	// Format title with beads ID so orch status can match sessions
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)
	cmd := client.BuildSpawnCommand(minimalPrompt, sessionTitle, cfg.Model)
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
	}

	// Register agent in general registry
	registerAgent(cfg, result.SessionID, "", registry.ModeHeadless, cfg.Model)

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
	client := opencode.NewClient(serverURL)

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

	// Start background cleanup goroutine
	result.StartBackgroundCleanup()

	// Register agent in general registry
	registerAgent(cfg, sessionID, "", registry.ModeHeadless, cfg.Model)

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

// startHeadlessSession starts an opencode session and extracts the session ID.
// Returns the result with session ID and resources for cleanup.
// Note: Uses CLI mode instead of HTTP API because OpenCode's HTTP API ignores the model parameter.
// CLI mode correctly honors the --model flag.
// See: .kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md
func startHeadlessSession(client *opencode.Client, serverURL, sessionTitle, minimalPrompt string, cfg *spawn.Config) (*headlessSpawnResult, error) {
	cmd := client.BuildSpawnCommand(minimalPrompt, sessionTitle, cfg.Model)
	cmd.Dir = cfg.ProjectDir
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		spawnErr := spawn.WrapSpawnError(err, "Failed to get stdout pipe")
		return nil, spawnErr
	}

	// Discard stderr in headless mode (no TUI to display it)
	cmd.Stderr = nil

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
		spawnErr := spawn.WrapSpawnError(err, "Failed to extract session ID")
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
	var sessionName string
	var err error

	// Meta-orchestrators go into 'meta-orchestrator' tmux session
	// Orchestrator skills go into the 'orchestrator' tmux session
	// Workers go into 'workers-{project}' session
	if cfg.IsMetaOrchestrator {
		sessionName, err = tmux.EnsureMetaOrchestratorSession()
	} else if cfg.IsOrchestrator {
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

	// Build opencode command using tmux package
	opencodeCmd := tmux.BuildOpencodeAttachCommand(&tmux.OpencodeAttachConfig{
		ServerURL:  serverURL,
		ProjectDir: cfg.ProjectDir,
		Model:      cfg.Model,
	})

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

	// Capture session ID from API with retry (OpenCode may not have registered yet)
	// Uses 3 attempts with 500ms initial delay, doubling each time (500ms, 1s, 2s)
	// Matches by directory + creation time (within 30s), not by title
	client := opencode.NewClient(serverURL)
	sessionID, _ := client.FindRecentSessionWithRetry(cfg.ProjectDir, 3, 500*time.Millisecond)
	// Note: We silently ignore errors here since window_id is sufficient for tmux monitoring

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
	}

	// Register agent in general registry
	registerAgent(cfg, sessionID, windowTarget, registry.ModeTmux, cfg.Model)

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

	// Focus the newly created window
	selectCmd := exec.Command("tmux", "select-window", "-t", windowTarget)
	if err := selectCmd.Run(); err != nil {
		// Non-fatal - window was created successfully
		fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
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

	// Focus the newly created window
	selectCmd := exec.Command("tmux", "select-window", "-t", result.Window)
	if err := selectCmd.Run(); err != nil {
		// Non-fatal
		fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
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

// determineBeadsID determines the beads ID to use for an agent.
// It returns an error if beads issue creation fails and --no-track is not set.
// The createBeadsFn parameter allows for dependency injection in tests.
func determineBeadsID(projectName, skillName, task, spawnIssue string, spawnNoTrack bool, createBeadsFn func(string, string, string) (string, error)) (string, error) {
	// If explicit issue ID provided via --issue flag, resolve it to full ID
	if spawnIssue != "" {
		return resolveShortBeadsID(spawnIssue)
	}

	// If --no-track flag is set, generate a local-only ID
	if spawnNoTrack {
		return fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix()), nil
	}

	// Create a new beads issue (default behavior)
	beadsID, err := createBeadsFn(projectName, skillName, task)
	if err != nil {
		return "", fmt.Errorf("failed to create beads issue: %w", err)
	}

	return beadsID, nil
}

// createBeadsIssue creates a new beads issue for tracking the agent.
// It uses the beads RPC client when available, falling back to the bd CLI.
func createBeadsIssue(projectName, skillName, task string) (string, error) {
	// Build issue title
	title := fmt.Sprintf("[%s] %s: %s", projectName, skillName, truncate(task, 50))

	// Try RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()

			issue, err := client.Create(&beads.CreateArgs{
				Title:     title,
				IssueType: "task",
				Priority:  2, // Default P2
			})
			if err == nil {
				return issue.ID, nil
			}
			// Fall through to CLI fallback on RPC error
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackCreate(title, "", "task", 2, nil)
	if err != nil {
		return "", err
	}

	return issue.ID, nil
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

// GapCheckResult contains the results of a pre-spawn gap check.
type GapCheckResult struct {
	Context     string             // Formatted context to include in SPAWN_CONTEXT.md
	GapAnalysis *spawn.GapAnalysis // Gap analysis results for further processing
	Blocked     bool               // True if spawn should be blocked due to gaps
	BlockReason string             // Reason for blocking (if Blocked is true)
}

// runPreSpawnKBCheck runs kb context check before spawning an agent.
// Returns formatted context string to include in SPAWN_CONTEXT.md, or empty string if no matches.
// Also performs gap analysis and displays warnings for sparse or missing context.
func runPreSpawnKBCheck(task string) string {
	result := runPreSpawnKBCheckFull(task)
	return result.Context
}

// runPreSpawnKBCheckFull runs kb context check with full gap analysis results.
// This allows callers to access gap analysis for gating decisions.
func runPreSpawnKBCheckFull(task string) *GapCheckResult {
	gcr := &GapCheckResult{}

	// Extract keywords from task description
	// Try with 3 keywords first (more specific), fall back to 1 keyword (more broad)
	keywords := spawn.ExtractKeywords(task, 3)
	if keywords == "" {
		// Perform gap analysis even when no keywords extracted
		gcr.GapAnalysis = spawn.AnalyzeGaps(nil, task)
		if gcr.GapAnalysis.ShouldWarnAboutGaps() {
			// Use prominent warning format for better visibility
			fmt.Fprintf(os.Stderr, "%s", gcr.GapAnalysis.FormatProminentWarning())
		}
		return gcr
	}

	fmt.Printf("Checking kb context for: %q\n", keywords)

	// Run kb context check
	result, err := spawn.RunKBContextCheck(keywords)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
		return gcr
	}

	// If no matches with multiple keywords, try with just the first keyword
	if result == nil || !result.HasMatches {
		firstKeyword := spawn.ExtractKeywords(task, 1)
		if firstKeyword != "" && firstKeyword != keywords {
			fmt.Printf("Trying broader search for: %q\n", firstKeyword)
			result, err = spawn.RunKBContextCheck(firstKeyword)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
				return gcr
			}
		}
	}

	// Perform gap analysis to detect context gaps
	gcr.GapAnalysis = spawn.AnalyzeGaps(result, keywords)
	if gcr.GapAnalysis.ShouldWarnAboutGaps() {
		// Use prominent warning format for better visibility
		fmt.Fprintf(os.Stderr, "%s", gcr.GapAnalysis.FormatProminentWarning())
	}

	if result == nil || !result.HasMatches {
		fmt.Println("No prior knowledge found.")
		return gcr
	}

	// Always include kb context in spawn - the orchestrator has already decided to spawn
	// No interactive prompt needed; context is automatically included
	fmt.Printf("Found %d relevant context entries - including in spawn context.\n", len(result.Matches))

	// Include gap summary in spawn context if there are significant gaps
	contextContent := spawn.FormatContextForSpawn(result)
	if gapSummary := gcr.GapAnalysis.FormatGapSummary(); gapSummary != "" {
		contextContent = gapSummary + "\n\n" + contextContent
	}

	gcr.Context = contextContent
	return gcr
}

// checkGapGating checks if spawn should be blocked due to context gaps.
// Returns an error if spawn should be blocked, nil otherwise.
func checkGapGating(gapAnalysis *spawn.GapAnalysis, gateEnabled, skipGate bool, threshold int) error {
	// Skip gating if not enabled or explicitly bypassed
	if !gateEnabled || skipGate {
		return nil
	}

	// No gap analysis means no gating
	if gapAnalysis == nil {
		return nil
	}

	// Check if quality is below threshold
	if threshold <= 0 {
		threshold = spawn.DefaultGateThreshold
	}

	if gapAnalysis.ShouldBlockSpawn(threshold) {
		// Display the block message
		fmt.Fprintf(os.Stderr, "%s", gapAnalysis.FormatGateBlockMessage())
		return fmt.Errorf("spawn blocked: context quality %d is below threshold %d", gapAnalysis.ContextQuality, threshold)
	}

	return nil
}

// recordGapForLearning records a gap event for the learning loop.
// This builds up a history of gaps that can be used to suggest improvements.
func recordGapForLearning(gapAnalysis *spawn.GapAnalysis, skill, task string) {
	// Load existing tracker
	tracker, err := spawn.LoadTracker()
	if err != nil {
		// Don't fail spawn for learning loop errors
		fmt.Fprintf(os.Stderr, "Warning: failed to load gap tracker: %v\n", err)
		return
	}

	// Record the gap
	tracker.RecordGap(gapAnalysis, skill, task)

	// Check for recurring patterns and display suggestions
	suggestions := tracker.FindRecurringGaps()
	if len(suggestions) > 0 {
		// Only show suggestions if there are high-priority ones
		hasHighPriority := false
		for _, s := range suggestions {
			if s.Priority == "high" && s.Count >= spawn.RecurrenceThreshold {
				hasHighPriority = true
				break
			}
		}
		if hasHighPriority {
			fmt.Fprintf(os.Stderr, "%s", spawn.FormatSuggestions(suggestions))
		}
	}

	// Save tracker
	if err := tracker.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save gap tracker: %v\n", err)
	}
}

// showTriageBypassRequired displays a warning and returns an error when --bypass-triage is not provided.
// This creates friction to encourage the daemon-driven workflow over manual spawning.
func showTriageBypassRequired(skillName, task string) error {
	fmt.Fprintf(os.Stderr, `
┌─────────────────────────────────────────────────────────────────────────────┐
│  ⚠️  TRIAGE BYPASS REQUIRED                                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  Manual spawn requires --bypass-triage flag.                                │
│                                                                             │
│  The preferred workflow is daemon-driven triage:                            │
│    1. Create issue: bd create "task" --type task -l triage:ready            │
│    2. Daemon auto-spawns: orch daemon run                                   │
│                                                                             │
│  Manual spawn is for exceptions only:                                       │
│    - Single urgent item requiring immediate attention                       │
│    - Complex/ambiguous task needing custom context                          │
│    - Skill selection requires orchestrator judgment                         │
│                                                                             │
│  To proceed with manual spawn, add --bypass-triage:                         │
│    orch spawn --bypass-triage %s "%s"                          │
└─────────────────────────────────────────────────────────────────────────────┘

`, skillName, truncate(task, 30))
	return fmt.Errorf("spawn blocked: --bypass-triage flag required for manual spawns")
}

// logTriageBypass logs a triage bypass event to events.jsonl for Phase 2 review.
// This tracks how often manual spawns occur vs daemon-driven spawns.
func logTriageBypass(skillName, task string) {
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "spawn.triage_bypassed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"skill": skillName,
			"task":  task,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log triage bypass: %v\n", err)
	}
}

// registerAgent registers any agent (worker or orchestrator) in the general agent registry.
func registerAgent(cfg *spawn.Config, sessionID, tmuxWindow, mode, modelSpec string) {
	agentReg, err := registry.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open agent registry: %v\n", err)
		return
	}

	agent := &registry.Agent{
		ID:         cfg.WorkspaceName,
		BeadsID:    cfg.BeadsID,
		Mode:       mode,
		SessionID:  sessionID,
		TmuxWindow: tmuxWindow,
		Model:      modelSpec,
		ProjectDir: cfg.ProjectDir,
		Skill:      cfg.SkillName,
		Status:     registry.StateActive,
	}

	if err := agentReg.Register(agent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to register agent in registry: %v\n", err)
		return
	}

	if err := agentReg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save agent registry: %v\n", err)
	}
}
