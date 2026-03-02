package control

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverControlPlaneFiles(t *testing.T) {
	// Create a temporary settings.json with hook references
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, ".orch", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create hook script files that are referenced
	hookFiles := []string{
		"gate-bd-close.py",
		"enforce-phase-complete.py",
		"gate-orchestrator-code-access.py",
	}
	for _, name := range hookFiles {
		path := filepath.Join(hooksDir, name)
		if err := os.WriteFile(path, []byte("# hook"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	settingsPath := filepath.Join(tmpDir, "settings.json")
	settingsContent := `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "` + filepath.Join(hooksDir, "gate-bd-close.py") + `"
          }
        ]
      },
      {
        "matcher": "Read|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "` + filepath.Join(hooksDir, "gate-orchestrator-code-access.py") + `"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "` + filepath.Join(hooksDir, "enforce-phase-complete.py") + `"
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/some/informational/hook.sh"
          }
        ]
      }
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverControlPlaneFiles(settingsPath)
	if err != nil {
		t.Fatalf("DiscoverControlPlaneFiles failed: %v", err)
	}

	// Should include settings.json itself
	found := make(map[string]bool)
	for _, f := range files {
		found[f] = true
	}

	if !found[settingsPath] {
		t.Error("settings.json should be in control plane files")
	}

	// Should include enforcement hooks (PreToolUse, Stop) that exist on disk
	for _, name := range hookFiles {
		path := filepath.Join(hooksDir, name)
		if !found[path] {
			t.Errorf("expected %s in control plane files", name)
		}
	}

	// Should NOT include informational hooks (SessionStart)
	if found["/some/informational/hook.sh"] {
		t.Error("informational hook should not be in control plane files")
	}
}

func TestDiscoverControlPlaneFiles_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Settings referencing a hook file that doesn't exist
	settingsContent := `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "/nonexistent/hook.py"
          }
        ]
      }
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverControlPlaneFiles(settingsPath)
	if err != nil {
		t.Fatalf("should not error for missing hook files: %v", err)
	}

	// Should still include settings.json
	found := false
	for _, f := range files {
		if f == settingsPath {
			found = true
		}
	}
	if !found {
		t.Error("settings.json should always be included")
	}

	// Should NOT include nonexistent hook
	for _, f := range files {
		if f == "/nonexistent/hook.py" {
			t.Error("nonexistent hook should not be included")
		}
	}
}

func TestDiscoverControlPlaneFiles_ExpandsEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(hooksDir, "gate.py")
	if err := os.WriteFile(hookPath, []byte("# hook"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", tmpDir)

	settingsPath := filepath.Join(tmpDir, "settings.json")
	settingsContent := `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "$HOME/hooks/gate.py"
          }
        ]
      }
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverControlPlaneFiles(settingsPath)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	found := false
	for _, f := range files {
		if f == hookPath {
			found = true
		}
	}
	if !found {
		t.Errorf("expected expanded hook path %s in files, got %v", hookPath, files)
	}
}

func TestFileStatus(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	status, err := FileStatus(testFile)
	if err != nil {
		t.Fatalf("FileStatus failed: %v", err)
	}

	if status.Path != testFile {
		t.Errorf("expected path %s, got %s", testFile, status.Path)
	}
	if !status.Exists {
		t.Error("file should exist")
	}
	// By default, files are not locked
	if status.Locked {
		t.Error("file should not be locked by default")
	}
}

func TestLockUnlockCycle(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Lock the file
	if err := Lock([]string{testFile}); err != nil {
		t.Fatalf("Lock failed: %v", err)
	}

	// Verify it's locked
	status, err := FileStatus(testFile)
	if err != nil {
		t.Fatalf("FileStatus failed: %v", err)
	}
	if !status.Locked {
		t.Error("file should be locked after Lock()")
	}

	// Verify write fails
	err = os.WriteFile(testFile, []byte(`{"modified":true}`), 0644)
	if err == nil {
		t.Error("writing to locked file should fail")
	}

	// Unlock the file
	if err := Unlock([]string{testFile}); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	// Verify it's unlocked
	status, err = FileStatus(testFile)
	if err != nil {
		t.Fatalf("FileStatus failed: %v", err)
	}
	if status.Locked {
		t.Error("file should be unlocked after Unlock()")
	}

	// Verify write succeeds
	if err := os.WriteFile(testFile, []byte(`{"modified":true}`), 0644); err != nil {
		t.Errorf("writing to unlocked file should succeed: %v", err)
	}
}

func TestLockMissingFile(t *testing.T) {
	err := Lock([]string{"/nonexistent/file"})
	if err == nil {
		t.Error("Lock should fail for missing files")
	}
}

func TestUnlockMissingFile(t *testing.T) {
	// Unlock should silently skip missing files
	err := Unlock([]string{"/nonexistent/file"})
	if err != nil {
		t.Errorf("Unlock should skip missing files: %v", err)
	}
}

func TestEnsureLocked(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	hookFile := filepath.Join(hooksDir, "gate.py")
	if err := os.WriteFile(hookFile, []byte("# hook"), 0644); err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		t.Fatal(err)
	}
	settingsContent := `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "` + hookFile + `"
          }
        ]
      }
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Override HOME so DefaultSettingsPath() points to our tmp dir
	t.Setenv("HOME", tmpDir)

	// First call should lock both files
	n, err := EnsureLocked()
	if err != nil {
		t.Fatalf("EnsureLocked failed: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 files locked, got %d", n)
	}

	// Second call should lock 0 (already locked)
	n, err = EnsureLocked()
	if err != nil {
		t.Fatalf("EnsureLocked (second call) failed: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 files locked on second call, got %d", n)
	}

	// Clean up: unlock files so tmpDir cleanup works
	Unlock([]string{settingsPath, hookFile})
}

func TestEnsureLocked_NoSettings(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// No settings.json exists — should return 0, nil
	n, err := EnsureLocked()
	if err != nil {
		t.Fatalf("EnsureLocked should not error when settings.json missing: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestFileStatus_Missing(t *testing.T) {
	status, err := FileStatus("/nonexistent/file")
	if err != nil {
		t.Fatalf("FileStatus should not error for missing files: %v", err)
	}
	if status.Exists {
		t.Error("file should not exist")
	}
	if status.Locked {
		t.Error("missing file should not be locked")
	}
}
