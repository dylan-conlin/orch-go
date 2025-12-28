package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// learnCmd delegates to kb learn (the learning loop has moved to kb-cli).
// This wrapper is kept for backward compatibility.
var learnCmd = &cobra.Command{
	Use:   "learn",
	Short: "Review and act on system learning suggestions (delegates to kb learn)",
	Long: `Review recurring context gaps and get suggestions for improvement.

NOTE: This command now delegates to 'kb learn'. The learning loop has moved
to kb-cli as part of Phase 1 consolidation. Use 'kb learn' directly for
new usage.

Examples:
  kb learn                   # Show suggestions
  kb learn patterns          # Analyze gap patterns
  kb learn act 1             # Run first suggestion's command
  orch learn                 # (deprecated) Same as kb learn`,
	DisableFlagParsing: true, // Pass all args through to kb
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKbLearn(args)
	},
}

func init() {
	rootCmd.AddCommand(learnCmd)
}

// runKbLearn delegates to kb learn command.
func runKbLearn(args []string) error {
	// Build command: kb learn [args...]
	cmdArgs := append([]string{"learn"}, args...)

	kbCmd := exec.Command("kb", cmdArgs...)
	kbCmd.Stdout = os.Stdout
	kbCmd.Stderr = os.Stderr
	kbCmd.Stdin = os.Stdin

	err := kbCmd.Run()
	if err != nil {
		// Check if kb is not installed
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr
		}
		return fmt.Errorf("failed to run kb learn: %w (is kb installed?)", err)
	}

	return nil
}
