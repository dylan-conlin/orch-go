package daemon

import (
	"errors"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
)

type mockAuditSelectService struct {
	issues   []beads.Issue
	issueErr error
	forced   map[string]bool
	forceErr error
	labeled  []string
	labelErr error
}

func (m *mockAuditSelectService) RecentClosedIssues(window time.Duration) ([]beads.Issue, error) {
	return m.issues, m.issueErr
}

func (m *mockAuditSelectService) ForcedBeadsIDs() (map[string]bool, error) {
	return m.forced, m.forceErr
}

func (m *mockAuditSelectService) LabelForAudit(issueID string) error {
	if m.labelErr != nil {
		return m.labelErr
	}
	m.labeled = append(m.labeled, issueID)
	return nil
}

func TestRunPeriodicAuditSelect_NotDue(t *testing.T) {
	d := &Daemon{
		Scheduler: NewPeriodicScheduler(),
	}
	d.Scheduler.Register(TaskAuditSelect, true, 168*time.Hour)
	d.Scheduler.SetLastRun(TaskAuditSelect, time.Now()) // just ran

	result := d.RunPeriodicAuditSelect()
	if result != nil {
		t.Error("expected nil when not due")
	}
}

func TestRunPeriodicAuditSelect_NoIssues(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		AuditSelectEnabled:      true,
		AuditSelectInterval:     168 * time.Hour,
		AuditSelectCount:        2,
		AuditAutoCompleteWeight: 0.6,
	})
	d.AuditSelect = &mockAuditSelectService{}

	result := d.RunPeriodicAuditSelect()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Message != "no eligible issues for audit selection" {
		t.Errorf("Message = %q, want %q", result.Message, "no eligible issues for audit selection")
	}
}

func TestRunPeriodicAuditSelect_SelectsAndLabels(t *testing.T) {
	svc := &mockAuditSelectService{
		issues: []beads.Issue{
			{ID: "orch-go-001", Title: "Feature A"},
			{ID: "orch-go-002", Title: "Feature B"},
			{ID: "orch-go-003", Title: "Feature C"},
			{ID: "orch-go-004", Title: "Feature D"},
		},
		forced: map[string]bool{
			"orch-go-001": true,
			"orch-go-003": true,
		},
	}
	d := NewWithConfig(daemonconfig.Config{
		AuditSelectEnabled:      true,
		AuditSelectInterval:     168 * time.Hour,
		AuditSelectCount:        2,
		AuditAutoCompleteWeight: 0.6,
	})
	d.AuditSelect = svc

	result := d.RunPeriodicAuditSelect()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if len(result.Selected) != 2 {
		t.Errorf("Selected count = %d, want 2", len(result.Selected))
	}
	if len(svc.labeled) != 2 {
		t.Errorf("labeled count = %d, want 2", len(svc.labeled))
	}
}

func TestRunPeriodicAuditSelect_Error(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		AuditSelectEnabled:      true,
		AuditSelectInterval:     168 * time.Hour,
		AuditSelectCount:        2,
		AuditAutoCompleteWeight: 0.6,
	})
	d.AuditSelect = &mockAuditSelectService{
		issueErr: errors.New("beads unavailable"),
	}

	result := d.RunPeriodicAuditSelect()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == nil {
		t.Error("expected error")
	}
}

func TestRunPeriodicAuditSelect_MarksRun(t *testing.T) {
	d := NewWithConfig(daemonconfig.Config{
		AuditSelectEnabled:      true,
		AuditSelectInterval:     168 * time.Hour,
		AuditSelectCount:        2,
		AuditAutoCompleteWeight: 0.6,
	})
	d.AuditSelect = &mockAuditSelectService{}

	d.RunPeriodicAuditSelect()

	if d.Scheduler.LastRunTime(TaskAuditSelect).IsZero() {
		t.Error("expected LastRunTime to be set after running")
	}
}

func TestWeightedSelection_AllAutoCompleted(t *testing.T) {
	autoPool := []beads.Issue{
		{ID: "a1"}, {ID: "a2"}, {ID: "a3"},
	}
	result := weightedSelection(autoPool, nil, 2, 0.6)
	if len(result) != 2 {
		t.Errorf("got %d, want 2 (should fall back to auto pool)", len(result))
	}
}

func TestWeightedSelection_NoAutoCompleted(t *testing.T) {
	otherPool := []beads.Issue{
		{ID: "o1"}, {ID: "o2"}, {ID: "o3"},
	}
	result := weightedSelection(nil, otherPool, 2, 0.6)
	if len(result) != 2 {
		t.Errorf("got %d, want 2 (should fall back to other pool)", len(result))
	}
}

func TestWeightedSelection_SmallPools(t *testing.T) {
	autoPool := []beads.Issue{{ID: "a1"}}
	otherPool := []beads.Issue{{ID: "o1"}}
	result := weightedSelection(autoPool, otherPool, 5, 0.6)
	// Only 2 total available
	if len(result) != 2 {
		t.Errorf("got %d, want 2 (capped by pool sizes)", len(result))
	}
}

func TestWeightedSelection_WeightDistribution(t *testing.T) {
	// With 10 in each pool and selectCount=10, weight=0.6:
	// autoTarget=6, otherTarget=4
	autoPool := make([]beads.Issue, 10)
	otherPool := make([]beads.Issue, 10)
	for i := range autoPool {
		autoPool[i] = beads.Issue{ID: "a"}
	}
	for i := range otherPool {
		otherPool[i] = beads.Issue{ID: "o"}
	}

	result := weightedSelection(autoPool, otherPool, 10, 0.6)
	if len(result) != 10 {
		t.Fatalf("got %d, want 10", len(result))
	}

	autoCount := 0
	for _, r := range result {
		if r.ID == "a" {
			autoCount++
		}
	}
	if autoCount != 6 {
		t.Errorf("auto count = %d, want 6 (60%% weight)", autoCount)
	}
}

func TestRunAuditSelection_ForcedIDsErrorFallsBack(t *testing.T) {
	svc := &mockAuditSelectService{
		issues: []beads.Issue{
			{ID: "orch-go-001"}, {ID: "orch-go-002"},
		},
		forceErr: errors.New("events unreadable"),
	}

	result := runAuditSelection(svc, 2, 0.6, 168*time.Hour)
	if result.Error != nil {
		t.Errorf("forced ID error should be non-fatal, got: %v", result.Error)
	}
	if len(result.Selected) != 2 {
		t.Errorf("Selected = %d, want 2 (uniform fallback)", len(result.Selected))
	}
}

func TestRunAuditSelection_LabelFailure(t *testing.T) {
	svc := &mockAuditSelectService{
		issues: []beads.Issue{
			{ID: "orch-go-001"}, {ID: "orch-go-002"},
		},
		labelErr: errors.New("label failed"),
	}

	result := runAuditSelection(svc, 2, 0.6, 168*time.Hour)
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Message != "audit selection found candidates but failed to label any" {
		t.Errorf("Message = %q", result.Message)
	}
}
