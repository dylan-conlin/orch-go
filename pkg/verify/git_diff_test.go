package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// gitTimestampGranularity is the sleep duration needed to ensure git commit
// timestamps are distinct. Git commit timestamps have second granularity,
// so we sleep 1.1 seconds to guarantee a different timestamp.
// This is a known limitation of git's timestamp resolution.
const gitTimestampGranularity = 1100 * time.Millisecond

func TestParseDeltaFiles(t *testing.T) {
	tests := []struct {
		name     string
		delta    string
		expected []string
	}{
		{
			name: "backtick quoted files",
			delta: `### Files Modified
- ` + "`pkg/verify/check.go`" + ` - Added git diff verification
- ` + "`pkg/verify/git_diff.go`" + ` - New verification module`,
			expected: []string{"pkg/verify/check.go", "pkg/verify/git_diff.go"},
		},
		{
			name: "bold paths",
			delta: `### Files Modified
- **pkg/verify/check.go** - Added git diff verification`,
			expected: []string{"pkg/verify/check.go"},
		},
		{
			name: "mixed formats backtick and bold",
			delta: `### Files Created
- ` + "`pkg/new_file.go`" + ` - New file

### Files Modified
- ` + "`pkg/existing.go`" + ` - Modified existing
- **pkg/another.go** - Another file`,
			expected: []string{"pkg/new_file.go", "pkg/existing.go", "pkg/another.go"},
		},
		{
			name:  "empty delta",
			delta: "",
			expected: nil,
		},
		{
			name: "no files section",
			delta: `### Commits
- ` + "`abc123`" + ` - Some commit message`,
			expected: nil,
		},
		{
			name: "database files",
			delta: `### Files Modified
- ` + "`.beads/beads.db`" + ` - Updated database`,
			expected: []string{".beads/beads.db"},
		},
		{
			name: "skip URLs",
			delta: `See https://example.com/path/to/file.go for details
- ` + "`pkg/verify/check.go`" + ` - Real file`,
			expected: []string{"pkg/verify/check.go"},
		},
		{
			name: "skip sentences",
			delta: `This is a sentence.
- ` + "`pkg/verify/check.go`" + ` - Real file`,
			expected: []string{"pkg/verify/check.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synthesis := &Synthesis{Delta: tt.delta}
			got := ParseDeltaFiles(synthesis)

			if len(got) != len(tt.expected) {
				t.Errorf("ParseDeltaFiles() got %d files, want %d", len(got), len(tt.expected))
				t.Errorf("Got: %v", got)
				t.Errorf("Expected: %v", tt.expected)
				return
			}

			// Create a map of expected files for easy lookup
			expectedMap := make(map[string]bool)
			for _, f := range tt.expected {
				expectedMap[f] = true
			}

			for _, file := range got {
				if !expectedMap[file] {
					t.Errorf("ParseDeltaFiles() got unexpected file %q", file)
				}
			}
		})
	}
}

func TestParseDeltaFiles_NilSynthesis(t *testing.T) {
	got := ParseDeltaFiles(nil)
	if got != nil {
		t.Errorf("ParseDeltaFiles(nil) = %v, want nil", got)
	}
}

func TestIsLikelyFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Valid file paths
		{"pkg/verify/check.go", true},
		{"main.go", true},
		{".beads/beads.db", true},
		{"path/to/file.txt", true},
		{"README.md", true},

		// Invalid paths
		{"", false},                                  // empty
		{"no-extension", false},                      // no extension
		{"https://example.com/file.go", false},       // URL
		{"This is a sentence.", false},               // sentence (contains space)
		{"e.g.", false},                              // abbreviation
		{"i.e.", false},                              // abbreviation
		{"etc.", false},                              // abbreviation
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isLikelyFilePath(tt.input)
			if got != tt.expected {
				t.Errorf("isLikelyFilePath(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"./path/to/file.go", "path/to/file.go"},
		{"/path/to/file.go", "path/to/file.go"},
		{"path/to/file.go", "path/to/file.go"},
		{"path\\to\\file.go", "path/to/file.go"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePath(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetGitDiffFiles(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	initialFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Record time before new commits
	beforeCommits := time.Now()
	time.Sleep(gitTimestampGranularity) // Required: git commits have second-precision timestamps

	// Create a new file and commit
	newFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(newFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}
	cmd = exec.Command("git", "add", "main.go")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "add main.go")
	cmd.Dir = tmpDir
	cmd.Run()

	// Test: should find main.go in diff since beforeCommits
	files, err := GetGitDiffFiles(tmpDir, beforeCommits)
	if err != nil {
		t.Fatalf("GetGitDiffFiles() error = %v", err)
	}

	found := false
	for _, f := range files {
		if f == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GetGitDiffFiles() should include main.go, got: %v", files)
	}
}

func TestVerifyGitDiff_ClaimsMatchDiff(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	initialFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create workspace
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Write spawn time
	spawnTime := time.Now()
	if err := spawn.WriteSpawnTime(workspaceDir, spawnTime); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}

	time.Sleep(gitTimestampGranularity) // Required: git commits have second-precision timestamps

	// Create and commit a file
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	cmd = exec.Command("git", "add", "main.go")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "add main.go")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create SYNTHESIS.md that correctly claims main.go
	synthesisContent := `# Session Synthesis

## TLDR
Added main.go

## Delta (What Changed)

### Files Created
- ` + "`main.go`" + ` - New main file
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	// Test: should pass - claimed file matches diff
	result := VerifyGitDiff(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass when claims match diff, errors: %v", result.Errors)
	}
	if len(result.MissingFromDiff) > 0 {
		t.Errorf("Expected no missing files, got: %v", result.MissingFromDiff)
	}
}

func TestVerifyGitDiff_ClaimsFilesNotInDiff(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	initialFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create workspace
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Write spawn time
	spawnTime := time.Now()
	if err := spawn.WriteSpawnTime(workspaceDir, spawnTime); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}

	// Create SYNTHESIS.md that claims files that don't exist in git diff
	synthesisContent := `# Session Synthesis

## TLDR
Made changes

## Delta (What Changed)

### Files Modified
- ` + "`pkg/nonexistent.go`" + ` - This file was never modified
- ` + "`pkg/another.go`" + ` - This one too
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	// Test: should fail - claiming files that aren't in diff
	result := VerifyGitDiff(workspaceDir, tmpDir)
	if result.Passed {
		t.Error("VerifyGitDiff() should fail when claims don't match diff")
	}
	if len(result.MissingFromDiff) != 2 {
		t.Errorf("Expected 2 missing files, got: %v", result.MissingFromDiff)
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for false positive detection")
	}
}

func TestVerifyGitDiff_NoSynthesis(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace without SYNTHESIS.md
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Test: should pass (with warning) - no SYNTHESIS.md to verify
	result := VerifyGitDiff(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass when no SYNTHESIS.md, errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warning about missing SYNTHESIS.md")
	}
}

func TestVerifyGitDiff_EmptyDelta(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace with SYNTHESIS.md but empty Delta
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	synthesisContent := `# Session Synthesis

## TLDR
Investigation complete

## Delta (What Changed)

No code changes (investigation only).
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	// Test: should pass - no files claimed, nothing to verify
	result := VerifyGitDiff(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass when no files claimed, errors: %v", result.Errors)
	}
}

func TestVerifyGitDiffForCompletion(t *testing.T) {
	tmpDir := t.TempDir()

	// Test: empty workspace path returns nil
	result := VerifyGitDiffForCompletion("", tmpDir)
	if result != nil {
		t.Error("VerifyGitDiffForCompletion() should return nil for empty workspace")
	}

	// Test: empty project dir returns nil
	result = VerifyGitDiffForCompletion(tmpDir, "")
	if result != nil {
		t.Error("VerifyGitDiffForCompletion() should return nil for empty project dir")
	}

	// Create workspace with no files claimed
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	synthesisContent := `# Session Synthesis

## Delta (What Changed)

No changes.
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	// Test: no files claimed returns nil
	result = VerifyGitDiffForCompletion(workspaceDir, tmpDir)
	if result != nil {
		t.Error("VerifyGitDiffForCompletion() should return nil when no files claimed")
	}
}
