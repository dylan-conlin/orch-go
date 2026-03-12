package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPrefix_MultiSegment(t *testing.T) {
	r := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"bd":      "/home/user/beads",
		},
	}

	tests := []struct {
		issueID string
		want    string
	}{
		{"orch-go-1169", "orch-go"},
		{"bd-85487068", "bd"},
		{"unknown-123", "unknown"},
		{"", ""},
		{"no-dash-suffix", "no-dash"},
	}

	for _, tt := range tests {
		got := r.ExtractPrefix(tt.issueID)
		if got != tt.want {
			t.Errorf("ExtractPrefix(%q) = %q, want %q", tt.issueID, got, tt.want)
		}
	}
}

func TestExtractPrefix_LongestMatch(t *testing.T) {
	r := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch":    "/home/user/orch-cli",
			"orch-go": "/home/user/orch-go",
		},
	}

	got := r.ExtractPrefix("orch-go-1169")
	if got != "orch-go" {
		t.Errorf("ExtractPrefix(orch-go-1169) = %q, want 'orch-go' (longest match)", got)
	}

	got = r.ExtractPrefix("orch-555")
	if got != "orch" {
		t.Errorf("ExtractPrefix(orch-555) = %q, want 'orch'", got)
	}
}

func TestExtractPrefix_NilRegistry(t *testing.T) {
	var r *ProjectRegistry
	got := r.ExtractPrefix("orch-go-1169")
	if got != "" {
		t.Errorf("ExtractPrefix on nil registry = %q, want empty string", got)
	}
}

func TestResolve_CrossProject(t *testing.T) {
	r := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"bd":      "/home/user/beads",
		},
		currentDir: "/home/user/orch-go",
	}

	got := r.Resolve("orch-go-1169")
	if got != "" {
		t.Errorf("Resolve(orch-go-1169) = %q, want empty (same project)", got)
	}

	got = r.Resolve("bd-85487068")
	if got != "/home/user/beads" {
		t.Errorf("Resolve(bd-85487068) = %q, want '/home/user/beads'", got)
	}

	got = r.Resolve("unknown-123")
	if got != "" {
		t.Errorf("Resolve(unknown-123) = %q, want empty (unknown prefix)", got)
	}
}

func TestResolve_NilRegistry(t *testing.T) {
	var r *ProjectRegistry
	got := r.Resolve("orch-go-1169")
	if got != "" {
		t.Errorf("Resolve on nil registry = %q, want empty string", got)
	}
}

func TestResolvePrefix_FromBeadsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads dir: %v", err)
	}
	configContent := "issue-prefix: custom-prefix\n"
	if err := os.WriteFile(filepath.Join(beadsDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	got := resolvePrefix(tmpDir)
	if got != "custom-prefix" {
		t.Errorf("resolvePrefix(%q) = %q, want 'custom-prefix'", tmpDir, got)
	}
}

func TestResolvePrefix_FallbackToBasename(t *testing.T) {
	tmpDir := t.TempDir()
	got := resolvePrefix(tmpDir)
	want := filepath.Base(tmpDir)
	if got != want {
		t.Errorf("resolvePrefix(%q) = %q, want %q (basename fallback)", tmpDir, got, want)
	}
}

func TestProjects_ReturnsAllEntries(t *testing.T) {
	r := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"bd":      "/home/user/beads",
		},
		currentDir: "/home/user/orch-go",
	}

	projects := r.Projects()
	if len(projects) != 2 {
		t.Fatalf("Projects() returned %d entries, want 2", len(projects))
	}

	found := make(map[string]string)
	for _, p := range projects {
		found[p.Prefix] = p.Dir
	}
	if found["orch-go"] != "/home/user/orch-go" {
		t.Errorf("Projects() missing orch-go entry")
	}
	if found["bd"] != "/home/user/beads" {
		t.Errorf("Projects() missing bd entry")
	}
}

func TestProjects_NilRegistry(t *testing.T) {
	var r *ProjectRegistry
	projects := r.Projects()
	if projects != nil {
		t.Errorf("Projects() on nil registry = %v, want nil", projects)
	}
}

func TestProjects_EmptyRegistry(t *testing.T) {
	r := &ProjectRegistry{
		prefixToDir: make(map[string]string),
		currentDir:  "/home/user/orch-go",
	}
	projects := r.Projects()
	if len(projects) != 0 {
		t.Errorf("Projects() on empty registry returned %d entries, want 0", len(projects))
	}
}

