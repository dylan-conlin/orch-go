// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/activity"
)

// StateDBPhaseChecker is an injectable function that checks state.db for phase status.
// This avoids a circular dependency between verify and state packages.
// Set by cmd/orch or other callers that have access to pkg/state.
// Returns (phase, summary, found, error).
var StateDBPhaseChecker func(beadsID string) (phase string, summary string, found bool, err error)

// Gate names for verification tracking.
// These constants are used in events to identify which verification gates failed.
const (
	GatePhaseComplete      = "phase_complete"       // Phase: Complete not reported
	GateSynthesis          = "synthesis"            // SYNTHESIS.md missing
	GateHandoffContent     = "handoff_content"      // SYNTHESIS.md has empty/placeholder content
	GateConstraint         = "constraint"           // Constraint verification failed
	GatePhaseGate          = "phase_gate"           // Required phase gate not passed
	GateSkillOutput        = "skill_output"         // Required skill outputs missing
	GateVisualVerify       = "visual_verification"  // Visual verification required
	GateTestEvidence       = "test_evidence"        // Test execution evidence required
	GateModelConnection    = "model_connection"     // Model probe/candidate evidence required
	GateVerificationSpec   = "verification_spec"    // VERIFICATION_SPEC executable checks failed
	GateGitDiff            = "git_diff"             // Git diff doesn't match claims
	GateBuild              = "build"                // Project build failed
	GateDecisionPatchLimit = "decision_patch_limit" // Decision patch limit exceeded
	GateDashboardHealth    = "dashboard_health"     // Dashboard API health check failed
	GateAgentRunning       = "agent_running"        // Agent appears still running
	GateCommitEvidence     = "commit_evidence"      // No commits on agent branch
)

// Two-Tier Verification System
//
// The verification system uses a two-tier architecture to balance quality with velocity:
//
// **Tier 1 (Core 5)**: The essential gates that prevent ghost completions and
// broken handoffs. Always run regardless of mode.
//
// **Tier 2 (Quality 10)**: Process compliance and secondary quality checks.
// Skipped in batch mode for rapid iteration.
//
// This separation allows:
// - Batch mode (--batch): Fast completion for trusted agents, running only core checks
// - Careful mode (default): Full quality assurance with both tiers
// - Selective skipping: Individual gates can be bypassed with --skip-* flags

// GateResult represents the result of a single verification gate.
type GateResult struct {
	Gate    string // Gate constant name (e.g., GateBuild)
	Passed  bool
	Skipped bool   // Whether this gate was skipped (batch mode or skip memory)
	Error   string // Error message if failed (empty if passed)
}

// Tier 1 (Core Gates): The 5 essential gates that always run, even in batch mode.
//
// Philosophy: These gates prevent the two most costly failure modes:
// 1. Ghost completions (issues close with no actual work landed)
// 2. Broken handoffs (next session wastes context re-discovering state)
//
// Characteristics:
// - Always executed regardless of mode (batch/careful)
// - Block completion unconditionally when failed
// - Each gate has a clear, non-overlapping failure mode it prevents
//
// The Core 5:
// - phase_complete: Agent self-reported completion (prevents premature close)
// - commit_evidence: Commits exist on branch (prevents ghost completions)
// - synthesis: SYNTHESIS.md exists (prevents broken handoffs)
// - test_evidence: Tests were run (prevents shipping untested code)
// - git_diff: Diff matches SYNTHESIS claims (prevents fiction in handoffs)
var CoreGates = map[string]bool{
	GatePhaseComplete:  true,
	GateCommitEvidence: true,
	GateSynthesis:      true,
	GateTestEvidence:   true,
	GateGitDiff:        true,
}

// Tier 2 (Quality Gates): Process compliance checks skipped in batch mode (--batch).
//
// Philosophy: These gates enforce process standards and catch issues that don't
// make the work fundamentally broken but improve quality. Skipping them in batch
// mode enables rapid iteration for trusted agents.
//
// Demoted from Core (Phase 2 simplification):
// - build, model_connection, verification_spec, visual_verify
//   These are valuable but not essential — a passing build doesn't guarantee
//   correctness, and visual/model checks are skill-specific.
var QualityGates = map[string]bool{
	GateBuild:              true,
	GateModelConnection:    true,
	GateVerificationSpec:   true,
	GateVisualVerify:       true,
	GateConstraint:         true,
	GatePhaseGate:          true,
	GateSkillOutput:        true,
	GateDecisionPatchLimit: true,
	GateDashboardHealth:    true,
	GateHandoffContent:     true,
}

