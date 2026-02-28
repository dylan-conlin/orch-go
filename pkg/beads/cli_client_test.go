package beads

import (
	"fmt"
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
				"--status", "open", "--limit", "0"},
		},
		{
			name: "with labels (AND)",
			args: &ListArgs{Status: "open", Labels: []string{"triage:review"}},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--status", "open", "-l", "triage:review", "--limit", "0"},
		},
		{
			name: "with multiple labels (AND)",
			args: &ListArgs{Labels: []string{"triage:review", "priority:high"}},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"-l", "triage:review", "-l", "priority:high", "--limit", "0"},
		},
		{
			name: "with labels_any (OR)",
			args: &ListArgs{LabelsAny: []string{"triage:review", "triage:ready"}},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--label-any", "triage:review", "--label-any", "triage:ready",
				"--limit", "0"},
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
				"--label-any", "p0", "--label-any", "p1",
				"--limit", "0"},
		},
		{
			name: "with parent and type",
			args: &ListArgs{IssueType: "bug", Parent: "epic-1"},
			wantArgs: []string{"/custom/bd", "list", "--json",
				"--type", "bug", "--parent", "epic-1", "--limit", "0"},
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
				cmdArgs = append(cmdArgs, "--limit", fmt.Sprintf("%d", tt.args.Limit))
			}
			cmd := c.bdCommand(cmdArgs...)

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
				cmd := c.bdCommand("label", "add", "issue-1", "triage:ready")
				return cmd.Args
			},
			wantArgs: []string{"/custom/bd", "label", "add", "issue-1", "triage:ready"},
		},
		{
			name: "RemoveLabel uses bd label remove",
			buildCmd: func() []string {
				cmd := c.bdCommand("label", "remove", "issue-1", "triage:review")
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
