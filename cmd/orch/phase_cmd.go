// Package main provides the phase command for direct agent phase reporting via SQLite.
//
// Agents call `orch phase <beads_id> <phase> [summary]` which:
//  1. Writes directly to ~/.orch/state.db (~1ms) for runtime visibility
//  2. Fires a background bd comment for permanent audit trail in beads
//
// This means agents only need ONE command (`orch phase`) to get both
// fast runtime state AND searchable audit history. The bd comment is
// non-blocking — if it fails, the phase is still recorded in SQLite.
//
// See: .kb/investigations/2026-02-06-design-single-source-agent-state.md (Fork 1)
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/spf13/cobra"
)

var phaseSkipComment bool

var phaseCmd = &cobra.Command{
	Use:   "phase <beads_id> <phase> [summary...]",
	Short: "Report agent phase transition (SQLite + bd comment)",
	Long: `Report an agent's phase transition to the state database and beads.

Writes phase directly to SQLite (~1ms) for immediate visibility in
orch status and the dashboard, then fires a background bd comment
for permanent searchable audit trail in beads.

Agents only need this ONE command for both runtime speed and audit.

Use --no-comment to skip the bd comment (e.g., in tests or when
beads is unavailable).

Examples:
  orch phase orch-go-12345 Planning "Analyzing codebase structure"
  orch phase orch-go-12345 Implementing "Building SQLite schema"
  orch phase orch-go-12345 Complete "All tests passing, ready for review"
  orch phase orch-go-12345 BLOCKED "Need clarification on API contract"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		phase := args[1]
		summary := ""
		if len(args) > 2 {
			summary = strings.Join(args[2:], " ")
		}
		return runPhase(beadsID, phase, summary)
	},
}

func init() {
	phaseCmd.Flags().BoolVar(&phaseSkipComment, "no-comment", false, "Skip bd comment (SQLite only)")
	rootCmd.AddCommand(phaseCmd)
}

func runPhase(beadsID, phase, summary string) error {
	return runPhaseWithDB(beadsID, phase, summary, "", !phaseSkipComment)
}

// formatPhaseComment formats a phase transition as a bd comment string.
// Follows the established "Phase: X - summary" convention that orch complete parses.
func formatPhaseComment(phase, summary string) string {
	switch phase {
	case "BLOCKED":
		if summary != "" {
			return fmt.Sprintf("BLOCKED: %s", summary)
		}
		return "BLOCKED"
	case "QUESTION":
		if summary != "" {
			return fmt.Sprintf("QUESTION: %s", summary)
		}
		return "QUESTION"
	default:
		if summary != "" {
			return fmt.Sprintf("Phase: %s - %s", phase, summary)
		}
		return fmt.Sprintf("Phase: %s", phase)
	}
}

// runPhaseWithDB writes a phase update to the state database and optionally
// fires a bd comment for audit trail.
// If dbPath is empty, uses the default path (~/.orch/state.db).
// If writeComment is true, also writes a bd comment (non-blocking on failure).
func runPhaseWithDB(beadsID, phase, summary, dbPath string, writeComment bool) error {
	// Open the state database
	var db *state.DB
	var err error
	if dbPath != "" {
		db, err = state.Open(dbPath)
	} else {
		db, err = state.OpenDefault()
	}
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	if db == nil {
		return fmt.Errorf("could not determine state database path")
	}
	defer db.Close()

	// Write phase directly to SQLite (~1ms)
	if err := db.UpdatePhaseByBeadsID(beadsID, phase, summary); err != nil {
		return fmt.Errorf("failed to update phase: %w", err)
	}

	// Invalidate serve cache to ensure dashboard shows updated phase immediately.
	// This is non-critical so we just call it and ignore errors.
	invalidateServeCache()

	// Fire bd comment for backward compatibility and audit trail.
	// This is best-effort — if beads is unavailable, the phase is still
	// recorded in SQLite. We run it synchronously but don't fail the command
	// if the comment fails. The comment follows the established format that
	// orch complete parses for phase detection.
	if writeComment {
		comment := formatPhaseComment(phase, summary)
		if err := beads.FallbackAddComment(beadsID, comment); err != nil {
			// Non-fatal: phase was already written to SQLite.
			// Print warning so agents can see if comment failed.
			fmt.Fprintf(os.Stderr, "warning: bd comment failed (phase still recorded in SQLite): %v\n", err)
		}
	}

	// Confirm to caller
	if summary != "" {
		fmt.Printf("Phase: %s - %s (beads_id: %s)\n", phase, summary, beadsID)
	} else {
		fmt.Printf("Phase: %s (beads_id: %s)\n", phase, beadsID)
	}

	return nil
}
