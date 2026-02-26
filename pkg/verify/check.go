// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// Gate names for verification tracking.
// These constants are used in events to identify which verification gates failed.
const (
	GatePhaseComplete      = "phase_complete"       // Phase: Complete not reported
	GateSynthesis          = "synthesis"            // SYNTHESIS.md missing
	GateSessionHandoff     = "session_handoff"      // SESSION_HANDOFF.md missing (orchestrator)
	GateHandoffContent     = "handoff_content"      // SESSION_HANDOFF.md has empty/placeholder content
	GateConstraint         = "constraint"           // Constraint verification failed
	GatePhaseGate          = "phase_gate"           // Required phase gate not passed
	GateSkillOutput        = "skill_output"         // Required skill outputs missing
	GateVisualVerify       = "visual_verification"  // Visual verification required
	GateTestEvidence       = "test_evidence"        // Test execution evidence required
	GateGitDiff            = "git_diff"             // Git diff doesn't match claims
	GateAccretion          = "accretion"            // File size accretion detected
	GateBuild              = "build"                // Project build failed
	GateVet                = "vet"                  // Go vet failed
	GateDecisionPatchLimit = "decision_patch_limit" // Decision patch limit exceeded
	GateExplainBack        = "explain_back"         // Human explanation of what was built required
)

// VerificationResult represents the result of a completion verification.
type VerificationResult struct {
	Passed      bool     // Whether all checks passed
	Errors      []string // Errors that prevent completion
	Warnings    []string // Warnings that don't block completion
	Phase       PhaseStatus
	GatesFailed []string // Names of gates that failed (for event tracking)
	Skill       string   // Skill name extracted from workspace
}

// Tier constants for orchestrator spawns.
const (
	// TierOrchestrator is for orchestrator-type skills that produce SESSION_HANDOFF.md
	// instead of SYNTHESIS.md and skip beads-dependent checks.
	TierOrchestrator = "orchestrator"
)

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

// VerifySessionHandoff checks if SESSION_HANDOFF.md exists and is not empty.
// Used for orchestrator-type skills instead of SYNTHESIS.md.
func VerifySessionHandoff(workspacePath string) (bool, error) {
	if workspacePath == "" {
		return false, nil
	}
	handoffPath := filepath.Join(workspacePath, "SESSION_HANDOFF.md")
	info, err := os.Stat(handoffPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Size() > 0, nil
}

// HandoffContentValidation contains the results of validating handoff content.
type HandoffContentValidation struct {
	Valid         bool     // Whether the handoff has actual content
	Errors        []string // Specific validation failures
	TLDRFilled    bool     // Whether TLDR section has actual content
	OutcomeFilled bool     // Whether Outcome field has a valid value
}

// ValidateHandoffContent checks if SESSION_HANDOFF.md has actual content,
// not just the empty template. It validates:
// - TLDR section is filled (not placeholder text)
// - Outcome field is set to a valid value (success, partial, blocked, failed)
//
// This prevents orchestrators from completing with empty handoffs that waste
// context for the next session.
func ValidateHandoffContent(workspacePath string) (HandoffContentValidation, error) {
	result := HandoffContentValidation{
		Valid: true,
	}

	if workspacePath == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "workspace path is required")
		return result, nil
	}

	handoffPath := filepath.Join(workspacePath, "SESSION_HANDOFF.md")
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Valid = false
			result.Errors = append(result.Errors, "SESSION_HANDOFF.md not found")
			return result, nil
		}
		return result, err
	}

	contentStr := string(content)

	// Validate TLDR section has actual content
	result.TLDRFilled = validateTLDRContent(contentStr)
	if !result.TLDRFilled {
		result.Valid = false
		result.Errors = append(result.Errors, "TLDR section is empty or contains only placeholder text")
	}

	// Validate Outcome field has a valid value
	result.OutcomeFilled = validateOutcomeField(contentStr)
	if !result.OutcomeFilled {
		result.Valid = false
		result.Errors = append(result.Errors, "Outcome field is not filled (must be: success, partial, blocked, or failed)")
	}

	return result, nil
}

