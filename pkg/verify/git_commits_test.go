package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestIsCodeProducingSkill(t *testing.T) {
	tests := []struct {
		skillName string
		expected  bool
	}{
		// Code-producing skills
		{"feature-impl", true},
		{"systematic-debugging", true},
		{"reliability-testing", true},
		{"Feature-Impl", true}, // case insensitive
		{"SYSTEMATIC-DEBUGGING", true},

		// Artifact-producing skills (exempt)
		{"investigation", false},
		{"architect", false},
		{"research", false},
		{"design-session", false},
		{"codebase-audit", false},
		{"issue-creation", false},
		{"writing-skills", false},

		// Unknown skills (permissive default)
		{"unknown-skill", false},
		{"custom-skill", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.skillName, func(t *testing.T) {
			got := IsCodeProducingSkill(tt.skillName)
			if got != tt.expected {
				t.Errorf("IsCodeProducingSkill(%q) = %v, want %v", tt.skillName, got, tt.expected)
			}
		})
	}
}

func TestCountCommitsSinceTime(t *testing.T) {
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

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git email: %v", err)
	}
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git name: %v", err)
	}

	// Record time before commits
	beforeCommits := time.Now()

	// Wait a moment to ensure time difference
	time.Sleep(100 * time.Millisecond)

	// Create a file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "test commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// Test: should find 1 commit since beforeCommits
	count, err := CountCommitsSinceTime(tmpDir, beforeCommits)
	if err != nil {
		t.Fatalf("CountCommitsSinceTime() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CountCommitsSinceTime() = %d, want 1", count)
	}

	// Test: should find 0 commits since a future time
	// Using a future time is more reliable than waiting for git's second granularity
	futureTime := time.Now().Add(1 * time.Hour)
	count, err = CountCommitsSinceTime(tmpDir, futureTime)
	if err != nil {
		t.Fatalf("CountCommitsSinceTime() error = %v", err)
	}
	if count != 0 {
		t.Errorf("CountCommitsSinceTime() = %d, want 0", count)
	}

	// Test: zero time should return error
	_, err = CountCommitsSinceTime(tmpDir, time.Time{})
	if err == nil {
		t.Error("CountCommitsSinceTime() should error on zero time")
	}
}

func TestVerifyGitCommits_CodeProducingSkill(t *testing.T) {
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

	// Create an initial commit (required for git log to work)
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

	// Wait to ensure spawn time is clearly after the initial commit
	// Git's --since has second granularity, so we need to wait at least 1 second
	time.Sleep(1100 * time.Millisecond)

	// Create workspace with code-producing skill
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Write SPAWN_CONTEXT.md with code-producing skill
	spawnContext := `## SKILL GUIDANCE (feature-impl)

This is a test spawn context for feature implementation.
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
	}

	// Write spawn time (now - after the initial commit)
	// Use time.Now() which is definitely after the initial commit since we waited 1.1s
	spawnTime := time.Now()
	if err := spawn.WriteSpawnTime(workspaceDir, spawnTime); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}

	// Test: should fail - no commits since spawn time
	result := VerifyGitCommits(workspaceDir, tmpDir)
	t.Logf("Result: Passed=%v, IsCodeSkill=%v, SkillName=%q, CommitCount=%d, SpawnTime=%v",
		result.Passed, result.IsCodeSkill, result.SkillName, result.CommitCount, result.SpawnTime)
	t.Logf("Errors: %v", result.Errors)
	t.Logf("Warnings: %v", result.Warnings)
	if result.Passed {
		t.Error("VerifyGitCommits() should fail for code-producing skill with no commits")
	}
	if result.SkillName != "feature-impl" {
		t.Errorf("SkillName = %q, want %q", result.SkillName, "feature-impl")
	}
	if result.IsCodeSkill != true {
		t.Error("IsCodeSkill should be true for feature-impl")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for missing commits")
	}

	// Now create a commit
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	cmd = exec.Command("git", "add", "main.go")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "add main.go")
	cmd.Dir = tmpDir
	cmd.Run()

	// Test: should pass - has commits since spawn time
	result = VerifyGitCommits(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitCommits() should pass after commit, errors: %v", result.Errors)
	}
	if result.CommitCount < 1 {
		t.Errorf("CommitCount = %d, want >= 1", result.CommitCount)
	}
}

func TestVerifyGitCommits_ArtifactProducingSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace with artifact-producing skill (investigation)
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Write SPAWN_CONTEXT.md with artifact-producing skill
	spawnContext := `## SKILL GUIDANCE (investigation)

This is a test spawn context for investigation.
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
	}

	// Write spawn time
	spawnTime := time.Now().Add(-5 * time.Minute)
	if err := spawn.WriteSpawnTime(workspaceDir, spawnTime); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}

	// Test: should pass - artifact-producing skills are exempt
	result := VerifyGitCommits(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitCommits() should pass for artifact-producing skill, errors: %v", result.Errors)
	}
	if result.SkillName != "investigation" {
		t.Errorf("SkillName = %q, want %q", result.SkillName, "investigation")
	}
	if result.IsCodeSkill != false {
		t.Error("IsCodeSkill should be false for investigation")
	}
}

func TestVerifyGitCommits_MissingSpawnTime(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace without spawn time file
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Write SPAWN_CONTEXT.md with code-producing skill
	spawnContext := `## SKILL GUIDANCE (feature-impl)

This is a test spawn context.
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
	}

	// Test: should pass (with warning) - no spawn time to verify against
	result := VerifyGitCommits(workspaceDir, tmpDir)
	if !result.Passed {
		t.Errorf("VerifyGitCommits() should pass when spawn time is missing, errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warning about missing spawn time")
	}
}

func TestVerifyGitCommitsForCompletion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace", "test-agent")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Test: artifact-producing skill should return nil (no verification needed)
	spawnContext := `## SKILL GUIDANCE (investigation)
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
	}
	if err := spawn.WriteSpawnTime(workspaceDir, time.Now().Add(-5*time.Minute)); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}

	result := VerifyGitCommitsForCompletion(workspaceDir, tmpDir)
	if result != nil {
		t.Error("VerifyGitCommitsForCompletion() should return nil for artifact-producing skill")
	}

	// Test: code-producing skill without spawn time should return nil
	spawnContext = `## SKILL GUIDANCE (feature-impl)
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
	}
	// Remove spawn time file
	os.Remove(filepath.Join(workspaceDir, ".spawn_time"))

	result = VerifyGitCommitsForCompletion(workspaceDir, tmpDir)
	if result != nil {
		t.Error("VerifyGitCommitsForCompletion() should return nil when spawn time is missing")
	}
}
