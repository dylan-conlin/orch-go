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

// getHeadCommit returns the HEAD commit hash.
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
