//go:build integration
// +build integration

package spawn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func TestCreateWorktree_ConcurrentIsolationAndCleanup(t *testing.T) {
	repo := setupGitRepoForWorktreeTest(t)

	workspaces := []string{
		"og-feat-integration-test-concurrent-09feb-a1b2",
		"og-feat-integration-test-concurrent-09feb-c3d4",
		"og-feat-integration-test-concurrent-09feb-e5f6",
		"og-feat-integration-test-concurrent-09feb-g7h8",
		"og-feat-integration-test-concurrent-09feb-i9j0",
	}

	for _, workspace := range workspaces {
		workspace := workspace
		t.Cleanup(func() {
			_ = RemoveWorktree(repo, workspace)
		})
	}

	type worktreeResult struct {
		workspace string
		dir       string
		branch    string
		err       error
	}

	results := make(chan worktreeResult, len(workspaces))
	var wg sync.WaitGroup
	for _, workspace := range workspaces {
		workspace := workspace
		wg.Add(1)
		go func() {
			defer wg.Done()
			dir, branch, err := CreateWorktree(repo, workspace)
			results <- worktreeResult{workspace: workspace, dir: dir, branch: branch, err: err}
		}()
	}

	wg.Wait()
	close(results)

	byWorkspace := make(map[string]worktreeResult, len(workspaces))
	for result := range results {
		if result.err != nil {
			t.Fatalf("CreateWorktree(%q) failed: %v", result.workspace, result.err)
		}
		if _, exists := byWorkspace[result.workspace]; exists {
			t.Fatalf("duplicate result for workspace %q", result.workspace)
		}
		byWorkspace[result.workspace] = result
	}

	for _, workspace := range workspaces {
		result, ok := byWorkspace[workspace]
		if !ok {
			t.Fatalf("missing result for workspace %q", workspace)
		}

		wantDir := filepath.Join(repo, ".orch", "worktrees", workspace)
		if result.dir != wantDir {
			t.Fatalf("workspace %q dir = %q, want %q", workspace, result.dir, wantDir)
		}

		wantBranch := "agent/" + workspace
		if result.branch != wantBranch {
			t.Fatalf("workspace %q branch = %q, want %q", workspace, result.branch, wantBranch)
		}

		headBranch := runGitInDir(t, result.dir, "rev-parse", "--abbrev-ref", "HEAD")
		if headBranch != wantBranch {
			t.Fatalf("workspace %q HEAD branch = %q, want %q", workspace, headBranch, wantBranch)
		}
	}

	for _, workspace := range workspaces {
		result := byWorkspace[workspace]
		markerFile := filepath.Join(result.dir, markerFilename(workspace))
		contents := []byte("workspace=" + workspace + "\n")
		if err := os.WriteFile(markerFile, contents, 0644); err != nil {
			t.Fatalf("write marker file for %q failed: %v", workspace, err)
		}
	}

	for _, owner := range workspaces {
		filename := markerFilename(owner)
		for _, candidate := range workspaces {
			path := filepath.Join(byWorkspace[candidate].dir, filename)
			_, err := os.Stat(path)
			if owner == candidate {
				if err != nil {
					t.Fatalf("expected marker file in owner worktree %q: %v", owner, err)
				}
				continue
			}
			if err == nil {
				t.Fatalf("marker %q from %q unexpectedly present in %q", filename, owner, candidate)
			}
			if !os.IsNotExist(err) {
				t.Fatalf("stat marker %q in %q failed: %v", filename, candidate, err)
			}
		}
	}

	for _, workspace := range workspaces {
		if err := RemoveWorktree(repo, workspace); err != nil {
			t.Fatalf("RemoveWorktree(%q) failed: %v", workspace, err)
		}
	}

	for _, workspace := range workspaces {
		worktreePath := filepath.Join(repo, ".orch", "worktrees", workspace)
		if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
			t.Fatalf("worktree path still exists for %q: %s", workspace, worktreePath)
		}

		branch := "agent/" + workspace
		if gitBranchExistsForIntegrationTest(repo, branch) {
			t.Fatalf("branch %q still exists after cleanup", branch)
		}
	}
}

func TestWriteContext_OpencodeRuntimeDirFlow(t *testing.T) {
	repo := setupGitRepoForWorktreeTest(t)
	workspace := "og-feat-integration-runtime-dir-09feb-z1y2"

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

	if got := cfg.RuntimeDir(); got != repo {
		t.Fatalf("cfg.RuntimeDir() before WriteContext = %q, want %q", got, repo)
	}

	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}
	t.Cleanup(func() {
		_ = RemoveWorktree(repo, workspace)
	})

	wantWorktreeDir := filepath.Join(repo, ".orch", "worktrees", workspace)
	if cfg.CWD != wantWorktreeDir {
		t.Fatalf("cfg.CWD = %q, want %q", cfg.CWD, wantWorktreeDir)
	}

	if got := cfg.RuntimeDir(); got != wantWorktreeDir {
		t.Fatalf("cfg.RuntimeDir() after WriteContext = %q, want %q", got, wantWorktreeDir)
	}
}

func markerFilename(workspace string) string {
	return fmt.Sprintf("marker-%s.txt", workspace)
}

func gitBranchExistsForIntegrationTest(repo, branch string) bool {
	cmd := exec.Command("git", "-C", repo, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}
