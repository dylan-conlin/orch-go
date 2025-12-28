package beads

import (
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
