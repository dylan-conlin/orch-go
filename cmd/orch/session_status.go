package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/spf13/cobra"
)

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
	Active        bool                      `json:"active"`
	Goal          string                    `json:"goal,omitempty"`
	StartedAt     string                    `json:"started_at,omitempty"`
	Duration      string                    `json:"duration,omitempty"`
	WorkspacePath string                    `json:"workspace_path,omitempty"`
	Spawns        []session.SpawnStatus     `json:"spawns,omitempty"`
	Counts        *SpawnCounts              `json:"counts,omitempty"`
	Checkpoint    *session.CheckpointStatus `json:"checkpoint,omitempty"`
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
		output.WorkspacePath = sess.WorkspacePath

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

		// Get checkpoint status using orchestrator thresholds
		// (orch session is for orchestrator sessions, not agent sessions)
		output.Checkpoint = store.GetCheckpointStatusWithType(session.SessionTypeOrchestrator)
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
	fmt.Printf("  Goal:      %s\n", output.Goal)
	fmt.Printf("  Duration:  %s", output.Duration)

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

	if output.WorkspacePath != "" {
		fmt.Printf("  Workspace: %s\n", output.WorkspacePath)
	}

	fmt.Printf("  Spawns:    %d total", output.Counts.Total)
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
