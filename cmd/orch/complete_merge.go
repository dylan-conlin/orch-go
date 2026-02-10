package main

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

const beadsIssuesPath = ".beads/issues.jsonl"

// integrateAgentBranch cherry-picks agent commits onto the base branch.
//
// Previous approach used rebase + ff-only merge, but git rebase checks ALL
// worktrees for dirty state. If any worktree (e.g. master) has uncommitted
// files (common: .beads/issues.jsonl), rebase fails even though we're
// operating on the agent worktree. Cherry-pick only checks the target
// worktree, avoiding this cross-worktree dirty state problem.
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

	mergeDir, err := findBranchWorktree(source, base)
	if err != nil {
		return err
	}
	if err := prepareMergeDirForFFMerge(mergeDir); err != nil {
		return fmt.Errorf("failed to prepare merge worktree %s: %w", mergeDir, err)
	}

	// Find the merge-base between base and agent branch
	mergeBase, err := runGitMerge(worktree, "merge-base", base, target.GitBranch)
	if err != nil {
		return fmt.Errorf("failed to find merge-base between %s and %s: %w", base, target.GitBranch, err)
	}

	// Get list of commits unique to agent branch (oldest first for cherry-pick order)
	commitList, err := runGitMerge(worktree, "rev-list", "--reverse", mergeBase+".."+target.GitBranch)
	if err != nil {
		return fmt.Errorf("failed to list commits for %s: %w", target.GitBranch, err)
	}
	if strings.TrimSpace(commitList) == "" {
		// No commits to cherry-pick — branches are identical
		return nil
	}

	commits := strings.Split(strings.TrimSpace(commitList), "\n")
	fmt.Printf("Cherry-picking %d commit(s) from %s onto %s\n", len(commits), target.GitBranch, base)
	for _, commit := range commits {
		commit = strings.TrimSpace(commit)
		if commit == "" {
			continue
		}
		if _, err := runGitMerge(mergeDir, "cherry-pick", commit); err != nil {
			errMsg := err.Error()
			// Detect "empty commit" — happens when the commit's changes are
			// already present on the base branch (e.g. previously cherry-picked
			// or independently applied). Skip gracefully instead of failing.
			if isEmptyCherryPick(errMsg) {
				// Reset the cherry-pick state and skip this commit
				_, _ = runGitMerge(mergeDir, "cherry-pick", "--abort")
				subject := commitSubject(mergeDir, commit)
				fmt.Printf("  Skipping already-applied commit: %s %s\n", commit[:minInt(7, len(commit))], subject)
				continue
			}
			// Abort any in-progress cherry-pick before returning
			_, _ = runGitMerge(mergeDir, "cherry-pick", "--abort")
			return fmt.Errorf("cherry-pick failed for %s onto %s: %w", target.GitBranch, base, err)
		}
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

func prepareMergeDirForFFMerge(dir string) error {
	status, err := runGitMerge(dir, "status", "--porcelain")
	if err != nil {
		return err
	}

	dirtyPaths := parseDirtyPaths(status)
	if len(dirtyPaths) != 1 || dirtyPaths[0] != beadsIssuesPath {
		return nil
	}

	if _, err := runGitMerge(dir, "restore", "--staged", "--worktree", "--", beadsIssuesPath); err != nil {
		if _, checkoutErr := runGitMerge(dir, "checkout", "--", beadsIssuesPath); checkoutErr != nil {
			return err
		}
		_, _ = runGitMerge(dir, "reset", "--", beadsIssuesPath)
	}

	fmt.Printf("Discarded local %s changes before ff-only merge\n", beadsIssuesPath)
	return nil
}

func parseDirtyPaths(status string) []string {
	if strings.TrimSpace(status) == "" {
		return nil
	}

	seen := make(map[string]struct{})
	paths := make([]string, 0)
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		pathStart := -1
		switch {
		case len(line) >= 3 && line[2] == ' ':
			pathStart = 3
		case len(line) >= 2 && line[1] == ' ':
			pathStart = 2
		default:
			if idx := strings.IndexByte(line, ' '); idx >= 0 {
				pathStart = idx + 1
			}
		}
		if pathStart <= 0 || pathStart >= len(line) {
			continue
		}

		path := strings.TrimSpace(line[pathStart:])
		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = strings.TrimSpace(path[idx+4:])
		}
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}

	sort.Strings(paths)
	return paths
}

func runGitMerge(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	// Set ORCH_COMPLETING=1 so post-commit hooks skip auto-rebuild.
	// During cherry-pick, commits trigger the hook which runs make install,
	// dirtying the workspace and causing subsequent git operations to fail.
	cmd.Env = append(cmd.Environ(), "ORCH_COMPLETING=1")
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

// isEmptyCherryPick detects when a cherry-pick fails because the commit's changes
// are already present on the target branch, resulting in an empty commit.
func isEmptyCherryPick(errMsg string) bool {
	// Git messages for this case vary by version:
	// - "The previous cherry-pick is now empty"
	// - "nothing to commit"
	// - "empty" in the context of cherry-pick
	lower := strings.ToLower(errMsg)
	if strings.Contains(lower, "cherry-pick is now empty") {
		return true
	}
	if strings.Contains(lower, "nothing to commit") {
		return true
	}
	return false
}

// commitSubject returns the first line of a commit's message, for display purposes.
func commitSubject(dir, commit string) string {
	out, err := runGitMerge(dir, "log", "--format=%s", "-1", commit)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// minInt returns the smaller of two ints.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
