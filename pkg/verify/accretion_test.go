package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyAccretionForCompletion(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "accretion-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	if err := initGitRepo(tmpDir); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	tests := []struct {
		name           string
		setupFiles     map[string]int // filename -> line count for initial commit
		modifyFiles    map[string]fileModification
		expectPassed   bool
		expectErrors   int
		expectWarnings int
	}{
		{
			name: "small file with small change passes",
			setupFiles: map[string]int{
				"small.go": 100,
			},
			modifyFiles: map[string]fileModification{
				"small.go": {addLines: 10, removeLines: 0},
			},
			expectPassed:   true,
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "large file with small change passes",
			setupFiles: map[string]int{
				"medium.go": 900,
			},
			modifyFiles: map[string]fileModification{
				"medium.go": {addLines: 30, removeLines: 0},
			},
			expectPassed:   true,
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "file >800 lines with +50 net lines triggers warning",
			setupFiles: map[string]int{
				"medium.go": 850,
			},
			modifyFiles: map[string]fileModification{
				"medium.go": {addLines: 60, removeLines: 5},
			},
			expectPassed:   true,
			expectErrors:   0,
			expectWarnings: 1,
		},
		{
			name: "pre-existing bloated file >1500 lines downgrades to warning",
			setupFiles: map[string]int{
				"large.go": 1600,
			},
			modifyFiles: map[string]fileModification{
				"large.go": {addLines: 70, removeLines: 10},
			},
			expectPassed:   true,
			expectErrors:   0,
			expectWarnings: 1, // pre-existing bloat warning
		},
		{
			name: "extraction work (net negative delta) passes",
			setupFiles: map[string]int{
				"bloated.go": 2000,
			},
			modifyFiles: map[string]fileModification{
				"bloated.go": {addLines: 50, removeLines: 200},
			},
			expectPassed:   true,
			expectErrors:   0,
			expectWarnings: 1, // extraction auto-pass warning
		},
		{
			name: "multiple files, mixed results (pre-existing bloat is warning)",
			setupFiles: map[string]int{
				"ok.go":       500,
				"warning.go":  900,
				"critical.go": 1700,
			},
			modifyFiles: map[string]fileModification{
				"ok.go":       {addLines: 20, removeLines: 0},
				"warning.go":  {addLines: 60, removeLines: 5},
				"critical.go": {addLines: 80, removeLines: 10},
			},
			expectPassed:   true,
			expectErrors:   0,
			expectWarnings: 2, // critical.go (pre-existing bloat) + warning.go
		},
		{
			name: "net negative across all files passes",
			setupFiles: map[string]int{
				"extract_from.go": 1800,
				"extract_to.go":   100,
			},
			modifyFiles: map[string]fileModification{
				"extract_from.go": {addLines: 50, removeLines: 500},
				"extract_to.go":   {addLines: 400, removeLines: 0},
			},
			expectPassed:   true,
			expectErrors:   0,
			expectWarnings: 1, // extraction auto-pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any changes from previous test
			cleanGitRepo(t, tmpDir)

			// Create and commit initial files
			for filename, lineCount := range tt.setupFiles {
				createFileWithLines(t, tmpDir, filename, lineCount)
			}
			commitFiles(t, tmpDir, "Initial commit")

			// Modify files (unstaged changes)
			for filename, mod := range tt.modifyFiles {
				modifyFile(t, tmpDir, filename, mod.addLines, mod.removeLines)
			}

			// Run verification
			result := VerifyAccretionForCompletion(tmpDir, tmpDir)

			// Check results
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Passed != tt.expectPassed {
				t.Errorf("expected Passed=%v, got %v", tt.expectPassed, result.Passed)
			}

			if len(result.Errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}

			if len(result.Warnings) != tt.expectWarnings {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectWarnings, len(result.Warnings), result.Warnings)
			}
		})
	}
}

func TestVerifyAccretionForCompletion_NilInputs(t *testing.T) {
	// Both empty workspace and projectDir should return nil
	result := VerifyAccretionForCompletion("", "/some/dir")
	if result != nil {
		t.Error("expected nil when workspacePath is empty")
	}

	result = VerifyAccretionForCompletion("/some/workspace", "")
	if result != nil {
		t.Error("expected nil when projectDir is empty")
	}

	result = VerifyAccretionForCompletion("", "")
	if result != nil {
		t.Error("expected nil when both are empty")
	}
}

