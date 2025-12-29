// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/question"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/usage"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Global flags
	serverURL string

	// Version information (set at build time via ldflags)
	version   = "dev"
	buildTime = "unknown"
	sourceDir = "unknown" // Absolute path to source directory
	gitHash   = "unknown" // Full git commit hash at build time
)

func main() {
	// Check for stale binary and auto-rebuild if needed
	if shouldAutoRebuild() {
		if err := autoRebuild(); err != nil {
			// Non-fatal: warn and continue with stale binary
			fmt.Fprintf(os.Stderr, "⚠️  Auto-rebuild failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "    Running with stale binary. Manual fix: cd %s && make install\n\n", sourceDir)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "orch-go",
	Short: "OpenCode orchestration CLI",
	Long: `orch-go is a CLI tool for orchestrating OpenCode sessions.

It provides commands for spawning new sessions, sending messages to existing
sessions, and monitoring session events via SSE.`,
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:4096", "OpenCode server URL")

	rootCmd.AddCommand(spawnCmd)
	sendCmd.Flags().BoolVar(&sendAsync, "async", true, "Send message asynchronously (non-blocking)")
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(completeCmd)
	rootCmd.AddCommand(workCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(tailCmd)
	rootCmd.AddCommand(questionCmd)
	rootCmd.AddCommand(abandonCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(focusCmd)
	rootCmd.AddCommand(driftCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(portCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(retriesCmd)
	rootCmd.AddCommand(sessionsCmd)
	rootCmd.AddCommand(actionCmd)
}

var (
	versionSource bool // Show source info and staleness check
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long: `Print version information.

Use --source to see where the binary was built from and check if it's stale.`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionSource {
			runVersionSource()
			return
		}
		fmt.Printf("orch version %s\n", version)
		fmt.Printf("build time: %s\n", buildTime)
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionSource, "source", false, "Show source location and staleness check")
}

// runVersionSource shows where the binary was built from and checks staleness.
func runVersionSource() {
	fmt.Printf("orch version %s\n", version)
	fmt.Printf("build time:  %s\n", buildTime)
	fmt.Printf("source dir:  %s\n", sourceDir)
	fmt.Printf("git hash:    %s\n", gitHash)

	// Check if source directory exists
	if sourceDir == "unknown" {
		fmt.Println("\n⚠️  Source directory not embedded (dev build)")
		return
	}

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Printf("\n⚠️  Source directory not found: %s\n", sourceDir)
		return
	}

	// Check current git hash in source directory
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = sourceDir
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("\n⚠️  Could not get current git hash: %v\n", err)
		return
	}

	currentHash := strings.TrimSpace(string(output))

	// Compare hashes
	if gitHash == "unknown" {
		fmt.Println("\n⚠️  Git hash not embedded (dev build)")
		fmt.Printf("current HEAD: %s\n", currentHash[:12])
	} else if currentHash == gitHash {
		fmt.Println("\nstatus: ✓ UP TO DATE")
	} else {
		fmt.Println("\nstatus: ⚠️  STALE")
		fmt.Printf("binary hash:  %s\n", gitHash[:12])
		fmt.Printf("current HEAD: %s\n", currentHash[:12])
		fmt.Printf("\nrebuild: cd %s && make install\n", sourceDir)
	}
}

// shouldAutoRebuild checks if the binary is stale and auto-rebuild is enabled.
// Auto-rebuild is disabled if ORCH_NO_AUTO_REBUILD=1 is set.
func shouldAutoRebuild() bool {
	// Disable auto-rebuild via environment variable
	if os.Getenv("ORCH_NO_AUTO_REBUILD") == "1" {
		return false
	}

	// Skip for dev builds
	if sourceDir == "unknown" || gitHash == "unknown" {
		return false
	}

	// Check if source directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return false
	}

	// Get current git hash
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = sourceDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	currentHash := strings.TrimSpace(string(output))
	return currentHash != gitHash
}

// autoRebuild rebuilds and re-executes the binary.
// This replaces the current process with the fresh binary.
func autoRebuild() error {
	fmt.Fprintf(os.Stderr, "🔄 Binary is stale, auto-rebuilding...\n")

	// Run make install
	cmd := exec.Command("make", "install")
	cmd.Dir = sourceDir
	cmd.Stdout = os.Stderr // Show build output on stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Rebuilt successfully, re-executing...\n\n")

	// Re-execute the binary with same arguments
	// The new binary is at ~/bin/orch (where make install puts it)
	binaryPath := filepath.Join(os.Getenv("HOME"), "bin", "orch")

	// Use syscall.Exec to replace current process (Unix only)
	// This avoids nested process issues
	return execReplaceProcess(binaryPath, os.Args)
}

// execReplaceProcess replaces the current process with a new executable.
// Uses syscall.Exec to replace the current process (Unix-specific, macOS/Linux).
func execReplaceProcess(path string, args []string) error {
	env := os.Environ()

	// Set flag to prevent infinite rebuild loop
	env = append(env, "ORCH_NO_AUTO_REBUILD=1")

	// syscall.Exec replaces current process - doesn't return on success
	return syscall.Exec(path, args, env)
}

// DefaultMaxAgents is the default maximum number of concurrent agents.
const DefaultMaxAgents = 5

var (
	// Spawn command flags
	spawnSkill             string
	spawnIssue             string
	spawnPhases            string
	spawnMode              string
	spawnValidation        string
	spawnInline            bool   // Run inline (blocking) with TUI
	spawnHeadless          bool   // Run headless via HTTP API (automation/scripting)
	spawnTmux              bool   // Run in tmux window (opt-in, overrides default headless)
	spawnAttach            bool   // Attach to tmux window after spawning
	spawnModel             string // Model to use for standalone spawns
	spawnNoTrack           bool   // Opt-out of beads tracking
	spawnMCP               string // MCP server config (e.g., "playwright", "glass")
	spawnSkipArtifactCheck bool   // Bypass pre-spawn kb context check
	spawnMaxAgents         int    // Maximum concurrent agents (0 = use default or env var)
	spawnAutoInit          bool   // Auto-initialize .orch and .beads if missing
	spawnLight             bool   // Light tier spawn (skips SYNTHESIS.md requirement)
	spawnFull              bool   // Full tier spawn (requires SYNTHESIS.md)
	spawnWorkdir           string // Target project directory (defaults to current directory)
	spawnGateOnGap         bool   // Block spawn if context quality is too low
	spawnSkipGapGate       bool   // Explicitly bypass gap gating (documents conscious decision)
	spawnGapThreshold      int    // Custom gap quality threshold (default 20)
	spawnVerbose           bool   // Show stderr output in real-time for debugging
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [skill] [task]",
	Short: "Spawn a new agent with skill context (default: headless)",
	Long: `Spawn a new OpenCode session with skill context.

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

Concurrency Limiting:
  By default, limits concurrent agents to 5. This prevents runaway agent spawning.
  Configure via --max-agents flag or ORCH_MAX_AGENTS environment variable.
  Set to 0 to disable the limit (not recommended).

Auto-Initialization:
  Use --auto-init to automatically run 'orch init' if .orch/ or .beads/ are missing.
  This is useful for spawning in new projects without prior setup.

Error Visibility:
  --verbose:          Show stderr output in real-time for debugging headless spawns
  
  By default, headless spawns capture stderr and log errors to events.jsonl on failure.
  Use --verbose to see stderr in real-time when debugging spawn issues.

Model aliases: opus, sonnet, haiku (Anthropic), flash, pro (Google)
Full format: provider/model (e.g., anthropic/claude-opus-4-5-20251101)

Examples:
  # Headless mode (default) - automation-friendly, returns immediately
  orch-go spawn investigation "explore the codebase"
  orch-go spawn feature-impl "add feature" --phases implementation,validation
  orch-go spawn --issue proj-123 feature-impl "implement the feature"
  
  # Tmux mode (opt-in) - visible, interruptible
  orch-go spawn --tmux investigation "explore codebase"
  orch-go spawn --attach investigation "explore codebase"      # Tmux + attach immediately
  
  # Inline mode - blocking with TUI, for debugging
  orch-go spawn --inline investigation "explore codebase"
  
  # Gap gating - block spawn on poor context quality
  orch-go spawn --gate-on-gap investigation "important task"   # Block if context < 20
  orch-go spawn --gate-on-gap --gap-threshold 30 feature-impl "critical" # Block if < 30
  orch-go spawn --skip-gap-gate investigation "proceed anyway" # Document bypass
  
  # Other options
  orch-go spawn --model opus investigation "analyze code"      # Use Claude Opus
  orch-go spawn --model flash investigation "quick check"      # Use Gemini Flash
  orch-go spawn --no-track investigation "exploratory work"    # Skip beads tracking
  orch-go spawn --mcp playwright feature-impl "add UI feature" # With Playwright MCP (full browser)
  orch-go spawn --mcp glass feature-impl "verify dashboard"    # With Glass MCP (shared Chrome)
  orch-go spawn --skip-artifact-check investigation "fresh start"  # Skip kb context check
  orch-go spawn --max-agents 10 investigation "task"           # Allow up to 10 concurrent agents
  orch-go spawn --auto-init investigation "new project"        # Auto-init if needed
  orch-go spawn --light feature-impl "quick fix"               # Light tier (no synthesis)
  orch-go spawn --full investigation "deep analysis"           # Full tier (require synthesis)
  orch-go spawn --workdir ~/other-project investigation "task" # Spawn for different project
  orch-go spawn --verbose investigation "debug spawn issues"   # Show stderr in real-time`,
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
	spawnCmd.Flags().StringVar(&spawnValidation, "validation", "tests", "Validation level: none, tests, smoke-test")
	spawnCmd.Flags().BoolVar(&spawnInline, "inline", false, "Run inline (blocking) with TUI")
	spawnCmd.Flags().BoolVar(&spawnHeadless, "headless", false, "Run headless via HTTP API (default behavior, flag is redundant)")
	spawnCmd.Flags().BoolVar(&spawnTmux, "tmux", false, "Run in tmux window (opt-in for visual monitoring)")
	spawnCmd.Flags().BoolVar(&spawnAttach, "attach", false, "Attach to tmux window after spawning (implies --tmux)")
	spawnCmd.Flags().StringVar(&spawnModel, "model", "", "Model alias (opus, sonnet, haiku, flash, pro) or provider/model format")
	spawnCmd.Flags().BoolVar(&spawnNoTrack, "no-track", false, "Opt-out of beads issue tracking (ad-hoc work)")
	spawnCmd.Flags().StringVar(&spawnMCP, "mcp", "", "MCP server config: 'playwright' (full browser) or 'glass' (shared Chrome, requires Chrome --remote-debugging-port=9222)")
	spawnCmd.Flags().BoolVar(&spawnSkipArtifactCheck, "skip-artifact-check", false, "Bypass pre-spawn kb context check")
	spawnCmd.Flags().IntVar(&spawnMaxAgents, "max-agents", 0, "Maximum concurrent agents (default 5, 0 to disable limit, or use ORCH_MAX_AGENTS env var)")
	spawnCmd.Flags().BoolVar(&spawnAutoInit, "auto-init", false, "Auto-initialize .orch and .beads if missing")
	spawnCmd.Flags().BoolVar(&spawnLight, "light", false, "Light tier spawn (skips SYNTHESIS.md requirement on completion)")
	spawnCmd.Flags().BoolVar(&spawnFull, "full", false, "Full tier spawn (requires SYNTHESIS.md for knowledge externalization)")
	spawnCmd.Flags().StringVar(&spawnWorkdir, "workdir", "", "Target project directory (defaults to current directory)")
	spawnCmd.Flags().BoolVar(&spawnGateOnGap, "gate-on-gap", false, "Block spawn if context quality is too low (enforces Gate Over Remind)")
	spawnCmd.Flags().BoolVar(&spawnSkipGapGate, "skip-gap-gate", false, "Explicitly bypass gap gating (documents conscious decision to proceed without context)")
	spawnCmd.Flags().IntVar(&spawnGapThreshold, "gap-threshold", 0, "Custom gap quality threshold (default 20, only used with --gate-on-gap)")
	spawnCmd.Flags().BoolVar(&spawnVerbose, "verbose", false, "Show stderr output in real-time for debugging headless spawns")
}

var sendCmd = &cobra.Command{
	Use:   "send [identifier] [message]",
	Short: "Send a message to an existing session",
	Long: `Send a message to an existing OpenCode session.

The identifier can be:
  - A full session ID (ses_xxx)
  - A beads issue ID (project-xxxx) - looked up via workspace or API
  - A workspace name - looked up via workspace file

The session can be running or completed. Response text is streamed to stdout
as it's received from the agent.

Examples:
  orch-go send ses_abc123 "what files did you modify?"
  orch-go send orch-go-3anf "can you explain the changes?"
  orch-go send og-debug-fix-issue-21dec "status update?"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]
		message := strings.Join(args[1:], " ")
		return runSend(serverURL, identifier, message)
	},
}

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor SSE events for session completion",
	Long:  "Monitor the OpenCode server for session events and send notifications on completion.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMonitor(serverURL)
	},
}

var (
	// Status command flags
	statusJSON         bool
	statusAll          bool   // Include phantom agents (default: hide)
	statusProject      string // Filter by project
	statusSessionStart bool   // Output only surfacing info for SessionStart hooks
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show swarm status and active agents",
	Long: `Show swarm status including active/queued/completed agent counts,
per-account usage percentages, and individual agent details.

By default, phantom agents (beads issue open but no running agent) are hidden.
Use --all to include them.

The --session-start flag outputs only SessionStart surfacing info (architect
recommendations, usage warnings) for use in SessionStart hooks. This creates
pressure to review high-value design work that would otherwise accumulate silently.

Examples:
  orch-go status                  # Show active agents only
  orch-go status --all            # Include phantom agents
  orch-go status --project snap   # Filter by project
  orch-go status --json           # Output as JSON for scripting
  orch-go status --session-start  # Output surfacing info for hooks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if statusSessionStart {
			return runStatusSessionStart()
		}
		return runStatus(serverURL)
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON for scripting")
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "Include phantom agents")
	statusCmd.Flags().StringVar(&statusProject, "project", "", "Filter by project")
	statusCmd.Flags().BoolVar(&statusSessionStart, "session-start", false, "Output only surfacing info for SessionStart hooks")
}

var (
	// Complete command flags
	completeForce   bool
	completeReason  string
	completeApprove bool
	completeWorkdir string
)

var completeCmd = &cobra.Command{
	Use:   "complete [beads-id]",
	Short: "Complete an agent and close the beads issue",
	Long: `Complete an agent's work by verifying Phase: Complete and closing the beads issue.

Checks that the agent has reported "Phase: Complete" via beads comments before
closing the issue. Use --force to skip phase verification.

For agents that modified web/ files (UI tasks), --approve is required to explicitly
confirm human review of the visual changes. This prevents agents from self-certifying
UI correctness.

For cross-project completion (agents spawned with --workdir in another project),
the command auto-detects the project from the workspace's SPAWN_CONTEXT.md.
Use --workdir as explicit override when auto-detection fails.

Examples:
  orch-go complete proj-123
  orch-go complete proj-123 --reason "All tests passing"
  orch-go complete proj-123 --approve       # Approve UI changes after visual review
  orch-go complete proj-123 --force         # Skip all verification
  orch-go complete kb-cli-123 --workdir ~/projects/kb-cli  # Cross-project completion`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runComplete(beadsID, completeWorkdir)
	},
}

func init() {
	completeCmd.Flags().BoolVarP(&completeForce, "force", "f", false, "Skip phase verification")
	completeCmd.Flags().StringVarP(&completeReason, "reason", "r", "", "Reason for closing (default: uses phase summary)")
	completeCmd.Flags().BoolVar(&completeApprove, "approve", false, "Approve visual changes for UI tasks (adds approval comment)")
	completeCmd.Flags().StringVar(&completeWorkdir, "workdir", "", "Target project directory (for cross-project completion)")
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
		return runWork(serverURL, beadsID, workInline)
	},
}

