// Package verify provides blame attribution for the build gate.
//
// When concurrent agents break the build (e.g., duplicate declarations), every
// subsequent orch complete fails the build gate. This file provides blame
// attribution: Check if the build failure is caused by THIS agent's commits
// vs pre-existing. Only gate if this agent broke it.
//
// Gate skip memory (for persisting skip decisions across completions) has been
// generalized to gate_skip_memory.go which supports any gate, not just build.
package verify

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// BuildBlameResult represents the result of blame attribution for a build failure.
type BuildBlameResult struct {
	AgentCausedFailure bool   // Whether this agent's commits caused the build failure
	PreExisting        bool   // Whether the build was already broken before this agent
	BlameDetail        string // Human-readable explanation
}

// AttributeBuildFailure determines whether the build failure was caused by
// this agent's commits or was pre-existing.
//
// Strategy:
// 1. Read the agent's spawn time from workspace
// 2. Find commits made since spawn time
// 3. If no commits since spawn time, failure is pre-existing
// 4. Stash this agent's commits (git stash), try building, restore
//   - If build still fails without agent's changes → pre-existing
//   - If build passes without agent's changes → agent caused it
//
// For safety and simplicity, we use a lightweight approach:
// Check if the build was already broken at the commit before the agent's
// first commit, by running git stash/build/restore.
func AttributeBuildFailure(workspacePath, projectDir string) BuildBlameResult {
	result := BuildBlameResult{
		AgentCausedFailure: true, // Default: assume agent is responsible
	}

	spawnTime := spawn.ReadSpawnTime(workspacePath)
	if spawnTime.IsZero() {
		result.BlameDetail = "no spawn time found, assuming agent responsibility"
		return result
	}

	// Find commits since spawn time
	commits := commitsAfterTime(projectDir, spawnTime)
	if len(commits) == 0 {
		// No commits from this agent - build failure is pre-existing
		result.AgentCausedFailure = false
		result.PreExisting = true
		result.BlameDetail = "no commits since spawn time, build failure is pre-existing"
		return result
	}

	// Find the first commit hash from this agent
	firstCommit := commits[len(commits)-1] // oldest commit (list is newest-first)

	// Get the parent of the first commit (the state before this agent)
	parent := parentCommit(projectDir, firstCommit)
	if parent == "" {
		result.BlameDetail = "could not determine parent commit, assuming agent responsibility"
		return result
	}

	// Test if the build was already broken at the parent commit.
	// We use `git stash` approach: check out parent, build, come back.
	// For safety, use a simpler approach: compile at parent commit using worktree-free method.
	broken := wasBuildBrokenAt(projectDir, parent)
	if broken {
		result.AgentCausedFailure = false
		result.PreExisting = true
		result.BlameDetail = fmt.Sprintf("build was already broken at %s (before agent's commits)", parent[:8])
		return result
	}

	result.BlameDetail = fmt.Sprintf("build was passing at %s (before agent's commits), agent introduced the failure", parent[:8])
	return result
}

// commitsAfterTime returns commit hashes made after the given time, newest first.
func commitsAfterTime(projectDir string, after time.Time) []string {
	// Use --after with ISO format
	cmd := exec.Command("git", "log", "--format=%H",
		"--after="+after.Format(time.RFC3339))
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var commits []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			commits = append(commits, line)
		}
	}
	return commits
}

// parentCommit returns the parent hash of the given commit.
func parentCommit(projectDir, commit string) string {
	cmd := exec.Command("git", "rev-parse", commit+"^")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// wasBuildBrokenAt checks if the build was broken at a specific commit.
// Uses `git stash` to temporarily check out the state at that commit,
// runs the build, then restores. This is safe because it uses git stash
// which preserves the working tree.
//
// For a less invasive approach, we use `git show` to check if the commit
// introduced Go compilation errors by building at that revision using
// a temporary worktree.
func wasBuildBrokenAt(projectDir, commit string) bool {
	// Create a temporary worktree to avoid disturbing the main working tree
	tmpDir, err := os.MkdirTemp("", "orch-build-blame-*")
	if err != nil {
		return false // Can't determine, assume not broken
	}
	defer os.RemoveAll(tmpDir)

	// Use git worktree to check out the specific commit
	cmd := exec.Command("git", "worktree", "add", "--detach", tmpDir, commit)
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		return false // Can't create worktree, assume not broken
	}
	defer func() {
		// Clean up worktree
		rm := exec.Command("git", "worktree", "remove", "--force", tmpDir)
		rm.Dir = projectDir
		rm.Run()
	}()

	// Try building in the temporary worktree
	_, buildErr := RunGoTestCompile(tmpDir)
	return buildErr != nil
}
