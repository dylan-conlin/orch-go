package orient

import (
	"testing"
)

func TestComputeDivergence_NoAlert_SmallGap(t *testing.T) {
	input := DivergenceInput{
		CompletionRate:        0.85,
		SessionOrphans:        0,
		SessionInvestigations: 5,
		StaleDecisions:        1,
		TotalDecisions:        40,
		Days:                  7,
	}

	alerts := ComputeDivergence(input)

	for _, a := range alerts {
		if a.Type == "merge_gap" {
			t.Errorf("expected no merge_gap alert for 5%% gap, got %v", a)
		}
	}
}

func TestComputeDivergence_HighSessionOrphans(t *testing.T) {
	// High completion rate but many session orphans → producing work that doesn't connect
	input := DivergenceInput{
		CompletionRate:        0.90,
		SessionOrphans:        4,
		SessionInvestigations: 6,
		StaleDecisions:        1,
		TotalDecisions:        40,
		Days:                  7,
	}

	alerts := ComputeDivergence(input)

	found := false
	for _, a := range alerts {
		if a.Type == "session_orphans" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected session_orphans alert for 4 unlinked, got %v", alerts)
	}
}

func TestComputeDivergence_StaleDecisions(t *testing.T) {
	input := DivergenceInput{
		CompletionRate:        0.90,
		SessionOrphans:        0,
		SessionInvestigations: 3,
		StaleDecisions:        8,
		TotalDecisions:        20,
		Days:                  7,
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
	input := DivergenceInput{
		CompletionRate:         0.90,
		SelfReportedCompletion: 0.95,
		ReworkRate:             0.30,
		SessionOrphans:         0,
		SessionInvestigations:  3,
		StaleDecisions:         1,
		TotalDecisions:         40,
		Days:                   7,
	}

	alerts := ComputeDivergence(input)

	found := false
	for _, a := range alerts {
		if a.Type == "rework_gap" {
			found = true
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
	input := DivergenceInput{}

	alerts := ComputeDivergence(input)

	if len(alerts) != 0 {
		t.Errorf("expected no alerts for zero data, got %v", alerts)
	}
}

func TestComputeDivergence_ZeroCompletions(t *testing.T) {
	input := DivergenceInput{
		CompletionRate:        0,
		SessionOrphans:        3,
		SessionInvestigations: 5,
		StaleDecisions:        5,
		TotalDecisions:        10,
		Days:                  7,
	}

	alerts := ComputeDivergence(input)

	for _, a := range alerts {
		if a.Type == "rework_gap" {
			t.Errorf("expected no rate-based alert with zero completions, got %v", a)
		}
	}
}

func TestComputeDivergence_SessionOrphans_BelowThreshold(t *testing.T) {
	input := DivergenceInput{
		SessionOrphans:        1,
		SessionInvestigations: 3,
		Days:                  7,
	}

	alerts := ComputeDivergence(input)

	for _, a := range alerts {
		if a.Type == "session_orphans" {
			t.Errorf("expected no session_orphans alert for 1 orphan (below threshold), got %v", a)
		}
	}
}

func TestFormatDivergenceAlerts(t *testing.T) {
	alerts := []DivergenceAlert{
		{
			Type:    "session_orphans",
			Message: "3 unlinked investigations this session — work not connecting to knowledge base",
			Gap:     0.60,
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
	if !contains(output, "session_orphans") {
		t.Errorf("expected session_orphans alert type in output, got %q", output)
	}
}

func TestFormatDivergenceAlerts_Empty(t *testing.T) {
	output := FormatDivergenceAlerts(nil)
	if output != "" {
		t.Errorf("expected empty output for nil alerts, got %q", output)
	}
}

// contains is defined in debrief_test.go