func init() {
	workCmd.Flags().BoolVar(&workInline, "inline", false, "Run inline (blocking) with TUI")
}

var (
	// Tail command flags
	tailLines int

	// Send command flags
	sendAsync bool
)

var tailCmd = &cobra.Command{
	Use:   "tail [beads-id]",
	Short: "Capture recent output from an agent",
	Long: `Capture recent output from an agent for debugging.

Fetches messages from the OpenCode API for the agent's session.

Examples:
  orch-go tail proj-123              # Capture last 50 lines (default)
  orch-go tail proj-123 --lines 100  # Capture last 100 lines
  orch-go tail proj-123 -n 20        # Capture last 20 lines`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runTail(beadsID, tailLines)
	},
}

func init() {
	tailCmd.Flags().IntVarP(&tailLines, "lines", "n", 50, "Number of lines to capture")
}

func runTail(beadsID string, lines int) error {
	client := opencode.NewClient(serverURL)
	projectDir, _ := os.Getwd()

	// Strategy: Workspace file first (fast path), then derived lookups
	//
	// 1. Try to find session ID via workspace file (fast path)
	// 2. If workspace file has session ID, fetch messages from OpenCode API
	// 3. If no workspace file or API fails, find tmux window by beadsID and capture pane
	// 4. If tmux window found, also try to find matching OpenCode session by title

	// Try workspace file lookup for session ID (fast path)
	// Use findWorkspaceByBeadsID which correctly scans SPAWN_CONTEXT.md for beads ID
	var sessionID string
	var agentID string = beadsID

	workspacePath, workspaceName := findWorkspaceByBeadsID(projectDir, beadsID)
	if workspacePath != "" {
		sessionID = spawn.ReadSessionID(workspacePath)
		if sessionID != "" {
			agentID = workspaceName
		}
	}

	// If we have a session ID (from workspace file), try OpenCode API first
	if sessionID != "" {
		messages, err := client.GetMessages(sessionID)
		if err == nil && len(messages) > 0 {
			textLines := opencode.ExtractRecentText(messages, lines)
			fmt.Printf("=== Output from %s (via API, last %d lines) ===\n", agentID, lines)
			for _, line := range textLines {
				fmt.Println(line)
			}
			fmt.Printf("=== End of output ===\n")
			return nil
		}
		// If API fails, fall through to derived lookups
	}

	// Derived lookup: Find tmux window by beadsID
	sessions, err := tmux.ListWorkersSessions()
	if err != nil {
		// tmux not available, and no API session found
		return fmt.Errorf("no agent found for beads ID: %s (no tmux sessions, no API session)", beadsID)
	}

	for _, session := range sessions {
		window, err := tmux.FindWindowByBeadsID(session, beadsID)
		if err != nil || window == nil {
			continue
		}

		// Found tmux window - try to find matching OpenCode session first
		// This gives us richer output than just pane capture
		allSessions, err := client.ListSessions(projectDir)
		if err == nil {
			for _, s := range allSessions {
				// Match session by title containing beadsID or workspace name
				if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
					messages, err := client.GetMessages(s.ID)
					if err == nil && len(messages) > 0 {
						textLines := opencode.ExtractRecentText(messages, lines)
						fmt.Printf("=== Output from %s (via API, last %d lines) ===\n", agentID, lines)
						for _, line := range textLines {
							fmt.Println(line)
						}
						fmt.Printf("=== End of output ===\n")
						return nil
					}
				}
			}
		}

		// Fallback: capture tmux pane directly
		output, err := tmux.CaptureLines(window.Target, lines)
		if err == nil {
			printTmuxOutput(agentID, window.Target, lines, output)
			return nil
		}
	}

	return fmt.Errorf("no agent found for beads ID: %s (checked tmux and API)", beadsID)
}

func printTmuxOutput(agentID, target string, lines int, output []string) {
	fmt.Printf("=== Output from %s (via tmux %s, last %d lines) ===\n", agentID, target, lines)
	for _, line := range output {
		fmt.Println(line)
	}
	fmt.Printf("=== End of output ===\n")
}

var questionCmd = &cobra.Command{
	Use:   "question [beads-id]",
	Short: "Extract pending question from an agent's session",
	Long: `Extract pending question from an agent's session.

Finds the OpenCode session associated with the beads issue ID and extracts
any pending question the agent is asking. Useful for monitoring agents
that are blocked waiting for user input.

Examples:
  orch-go question proj-123  # Extract question from agent's session`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runQuestion(beadsID)
	},
}

func runQuestion(beadsID string) error {
	client := opencode.NewClient(serverURL)
	projectDir, _ := os.Getwd()

	// Strategy: Workspace file first (fast path), then derived lookups
	//
	// 1. Try to find session ID via workspace file (fast path)
	// 2. If workspace file has session ID, fetch messages from OpenCode API
	// 3. If no workspace file or API fails, find tmux window by beadsID and check pane
	// 4. If tmux window found, also try to find matching OpenCode session by title

	// Try workspace file lookup for session ID (fast path)
	// Use findWorkspaceByBeadsID which correctly scans SPAWN_CONTEXT.md for beads ID
	var sessionID string
	workspacePath, _ := findWorkspaceByBeadsID(projectDir, beadsID)
	if workspacePath != "" {
		sessionID = spawn.ReadSessionID(workspacePath)
	}

	// If we have a session ID (from workspace file), try OpenCode API first
	if sessionID != "" {
		messages, err := client.GetMessages(sessionID)
		if err == nil && len(messages) > 0 {
			textLines := opencode.ExtractRecentText(messages, 100)
			content := strings.Join(textLines, "\n")
			q := question.Extract(content)
			if q != "" {
				fmt.Printf("Pending question (via API):\n%s\n", q)
				return nil
			}
			// No question in API - might still be pending in terminal, continue to tmux
		}
	}

	// Derived lookup: Find tmux window by beadsID
	sessions, err := tmux.ListWorkersSessions()
	if err != nil {
		fmt.Println("No pending question found (no tmux sessions)")
		return nil
	}

	for _, session := range sessions {
		window, err := tmux.FindWindowByBeadsID(session, beadsID)
		if err != nil || window == nil {
			continue
		}

		// Found tmux window - try to find matching OpenCode session first
		// This gives us richer message history than just pane capture
		if sessionID == "" {
			allSessions, err := client.ListSessions(projectDir)
			if err == nil {
				for _, s := range allSessions {
					// Match session by title containing beadsID or workspace name
					if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
						messages, err := client.GetMessages(s.ID)
						if err == nil && len(messages) > 0 {
							textLines := opencode.ExtractRecentText(messages, 100)
							content := strings.Join(textLines, "\n")
							q := question.Extract(content)
							if q != "" {
								fmt.Printf("Pending question (via API):\n%s\n", q)
								return nil
							}
						}
					}
				}
			}
		}

		// Fallback: check tmux pane directly
		lines, err := tmux.CaptureLines(window.Target, 100)
		if err == nil {
			content := strings.Join(lines, "\n")
			q := question.Extract(content)
			if q != "" {
				fmt.Printf("Pending question (via tmux %s):\n%s\n", window.Target, q)
				return nil
			}
		}
	}

	fmt.Println("No pending question found (checked API and tmux)")
	return nil
}

var (
	// Abandon command flags
	abandonReason  string
	abandonWorkdir string
)

