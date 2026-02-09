package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func integrateAgentBranch(target *CompletionTarget) error {
	if target == nil {
		return nil
	}
	if target.IsOrchestratorSession || target.IsUntracked || target.BeadsID == "" {
		return nil
	}
	if strings.TrimSpace(target.GitBranch) == "" {
		return nil
	}

	worktree := strings.TrimSpace(target.gitDir())
	if worktree == "" {
		return fmt.Errorf("missing git_worktree_dir for %s", target.BeadsID)
	}

	source := strings.TrimSpace(target.sourceDir())
	if source == "" {
		source = worktree
	}

	base, err := readBranchName(source)
	if err != nil {
		return fmt.Errorf("failed to read base branch in %s: %w", source, err)
	}
	if base == "" {
		return fmt.Errorf("could not determine base branch in %s", source)
	}
	if base == target.GitBranch {
		return nil
	}

	if err := ensureBranchCheckedOut(worktree, target.GitBranch); err != nil {
		return fmt.Errorf("failed to checkout %s in %s: %w", target.GitBranch, worktree, err)
	}

	fmt.Printf("Rebasing %s onto %s\n", target.GitBranch, base)
	if _, err := runGitMerge(worktree, "rebase", base); err != nil {
		return fmt.Errorf("rebase failed for %s onto %s: %w", target.GitBranch, base, err)
	}

	mergeDir, err := findBranchWorktree(source, base)
	if err != nil {
		return err
	}

	fmt.Printf("Merging %s into %s (ff-only)\n", target.GitBranch, base)
	if _, err := runGitMerge(mergeDir, "merge", "--ff-only", target.GitBranch); err != nil {
		return fmt.Errorf("fast-forward merge failed for %s into %s: %w", target.GitBranch, base, err)
	}

	return nil
}

func ensureBranchCheckedOut(dir, branch string) error {
	current, err := readBranchName(dir)
	if err == nil && current == branch {
		return nil
	}

	_, err = runGitMerge(dir, "switch", branch)
	if err == nil {
		return nil
	}

	_, err = runGitMerge(dir, "checkout", branch)
	return err
}

func runGitMerge(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, text)
	}
	return text, nil
}

func readBranchName(dir string) (string, error) {
	out, err := runGitMerge(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if out == "HEAD" {
		return "", nil
	}
	return out, nil
}

func findBranchWorktree(repoDir, branch string) (string, error) {
	out, err := runGitMerge(repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("failed to list git worktrees: %w", err)
	}

	w := ""
	b := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			if b == branch && w != "" {
				return w, nil
			}
			w = strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
			b = ""
			continue
		}
		if strings.HasPrefix(line, "branch refs/heads/") {
			b = strings.TrimSpace(strings.TrimPrefix(line, "branch refs/heads/"))
		}
	}

	if b == branch && w != "" {
		return w, nil
	}

	current, currentErr := readBranchName(repoDir)
	if currentErr == nil && current == branch {
		return repoDir, nil
	}

	return "", fmt.Errorf("base branch %s is not checked out in any worktree", branch)
}
