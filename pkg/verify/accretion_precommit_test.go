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
			name: "staged file at 1501 lines blocks when agent caused it",
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
			name: "staged file at 2000 lines blocks when agent caused it",
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
			name: "pre-existing bloat file warns instead of blocking",
			setupFiles: map[string]int{
				"legacy.go": 1600,
			},
			stageFiles: map[string]int{
				"legacy.go": 1650,
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "pre-existing bloat at exactly 1501 warns instead of blocking",
			setupFiles: map[string]int{
				"border.go": 1501,
			},
			stageFiles: map[string]int{
				"border.go": 1560,
			},
			expectPassed: true,
			expectCount:  0,
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

func TestCheckStagedAccretion_Warning800Threshold(t *testing.T) {
	tmpDir := setupGitRepoForStaged(t)

	tests := []struct {
		name           string
		initialLines   int
		stagedLines    int
		expectWarnings int
	}{
		{
			name:           "801 lines with +30 net delta triggers warning",
			initialLines:   770,
			stagedLines:    801,
			expectWarnings: 1,
		},
		{
			name:           "850 lines with +29 net delta no warning",
			initialLines:   821,
			stagedLines:    850,
			expectWarnings: 0,
		},
		{
			name:           "900 lines with +50 net delta triggers warning",
			initialLines:   850,
			stagedLines:    900,
			expectWarnings: 1,
		},
		{
			name:           "800 lines exactly does not trigger (must be >800)",
			initialLines:   770,
			stagedLines:    800,
			expectWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanGitRepo(t, tmpDir)

			createFileWithLines(t, tmpDir, "growing.go", tt.initialLines)
			commitFiles(t, tmpDir, "Initial commit")

			createFileWithLines(t, tmpDir, "growing.go", tt.stagedLines)
			stageAllFiles(t, tmpDir)

			result := CheckStagedAccretion(tmpDir)
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if !result.Passed {
				t.Error("800-threshold warnings should not block the commit")
			}
			if len(result.WarningFiles) != tt.expectWarnings {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectWarnings, len(result.WarningFiles), result.WarningFiles)
			}
		})
	}
}

func TestCheckStagedAccretion_Warning600Threshold(t *testing.T) {
	tmpDir := setupGitRepoForStaged(t)

	tests := []struct {
		name           string
		initialLines   int
		stagedLines    int
		expectWarnings int
	}{
		{
			name:           "650 lines with +50 net delta triggers warning",
			initialLines:   600,
			stagedLines:    650,
			expectWarnings: 1,
		},
		{
			name:           "650 lines with +49 net delta no warning",
			initialLines:   601,
			stagedLines:    650,
			expectWarnings: 0,
		},
		{
			name:           "700 lines with +80 net delta triggers warning",
			initialLines:   620,
			stagedLines:    700,
			expectWarnings: 1,
		},
		{
			name:           "600 lines exactly does not trigger (must be >600)",
			initialLines:   550,
			stagedLines:    600,
			expectWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanGitRepo(t, tmpDir)

			createFileWithLines(t, tmpDir, "medium.go", tt.initialLines)
			commitFiles(t, tmpDir, "Initial commit")

			createFileWithLines(t, tmpDir, "medium.go", tt.stagedLines)
			stageAllFiles(t, tmpDir)

			result := CheckStagedAccretion(tmpDir)
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if !result.Passed {
				t.Error("600-threshold warnings should not block the commit")
			}
			if len(result.WarningFiles) != tt.expectWarnings {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectWarnings, len(result.WarningFiles), result.WarningFiles)
			}
		})
	}
}

