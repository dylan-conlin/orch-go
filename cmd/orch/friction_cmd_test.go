package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeEvidencePath_RelativeInsideProject(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	evidenceDir := filepath.Join(projectDir, ".kb", "investigations")
	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(evidenceDir, "sample.md")
	if err := os.WriteFile(path, []byte("ok"), 0644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}

	normalized, err := normalizeEvidencePath(projectDir, ".kb/investigations/sample.md")
	if err != nil {
		t.Fatalf("normalizeEvidencePath failed: %v", err)
	}
	if normalized != ".kb/investigations/sample.md" {
		t.Fatalf("expected relative path, got %q", normalized)
	}
}

func TestNormalizeEvidencePath_AbsoluteOutsideProject(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "evidence.md")
	if err := os.WriteFile(outsideFile, []byte("ok"), 0644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}

	normalized, err := normalizeEvidencePath(projectDir, outsideFile)
	if err != nil {
		t.Fatalf("normalizeEvidencePath failed: %v", err)
	}
	if normalized != outsideFile {
		t.Fatalf("expected absolute outside-project path, got %q", normalized)
	}
}

func TestNormalizeEvidencePath_MissingFile(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	_, err := normalizeEvidencePath(projectDir, "missing.md")
	if err == nil {
		t.Fatal("expected error for missing evidence path")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestFrictionLedgerPath(t *testing.T) {
	t.Parallel()

	projectDir := "/tmp/example-project"
	got := frictionLedgerPath(projectDir)
	want := filepath.Join(projectDir, ".orch", "friction-ledger.jsonl")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
