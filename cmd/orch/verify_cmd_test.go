package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestVerifyCommandRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"verify"})
	if err != nil {
		t.Fatalf("verify command not registered: %v", err)
	}
	if cmd == nil || cmd.Name() != "verify" {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}

func TestDiscoverAndFilterVerificationTargets(t *testing.T) {
	projectDir := t.TempDir()
	activeDir := filepath.Join(projectDir, ".orch", "workspace")
	archivedDir := filepath.Join(activeDir, "archived")

	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		t.Fatalf("mkdir archived: %v", err)
	}

	createWorkspaceWithSpec(t, filepath.Join(activeDir, "og-feat-complete-a"), "orch-go-a111", true)
	createWorkspaceWithSpec(t, filepath.Join(activeDir, "og-feat-incomplete-b"), "orch-go-b222", false)
	createWorkspaceWithSpec(t, filepath.Join(archivedDir, "og-feat-complete-c"), "orch-go-c333", true)

	targets, err := discoverVerificationTargets(projectDir)
	if err != nil {
		t.Fatalf("discoverVerificationTargets error: %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("targets len = %d, want 3", len(targets))
	}

	completed := filterCompletedVerificationTargets(targets)
	if len(completed) != 2 {
		t.Fatalf("completed len = %d, want 2", len(completed))
	}

	for _, target := range completed {
		if !target.HasSynthesis {
			t.Fatalf("filtered target without synthesis: %+v", target)
		}
	}
}

func TestSelectLatestVerificationTarget(t *testing.T) {
	now := time.Now()
	targets := []verificationTarget{
		{WorkspaceName: "older", ModTime: now.Add(-2 * time.Hour)},
		{WorkspaceName: "newer", ModTime: now.Add(-1 * time.Hour)},
	}

	latest := selectLatestVerificationTarget(targets)
	if latest.WorkspaceName != "newer" {
		t.Fatalf("latest = %q, want newer", latest.WorkspaceName)
	}
}

func createWorkspaceWithSpec(t *testing.T, workspacePath, beadsID string, withSynthesis bool) {
	t.Helper()

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workspacePath, ".beads_id"), []byte(beadsID), 0644); err != nil {
		t.Fatalf("write .beads_id: %v", err)
	}

	spec := `version: 1
scope:
  beads_id: ` + beadsID + `
  workspace: test
  skill: feature-impl
verification:
  - id: smoke
    method: cli_smoke
    tier: full
    command: "printf ok"
    expect:
      exit_code: 0
`

	if err := os.WriteFile(filepath.Join(workspacePath, verify.VerificationSpecFileName), []byte(spec), 0644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	if withSynthesis {
		if err := os.WriteFile(filepath.Join(workspacePath, "SYNTHESIS.md"), []byte("done"), 0644); err != nil {
			t.Fatalf("write synthesis: %v", err)
		}
	}
}
