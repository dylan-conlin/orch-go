package main

import (
	"bytes"
	"testing"
)

func TestHarnessCommandRegistered(t *testing.T) {
	// Verify orch harness is registered as a subcommand
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "harness" {
			found = true
			// Verify subcommands
			subCmds := make(map[string]bool)
			for _, sub := range cmd.Commands() {
				subCmds[sub.Use] = true
			}
			for _, expected := range []string{"init", "check", "lock", "unlock", "status", "verify", "snapshot", "report"} {
				if !subCmds[expected] {
					t.Errorf("harness missing subcommand %q", expected)
				}
			}
			break
		}
	}
	if !found {
		t.Error("harness command not registered on rootCmd")
	}
}

func TestHarnessHelpOutput(t *testing.T) {
	// Verify help mentions hard harness and escape hatch workflow
	long := harnessCmd.Long
	for _, want := range []string{"hard harness", "lock", "unlock", "status"} {
		if !bytes.Contains([]byte(long), []byte(want)) {
			t.Errorf("harness Long description missing %q", want)
		}
	}
}

func TestControlAndHarnessBothRegistered(t *testing.T) {
	// Both orch control and orch harness should exist (backward compatibility)
	cmds := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		cmds[cmd.Use] = true
	}
	if !cmds["control"] {
		t.Error("control command should still be registered")
	}
	if !cmds["harness"] {
		t.Error("harness command should be registered")
	}
}

func TestFindHarnessBinary(t *testing.T) {
	// Just verify the function doesn't panic and returns a result
	path, err := findHarnessBinary()
	if err != nil {
		t.Skipf("harness binary not installed: %v", err)
	}
	if path == "" {
		t.Error("findHarnessBinary returned empty path with nil error")
	}
}
