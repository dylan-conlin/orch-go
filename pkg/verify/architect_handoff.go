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
	"close":     true,
	"implement": true,
	"escalate":  true,
	"spawn":     true,
	"continue":  true,
	"fix":       true,
	"refactor":  true,
}

// IsActionableArchitectRecommendation returns true if the recommendation
// indicates follow-up work is needed (not just closing the issue).
// Exported for use by the auto-create mechanism in complete_architect.go.
func IsActionableArchitectRecommendation(recommendation string) bool {
	r := strings.ToLower(strings.TrimSpace(recommendation))
	switch r {
	case "implement", "escalate", "spawn", "continue", "fix", "refactor":
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
// that an implementation issue exists (title containing "from architect <beadsID>").
// The auto-create mechanism runs before this gate in the completion pipeline,
// so by the time this check runs, the issue should exist.
//
// Returns a passing result for non-architect skills.
func VerifyArchitectHandoff(workspacePath, skill, beadsID, projectDir string) *VerificationResult {
	result := &VerificationResult{Passed: true}

	// Only applies to architect skill
	if skill != "architect" {
		return result
	}

	// Try to parse SYNTHESIS.md
	synthesis, err := ParseSynthesis(workspacePath)
	if err != nil {
		// If SYNTHESIS.md doesn't exist, the synthesis gate handles that separately
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
		exists, err := HasImplementationFollowUp(beadsID, projectDir)
		if err != nil {
			// Beads query failed — don't block completion on infrastructure issues,
			// but warn so the orchestrator can investigate.
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Could not verify implementation issue exists for architect %s: %v", beadsID, err))
		} else if !exists {
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Architect recommendation is %q but no implementation issue found for %s. "+
					"Expected an issue with title containing \"(from architect %s)\". "+
					"Auto-create may have failed — check stderr output above, or create manually via: "+
					"bd create \"<title> (from architect %s)\" --type task -l triage:ready",
					recommendation, beadsID, beadsID, beadsID))
			result.GatesFailed = append(result.GatesFailed, GateArchitectHandoff)
			return result
		}
	}

	return result
}

// formatValidRecommendations returns a human-readable list of valid recommendations.
func formatValidRecommendations() string {
	return "close, implement, escalate, spawn, continue, fix, refactor"
}
