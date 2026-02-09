package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	statedb "github.com/dylan-conlin/orch-go/pkg/state"
)

const gitCleanupRetries = 3

const gitCleanupRetryDelay = 250 * time.Millisecond

var gitCleanupSleep = time.Sleep

type gitIsolationCleanupResult struct {
	SourceProjectDir string
	WorktreeDir      string
	Branch           string
	WorktreeRemoved  bool
	BranchDeleted    bool
}

type gitIsolationMetadata struct {
	sourceProjectDir string
	worktreeDir      string
	branch           string
}

type staleWorktreeJanitorResult struct {
	Candidates      int
	WorktreesPruned int
	BranchesDeleted int
	SkippedActive   int
	SkippedFresh    int
	Failures        int
}

// RemoveWorktree removes a managed git worktree for the workspace.
// Returns true when removal succeeded or the worktree no longer exists.
func RemoveWorktree(projectDir, workspaceName string) bool {
	root := canonicalPath(projectDir)
	workspace := strings.TrimSpace(workspaceName)
	if root == "" || workspace == "" {
		return false
	}

	worktreeDir := filepath.Join(root, ".orch", "worktrees", workspace)
	return removeManagedWorktree(root, worktreeDir)
}

// CleanAgentBranch deletes an agent branch from the source repository.
// Returns true when deletion succeeded or the branch does not exist.
func CleanAgentBranch(projectDir, branchName string) bool {
	root := canonicalPath(projectDir)
	if root == "" {
		return false
	}
	return deleteManagedBranch(root, branchName)
}

func cleanupManagedGitIsolation(workspacePath, fallbackSourceDir string) gitIsolationCleanupResult {
	meta := resolveGitIsolationMetadata(workspacePath, fallbackSourceDir)
	result := gitIsolationCleanupResult{
		SourceProjectDir: meta.sourceProjectDir,
		WorktreeDir:      meta.worktreeDir,
		Branch:           meta.branch,
	}

	if meta.sourceProjectDir == "" || meta.worktreeDir == "" {
		return result
	}

	workspaceName := managedWorkspaceName(meta.sourceProjectDir, meta.worktreeDir)
	if workspaceName != "" {
		result.WorktreeRemoved = RemoveWorktree(meta.sourceProjectDir, workspaceName)
	} else {
		result.WorktreeRemoved = removeManagedWorktree(meta.sourceProjectDir, meta.worktreeDir)
	}
	if shouldDeleteManagedBranch(meta.branch) {
		result.BranchDeleted = CleanAgentBranch(meta.sourceProjectDir, meta.branch)
	}

	return result
}

func cleanupStaleManagedWorktrees(projectDir string, staleDays int, dryRun bool) (staleWorktreeJanitorResult, error) {
	result := staleWorktreeJanitorResult{}
	worktreeRoot := filepath.Join(projectDir, ".orch", "worktrees")

	if _, err := os.Stat(worktreeRoot); os.IsNotExist(err) {
		return result, nil
	}

	entries, err := os.ReadDir(worktreeRoot)
	if err != nil {
		return result, fmt.Errorf("failed to read worktree root: %w", err)
	}

	if staleDays < 0 {
		staleDays = 0
	}

	active, err := activeManagedWorkspaces(projectDir)
	if err != nil {
		return result, err
	}
	cutoff := time.Now().AddDate(0, 0, -staleDays)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspaceName := strings.TrimSpace(entry.Name())
		if workspaceName == "" {
			continue
		}

		if _, ok := active[workspaceName]; ok {
			result.SkippedActive++
			continue
		}

		path := canonicalPath(filepath.Join(worktreeRoot, workspaceName))
		if path == "" {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			result.Failures++
			fmt.Fprintf(os.Stderr, "Warning: failed to stat worktree %s: %v\n", path, err)
			continue
		}

		if staleDays > 0 && info.ModTime().After(cutoff) {
			result.SkippedFresh++
			continue
		}

		result.Candidates++
		branch := gitCurrentBranch(path)
		if branch == "" {
			branch = "agent/" + workspaceName
		}
		age := int(time.Since(info.ModTime()).Hours() / 24)

		if dryRun {
			fmt.Printf("  [DRY-RUN] Would prune orphaned worktree: %s (%d days old)\n", path, age)
			result.WorktreesPruned++
			if shouldDeleteManagedBranch(branch) {
				fmt.Printf("  [DRY-RUN] Would delete branch: %s\n", branch)
				result.BranchesDeleted++
			}
			continue
		}

		if !RemoveWorktree(projectDir, workspaceName) {
			result.Failures++
			continue
		}

		result.WorktreesPruned++
		if shouldDeleteManagedBranch(branch) {
			if CleanAgentBranch(projectDir, branch) {
				result.BranchesDeleted++
			} else {
				result.Failures++
			}
		}
	}

	return result, nil
}

