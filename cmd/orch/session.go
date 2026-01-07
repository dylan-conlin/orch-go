// Package main provides the CLI entry point for orch-go.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

// ============================================================================
// Session Command - Manage orchestrator work sessions
// ============================================================================

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage orchestrator work sessions",
	Long: `Manage orchestrator work sessions.

A session represents a focused work period with:
- A goal (north star priority)
- Start time
- Tracked spawns during the session

Session status derives agent state at query time via actual liveness checks,
not stored state. This prevents stale tracking.

Examples:
  orch session start "Ship snap MVP"    # Start a new session
  orch session status                   # Show current session status
  orch session end                      # End the current session`,
}

var (
	sessionJSON bool
)

func init() {
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionEndCmd)

	// Add --json flag to status command
	sessionStatusCmd.Flags().BoolVar(&sessionJSON, "json", false, "Output as JSON")

	rootCmd.AddCommand(sessionCmd)
}

// ============================================================================
// Session Start Command
// ============================================================================

var sessionStartCmd = &cobra.Command{
	Use:   "start [goal]",
	Short: "Start a new orchestrator work session",
	Long: `Start a new orchestrator work session with a focus goal.

The session tracks:
- Your focus goal
- When the session started
- Agents spawned during the session

If a session is already active, it will be replaced.

Examples:
  orch session start "Ship snap MVP"
  orch session start "Fix auth bugs"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := strings.Join(args, " ")
		return runSessionStart(goal)
	},
}

func runSessionStart(goal string) error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Check if session was already active
	wasActive := store.IsActive()

	if err := store.Start(goal); err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	// Create session workspace with SESSION_HANDOFF.md
	workspacePath, err := createSessionWorkspace(goal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create session workspace: %v\n", err)
		// Continue anyway - workspace is nice-to-have for interactive sessions
	}

	// Log the session start
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"goal":       goal,
		"was_active": wasActive,
		"started_at": time.Now().Format(session.TimeFormat),
	}
	if workspacePath != "" {
		eventData["workspace_path"] = workspacePath
	}
	event := events.Event{
		Type:      "session.started",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	if wasActive {
		fmt.Println("Previous session ended.")
	}
	fmt.Printf("Session started: %s\n", goal)
	fmt.Printf("  Start time: %s\n", time.Now().Format("15:04"))
	if workspacePath != "" {
		fmt.Printf("  Workspace:  %s\n", workspacePath)
	}

	return nil
}

// createSessionWorkspace creates a workspace directory for interactive orchestrator sessions.
// This provides parity with spawned orchestrators by pre-creating SESSION_HANDOFF.md.
// Workspace is created in ~/.orch/session/{date}/ to match existing session directory structure.
func createSessionWorkspace(goal string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Use date-based directory for session workspace (matches existing structure)
	dateStr := time.Now().Format("2006-01-02")
	workspacePath := filepath.Join(home, ".orch", "session", dateStr)

	// Create workspace directory
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Generate workspace name for SESSION_HANDOFF.md
	// Interactive sessions use "interactive-" prefix + date + time suffix
	timeStr := time.Now().Format("150405")
	workspaceName := fmt.Sprintf("interactive-%s-%s", dateStr, timeStr)
	startTime := time.Now().Format("2006-01-02 15:04")

	// Use "Interactive session" as default goal if empty
	sessionGoal := goal
	if sessionGoal == "" {
		sessionGoal = "Interactive session"
	}

	// Generate pre-filled SESSION_HANDOFF.md content
	content, err := spawn.GeneratePreFilledSessionHandoff(workspaceName, sessionGoal, startTime)
	if err != nil {
		return "", fmt.Errorf("failed to generate session handoff: %w", err)
	}

	// Write SESSION_HANDOFF.md
	handoffPath := filepath.Join(workspacePath, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write session handoff: %w", err)
	}

	return workspacePath, nil
}

// ============================================================================
// Session Status Command
// ============================================================================

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current session status with spawn reconciliation",
	Long: `Show current session status including spawns with reconciled states.

Spawn states are derived at query time by checking:
- OpenCode session liveness
- tmux window existence
- Beads issue status

This ensures accurate state rather than trusting potentially stale stored data.

States:
  active    - Agent is running (has live OpenCode session or tmux window)
  completed - Agent finished (beads issue closed, no live session)
  phantom   - Agent lost (beads issue open, but no live session)

Examples:
  orch session status         # Show status
  orch session status --json  # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionStatus()
	},
}

// SessionStatusOutput is the JSON output format for session status.
type SessionStatusOutput struct {
	Active     bool                      `json:"active"`
	Goal       string                    `json:"goal,omitempty"`
	StartedAt  string                    `json:"started_at,omitempty"`
	Duration   string                    `json:"duration,omitempty"`
	Spawns     []session.SpawnStatus     `json:"spawns,omitempty"`
	Counts     *SpawnCounts              `json:"counts,omitempty"`
	Checkpoint *session.CheckpointStatus `json:"checkpoint,omitempty"`
}