func TestVerifyAccretionForCompletion_BoundaryValues(t *testing.T) {
	// Test boundary values: exactly 800, exactly 1500, exactly 50 delta
	tmpDir, err := os.MkdirTemp("", "accretion-boundary-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := initGitRepo(tmpDir); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Note: the accretion check uses currentLines (post-modification total), not initial size.
	// Threshold checks: currentLines > 800 (warning), currentLines > 1500 (critical).
	// Delta threshold: netDelta >= 50 (AccretionDeltaThreshold).
	// So "initial 1450 + 50 added = 1500 current" is NOT > 1500, no error.
	// And "initial 1452 + 50 added = ~1502 current" IS > 1500, error.
	tests := []struct {
		name         string
		initialLines int
		addLines     int
		removeLines  int
		expectPassed bool
		expectErrors int
	}{
		{
			name:         "current ~850 lines (800+50) = warning zone (above 800)",
			initialLines: 800,
			addLines:     50,
			removeLines:  0,
			expectPassed: true,
			expectErrors: 0,
		},
		{
			name:         "49 net additions = below delta threshold, no check",
			initialLines: 900,
			addLines:     49,
			removeLines:  0,
			expectPassed: true,
			expectErrors: 0,
		},
		{
			name:         "current ~1450 lines (1400+50) = warning only, not critical",
			initialLines: 1400,
			addLines:     50,
			removeLines:  0,
			expectPassed: true,
			expectErrors: 0,
		},
		{
			name:         "current ~1550 lines (1500+50) = advisory warning (above 1500)",
			initialLines: 1500,
			addLines:     50,
			removeLines:  0,
			expectPassed: true, // advisory: never blocks
			expectErrors: 0,
		},
		{
			name:         "current ~1551 lines (1501+50) = advisory warning (well above 1500)",
			initialLines: 1501,
			addLines:     50,
			removeLines:  0,
			expectPassed: true, // advisory: never blocks
			expectErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanGitRepo(t, tmpDir)

			createFileWithLines(t, tmpDir, "boundary.go", tt.initialLines)
			commitFiles(t, tmpDir, "Initial commit")
			modifyFile(t, tmpDir, "boundary.go", tt.addLines, tt.removeLines)

			result := VerifyAccretionForCompletion(tmpDir, tmpDir)
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Passed != tt.expectPassed {
				t.Errorf("Passed = %v, want %v (errors: %v, warnings: %v)",
					result.Passed, tt.expectPassed, result.Errors, result.Warnings)
			}

			if len(result.Errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}
		})
	}
}