var abandonCmd = &cobra.Command{
	Use:   "abandon [beads-id]",
	Short: "Abandon a stuck or frozen agent",
	Long: `Abandon an agent and kill its tmux window.

Use this command for stuck or frozen agents that are not responding.
The agent's beads issue is NOT closed - you can restart work with 'orch work'.

When --reason is provided, a FAILURE_REPORT.md is generated in the agent's workspace
documenting what went wrong and recommendations for retry.

For cross-project abandonment, use --workdir to specify the target project directory
where the beads issue lives.

Examples:
  orch-go abandon proj-123                                      # Abandon agent in current project
  orch-go abandon proj-123 --reason "Out of context"            # Abandon with failure report
  orch-go abandon proj-123 --reason "Stuck in loop"             # Document the failure
  orch-go abandon kb-cli-123 --workdir ~/projects/kb-cli        # Abandon agent in another project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runAbandon(beadsID, abandonReason, abandonWorkdir)
	},
}

func init() {
	abandonCmd.Flags().StringVar(&abandonReason, "reason", "", "Reason for abandonment (generates FAILURE_REPORT.md)")
	abandonCmd.Flags().StringVar(&abandonWorkdir, "workdir", "", "Target project directory (for cross-project abandonment)")
}

func runAbandon(beadsID, reason, workdir string) error {
	// Strategy: Check liveness directly via tmux and OpenCode, not registry
	// An agent is "alive" if it has a tmux window OR an active OpenCode session

	// Determine project directory - use --workdir if provided, otherwise current directory
	var projectDir string
	var err error
	if workdir != "" {
		projectDir, err = filepath.Abs(workdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		// Verify directory exists
		if stat, err := os.Stat(projectDir); err != nil {
			return fmt.Errorf("workdir does not exist: %s", projectDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", projectDir)
		}
		// Set DefaultDir for beads client to find the correct socket
		beads.DefaultDir = projectDir
	} else {
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// First, verify the beads issue exists
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		// Provide helpful error message for cross-project issues
		projectName := filepath.Base(projectDir)
		issuePrefix := strings.Split(beadsID, "-")[0]
		if len(strings.Split(beadsID, "-")) > 1 {
			issuePrefix = strings.Join(strings.Split(beadsID, "-")[:len(strings.Split(beadsID, "-"))-1], "-")
		}
		if issuePrefix != projectName {
			return fmt.Errorf("failed to get beads issue %s: %w\n\nHint: The issue ID suggests it belongs to project '%s', but you're in '%s'.\nTry: orch abandon %s --workdir ~/path/to/%s", beadsID, err, issuePrefix, projectName, beadsID, issuePrefix)
		}
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	if issue.Status == "closed" {
		return fmt.Errorf("issue %s is already closed - nothing to abandon", beadsID)
	}

	client := opencode.NewClient(serverURL)

	// Check for tmux window
	var windowInfo *tmux.WindowInfo
	sessions, _ := tmux.ListWorkersSessions()
	for _, session := range sessions {
		window, err := tmux.FindWindowByBeadsID(session, beadsID)
		if err == nil && window != nil {
			windowInfo = window
			break
		}
	}

	// Check for OpenCode session
	var sessionID string
	allSessions, _ := client.ListSessions(projectDir)
	for _, s := range allSessions {
		if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
			sessionID = s.ID
			break
		}
	}

	// Find workspace for logging
	workspacePath, agentName := findWorkspaceByBeadsID(projectDir, beadsID)
	if agentName == "" {
		agentName = beadsID // Use beads ID as fallback
	}

	// Report what we found
	if windowInfo != nil {
		fmt.Printf("Found tmux window: %s\n", windowInfo.Target)
	}
	if sessionID != "" {
		fmt.Printf("Found OpenCode session: %s\n", sessionID[:12])
	}

	// If neither found, warn but still allow abandonment
	if windowInfo == nil && sessionID == "" {
		fmt.Printf("Note: No active tmux window or OpenCode session found for %s\n", beadsID)
		fmt.Printf("The agent may have already exited.\n")
	}

	// Optionally kill the tmux window if it exists
	if windowInfo != nil {
		fmt.Printf("Killing tmux window: %s\n", windowInfo.Target)
		if err := tmux.KillWindow(windowInfo.Target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to kill tmux window: %v\n", err)
		}
	}

	// Generate FAILURE_REPORT.md if reason is provided
	if reason != "" && workspacePath != "" {
		// Ensure the failure report template exists in the project
		if err := spawn.EnsureFailureReportTemplate(projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to ensure failure report template: %v\n", err)
		}

		// Generate and write the failure report
		reportPath, err := spawn.WriteFailureReport(workspacePath, agentName, beadsID, reason, issue.Title)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write failure report: %v\n", err)
		} else {
			fmt.Printf("Generated failure report: %s\n", reportPath)
		}
	}

	// Log the abandonment
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"beads_id": beadsID,
		"agent_id": agentName,
	}
	if windowInfo != nil {
		eventData["window_id"] = windowInfo.ID
		eventData["window_target"] = windowInfo.Target
	}
	if sessionID != "" {
		eventData["session_id"] = sessionID
	}
	if workspacePath != "" {
		eventData["workspace_path"] = workspacePath
	}
	if reason != "" {
		eventData["reason"] = reason
	}
	event := events.Event{
		Type:      "agent.abandoned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Reset beads status to open so respawn works without manual bd update
	if err := verify.UpdateIssueStatus(beadsID, "open"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to reset beads status: %v\n", err)
	} else {
		fmt.Printf("Reset beads status: in_progress → open\n")
	}

	fmt.Printf("Abandoned agent: %s\n", agentName)
	fmt.Printf("  Beads ID: %s\n", beadsID)
	if reason != "" {
		fmt.Printf("  Reason: %s\n", reason)
	}
	fmt.Printf("  Use 'orch work %s' to restart work on this issue\n", beadsID)

	return nil
}

// InferSkillFromIssueType maps issue types to appropriate skills.
// Returns an error for types that cannot be spawned (e.g., epic) or unknown types.
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

func runWork(serverURL, beadsID string, inline bool) error {
	// Get issue details
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Infer skill from issue type
	skillName, err := InferSkillFromIssueType(issue.IssueType)
	if err != nil {
		return fmt.Errorf("cannot work on issue %s: %w", beadsID, err)
	}

	// Use issue title and description as the task for full context
	task := issue.Title
	if issue.Description != "" {
		task = issue.Title + "\n\n" + issue.Description
	}

	// Set the spawnIssue flag so runSpawnWithSkill uses the existing issue
	spawnIssue = beadsID

	fmt.Printf("Starting work on: %s\n", beadsID)
	fmt.Printf("  Title:  %s\n", issue.Title)
	fmt.Printf("  Type:   %s\n", issue.IssueType)
	fmt.Printf("  Skill:  %s\n", skillName)

	return runSpawnWithSkill(serverURL, skillName, task, inline, true, false, false) // headless=true for work command (daemon uses this)
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

// determineSpawnTier determines the spawn tier based on flags and skill defaults.
// Priority: --light flag > --full flag > skill default > TierFull (conservative)
func determineSpawnTier(skillName string, lightFlag, fullFlag bool) string {
	// Explicit flags take precedence
	if lightFlag {
		return spawn.TierLight
	}
	if fullFlag {
		return spawn.TierFull
	}
	// Fall back to skill default
	return spawn.DefaultTierForSkill(skillName)
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

func runSpawnWithSkill(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool) error {
	// Validate MCP server name early (fail fast before any work)
	if err := spawn.ValidateMCPName(spawnMCP); err != nil {
		return err
	}

	// Check concurrency limit before spawning
	if err := checkConcurrencyLimit(); err != nil {
		return err
	}

	// Auto-switch account if current account is over usage thresholds
	if err := checkAndAutoSwitchAccount(); err != nil {
		// Log warning but don't block spawn - continue with current account
		fmt.Fprintf(os.Stderr, "Warning: auto-switch failed: %v\n", err)
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

	// Generate workspace name
	workspaceName := spawn.GenerateWorkspaceName(skillName, task)

	// Load skill content with dependencies (e.g., worker-base patterns)
	loader := skills.DefaultLoader()
	skillContent, err := loader.LoadSkillWithDependencies(skillName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load skill '%s': %v\n", skillName, err)
		skillContent = "" // Continue without skill content
	}

	// Determine beads ID - either from flag, create new issue, or skip if --no-track
	beadsID, err := determineBeadsID(projectName, skillName, task, spawnIssue, spawnNoTrack, createBeadsIssue)
	if err != nil {
		return fmt.Errorf("failed to determine beads ID: %w", err)
	}
	if spawnNoTrack {
		fmt.Println("Skipping beads tracking (--no-track)")
	}

	// Check for retry patterns on existing issues - surface to prevent blind respawning
	if !spawnNoTrack && spawnIssue != "" {
		if stats, err := verify.GetFixAttemptStats(beadsID); err == nil && stats.IsRetryPattern() {
			warning := verify.FormatRetryWarning(stats)
			if warning != "" {
				fmt.Fprintf(os.Stderr, "\n%s\n", warning)
			}
		}
	}

	// Check if issue is already being worked on (prevent duplicate spawns)
	if !spawnNoTrack && spawnIssue != "" {
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
	if !spawnNoTrack && spawnIssue != "" {
		if err := verify.UpdateIssueStatus(beadsID, "in_progress"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
			// Continue anyway
		}
		// Remove triage:ready label since issue is now in_progress
		if err := verify.RemoveTriageReadyLabel(beadsID); err != nil {
			// Don't warn for non-critical label removal - it may not have the label
			// and that's fine
		}
	}

	// Resolve model - convert aliases to full format
	resolvedModel := model.Resolve(spawnModel)

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

	// Build spawn config
	cfg := &spawn.Config{
		Task:              task,
		SkillName:         skillName,
		Project:           projectName,
		ProjectDir:        projectDir,
		WorkspaceName:     workspaceName,
		SkillContent:      skillContent,
		BeadsID:           beadsID,
		Phases:            spawnPhases,
		Mode:              spawnMode,
		Validation:        spawnValidation,
		Model:             resolvedModel.Format(),
		MCP:               spawnMCP,
		Tier:              tier,
		NoTrack:           spawnNoTrack,
		SkipArtifactCheck: spawnSkipArtifactCheck,
		KBContext:         kbContext,
		IncludeServers:    spawn.DefaultIncludeServersForSkill(skillName),
		GapAnalysis:       gapAnalysis,
	}

	// Pre-spawn token estimation and validation
	if err := spawn.ValidateContextSize(cfg); err != nil {
		return fmt.Errorf("pre-spawn validation failed: %w", err)
	}

	// Warn about large contexts (but don't block)
	if shouldWarn, warning := spawn.ShouldWarnAboutSize(cfg); shouldWarn {
		fmt.Fprintf(os.Stderr, "%s", warning)
	}

	// Write SPAWN_CONTEXT.md
	if err := spawn.WriteContext(cfg); err != nil {
		return fmt.Errorf("failed to write spawn context: %w", err)
	}

	// Generate minimal prompt
	minimalPrompt := spawn.MinimalPrompt(cfg)

	// Spawn mode: inline (blocking TUI), tmux (opt-in), or headless (default)
	if inline {
		// Inline mode (blocking) - run in current terminal with TUI
		return runSpawnInline(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	if tmux || attach {
		// Tmux mode (opt-in) - visible, interruptible, prevents runaway spawns
		// attach implies tmux
		return runSpawnTmux(serverURL, cfg, minimalPrompt, beadsID, skillName, task, attach)
	}

	// Default: Headless mode - spawn via HTTP API (automation-friendly, no TUI overhead)
	return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task, spawnVerbose)
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

	// Add MCP config if specified
	if cfg.MCP != "" {
		mcpConfigContent, err := spawn.GenerateMCPConfig(cfg.MCP)
		if err != nil {
			return fmt.Errorf("failed to generate MCP config: %w", err)
		}
		cmd.Env = append(cmd.Env, "OPENCODE_CONFIG_CONTENT="+mcpConfigContent)
	}

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

// runSpawnHeadless spawns the agent using the HTTP API without a TUI.
// This is useful for automation and daemon-driven spawns.
// Uses the HTTP API directly which properly supports directory configuration.
// Includes retry logic for transient network failures.
// The verbose flag enables additional logging for debugging.
func runSpawnHeadless(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string, verbose bool) error {
	// Use client with directory so all API calls include x-opencode-directory header
	client := opencode.NewClientWithDirectory(serverURL, cfg.ProjectDir)

	// Format title with beads ID so orch status can match sessions
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)

	// Use retry logic for transient failures (network issues, server temporarily unavailable)
	retryCfg := spawn.DefaultRetryConfig()
	result, retryResult := spawn.Retry(retryCfg, func() (*headlessSpawnResult, error) {
		return startHeadlessSessionAPI(client, sessionTitle, minimalPrompt, cfg, verbose)
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
	SessionID    string
	cmd          *exec.Cmd
	stdout       io.ReadCloser
	stderrBuffer *bytes.Buffer // Captured stderr for error visibility
	verbose      bool          // Whether to output stderr in real-time
}

// StartBackgroundCleanup starts a goroutine to drain stdout, wait for the process,
// and log any stderr output or exit errors to the events log for later analysis.
func (r *headlessSpawnResult) StartBackgroundCleanup() {
	if r.stdout == nil || r.cmd == nil {
		return
	}
	go func() {
		// Drain remaining stdout
		io.Copy(io.Discard, r.stdout)
		// Wait for process to complete (cleanup)
		err := r.cmd.Wait()

		// Check for errors to log
		var hasError bool
		var errorDetails []string

		if err != nil {
			hasError = true
			errorDetails = append(errorDetails, fmt.Sprintf("exit_error: %v", err))
		}

		if r.stderrBuffer != nil && r.stderrBuffer.Len() > 0 {
			stderrContent := strings.TrimSpace(r.stderrBuffer.String())
			if stderrContent != "" {
				hasError = true
				errorDetails = append(errorDetails, fmt.Sprintf("stderr: %s", stderrContent))

				// In verbose mode, stderr was already printed in real-time.
				// In non-verbose mode, print a warning with the stderr content if there's an error.
				if !r.verbose && err != nil {
					fmt.Fprintf(os.Stderr, "\n⚠️  Agent process exited with error.\nStderr output:\n%s\n", stderrContent)
				}
			}
		}

		// Log errors to events for later analysis
		if hasError {
			logger := events.NewLogger(events.DefaultLogPath())
			event := events.Event{
				Type:      "session.error",
				SessionID: r.SessionID,
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"session_id": r.SessionID,
					"errors":     errorDetails,
				},
			}
			if logErr := logger.Log(event); logErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log session error: %v\n", logErr)
			}
		}
	}()
}

// startHeadlessSession starts an opencode session and extracts the session ID.
// Returns the result with session ID and resources for cleanup.
// The verbose flag enables real-time stderr output for debugging.
func startHeadlessSession(client *opencode.Client, serverURL, sessionTitle, minimalPrompt string, cfg *spawn.Config, verbose bool) (*headlessSpawnResult, error) {
	cmd := client.BuildSpawnCommand(minimalPrompt, sessionTitle, cfg.Model)
	cmd.Dir = cfg.ProjectDir
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		spawnErr := spawn.WrapSpawnError(err, "Failed to get stdout pipe")
		return nil, spawnErr
	}

	// Capture stderr for error visibility
	// In verbose mode, also tee stderr to os.Stderr for real-time output
	var stderrBuffer bytes.Buffer
	if verbose {
		// Tee stderr to both the buffer and os.Stderr for real-time debugging
		cmd.Stderr = io.MultiWriter(&stderrBuffer, os.Stderr)
	} else {
		// Capture stderr to buffer for logging on errors
		cmd.Stderr = &stderrBuffer
	}

	if err := cmd.Start(); err != nil {
		spawnErr := spawn.WrapSpawnError(err, "Failed to start opencode process")
		return nil, spawnErr
	}

	// Process stdout to extract session ID, then let the process run in background
	// We need to read at least until we get the session ID
	sessionID, err := opencode.ExtractSessionIDFromReader(stdout)
	if err != nil {
		// Include any stderr content in the error message for visibility
		stderrContent := strings.TrimSpace(stderrBuffer.String())
		if stderrContent != "" {
			err = fmt.Errorf("%w\nStderr: %s", err, stderrContent)
		}
		// Try to kill the process if we couldn't get session ID
		cmd.Process.Kill()
		spawnErr := spawn.WrapSpawnError(err, "Failed to extract session ID")
		return nil, spawnErr
	}

	return &headlessSpawnResult{
		SessionID:    sessionID,
		cmd:          cmd,
		stdout:       stdout,
		stderrBuffer: &stderrBuffer,
		verbose:      verbose,
	}, nil
}

// startHeadlessSessionAPI creates a session via HTTP API and sends the initial prompt.
// This is the preferred method for headless spawns as it properly handles the directory.
// The opencode CLI's --attach mode has a bug where it always uses "/" as the directory.
func startHeadlessSessionAPI(client *opencode.Client, sessionTitle, minimalPrompt string, cfg *spawn.Config, verbose bool) (*headlessSpawnResult, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "Creating session via API: title=%s, directory=%s\n", sessionTitle, cfg.ProjectDir)
	}

	// Generate MCP config if specified
	var opts *opencode.CreateSessionOptions
	if cfg.MCP != "" {
		mcpConfigContent, err := spawn.GenerateMCPConfig(cfg.MCP)
		if err != nil {
			return nil, fmt.Errorf("failed to generate MCP config: %w", err)
		}
		opts = &opencode.CreateSessionOptions{
			MCPConfigContent: mcpConfigContent,
		}
	}

	// Create session via HTTP API with correct directory
	session, err := client.CreateSessionWithOptions(sessionTitle, cfg.ProjectDir, cfg.Model, opts)
	if err != nil {
		return nil, spawn.WrapSpawnError(err, "Failed to create session via API")
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Created session: %s\n", session.ID)
	}

	// Send the initial prompt
	if err := client.SendPrompt(session.ID, minimalPrompt, cfg.Model); err != nil {
		return nil, spawn.WrapSpawnError(err, "Failed to send initial prompt")
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Sent initial prompt to session %s\n", session.ID)
	}

	// Return result - no process to manage since we're using the API
	return &headlessSpawnResult{
		SessionID: session.ID,
		// cmd, stdout, stderrBuffer are nil since we're using API
		verbose: verbose,
	}, nil
}

// runSpawnTmux spawns the agent in a tmux window (interactive, returns immediately).
// Creates a tmux window in workers-{project} session, runs opencode there, and returns.
func runSpawnTmux(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string, attach bool) error {
	// Ensure workers tmux session exists
	sessionName, err := tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
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

	// Generate MCP config if specified
	var mcpConfigContent string
	if cfg.MCP != "" {
		var err error
		mcpConfigContent, err = spawn.GenerateMCPConfig(cfg.MCP)
		if err != nil {
			return fmt.Errorf("failed to generate MCP config: %w", err)
		}
	}

	// Build opencode command using tmux package
	opencodeCmd := tmux.BuildOpencodeAttachCommand(&tmux.OpencodeAttachConfig{
		ServerURL:        serverURL,
		ProjectDir:       cfg.ProjectDir,
		Model:            cfg.Model,
		MCPConfigContent: mcpConfigContent,
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
	client := opencode.NewClient(serverURL)
	sessionID, _ := client.FindRecentSessionWithRetry(cfg.ProjectDir, "", 3, 500*time.Millisecond)
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

// determineBeadsID determines the beads ID to use for an agent.
// It returns an error if beads issue creation fails and --no-track is not set.
// The createBeadsFn parameter allows for dependency injection in tests.
func determineBeadsID(projectName, skillName, task, spawnIssue string, spawnNoTrack bool, createBeadsFn func(string, string, string) (string, error)) (string, error) {
	// If explicit issue ID provided via --issue flag, use it
	if spawnIssue != "" {
		return spawnIssue, nil
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

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// extractBeadsIDFromTitle extracts beads ID from an OpenCode session title.
// Looks for patterns like "[beads-id]" at the end of the title.
func extractBeadsIDFromTitle(title string) string {
	// Look for "[beads-id]" pattern
	if start := strings.LastIndex(title, "["); start != -1 {
		if end := strings.LastIndex(title, "]"); end != -1 && end > start {
			return strings.TrimSpace(title[start+1 : end])
		}
	}
	return ""
}

// extractSkillFromTitle extracts skill from an OpenCode session title.
// Infers skill from common workspace name prefixes (og-feat-, og-inv-, og-debug-, etc.)
func extractSkillFromTitle(title string) string {
	titleLower := strings.ToLower(title)
	// Check for workspace name patterns
	if strings.Contains(titleLower, "-feat-") {
		return "feature-impl"
	}
	if strings.Contains(titleLower, "-inv-") {
		return "investigation"
	}
	if strings.Contains(titleLower, "-debug-") {
		return "systematic-debugging"
	}
	if strings.Contains(titleLower, "-arch-") {
		return "architect"
	}
	if strings.Contains(titleLower, "-audit-") {
		return "codebase-audit"
	}
	if strings.Contains(titleLower, "-research-") {
		return "research"
	}
	return ""
}

// extractBeadsIDFromWindowName extracts beads ID from a tmux window name.
// Window names follow format: "emoji workspace-name [beads-id]"
func extractBeadsIDFromWindowName(name string) string {
	// Look for "[beads-id]" pattern
	if start := strings.LastIndex(name, "["); start != -1 {
		if end := strings.LastIndex(name, "]"); end != -1 && end > start {
			return strings.TrimSpace(name[start+1 : end])
		}
	}
	return ""
}

// extractSkillFromWindowName extracts skill from a tmux window name.
// First tries to match skill emoji, then falls back to workspace name patterns.
func extractSkillFromWindowName(name string) string {
	// Try emoji matching first (most reliable)
	for skill, emoji := range tmux.SKILL_EMOJIS {
		if strings.Contains(name, emoji) {
			return skill
		}
	}
	// Fall back to workspace name patterns
	return extractSkillFromTitle(name)
}

// resolveSessionID resolves an identifier to an OpenCode session ID.
// The identifier can be:
// 1. A full OpenCode session ID (ses_xxx) - verified against API, returned if valid
// 2. A beads ID (project-xxxx) - looked up via workspace SPAWN_CONTEXT.md or API
// 3. A workspace name - looked up via workspace file
//
// Returns the resolved session ID or an error if resolution fails.
func resolveSessionID(serverURL, identifier string) (string, error) {
	// If it looks like a full session ID, verify it exists
	if strings.HasPrefix(identifier, "ses_") {
		// Validate the session ID has content after the prefix
		suffix := strings.TrimPrefix(identifier, "ses_")
		if len(suffix) < 8 { // Session IDs have substantial content after ses_
			return "", fmt.Errorf("invalid session ID format: %s (too short)", identifier)
		}
		// Verify the session exists in OpenCode
		client := opencode.NewClient(serverURL)
		_, err := client.GetSession(identifier)
		if err != nil {
			return "", fmt.Errorf("session not found in OpenCode: %s", identifier)
		}
		return identifier, nil
	}

	client := opencode.NewClient(serverURL)
	projectDir, _ := os.Getwd()

	// Strategy 1: Use findWorkspaceByBeadsID which scans SPAWN_CONTEXT.md
	// This is the authoritative way to find workspace by beads ID
	workspacePath, _ := findWorkspaceByBeadsID(projectDir, identifier)
	if workspacePath != "" {
		sessionID := spawn.ReadSessionID(workspacePath)
		if sessionID != "" {
			return sessionID, nil
		}
	}

	// Strategy 2: Direct workspace name match (for workspace name identifiers)
	workspaceBase := filepath.Join(projectDir, ".orch", "workspace")
	if entries, err := os.ReadDir(workspaceBase); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.Contains(entry.Name(), identifier) {
				wp := filepath.Join(workspaceBase, entry.Name())
				sessionID := spawn.ReadSessionID(wp)
				if sessionID != "" {
					return sessionID, nil
				}
			}
		}
	}

	// Strategy 3: API lookup - search sessions by title containing identifier
	allSessions, err := client.ListSessions(projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %w", err)
	}

	for _, s := range allSessions {
		// Match session by title containing identifier (beads ID or workspace name)
		if strings.Contains(s.Title, identifier) || extractBeadsIDFromTitle(s.Title) == identifier {
			return s.ID, nil
		}
	}

	// Strategy 4: tmux window lookup as last resort - find window, then try to get session
	sessions, err := tmux.ListWorkersSessions()
	if err == nil {
		for _, session := range sessions {
			window, err := tmux.FindWindowByBeadsID(session, identifier)
			if err != nil || window == nil {
				continue
			}

			// Found tmux window - try to find matching OpenCode session by window name
			// Window names have workspace names in them
			for _, s := range allSessions {
				if strings.Contains(window.Name, s.Title) || strings.Contains(s.Title, identifier) {
					return s.ID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no session found for identifier: %s (checked workspace files, API sessions, and tmux windows)", identifier)
}

func runSend(serverURL, identifier, message string) error {
	// First, try to resolve identifier to OpenCode session ID
	sessionID, resolveErr := resolveSessionID(serverURL, identifier)

	client := opencode.NewClient(serverURL)

	// If resolution succeeded, use OpenCode API
	if resolveErr == nil && sessionID != "" {
		return sendViaOpenCodeAPI(client, sessionID, identifier, message)
	}

	// OpenCode session not found - try tmux send-keys fallback
	// This handles tmux agents where session ID wasn't captured or title doesn't match
	windowInfo, err := findTmuxWindowByIdentifier(identifier)
	if err != nil || windowInfo == nil {
		// Neither OpenCode session nor tmux window found
		if resolveErr != nil {
			return fmt.Errorf("failed to resolve session and no tmux window found: %w", resolveErr)
		}
		return fmt.Errorf("no session or tmux window found for identifier: %s", identifier)
	}

	// Found tmux window - send via send-keys
	return sendViaTmux(windowInfo, identifier, message)
}

// sendViaOpenCodeAPI sends a message using the OpenCode HTTP API.
func sendViaOpenCodeAPI(client *opencode.Client, sessionID, identifier, message string) error {
	// Log the send event first (before streaming starts)
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.send",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message":    message,
			"async":      sendAsync,
			"identifier": identifier,
			"method":     "api",
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	if sendAsync {
		// Send message asynchronously (non-blocking, no model for Q&A)
		if err := client.SendMessageAsync(sessionID, message, ""); err != nil {
			return fmt.Errorf("failed to send message asynchronously: %w", err)
		}
		fmt.Printf("✓ Message sent to session %s (via API)\n", sessionID)
		return nil
	}

	// Send message and stream the response to stdout (blocking)
	if err := client.SendMessageWithStreaming(sessionID, message, os.Stdout); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Add newline at end for clean output
	fmt.Println()

	return nil
}

// sendViaTmux sends a message to a tmux window using send-keys.
// This is used as a fallback when the OpenCode session ID cannot be resolved.
func sendViaTmux(windowInfo *tmux.WindowInfo, identifier, message string) error {
	// Log the send event
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.send",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message":       message,
			"identifier":    identifier,
			"method":        "tmux",
			"window_target": windowInfo.Target,
			"window_id":     windowInfo.ID,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Send the message using tmux send-keys in literal mode
	if err := tmux.SendKeysLiteral(windowInfo.Target, message); err != nil {
		return fmt.Errorf("failed to send message via tmux: %w", err)
	}

	// Send Enter to submit the message
	if err := tmux.SendEnter(windowInfo.Target); err != nil {
		return fmt.Errorf("failed to send enter via tmux: %w", err)
	}

	fmt.Printf("✓ Message sent to %s (via tmux %s)\n", identifier, windowInfo.Target)
	return nil
}

// findTmuxWindowByIdentifier searches for a tmux window matching the identifier.
// The identifier can be a beads ID, workspace name, or partial match.
func findTmuxWindowByIdentifier(identifier string) (*tmux.WindowInfo, error) {
	sessions, err := tmux.ListWorkersSessions()
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		// First try exact beads ID match (format: "[beads-id]" in window name)
		window, err := tmux.FindWindowByBeadsID(session, identifier)
		if err == nil && window != nil {
			return window, nil
		}

		// Also try partial match on window name (for workspace name matches)
		windows, err := tmux.ListWindows(session)
		if err != nil {
			continue
		}
		for i := range windows {
			if strings.Contains(windows[i].Name, identifier) {
				return &windows[i], nil
			}
		}
	}

	return nil, nil // Not found (no error, just not found)
}

// SwarmStatus represents aggregate swarm information.
type SwarmStatus struct {
	Active     int `json:"active"`
	Processing int `json:"processing,omitempty"` // Agents actively generating response
	Idle       int `json:"idle,omitempty"`       // Agents with session but not processing
	Phantom    int `json:"phantom,omitempty"`    // Agents with open beads issue but not running
	Queued     int `json:"queued"`
	Completed  int `json:"completed_today"`
}

// AccountUsage represents usage info for a single account.
type AccountUsage struct {
	Name        string  `json:"name"`
	Email       string  `json:"email,omitempty"`
	UsedPercent float64 `json:"used_percent"`
	ResetTime   string  `json:"reset_time,omitempty"`
	IsActive    bool    `json:"is_active"`
}

// AgentInfo represents information about an active agent.
type AgentInfo struct {
	SessionID    string               `json:"session_id"`
	BeadsID      string               `json:"beads_id,omitempty"`
	Skill        string               `json:"skill,omitempty"`
	Account      string               `json:"account,omitempty"`
	Runtime      string               `json:"runtime"`
	Title        string               `json:"title,omitempty"`
	Window       string               `json:"window,omitempty"`
	Phase        string               `json:"phase,omitempty"`         // Current phase from beads comments
	Task         string               `json:"task,omitempty"`          // Task description (truncated)
	Project      string               `json:"project,omitempty"`       // Project name derived from beads ID or workspace
	IsPhantom    bool                 `json:"is_phantom,omitempty"`    // True if beads issue open but agent not running
	IsProcessing bool                 `json:"is_processing,omitempty"` // True if session is actively generating a response
	IsCompleted  bool                 `json:"is_completed,omitempty"`  // True if beads issue is closed
	Tokens       *opencode.TokenStats `json:"tokens,omitempty"`        // Token usage for the session
}

// StatusOutput represents the full status output for JSON serialization.
type StatusOutput struct {
	Swarm    SwarmStatus    `json:"swarm"`
	Accounts []AccountUsage `json:"accounts"`
	Agents   []AgentInfo    `json:"agents"`
}

func runStatus(serverURL string) error {
	client := opencode.NewClient(serverURL)
	now := time.Now()

	agents := make([]AgentInfo, 0)
	seenBeadsIDs := make(map[string]bool)

	// Get current project directory for session queries
	projectDir, _ := os.Getwd()

	// === OPTIMIZED: Batch fetch all data upfront ===
	// 1. Fetch OpenCode sessions from current project directory FIRST.
	// OpenCode stores sessions per-directory, so ListSessions("") only returns global sessions.
	// Sessions created with x-opencode-directory header need to be queried with that directory.
	var sessions []opencode.Session
	seenSessionIDs := make(map[string]bool)

	// Query current project directory first (most likely to have active agents)
	if projectDir != "" {
		dirSessions, err := client.ListSessions(projectDir)
		if err == nil {
			for _, s := range dirSessions {
				if !seenSessionIDs[s.ID] {
					seenSessionIDs[s.ID] = true
					sessions = append(sessions, s)
				}
			}
		}
	}

	// Also query global sessions to catch any that weren't created with directory header
	globalSessions, err := client.ListSessions("")
	if err != nil {
		// Only fail if we have no sessions at all
		if len(sessions) == 0 {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
	} else {
		for _, s := range globalSessions {
			if !seenSessionIDs[s.ID] {
				seenSessionIDs[s.ID] = true
				sessions = append(sessions, s)
			}
		}
	}

	// Build a map of session ID -> session for quick lookup
	sessionMap := make(map[string]*opencode.Session)
	// Also build a map of beadsID -> session for matching
	beadsToSession := make(map[string]*opencode.Session)
	const maxIdleTime = 30 * time.Minute

	for i := range sessions {
		s := &sessions[i]
		sessionMap[s.ID] = s

		// Only consider recently active sessions for beads matching
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				beadsToSession[beadsID] = s
			}
		}
	}

	// 2. Collect beads IDs first, then batch fetch issues later
	// (openIssues removed - we now use allIssues to check both open and closed status)

	// 3. Collect all beads IDs we need comments for
	var beadsIDsToFetch []string

	// Track project directories for cross-project agents (beadsID -> projectDir)
	beadsProjectDirs := make(map[string]string)

	// Phase 1: Collect agents from tmux windows (primary source of truth for "active")
	type tmuxAgent struct {
		beadsID string
		skill   string
		project string
		window  string
		title   string
	}
	var tmuxAgents []tmuxAgent

	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, w := range windows {
			// Skip known non-agent windows
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" {
				continue
			}

			tmuxAgents = append(tmuxAgents, tmuxAgent{
				beadsID: beadsID,
				skill:   extractSkillFromWindowName(w.Name),
				project: extractProjectFromBeadsID(beadsID),
				window:  w.Target,
				title:   w.Name,
			})

			if beadsID != "" && !seenBeadsIDs[beadsID] {
				beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
				seenBeadsIDs[beadsID] = true
			}
		}
	}

	// Phase 2: Collect beads IDs from active OpenCode sessions
	type opcodeAgent struct {
		session *opencode.Session
		beadsID string
		skill   string
		project string
	}
	var opcodeAgents []opcodeAgent

	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) > maxIdleTime {
			continue
		}

		beadsID := extractBeadsIDFromTitle(s.Title)
		if beadsID == "" {
			continue
		}

		// Skip if already tracked via tmux
		if seenBeadsIDs[beadsID] {
			continue
		}

		sessionCopy := s // Copy to avoid closure issues
		opcodeAgents = append(opcodeAgents, opcodeAgent{
			session: &sessionCopy,
			beadsID: beadsID,
			skill:   extractSkillFromTitle(s.Title),
			project: extractProjectFromBeadsID(beadsID),
		})

		beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
		seenBeadsIDs[beadsID] = true
	}

	// Build workspace cache ONCE (O(n) scan) instead of O(n*m) findWorkspaceByBeadsID calls
	// This is the key performance optimization - we scan 702 dirs once instead of once per beadsID
	wsCache := buildWorkspaceCache(projectDir)

	// Look up workspaces to get project directories for cross-project agents (O(1) per lookup)
	for _, beadsID := range beadsIDsToFetch {
		workspacePath := wsCache.lookupWorkspace(beadsID)
		if workspacePath != "" {
			agentProjectDir := wsCache.lookupProjectDir(beadsID)
			if agentProjectDir != "" && agentProjectDir != projectDir {
				beadsProjectDirs[beadsID] = agentProjectDir
			}
		}
	}

	// 4. Batch fetch all comments with project-aware lookup for cross-project agents
	commentsMap := verify.GetCommentsBatchWithProjectDirs(beadsIDsToFetch, beadsProjectDirs)

	// 5. Batch fetch issue details to check closed status
	// This also provides task info for closed issues (not returned by ListOpenIssues)
	allIssues, _ := verify.GetIssuesBatch(beadsIDsToFetch)

	// === Now build agents from collected data ===

	// Process tmux agents
	for _, ta := range tmuxAgents {
		// Check if there's an active OpenCode session for this beads ID
		session := beadsToSession[ta.beadsID]
		sessionID := ""
		runtime := "unknown"
		isPhantom := true
		isProcessing := false

		if session != nil {
			sessionID = session.ID
			createdAt := time.Unix(session.Time.Created/1000, 0)
			runtime = formatDuration(now.Sub(createdAt))
			isPhantom = false
			// Check if the session is actively processing (has pending response)
			isProcessing = client.IsSessionProcessing(session.ID)
		} else {
			sessionID = "tmux-stalled"
		}

		// Get phase from pre-fetched comments
		var phase, task string
		var isCompleted bool
		if comments, ok := commentsMap[ta.beadsID]; ok {
			phaseStatus := verify.ParsePhaseFromComments(comments)
			if phaseStatus.Found {
				phase = phaseStatus.Phase
			}
		}
		// Get task and check closed status from pre-fetched issues
		if issue, ok := allIssues[ta.beadsID]; ok {
			task = truncate(issue.Title, 40)
			isCompleted = strings.EqualFold(issue.Status, "closed")
		}

		agents = append(agents, AgentInfo{
			SessionID:    sessionID,
			BeadsID:      ta.beadsID,
			Skill:        ta.skill,
			Title:        ta.title,
			Runtime:      runtime,
			Window:       ta.window,
			Phase:        phase,
			Task:         task,
			Project:      ta.project,
			IsPhantom:    isPhantom,
			IsProcessing: isProcessing,
			IsCompleted:  isCompleted,
		})
	}

	// Process OpenCode agents (not in tmux)
	for _, oa := range opcodeAgents {
		createdAt := time.Unix(oa.session.Time.Created/1000, 0)
		runtime := formatDuration(now.Sub(createdAt))

		// OpenCode agents are NOT phantom because they have a running session.
		// Phantom means "beads issue open but agent not running" - but these agents ARE running.
		isPhantom := false

		// Check if the session is actively processing (has pending response)
		isProcessing := client.IsSessionProcessing(oa.session.ID)

		// Get issue for task and closed status
		issue := allIssues[oa.beadsID]

		// Get phase from pre-fetched comments
		var phase, task string
		var isCompleted bool
		if comments, ok := commentsMap[oa.beadsID]; ok {
			phaseStatus := verify.ParsePhaseFromComments(comments)
			if phaseStatus.Found {
				phase = phaseStatus.Phase
			}
		}
		if issue != nil {
			task = truncate(issue.Title, 40)
			isCompleted = strings.EqualFold(issue.Status, "closed")
		}

		agents = append(agents, AgentInfo{
			SessionID:    oa.session.ID,
			Title:        oa.session.Title,
			Runtime:      runtime,
			BeadsID:      oa.beadsID,
			Skill:        oa.skill,
			Phase:        phase,
			Task:         task,
			Project:      oa.project,
			IsPhantom:    isPhantom,
			IsProcessing: isProcessing,
			IsCompleted:  isCompleted,
		})
	}

	// Phase 3: Filter agents based on flags
	filteredAgents := make([]AgentInfo, 0)
	for _, agent := range agents {
		// Filter by project if specified
		if statusProject != "" && agent.Project != statusProject {
			continue
		}
		// Filter phantoms unless --all is set
		if agent.IsPhantom && !statusAll {
			continue
		}
		// Filter completed agents (beads issue closed) unless --all is set
		if agent.IsCompleted && !statusAll {
			continue
		}
		filteredAgents = append(filteredAgents, agent)
	}

	// Phase 4: Build swarm status (counts before filtering)
	activeCount := 0
	processingCount := 0
	idleCount := 0
	phantomCount := 0
	completedCount := 0
	for _, agent := range agents {
		if agent.IsPhantom {
			phantomCount++
		} else if agent.IsCompleted {
			// Completed agents (beads issue closed) don't count as active
			completedCount++
		} else {
			activeCount++
			if agent.IsProcessing {
				processingCount++
			} else {
				idleCount++
			}
		}
	}

	swarm := SwarmStatus{
		Active:     activeCount,
		Processing: processingCount,
		Idle:       idleCount,
		Phantom:    phantomCount,
		Queued:     0,              // TODO: implement queuing system
		Completed:  completedCount, // Agents with closed beads issues
	}

	// Fetch account usage information
	accounts := getAccountUsage()

	// Fetch token usage for each agent with a valid session ID
	for i := range filteredAgents {
		if filteredAgents[i].SessionID != "" && filteredAgents[i].SessionID != "tmux-stalled" {
			tokens, err := client.GetSessionTokens(filteredAgents[i].SessionID)
			if err == nil && tokens != nil {
				filteredAgents[i].Tokens = tokens
			}
		}
	}

	// Build output (use filtered agents for display)
	output := StatusOutput{
		Swarm:    swarm,
		Accounts: accounts,
		Agents:   filteredAgents,
	}

	// Output as JSON if flag is set
	if statusJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Print human-readable output
	printSwarmStatus(output, statusAll)
	return nil
}

// runStatusSessionStart outputs only SessionStart surfacing info.
// This is designed for use in SessionStart hooks to create pressure
// to review high-value design work (architect recommendations) and
// check usage limits.
//
// Output format is human-readable but minimal - just the warnings.
// If there's nothing to surface, output is empty.
func runStatusSessionStart() error {
	var hasOutput bool

	// Surface architect recommendations if any
	surface, err := GetArchitectRecommendationsSurface()
	if err == nil && surface.TotalCount > 0 {
		fmt.Print(FormatArchitectRecommendationsSurface(surface))
		hasOutput = true
	}

	// Surface usage warnings if at risk
	usageWarning := getUsageWarningForSession()
	if usageWarning != "" {
		if hasOutput {
			fmt.Println() // Add separator
		}
		fmt.Print(usageWarning)
		hasOutput = true
	}

	// If nothing to surface, output is empty (silent success)
	// This allows hooks to check for empty output
	return nil
}

// getUsageWarningForSession returns a usage warning message if usage is high.
// Returns empty string if usage is OK.
func getUsageWarningForSession() string {
	// Get current usage summary (includes warning status)
	summary, isWarning := usage.GetUsageSummary()
	if !isWarning {
		return "" // Usage is OK
	}

	// Return the warning with action suggestion
	return fmt.Sprintf("%s\n   Consider: orch account switch <backup-account>\n", summary)
}

// extractProjectFromBeadsID extracts the project name from a beads ID.
// Beads IDs follow the format: project-xxxx (e.g., orch-go-3anf)
func extractProjectFromBeadsID(beadsID string) string {
	if beadsID == "" {
		return ""
	}
	// Find the last hyphen followed by 4 alphanumeric characters (the hash)
	// The project is everything before that
	parts := strings.Split(beadsID, "-")
	if len(parts) < 2 {
		return beadsID
	}
	// The last part should be the 4-char hash, join everything else
	return strings.Join(parts[:len(parts)-1], "-")
}

// extractDateFromWorkspaceName parses the date suffix from a workspace name.
// Workspace names follow format: prefix-description-DDmon (e.g., og-feat-add-feature-24dec)
// Returns zero time if no valid date found.
func extractDateFromWorkspaceName(name string) time.Time {
	// Month abbreviations (lowercase)
	months := map[string]time.Month{
		"jan": time.January,
		"feb": time.February,
		"mar": time.March,
		"apr": time.April,
		"may": time.May,
		"jun": time.June,
		"jul": time.July,
		"aug": time.August,
		"sep": time.September,
		"oct": time.October,
		"nov": time.November,
		"dec": time.December,
	}

	// Get the last segment after the final hyphen
	parts := strings.Split(name, "-")
	if len(parts) == 0 {
		return time.Time{}
	}
	lastPart := strings.ToLower(parts[len(parts)-1])

	// Pattern: 1-2 digits followed by 3-letter month abbreviation (e.g., "24dec", "5jan")
	if len(lastPart) < 4 || len(lastPart) > 5 {
		return time.Time{}
	}

	// Extract the month abbreviation (last 3 chars)
	monthStr := lastPart[len(lastPart)-3:]
	month, ok := months[monthStr]
	if !ok {
		return time.Time{}
	}

	// Extract the day (remaining digits)
	dayStr := lastPart[:len(lastPart)-3]
	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		return time.Time{}
	}

	// Use current year, adjusting for year boundary
	// (if the date is in the future within this calendar, it's probably from last year)
	now := time.Now()
	year := now.Year()
	parsedDate := time.Date(year, month, day, 12, 0, 0, 0, time.Local)

	// If the parsed date is more than a week in the future, assume it's from last year
	if parsedDate.After(now.AddDate(0, 0, 7)) {
		parsedDate = time.Date(year-1, month, day, 12, 0, 0, 0, time.Local)
	}

	return parsedDate
}

// getPhaseAndTask retrieves the current phase and task description from beads.
func getPhaseAndTask(beadsID string) (phase, task string) {
	// Get issue for task description
	issue, err := verify.GetIssue(beadsID)
	if err == nil {
		task = truncate(issue.Title, 40)
	}

	// Get phase from comments
	status, err := verify.GetPhaseStatus(beadsID)
	if err == nil && status.Found {
		phase = status.Phase
	}

	return phase, task
}

// getAccountUsage fetches usage info for all configured accounts.
func getAccountUsage() []AccountUsage {
	var accounts []AccountUsage

	// Get current account usage
	currentUsage := usage.FetchUsage()
	if currentUsage.Error == "" && currentUsage.SevenDay != nil {
		current := AccountUsage{
			Name:        "current",
			Email:       currentUsage.Email,
			UsedPercent: currentUsage.SevenDay.Utilization,
			IsActive:    true,
		}
		if currentUsage.SevenDay.ResetsAt != nil {
			current.ResetTime = currentUsage.SevenDay.TimeUntilReset()
		}
		accounts = append(accounts, current)
	}

	// Try to get saved accounts info (without switching)
	cfg, err := account.LoadConfig()
	if err == nil {
		for name, acc := range cfg.Accounts {
			if acc.Source == "saved" {
				// Check if this is the current account (by email match)
				isCurrentAccount := false
				for i := range accounts {
					if accounts[i].Email == acc.Email {
						accounts[i].Name = name // Update name to the saved account name
						isCurrentAccount = true
						break
					}
				}
				if !isCurrentAccount {
					// Add as a saved account (no live usage data without switching)
					accounts = append(accounts, AccountUsage{
						Name:     name,
						Email:    acc.Email,
						IsActive: false,
					})
				}
			}
		}
	}

	return accounts
}

// Terminal width thresholds for adaptive output
const (
	termWidthWide   = 120 // Full table with all columns
	termWidthNarrow = 100 // Drop TASK column, abbreviate SKILL
	termWidthMin    = 80  // Minimum supported width (vertical card format)
)

// getTerminalWidth returns the current terminal width, or a default if detection fails.
// Returns the width and whether we're outputting to a real terminal.
func getTerminalWidth() (int, bool) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Not a terminal (piped output) - use wide format
		return termWidthWide + 1, false
	}
	return width, true
}

// printSwarmStatus prints the swarm status in human-readable format.
// Adapts output format based on terminal width.
func printSwarmStatus(output StatusOutput, showAll bool) {
	width, _ := getTerminalWidth()
	printSwarmStatusWithWidth(output, showAll, width)
}

// printSwarmStatusWithWidth prints swarm status with explicit width (for testing).
func printSwarmStatusWithWidth(output StatusOutput, showAll bool, termWidth int) {
	// Print swarm summary header with processing breakdown
	fmt.Printf("SWARM STATUS: Active: %d", output.Swarm.Active)
	if output.Swarm.Active > 0 {
		fmt.Printf(" (running: %d, idle: %d)", output.Swarm.Processing, output.Swarm.Idle)
	}
	if output.Swarm.Completed > 0 {
		fmt.Printf(", Completed: %d", output.Swarm.Completed)
		if !showAll {
			fmt.Printf(" (use --all to show)")
		}
	}
	if output.Swarm.Phantom > 0 {
		fmt.Printf(", Phantom: %d", output.Swarm.Phantom)
		if !showAll {
			fmt.Printf(" (use --all to show)")
		}
	}
	fmt.Println()
	fmt.Println()

	// Surface architect recommendations if any (with rich detail for SessionStart awareness)
	if surface, err := GetArchitectRecommendationsSurface(); err == nil && surface.TotalCount > 0 {
		fmt.Print(FormatArchitectRecommendationsSurface(surface))
		fmt.Println()
	}

	// Print account usage
	if len(output.Accounts) > 0 {
		fmt.Println("ACCOUNTS")
		for _, acc := range output.Accounts {
			activeMarker := ""
			if acc.IsActive {
				activeMarker = " *"
			}
			usageStr := "N/A"
			if acc.UsedPercent > 0 || acc.IsActive {
				usageStr = fmt.Sprintf("%.0f%% used", acc.UsedPercent)
				if acc.ResetTime != "" {
					usageStr += fmt.Sprintf(" (resets in %s)", acc.ResetTime)
				}
			}
			name := acc.Name
			if acc.Email != "" && acc.Name == "current" {
				name = acc.Email
			}
			fmt.Printf("  %-20s %s%s\n", name+":", usageStr, activeMarker)
		}
		fmt.Println()
	}

	// Print agents in format appropriate for terminal width
	if len(output.Agents) > 0 {
		fmt.Println("AGENTS")
		if termWidth < termWidthMin {
			printAgentsCardFormat(output.Agents)
		} else if termWidth < termWidthNarrow {
			printAgentsNarrowFormat(output.Agents)
		} else {
			printAgentsWideFormat(output.Agents)
		}
	} else {
		fmt.Println("No active agents")
	}
}

// printAgentsWideFormat prints agents in full table format (>120 chars).
// Columns: BEADS ID, STATUS, PHASE, TASK, SKILL, RUNTIME, TOKENS
func printAgentsWideFormat(agents []AgentInfo) {
	fmt.Printf("  %-18s %-8s %-12s %-28s %-15s %-8s %s\n", "BEADS ID", "STATUS", "PHASE", "TASK", "SKILL", "RUNTIME", "TOKENS")
	fmt.Printf("  %s\n", strings.Repeat("-", 115))

	for _, agent := range agents {
		beadsID := agent.BeadsID
		if beadsID == "" {
			beadsID = "-"
		}
		phase := agent.Phase
		if phase == "" {
			phase = "-"
		}
		task := agent.Task
		if task == "" {
			task = "-"
		}
		skill := agent.Skill
		if skill == "" {
			skill = "-"
		}
		status := getAgentStatus(agent)
		tokens := formatTokenStatsCompact(agent.Tokens)

		fmt.Printf("  %-18s %-8s %-12s %-28s %-15s %-8s %s\n",
			beadsID,
			status,
			truncate(phase, 10),
			truncate(task, 26),
			truncate(skill, 13),
			agent.Runtime,
			tokens)
	}
}

// printAgentsNarrowFormat prints agents in narrow format (80-100 chars).
// Drops TASK column, abbreviates SKILL.
// Columns: BEADS ID, STATUS, PHASE, SKILL, RUNTIME, TOKENS
func printAgentsNarrowFormat(agents []AgentInfo) {
	fmt.Printf("  %-18s %-8s %-12s %-10s %-8s %s\n", "BEADS ID", "STATUS", "PHASE", "SKILL", "RUNTIME", "TOKENS")
	fmt.Printf("  %s\n", strings.Repeat("-", 75))

	for _, agent := range agents {
		beadsID := agent.BeadsID
		if beadsID == "" {
			beadsID = "-"
		}
		phase := agent.Phase
		if phase == "" {
			phase = "-"
		}
		skill := abbreviateSkill(agent.Skill)
		if skill == "" {
			skill = "-"
		}
		status := getAgentStatus(agent)
		tokens := formatTokenStatsCompact(agent.Tokens)

		fmt.Printf("  %-18s %-8s %-12s %-10s %-8s %s\n",
			beadsID,
			status,
			truncate(phase, 10),
			truncate(skill, 8),
			agent.Runtime,
			tokens)
	}
}

// printAgentsCardFormat prints agents in vertical card format (<80 chars).
// Each agent is a multi-line block for readability on very narrow terminals.
func printAgentsCardFormat(agents []AgentInfo) {
	for i, agent := range agents {
		if i > 0 {
			fmt.Println()
		}
		beadsID := agent.BeadsID
		if beadsID == "" {
			beadsID = "-"
		}
		phase := agent.Phase
		if phase == "" {
			phase = "-"
		}
		task := agent.Task
		if task == "" {
			task = "-"
		}
		skill := agent.Skill
		if skill == "" {
			skill = "-"
		}
		status := getAgentStatus(agent)

		fmt.Printf("  %s [%s]\n", beadsID, status)
		fmt.Printf("    Phase: %s | Skill: %s\n", phase, skill)
		fmt.Printf("    Task: %s\n", truncate(task, 50))
		fmt.Printf("    Runtime: %s | Tokens: %s\n", agent.Runtime, formatTokenStats(agent.Tokens))
	}
}

// getAgentStatus returns a status string based on agent state.
func getAgentStatus(agent AgentInfo) string {
	if agent.IsCompleted {
		return "completed"
	}
	if agent.IsPhantom {
		return "phantom"
	}
	if agent.IsProcessing {
		return "running"
	}
	return "idle"
}

// abbreviateSkill returns a shortened version of skill names for narrow displays.
func abbreviateSkill(skill string) string {
	abbreviations := map[string]string{
		"feature-impl":         "feat",
		"investigation":        "inv",
		"systematic-debugging": "debug",
		"architect":            "arch",
		"codebase-audit":       "audit",
		"reliability-testing":  "rel-test",
		"issue-creation":       "issue",
		"design-session":       "design",
		"research":             "research",
	}
	if abbr, ok := abbreviations[skill]; ok {
		return abbr
	}
	return skill
}

// formatTokenCount formats a token count with K/M suffixes for readability.
func formatTokenCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}
	if count < 1000000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(count)/1000000)
}

// formatTokenStats returns a formatted string of token usage.
func formatTokenStats(tokens *opencode.TokenStats) string {
	if tokens == nil {
		return "-"
	}
	// Format: "in:X out:Y (cache:Z)"
	result := fmt.Sprintf("in:%s out:%s", formatTokenCount(tokens.InputTokens), formatTokenCount(tokens.OutputTokens))
	if tokens.CacheReadTokens > 0 {
		result += fmt.Sprintf(" (cache:%s)", formatTokenCount(tokens.CacheReadTokens))
	}
	return result
}

// formatTokenStatsCompact returns a compact formatted string of token usage for table display.
// Shows total tokens with input/output breakdown: "12.5K (8K/4K)"
func formatTokenStatsCompact(tokens *opencode.TokenStats) string {
	if tokens == nil {
		return "-"
	}
	total := tokens.TotalTokens
	if total == 0 {
		total = tokens.InputTokens + tokens.OutputTokens
	}
	if total == 0 {
		return "-"
	}
	// Format: "total (in/out)" for quick scanning
	return fmt.Sprintf("%s (%s/%s)",
		formatTokenCount(total),
		formatTokenCount(tokens.InputTokens),
		formatTokenCount(tokens.OutputTokens))
}

// findWorkspaceByBeadsID searches for a workspace directory spawned from the beads ID.
// Looks in .orch/workspace/ for directories that match the beads ID in their name
// or contain a SPAWN_CONTEXT.md with "spawned from beads issue: **beadsID**".
// Returns the workspace path and agent name (directory name) if found.
func findWorkspaceByBeadsID(projectDir, beadsID string) (workspacePath, agentName string) {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return "", ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		// Check if the beads ID is in the directory name
		// Workspace names follow format: og-feat-description-21dec
		// Beads ID format: project-xxxx (e.g., orch-go-3anf)
		if strings.Contains(dirName, beadsID) {
			return dirPath, dirName
		}

		// Check SPAWN_CONTEXT.md for authoritative "spawned from beads issue" line
		// This is more precise than just checking if beadsID appears anywhere
		spawnContextPath := filepath.Join(dirPath, "SPAWN_CONTEXT.md")
		if content, err := os.ReadFile(spawnContextPath); err == nil {
			contentStr := string(content)
			// Look for the authoritative beads issue declaration
			// Pattern: "spawned from beads issue: **orch-go-xxxx**" or similar
			for _, line := range strings.Split(contentStr, "\n") {
				lineLower := strings.ToLower(line)
				if strings.Contains(lineLower, "spawned from beads issue:") {
					if strings.Contains(line, beadsID) {
						return dirPath, dirName
					}
					// Found the authoritative line but beads ID doesn't match
					// Don't continue searching this file - this workspace has a different ID
					break
				}
			}
		}
	}

	return "", ""
}

func runComplete(beadsID, workdir string) error {
	// Get current directory as base project dir
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine beads project directory:
	// 1. If --workdir provided, use that
	// 2. Otherwise, try to auto-detect from workspace SPAWN_CONTEXT.md
	// 3. Fall back to current directory
	var beadsProjectDir string
	var workspacePath, agentName string

	// First, find workspace in current project (even for cross-project agents, workspace is in orchestrator's project)
	workspacePath, agentName = findWorkspaceByBeadsID(currentDir, beadsID)

	if workdir != "" {
		// Explicit --workdir flag provided
		beadsProjectDir, err = filepath.Abs(workdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		// Verify directory exists
		if stat, err := os.Stat(beadsProjectDir); err != nil {
			return fmt.Errorf("workdir does not exist: %s", beadsProjectDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", beadsProjectDir)
		}
		fmt.Printf("Using explicit workdir: %s\n", beadsProjectDir)
	} else if workspacePath != "" {
		// Try to extract PROJECT_DIR from workspace SPAWN_CONTEXT.md
		projectDirFromWorkspace := extractProjectDirFromWorkspace(workspacePath)
		if projectDirFromWorkspace != "" && projectDirFromWorkspace != currentDir {
			// Cross-project agent detected
			beadsProjectDir = projectDirFromWorkspace
			fmt.Printf("Auto-detected cross-project: %s\n", filepath.Base(beadsProjectDir))
		} else {
			beadsProjectDir = currentDir
		}
	} else {
		beadsProjectDir = currentDir
	}

	// Set beads.DefaultDir for cross-project operations BEFORE any beads operations
	if beadsProjectDir != currentDir {
		beads.DefaultDir = beadsProjectDir
	}

	// Get issue to verify it exists
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		// Provide helpful error message for cross-project issues
		projectName := filepath.Base(beadsProjectDir)
		issuePrefix := strings.Split(beadsID, "-")[0]
		if len(strings.Split(beadsID, "-")) > 1 {
			issuePrefix = strings.Join(strings.Split(beadsID, "-")[:len(strings.Split(beadsID, "-"))-1], "-")
		}
		if issuePrefix != projectName {
			return fmt.Errorf("failed to get beads issue %s: %w\n\nHint: The issue ID suggests it belongs to project '%s', but you're in '%s'.\nTry: orch complete %s --workdir ~/path/to/%s", beadsID, err, issuePrefix, projectName, beadsID, issuePrefix)
		}
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Check if already closed
	isClosed := issue.Status == "closed"
	if isClosed {
		fmt.Printf("Issue %s is already closed in beads\n", beadsID)
	}

	// If --approve flag is set, add approval comment BEFORE verification
	// This ensures the visual verification gate sees the approval
	if completeApprove {
		approvalComment := "✅ APPROVED - Visual changes reviewed and approved by orchestrator"
		if err := addApprovalComment(beadsID, approvalComment); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add approval comment: %v\n", err)
			// Continue anyway - the approval might already exist or we can fallback
		} else {
			fmt.Printf("Added approval: %s\n", approvalComment)
		}
	}

	// Verify phase status unless force flag is set
	if !completeForce {
		// Workspace already found at top of function
		if workspacePath != "" {
			fmt.Printf("Workspace: %s\n", agentName)
		}

		// Use beadsProjectDir for verification (where the beads issue lives)
		result, err := verify.VerifyCompletionFull(beadsID, workspacePath, beadsProjectDir, "")
		if err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}

		if !result.Passed {
			fmt.Fprintf(os.Stderr, "Cannot complete agent - verification failed:\n")
			for _, e := range result.Errors {
				fmt.Fprintf(os.Stderr, "  - %s\n", e)
			}
			fmt.Fprintf(os.Stderr, "\nAgent must run: bd comment %s \"Phase: Complete - <summary>\"\n", beadsID)
			fmt.Fprintf(os.Stderr, "Or use --force to skip verification\n")
			return fmt.Errorf("verification failed")
		}

		// Print constraint warnings
		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "⚠️  %s\n", w)
		}

		// Print phase info
		if result.Phase.Found {
			fmt.Printf("Phase: %s\n", result.Phase.Phase)
			if result.Phase.Summary != "" {
				fmt.Printf("Summary: %s\n", result.Phase.Summary)
			}
		}
	} else {
		fmt.Println("Skipping phase verification (--force)")
	}

	// Check liveness before closing - warn if agent appears still running
	if !completeForce {
		liveness := state.GetLiveness(beadsID, serverURL, beadsProjectDir)
		if liveness.IsAlive() {
			// Build warning message with details about what's still running
			var runningDetails []string
			if liveness.TmuxLive {
				detail := "tmux window"
				if liveness.WindowID != "" {
					detail += " (" + liveness.WindowID + ")"
				}
				runningDetails = append(runningDetails, detail)
			}
			if liveness.OpencodeLive {
				detail := "OpenCode session"
				if liveness.SessionID != "" {
					detail += " (" + liveness.SessionID[:12] + ")"
				}
				runningDetails = append(runningDetails, detail)
			}

			fmt.Fprintf(os.Stderr, "⚠️  Agent appears still running: %s\n", strings.Join(runningDetails, ", "))

			// Check if stdin is a terminal for interactive prompting
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("agent still running and stdin is not a terminal; use --force to complete anyway")
			}

			// Prompt user for confirmation
			fmt.Fprint(os.Stderr, "Proceed anyway? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("aborted: agent still running")
			}

			fmt.Println("Proceeding with completion despite liveness warning...")
		}
	}

	// Check synthesis for follow-up recommendations (workspace already found at top)
	if workspacePath != "" {
		synthesis, err := verify.ParseSynthesis(workspacePath)
		if err == nil && synthesis != nil {
			// Check if there are follow-up recommendations to surface
			hasFollowUp := false
			if synthesis.Recommendation == "spawn-follow-up" || synthesis.Recommendation == "escalate" || synthesis.Recommendation == "resume" {
				hasFollowUp = true
			}
			if len(synthesis.NextActions) > 0 {
				hasFollowUp = true
			}

			if hasFollowUp {
				fmt.Println("\n--- Follow-up Recommendations ---")

				if synthesis.Recommendation != "" && synthesis.Recommendation != "close" {
					fmt.Printf("Recommendation: %s\n", synthesis.Recommendation)
				}

				// Collect all actionable items
				var actionableItems []string
				actionableItems = append(actionableItems, synthesis.NextActions...)
				actionableItems = append(actionableItems, synthesis.AreasToExplore...)
				actionableItems = append(actionableItems, synthesis.Uncertainties...)

				if len(actionableItems) > 0 {
					fmt.Printf("\n%d actionable items found:\n", len(actionableItems))
					for i, action := range actionableItems {
						fmt.Printf("  %d. %s\n", i+1, action)
					}
				}

				fmt.Println("\n---------------------------------")

				// Prompt for each actionable item (only if stdin is a terminal)
				if len(actionableItems) > 0 {
					if !term.IsTerminal(int(os.Stdin.Fd())) {
						fmt.Println("(Skipping interactive prompts - stdin is not a terminal)")
					} else {
						reader := bufio.NewReader(os.Stdin)
						createdCount := 0

						for i, action := range actionableItems {
							fmt.Printf("\n[%d/%d] %s\n", i+1, len(actionableItems), action)
							fmt.Print("Create issue? [y/N/q to quit]: ")
							response, err := reader.ReadString('\n')
							if err != nil {
								break
							}
							response = strings.TrimSpace(strings.ToLower(response))

							if response == "q" || response == "quit" {
								fmt.Println("Skipping remaining items.")
								break
							}

							if response == "y" || response == "yes" {
								// Create the issue using beads
								issue, err := beads.FallbackCreate(action, "", "task", 2, []string{"triage:review"})
								if err != nil {
									fmt.Fprintf(os.Stderr, "  Failed to create issue: %v\n", err)
								} else {
									fmt.Printf("  Created: %s\n", issue.ID)
									createdCount++
								}
							}
						}

						if createdCount > 0 {
							fmt.Printf("\n✓ Created %d follow-up issue(s)\n", createdCount)
						}
					}
				}
			}
		}
	}

	// Determine close reason
	reason := completeReason
	if reason == "" {
		// Try to get summary from phase status
		status, _ := verify.GetPhaseStatus(beadsID)
		if status.Summary != "" {
			reason = status.Summary
		} else {
			reason = "Completed via orch complete"
		}
	}

	// Close the beads issue if not already closed
	if !isClosed {
		if err := verify.CloseIssue(beadsID, reason); err != nil {
			return fmt.Errorf("failed to close issue: %w", err)
		}
		fmt.Printf("Closed beads issue: %s\n", beadsID)
	}
	fmt.Printf("Reason: %s\n", reason)

	// Clean up tmux window if it exists (prevents phantom accumulation)
	if window, sessionName, err := tmux.FindWindowByBeadsIDAllSessions(beadsID); err == nil && window != nil {
		if err := tmux.KillWindow(window.Target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close tmux window %s: %v\n", window.Target, err)
		} else {
			fmt.Printf("Closed tmux window: %s:%s\n", sessionName, window.Name)
		}
	}

	// Auto-rebuild if agent committed Go changes (in the beads project)
	if hasGoChangesInRecentCommits(beadsProjectDir) {
		fmt.Println("Detected Go file changes in recent commits")
		if err := runAutoRebuild(beadsProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-rebuild failed: %v\n", err)
		} else {
			fmt.Println("Auto-rebuild completed: make install")
			// Restart orch serve if running
			if restarted, err := restartOrchServe(beadsProjectDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restart orch serve: %v\n", err)
			} else if restarted {
				fmt.Println("Restarted orch serve")
			}
		}

		// Check for new CLI commands that may need skill documentation
		newCommands := detectNewCLICommands(beadsProjectDir)
		if len(newCommands) > 0 {
			fmt.Println()
			fmt.Println("┌─────────────────────────────────────────────────────────────┐")
			fmt.Println("│  📚 NEW CLI COMMANDS DETECTED                               │")
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			for _, cmd := range newCommands {
				fmt.Printf("│  • %s\n", cmd)
			}
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			fmt.Println("│  Consider updating skill documentation:                     │")
			fmt.Println("│  - ~/.claude/skills/meta/orchestrator/SKILL.md              │")
			fmt.Println("│  - docs/orch-commands-reference.md                          │")
			fmt.Println("└─────────────────────────────────────────────────────────────┘")
		}
	}

	// Log the completion
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agent.completed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id": beadsID,
			"reason":   reason,
			"forced":   completeForce,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	return nil
}

// addApprovalComment adds an approval comment to a beads issue.
// This is used by --approve flag to mark visual changes as human-reviewed.
func addApprovalComment(beadsID, comment string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		// Use "orchestrator" as the author for approval comments
		err := client.AddComment(beadsID, "orchestrator", comment)
		if err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackAddComment(beadsID, comment)
}

// hasGoChangesInRecentCommits checks if any of the last 5 commits contain changes
// to cmd/orch/*.go or pkg/*.go files.
func hasGoChangesInRecentCommits(projectDir string) bool {
	// Get changed files from last 5 commits
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails (e.g., not enough commits), try last 1 commit
		cmd = exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return false
		}
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if file matches cmd/orch/*.go or pkg/*.go or pkg/**/*.go
		if strings.HasPrefix(line, "cmd/orch/") && strings.HasSuffix(line, ".go") {
			return true
		}
		if strings.HasPrefix(line, "pkg/") && strings.HasSuffix(line, ".go") {
			return true
		}
	}
	return false
}

// detectNewCLICommands checks if any of the last 5 commits added new CLI command files
// to cmd/orch/. A file is considered a new command if:
// 1. It's in cmd/orch/*.go (not a test file)
// 2. It was added (not modified) in recent commits
// 3. It contains cobra.Command definitions
// Returns the list of new command file names (without path prefix).
func detectNewCLICommands(projectDir string) []string {
	var newCommands []string

	// Get files added (not modified) in last 5 commits
	// The 'A' status means added
	cmd := exec.Command("git", "diff", "--name-status", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails (e.g., not enough commits), try last 1 commit
		cmd = exec.Command("git", "diff", "--name-status", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return nil
		}
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse status line: "A\tcmd/orch/newcmd.go" or "M\tcmd/orch/main.go"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		filePath := parts[1]

		// Only care about added files (not modified)
		if status != "A" {
			continue
		}

		// Only check cmd/orch/*.go files (not test files)
		if !strings.HasPrefix(filePath, "cmd/orch/") || !strings.HasSuffix(filePath, ".go") {
			continue
		}
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}

		// Read the file to check if it contains cobra command definitions
		fullPath := filepath.Join(projectDir, filePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		// Look for cobra command pattern: "var xxxCmd = &cobra.Command{"
		if strings.Contains(string(content), "cobra.Command{") &&
			strings.Contains(string(content), "rootCmd.AddCommand(") {
			// Extract just the filename
			fileName := filepath.Base(filePath)
			newCommands = append(newCommands, fileName)
		}
	}

	return newCommands
}

// runAutoRebuild runs make install in the project directory.
func runAutoRebuild(projectDir string) error {
	cmd := exec.Command("make", "install")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// restartOrchServe checks if orch serve is running and restarts it.
// Returns true if it was restarted, false if it wasn't running.
func restartOrchServe(projectDir string) (bool, error) {
	// Find the orch serve process
	// We look for processes matching "orch serve" or "orch-go serve"
	cmd := exec.Command("pgrep", "-f", "orch.*serve")
	output, err := cmd.Output()
	if err != nil {
		// No process found - that's fine, just means serve isn't running
		return false, nil
	}

	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(pids) == 0 || pids[0] == "" {
		return false, nil
	}

	// Get the current PID to avoid killing ourselves
	currentPID := os.Getpid()

	// Kill the serve process(es)
	var killedAny bool
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
		if err != nil {
			continue
		}
		// Don't kill ourselves
		if pid == currentPID {
			continue
		}
		// Send SIGTERM for graceful shutdown
		killCmd := exec.Command("kill", "-TERM", pidStr)
		if err := killCmd.Run(); err == nil {
			killedAny = true
		}
	}

	if !killedAny {
		return false, nil
	}

	// Wait a moment for the process to stop
	time.Sleep(500 * time.Millisecond)

	// Start orch serve in the background
	// We use nohup to ensure it survives after we exit
	serveCmd := exec.Command("nohup", "orch", "serve")
	serveCmd.Dir = projectDir
	// Redirect output to files to avoid blocking
	devNull, _ := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	serveCmd.Stdout = devNull
	serveCmd.Stderr = devNull
	if err := serveCmd.Start(); err != nil {
		return true, fmt.Errorf("killed old serve but failed to start new: %w", err)
	}

	return true, nil
}

func runMonitor(serverURL string) error {
	// Use the new CompletionService which handles:
	// - SSE monitoring with automatic reconnection
	// - Desktop notifications
	// - Registry updates
	// - Beads phase updates
	service, err := opencode.NewCompletionService(serverURL)
	if err != nil {
		return fmt.Errorf("failed to create completion service: %w", err)
	}

	fmt.Printf("Monitoring SSE events at %s/event...\n", serverURL)
	fmt.Println("On session completion:")
	fmt.Println("  - Desktop notification sent")
	fmt.Println("  - Registry updated")
	fmt.Println("  - Beads phase updated (if applicable)")
	fmt.Println("Press Ctrl+C to stop")

	service.Start()

	// Block forever - the user will Ctrl+C to stop
	select {}
}

var (
	// Clean command flags
	cleanDryRun         bool
	cleanVerifyOpenCode bool
	cleanWindows        bool
	cleanPhantoms       bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "List completed agents and optionally close their resources",
	Long: `List completed agents and optionally clean up their resources.

By default, this command only REPORTS what could be cleaned - it does not delete
anything. Workspace directories are always preserved for investigation reference.

What counts as "completed":
- Workspaces with SYNTHESIS.md file
- Workspaces whose beads issue is closed

Optional cleanup actions:
  --windows         Close tmux windows for completed agents
  --phantoms        Close phantom tmux windows (beads ID but no active session)
  --verify-opencode Delete orphaned OpenCode disk sessions (not tracked in workspaces)

Note: This command never deletes workspace directories - they are kept for 
investigation reference. Use 'rm -rf .orch/workspace/<name>' to manually delete.

Examples:
  orch-go clean                   # List completed agents (no changes)
  orch-go clean --dry-run         # Preview mode (same as default)
  orch-go clean --windows         # Close tmux windows for completed agents
  orch-go clean --phantoms        # Close phantom tmux windows
  orch-go clean --verify-opencode # Delete orphaned OpenCode disk sessions`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClean(cleanDryRun, cleanVerifyOpenCode, cleanWindows, cleanPhantoms)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanCmd.Flags().BoolVar(&cleanVerifyOpenCode, "verify-opencode", false, "Also verify OpenCode disk sessions (slower)")
	cleanCmd.Flags().BoolVar(&cleanWindows, "windows", false, "Close tmux windows for completed agents")
	cleanCmd.Flags().BoolVar(&cleanPhantoms, "phantoms", false, "Close all phantom tmux windows (stale agent windows)")
}

