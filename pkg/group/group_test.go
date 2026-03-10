package group

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	// Create a temp groups.yaml
	dir := t.TempDir()
	configPath := filepath.Join(dir, "groups.yaml")
	content := `groups:
  orch:
    account: personal
    projects:
      - orch-go
      - orch-cli
      - kb-cli
  scs:
    account: work
    parent: scs-special-projects
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if len(cfg.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(cfg.Groups))
	}

	orch := cfg.Groups["orch"]
	if orch.Account != "personal" {
		t.Errorf("orch.Account = %q, want %q", orch.Account, "personal")
	}
	if len(orch.Projects) != 3 {
		t.Errorf("orch.Projects count = %d, want 3", len(orch.Projects))
	}

	scs := cfg.Groups["scs"]
	if scs.Account != "work" {
		t.Errorf("scs.Account = %q, want %q", scs.Account, "work")
	}
	if scs.Parent != "scs-special-projects" {
		t.Errorf("scs.Parent = %q, want %q", scs.Parent, "scs-special-projects")
	}
}

func TestLoadFromFile_NotExists(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/groups.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestDefaultConfigPath_PrefersKbDir(t *testing.T) {
	// Create temp home with both ~/.kb/ and ~/.orch/ groups.yaml
	home := t.TempDir()
	t.Setenv("HOME", home)

	kbDir := filepath.Join(home, ".kb")
	orchDir := filepath.Join(home, ".orch")
	os.MkdirAll(kbDir, 0755)
	os.MkdirAll(orchDir, 0755)

	kbContent := `groups:
  kb-group:
    projects:
      - from-kb
`
	orchContent := `groups:
  orch-group:
    projects:
      - from-orch
`
	os.WriteFile(filepath.Join(kbDir, "groups.yaml"), []byte(kbContent), 0644)
	os.WriteFile(filepath.Join(orchDir, "groups.yaml"), []byte(orchContent), 0644)

	// Should prefer ~/.kb/groups.yaml
	path := DefaultConfigPath()
	if !filepath.IsAbs(path) {
		t.Fatalf("expected absolute path, got %q", path)
	}
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if _, ok := cfg.Groups["kb-group"]; !ok {
		t.Error("expected kb-group from ~/.kb/groups.yaml, got groups from wrong file")
	}
}

func TestDefaultConfigPath_FallsBackToOrch(t *testing.T) {
	// Create temp home with only ~/.orch/groups.yaml (no ~/.kb/)
	home := t.TempDir()
	t.Setenv("HOME", home)

	orchDir := filepath.Join(home, ".orch")
	os.MkdirAll(orchDir, 0755)

	orchContent := `groups:
  orch-group:
    projects:
      - from-orch
