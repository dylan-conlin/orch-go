// Package verify provides blame attribution and build skip memory for the build gate.
//
// When concurrent agents break the build (e.g., duplicate declarations), every
// subsequent orch complete fails the build gate. This file provides:
//
//  1. Blame attribution: Check if the build failure is caused by THIS agent's
//     commits vs pre-existing. Only gate if this agent broke it.
//
//  2. Build skip memory: Once orchestrator skips build gate with a reason,
//     remember that reason for subsequent completions in the same session.
package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// BuildSkipMemory represents a persisted build gate skip decision.
// Stored at .orch/build-skip.json to remember across completions.
type BuildSkipMemory struct {
	Reason    string    `json:"reason"`     // Why the build gate was skipped
	SkippedAt time.Time `json:"skipped_at"` // When it was skipped
	SkippedBy string    `json:"skipped_by"` // Who skipped it (beads ID or workspace name)
	ExpiresAt time.Time `json:"expires_at"` // When this skip expires (auto-cleanup)
}

// BuildBlameResult represents the result of blame attribution for a build failure.
type BuildBlameResult struct {
	AgentCausedFailure bool   // Whether this agent's commits caused the build failure
	PreExisting        bool   // Whether the build was already broken before this agent
	BlameDetail        string // Human-readable explanation
}

const (
	// BuildSkipFilename is the name of the build skip memory file.
	BuildSkipFilename = "build-skip.json"

	// BuildSkipDuration is how long a build skip decision lasts.
	// After this duration, the skip memory expires and build gates resume.
	BuildSkipDuration = 2 * time.Hour
)

// buildSkipPath returns the path to the build skip memory file.
func buildSkipPath(projectDir string) string {
	return filepath.Join(projectDir, ".orch", BuildSkipFilename)
}

// ReadBuildSkipMemory reads the persisted build gate skip from disk.
// Returns nil if no skip exists or if it has expired.
func ReadBuildSkipMemory(projectDir string) *BuildSkipMemory {
	path := buildSkipPath(projectDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var skip BuildSkipMemory
	if err := json.Unmarshal(data, &skip); err != nil {
		return nil
	}

	// Check expiry
	if time.Now().After(skip.ExpiresAt) {
		// Expired - clean up
		os.Remove(path)
		return nil
	}

	return &skip
}

// WriteBuildSkipMemory persists a build gate skip decision to disk.
func WriteBuildSkipMemory(projectDir, reason, skippedBy string) error {
	skip := BuildSkipMemory{
		Reason:    reason,
		SkippedAt: time.Now(),
		SkippedBy: skippedBy,
		ExpiresAt: time.Now().Add(BuildSkipDuration),
	}

	data, err := json.MarshalIndent(skip, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal build skip: %w", err)
	}

	path := buildSkipPath(projectDir)

	// Ensure .orch directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create .orch directory: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// ClearBuildSkipMemory removes the build skip memory file.
// Called when the build starts passing again.
func ClearBuildSkipMemory(projectDir string) {
	os.Remove(buildSkipPath(projectDir))
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
