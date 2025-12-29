// Package main provides the CLI entry point for orch-go.
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/action"
	"github.com/spf13/cobra"
)

// ============================================================================
// Action Command - Log and query action outcomes
// ============================================================================

var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "Log and query action outcomes for behavioral pattern detection",
	Long: `Log and query action outcomes for behavioral pattern detection.

This command enables hooks to log tool outcomes (success/empty/error) which
can then be analyzed to detect behavioral patterns like repeated futile actions.

The patterns can be surfaced via 'orch patterns' to help orchestrators avoid
blind respawning and identify systemic issues.

Examples:
  orch action log --tool Read --target "/path/to/file" --outcome empty
  orch action log --tool Bash --target "orch status" --outcome success
  orch action log --tool Read --target "/missing/file" --outcome error --error "file not found"
  orch action summary        # Show action log summary
  orch action prune --days 7 # Remove old entries`,
}

var actionLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Log an action outcome",
	Long: `Log an action outcome for pattern detection.

This is typically called by PostToolUse hooks to record tool outcomes.
The logged events are analyzed by 'orch patterns' to detect behavioral patterns.

Outcomes:
  success  - Action completed successfully with expected result
  empty    - Action succeeded but returned empty/no result
  error    - Action failed with an error
  fallback - Action required a fallback approach`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runActionLog()
	},
}

var actionSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show action log summary",
	Long:  `Display a summary of the action log including event count and detected patterns.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runActionSummary()
	},
}

var actionPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old action log entries",
	Long:  `Remove action log entries older than the specified number of days.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runActionPrune()
	},
}

var (
	// Log command flags
	actionTool       string
	actionTarget     string
	actionOutcome    string
	actionError      string
	actionFallback   string
	actionSessionID  string
	actionWorkspace  string
	actionContext    string
	actionJSON       bool

	// Prune command flags
	actionPruneDays int
)

func init() {
	// Log command flags
	actionLogCmd.Flags().StringVar(&actionTool, "tool", "", "Tool name (e.g., Read, Bash, Glob)")
	actionLogCmd.Flags().StringVar(&actionTarget, "target", "", "What the tool acted on (file path, command, etc.)")
	actionLogCmd.Flags().StringVar(&actionOutcome, "outcome", "", "Outcome: success, empty, error, fallback")
	actionLogCmd.Flags().StringVar(&actionError, "error", "", "Error message (when outcome is error)")
	actionLogCmd.Flags().StringVar(&actionFallback, "fallback", "", "Fallback action taken (when outcome is fallback)")
	actionLogCmd.Flags().StringVar(&actionSessionID, "session", "", "OpenCode session ID")
	actionLogCmd.Flags().StringVar(&actionWorkspace, "workspace", "", "Workspace name")
	actionLogCmd.Flags().StringVar(&actionContext, "context", "", "Additional context about the action")
	actionLogCmd.Flags().BoolVar(&actionJSON, "json", false, "Output result as JSON")

	// Required flags for log
	actionLogCmd.MarkFlagRequired("tool")
	actionLogCmd.MarkFlagRequired("target")
	actionLogCmd.MarkFlagRequired("outcome")

	// Prune command flags
	actionPruneCmd.Flags().IntVar(&actionPruneDays, "days", 7, "Remove entries older than this many days")

	// Add subcommands
	actionCmd.AddCommand(actionLogCmd)
	actionCmd.AddCommand(actionSummaryCmd)
	actionCmd.AddCommand(actionPruneCmd)
}

func runActionLog() error {
	// Validate outcome
	var outcome action.Outcome
	switch strings.ToLower(actionOutcome) {
	case "success":
		outcome = action.OutcomeSuccess
	case "empty":
		outcome = action.OutcomeEmpty
	case "error":
		outcome = action.OutcomeError
	case "fallback":
		outcome = action.OutcomeFallback
	default:
		return fmt.Errorf("invalid outcome %q: must be success, empty, error, or fallback", actionOutcome)
	}

	// Build event
	event := action.ActionEvent{
		Timestamp:      time.Now().UTC(),
		Tool:           actionTool,
		Target:         actionTarget,
		Outcome:        outcome,
		ErrorMessage:   actionError,
		FallbackAction: actionFallback,
		SessionID:      actionSessionID,
		Workspace:      actionWorkspace,
		Context:        actionContext,
	}

	// Log the event
	logger := action.NewDefaultLogger()
	if err := logger.Log(event); err != nil {
		return fmt.Errorf("failed to log action: %w", err)
	}

	// Output result
	if actionJSON {
		result := map[string]interface{}{
			"logged": true,
			"event":  event,
		}
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(data))
	} else {
		// Silent success for hook use - no output unless error
		// This keeps hooks fast and prevents noise in tool output
	}

	return nil
}

func runActionSummary() error {
	tracker, err := action.LoadTracker("")
	if err != nil {
		return fmt.Errorf("failed to load action tracker: %w", err)
	}

	patterns := tracker.FindPatterns()

	fmt.Println("Action Log Summary")
	fmt.Println("==================")
	fmt.Printf("Total events: %d\n", len(tracker.Events))
	fmt.Printf("Detected patterns: %d\n", len(patterns))

	if len(patterns) > 0 {
		fmt.Println("\nTop patterns:")
		for i, p := range patterns {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. [%s] %s → %s (%dx)\n", i+1, p.Outcome, p.Tool, p.Target, p.Count)
		}
		fmt.Println("\nRun 'orch patterns' for detailed analysis.")
	}

	return nil
}

func runActionPrune() error {
	maxAge := time.Duration(actionPruneDays) * 24 * time.Hour

	pruned, err := action.Prune("", maxAge)
	if err != nil {
		return fmt.Errorf("failed to prune action log: %w", err)
	}

	if pruned == 0 {
		fmt.Printf("No events older than %d days to prune\n", actionPruneDays)
	} else {
		fmt.Printf("Pruned %d events older than %d days\n", pruned, actionPruneDays)
	}

	return nil
}