// validateTLDRContent checks if the TLDR section contains actual content.
// Returns false if:
// - TLDR section is missing
// - TLDR section contains only placeholder text like "[1-2 sentence summary..."
// - TLDR section contains only template instructions like "[Fill within first 5 tool calls..."
func validateTLDRContent(content string) bool {
	// Find the TLDR section
	tldrIdx := strings.Index(content, "## TLDR")
	if tldrIdx == -1 {
		return false
	}

	// Find the end of TLDR section (next ## header or ---)
	afterTLDR := content[tldrIdx+len("## TLDR"):]
	endIdx := strings.Index(afterTLDR, "\n---")
	if endIdx == -1 {
		endIdx = strings.Index(afterTLDR, "\n## ")
	}

	var tldrContent string
	if endIdx == -1 {
		tldrContent = afterTLDR
	} else {
		tldrContent = afterTLDR[:endIdx]
	}

	// Clean and check content
	tldrContent = strings.TrimSpace(tldrContent)

	// Check for placeholder patterns
	placeholderPatterns := []string{
		"[1-2 sentence summary",
		"[Fill within first 5 tool calls",
		"[What is this session trying to accomplish",
		"{session-goal}",
		"{describe what happened}",
	}

	for _, pattern := range placeholderPatterns {
		if strings.Contains(strings.ToLower(tldrContent), strings.ToLower(pattern)) {
			return false
		}
	}

	// Content should have meaningful length after removing whitespace
	// A real TLDR should have at least 20 characters
	return len(tldrContent) >= 20
}

// validateOutcomeField checks if the Outcome field has a valid value.
// Valid values are: success, partial, blocked, failed
// Returns false if:
// - Outcome field is missing
// - Outcome field contains placeholder like "{success | partial | blocked | failed}"
func validateOutcomeField(content string) bool {
	// Look for the Outcome line in the header section
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "**Outcome:**") {
			// Extract the value after "**Outcome:**"
			value := strings.TrimPrefix(line, "**Outcome:**")
			value = strings.TrimSpace(value)

			// Check for placeholder pattern
			if strings.Contains(value, "{") || strings.Contains(value, "|") {
				return false
			}

			// Check for valid outcome values
			validOutcomes := []string{"success", "partial", "blocked", "failed"}
			valueLower := strings.ToLower(value)
			for _, valid := range validOutcomes {
				if strings.Contains(valueLower, valid) {
					return true
				}
			}
			return false
		}
	}
	return false
}

// VerifyCompletion checks if an agent is ready for completion.
// Returns a VerificationResult with any errors or warnings.
// Uses VerifyCompletionWithTier with an empty tier (reads from workspace).
func VerifyCompletion(beadsID string, workspacePath string) (VerificationResult, error) {
	return VerifyCompletionWithTier(beadsID, workspacePath, "")
}

// isOrchestrator returns true if the tier is TierOrchestrator.
func isOrchestrator(tier string) bool {
	return tier == TierOrchestrator
}

// VerifyCompletionFull checks if an agent is ready for completion including skill constraints
// and phase gates. This extends VerifyCompletion with:
// 1. Constraint verification from SPAWN_CONTEXT.md (file patterns must match)
// 2. Phase gate verification (required phases must be reported via beads comments)
// 3. Skill output verification from skill.yaml outputs.required section
//
// For orchestrator tier, beads-dependent checks are skipped since orchestrators
// manage sessions rather than issues.
//
// The projectDir is used to verify that constraint patterns match actual files.
func VerifyCompletionFull(beadsID, workspacePath, projectDir, tier, serverURL string) (VerificationResult, error) {
	// Delegate to the cached version without pre-fetched comments
	return VerifyCompletionFullWithComments(beadsID, workspacePath, projectDir, tier, serverURL, nil)
}

