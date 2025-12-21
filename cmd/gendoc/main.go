// Package main generates CLI documentation using Cobra's doc generator.
//
// This tool mirrors the command structure from cmd/orch/main.go and generates
// markdown documentation for all commands.
//
// Usage:
//
//	go run ./cmd/gendoc
//
// This generates docs in docs/cli/ directory.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	// Default output directory
	outputDir := "docs/cli"

	// Allow override via argument
	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Build command tree
	rootCmd := buildCommandTree()

	// Custom file prepender to add frontmatter
	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		name = strings.TrimSuffix(name, filepath.Ext(name))
		name = strings.ReplaceAll(name, "_", " ")
		return fmt.Sprintf(`---
title: "%s"
generated: "%s"
---

`, name, time.Now().Format("2006-01-02"))
	}

	// Custom link handler for cross-references
	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, filepath.Ext(name))
		return base + ".md"
	}

	// Generate markdown docs
	if err := doc.GenMarkdownTreeCustom(rootCmd, outputDir, filePrepender, linkHandler); err != nil {
		log.Fatalf("Failed to generate docs: %v", err)
	}

	fmt.Printf("Documentation generated in %s/\n", outputDir)

	// List generated files
	files, _ := filepath.Glob(filepath.Join(outputDir, "*.md"))
	for _, f := range files {
		fmt.Printf("  - %s\n", filepath.Base(f))
	}
}

// noopRun is a placeholder Run function to make commands "runnable" for doc generation.
// Cobra's doc generator skips non-runnable commands.
var noopRun = func(cmd *cobra.Command, args []string) {}

// buildCommandTree constructs the Cobra command tree for documentation.
// This mirrors the structure in cmd/orch/main.go but without the RunE implementations.
func buildCommandTree() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "orch-go",
		Short: "OpenCode orchestration CLI",
		Long: `orch-go is a CLI tool for orchestrating OpenCode sessions.

It provides commands for spawning new sessions, sending messages to existing
sessions, and monitoring session events via SSE.`,
		Run: noopRun,
	}

	// Global flags
	rootCmd.PersistentFlags().String("server", "http://127.0.0.1:4096", "OpenCode server URL")

	// Add all commands
	rootCmd.AddCommand(buildSpawnCmd())
	rootCmd.AddCommand(buildAskCmd())
	rootCmd.AddCommand(buildSendCmd())
	rootCmd.AddCommand(buildMonitorCmd())
	rootCmd.AddCommand(buildStatusCmd())
	rootCmd.AddCommand(buildCompleteCmd())
	rootCmd.AddCommand(buildWorkCmd())
	rootCmd.AddCommand(buildDaemonCmd())
	rootCmd.AddCommand(buildTailCmd())
	rootCmd.AddCommand(buildQuestionCmd())
	rootCmd.AddCommand(buildAbandonCmd())
	rootCmd.AddCommand(buildCleanCmd())
	rootCmd.AddCommand(buildAccountCmd())
	rootCmd.AddCommand(buildWaitCmd())
	rootCmd.AddCommand(buildFocusCmd())
	rootCmd.AddCommand(buildDriftCmd())
	rootCmd.AddCommand(buildNextCmd())
	rootCmd.AddCommand(buildReviewCmd())
	rootCmd.AddCommand(buildUsageCmd())
	rootCmd.AddCommand(buildServeCmd())
	rootCmd.AddCommand(buildResumeCmd())

	return rootCmd
}

func buildSpawnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spawn [skill] [task]",
		Short: "Spawn a new OpenCode session with skill context",
		Long: `Spawn a new OpenCode session with skill context.

By default, spawns the agent headlessly via HTTP API (no TUI) and returns immediately.
Use --inline to run in the current terminal (blocking with TUI).

Model aliases: opus, sonnet, haiku (Anthropic), flash, pro (Google)
Full format: provider/model (e.g., anthropic/claude-opus-4-5-20251101)

Examples:
  orch-go spawn investigation "explore the codebase"           # Default: headless
  orch-go spawn feature-impl "add new spawn command" --phases implementation,validation
  orch-go spawn --issue proj-123 feature-impl "implement the feature"
  orch-go spawn --inline investigation "explore codebase"      # Run inline (blocking TUI)
  orch-go spawn --model opus investigation "explore the codebase"
  orch-go spawn --model flash investigation "explore the codebase"
  orch-go spawn --no-track investigation "exploratory work"
  orch-go spawn --mcp playwright feature-impl "add UI feature"
  orch-go spawn --skip-artifact-check investigation "fresh start"`,
		Run: noopRun,
	}

	cmd.Flags().String("issue", "", "Beads issue ID for tracking")
	cmd.Flags().String("phases", "", "Feature-impl phases (e.g., implementation,validation)")
	cmd.Flags().String("mode", "tdd", "Implementation mode: tdd or direct")
	cmd.Flags().String("validation", "tests", "Validation level: none, tests, smoke-test")
	cmd.Flags().Bool("inline", false, "Run inline (blocking) with TUI")
	cmd.Flags().String("model", "", "Model alias (opus, sonnet, haiku, flash, pro) or provider/model format")
	cmd.Flags().Bool("no-track", false, "Opt-out of beads issue tracking (ad-hoc work)")
	cmd.Flags().String("mcp", "", "MCP server config (e.g., 'playwright' for browser automation)")
	cmd.Flags().Bool("skip-artifact-check", false, "Bypass pre-spawn kb context check")

	return cmd
}

func buildAskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask [session-id] [prompt]",
		Short: "Send a message to an existing session (alias for send)",
		Long:  "Send a message to an existing OpenCode session. This is an alias for the 'send' command.",
		Run:   noopRun,
	}
}

func buildSendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send [session-id] [message]",
		Short: "Send a message to an existing session",
		Long: `Send a message to an existing OpenCode session.

The session can be running or completed. Response text is streamed to stdout
as it's received from the agent.

Examples:
  orch-go send ses_abc123 "what files did you modify?"
  orch-go send ses_xyz789 "can you explain the changes?"`,
		Run: noopRun,
	}
}

func buildMonitorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "monitor",
		Short: "Monitor SSE events for session completion",
		Long:  "Monitor the OpenCode server for session events and send notifications on completion.",
		Run:   noopRun,
	}
}

func buildStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List active OpenCode sessions",
		Long: `List all active OpenCode sessions with their status.

Shows session ID, workspace/title, directory, and last update time.`,
		Run: noopRun,
	}
}

func buildCompleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete [beads-id]",
		Short: "Complete an agent and close the beads issue",
		Long: `Complete an agent's work by verifying Phase: Complete and closing the beads issue.

Checks that the agent has reported "Phase: Complete" via beads comments before
closing the issue. Use --force to skip phase verification.

Examples:
  orch-go complete proj-123
  orch-go complete proj-123 --reason "All tests passing"
  orch-go complete proj-123 --force`,
		Run: noopRun,
	}

	cmd.Flags().BoolP("force", "f", false, "Skip phase verification")
	cmd.Flags().StringP("reason", "r", "", "Reason for closing (default: uses phase summary)")

	return cmd
}

func buildWorkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work [beads-id]",
		Short: "Start work on a beads issue with skill inference",
		Long: `Start work on a beads issue by inferring the skill from the issue type.

The skill is automatically determined from the issue type:
  - bug         → systematic-debugging
  - feature     → feature-impl
  - task        → feature-impl
  - investigation → investigation

The issue description becomes the task prompt for the spawned agent.

Examples:
  orch-go work proj-123           # Start work headlessly (default)
  orch-go work proj-123 --inline  # Start work inline (blocking TUI)`,
		Run: noopRun,
	}

	cmd.Flags().Bool("inline", false, "Run inline (blocking) with TUI")

	return cmd
}