// DefaultLivenessChecker checks if tmux windows and OpenCode sessions exist.
type DefaultLivenessChecker struct {
	client *opencode.Client
}

// NewDefaultLivenessChecker creates a new liveness checker.
func NewDefaultLivenessChecker(serverURL string) *DefaultLivenessChecker {
	return &DefaultLivenessChecker{
		client: opencode.NewClient(serverURL),
	}
}

// WindowExists checks if a tmux window ID exists.
func (c *DefaultLivenessChecker) WindowExists(windowID string) bool {
	return tmux.WindowExistsByID(windowID)
}

// SessionExists checks if an OpenCode session ID exists.
func (c *DefaultLivenessChecker) SessionExists(sessionID string) bool {
	return c.client.SessionExists(sessionID)
}

// DefaultBeadsStatusChecker checks beads issue status using the verify package.
type DefaultBeadsStatusChecker struct{}

// NewDefaultBeadsStatusChecker creates a new beads status checker.
func NewDefaultBeadsStatusChecker() *DefaultBeadsStatusChecker {
	return &DefaultBeadsStatusChecker{}
}

// IsIssueClosed checks if a beads issue is closed.
func (c *DefaultBeadsStatusChecker) IsIssueClosed(beadsID string) bool {
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		// If we can't get the issue, assume it's not closed
		// (could be network error, issue not found, etc.)
		return false
	}
	return issue.Status == "closed"
}

