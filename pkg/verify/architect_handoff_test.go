package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyArchitectHandoff_NonArchitectSkill(t *testing.T) {
	// Non-architect skills should always pass
	skills := []string{"feature-impl", "systematic-debugging", "investigation", ""}
	for _, skill := range skills {
		result := VerifyArchitectHandoff("/tmp/nonexistent", skill, "", "")
		if !result.Passed {
			t.Errorf("VerifyArchitectHandoff(%q) should pass for non-architect skill, got errors: %v", skill, result.Errors)
		}
	}
}

func TestVerifyArchitectHandoff_MissingSynthesis(t *testing.T) {
	// Missing SYNTHESIS.md should pass (handled by synthesis gate)
	dir := t.TempDir()
	result := VerifyArchitectHandoff(dir, "architect", "", "")
	if !result.Passed {
		t.Errorf("should pass when SYNTHESIS.md is missing (synthesis gate handles that)")
	}
}

func TestVerifyArchitectHandoff_MissingRecommendation(t *testing.T) {
	dir := t.TempDir()
	synthesis := `## TLDR
Designed a new caching layer.

## Next
The team should implement this design.
`
	if err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifyArchitectHandoff(dir, "architect", "", "")
	if result.Passed {
		t.Error("should fail when **Recommendation:** field is missing")
	}
	if len(result.GatesFailed) != 1 || result.GatesFailed[0] != GateArchitectHandoff {
		t.Errorf("expected gate %q failed, got %v", GateArchitectHandoff, result.GatesFailed)
	}
}

func TestVerifyArchitectHandoff_ValidRecommendations(t *testing.T) {
	validValues := []string{"close", "implement", "escalate", "spawn", "continue", "fix", "refactor"}

	for _, rec := range validValues {
		t.Run(rec, func(t *testing.T) {
			dir := t.TempDir()
			synthesis := `## TLDR
Designed a new feature.

## Next
**Recommendation:** ` + rec + `

Follow-up work needed.
`
			if err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
				t.Fatal(err)
			}

			// Without beadsID, skips implementation issue check
			result := VerifyArchitectHandoff(dir, "architect", "", "")
			if !result.Passed {
				t.Errorf("should pass for valid recommendation %q, got errors: %v", rec, result.Errors)
			}
		})
	}
}

func TestVerifyArchitectHandoff_UnrecognizedRecommendation(t *testing.T) {
	dir := t.TempDir()
	synthesis := `## TLDR
Reviewed the architecture.

## Next
**Recommendation:** maybe

Not sure what to do.
`
	if err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifyArchitectHandoff(dir, "architect", "", "")
	if result.Passed {
		t.Error("should fail for unrecognized recommendation 'maybe'")
	}
	if len(result.Errors) == 0 {
		t.Error("should have error message")
	}
}

func TestVerifyArchitectHandoff_CapitalizedRecommendation(t *testing.T) {
	// The regex extracts the value, and extractRecommendation lowercases it
	dir := t.TempDir()
	synthesis := `## TLDR
Designed improvements.

## Next
**Recommendation:** Implement

Create the implementation.
`
	if err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifyArchitectHandoff(dir, "architect", "", "")
	if !result.Passed {
		t.Errorf("should pass for capitalized recommendation 'Implement', got errors: %v", result.Errors)
	}
}

// TestVerifyArchitectHandoff_ReproductionScenario reproduces the exact bug:
// architect agents complete with design docs but no **Recommendation:** field,
// causing maybeAutoCreateImplementationIssue() to silently skip issue creation.
func TestVerifyArchitectHandoff_ReproductionScenario(t *testing.T) {
	dir := t.TempDir()

	// This is the kind of SYNTHESIS.md that the 3 failing architects produced:
	// - Good TLDR, good design doc, good Next section
	// - Missing **Recommendation:** field
	synthesis := `## TLDR
Designed exploration mode decomposition with parallel agent spawning and verification gates.

## Delta (What Changed)
Created design document at .kb/investigations/2026-03-11-design-exploration-mode.md

## Evidence (What Was Observed)
Reviewed existing spawn pipeline and gate infrastructure.

## Knowledge (What Was Learned)
The exploration mode requires three phases: decompose, parallelize, verify.

## Next (What Should Happen)
The team should implement the exploration mode design. Key steps:
- Add decomposition logic to spawn pipeline
- Create parallel verification gates
- Wire into daemon autonomous processing
`
	if err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
		t.Fatal(err)
	}

	result := VerifyArchitectHandoff(dir, "architect", "", "")

	// This MUST fail — the missing Recommendation field is exactly the bug
	if result.Passed {
		t.Fatal("REPRODUCTION FAILED: gate should block architect without **Recommendation:** field")
	}
	if len(result.GatesFailed) != 1 || result.GatesFailed[0] != GateArchitectHandoff {
		t.Errorf("expected GateArchitectHandoff failure, got %v", result.GatesFailed)
	}
	// Error message should guide the architect to fix it
	if len(result.Errors) == 0 {
		t.Fatal("expected error message with valid recommendation values")
	}
}

func TestVerifyArchitectHandoff_CloseRecommendation_NoIssueNeeded(t *testing.T) {
	// "close" recommendation should pass without needing an implementation issue
	dir := t.TempDir()
	synthesis := `## TLDR
Reviewed architecture, no changes needed.

## Next
**Recommendation:** close

No follow-up required.
`
	if err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
		t.Fatal(err)
	}

	// Even with beadsID, "close" doesn't require implementation issue
	result := VerifyArchitectHandoff(dir, "architect", "orch-go-test1", dir)
	if !result.Passed {
		t.Errorf("should pass for 'close' recommendation without implementation issue, got errors: %v", result.Errors)
	}
}

func TestIsActionableArchitectRecommendation(t *testing.T) {
	tests := []struct {
		recommendation string
		expected       bool
	}{
		{"implement", true},
		{"escalate", true},
		{"spawn", true},
		{"continue", true},
		{"fix", true},
		{"refactor", true},
		{"close", false},
		{"", false},
		{"maybe", false},
		{"Implement", true},  // case insensitive
		{"REFACTOR", true},   // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.recommendation, func(t *testing.T) {
			got := IsActionableArchitectRecommendation(tt.recommendation)
			if got != tt.expected {
				t.Errorf("IsActionableArchitectRecommendation(%q) = %v, want %v", tt.recommendation, got, tt.expected)
			}
		})
	}
}
