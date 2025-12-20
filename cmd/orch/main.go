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
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/gen2brain/beeep"
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
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(statusCmd)
}

var (
	// Spawn command flags
	spawnSkill      string
	spawnIssue      string
	spawnPhases     string
	spawnMode       string
	spawnValidation string
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [skill] [task]",
	Short: "Spawn a new OpenCode session with skill context",
	Long: `Spawn a new OpenCode session with skill context.

Examples:
  orch-go spawn investigation "explore the codebase"
  orch-go spawn feature-impl "add new spawn command" --phases implementation,validation
  orch-go spawn --issue proj-123 feature-impl "implement the feature"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		task := strings.Join(args[1:], " ")

		return runSpawnWithSkill(serverURL, skillName, task)
	},
}

func init() {
	spawnCmd.Flags().StringVar(&spawnIssue, "issue", "", "Beads issue ID for tracking")
	spawnCmd.Flags().StringVar(&spawnPhases, "phases", "", "Feature-impl phases (e.g., implementation,validation)")
	spawnCmd.Flags().StringVar(&spawnMode, "mode", "tdd", "Implementation mode: tdd or direct")
	spawnCmd.Flags().StringVar(&spawnValidation, "validation", "tests", "Validation level: none, tests, smoke-test")
}

var askCmd = &cobra.Command{
	Use:   "ask [session-id] [prompt]",
	Short: "Send a message to an existing session",
	Long:  "Send a message to an existing OpenCode session.",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		prompt := args[1]
		for i := 2; i < len(args); i++ {
			prompt += " " + args[i]
		}

		return runAsk(serverURL, sessionID, prompt)
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

func runSpawnWithSkill(serverURL, skillName, task string) error {
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

	// Spawn opencode session
	client := opencode.NewClient(serverURL)
	cmd := client.BuildSpawnCommand(minimalPrompt, workspaceName)
	cmd.Stderr = os.Stderr
	cmd.Dir = projectDir

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
			"skill":     skillName,
			"task":      task,
			"workspace": workspaceName,
			"beads_id":  beadsID,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary
	fmt.Printf("Spawned agent:\n")
	fmt.Printf("  Session ID: %s\n", result.SessionID)
	fmt.Printf("  Workspace:  %s\n", workspaceName)
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

	// Parse issue ID from output (expected format: "Created issue: proj-123")
	outputStr := strings.TrimSpace(string(output))
	parts := strings.Split(outputStr, " ")
	if len(parts) > 0 {
		// Take the last word which should be the issue ID
		return parts[len(parts)-1], nil
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

func runAsk(serverURL, sessionID, prompt string) error {
	client := opencode.NewClient(serverURL)
	cmd := client.BuildAskCommand(sessionID, prompt)
	cmd.Stderr = os.Stderr

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

	// Log the Q&A
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.ask",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"prompt":      prompt,
			"event_count": len(result.Events),
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Q&A complete for session: %s\n", sessionID)
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

func runMonitor(serverURL string) error {
	sseURL := serverURL + "/event"
	client := opencode.NewSSEClient(sseURL)

	fmt.Printf("Monitoring SSE events at %s...\n", sseURL)

	sseEvents := make(chan opencode.SSEEvent, 100)
	errChan := make(chan error, 1)

	go func() {
		if err := client.Connect(sseEvents); err != nil {
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
				title := "OpenCode Session Update"
				body := fmt.Sprintf("Session %s: completed", sessionID)
				if err := beeep.Notify(title, body, ""); err != nil {
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
