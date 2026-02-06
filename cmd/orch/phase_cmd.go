// Package main provides the phase command for direct agent phase reporting via SQLite.
//
// This replaces the beads-comment-based phase reporting for the runtime hot path.
// Agents call `orch phase <beads_id> <phase> [summary]` which writes directly to
// ~/.orch/state.db (~1ms) instead of going through `bd comment` (~700ms).
//
// The hybrid approach: orch phase for runtime speed, bd comment for audit trail.
// Both can coexist — agents are instructed to use orch phase for runtime reporting
// and optionally bd comment for permanent searchable history.
//
// See: .kb/investigations/2026-02-06-design-single-source-agent-state.md (Fork 1)
package main

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/spf13/cobra"
)

var phaseCmd = &cobra.Command{
	Use:   "phase <beads_id> <phase> [summary...]",
	Short: "Report agent phase transition (direct SQLite write)",
	Long: `Report an agent's phase transition directly to the state database.

This is the fast path for phase reporting (~1ms) that replaces parsing
beads comments (~700ms/issue) in the orch status hot path.

Agents should call this at phase transitions (Planning, Implementing,
Testing, Complete, etc.) for immediate visibility in orch status and
the dashboard.

For audit trail, agents should ALSO use 'bd comment' to create permanent
searchable history in beads.

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
	rootCmd.AddCommand(phaseCmd)
}

func runPhase(beadsID, phase, summary string) error {
	return runPhaseWithDB(beadsID, phase, summary, "")
}

// runPhaseWithDB writes a phase update to the state database.
// If dbPath is empty, uses the default path (~/.orch/state.db).
func runPhaseWithDB(beadsID, phase, summary, dbPath string) error {
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

	// Write phase directly to SQLite
	if err := db.UpdatePhaseByBeadsID(beadsID, phase, summary); err != nil {
		return fmt.Errorf("failed to update phase: %w", err)
	}

	// Invalidate serve cache to ensure dashboard shows updated phase immediately.
	// This is non-critical so we just call it and ignore errors.
	invalidateServeCache()

	// Confirm to caller
	if summary != "" {
		fmt.Printf("Phase: %s - %s (beads_id: %s)\n", phase, summary, beadsID)
	} else {
		fmt.Printf("Phase: %s (beads_id: %s)\n", phase, beadsID)
	}

	return nil
}
