package daemon

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/plan"
)

// mockPlanStatusQuerier returns pre-configured statuses for beads IDs.
type mockPlanStatusQuerier struct {
	statuses map[string]string
}

func (m *mockPlanStatusQuerier) QueryIssueStatus(id string) string {
	if s, ok := m.statuses[id]; ok {
		return s
	}
	return "unknown"
}

func TestDetectPlanStaleness_UnhydratedPlan(t *testing.T) {
	p := &plan.File{
		Title:    "Test Plan",
		Status:   "active",
		Filename: "2026-03-11-test-plan.md",
		Phases: []plan.Phase{
			{Name: "First", BeadsIDs: nil},
			{Name: "Second", BeadsIDs: nil},
		},
	}

	results := detectPlanStaleness(p, "test-plan", &mockPlanStatusQuerier{})

	if len(results) != 1 {
		t.Fatalf("expected 1 stale result, got %d", len(results))
	}
	if results[0].StalenessType != StalenessUnhydrated {
		t.Errorf("expected StalenessUnhydrated, got %s", results[0].StalenessType)
	}
	if results[0].Slug != "test-plan" {
		t.Errorf("expected slug 'test-plan', got %q", results[0].Slug)
	}
}

func TestDetectPlanStaleness_AdvancementStall(t *testing.T) {
	querier := &mockPlanStatusQuerier{
		statuses: map[string]string{
			"orch-go-p1": "closed",
			"orch-go-p2": "open",
		},
	}

	p := &plan.File{
		Title:    "Test Plan",
		Status:   "active",
		Filename: "2026-03-11-test-plan.md",
		Phases: []plan.Phase{
			{Name: "First", BeadsIDs: []string{"orch-go-p1"}, DependsOn: "Nothing"},
			{Name: "Second", BeadsIDs: []string{"orch-go-p2"}, DependsOn: "Phase 1"},
		},
	}

	results := detectPlanStaleness(p, "test-plan", querier)

	hasStall := false
	for _, r := range results {
		if r.StalenessType == StalenessAdvancementStall {
			hasStall = true
		}
	}
	if !hasStall {
		t.Error("expected advancement stall to be detected")
	}
}

func TestDetectPlanStaleness_NoAdvancementStallWhenInProgress(t *testing.T) {
	querier := &mockPlanStatusQuerier{
		statuses: map[string]string{
			"orch-go-p1": "closed",
			"orch-go-p2": "in_progress",
		},
	}

	p := &plan.File{
		Title:    "Test Plan",
		Status:   "active",
		Filename: "2026-03-11-test-plan.md",
		Phases: []plan.Phase{
			{Name: "First", BeadsIDs: []string{"orch-go-p1"}, DependsOn: "Nothing"},
			{Name: "Second", BeadsIDs: []string{"orch-go-p2"}, DependsOn: "Phase 1"},
		},
	}

	results := detectPlanStaleness(p, "test-plan", querier)

	for _, r := range results {
		if r.StalenessType == StalenessAdvancementStall {
			t.Error("should not detect advancement stall when next phase is in progress")
		}
	}
}

func TestDetectPlanStaleness_NoProgress(t *testing.T) {
	querier := &mockPlanStatusQuerier{
		statuses: map[string]string{
			"orch-go-p1": "open",
			"orch-go-p2": "open",
		},
	}

	p := &plan.File{
		Title:    "Test Plan",
		Status:   "active",
		Filename: "2026-03-11-test-plan.md",
		Phases: []plan.Phase{
			{Name: "First", BeadsIDs: []string{"orch-go-p1"}, DependsOn: "Nothing"},
			{Name: "Second", BeadsIDs: []string{"orch-go-p2"}, DependsOn: "Phase 1"},
		},
	}

	results := detectPlanStaleness(p, "test-plan", querier)

	hasNoProgress := false
	for _, r := range results {
		if r.StalenessType == StalenessNoProgress {
			hasNoProgress = true
		}
	}
	if !hasNoProgress {
		t.Error("expected no-progress staleness to be detected")
	}
}

