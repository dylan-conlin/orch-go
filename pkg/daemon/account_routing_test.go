package daemon

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/group"
)

func TestResolveAccountForProject_WithGroupConfig(t *testing.T) {
	d := &Daemon{
		GroupConfig: &group.Config{
			Groups: map[string]group.Group{
				"orch": {Account: "personal", Projects: []string{"orch-go", "beads"}},
				"scs":  {Account: "work", Projects: []string{"toolshed"}},
			},
		},
		KBProjects: map[string]string{
			"orch-go":  "/home/user/orch-go",
			"beads":    "/home/user/beads",
			"toolshed": "/home/user/work/toolshed",
		},
	}

	if got := d.resolveAccountForProject("/home/user/orch-go"); got != "personal" {
		t.Errorf("expected personal for orch-go, got %q", got)
	}
	if got := d.resolveAccountForProject("/home/user/work/toolshed"); got != "work" {
		t.Errorf("expected work for toolshed, got %q", got)
	}
}

func TestResolveAccountForProject_NoGroupConfig(t *testing.T) {
	d := &Daemon{}
	if got := d.resolveAccountForProject("/home/user/orch-go"); got != "" {
		t.Errorf("expected empty when no group config, got %q", got)
	}
}

func TestResolveAccountForProject_EmptyProjectDir(t *testing.T) {
	d := &Daemon{
		GroupConfig: &group.Config{
			Groups: map[string]group.Group{
				"orch": {Account: "personal", Projects: []string{"orch-go"}},
			},
		},
	}
	// Empty projectDir means local project — no account override needed
	if got := d.resolveAccountForProject(""); got != "" {
		t.Errorf("expected empty for empty projectDir, got %q", got)
	}
}

func TestResolveAccountForProject_UngroupedProject(t *testing.T) {
	d := &Daemon{
		GroupConfig: &group.Config{
			Groups: map[string]group.Group{
				"orch": {Account: "personal", Projects: []string{"orch-go"}},
			},
		},
		KBProjects: map[string]string{
			"dotfiles": "/home/user/dotfiles",
		},
	}
	if got := d.resolveAccountForProject("/home/user/dotfiles"); got != "" {
		t.Errorf("expected empty for ungrouped project, got %q", got)
	}
}

func TestBuildKBProjectsMap(t *testing.T) {
	registry := NewProjectRegistryFromMap(map[string]string{
		"orch-go":  "/home/user/orch-go",
		"toolshed": "/home/user/work/scs-special-projects/toolshed",
	}, "/home/user/orch-go")

	m := BuildKBProjectsMap(registry)
	if m["orch-go"] != "/home/user/orch-go" {
		t.Errorf("expected orch-go mapped, got %q", m["orch-go"])
	}
	if m["toolshed"] != "/home/user/work/scs-special-projects/toolshed" {
		t.Errorf("expected toolshed mapped, got %q", m["toolshed"])
	}
}

func TestBuildKBProjectsMap_NilRegistry(t *testing.T) {
	m := BuildKBProjectsMap(nil)
	if m != nil {
		t.Errorf("expected nil for nil registry, got %v", m)
	}
}

func TestSpawnIssue_PassesAccountToSpawner(t *testing.T) {
	var capturedAccount string
	d := &Daemon{
		Config: DefaultConfig(),
		GroupConfig: &group.Config{
			Groups: map[string]group.Group{
				"scs": {Account: "work", Projects: []string{"toolshed"}},
			},
		},
		KBProjects: map[string]string{
			"toolshed": "/home/user/work/toolshed",
		},
		Issues: &mockIssueQuerier{
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		StatusUpdater: &mockIssueUpdater{},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				capturedAccount = account
				return nil
			},
		},
		SpawnedIssues: NewSpawnedIssueTracker(),
	}

	issue := &Issue{
		ID:         "toolshed-123",
		Title:      "Fix bug",
		ProjectDir: "/home/user/work/toolshed",
	}

	result, _, err := d.spawnIssue(issue, "systematic-debugging", "opus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Processed {
		t.Fatalf("expected issue to be processed, got: %s", result.Message)
	}
	if capturedAccount != "work" {
		t.Errorf("expected account 'work' passed to spawner, got %q", capturedAccount)
	}
}

func TestSpawnIssue_NoAccountForLocalProject(t *testing.T) {
	var capturedAccount string
	d := &Daemon{
		Config: DefaultConfig(),
		GroupConfig: &group.Config{
			Groups: map[string]group.Group{
				"orch": {Account: "personal", Projects: []string{"orch-go"}},
			},
		},
		KBProjects: map[string]string{
			"orch-go": "/home/user/orch-go",
		},
		Issues: &mockIssueQuerier{
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		StatusUpdater: &mockIssueUpdater{},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				capturedAccount = account
				return nil
			},
		},
		SpawnedIssues: NewSpawnedIssueTracker(),
	}

	// Local project issue has empty ProjectDir
	issue := &Issue{
		ID:    "orch-go-456",
		Title: "Add feature",
	}

	result, _, err := d.spawnIssue(issue, "feature-impl", "opus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Processed {
		t.Fatalf("expected issue to be processed, got: %s", result.Message)
	}
	// Empty ProjectDir means local project — account should be empty (use default)
	if capturedAccount != "" {
		t.Errorf("expected empty account for local project, got %q", capturedAccount)
	}
}
