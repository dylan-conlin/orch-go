package main

import (
	"strings"
	"testing"
)

// TestHiddenCommands tests that dead commands are hidden from help output but still functional.
func TestHiddenCommands(t *testing.T) {
	// List of commands that should be hidden (dead commands with low/no usage)
	hiddenCommands := []string{
		"claim",
		"registry",
		"retries",
		"deploy",
		"config",
		"test-report",
		"model",
		"logs",
		"docs",
		"guarded",
		"emit",
		"history",
		"changelog",
		"transcript",
		"swarm",
		"tokens",
		"fetch-md",
	}

	t.Run("hidden commands not in help output", func(t *testing.T) {
		// Get help output
		cmd := rootCmd
		helpOutput := cmd.UsageString()

		// Verify each hidden command is NOT in help output
		for _, cmdName := range hiddenCommands {
			if strings.Contains(helpOutput, cmdName) {
				t.Errorf("Hidden command %q should not appear in help output", cmdName)
			}
		}
	})

	t.Run("hidden commands are still registered", func(t *testing.T) {
		// Verify each hidden command is still registered and accessible
		for _, cmdName := range hiddenCommands {
			cmd, _, err := rootCmd.Find([]string{cmdName})
			if err != nil {
				t.Errorf("Hidden command %q should still be registered: %v", cmdName, err)
				continue
			}
			if cmd == nil || cmd.Name() != cmdName {
				t.Errorf("Hidden command %q not found or has wrong name", cmdName)
			}
			if !cmd.Hidden {
				t.Errorf("Command %q should have Hidden=true", cmdName)
			}
		}
	})

	t.Run("visible commands appear in help", func(t *testing.T) {
		// Test a few known active commands to ensure they're NOT hidden
		visibleCommands := []string{
			"spawn",
			"status",
			"complete",
			"review",
			"serve",
		}

		helpOutput := rootCmd.UsageString()

		for _, cmdName := range visibleCommands {
			if !strings.Contains(helpOutput, cmdName) {
				t.Errorf("Visible command %q should appear in help output", cmdName)
			}

			// Also verify Hidden=false
			cmd, _, err := rootCmd.Find([]string{cmdName})
			if err != nil {
				t.Errorf("Command %q should be registered: %v", cmdName, err)
				continue
			}
			if cmd != nil && cmd.Hidden {
				t.Errorf("Command %q should NOT be hidden", cmdName)
			}
		}
	})
}
