// Package verify provides verification helpers for agent completion.
package verify

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// BuildVerificationResult represents the result of checking if the project builds and passes vet.
type BuildVerificationResult struct {
	Passed      bool     // Whether all checks (build + vet) succeeded
	BuildPassed bool     // Whether go build succeeded
	VetPassed   bool     // Whether go vet succeeded
	HasGoFiles  bool     // Whether Go files exist in the project
	BuildOutput string   // Output from the build command (truncated if long)
	VetOutput   string   // Output from the vet command (truncated if long)
	Errors      []string // Error messages (blocking)
	Warnings    []string // Warning messages (non-blocking)
	SkillName   string   // Skill that was used
}

// IsGoProject checks if the project directory contains Go files.
// Looks for go.mod or any .go files in common locations.
func IsGoProject(projectDir string) bool {
	// Check for go.mod (primary indicator)
	goModPath := filepath.Join(projectDir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		return true
	}

	// Check for any .go files in the root or common directories
	patterns := []string{
		filepath.Join(projectDir, "*.go"),
		filepath.Join(projectDir, "cmd", "**", "*.go"),
		filepath.Join(projectDir, "pkg", "**", "*.go"),
		filepath.Join(projectDir, "internal", "**", "*.go"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return true
		}
	}

	return false
}

// HasGoChangesInRecentCommits checks if any Go files were modified
// in recent commits that would require build verification.
func HasGoChangesInRecentCommits(projectDir string) bool {
	// Get changed files from last 5 commits
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Try with fewer commits
		cmd = exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return false
		}
	}

	return hasGoChangesInFiles(string(output))
}

// hasGoChangesInFiles checks if any files in the output are Go files.
func hasGoChangesInFiles(gitOutput string) bool {
	lines := strings.Split(gitOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, ".go") {
			return true
		}
	}
	return false
}

// RunGoBuild runs 'go build ./...' in the project directory.
// Returns the build output and any error that occurred.
func RunGoBuild(projectDir string) (string, error) {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = projectDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout and stderr
	output := stdout.String() + stderr.String()

	return output, err
}

// RunGoVet runs 'go vet ./...' in the project directory.
// Returns the vet output and any error that occurred.
func RunGoVet(projectDir string) (string, error) {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = projectDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout and stderr
	output := stdout.String() + stderr.String()

	return output, err
}

// VerifyBuild checks if the Go project builds and passes vet.
// This is a gate that blocks completion if Go files were modified
// but the project fails to build or vet.
//
// The verification passes if:
// 1. The project is not a Go project (no go.mod or .go files), OR
// 2. No Go files were modified in recent commits, OR
// 3. The skill is not an implementation-focused skill, OR
// 4. The project builds successfully with 'go build ./...' AND passes 'go vet ./...'
func VerifyBuild(workspacePath, projectDir string) BuildVerificationResult {
	result := BuildVerificationResult{Passed: true, BuildPassed: true, VetPassed: true}

	// Extract skill name for tracking
	skillName, _ := ExtractSkillNameFromSpawnContext(workspacePath)
	result.SkillName = skillName

	// Gate selection is handled by the verify level system (V0-V3) in check.go.
	// This function runs unconditionally when called — the caller decides whether to invoke it.

	// Check if this is a Go project
	result.HasGoFiles = IsGoProject(projectDir)
	if !result.HasGoFiles {
		result.Warnings = append(result.Warnings,
			"not a Go project - build verification not required")
		return result
	}

	// Check if Go files were modified
	if !HasGoChangesInRecentCommits(projectDir) {
		result.Warnings = append(result.Warnings,
			"no Go files modified - build verification not required")
		return result
	}

	// Run go build and vet against committed state (stash uncommitted changes).
	// This ensures the gate verifies what's actually committed, not working-tree edits.
	var buildOutput, vetOutput string
	var buildErr, vetErr error

	stashErr := withCommittedState(projectDir, func() error {
		buildOutput, buildErr = RunGoBuild(projectDir)
		if buildErr != nil {
			return buildErr // skip vet if build fails
		}
		vetOutput, vetErr = RunGoVet(projectDir)
		return nil
	})

	// If stash mechanism itself failed, the build/vet still ran against working tree
	_ = stashErr

	result.BuildOutput = truncateOutput(buildOutput, 500)

	if buildErr != nil {
		result.Passed = false
		result.BuildPassed = false
		result.VetPassed = false // not run
		result.Errors = append(result.Errors,
			"'go build ./...' failed (against committed state)",
			"Build must pass before completion",
		)
		if buildOutput != "" {
			result.Errors = append(result.Errors, "Build output: "+result.BuildOutput)
		}
		// Skip vet if build fails - vet errors would be noise
		return result
	}

	result.VetOutput = truncateOutput(vetOutput, 500)

	if vetErr != nil {
		result.Passed = false
		result.VetPassed = false
		result.Errors = append(result.Errors,
			"'go vet ./...' failed (against committed state)",
			"Vet must pass before completion",
		)
		if vetOutput != "" {
			result.Errors = append(result.Errors, "Vet output: "+result.VetOutput)
		}
	}

	return result
}

