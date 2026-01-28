// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"os/exec"
	"strings"
)

// EscalationLevel represents the level of human attention required for completion.
// Higher levels require more human involvement.
type EscalationLevel int

const (
	// EscalationNone - Auto-complete silently. No human attention needed.
	// Used for clean completions with no recommendations and no issues.
	EscalationNone EscalationLevel = iota

	// EscalationInfo - Auto-complete, but log for optional review.
	// Dashboard shows these as "worth reviewing" but doesn't block.
	// Used for completions with informational recommendations.
	EscalationInfo

	// EscalationReview - Auto-complete, but queue for mandatory review.
	// Orchestrator should review synthesis before spawning next agent.
	// Used for knowledge-producing skills and significant recommendations.
	EscalationReview

	// EscalationBlock - Do NOT auto-complete. Surface immediately.
	// Requires human decision (e.g., visual approval).
	// Used for UI changes that need visual verification approval.
	EscalationBlock

	// EscalationFailed - Do NOT auto-complete. Failure state.
	// Requires intervention to fix before completion can proceed.
	// Used for verification failures.
	EscalationFailed
)

// String returns a human-readable name for the escalation level.
func (e EscalationLevel) String() string {
	switch e {
	case EscalationNone:
		return "none"
	case EscalationInfo:
		return "info"
	case EscalationReview:
		return "review"
	case EscalationBlock:
		return "block"
	case EscalationFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ShouldAutoComplete returns true if this escalation level allows auto-completion.
func (e EscalationLevel) ShouldAutoComplete() bool {
	return e <= EscalationReview
}

// RequiresHumanReview returns true if this escalation level requires human review.
func (e EscalationLevel) RequiresHumanReview() bool {
	return e >= EscalationReview
}

// knowledgeProducingSkills are skills that produce recommendations, decisions,
// or other knowledge artifacts that need human absorption to extract value.
// These should always surface for review regardless of verification status.
var knowledgeProducingSkills = map[string]bool{
	"investigation":  true,
	"architect":      true,
	"research":       true,
	"design-session": true,
	"codebase-audit": true,
	"issue-creation": true,
}

// IsKnowledgeProducingSkill returns true if the skill produces knowledge artifacts.
func IsKnowledgeProducingSkill(skillName string) bool {
	return knowledgeProducingSkills[strings.ToLower(skillName)]
}

// EscalationInput contains all the signals used to determine escalation level.
type EscalationInput struct {
	// Verification results
	VerificationPassed bool     // Did all verification checks pass?
	VerificationErrors []string // Error messages from verification

	// Synthesis/recommendation data
	SkillName      string   // Name of the skill used (e.g., "feature-impl", "investigation")
	Outcome        string   // Synthesis outcome (success, partial, blocked, failed)
	Recommendation string   // Synthesis recommendation (close, spawn-follow-up, escalate, resume)
	NextActions    []string // Follow-up items from synthesis

	// Visual verification
	HasWebChanges       bool // Were web/ files modified?
	HasVisualEvidence   bool // Is there visual verification evidence?
	NeedsVisualApproval bool // Does visual verification need human approval?

	// File change scope
	FileCount int // Number of files changed (0 if unknown)

	// Decision patch detection
	DecisionsWithoutBlocks []DecisionWithoutBlocks // Decisions referenced but lacking blocks: frontmatter

	// Context
	WorkspacePath string // Path to agent workspace (for additional analysis)
	ProjectDir    string // Path to project directory (for git operations)
}

// DetermineEscalation analyzes completion signals and returns the appropriate escalation level.
// The decision tree (in order of precedence):
//
//  1. VERIFICATION FAILED? → EscalationFailed
//  2. SKILL IS KNOWLEDGE-PRODUCING? (investigation, architect, etc.)
//     → Has recommendations? → EscalationReview
//     → No recommendations? → EscalationInfo
//  3. VISUAL VERIFICATION NEEDS APPROVAL? → EscalationBlock
//  4. OUTCOME != "success"? → EscalationReview
//  5. HAS RECOMMENDATIONS? (NextActions > 0 OR Recommendation = spawn-follow-up/escalate/resume)
//     → Large scope (10+ files)? → EscalationReview
//     → Normal scope? → EscalationInfo
//  6. DECISION PATCH WITHOUT BLOCKS? → EscalationInfo
//  7. LARGE SCOPE? (10+ files) → EscalationInfo
//  8. OTHERWISE → EscalationNone
func DetermineEscalation(input EscalationInput) EscalationLevel {
	// 1. Verification failed - highest priority
	if !input.VerificationPassed {
		return EscalationFailed
	}

	// 2. Knowledge-producing skills need review to extract value
	if IsKnowledgeProducingSkill(input.SkillName) {
		if hasSignificantRecommendations(input) {
			return EscalationReview
		}
		// Even without recommendations, knowledge work is worth reviewing
		return EscalationInfo
	}

	// 3. Visual verification needs human approval
	if input.NeedsVisualApproval {
		return EscalationBlock
	}

	// 4. Non-success outcomes need review
	if input.Outcome != "" && input.Outcome != "success" {
		return EscalationReview
	}

	// 5. Has recommendations - needs at least Info level
	if hasSignificantRecommendations(input) {
		if input.FileCount > 10 {
			return EscalationReview // Large scope + recommendations = higher priority
		}
		return EscalationInfo
	}

	// 6. Decision patch without blocks: frontmatter - suggest adding keywords
	if len(input.DecisionsWithoutBlocks) > 0 {
		return EscalationInfo
	}

	// 7. Large scope without recommendations - still worth noting
	if input.FileCount > 10 {
		return EscalationInfo
	}

	// 8. Clean completion - no human attention needed
	return EscalationNone
}

// hasSignificantRecommendations returns true if the synthesis contains
// recommendations that should be surfaced to the orchestrator.
func hasSignificantRecommendations(input EscalationInput) bool {
	// Check for follow-up items
	if len(input.NextActions) > 0 {
		return true
	}

	// Check for non-close recommendations
	rec := strings.ToLower(input.Recommendation)
	return rec == "spawn-follow-up" || rec == "escalate" || rec == "resume" || rec == "continue"
}

// DetermineEscalationFromCompletion is a convenience function that builds
// EscalationInput from verification results and synthesis, then determines escalation.
func DetermineEscalationFromCompletion(
	verificationResult VerificationResult,
	synthesis *Synthesis,
	beadsID, workspacePath, projectDir string,
) EscalationLevel {
	input := EscalationInput{
		VerificationPassed: verificationResult.Passed,
		VerificationErrors: verificationResult.Errors,
		WorkspacePath:      workspacePath,
		ProjectDir:         projectDir,
	}

	// Extract skill name from workspace
	if workspacePath != "" {
		input.SkillName, _ = ExtractSkillNameFromSpawnContext(workspacePath)
	}

	// Extract synthesis data if available
	if synthesis != nil {
		input.Outcome = strings.ToLower(synthesis.Outcome)
		input.Recommendation = strings.ToLower(synthesis.Recommendation)
		input.NextActions = synthesis.NextActions
	}

	// Check visual verification status
	if projectDir != "" {
		visualResult := VerifyVisualVerification(beadsID, workspacePath, projectDir)
		input.HasWebChanges = visualResult.HasWebChanges
		input.HasVisualEvidence = visualResult.HasEvidence
		input.NeedsVisualApproval = visualResult.NeedsApproval
	}

	// Get file count from git
	if projectDir != "" {
		input.FileCount = countRecentFileChanges(projectDir)
	}

	// Check for decisions without blocks: frontmatter
	if workspacePath != "" && projectDir != "" {
		decisionsWithoutBlocks, _ := FindDecisionsWithoutBlocksFrontmatter(workspacePath, projectDir)
		input.DecisionsWithoutBlocks = decisionsWithoutBlocks
	}

	return DetermineEscalation(input)
}

// countRecentFileChanges counts the number of files changed in recent commits.
func countRecentFileChanges(projectDir string) int {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails (e.g., not enough commits), try last 1 commit
		cmd = exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return 0
		}
	}

	count := 0
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// EscalationReason provides a human-readable explanation for an escalation level.
type EscalationReason struct {
	Level       EscalationLevel
	Reason      string   // Primary reason for this level
	Details     []string // Additional context
	CanOverride bool     // Whether human can override (e.g., Block -> complete with --approve)
}

// ExplainEscalation returns a detailed explanation of why a particular
// escalation level was determined. Useful for debugging and dashboard display.
func ExplainEscalation(input EscalationInput) EscalationReason {
	level := DetermineEscalation(input)
	reason := EscalationReason{Level: level}

	switch level {
	case EscalationFailed:
		reason.Reason = "Verification failed"
		reason.Details = input.VerificationErrors
		reason.CanOverride = false

	case EscalationBlock:
		reason.Reason = "Visual verification requires human approval"
		reason.Details = []string{
			"web/ files were modified",
			"Visual verification evidence found, but explicit approval needed",
			"Use: orch complete <id> --approve",
		}
		reason.CanOverride = true

	case EscalationReview:
		if IsKnowledgeProducingSkill(input.SkillName) && hasSignificantRecommendations(input) {
			reason.Reason = "Knowledge-producing skill with recommendations"
			reason.Details = append([]string{
				"Skill: " + input.SkillName,
				"Recommendations should be reviewed for follow-up work",
			}, formatRecommendations(input)...)
		} else if input.Outcome != "" && input.Outcome != "success" {
			reason.Reason = "Non-success outcome: " + input.Outcome
			reason.Details = []string{"Review synthesis to understand partial/blocked state"}
		} else if hasSignificantRecommendations(input) && input.FileCount > 10 {
			reason.Reason = "Large scope with recommendations"
			reason.Details = append([]string{
				"Files changed: " + string(rune(input.FileCount)),
				"Recommendations need review",
			}, formatRecommendations(input)...)
		}
		reason.CanOverride = true

	case EscalationInfo:
		if IsKnowledgeProducingSkill(input.SkillName) {
			reason.Reason = "Knowledge-producing skill (optional review)"
			reason.Details = []string{"Skill: " + input.SkillName}
		} else if hasSignificantRecommendations(input) {
			reason.Reason = "Has recommendations (optional review)"
			reason.Details = formatRecommendations(input)
		} else if len(input.DecisionsWithoutBlocks) > 0 {
			reason.Reason = "Decision patch detected - consider adding blocks: keywords"
			reason.Details = []string{}
			for _, decision := range input.DecisionsWithoutBlocks {
				reason.Details = append(reason.Details,
					fmt.Sprintf("Action: Consider adding blocks: keywords to %s", decision.Filename))
			}
		} else if input.FileCount > 10 {
			reason.Reason = "Large scope (optional review)"
			reason.Details = []string{"Files changed: " + string(rune(input.FileCount))}
		}
		reason.CanOverride = true

	case EscalationNone:
		reason.Reason = "Clean completion"
		reason.Details = []string{"All checks passed, no recommendations"}
		reason.CanOverride = true
	}

	return reason
}

// formatRecommendations formats recommendation data for display.
func formatRecommendations(input EscalationInput) []string {
	var details []string
	if input.Recommendation != "" && input.Recommendation != "close" {
		details = append(details, "Recommendation: "+input.Recommendation)
	}
	for _, action := range input.NextActions {
		details = append(details, "Action: "+action)
	}
	return details
}