// IsCoreGate returns true if the gate is a Tier 1 core gate.
func IsCoreGate(gate string) bool {
	return CoreGates[gate]
}

// IsQualityGate returns true if the gate is a Tier 2 quality gate.
func IsQualityGate(gate string) bool {
	return QualityGates[gate]
}

// VerificationResult represents the result of a completion verification.
type VerificationResult struct {
	Passed      bool     // Whether all checks passed
	Errors      []string // Errors that prevent completion
	Warnings    []string // Warnings that don't block completion
	Phase       PhaseStatus
	GatesFailed []string     // Names of gates that failed (for event tracking)
	GateResults []GateResult // Per-gate pass/fail results (ordered by check sequence)
	Skill       string       // Skill name extracted from workspace
}

// Tier constants for orchestrator spawns.
const (
	// TierOrchestrator is for orchestrator-type skills that produce SYNTHESIS.md
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

// HandoffContentValidation contains the results of validating handoff content.
type HandoffContentValidation struct {
	Valid         bool     // Whether the handoff has actual content
	Errors        []string // Specific validation failures
	TLDRFilled    bool     // Whether TLDR section has actual content
	OutcomeFilled bool     // Whether Outcome field has a valid value
}

// ValidateHandoffContent checks if SYNTHESIS.md has actual content,
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

	handoffPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Valid = false
			result.Errors = append(result.Errors, "SYNTHESIS.md not found")
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

// isOrchestrator returns true if the tier is TierOrchestrator.
func isOrchestrator(tier string) bool {
	return tier == TierOrchestrator
}

// VerifyCompletionFull checks if an agent is ready for completion including skill constraints
// and phase gates. It verifies:
// 1. Phase: Complete status (with ACTIVITY.json and state.db fallbacks)
// 2. SYNTHESIS.md exists and is non-empty
// 3. Backend deliverables (opencode transcript or tmux capture)
// 4. Constraint verification from SPAWN_CONTEXT.md (file patterns must match)
// 5. Phase gate verification (required phases must be reported via beads comments)
// 6. Skill output verification from skill.yaml outputs.required section
//
// For orchestrator tier, beads-dependent checks are skipped since orchestrators
// manage sessions rather than issues.
//
// If comments is nil, comments will be fetched from beads API.
func VerifyCompletionFull(beadsID, workspacePath, projectDir, tier, serverURL string, comments []Comment) (VerificationResult, error) {
	// Determine tier if not provided (needed for orchestrator check below)
	if tier == "" && workspacePath != "" {
		tier = ReadTierFromWorkspace(workspacePath)
	}

	// Run phase and synthesis verification
	// Pass projectDir for skill resolution fallback: when workspacePath is the
	// project root (headless spawns), spawn artifacts live in .orch/workspace/{name}/
	result, err := verifyPhaseAndSynthesis(beadsID, workspacePath, projectDir, tier, comments)
	if err != nil {
		return result, err
	}

	// If phase/synthesis verification failed, no need to check constraints
	if !result.Passed {
		return result, nil
	}

	isOrch := isOrchestrator(tier)

	// Verify backend deliverables (opencode transcript or tmux capture)
	if !isOrch && workspacePath != "" {
		mergeBackendResult(&result, VerifyBackendDeliverables(workspacePath, beadsID, serverURL, ""))
	}

	// Skip constraint/gate verification if no workspace or project dir
	if workspacePath == "" || projectDir == "" {
		return result, nil
	}

	// Run worker-specific gates (skip for orchestrator tier)
	if !isOrch {
		verifyWorkerGates(&result, beadsID, workspacePath, projectDir, serverURL, comments, result.Skill)
	}

	// Run gates that apply to all tiers
	verifyCommonGates(&result, workspacePath, projectDir)

	return result, nil
}

// VerifyCompletionForReview is a lightweight verification for orch review command.
// It checks only the essential requirements (Phase: Complete, SYNTHESIS.md) and skips
// expensive checks (git diff, go build) that are deferred to orch complete.
// This enables O(1) verification per workspace instead of O(n) git/build commands.
func VerifyCompletionForReview(beadsID, workspacePath, tier, serverURL string, comments []Comment) (VerificationResult, error) {
	// Determine tier if not provided (needed for orchestrator check below)
	if tier == "" && workspacePath != "" {
		tier = ReadTierFromWorkspace(workspacePath)
	}

	// Run phase and synthesis verification directly
	// VerifyCompletionForReview doesn't have projectDir context, pass workspacePath
	// as fallback (sufficient when called with canonical workspace path)
	result, err := verifyPhaseAndSynthesis(beadsID, workspacePath, workspacePath, tier, comments)
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

// gateCheckResult represents the outcome of a single verification gate check.
// This provides a uniform interface for merging results from the various
// verification functions, each of which has its own result type.
type gateCheckResult struct {
	gate     string   // Gate constant (e.g., GateBuild)
	passed   bool     // Whether the gate passed
	errors   []string // Error messages (blocking)
	warnings []string // Warning messages (non-blocking)
}

// mergeGateResult merges a single gate check result into the overall VerificationResult.
// This eliminates the repetitive 10-line merge pattern used by each gate check.
func mergeGateResult(result *VerificationResult, gr gateCheckResult) {
	if !gr.passed {
		result.Passed = false
		result.Errors = append(result.Errors, gr.errors...)
		result.GatesFailed = append(result.GatesFailed, gr.gate)
		result.GateResults = append(result.GateResults, GateResult{Gate: gr.gate, Passed: false, Error: joinErrors(gr.errors)})
	} else {
		result.GateResults = append(result.GateResults, GateResult{Gate: gr.gate, Passed: true})
	}
	result.Warnings = append(result.Warnings, gr.warnings...)
}

// mergeBackendResult merges backend deliverable warnings into the overall result.
// Backend checks currently don't block completion to avoid breaking existing workflows.
func mergeBackendResult(result *VerificationResult, backendResult *BackendResult) {
	if backendResult != nil {
		result.Warnings = append(result.Warnings, backendResult.Warnings...)
	}
}

// verifyWorkerGates runs verification gates specific to worker (non-orchestrator) spawns.
// These gates are skipped for orchestrator tier since they depend on beads or are
// worker-specific (constraints, phase gates, visual verification, test evidence, etc.).
func verifyWorkerGates(result *VerificationResult, beadsID, workspacePath, projectDir, serverURL string, comments []Comment, skillName string) {
	// Verify skill constraints from SPAWN_CONTEXT.md
	checkConstraints(result, workspacePath, projectDir)

	// Verify phase gates (required phases reported in beads comments)
	checkPhaseGates(result, workspacePath, beadsID, comments)

	// Verify visual verification for web/ changes
	checkVisualVerification(result, beadsID, workspacePath, projectDir, comments)

	if IsSkillRequiringModelConnection(skillName) {
		checkModelConnection(result, skillName, workspacePath, projectDir)
	} else {
		// Verify test execution evidence for code changes
		// Pass pre-extracted skillName to avoid re-extraction with wrong path
		checkTestEvidence(result, beadsID, workspacePath, projectDir, skillName, comments)
	}

	// Verify git diff against SYNTHESIS claims
	checkGitDiff(result, workspacePath, projectDir)

	// Verify dashboard health for dashboard-touching changes
	verifyDashboardHealthGate(result, workspacePath, projectDir, serverURL)

	// Verify decision patch count (prevent patch accumulation)
	checkDecisionPatchCount(result, workspacePath, projectDir)
}

// checkModelConnection verifies model probe/candidate evidence for knowledge skills.
func checkModelConnection(result *VerificationResult, skillName, workspacePath, projectDir string) {
	modelResult := VerifyModelConnectionForCompletion(skillName, workspacePath, projectDir)
	if modelResult == nil {
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:     GateModelConnection,
		passed:   modelResult.Passed,
		errors:   modelResult.Errors,
		warnings: modelResult.Warnings,
	})
}

// verifyCommonGates runs verification gates that apply to all tiers (including orchestrator).
func verifyCommonGates(result *VerificationResult, workspacePath, projectDir string) {
	// Verify skill outputs from skill.yaml outputs.required section
	checkSkillOutputs(result, workspacePath, projectDir)

	// Verify build for Go projects (relevant for all tiers if code changes)
	checkBuild(result, workspacePath, projectDir)
}

// checkConstraints verifies skill constraints from SPAWN_CONTEXT.md.
func checkConstraints(result *VerificationResult, workspacePath, projectDir string) {
	constraintResult, err := VerifyConstraintsForCompletion(workspacePath, projectDir)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify constraints: %v", err))
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:     GateConstraint,
		passed:   constraintResult.Passed,
		errors:   constraintResult.Errors,
		warnings: constraintResult.Warnings,
	})
}