func TestCheckStagedAccretion_NewFileWarnings(t *testing.T) {
	tmpDir := setupGitRepoForStaged(t)

	// New file at 850 lines (all net new) should trigger 800-threshold warning
	cleanGitRepo(t, tmpDir)
	createFileWithLines(t, tmpDir, "newbig.go", 850)
	stageAllFiles(t, tmpDir)

	result := CheckStagedAccretion(tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Passed {
		t.Error("850-line new file should warn, not block")
	}
	if len(result.WarningFiles) != 1 {
		t.Errorf("expected 1 warning for new 850-line file, got %d", len(result.WarningFiles))
	}
}

func TestCheckStagedAccretion_PreExistingBloatWarning(t *testing.T) {
	tmpDir := setupGitRepoForStaged(t)

	// File already 1700 lines in HEAD — agent adds 50 more
	createFileWithLines(t, tmpDir, "legacy.go", 1700)
	commitFiles(t, tmpDir, "Initial commit")

	createFileWithLines(t, tmpDir, "legacy.go", 1750)
	stageAllFiles(t, tmpDir)

	result := CheckStagedAccretion(tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Pre-existing bloat: should warn, NOT block
	if !result.Passed {
		t.Errorf("expected Passed=true for pre-existing bloat, got blocked: %v", result.BlockedFiles)
	}
	if len(result.BlockedFiles) != 0 {
		t.Errorf("expected 0 blocked files for pre-existing bloat, got %d: %v", len(result.BlockedFiles), result.BlockedFiles)
	}
	if len(result.WarningFiles) != 1 {
		t.Errorf("expected 1 warning for pre-existing bloat, got %d: %v", len(result.WarningFiles), result.WarningFiles)
	}
}

func TestCheckStagedAccretion_BlockTakesPrecedenceOverWarning(t *testing.T) {
	tmpDir := setupGitRepoForStaged(t)

	// File at 1600 lines should be in BlockedFiles, not WarningFiles
	createFileWithLines(t, tmpDir, "huge.go", 100)
	commitFiles(t, tmpDir, "Initial commit")

	createFileWithLines(t, tmpDir, "huge.go", 1600)
	stageAllFiles(t, tmpDir)

	result := CheckStagedAccretion(tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Passed {
		t.Error("1600-line file should block")
	}
	if len(result.BlockedFiles) != 1 {
		t.Errorf("expected 1 blocked file, got %d", len(result.BlockedFiles))
	}
	if len(result.WarningFiles) != 0 {
		t.Errorf("expected 0 warnings (blocked takes precedence), got %d", len(result.WarningFiles))
	}
}

func TestFormatStagedAccretionWarnings(t *testing.T) {
	// nil result
	if msg := FormatStagedAccretionWarnings(nil); msg != "" {
		t.Errorf("expected empty for nil, got %q", msg)
	}

	// No warnings
	result := &StagedAccretionResult{Passed: true}
	if msg := FormatStagedAccretionWarnings(result); msg != "" {
		t.Errorf("expected empty for no warnings, got %q", msg)
	}

	// With growth warnings (800/600 threshold)
	result = &StagedAccretionResult{
		Passed: true,
		WarningFiles: []StagedFileInfo{
			{Path: "big.go", Lines: 850, NetDelta: 40, Threshold: 800},
		},
	}
	msg := FormatStagedAccretionWarnings(result)
	if msg == "" {
		t.Fatal("expected non-empty warning message")
	}
	if !strings.Contains(msg, "big.go") {
		t.Errorf("warning should contain filename, got: %s", msg)
	}
	if !strings.Contains(msg, "approaching") {
		t.Errorf("growth warning should contain 'approaching', got: %s", msg)
	}

	// With pre-existing bloat warning (1500 threshold)
	result = &StagedAccretionResult{
		Passed: true,
		WarningFiles: []StagedFileInfo{
			{Path: "legacy.go", Lines: 1700, NetDelta: 50, Threshold: AccretionCriticalThreshold},
		},
	}
	msg = FormatStagedAccretionWarnings(result)
	if msg == "" {
		t.Fatal("expected non-empty warning for pre-existing bloat")
	}
	if !strings.Contains(msg, "pre-existing bloat") {
		t.Errorf("pre-existing bloat warning should contain 'pre-existing bloat', got: %s", msg)
	}
	if !strings.Contains(msg, "legacy.go") {
		t.Errorf("warning should contain filename, got: %s", msg)
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
