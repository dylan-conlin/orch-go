package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyArchitectHandoff_NonArchitectSkill(t *testing.T) {
	// Non-architect skills should always pass
	skills := []string{"feature-impl", "systematic-debugging", "investigation", ""}
	for _, skill := range skills {
		result := VerifyArchitectHandoff("/tmp/nonexistent", skill, "", "", nil)
		if !result.Passed {
			t.Errorf("VerifyArchitectHandoff(%q) should pass for non-architect skill, got errors: %v", skill, result.Errors)
		}
	}
}

func TestVerifyArchitectHandoff_MissingSynthesis(t *testing.T) {
	// Missing SYNTHESIS.md must FAIL for architect skill.
	// Previously this passed (deferring to V2+ synthesis gate), but architect
	// defaults to V1 where the synthesis gate doesn't run — allowing architects
	// to complete without SYNTHESIS.md and without implementation issues.
	dir := t.TempDir()
	result := VerifyArchitectHandoff(dir, "architect", "", "", nil)
	if result.Passed {
		t.Error("should FAIL when SYNTHESIS.md is missing for architect skill")
	}
	if len(result.GatesFailed) != 1 || result.GatesFailed[0] != GateArchitectHandoff {
		t.Errorf("expected gate %q failed, got %v", GateArchitectHandoff, result.GatesFailed)
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

	result := VerifyArchitectHandoff(dir, "architect", "", "", nil)
	if result.Passed {
		t.Error("should fail when **Recommendation:** field is missing")
	}
	if len(result.GatesFailed) != 1 || result.GatesFailed[0] != GateArchitectHandoff {
		t.Errorf("expected gate %q failed, got %v", GateArchitectHandoff, result.GatesFailed)
	}
}

func TestVerifyArchitectHandoff_ValidRecommendations(t *testing.T) {
	validValues := []string{"close", "implement", "escalate", "spawn", "spawn-follow-up", "continue", "fix", "refactor"}

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
			result := VerifyArchitectHandoff(dir, "architect", "", "", nil)
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

	result := VerifyArchitectHandoff(dir, "architect", "", "", nil)
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

	result := VerifyArchitectHandoff(dir, "architect", "", "", nil)
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

	result := VerifyArchitectHandoff(dir, "architect", "", "", nil)

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
	result := VerifyArchitectHandoff(dir, "architect", "orch-go-test1", dir, nil)
	if !result.Passed {
		t.Errorf("should pass for 'close' recommendation without implementation issue, got errors: %v", result.Errors)
	}
}

func TestVerifyArchitectHandoff_CommentOptOut(t *testing.T) {
	// Architect with actionable recommendation but explicit opt-out in comments
	// should pass with a warning.
	dir := t.TempDir()
	synthesis := `## TLDR
Designed improvements but surfaced blocking questions only.

## Next
**Recommendation:** implement

Need to resolve design questions first.
`
	if err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
		t.Fatal(err)
	}

	comments := []Comment{
		{Text: "Phase: Planning - Analyzing architecture"},
		{Text: "Phase: Handoff - No implementation issues: design surfaced only blocking questions, no actionable work yet"},
	}

	// Use empty beadsID so we skip the title-pattern check but exercise the comment path.
	// With beadsID set, HasImplementationFollowUp would fail (no beads running).
	// Instead, test the comment functions directly.
	if !hasHandoffOptOut(comments) {
		t.Fatal("hasHandoffOptOut should detect 'No implementation issues:' in comments")
	}
}

func TestVerifyArchitectHandoff_CommentEvidence(t *testing.T) {
	// Architect manually created issues and reported them in Phase: Handoff comment
	comments := []Comment{
		{Text: "Phase: Planning - Reviewing codebase"},
		{Text: "Phase: Handoff - Created implementation issues: orch-go-abc12, orch-go-def34"},
	}

	if !hasHandoffIssueEvidence(comments) {
		t.Fatal("hasHandoffIssueEvidence should detect 'Created implementation issues' in comments")
	}
}

func TestVerifyArchitectHandoff_NoCommentEvidence(t *testing.T) {
	// Comments without handoff evidence
	comments := []Comment{
		{Text: "Phase: Planning - Analyzing architecture"},
		{Text: "Phase: Complete - Design written"},
	}

	if hasHandoffIssueEvidence(comments) {
		t.Error("should not detect implementation issue evidence in generic comments")
	}
	if hasHandoffOptOut(comments) {
		t.Error("should not detect opt-out in generic comments")
	}
}

func TestVerifyArchitectHandoff_NilComments(t *testing.T) {
	// Nil comments should not panic
	if hasHandoffIssueEvidence(nil) {
		t.Error("nil comments should return false")
	}
	if hasHandoffOptOut(nil) {
		t.Error("nil comments should return false")
	}
}

// TestVerifyArchitectHandoff_MissingSynthesis_RootCause reproduces the exact root cause:
// architect at V1 verification level has no SYNTHESIS.md, the architect_handoff gate
// previously returned passing (deferring to V2+ synthesis gate that never runs for V1).
// This is the gap that allowed 9/9 architect completions to close without implementation issues.
func TestVerifyArchitectHandoff_MissingSynthesis_RootCause(t *testing.T) {
	dir := t.TempDir()
	// No SYNTHESIS.md file created — simulates architect that didn't write one

	result := VerifyArchitectHandoff(dir, "architect", "orch-go-test1", dir, nil)

	if result.Passed {
		t.Fatal("ROOT CAUSE REPRODUCTION: architect without SYNTHESIS.md must NOT pass handoff gate. " +
			"This was the exact gap: architect_handoff returned passing, synthesis gate (V2+) " +
			"doesn't run for V1 architect skill, so completion succeeded without any implementation issues.")
	}

	if len(result.GatesFailed) == 0 || result.GatesFailed[0] != GateArchitectHandoff {
		t.Errorf("expected GateArchitectHandoff in failed gates, got %v", result.GatesFailed)
	}
}

