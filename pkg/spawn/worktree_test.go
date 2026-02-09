package spawn

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWorktree(t *testing.T) {
	repo := setupGitRepoForWorktreeTest(t)
	workspace := "og-feat-worktree-09feb-abcd"

	worktreeDir, branch, err := CreateWorktree(repo, workspace)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	wantWorktreeDir := filepath.Join(repo, ".orch", "worktrees", workspace)
	if worktreeDir != wantWorktreeDir {
		t.Fatalf("worktree dir = %q, want %q", worktreeDir, wantWorktreeDir)
	}

	wantBranch := "agent/" + workspace
	if branch != wantBranch {
		t.Fatalf("branch = %q, want %q", branch, wantBranch)
	}

	if _, err := os.Stat(filepath.Join(worktreeDir, ".git")); err != nil {
		t.Fatalf("expected git metadata in worktree: %v", err)
	}

	headBranch := runGitInDir(t, worktreeDir, "rev-parse", "--abbrev-ref", "HEAD")
	if headBranch != wantBranch {
		t.Fatalf("worktree HEAD branch = %q, want %q", headBranch, wantBranch)
	}
}

func TestCreateWorktree_IdempotentWhenAlreadyExists(t *testing.T) {
	repo := setupGitRepoForWorktreeTest(t)
	workspace := "og-feat-worktree-09feb-ef12"

	firstDir, firstBranch, err := CreateWorktree(repo, workspace)
	if err != nil {
		t.Fatalf("first CreateWorktree failed: %v", err)
	}

	secondDir, secondBranch, err := CreateWorktree(repo, workspace)
	if err != nil {
		t.Fatalf("second CreateWorktree failed: %v", err)
	}

	if secondDir != firstDir {
		t.Fatalf("second dir = %q, want %q", secondDir, firstDir)
	}
	if secondBranch != firstBranch {
		t.Fatalf("second branch = %q, want %q", secondBranch, firstBranch)
	}
}

func TestWriteContext_OpencodeCreatesWorktreeAndSetsCWD(t *testing.T) {
	repo := setupGitRepoForWorktreeTest(t)
	workspace := "og-feat-worktree-context-09feb-1a2b"

	cfg := &Config{
		Task:          "test task",
		Project:       "orch-go",
		ProjectDir:    repo,
		WorkspaceName: workspace,
		BeadsID:       "orch-go-123",
		SkillName:     "feature-impl",
		Tier:          TierLight,
		SpawnMode:     "opencode",
	}

	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}

	wantWorktreeDir := filepath.Join(repo, ".orch", "worktrees", workspace)
	if cfg.CWD != wantWorktreeDir {
		t.Fatalf("cfg.CWD = %q, want %q", cfg.CWD, wantWorktreeDir)
	}

	manifest, err := ReadAgentManifest(cfg.WorkspacePath())
	if err != nil {
		t.Fatalf("ReadAgentManifest failed: %v", err)
	}

	if manifest.SourceProjectDir != repo {
		t.Fatalf("manifest.SourceProjectDir = %q, want %q", manifest.SourceProjectDir, repo)
	}
	if manifest.ProjectDir != repo {
		t.Fatalf("manifest.ProjectDir = %q, want %q", manifest.ProjectDir, repo)
	}
	if manifest.GitWorktreeDir != wantWorktreeDir {
		t.Fatalf("manifest.GitWorktreeDir = %q, want %q", manifest.GitWorktreeDir, wantWorktreeDir)
	}

	wantBranch := "agent/" + workspace
	if manifest.GitBranch != wantBranch {
		t.Fatalf("manifest.GitBranch = %q, want %q", manifest.GitBranch, wantBranch)
	}

	headBranch := runGitInDir(t, wantWorktreeDir, "rev-parse", "--abbrev-ref", "HEAD")
	if headBranch != wantBranch {
		t.Fatalf("worktree HEAD branch = %q, want %q", headBranch, wantBranch)
	}
}

func setupGitRepoForWorktreeTest(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()
	runGitInDir(t, repo, "init")
	runGitInDir(t, repo, "config", "user.name", "Orch Test")
	runGitInDir(t, repo, "config", "user.email", "orch-test@example.com")

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}
	runGitInDir(t, repo, "add", "README.md")
	runGitInDir(t, repo, "commit", "-m", "initial commit")

	return repo
}

func runGitInDir(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output))
}