// SpawnCounts summarizes spawn states.
type SpawnCounts struct {
	Total     int `json:"total"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Phantom   int `json:"phantom"`
}

func runSessionStatus() error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	sess := store.Get()

	// Build output
	output := SessionStatusOutput{
		Active: sess != nil,
	}

	if sess != nil {
		output.Goal = sess.Goal
		output.StartedAt = sess.StartedAt.Format(session.TimeFormat)
		output.Duration = formatSessionDuration(store.Duration())

		// Get spawn statuses with reconciliation
		statuses := store.GetSpawnStatuses(serverURL)
		output.Spawns = statuses

		// Count by state
		counts := &SpawnCounts{Total: len(statuses)}
		for _, s := range statuses {
			switch s.State {
			case "active":
				counts.Active++
			case "completed":
				counts.Completed++
			case "phantom":
				counts.Phantom++
			}
		}
		output.Counts = counts

		// Get checkpoint status
		output.Checkpoint = store.GetCheckpointStatus()
	}

	// JSON output
	if sessionJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	if !output.Active {
		fmt.Println("No active session")
		fmt.Println("\nStart a session with: orch session start \"your goal\"")
		return nil
	}

	fmt.Printf("Session active:\n")
	fmt.Printf("  Goal:     %s\n", output.Goal)
	fmt.Printf("  Duration: %s", output.Duration)

	// Show checkpoint status inline with duration
	if output.Checkpoint != nil {
		switch output.Checkpoint.Level {
		case "exceeded":
			fmt.Printf(" ⛔")
		case "strong":
			fmt.Printf(" 🔴")
		case "warning":
			fmt.Printf(" 🟡")
		}
	}
	fmt.Println()

	fmt.Printf("  Spawns:   %d total", output.Counts.Total)
	if output.Counts.Active > 0 {
		fmt.Printf(" (%d active", output.Counts.Active)
		if output.Counts.Completed > 0 {
			fmt.Printf(", %d completed", output.Counts.Completed)
		}
		if output.Counts.Phantom > 0 {
			fmt.Printf(", %d phantom", output.Counts.Phantom)
		}
		fmt.Printf(")")
	}
	fmt.Println()

	// Show spawn details if any
	if len(output.Spawns) > 0 {
		fmt.Println("\nSpawns:")
		for _, spawn := range output.Spawns {
			stateIcon := stateToIcon(spawn.State)
			age := formatSessionDuration(time.Since(spawn.SpawnedAt))
			fmt.Printf("  %s %s (%s) - %s ago\n", stateIcon, spawn.BeadsID, spawn.Skill, age)
		}
	}

	// Show checkpoint warning if applicable
	if output.Checkpoint != nil && output.Checkpoint.Level != "ok" {
		fmt.Println()
		switch output.Checkpoint.Level {
		case "exceeded":
			fmt.Printf("⛔ CHECKPOINT EXCEEDED: %s\n", output.Checkpoint.Message)
			fmt.Println("   Session has run too long. Quality may be degraded.")
			fmt.Println("   Action: Run 'orch session end' and start fresh.")
		case "strong":
			fmt.Printf("🔴 CHECKPOINT STRONGLY RECOMMENDED: %s\n", output.Checkpoint.Message)
			fmt.Printf("   Time until max: %s\n", formatSessionDuration(output.Checkpoint.NextThreshold))
			fmt.Println("   Action: Write SESSION_HANDOFF.md, consider ending session.")
		case "warning":
			fmt.Printf("🟡 CHECKPOINT SUGGESTED: %s\n", output.Checkpoint.Message)
			fmt.Printf("   Time until strong warning: %s\n", formatSessionDuration(output.Checkpoint.NextThreshold))
			fmt.Println("   Action: Assess progress, write interim handoff if needed.")
		}
	}

	return nil
}

// stateToIcon converts state to a visual indicator.
func stateToIcon(state string) string {
	switch state {
	case "active":
		return "🟢"
	case "completed":
		return "✅"
	case "phantom":
		return "👻"
	default:
		return "❓"
	}
}

// formatSessionDuration formats a duration in a human-readable way for session display.
func formatSessionDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// ============================================================================
// Session End Command
// ============================================================================

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: "End the current session",
	Long: `End the current orchestrator work session.

This clears the session state. Use before:
- Taking a break
- Handing off to another orchestrator
- Changing focus to a different goal

The session summary is logged for posterity.

Examples:
  orch session end`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionEnd()
	},
}

func runSessionEnd() error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	if !store.IsActive() {
		fmt.Println("No active session to end")
		return nil
	}

	// Get session info before ending
	duration := store.Duration()
	spawnCount := store.SpawnCount()

	// Get spawn statuses for final summary
	statuses := store.GetSpawnStatuses(serverURL)
	activeCount := 0
	for _, s := range statuses {
		if s.State == "active" {
			activeCount++
		}
	}

	// End the session
	ended, err := store.End()
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Log the session end
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.ended",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"goal":          ended.Goal,
			"started_at":    ended.StartedAt.Format(session.TimeFormat),
			"duration":      duration.String(),
			"spawn_count":   spawnCount,
			"active_at_end": activeCount,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Session ended: %s\n", ended.Goal)
	fmt.Printf("  Duration:  %s\n", formatSessionDuration(duration))
	fmt.Printf("  Spawns:    %d total\n", spawnCount)

	if activeCount > 0 {
		fmt.Printf("\n⚠️  %d agent(s) still active. Use 'orch status' to monitor.\n", activeCount)
	}

	// Show checkpoint advice based on session duration
	if duration >= session.CheckpointMaxDuration {
		fmt.Println("\n⛔ Session exceeded 4h checkpoint max.")
		fmt.Println("   Consider shorter sessions to maintain quality.")
	} else if duration >= session.CheckpointStrongDuration {
		fmt.Println("\n🟡 Session was 3h+. Good to hand off, but review quality of late work.")
	}

	return nil
}
