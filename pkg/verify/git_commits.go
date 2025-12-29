// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// GitCommitResult represents the result of verifying git commits exist.
type GitCommitResult struct {
	Passed        bool     // Whether verification passed
	HasCommits    bool     // Whether any commits exist since spawn time
	CommitCount   int      // Number of commits found
	Errors        []string // Error messages (blocking)
	Warnings      []string // Warning messages (non-blocking)
	SkillName     string   // Skill that was used
	SpawnTime     time.Time // When the agent was spawned
	IsCodeSkill   bool     // Whether this skill typically produces code
}

// Skills that typically produce code and should have git commits.
// These skills modify source code as their primary output.
var codeProducingSkills = map[string]bool{
	"feature-impl":         true, // Primary implementation skill
	"systematic-debugging": true, // Debug fixes should produce commits
	"reliability-testing":  true, // May produce fix commits
}

// Skills that produce artifacts (investigations, decisions) rather than code.
// These skills are exempt from git commit verification.
var artifactProducingSkills = map[string]bool{
	"investigation":   true, // Research skill, produces .kb/ investigations
	"architect":       true, // Design skill, produces decisions
	"research":        true, // External research, produces reports
	"design-session":  true, // Scoping skill, produces epics
	"codebase-audit":  true, // Audit skill, produces reports
	"issue-creation":  true, // Triage skill, creates beads issues
	"writing-skills":  true, // Meta skill, modifies skills (may not always commit)
}

// IsCodeProducingSkill determines if a skill typically produces code commits.
//
// The logic is:
// 1. If skill is explicitly artifact-producing (investigation, architect, etc.) -> false
// 2. If skill is explicitly code-producing (feature-impl, debugging) -> true
// 3. If skill is unknown -> false (permissive default)
func IsCodeProducingSkill(skillName string) bool {
	if skillName == "" {
		return false
	}

	skillName = strings.ToLower(skillName)

	// Check explicit exclusions first (artifact-producing skills)
	if artifactProducingSkills[skillName] {
		return false
	}

	// Check explicit inclusions (code-producing skills)
	if codeProducingSkills[skillName] {
		return true
	}

	// Unknown skill - be permissive (don't block completion)
	return false
}

// CountCommitsSinceTime returns the number of git commits since the given time.
// Uses `git log --since` with ISO 8601 format for precision.
func CountCommitsSinceTime(projectDir string, since time.Time) (int, error) {
	if since.IsZero() {
		return 0, fmt.Errorf("spawn time is zero")
	}

	// Format time for git --since flag (ISO 8601)
	sinceStr := since.Format(time.RFC3339)

	// Count commits since spawn time
	cmd := exec.Command("git", "log", "--oneline", "--since="+sinceStr)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to run git log: %w", err)
	}

	// Count non-empty lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	return count, nil
}

// VerifyGitCommits checks if git commits exist since the agent was spawned.
// This is a gate that blocks completion if a code-producing skill has no commits.
//
// The verification passes if:
// 1. The skill is not a code-producing skill (investigation, architect, etc.), OR
// 2. At least one git commit exists since spawn time
//
// The verification fails if:
// 1. The skill is code-producing (feature-impl, systematic-debugging), AND
// 2. No git commits exist since spawn time
func VerifyGitCommits(workspacePath, projectDir string) GitCommitResult {
	result := GitCommitResult{Passed: true}

	// Extract skill name
	skillName, _ := ExtractSkillNameFromSpawnContext(workspacePath)
	result.SkillName = skillName

	// Check if this is a code-producing skill
	result.IsCodeSkill = IsCodeProducingSkill(skillName)

	// If not a code-producing skill, skip verification
	if !result.IsCodeSkill {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("skill '%s' is artifact-producing - git commit verification skipped", skillName))
		return result
	}

	// Read spawn time from workspace
	spawnTime := spawn.ReadSpawnTime(workspacePath)
	result.SpawnTime = spawnTime

	if spawnTime.IsZero() {
		result.Warnings = append(result.Warnings,
			"no .spawn_time file found in workspace - git commit verification skipped")
		return result
	}

	// Count commits since spawn time
	count, err := CountCommitsSinceTime(projectDir, spawnTime)
	if err != nil {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("failed to count git commits: %v", err))
		// Don't fail verification if git command fails
		return result
	}

	result.CommitCount = count
	result.HasCommits = count > 0

	if !result.HasCommits {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("skill '%s' is code-producing but no git commits found since spawn time (%s)",
				skillName, spawnTime.Format(time.RFC3339)),
			"Agent reported Phase: Complete but has no git commits",
			"This is a false positive - agent did not actually commit any work",
			"Either the agent failed to commit, or the wrong skill was used",
		)
	}

	return result
}

// VerifyGitCommitsForCompletion is a convenience function for use in VerifyCompletionFull.
// Returns nil if no verification is needed (artifact-producing skill or missing spawn time).
// Returns blocking result if a code-producing skill has no commits.
func VerifyGitCommitsForCompletion(workspacePath, projectDir string) *GitCommitResult {
	result := VerifyGitCommits(workspacePath, projectDir)

	// Return nil if not a code-producing skill - no action needed
	if !result.IsCodeSkill {
		return nil
	}

	// Return nil if spawn time is missing - can't verify
	if result.SpawnTime.IsZero() {
		return nil
	}

	return &result
}
