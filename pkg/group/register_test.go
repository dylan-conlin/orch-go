package group

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterProject_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "groups.yaml")

	added, err := RegisterProject(configPath, "my-project", "personal")
	if err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}
	if !added {
		t.Error("expected added=true for new file")
	}

	// Verify file was created and contains the project
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	g, ok := cfg.Groups["personal"]
	if !ok {
		t.Fatal("expected 'personal' group to exist")
	}
	if len(g.Projects) != 1 || g.Projects[0] != "my-project" {
		t.Errorf("expected [my-project], got %v", g.Projects)
	}
}

func TestRegisterProject_AddsToExistingGroup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "groups.yaml")

	// Create initial groups.yaml
	content := `groups:
  orch:
    account: personal
    projects:
      - orch-go
      - orch-cli
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	added, err := RegisterProject(configPath, "new-project", "orch")
	if err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}
	if !added {
		t.Error("expected added=true")
	}

	// Verify
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	g := cfg.Groups["orch"]
	if len(g.Projects) != 3 {
		t.Fatalf("expected 3 projects, got %d: %v", len(g.Projects), g.Projects)
	}

	found := false
	for _, p := range g.Projects {
		if p == "new-project" {
			found = true
		}
	}
	if !found {
		t.Error("new-project not found in group")
	}

	// Account should be preserved
	if g.Account != "personal" {
		t.Errorf("expected account 'personal', got %q", g.Account)
	}
}

func TestRegisterProject_CreatesNewGroup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "groups.yaml")

	// Create initial groups.yaml with one group
	content := `groups:
  orch:
    projects:
      - orch-go
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	added, err := RegisterProject(configPath, "work-tool", "work")
	if err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}
	if !added {
		t.Error("expected added=true for new group")
	}

	// Verify both groups exist
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if len(cfg.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(cfg.Groups))
	}
	if _, ok := cfg.Groups["orch"]; !ok {
		t.Error("expected orch group to still exist")
	}
	if _, ok := cfg.Groups["work"]; !ok {
		t.Error("expected work group to exist")
	}
}

func TestRegisterProject_Idempotent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "groups.yaml")

	// Create initial groups.yaml
	content := `groups:
  orch:
    projects:
      - orch-go
      - my-project
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	added, err := RegisterProject(configPath, "my-project", "orch")
	if err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}
	if added {
		t.Error("expected added=false for already-registered project")
	}

	// Verify no duplicate
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	g := cfg.Groups["orch"]
	if len(g.Projects) != 2 {
		t.Errorf("expected 2 projects (no duplicate), got %d: %v", len(g.Projects), g.Projects)
	}
}

func TestRegisterProject_CreatesDirIfNeeded(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "nested", "dir", "groups.yaml")

	added, err := RegisterProject(configPath, "my-project", "default")
	if err != nil {
		t.Fatalf("RegisterProject() error = %v", err)
	}
	if !added {
		t.Error("expected added=true")
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected groups.yaml to be created in nested directory")
	}
}

func TestAutoDetectGroupFromConfig_ExplicitMemberSibling(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"personal": {
				Projects: []string{"orch-go", "beads"},
			},
			"work": {
				Projects: []string{"toolshed"},
			},
		},
	}

	memberPaths := map[string]string{
		"orch-go":  "/home/user/personal/orch-go",
		"beads":    "/home/user/personal/beads",
		"toolshed": "/home/user/work/toolshed",
	}

	// New project is a sibling of orch-go
	group := AutoDetectGroupFromConfig(cfg, "/home/user/personal/new-project", memberPaths)
	if group != "personal" {
		t.Errorf("expected 'personal', got %q", group)
	}

	// Project in work directory
	group = AutoDetectGroupFromConfig(cfg, "/home/user/work/new-tool", memberPaths)
	if group != "work" {
		t.Errorf("expected 'work', got %q", group)
	}
}

func TestAutoDetectGroupFromConfig_ParentInferred(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"corp": {
				Parent: "monorepo",
			},
		},
	}

	memberPaths := map[string]string{
		"monorepo": "/home/user/work/monorepo",
	}

	// New project under the parent
	group := AutoDetectGroupFromConfig(cfg, "/home/user/work/monorepo/new-service", memberPaths)
	if group != "corp" {
		t.Errorf("expected 'corp', got %q", group)
	}
}

func TestAutoDetectGroupFromConfig_NoMatch(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"personal": {
				Projects: []string{"orch-go"},
			},
		},
	}

	memberPaths := map[string]string{
		"orch-go": "/home/user/personal/orch-go",
	}

	// Completely unrelated directory
	group := AutoDetectGroupFromConfig(cfg, "/opt/random/project", memberPaths)
	if group != "" {
		t.Errorf("expected empty string, got %q", group)
	}
}

func TestAutoDetectGroupFromConfig_NilConfig(t *testing.T) {
	group := AutoDetectGroupFromConfig(nil, "/some/path", nil)
	if group != "" {
		t.Errorf("expected empty string for nil config, got %q", group)
	}
}

func TestWriteConfig_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "groups.yaml")

	cfg := &Config{
		Groups: map[string]Group{
			"test": {
				Account:  "myaccount",
				Projects: []string{"proj-a", "proj-b"},
			},
		},
	}

	if err := writeConfig(configPath, cfg); err != nil {
		t.Fatalf("writeConfig() error = %v", err)
	}

	loaded, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	g := loaded.Groups["test"]
	if g.Account != "myaccount" {
		t.Errorf("account = %q, want 'myaccount'", g.Account)
	}
	if len(g.Projects) != 2 {
		t.Errorf("projects count = %d, want 2", len(g.Projects))
	}
}
