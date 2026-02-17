// Package orch provides orchestration-level utilities for agent management.
package orch

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Mode holds the implementation mode flag value for spawn commands.
// Valid values: "tdd", "direct", or "verification-first"
// This is extracted from cmd/orch/spawn_cmd.go to pkg/orch for better
// separation of concerns and reusability across spawn-related commands.
var Mode string

// validModes lists the accepted values for --mode.
var validModes = []string{"tdd", "direct", "verification-first"}

// ValidateMode checks that the --mode value is valid.
// Returns a helpful error if the user passes a backend name (claude/opencode)
// instead of an implementation mode.
func ValidateMode(mode string) error {
	// Check for common mistake: passing backend names to --mode
	if mode == "claude" || mode == "opencode" {
		return fmt.Errorf("invalid --mode value %q: '%s' is a backend, not an implementation mode.\n  Use --backend %s instead.\n  Valid --mode values: tdd, direct, verification-first", mode, mode, mode)
	}
	for _, valid := range validModes {
		if mode == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid --mode value %q. Valid values: tdd, direct, verification-first", mode)
}

// RegisterModeFlag registers the --mode flag on the given cobra command.
// This provides a centralized way to register the mode flag across
// different spawn-related commands (spawn, work, etc.).
//
// Default value is "tdd" (test-driven development mode).
//
// Parameters:
//   - cmd: the cobra command to register the flag on
func RegisterModeFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&Mode, "mode", "tdd", "Implementation mode: tdd, direct, or verification-first")
}