// DefaultCompletionIndicatorChecker checks for completion indicators (SYNTHESIS.md, Phase: Complete).
// This is used to determine if an agent completed its work.
type DefaultCompletionIndicatorChecker struct{}

// NewDefaultCompletionIndicatorChecker creates a new completion indicator checker.
func NewDefaultCompletionIndicatorChecker() *DefaultCompletionIndicatorChecker {
	return &DefaultCompletionIndicatorChecker{}
}

// SynthesisExists checks if SYNTHESIS.md exists in the agent's workspace.
func (c *DefaultCompletionIndicatorChecker) SynthesisExists(workspacePath string) bool {
	exists, err := verify.VerifySynthesis(workspacePath)
	if err != nil {
		// If we can't check (e.g., directory doesn't exist), assume no synthesis
		return false
	}
	return exists
}

// IsPhaseComplete checks if beads shows Phase: Complete for the agent.
func (c *DefaultCompletionIndicatorChecker) IsPhaseComplete(beadsID string) bool {
	complete, err := verify.IsPhaseComplete(beadsID)
	if err != nil {
		// If we can't check (e.g., beads error), assume not complete
		return false
	}
	return complete
}

// CleanableWorkspace represents a workspace that can be cleaned.
type CleanableWorkspace struct {
	Name       string // Workspace directory name
	Path       string // Full path to workspace
	BeadsID    string // Beads issue ID (extracted from SPAWN_CONTEXT.md)
	IsComplete bool   // Has SYNTHESIS.md
	Reason     string // Why it's cleanable
}

