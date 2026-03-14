package daemon

import (
	"fmt"
	"testing"
	"time"
)

// --- Recurring Bugs Detector Tests ---

type mockRecurringBugsSource struct {
	listFunc    func(minReworks int) ([]ReworkedIssue, error)
	hasOpenFunc func(issueID string) (bool, error)
}

func (m *mockRecurringBugsSource) ListClosedIssuesWithRework(minReworks int) ([]ReworkedIssue, error) {
	if m.listFunc != nil {
		return m.listFunc(minReworks)
	}
	return nil, nil
}

func (m *mockRecurringBugsSource) HasOpenIssue(issueID string) (bool, error) {
	if m.hasOpenFunc != nil {
		return m.hasOpenFunc(issueID)
	}
	return false, nil
}

func TestRecurringBugsDetector_NoSource(t *testing.T) {
	d := &RecurringBugsDetector{}
	_, err := d.Detect()
	if err == nil {
		t.Error("expected error for nil source")
	}
}

func TestRecurringBugsDetector_NoReworks(t *testing.T) {
	d := &RecurringBugsDetector{
		Source: &mockRecurringBugsSource{
			listFunc: func(minReworks int) ([]ReworkedIssue, error) {
				return nil, nil
			},
		},
	}
	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("got %d suggestions, want 0", len(suggestions))
	}
}