// checkPhaseGates verifies that required phases were reported in beads comments.
func checkPhaseGates(result *VerificationResult, workspacePath, beadsID string, comments []Comment) {
	phaseGateResult, err := VerifyPhaseGatesForCompletionWithComments(workspacePath, beadsID, comments)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify phase gates: %v", err))
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:   GatePhaseGate,
		passed: phaseGateResult.Passed,
		errors: phaseGateResult.Errors,
	})
}

// checkVisualVerification verifies visual verification evidence for web/ changes.
func checkVisualVerification(result *VerificationResult, beadsID, workspacePath, projectDir string, comments []Comment) {
	visualResult := VerifyVisualVerificationForCompletionWithComments(beadsID, workspacePath, projectDir, comments)
	if visualResult == nil {
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:     GateVisualVerify,
		passed:   visualResult.Passed,
		errors:   visualResult.Errors,
		warnings: visualResult.Warnings,
	})
}

// checkTestEvidence verifies test execution evidence for code changes.
func checkTestEvidence(result *VerificationResult, beadsID, workspacePath, projectDir, skillName string, comments []Comment) {
	testResult := VerifyTestEvidenceForCompletionWithSkill(beadsID, workspacePath, projectDir, skillName, comments)
	if testResult == nil {
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:     GateTestEvidence,
		passed:   testResult.Passed,
		errors:   testResult.Errors,
		warnings: testResult.Warnings,
	})
}

