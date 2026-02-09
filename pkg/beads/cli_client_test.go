package beads

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCLIClient(t *testing.T) {
	// Test default creation
	c := NewCLIClient()
	if c.BdPath != "bd" {
		t.Errorf("BdPath = %q, want %q", c.BdPath, "bd")
	}
	if c.WorkDir != "" {
		t.Errorf("WorkDir = %q, want empty", c.WorkDir)
	}
}

func TestNewCLIClient_WithOptions(t *testing.T) {
	c := NewCLIClient(
		WithWorkDir("/some/dir"),
		WithBdPath("/usr/bin/bd"),
		WithEnv([]string{"FOO=bar"}),
	)

	if c.WorkDir != "/some/dir" {
		t.Errorf("WorkDir = %q, want %q", c.WorkDir, "/some/dir")
	}
	if c.BdPath != "/usr/bin/bd" {
		t.Errorf("BdPath = %q, want %q", c.BdPath, "/usr/bin/bd")
	}
	if len(c.Env) != 1 || c.Env[0] != "FOO=bar" {
		t.Errorf("Env = %v, want %v", c.Env, []string{"FOO=bar"})
	}
}

func TestCLIClient_bdCommand(t *testing.T) {
	c := NewCLIClient(
		WithWorkDir("/test/dir"),
		WithBdPath("/custom/bd"),
	)

	cmd := c.bdCommand("show", "issue-123", "--json")

	if cmd.Path != "/custom/bd" {
		t.Errorf("cmd.Path = %q, want %q", cmd.Path, "/custom/bd")
	}
	if cmd.Dir != "/test/dir" {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, "/test/dir")
	}
	// Args includes the command itself as first element
	expectedArgs := []string{"/custom/bd", "show", "issue-123", "--json"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("cmd.Args length = %d, want %d", len(cmd.Args), len(expectedArgs))
	}
	for i, arg := range expectedArgs {
		if cmd.Args[i] != arg {
			t.Errorf("cmd.Args[%d] = %q, want %q", i, cmd.Args[i], arg)
		}
	}
}

func TestCLIClient_ImplementsBeadsClient(t *testing.T) {
	// This test verifies that CLIClient implements the BeadsClient interface.
	// The compilation will fail if it doesn't.
	var _ BeadsClient = (*CLIClient)(nil)
	var _ BeadsClient = NewCLIClient()
}

func TestCLIClient_AddLabels_UsesSingleUpdateCommand(t *testing.T) {
	workDir := t.TempDir()
	invocations := filepath.Join(workDir, "invocations.log")
	scriptPath := filepath.Join(workDir, "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\\n' \"$*\" >> \"" + invocations + "\"",
		"printf 'ok'",
		"exit 0",
	}, "\n") + "\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	c := NewCLIClient(WithWorkDir(workDir), WithBdPath(scriptPath))
	if err := c.AddLabels("orch-go-123", "triage:ready", "area:beads", "effort:small"); err != nil {
		t.Fatalf("AddLabels failed: %v", err)
	}

	calls := readInvocationLines(t, invocations)
	if len(calls) != 1 {
		t.Fatalf("invocation count = %d, want 1", len(calls))
	}

	if !strings.Contains(calls[0], "update orch-go-123 --add-label triage:ready --add-label area:beads --add-label effort:small") {
		t.Fatalf("unexpected command args: %q", calls[0])
	}
}

func TestFallbackAddLabels_UsesSingleUpdateCommand(t *testing.T) {
	workDir := t.TempDir()
	invocations := filepath.Join(workDir, "invocations.log")
	scriptPath := filepath.Join(workDir, "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\\n' \"$*\" >> \"" + invocations + "\"",
		"printf 'ok'",
		"exit 0",
	}, "\n") + "\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	oldPath := BdPath
	oldDir := DefaultDir
	t.Cleanup(func() {
		BdPath = oldPath
		DefaultDir = oldDir
	})
	BdPath = scriptPath
	DefaultDir = workDir

	if err := FallbackAddLabels("orch-go-456", "triage:ready", "area:cli"); err != nil {
		t.Fatalf("FallbackAddLabels failed: %v", err)
	}

	calls := readInvocationLines(t, invocations)
	if len(calls) != 1 {
		t.Fatalf("invocation count = %d, want 1", len(calls))
	}

	if !strings.Contains(calls[0], "update orch-go-456 --add-label triage:ready --add-label area:cli") {
		t.Fatalf("unexpected command args: %q", calls[0])
	}
}
