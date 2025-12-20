// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/notify"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	serverURL string

	// Version information (set at build time)
	version = "dev"
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
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(completeCmd)
}

var (
	// Spawn command flags
	spawnSkill      string
	spawnIssue      string
	spawnPhases     string
	spawnMode       string
	spawnValidation string
	spawnInline     bool // Run inline (blocking) instead of in tmux
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [skill] [task]",
	Short: "Spawn a new OpenCode session with skill context",
	Long: `Spawn a new OpenCode session with skill context.

By default, spawns the agent in a tmux window and returns immediately.
Use --inline to run in the current terminal (blocking).

Examples:
  orch-go spawn investigation "explore the codebase"
  orch-go spawn feature-impl "add new spawn command" --phases implementation,validation
  orch-go spawn --issue proj-123 feature-impl "implement the feature"
  orch-go spawn --inline investigation "explore codebase"  # Run inline (blocking)`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		task := strings.Join(args[1:], " ")

		return runSpawnWithSkill(serverURL, skillName, task, spawnInline)
	},
}

func init() {
	spawnCmd.Flags().StringVar(&spawnIssue, "issue", "", "Beads issue ID for tracking")
	spawnCmd.Flags().StringVar(&spawnPhases, "phases", "", "Feature-impl phases (e.g., implementation,validation)")
	spawnCmd.Flags().StringVar(&spawnMode, "mode", "tdd", "Implementation mode: tdd or direct")
	spawnCmd.Flags().StringVar(&spawnValidation, "validation", "tests", "Validation level: none, tests, smoke-test")
	spawnCmd.Flags().BoolVar(&spawnInline, "inline", false, "Run inline (blocking) instead of in tmux")
}

var askCmd = &cobra.Command{
	Use:   "ask [session-id] [prompt]",
	Short: "Send a message to an existing session (alias for send)",
	Long:  "Send a message to an existing OpenCode session. This is an alias for the 'send' command.",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		prompt := strings.Join(args[1:], " ")
		return runSend(serverURL, sessionID, prompt)
	},
}

var sendCmd = &cobra.Command{
	Use:   "send [session-id] [message]",
	Short: "Send a message to an existing session",
	Long: `Send a message to an existing OpenCode session.

The session can be running or completed. Response text is streamed to stdout
as it's received from the agent.

Examples:
  orch-go send ses_abc123 "what files did you modify?"
  orch-go send ses_xyz789 "can you explain the changes?"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		message := strings.Join(args[1:], " ")
		return runSend(serverURL, sessionID, message)
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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "List active OpenCode sessions",
	Long: `List all active OpenCode sessions with their status.

Shows session ID, workspace/title, directory, and last update time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(serverURL)
	},
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

func runSpawnWithSkill(serverURL, skillName, task string, inline bool) error {
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

	// Determine beads ID - either from flag or create new issue
	beadsID := spawnIssue
	if beadsID == "" {
		// Create a new beads issue
		beadsID, err = createBeadsIssue(projectName, skillName, task)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create beads issue: %v\n", err)
			beadsID = fmt.Sprintf("%s-%d", projectName, time.Now().Unix()) // Fallback ID
		}
	}

	// Update beads issue status to in_progress
	if err := verify.UpdateIssueStatus(beadsID, "in_progress"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
		// Continue anyway
	}

	// Build spawn config
	cfg := &spawn.Config{
		Task:          task,
		SkillName:     skillName,
		Project:       projectName,
		ProjectDir:    projectDir,
		WorkspaceName: workspaceName,
		SkillContent:  skillContent,
		BeadsID:       beadsID,
		Phases:        spawnPhases,
		Mode:          spawnMode,
		Validation:    spawnValidation,
	}

	// Write SPAWN_CONTEXT.md
	if err := spawn.WriteContext(cfg); err != nil {
		return fmt.Errorf("failed to write spawn context: %w", err)
	}

	// Generate minimal prompt
	minimalPrompt := spawn.MinimalPrompt(cfg)

	// Decide spawn mode: tmux (default) or inline
	useTmux := !inline && tmux.IsAvailable()

	if useTmux {
		return runSpawnInTmux(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Inline mode (blocking) - original behavior
	return runSpawnInline(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}

// runSpawnInTmux spawns the agent in a tmux window and returns immediately.
func runSpawnInTmux(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	// Ensure workers session exists
	sessionName, err := tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to ensure workers session: %w", err)
	}

	// Build window name
	windowName := tmux.BuildWindowName(cfg.WorkspaceName, skillName, beadsID)

	// Create window
	windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}

	// Build opencode command (without --format json so TUI shows)
	tmuxCfg := &tmux.SpawnConfig{
		ServerURL:     serverURL,
		Prompt:        minimalPrompt,
		Title:         cfg.WorkspaceName,
		ProjectDir:    cfg.ProjectDir,
		WorkspaceName: cfg.WorkspaceName,
	}
	cmd := tmux.BuildSpawnCommand(tmuxCfg)
	opencodeCmd := strings.Join(cmd.Args, " ")

	// Send the command to the tmux window
	if err := tmux.SendKeysLiteral(windowTarget, opencodeCmd); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	// Send Enter to execute
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	// Select the window to focus it
	_ = tmux.SelectWindow(windowTarget)

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"skill":      skillName,
			"task":       task,
			"workspace":  cfg.WorkspaceName,
			"beads_id":   beadsID,
			"window":     windowTarget,
			"window_id":  windowID,
			"spawn_mode": "tmux",
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary
	fmt.Printf("Spawned agent:\n")
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Window:     %s\n", windowTarget)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Context:    %s\n", cfg.ContextFilePath())

	return nil
}

