// Package verify provides verification helpers for agent completion.
// This file implements the Architect Handoff verification gate.
// Ensures architect agents declare explicit recommendations before completion,
// enabling the auto-create mechanism in complete_architect.go to function.
//
// Root cause: architect agents completed with design docs but no implementation
// issues because the handoff constraint lived in skill prose (soft harness),
// not in the verification pipeline (hard harness).
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

// VerifyArchitectHandoff checks that architect agents have declared an explicit
// recommendation in their SYNTHESIS.md. Without this field, the auto-create
// mechanism in complete_architect.go silently skips issue creation.
//
// Returns a passing result for non-architect skills.
func VerifyArchitectHandoff(workspacePath, skill string) *VerificationResult {
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

	return result
}

// formatValidRecommendations returns a human-readable list of valid recommendations.
func formatValidRecommendations() string {
	return "close, implement, escalate, spawn, continue, fix, refactor"
}