// checkGitDiff verifies git diff against SYNTHESIS claims.
func checkGitDiff(result *VerificationResult, workspacePath, projectDir string) {
	gitDiffResult := VerifyGitDiffForCompletion(workspacePath, projectDir)
	if gitDiffResult == nil {
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:     GateGitDiff,
		passed:   gitDiffResult.Passed,
		errors:   gitDiffResult.Errors,
		warnings: gitDiffResult.Warnings,
	})
}

// verifyDashboardHealthGate verifies dashboard API health for dashboard-touching changes.
func verifyDashboardHealthGate(result *VerificationResult, workspacePath, projectDir, serverURL string) {
	dashboardResult := VerifyDashboardHealth(workspacePath, projectDir, serverURL)
	if dashboardResult == nil {
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:     GateDashboardHealth,
		passed:   dashboardResult.Passed,
		errors:   dashboardResult.Errors,
		warnings: dashboardResult.Warnings,
	})
}

// checkDecisionPatchCount verifies decision patch count limits.
func checkDecisionPatchCount(result *VerificationResult, workspacePath, projectDir string) {
	patchResult := VerifyDecisionPatchCount(workspacePath, projectDir)
	if patchResult == nil {
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:     GateDecisionPatchLimit,
		passed:   patchResult.Passed,
		errors:   patchResult.Errors,
		warnings: patchResult.Warnings,
	})
}

// checkSkillOutputs verifies skill outputs from skill.yaml outputs.required section.
func checkSkillOutputs(result *VerificationResult, workspacePath, projectDir string) {
	skillOutputResult, err := VerifySkillOutputsForCompletion(workspacePath, projectDir)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify skill outputs: %v", err))
		return
	}
	if skillOutputResult == nil {
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:     GateSkillOutput,
		passed:   skillOutputResult.Passed,
		errors:   skillOutputResult.Errors,
		warnings: skillOutputResult.Warnings,
	})
}

// checkBuild verifies the Go project builds successfully.
func checkBuild(result *VerificationResult, workspacePath, projectDir string) {
	buildResult := VerifyBuildForCompletion(workspacePath, projectDir)
	if buildResult == nil {
		return
	}
	mergeGateResult(result, gateCheckResult{
		gate:     GateBuild,
		passed:   buildResult.Passed,
		errors:   buildResult.Errors,
		warnings: buildResult.Warnings,
	})
}

