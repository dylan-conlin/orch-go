package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectCrossRepoModel(t *testing.T) {
	// Create temp dirs to simulate git repos
	repoA := t.TempDir()
	repoB := t.TempDir()

	// Set up .git in repoA (simulates a git repo)
	if err := os.MkdirAll(repoA+"/.git", 0755); err != nil {
		t.Fatal(err)
	}
	// Set up .git in repoB
	if err := os.MkdirAll(repoB+"/.git", 0755); err != nil {
		t.Fatal(err)
	}
	// Set up .kb/models in repoB for model path
	modelDir := repoB + "/.kb/models/spawn-architecture"
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name             string
		primaryModelPath string
		projectDir       string
		wantCrossRepo    bool
		wantDir          string
	}{
		{
			name:             "same repo - no cross-repo",
			primaryModelPath: repoA + "/.kb/models/spawn-architecture/model.md",
			projectDir:       repoA,
			wantCrossRepo:    false,
		},
		{
			name:             "different repos - cross-repo detected",
			primaryModelPath: repoB + "/.kb/models/spawn-architecture/model.md",
			projectDir:       repoA,
			wantCrossRepo:    true,
			wantDir:          repoB,
		},
		{
			name:             "empty model path",
			primaryModelPath: "",
			projectDir:       repoA,
			wantCrossRepo:    false,
		},
		{
			name:             "empty project dir",
			primaryModelPath: repoB + "/.kb/models/spawn-architecture/model.md",
			projectDir:       "",
			wantCrossRepo:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectCrossRepoModel(tt.primaryModelPath, tt.projectDir)
			if tt.wantCrossRepo {
				if got == "" {
					t.Errorf("DetectCrossRepoModel() = empty, want cross-repo dir")
				}
				// Compare resolved paths since EvalSymlinks may resolve /var → /private/var on macOS
				if tt.wantDir != "" {
					resolvedWant, err := filepath.EvalSymlinks(tt.wantDir)
					if err != nil {
						resolvedWant = tt.wantDir
					}
					if got != resolvedWant {
						t.Errorf("DetectCrossRepoModel() = %q, want %q", got, resolvedWant)
					}
				}
			} else {
				if got != "" {
					t.Errorf("DetectCrossRepoModel() = %q, want empty", got)
				}
			}
		})
	}
}

func TestDetectCrossRepoModel_Symlink(t *testing.T) {
	// Simulates ~/.kb/ being a symlink to {projectDir}/.kb/global/
	projectDir := t.TempDir()

	// Create project structure: .kb/global/models/foo/model.md
	globalModelDir := projectDir + "/.kb/global/models/foo"
	if err := os.MkdirAll(globalModelDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalModelDir+"/model.md", []byte("# Foo"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .git so it's detected as a repo
	if err := os.MkdirAll(projectDir+"/.git", 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink: {tempDir}/home/.kb -> {projectDir}/.kb/global/
	fakeHome := t.TempDir()
	symlinkPath := fakeHome + "/.kb"
	if err := os.Symlink(projectDir+"/.kb/global", symlinkPath); err != nil {
		t.Skipf("Cannot create symlink (maybe no permission): %v", err)
	}

	// The model path through the symlink should be detected as same-repo
	modelPathViaSymlink := symlinkPath + "/models/foo/model.md"
	result := DetectCrossRepoModel(modelPathViaSymlink, projectDir)
	if result != "" {
		t.Errorf("DetectCrossRepoModel should return empty for symlinked ~/.kb/ path, got %q", result)
	}
}

func TestNormalizeGlobalKBPaths(t *testing.T) {
	// Create project with .kb/global/ structure
	projectDir := t.TempDir()
	globalKBDir := projectDir + "/.kb/global"
	if err := os.MkdirAll(globalKBDir+"/models/foo", 0755); err != nil {
		t.Fatal(err)
	}

	// Create a "home" dir with ~/.kb symlink pointing to .kb/global/
	fakeHome := t.TempDir()
	symlinkPath := fakeHome + "/.kb"
	if err := os.Symlink(globalKBDir, symlinkPath); err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	// Override HOME for this test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", fakeHome)
	defer os.Setenv("HOME", origHome)

	matches := []KBContextMatch{
		{
			Type:  "model",
			Title: "Foo Model",
			Path:  symlinkPath + "/models/foo/model.md",
		},
		{
			Type:  "guide",
			Title: "Local Guide",
			Path:  projectDir + "/.kb/guides/local.md",
		},
	}

	normalized := normalizeGlobalKBPaths(matches, projectDir)

	// First match should have its path normalized to .kb/global/
	if !strings.Contains(normalized[0].Path, ".kb/global/models/foo/model.md") {
		t.Errorf("Expected path to contain .kb/global/models/foo/model.md, got %q", normalized[0].Path)
	}
	// Second match (local) should be unchanged
	if normalized[1].Path != projectDir+"/.kb/guides/local.md" {
		t.Errorf("Local path should be unchanged, got %q", normalized[1].Path)
	}
}

func TestEvalSymlinksWithFallback(t *testing.T) {
	// Test with existing path
	existingDir := t.TempDir()
	result := evalSymlinksWithFallback(existingDir)
	// Should resolve (on macOS, /var → /private/var)
	resolved, _ := filepath.EvalSymlinks(existingDir)
	if result != resolved {
		t.Errorf("evalSymlinksWithFallback(%q) = %q, want %q", existingDir, result, resolved)
	}

	// Test with non-existent file under existing parent
	nonExistent := existingDir + "/does/not/exist.md"
	result = evalSymlinksWithFallback(nonExistent)
	// Should resolve parent and append remaining
	if !strings.HasSuffix(result, "/does/not/exist.md") {
		t.Errorf("evalSymlinksWithFallback(%q) should end with /does/not/exist.md, got %q", nonExistent, result)
	}
}
