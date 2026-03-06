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
