// Package verify provides verification helpers for agent completion.
package verify

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultBuildTimeout is the maximum time to wait for build/compile commands.
// 60 seconds is generous for incremental builds while preventing indefinite hangs.
const DefaultBuildTimeout = 60 * time.Second

// BuildVerificationResult represents the result of checking if the project builds.
type BuildVerificationResult struct {
	Passed      bool     // Whether the build succeeded
	HasGoFiles  bool     // Whether Go files exist in the project
	BuildOutput string   // Output from the build command (truncated if long)
	Errors      []string // Error messages (blocking)
	Warnings    []string // Warning messages (non-blocking)
	SkillName   string   // Skill that was used
	PreExisting bool     // Whether the build failure was pre-existing (not caused by this agent)
	SkipMemory  bool     // Whether the build gate was auto-skipped via build skip memory
	BlameDetail string   // Human-readable blame attribution detail
}

// Skills explicitly excluded from build verification requirements.
// These skills produce documentation/research artifacts, not code changes.
// Any skill NOT in this list will trigger build verification if Go files were changed.
var skillsExcludedFromBuildVerification = map[string]bool{
	"investigation":  true, // Research skill, produces investigations
	"architect":      true, // Design skill, produces decisions
	"research":       true, // External research, no code changes
	"design-session": true, // Scoping skill, produces epics
	"codebase-audit": true, // Audit skill, produces reports
	"issue-creation": true, // Triage skill, creates issues
	"writing-skills": true, // Meta skill, modifies skills not Go code
}

// IsSkillExcludedFromBuildVerification returns true if the skill is explicitly
// excluded from build verification (documentation/research-only skills).
func IsSkillExcludedFromBuildVerification(skillName string) bool {
	if skillName == "" {
		return false
	}
	return skillsExcludedFromBuildVerification[strings.ToLower(skillName)]
}

// IsSkillRequiringBuildVerification determines if a skill requires build verification.
//
// The logic is:
// 1. If skill is explicitly excluded (investigation, architect, etc.) -> false
// 2. Otherwise -> true (build verification required when Go files are changed)
//
// This is a restrictive default: any skill that touches Go code must pass the build gate.
// Previously used a permissive default (unknown skills skipped), which allowed agents
// to leave broken builds (e.g., 23 files with incomplete refactoring in 2026-02-06 session).
func IsSkillRequiringBuildVerification(skillName string) bool {
	// Empty skill name: still require build verification.
	// Agents without skills can still modify Go code and break the build.
	if skillName == "" {
		return true
	}

	skillName = strings.ToLower(skillName)

	// Check explicit exclusions - these are documentation/research skills
	if skillsExcludedFromBuildVerification[skillName] {
		return false
	}

	// All other skills: require build verification when Go files are changed
	return true
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
	files, err := getChangedFiles(projectDir, "")
	if err != nil {
		return false
	}

	return hasGoChangesInFiles(strings.Join(files, "\n"))
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
//
// DEPRECATED: Use RunGoTestCompile instead - it also compiles test files
// which catches signature mismatches between production code and tests.
func RunGoBuild(projectDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultBuildTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = projectDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout and stderr
	output := stdout.String() + stderr.String()

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("go build timed out after %v", DefaultBuildTimeout)
	}

	return output, err
}

// RunGoTestCompile compiles all Go code including test files without running tests.
// Uses 'go test -run=^$' which compiles all code (production and test) but runs no tests
// (the pattern '^$' matches no test names).
//
// This is preferred over RunGoBuild because 'go build' only compiles production
// code - it doesn't compile *_test.go files. This means changes to function
// signatures can break tests without being caught by 'go build'.
//
// Returns the output and any error that occurred.
func RunGoTestCompile(projectDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultBuildTimeout)
	defer cancel()

	// Use 'go test -run=^$ ./...' to compile all packages including tests
	// The -run=^$ flag matches no test names, so it compiles but runs nothing
	// This catches compilation errors in both production and test code
	cmd := exec.CommandContext(ctx, "go", "test", "-run=^$", "./...")
	cmd.Dir = projectDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout and stderr
	output := stdout.String() + stderr.String()

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("go test compile timed out after %v", DefaultBuildTimeout)
	}

	return output, err
}

