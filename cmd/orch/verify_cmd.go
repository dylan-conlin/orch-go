package main

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verification commands for two-gate verification system",
	Long: `Verification commands for the verifiability-first hard constraint.

The two-gate verification system requires:
  Gate 1 (Comprehension): Dylan explains what was built via --explain flag
  Gate 2 (Behavioral): Dylan verifies behavior via --verified flag

Commands:
  heartbeat - Update verification heartbeat timestamp (gate2 activity signal)`,
}

var verifyHeartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "Update verification heartbeat timestamp",
	Long: `Update the verification heartbeat file with current timestamp.

This signals that human verification activity has occurred. The daemon reads
this heartbeat before spawning agents - if the heartbeat is stale (>24h old),
autonomous spawning is halted to prevent multi-day autonomous drift.

The heartbeat is automatically updated when running:
  - orch complete --explain "..." (gate1: comprehension)

But can be manually updated with this command for gate2 (behavioral verification)
activities that don't go through orch complete.

Examples:
  orch verify heartbeat              # Update heartbeat timestamp
  orch verify heartbeat && orch daemon resume  # Resume after heartbeat update`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := control.Ack(); err != nil {
			return fmt.Errorf("failed to update heartbeat: %w", err)
		}

		age := control.HeartbeatAgeHours()
		fmt.Println("✓ Verification heartbeat updated")
		fmt.Printf("  Path: %s\n", control.HeartbeatPath)
		fmt.Printf("  Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))

		// Show status
		if age >= 0 {
			fmt.Printf("  Previous age: %.1f hours ago\n", age)
		}

		fmt.Println()
		fmt.Println("Daemon will resume spawning on next poll cycle (within 15s).")

		return nil
	},
}

func init() {
	verifyCmd.AddCommand(verifyHeartbeatCmd)
}
