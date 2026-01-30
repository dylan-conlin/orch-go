// Package main provides the tail command for capturing agent output.
// Extracted from main.go as part of the main.go refactoring.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

var (
	// Tail command flags
	tailLines     int
	tailSessionID string // Direct session ID for non-orch-spawned sessions
)

var tailCmd = &cobra.Command{
	Use:   "tail [beads-id]",
	Short: "Capture recent output from an agent",
	Long: `Capture recent output from an agent for debugging.

Fetches messages from the OpenCode API for the agent's session.

Examples:
  orch-go tail proj-123              # Capture last 50 lines (default)
  orch-go tail proj-123 --lines 100  # Capture last 100 lines
  orch-go tail proj-123 -n 20        # Capture last 20 lines
  orch-go tail --session ses_xxx     # Capture from a specific OpenCode session (for non-orch-spawned sessions)`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Direct session ID access for non-orch-spawned sessions
		if tailSessionID != "" {
			return runTailBySessionID(tailSessionID, tailLines)
		}
		// Standard beads ID lookup
		if len(args) == 0 {
			return fmt.Errorf("beads-id required (or use --session for direct session access)")
		}
		beadsID := args[0]
		return runTail(beadsID, tailLines)
	},
}

func init() {
	tailCmd.Flags().IntVarP(&tailLines, "lines", "n", 50, "Number of lines to capture")
	tailCmd.Flags().StringVar(&tailSessionID, "session", "", "OpenCode session ID for direct access (for non-orch-spawned sessions)")
}

// runTailBySessionID fetches messages directly from an OpenCode session by its ID.
// This is used for sessions that weren't spawned via orch (no beads ID association).
func runTailBySessionID(sessionID string, lines int) error {
	client := opencode.NewClient(serverURL)

	// Verify session exists
	session, err := client.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %s (ensure OpenCode server is running and session exists)", sessionID)
	}

	// Fetch messages
	messages, err := client.GetMessages(sessionID)
	if err != nil {
		return fmt.Errorf("failed to fetch messages for session %s: %w", sessionID, err)
	}

	if len(messages) == 0 {
		fmt.Printf("=== No messages in session %s (%s) ===\n", truncateSessionID(sessionID), session.Title)
		return nil
	}

	// Extract and display recent text
	textLines := opencode.ExtractRecentText(messages, lines)
	fmt.Printf("=== Output from session %s (%s, last %d lines) ===\n", truncateSessionID(sessionID), session.Title, lines)
	for _, line := range textLines {
		fmt.Println(line)
	}
	fmt.Printf("=== End of output ===\n")
	return nil
}

// truncateSessionID shortens a session ID for display (ses_xxx... -> ses_xxx...)
func truncateSessionID(id string) string {
	if len(id) <= 16 {
		return id
	}
	return id[:16] + "..."
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
