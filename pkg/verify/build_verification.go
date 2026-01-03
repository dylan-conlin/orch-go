// Package verify provides verification helpers for agent completion.
package verify

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildVerificationResult represents the result of checking if the project builds.
type BuildVerificationResult struct {
	Passed       bool     // Whether the build succeeded
	HasGoFiles   bool     // Whether Go files exist in the project
	BuildOutput  string   // Output from the build command (truncated if long)
	Errors       []string // Error messages (blocking)
	Warnings     []string // Warning messages (non-blocking)
	SkillName    string   // Skill that was used
}

// Skills that require build verification before completion.
// Only implementation-focused skills that may modify Go code need build verification.
var skillsRequiringBuildVerification = map[string]bool{
	"feature-impl":         true, // Primary implementation skill
	"systematic-debugging": true, // Debug fixes should build
	"reliability-testing":  true, // Testing skill may modify code
}

// Skills explicitly excluded from build verification requirements.
// These skills may modify files but typically don't break builds.
var skillsExcludedFromBuildVerification = map[string]bool{
	"investigation":  true, // Research skill, produces investigations
	"architect":      true, // Design skill, produces decisions
	"research":       true, // External research, no code changes
	"design-session": true, // Scoping skill, produces epics
	"codebase-audit": true, // Audit skill, produces reports
	"issue-creation": true, // Triage skill, creates issues
	"writing-skills": true, // Meta skill, modifies skills not Go code
}

// IsSkillRequiringBuildVerification determines if a skill requires build verification.
//
// The logic is:
// 1. If skill is explicitly excluded (investigation, architect, etc.) -> false
// 2. If skill is explicitly included (feature-impl, debugging) -> true
// 3. If skill is unknown -> false (permissive default)
func IsSkillRequiringBuildVerification(skillName string) bool {
	if skillName == "" {
		return false
	}

	skillName = strings.ToLower(skillName)

	// Check explicit exclusions first
	if skillsExcludedFromBuildVerification[skillName] {
		return false
	}

	// Check explicit inclusions
	if skillsRequiringBuildVerification[skillName] {
		return true
	}

	// Unknown skill - be permissive
	return false
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

// VerifyBuild checks if the Go project builds successfully.
// This is a gate that blocks completion if Go files were modified
// but the project fails to build.
//
// The verification passes if:
// 1. The project is not a Go project (no go.mod or .go files), OR
// 2. No Go files were modified in recent commits, OR
// 3. The skill is not an implementation-focused skill, OR
// 4. The project builds successfully with 'go build ./...'
func VerifyBuild(workspacePath, projectDir string) BuildVerificationResult {
	result := BuildVerificationResult{Passed: true}

	// Extract skill name for skill-based gating
	skillName, _ := ExtractSkillNameFromSpawnContext(workspacePath)
	result.SkillName = skillName

	// Check if skill requires build verification
	if !IsSkillRequiringBuildVerification(skillName) {
		result.Warnings = append(result.Warnings,
			"skill '"+skillName+"' does not require build verification")
		return result
	}

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

	// Run go build
	output, err := RunGoBuild(projectDir)
	result.BuildOutput = truncateOutput(output, 500)

	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors,
			"'go build ./...' failed",
			"Build must pass before completion",
		)
		if output != "" {
			result.Errors = append(result.Errors, "Build output: "+result.BuildOutput)
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

// VerifyBuildForCompletion is a convenience function for use in VerifyCompletionFull.
// Returns nil if no verification is needed (not a Go project, no Go changes, or non-implementation skill).
// Returns EscalationBlock level result if build fails.
func VerifyBuildForCompletion(workspacePath, projectDir string) *BuildVerificationResult {
	result := VerifyBuild(workspacePath, projectDir)

	// Return nil if not a Go project - no action needed
	if !result.HasGoFiles {
		return nil
	}

	// Return nil if skill doesn't require build verification
	if !IsSkillRequiringBuildVerification(result.SkillName) {
		return nil
	}

	// Return nil if no Go changes (after checking skill - we want the skill warning)
	if !HasGoChangesInRecentCommits(projectDir) {
		return nil
	}

	return &result
}
