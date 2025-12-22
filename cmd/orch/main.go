// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
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
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://127.0.0.1:4096", "OpenCode server URL")

	rootCmd.AddCommand(spawnCmd)
	rootCmd.AddCommand(askCmd)
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
	spawnAttach            bool   // Attach to tmux window after spawning
	spawnModel             string // Model to use for standalone spawns
	spawnNoTrack           bool   // Opt-out of beads tracking
	spawnMCP               string // MCP server config (e.g., "playwright")
	spawnSkipArtifactCheck bool   // Bypass pre-spawn kb context check
	spawnMaxAgents         int    // Maximum concurrent agents (0 = use default or env var)
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [skill] [task]",
	Short: "Spawn a new OpenCode session with skill context",
	Long: `Spawn a new OpenCode session with skill context.

By default, spawns the agent in a tmux window (visible, interruptible).
Use --inline to run in the current terminal (blocking with TUI).
Use --headless for automation/scripting (no TUI, fire-and-forget).
Use --attach to spawn in tmux and attach immediately.

Concurrency Limiting:
  By default, limits concurrent agents to 5. This prevents runaway agent spawning.
  Configure via --max-agents flag or ORCH_MAX_AGENTS environment variable.
  Set to 0 to disable the limit (not recommended).

Model aliases: opus, sonnet, haiku (Anthropic), flash, pro (Google)
Full format: provider/model (e.g., anthropic/claude-opus-4-5-20251101)

Examples:
  orch-go spawn investigation "explore the codebase"           # Default: tmux window
  orch-go spawn feature-impl "add new spawn command" --phases implementation,validation
  orch-go spawn --issue proj-123 feature-impl "implement the feature"
  orch-go spawn --inline investigation "explore codebase"      # Run inline (blocking TUI)
  orch-go spawn --headless investigation "explore codebase"    # Fire-and-forget (automation)
  orch-go spawn --attach investigation "explore codebase"      # Tmux + attach immediately
  orch-go spawn --model opus investigation "explore the codebase"  # Use Claude Opus
  orch-go spawn --model flash investigation "explore the codebase"  # Use Gemini Flash
  orch-go spawn --no-track investigation "exploratory work"    # Skip beads tracking
  orch-go spawn --mcp playwright feature-impl "add UI feature" # With Playwright MCP
  orch-go spawn --skip-artifact-check investigation "fresh start"  # Skip kb context check
  orch-go spawn --max-agents 10 investigation "task"           # Allow up to 10 concurrent agents`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		task := strings.Join(args[1:], " ")

		return runSpawnWithSkill(serverURL, skillName, task, spawnInline, spawnHeadless, spawnAttach)
	},
}

func init() {
	spawnCmd.Flags().StringVar(&spawnIssue, "issue", "", "Beads issue ID for tracking")
	spawnCmd.Flags().StringVar(&spawnPhases, "phases", "", "Feature-impl phases (e.g., implementation,validation)")
	spawnCmd.Flags().StringVar(&spawnMode, "mode", "tdd", "Implementation mode: tdd or direct")
	spawnCmd.Flags().StringVar(&spawnValidation, "validation", "tests", "Validation level: none, tests, smoke-test")
	spawnCmd.Flags().BoolVar(&spawnInline, "inline", false, "Run inline (blocking) with TUI")
	spawnCmd.Flags().BoolVar(&spawnHeadless, "headless", false, "Run headless via HTTP API (for automation/scripting)")
	spawnCmd.Flags().BoolVar(&spawnAttach, "attach", false, "Attach to tmux window after spawning (implies --tmux)")
	spawnCmd.Flags().StringVar(&spawnModel, "model", "", "Model alias (opus, sonnet, haiku, flash, pro) or provider/model format")
	spawnCmd.Flags().BoolVar(&spawnNoTrack, "no-track", false, "Opt-out of beads issue tracking (ad-hoc work)")
	spawnCmd.Flags().StringVar(&spawnMCP, "mcp", "", "MCP server config (e.g., 'playwright' for browser automation)")
	spawnCmd.Flags().BoolVar(&spawnSkipArtifactCheck, "skip-artifact-check", false, "Bypass pre-spawn kb context check")
	spawnCmd.Flags().IntVar(&spawnMaxAgents, "max-agents", 0, "Maximum concurrent agents (default 5, 0 to disable limit, or use ORCH_MAX_AGENTS env var)")
}

