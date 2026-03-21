package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecisionsBudgetCmd(t *testing.T) {
	dir := t.TempDir()
	decDir := filepath.Join(dir, ".kb", "decisions")
	os.MkdirAll(decDir, 0o755)

	// Create a few test decisions
	for _, name := range []string{"2026-03-01-a.md", "2026-03-02-b.md"} {
		os.WriteFile(filepath.Join(decDir, name), []byte(`# Decision: Test

**Date:** 2026-03-01
**Status:** Accepted
**Enforcement:** gate

## Context
Test.
`), 0o644)
	}

	// Change to temp dir for the command
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cmd := decisionsCmd
	cmd.SetArgs([]string{"budget"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("budget command failed: %v", err)
	}
}

func TestDecisionsAuditCmd(t *testing.T) {
	dir := t.TempDir()
	decDir := filepath.Join(dir, ".kb", "decisions")
	os.MkdirAll(decDir, 0o755)

	// One with enforcement, one without
	os.WriteFile(filepath.Join(decDir, "2026-03-01-classified.md"), []byte(`# Decision: Classified

**Date:** 2026-03-01
**Status:** Accepted
**Enforcement:** convention

## Context
Test.
`), 0o644)

	os.WriteFile(filepath.Join(decDir, "2026-03-02-unclassified.md"), []byte(`# Decision: Unclassified

**Date:** 2026-03-02
**Status:** Accepted

## Context
Test.
`), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cmd := decisionsCmd
	cmd.SetArgs([]string{"audit"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("audit command failed: %v", err)
	}
}
