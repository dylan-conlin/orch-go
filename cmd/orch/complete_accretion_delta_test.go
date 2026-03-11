package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectAccretionFromBaseline(t *testing.T) {
	// Create a temp git repo with a known baseline
	tmpDir, err := os.MkdirTemp("", "accretion-baseline-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	initTestGitRepo(t, tmpDir)

	// Create initial file and commit (this is the "baseline")
	writeTestFile(t, tmpDir, "pkg/main.go", 100)
	gitAddCommit(t, tmpDir, "initial commit")

	// Record the baseline SHA
	baseline := gitHead(t, tmpDir)

	// Make changes after baseline (simulating agent work)
	writeTestFile(t, tmpDir, "pkg/main.go", 150) // grew by ~50 lines
	writeTestFile(t, tmpDir, "pkg/new.go", 30)    // new file
	gitAddCommit(t, tmpDir, "agent work")

	result := collectAccretionFromBaseline(tmpDir, baseline)
	if result == nil {
		t.Fatal("expected non-nil result from baseline approach")
	}

	if result.TotalFiles == 0 {
		t.Error("expected at least one file delta")
	}

	// Should capture pkg/main.go and pkg/new.go changes
	foundMain := false
	foundNew := false
	for _, d := range result.FileDeltas {
		if d.Path == "pkg/main.go" {
			foundMain = true
		}
		if d.Path == "pkg/new.go" {
			foundNew = true
		}
	}
	if !foundMain {
		t.Error("expected pkg/main.go in file deltas")
	}
	if !foundNew {
		t.Error("expected pkg/new.go in file deltas")
	}

	if result.NetDelta <= 0 {
		t.Errorf("expected positive net delta, got %d", result.NetDelta)
	}
}

func TestCollectAccretionFromBaseline_NoChanges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "accretion-nochange-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	initTestGitRepo(t, tmpDir)

	writeTestFile(t, tmpDir, "main.go", 10)
	gitAddCommit(t, tmpDir, "initial")

	baseline := gitHead(t, tmpDir)

	// No changes since baseline
	result := collectAccretionFromBaseline(tmpDir, baseline)
	if result != nil {
		t.Errorf("expected nil result when no changes, got %d files", result.TotalFiles)
	}
}

func TestCollectAccretionFromBaseline_DetectsRiskFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "accretion-risk-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	initTestGitRepo(t, tmpDir)

	// Create a large file (>800 lines)
	writeTestFile(t, tmpDir, "big.go", 900)
	gitAddCommit(t, tmpDir, "initial")

	baseline := gitHead(t, tmpDir)

	// Add more lines to push it further
	writeTestFile(t, tmpDir, "big.go", 1000)
	gitAddCommit(t, tmpDir, "agent adds lines")

	result := collectAccretionFromBaseline(tmpDir, baseline)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.RiskFiles == 0 {
		t.Error("expected at least one risk file (>800 lines with growth)")
	}
}

func TestCollectAccretionDelta_UsesBaselineWhenAvailable(t *testing.T) {
	// Integration test: collectAccretionDelta should use baseline path
	// when manifest has GitBaseline set
	tmpDir, err := os.MkdirTemp("", "accretion-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	initTestGitRepo(t, tmpDir)

	writeTestFile(t, tmpDir, "cmd/app.go", 50)
	gitAddCommit(t, tmpDir, "initial")

	baseline := gitHead(t, tmpDir)

	// Create a workspace directory with manifest
	wsDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	// Write AGENT_MANIFEST.json with git baseline
	manifestJSON := `{
		"skill": "feature-impl",
		"beads_id": "test-123",
		"project_dir": "` + tmpDir + `",
		"git_baseline": "` + baseline + `",
		"spawn_time": "2026-01-01T00:00:00Z"
	}`
	if err := os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"), []byte(manifestJSON), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Agent makes code changes OUTSIDE the workspace directory
	writeTestFile(t, tmpDir, "cmd/app.go", 80)
	writeTestFile(t, tmpDir, "pkg/util.go", 40)
	gitAddCommit(t, tmpDir, "agent code changes")

	result := collectAccretionDelta(tmpDir, wsDir)
	if result == nil {
		t.Fatal("expected non-nil result — this was the bug: workspace path filter missed code commits")
	}

	if result.TotalFiles < 2 {
		t.Errorf("expected at least 2 files (cmd/app.go, pkg/util.go), got %d", result.TotalFiles)
	}

	// Verify it captured code files outside the workspace
	foundApp := false
	foundUtil := false
	for _, d := range result.FileDeltas {
		if d.Path == "cmd/app.go" {
			foundApp = true
		}
		if d.Path == "pkg/util.go" {
			foundUtil = true
		}
	}
	if !foundApp {
		t.Error("expected cmd/app.go in deltas — baseline approach should capture code outside workspace")
	}
	if !foundUtil {
		t.Error("expected pkg/util.go in deltas — baseline approach should capture code outside workspace")
	}
}

func TestParseNumstatLines(t *testing.T) {
	input := "10\t5\tpkg/main.go\n20\t0\tpkg/new.go\n-\t-\tbinary.dat\n"
	fileDeltas := make(map[string]*struct {
		Path         string
		LinesAdded   int
		LinesRemoved int
		NetDelta     int
	})

	// Test the actual parseNumstatLines function via parseNumstatOutput
	result := parseNumstatOutput("/tmp", input)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.TotalFiles != 3 {
		t.Errorf("expected 3 files, got %d", result.TotalFiles)
	}

	_ = fileDeltas // suppress unused
}

// Test helpers

func initTestGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git init failed: %v", err)
		}
	}
}

func writeTestFile(t *testing.T, dir, relPath string, lineCount int) {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	lines := make([]string, lineCount)
	for i := range lines {
		lines[i] = "// line " + strings.Repeat("x", i%50)
	}
	if err := os.WriteFile(fullPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func gitAddCommit(t *testing.T, dir, msg string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", msg},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", args[1], err, out)
		}
	}
}

func gitHead(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}
	return strings.TrimSpace(string(out))
}