// truncateOutput truncates output to a maximum number of characters.
func truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "... (truncated)"
}

// isWorktreeDirty checks if the git working tree has uncommitted changes
// (staged, unstaged, or untracked files).
func isWorktreeDirty(projectDir string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return false // not a git repo or error — treat as clean
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// withCommittedState runs fn against the committed state of the working tree.
// If there are uncommitted changes (staged, unstaged, or untracked), they are
// stashed before fn runs and restored after. This ensures build/vet verify
// what's actually committed, not what's in the working tree.
func withCommittedState(projectDir string, fn func() error) error {
	if !isWorktreeDirty(projectDir) {
		return fn()
	}

	// Stash all changes including untracked files
	stashCmd := exec.Command("git", "stash", "push", "--include-untracked", "-m", "orch-build-gate-verify")
	stashCmd.Dir = projectDir
	if out, err := stashCmd.CombinedOutput(); err != nil {
		// Stash failed — run against working tree as fallback
		_ = out
		return fn()
	}

	// Always restore stash, even if fn fails
	defer func() {
		popCmd := exec.Command("git", "stash", "pop")
		popCmd.Dir = projectDir
		if out, err := popCmd.CombinedOutput(); err != nil {
			// Pop failed (shouldn't happen with our own stash) — log but don't mask fn error
			_ = fmt.Errorf("git stash pop failed: %v: %s", err, out)
		}
	}()

	return fn()
}

// HasGoChangesForAgent checks if the agent modified any Go files based on
// the agent's spawn baseline (from AGENT_MANIFEST.json). This is more precise
// than HasGoChangesInRecentCommits which checks HEAD~5..HEAD globally and may
// include changes from other agents.
// Returns false if the workspace is empty, has no manifest, or no baseline.
func HasGoChangesForAgent(workspacePath, projectDir string) bool {
	if workspacePath == "" || projectDir == "" {
		return false
	}

	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	baseline := manifest.GitBaseline
	if baseline == "" {
		// No baseline — fall back to global check
		return HasGoChangesInRecentCommits(projectDir)
	}

	cmd := exec.Command("git", "diff", "--name-only", baseline+"..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Baseline may be gc'd — fall back to global check
		return HasGoChangesInRecentCommits(projectDir)
	}

	return hasGoChangesInFiles(string(output))
}

// VerifyBuildForCompletion is a convenience function for use in VerifyCompletionFull.
// Returns nil if no verification is needed (not a Go project, no Go changes, or non-implementation skill).
// Returns EscalationBlock level result if build fails.
// Uses agent-specific baseline to avoid running go build when THIS agent didn't touch Go files.
func VerifyBuildForCompletion(workspacePath, projectDir string) *BuildVerificationResult {
	// Return nil if not a Go project - no action needed
	if !IsGoProject(projectDir) {
		return nil
	}

	// Check agent-specific Go changes first (scoped to this agent's baseline).
	// This avoids running go build/vet for agents that only modified .kb/, .md, etc.
	if !HasGoChangesForAgent(workspacePath, projectDir) {
		return nil
	}

	result := VerifyBuild(workspacePath, projectDir)
	if !result.HasGoFiles {
		return nil
	}

	return &result
}
