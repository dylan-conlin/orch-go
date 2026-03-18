package daemon

import (
	"fmt"
	"testing"
	"time"
)

// mockTriggerScanService implements TriggerScanService for tests.
type mockTriggerScanService struct {
	CountOpenFunc   func() (int, error)
	HasOpenFunc     func(detectorName, key string) (bool, error)
	CreateIssueFunc func(s TriggerSuggestion) (string, error)
}

func (m *mockTriggerScanService) CountOpenTriggerIssues() (int, error) {
	if m.CountOpenFunc != nil {
		return m.CountOpenFunc()
	}
	return 0, nil
}

func (m *mockTriggerScanService) HasOpenTriggerIssue(detectorName, key string) (bool, error) {
	if m.HasOpenFunc != nil {
		return m.HasOpenFunc(detectorName, key)
	}
	return false, nil
}

func (m *mockTriggerScanService) CreateTriggerIssue(s TriggerSuggestion) (string, error) {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(s)
	}
	return "test-trigger-001", nil
}

// mockPatternDetector is a test detector that returns configured suggestions.
type mockPatternDetector struct {
	name       string
	detectFunc func() ([]TriggerSuggestion, error)
}

func (m *mockPatternDetector) Name() string { return m.name }
func (m *mockPatternDetector) Detect() ([]TriggerSuggestion, error) {
	if m.detectFunc != nil {
		return m.detectFunc()
	}
	return nil, nil
}

func TestDaemon_RunPeriodicTriggerScan_NotDue(t *testing.T) {
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
	}
	d := &Daemon{
		Config:      cfg,
		Scheduler:   NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{},
	}
	d.Scheduler.SetLastRun(TaskTriggerScan, time.Now())

	result := d.RunPeriodicTriggerScan(nil)
	if result != nil {
		t.Error("RunPeriodicTriggerScan() should return nil when not due")
	}
}

func TestDaemon_RunPeriodicTriggerScan_ServiceNotConfigured(t *testing.T) {
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	result := d.RunPeriodicTriggerScan(nil)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Error == nil {
		t.Error("expected error for unconfigured service")
	}
}

func TestDaemon_RunPeriodicTriggerScan_NoDetectors(t *testing.T) {
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
	}
	d := &Daemon{
		Config:      cfg,
		Scheduler:   NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{},
	}

	result := d.RunPeriodicTriggerScan(nil)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Detected != 0 {
		t.Errorf("Detected = %d, want 0", result.Detected)
	}
	if result.Message != "Trigger scan: no patterns detected" {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestDaemon_RunPeriodicTriggerScan_CreatesIssue(t *testing.T) {
	createCalled := 0
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
		TriggerBudgetMax:    10,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{
			CountOpenFunc: func() (int, error) { return 0, nil },
			HasOpenFunc:   func(_, _ string) (bool, error) { return false, nil },
			CreateIssueFunc: func(s TriggerSuggestion) (string, error) {
				createCalled++
				return fmt.Sprintf("orch-go-trig%d", createCalled), nil
			},
		},
	}

	detectors := []PatternDetector{
		&mockPatternDetector{
			name: "test_detector",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return []TriggerSuggestion{
					{Detector: "test_detector", Key: "key-1", Title: "Test issue", IssueType: "task", Priority: 3},
				}, nil
			},
		},
	}

	result := d.RunPeriodicTriggerScan(detectors)
	if result == nil {
		t.Fatal("expected result")
	}
	if createCalled != 1 {
		t.Errorf("CreateTriggerIssue called %d times, want 1", createCalled)
	}
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1", result.Created)
	}
	if len(result.CreatedIssues) != 1 {
		t.Errorf("CreatedIssues = %v, want 1 item", result.CreatedIssues)
	}
}

func TestDaemon_RunPeriodicTriggerScan_BudgetEnforced(t *testing.T) {
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
		TriggerBudgetMax:    5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{
			CountOpenFunc: func() (int, error) { return 5, nil },
			HasOpenFunc:   func(_, _ string) (bool, error) { return false, nil },
		},
	}

	detectors := []PatternDetector{
		&mockPatternDetector{
			name: "test_detector",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return []TriggerSuggestion{
					{Detector: "test_detector", Key: "k1", Title: "Issue 1", IssueType: "task", Priority: 3},
				}, nil
			},
		},
	}

	result := d.RunPeriodicTriggerScan(detectors)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Created != 0 {
		t.Errorf("Created = %d, want 0 (budget exhausted)", result.Created)
	}
	if result.SkippedBudget != 1 {
		t.Errorf("SkippedBudget = %d, want 1", result.SkippedBudget)
	}
}