// findCleanableWorkspaces scans .orch/workspace/ for completed/abandoned workspaces.
// Returns workspaces that have SYNTHESIS.md OR whose beads issue is closed.
func findCleanableWorkspaces(projectDir string, beadsChecker *DefaultBeadsStatusChecker) []CleanableWorkspace {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil
	}

	var cleanable []CleanableWorkspace

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		// Extract beads ID from SPAWN_CONTEXT.md
		beadsID := ""
		spawnContextPath := filepath.Join(dirPath, "SPAWN_CONTEXT.md")
		if content, err := os.ReadFile(spawnContextPath); err == nil {
			// Look for "beads issue: **xxx**" pattern
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.Contains(line, "beads issue:") || strings.Contains(line, "BEADS ISSUE:") {
					// Extract beads ID from the line
					parts := strings.Fields(line)
					for _, part := range parts {
						// Look for pattern like "orch-go-xxxx" or similar
						if strings.Contains(part, "-") && !strings.HasPrefix(part, "beads") && !strings.HasPrefix(part, "BEADS") {
							// Clean up markdown formatting
							beadsID = strings.Trim(part, "*`[]")
							break
						}
					}
				}
			}
		}

		workspace := CleanableWorkspace{
			Name:    dirName,
			Path:    dirPath,
			BeadsID: beadsID,
		}

		// Check for SYNTHESIS.md (completion indicator)
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		if info, err := os.Stat(synthesisPath); err == nil && info.Size() > 0 {
			workspace.IsComplete = true
			workspace.Reason = "SYNTHESIS.md exists"
			cleanable = append(cleanable, workspace)
			continue
		}

		// Check beads issue status if we have a beads ID
		if beadsID != "" && beadsChecker.IsIssueClosed(beadsID) {
			workspace.IsComplete = true
			workspace.Reason = "beads issue closed"
			cleanable = append(cleanable, workspace)
			continue
		}

		// Check if workspace is orphaned (no tmux window, no OpenCode session, no active beads issue)
		// This would be a workspace from a crashed or abandoned agent
		// For now, we only clean explicitly completed workspaces
	}

	return cleanable
}