// verifyPhaseAndSynthesis checks the core completion signals: phase status and synthesis.
// It handles tier resolution, orchestrator dispatch, phase_complete checking
// (with ACTIVITY.json and state.db fallbacks), and synthesis checking.
func verifyPhaseAndSynthesis(beadsID, workspacePath, projectDir, tier string, comments []Comment) (VerificationResult, error) {
	result := VerificationResult{
		Passed: true,
	}

	// Extract skill name for tracking.
	// Uses ResolveSkillName which falls back to searching .orch/workspace/ by beadsID
	// when workspacePath (artifacts dir) doesn't contain spawn artifacts (headless spawns).
	if workspacePath != "" {
		result.Skill = ResolveSkillName(workspacePath, projectDir, beadsID)
	}

	// Determine tier if not provided
	if tier == "" && workspacePath != "" {
		tier = ReadTierFromWorkspace(workspacePath)
	}

	// Orchestrator tier: skip beads-dependent checks, verify SYNTHESIS.md instead
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
	phaseComplete := status.Found && strings.EqualFold(status.Phase, "Complete")
	// Fallback: Check ACTIVITY.json for Phase: Complete attempts
	// This handles the case where bd comment reports success but the comment
	// fails to persist (a known beads bug - see orch-go-21112).
	if !phaseComplete && workspacePath != "" {
		attempt := activity.DetectPhaseCompleteAttempt(workspacePath)
		if attempt.Found && attempt.ReportedSuccess {
			// Agent attempted to report Phase: Complete and bd said it succeeded,
			// but the comment didn't persist. This is a beads bug, not agent fault.
			// Treat as Phase: Complete with a warning.
			result.Phase = PhaseStatus{
				Phase:   "Complete",
				Summary: "(recovered from ACTIVITY.json - bd comment failed to persist)",
				Found:   true,
			}
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Phase: Complete recovered from ACTIVITY.json - bd comment reported success but failed to persist to beads. Agent attempted at timestamp %d.", attempt.Timestamp))
			phaseComplete = true
		}
	}

	// Fallback: Check state.db where orch phase writes directly.
	// This handles the case where bd comment fails from worktree context
	// (a known issue - see orch-go-ugla4) but orch phase succeeded writing
	// to SQLite. The orch phase command writes to state.db (~1ms) and then
	// attempts bd comment as a secondary audit trail.
	if !phaseComplete && beadsID != "" && StateDBPhaseChecker != nil {
		dbPhase, dbSummary, dbFound, dbErr := StateDBPhaseChecker(beadsID)
		if dbErr == nil && dbFound && strings.EqualFold(dbPhase, "Complete") {
			result.Phase = PhaseStatus{
				Phase:   "Complete",
				Summary: fmt.Sprintf("(recovered from state.db - bd comment may have failed) %s", dbSummary),
				Found:   true,
			}
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Phase: Complete recovered from state.db (orch phase wrote to SQLite but bd comment may not have persisted for %s)", beadsID))
			phaseComplete = true
		}
	}

	if !phaseComplete {
		if !status.Found {
			errMsg := fmt.Sprintf("agent has not reported any Phase status for %s — use --skip-phase-complete --skip-reason '<reason>' to bypass", beadsID)
			result.Passed = false
			result.Errors = append(result.Errors, errMsg)
			result.GatesFailed = append(result.GatesFailed, GatePhaseComplete)
			result.GateResults = append(result.GateResults, GateResult{Gate: GatePhaseComplete, Passed: false, Error: errMsg})
			return result, nil
		}

		errMsg := fmt.Sprintf("agent phase is '%s', not 'Complete' (beads: %s) — use --skip-phase-complete --skip-reason '<reason>' to bypass", status.Phase, beadsID)
		result.Passed = false
		result.Errors = append(result.Errors, errMsg)
		result.GatesFailed = append(result.GatesFailed, GatePhaseComplete)
		result.GateResults = append(result.GateResults, GateResult{Gate: GatePhaseComplete, Passed: false, Error: errMsg})
		return result, nil
	}
	result.GateResults = append(result.GateResults, GateResult{Gate: GatePhaseComplete, Passed: true, Skipped: false})

	// Check for SYNTHESIS.md (only for full tier)
	if workspacePath != "" && tier != "light" {
		ok, err := VerifySynthesis(workspacePath)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify SYNTHESIS.md: %v", err))
		} else if !ok {
			errMsg := fmt.Sprintf("SYNTHESIS.md is missing or empty in workspace: %s", workspacePath)
			result.Passed = false
			result.Errors = append(result.Errors, errMsg)
			result.GatesFailed = append(result.GatesFailed, GateSynthesis)
			result.GateResults = append(result.GateResults, GateResult{Gate: GateSynthesis, Passed: false, Error: errMsg})
		} else {
			result.GateResults = append(result.GateResults, GateResult{Gate: GateSynthesis, Passed: true})
		}
	}

	return result, nil
}

