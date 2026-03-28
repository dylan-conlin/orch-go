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

func TestThreadPromoteCmd_DryRun(t *testing.T) {
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
title: "Converged idea"
status: converged
created: 2026-03-20
updated: 2026-03-27
resolved_to: ""
---

# Converged idea

## 2026-03-27

This thread has converged.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-03-20-converged-idea.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()
	threadWorkdir = ""

	// Test dry-run as model
	threadPromoteAs = "model"
	threadPromoteDryRun = true
	defer func() {
		threadPromoteAs = "model"
		threadPromoteDryRun = false
	}()

	var stdout bytes.Buffer
	cmd := threadPromoteCmd
	cmd.SetOut(&stdout)

	if err := cmd.RunE(cmd, []string{"converged-idea"}); err != nil {
		t.Fatalf("thread promote --dry-run failed: %v", err)
	}

	// Thread status should NOT have changed
	data, _ := os.ReadFile(filepath.Join(threadsDir, "2026-03-20-converged-idea.md"))
	if strings.Contains(string(data), "status: promoted") {
		t.Fatal("dry-run should not change thread status")
	}

	// No model directory should exist
	modelDir := filepath.Join(dir, ".kb", "models", "converged-idea")
	if _, err := os.Stat(modelDir); !os.IsNotExist(err) {
		t.Fatal("dry-run should not create model directory")
	}
}

func TestThreadPromoteCmd_Model(t *testing.T) {
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
title: "Named incompleteness"
status: converged
created: 2026-03-20
updated: 2026-03-27
resolved_to: ""
---

# Named incompleteness

## 2026-03-27

Generative systems are organized around named incompleteness.
`
	if err := os.WriteFile(filepath.Join(threadsDir, "2026-03-20-named-incompleteness.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()
	threadWorkdir = ""

	threadPromoteAs = "model"
	threadPromoteDryRun = false
	defer func() {
		threadPromoteAs = "model"
		threadPromoteDryRun = false
	}()

	if err := threadPromoteCmd.RunE(threadPromoteCmd, []string{"named-incompleteness"}); err != nil {
		t.Fatalf("thread promote failed: %v", err)
	}

	// Verify model was scaffolded
	modelPath := filepath.Join(dir, ".kb", "models", "named-incompleteness", "model.md")
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("model not created: %v", err)
	}
	if !strings.Contains(string(data), "# Model: Named incompleteness") {
		t.Error("model missing title")
	}
	if !strings.Contains(string(data), "Promoted From:") {
		t.Error("model missing provenance")
	}

	// Verify probes/ dir was created
	probesDir := filepath.Join(dir, ".kb", "models", "named-incompleteness", "probes")
	info, err := os.Stat(probesDir)
	if err != nil || !info.IsDir() {
		t.Error("probes directory not created")
	}

	// Verify thread status updated
	threadData, _ := os.ReadFile(filepath.Join(threadsDir, "2026-03-20-named-incompleteness.md"))
	if !strings.Contains(string(threadData), "status: promoted") {
		t.Error("thread status not updated to promoted")
	}
	if !strings.Contains(string(threadData), "promoted_to:") {
		t.Error("thread missing promoted_to field")
	}
}

func TestThreadPromoteCmd_Decision(t *testing.T) {
	origDir, _ := os.Getwd()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	threadsDir := filepath.Join(dir, ".kb", "threads")
	decisionsDir := filepath.Join(dir, ".kb", "decisions")
	os.MkdirAll(threadsDir, 0o755)
	os.MkdirAll(decisionsDir, 0o755)

	content := `---
title: "Product surface elements"
status: converged
created: 2026-03-25
updated: 2026-03-27
resolved_to: ""
---

# Product surface elements

## 2026-03-27

Five elements define the product surface.
`
	os.WriteFile(filepath.Join(threadsDir, "2026-03-25-product-surface-elements.md"), []byte(content), 0o644)

	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()
	threadWorkdir = ""

	threadPromoteAs = "decision"
	threadPromoteDryRun = false
	defer func() {
		threadPromoteAs = "model"
		threadPromoteDryRun = false
	}()

	if err := threadPromoteCmd.RunE(threadPromoteCmd, []string{"product-surface-elements"}); err != nil {
		t.Fatalf("thread promote --as decision failed: %v", err)
	}

	// Find the decision file
	entries, _ := os.ReadDir(decisionsDir)
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), "product-surface-elements") {
			data, _ := os.ReadFile(filepath.Join(decisionsDir, e.Name()))
			if strings.Contains(string(data), "# Decision: Product surface elements") {
				found = true
			}
			break
		}
	}
	if !found {
		t.Error("decision file not created or missing title")
	}
}

func TestThreadPromoteCmd_RejectsNonConverged(t *testing.T) {
	origDir, _ := os.Getwd()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	threadsDir := filepath.Join(dir, ".kb", "threads")
	os.MkdirAll(threadsDir, 0o755)

	content := `---
title: "Active thread"
status: active
created: 2026-03-20
updated: 2026-03-27
resolved_to: ""
---

# Active thread

## 2026-03-27

Still working.
`
	os.WriteFile(filepath.Join(threadsDir, "2026-03-20-active-thread.md"), []byte(content), 0o644)

	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()
	threadWorkdir = ""

	threadPromoteAs = "model"
	threadPromoteDryRun = false
	defer func() {
		threadPromoteAs = "model"
		threadPromoteDryRun = false
	}()

	err := threadPromoteCmd.RunE(threadPromoteCmd, []string{"active-thread"})
	if err == nil {
		t.Fatal("expected error promoting non-converged thread")
	}
	if !strings.Contains(err.Error(), "converged") {
		t.Errorf("error should mention converged, got: %v", err)
	}
}