func activeManagedWorkspaces(projectDir string) (map[string]struct{}, error) {
	active := map[string]struct{}{}
	db, err := statedb.OpenDefault()
	if err != nil {
		return nil, fmt.Errorf("failed to open state db: %w", err)
	}
	if db == nil {
		return nil, fmt.Errorf("state db unavailable")
	}
	defer db.Close()

	agents, err := db.ListActiveAgents()
	if err != nil {
		return nil, fmt.Errorf("failed to list active agents: %w", err)
	}

	root := canonicalPath(projectDir)
	projectName := filepath.Base(root)
	for _, agent := range agents {
		if agent == nil || strings.TrimSpace(agent.WorkspaceName) == "" {
			continue
		}

		agentProjectDir := canonicalPath(agent.ProjectDir)
		if agentProjectDir != "" {
			if agentProjectDir != root {
				continue
			}
		} else if strings.TrimSpace(agent.ProjectName) != "" && strings.TrimSpace(agent.ProjectName) != projectName {
			continue
		}

		active[agent.WorkspaceName] = struct{}{}
	}

	return active, nil
}

func resolveGitIsolationMetadata(workspacePath, fallbackSourceDir string) gitIsolationMetadata {
	meta := gitIsolationMetadata{
		sourceProjectDir: strings.TrimSpace(fallbackSourceDir),
	}

	sourceProjectDir, worktreeDir, branch := readGitIsolationManifest(workspacePath)
	if sourceProjectDir != "" {
		meta.sourceProjectDir = sourceProjectDir
	}
	meta.worktreeDir = worktreeDir
	meta.branch = branch

	if meta.sourceProjectDir == "" {
		return gitIsolationMetadata{}
	}

	if !isManagedWorktreePath(meta.sourceProjectDir, meta.worktreeDir) {
		return gitIsolationMetadata{sourceProjectDir: meta.sourceProjectDir}
	}

	if meta.branch == "" {
		meta.branch = gitCurrentBranch(meta.worktreeDir)
	}

	return meta
}

func isManagedWorktreePath(sourceProjectDir, worktreeDir string) bool {
	if strings.TrimSpace(sourceProjectDir) == "" || strings.TrimSpace(worktreeDir) == "" {
		return false
	}

	source := canonicalPath(sourceProjectDir)
	worktree := canonicalPath(worktreeDir)
	if source == "" || worktree == "" {
		return false
	}

	root := filepath.Join(source, ".orch", "worktrees")
	rel, err := filepath.Rel(root, worktree)
	if err != nil {
		return false
	}

	if rel == "." || rel == ".." {
		return false
	}

	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}

	return true
}

func managedWorkspaceName(sourceProjectDir, worktreeDir string) string {
	if !isManagedWorktreePath(sourceProjectDir, worktreeDir) {
		return ""
	}

	root := filepath.Join(canonicalPath(sourceProjectDir), ".orch", "worktrees")
	worktree := canonicalPath(worktreeDir)
	rel, err := filepath.Rel(root, worktree)
	if err != nil || rel == "." {
		return ""
	}

	if strings.HasPrefix(rel, "..") {
		return ""
	}

	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) == 0 {
		return ""
	}

	return strings.TrimSpace(parts[0])
}