var askCmd = &cobra.Command{
	Use:   "ask [identifier] [prompt]",
	Short: "Send a message to an existing session (alias for send)",
	Long:  "Send a message to an existing OpenCode session. This is an alias for the 'send' command. Supports session IDs, beads IDs, or workspace names.",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]
		prompt := strings.Join(args[1:], " ")
		return runSend(serverURL, identifier, prompt)
	},
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
	statusJSON    bool
	statusAll     bool   // Include phantom agents (default: hide)
	statusProject string // Filter by project
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show swarm status and active agents",
	Long: `Show swarm status including active/queued/completed agent counts,
per-account usage percentages, and individual agent details.

By default, phantom agents (beads issue open but no running agent) are hidden.
Use --all to include them.

Examples:
  orch-go status              # Show active agents only
  orch-go status --all        # Include phantom agents
  orch-go status --project snap  # Filter by project
  orch-go status --json       # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(serverURL)
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON for scripting")
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "Include phantom agents")
	statusCmd.Flags().StringVar(&statusProject, "project", "", "Filter by project")
}

var (
	// Complete command flags
	completeForce  bool
	completeReason string
)

var completeCmd = &cobra.Command{
	Use:   "complete [beads-id]",
	Short: "Complete an agent and close the beads issue",
	Long: `Complete an agent's work by verifying Phase: Complete and closing the beads issue.

Checks that the agent has reported "Phase: Complete" via beads comments before
closing the issue. Use --force to skip phase verification.

Examples:
  orch-go complete proj-123
  orch-go complete proj-123 --reason "All tests passing"
  orch-go complete proj-123 --force  # Skip phase verification`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runComplete(beadsID)
	},
}

func init() {
	completeCmd.Flags().BoolVarP(&completeForce, "force", "f", false, "Skip phase verification")
	completeCmd.Flags().StringVarP(&completeReason, "reason", "r", "", "Reason for closing (default: uses phase summary)")
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
	abandonReason string
)

var abandonCmd = &cobra.Command{
	Use:   "abandon [beads-id]",
	Short: "Abandon a stuck or frozen agent",
	Long: `Abandon an agent and kill its tmux window.

Use this command for stuck or frozen agents that are not responding.
The agent's beads issue is NOT closed - you can restart work with 'orch work'.

When --reason is provided, a FAILURE_REPORT.md is generated in the agent's workspace
documenting what went wrong and recommendations for retry.

Examples:
  orch-go abandon proj-123                                    # Abandon agent
  orch-go abandon proj-123 --reason "Out of context"          # Abandon with failure report
  orch-go abandon proj-123 --reason "Stuck in loop"           # Document the failure`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runAbandon(beadsID, abandonReason)
	},
}

func init() {
	abandonCmd.Flags().StringVar(&abandonReason, "reason", "", "Reason for abandonment (generates FAILURE_REPORT.md)")
}

