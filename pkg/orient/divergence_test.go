package orient

import (
	"testing"
)

func TestComputeDivergence_NoAlert_SmallGap(t *testing.T) {
	// 85% completion, 80% merge → 5% gap → no alert
	input := DivergenceInput{
		CompletionRate: 0.85,
		OrphanRate:     10.0,
		StaleDecisions: 1,
		TotalDecisions: 40,
		Days:           7,
	}

	alerts := ComputeDivergence(input)

	for _, a := range alerts {
		if a.Type == "merge_gap" {
			t.Errorf("expected no merge_gap alert for 5%% gap, got %v", a)
		}
	}
}

func TestComputeDivergence_HighOrphanRate(t *testing.T) {
	// High completion rate but high orphan rate → producing work that doesn't connect
	input := DivergenceInput{
		CompletionRate: 0.90,
		OrphanRate:     45.0, // 45% orphan rate
		StaleDecisions: 1,
		TotalDecisions: 40,
		Days:           7,
	}

	alerts := ComputeDivergence(input)

	found := false
	for _, a := range alerts {
		if a.Type == "orphan_rate" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected orphan_rate alert for 45%% orphan rate, got %v", alerts)
	}
}

func TestComputeDivergence_StaleDecisions(t *testing.T) {
	// High completion but many stale decisions → busy but not acting on decisions
	input := DivergenceInput{
		CompletionRate: 0.90,
		OrphanRate:     10.0,
		StaleDecisions: 8,
		TotalDecisions: 20,
		Days:           7,
	}

	alerts := ComputeDivergence(input)

	found := false
	for _, a := range alerts {
		if a.Type == "stale_decisions" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected stale_decisions alert for 40%% stale rate, got %v", alerts)
	}
}

func TestComputeDivergence_ReworkGap(t *testing.T) {
	// High self-reported success but high rework rate → declaring success on work that gets redone
	input := DivergenceInput{
		CompletionRate:    0.90,
		SelfReportedSuccess: 0.95,
		ReworkRate:        0.30,
		OrphanRate:        10.0,
		StaleDecisions:    1,
		TotalDecisions:    40,
		Days:              7,
	}

	alerts := ComputeDivergence(input)

	found := false
	for _, a := range alerts {
		if a.Type == "rework_gap" {
			found = true
			// Expected: |0.95 - (1 - 0.30)| = |0.95 - 0.70| = 0.25
			if a.Gap < 0.20 {
				t.Errorf("expected rework gap >= 0.20, got %.2f", a.Gap)
			}
		}
	}
	if !found {
		t.Errorf("expected rework_gap alert, got %v", alerts)
	}
}

func TestComputeDivergence_Empty(t *testing.T) {
	// Zero data → no alerts (fail-open)
	input := DivergenceInput{}

	alerts := ComputeDivergence(input)

	if len(alerts) != 0 {
		t.Errorf("expected no alerts for zero data, got %v", alerts)
	}
}

func TestComputeDivergence_ZeroCompletions(t *testing.T) {
	// No completions → skip rate-based alerts
	input := DivergenceInput{
		CompletionRate: 0,
		OrphanRate:     50.0, // high but no completions, still alert
		StaleDecisions: 5,
		TotalDecisions: 10,
		Days:           7,
	}

	alerts := ComputeDivergence(input)

	// Should still alert on orphan rate and stale decisions (impact metrics)
	// but not on merge_gap or rework_gap (need completions for rate comparison)
	for _, a := range alerts {
		if a.Type == "rework_gap" {
			t.Errorf("expected no rate-based alert with zero completions, got %v", a)
		}
	}
}

func TestFormatDivergenceAlerts(t *testing.T) {
	alerts := []DivergenceAlert{
		{
			Type:    "orphan_rate",
			Message: "45% investigation orphan rate — work not connecting to knowledge base",
			Gap:     0.45,
			Level:   "warning",
		},
	}

	output := FormatDivergenceAlerts(alerts)

	if output == "" {
		t.Error("expected non-empty output")
	}
	if !contains(output, "Metric divergence") {
		t.Errorf("expected header, got %q", output)
	}
	if !contains(output, "orphan_rate") {
		t.Errorf("expected orphan_rate alert type in output, got %q", output)
	}
}

func TestFormatDivergenceAlerts_Empty(t *testing.T) {
	output := FormatDivergenceAlerts(nil)
	if output != "" {
		t.Errorf("expected empty output for nil alerts, got %q", output)
	}
}

// contains is defined in debrief_test.go
