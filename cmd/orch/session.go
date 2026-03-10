// Package main provides the CLI entry point for orch-go.
package main

import (
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
	sessionJSON        bool
	resumeForInjection bool
	resumeCheck        bool
	validateJSON       bool
)

func init() {
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionEndCmd)
	sessionCmd.AddCommand(sessionResumeCmd)
	sessionCmd.AddCommand(sessionMigrateCmd)
	sessionCmd.AddCommand(sessionValidateCmd)
	sessionCmd.AddCommand(sessionLabelCmd)

	// Add --json flag to status command
	sessionStatusCmd.Flags().BoolVar(&sessionJSON, "json", false, "Output as JSON")

	// Add flags for resume command
	sessionResumeCmd.Flags().BoolVar(&resumeForInjection, "for-injection", false, "Output condensed format for hook injection")
	sessionResumeCmd.Flags().BoolVar(&resumeCheck, "check", false, "Check if handoff exists (exit code only)")

	// Add --json flag for validate command
	sessionValidateCmd.Flags().BoolVar(&validateJSON, "json", false, "Output as JSON")

	rootCmd.AddCommand(sessionCmd)
}

// ============================================================================
// Session Resume Command
// ============================================================================

var sessionResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume orchestrator session by injecting prior handoff",
	Long: `Resume an orchestrator session by discovering and displaying the most recent SESSION_HANDOFF.md.

This command walks up the directory tree to find .orch/session/latest/SESSION_HANDOFF.md
and displays it in the format appropriate for the use case.

Modes:
  Default (interactive):  Display formatted handoff for manual review
  --for-injection:        Output condensed format for hook injection (no decorations)
  --check:                Just check if handoff exists (exit code 0 if yes, 1 if no)

Discovery:
  1. Starts from current directory
  2. Walks up directory tree looking for .orch/session/latest symlink
  3. Reads SESSION_HANDOFF.md from the symlink target
  4. Fails gracefully if no handoff found (valid for fresh sessions)

Examples:
  orch session resume                  # Interactive display
  orch session resume --for-injection  # For hooks (condensed format)
  orch session resume --check          # Check existence only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionResume()
	},
}

// ============================================================================
// Session Validate Command
// ============================================================================

var sessionValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Show unfilled handoff sections without ending session",
	Long: `Validate SESSION_HANDOFF.md quality by showing unfilled sections.

This command checks the active session handoff for placeholder patterns
and displays which sections still need to be filled. Unlike 'session end',
it does NOT prompt for input or archive the handoff.

Use cases:
- Check handoff quality mid-session
- Debug validation logic
- Verify handoff is ready before ending session

The command looks for the active session handoff in:
  .orch/session/{window-name}/active/SESSION_HANDOFF.md

If no active handoff exists, it reports that state.

Examples:
  orch session validate          # Human-readable output
  orch session validate --json   # Machine-readable output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionValidate()
	},
}
