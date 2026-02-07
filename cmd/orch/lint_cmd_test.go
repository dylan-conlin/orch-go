package main

import (
	"testing"
)

func TestCollectCommands(t *testing.T) {
	commands := collectCommands(rootCmd)

	// Should have at least the known commands
	expectedCommands := []string{"spawn", "send", "status", "complete", "lint"}
	for _, cmd := range expectedCommands {
		if _, ok := commands[cmd]; !ok {
			t.Errorf("expected command %q to be collected, but it wasn't", cmd)
		}
	}

	// lint should have --skills flag
	if lintInfo, ok := commands["lint"]; ok {
		if !lintInfo.Flags["skills"] {
			t.Error("expected lint command to have --skills flag")
		}
	}

	// spawn should have some flags
	if spawnInfo, ok := commands["spawn"]; ok {
		if len(spawnInfo.Flags) == 0 {
			t.Error("expected spawn command to have some flags")
		}
	}

	// Should not include hidden commands in the count,
	// but they should still be introspectable for lint purposes
	// (hidden commands are valid CLI commands, just not shown in help)
	if len(commands) == 0 {
		t.Error("expected at least some commands to be collected")
	}
}

func TestCollectCommandsIncludesSubcommands(t *testing.T) {
	commands := collectCommands(rootCmd)

	// kb should have subcommands
	if kbInfo, ok := commands["kb"]; ok {
		if len(kbInfo.Subcommands) == 0 {
			t.Error("expected kb command to have subcommands")
		}
	}
}