`
	os.WriteFile(filepath.Join(orchDir, "groups.yaml"), []byte(orchContent), 0644)

	// Should fall back to ~/.orch/groups.yaml
	path := DefaultConfigPath()
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if _, ok := cfg.Groups["orch-group"]; !ok {
		t.Error("expected orch-group from ~/.orch/groups.yaml fallback")
	}
}

func TestDefaultConfigPath_NeitherExists(t *testing.T) {
	// Create temp home with no groups.yaml anywhere
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Should return ~/.kb/groups.yaml (primary) even if it doesn't exist
	path := DefaultConfigPath()
	expected := filepath.Join(home, ".kb", "groups.yaml")
	if path != expected {
		t.Errorf("DefaultConfigPath() = %q, want %q", path, expected)
	}
}

func TestGroupsForProject_ExplicitMembership(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"orch": {
				Account:  "personal",
				Projects: []string{"orch-go", "orch-cli", "kb-cli"},
			},
			"scs": {
				Account: "work",
				Parent:  "scs-special-projects",
			},
		},
	}

	// kbProjects: name -> path
	kbProjects := map[string]string{
		"orch-go":  "/home/user/orch-go",
		"orch-cli": "/home/user/orch-cli",
		"kb-cli":   "/home/user/kb-cli",
	}

	groups := cfg.GroupsForProject("orch-go", kbProjects)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group for orch-go, got %d", len(groups))
	}
	if groups[0].Name != "orch" {
		t.Errorf("expected group name %q, got %q", "orch", groups[0].Name)
	}
}

func TestGroupsForProject_ParentInference(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"scs": {
				Account: "work",
				Parent:  "scs-special-projects",
			},
		},
	}

	kbProjects := map[string]string{
		"scs-special-projects": "/home/user/work/scs-special-projects",
		"toolshed":             "/home/user/work/scs-special-projects/toolshed",
		"price-watch":          "/home/user/work/scs-special-projects/price-watch",
		"orch-go":              "/home/user/personal/orch-go",
	}

	// toolshed is a child of scs-special-projects
	groups := cfg.GroupsForProject("toolshed", kbProjects)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group for toolshed, got %d", len(groups))
	}
	if groups[0].Name != "scs" {
		t.Errorf("expected group name %q, got %q", "scs", groups[0].Name)
	}

	// price-watch is also a child
	groups = cfg.GroupsForProject("price-watch", kbProjects)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group for price-watch, got %d", len(groups))
	}

	// orch-go is NOT a child of scs-special-projects
	groups = cfg.GroupsForProject("orch-go", kbProjects)
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for orch-go, got %d", len(groups))
	}
}

func TestGroupsForProject_ParentIsSelfMember(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"scs": {
				Account: "work",
				Parent:  "scs-special-projects",
			},
		},
	}

	kbProjects := map[string]string{
		"scs-special-projects": "/home/user/work/scs-special-projects",
		"toolshed":             "/home/user/work/scs-special-projects/toolshed",
	}

	// The parent project itself should be a member of its own group
	groups := cfg.GroupsForProject("scs-special-projects", kbProjects)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group for parent scs-special-projects, got %d", len(groups))
	}
	if groups[0].Name != "scs" {
		t.Errorf("expected group name %q, got %q", "scs", groups[0].Name)
	}
}

func TestGroupsForProject_Ungrouped(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"orch": {
				Account:  "personal",
				Projects: []string{"orch-go"},
			},
		},
	}

	kbProjects := map[string]string{
		"dotfiles": "/home/user/dotfiles",
	}

	groups := cfg.GroupsForProject("dotfiles", kbProjects)
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for ungrouped project, got %d", len(groups))
	}
}

func TestGroupsForProject_MultipleGroups(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"orch": {
				Account:  "personal",
				Projects: []string{"opencode"},
			},
			"tools": {
				Account:  "personal",
				Projects: []string{"opencode", "beads"},
			},
		},
	}

	kbProjects := map[string]string{
		"opencode": "/home/user/opencode",
	}

	groups := cfg.GroupsForProject("opencode", kbProjects)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups for opencode, got %d", len(groups))
	}
}

func TestSiblingsOf(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"orch": {
				Account:  "personal",
				Projects: []string{"orch-go", "orch-cli", "kb-cli"},
			},
		},
	}

	kbProjects := map[string]string{
		"orch-go":  "/home/user/orch-go",
		"orch-cli": "/home/user/orch-cli",
		"kb-cli":   "/home/user/kb-cli",
	}

	siblings := cfg.SiblingsOf("orch-go", kbProjects)
	if len(siblings) != 2 {
		t.Fatalf("expected 2 siblings for orch-go, got %d: %v", len(siblings), siblings)
	}

	// Should contain orch-cli and kb-cli but not orch-go itself
	found := map[string]bool{}
	for _, s := range siblings {
		found[s] = true
	}
	if !found["orch-cli"] {
		t.Error("expected orch-cli in siblings")
	}
	if !found["kb-cli"] {
		t.Error("expected kb-cli in siblings")
	}
	if found["orch-go"] {
		t.Error("orch-go should not be in its own siblings")
	}
}

func TestSiblingsOf_ParentInference(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"scs": {
				Account: "work",
				Parent:  "scs-special-projects",
			},
		},
	}

	kbProjects := map[string]string{
		"scs-special-projects": "/home/user/work/scs-special-projects",
		"toolshed":             "/home/user/work/scs-special-projects/toolshed",
		"price-watch":          "/home/user/work/scs-special-projects/price-watch",
		"sendassist":           "/home/user/work/scs-special-projects/sendassist",
		"orch-go":              "/home/user/personal/orch-go",
	}

	siblings := cfg.SiblingsOf("toolshed", kbProjects)

	// Should include price-watch, sendassist, scs-special-projects but NOT toolshed or orch-go
	found := map[string]bool{}
	for _, s := range siblings {
		found[s] = true
	}
	if !found["price-watch"] {
		t.Error("expected price-watch in siblings")
	}
	if !found["sendassist"] {
		t.Error("expected sendassist in siblings")
	}
	if !found["scs-special-projects"] {
		t.Error("expected scs-special-projects in siblings")
	}
	if found["toolshed"] {
		t.Error("toolshed should not be in its own siblings")
	}
	if found["orch-go"] {
		t.Error("orch-go should not be in scs siblings")
	}
}

func TestSiblingsOf_Ungrouped(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"orch": {
				Account:  "personal",
				Projects: []string{"orch-go"},
			},
		},
	}

	kbProjects := map[string]string{
		"dotfiles": "/home/user/dotfiles",
	}

	siblings := cfg.SiblingsOf("dotfiles", kbProjects)
	if len(siblings) != 0 {
		t.Errorf("expected 0 siblings for ungrouped project, got %d", len(siblings))
	}
}

func TestAllProjectsInGroups(t *testing.T) {
	groups := []Group{
		{
			Name:     "orch",
			Projects: []string{"orch-go", "orch-cli"},
		},
		{
			Name:     "tools",
			Projects: []string{"beads", "orch-go"}, // orch-go is in both
		},
	}

	projects := AllProjectsInGroups(groups)

	// Should deduplicate: orch-go, orch-cli, beads
	if len(projects) != 3 {
		t.Fatalf("expected 3 unique projects, got %d: %v", len(projects), projects)
	}

	found := map[string]bool{}
	for _, p := range projects {
		found[p] = true
	}
	if !found["orch-go"] || !found["orch-cli"] || !found["beads"] {
		t.Errorf("missing expected projects in %v", projects)
	}
}

func TestResolveGroupMembers_ParentGroup(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"scs": {
				Account: "work",
				Parent:  "scs-special-projects",
			},
		},
	}

	kbProjects := map[string]string{
		"scs-special-projects": "/home/user/work/scs-special-projects",
		"toolshed":             "/home/user/work/scs-special-projects/toolshed",
		"price-watch":          "/home/user/work/scs-special-projects/price-watch",
		"orch-go":              "/home/user/personal/orch-go",
	}

	members := cfg.ResolveGroupMembers("scs", kbProjects)

	if len(members) != 3 {
		t.Fatalf("expected 3 members (parent + 2 children), got %d: %v", len(members), members)
	}

	found := map[string]bool{}
	for _, m := range members {
		found[m] = true
	}
	if !found["scs-special-projects"] {
		t.Error("expected parent project in members")
	}
	if !found["toolshed"] {
		t.Error("expected toolshed in members")
	}
	if !found["price-watch"] {
		t.Error("expected price-watch in members")
	}
	if found["orch-go"] {
		t.Error("orch-go should not be in scs members")
	}
}
