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

func TestReadBuildSkipMemory_NoFile(t *testing.T) {
	dir := t.TempDir()
	result := ReadBuildSkipMemory(dir)
	if result != nil {
		t.Errorf("expected nil for missing file, got %+v", result)
	}
}

func TestWriteAndReadBuildSkipMemory(t *testing.T) {
	dir := t.TempDir()

	// Create .orch directory
	orchDir := filepath.Join(dir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := WriteBuildSkipMemory(dir, "concurrent agents broke the build", "orch-go-abc1")
	if err != nil {
		t.Fatalf("WriteBuildSkipMemory failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(orchDir, BuildSkipFilename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("build skip file was not created")
	}

	// Read it back
	skip := ReadBuildSkipMemory(dir)
	if skip == nil {
		t.Fatal("ReadBuildSkipMemory returned nil")
	}

	if skip.Reason != "concurrent agents broke the build" {
		t.Errorf("Reason = %q, want %q", skip.Reason, "concurrent agents broke the build")
	}
	if skip.SkippedBy != "orch-go-abc1" {
		t.Errorf("SkippedBy = %q, want %q", skip.SkippedBy, "orch-go-abc1")
	}
	if skip.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestReadBuildSkipMemory_Expired(t *testing.T) {
	dir := t.TempDir()
	orchDir := filepath.Join(dir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write an expired entry directly
	expired := BuildSkipMemory{
		Reason:    "old failure",
		SkippedAt: time.Now().Add(-3 * time.Hour),
		SkippedBy: "old-agent",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}
	data, _ := json.MarshalIndent(expired, "", "  ")
	path := filepath.Join(orchDir, BuildSkipFilename)
	os.WriteFile(path, data, 0644)

	// Should return nil for expired entries
	result := ReadBuildSkipMemory(dir)
	if result != nil {
		t.Errorf("expected nil for expired entry, got %+v", result)
	}

	// File should be cleaned up
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expired file should have been cleaned up")
	}
}

func TestReadBuildSkipMemory_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	orchDir := filepath.Join(dir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(orchDir, BuildSkipFilename)
	os.WriteFile(path, []byte("not json"), 0644)

	result := ReadBuildSkipMemory(dir)
	if result != nil {
		t.Errorf("expected nil for invalid JSON, got %+v", result)
	}
}

func TestClearBuildSkipMemory(t *testing.T) {
	dir := t.TempDir()
	orchDir := filepath.Join(dir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a skip entry
	err := WriteBuildSkipMemory(dir, "test", "agent")
	if err != nil {
		t.Fatal(err)
	}

	// Verify it exists
	if skip := ReadBuildSkipMemory(dir); skip == nil {
		t.Fatal("expected skip to exist")
	}

	// Clear it
	ClearBuildSkipMemory(dir)

	// Should be gone
	if skip := ReadBuildSkipMemory(dir); skip != nil {
		t.Errorf("expected nil after clear, got %+v", skip)
	}
}

func TestWriteBuildSkipMemory_CreatesOrchDir(t *testing.T) {
	dir := t.TempDir()
	// Don't pre-create .orch - WriteBuildSkipMemory should create it

	err := WriteBuildSkipMemory(dir, "test reason", "test-agent")
	if err != nil {
		t.Fatalf("WriteBuildSkipMemory failed: %v", err)
	}

	// Verify .orch directory was created
	orchDir := filepath.Join(dir, ".orch")
	if _, err := os.Stat(orchDir); os.IsNotExist(err) {
		t.Error(".orch directory should have been created")
	}

	skip := ReadBuildSkipMemory(dir)
	if skip == nil {
		t.Fatal("ReadBuildSkipMemory returned nil after write")
	}
}

func TestAttributeBuildFailure_NoSpawnTime(t *testing.T) {
	// Create workspace without spawn time
	workspace := t.TempDir()
	projectDir := t.TempDir()

	result := AttributeBuildFailure(workspace, projectDir)

	// Should default to agent responsibility when no spawn time
	if !result.AgentCausedFailure {
		t.Error("expected AgentCausedFailure=true when no spawn time")
	}
	if result.PreExisting {
		t.Error("expected PreExisting=false when no spawn time")
	}
}

func TestCommitsAfterTime(t *testing.T) {
	dir := t.TempDir()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create initial commit with explicit date in the past
	os.WriteFile(filepath.Join(dir, "file1.go"), []byte("package main"), 0644)
	runGit(t, dir, "add", ".")
	cmd := exec.Command("git", "commit", "-m", "initial",
		"--date=2020-01-01T00:00:00Z")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_COMMITTER_DATE=2020-01-01T00:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}

	// Boundary between old and new commits
	boundary := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create commit after boundary
	os.WriteFile(filepath.Join(dir, "file2.go"), []byte("package main\nfunc foo(){}"), 0644)
	runGit(t, dir, "add", ".")
	cmd = exec.Command("git", "commit", "-m", "after boundary",
		"--date=2025-06-01T00:00:00Z")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_COMMITTER_DATE=2025-06-01T00:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}

	commits := commitsAfterTime(dir, boundary)
	if len(commits) != 1 {
		t.Errorf("expected 1 commit after boundary, got %d", len(commits))
	}
}

func TestParentCommit(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create two commits
	os.WriteFile(filepath.Join(dir, "file1.go"), []byte("package main"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "first")

	os.WriteFile(filepath.Join(dir, "file2.go"), []byte("package main\nfunc foo(){}"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "second")

	// Get HEAD hash
	head := getHeadCommit(t, dir)
	parent := parentCommit(dir, head)

	if parent == "" {
		t.Fatal("parentCommit returned empty string")
	}
	if parent == head {
		t.Error("parent should differ from HEAD")
	}
}

func TestBuildSkipPath(t *testing.T) {
	path := buildSkipPath("/projects/orch-go")
	expected := filepath.Join("/projects/orch-go", ".orch", BuildSkipFilename)
	if path != expected {
		t.Errorf("buildSkipPath = %q, want %q", path, expected)
	}
}

func TestAttributeBuildFailure_NoCommitsSinceSpawn(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create initial commit with an explicit date in the past
	os.WriteFile(filepath.Join(dir, "file1.go"), []byte("package main"), 0644)
	runGit(t, dir, "add", ".")
	cmd := exec.Command("git", "commit", "-m", "initial",
		"--date=2020-01-01T00:00:00Z")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_COMMITTER_DATE=2020-01-01T00:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}

	// Create workspace with spawn time well after the commit
	workspace := t.TempDir()
	spawnTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := spawn.WriteSpawnTime(workspace, spawnTime); err != nil {
		t.Fatal(err)
	}

	result := AttributeBuildFailure(workspace, dir)

	if result.AgentCausedFailure {
		t.Errorf("expected AgentCausedFailure=false when no commits since spawn, blame: %s", result.BlameDetail)
	}
	if !result.PreExisting {
		t.Errorf("expected PreExisting=true when no commits since spawn, blame: %s", result.BlameDetail)
	}
}

// runGit runs a git command in a directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

// Helper: get HEAD commit hash
func getHeadCommit(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}
	return strings.TrimSpace(string(out))
}
