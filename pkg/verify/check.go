// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VerificationResult represents the result of a completion verification.
type VerificationResult struct {
	Passed   bool     // Whether all checks passed
	Errors   []string // Errors that prevent completion
	Warnings []string // Warnings that don't block completion
	Phase    PhaseStatus
}

// VerifySynthesis checks if SYNTHESIS.md exists and is not empty.
func VerifySynthesis(workspacePath string) (bool, error) {
	if workspacePath == "" {
		return false, nil
	}
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	info, err := os.Stat(synthesisPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Size() > 0, nil
}

// VerifyCompletion checks if an agent is ready for completion.
// Returns a VerificationResult with any errors or warnings.
// Uses VerifyCompletionWithTier with an empty tier (reads from workspace).
func VerifyCompletion(beadsID string, workspacePath string) (VerificationResult, error) {
	return VerifyCompletionWithTier(beadsID, workspacePath, "")
}

// VerifyCompletionFull checks if an agent is ready for completion including skill constraints
// and phase gates. This extends VerifyCompletion with:
// 1. Constraint verification from SPAWN_CONTEXT.md (file patterns must match)
// 2. Phase gate verification (required phases must be reported via beads comments)
// 3. Skill output verification from skill.yaml outputs.required section
//
// The projectDir is used to verify that constraint patterns match actual files.
func VerifyCompletionFull(beadsID, workspacePath, projectDir, tier string) (VerificationResult, error) {
	// First run standard verification
	result, err := VerifyCompletionWithTier(beadsID, workspacePath, tier)
	if err != nil {
		return result, err
	}

	// If standard verification failed, no need to check constraints
	if !result.Passed {
		return result, nil
	}

	// Skip constraint verification if no workspace
	if workspacePath == "" {
		return result, nil
	}

	// Skip constraint verification if no project dir
	if projectDir == "" {
		return result, nil
	}

	// Verify skill constraints from SPAWN_CONTEXT.md
	constraintResult, err := VerifyConstraintsForCompletion(workspacePath, projectDir)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify constraints: %v", err))
		// Continue to phase gate verification even if constraints failed to parse
	} else {
		// Merge constraint results
		if !constraintResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, constraintResult.Errors...)
		}
		result.Warnings = append(result.Warnings, constraintResult.Warnings...)
	}

	// Verify phase gates from SPAWN_CONTEXT.md
	// This checks that required phases were reported in beads comments
	phaseGateResult, err := VerifyPhaseGatesForCompletion(workspacePath, beadsID)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify phase gates: %v", err))
	} else if !phaseGateResult.Passed {
		result.Passed = false
		result.Errors = append(result.Errors, phaseGateResult.Errors...)
	}

	// Verify skill outputs from skill.yaml outputs.required section
	// This is the "skillc verify" integration - checks that required skill outputs exist
	skillOutputResult, err := VerifySkillOutputsForCompletion(workspacePath, projectDir)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify skill outputs: %v", err))
	} else if skillOutputResult != nil {
		// Only add results if skill had outputs.required defined
		if !skillOutputResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, skillOutputResult.Errors...)
		}
		result.Warnings = append(result.Warnings, skillOutputResult.Warnings...)
	}

	// Verify visual verification for web/ changes
	// This gates completion when web files are modified without visual verification evidence
	visualResult := VerifyVisualVerificationForCompletion(beadsID, workspacePath, projectDir)
	if visualResult != nil {
		if !visualResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, visualResult.Errors...)
		}
		result.Warnings = append(result.Warnings, visualResult.Warnings...)
	}

	// Verify test execution evidence for code changes
	// This gates completion when code files are modified without test execution evidence
	testEvidenceResult := VerifyTestEvidenceForCompletion(beadsID, workspacePath, projectDir)
	if testEvidenceResult != nil {
		if !testEvidenceResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, testEvidenceResult.Errors...)
		}
		result.Warnings = append(result.Warnings, testEvidenceResult.Warnings...)
	}

	// Verify git diff against SYNTHESIS claims
	// This detects false positives where agent claims to modify files but didn't
	gitDiffResult := VerifyGitDiffForCompletion(workspacePath, projectDir)
	if gitDiffResult != nil {
		if !gitDiffResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, gitDiffResult.Errors...)
		}
		result.Warnings = append(result.Warnings, gitDiffResult.Warnings...)
	}

	// Verify build for Go projects
	// This gates completion when Go files are modified but the project doesn't build
	buildResult := VerifyBuildForCompletion(workspacePath, projectDir)
	if buildResult != nil {
		if !buildResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, buildResult.Errors...)
		}
		result.Warnings = append(result.Warnings, buildResult.Warnings...)
	}

	return result, nil
}

// VerifyCompletionWithTier checks if an agent is ready for completion.
// The tier parameter specifies the spawn tier ("light" or "full").
// If tier is empty, it will be read from the workspace's .tier file.
// Light tier spawns skip the SYNTHESIS.md requirement.
// Returns a VerificationResult with any errors or warnings.
func VerifyCompletionWithTier(beadsID string, workspacePath string, tier string) (VerificationResult, error) {
	result := VerificationResult{
		Passed: true,
	}

	// Get phase status
	status, err := GetPhaseStatus(beadsID)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to get phase status: %v", err))
		return result, nil
	}

	result.Phase = status

	// Check if Phase: Complete was reported
	if !status.Found {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("agent has not reported any Phase status for %s", beadsID))
		return result, nil
	}

	if !strings.EqualFold(status.Phase, "Complete") {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("agent phase is '%s', not 'Complete' (beads: %s)", status.Phase, beadsID))
		return result, nil
	}

	// Determine tier if not provided
	if tier == "" && workspacePath != "" {
		tier = ReadTierFromWorkspace(workspacePath)
	}

	// Check for SYNTHESIS.md (only for full tier)
	if workspacePath != "" && tier != "light" {
		ok, err := VerifySynthesis(workspacePath)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify SYNTHESIS.md: %v", err))
		} else if !ok {
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("SYNTHESIS.md is missing or empty in workspace: %s", workspacePath))
		}
	}

	return result, nil
}

// ReadTierFromWorkspace reads the spawn tier from the workspace's .tier file.
// Returns "full" as the conservative default if the file doesn't exist.
func ReadTierFromWorkspace(workspacePath string) string {
	tierFile := filepath.Join(workspacePath, ".tier")
	data, err := os.ReadFile(tierFile)
	if err != nil {
		return "full" // Conservative default
	}
	tier := strings.TrimSpace(string(data))
	if tier == "" {
		return "full"
	}
	return tier
}
