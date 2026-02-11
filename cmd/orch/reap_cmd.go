// Package main provides the reap command for killing orphaned agent processes.
// This is a standalone process reaper that works independently of the daemon,
// designed to prevent zombie bun processes from accumulating and crashing the system.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/process"
	"github.com/spf13/cobra"
)

var (
	reapDryRun bool
	reapForce  bool // Kill all agent processes regardless of session status
)

var reapCmd = &cobra.Command{
	Use:   "reap",
	Short: "Kill orphaned bun agent processes",
	Long: `Find and kill bun agent processes that are no longer associated with
any active OpenCode session. This prevents zombie processes from accumulating
and consuming all system memory.

This command works independently of the daemon - you can run it manually or
via cron/launchd as a safety net.

Behavior:
  - Queries OpenCode API for active sessions
  - Finds all bun agent processes (src/index.ts, not serve --port)
  - Kills processes not matched to any active session
  - If OpenCode API is unavailable, uses --force to kill all agent processes

Examples:
  orch reap              # Kill orphaned agent processes
  orch reap --dry-run    # Preview what would be killed
  orch reap --force      # Kill ALL agent processes (use after reboot or crash)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReap(serverURL, reapDryRun, reapForce)
	},
}

func init() {
	reapCmd.Flags().BoolVar(&reapDryRun, "dry-run", false, "Preview what would be killed without killing")
	reapCmd.Flags().BoolVar(&reapForce, "force", false, "Kill ALL agent processes regardless of session status")
}

func runReap(serverURL string, dryRun, force bool) error {
	// Step 1: Find all agent processes
	agents, err := process.FindAgentProcesses()
	if err != nil {
		return fmt.Errorf("failed to find agent processes: %w", err)
	}

	if len(agents) == 0 {
		fmt.Println("No agent processes found")
		return nil
	}

	fmt.Printf("Found %d agent process(es)\n", len(agents))

	// Step 2: Determine which are orphans
	var toKill []process.OrphanProcess

	if force {
		fmt.Println("Force mode: killing ALL agent processes")
		toKill = agents
	} else {
		// Try to query OpenCode for active sessions
		client := opencode.NewClient(serverURL)
		sessions, apiErr := client.ListSessions("")
		if apiErr != nil {
			// API unavailable — we can't determine what's active
			fmt.Fprintf(os.Stderr, "⚠️  OpenCode API unavailable: %v\n", apiErr)
			fmt.Fprintf(os.Stderr, "   Cannot determine which processes are active.\n")
			fmt.Fprintf(os.Stderr, "   Use --force to kill all agent processes.\n")

			// Still do ledger-based cleanup as a fallback
			ledger := process.NewDefaultLedger()
			entries, ledgerErr := ledger.ReadAll()
			if ledgerErr == nil && len(entries) > 0 {
				fmt.Printf("\nLedger has %d entries, sweeping stale...\n", len(entries))
				sweepResult := ledger.Sweep()
				if sweepResult.StaleRemoved > 0 {
					fmt.Printf("Removed %d stale ledger entries (dead processes)\n", sweepResult.StaleRemoved)
				}
			}

			return nil
		}

		// Build active session maps
		activeIDs := make(map[string]bool, len(sessions))
		activeTitles := make(map[string]bool)
		for _, s := range sessions {
			if s.ID != "" {
				activeIDs[s.ID] = true
			}
			if s.Title != "" {
				activeTitles[s.Title] = true
			}
		}

		fmt.Printf("OpenCode reports %d active session(s)\n", len(sessions))

		// Find orphans (processes not in any active session)
		orphans, orphanErr := process.FindOrphanProcesses(activeTitles, activeIDs)
		if orphanErr != nil {
			return fmt.Errorf("failed to find orphan processes: %w", orphanErr)
		}

		toKill = orphans
	}

	if len(toKill) == 0 {
		fmt.Println("No orphaned agent processes found")
		return nil
	}

	fmt.Printf("\n%d orphaned process(es) to kill:\n", len(toKill))

	killed := 0
	for _, agent := range toKill {
		label := formatAgentLabel(agent)
		if dryRun {
			fmt.Printf("  [DRY-RUN] Would kill: PID %d %s\n", agent.PID, label)
			killed++
			continue
		}
		if process.Terminate(agent.PID, "bun (reap)") {
			fmt.Printf("  Killed: PID %d %s\n", agent.PID, label)
			killed++
		} else {
			fmt.Printf("  Already dead: PID %d %s\n", agent.PID, label)
		}
	}

	// Also sweep the process ledger
	ledger := process.NewDefaultLedger()
	sweepResult := ledger.Sweep()
	if sweepResult.StaleRemoved > 0 {
		fmt.Printf("\nSwept %d stale ledger entries\n", sweepResult.StaleRemoved)
	}

	verb := "Killed"
	if dryRun {
		verb = "Would kill"
	}
	fmt.Printf("\n%s %d/%d orphaned process(es)\n", verb, killed, len(toKill))

	if killed > 0 && !dryRun {
		// Log memory recovery hint
		fmt.Println("\n💡 Memory freed. Run 'orch status' to check system health.")
	}

	return nil
}

func formatAgentLabel(agent process.OrphanProcess) string {
	parts := []string{}
	if agent.WorkspaceName != "" {
		parts = append(parts, agent.WorkspaceName)
	}
	if agent.BeadsID != "" {
		parts = append(parts, fmt.Sprintf("[%s]", agent.BeadsID))
	}
	if agent.SessionID != "" {
		parts = append(parts, fmt.Sprintf("session=%s", agent.SessionID))
	}

	// For headless processes with no metadata, show process age
	if len(parts) == 0 {
		startTime, err := process.ProcessStartTime(agent.PID)
		if err == nil {
			age := time.Since(startTime).Truncate(time.Minute)
			parts = append(parts, fmt.Sprintf("(age: %s, no session metadata)", age))
		} else {
			parts = append(parts, "(no session metadata)")
		}
	}

	if len(parts) == 0 {
		return "(unknown)"
	}
	return fmt.Sprintf("(%s)", joinNonEmpty(parts, " "))
}

func joinNonEmpty(parts []string, sep string) string {
	result := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if result != "" {
			result += sep
		}
		result += p
	}
	return result
}