func TestRecurringBugsDetector_FindsReworkedIssues(t *testing.T) {
	d := &RecurringBugsDetector{
		Source: &mockRecurringBugsSource{
			listFunc: func(minReworks int) ([]ReworkedIssue, error) {
				return []ReworkedIssue{
					{ID: "orch-go-abc12", Title: "Fix spawn race", ReworkCount: 3},
				}, nil
			},
			hasOpenFunc: func(issueID string) (bool, error) {
				return false, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	if suggestions[0].Detector != "recurring_bugs" {
		t.Errorf("Detector = %q, want recurring_bugs", suggestions[0].Detector)
	}
	if suggestions[0].Key != "orch-go-abc12" {
		t.Errorf("Key = %q, want orch-go-abc12", suggestions[0].Key)
	}
	if suggestions[0].IssueType != "bug" {
		t.Errorf("IssueType = %q, want bug", suggestions[0].IssueType)
	}
	if suggestions[0].Priority != 2 {
		t.Errorf("Priority = %d, want 2", suggestions[0].Priority)
	}
}

func TestRecurringBugsDetector_SkipsExistingIssue(t *testing.T) {
	d := &RecurringBugsDetector{
		Source: &mockRecurringBugsSource{
			listFunc: func(minReworks int) ([]ReworkedIssue, error) {
				return []ReworkedIssue{
					{ID: "orch-go-abc12", Title: "Fix spawn race", ReworkCount: 3},
				}, nil
			},
			hasOpenFunc: func(issueID string) (bool, error) {
				return true, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("got %d suggestions, want 0 (already tracked)", len(suggestions))
	}
}

func TestRecurringBugsDetector_Name(t *testing.T) {
	d := &RecurringBugsDetector{}
	if d.Name() != "recurring_bugs" {
		t.Errorf("Name() = %q, want recurring_bugs", d.Name())
	}
}

// --- Investigation Orphans Detector Tests ---

type mockInvestigationOrphansSource struct {
	listFunc    func() ([]OrphanedInvestigation, error)
	hasOpenFunc func(slug string) (bool, error)
}

func (m *mockInvestigationOrphansSource) ListActiveInvestigations() ([]OrphanedInvestigation, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return nil, nil
}

func (m *mockInvestigationOrphansSource) HasOpenIssueForInvestigation(slug string) (bool, error) {
	if m.hasOpenFunc != nil {
		return m.hasOpenFunc(slug)
	}
	return false, nil
}

func TestInvestigationOrphansDetector_NoSource(t *testing.T) {
	d := &InvestigationOrphansDetector{}
	_, err := d.Detect()
	if err == nil {
		t.Error("expected error for nil source")
	}
}

func TestInvestigationOrphansDetector_NoOrphans(t *testing.T) {
	d := &InvestigationOrphansDetector{
		Source: &mockInvestigationOrphansSource{
			listFunc: func() ([]OrphanedInvestigation, error) {
				return nil, nil
			},
		},
	}
	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("got %d suggestions, want 0", len(suggestions))
	}
}

func TestInvestigationOrphansDetector_FindsOrphans(t *testing.T) {
	d := &InvestigationOrphansDetector{
		Source: &mockInvestigationOrphansSource{
			listFunc: func() ([]OrphanedInvestigation, error) {
				return []OrphanedInvestigation{
					{Path: ".kb/investigations/2026-03-01-stale-thing.md", Slug: "stale-thing", Age: 10 * 24 * time.Hour},
				}, nil
			},
			hasOpenFunc: func(slug string) (bool, error) {
				return false, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	if suggestions[0].Detector != "investigation_orphans" {
		t.Errorf("Detector = %q", suggestions[0].Detector)
	}
	if suggestions[0].Key != "stale-thing" {
		t.Errorf("Key = %q", suggestions[0].Key)
	}
	if suggestions[0].IssueType != "task" {
		t.Errorf("IssueType = %q, want task", suggestions[0].IssueType)
	}
}

func TestInvestigationOrphansDetector_SkipsRecent(t *testing.T) {
	d := &InvestigationOrphansDetector{
		Source: &mockInvestigationOrphansSource{
			listFunc: func() ([]OrphanedInvestigation, error) {
				return []OrphanedInvestigation{
					{Path: ".kb/investigations/2026-03-13-recent.md", Slug: "recent", Age: 1 * 24 * time.Hour},
				}, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("got %d suggestions, want 0 (too recent)", len(suggestions))
	}
}

func TestInvestigationOrphansDetector_SkipsTracked(t *testing.T) {
	d := &InvestigationOrphansDetector{
		Source: &mockInvestigationOrphansSource{
			listFunc: func() ([]OrphanedInvestigation, error) {
				return []OrphanedInvestigation{
					{Path: ".kb/investigations/2026-03-01-tracked.md", Slug: "tracked", Age: 10 * 24 * time.Hour},
				}, nil
			},
			hasOpenFunc: func(slug string) (bool, error) {
				return true, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("got %d suggestions, want 0 (already tracked)", len(suggestions))
	}
}

func TestInvestigationOrphansDetector_Name(t *testing.T) {
	d := &InvestigationOrphansDetector{}
	if d.Name() != "investigation_orphans" {
		t.Errorf("Name() = %q", d.Name())
	}
}

// --- Thread Staleness Detector Tests ---

type mockThreadStalenessSource struct {
	listFunc func() ([]StaleThread, error)
}

func (m *mockThreadStalenessSource) ListOpenThreads() ([]StaleThread, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return nil, nil
}

func TestThreadStalenessDetector_NoSource(t *testing.T) {
	d := &ThreadStalenessDetector{}
	_, err := d.Detect()
	if err == nil {
		t.Error("expected error for nil source")
	}
}

func TestThreadStalenessDetector_NoStaleThreads(t *testing.T) {
	d := &ThreadStalenessDetector{
		Source: &mockThreadStalenessSource{
			listFunc: func() ([]StaleThread, error) {
				return nil, nil
			},
		},
	}
	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("got %d suggestions, want 0", len(suggestions))
	}
}

func TestThreadStalenessDetector_FindsStaleThreads(t *testing.T) {
	d := &ThreadStalenessDetector{
		Source: &mockThreadStalenessSource{
			listFunc: func() ([]StaleThread, error) {
				return []StaleThread{
					{Slug: "old-thread", Title: "Old Discussion", Age: 14 * 24 * time.Hour},
				}, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	if suggestions[0].Detector != "thread_staleness" {
		t.Errorf("Detector = %q", suggestions[0].Detector)
	}
	if suggestions[0].Key != "old-thread" {
		t.Errorf("Key = %q", suggestions[0].Key)
	}
	if suggestions[0].Priority != 4 {
		t.Errorf("Priority = %d, want 4", suggestions[0].Priority)
	}
}

func TestThreadStalenessDetector_SkipsRecent(t *testing.T) {
	d := &ThreadStalenessDetector{
		Source: &mockThreadStalenessSource{
			listFunc: func() ([]StaleThread, error) {
				return []StaleThread{
					{Slug: "recent-thread", Title: "Recent Discussion", Age: 2 * 24 * time.Hour},
				}, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("got %d suggestions, want 0 (too recent)", len(suggestions))
	}
}

func TestThreadStalenessDetector_MultipleThreads(t *testing.T) {
	d := &ThreadStalenessDetector{
		Source: &mockThreadStalenessSource{
			listFunc: func() ([]StaleThread, error) {
				return []StaleThread{
					{Slug: "stale-1", Title: "Thread A", Age: 10 * 24 * time.Hour},
					{Slug: "recent", Title: "Thread B", Age: 2 * 24 * time.Hour},
					{Slug: "stale-2", Title: "Thread C", Age: 30 * 24 * time.Hour},
				}, nil
			},
		},
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 2 {
		t.Errorf("got %d suggestions, want 2 (skip recent)", len(suggestions))
	}
}

func TestThreadStalenessDetector_CustomThreshold(t *testing.T) {
	d := &ThreadStalenessDetector{
		Source: &mockThreadStalenessSource{
			listFunc: func() ([]StaleThread, error) {
				return []StaleThread{
					{Slug: "thread-1", Title: "Thread A", Age: 5 * 24 * time.Hour},
				}, nil
			},
		},
		MaxStale: 3 * 24 * time.Hour,
	}

	suggestions, err := d.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Errorf("got %d suggestions, want 1 (5d > 3d threshold)", len(suggestions))
	}
}

func TestThreadStalenessDetector_Name(t *testing.T) {
	d := &ThreadStalenessDetector{}
	if d.Name() != "thread_staleness" {
		t.Errorf("Name() = %q", d.Name())
	}
}

// --- Integration: Multiple Detectors in TriggerScan ---

func TestDaemon_RunPeriodicTriggerScan_WithRealDetectors(t *testing.T) {
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
				return fmt.Sprintf("orch-go-t%d", createCount), nil
			},
		},
	}

	detectors := []PatternDetector{
		&RecurringBugsDetector{
			Source: &mockRecurringBugsSource{
				listFunc: func(minReworks int) ([]ReworkedIssue, error) {
					return []ReworkedIssue{
						{ID: "orch-go-bug1", Title: "Flaky test", ReworkCount: 3},
					}, nil
				},
				hasOpenFunc: func(id string) (bool, error) { return false, nil },
			},
		},
		&ThreadStalenessDetector{
			Source: &mockThreadStalenessSource{
				listFunc: func() ([]StaleThread, error) {
					return []StaleThread{
						{Slug: "old-thread", Title: "Stale Discussion", Age: 14 * 24 * time.Hour},
					}, nil
				},
			},
		},
		&InvestigationOrphansDetector{
			Source: &mockInvestigationOrphansSource{
				listFunc: func() ([]OrphanedInvestigation, error) {
					return []OrphanedInvestigation{
						{Slug: "orphan-inv", Path: ".kb/investigations/orphan.md", Age: 5 * 24 * time.Hour},
					}, nil
				},
				hasOpenFunc: func(slug string) (bool, error) { return false, nil },
			},
		},
	}

	result := d.RunPeriodicTriggerScan(detectors)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Detected != 3 {
		t.Errorf("Detected = %d, want 3", result.Detected)
	}
	if result.Created != 3 {
		t.Errorf("Created = %d, want 3", result.Created)
	}
}