func removeManagedWorktree(sourceProjectDir, worktreeDir string) bool {
	if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
		return true
	}

	lastOutput := ""
	for i := 0; i < gitCleanupRetries; i++ {
		output, err := runGitCleanup(sourceProjectDir, "worktree", "remove", "--force", worktreeDir)
		lastOutput = output
		if err == nil || isWorktreeAlreadyRemoved(output) {
			fmt.Printf("Removed git worktree: %s\n", worktreeDir)
			return true
		}

		if i < gitCleanupRetries-1 {
			_, _ = runGitCleanup(sourceProjectDir, "worktree", "prune")
			gitCleanupSleep(gitCleanupRetryDelay)
		}
	}

	fmt.Fprintf(os.Stderr, "Warning: failed to remove git worktree %s after %d attempts: %s\n", worktreeDir, gitCleanupRetries, strings.TrimSpace(lastOutput))
	return false
}

func deleteManagedBranch(sourceProjectDir, branch string) bool {
	if !shouldDeleteManagedBranch(branch) {
		return false
	}

	if !gitBranchExists(sourceProjectDir, branch) {
		return true
	}

	if current := gitCurrentBranch(sourceProjectDir); current == branch {
		fmt.Fprintf(os.Stderr, "Warning: skipping delete of active branch %s in %s\n", branch, sourceProjectDir)
		return false
	}

	lastOutput := ""
	for i := 0; i < gitCleanupRetries; i++ {
		output, err := runGitCleanup(sourceProjectDir, "branch", "-d", branch)
		lastOutput = output
		if err == nil || isBranchAlreadyRemoved(output) {
			fmt.Printf("Deleted git branch: %s\n", branch)
			return true
		}

		if i < gitCleanupRetries-1 {
			gitCleanupSleep(gitCleanupRetryDelay)
		}
	}

	fmt.Fprintf(os.Stderr, "Warning: failed to delete git branch %s after %d attempts: %s\n", branch, gitCleanupRetries, strings.TrimSpace(lastOutput))
	return false
}

func shouldDeleteManagedBranch(branch string) bool {
	return strings.HasPrefix(strings.TrimSpace(branch), "agent/")
}

func gitCurrentBranch(dir string) string {
	output, err := runGitCleanup(dir, "branch", "--show-current")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}

func gitBranchExists(dir, branch string) bool {
	if strings.TrimSpace(dir) == "" || strings.TrimSpace(branch) == "" {
		return false
	}

	cmd := exec.Command("git", "-C", dir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

func runGitCleanup(dir string, args ...string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("git directory is empty")
	}

	allArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", allArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func isWorktreeAlreadyRemoved(output string) bool {
	text := strings.ToLower(output)
	return strings.Contains(text, "is not a working tree") ||
		strings.Contains(text, "no such file") ||
		strings.Contains(text, "not found") ||
		strings.Contains(text, "does not exist")
}

func isBranchAlreadyRemoved(output string) bool {
	text := strings.ToLower(output)
	return strings.Contains(text, "not found") ||
		strings.Contains(text, "not a valid branch") ||
		strings.Contains(text, "unknown revision")
}

func canonicalPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(absPath)
}

func readGitIsolationManifest(workspacePath string) (string, string, string) {
	manifestPath := filepath.Join(workspacePath, spawn.AgentManifestFilename)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", "", ""
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", "", ""
	}

	sourceProjectDir := readManifestString(manifest, "source_project_dir")
	if sourceProjectDir == "" {
		sourceProjectDir = readManifestString(manifest, "project_dir")
	}

	gitWorktreeDir := readManifestString(manifest, "git_worktree_dir")
	if gitWorktreeDir == "" {
		gitWorktreeDir = sourceProjectDir
	}

	gitBranch := readManifestString(manifest, "git_branch")
	return sourceProjectDir, gitWorktreeDir, gitBranch
}

func readManifestString(manifest map[string]interface{}, key string) string {
	raw, ok := manifest[key]
	if !ok {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}
