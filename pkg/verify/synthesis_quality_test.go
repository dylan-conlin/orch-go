package verify

import (
	"testing"
)

func TestComputeSynthesisQuality_FullSynthesis(t *testing.T) {
	s := &Synthesis{
		TLDR:      "Discovered that connective reasoning matters because it separates insight from reporting.",
		Delta:     "Modified pkg/verify/synthesis_quality.go, added 6 quality signals.",
		Evidence:  "Tests PASS: go test ./pkg/verify/... — 12 passed, 0 failed.",
		Knowledge: "This confirms the knowledge accretion model's claim about .kb/models/ connection. The key insight is that structural metadata enables selection pressure without LLM judgment.",
		Next:      "**Recommendation:** close",
		UnexploredQuestions: "How does signal_count correlate with brief feedback ratings?\nWhat threshold separates useful from noisy signals?",
	}

	q := ComputeSynthesisQuality(s)

	if q.Total != 6 {
		t.Errorf("Total should be 6, got %d", q.Total)
	}

	// All 6 signals should fire on this well-formed synthesis
	if q.SignalCount != 6 {
		t.Errorf("Expected 6 signals, got %d", q.SignalCount)
		for _, sig := range q.Signals {
			t.Logf("  %s: detected=%v score=%s", sig.Name, sig.Detected, sig.Score)
		}
	}
}

func TestComputeSynthesisQuality_EmptySynthesis(t *testing.T) {
	s := &Synthesis{}
	q := ComputeSynthesisQuality(s)

	if q.SignalCount != 0 {
		t.Errorf("Expected 0 signals for empty synthesis, got %d", q.SignalCount)
	}
	if q.Total != 6 {
		t.Errorf("Total should always be 6, got %d", q.Total)
	}
}

func TestComputeSynthesisQuality_StructuralCompleteness(t *testing.T) {
	tests := []struct {
		name  string
		synth Synthesis
		score string
	}{
		{"all 4 sections", Synthesis{TLDR: "x", Delta: "x", Evidence: "x", Knowledge: "x"}, "4/4"},
		{"3 sections", Synthesis{TLDR: "x", Delta: "x", Evidence: "x"}, "3/4"},
		{"1 section", Synthesis{TLDR: "x"}, "1/4"},
		{"0 sections", Synthesis{}, "0/4"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := ComputeSynthesisQuality(&tc.synth)
			var found bool
			for _, sig := range q.Signals {
				if sig.Name == "structural_completeness" {
					found = true
					if sig.Score != tc.score {
						t.Errorf("Expected score %s, got %s", tc.score, sig.Score)
					}
					// Detected when 3+ sections
					expectDetected := tc.score == "3/4" || tc.score == "4/4"
					if sig.Detected != expectDetected {
						t.Errorf("Expected detected=%v for score %s", expectDetected, tc.score)
					}
				}
			}
			if !found {
				t.Error("structural_completeness signal not found")
			}
		})
	}
}

func TestComputeSynthesisQuality_EvidenceSpecificity(t *testing.T) {
	tests := []struct {
		name     string
		synth    Synthesis
		detected bool
	}{
		{"file path in evidence", Synthesis{Evidence: "Modified pkg/verify/check.go"}, true},
		{"test output", Synthesis{Evidence: "Tests PASS with 12 assertions"}, true},
		{"go file ref in delta", Synthesis{Delta: "Created synthesis_quality.go"}, true},
		{"vague evidence", Synthesis{Evidence: "Everything worked fine"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := ComputeSynthesisQuality(&tc.synth)
			for _, sig := range q.Signals {
				if sig.Name == "evidence_specificity" {
					if sig.Detected != tc.detected {
						t.Errorf("Expected detected=%v, got %v", tc.detected, sig.Detected)
					}
					return
				}
			}
			t.Error("evidence_specificity signal not found")
		})
	}
}

func TestComputeSynthesisQuality_ModelConnection(t *testing.T) {
	tests := []struct {
		name     string
		synth    Synthesis
		detected bool
	}{
		{"kb models reference", Synthesis{Knowledge: "This extends .kb/models/knowledge-accretion"}, true},
		{"confirms language", Synthesis{Knowledge: "This confirms the prediction about signal quality"}, true},
		{"contradicts language", Synthesis{Knowledge: "This contradicts the assumption about completeness"}, true},
		{"no model connection", Synthesis{Knowledge: "We added a new feature"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := ComputeSynthesisQuality(&tc.synth)
			for _, sig := range q.Signals {
				if sig.Name == "model_connection" {
					if sig.Detected != tc.detected {
						t.Errorf("Expected detected=%v, got %v", tc.detected, sig.Detected)
					}
					return
				}
			}
			t.Error("model_connection signal not found")
		})
	}
}

func TestComputeSynthesisQuality_ConnectiveReasoning(t *testing.T) {
	tests := []struct {
		name     string
		synth    Synthesis
		detected bool
	}{
		{"because in knowledge", Synthesis{Knowledge: "This matters because it reduces cognitive load"}, true},
		{"the insight in TLDR", Synthesis{TLDR: "The insight is that signals enable selection"}, true},
		{"no connectives", Synthesis{Knowledge: "Added quality signals", TLDR: "New feature"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := ComputeSynthesisQuality(&tc.synth)
			for _, sig := range q.Signals {
				if sig.Name == "connective_reasoning" {
					if sig.Detected != tc.detected {
						t.Errorf("Expected detected=%v, got %v", tc.detected, sig.Detected)
					}
					return
				}
			}
			t.Error("connective_reasoning signal not found")
		})
	}
}

func TestComputeSynthesisQuality_TensionQuality(t *testing.T) {
	tests := []struct {
		name     string
		synth    Synthesis
		detected bool
	}{
		{"questions present", Synthesis{UnexploredQuestions: "How does X relate to Y?"}, true},
		{"no question marks", Synthesis{UnexploredQuestions: "Need to investigate more"}, false},
		{"empty", Synthesis{}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := ComputeSynthesisQuality(&tc.synth)
			for _, sig := range q.Signals {
				if sig.Name == "tension_quality" {
					if sig.Detected != tc.detected {
						t.Errorf("Expected detected=%v, got %v", tc.detected, sig.Detected)
					}
					return
				}
			}
			t.Error("tension_quality signal not found")
		})
	}
}

func TestComputeSynthesisQuality_InsightVsReport(t *testing.T) {
	tests := []struct {
		name     string
		synth    Synthesis
		detected bool
	}{
		{
			"insight lines dominate",
			Synthesis{Knowledge: "The key finding is that structural metadata enables selection.\nThis means agents can be ranked without LLM judgment.\nThe implication is reduced cognitive load."},
			true,
		},
		{
			"all action verb lines",
			Synthesis{Knowledge: "Added quality signals.\nFixed the ordering.\nImplemented frontmatter."},
			false,
		},
		{
			"empty knowledge",
			Synthesis{},
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := ComputeSynthesisQuality(&tc.synth)
			for _, sig := range q.Signals {
				if sig.Name == "insight_vs_report" {
					if sig.Detected != tc.detected {
						t.Errorf("Expected detected=%v, got %v", tc.detected, sig.Detected)
					}
					return
				}
			}
			t.Error("insight_vs_report signal not found")
		})
	}
}
