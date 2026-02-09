package spawn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const agentBranchPrefix = "agent/"

// CreateWorktree creates an isolated git worktree and branch for a workspace.
//
// Layout:
//   - Worktree: {projectDir}/.orch/worktrees/{workspaceName}
//   - Branch:   agent/{workspaceName}
//
// Returns the created worktree directory and branch name.
func CreateWorktree(projectDir, workspaceName string) (string, string, error) {
	projectDir = strings.TrimSpace(projectDir)
	workspaceName = strings.TrimSpace(workspaceName)
	if projectDir == "" {
		return "", "", fmt.Errorf("project directory is required")
	}
	if workspaceName == "" {
		return "", "", fmt.Errorf("workspace name is required")
	}

	sourceDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve project directory: %w", err)
	}

	branch := agentBranchPrefix + workspaceName
	worktreeDir := filepath.Join(sourceDir, ".orch", "worktrees", workspaceName)

	if stat, statErr := os.Stat(worktreeDir); statErr == nil {
		if !stat.IsDir() {
			return "", "", fmt.Errorf("worktree path exists but is not a directory: %s", worktreeDir)
		}
		if _, gitErr := os.Stat(filepath.Join(worktreeDir, ".git")); gitErr == nil {
			return worktreeDir, branch, nil
		}
		return "", "", fmt.Errorf("worktree directory exists but is not a git worktree: %s", worktreeDir)
	} else if !os.IsNotExist(statErr) {
		return "", "", fmt.Errorf("failed to stat worktree path: %w", statErr)
	}

	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0755); err != nil {
		return "", "", fmt.Errorf("failed to create worktree root: %w", err)
	}

	cmd := exec.Command("git", "-C", sourceDir, "worktree", "add", "-b", branch, worktreeDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("git worktree add failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return worktreeDir, branch, nil
}