func runAbandon(beadsID, reason string) error {
	// Strategy: Check liveness directly via tmux and OpenCode, not registry
	// An agent is "alive" if it has a tmux window OR an active OpenCode session

	// First, verify the beads issue exists
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	if issue.Status == "closed" {
		return fmt.Errorf("issue %s is already closed - nothing to abandon", beadsID)
	}

	// Get current directory for OpenCode client
	projectDir, _ := os.Getwd()
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

	// Use issue title as the task (description is often longer form context)
	task := issue.Title

	// Set the spawnIssue flag so runSpawnWithSkill uses the existing issue
	spawnIssue = beadsID

	fmt.Printf("Starting work on: %s\n", beadsID)
	fmt.Printf("  Title:  %s\n", issue.Title)
	fmt.Printf("  Type:   %s\n", issue.IssueType)
	fmt.Printf("  Skill:  %s\n", skillName)

	return runSpawnWithSkill(serverURL, skillName, task, inline, false, false) // headless=false means tmux (default)
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
	cmd := exec.Command("sh", "-c", "opencode serve --port 4096 </dev/null >/dev/null 2>&1 &")
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

	// Filter to only count active sessions (updated within last 30 minutes)
	// This prevents stale sessions from blocking new spawns after restart
	now := time.Now()
	staleThreshold := 30 * time.Minute
	activeCount := 0
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		idleTime := now.Sub(updatedAt)
		if idleTime < staleThreshold {
			activeCount++
		}
	}

	if activeCount >= maxAgents {
		return fmt.Errorf("concurrency limit reached: %d active agents (max %d). Use 'orch status' to see active agents, 'orch complete' to finish agents, or --max-agents to increase limit", activeCount, maxAgents)
	}

	return nil
}

