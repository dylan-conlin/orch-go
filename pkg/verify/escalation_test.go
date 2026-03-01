package verify

import (
	"testing"
)

func TestEscalationLevel_String(t *testing.T) {
	tests := []struct {
		level    EscalationLevel
		expected string
	}{
		{EscalationNone, "none"},
		{EscalationInfo, "info"},
		{EscalationReview, "review"},
		{EscalationBlock, "block"},
		{EscalationFailed, "failed"},
		{EscalationLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("EscalationLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEscalationLevel_ShouldAutoComplete(t *testing.T) {
	tests := []struct {
		level    EscalationLevel
		expected bool
	}{
		{EscalationNone, true},
		{EscalationInfo, true},
		{EscalationReview, true},
		{EscalationBlock, false},
		{EscalationFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.ShouldAutoComplete(); got != tt.expected {
				t.Errorf("EscalationLevel.ShouldAutoComplete() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEscalationLevel_RequiresHumanReview(t *testing.T) {
	tests := []struct {
		level    EscalationLevel
		expected bool
	}{
		{EscalationNone, false},
		{EscalationInfo, false},
		{EscalationReview, true},
		{EscalationBlock, true},
		{EscalationFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.RequiresHumanReview(); got != tt.expected {
				t.Errorf("EscalationLevel.RequiresHumanReview() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsKnowledgeProducingSkill(t *testing.T) {
	tests := []struct {
		skillName string
		expected  bool
	}{
		{"investigation", true},
		{"architect", true},
		{"research", true},
		{"design-session", true},
		{"codebase-audit", true},
		{"issue-creation", true},
		{"INVESTIGATION", true}, // case insensitive
		{"Architect", true},
		{"feature-impl", false},
		{"systematic-debugging", false},
		{"reliability-testing", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.skillName, func(t *testing.T) {
			if got := IsKnowledgeProducingSkill(tt.skillName); got != tt.expected {
				t.Errorf("IsKnowledgeProducingSkill(%q) = %v, want %v", tt.skillName, got, tt.expected)
			}
		})
	}
}

func TestDetermineEscalation_VerificationFailed(t *testing.T) {
	// Rule 1: Verification failed -> EscalationFailed
	input := EscalationInput{
		VerificationPassed: false,
		VerificationErrors: []string{"Phase not complete"},
	}

	got := DetermineEscalation(input)
	if got != EscalationFailed {
		t.Errorf("DetermineEscalation() = %v, want EscalationFailed", got)
	}
}

func TestDetermineEscalation_KnowledgeProducingSkillWithRecommendations(t *testing.T) {
	// Rule 2: Knowledge skill + recommendations -> EscalationReview
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "investigation",
		NextActions:        []string{"Create follow-up issue"},
	}

	got := DetermineEscalation(input)
	if got != EscalationReview {
		t.Errorf("DetermineEscalation() = %v, want EscalationReview", got)
	}
}

func TestDetermineEscalation_KnowledgeProducingSkillWithoutRecommendations(t *testing.T) {
	// Rule 2: Knowledge skill without recommendations -> EscalationInfo
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "investigation",
		Recommendation:     "close",
	}

	got := DetermineEscalation(input)
	if got != EscalationInfo {
		t.Errorf("DetermineEscalation() = %v, want EscalationInfo", got)
	}
}

func TestDetermineEscalation_VisualApprovalNeeded(t *testing.T) {
	// Rule 3: Visual verification needs approval -> EscalationBlock
	input := EscalationInput{
		VerificationPassed:  true,
		SkillName:           "feature-impl",
		HasWebChanges:       true,
		HasVisualEvidence:   true,
		NeedsVisualApproval: true,
	}

	got := DetermineEscalation(input)
	if got != EscalationBlock {
		t.Errorf("DetermineEscalation() = %v, want EscalationBlock", got)
	}
}

func TestDetermineEscalation_NonSuccessOutcome(t *testing.T) {
	// Rule 4: Non-success outcome -> EscalationReview
	tests := []struct {
		outcome string
	}{
		{"partial"},
		{"blocked"},
		{"failed"},
	}

	for _, tt := range tests {
		t.Run(tt.outcome, func(t *testing.T) {
			input := EscalationInput{
				VerificationPassed: true,
				SkillName:          "feature-impl",
				Outcome:            tt.outcome,
			}

			got := DetermineEscalation(input)
			if got != EscalationReview {
				t.Errorf("DetermineEscalation(outcome=%q) = %v, want EscalationReview", tt.outcome, got)
			}
		})
	}
}

func TestDetermineEscalation_RecommendationsWithLargeScope(t *testing.T) {
	// Rule 5: Recommendations + large scope -> EscalationReview
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "spawn-follow-up",
		FileCount:          15, // > 10
	}

	got := DetermineEscalation(input)
	if got != EscalationReview {
		t.Errorf("DetermineEscalation() = %v, want EscalationReview", got)
	}
}

func TestDetermineEscalation_RecommendationsWithNormalScope(t *testing.T) {
	// Rule 5: Recommendations + normal scope -> EscalationInfo
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		NextActions:        []string{"Consider adding more tests"},
		FileCount:          5, // < 10
	}

	got := DetermineEscalation(input)
	if got != EscalationInfo {
		t.Errorf("DetermineEscalation() = %v, want EscalationInfo", got)
	}
}

func TestDetermineEscalation_LargeScopeWithoutRecommendations(t *testing.T) {
	// Rule 6: Large scope without recommendations -> EscalationInfo
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "close",
		FileCount:          12, // > 10
	}

	got := DetermineEscalation(input)
	if got != EscalationInfo {
		t.Errorf("DetermineEscalation() = %v, want EscalationInfo", got)
	}
}

func TestDetermineEscalation_CleanCompletion(t *testing.T) {
	// Rule 7: Clean completion -> EscalationNone
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "close",
		FileCount:          3, // < 10
	}

	got := DetermineEscalation(input)
	if got != EscalationNone {
		t.Errorf("DetermineEscalation() = %v, want EscalationNone", got)
	}
}

func TestDetermineEscalation_EmptyInput(t *testing.T) {
	// Empty input with verification passed should be clean
	input := EscalationInput{
		VerificationPassed: true,
	}

	got := DetermineEscalation(input)
	if got != EscalationNone {
		t.Errorf("DetermineEscalation() = %v, want EscalationNone", got)
	}
}

func TestDetermineEscalation_RecommendationTypes(t *testing.T) {
	// Test different recommendation types trigger escalation
	recommendationsNeedingReview := []string{
		"spawn-follow-up",
		"escalate",
		"resume",
		"continue",
		"SPAWN-FOLLOW-UP", // case insensitive
	}

	for _, rec := range recommendationsNeedingReview {
		t.Run(rec, func(t *testing.T) {
			input := EscalationInput{
				VerificationPassed: true,
				SkillName:          "feature-impl",
				Recommendation:     rec,
				FileCount:          5,
			}

			got := DetermineEscalation(input)
			if got != EscalationInfo {
				t.Errorf("DetermineEscalation(recommendation=%q) = %v, want EscalationInfo", rec, got)
			}
		})
	}
}

func TestDetermineEscalation_CloseRecommendation(t *testing.T) {
	// "close" recommendation should not trigger escalation by itself
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Recommendation:     "close",
		FileCount:          5,
	}

	got := DetermineEscalation(input)
	if got != EscalationNone {
		t.Errorf("DetermineEscalation() = %v, want EscalationNone", got)
	}
}

func TestExplainEscalation_Failed(t *testing.T) {
	input := EscalationInput{
		VerificationPassed: false,
		VerificationErrors: []string{"Phase not complete", "SYNTHESIS.md missing"},
	}

	reason := ExplainEscalation(input)
	if reason.Level != EscalationFailed {
		t.Errorf("ExplainEscalation().Level = %v, want EscalationFailed", reason.Level)
	}
	if reason.Reason != "Verification failed" {
		t.Errorf("ExplainEscalation().Reason = %q, want 'Verification failed'", reason.Reason)
	}
	if len(reason.Details) != 2 {
		t.Errorf("ExplainEscalation().Details length = %d, want 2", len(reason.Details))
	}
	if reason.CanOverride {
		t.Error("ExplainEscalation().CanOverride = true, want false for Failed")
	}
}

func TestExplainEscalation_Block(t *testing.T) {
	input := EscalationInput{
		VerificationPassed:  true,
		NeedsVisualApproval: true,
	}

	reason := ExplainEscalation(input)
	if reason.Level != EscalationBlock {
		t.Errorf("ExplainEscalation().Level = %v, want EscalationBlock", reason.Level)
	}
	if !reason.CanOverride {
		t.Error("ExplainEscalation().CanOverride = false, want true for Block")
	}
}

func TestExplainEscalation_KnowledgeProducing(t *testing.T) {
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "investigation",
		NextActions:        []string{"Create beads issue for follow-up"},
	}

	reason := ExplainEscalation(input)
	if reason.Level != EscalationReview {
		t.Errorf("ExplainEscalation().Level = %v, want EscalationReview", reason.Level)
	}
	if reason.Reason != "Knowledge-producing skill with recommendations" {
		t.Errorf("ExplainEscalation().Reason = %q, unexpected", reason.Reason)
	}
}

func TestExplainEscalation_CleanCompletion(t *testing.T) {
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "close",
	}

	reason := ExplainEscalation(input)
	if reason.Level != EscalationNone {
		t.Errorf("ExplainEscalation().Level = %v, want EscalationNone", reason.Level)
	}
	if reason.Reason != "Clean completion" {
		t.Errorf("ExplainEscalation().Reason = %q, want 'Clean completion'", reason.Reason)
	}
}

func TestDetermineEscalation_HyphenatedRecommendationFromParser(t *testing.T) {
	// End-to-end: parsed "spawn-follow-up" from synthesis should trigger escalation.
	// This is the bug reproduction: \w+ regex truncated "spawn-follow-up" to "spawn",
	// so hasSignificantRecommendations never matched.
	input := EscalationInput{
		VerificationPassed: true,
		SkillName:          "feature-impl",
		Outcome:            "success",
		Recommendation:     "spawn-follow-up",
		FileCount:          5,
	}

	got := DetermineEscalation(input)
	if got != EscalationInfo {
		t.Errorf("DetermineEscalation(recommendation='spawn-follow-up') = %v, want EscalationInfo", got)
	}

	// Verify hasSignificantRecommendations returns true
	if !hasSignificantRecommendations(input) {
		t.Error("hasSignificantRecommendations() = false for 'spawn-follow-up', want true")
	}
}

// Test the decision tree order of precedence
func TestDetermineEscalation_Precedence(t *testing.T) {
	// Verification failure should override everything
	input := EscalationInput{
		VerificationPassed:  false,
		SkillName:           "investigation", // Would be Review
		NeedsVisualApproval: true,            // Would be Block
	}
	if got := DetermineEscalation(input); got != EscalationFailed {
		t.Errorf("Failed should override knowledge skill and visual: got %v", got)
	}

	// Knowledge skill should override visual approval check for non-feature-impl
	input = EscalationInput{
		VerificationPassed:  true,
		SkillName:           "investigation",
		NeedsVisualApproval: false, // Investigation wouldn't need this anyway
		NextActions:         []string{"follow up"},
	}
	if got := DetermineEscalation(input); got != EscalationReview {
		t.Errorf("Knowledge skill with recommendations should be Review: got %v", got)
	}
}
