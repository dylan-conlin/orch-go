// Package orch provides orchestration-level utilities for agent management.
package orch

import "github.com/spf13/cobra"

// Mode holds the implementation mode flag value for spawn commands.
// Valid values: "tdd" or "direct"
// This is extracted from cmd/orch/spawn_cmd.go to pkg/orch for better
// separation of concerns and reusability across spawn-related commands.
var Mode string

// RegisterModeFlag registers the --mode flag on the given cobra command.
// This provides a centralized way to register the mode flag across
// different spawn-related commands (spawn, work, etc.).
//
// Default value is "tdd" (test-driven development mode).
//
// Parameters:
//   - cmd: the cobra command to register the flag on
func RegisterModeFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&Mode, "mode", "tdd", "Implementation mode: tdd or direct")
}