func runClean(dryRun bool, verifyOpenCode bool, closeWindows bool, cleanPhantoms bool) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find completed workspaces using derived lookups
	fmt.Println("Scanning workspaces for completed agents...")
	beadsChecker := NewDefaultBeadsStatusChecker()
	cleanableWorkspaces := findCleanableWorkspaces(projectDir, beadsChecker)

	fmt.Printf("\nFound %d completed workspaces\n", len(cleanableWorkspaces))

	if len(cleanableWorkspaces) == 0 && !verifyOpenCode && !cleanPhantoms {
		fmt.Println("No completed agents found")
		return nil
	}

	// Track cleanup stats
	windowsClosed := 0

	// List completed workspaces
	if len(cleanableWorkspaces) > 0 {
		fmt.Printf("\nCompleted workspaces:\n")
		for _, ws := range cleanableWorkspaces {
			fmt.Printf("  %s (%s)\n", ws.Name, ws.Reason)

			// Close tmux window if --windows flag is set
			if closeWindows && !dryRun {
				if window, sessionName, _ := tmux.FindWindowByWorkspaceNameAllSessions(ws.Name); window != nil {
					if err := tmux.KillWindow(window.Target); err != nil {
						fmt.Fprintf(os.Stderr, "    Warning: failed to close window %s: %v\n", window.Name, err)
					} else {
						fmt.Printf("    Closed window: %s in session %s\n", window.Name, sessionName)
						windowsClosed++
					}
				}
			} else if closeWindows && dryRun {
				if window, sessionName, _ := tmux.FindWindowByWorkspaceNameAllSessions(ws.Name); window != nil {
					fmt.Printf("    [DRY-RUN] Would close window: %s in session %s\n", window.Name, sessionName)
				}
			}
		}
	}

	// Verify and clean OpenCode disk sessions (optional)
	var diskSessionsDeleted int
	if verifyOpenCode {
		diskSessionsDeleted, err = cleanOrphanedDiskSessions(serverURL, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean disk sessions: %v\n", err)
		}
	}

	// Clean phantom tmux windows (optional)
	var phantomsClosed int
	if cleanPhantoms {
		phantomsClosed, err = cleanPhantomWindows(serverURL, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean phantom windows: %v\n", err)
		}
	}

	// Check if any cleanup actions were taken or would be taken
	hasCleanupActions := closeWindows || cleanPhantoms || verifyOpenCode

	if dryRun {
		if hasCleanupActions {
			fmt.Printf("\nDry run complete.")
			if closeWindows {
				// Count potential windows to close
				windowCount := 0
				for _, ws := range cleanableWorkspaces {
					if window, _, _ := tmux.FindWindowByWorkspaceNameAllSessions(ws.Name); window != nil {
						windowCount++
					}
				}
				if windowCount > 0 {
					fmt.Printf(" Would close %d tmux windows.", windowCount)
				}
			}
			if cleanPhantoms && phantomsClosed > 0 {
				fmt.Printf(" Would close %d phantom windows.", phantomsClosed)
			}
			if verifyOpenCode && diskSessionsDeleted > 0 {
				fmt.Printf(" Would delete %d orphaned disk sessions.", diskSessionsDeleted)
			}
			fmt.Println()
		}
		return nil
	}

	// Log if any cleanup actions were taken
	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 {
		projectName := filepath.Base(projectDir)
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "agents.cleaned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"completed_workspaces":  len(cleanableWorkspaces),
				"windows_closed":        windowsClosed,
				"phantoms_closed":       phantomsClosed,
				"disk_sessions_deleted": diskSessionsDeleted,
				"project":               projectName,
				"verify_opencode":       verifyOpenCode,
				"close_windows":         closeWindows,
				"clean_phantoms":        cleanPhantoms,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
		}
	}

	// Print summary of actions taken (not misleading "cleaned X workspaces")
	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 {
		fmt.Println()
		if windowsClosed > 0 {
			fmt.Printf("Closed %d tmux windows\n", windowsClosed)
		}
		if phantomsClosed > 0 {
			fmt.Printf("Closed %d phantom windows\n", phantomsClosed)
		}
		if diskSessionsDeleted > 0 {
			fmt.Printf("Deleted %d orphaned disk sessions\n", diskSessionsDeleted)
		}
	} else if !hasCleanupActions {
		// Default: just listing completed workspaces
		fmt.Printf("\nNote: Workspace directories are preserved. Use --windows, --phantoms, or --verify-opencode to clean up resources.\n")
	}

	return nil
}

// cleanOrphanedDiskSessions finds and deletes OpenCode disk sessions that aren't tracked via workspace files.
// Returns the number of sessions deleted and any error encountered.
func cleanOrphanedDiskSessions(serverURL string, dryRun bool) (int, error) {
	// Get current project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Printf("\nVerifying OpenCode disk sessions for %s...\n", projectDir)

	client := opencode.NewClient(serverURL)

	// Fetch all disk sessions for this directory
	diskSessions, err := client.ListDiskSessions(projectDir)
	if err != nil {
		return 0, fmt.Errorf("failed to list disk sessions: %w", err)
	}

	fmt.Printf("  Found %d disk sessions\n", len(diskSessions))

	// Build a set of session IDs that are tracked via workspace files
	trackedSessionIDs := make(map[string]bool)
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	if entries, err := os.ReadDir(workspaceDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				sessionID := spawn.ReadSessionID(filepath.Join(workspaceDir, entry.Name()))
				if sessionID != "" {
					trackedSessionIDs[sessionID] = true
				}
			}
		}
	}

	fmt.Printf("  Workspaces track %d session IDs\n", len(trackedSessionIDs))

	// Find orphaned sessions (disk sessions not tracked in workspaces)
	// IMPORTANT: Exclude sessions that are actively processing (e.g., the current orchestrator session)
	// The orchestrator/interactive sessions don't have workspace .session_id files, but they're
	// still valid sessions that should not be deleted.
	//
	// We use two heuristics to detect active sessions (no extra API calls needed):
	// 1. Recently updated sessions (within last 5 minutes) - likely in use
	// 2. Sessions that are currently processing (expensive check, only if recently updated)
	var orphanedSessions []opencode.Session
	var skippedActive int
	now := time.Now()
	const recentActivityThreshold = 5 * time.Minute

	for _, session := range diskSessions {
		if !trackedSessionIDs[session.ID] {
			// First, quick check: was this session recently active? (using data we already have)
			updatedAt := time.Unix(session.Time.Updated/1000, 0)
			isRecentlyActive := now.Sub(updatedAt) <= recentActivityThreshold

			if isRecentlyActive {
				// Session is recently active - check if it's actually processing
				// This is the expensive check, but we only do it for recently active sessions
				if client.IsSessionProcessing(session.ID) {
					skippedActive++
					continue
				}
			}
			orphanedSessions = append(orphanedSessions, session)
		}
	}

	if skippedActive > 0 {
		fmt.Printf("  Skipped %d active sessions (currently processing)\n", skippedActive)
	}

	if len(orphanedSessions) == 0 {
		fmt.Println("  No orphaned disk sessions found")
		return 0, nil
	}

	fmt.Printf("  Found %d orphaned disk sessions:\n", len(orphanedSessions))

	// Delete orphaned sessions
	deleted := 0
	for _, session := range orphanedSessions {
		title := session.Title
		if title == "" {
			title = "(untitled)"
		}

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would delete: %s (%s)\n", session.ID[:12], title)
			deleted++
			continue
		}

		if err := client.DeleteSession(session.ID); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to delete %s: %v\n", session.ID[:12], err)
			continue
		}

		fmt.Printf("    Deleted: %s (%s)\n", session.ID[:12], title)
		deleted++
	}

	return deleted, nil
}

// cleanPhantomWindows finds and closes tmux windows that are phantoms
// (have a beads ID in the window name but no active OpenCode session).
// Returns the number of windows closed and any error encountered.
func cleanPhantomWindows(serverURL string, dryRun bool) (int, error) {
	client := opencode.NewClient(serverURL)
	now := time.Now()
	const maxIdleTime = 30 * time.Minute

	fmt.Println("\nScanning for phantom tmux windows...")

	// Get all OpenCode sessions and build a map of recently active beads IDs
	sessions, err := client.ListSessions("")
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	activeBeadsIDs := make(map[string]bool)
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				activeBeadsIDs[beadsID] = true
			}
		}
	}

	fmt.Printf("  Found %d active OpenCode sessions\n", len(activeBeadsIDs))

	// Scan all workers sessions for phantom windows
	var phantomWindows []struct {
		window      *tmux.WindowInfo
		sessionName string
		beadsID     string
	}

	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, err := tmux.ListWindows(sessionName)
		if err != nil {
			continue
		}

		for _, w := range windows {
			// Skip known non-agent windows
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" {
				continue
			}

			// If beads ID is not in active sessions, it's a phantom
			if !activeBeadsIDs[beadsID] {
				windowCopy := w
				phantomWindows = append(phantomWindows, struct {
					window      *tmux.WindowInfo
					sessionName string
					beadsID     string
				}{&windowCopy, sessionName, beadsID})
			}
		}
	}

	if len(phantomWindows) == 0 {
		fmt.Println("  No phantom windows found")
		return 0, nil
	}

	fmt.Printf("  Found %d phantom windows:\n", len(phantomWindows))

	// Close phantom windows
	closed := 0
	for _, pw := range phantomWindows {
		if dryRun {
			fmt.Printf("    [DRY-RUN] Would close: %s:%s\n", pw.sessionName, pw.window.Name)
			closed++
			continue
		}

		if err := tmux.KillWindow(pw.window.Target); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to close %s: %v\n", pw.window.Name, err)
			continue
		}

		fmt.Printf("    Closed: %s:%s\n", pw.sessionName, pw.window.Name)
		closed++
	}

	return closed, nil
}

