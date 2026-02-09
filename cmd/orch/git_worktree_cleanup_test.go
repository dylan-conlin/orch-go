package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	statedb "github.com/dylan-conlin/orch-go/pkg/state"
)

func TestCleanupManagedGitIsolationSkipsSharedTree(t *testing.T) {
	repo, branch := initGitRepoForWorktree(t)
	workspace := filepath.Join(repo, ".orch", "workspace", "ws-shared")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	if err := writeManifestForWorktreeTest(workspace, repo, repo, branch, "orch-go-test-1"); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	result := cleanupManagedGitIsolation(workspace, repo)
	if result.WorktreeRemoved {
		t.Fatalf("expected worktree cleanup to skip shared tree")
	}
	if result.BranchDeleted {
		t.Fatalf("expected branch cleanup to skip non-agent branch")
	}
	if !gitBranchExistsForWorktreeTest(repo, branch) {
		t.Fatalf("expected branch %s to remain", branch)
	}
}

func TestCleanupManagedGitIsolationRemovesWorktreeAndBranch(t *testing.T) {
	repo, _ := initGitRepoForWorktree(t)
	branch := "agent/test-cleanup"
	worktree := filepath.Join(repo, ".orch", "worktrees", "ws-cleanup")
	if err := os.MkdirAll(filepath.Dir(worktree), 0755); err != nil {
		t.Fatalf("mkdir worktree root: %v", err)
	}

	runGitForWorktreeTest(t, repo, "branch", branch)
	runGitForWorktreeTest(t, repo, "worktree", "add", worktree, branch)

	workspace := filepath.Join(repo, ".orch", "workspace", "ws-cleanup")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	if err := writeManifestForWorktreeTest(workspace, repo, worktree, branch, "orch-go-test-2"); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	result := cleanupManagedGitIsolation(workspace, repo)
	if !result.WorktreeRemoved {
		t.Fatalf("expected worktree removal")
	}
	if !result.BranchDeleted {
		t.Fatalf("expected branch deletion")
	}
	if _, err := os.Stat(worktree); !os.IsNotExist(err) {
		t.Fatalf("expected worktree %s to be removed", worktree)
	}
	if gitBranchExistsForWorktreeTest(repo, branch) {
		t.Fatalf("expected branch %s to be deleted", branch)
	}
}

func TestCleanupManagedGitIsolationIdempotent(t *testing.T) {
	repo, _ := initGitRepoForWorktree(t)
	branch := "agent/test-idempotent"
	worktree := filepath.Join(repo, ".orch", "worktrees", "ws-idempotent")
	if err := os.MkdirAll(filepath.Dir(worktree), 0755); err != nil {
		t.Fatalf("mkdir worktree root: %v", err)
	}

	runGitForWorktreeTest(t, repo, "branch", branch)
	runGitForWorktreeTest(t, repo, "worktree", "add", worktree, branch)
	runGitForWorktreeTest(t, repo, "worktree", "remove", "--force", worktree)
	runGitForWorktreeTest(t, repo, "branch", "-D", branch)

	workspace := filepath.Join(repo, ".orch", "workspace", "ws-idempotent")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	if err := writeManifestForWorktreeTest(workspace, repo, worktree, branch, "orch-go-test-3"); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	result := cleanupManagedGitIsolation(workspace, repo)
	if !result.WorktreeRemoved {
		t.Fatalf("expected idempotent worktree removal to be true")
	}
	if !result.BranchDeleted {
		t.Fatalf("expected idempotent branch deletion to be true")
	}
}