// TestVerifyCompletionFullWithComments_ArchitectHandoffGateFailure verifies that when an
// architect agent's workspace has a missing Recommendation field, the full verification pipeline
// correctly includes "architect_handoff" in GatesFailed. This is the path the daemon auto-complete
// exercises — if this gate is not in GatesFailed, handleVerificationFailure won't surface the
// right diagnostic when labeling daemon:verification-failed.
func TestVerifyCompletionFullWithComments_ArchitectHandoffGateFailure(t *testing.T) {
	dir := t.TempDir()

	// Create AGENT_MANIFEST.json identifying this as an architect agent at V1 verification
	manifest := `{"workspace_name":"og-arch-test","skill":"architect","beads_id":"orch-go-test-arch","project_dir":"` + dir + `","spawn_time":"2026-03-27T00:00:00Z","tier":"full","verify_level":"V1","review_tier":"review"}`
	if err := os.WriteFile(filepath.Join(dir, "AGENT_MANIFEST.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create SPAWN_CONTEXT.md with architect skill (needed for skill name extraction)
	spawnContext := `## SKILL GUIDANCE (architect)
Some architect instructions here.
`
	if err := os.WriteFile(filepath.Join(dir, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatal(err)
	}

	// Create SYNTHESIS.md WITHOUT **Recommendation:** field — this triggers architect_handoff failure
	synthesis := `## TLDR
Designed a new caching architecture.

## Next
The team should implement the caching layer.
`
	if err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
		t.Fatal(err)
	}

	// Provide pre-fetched comments indicating Phase: Complete
	// (bypasses beads API — simulates what the daemon does after fetching comments)
	comments := []Comment{
		{Text: "Phase: Planning - Analyzing architecture"},
		{Text: "Phase: Complete - Design document written"},
	}

	// Run the full verification pipeline that the daemon calls
	result, err := VerifyCompletionFullWithComments(
		"orch-go-test-arch",
		dir,    // workspacePath
		dir,    // projectDir
		"full", // tier
		comments,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verification MUST fail
	if result.Passed {
		t.Fatal("expected verification to fail for architect without Recommendation field")
	}

	// architect_handoff MUST be in GatesFailed
	foundGate := false
	for _, g := range result.GatesFailed {
		if g == GateArchitectHandoff {
			foundGate = true
		}
	}
	if !foundGate {
		t.Errorf("expected %q in GatesFailed, got %v", GateArchitectHandoff, result.GatesFailed)
	}

	// Skill should be extracted as "architect"
	if result.Skill != "architect" {
		t.Errorf("expected skill 'architect', got %q", result.Skill)
	}

	// Error message should mention Recommendation
	foundRecommendationError := false
	for _, e := range result.Errors {
		if strings.Contains(e, "Recommendation") {
			foundRecommendationError = true
		}
	}
	if !foundRecommendationError {
		t.Errorf("expected error mentioning 'Recommendation', got: %v", result.Errors)
	}
}

// TestVerifyCompletionFullWithComments_ArchitectHandoffGatePass verifies that when
// an architect agent HAS a valid Recommendation and close recommendation, the gate passes.
// This is the positive counterpart to the failure test above.
func TestVerifyCompletionFullWithComments_ArchitectHandoffGatePass(t *testing.T) {
	dir := t.TempDir()

	manifest := `{"workspace_name":"og-arch-pass","skill":"architect","beads_id":"orch-go-test-pass","project_dir":"` + dir + `","spawn_time":"2026-03-27T00:00:00Z","tier":"full","verify_level":"V1","review_tier":"review"}`
	if err := os.WriteFile(filepath.Join(dir, "AGENT_MANIFEST.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	spawnContext := "## SKILL GUIDANCE (architect)\nSome instructions.\n"
	if err := os.WriteFile(filepath.Join(dir, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatal(err)
	}

	// Valid SYNTHESIS.md with close recommendation (no follow-up issue needed)
	synthesis := "## TLDR\nReviewed architecture, no changes needed.\n\n## Next\n**Recommendation:** close\n\nNo follow-up required.\n"
	if err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
		t.Fatal(err)
	}

	comments := []Comment{
		{Text: "Phase: Complete - Architecture review complete, no changes needed"},
	}

	result, err := VerifyCompletionFullWithComments("orch-go-test-pass", dir, dir, "full", comments)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// architect_handoff should NOT be in GatesFailed
	for _, g := range result.GatesFailed {
		if g == GateArchitectHandoff {
			t.Errorf("architect_handoff should NOT be in GatesFailed for valid close recommendation, got %v", result.GatesFailed)
		}
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
		{"spawn-follow-up", true},
		{"close", false},
		{"", false},
		{"maybe", false},
		{"Implement", true},       // case insensitive
		{"REFACTOR", true},        // case insensitive
		{"Spawn-Follow-Up", true}, // case insensitive
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