// VerifyBuild checks if the Go project builds successfully, including test files.
// This is a gate that blocks completion if Go files were modified
// but the project fails to compile.
//
// The verification passes if:
// 1. The skill is explicitly excluded from build verification (documentation/research skills), OR
// 2. The project is not a Go project (no go.mod or .go files), OR
// 3. No Go files were modified in recent commits, OR
// 4. The project compiles successfully (both production and test code), OR
// 5. Build failure is pre-existing (not caused by this agent - blame attribution), OR
// 6. Build gate was previously skipped and skip memory is still valid
//
// IMPORTANT: The default is restrictive - any agent that modifies Go files must pass
// the build gate, regardless of skill name. Only explicitly excluded skills (investigation,
// architect, etc.) skip the gate. This prevents agents from leaving broken builds
// (e.g., partial refactorings with undefined variable errors).
//
// Uses 'go test -run=^$' instead of 'go build' because 'go build'
// does NOT compile test files (*_test.go). This means function signature changes
// can break tests without being caught. Using 'go test -run=^$' ensures both
// production code AND test code compile correctly.
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

	// Check gate skip memory - if orchestrator already skipped build gate recently,
	// auto-skip for subsequent completions with a warning
	if skip := ReadGateSkipMemory(projectDir, GateBuild); skip != nil {
		result.SkipMemory = true
		result.Warnings = append(result.Warnings,
			"build gate auto-skipped (prior skip by "+skip.SetBy+": "+skip.Reason+")")
		result.Warnings = append(result.Warnings,
			"build skip expires at "+skip.ExpiresAt.Format("15:04:05"))
		return result
	}

	// Run 'go test -run=^$' to compile both production code and test files
	// This catches signature mismatches that 'go build' would miss
	output, err := RunGoTestCompile(projectDir)
	result.BuildOutput = truncateOutput(output, 500)

	if err != nil {
		// Build failed - check blame attribution before blocking
		blame := AttributeBuildFailure(workspacePath, projectDir)
		result.BlameDetail = blame.BlameDetail

		if blame.PreExisting {
			// Build was already broken before this agent's commits
			// Pass the gate with a warning instead of blocking
			result.PreExisting = true
			result.Warnings = append(result.Warnings,
				"build failure is pre-existing (not caused by this agent)")
			result.Warnings = append(result.Warnings,
				"blame: "+blame.BlameDetail)
			if output != "" {
				result.Warnings = append(result.Warnings,
					"build output (for reference): "+result.BuildOutput)
			}
			return result
		}

		// Agent caused the build failure - block completion
		result.Passed = false
		result.Errors = append(result.Errors,
			"'go test -run=^$ ./...' failed (compilation error in production or test code)",
			"Both production and test code must compile before completion",
		)
		if blame.BlameDetail != "" {
			result.Errors = append(result.Errors, "blame: "+blame.BlameDetail)
		}
		if output != "" {
			result.Errors = append(result.Errors, "Compilation output: "+result.BuildOutput)
		}
	} else {
		// Build passed - clear any stale build skip memory
		ClearGateSkipMemory(projectDir, GateBuild)
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
// Returns nil if no verification is needed (not a Go project, no Go changes, or excluded skill).
// Returns EscalationBlock level result if build fails.
// Returns a passing result (with warnings) if failure is pre-existing or skip memory is active.
func VerifyBuildForCompletion(workspacePath, projectDir string) *BuildVerificationResult {
	result := VerifyBuild(workspacePath, projectDir)

	// Return nil if not a Go project - no action needed
	if !result.HasGoFiles {
		return nil
	}

	// Return nil if skill is explicitly excluded from build verification
	if IsSkillExcludedFromBuildVerification(result.SkillName) {
		return nil
	}

	// Return nil if no Go changes
	if !HasGoChangesInRecentCommits(projectDir) {
		return nil
	}

	return &result
}

// RecordBuildSkip persists a build gate skip decision for future completions.
// Called when the orchestrator uses --skip-build --skip-reason to bypass the build gate.
// Subsequent completions will auto-skip the build gate until the skip expires.
//
// Deprecated: Use WriteGateSkipMemory(projectDir, GateBuild, reason, skippedBy) directly.
func RecordBuildSkip(projectDir, reason, skippedBy string) error {
	return WriteGateSkipMemory(projectDir, GateBuild, reason, skippedBy)
}

// RecordGateSkip persists a gate skip decision for future completions.
// Called when the orchestrator uses --skip-* --skip-reason to bypass a gate.
// Subsequent completions will auto-skip the gate until the skip expires.
func RecordGateSkip(projectDir, gate, reason, skippedBy string) error {
	return WriteGateSkipMemory(projectDir, gate, reason, skippedBy)
}
