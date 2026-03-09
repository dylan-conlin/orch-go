package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckStagedAccretion(t *testing.T) {
	tmpDir := setupGitRepoForStaged(t)

	tests := []struct {
		name         string
		setupFiles   map[string]int // filename -> line count for initial commit
		stageFiles   map[string]int // filename -> new line count to stage
		expectPassed bool
		expectCount  int // number of blocked files
	}{
		{
			name: "small staged file passes",
			setupFiles: map[string]int{
				"small.go": 100,
			},
			stageFiles: map[string]int{
				"small.go": 200,
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "staged file at exactly 1500 lines passes",
			setupFiles: map[string]int{
				"exact.go": 100,
			},
			stageFiles: map[string]int{
				"exact.go": 1500,
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "staged file at 1501 lines blocks",
			setupFiles: map[string]int{
				"big.go": 100,
			},
			stageFiles: map[string]int{
				"big.go": 1501,
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "staged file at 2000 lines blocks",
			setupFiles: map[string]int{
				"huge.go": 100,
			},
			stageFiles: map[string]int{
				"huge.go": 2000,
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "new file over threshold blocks",
			setupFiles: map[string]int{},
			stageFiles: map[string]int{
				"newbig.go": 1600,
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "multiple files one over threshold",
			setupFiles: map[string]int{
				"ok.go": 100,
			},
			stageFiles: map[string]int{
				"ok.go":  500,
				"big.ts": 1800,
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "non-source file over threshold ignored",
			setupFiles: map[string]int{},
			stageFiles: map[string]int{
				"README.md": 2000,
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "vendor file over threshold ignored",
			setupFiles: map[string]int{},
			stageFiles: map[string]int{
				"vendor/lib/big.go": 2000,
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "typescript file over threshold blocks",
			setupFiles: map[string]int{},
			stageFiles: map[string]int{
				"app.ts": 1600,
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "svelte file over threshold blocks",
			setupFiles: map[string]int{},
			stageFiles: map[string]int{
				"Component.svelte": 1600,
			},
			expectPassed: false,
			expectCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanGitRepo(t, tmpDir)

			// Create and commit initial files
			for filename, lineCount := range tt.setupFiles {
				createFileWithLines(t, tmpDir, filename, lineCount)
			}
			if len(tt.setupFiles) > 0 {
				commitFiles(t, tmpDir, "Initial commit")
			}

			// Create/modify files and stage them
			for filename, lineCount := range tt.stageFiles {
				// Ensure parent directory exists
				dir := filepath.Dir(filepath.Join(tmpDir, filename))
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create dir %s: %v", dir, err)
				}
				createFileWithLines(t, tmpDir, filename, lineCount)
			}
			// Stage all changes
			stageAllFiles(t, tmpDir)

			result := CheckStagedAccretion(tmpDir)
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Passed != tt.expectPassed {
				t.Errorf("Passed = %v, want %v; blocked: %v", result.Passed, tt.expectPassed, result.BlockedFiles)
			}

			if len(result.BlockedFiles) != tt.expectCount {
				t.Errorf("expected %d blocked files, got %d: %v", tt.expectCount, len(result.BlockedFiles), result.BlockedFiles)
			}
		})
	}
}

func TestCheckStagedAccretion_EmptyStaging(t *testing.T) {
	tmpDir := setupGitRepoForStaged(t)

	// Create and commit a file, but don't stage anything new
	createFileWithLines(t, tmpDir, "existing.go", 2000)
	commitFiles(t, tmpDir, "Initial commit")

	result := CheckStagedAccretion(tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Passed {
		t.Error("expected pass when nothing is staged")
	}
}

func TestCheckStagedAccretion_DeletedFile(t *testing.T) {
	tmpDir := setupGitRepoForStaged(t)

	// Create and commit a large file
	createFileWithLines(t, tmpDir, "todelete.go", 2000)
	commitFiles(t, tmpDir, "Initial commit")

	// Delete and stage the deletion
	os.Remove(filepath.Join(tmpDir, "todelete.go"))
	cmd := exec.Command("git", "add", "todelete.go")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	result := CheckStagedAccretion(tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Passed {
		t.Error("expected pass when file is deleted")
	}
}

func TestCheckStagedAccretion_GeneratedFileIgnored(t *testing.T) {
	tmpDir := setupGitRepoForStaged(t)

	// Stage a large generated file
	createFileWithLines(t, tmpDir, "types.gen.go", 2000)
	stageAllFiles(t, tmpDir)

	result := CheckStagedAccretion(tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Passed {
		t.Errorf("expected pass for generated file, blocked: %v", result.BlockedFiles)
	}
}

// Helper functions

func setupGitRepoForStaged(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "accretion-staged-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	if err := initGitRepo(tmpDir); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Need at least one commit for git diff --cached to work
	placeholder := filepath.Join(tmpDir, ".gitkeep")
	if err := os.WriteFile(placeholder, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create .gitkeep: %v", err)
	}
	commitFiles(t, tmpDir, "Initial setup")

	return tmpDir
}

func stageAllFiles(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
}

// createFileInSubdir creates a file with the given line count, creating parent dirs as needed.
func createFileInSubdir(t *testing.T, dir, relPath string, lineCount int) {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	parent := filepath.Dir(fullPath)
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", parent, err)
	}

	lines := make([]string, lineCount)
	for i := 0; i < lineCount; i++ {
		lines[i] = "// Line " + string(rune('0'+i%10))
	}
	content := strings.Join(lines, "\n") + "\n"

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file %s: %v", relPath, err)
	}
}