func TestDaemon_RunPeriodicTriggerScan_DedupSkips(t *testing.T) {
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
		TriggerBudgetMax:    10,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{
			CountOpenFunc: func() (int, error) { return 0, nil },
			HasOpenFunc: func(detector, key string) (bool, error) {
				return key == "existing-key", nil
			},
		},
	}

	detectors := []PatternDetector{
		&mockPatternDetector{
			name: "test_detector",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return []TriggerSuggestion{
					{Detector: "test_detector", Key: "existing-key", Title: "Already exists", IssueType: "task", Priority: 3},
				}, nil
			},
		},
	}

	result := d.RunPeriodicTriggerScan(detectors)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Created != 0 {
		t.Errorf("Created = %d, want 0", result.Created)
	}
	if result.SkippedDedup != 1 {
		t.Errorf("SkippedDedup = %d, want 1", result.SkippedDedup)
	}
}

func TestDaemon_RunPeriodicTriggerScan_MultipleDetectors(t *testing.T) {
	createCount := 0
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
		TriggerBudgetMax:    10,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{
			CountOpenFunc: func() (int, error) { return 0, nil },
			HasOpenFunc:   func(_, _ string) (bool, error) { return false, nil },
			CreateIssueFunc: func(s TriggerSuggestion) (string, error) {
				createCount++
				return fmt.Sprintf("orch-go-trig%d", createCount), nil
			},
		},
	}

	detectors := []PatternDetector{
		&mockPatternDetector{
			name: "detector_a",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return []TriggerSuggestion{
					{Detector: "detector_a", Key: "a1", Title: "Issue A", IssueType: "task", Priority: 3},
				}, nil
			},
		},
		&mockPatternDetector{
			name: "detector_b",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return []TriggerSuggestion{
					{Detector: "detector_b", Key: "b1", Title: "Issue B", IssueType: "bug", Priority: 2},
					{Detector: "detector_b", Key: "b2", Title: "Issue C", IssueType: "task", Priority: 3},
				}, nil
			},
		},
	}

	result := d.RunPeriodicTriggerScan(detectors)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Created != 3 {
		t.Errorf("Created = %d, want 3", result.Created)
	}
	if result.Detected != 3 {
		t.Errorf("Detected = %d, want 3", result.Detected)
	}
}

func TestDaemon_RunPeriodicTriggerScan_BudgetDecrements(t *testing.T) {
	createCount := 0
	openCount := 0
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
		TriggerBudgetMax:    2,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{
			CountOpenFunc: func() (int, error) { return openCount, nil },
			HasOpenFunc:   func(_, _ string) (bool, error) { return false, nil },
			CreateIssueFunc: func(s TriggerSuggestion) (string, error) {
				createCount++
				openCount++
				return fmt.Sprintf("orch-go-trig%d", createCount), nil
			},
		},
	}

	detectors := []PatternDetector{
		&mockPatternDetector{
			name: "greedy_detector",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return []TriggerSuggestion{
					{Detector: "greedy_detector", Key: "g1", Title: "Issue 1", IssueType: "task", Priority: 3},
					{Detector: "greedy_detector", Key: "g2", Title: "Issue 2", IssueType: "task", Priority: 3},
					{Detector: "greedy_detector", Key: "g3", Title: "Issue 3", IssueType: "task", Priority: 3},
				}, nil
			},
		},
	}

	result := d.RunPeriodicTriggerScan(detectors)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Created != 2 {
		t.Errorf("Created = %d, want 2 (budget limit)", result.Created)
	}
	if result.SkippedBudget != 1 {
		t.Errorf("SkippedBudget = %d, want 1", result.SkippedBudget)
	}
}

func TestDaemon_RunPeriodicTriggerScan_DetectorError(t *testing.T) {
	createCount := 0
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
		TriggerBudgetMax:    10,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{
			CountOpenFunc: func() (int, error) { return 0, nil },
			HasOpenFunc:   func(_, _ string) (bool, error) { return false, nil },
			CreateIssueFunc: func(s TriggerSuggestion) (string, error) {
				createCount++
				return "orch-go-ok", nil
			},
		},
	}

	detectors := []PatternDetector{
		&mockPatternDetector{
			name: "broken_detector",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return nil, fmt.Errorf("detector failed")
			},
		},
		&mockPatternDetector{
			name: "working_detector",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return []TriggerSuggestion{
					{Detector: "working_detector", Key: "w1", Title: "Good issue", IssueType: "task", Priority: 3},
				}, nil
			},
		},
	}

	result := d.RunPeriodicTriggerScan(detectors)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1 (broken detector skipped)", result.Created)
	}
	if result.DetectorErrors != 1 {
		t.Errorf("DetectorErrors = %d, want 1", result.DetectorErrors)
	}
}

func TestDaemon_RunPeriodicTriggerScan_UpdatesScheduler(t *testing.T) {
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
		TriggerBudgetMax:    10,
	}
	d := &Daemon{
		Config:      cfg,
		Scheduler:   NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{
			CountOpenFunc: func() (int, error) { return 0, nil },
		},
	}

	before := d.Scheduler.LastRunTime(TaskTriggerScan)
	if !before.IsZero() {
		t.Fatal("expected zero LastRunTime before first run")
	}

	d.RunPeriodicTriggerScan(nil)

	after := d.Scheduler.LastRunTime(TaskTriggerScan)
	if after.IsZero() {
		t.Error("expected non-zero LastRunTime after run")
	}
}

