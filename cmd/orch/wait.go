// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Wait command flags
	waitPhase    string
	waitTimeout  string
	waitInterval int
	waitQuiet    bool
)

var waitCmd = &cobra.Command{
	Use:   "wait [beads-id]",
	Short: "Block until agent reaches specified phase",
	Long: `Block until an agent reaches a specified phase, polling at regular intervals.

Replaces manual 'sleep X && orch-go check' loops with cleaner workflow.
Useful for scripting and automation.

Examples:
  orch-go wait proj-123                    # Wait for Complete (default)
  orch-go wait proj-123 --phase Complete   # Explicit phase
  orch-go wait proj-123 --timeout 5m       # 5 minute timeout
  orch-go wait proj-123 -q                 # Quiet mode (no progress)

Exit codes:
  0 - Agent reached target phase
  1 - Timeout reached
  2 - Error (agent not found, invalid args)

Timeout format:
  30s  - 30 seconds
  5m   - 5 minutes
  1h   - 1 hour
  1h30m - 1 hour 30 minutes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runWait(beadsID)
	},
}

func init() {
	waitCmd.Flags().StringVar(&waitPhase, "phase", "Complete", "Target phase to wait for")
	waitCmd.Flags().StringVar(&waitTimeout, "timeout", "30m", "Timeout duration (e.g., 30s, 5m, 1h)")
	waitCmd.Flags().IntVar(&waitInterval, "interval", 5, "Poll interval in seconds")
	waitCmd.Flags().BoolVarP(&waitQuiet, "quiet", "q", false, "Suppress progress output")
}

// parseTimeout parses a timeout string like '30s', '5m', '1h', '1h30m' to time.Duration.
// Returns error if format is invalid or duration is zero.
func parseTimeout(timeout string) (time.Duration, error) {
	if timeout == "" {
		return 0, fmt.Errorf("empty timeout")
	}

	// Try to parse as integer (seconds)
	if n, err := strconv.Atoi(timeout); err == nil {
		if n <= 0 {
			return 0, fmt.Errorf("timeout must be positive")
		}
		return time.Duration(n) * time.Second, nil
	}

	// Parse duration components
	var total time.Duration
	pattern := regexp.MustCompile(`(\d+)([smhSMH])`)
	matches := pattern.FindAllStringSubmatch(strings.ToLower(timeout), -1)

	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid timeout format: %s", timeout)
	}

	for _, match := range matches {
		value, _ := strconv.Atoi(match[1])
		unit := match[2]

		switch unit {
		case "s":
			total += time.Duration(value) * time.Second
		case "m":
			total += time.Duration(value) * time.Minute
		case "h":
			total += time.Duration(value) * time.Hour
		}
	}

	if total <= 0 {
		return 0, fmt.Errorf("timeout must be positive")
	}

	return total, nil
}

// formatDuration formats a duration to human-readable string.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		minutes := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs > 0 {
			return fmt.Sprintf("%dm %ds", minutes, secs)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}

func runWait(beadsID string) error {
	// Start timing
	startTime := time.Now()

	// Parse timeout
	timeout, err := parseTimeout(waitTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Use formats like: 30s, 5m, 1h, 1h30m")
		os.Exit(2)
	}

	// Verify issue exists
	_, err = verify.GetIssue(beadsID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	// Initial status message
	if !waitQuiet {
		fmt.Printf("Waiting for agent '%s' to reach phase '%s'...\n", beadsID, waitPhase)
		fmt.Printf("   Timeout: %s, Poll interval: %ds\n", waitTimeout, waitInterval)
	}

	// Polling loop
	var lastPhase string
	for {
		// Check phase status
		status, err := verify.GetPhaseStatus(beadsID)
		if err != nil {
			// Non-fatal: issue might not have comments yet
			if !waitQuiet {
				fmt.Printf("   Current phase: (no phase reported yet)\n")
			}
		} else {
			currentPhase := status.Phase
			if currentPhase == "" {
				currentPhase = "(no phase)"
			}

			// Log phase changes
			if currentPhase != lastPhase {
				if !waitQuiet {
					fmt.Printf("   Current phase: %s\n", currentPhase)
				}
				lastPhase = currentPhase
			}

			// Check if target phase reached (case-insensitive partial match)
			if status.Found && strings.Contains(strings.ToLower(currentPhase), strings.ToLower(waitPhase)) {
				// Success!
				elapsed := time.Since(startTime)

				// Log the successful wait
				logger := events.NewLogger(events.DefaultLogPath())
				event := events.Event{
					Type:      "agent.wait.complete",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"beads_id":     beadsID,
						"target_phase": waitPhase,
						"final_phase":  currentPhase,
						"elapsed_ms":   elapsed.Milliseconds(),
						"success":      true,
					},
				}
				if logErr := logger.Log(event); logErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", logErr)
				}

				if !waitQuiet {
					fmt.Printf("Agent '%s' reached phase '%s' after %s\n", beadsID, currentPhase, formatDuration(elapsed))
				}

				os.Exit(0)
			}
		}

		// Check timeout
		elapsed := time.Since(startTime)
		if elapsed >= timeout {
			// Log the timeout
			logger := events.NewLogger(events.DefaultLogPath())
			event := events.Event{
				Type:      "agent.wait.timeout",
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"beads_id":        beadsID,
					"target_phase":    waitPhase,
					"final_phase":     lastPhase,
					"elapsed_ms":      elapsed.Milliseconds(),
					"timeout_seconds": timeout.Seconds(),
				},
			}
			if logErr := logger.Log(event); logErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", logErr)
			}

			if !waitQuiet {
				fmt.Fprintf(os.Stderr, "Timeout after %s\n", formatDuration(elapsed))
				fmt.Fprintf(os.Stderr, "Agent '%s' is still at phase '%s'\n", beadsID, lastPhase)
			}

			os.Exit(1)
		}

		// Wait before next poll
		time.Sleep(time.Duration(waitInterval) * time.Second)
	}
}
