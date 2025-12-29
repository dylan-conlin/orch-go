// Package main provides the CLI entry point for orch-go.
//
// This file implements the `orch session` command for orchestrator session management.
// An orchestrator session is a "focus block" - a time-bounded period with a goal.
// This is distinct from `orch sessions` (plural) which searches OpenCode session history.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/sessions"
	"github.com/spf13/cobra"
)

// ============================================================================
// Session Command - Orchestrator session management
// ============================================================================

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage orchestrator sessions (focus blocks)",
	Long: `Manage orchestrator sessions (focus blocks).

An orchestrator session represents a focused work period with a specific goal.
Unlike worker sessions (spawn→complete), orchestrator sessions are composite:
they may include multiple agent spawns and have a strategic goal.

Session state is stored at ~/.orch/session.json.

Subcommands:
  start   - Start a new session with a goal
  status  - Show current session status
  end     - End the current session

Examples:
  orch session start "Ship snap MVP"     # Start session with goal
  orch session status                    # Show current session
  orch session end                       # End current session`,
}

var sessionStartCmd = &cobra.Command{
	Use:   "start [goal]",
	Short: "Start a new orchestrator session",
	Long: `Start a new orchestrator session with an optional goal.

Creates a new session with a unique ID and sets the focus to the given goal.
The session tracks when it started, its goal, and spawned agents.

If a goal is provided, it will also set the focus using 'orch focus'.

Examples:
  orch session start                           # Start without explicit goal
  orch session start "Ship snap MVP"           # Start with goal (also sets focus)
  orch session start "Fix auth bugs" --issue proj-123  # Start with focus and issue`,
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := ""
		if len(args) > 0 {
			goal = strings.Join(args, " ")
		}
		return runSessionStart(goal)
	},
}

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current session status",
	Long: `Show the current orchestrator session status.

Displays session ID, duration, goal, and spawned agents.

Examples:
  orch session status     # Show current session`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionStatus()
	},
}

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: "End the current session",
	Long: `End the current orchestrator session.

Clears the active session state. Use 'orch handoff' to generate a handoff
document before ending if you want to preserve context for the next session.

Examples:
  orch session end     # End current session`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionEnd()
	},
}

var (
	// Session start flags
	sessionStartIssue string
)

func init() {
	sessionStartCmd.Flags().StringVar(&sessionStartIssue, "issue", "", "Beads issue ID to associate with focus")

	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionEndCmd)

	// Register with root command
	rootCmd.AddCommand(sessionCmd)
}

func runSessionStart(goal string) error {
	// Create orchestrator session store
	store, err := sessions.NewOrchestratorStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Check if session already exists
	existing := store.Get()
	if existing != nil {
		fmt.Printf("Note: Ending previous session: %s (%s)\n",
			existing.ID,
			formatDuration(time.Since(existing.Started)))
	}

	// Start new session
	session, err := store.Start(goal)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	// If goal provided, also set focus
	if goal != "" {
		focusStore, err := focus.New("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize focus store: %v\n", err)
		} else {
			f := &focus.Focus{
				Goal:    goal,
				BeadsID: sessionStartIssue,
			}
			if err := focusStore.Set(f); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to set focus: %v\n", err)
			} else {
				// Update session with focus ID
				if err := store.SetFocusID(goal); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to link focus to session: %v\n", err)
				}
			}
		}
	}

	// Log the session start
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.orchestrator.started",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id": session.ID,
			"goal":       goal,
			"beads_id":   sessionStartIssue,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Output confirmation
	fmt.Printf("Session started:\n")
	fmt.Printf("  ID:      %s\n", session.ID)
	fmt.Printf("  Started: %s\n", session.Started.Format("2006-01-02 15:04:05"))
	if goal != "" {
		fmt.Printf("  Goal:    %s\n", goal)
	}
	if sessionStartIssue != "" {
		fmt.Printf("  Issue:   %s\n", sessionStartIssue)
	}

	return nil
}

func runSessionStatus() error {
	store, err := sessions.NewOrchestratorStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	session := store.Get()
	if session == nil {
		fmt.Println("No active session")
		fmt.Println("\nStart a session with: orch session start \"your goal\"")
		return nil
	}

	// Calculate duration
	duration := time.Since(session.Started)

	fmt.Printf("Current session:\n")
	fmt.Printf("  ID:       %s\n", session.ID)
	fmt.Printf("  Started:  %s\n", session.Started.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Duration: %s\n", formatDuration(duration))
	if session.Goal != "" {
		fmt.Printf("  Goal:     %s\n", session.Goal)
	}
	if session.FocusID != "" {
		fmt.Printf("  Focus:    %s\n", session.FocusID)
	}

	// Show spawned agents
	if len(session.Spawns) > 0 {
		fmt.Printf("\nSpawned agents (%d):\n", len(session.Spawns))
		for _, spawn := range session.Spawns {
			age := formatDuration(time.Since(spawn.SpawnedAt))
			fmt.Printf("  - %s (%s) - %s ago\n", spawn.BeadsID, spawn.Skill, age)
		}
	} else {
		fmt.Printf("\nNo agents spawned yet\n")
	}

	return nil
}

func runSessionEnd() error {
	store, err := sessions.NewOrchestratorStore("")
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	session, err := store.End()
	if err != nil {
		if err.Error() == "no active session" {
			fmt.Println("No active session to end")
			return nil
		}
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Calculate duration
	duration := time.Since(session.Started)

	// Log the session end
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.orchestrator.ended",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"session_id": session.ID,
			"goal":       session.Goal,
			"duration":   duration.Seconds(),
			"spawns":     len(session.Spawns),
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Session ended:\n")
	fmt.Printf("  ID:       %s\n", session.ID)
	fmt.Printf("  Duration: %s\n", formatDuration(duration))
	if session.Goal != "" {
		fmt.Printf("  Goal:     %s\n", session.Goal)
	}
	fmt.Printf("  Spawns:   %d agents\n", len(session.Spawns))

	fmt.Println("\nTip: Run 'orch handoff' to generate a handoff document for the next session")

	return nil
}

// Note: formatDuration is defined in wait.go and shared across commands
