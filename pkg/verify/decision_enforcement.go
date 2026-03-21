// Package verify provides verification helpers for agent completion.
// This file implements the Decision Enforcement verification gate.
// When architect sessions produce decisions, those decisions must declare
// an enforcement type: gate, hook, convention, or context-only.
// This classifies decisions at birth so stale detection and budget caps
// can operate on enforcement type.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GateDecisionEnforcement is the gate name for decision enforcement type verification.
const GateDecisionEnforcement = "decision_enforcement"

// validEnforcementTypes are the recognized enforcement type values for decisions.
var validEnforcementTypes = map[string]bool{
	"gate":         true, // Enforced by completion/spawn gate code
	"hook":         true, // Enforced by pre-commit or other hooks
	"convention":   true, // Enforced by skill prose / agent behavior
	"context-only": true, // Informational — no enforcement mechanism
}

// IsValidEnforcementType returns true if the enforcement type is recognized.
func IsValidEnforcementType(enforcementType string) bool {
	return validEnforcementTypes[strings.ToLower(strings.TrimSpace(enforcementType))]
}

// regexEnforcement matches **Enforcement:** field in decision files.
var regexEnforcement = regexp.MustCompile(`(?m)^\*\*Enforcement:\*\*\s*(.+)$`)

// ExtractEnforcementType reads a decision file and extracts the **Enforcement:** field value.
// Returns empty string if the field is not found.
func ExtractEnforcementType(decisionPath string) string {
	data, err := os.ReadFile(decisionPath)
	if err != nil {
		return ""
	}
	matches := regexEnforcement.FindStringSubmatch(string(data))
	if len(matches) >= 2 {
		return strings.ToLower(strings.TrimSpace(matches[1]))
	}
	return ""
}

// VerifyDecisionEnforcement checks that decisions referenced in architect SYNTHESIS.md
// declare a valid **Enforcement:** type. This classifies decisions at birth so that
// lifecycle management (auto-archive, budget cap) can operate on enforcement type.
//
// Returns nil for non-architect skills or when no decisions are referenced.
func VerifyDecisionEnforcement(workspacePath, skill, projectDir string) *VerificationResult {
	if skill != "architect" {
		return nil
	}

	if workspacePath == "" {
		return nil
	}

	// Read SYNTHESIS.md to find decision references
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	data, err := os.ReadFile(synthesisPath)
	if err != nil {
		return nil
	}

	// Reuse the existing decision reference finder from decision_patches.go
	decisionRefs := findDecisionReferences(string(data))
	if len(decisionRefs) == 0 {
		return nil
	}

	result := &VerificationResult{
		Passed:      true,
		GatesFailed: []string{},
	}

	for _, ref := range decisionRefs {
		// Resolve the decision file path
		var decisionPath string
		if filepath.IsAbs(ref) {
			decisionPath = ref
		} else if projectDir != "" {
			decisionPath = filepath.Join(projectDir, ref)
		} else {
			continue
		}

		// Check if file exists
		if _, err := os.Stat(decisionPath); err != nil {
			continue
		}

		enforcement := ExtractEnforcementType(decisionPath)
		if enforcement == "" {
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Decision %s missing **Enforcement:** field. "+
					"Add one of: gate, hook, convention, context-only",
					filepath.Base(decisionPath)))
			result.GatesFailed = append(result.GatesFailed, GateDecisionEnforcement)
		} else if !IsValidEnforcementType(enforcement) {
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Decision %s has invalid enforcement type %q. "+
					"Valid types: gate, hook, convention, context-only",
					filepath.Base(decisionPath), enforcement))
			result.GatesFailed = append(result.GatesFailed, GateDecisionEnforcement)
		}
	}

	if result.Passed && len(result.Errors) == 0 {
		return nil
	}
	return result
}