// VerifyCompletionForReview is a lightweight verification for orch review command.
// It checks only the essential requirements (Phase: Complete, SYNTHESIS.md) and skips
// expensive checks (git diff, go build) that are deferred to orch complete.
// This enables O(1) verification per workspace instead of O(n) git/build commands.
func VerifyCompletionForReview(beadsID, workspacePath, tier, serverURL string, comments []Comment) (VerificationResult, error) {
	// Determine tier if not provided
	if tier == "" && workspacePath != "" {
		tier = ReadTierFromWorkspace(workspacePath)
	}

	// First run standard verification (uses comments for phase status)
	result, err := VerifyCompletionWithTierAndComments(beadsID, workspacePath, tier, comments)
	if err != nil {
		return result, err
	}

	// Verify backend deliverables (opencode transcript or tmux capture)
	if tier != TierOrchestrator && workspacePath != "" && result.Passed {
		backendResult := VerifyBackendDeliverables(workspacePath, beadsID, serverURL, "")
		if backendResult != nil {
			result.Warnings = append(result.Warnings, backendResult.Warnings...)
		}
	}

	return result, nil
}

// VerifyCompletionFullWithComments is like VerifyCompletionFull but accepts pre-fetched comments.
// This avoids O(n) beads API calls when verifying multiple completions in batch.
// If comments is nil, comments will be fetched from beads API.
func VerifyCompletionFullWithComments(beadsID, workspacePath, projectDir, tier, serverURL string, comments []Comment) (VerificationResult, error) {
	// Determine tier if not provided (needed for orchestrator check below)
	if tier == "" && workspacePath != "" {
		tier = ReadTierFromWorkspace(workspacePath)
	}

	// First run standard verification (uses comments for phase status)
	result, err := VerifyCompletionWithTierAndComments(beadsID, workspacePath, tier, comments)
	if err != nil {
		return result, err
	}

	// If standard verification failed, no need to check constraints
	if !result.Passed {
		return result, nil
	}

	// Verify backend deliverables (opencode transcript or tmux capture)
	if !isOrchestrator(tier) && workspacePath != "" {
		backendResult := VerifyBackendDeliverables(workspacePath, beadsID, serverURL, "")
		if backendResult != nil {
			// Merge warnings (backend checks currently don't block unless we decide otherwise)
			result.Warnings = append(result.Warnings, backendResult.Warnings...)
			if !backendResult.Passed {
				// For now, we don't block completion on backend check failures
				// to avoid breaking existing workflows, but we log the warnings.
				// If we want to block, we would set result.Passed = false here.
			}
		}
	}

	// Skip constraint verification if no workspace
	if workspacePath == "" {
		return result, nil
	}

	// Skip constraint verification if no project dir
	if projectDir == "" {
		return result, nil
	}

	// Check if this is an orchestrator tier spawn
	isOrch := isOrchestrator(tier)

	// Verify skill constraints from SPAWN_CONTEXT.md
	// Skip for orchestrator tier since they use ORCHESTRATOR_CONTEXT.md
	if !isOrch {
		constraintResult, err := VerifyConstraintsForCompletion(workspacePath, projectDir)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify constraints: %v", err))
			// Continue to phase gate verification even if constraints failed to parse
		} else {
			// Merge constraint results
			if !constraintResult.Passed {
				result.Passed = false
				result.Errors = append(result.Errors, constraintResult.Errors...)
				result.GatesFailed = append(result.GatesFailed, GateConstraint)
			}
			result.Warnings = append(result.Warnings, constraintResult.Warnings...)
		}
	}

	// Verify phase gates from SPAWN_CONTEXT.md
	// This checks that required phases were reported in beads comments
	// Skip for orchestrator tier (beads-dependent)
	if !isOrch {
		phaseGateResult, err := VerifyPhaseGatesForCompletionWithComments(workspacePath, beadsID, comments)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify phase gates: %v", err))
		} else if !phaseGateResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, phaseGateResult.Errors...)
			result.GatesFailed = append(result.GatesFailed, GatePhaseGate)
		}
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
			result.GatesFailed = append(result.GatesFailed, GateSkillOutput)
		}
		result.Warnings = append(result.Warnings, skillOutputResult.Warnings...)
	}

	// Verify visual verification for web/ changes
	// This gates completion when web files are modified without visual verification evidence
	// Skip for orchestrator tier (beads-dependent)
	if !isOrch {
		visualResult := VerifyVisualVerificationForCompletionWithComments(beadsID, workspacePath, projectDir, comments)
		if visualResult != nil {
			if !visualResult.Passed {
				result.Passed = false
				result.Errors = append(result.Errors, visualResult.Errors...)
				result.GatesFailed = append(result.GatesFailed, GateVisualVerify)
			}
			result.Warnings = append(result.Warnings, visualResult.Warnings...)
		}
	}

	// Verify test execution evidence for code changes
	// This gates completion when code files are modified without test execution evidence
	// Skip for orchestrator tier (beads-dependent)
	if !isOrch {
		testEvidenceResult := VerifyTestEvidenceForCompletionWithComments(beadsID, workspacePath, projectDir, comments)
		if testEvidenceResult != nil {
			if !testEvidenceResult.Passed {
				result.Passed = false
				result.Errors = append(result.Errors, testEvidenceResult.Errors...)
				result.GatesFailed = append(result.GatesFailed, GateTestEvidence)
			}
			result.Warnings = append(result.Warnings, testEvidenceResult.Warnings...)
		}
	}

	// Verify git diff against SYNTHESIS claims
	// This detects false positives where agent claims to modify files but didn't
	// Skip for orchestrator tier (they produce SESSION_HANDOFF.md, not SYNTHESIS.md)
	if !isOrch {
		gitDiffResult := VerifyGitDiffForCompletion(workspacePath, projectDir)
		if gitDiffResult != nil {
			if !gitDiffResult.Passed {
				result.Passed = false
				result.Errors = append(result.Errors, gitDiffResult.Errors...)
				result.GatesFailed = append(result.GatesFailed, GateGitDiff)
			}
			result.Warnings = append(result.Warnings, gitDiffResult.Warnings...)
		}
	}

	// Verify accretion (file size growth) for non-orchestrator tiers
	// This gates completion when agents add >50 lines to already-large files (>800 or >1500 lines)
	// Skip for orchestrator tier (orchestrators don't write code)
	if !isOrch {
		accretionResult := VerifyAccretionForCompletion(workspacePath, projectDir)
		if accretionResult != nil {
			if !accretionResult.Passed {
				result.Passed = false
				result.Errors = append(result.Errors, accretionResult.Errors...)
				result.GatesFailed = append(result.GatesFailed, GateAccretion)
			}
			result.Warnings = append(result.Warnings, accretionResult.Warnings...)
		}
	}

	// Verify build and vet for Go projects
	// This gates completion when Go files are modified but the project doesn't build or vet
	// Note: This check is still relevant for orchestrators if they make code changes
	buildResult := VerifyBuildForCompletion(workspacePath, projectDir)
	if buildResult != nil {
		if !buildResult.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, buildResult.Errors...)
			if !buildResult.BuildPassed {
				result.GatesFailed = append(result.GatesFailed, GateBuild)
			}
			if !buildResult.VetPassed {
				result.GatesFailed = append(result.GatesFailed, GateVet)
			}
		}
		result.Warnings = append(result.Warnings, buildResult.Warnings...)
	}

	// Verify decision patch count (prevent launchd-style patch accumulation)
	// After N patches to same decision, require architect review before more patches
	// Skip for orchestrator tier (they don't produce investigation patches)
	if !isOrch {
		decisionPatchResult := VerifyDecisionPatchCount(workspacePath, projectDir)
		if decisionPatchResult != nil {
			if !decisionPatchResult.Passed {
				result.Passed = false
				result.Errors = append(result.Errors, decisionPatchResult.Errors...)
				result.GatesFailed = append(result.GatesFailed, GateDecisionPatchLimit)
			}
			result.Warnings = append(result.Warnings, decisionPatchResult.Warnings...)
		}
	}

	return result, nil
}

