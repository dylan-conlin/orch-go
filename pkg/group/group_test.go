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
  corp:
    account: work
    parent: corp-monorepo
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

	corp := cfg.Groups["corp"]
	if corp.Account != "work" {
		t.Errorf("corp.Account = %q, want %q", corp.Account, "work")
	}
	if corp.Parent != "corp-monorepo" {
		t.Errorf("corp.Parent = %q, want %q", corp.Parent, "corp-monorepo")
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
			"corp": {
				Account: "work",
				Parent:  "corp-monorepo",
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
			"corp": {
				Account: "work",
				Parent:  "corp-monorepo",
			},
		},
	}

	kbProjects := map[string]string{
		"corp-monorepo": "/home/user/work/corp-monorepo",
		"toolshed":             "/home/user/work/corp-monorepo/toolshed",
		"price-watch":          "/home/user/work/corp-monorepo/price-watch",
		"orch-go":              "/home/user/personal/orch-go",
	}

	// toolshed is a child of corp-monorepo
	groups := cfg.GroupsForProject("toolshed", kbProjects)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group for toolshed, got %d", len(groups))
	}
	if groups[0].Name != "corp" {
		t.Errorf("expected group name %q, got %q", "corp", groups[0].Name)
	}

	// price-watch is also a child
	groups = cfg.GroupsForProject("price-watch", kbProjects)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group for price-watch, got %d", len(groups))
	}

	// orch-go is NOT a child of corp-monorepo
	groups = cfg.GroupsForProject("orch-go", kbProjects)
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for orch-go, got %d", len(groups))
	}
}

func TestGroupsForProject_ParentIsSelfMember(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"corp": {
				Account: "work",
				Parent:  "corp-monorepo",
			},
		},
	}

	kbProjects := map[string]string{
		"corp-monorepo": "/home/user/work/corp-monorepo",
		"toolshed":             "/home/user/work/corp-monorepo/toolshed",
	}

	// The parent project itself should be a member of its own group
	groups := cfg.GroupsForProject("corp-monorepo", kbProjects)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group for parent corp-monorepo, got %d", len(groups))
	}
	if groups[0].Name != "corp" {
		t.Errorf("expected group name %q, got %q", "corp", groups[0].Name)
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
			"corp": {
				Account: "work",
				Parent:  "corp-monorepo",
			},
		},
	}

	kbProjects := map[string]string{
		"corp-monorepo": "/home/user/work/corp-monorepo",
		"toolshed":             "/home/user/work/corp-monorepo/toolshed",
		"price-watch":          "/home/user/work/corp-monorepo/price-watch",
		"sendassist":           "/home/user/work/corp-monorepo/sendassist",
		"orch-go":              "/home/user/personal/orch-go",
	}

	siblings := cfg.SiblingsOf("toolshed", kbProjects)

	// Should include price-watch, sendassist, corp-monorepo but NOT toolshed or orch-go
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
	if !found["corp-monorepo"] {
		t.Error("expected corp-monorepo in siblings")
	}
	if found["toolshed"] {
		t.Error("toolshed should not be in its own siblings")
	}
	if found["orch-go"] {
		t.Error("orch-go should not be in corp siblings")
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
			"corp": {
				Account: "work",
				Parent:  "corp-monorepo",
			},
		},
	}

	kbProjects := map[string]string{
		"corp-monorepo": "/home/user/work/corp-monorepo",
		"toolshed":             "/home/user/work/corp-monorepo/toolshed",
		"price-watch":          "/home/user/work/corp-monorepo/price-watch",
		"orch-go":              "/home/user/personal/orch-go",
	}

	members := cfg.ResolveGroupMembers("corp", kbProjects)

	if len(members) != 3 {
		t.Fatalf("expected 3 members (parent + 2 children), got %d: %v", len(members), members)
	}

	found := map[string]bool{}
	for _, m := range members {
		found[m] = true
	}
	if !found["corp-monorepo"] {
		t.Error("expected parent project in members")
	}
	if !found["toolshed"] {
		t.Error("expected toolshed in members")
	}
	if !found["price-watch"] {
		t.Error("expected price-watch in members")
	}
	if found["orch-go"] {
		t.Error("orch-go should not be in corp members")
	}
}

func TestAccountForProjectDir_ExplicitMember(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"orch": {Account: "personal", Projects: []string{"orch-go", "beads"}},
			"corp":  {Account: "work", Projects: []string{"toolshed"}},
		},
	}
	kbProjects := map[string]string{
		"orch-go":  "/home/user/orch-go",
		"beads":    "/home/user/beads",
		"toolshed": "/home/user/work/toolshed",
	}

	if got := cfg.AccountForProjectDir("/home/user/orch-go", kbProjects); got != "personal" {
		t.Errorf("expected personal for orch-go, got %q", got)
	}
	if got := cfg.AccountForProjectDir("/home/user/work/toolshed", kbProjects); got != "work" {
		t.Errorf("expected work for toolshed, got %q", got)
	}
}

func TestAccountForProjectDir_ParentInferred(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"corp": {Account: "work", Parent: "corp-monorepo"},
		},
	}
	kbProjects := map[string]string{
		"corp-monorepo": "/home/user/work/corp-monorepo",
		"price-watch":          "/home/user/work/corp-monorepo/price-watch",
	}

	if got := cfg.AccountForProjectDir("/home/user/work/corp-monorepo/price-watch", kbProjects); got != "work" {
		t.Errorf("expected work for price-watch (parent-inferred), got %q", got)
	}
}

func TestAccountForProjectDir_Ungrouped(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"orch": {Account: "personal", Projects: []string{"orch-go"}},
		},
	}
	kbProjects := map[string]string{
		"dotfiles": "/home/user/dotfiles",
	}

	if got := cfg.AccountForProjectDir("/home/user/dotfiles", kbProjects); got != "" {
		t.Errorf("expected empty for ungrouped project, got %q", got)
	}
}

func TestAccountForProjectDir_NilConfig(t *testing.T) {
	var cfg *Config
	if got := cfg.AccountForProjectDir("/home/user/foo", nil); got != "" {
		t.Errorf("expected empty for nil config, got %q", got)
	}
}

func TestAccountForProjectDir_EmptyDir(t *testing.T) {
	cfg := &Config{
		Groups: map[string]Group{
			"orch": {Account: "personal", Projects: []string{"orch-go"}},
		},
	}
	if got := cfg.AccountForProjectDir("", nil); got != "" {
		t.Errorf("expected empty for empty dir, got %q", got)
	}
}
