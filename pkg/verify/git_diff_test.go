package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
			name:     "empty delta",
			delta:    "",
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
		{
			name: "skip event type names",
			delta: `### Evidence
- Old method: ` + "`hasCodeChanges=true`" + ` (incorrect)
- New method: ` + "`hasCodeChanges=false`" + ` (correct)
- Event: ` + "`session.created`" + ` plugin
- Event: ` + "`agent.spawned`" + ` event

### Files Modified
- ` + "`pkg/verify/test_evidence.go`" + ` - Fixed the bug`,
			expected: []string{"pkg/verify/test_evidence.go"},
		},
		{
			name: "skip version numbers",
			delta: `### Notes
- Using version ` + "`v0.33.2`" + ` of the library

### Files Modified
- ` + "`go.mod`" + ` - Updated dependency`,
			expected: []string{"go.mod"},
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
		// Valid file paths with known extensions
		{"pkg/verify/check.go", true},
		{"main.go", true},
		{".beads/beads.db", true},
		{"path/to/file.txt", true},
		{"README.md", true},
		{"config.yaml", true},
		{"./local/file.go", true},
		{".gitignore", true}, // dotfile with no extension
		{".env", true},       // dotfile with no extension
		{".env.local", true}, // dotfile with extension
		{"package.json", true},
		{"web/src/routes/page.svelte", true},
		{"web/src/lib/api.ts", true},
		{".kb/investigations/2026-01-08-test.md", true},

		// Event type names (the bug - these were matching as file paths)
		// These have . but NOT a known file extension
		{"session.created", false},
		{"agent.spawned", false},
		{"task.completed", false},
		{"Phase.Complete", false},
		{"worker.started", false},
		{"agent.completed", false},
		{"spawn.timeout", false},

		// Version numbers (not files)
		{"v0.33.2", false},
		{"v1.0.0", false},
		{"1.2.3", false},

		// Invalid paths - other reasons
		{"", false},                            // empty
		{"no-extension", false},                // no extension
		{"https://example.com/file.go", false}, // URL
		{"This is a sentence.", false},         // sentence (contains space)
		{"e.g.", false},                        // abbreviation
		{"i.e.", false},                        // abbreviation
		{"etc.", false},                        // abbreviation

		// Special patterns that should NOT match
		{"hasCodeChanges=true", false},               // assignment, contains =
		{"key=value.json", false},                    // assignment pattern, contains =
		{"og-work-dashboard-two-modes-27dec", false}, // workspace name (no extension)
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
	files, err := GetGitDiffFiles(tmpDir, beforeCommits, "")
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

// TestVerifyGitDiff_NoSpawnTime tests the fix for the false positive bug.
// When spawn_time file is missing (old workspaces), verification should pass
// with a warning instead of failing with false positives.
// See: orch-go-fhfhk - "git diff verification false positive"
func TestVerifyGitDiff_NoSpawnTime(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary git repository with commits
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

	// Create and commit files (simulating agent work)
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

	// Create workspace WITHOUT spawn_time file (simulating old workspace)
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "old-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create SYNTHESIS.md that claims the committed file
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

	// Note: We intentionally do NOT write spawn_time file

	// Test: should PASS with warning, not fail
	// Before the fix, this would fail because:
	// 1. ReadSpawnTime returns zero time (no file)
	// 2. GetGitDiffFiles uses "git diff --name-only HEAD" for zero time
	// 3. That returns empty (changes are committed, not uncommitted)
	// 4. Verification fails: claimed files not in diff
	result := VerifyGitDiff(workspaceDir, tmpDir)

	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass when spawn_time is missing, not fail with false positive")
		t.Errorf("Errors: %v", result.Errors)
	}

	// Should have a warning about missing spawn time or baseline
	hasSpawnTimeWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "spawn time and git baseline unavailable") {
			hasSpawnTimeWarning = true
			break
		}
	}
	if !hasSpawnTimeWarning {
		t.Error("Expected warning about spawn time being unavailable")
		t.Errorf("Warnings: %v", result.Warnings)
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

func TestIsExternalPath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// External paths - home directory
		{"~/external/file.ts", true},
		{"~/Documents/project/main.go", true},
		{"~/.config/settings.json", true},

		// External paths - absolute
		{"/Users/dylan/other-project/file.go", true},
		{"/etc/config.yaml", true},
		{"/tmp/test.txt", true},

		// External paths - relative traversal
		{"../other-project/file.go", true},
		{"../../parent/file.ts", true},
		{"path/to/../../../external.go", true},

		// Local paths - should NOT be external
		{"pkg/verify/check.go", false},
		{"main.go", false},
		{".beads/beads.db", false},
		{"./local/file.go", false},
		{".kb/investigations/test.md", false},
		{"web/src/routes/page.svelte", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsExternalPath(tt.input)
			if got != tt.expected {
				t.Errorf("IsExternalPath(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsWorkspaceArtifactPath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{input: ".orch/workspace/og-arch-test/SYNTHESIS.md", expected: true},
		{input: "./.orch/workspace/og-arch-test/render.yaml", expected: true},
		{input: ".orch/workspace-archive/og-arch-test/SYNTHESIS.md", expected: true},
		{input: ".orch/templates/SYNTHESIS.md", expected: false},
		{input: "pkg/verify/git_diff.go", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsWorkspaceArtifactPath(tt.input)
			if got != tt.expected {
				t.Errorf("IsWorkspaceArtifactPath(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	// Test home directory expansion
	path, err := ExpandPath("~/test.txt")
	if err != nil {
		t.Fatalf("ExpandPath() error = %v", err)
	}
	if path == "~/test.txt" || path == "" {
		t.Errorf("ExpandPath() should expand ~/, got %q", path)
	}

	// Test non-home path is unchanged
	path, err = ExpandPath("/absolute/path.go")
	if err != nil {
		t.Fatalf("ExpandPath() error = %v", err)
	}
	if path != "/absolute/path.go" {
		t.Errorf("ExpandPath() should not change absolute path, got %q", path)
	}

	// Test relative path is unchanged
	path, err = ExpandPath("relative/path.go")
	if err != nil {
		t.Fatalf("ExpandPath() error = %v", err)
	}
	if path != "relative/path.go" {
		t.Errorf("ExpandPath() should not change relative path, got %q", path)
	}
}

func TestVerifyExternalFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file with recent modification time
	testFile := filepath.Join(tmpDir, "test_external.go")
	if err := os.WriteFile(testFile, []byte("package test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test: file exists and was modified after spawn time
	spawnTime := time.Now().Add(-1 * time.Hour) // 1 hour ago
	result := VerifyExternalFile(testFile, spawnTime)
	if !result.Exists {
		t.Error("VerifyExternalFile() should report file exists")
	}
	if !result.Valid {
		t.Errorf("VerifyExternalFile() should pass for recent file: %s", result.Error)
	}

	// Test: file exists but was NOT modified after spawn time
	futureTime := time.Now().Add(1 * time.Hour) // 1 hour in future
	result = VerifyExternalFile(testFile, futureTime)
	if !result.Exists {
		t.Error("VerifyExternalFile() should report file exists")
	}
	if result.Valid {
		t.Error("VerifyExternalFile() should fail for file older than spawn time")
	}

	// Test: file does not exist
	result = VerifyExternalFile(filepath.Join(tmpDir, "nonexistent.go"), spawnTime)
	if result.Exists {
		t.Error("VerifyExternalFile() should report file does not exist")
	}
	if result.Valid {
		t.Error("VerifyExternalFile() should fail for non-existent file")
	}

	// Test: zero spawn time (should be valid)
	result = VerifyExternalFile(testFile, time.Time{})
	if !result.Valid {
		t.Error("VerifyExternalFile() should pass with zero spawn time")
	}
}

func TestVerifyGitDiff_ExternalFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Write spawn time from 1 hour ago
	spawnTime := time.Now().Add(-1 * time.Hour)
	if err := spawn.WriteSpawnTime(workspaceDir, spawnTime); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}

	// Create a recent external file
	externalFile := filepath.Join(tmpDir, "external_file.ts")
	if err := os.WriteFile(externalFile, []byte("// external"), 0644); err != nil {
		t.Fatalf("failed to create external file: %v", err)
	}

	// Create SYNTHESIS.md that claims the external file
	synthesisContent := `# Session Synthesis

## TLDR
Modified external file

## Delta (What Changed)

### Files Modified
- ` + "`" + externalFile + "`" + ` - External file
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	// Test: should pass - external file exists and was modified after spawn time
	result := VerifyGitDiff(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass for valid external file, errors: %v", result.Errors)
	}
	if len(result.ExternalFiles) != 1 {
		t.Errorf("Expected 1 external file, got: %v", result.ExternalFiles)
	}
	if len(result.InvalidExternalFiles) > 0 {
		t.Errorf("Expected no invalid external files, got: %v", result.InvalidExternalFiles)
	}
}

func TestVerifyGitDiff_WorkspaceArtifactsIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceName := "og-arch-design-state-07feb-abcd"
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", workspaceName)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	if err := spawn.WriteSpawnTime(workspaceDir, time.Now().Add(-30*time.Minute)); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}

	synthesisContent := `# Session Synthesis

## TLDR
Design-only session

## Delta (What Changed)

### Files Created
- ` + "`.orch/workspace/" + workspaceName + `/SYNTHESIS.md` + "`" + ` - Session synthesis artifact
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	result := VerifyGitDiff(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass when only workspace artifacts are claimed, errors: %v", result.Errors)
	}
	if len(result.IgnoredWorkspaceFiles) != 1 {
		t.Errorf("expected 1 ignored workspace file, got: %v", result.IgnoredWorkspaceFiles)
	}
	if len(result.MissingFromDiff) != 0 {
		t.Errorf("expected no missing files from diff, got: %v", result.MissingFromDiff)
	}
	if !strings.Contains(strings.Join(result.Warnings, "\n"), "workspace artifact file(s) skipped") {
		t.Errorf("expected warning about skipped workspace artifacts, got: %v", result.Warnings)
	}
}

func TestVerifyGitDiff_ExternalFileNotExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Write spawn time
	if err := spawn.WriteSpawnTime(workspaceDir, time.Now()); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}

	// Create SYNTHESIS.md that claims a non-existent external file
	synthesisContent := `# Session Synthesis

## TLDR
Modified external file

## Delta (What Changed)

### Files Modified
- ` + "`/nonexistent/path/file.go`" + ` - Does not exist
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	// Test: should fail - external file does not exist
	result := VerifyGitDiff(workspaceDir, tmpDir)
	if result.Passed {
		t.Error("VerifyGitDiff() should fail for non-existent external file")
	}
	if len(result.InvalidExternalFiles) != 1 {
		t.Errorf("Expected 1 invalid external file, got: %v", result.InvalidExternalFiles)
	}
}

func TestVerifyGitDiff_MixedLocalAndExternal(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

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

	time.Sleep(gitTimestampGranularity) // Required for git timestamps

	// Create and commit a local file
	localFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(localFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write local file: %v", err)
	}
	cmd = exec.Command("git", "add", "main.go")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "add main.go")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create an external file (outside the git repo)
	externalDir := t.TempDir() // Different temp dir
	externalFile := filepath.Join(externalDir, "external.ts")
	if err := os.WriteFile(externalFile, []byte("// external"), 0644); err != nil {
		t.Fatalf("failed to create external file: %v", err)
	}

	// Create SYNTHESIS.md that claims both local and external files
	synthesisContent := `# Session Synthesis

## TLDR
Modified local and external files

## Delta (What Changed)

### Files Modified
- ` + "`main.go`" + ` - Local file
- ` + "`" + externalFile + "`" + ` - External file
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	// Test: should pass - local file in git diff, external file exists with recent mtime
	result := VerifyGitDiff(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass for mixed local/external, errors: %v", result.Errors)
	}
	if len(result.ExternalFiles) != 1 {
		t.Errorf("Expected 1 external file, got: %v", result.ExternalFiles)
	}
	if len(result.MissingFromDiff) > 0 {
		t.Errorf("Expected no missing local files, got: %v", result.MissingFromDiff)
	}
}

func TestVerifyGitDiff_WithAgentManifest(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	cmd.Run()

	// Configure git user
	exec.Command("git", "config", "user.email", "test@test.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Create initial commit and get SHA
	initialFile := filepath.Join(tmpDir, "README.md")
	os.WriteFile(initialFile, []byte("# Initial"), 0644)
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	output, _ := cmd.Output()
	baselineSHA := strings.TrimSpace(string(output))

	// Create workspace
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	os.MkdirAll(workspaceDir, 0755)

	// Write AGENT_MANIFEST.json with baseline
	manifest := spawn.AgentManifest{
		WorkspaceName: "test-agent",
		GitBaseline:   baselineSHA,
		ProjectDir:    tmpDir,
	}
	spawn.WriteAgentManifest(workspaceDir, manifest)

	// Create and commit a file AFTER baseline
	mainFile := filepath.Join(tmpDir, "main.go")
	os.WriteFile(mainFile, []byte("package main"), 0644)
	cmd = exec.Command("git", "add", "main.go")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "add main.go")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create SYNTHESIS.md claiming main.go
	synthesisContent := `# Session Synthesis
## Delta
- ` + "`main.go`" + `
`
	os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644)

	// Test: should pass using baseline SHA even if spawnTime is zero
	result := VerifyGitDiff(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass using baseline SHA, errors: %v", result.Errors)
	}
}