// VerifyCompletionWithTier checks if an agent is ready for completion.
// The tier parameter specifies the spawn tier ("light", "full", or "orchestrator").
// If tier is empty, it will be read from the workspace's .tier file.
// Light tier spawns skip the SYNTHESIS.md requirement.
// Orchestrator tier spawns:
//   - Skip beads-dependent phase checks (orchestrators manage sessions, not issues)
//   - Check SESSION_HANDOFF.md instead of SYNTHESIS.md
//
// Returns a VerificationResult with any errors or warnings.
func VerifyCompletionWithTier(beadsID string, workspacePath string, tier string) (VerificationResult, error) {
	return VerifyCompletionWithTierAndComments(beadsID, workspacePath, tier, nil)
}

// VerifyCompletionWithTierAndComments is like VerifyCompletionWithTier but accepts pre-fetched comments.
// If comments is nil, comments will be fetched from beads API.
func VerifyCompletionWithTierAndComments(beadsID string, workspacePath string, tier string, comments []Comment) (VerificationResult, error) {
	result := VerificationResult{
		Passed: true,
	}

	// Extract skill name for tracking
	if workspacePath != "" {
		result.Skill, _ = ExtractSkillNameFromSpawnContext(workspacePath)
	}

	// Determine tier if not provided
	if tier == "" && workspacePath != "" {
		tier = ReadTierFromWorkspace(workspacePath)
	}

	// Orchestrator tier: skip beads-dependent checks, verify SESSION_HANDOFF.md instead
	if tier == TierOrchestrator {
		return VerifyOrchestratorCompletion(workspacePath), nil
	}

	// Standard worker verification: beads-based phase tracking
	// Get phase status (using pre-fetched comments if available)
	var status PhaseStatus
	var err error
	if comments != nil {
		status = ParsePhaseFromComments(comments)
	} else {
		status, err = GetPhaseStatus(beadsID)
		if err != nil {
			result.Passed = false
			result.Errors = append(result.Errors, fmt.Sprintf("failed to get phase status: %v", err))
			result.GatesFailed = append(result.GatesFailed, GatePhaseComplete)
			return result, nil
		}
	}

	result.Phase = status

	// Check if Phase: Complete was reported
	if !status.Found {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("agent has not reported any Phase status for %s", beadsID))
		result.GatesFailed = append(result.GatesFailed, GatePhaseComplete)
		return result, nil
	}

	if !strings.EqualFold(status.Phase, "Complete") {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("agent phase is '%s', not 'Complete' (beads: %s)", status.Phase, beadsID))
		result.GatesFailed = append(result.GatesFailed, GatePhaseComplete)
		return result, nil
	}

	// Check for SYNTHESIS.md (only for full tier AND code-producing skills)
	// Knowledge-producing skills (investigation, architect, research, etc.) produce
	// artifacts as their deliverable - SYNTHESIS.md is redundant
	if workspacePath != "" && tier != "light" {
		// Check if skill requires synthesis
		skillName := result.Skill
		if skillName == "" {
			skillName, _ = ExtractSkillNameFromSpawnContext(workspacePath)
		}

		// Auto-skip for knowledge-producing skills - their artifacts ARE the deliverable
		if IsKnowledgeProducingSkill(skillName) {
			logAutoSkip(beadsID, workspacePath, GateSynthesis,
				skillName+" skill produces knowledge artifacts, not SYNTHESIS.md",
				skillName)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("synthesis gate auto-skipped for %s skill", strings.ToLower(skillName)))
		} else {
			ok, err := VerifySynthesis(workspacePath)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify SYNTHESIS.md: %v", err))
			} else if !ok {
				result.Passed = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("SYNTHESIS.md is missing or empty in workspace: %s", workspacePath))
				result.GatesFailed = append(result.GatesFailed, GateSynthesis)
			}
		}
	}

	return result, nil
}

