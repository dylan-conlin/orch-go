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

			// Rolling average
			if cfg.RollingAvgHalt > 0 {
				fmt.Printf("  Rolling avg:     warn=%d halt=%d (%dd window)",
					cfg.RollingAvgWarn, cfg.RollingAvgHalt, cfg.RollingWindowDays)
				if status.RollingDays > 0 {
					fmt.Printf(" (current: %d/day over %dd)", status.RollingAvg, status.RollingDays)
					if status.RollingAvg >= cfg.RollingAvgHalt {
						fmt.Print(" ← EXCEEDED")
					} else if status.RollingAvg >= cfg.RollingAvgWarn {
						fmt.Print(" ← WARNING")
					}
				}
				fmt.Println()
			}

			// Unverified velocity
			if cfg.MaxUnverifiedDays > 0 {
				fmt.Printf("  Unverified:      halt after %dd without ack (if commits/day > %d)",
					cfg.MaxUnverifiedDays, cfg.UnverifiedDailyMin)
				if status.HeartbeatAge >= 0 {
					fmt.Printf(" (heartbeat: %dd ago)", status.HeartbeatAge)
					if status.HeartbeatAge >= cfg.MaxUnverifiedDays && status.CommitsToday >= cfg.UnverifiedDailyMin {
						fmt.Print(" ← EXCEEDED")
					}
				} else {
					fmt.Print(" (no heartbeat — run 'orch control ack')")
				}
				fmt.Println()
			}

			// Hard cap
			if cfg.DailyHardCap > 0 {
				fmt.Printf("  Hard cap:        %d/day", cfg.DailyHardCap)
				if status.MetricsExist {
					fmt.Printf(" (current: %d)", status.CommitsToday)
					if status.CommitsToday >= cfg.DailyHardCap {
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
			if status.RollingDays > 0 {
				fmt.Printf("  Rolling avg: %d/day (%dd)\n", status.RollingAvg, status.RollingDays)
			}
			if status.HeartbeatAge >= 0 {
				fmt.Printf("  Heartbeat: %dd ago\n", status.HeartbeatAge)
			} else {
				fmt.Println("  Heartbeat: never (run 'orch control ack')")
			}

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

		// Clear halt and touch heartbeat
		if err := control.ClearHalt(); err != nil {
			return fmt.Errorf("failed to clear halt: %w", err)
		}
		if err := control.Ack(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to touch heartbeat: %v\n", err)
		}

		fmt.Println("✓ Halt cleared + heartbeat refreshed. Daemon will resume on next poll cycle (within 60s).")
		return nil
	},
}

var controlAckCmd = &cobra.Command{
	Use:   "ack",
	Short: "Acknowledge system health (touch heartbeat)",
	Long: `Touch the heartbeat file to signal that a human is actively monitoring.

The control plane tracks how long since the last human acknowledgment.
If agents continue committing without human ack for MAX_UNVERIFIED_DAYS
(default: 2 days), and daily commits exceed UNVERIFIED_DAILY_MIN
(default: 15), the circuit breaker halts the daemon.

Run this periodically when the system is operating normally:
  orch control ack

The 'resume' command also implicitly acks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := control.Ack(); err != nil {
			return fmt.Errorf("failed to ack: %w", err)
		}

		fmt.Println("✓ Heartbeat refreshed.")
		fmt.Println()

		// Show current metrics
		status, err := control.Status()
		if err != nil {
			return nil // ack succeeded, status is nice-to-have
		}

		if status.MetricsExist {
			fmt.Printf("  Commits today: %d\n", status.CommitsToday)
			if status.RollingDays > 0 {
				fmt.Printf("  Rolling avg: %d/day (%dd)\n", status.RollingAvg, status.RollingDays)
			}
			if status.FixFeatRatio != "" {
				fmt.Printf("  Fix:feat ratio: %s\n", status.FixFeatRatio)
			}
		}

		return nil
	},
}

var controlInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default control-plane.conf",
	Long: `Create ~/.orch/control-plane.conf with v2 defaults.

Three-layer circuit breaker:
1. Rolling average:  warn at 50/day, halt at 70/day (3-day window)
2. Unverified velocity: halt after 2 days without 'orch control ack' + >15 commits/day
3. Hard cap: 150 commits/day absolute maximum

Other thresholds:
- FIX_FEAT_RATIO_THRESHOLD=50 (0.5:1 fix:feat)
- PROTECTED_PATHS="cmd/orch/ pkg/daemon/ pkg/spawn/ pkg/verify/ plugins/"

After creating the config, install the post-commit hook:
  [ -x "$HOME/.orch/hooks/control-plane-post-commit.sh" ] && "$HOME/.orch/hooks/control-plane-post-commit.sh" || true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(control.ConfigPath); err == nil {
			return fmt.Errorf("config already exists: %s", control.ConfigPath)
		}

		if err := control.InitConfig(); err != nil {
			return err
		}

		fmt.Printf("✓ Created: %s\n", control.ConfigPath)
		fmt.Println()
		fmt.Println("Default thresholds (v2):")
		fmt.Println("  Rolling avg:     warn=50 halt=70 (3-day window)")
		fmt.Println("  Unverified:      halt after 2d without ack + >15 commits/day")
		fmt.Println("  Hard cap:        150/day")
		fmt.Println("  Fix:feat ratio:  50%")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Review and adjust thresholds in the config file")
		fmt.Println("2. Install the post-commit hook (see 'orch control init --help')")
		fmt.Println("3. Run 'orch control ack' to initialize heartbeat")

		return nil
	},
}

func init() {
	controlCmd.AddCommand(controlStatusCmd)
	controlCmd.AddCommand(controlResumeCmd)
	controlCmd.AddCommand(controlAckCmd)
	controlCmd.AddCommand(controlInitCmd)
}
