// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// SynthesisContentResult represents the result of validating SYNTHESIS.md content
// against primary sources (git, beads comments, spawn time).
type SynthesisContentResult struct {
	Passed   bool     // Whether verification passed (all critical checks)
	Errors   []string // Error messages (blocking - false claims detected)
	Warnings []string // Warning messages (non-blocking - uncorroborated claims)

	// Validation details
	HasSynthesis           bool   // Whether SYNTHESIS.md exists and was parsed
	HasEvidenceSection     bool   // Whether Evidence section has content
	EvidenceClaimsTestPass bool   // Whether Evidence section claims tests passed
	BeadsHasTestEvidence   bool   // Whether beads comments have test evidence
	DurationReasonable     bool   // Whether claimed duration is reasonable
	ClaimedDuration        string // Duration claimed in SYNTHESIS.md
	ActualDuration         string // Actual duration from spawn time to now
}

// testPassClaims defines patterns that indicate a claim of test success in SYNTHESIS.
// These are simpler than the evidence patterns because they detect claims, not evidence.
var testPassClaims = []*regexp.Regexp{
	regexp.MustCompile(`(?i)tests?\s+pass(ed|ing)?`),
	regexp.MustCompile(`(?i)all\s+tests?\s+(pass|succeed)`),
	regexp.MustCompile(`(?i)\d+\s+tests?\s+pass(ed|ing)?`),
	regexp.MustCompile(`(?i)test\s+suite\s+pass(ed|ing)?`),
	regexp.MustCompile(`(?i)verification\s+pass(ed)?`),
	regexp.MustCompile(`(?i)build\s+(passed|succeeded|successful)`),
}

// VerifySynthesisContent validates SYNTHESIS.md content against primary sources.
// This implements the Evidence Hierarchy principle by treating primary sources
// (git commits, beads comments, spawn time) as authoritative.
//
// The verification:
// - Parses SYNTHESIS.md and extracts claims
// - Cross-validates test claims in Evidence section against beads comment patterns
// - Validates duration claims against actual spawn-to-completion time
//
// Returns warnings for uncorroborated claims (not errors, to avoid false blocks).
// This is a "trust but verify" approach that flags suspicious claims without blocking.
func VerifySynthesisContent(beadsID, workspacePath, projectDir string) SynthesisContentResult {
	result := SynthesisContentResult{Passed: true}

	// Skip if no workspace
	if workspacePath == "" {
		result.Warnings = append(result.Warnings, "no workspace path - skipping synthesis content verification")
		return result
	}

	// Parse SYNTHESIS.md
	synthesis, err := ParseSynthesis(workspacePath)
	if err != nil {
		// No SYNTHESIS.md is handled by VerifySynthesis - not an error here
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("could not parse SYNTHESIS.md: %v - skipping content verification", err))
		return result
	}
	result.HasSynthesis = true

	// Check if Evidence section exists and has content
	result.HasEvidenceSection = synthesis.Evidence != ""

	// Check if Evidence section claims tests passed
	result.EvidenceClaimsTestPass = evidenceClaimsTestPass(synthesis.Evidence)

	// If Evidence claims tests passed, check beads comments for actual test evidence
	if result.EvidenceClaimsTestPass {
		comments, err := GetComments(beadsID)
		if err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("failed to get beads comments for cross-validation: %v", err))
		} else {
			hasEvidence, _ := HasTestExecutionEvidence(comments)
			result.BeadsHasTestEvidence = hasEvidence

			if !hasEvidence {
				// This is a warning, not an error - agent may have forgotten to report test output
				result.Warnings = append(result.Warnings,
					"SYNTHESIS.md Evidence section claims tests passed but no test execution evidence in beads comments",
					"Consider adding test output via: bd comment <id> 'Tests: go test ./... - PASS (N tests)'",
				)
			}
		}
	}

	// Validate duration claims
	result.ClaimedDuration = synthesis.Duration
	if synthesis.Duration != "" {
		spawnTime := spawn.ReadSpawnTime(workspacePath)
		if !spawnTime.IsZero() {
			actualDuration := time.Since(spawnTime)
			result.ActualDuration = formatDuration(actualDuration)
			result.DurationReasonable = isDurationReasonable(synthesis.Duration, actualDuration)

			if !result.DurationReasonable {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("SYNTHESIS.md claims duration '%s' but actual time is ~%s",
						synthesis.Duration, result.ActualDuration),
				)
			}
		}
	}

	return result
}

// evidenceClaimsTestPass checks if the Evidence section claims tests passed.
func evidenceClaimsTestPass(evidence string) bool {
	if evidence == "" {
		return false
	}
	for _, pattern := range testPassClaims {
		if pattern.MatchString(evidence) {
			return true
		}
	}
	return false
}

// formatDuration formats a duration in a human-readable way similar to SYNTHESIS claims.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// isDurationReasonable checks if the claimed duration is reasonably close to actual.
// Allows for some slack since SYNTHESIS is written during session, not at exact completion.
func isDurationReasonable(claimed string, actual time.Duration) bool {
	// Parse claimed duration
	claimedDuration := parseDurationClaim(claimed)
	if claimedDuration == 0 {
		// Can't parse - give benefit of the doubt
		return true
	}

	// Allow 50% variance to account for timing differences
	// (SYNTHESIS written before Phase: Complete, etc.)
	minReasonable := claimedDuration / 2
	maxReasonable := claimedDuration * 3 / 2

	return actual >= minReasonable && actual <= maxReasonable
}

// parseDurationClaim attempts to parse a duration string from SYNTHESIS.md.
// Handles patterns like "45m", "1.5h", "2 hours", "90 minutes", "~1h", etc.
func parseDurationClaim(s string) time.Duration {
	s = strings.TrimSpace(strings.ToLower(s))
	// Remove common prefixes
	s = strings.TrimPrefix(s, "~")
	s = strings.TrimPrefix(s, "about ")
	s = strings.TrimPrefix(s, "approximately ")

	// Try standard Go duration format first
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}

	// Try patterns with words
	patterns := []struct {
		re         *regexp.Regexp
		multiplier time.Duration
	}{
		{regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(?:hours?|h)$`), time.Hour},
		{regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(?:minutes?|min|m)$`), time.Minute},
		{regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(?:seconds?|sec|s)$`), time.Second},
	}

	for _, p := range patterns {
		if matches := p.re.FindStringSubmatch(s); len(matches) >= 2 {
			var value float64
			fmt.Sscanf(matches[1], "%f", &value)
			return time.Duration(value * float64(p.multiplier))
		}
	}

	return 0
}

// VerifySynthesisContentForCompletion is a convenience function for use in VerifyCompletionFull.
// Returns nil if there's nothing to validate (no SYNTHESIS.md).
// Returns the result with warnings for uncorroborated claims.
func VerifySynthesisContentForCompletion(beadsID, workspacePath, projectDir string) *SynthesisContentResult {
	// Skip if no workspace
	if workspacePath == "" {
		return nil
	}

	result := VerifySynthesisContent(beadsID, workspacePath, projectDir)

	// Return nil if no synthesis to validate
	if !result.HasSynthesis {
		return nil
	}

	return &result
}
