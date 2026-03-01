// Package verify provides verification helpers for agent completion.
// This file implements the Architectural Choices verification gate.
// See: .kb/investigations/2026-02-20-design-tradeoff-visibility-for-non-code-reading-orchestrator.md
package verify

import (
	"fmt"
	"strings"
)

// GateArchitecturalChoices is the gate name for architectural choices verification.
const GateArchitecturalChoices = "architectural_choices"

// VerifyArchitecturalChoices checks if SYNTHESIS.md contains an "Architectural Choices"
// section for skills that require tradeoff declaration. Returns a passing result for
// skills not subject to this gate.
//
// The section must contain actual content — either:
// - Structured choices with "What I chose" / "What I rejected" / "Why" / "Risk accepted"
// - Or the explicit no-choices declaration: "No architectural choices — task was within existing patterns."
func VerifyArchitecturalChoices(workspacePath, skill string) *VerificationResult {
	result := &VerificationResult{Passed: true}

	// Gate selection is handled by the verify level system (V0-V3) in check.go.
	// This function runs unconditionally when called — the caller decides whether to invoke it.

	// Try to parse SYNTHESIS.md
	synthesis, err := ParseSynthesis(workspacePath)
	if err != nil {
		// If SYNTHESIS.md doesn't exist, the synthesis gate handles that separately
		return result
	}

	// Check if Architectural Choices section exists and has content
	if strings.TrimSpace(synthesis.ArchitecturalChoices) == "" {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("SYNTHESIS.md missing 'Architectural Choices' section (required for %s skill). "+
				"Add the section with your tradeoffs, or declare: "+
				"'No architectural choices — task was within existing patterns.'", skill))
		result.GatesFailed = append(result.GatesFailed, GateArchitecturalChoices)
	}

	return result
}

// ExtractArchitecturalChoicesContent extracts the Architectural Choices section
// from raw SYNTHESIS.md content. Used by the completion pipeline to surface
// tradeoff content to the orchestrator.
func ExtractArchitecturalChoicesContent(content string) string {
	return extractSection(content, "Architectural Choices")
}

// FormatArchitecturalChoicesForCompletion formats the architectural choices content
// for display during orch complete. Returns empty string if no choices to surface.
func FormatArchitecturalChoicesForCompletion(workspacePath string) string {
	synthesis, err := ParseSynthesis(workspacePath)
	if err != nil || synthesis == nil {
		return ""
	}

	choices := strings.TrimSpace(synthesis.ArchitecturalChoices)
	if choices == "" {
		return ""
	}

	// Don't surface the no-choices declaration
	if strings.HasPrefix(strings.ToLower(choices), "no architectural choices") {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n--- Architectural Choices ---\n")
	sb.WriteString(choices)
	sb.WriteString("\n-----------------------------\n")
	return sb.String()
}
