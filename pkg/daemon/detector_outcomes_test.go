package daemon

import (
	"testing"
)

func TestComputeDetectorOutcomes_Empty(t *testing.T) {
	svc := &mockDetectorOutcomeService{
		issues: nil,
	}

	outcomes := ComputeDetectorOutcomes(svc)
	if len(outcomes) != 0 {
		t.Errorf("expected 0 outcomes, got %d", len(outcomes))
	}
}

func TestComputeDetectorOutcomes_SingleDetector(t *testing.T) {
	svc := &mockDetectorOutcomeService{
		issues: []DetectorIssue{
			{ID: "orch-go-001", Detector: "hotspot_acceleration", Status: "closed", Outcome: "completed"},
			{ID: "orch-go-002", Detector: "hotspot_acceleration", Status: "closed", Outcome: "completed"},
			{ID: "orch-go-003", Detector: "hotspot_acceleration", Status: "closed", Outcome: "abandoned"},
			{ID: "orch-go-004", Detector: "hotspot_acceleration", Status: "open", Outcome: ""},
		},
	}

	outcomes := ComputeDetectorOutcomes(svc)
	if len(outcomes) != 1 {
		t.Fatalf("expected 1 outcome, got %d", len(outcomes))
	}

	o := outcomes["hotspot_acceleration"]
	if o.Detector != "hotspot_acceleration" {
		t.Errorf("Detector = %q, want %q", o.Detector, "hotspot_acceleration")
	}
	if o.IssuesCreated != 4 {
		t.Errorf("IssuesCreated = %d, want 4", o.IssuesCreated)
	}
	if o.Completed != 2 {
		t.Errorf("Completed = %d, want 2", o.Completed)
	}
	if o.Abandoned != 1 {
		t.Errorf("Abandoned = %d, want 1", o.Abandoned)
	}
	// ResolutionRate = completed / (completed + abandoned) = 2/3 ≈ 0.667
	if o.ResolutionRate < 0.66 || o.ResolutionRate > 0.68 {
		t.Errorf("ResolutionRate = %f, want ~0.667", o.ResolutionRate)
	}
}

func TestComputeDetectorOutcomes_MultipleDetectors(t *testing.T) {
	svc := &mockDetectorOutcomeService{
		issues: []DetectorIssue{
			{ID: "orch-go-001", Detector: "hotspot_acceleration", Status: "closed", Outcome: "abandoned"},
			{ID: "orch-go-002", Detector: "hotspot_acceleration", Status: "closed", Outcome: "abandoned"},
			{ID: "orch-go-003", Detector: "knowledge_decay", Status: "closed", Outcome: "completed"},
		},
	}

	outcomes := ComputeDetectorOutcomes(svc)
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %d", len(outcomes))
	}

	hotspot := outcomes["hotspot_acceleration"]
	if hotspot.ResolutionRate != 0 {
		t.Errorf("hotspot ResolutionRate = %f, want 0 (all abandoned)", hotspot.ResolutionRate)
	}

	decay := outcomes["knowledge_decay"]
	if decay.ResolutionRate != 1.0 {
		t.Errorf("decay ResolutionRate = %f, want 1.0 (all completed)", decay.ResolutionRate)
	}
}

func TestDetectorBudgetAdjustment_HighPerformance(t *testing.T) {
	outcomes := map[string]*DetectorOutcome{
		"knowledge_decay": {Detector: "knowledge_decay", ResolutionRate: 0.8, IssuesCreated: 10, Completed: 8, Abandoned: 2},
	}

	adjusted := AdjustedBudget(10, "knowledge_decay", outcomes)
	// High-performing detector keeps full budget
	if adjusted != 10 {
		t.Errorf("budget = %d, want 10 (high performing detector)", adjusted)
	}
}

func TestDetectorBudgetAdjustment_LowPerformance(t *testing.T) {
	outcomes := map[string]*DetectorOutcome{
		"hotspot_acceleration": {Detector: "hotspot_acceleration", ResolutionRate: 0.2, IssuesCreated: 20, Completed: 4, Abandoned: 16},
	}

	adjusted := AdjustedBudget(10, "hotspot_acceleration", outcomes)
	// ResolutionRate < 0.3 → budget halved
	if adjusted != 5 {
		t.Errorf("budget = %d, want 5 (low performance → halved)", adjusted)
	}
}

func TestDetectorBudgetAdjustment_VeryLowPerformance(t *testing.T) {
	outcomes := map[string]*DetectorOutcome{
		"hotspot_acceleration": {Detector: "hotspot_acceleration", ResolutionRate: 0.05, IssuesCreated: 40, Completed: 2, Abandoned: 38},
	}

	adjusted := AdjustedBudget(10, "hotspot_acceleration", outcomes)
	// ResolutionRate < 0.1 → disabled (budget = 0)
	if adjusted != 0 {
		t.Errorf("budget = %d, want 0 (very low performance → disabled)", adjusted)
	}
}

func TestDetectorBudgetAdjustment_UnknownDetector(t *testing.T) {
	outcomes := map[string]*DetectorOutcome{}

	adjusted := AdjustedBudget(10, "new_detector", outcomes)
	// Unknown detector keeps full budget
	if adjusted != 10 {
		t.Errorf("budget = %d, want 10 (unknown detector keeps full budget)", adjusted)
	}
}

func TestDetectorBudgetAdjustment_InsufficientSamples(t *testing.T) {
	outcomes := map[string]*DetectorOutcome{
		"hotspot_acceleration": {Detector: "hotspot_acceleration", ResolutionRate: 0.0, IssuesCreated: 2, Completed: 0, Abandoned: 2},
	}

	adjusted := AdjustedBudget(10, "hotspot_acceleration", outcomes)
	// Only 2 resolved issues — too few samples to penalize. Keep full budget.
	if adjusted != 10 {
		t.Errorf("budget = %d, want 10 (insufficient samples → no penalty)", adjusted)
	}
}

// --- Mocks ---

type mockDetectorOutcomeService struct {
	issues []DetectorIssue
}

func (m *mockDetectorOutcomeService) ListDetectorIssues() ([]DetectorIssue, error) {
	return m.issues, nil
}