// runSpawnInline spawns the agent inline (blocking) - original behavior.
func runSpawnInline(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	// Spawn opencode session
	client := opencode.NewClient(serverURL)
	cmd := client.BuildSpawnCommand(minimalPrompt, cfg.WorkspaceName)
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

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.spawned",
		SessionID: result.SessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"skill":      skillName,
			"task":       task,
			"workspace":  cfg.WorkspaceName,
			"beads_id":   beadsID,
			"spawn_mode": "inline",
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary
	fmt.Printf("Spawned agent:\n")
	fmt.Printf("  Session ID: %s\n", result.SessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Context:    %s\n", cfg.ContextFilePath())

	return nil
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

	// Parse issue ID from output (multi-line format with first line: "✓ Created issue: proj-123")
	outputStr := strings.TrimSpace(string(output))

	// Split by newline and parse first line only
	lines := strings.Split(outputStr, "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("empty output from bd create")
	}

	firstLine := strings.TrimSpace(lines[0])

	// Look for "issue:" in the first line and extract the ID after it
	parts := strings.Fields(firstLine)
	for i, part := range parts {
		if strings.Contains(part, "issue:") {
			// Issue ID should be the next word after "issue:"
			if i+1 < len(parts) {
				return parts[i+1], nil
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

func runSend(serverURL, sessionID, message string) error {
	client := opencode.NewClient(serverURL)
	cmd := client.BuildAskCommand(sessionID, message)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	// Stream text content to stdout as it arrives
	result, err := opencode.ProcessOutputWithStreaming(stdout, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to process output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("opencode exited with error: %w", err)
	}

	// Log the send
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.send",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message":     message,
			"event_count": len(result.Events),
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print newline after streamed content for cleaner output
	fmt.Println()
	return nil
}

func runStatus(serverURL string) error {
	client := opencode.NewClient(serverURL)
	sessions, err := client.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions")
		return nil
	}

	// Print table header
	fmt.Printf("%-35s %-30s %-40s %s\n", "SESSION ID", "TITLE", "DIRECTORY", "UPDATED")
	fmt.Printf("%s\n", strings.Repeat("-", 130))

	// Print each session
	for _, s := range sessions {
		// Format updated time
		updated := time.Unix(s.Time.Updated/1000, 0).Format("2006-01-02 15:04:05")

		// Truncate long fields
		title := truncate(s.Title, 28)
		dir := truncate(s.Directory, 38)

		fmt.Printf("%-35s %-30s %-40s %s\n", s.ID, title, dir, updated)
	}

	fmt.Printf("\nTotal: %d sessions\n", len(sessions))
	return nil
}

func runComplete(beadsID string) error {
	// Get issue to verify it exists
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Check if already closed
	if issue.Status == "closed" {
		fmt.Printf("Issue %s is already closed\n", beadsID)
		return nil
	}

	// Verify phase status unless force flag is set
	if !completeForce {
		result, err := verify.VerifyCompletion(beadsID)
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

	// Close the beads issue
	if err := verify.CloseIssue(beadsID, reason); err != nil {
		return fmt.Errorf("failed to close issue: %w", err)
	}

	fmt.Printf("Closed beads issue: %s\n", beadsID)
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
	sseURL := serverURL + "/event"
	sseClient := opencode.NewSSEClient(sseURL)
	apiClient := opencode.NewClient(serverURL)
	notifier := notify.Default()

	fmt.Printf("Monitoring SSE events at %s...\n", sseURL)

	sseEvents := make(chan opencode.SSEEvent, 100)
	errChan := make(chan error, 1)

	go func() {
		if err := sseClient.Connect(sseEvents); err != nil {
			errChan <- err
		}
		close(sseEvents)
	}()

	logger := events.NewLogger(events.DefaultLogPath())
	var sessionEvents []opencode.SSEEvent

	for {
		select {
		case event, ok := <-sseEvents:
			if !ok {
				return nil
			}

			// Log every event
			logEvent := events.Event{
				Type:      event.Event,
				Timestamp: time.Now().Unix(),
				Data:      map[string]interface{}{"raw_data": event.Data},
			}

			// Parse session info if available
			if event.Event == "session.status" || event.Event == "session.created" {
				status, sid := opencode.ParseSessionStatus(event.Data)
				if sid != "" {
					logEvent.SessionID = sid
				}
				if status != "" {
					logEvent.Data["status"] = status
				}
			}

			if err := logger.Log(logEvent); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
			}

			fmt.Printf("[%s] %s\n", event.Event, event.Data)
			sessionEvents = append(sessionEvents, event)

			// Check for completion
			sessionID, completed := opencode.DetectCompletion(sessionEvents)
			if completed {
				// Try to get session title for notification
				workspace := ""
				if session, err := apiClient.GetSession(sessionID); err == nil && session != nil {
					workspace = session.Title
				}

				if err := notifier.SessionComplete(sessionID, workspace); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to send notification: %v\n", err)
				}
				fmt.Printf("\nSession %s completed!\n", sessionID)

				// Log completion
				completionEvent := events.Event{
					Type:      "session.completed",
					SessionID: sessionID,
					Timestamp: time.Now().Unix(),
				}
				if err := logger.Log(completionEvent); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log completion: %v\n", err)
				}

				// Reset for next session
				sessionEvents = nil
			}

		case err := <-errChan:
			return fmt.Errorf("SSE connection error: %w", err)
		}
	}
}