func TestDaemon_RunPeriodicTriggerScan_OutcomeBudgetAdjustment(t *testing.T) {
	// A detector with very low resolution rate (<10%) should be disabled.
	// A detector with low resolution rate (<30%) should have budget halved.
	// A detector with good resolution rate should keep full budget.
	createCount := 0
	cfg := Config{
		TriggerScanEnabled:  true,
		TriggerScanInterval: time.Hour,
		TriggerBudgetMax:    10,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		TriggerScan: &mockTriggerScanService{
			CountOpenFunc: func() (int, error) { return 0, nil },
			HasOpenFunc:   func(_, _ string) (bool, error) { return false, nil },
			CreateIssueFunc: func(s TriggerSuggestion) (string, error) {
				createCount++
				return fmt.Sprintf("orch-go-trig%d", createCount), nil
			},
		},
		// Provide outcome data: bad_detector has 5% resolution rate (disabled),
		// weak_detector has 20% (halved), good_detector has 80% (full budget).
		DetectorOutcomes: &mockDetectorOutcomeService{
			issues: []DetectorIssue{
				// bad_detector: 1 completed, 19 abandoned → 5% resolution rate
				{ID: "o-01", Detector: "bad_detector", Status: "closed", Outcome: "completed"},
				{ID: "o-02", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-03", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-04", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-05", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-06", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-07", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-08", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-09", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-10", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-11", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-12", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-13", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-14", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-15", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-16", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-17", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-18", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-19", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-20", Detector: "bad_detector", Status: "closed", Outcome: "abandoned"},
				// good_detector: 8 completed, 2 abandoned → 80% resolution rate
				{ID: "o-21", Detector: "good_detector", Status: "closed", Outcome: "completed"},
				{ID: "o-22", Detector: "good_detector", Status: "closed", Outcome: "completed"},
				{ID: "o-23", Detector: "good_detector", Status: "closed", Outcome: "completed"},
				{ID: "o-24", Detector: "good_detector", Status: "closed", Outcome: "completed"},
				{ID: "o-25", Detector: "good_detector", Status: "closed", Outcome: "completed"},
				{ID: "o-26", Detector: "good_detector", Status: "closed", Outcome: "completed"},
				{ID: "o-27", Detector: "good_detector", Status: "closed", Outcome: "completed"},
				{ID: "o-28", Detector: "good_detector", Status: "closed", Outcome: "completed"},
				{ID: "o-29", Detector: "good_detector", Status: "closed", Outcome: "abandoned"},
				{ID: "o-30", Detector: "good_detector", Status: "closed", Outcome: "abandoned"},
			},
		},
	}

	detectors := []PatternDetector{
		&mockPatternDetector{
			name: "bad_detector",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return []TriggerSuggestion{
					{Detector: "bad_detector", Key: "b1", Title: "Bad issue", IssueType: "task", Priority: 3},
				}, nil
			},
		},
		&mockPatternDetector{
			name: "good_detector",
			detectFunc: func() ([]TriggerSuggestion, error) {
				return []TriggerSuggestion{
					{Detector: "good_detector", Key: "g1", Title: "Good issue", IssueType: "task", Priority: 3},
				}, nil
			},
		},
	}

	result := d.RunPeriodicTriggerScan(detectors)
	if result == nil {
		t.Fatal("expected result")
	}
	// bad_detector (5% resolution) → disabled (budget=0), skipped
	// good_detector (80% resolution) → full budget, created
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1 (bad detector disabled)", result.Created)
	}
	if result.SkippedBudget != 1 {
		t.Errorf("SkippedBudget = %d, want 1 (bad detector budget=0)", result.SkippedBudget)
	}
}

func TestTriggerBudget_CanCreate(t *testing.T) {
	tests := []struct {
		max         int
		currentOpen int
		want        bool
	}{
		{10, 0, true},
		{10, 5, true},
		{10, 9, true},
		{10, 10, false},
		{10, 15, false},
		{0, 0, false},
	}

	for _, tt := range tests {
		b := TriggerBudget{MaxOpen: tt.max}
		got := b.CanCreate(tt.currentOpen)
		if got != tt.want {
			t.Errorf("TriggerBudget{Max:%d}.CanCreate(%d) = %v, want %v", tt.max, tt.currentOpen, got, tt.want)
		}
	}
}

func TestDefaultConfig_IncludesTriggerScan(t *testing.T) {
	config := DefaultConfig()

	if !config.TriggerScanEnabled {
		t.Error("DefaultConfig().TriggerScanEnabled should be true")
	}
	if config.TriggerScanInterval != time.Hour {
		t.Errorf("DefaultConfig().TriggerScanInterval = %v, want 1h", config.TriggerScanInterval)
	}
	if config.TriggerBudgetMax != 10 {
		t.Errorf("DefaultConfig().TriggerBudgetMax = %d, want 10", config.TriggerBudgetMax)
	}
}