func TestCurrentDir_ReturnsDir(t *testing.T) {
	r := &ProjectRegistry{
		prefixToDir: map[string]string{},
		currentDir:  "/home/user/orch-go",
	}
	if got := r.CurrentDir(); got != "/home/user/orch-go" {
		t.Errorf("CurrentDir() = %q, want '/home/user/orch-go'", got)
	}
}

func TestCurrentDir_NilRegistry(t *testing.T) {
	var r *ProjectRegistry
	if got := r.CurrentDir(); got != "" {
		t.Errorf("CurrentDir() on nil registry = %q, want empty", got)
	}
}

func TestResolvePrefix_EmptyPrefixFallsBack(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads dir: %v", err)
	}
	configContent := "issue-prefix: \"\"\n"
	if err := os.WriteFile(filepath.Join(beadsDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	got := resolvePrefix(tmpDir)
	want := filepath.Base(tmpDir)
	if got != want {
		t.Errorf("resolvePrefix(%q) = %q, want %q (basename fallback for empty prefix)", tmpDir, got, want)
	}
}

func TestResolveProjectDirectory_WithWorkdir(t *testing.T) {
	tmpDir := t.TempDir()
	dir, name, err := ResolveProjectDirectory(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != tmpDir {
		t.Errorf("dir = %q, want %q", dir, tmpDir)
	}
	if name != filepath.Base(tmpDir) {
		t.Errorf("name = %q, want %q", name, filepath.Base(tmpDir))
	}
}

func TestResolveProjectDirectory_WithoutWorkdir(t *testing.T) {
	dir, name, err := ResolveProjectDirectory("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("dir = %q, want %q", dir, cwd)
	}
	if name != filepath.Base(cwd) {
		t.Errorf("name = %q, want %q", name, filepath.Base(cwd))
	}
}

func TestResolveProjectDirectory_NonexistentWorkdir(t *testing.T) {
	_, _, err := ResolveProjectDirectory("/nonexistent/path/xyz")
	if err == nil {
		t.Error("expected error for nonexistent workdir")
	}
}

func TestResolveProjectFrom_WorkdirOverride(t *testing.T) {
	tmpDir := t.TempDir()
	r := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
		},
		currentDir: "/home/user/orch-go",
	}

	// Workdir override takes highest priority, even when registry would match
	got, err := ResolveProjectFrom(r, "orch-go-123", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tmpDir {
		t.Errorf("ResolveProjectFrom with workdir = %q, want %q", got, tmpDir)
	}
}

func TestResolveProjectFrom_RegistryLookup(t *testing.T) {
	r := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"pw":      "/home/user/price-watch",
		},
		currentDir: "/home/user/orch-go",
	}

	// Cross-project: pw prefix should resolve to price-watch dir
	got, err := ResolveProjectFrom(r, "pw-abc123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/home/user/price-watch" {
		t.Errorf("ResolveProjectFrom(pw-abc123) = %q, want '/home/user/price-watch'", got)
	}
}

func TestResolveProjectFrom_SameProjectFallsToCWD(t *testing.T) {
	r := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
		},
		currentDir: "/home/user/orch-go",
	}

	// Same project: Resolve returns "" → falls through to CWD
	got, err := ResolveProjectFrom(r, "orch-go-123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fall back to CWD since Resolve returns "" for same project
	cwd, _ := os.Getwd()
	if got != cwd {
		t.Errorf("ResolveProjectFrom(same project) = %q, want CWD %q", got, cwd)
	}
}

func TestResolveProjectFrom_NilRegistry(t *testing.T) {
	// Nil registry: falls through to CWD
	got, err := ResolveProjectFrom(nil, "pw-abc123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cwd, _ := os.Getwd()
	if got != cwd {
		t.Errorf("ResolveProjectFrom(nil registry) = %q, want CWD %q", got, cwd)
	}
}

func TestResolveProjectFrom_EmptyBeadsID(t *testing.T) {
	r := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
		},
		currentDir: "/home/user/orch-go",
	}

	// Empty beads ID: falls through to CWD
	got, err := ResolveProjectFrom(r, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cwd, _ := os.Getwd()
	if got != cwd {
		t.Errorf("ResolveProjectFrom(empty beadsID) = %q, want CWD %q", got, cwd)
	}
}

