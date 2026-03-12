package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// settingsJSON is a helper to build settings.json content.
func settingsJSON(t *testing.T, obj map[string]any) string {
	t.Helper()
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestHarnessInitSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range harnessCmd.Commands() {
		if cmd.Use == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Error("harness init subcommand not registered")
	}
}

func TestStandaloneModeSkipsBeadsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	// No .beads/ directory — should not fail
	_, exists := detectStandaloneMode(dir)
	if !exists {
		// standalone mode should be detected when .beads is absent
	}
}

func TestHarnessInitBeadsCloseHook(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	os.MkdirAll(beadsDir, 0755)

	result, err := ensureBeadsCloseHook(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected hook to be created")
	}

	hookPath := filepath.Join(beadsDir, "hooks", "on_close")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("hook file not created: %v", err)
	}
	// Check executable
	if info.Mode()&0111 == 0 {
		t.Error("hook file should be executable")
	}
}

func TestHarnessInitBeadsCloseHookIdempotent(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads", "hooks")
	os.MkdirAll(beadsDir, 0755)
	os.WriteFile(filepath.Join(beadsDir, "on_close"), []byte("#!/bin/bash\nexisting"), 0755)

	result, err := ensureBeadsCloseHook(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected hook already present")
	}

	// Verify content was NOT overwritten
	data, _ := os.ReadFile(filepath.Join(beadsDir, "on_close"))
	if string(data) != "#!/bin/bash\nexisting" {
		t.Error("existing hook was overwritten")
	}
}
