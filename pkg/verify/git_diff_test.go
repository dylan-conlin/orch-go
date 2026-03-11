package verify

import (
	"encoding/json"
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

	// Create initial commit with pre-existing files
	initialFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	// Also create pkg files that exist in the project before agent spawns
	os.MkdirAll(filepath.Join(tmpDir, "pkg"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "pkg", "unchanged.go"), []byte("package pkg\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "pkg", "also_unchanged.go"), []byte("package pkg\n"), 0644)
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial commit with pkg files")
	cmd.Dir = tmpDir
	cmd.Run()

	// Record baseline (agent spawns here — files already exist)
	baselineCmd := exec.Command("git", "rev-parse", "HEAD")
	baselineCmd.Dir = tmpDir
	baselineOut, _ := baselineCmd.Output()
	baseline := strings.TrimSpace(string(baselineOut))

	// Create workspace with baseline
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Write manifest with baseline
	manifest := spawn.AgentManifest{GitBaseline: baseline, WorkspaceName: "test-agent", ProjectDir: tmpDir}
	manifestData, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(workspaceDir, "AGENT_MANIFEST.json"), manifestData, 0644)

	// Agent makes some other commit (so there IS a diff, just not for the claimed files)
	os.WriteFile(filepath.Join(tmpDir, "new_file.go"), []byte("package main\n"), 0644)
	cmd = exec.Command("git", "add", "new_file.go")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "agent adds new file")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create SYNTHESIS.md that claims files that exist but aren't in the diff
	synthesisContent := `# Session Synthesis

## TLDR
Made changes

## Delta (What Changed)

### Files Modified
- ` + "`pkg/unchanged.go`" + ` - This file was never modified by agent
- ` + "`pkg/also_unchanged.go`" + ` - This one too
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	// Test: should fail - claiming files that exist in project but aren't in diff
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

func TestVerifyGitDiff_CrossRepoBaselineDiscarded(t *testing.T) {
	// When an agent is spawned from repo A to work in repo B, the AGENT_MANIFEST.json
	// contains a git baseline SHA from repo A. Using that SHA in repo B fails because
	// the SHA doesn't exist there. Verification should discard the baseline and fall
	// back to spawn time when manifest.ProjectDir != the passed projectDir.

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create "spawning repo" (repo A - e.g., orch-go)
	repoA := t.TempDir()
	runGit := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v in %s failed: %v", args, filepath.Base(dir), err)
		}
	}

	runGit(repoA, "init")
	os.WriteFile(filepath.Join(repoA, "README.md"), []byte("# Repo A"), 0644)
	runGit(repoA, "add", "README.md")
	runGit(repoA, "commit", "-m", "initial repo A")

	// Get baseline SHA from repo A
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoA
	output, _ := cmd.Output()
	repoABaseline := strings.TrimSpace(string(output))

	// Create "target repo" (repo B - e.g., skillc)
	repoB := t.TempDir()
	runGit(repoB, "init")
	os.WriteFile(filepath.Join(repoB, "README.md"), []byte("# Repo B"), 0644)
	runGit(repoB, "add", "README.md")
	runGit(repoB, "commit", "-m", "initial repo B")

	// Sleep to ensure spawn time boundary
	time.Sleep(gitTimestampGranularity)
	spawnTime := time.Now()

	// Simulate agent committing work in repo B after spawn
	time.Sleep(gitTimestampGranularity)
	os.WriteFile(filepath.Join(repoB, "skill.go"), []byte("package skill"), 0644)
	runGit(repoB, "add", "skill.go")
	runGit(repoB, "commit", "-m", "add skill.go")

	// Create workspace (lives under repo A's .orch/workspace/)
	workspaceDir := filepath.Join(repoA, ".orch", "workspace", "test-cross-repo")
	os.MkdirAll(workspaceDir, 0755)

	// Write AGENT_MANIFEST.json: ProjectDir=repoA (spawning repo), baseline=repoA SHA
	manifest := spawn.AgentManifest{
		WorkspaceName: "test-cross-repo",
		GitBaseline:   repoABaseline,
		ProjectDir:    repoA,
		SpawnTime:     spawnTime.Format(time.RFC3339),
	}
	spawn.WriteAgentManifest(workspaceDir, manifest)

	// Create SYNTHESIS.md claiming skill.go
	synthesisContent := "# Session Synthesis\n## Delta\n- `skill.go`\n"
	os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644)

	// Test: VerifyGitDiff called with projectDir=repoB (the target repo)
	// should discard the repoA baseline and use spawn time instead
	result := VerifyGitDiff(workspaceDir, repoB)
	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass for cross-repo (discard mismatched baseline), errors: %v", result.Errors)
	}
	if len(result.ActualFiles) == 0 {
		t.Errorf("VerifyGitDiff() should find skill.go in actual diff, actualFiles: %v", result.ActualFiles)
	}
}

// TestVerifyGitDiff_NonexistentLocalFilesDowngradedToWarning verifies that when
// SYNTHESIS claims local-looking files that don't exist in the project, they're
// downgraded to warnings instead of errors. This handles the cross-repo false positive:
// agents working across repos may claim files from another repo that look like local paths.
func TestVerifyGitDiff_NonexistentLocalFilesDowngradedToWarning(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()

	// Initialize git repo
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init")
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	run("git", "add", "main.go")
	run("git", "commit", "-m", "initial")

	// Create workspace
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	os.MkdirAll(workspaceDir, 0755)
	spawnTime := time.Now()
	spawn.WriteSpawnTime(workspaceDir, spawnTime)

	time.Sleep(gitTimestampGranularity)

	// Agent modifies local file
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nfunc main() {}"), 0644)
	run("git", "add", "main.go")
	run("git", "commit", "-m", "feat: update main")

	// SYNTHESIS claims local file AND a file from another repo (that doesn't exist here)
	synthesisContent := `# Session Synthesis

## TLDR
Cross-repo work

## Delta (What Changed)

### Files Modified
- ` + "`main.go`" + ` - Updated main
- ` + "`packages/opencode/src/session.ts`" + ` - Fixed session handling in opencode fork
`
	os.WriteFile(filepath.Join(workspaceDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644)

	result := VerifyGitDiff(workspaceDir, tmpDir)
	// Should pass — the nonexistent claimed file should be downgraded to warning
	if !result.Passed {
		t.Errorf("VerifyGitDiff() should pass when missing files don't exist in project (cross-repo), errors: %v", result.Errors)
	}
	// Should have a warning about the nonexistent file
	hasWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "packages/opencode/src/session.ts") || strings.Contains(w, "not found in project") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Errorf("expected warning about cross-repo file, got warnings: %v", result.Warnings)
	}
}
