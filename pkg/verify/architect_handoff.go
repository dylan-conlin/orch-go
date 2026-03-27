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
	"os"
	"path/filepath"
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
// For multi-phase designs (SYNTHESIS.md contains Phase N/Layer N/Step N/Stage N),
// the gate requires one issue per phase, not just one issue total.
//
// Issue verification uses three signals (in order):
//  1. Title pattern count: issues with "(from architect <beadsID>)" in title
//  2. Comment evidence count: issue IDs in "Phase: Handoff - Created implementation issues:"
//  3. Comment opt-out: "No implementation issues:" in Phase: Handoff comment
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

	// For actionable recommendations, verify implementation issues exist.
	if IsActionableArchitectRecommendation(recommendation) {
		// Detect multi-phase structure in SYNTHESIS.md
		phaseCount := detectPhasesFromWorkspace(workspacePath)
		requiredIssues := 1
		if phaseCount > 1 {
			requiredIssues = phaseCount
		}

		// Check 1: Title pattern count (auto-created issues, requires beadsID)
		titleCount := 0
		if beadsID != "" {
			count, err := CountImplementationFollowUps(beadsID, projectDir)
			if err != nil {
				// Beads query failed — don't block on infrastructure issues, but warn.
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Could not verify implementation issues for architect %s: %v", beadsID, err))
				return result
			}
			titleCount = count
			if titleCount >= requiredIssues {
				return result
			}
		}

		// Check 2: Comment evidence — count issue IDs reported in handoff comment
		commentCount := countHandoffIssueEvidence(comments)
		if commentCount >= requiredIssues {
			return result
		}

		// Check 3: Comment opt-out — architect explicitly declared no issues with reason
		if hasHandoffOptOut(comments) {
			result.Warnings = append(result.Warnings,
				"Architect declared 'No implementation issues' — design is advisory only")
			return result
		}

		// If beadsID is empty and no comment evidence: skip check (unit test compat)
		if beadsID == "" && commentCount == 0 {
			return result
		}

		// Not enough issues for the detected phases
		totalFound := titleCount + commentCount
		result.Passed = false
		if requiredIssues > 1 {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Multi-phase design detected (%d phases) but only %d implementation issue(s) found for architect %s. "+
					"Each phase needs a corresponding issue.\n"+
					"  Detected phases: %d (from Phase/Layer/Step/Stage indicators in SYNTHESIS.md)\n"+
					"  Issues found: %d (title pattern: %d, comment evidence: %d)\n"+
					"  Expected one of:\n"+
					"    1. %d issues with title containing \"(from architect %s)\" (auto-created)\n"+
					"    2. Comment: \"Phase: Handoff - Created implementation issues: <id1>, <id2>, ...\" (%d IDs)\n"+
					"    3. Comment: \"Phase: Handoff - No implementation issues: <reason>\" (opt-out)",
					requiredIssues, totalFound, beadsID,
					requiredIssues, totalFound, titleCount, commentCount,
					requiredIssues, beadsID, requiredIssues))
		} else {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Architect recommendation is %q but no implementation issue found for %s. "+
					"Expected one of:\n"+
					"  1. Issue with title containing \"(from architect %s)\" (auto-created)\n"+
					"  2. Comment: \"Phase: Handoff - Created implementation issues: <ids>\"\n"+
					"  3. Comment: \"Phase: Handoff - No implementation issues: <reason>\"\n"+
					"Auto-create may have failed — create manually via: "+
					"bd create \"<title> (from architect %s)\" --type task -l triage:ready",
					recommendation, beadsID, beadsID, beadsID))
		}
		result.GatesFailed = append(result.GatesFailed, GateArchitectHandoff)
		return result
	}

	return result
}

// countHandoffIssueEvidence counts issue IDs in a "Phase: Handoff - Created implementation issues:"
// comment. Returns 0 if no such comment exists.
func countHandoffIssueEvidence(comments []Comment) int {
	for _, c := range comments {
		lower := strings.ToLower(c.Text)
		if strings.Contains(lower, "phase: handoff") && strings.Contains(lower, "created implementation issues") {
			// Extract the IDs portion after "created implementation issues:"
			idx := strings.Index(lower, "created implementation issues:")
			if idx == -1 {
				continue
			}
			idsStr := c.Text[idx+len("created implementation issues:"):]
			idsStr = strings.TrimSpace(idsStr)
			if idsStr == "" {
				return 0
			}
			// Split by comma and count non-empty entries
			parts := strings.Split(idsStr, ",")
			count := 0
			for _, p := range parts {
				if strings.TrimSpace(p) != "" {
					count++
				}
			}
			return count
		}
	}
	return 0
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

// detectPhasesFromWorkspace reads SYNTHESIS.md from the workspace and detects
// multi-phase structure. Returns 0 if no phases detected or file unreadable.
func detectPhasesFromWorkspace(workspacePath string) int {
	if workspacePath == "" {
		return 0
	}
	data, err := os.ReadFile(filepath.Join(workspacePath, "SYNTHESIS.md"))
	if err != nil {
		return 0
	}
	return DetectPhases(string(data))
}

// formatValidRecommendations returns a human-readable list of valid recommendations.
func formatValidRecommendations() string {
	return "close, implement, escalate, spawn, spawn-follow-up, continue, fix, refactor"
}
