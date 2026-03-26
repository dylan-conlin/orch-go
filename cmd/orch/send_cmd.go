// Package main provides the send command for messaging existing sessions.
// Extracted from main.go as part of the main.go refactoring.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/identity"
	"github.com/dylan-conlin/orch-go/pkg/execution"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

var (
	// Send command flags
	sendAsync   bool
	sendWorkdir string
)

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

func init() {
	sendCmd.Flags().BoolVar(&sendAsync, "async", true, "Send message asynchronously (non-blocking)")
	sendCmd.Flags().StringVar(&sendWorkdir, "workdir", "", "Target project directory (for cross-project sends)")
}

func runSend(serverURL, identifier, message string) error {
	// Resolve project directory for cross-project workspace lookups
	projectDir, err := identity.ResolveProject(identifier, sendWorkdir)
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	client := execution.NewOpenCodeAdapter(serverURL)

	// Strategy 1: Workspace lookup by beads ID (uses resolved project dir)
	workspacePath, _ := findWorkspaceByBeadsID(projectDir, identifier)
	if workspacePath != "" {
		sessionID := spawn.ReadSessionID(workspacePath)
		if sessionID != "" && isOpenCodeSessionID(sessionID) {
			return sendViaOpenCodeAPI(client, sessionID, identifier, message)
		}
	}

	// Strategy 2: Try resolveSessionID (covers ses_xxx, workspace name, API lookup)
	sessionID, resolveErr := resolveSessionID(serverURL, identifier)
	if resolveErr == nil && sessionID != "" {
		return sendViaOpenCodeAPI(client, sessionID, identifier, message)
	}

	// Strategy 3: tmux send-keys fallback
	windowInfo, err := findTmuxWindowByIdentifier(identifier)
	if err != nil || windowInfo == nil {
		if resolveErr != nil {
			return fmt.Errorf("failed to resolve session and no tmux window found: %w", resolveErr)
		}
		return fmt.Errorf("no session or tmux window found for identifier: %s", identifier)
	}

	return sendViaTmux(windowInfo, identifier, message)
}

// sendViaOpenCodeAPI sends a message using the OpenCode HTTP API.
func sendViaOpenCodeAPI(client execution.SessionClient, sessionID, identifier, message string) error {
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
		if err := client.SendMessageAsync(context.Background(), execution.SessionHandle(sessionID), message, ""); err != nil {
			return fmt.Errorf("failed to send message asynchronously: %w", err)
		}
		fmt.Printf("✓ Message sent to session %s (via API)\n", sessionID)
		return nil
	}

	// Send message and stream the response to stdout (blocking)
	if err := client.SendMessageWithStreaming(context.Background(), execution.SessionHandle(sessionID), message, os.Stdout); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Add newline at end for clean output
	fmt.Println()

	return nil
}

// sendViaTmux sends a message to a tmux window using send-keys.
// This is used as a fallback when the OpenCode session ID cannot be resolved.
func sendViaTmux(windowInfo *tmux.WindowInfo, identifier, message string) error {
	// Prefer stable window ID (@xxx) over session:index target.
	// Window indices can change when windows are created/destroyed,
	// but window IDs are stable for the lifetime of the window.
	target := windowInfo.ID
	if target == "" {
		target = windowInfo.Target
	}

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
			"target_used":   target,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Send text and submit with delay between text and Enter.
	// Without the delay, Enter gets processed before the TUI has fully ingested
	// the pasted text, causing the message to sit in the input without submitting.
	// This matches the Python orch-cli pattern (send.py:101-110).
	if err := tmux.SendTextAndSubmit(target, message, tmux.DefaultSendDelay); err != nil {
		return fmt.Errorf("failed to send message via tmux: %w", err)
	}

	fmt.Printf("✓ Message sent to %s (via tmux %s)\n", identifier, target)
	return nil
}
