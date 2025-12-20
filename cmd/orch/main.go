// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
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
}

var spawnCmd = &cobra.Command{
	Use:   "spawn [prompt]",
	Short: "Spawn a new OpenCode session",
	Long:  "Spawn a new OpenCode session with the given prompt.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := args[0]
		for i := 1; i < len(args); i++ {
			prompt += " " + args[i]
		}

		title := fmt.Sprintf("orch-go-%d", time.Now().Unix())
		return runSpawn(serverURL, prompt, title)
	},
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

func runSpawn(serverURL, prompt, title string) error {
	client := opencode.NewClient(serverURL)
	cmd := client.BuildSpawnCommand(prompt, title)
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

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.spawned",
		SessionID: result.SessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"prompt": prompt,
			"title":  title,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Session ID: %s\n", result.SessionID)
	return nil
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