func TestEqual_SameRegistries(t *testing.T) {
	r1 := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"bd":      "/home/user/beads",
		},
		currentDir: "/home/user/orch-go",
	}
	r2 := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"bd":      "/home/user/beads",
		},
		currentDir: "/home/user/orch-go",
	}
	if !r1.Equal(r2) {
		t.Error("Equal should return true for identical registries")
	}
}

func TestEqual_DifferentRegistries(t *testing.T) {
	r1 := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
		},
	}
	r2 := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"bd":      "/home/user/beads",
		},
	}
	if r1.Equal(r2) {
		t.Error("Equal should return false for different registries")
	}
}

func TestEqual_NilRegistries(t *testing.T) {
	var r1, r2 *ProjectRegistry
	if !r1.Equal(r2) {
		t.Error("Equal should return true for two nil registries")
	}

	r3 := &ProjectRegistry{prefixToDir: map[string]string{}}
	if r1.Equal(r3) {
		t.Error("Equal should return false for nil vs non-nil")
	}
}

func TestDiff_AddedAndRemoved(t *testing.T) {
	r1 := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"bd":      "/home/user/beads",
		},
	}
	r2 := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"pw":      "/home/user/price-watch",
		},
	}
	added, removed := r1.Diff(r2)
	if len(added) != 1 || added[0] != "pw" {
		t.Errorf("added = %v, want [pw]", added)
	}
	if len(removed) != 1 || removed[0] != "bd" {
		t.Errorf("removed = %v, want [bd]", removed)
	}
}

func TestDiff_NilRegistries(t *testing.T) {
	var r1 *ProjectRegistry
	r2 := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
		},
	}
	added, removed := r1.Diff(r2)
	if len(added) != 1 || added[0] != "orch-go" {
		t.Errorf("added = %v, want [orch-go]", added)
	}
	if len(removed) != 0 {
		t.Errorf("removed = %v, want []", removed)
	}
}

func TestHasBeadsDir(t *testing.T) {
	// Directory with .beads
	withBeads := t.TempDir()
	if err := os.MkdirAll(filepath.Join(withBeads, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}
	if !hasBeadsDir(withBeads) {
		t.Error("hasBeadsDir should return true for directory with .beads")
	}

	// Directory without .beads
	withoutBeads := t.TempDir()
	if hasBeadsDir(withoutBeads) {
		t.Error("hasBeadsDir should return false for directory without .beads")
	}
}

func TestDiscoverProjectPath(t *testing.T) {
	parentDir := t.TempDir()

	// Create a project with .beads
	projectDir := filepath.Join(parentDir, "my-project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	knownParents := map[string]bool{parentDir: true}

	got := discoverProjectPath("my-project", knownParents)
	if got != projectDir {
		t.Errorf("discoverProjectPath = %q, want %q", got, projectDir)
	}

	// Non-existent project
	got = discoverProjectPath("nonexistent", knownParents)
	if got != "" {
		t.Errorf("discoverProjectPath(nonexistent) = %q, want empty", got)
	}
}

func TestDiscoverProjectPath_NoBeadsDir(t *testing.T) {
	parentDir := t.TempDir()

	// Create a directory WITHOUT .beads
	projectDir := filepath.Join(parentDir, "no-beads-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	knownParents := map[string]bool{parentDir: true}

	got := discoverProjectPath("no-beads-project", knownParents)
	if got != "" {
		t.Errorf("discoverProjectPath should return empty for project without .beads, got %q", got)
	}
}

func TestBuildProjectDirNames(t *testing.T) {
	names := BuildProjectDirNames(nil)
	if len(names) != 0 {
		t.Errorf("nil registry should return empty map, got %v", names)
	}

	r := &ProjectRegistry{
		prefixToDir: map[string]string{
			"orch-go": "/home/user/orch-go",
			"pw":      "/home/user/price-watch",
		},
	}
	names = BuildProjectDirNames(r)
	if names["orch-go"] != "orch-go" {
		t.Errorf("names[orch-go] = %q, want 'orch-go'", names["orch-go"])
	}
	if names["pw"] != "price-watch" {
		t.Errorf("names[pw] = %q, want 'price-watch'", names["pw"])
	}
}
