// Package verify provides verification helpers for agent completion.
package verify

import (
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	recentDiffRange   = "HEAD~5..HEAD"
	fallbackDiffRange = "HEAD~1..HEAD"
)

// GetChangedFiles returns changed files from either recent commits or since a specific time.
// If since is empty, it checks HEAD~5..HEAD with HEAD~1..HEAD fallback.
// If since is non-empty, it checks all files from git log --since.
func GetChangedFiles(projectDir, since string) ([]string, error) {
	return getChangedFiles(projectDir, since)
}

func getChangedFiles(projectDir, since string) ([]string, error) {
	if since == "" {
		output, err := runGitDiffWithFallback(projectDir, "--name-only")
		if err != nil {
			return nil, err
		}
		return parseFileList(output), nil
	}

	output, err := runGitOutput(projectDir, "log", "--name-only", "--since="+since, "--format=")
	if err != nil {
		return nil, err
	}

	return parseFileList(output), nil
}

// GetChangedNameStatus returns git diff --name-status lines from recent commits.
// Uses HEAD~5..HEAD with HEAD~1..HEAD fallback.
func GetChangedNameStatus(projectDir string) ([]string, error) {
	return getChangedNameStatus(projectDir)
}

func getChangedNameStatus(projectDir string) ([]string, error) {
	output, err := runGitDiffWithFallback(projectDir, "--name-status")
	if err != nil {
		return nil, err
	}
	return parseFileList(output), nil
}

func getChangedNumstat(projectDir string) (string, error) {
	return runGitDiffWithFallback(projectDir, "--numstat")
}

// getCommitHashes returns commit hashes since a timestamp, optionally scoped to a path.
// If path is absolute and projectDir is absolute, it is converted to a project-relative path.
func getCommitHashes(projectDir, path, since string) ([]string, error) {
	args := []string{"log"}
	if since != "" {
		args = append(args, "--since="+since)
	}
	args = append(args, "--format=%H")

	if path != "" {
		args = append(args, "--", resolveGitPath(projectDir, path))
	}

	output, err := runGitOutput(projectDir, args...)
	if err != nil {
		return nil, err
	}

	return parseFileList(output), nil
}

// getFileChangesForCommit returns all file paths changed by a commit.
func getFileChangesForCommit(projectDir, hash string) ([]string, error) {
	output, err := runGitOutput(projectDir, "show", "--name-only", "--format=", hash)
	if err != nil {
		return nil, err
	}

	return parseFileList(output), nil
}

func getNumstatForCommit(projectDir, hash string) (string, error) {
	return runGitOutput(projectDir, "show", "--numstat", "--format=", hash)
}

func fileExistsInPreviousCommit(projectDir, filePath string) bool {
	_, err := runGitOutput(projectDir, "cat-file", "-e", "HEAD~1:"+filePath)
	return err == nil
}

// GetLatestCommitUnixTimestamp returns git log -1 --format=%ct output for pathspecs.
func GetLatestCommitUnixTimestamp(projectDir string, pathspecs ...string) (string, error) {
	args := []string{"log", "-1", "--format=%ct"}
	if len(pathspecs) > 0 {
		args = append(args, "--")
		args = append(args, pathspecs...)
	}

	output, err := runGitOutput(projectDir, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

func runGitDiffWithFallback(projectDir string, diffArgs ...string) (string, error) {
	primary := append([]string{"diff"}, append(diffArgs, recentDiffRange)...)
	output, err := runGitOutput(projectDir, primary...)
	if err == nil {
		return output, nil
	}

	fallback := append([]string{"diff"}, append(diffArgs, fallbackDiffRange)...)
	return runGitOutput(projectDir, fallback...)
}

func runGitOutput(projectDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	return string(output), err
}

func resolveGitPath(projectDir, path string) string {
	resolved := path
	if filepath.IsAbs(path) && filepath.IsAbs(projectDir) {
		rel, err := filepath.Rel(projectDir, path)
		if err == nil {
			resolved = rel
		}
	}
	return resolved
}
