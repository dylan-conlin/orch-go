// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/question"
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
	// Complete command flags
	completeForce            bool
	completeReason           string
	completeApprove          bool
	completeWorkdir          string
	completeNoChangelogCheck bool
	completeSkipReproCheck   bool
	completeSkipReproReason  string
)

var completeCmd = &cobra.Command{
	Use:   "complete [beads-id]",
	Short: "Complete an agent and close the beads issue",
	Long: `Complete an agent's work by verifying Phase: Complete and closing the beads issue.

Checks that the agent has reported "Phase: Complete" via beads comments before
closing the issue. Use --force to skip phase and liveness verification.

For agents that modified web/ files (UI tasks), --approve is required to explicitly
confirm human review of the visual changes. This prevents agents from self-certifying
UI correctness.

For cross-project completion (agents spawned with --workdir in another project),
the command auto-detects the project from the workspace's SPAWN_CONTEXT.md.
Use --workdir as explicit override when auto-detection fails.

For bug-type issues, prompts the orchestrator to verify that the original
reproduction no longer occurs. This repro verification runs even with --force.
Use --skip-repro-check with --skip-repro-reason to bypass.

Examples:
  orch-go complete proj-123
  orch-go complete proj-123 --reason "All tests passing"
  orch-go complete proj-123 --approve       # Approve UI changes after visual review
  orch-go complete proj-123 --force         # Skip phase/liveness verification (not repro)
  orch-go complete kb-cli-123 --workdir ~/projects/kb-cli  # Cross-project completion
  orch-go complete proj-123 --skip-repro-check --skip-repro-reason "Repro verified via automated test"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runComplete(beadsID, completeWorkdir)
	},
}

func init() {
	completeCmd.Flags().BoolVarP(&completeForce, "force", "f", false, "Skip phase and liveness verification (repro still runs)")
	completeCmd.Flags().StringVarP(&completeReason, "reason", "r", "", "Reason for closing (default: uses phase summary)")
	completeCmd.Flags().BoolVar(&completeApprove, "approve", false, "Approve visual changes for UI tasks (adds approval comment)")
	completeCmd.Flags().StringVar(&completeWorkdir, "workdir", "", "Target project directory (for cross-project completion)")
	completeCmd.Flags().BoolVar(&completeNoChangelogCheck, "no-changelog-check", false, "Skip changelog detection for notable changes")
	completeCmd.Flags().BoolVar(&completeSkipReproCheck, "skip-repro-check", false, "Skip reproduction verification for bug issues (requires --reason)")
	completeCmd.Flags().StringVar(&completeSkipReproReason, "skip-repro-reason", "", "Reason for skipping reproduction verification")
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

func runComplete(beadsID, workdir string) error {
	// Resolve short beads ID to full ID (e.g., "qdaa" -> "orch-go-qdaa")
	// This ensures all downstream operations use the full ID consistently
	resolvedID, err := resolveShortBeadsID(beadsID)
	if err != nil {
		return fmt.Errorf("failed to resolve beads ID: %w", err)
	}
	beadsID = resolvedID

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
	// BUT: Skip this check if Phase: Complete was reported - agent said it's done,
	// so whether its session is still open is irrelevant.
	// This prevents false positives from OpenCode sessions that persist to disk.
	if !completeForce {
		// Check if Phase: Complete was reported
		phaseComplete, _ := verify.IsPhaseComplete(beadsID)

		// Only check liveness if agent hasn't reported completion
		if !phaseComplete {
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
	}

	// DISABLED: Reproduction verification gate (Jan 4, 2026)
	// This was added to ensure bugs are actually fixed before closing, but it created
	// too much friction - agents couldn't complete without manual intervention.
	// Keeping the code commented for potential future re-enablement with better UX.
	// See: .kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md
	/*
		if !completeSkipReproCheck {
			reproResult, err := verify.GetReproForCompletion(beadsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check reproduction: %v\n", err)
			} else if reproResult != nil && reproResult.IsBug {
				// ... gate logic disabled ...
			}
		}
	*/
	_ = completeSkipReproCheck  // silence unused variable warning
	_ = completeSkipReproReason // silence unused variable warning

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

		// Remove triage:ready label on successful completion
		// This ensures failed/abandoned agents leave issues in ready queue for daemon retry
		if err := verify.RemoveTriageReadyLabel(beadsID); err != nil {
			// Non-critical - the issue may not have had this label
			// or it was already removed
		}
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

	// Check for notable changelog entries (BREAKING/behavioral changes, especially skill changes)
	if !completeNoChangelogCheck {
		// Extract agent's skill from workspace if available
		var agentSkill string
		if workspacePath != "" {
			agentSkill, _ = verify.ExtractSkillNameFromSpawnContext(workspacePath)
		}

		notableEntries := detectNotableChangelogEntries(beadsProjectDir, agentSkill)
		if len(notableEntries) > 0 {
			fmt.Println()
			fmt.Println("┌─────────────────────────────────────────────────────────────┐")
			fmt.Println("│  ⚠️  NOTABLE ECOSYSTEM CHANGES DETECTED                      │")
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			for _, entry := range notableEntries {
				// Wrap long entries
				if len(entry) > 55 {
					fmt.Printf("│  %s\n", entry[:55])
					fmt.Printf("│    %s\n", entry[55:])
				} else {
					fmt.Printf("│  %s\n", entry)
				}
			}
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			fmt.Println("│  Review recent changes that may affect agent behavior       │")
			fmt.Println("│  Run: orch changelog --days 3                               │")
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

	// Invalidate orch serve cache to ensure dashboard shows updated status immediately.
	// Without this, the TTL cache holds stale "active" status after completion.
	invalidateServeCache()

	return nil
}

// invalidateServeCache sends a request to orch serve to invalidate its caches.
// This ensures the dashboard shows updated agent status immediately after completion.
// Silently fails if orch serve is not running (cache will refresh via TTL).
func invalidateServeCache() {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Post(
		fmt.Sprintf("http://localhost:%d/api/cache/invalidate", DefaultServePort),
		"application/json",
		nil,
	)
	if err != nil {
		// Silent failure - orch serve might not be running
		return
	}
	defer resp.Body.Close()
	// We don't care about the response - if it worked, great; if not, TTL will eventually refresh
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

// NotableChangelogEntry represents a notable change from the changelog.
type NotableChangelogEntry struct {
	Commit CommitInfo
	Reason string // Why this is notable (e.g., "BREAKING", "skill-relevant", "behavioral")
}

// detectNotableChangelogEntries checks recent commits across ecosystem repos for
// notable changes that the orchestrator should be aware of:
// - BREAKING changes
// - Behavioral changes (feat/fix commits)
// - Skill changes relevant to the agent's skill
// Returns formatted strings for display.
func detectNotableChangelogEntries(projectDir string, agentSkill string) []string {
	var entries []string

	// Get changelog data for last 3 days (recent enough to be relevant)
	result, err := GetChangelog(3, "all")
	if err != nil {
		return nil
	}

	// Iterate through commits looking for notable entries
	for _, dateCommits := range result.CommitsByDate {
		for _, commit := range dateCommits {
			var reasons []string

			// Check for BREAKING changes
			if commit.SemanticInfo.IsBreaking {
				reasons = append(reasons, "BREAKING")
			}

			// Check for behavioral changes (feat/fix)
			if commit.SemanticInfo.ChangeType == ChangeTypeBehavioral {
				// Only surface if it's in a category that could affect agents
				if commit.Category == "skills" || commit.Category == "skill-behavioral" ||
					commit.Category == "cmd" || commit.Category == "pkg" {
					reasons = append(reasons, "behavioral")
				}
			}

			// Check for skill-relevant changes
			if agentSkill != "" && isSkillRelevantChange(commit, agentSkill) {
				reasons = append(reasons, fmt.Sprintf("relevant to %s", agentSkill))
			}

			// If we have reasons, add to the list
			if len(reasons) > 0 {
				icon := "📌"
				if commit.SemanticInfo.IsBreaking {
					icon = "🚨"
				} else if strings.Contains(strings.Join(reasons, ","), "relevant to") {
					icon = "🎯"
				}

				entry := fmt.Sprintf("%s [%s] %s (%s)",
					icon,
					commit.Repo,
					truncateString(commit.Subject, 40),
					strings.Join(reasons, ", "))
				entries = append(entries, entry)
			}
		}
	}

	// Limit to top 5 most notable entries to avoid noise
	if len(entries) > 5 {
		entries = entries[:5]
	}

	return entries
}

// isSkillRelevantChange checks if a commit affects files related to a specific skill.
func isSkillRelevantChange(commit CommitInfo, skillName string) bool {
	for _, file := range commit.Files {
		// Check for skill-specific paths (handles both "skills/" prefix and "/skills/")
		if strings.Contains(file, "skills/") {
			// Check if this skill is mentioned in the path
			if strings.Contains(file, "/"+skillName+"/") ||
				strings.Contains(file, "/"+skillName+".") ||
				strings.HasPrefix(file, "skills/"+skillName+"/") ||
				strings.Contains(file, "/skills/"+skillName+"/") {
				return true
			}
		}

		// Check for SPAWN_CONTEXT or spawn package changes (affects all skills)
		if strings.Contains(file, "SPAWN_CONTEXT") ||
			strings.Contains(file, "pkg/spawn/") {
			return true
		}

		// Check for skill verification changes
		if strings.Contains(file, "pkg/verify/skill") {
			return true
		}
	}
	return false
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
	cleanInvestigations bool
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
  --investigations  Archive empty investigation files (agents died before filling template)

Note: This command never deletes workspace directories - they are kept for 
investigation reference. Use 'rm -rf .orch/workspace/<name>' to manually delete.

Examples:
  orch-go clean                    # List completed agents (no changes)
  orch-go clean --dry-run          # Preview mode (same as default)
  orch-go clean --windows          # Close tmux windows for completed agents
  orch-go clean --phantoms         # Close phantom tmux windows
  orch-go clean --verify-opencode  # Delete orphaned OpenCode disk sessions
  orch-go clean --investigations   # Archive empty investigation templates`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClean(cleanDryRun, cleanVerifyOpenCode, cleanWindows, cleanPhantoms, cleanInvestigations)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanCmd.Flags().BoolVar(&cleanVerifyOpenCode, "verify-opencode", false, "Also verify OpenCode disk sessions (slower)")
	cleanCmd.Flags().BoolVar(&cleanWindows, "windows", false, "Close tmux windows for completed agents")
	cleanCmd.Flags().BoolVar(&cleanPhantoms, "phantoms", false, "Close all phantom tmux windows (stale agent windows)")
	cleanCmd.Flags().BoolVar(&cleanInvestigations, "investigations", false, "Archive empty investigation files to .kb/investigations/archived/")
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

func runClean(dryRun bool, verifyOpenCode bool, closeWindows bool, cleanPhantoms bool, cleanInvestigations bool) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find completed workspaces using derived lookups
	fmt.Println("Scanning workspaces for completed agents...")
	beadsChecker := NewDefaultBeadsStatusChecker()
	cleanableWorkspaces := findCleanableWorkspaces(projectDir, beadsChecker)

	fmt.Printf("\nFound %d completed workspaces\n", len(cleanableWorkspaces))

	if len(cleanableWorkspaces) == 0 && !verifyOpenCode && !cleanPhantoms && !cleanInvestigations {
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

	// Clean empty investigation files (optional)
	var investigationsArchived int
	if cleanInvestigations {
		investigationsArchived, err = archiveEmptyInvestigations(projectDir, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive empty investigations: %v\n", err)
		}
	}

	// Check if any cleanup actions were taken or would be taken
	hasCleanupActions := closeWindows || cleanPhantoms || verifyOpenCode || cleanInvestigations

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
			if cleanInvestigations && investigationsArchived > 0 {
				fmt.Printf(" Would archive %d empty investigations.", investigationsArchived)
			}
			fmt.Println()
		}
		return nil
	}

	// Log if any cleanup actions were taken
	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 || investigationsArchived > 0 {
		projectName := filepath.Base(projectDir)
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "agents.cleaned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"completed_workspaces":    len(cleanableWorkspaces),
				"windows_closed":          windowsClosed,
				"phantoms_closed":         phantomsClosed,
				"disk_sessions_deleted":   diskSessionsDeleted,
				"investigations_archived": investigationsArchived,
				"project":                 projectName,
				"verify_opencode":         verifyOpenCode,
				"close_windows":           closeWindows,
				"clean_phantoms":          cleanPhantoms,
				"clean_investigations":    cleanInvestigations,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
		}
	}

	// Print summary of actions taken (not misleading "cleaned X workspaces")
	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 || investigationsArchived > 0 {
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
		if investigationsArchived > 0 {
			fmt.Printf("Archived %d empty investigation files\n", investigationsArchived)
		}
	} else if !hasCleanupActions {
		// Default: just listing completed workspaces
		fmt.Printf("\nNote: Workspace directories are preserved. Use --windows, --phantoms, --verify-opencode, or --investigations to clean up resources.\n")
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

// emptyInvestigationPlaceholders are patterns that indicate an investigation file was never filled in.
// These are template placeholders from kb create investigation that agents should replace.
var emptyInvestigationPlaceholders = []string{
	"[Brief, descriptive title]",
	"[Clear, specific question",
	"[Concrete observations, data, examples]",
	"[File paths with line numbers",
	"[Explanation of the insight",
}

// isEmptyInvestigation checks if an investigation file still has template placeholders.
// Returns true if the file contains multiple placeholder patterns, indicating it was never filled in.
func isEmptyInvestigation(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	contentStr := string(content)
	placeholderCount := 0
	for _, placeholder := range emptyInvestigationPlaceholders {
		if strings.Contains(contentStr, placeholder) {
			placeholderCount++
		}
	}

	// Require at least 2 placeholder patterns to be considered empty
	// (to avoid false positives from files that just mention placeholders in documentation)
	return placeholderCount >= 2
}

// archiveEmptyInvestigations moves empty investigation files to .kb/investigations/archived/.
// Returns the number of files archived and any error encountered.
func archiveEmptyInvestigations(projectDir string, dryRun bool) (int, error) {
	investigationsDir := filepath.Join(projectDir, ".kb", "investigations")
	archivedDir := filepath.Join(investigationsDir, "archived")

	// Check if investigations directory exists
	if _, err := os.Stat(investigationsDir); os.IsNotExist(err) {
		fmt.Println("\nNo .kb/investigations directory found")
		return 0, nil
	}

	fmt.Println("\nScanning for empty investigation files...")

	// Find all empty investigation files
	var emptyFiles []string
	err := filepath.Walk(investigationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories and non-markdown files
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Skip files already in archived folder
		if strings.Contains(path, "/archived/") {
			return nil
		}

		if isEmptyInvestigation(path) {
			emptyFiles = append(emptyFiles, path)
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to scan investigations: %w", err)
	}

	if len(emptyFiles) == 0 {
		fmt.Println("  No empty investigation files found")
		return 0, nil
	}

	fmt.Printf("  Found %d empty investigation files:\n", len(emptyFiles))

	// Create archived directory if needed
	if !dryRun {
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create archived directory: %w", err)
		}
	}

	// Archive empty files
	archived := 0
	for _, path := range emptyFiles {
		filename := filepath.Base(path)

		// Preserve subdirectory structure (e.g., simple/)
		relPath, _ := filepath.Rel(investigationsDir, path)
		destDir := filepath.Join(archivedDir, filepath.Dir(relPath))
		destPath := filepath.Join(destDir, filename)

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would archive: %s\n", relPath)
			archived++
			continue
		}

		// Create destination subdirectory if needed
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to create directory %s: %v\n", destDir, err)
			continue
		}

		// Move file to archived
		if err := os.Rename(path, destPath); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", relPath, err)
			continue
		}

		fmt.Printf("    Archived: %s\n", relPath)
		archived++
	}

	return archived, nil
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