func TestCleanupStaleManagedWorktreesPrunesOrphansByStateDB(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	repo, _ := initGitRepoForWorktree(t)
	worktreeRoot := filepath.Join(repo, ".orch", "worktrees")
	if err := os.MkdirAll(worktreeRoot, 0755); err != nil {
		t.Fatalf("mkdir worktree root: %v", err)
	}

	linkedBranch := "agent/linked"
	linkedWorktree := filepath.Join(worktreeRoot, "linked")
	runGitForWorktreeTest(t, repo, "branch", linkedBranch)
	runGitForWorktreeTest(t, repo, "worktree", "add", linkedWorktree, linkedBranch)

	orphanBranch := "agent/orphan"
	orphanWorktree := filepath.Join(worktreeRoot, "orphan")
	runGitForWorktreeTest(t, repo, "branch", orphanBranch)
	runGitForWorktreeTest(t, repo, "worktree", "add", orphanWorktree, orphanBranch)

	db, err := statedb.OpenDefault()
	if err != nil {
		t.Fatalf("open state db: %v", err)
	}
	defer db.Close()

	if err := db.InsertAgent(&statedb.Agent{
		WorkspaceName: "linked",
		BeadsID:       "orch-go-linked",
		Mode:          "opencode",
		ProjectDir:    repo,
		ProjectName:   filepath.Base(repo),
		SpawnTime:     time.Now().UnixMilli(),
	}); err != nil {
		t.Fatalf("insert active agent: %v", err)
	}

	result, err := cleanupStaleManagedWorktrees(repo, 0, false)
	if err != nil {
		t.Fatalf("cleanupStaleManagedWorktrees returned error: %v", err)
	}
	if result.SkippedActive != 1 {
		t.Fatalf("expected 1 active worktree skipped, got %d", result.SkippedActive)
	}
	if result.WorktreesPruned != 1 {
		t.Fatalf("expected 1 pruned worktree, got %d", result.WorktreesPruned)
	}
	if result.BranchesDeleted != 1 {
		t.Fatalf("expected 1 deleted branch, got %d", result.BranchesDeleted)
	}
	if _, err := os.Stat(linkedWorktree); err != nil {
		t.Fatalf("expected linked worktree to remain: %v", err)
	}
	if _, err := os.Stat(orphanWorktree); !os.IsNotExist(err) {
		t.Fatalf("expected orphan worktree to be removed")
	}
	if !gitBranchExistsForWorktreeTest(repo, linkedBranch) {
		t.Fatalf("expected linked branch to remain")
	}
	if gitBranchExistsForWorktreeTest(repo, orphanBranch) {
		t.Fatalf("expected orphan branch to be deleted")
	}
}

func initGitRepoForWorktree(t *testing.T) (string, string) {
	t.Helper()

	repo := t.TempDir()
	runGitForWorktreeTest(t, repo, "init")
	runGitForWorktreeTest(t, repo, "config", "user.email", "test@example.com")
	runGitForWorktreeTest(t, repo, "config", "user.name", "Test User")

	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("init\n"), 0644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitForWorktreeTest(t, repo, "add", "README.md")
	runGitForWorktreeTest(t, repo, "commit", "-m", "init")

	branch := strings.TrimSpace(runGitForWorktreeTest(t, repo, "branch", "--show-current"))
	if branch == "" {
		branch = "master"
	}

	return repo, branch
}

func runGitForWorktreeTest(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\noutput: %s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}

func gitBranchExistsForWorktreeTest(dir, branch string) bool {
	cmd := exec.Command("git", "-C", dir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

func writeManifestForWorktreeTest(workspacePath, sourceProjectDir, gitWorktreeDir, gitBranch, beadsID string) error {
	manifest := map[string]string{
		"workspace_name":     filepath.Base(workspacePath),
		"skill":              "feature-impl",
		"beads_id":           beadsID,
		"project_dir":        sourceProjectDir,
		"source_project_dir": sourceProjectDir,
		"git_worktree_dir":   gitWorktreeDir,
		"git_branch":         gitBranch,
		"spawn_time":         time.Now().Format(time.RFC3339),
		"tier":               "light",
		"spawn_mode":         "opencode",
		"model":              "test-model",
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(workspacePath, "AGENT_MANIFEST.json")
	return os.WriteFile(manifestPath, append(data, '\n'), 0644)
}