// ============================================================================
// Account Management
// ============================================================================

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage Claude Max accounts",
	Long: `Manage multiple Claude Max accounts for usage tracking and rate limit arbitrage.

Accounts are stored in ~/.orch/accounts.yaml with refresh tokens for switching.

Examples:
  orch-go account list              # List all saved accounts
  orch-go account switch personal   # Switch to 'personal' account
  orch-go account remove work       # Remove 'work' account`,
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved accounts",
	Long:  "List all saved accounts with their email and default status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountList()
	},
}

var accountSwitchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Switch to a saved account",
	Long: `Switch to a saved account by refreshing its OAuth tokens.

This updates the OpenCode auth file with new tokens from the saved refresh token.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountSwitch(args[0])
	},
}

var accountRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a saved account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountRemove(args[0])
	},
}

var (
	// Account add command flags
	accountAddSetDefault bool
)

var accountAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new account via OAuth login",
	Long: `Add a new Claude Max account by initiating an OAuth login flow.

This command opens your browser for authentication with Anthropic. After
successful login, the refresh token is saved to ~/.orch/accounts.yaml.

The account can then be switched to using 'orch account switch <name>'.

Examples:
  orch-go account add personal           # Add account named 'personal'
  orch-go account add work --default     # Add as default account`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountAdd(args[0], accountAddSetDefault)
	},
}

func init() {
	accountAddCmd.Flags().BoolVar(&accountAddSetDefault, "default", false, "Set as default account")

	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountSwitchCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountAddCmd)
}

func runAccountList() error {
	accounts, err := account.ListAccountInfo()
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("No saved accounts")
		fmt.Println("\nTo add an account:")
		fmt.Println("  orch account add <name>")
		return nil
	}

	fmt.Printf("%-15s %-35s %-10s\n", "NAME", "EMAIL", "DEFAULT")
	fmt.Printf("%s\n", strings.Repeat("-", 65))

	for _, acc := range accounts {
		def := ""
		if acc.IsDefault {
			def = "✓"
		}
		fmt.Printf("%-15s %-35s %-10s\n", acc.Name, acc.Email, def)
	}

	return nil
}

func runAccountSwitch(name string) error {
	email, err := account.SwitchAccount(name)
	if err != nil {
		// Check if it's a token refresh error to provide actionable guidance
		if tokenErr, ok := err.(*account.TokenRefreshError); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n", tokenErr.Error())
			fmt.Fprintf(os.Stderr, "\n%s\n", tokenErr.ActionableGuidance())
			return fmt.Errorf("token refresh failed for account '%s'", name)
		}
		return err
	}

	fmt.Printf("Switched to account: %s (%s)\n", name, email)
	return nil
}

func runAccountRemove(name string) error {
	cfg, err := account.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}

	if err := cfg.Remove(name); err != nil {
		return fmt.Errorf("failed to remove account: %w", err)
	}

	if err := account.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed account: %s\n", name)
	return nil
}

func runAccountAdd(name string, setDefault bool) error {
	// Check if account already exists
	cfg, err := account.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}

	if _, err := cfg.Get(name); err == nil {
		return fmt.Errorf("account '%s' already exists. Use 'orch account remove %s' first, or choose a different name", name, name)
	}

	fmt.Printf("Adding account '%s'...\n", name)
	fmt.Println()

	email, err := account.AddAccount(name, setDefault, nil)
	if err != nil {
		return fmt.Errorf("failed to add account: %w", err)
	}

	fmt.Println()
	fmt.Printf("Successfully added account '%s' (%s)\n", name, email)
	if setDefault {
		fmt.Println("Set as default account")
	}
	fmt.Println("\nThe account is now active. Use 'orch account switch <name>' to change accounts later.")

	return nil
}

// ============================================================================
// Usage Tracking (Placeholder - defers to Python for now)
// ============================================================================

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show Claude Max usage limits",
	Long: `Show Claude Max weekly usage limits.

Reads OAuth token from OpenCode's auth.json and fetches usage data
from the Anthropic API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUsage()
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}

func runUsage() error {
	info := usage.FetchUsage()
	fmt.Println(usage.FormatDisplay(info))
	return nil
}

// ============================================================================
// Port Allocation Management
// ============================================================================

var portCmd = &cobra.Command{
	Use:   "port",
	Short: "Manage port allocations for projects",
	Long: `Manage port allocations to prevent conflicts across projects.

Ports are allocated from predefined ranges by purpose:
  - vite: 5173-5199 (dev servers)
  - api:  3333-3399 (API servers)

Allocations are stored in ~/.orch/ports.yaml.

Examples:
  orch-go port allocate myproject web vite    # Allocate a vite port
  orch-go port allocate myproject api api     # Allocate an API port
  orch-go port list                           # List all allocations
  orch-go port list -p myproject              # List allocations for a project
  orch-go port release myproject web          # Release a port allocation
  orch-go port release --port 5173            # Release by port number`,
}

var (
	portListProject string
	portReleasePort int
)

var portAllocateCmd = &cobra.Command{
	Use:   "allocate [project] [service] [purpose]",
	Short: "Allocate a port for a project service",
	Long: `Allocate a port for a project/service from a purpose range.

Purpose can be:
  - vite: Dev server ports (5173-5199)
  - api:  API server ports (3333-3399)

If the project/service already has an allocation for this purpose,
returns the existing port (idempotent).

Examples:
  orch-go port allocate snap web vite     # Allocate a vite port for snap/web
  orch-go port allocate snap api api      # Allocate an API port for snap/api`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPortAllocate(args[0], args[1], args[2])
	},
}

var portListCmd = &cobra.Command{
	Use:   "list",
	Short: "List port allocations",
	Long: `List all port allocations or filter by project.

Examples:
  orch-go port list                  # List all allocations
  orch-go port list -p myproject     # List allocations for a project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPortList(portListProject)
	},
}

var portReleaseCmd = &cobra.Command{
	Use:   "release [project] [service]",
	Short: "Release a port allocation",
	Long: `Release a port allocation by project/service or by port number.

Examples:
  orch-go port release myproject web   # Release by project/service
  orch-go port release --port 5173     # Release by port number`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --port flag is set, release by port
		if portReleasePort > 0 {
			return runPortReleaseByPort(portReleasePort)
		}
		// Otherwise require project and service args
		if len(args) < 2 {
			return fmt.Errorf("requires project and service arguments, or --port flag")
		}
		return runPortRelease(args[0], args[1])
	},
}

var portTmuxinatorCmd = &cobra.Command{
	Use:   "tmuxinator [project] [project-dir]",
	Short: "Generate tmuxinator config with allocated ports",
	Long: `Generate or update a tmuxinator config file for a project's workers session.

The config includes server panes with the correct port numbers from the port registry.
This enables 'tmuxinator start workers-{project}' to launch dev servers with consistent ports.

Examples:
  orch port tmuxinator snap /path/to/snap     # Generate workers-snap.yml with ports
  orch port allocate snap web vite            # First allocate ports...
  orch port tmuxinator snap /path/to/snap     # ...then generate config with them`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPortTmuxinator(args[0], args[1])
	},
}

func init() {
	portListCmd.Flags().StringVarP(&portListProject, "project", "p", "", "Filter by project")
	portReleaseCmd.Flags().IntVar(&portReleasePort, "port", 0, "Release by port number")

	portCmd.AddCommand(portAllocateCmd)
	portCmd.AddCommand(portListCmd)
	portCmd.AddCommand(portReleaseCmd)
	portCmd.AddCommand(portTmuxinatorCmd)
}

func runPortAllocate(project, service, purpose string) error {
	reg, err := port.New("")
	if err != nil {
		return fmt.Errorf("failed to open port registry: %w", err)
	}

	portNum, err := reg.Allocate(project, service, purpose)
	if err != nil {
		if err == port.ErrRangeExhausted {
			return fmt.Errorf("no ports available in %s range", purpose)
		}
		if err == port.ErrInvalidPurpose {
			return fmt.Errorf("invalid purpose '%s' (use: vite, api)", purpose)
		}
		return fmt.Errorf("failed to allocate port: %w", err)
	}

	fmt.Printf("Allocated port %d for %s/%s (%s)\n", portNum, project, service, purpose)
	return nil
}

func runPortList(project string) error {
	reg, err := port.New("")
	if err != nil {
		return fmt.Errorf("failed to open port registry: %w", err)
	}

	var allocs []port.Allocation
	if project != "" {
		allocs = reg.ListByProject(project)
	} else {
		allocs = reg.List()
	}

	if len(allocs) == 0 {
		if project != "" {
			fmt.Printf("No port allocations for project: %s\n", project)
		} else {
			fmt.Println("No port allocations")
		}
		return nil
	}

	// Print header
	fmt.Printf("%-20s %-15s %-8s %-10s %s\n", "PROJECT", "SERVICE", "PORT", "PURPOSE", "ALLOCATED")
	fmt.Printf("%s\n", strings.Repeat("-", 75))

	for _, a := range allocs {
		// Parse and format timestamp
		allocatedAt := a.AllocatedAt
		if t, err := time.Parse(time.RFC3339, a.AllocatedAt); err == nil {
			allocatedAt = t.Format("2006-01-02 15:04")
		}
		fmt.Printf("%-20s %-15s %-8d %-10s %s\n", a.Project, a.Service, a.Port, a.Purpose, allocatedAt)
	}

	return nil
}

func runPortRelease(project, service string) error {
	reg, err := port.New("")
	if err != nil {
		return fmt.Errorf("failed to open port registry: %w", err)
	}

	// First find the allocation to show what's being released
	alloc := reg.Find(project, service)
	if alloc == nil {
		return fmt.Errorf("no allocation found for %s/%s", project, service)
	}

	portNum := alloc.Port
	if !reg.Release(project, service) {
		return fmt.Errorf("failed to release allocation")
	}

	fmt.Printf("Released port %d (%s/%s)\n", portNum, project, service)
	return nil
}

func runPortReleaseByPort(portNum int) error {
	reg, err := port.New("")
	if err != nil {
		return fmt.Errorf("failed to open port registry: %w", err)
	}

	// First find the allocation to show what's being released
	alloc := reg.FindByPort(portNum)
	if alloc == nil {
		return fmt.Errorf("no allocation found for port %d", portNum)
	}

	project := alloc.Project
	service := alloc.Service
	if !reg.ReleaseByPort(portNum) {
		return fmt.Errorf("failed to release allocation")
	}

	fmt.Printf("Released port %d (%s/%s)\n", portNum, project, service)
	return nil
}

func runPortTmuxinator(project, projectDir string) error {
	configPath, err := tmux.UpdateTmuxinatorConfig(project, projectDir)
	if err != nil {
		return fmt.Errorf("failed to generate tmuxinator config: %w", err)
	}

	// Get port allocations for display
	reg, err := port.New("")
	if err != nil {
		return fmt.Errorf("failed to open port registry: %w", err)
	}
	allocs := reg.ListByProject(project)

	fmt.Printf("Generated tmuxinator config: %s\n", configPath)
	if len(allocs) > 0 {
		fmt.Printf("\nPort allocations included:\n")
		for _, a := range allocs {
			fmt.Printf("  - %s/%s: port %d (%s)\n", a.Project, a.Service, a.Port, a.Purpose)
		}
	} else {
		fmt.Printf("\nNo port allocations found for project '%s'.\n", project)
		fmt.Printf("Use 'orch port allocate %s <service> <purpose>' to allocate ports.\n", project)
	}

	return nil
}

// ensureOrchScaffolding checks for .beads/ directory when beads tracking is enabled.
// If autoInit is true, it automatically initializes missing directories.
// If autoInit is false and .beads/ is missing (with tracking enabled), it returns an error with helpful suggestions.
// Note: .orch/ directories are created automatically by spawn.WriteContext(), so we don't check for them here.
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
		result, err := initProject(projectDir, false, false, false, false, true, true, "", "")
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

	// Detect source project from current working directory
	sourceProject := detectSourceProject()

	// Record the gap with project context
	tracker.RecordGapWithProject(gapAnalysis, skill, task, sourceProject)

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

// detectSourceProject returns the project directory name from the current working directory.
// Returns empty string if detection fails.
func detectSourceProject() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Base(cwd)
}

var retriesCmd = &cobra.Command{
	Use:   "retries",
	Short: "Show issues with retry patterns (failed attempts)",
	Long: `Show beads issues that have been retried after failures.

This helps surface flaky issues that may need reliability-testing instead
of repeated debugging attempts. A retry pattern is detected when:
- An issue has been spawned multiple times
- At least one attempt was abandoned (explicit failure)

Issues are sorted by severity:
1. Persistent failures (multiple attempts, no success) - shown first
2. Retry patterns (some attempts, some abandons)

Examples:
  orch retries                 # Show all issues with retry patterns
  orch retries orch-go-xxxx    # Show retry stats for a specific issue`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return runRetriesForIssue(args[0])
		}
		return runRetriesAll()
	},
}

func runRetriesForIssue(beadsID string) error {
	stats, err := verify.GetFixAttemptStats(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get retry stats: %w", err)
	}

	if stats.SpawnCount == 0 {
		fmt.Printf("No spawn history found for %s\n", beadsID)
		return nil
	}

	fmt.Printf("RETRY STATS: %s\n", beadsID)
	fmt.Printf("  Spawns:     %d\n", stats.SpawnCount)
	fmt.Printf("  Abandoned:  %d\n", stats.AbandonedCount)
	fmt.Printf("  Completed:  %d\n", stats.CompletedCount)
	if len(stats.Skills) > 0 {
		fmt.Printf("  Skills:     %s\n", strings.Join(stats.Skills, ", "))
	}
	if !stats.LastAttemptAt.IsZero() {
		fmt.Printf("  Last attempt: %s ago\n", formatDuration(time.Since(stats.LastAttemptAt)))
	}

	if stats.IsPersistentFailure() {
		fmt.Println()
		fmt.Println("🚨 PERSISTENT FAILURE PATTERN")
		fmt.Println("   This issue has failed multiple times without success.")
		fmt.Println("   Consider: orch spawn reliability-testing \"<task>\"")
	} else if stats.IsRetryPattern() {
		fmt.Println()
		fmt.Println("⚠️  RETRY PATTERN DETECTED")
		fmt.Println("   This issue has been respawned after previous failure(s).")
		fmt.Println("   Consider investigating root cause before more attempts.")
	}

	return nil
}

func runRetriesAll() error {
	patterns, err := verify.GetAllRetryPatterns()
	if err != nil {
		return fmt.Errorf("failed to get retry patterns: %w", err)
	}

	if len(patterns) == 0 {
		fmt.Println("No retry patterns detected")
		return nil
	}

	fmt.Printf("RETRY PATTERNS: %d issues with retry history\n\n", len(patterns))

	for _, stats := range patterns {
		// Status indicator
		indicator := "⚠️"
		if stats.IsPersistentFailure() {
			indicator = "🚨"
		}

		fmt.Printf("%s %s\n", indicator, stats.BeadsID)
		fmt.Printf("   Spawns: %d | Abandoned: %d | Completed: %d\n",
			stats.SpawnCount, stats.AbandonedCount, stats.CompletedCount)
		if len(stats.Skills) > 0 {
			fmt.Printf("   Skills: %s\n", strings.Join(stats.Skills, ", "))
		}
		if action := stats.SuggestedAction(); action != "" {
			fmt.Printf("   Suggested: %s\n", action)
		}
		fmt.Println()
	}

	return nil
}
