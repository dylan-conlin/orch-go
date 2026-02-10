package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrateAgentBranchFastForwardMerge(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	worktree := filepath.Join(t.TempDir(), "worktree")
	setupMergeRepo(t, repo)

	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0644); err != nil {
		t.Fatalf("write base file: %v", err)
	}
	if _, err := runGitMerge(repo, "add", "."); err != nil {
		t.Fatalf("git add base: %v", err)
	}
	if _, err := runGitMerge(repo, "commit", "-m", "base"); err != nil {
		t.Fatalf("git commit base: %v", err)
	}

	base, err := readBranchName(repo)
	if err != nil {
		t.Fatalf("read base branch: %v", err)
	}

	branch := "agent/test"
	if _, err := runGitMerge(repo, "branch", branch); err != nil {
		t.Fatalf("git branch: %v", err)
	}
	if _, err := runGitMerge(repo, "worktree", "add", worktree, branch); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	if err := os.WriteFile(filepath.Join(worktree, "agent.txt"), []byte("agent\n"), 0644); err != nil {
		t.Fatalf("write agent file: %v", err)
	}
	if _, err := runGitMerge(worktree, "add", "."); err != nil {
		t.Fatalf("git add agent: %v", err)
	}
	if _, err := runGitMerge(worktree, "commit", "-m", "agent change"); err != nil {
		t.Fatalf("git commit agent: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base 2\n"), 0644); err != nil {
		t.Fatalf("update base file: %v", err)
	}
	if _, err := runGitMerge(repo, "add", "."); err != nil {
		t.Fatalf("git add base update: %v", err)
	}
	if _, err := runGitMerge(repo, "commit", "-m", "base update"); err != nil {
		t.Fatalf("git commit base update: %v", err)
	}

	target := &CompletionTarget{
		BeadsID:          "orch-go-test",
		BeadsProjectDir:  repo,
		SourceProjectDir: repo,
		GitWorktreeDir:   worktree,
		GitBranch:        branch,
	}

	if err := integrateAgentBranch(target); err != nil {
		t.Fatalf("integrateAgentBranch() failed: %v", err)
	}

	head, err := runGitMerge(repo, "rev-parse", base)
	if err != nil {
		t.Fatalf("rev-parse base head: %v", err)
	}
	branchHead, err := runGitMerge(repo, "rev-parse", branch)
	if err != nil {
		t.Fatalf("rev-parse branch head: %v", err)
	}
	if head != branchHead {
		t.Fatalf("base head %s != branch head %s after merge", head, branchHead)
	}
}

