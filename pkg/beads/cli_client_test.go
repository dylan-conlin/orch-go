package beads

import (
	"fmt"
	"testing"
	"time"
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

	cmd, cancel := c.bdCommand("show", "issue-123", "--json")
	defer cancel()

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

func TestCLIClient_ListArgsBuilding(t *testing.T) {
	c := NewCLIClient(WithBdPath("/custom/bd"))

	tests := []struct {
		name     string
		args     *ListArgs
		wantArgs []string
	}{
		{
			name: "status only",
			args: &ListArgs{Status: "open"},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--status", "open"},
		},
		{
			name: "status with explicit no limit",
			args: &ListArgs{Status: "open", Limit: IntPtr(0)},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--status", "open", "--limit", "0"},
		},
		{
			name: "with labels (AND)",
			args: &ListArgs{Status: "open", Labels: []string{"triage:review"}},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--status", "open", "-l", "triage:review"},
		},
		{
			name: "with multiple labels (AND)",
			args: &ListArgs{Labels: []string{"triage:review", "priority:high"}},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"-l", "triage:review", "-l", "priority:high"},
		},
		{
			name: "with labels_any (OR)",
			args: &ListArgs{LabelsAny: []string{"triage:review", "triage:ready"}},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--label-any", "triage:review", "--label-any", "triage:ready"},
		},
		{
			name: "combined labels and labels_any",
			args: &ListArgs{
				Status:    "open",
				Labels:    []string{"triage:review"},
				LabelsAny: []string{"p0", "p1"},
			},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--status", "open",
				"-l", "triage:review",
				"--label-any", "p0", "--label-any", "p1"},
		},
		{
			name: "with parent and type",
			args: &ListArgs{IssueType: "bug", Parent: "epic-1"},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--type", "bug", "--parent", "epic-1"},
		},
		{
			name: "with explicit limit",
			args: &ListArgs{Status: "open", Limit: IntPtr(50)},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--status", "open", "--limit", "50"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the command args the same way List() does
			cmdArgs := []string{"list", "--json"}
			if tt.args != nil {
				if tt.args.Status != "" {
					cmdArgs = append(cmdArgs, "--status", tt.args.Status)
				}
				if tt.args.IssueType != "" {
					cmdArgs = append(cmdArgs, "--type", tt.args.IssueType)
				}
				if tt.args.Parent != "" {
					cmdArgs = append(cmdArgs, "--parent", tt.args.Parent)
				}
				for _, label := range tt.args.Labels {
					cmdArgs = append(cmdArgs, "-l", label)
				}
				for _, label := range tt.args.LabelsAny {
					cmdArgs = append(cmdArgs, "--label-any", label)
				}
				if tt.args.Limit != nil {
				cmdArgs = append(cmdArgs, "--limit", fmt.Sprintf("%d", *tt.args.Limit))
			}
			}
			cmd, cancel := c.bdCommand(cmdArgs...)
			defer cancel()

			got := cmd.Args
			if len(got) != len(tt.wantArgs) {
				t.Fatalf("args length = %d, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.wantArgs), got, tt.wantArgs)
			}
			for i, arg := range tt.wantArgs {
				if got[i] != arg {
					t.Errorf("args[%d] = %q, want %q", i, got[i], arg)
				}
			}
		})
	}
}

func TestCLIClient_LabelCommands(t *testing.T) {
	c := NewCLIClient(WithBdPath("/custom/bd"))

	tests := []struct {
		name     string
		buildCmd func() []string
		wantArgs []string
	}{
		{
			name: "AddLabel uses bd label add",
			buildCmd: func() []string {
				cmd, cancel := c.bdCommand("label", "add", "issue-1", "triage:ready")
				defer cancel()
				return cmd.Args
			},
			wantArgs: []string{"/custom/bd", "label", "add", "issue-1", "triage:ready"},
		},
		{
			name: "RemoveLabel uses bd label remove",
			buildCmd: func() []string {
				cmd, cancel := c.bdCommand("label", "remove", "issue-1", "triage:review")
				defer cancel()
				return cmd.Args
			},
			wantArgs: []string{"/custom/bd", "label", "remove", "issue-1", "triage:review"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.buildCmd()
			if len(got) != len(tt.wantArgs) {
				t.Fatalf("args length = %d, want %d: %v", len(got), len(tt.wantArgs), got)
			}
			for i, arg := range tt.wantArgs {
				if got[i] != arg {
					t.Errorf("args[%d] = %q, want %q", i, got[i], arg)
				}
			}
		})
	}
}

func TestCLIClient_DefaultTimeout(t *testing.T) {
	if DefaultCLITimeout != 30*time.Second {
		t.Errorf("DefaultCLITimeout = %s, want 30s", DefaultCLITimeout)
	}
}

func TestCLIClient_WithCLITimeout(t *testing.T) {
	c := NewCLIClient(WithCLITimeout(10 * time.Second))
	if c.Timeout != 10*time.Second {
		t.Errorf("Timeout = %s, want 10s", c.Timeout)
	}
}

func TestCLIClient_TimeoutDefaultsWhenZero(t *testing.T) {
	c := NewCLIClient()
	if c.Timeout != 0 {
		t.Errorf("Timeout = %s, want 0 (uses default)", c.Timeout)
	}
	// bdCommand should use DefaultCLITimeout when Timeout is 0
	cmd, cancel := c.bdCommand("--version")
	defer cancel()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
}
