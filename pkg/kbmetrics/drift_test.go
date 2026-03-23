package kbmetrics

import (
	"testing"
)

func TestClassifyTier(t *testing.T) {
	tests := []struct {
		annotation string
		want       EvidenceTier
	}{
		{"Observed (single experiment, N=10)", TierObserved},
		{"observed", TierObserved},
		{"Replicated (3 independent sources)", TierReplicated},
		{"replicated", TierReplicated},
		{"Validated (external + internal)", TierValidated},
		{"validated", TierValidated},
		{"Working hypothesis", TierHypothesis},
		{"working-hypothesis", TierHypothesis},
		{"Assumed (no direct measurement)", TierAssumed},
		{"assumed", TierAssumed},
		{"Single-source measured", TierObserved}, // legacy mapping
		{"Multi-source analytical", TierReplicated}, // legacy mapping
		{"something unknown", TierUnclassified},
	}

	for _, tt := range tests {
		got := ClassifyTier(tt.annotation)
		if got != tt.want {
			t.Errorf("ClassifyTier(%q) = %v, want %v", tt.annotation, got, tt.want)
		}
	}
}

func TestDetectDrift_OverclaimLanguage(t *testing.T) {
	// Claim marked "observed" but uses "fundamentally flawed" — should flag
	claims := []DriftInput{
		{
			ClaimText:  "Multi-agent frameworks that rely on messaging are fundamentally flawed.",
			Tier:       TierObserved,
			ClaimLine:  10,
		},
	}

	drifts := DetectDrift(claims)
	if len(drifts) == 0 {
		t.Fatal("expected drift for 'fundamentally flawed' at observed tier")
	}
	if drifts[0].Line != 10 {
		t.Errorf("Line = %d, want 10", drifts[0].Line)
	}
}

func TestDetectDrift_NoFlagForValidated(t *testing.T) {
	// Same strong language at "validated" tier — should NOT flag
	claims := []DriftInput{
		{
			ClaimText:  "Multi-agent frameworks that rely on messaging are fundamentally flawed.",
			Tier:       TierValidated,
			ClaimLine:  10,
		},
	}

	drifts := DetectDrift(claims)
	if len(drifts) != 0 {
		t.Errorf("expected no drift for validated tier, got %d", len(drifts))
	}
}

func TestDetectDrift_UniversalLanguage(t *testing.T) {
	claims := []DriftInput{
		{
			ClaimText:  "This pattern universally applies to all coordination systems.",
			Tier:       TierObserved,
			ClaimLine:  5,
		},
	}

	drifts := DetectDrift(claims)
	if len(drifts) == 0 {
		t.Fatal("expected drift for 'universally' at observed tier")
	}
}

func TestDetectDrift_ScopedLanguageOK(t *testing.T) {
	// Scoped language at observed tier — should NOT flag
	claims := []DriftInput{
		{
			ClaimText:  "In the tested same-file scenarios, messaging did not reduce conflicts.",
			Tier:       TierObserved,
			ClaimLine:  5,
		},
	}

	drifts := DetectDrift(claims)
	if len(drifts) != 0 {
		t.Errorf("expected no drift for scoped language, got %d", len(drifts))
	}
}

func TestDetectDrift_HypothesisWithAbsolute(t *testing.T) {
	claims := []DriftInput{
		{
			ClaimText:  "Communication never produces coordination outcomes.",
			Tier:       TierHypothesis,
			ClaimLine:  20,
		},
	}

	drifts := DetectDrift(claims)
	if len(drifts) == 0 {
		t.Fatal("expected drift for 'never' at hypothesis tier")
	}
}

func TestDetectDrift_ReplicatedWithModerateLanguage(t *testing.T) {
	// "Replicated" tier allows moderate-strength language
	claims := []DriftInput{
		{
			ClaimText:  "Communication consistently fails to produce coordination.",
			Tier:       TierReplicated,
			ClaimLine:  15,
		},
	}

	drifts := DetectDrift(claims)
	if len(drifts) != 0 {
		t.Errorf("expected no drift for 'consistently' at replicated tier, got %d", len(drifts))
	}
}

func TestDetectDrift_MultipleFlags(t *testing.T) {
	claims := []DriftInput{
		{
			ClaimText:  "This is always true and universally proven to be fundamentally correct.",
			Tier:       TierAssumed,
			ClaimLine:  1,
		},
	}

	drifts := DetectDrift(claims)
	if len(drifts) == 0 {
		t.Fatal("expected at least one drift")
	}
	// Should have multiple triggers
	if len(drifts[0].Triggers) < 2 {
		t.Errorf("expected multiple triggers, got %d", len(drifts[0].Triggers))
	}
}