func TestDetectPlanStaleness_HealthyPlan(t *testing.T) {
	querier := &mockPlanStatusQuerier{
		statuses: map[string]string{
			"orch-go-p1": "closed",
			"orch-go-p2": "in_progress",
			"orch-go-p3": "open",
		},
	}

	p := &plan.File{
		Title:    "Test Plan",
		Status:   "active",
		Filename: "2026-03-11-test-plan.md",
		Phases: []plan.Phase{
			{Name: "First", BeadsIDs: []string{"orch-go-p1"}, DependsOn: "Nothing"},
			{Name: "Second", BeadsIDs: []string{"orch-go-p2"}, DependsOn: "Phase 1"},
			{Name: "Third", BeadsIDs: []string{"orch-go-p3"}, DependsOn: "Phase 2"},
		},
	}

	results := detectPlanStaleness(p, "test-plan", querier)

	if len(results) != 0 {
		t.Errorf("expected no staleness for healthy plan, got %d results: %v", len(results), results)
	}
}

func TestDetectPlanStaleness_MultiPhaseAdvancementStall(t *testing.T) {
	// Phase 1 complete, Phase 2 complete, Phase 3 depends on Phase 2 but not started
	querier := &mockPlanStatusQuerier{
		statuses: map[string]string{
			"orch-go-p1": "closed",
			"orch-go-p2": "closed",
			"orch-go-p3": "open",
		},
	}

	p := &plan.File{
		Title:    "Multi-phase Plan",
		Status:   "active",
		Filename: "2026-03-11-multi.md",
		Phases: []plan.Phase{
			{Name: "First", BeadsIDs: []string{"orch-go-p1"}, DependsOn: "Nothing"},
			{Name: "Second", BeadsIDs: []string{"orch-go-p2"}, DependsOn: "Phase 1"},
			{Name: "Third", BeadsIDs: []string{"orch-go-p3"}, DependsOn: "Phase 2"},
		},
	}

	results := detectPlanStaleness(p, "multi", querier)

	stallCount := 0
	for _, r := range results {
		if r.StalenessType == StalenessAdvancementStall {
			stallCount++
		}
	}
	if stallCount != 1 {
		t.Errorf("expected 1 advancement stall (Phase 2→3), got %d", stallCount)
	}
}

func TestRunPeriodicPlanStaleness_Disabled(t *testing.T) {
	d := NewWithConfig(Config{PlanStalenessEnabled: false})
	result := d.RunPeriodicPlanStaleness()
	if result != nil {
		t.Error("expected nil result when disabled")
	}
}

func TestRunPeriodicPlanStaleness_NotDue(t *testing.T) {
	d := NewWithConfig(Config{PlanStalenessEnabled: true, PlanStalenessInterval: 0})
	result := d.RunPeriodicPlanStaleness()
	if result != nil {
		t.Error("expected nil result when interval is 0")
	}
}

func TestComputePhaseStatusFromQuerier(t *testing.T) {
	querier := &mockPlanStatusQuerier{
		statuses: map[string]string{
			"closed-1": "closed",
			"closed-2": "closed",
			"open-1":   "open",
			"wip-1":    "in_progress",
		},
	}

	tests := []struct {
		name     string
		beadsIDs []string
		want     string
	}{
		{"no issues", nil, "no-issues"},
		{"all closed", []string{"closed-1", "closed-2"}, "complete"},
		{"one open", []string{"open-1"}, "ready"},
		{"one in progress", []string{"wip-1"}, "in-progress"},
		{"mixed closed and open", []string{"closed-1", "open-1"}, "ready"},
		{"mixed closed and in progress", []string{"closed-1", "wip-1"}, "in-progress"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computePhaseStatusFromQuerier(tt.beadsIDs, querier)
			if got != tt.want {
				t.Errorf("computePhaseStatusFromQuerier(%v) = %q, want %q", tt.beadsIDs, got, tt.want)
			}
		})
	}
}
