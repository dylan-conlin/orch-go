// Package verify provides verification helpers for agent completion.
// This file implements the Architect Handoff verification gate.
// Ensures architect agents declare explicit recommendations before completion
// AND verifies that actionable recommendations have corresponding implementation issues.
//
// Root cause: architect agents completed with design docs but no implementation
// issues because (1) the handoff constraint lived in skill prose (soft harness),
// not in the verification pipeline (hard harness), and (2) the auto-create mechanism
// ran after gates and failed silently.
package verify

import (
	"fmt"
	"strings"
)

// GateArchitectHandoff is the gate name for architect handoff verification.
const GateArchitectHandoff = "architect_handoff"

// validArchitectRecommendations are the recognized recommendation values.
// Actionable ones trigger auto-create in complete_architect.go.
// "close" is valid but signals no follow-up needed.
var validArchitectRecommendations = map[string]bool{
	"close":          true,
	"implement":      true,
	"escalate":       true,
	"spawn":          true,
	"spawn-follow-up": true,
	"continue":       true,
	"fix":            true,
	"refactor":       true,
}

// IsActionableArchitectRecommendation returns true if the recommendation
// indicates follow-up work is needed (not just closing the issue).
// Exported for use by the auto-create mechanism in complete_architect.go.
func IsActionableArchitectRecommendation(recommendation string) bool {
	r := strings.ToLower(strings.TrimSpace(recommendation))
	switch r {
	case "implement", "escalate", "spawn", "spawn-follow-up", "continue", "fix", "refactor":
		return true
	default:
		return false
	}
}

// VerifyArchitectHandoff checks that architect agents have declared an explicit
// recommendation in their SYNTHESIS.md AND that actionable recommendations have
// corresponding implementation issues in beads.
//
// When beadsID is non-empty and the recommendation is actionable, the gate checks
// that an implementation issue exists via three signals (in order):
//  1. Title pattern: issue title contains "(from architect <beadsID>)" (auto-created)
//  2. Comment evidence: "Phase: Handoff - Created implementation issues:" in comments
//  3. Comment opt-out: "No implementation issues:" in Phase: Handoff comment
//
// The auto-create mechanism runs before this gate in the completion pipeline,
// so by the time this check runs, the title-pattern issue should exist.
//
// Returns a passing result for non-architect skills.
func VerifyArchitectHandoff(workspacePath, skill, beadsID, projectDir string, comments []Comment) *VerificationResult {
	result := &VerificationResult{Passed: true}

	// Only applies to architect skill
	if skill != "architect" {
		return result
	}

	// Try to parse SYNTHESIS.md
	synthesis, err := ParseSynthesis(workspacePath)
	if err != nil {
		// Architect skill REQUIRES SYNTHESIS.md with a Recommendation field.
		// Previously this returned passing (deferring to the V2+ synthesis gate),
		// but architect defaults to V1 where the synthesis gate doesn't run.
		// This was the root cause of 9/9 architect completions closing without
		// implementation issues — the gate silently passed.
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("SYNTHESIS.md is missing or unparseable — architect skill requires "+
				"SYNTHESIS.md with **Recommendation:** field. "+
				"Valid values: %s", formatValidRecommendations()))
		result.GatesFailed = append(result.GatesFailed, GateArchitectHandoff)
		return result
	}

	recommendation := strings.TrimSpace(synthesis.Recommendation)

	if recommendation == "" {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("SYNTHESIS.md missing **Recommendation:** field (required for architect skill). "+
				"Add to the Next section: **Recommendation:** <value>. "+
				"Valid values: %s", formatValidRecommendations()))
		result.GatesFailed = append(result.GatesFailed, GateArchitectHandoff)
		return result
	}

	if !validArchitectRecommendations[recommendation] {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Unrecognized architect recommendation %q. "+
				"Valid values: %s", recommendation, formatValidRecommendations()))
		result.GatesFailed = append(result.GatesFailed, GateArchitectHandoff)
		return result
	}

	// For actionable recommendations, verify implementation issue exists.
	// Skip this check if beadsID is empty (e.g., unit tests without beads context).
	if IsActionableArchitectRecommendation(recommendation) && beadsID != "" {
		// Check 1: Title pattern match (auto-created issues)
		exists, err := HasImplementationFollowUp(beadsID, projectDir)
		if err != nil {
			// Beads query failed — don't block completion on infrastructure issues,
			// but warn so the orchestrator can investigate.
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Could not verify implementation issue exists for architect %s: %v", beadsID, err))
			return result
		}

		if exists {
			return result
		}

		// Check 2: Comment evidence — architect manually created issues and reported them
		// Pattern: "Phase: Handoff - Created implementation issues: <ids>"
		if hasHandoffIssueEvidence(comments) {
			return result
		}

		// Check 3: Comment opt-out — architect explicitly declared no issues with reason
		// Pattern: "No implementation issues: <reason>"
		if hasHandoffOptOut(comments) {
			result.Warnings = append(result.Warnings,
				"Architect declared 'No implementation issues' — design is advisory only")
			return result
		}

		// No implementation issue found via any signal
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Architect recommendation is %q but no implementation issue found for %s. "+
				"Expected one of:\n"+
				"  1. Issue with title containing \"(from architect %s)\" (auto-created)\n"+
				"  2. Comment: \"Phase: Handoff - Created implementation issues: <ids>\"\n"+
				"  3. Comment: \"Phase: Handoff - No implementation issues: <reason>\"\n"+
				"Auto-create may have failed — create manually via: "+
				"bd create \"<title> (from architect %s)\" --type task -l triage:ready",
				recommendation, beadsID, beadsID, beadsID))
		result.GatesFailed = append(result.GatesFailed, GateArchitectHandoff)
		return result
	}

	return result
}

// hasHandoffIssueEvidence checks beads comments for evidence that the architect
// manually created implementation issues (reported in Phase: Handoff comment).
func hasHandoffIssueEvidence(comments []Comment) bool {
	for _, c := range comments {
		lower := strings.ToLower(c.Text)
		if strings.Contains(lower, "phase: handoff") && strings.Contains(lower, "created implementation issues") {
			return true
		}
	}
	return false
}

// hasHandoffOptOut checks beads comments for an explicit opt-out from creating
// implementation issues (architect declared design is advisory-only).
func hasHandoffOptOut(comments []Comment) bool {
	for _, c := range comments {
		lower := strings.ToLower(c.Text)
		if strings.Contains(lower, "no implementation issues:") {
			return true
		}
	}
	return false
}

// formatValidRecommendations returns a human-readable list of valid recommendations.
func formatValidRecommendations() string {
	return "close, implement, escalate, spawn, spawn-follow-up, continue, fix, refactor"
}