// VerifyOrchestratorCompletion checks if an orchestrator session is ready for completion.
// Orchestrators have different verification requirements than workers:
//   - No beads-dependent phase checks (orchestrators manage sessions, not issues)
//   - SYNTHESIS.md instead of SYNTHESIS.md
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
		errMsg := "workspace path is required for orchestrator verification"
		result.Passed = false
		result.Errors = append(result.Errors, errMsg)
		result.GatesFailed = append(result.GatesFailed, GateSynthesis)
		result.GateResults = append(result.GateResults, GateResult{Gate: GateSynthesis, Passed: false, Error: errMsg})
		return result
	}

	// Check for SYNTHESIS.md
	ok, err := VerifySynthesis(workspacePath)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify SYNTHESIS.md: %v", err))
	} else if !ok {
		errMsg := fmt.Sprintf("SYNTHESIS.md is missing or empty in workspace: %s", workspacePath)
		result.Passed = false
		result.Errors = append(result.Errors, errMsg)
		result.GatesFailed = append(result.GatesFailed, GateSynthesis)
		result.GateResults = append(result.GateResults, GateResult{Gate: GateSynthesis, Passed: false, Error: errMsg})
	}

	// Verify session ended properly by checking for "Session Ended" marker in SYNTHESIS.md
	if ok {
		sessionEnded, err := verifySessionEndedProperly(workspacePath)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify session end: %v", err))
		} else if !sessionEnded {
			result.Passed = false
			errMsg := "SYNTHESIS.md exists but session end not properly recorded"
			result.Errors = append(result.Errors, errMsg)
			result.GatesFailed = append(result.GatesFailed, GateSynthesis)
			// Update the gate result if it was previously marked as passed
			result.GateResults = append(result.GateResults, GateResult{Gate: GateSynthesis, Passed: false, Error: errMsg})
		}
	}

	// If synthesis checks passed and no synthesis gate result yet, mark it passed
	synthesisFailed := false
	for _, gr := range result.GateResults {
		if gr.Gate == GateSynthesis && !gr.Passed {
			synthesisFailed = true
			break
		}
	}
	if !synthesisFailed && ok {
		result.GateResults = append(result.GateResults, GateResult{Gate: GateSynthesis, Passed: true})
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
			result.GateResults = append(result.GateResults, GateResult{Gate: GateHandoffContent, Passed: false, Error: joinErrors(contentValidation.Errors)})
		} else {
			result.GateResults = append(result.GateResults, GateResult{Gate: GateHandoffContent, Passed: true})
		}
	}

	return result
}

// verifySessionEndedProperly checks if SYNTHESIS.md contains proper session end markers.
// Returns true if the session appears to have ended properly.
func verifySessionEndedProperly(workspacePath string) (bool, error) {
	handoffPath := filepath.Join(workspacePath, "SYNTHESIS.md")
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

// joinErrors joins multiple error strings into a single semicolon-separated string.
func joinErrors(errors []string) string {
	return strings.Join(errors, "; ")
}

// GateSkipFlag returns the CLI --skip-* flag name for a given gate constant.
func GateSkipFlag(gate string) string {
	switch gate {
	case GatePhaseComplete:
		return "--skip-phase-complete"
	case GateSynthesis:
		return "--skip-synthesis"
	case GateHandoffContent:
		return "--skip-handoff-content"
	case GateConstraint:
		return "--skip-constraint"
	case GatePhaseGate:
		return "--skip-phase-gate"
	case GateSkillOutput:
		return "--skip-skill-output"
	case GateVisualVerify:
		return "--skip-visual"
	case GateTestEvidence:
		return "--skip-test-evidence"
	case GateModelConnection:
		return "--skip-model-connection"
	case GateVerificationSpec:
		return "--skip-verification-spec"
	case GateGitDiff:
		return "--skip-git-diff"
	case GateBuild:
		return "--skip-build"
	case GateDecisionPatchLimit:
		return "--skip-decision-patch"
	case GateDashboardHealth:
		return "--skip-dashboard-health"
	case GateAgentRunning:
		return "--skip-agent-running"
	case GateCommitEvidence:
		return "--skip-commit-evidence"
	default:
		return "--skip-" + strings.ReplaceAll(gate, "_", "-")
	}
}

// GateDisplayName returns a human-readable uppercase name for a gate constant.
func GateDisplayName(gate string) string {
	return strings.ToUpper(gate)
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
