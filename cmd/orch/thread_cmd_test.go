package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreadsDir_DefaultUsesCurrentDir(t *testing.T) {
	// Save and restore
	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()

	threadWorkdir = ""
	dir, err := threadsDir()
	if err != nil {
		t.Fatalf("threadsDir() failed: %v", err)
	}

	cwd, _ := os.Getwd()
	expected := filepath.Join(cwd, ".kb", "threads")
	if dir != expected {
		t.Errorf("threadsDir() = %q, want %q", dir, expected)
	}
}

func TestThreadsDir_WorkdirOverride(t *testing.T) {
	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()

	tmpDir := t.TempDir()
	threadWorkdir = tmpDir

	dir, err := threadsDir()
	if err != nil {
		t.Fatalf("threadsDir() failed: %v", err)
	}

	expected := filepath.Join(tmpDir, ".kb", "threads")
	if dir != expected {
		t.Errorf("threadsDir() = %q, want %q", dir, expected)
	}
}

func TestThreadsDir_WorkdirNotExist(t *testing.T) {
	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()

	threadWorkdir = "/nonexistent/path/that/does/not/exist"

	_, err := threadsDir()
	if err == nil {
		t.Fatal("expected error for nonexistent workdir")
	}
	if !strings.Contains(err.Error(), "workdir does not exist") {
		t.Errorf("expected 'workdir does not exist' error, got: %v", err)
	}
}

func TestThreadsDir_WorkdirIsFile(t *testing.T) {
	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()

	tmpFile := filepath.Join(t.TempDir(), "not-a-dir")
	os.WriteFile(tmpFile, []byte("hello"), 0644)

	threadWorkdir = tmpFile

	_, err := threadsDir()
	if err == nil {
		t.Fatal("expected error when workdir is a file")
	}
	if !strings.Contains(err.Error(), "workdir is not a directory") {
		t.Errorf("expected 'workdir is not a directory' error, got: %v", err)
	}
}

func TestPromptResolvedTo_RequiresConfirmationForBlank(t *testing.T) {
	input := strings.NewReader("\nn\nbrief: resolved in debrief\n")
	var output bytes.Buffer

	resolvedTo, err := promptResolvedTo(bufio.NewReader(input), &output)
	if err != nil {
		t.Fatalf("promptResolvedTo failed: %v", err)
	}
	if resolvedTo != "brief: resolved in debrief" {
		t.Fatalf("resolvedTo = %q", resolvedTo)
	}
	if strings.Count(output.String(), "Resolved to (model, decision, or brief):") != 2 {
		t.Fatalf("expected re-prompt after declined blank confirmation, output=%q", output.String())
	}
}

func TestPromptResolvedTo_AcceptsConfirmedBlank(t *testing.T) {
	input := strings.NewReader("\ny\n")
	var output bytes.Buffer

	resolvedTo, err := promptResolvedTo(bufio.NewReader(input), &output)
	if err != nil {
		t.Fatalf("promptResolvedTo failed: %v", err)
	}
	if resolvedTo != "" {
		t.Fatalf("expected empty resolvedTo, got %q", resolvedTo)
	}
}

func TestThreadUpdateCmd_ResolvedPromptsForTarget(t *testing.T) {
	origDir, _ := os.Getwd()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	threadsDir := filepath.Join(dir, ".kb", "threads")
	if err := os.MkdirAll(threadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
title: "Prompt target"
status: active
created: 2026-03-01
updated: 2026-03-01
resolved_to: ""
---

# Prompt target

## 2026-03-01

Working note.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-03-01-prompt-target.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	origTerminalCheck := threadInputIsTerminal
	threadInputIsTerminal = func(reader io.Reader) bool { return true }
	defer func() { threadInputIsTerminal = origTerminalCheck }()

	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()
	threadWorkdir = ""

	threadUpdateStatus = ""
	threadUpdateTo = ""
	threadResolveTo = ""

	cmd := threadUpdateCmd
	cmd.SetIn(strings.NewReader(".kb/models/enforcement.md\n"))
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	threadUpdateStatus = "resolved"
	if err := cmd.RunE(cmd, []string{"prompt-target"}); err != nil {
		t.Fatalf("thread update command failed: %v", err)
	}

	updated, err := os.ReadFile(filepath.Join(threadsDir, "2026-03-01-prompt-target.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "resolved_to: \".kb/models/enforcement.md\"") {
		t.Fatalf("resolved_to not written: %s", string(updated))
	}
	if !strings.Contains(stderr.String(), "Resolved to (model, decision, or brief):") {
		t.Fatalf("missing prompt, stderr=%q", stderr.String())
	}

	threadUpdateStatus = ""
	threadUpdateTo = ""
	threadResolveTo = ""
}