func TestIntegrateAgentBranchRebaseConflict(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	worktree := filepath.Join(t.TempDir(), "worktree")
	setupMergeRepo(t, repo)

	if err := os.WriteFile(filepath.Join(repo, "shared.txt"), []byte("line\n"), 0644); err != nil {
		t.Fatalf("write shared file: %v", err)
	}
	if _, err := runGitMerge(repo, "add", "."); err != nil {
		t.Fatalf("git add shared: %v", err)
	}
	if _, err := runGitMerge(repo, "commit", "-m", "base"); err != nil {
		t.Fatalf("git commit base: %v", err)
	}

	branch := "agent/conflict"
	if _, err := runGitMerge(repo, "branch", branch); err != nil {
		t.Fatalf("git branch: %v", err)
	}
	if _, err := runGitMerge(repo, "worktree", "add", worktree, branch); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	if err := os.WriteFile(filepath.Join(worktree, "shared.txt"), []byte("branch\n"), 0644); err != nil {
		t.Fatalf("write branch change: %v", err)
	}
	if _, err := runGitMerge(worktree, "add", "."); err != nil {
		t.Fatalf("git add branch change: %v", err)
	}
	if _, err := runGitMerge(worktree, "commit", "-m", "branch change"); err != nil {
		t.Fatalf("git commit branch change: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "shared.txt"), []byte("base\n"), 0644); err != nil {
		t.Fatalf("write base change: %v", err)
	}
	if _, err := runGitMerge(repo, "add", "."); err != nil {
		t.Fatalf("git add base change: %v", err)
	}
	if _, err := runGitMerge(repo, "commit", "-m", "base change"); err != nil {
		t.Fatalf("git commit base change: %v", err)
	}

	target := &CompletionTarget{
		BeadsID:          "orch-go-test",
		BeadsProjectDir:  repo,
		SourceProjectDir: repo,
		GitWorktreeDir:   worktree,
		GitBranch:        branch,
	}

	err := integrateAgentBranch(target)
	if err == nil {
		t.Fatal("integrateAgentBranch() expected conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "rebase failed") {
		t.Fatalf("expected rebase failure error, got: %v", err)
	}
}

func TestIntegrateAgentBranchHandlesDirtyBeadsIssuesInMergeDir(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	worktree := filepath.Join(t.TempDir(), "worktree")
	setupMergeRepo(t, repo)

	if err := os.MkdirAll(filepath.Join(repo, ".beads"), 0755); err != nil {
		t.Fatalf("mkdir .beads: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".beads", "issues.jsonl"), []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write seed issues file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0644); err != nil {
		t.Fatalf("write base file: %v", err)
	}
	if _, err := runGitMerge(repo, "add", "."); err != nil {
		t.Fatalf("git add base: %v", err)
	}
	if _, err := runGitMerge(repo, "commit", "-m", "base"); err != nil {
		t.Fatalf("git commit base: %v", err)
	}

	base, err := readBranchName(repo)
	if err != nil {
		t.Fatalf("read base branch: %v", err)
	}

	branch := "agent/dirty-beads"
	if _, err := runGitMerge(repo, "branch", branch); err != nil {
		t.Fatalf("git branch: %v", err)
	}
	if _, err := runGitMerge(repo, "worktree", "add", worktree, branch); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	if err := os.WriteFile(filepath.Join(worktree, ".beads", "issues.jsonl"), []byte("branch\n"), 0644); err != nil {
		t.Fatalf("write branch issues file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "agent.txt"), []byte("agent\n"), 0644); err != nil {
		t.Fatalf("write agent file: %v", err)
	}
	if _, err := runGitMerge(worktree, "add", "."); err != nil {
		t.Fatalf("git add branch: %v", err)
	}
	if _, err := runGitMerge(worktree, "commit", "-m", "branch change"); err != nil {
		t.Fatalf("git commit branch: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base 2\n"), 0644); err != nil {
		t.Fatalf("write base update: %v", err)
	}
	if _, err := runGitMerge(repo, "add", "base.txt"); err != nil {
		t.Fatalf("git add base update: %v", err)
	}
	if _, err := runGitMerge(repo, "commit", "-m", "base update"); err != nil {
		t.Fatalf("git commit base update: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, ".beads", "issues.jsonl"), []byte("dirty\n"), 0644); err != nil {
		t.Fatalf("write dirty issues file: %v", err)
	}

	target := &CompletionTarget{
		BeadsID:          "orch-go-test",
		BeadsProjectDir:  repo,
		SourceProjectDir: repo,
		GitWorktreeDir:   worktree,
		GitBranch:        branch,
	}

	if err := integrateAgentBranch(target); err != nil {
		t.Fatalf("integrateAgentBranch() failed: %v", err)
	}

	head, err := runGitMerge(repo, "rev-parse", base)
	if err != nil {
		t.Fatalf("rev-parse base head: %v", err)
	}
	branchHead, err := runGitMerge(repo, "rev-parse", branch)
	if err != nil {
		t.Fatalf("rev-parse branch head: %v", err)
	}
	if head != branchHead {
		t.Fatalf("base head %s != branch head %s after merge", head, branchHead)
	}

	status, err := runGitMerge(repo, "status", "--porcelain", "--", ".beads/issues.jsonl")
	if err != nil {
		t.Fatalf("git status issues file: %v", err)
	}
	if strings.TrimSpace(status) != "" {
		t.Fatalf("expected .beads/issues.jsonl to be clean, got: %q", status)
	}
}

func TestIntegrateAgentBranchDoesNotDiscardBeadsWhenOtherFilesDirty(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	worktree := filepath.Join(t.TempDir(), "worktree")
	setupMergeRepo(t, repo)

	if err := os.MkdirAll(filepath.Join(repo, ".beads"), 0755); err != nil {
		t.Fatalf("mkdir .beads: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".beads", "issues.jsonl"), []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write seed issues file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0644); err != nil {
		t.Fatalf("write base file: %v", err)
	}
	if _, err := runGitMerge(repo, "add", "."); err != nil {
		t.Fatalf("git add base: %v", err)
	}
	if _, err := runGitMerge(repo, "commit", "-m", "base"); err != nil {
		t.Fatalf("git commit base: %v", err)
	}

	branch := "agent/dirty-beads-with-other"
	if _, err := runGitMerge(repo, "branch", branch); err != nil {
		t.Fatalf("git branch: %v", err)
	}
	if _, err := runGitMerge(repo, "worktree", "add", worktree, branch); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	if err := os.WriteFile(filepath.Join(worktree, ".beads", "issues.jsonl"), []byte("branch\n"), 0644); err != nil {
		t.Fatalf("write branch issues file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "agent.txt"), []byte("agent\n"), 0644); err != nil {
		t.Fatalf("write agent file: %v", err)
	}
	if _, err := runGitMerge(worktree, "add", "."); err != nil {
		t.Fatalf("git add branch: %v", err)
	}
	if _, err := runGitMerge(worktree, "commit", "-m", "branch change"); err != nil {
		t.Fatalf("git commit branch: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base 2\n"), 0644); err != nil {
		t.Fatalf("write base update: %v", err)
	}
	if _, err := runGitMerge(repo, "add", "base.txt"); err != nil {
		t.Fatalf("git add base update: %v", err)
	}
	if _, err := runGitMerge(repo, "commit", "-m", "base update"); err != nil {
		t.Fatalf("git commit base update: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, ".beads", "issues.jsonl"), []byte("dirty\n"), 0644); err != nil {
		t.Fatalf("write dirty issues file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("local dirty base\n"), 0644); err != nil {
		t.Fatalf("write local dirty base: %v", err)
	}

	target := &CompletionTarget{
		BeadsID:          "orch-go-test",
		BeadsProjectDir:  repo,
		SourceProjectDir: repo,
		GitWorktreeDir:   worktree,
		GitBranch:        branch,
	}

	err := integrateAgentBranch(target)
	if err == nil {
		t.Fatal("integrateAgentBranch() expected merge failure, got nil")
	}
	if !strings.Contains(err.Error(), "would be overwritten by merge") {
		t.Fatalf("expected merge-overwrite error, got: %v", err)
	}

	contents, err := os.ReadFile(filepath.Join(repo, ".beads", "issues.jsonl"))
	if err != nil {
		t.Fatalf("read dirty issues file: %v", err)
	}
	if string(contents) != "dirty\n" {
		t.Fatalf("expected dirty issues file to be preserved, got %q", string(contents))
	}

}

func setupMergeRepo(t *testing.T, repo string) {
	t.Helper()

	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if _, err := runGitMerge(repo, "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runGitMerge(repo, "config", "user.name", "Orch Test"); err != nil {
		t.Fatalf("git config user.name: %v", err)
	}
	if _, err := runGitMerge(repo, "config", "user.email", "orch-test@example.com"); err != nil {
		t.Fatalf("git config user.email: %v", err)
	}
}