// VerifyOrchestratorCompletion checks if an orchestrator session is ready for completion.
// Orchestrators have different verification requirements than workers:
//   - No beads-dependent phase checks (orchestrators manage sessions, not issues)
//   - SESSION_HANDOFF.md instead of SYNTHESIS.md
//   - Session end verification instead of Phase: Complete
//   - Content validation (TLDR and Outcome must be filled, not placeholders)
func VerifyOrchestratorCompletion(workspacePath string) VerificationResult {
	result := VerificationResult{
		Passed: true,
	}

	// Extract skill name for tracking
	if workspacePath != "" {
		result.Skill, _ = ExtractSkillNameFromSpawnContext(workspacePath)
	}

	if workspacePath == "" {
		result.Passed = false
		result.Errors = append(result.Errors, "workspace path is required for orchestrator verification")
		result.GatesFailed = append(result.GatesFailed, GateSessionHandoff)
		return result
	}

	// Check for SESSION_HANDOFF.md
	ok, err := VerifySessionHandoff(workspacePath)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify SESSION_HANDOFF.md: %v", err))
	} else if !ok {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("SESSION_HANDOFF.md is missing or empty in workspace: %s", workspacePath))
		result.GatesFailed = append(result.GatesFailed, GateSessionHandoff)
	}

	// Verify session ended properly by checking for "Session Ended" marker in SESSION_HANDOFF.md
	if ok {
		sessionEnded, err := verifySessionEndedProperly(workspacePath)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify session end: %v", err))
		} else if !sessionEnded {
			result.Passed = false
			result.Errors = append(result.Errors,
				"SESSION_HANDOFF.md exists but session end not properly recorded")
			result.GatesFailed = append(result.GatesFailed, GateSessionHandoff)
		}
	}

	// Validate handoff content (TLDR and Outcome must be filled, not placeholders)
	// This gate ensures orchestrators don't complete with empty template handoffs
	if ok {
		contentValidation, err := ValidateHandoffContent(workspacePath)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to validate handoff content: %v", err))
		} else if !contentValidation.Valid {
			result.Passed = false
			for _, e := range contentValidation.Errors {
				result.Errors = append(result.Errors, e)
			}
			result.GatesFailed = append(result.GatesFailed, GateHandoffContent)
		}
	}

	return result
}

