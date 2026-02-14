package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/spf13/cobra"
)

var controlCmd = &cobra.Command{
	Use:   "control",
	Short: "Manage control plane (circuit breakers, protected paths)",
	Long: `Control plane management commands.

The control plane is an immutable layer at ~/.orch/ that agents cannot modify.
It provides circuit breakers (commit limits, fix:feat ratios) and protected
path monitoring to prevent entropy spirals.

Commands:
  status  - Show control plane state and metrics
  resume  - Clear halt sentinel and resume daemon
  init    - Create default control-plane.conf`,
}

var controlStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show control plane state and recent metrics",
	Long: `Show control plane configuration, halt status, and recent metrics.

Displays:
- Configuration location and thresholds
- Halt status (if circuit breaker fired)
- Today's commit count
- Protected path violations
- Fix:feat ratio (7-day rolling window)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		status, err := control.Status()
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		fmt.Println("Control Plane Status")
		fmt.Printf("  Config:      %s\n", control.ConfigPath)

		if !status.ConfigExists {
			fmt.Println("  State:       NOT INITIALIZED")
			fmt.Println()
			fmt.Println("Run 'orch control init' to create default configuration.")
			return nil
		}

		// Halt status
		if status.Halted {
			if status.HaltInfo != nil {
				fmt.Printf("  State:       HALTED (since %s)\n", status.HaltInfo.TriggeredAt.Format("2006-01-02 15:04 MST"))
				fmt.Printf("  Reason:      %s\n", status.HaltInfo.Reason)
			} else {
				fmt.Println("  State:       HALTED")
			}
		} else {
			fmt.Println("  State:       ACTIVE")
		}
		fmt.Println()

		// Thresholds
		cfg := status.Config
		if cfg != nil {
			fmt.Println("Thresholds:")

			// Commits per day
			if cfg.MaxCommitsPerDay > 0 {
				fmt.Printf("  Commits/day:     %d", cfg.MaxCommitsPerDay)
				if status.MetricsExist {
					fmt.Printf(" (current: %d)", status.CommitsToday)
					if status.CommitsToday >= cfg.MaxCommitsPerDay {
						fmt.Print(" ← EXCEEDED")
					}
				}
				fmt.Println()
			}

			// Fix:feat ratio
			if cfg.FixFeatRatioThreshold > 0 {
				fmt.Printf("  Fix:feat ratio:  %d%%", cfg.FixFeatRatioThreshold)
				if status.FixFeatRatio != "" {
					fmt.Printf(" (current: %s)", status.FixFeatRatio)
				}
				fmt.Println()
			}

			// Churn ratio
			if cfg.ChurnRatioThreshold > 0 {
				fmt.Printf("  Churn ratio:     %d%%\n", cfg.ChurnRatioThreshold)
			}

			// Protected paths
			if len(cfg.ProtectedPaths) > 0 {
				fmt.Printf("  Protected paths: %s\n", strings.Join(cfg.ProtectedPaths, " "))
			}
			fmt.Println()
		}

		// Today's metrics
		if status.MetricsExist {
			fmt.Println("Today's Metrics:")
			fmt.Printf("  Commits: %d\n", status.CommitsToday)

			if len(status.Violations) > 0 {
				fmt.Printf("  Protected path violations: %d\n", len(status.Violations))
				for _, v := range status.Violations {
					fmt.Printf("    %s\n", v)
				}
			} else {
				fmt.Println("  Protected path violations: 0")
			}
			fmt.Println()
		}

		// Action prompt
		if status.Halted {
			fmt.Println("Run 'orch control resume' to clear halt and continue.")
		}

		return nil
	},
}

var controlResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Clear halt sentinel and resume daemon",
	Long: `Clear the halt sentinel file and resume daemon operations.

The daemon checks for the halt file at the top of each poll cycle.
Once cleared, the daemon will resume spawning agents on the next cycle
(within 60 seconds).

This command also displays current metrics as confirmation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if halted
		halted, reason := control.CheckHalt()
		if !halted {
			fmt.Println("Control plane is not halted.")
			return nil
		}

		fmt.Printf("Halted: %s\n", reason)
		fmt.Println()

		// Get current metrics before clearing
		status, _ := control.Status()
		if status != nil && status.MetricsExist {
			fmt.Println("Current Metrics:")
			fmt.Printf("  Commits today: %d\n", status.CommitsToday)
			if status.FixFeatRatio != "" {
				fmt.Printf("  Fix:feat ratio: %s\n", status.FixFeatRatio)
			}
			fmt.Println()
		}

		// Clear halt
		if err := control.ClearHalt(); err != nil {
			return fmt.Errorf("failed to clear halt: %w", err)
		}

		fmt.Println("✓ Halt cleared. Daemon will resume on next poll cycle (within 60s).")
		return nil
	},
}

var controlInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default control-plane.conf",
	Long: `Create ~/.orch/control-plane.conf with default thresholds.

Default configuration:
- MAX_COMMITS_PER_DAY=20
- FIX_FEAT_RATIO_THRESHOLD=50 (0.5:1 fix:feat)
- CHURN_RATIO_THRESHOLD=200 (2:1 created+deleted/net)
- PROTECTED_PATHS="cmd/orch/ pkg/daemon/ pkg/spawn/ pkg/verify/ plugins/"
- COOLDOWN_MINUTES=30

After creating the config, you must install the post-commit hook manually:

Add this line to .git/hooks/post-commit:
  [ -x "$HOME/.orch/hooks/control-plane-post-commit.sh" ] && "$HOME/.orch/hooks/control-plane-post-commit.sh" || true

Then create the hook script:
  cp ~/.orch/templates/control-plane-post-commit.sh ~/.orch/hooks/
  chmod +x ~/.orch/hooks/control-plane-post-commit.sh`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already exists
		if _, err := os.Stat(control.ConfigPath); err == nil {
			return fmt.Errorf("config already exists: %s", control.ConfigPath)
		}

		// Create config
		if err := control.InitConfig(); err != nil {
			return err
		}

		fmt.Printf("✓ Created: %s\n", control.ConfigPath)
		fmt.Println()
		fmt.Println("Default thresholds:")
		fmt.Println("  MAX_COMMITS_PER_DAY=20")
		fmt.Println("  FIX_FEAT_RATIO_THRESHOLD=50")
		fmt.Println("  PROTECTED_PATHS=\"cmd/orch/ pkg/daemon/ pkg/spawn/ pkg/verify/ plugins/\"")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Review and adjust thresholds in the config file")
		fmt.Println("2. Install the post-commit hook (see 'orch control init --help')")

		return nil
	},
}

func init() {
	controlCmd.AddCommand(controlStatusCmd)
	controlCmd.AddCommand(controlResumeCmd)
	controlCmd.AddCommand(controlInitCmd)
}