func buildDaemonCmd() *cobra.Command {
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the orch daemon for autonomous processing",
		Long: `Manage the orch daemon for autonomous processing of beads issues.

The daemon monitors beads for issues labeled 'triage:ready' and spawns
agents to work on them automatically.`,
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the daemon in the foreground",
		Long: `Run the autonomous processing daemon in the foreground.

The daemon polls beads for issues labeled 'triage:ready' and spawns
agents to work on them. It tracks active agents and respects concurrency limits.`,
		Run: noopRun,
	}
	runCmd.Flags().Duration("interval", 30*time.Second, "Poll interval for checking ready issues")
	runCmd.Flags().Int("max-workers", 3, "Maximum concurrent worker agents")
	runCmd.Flags().Bool("dry-run", false, "Show what would be spawned without spawning")

	previewCmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview what would be spawned without spawning",
		Long:  "Show issues that would be spawned by the daemon without actually spawning agents.",
		Run:   noopRun,
	}

	daemonCmd.AddCommand(runCmd)
	daemonCmd.AddCommand(previewCmd)

	return daemonCmd
}

func buildTailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tail [beads-id]",
		Short: "Capture recent output from an agent",
		Long: `Capture recent output from an agent for debugging.

Fetches messages from the OpenCode API for the agent's session.

Examples:
  orch-go tail proj-123              # Capture last 50 lines (default)
  orch-go tail proj-123 --lines 100  # Capture last 100 lines
  orch-go tail proj-123 -n 20        # Capture last 20 lines`,
		Run: noopRun,
	}

	cmd.Flags().IntP("lines", "n", 50, "Number of lines to capture")

	return cmd
}

func buildQuestionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "question [beads-id]",
		Short: "Extract pending question from an agent's session",
		Long: `Extract pending question from an agent's session.

Finds the OpenCode session associated with the beads issue ID and extracts
any pending question the agent is asking. Useful for monitoring agents
that are blocked waiting for user input.

Examples:
  orch-go question proj-123  # Extract question from agent's session`,
		Run: noopRun,
	}
}

func buildAbandonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abandon [beads-id]",
		Short: "Abandon a stuck or frozen agent",
		Long: `Abandon an agent and mark it abandoned in the registry.

Use this command for stuck or frozen agents that are not responding.
The agent's beads issue is NOT closed - you can restart work with 'orch work'.

Examples:
  orch-go abandon proj-123  # Abandon agent for issue proj-123`,
		Run: noopRun,
	}
}

func buildCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove completed agents from the registry",
		Long: `Remove completed and abandoned agents from the registry.

By default, only cleans agents that are marked as completed or abandoned in the registry.

Examples:
  orch-go clean              # Clean completed/abandoned agents
  orch-go clean --dry-run    # Show what would be cleaned`,
		Run: noopRun,
	}

	cmd.Flags().Bool("dry-run", false, "Show what would be cleaned without making changes")

	return cmd
}

func buildAccountCmd() *cobra.Command {
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "Manage Claude Max accounts",
		Long: `Manage multiple Claude Max accounts for usage tracking and rate limit arbitrage.

Accounts are stored in ~/.orch/accounts.yaml with refresh tokens for switching.

Examples:
  orch-go account list              # List all saved accounts
  orch-go account switch personal   # Switch to 'personal' account
  orch-go account remove work       # Remove 'work' account`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List saved accounts",
		Long:  "List all saved accounts with their email and default status.",
		Run:   noopRun,
	}

	switchCmd := &cobra.Command{
		Use:   "switch [name]",
		Short: "Switch to a saved account",
		Long: `Switch to a saved account by refreshing its OAuth tokens.

This updates the OpenCode auth file with new tokens from the saved refresh token.`,
		Run: noopRun,
	}

	removeCmd := &cobra.Command{
		Use:   "remove [name]",
		Short: "Remove a saved account",
		Run:   noopRun,
	}

	accountCmd.AddCommand(listCmd)
	accountCmd.AddCommand(switchCmd)
	accountCmd.AddCommand(removeCmd)

	return accountCmd
}

func buildWaitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait [beads-id]",
		Short: "Wait for an agent to reach a specific phase",
		Long: `Wait for an agent to reach a specific phase by monitoring beads comments.

Polls beads comments for phase updates and blocks until the target phase is reached
or the timeout expires.

Examples:
  orch-go wait proj-123                    # Wait for Phase: Complete (default)
  orch-go wait proj-123 --phase Implementing  # Wait for Phase: Implementing
  orch-go wait proj-123 --timeout 30m      # Wait up to 30 minutes`,
		Run: noopRun,
	}

	cmd.Flags().String("phase", "Complete", "Phase to wait for")
	cmd.Flags().Duration("timeout", 2*time.Hour, "Maximum time to wait")
	cmd.Flags().Duration("interval", 30*time.Second, "Poll interval")

	return cmd
}

func buildFocusCmd() *cobra.Command {
	focusCmd := &cobra.Command{
		Use:   "focus",
		Short: "Manage focus (north star) for cross-project prioritization",
		Long: `Manage the current focus/north star for guiding work prioritization.

The focus is stored in ~/.orch/focus.json and used by the daemon and
orchestrator to prioritize which issues to work on next.`,
	}

	setCmd := &cobra.Command{
		Use:   "set [focus]",
		Short: "Set the current focus",
		Long:  "Set the current focus/north star for work prioritization.",
		Run:   noopRun,
	}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get the current focus",
		Long:  "Display the current focus/north star.",
		Run:   noopRun,
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear the current focus",
		Long:  "Remove the current focus/north star.",
		Run:   noopRun,
	}

	focusCmd.AddCommand(setCmd)
	focusCmd.AddCommand(getCmd)
	focusCmd.AddCommand(clearCmd)

	return focusCmd
}

func buildDriftCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "drift",
		Short: "Check for focus drift across active agents",
		Long: `Analyze active agents and check if their work aligns with the current focus.

Reports agents that may have drifted from the north star or are working on
unrelated tasks.`,
		Run: noopRun,
	}
}

func buildNextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Suggest the next issue to work on based on focus",
		Long: `Analyze ready issues and suggest the next one to work on based on:
- Current focus/north star
- Issue priority and age
- Active agent workload

Helps the orchestrator decide what to spawn next.`,
		Run: noopRun,
	}
}

func buildReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Review agents ready for completion",
		Long: `Review agents that have reported Phase: Complete and are ready for verification.

Lists agents with Phase: Complete status and allows batch completion.

Examples:
  orch-go review              # List agents ready for review
  orch-go review --complete   # Complete all ready agents`,
		Run: noopRun,
	}

	cmd.Flags().Bool("complete", false, "Complete all ready agents")

	return cmd
}

func buildUsageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usage",
		Short: "Show Claude Max usage limits",
		Long: `Show Claude Max weekly usage limits.

Reads OAuth token from OpenCode's auth.json and fetches usage data
from the Anthropic API.`,
		Run: noopRun,
	}
}

func buildServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web dashboard server",
		Long: `Start a web dashboard server for monitoring agents and sessions.

The dashboard provides real-time status of active agents, session history,
and orchestration controls.

Examples:
  orch-go serve              # Start on default port 8080
  orch-go serve --port 3000  # Start on port 3000`,
		Run: noopRun,
	}

	cmd.Flags().Int("port", 8080, "Port to listen on")

	return cmd
}

func buildResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume [beads-id]",
		Short: "Resume a paused or stuck agent",
		Long: `Resume a paused agent by sending a continuation message.

Useful for agents that are waiting for input or have stalled. Sends a message
to the agent's session to continue work.

Examples:
  orch-go resume proj-123                   # Resume with default message
  orch-go resume proj-123 --message "continue"  # Resume with custom message`,
		Run: noopRun,
	}

	cmd.Flags().String("message", "Continue with your task", "Message to send to resume the agent")

	return cmd
}