// verifySessionEndedProperly checks if SESSION_HANDOFF.md contains proper session end markers.
// Returns true if the session appears to have ended properly.
func verifySessionEndedProperly(workspacePath string) (bool, error) {
	handoffPath := filepath.Join(workspacePath, "SESSION_HANDOFF.md")
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		return false, err
	}

	contentStr := string(content)

	// Check for common session end markers
	// These are patterns that indicate the session was properly concluded
	endMarkers := []string{
		"## Session Summary",    // Summary section indicates wrap-up
		"## Handoff",            // Handoff section indicates transition
		"## Next Steps",         // Next steps indicates planned continuation
		"**Session Ended:**",    // Explicit end marker
		"**Status:** Complete",  // Status marker
		"**Status:** Completed", // Alternative status marker
	}

	for _, marker := range endMarkers {
		if strings.Contains(contentStr, marker) {
			return true, nil
		}
	}

	// If no explicit markers, check for minimum content length
	// A proper handoff should have substantial content
	if len(strings.TrimSpace(contentStr)) > 100 {
		return true, nil
	}

	return false, nil
}

// ReadTierFromWorkspace reads the spawn tier from the workspace.
// Reads AGENT_MANIFEST.json first, falls back to .tier dotfile.
// Returns "full" as the conservative default if neither exists.
func ReadTierFromWorkspace(workspacePath string) string {
	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	if manifest.Tier != "" {
		return manifest.Tier
	}
	return "full" // Conservative default
}
