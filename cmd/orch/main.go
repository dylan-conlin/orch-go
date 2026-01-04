// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/question"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
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