func runSpawnWithSkill(serverURL, skillName, task string, inline bool, headless bool, attach bool) error {
	// Check concurrency limit before spawning
	if err := checkConcurrencyLimit(); err != nil {
		return err
	}

	// Get current directory as project dir
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get project name from directory
	projectName := filepath.Base(projectDir)

	// Generate workspace name
	workspaceName := spawn.GenerateWorkspaceName(skillName, task)

	// Load skill content
	loader := skills.DefaultLoader()
	skillContent, err := loader.LoadSkillContent(skillName)
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

	// Update beads issue status to in_progress (only if tracking a real issue)
	if !spawnNoTrack && spawnIssue != "" {
		if err := verify.UpdateIssueStatus(beadsID, "in_progress"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
			// Continue anyway
		}
	}

	// Resolve model - convert aliases to full format
	resolvedModel := model.Resolve(spawnModel)

	// Run pre-spawn kb context check unless skipped
	var kbContext string
	if !spawnSkipArtifactCheck {
		kbContext = runPreSpawnKBCheck(task)
	} else {
		fmt.Println("Skipping kb context check (--skip-artifact-check)")
	}

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
		NoTrack:           spawnNoTrack,
		SkipArtifactCheck: spawnSkipArtifactCheck,
		KBContext:         kbContext,
	}

	// Write SPAWN_CONTEXT.md
	if err := spawn.WriteContext(cfg); err != nil {
		return fmt.Errorf("failed to write spawn context: %w", err)
	}

	// Generate minimal prompt
	minimalPrompt := spawn.MinimalPrompt(cfg)

	// Spawn mode: inline (blocking TUI), headless (HTTP API), or tmux (default)
	if inline {
		// Inline mode (blocking) - run in current terminal with TUI
		return runSpawnInline(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	if headless {
		// Headless mode - spawn via HTTP API (for automation/scripting)
		return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Default: Tmux mode - visible, interruptible, prevents runaway spawns
	return runSpawnTmux(serverURL, cfg, minimalPrompt, beadsID, skillName, task, attach)
}

// runSpawnInline spawns the agent inline (blocking) - original behavior.
func runSpawnInline(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	// Spawn opencode session
	client := opencode.NewClient(serverURL)
	cmd := client.BuildSpawnCommand(minimalPrompt, cfg.WorkspaceName, cfg.Model)
	cmd.Stderr = os.Stderr
	cmd.Dir = cfg.ProjectDir

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
	inlineEvent := events.Event{
		Type:      "session.spawned",
		SessionID: result.SessionID,
		Timestamp: time.Now().Unix(),
		Data:      inlineEventData,
	}
	if err := inlineLogger.Log(inlineEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary
	fmt.Printf("Spawned agent:\n")
	fmt.Printf("  Session ID: %s\n", result.SessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)

	return nil
}

// runSpawnHeadless spawns the agent via HTTP API without a TUI.
// This is useful for automation and daemon-driven spawns.
// The agent is registered with window_id='headless' for tracking.
func runSpawnHeadless(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	client := opencode.NewClient(serverURL)

	// Create session via HTTP API
	sessionResp, err := client.CreateSession(cfg.WorkspaceName, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Send the prompt to start the agent
	if err := client.SendPrompt(sessionResp.ID, minimalPrompt); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}

	// Write session ID to workspace file for later lookups
	if err := spawn.WriteSessionID(cfg.WorkspacePath(), sessionResp.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
	}

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"session_id":          sessionResp.ID,
		"spawn_mode":          "headless",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionResp.ID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary
	fmt.Printf("Spawned agent (headless):\n")
	fmt.Printf("  Session ID: %s\n", sessionResp.ID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	fmt.Printf("  Context:    %s\n", cfg.ContextFilePath())

	return nil
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

	// Print spawn summary
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
	fmt.Printf("  Context:    %s\n", cfg.ContextFilePath())

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
func createBeadsIssue(projectName, skillName, task string) (string, error) {
	// Build issue title
	title := fmt.Sprintf("[%s] %s: %s", projectName, skillName, truncate(task, 50))

	// Run bd create command
	cmd := exec.Command("bd", "create", title)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bd create failed: %w", err)
	}

	// Parse issue ID from output (search all lines for "issue: <id>")
	outputStr := strings.TrimSpace(string(output))
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for "issue:" in the line and extract the ID after it
		parts := strings.Fields(line)
		for i, part := range parts {
			if strings.Contains(part, "issue:") {
				// Issue ID should be the next word after "issue:"
				if i+1 < len(parts) {
					return parts[i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not parse issue ID from: %s", outputStr)
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
		// Send message asynchronously (non-blocking)
		if err := client.SendMessageAsync(sessionID, message); err != nil {
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
	Active    int `json:"active"`
	Phantom   int `json:"phantom,omitempty"` // Agents with open beads issue but not running
	Queued    int `json:"queued"`
	Completed int `json:"completed_today"`
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
	SessionID string `json:"session_id"`
	BeadsID   string `json:"beads_id,omitempty"`
	Skill     string `json:"skill,omitempty"`
	Account   string `json:"account,omitempty"`
	Runtime   string `json:"runtime"`
	Title     string `json:"title,omitempty"`
	Window    string `json:"window,omitempty"`
	Phase     string `json:"phase,omitempty"`      // Current phase from beads comments
	Task      string `json:"task,omitempty"`       // Task description (truncated)
	Project   string `json:"project,omitempty"`    // Project name derived from beads ID or workspace
	IsPhantom bool   `json:"is_phantom,omitempty"` // True if beads issue open but agent not running
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
	projectDir, _ := os.Getwd()

	agents := make([]AgentInfo, 0)
	seenBeadsIDs := make(map[string]bool)

	// Phase 1: Collect agents from tmux windows (primary source of truth for "active")
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, w := range windows {
			// Skip "servers" and "zsh" windows
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}

			// Derive beads ID from window name (format: "... [beads-id]")
			beadsID := extractBeadsIDFromWindowName(w.Name)
			skill := extractSkillFromWindowName(w.Name)
			project := extractProjectFromBeadsID(beadsID)

			// Use state.GetLiveness() for accurate liveness check
			var liveness state.LivenessResult
			if beadsID != "" {
				liveness = state.GetLiveness(beadsID, serverURL, projectDir)
				seenBeadsIDs[beadsID] = true
			}

			// Get phase from beads comments if we have a beads ID
			var phase, task string
			if beadsID != "" {
				phase, task = getPhaseAndTask(beadsID)
			}

			// Calculate runtime from OpenCode session if available
			runtime := "unknown"
			if liveness.SessionID != "" {
				if session, err := client.GetSession(liveness.SessionID); err == nil {
					createdAt := time.Unix(session.Time.Created/1000, 0)
					runtime = formatDuration(now.Sub(createdAt))
				}
			}

			agent := AgentInfo{
				SessionID: liveness.SessionID,
				BeadsID:   beadsID,
				Skill:     skill,
				Title:     w.Name,
				Runtime:   runtime,
				Window:    w.Target,
				Phase:     phase,
				Task:      task,
				Project:   project,
				IsPhantom: liveness.IsPhantom(),
			}

			// If we got a session ID from tmux, use "tmux" as identifier
			if agent.SessionID == "" {
				agent.SessionID = "tmux"
			}

			agents = append(agents, agent)
		}
	}

	// Phase 2: Collect agents from OpenCode sessions (headless mode agents)
	// NOTE: ListSessions("") returns ALL persisted sessions (339+), not just active ones.
	// We filter by activity time to only show sessions updated within the last 30 minutes.
	sessions, err := client.ListSessions("")
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	// Maximum idle time to consider a session "active"
	const maxIdleTime = 30 * time.Minute

	for _, s := range sessions {
		// Calculate runtime and idle time
		createdAt := time.Unix(s.Time.Created/1000, 0)
		runtime := now.Sub(createdAt)
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		idleTime := now.Sub(updatedAt)

		// Early filter: skip sessions idle longer than maxIdleTime
		// This prevents 339 stale sessions from appearing as active
		if idleTime > maxIdleTime {
			continue
		}

		beadsID := extractBeadsIDFromTitle(s.Title)
		skill := extractSkillFromTitle(s.Title)
		project := extractProjectFromBeadsID(beadsID)

		// Skip if already tracked via tmux
		if beadsID != "" && seenBeadsIDs[beadsID] {
			continue
		}

		// Use state.GetLiveness() to check if this is actually live
		var liveness state.LivenessResult
		if beadsID != "" {
			liveness = state.GetLiveness(beadsID, serverURL, projectDir)
			seenBeadsIDs[beadsID] = true
		} else {
			// No beads ID - session is active if we got here (passed idle time check above)
			liveness.OpencodeLive = true
		}

		// Get phase from beads comments if we have a beads ID
		var phase, task string
		if beadsID != "" {
			phase, task = getPhaseAndTask(beadsID)
		}

		// Skip if not alive (neither tmux nor OpenCode)
		if !liveness.IsAlive() && beadsID != "" {
			// Not alive but has beads ID - might be phantom
			if liveness.IsPhantom() {
				// Include phantom agents in the list
				agent := AgentInfo{
					SessionID: s.ID,
					Title:     s.Title,
					Runtime:   formatDuration(runtime),
					BeadsID:   beadsID,
					Skill:     skill,
					Phase:     phase,
					Task:      task,
					Project:   project,
					IsPhantom: true,
				}
				agents = append(agents, agent)
			}
			continue
		}

		agent := AgentInfo{
			SessionID: s.ID,
			Title:     s.Title,
			Runtime:   formatDuration(runtime),
			BeadsID:   beadsID,
			Skill:     skill,
			Phase:     phase,
			Task:      task,
			Project:   project,
			IsPhantom: liveness.IsPhantom(),
		}

		agents = append(agents, agent)
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
		filteredAgents = append(filteredAgents, agent)
	}

	// Phase 4: Build swarm status (counts before filtering)
	activeCount := 0
	phantomCount := 0
	for _, agent := range agents {
		if agent.IsPhantom {
			phantomCount++
		} else {
			activeCount++
		}
	}

	swarm := SwarmStatus{
		Active:    activeCount,
		Phantom:   phantomCount,
		Queued:    0, // TODO: implement queuing system
		Completed: 0, // No longer tracked via registry
	}

	// Fetch account usage information
	accounts := getAccountUsage()

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

// printSwarmStatus prints the swarm status in human-readable format.
func printSwarmStatus(output StatusOutput, showPhantoms bool) {
	// Print swarm summary header
	fmt.Printf("SWARM STATUS: Active: %d", output.Swarm.Active)
	if output.Swarm.Phantom > 0 {
		fmt.Printf(", Phantom: %d", output.Swarm.Phantom)
		if !showPhantoms {
			fmt.Printf(" (use --all to show)")
		}
	}
	fmt.Println()
	fmt.Println()

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

	// Print agents table
	if len(output.Agents) > 0 {
		fmt.Println("AGENTS")
		// New column layout: BEADS ID (full), PHASE, TASK, SKILL, RUNTIME
		fmt.Printf("  %-18s %-12s %-40s %-20s %s\n", "BEADS ID", "PHASE", "TASK", "SKILL", "RUNTIME")
		fmt.Printf("  %s\n", strings.Repeat("-", 100))

		for _, agent := range output.Agents {
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

			// Add phantom indicator to phase
			if agent.IsPhantom {
				phase = "⚠ " + phase
			}

			fmt.Printf("  %-18s %-12s %-40s %-20s %s\n",
				beadsID,
				truncate(phase, 10),
				truncate(task, 38),
				truncate(skill, 18),
				agent.Runtime)
		}
	} else {
		fmt.Println("No active agents")
	}
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

func runComplete(beadsID string) error {
	// Get issue to verify it exists
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Check if already closed
	isClosed := issue.Status == "closed"
	if isClosed {
		fmt.Printf("Issue %s is already closed in beads\n", beadsID)
	}

	// Get current directory as project dir
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Verify phase status unless force flag is set
	if !completeForce {
		// Derive workspace path from project dir + beads ID
		// Strategy: Search .orch/workspace/ for directories containing the beads ID
		workspacePath, agentName := findWorkspaceByBeadsID(projectDir, beadsID)

		if workspacePath != "" {
			fmt.Printf("Workspace: %s\n", agentName)
		}

		result, err := verify.VerifyCompletion(beadsID, workspacePath)
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
		liveness := state.GetLiveness(beadsID, serverURL, projectDir)
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
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up completed/abandoned agents",
	Long: `Clean up completed and abandoned agents.

This command uses derived lookups (Phase 2 pattern):
1. Scans .orch/workspace/ for workspaces with SYNTHESIS.md (completed)
2. Cross-references with beads issue status (closed = completed)
3. Checks tmux/OpenCode liveness for active windows
4. Registry is updated for backwards compatibility but not required

Examples:
  orch-go clean                   # Clean completed agents (reports only)
  orch-go clean --dry-run         # Show what would be cleaned (no changes)
  orch-go clean --windows         # Also close tmux windows for completed agents
  orch-go clean --verify-opencode # Also check OpenCode disk sessions`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClean(cleanDryRun, cleanVerifyOpenCode, cleanWindows)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanCmd.Flags().BoolVar(&cleanVerifyOpenCode, "verify-opencode", false, "Also verify OpenCode disk sessions (slower)")
	cleanCmd.Flags().BoolVar(&cleanWindows, "windows", false, "Close tmux windows for completed agents")
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

func runClean(dryRun bool, verifyOpenCode bool, closeWindows bool) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find cleanable workspaces using derived lookups
	fmt.Println("Scanning workspaces for completed agents...")
	beadsChecker := NewDefaultBeadsStatusChecker()
	cleanableWorkspaces := findCleanableWorkspaces(projectDir, beadsChecker)

	fmt.Printf("\nFound %d cleanable workspaces\n", len(cleanableWorkspaces))

	if len(cleanableWorkspaces) == 0 && !verifyOpenCode {
		fmt.Println("No agents to clean")
		return nil
	}

	// Track cleanup stats
	workspacesCleaned := 0
	windowsClosed := 0

	// Clean workspaces found via derived lookup
	if len(cleanableWorkspaces) > 0 {
		fmt.Printf("\nWorkspaces to clean:\n")
		for _, ws := range cleanableWorkspaces {
			if dryRun {
				fmt.Printf("  [DRY-RUN] Would clean: %s (%s)\n", ws.Name, ws.Reason)
				// Also check if window would be closed
				if closeWindows {
					if window, sessionName, _ := tmux.FindWindowByWorkspaceNameAllSessions(ws.Name); window != nil {
						fmt.Printf("  [DRY-RUN] Would close window: %s in session %s\n", window.Name, sessionName)
					}
				}
			} else {
				fmt.Printf("  Cleaning: %s (%s)\n", ws.Name, ws.Reason)
				// Note: We don't delete the workspace directory itself
				// Workspaces are kept for investigation reference

				// Close tmux window if --windows flag is set
				if closeWindows {
					if window, sessionName, _ := tmux.FindWindowByWorkspaceNameAllSessions(ws.Name); window != nil {
						if err := tmux.KillWindow(window.Target); err != nil {
							fmt.Fprintf(os.Stderr, "    Warning: failed to close window %s: %v\n", window.Name, err)
						} else {
							fmt.Printf("    Closed window: %s in session %s\n", window.Name, sessionName)
							windowsClosed++
						}
					}
				}
			}
			workspacesCleaned++
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

	if dryRun {
		fmt.Printf("\nDry run complete. Would clean %d items", workspacesCleaned)
		if closeWindows {
			// Count potential windows to close
			windowCount := 0
			for _, ws := range cleanableWorkspaces {
				if window, _, _ := tmux.FindWindowByWorkspaceNameAllSessions(ws.Name); window != nil {
					windowCount++
				}
			}
			if windowCount > 0 {
				fmt.Printf(", %d tmux windows", windowCount)
			}
		}
		if verifyOpenCode {
			fmt.Printf(", %d orphaned disk sessions", diskSessionsDeleted)
		}
		fmt.Println(".")
		return nil
	}

	// Log the cleanup
	projectName := filepath.Base(projectDir)
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agents.cleaned",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"workspaces_cleaned":    workspacesCleaned,
			"windows_closed":        windowsClosed,
			"disk_sessions_deleted": diskSessionsDeleted,
			"project":               projectName,
			"verify_opencode":       verifyOpenCode,
			"close_windows":         closeWindows,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("\nCleaned %d workspaces", workspacesCleaned)
	if closeWindows && windowsClosed > 0 {
		fmt.Printf(", closed %d tmux windows", windowsClosed)
	}
	if verifyOpenCode && diskSessionsDeleted > 0 {
		fmt.Printf(", deleted %d orphaned disk sessions", diskSessionsDeleted)
	}
	fmt.Println()
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
	var orphanedSessions []opencode.Session
	for _, session := range diskSessions {
		if !trackedSessionIDs[session.ID] {
			orphanedSessions = append(orphanedSessions, session)
		}
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

// runPreSpawnKBCheck runs kb context check before spawning an agent.
// Returns formatted context string to include in SPAWN_CONTEXT.md, or empty string if no matches.
func runPreSpawnKBCheck(task string) string {
	// Extract keywords from task description
	// Try with 3 keywords first (more specific), fall back to 1 keyword (more broad)
	keywords := spawn.ExtractKeywords(task, 3)
	if keywords == "" {
		return ""
	}

	fmt.Printf("Checking kb context for: %q\n", keywords)

	// Run kb context check
	result, err := spawn.RunKBContextCheck(keywords)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
		return ""
	}

	// If no matches with multiple keywords, try with just the first keyword
	if result == nil || !result.HasMatches {
		firstKeyword := spawn.ExtractKeywords(task, 1)
		if firstKeyword != "" && firstKeyword != keywords {
			fmt.Printf("Trying broader search for: %q\n", firstKeyword)
			result, err = spawn.RunKBContextCheck(firstKeyword)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
				return ""
			}
		}
	}

	if result == nil || !result.HasMatches {
		fmt.Println("No prior knowledge found.")
		return ""
	}

	// Display results and prompt for acknowledgment
	if !spawn.DisplayContextAndPrompt(result) {
		fmt.Println("Context declined - proceeding without prior knowledge.")
		return ""
	}

	// Format context for inclusion in SPAWN_CONTEXT.md
	fmt.Println("Including prior knowledge in spawn context.")
	return spawn.FormatContextForSpawn(result)
}