func TestVerifyAccretionForCompletion_NonGoSourceFiles(t *testing.T) {
	// Accretion check should work for non-Go source files too (e.g., .ts, .py)
	tmpDir, err := os.MkdirTemp("", "accretion-nongo-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := initGitRepo(tmpDir); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Create a large TypeScript file (pre-existing bloat - already >1500 before agent)
	createFileWithLines(t, tmpDir, "large.ts", 1600)
	commitFiles(t, tmpDir, "Initial commit")

	// Add significant lines
	modifyFile(t, tmpDir, "large.ts", 60, 5)

	result := VerifyAccretionForCompletion(tmpDir, tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Pre-existing bloat should pass (downgraded to warning), not block
	if !result.Passed {
		t.Errorf("expected Passed=true for pre-existing bloated .ts file, got errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected at least one warning for pre-existing bloated .ts file")
	}
}

func TestVerifyAccretionForCompletion_ExcludesVendorFiles(t *testing.T) {
	// Files in vendor/ should be excluded from accretion checks
	tmpDir, err := os.MkdirTemp("", "accretion-vendor-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := initGitRepo(tmpDir); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Create vendor directory and a large file
	vendorDir := filepath.Join(tmpDir, "vendor", "pkg")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("failed to create vendor dir: %v", err)
	}

	// Create a large vendor file
	createFileWithLines(t, filepath.Join(tmpDir, "vendor", "pkg"), "lib.go", 2000)
	commitFiles(t, tmpDir, "Initial commit")

	// Add significant lines to vendor file
	modifyFile(t, filepath.Join(tmpDir, "vendor", "pkg"), "lib.go", 100, 5)

	result := VerifyAccretionForCompletion(tmpDir, tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Should pass because vendor files are excluded
	if !result.Passed {
		t.Errorf("expected Passed=true for vendor file, got errors: %v", result.Errors)
	}
}

func TestVerifyAccretionForCompletion_AgentCausedBloat(t *testing.T) {
	// Advisory: agent pushing file over threshold produces warning, not error
	tmpDir, err := os.MkdirTemp("", "accretion-agent-bloat-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := initGitRepo(tmpDir); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// File starts at 1400 lines (below critical threshold)
	createFileWithLines(t, tmpDir, "growing.go", 1400)
	commitFiles(t, tmpDir, "Initial commit")

	// Agent adds 120 lines → pushes to ~1520 (over 1500 threshold)
	modifyFile(t, tmpDir, "growing.go", 120, 0)

	result := VerifyAccretionForCompletion(tmpDir, tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Advisory gate: always passes, emits warning instead of error
	if !result.Passed {
		t.Error("advisory gate should always pass")
	}
	if len(result.Errors) != 0 {
		t.Errorf("advisory gate should produce 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected advisory warning for agent-caused bloat")
	}
}

func TestVerifyAccretionForCompletion_PreExistingBloatDetailed(t *testing.T) {
	// Detailed test: file already >1500 lines before agent's changes should NOT block
	tmpDir, err := os.MkdirTemp("", "accretion-preexisting-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := initGitRepo(tmpDir); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// File starts at 2000 lines (well above critical threshold)
	createFileWithLines(t, tmpDir, "legacy.go", 2000)
	commitFiles(t, tmpDir, "Initial commit")

	// Agent adds 100 lines to an already-bloated file
	modifyFile(t, tmpDir, "legacy.go", 100, 0)

	result := VerifyAccretionForCompletion(tmpDir, tmpDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Pre-existing bloat: should pass with warning, NOT block
	if !result.Passed {
		t.Errorf("expected Passed=true for pre-existing bloat, got errors: %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors for pre-existing bloat, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning for pre-existing bloat, got %d: %v", len(result.Warnings), result.Warnings)
	}

	// Verify warning message contains "pre-existing bloat"
	if len(result.Warnings) > 0 && !strings.Contains(result.Warnings[0], "pre-existing bloat") {
		t.Errorf("expected warning to contain 'pre-existing bloat', got: %s", result.Warnings[0])
	}
}

func TestIsSourceFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"pkg/verify/check.go", true},
		{"web/app.ts", true},
		{"web/component.tsx", true},
		{"src/main.py", true},
		{"lib/util.rb", true},
		{"vendor/pkg/lib.go", false},
		{"node_modules/lib/file.ts", false},
		{"dist/bundle.js", false},
		{"build/output.go", false},
		// Tool/workspace directories (deployed copies, not source code)
		{".opencode/plugin/coaching.ts", false},
		{".opencode/plugin/slow-find-warn.ts", false},
		{".orch/workspace/agent-123/SYNTHESIS.md", false},
		{".beads/hooks/on_close", false},
		// Build output directories
		{".svelte-kit/output/server/chunks/index.js", false},
		{"__pycache__/module.py", false},
		{".next/server/pages/index.js", false},
		{".nuxt/dist/server.js", false},
		{".output/server/index.js", false},
		{"types.gen.go", false},
		{"api.gen.ts", false},
		{"proto.pb.go", false},
		{"README.md", false},
		{"config.json", false},
		{"styles.css", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isSourceFile(tt.path)
			if got != tt.expected {
				t.Errorf("isSourceFile(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestGetFileLineCount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "linecount-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file with known line count
	testFile := filepath.Join(tmpDir, "test.go")
	lines := []string{
		"package main",
		"",
		"func main() {",
		"  println(\"hello\")",
		"}",
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test line count
	count, err := getFileLineCount(tmpDir, "test.go")
	if err != nil {
		t.Fatalf("getFileLineCount failed: %v", err)
	}

	// wc -l counts newlines, so 5 lines = 4 newlines (last line may not have newline)
	// Actual count depends on whether file ends with newline
	if count < 4 || count > 5 {
		t.Errorf("expected 4-5 lines, got %d", count)
	}
}

// Helper types and functions

type fileModification struct {
	addLines    int
	removeLines int
}

func initGitRepo(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Set git config for tests
	configCmds := [][]string{
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
	}
	for _, args := range configCmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func createFileWithLines(t *testing.T, dir, filename string, lineCount int) {
	t.Helper()

	path := filepath.Join(dir, filename)
	lines := make([]string, lineCount)
	for i := 0; i < lineCount; i++ {
		lines[i] = "// Line " + string(rune('0'+i%10))
	}
	content := strings.Join(lines, "\n") + "\n"

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file %s: %v", filename, err)
	}
}

func modifyFile(t *testing.T, dir, filename string, addLines, removeLines int) {
	t.Helper()

	path := filepath.Join(dir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", filename, err)
	}

	lines := strings.Split(string(content), "\n")

	// Remove lines from the end
	if removeLines > 0 && removeLines < len(lines) {
		lines = lines[:len(lines)-removeLines]
	}

	// Add lines to the end
	for i := 0; i < addLines; i++ {
		lines = append(lines, "// Added line "+string(rune('A'+i%26)))
	}

	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		t.Fatalf("failed to modify file %s: %v", filename, err)
	}
}

func commitFiles(t *testing.T, dir, message string) {
	t.Helper()

	// Add all files
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Commit
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
}

func cleanGitRepo(t *testing.T, dir string) {
	t.Helper()

	// Reset to HEAD (discard unstaged changes)
	cmd := exec.Command("git", "reset", "--hard", "HEAD")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Logf("git reset failed (may be empty repo): %v", err)
	}

	// Clean untracked files
	cmd = exec.Command("git", "clean", "-fd")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Logf("git clean failed: %v", err)
	}
}
