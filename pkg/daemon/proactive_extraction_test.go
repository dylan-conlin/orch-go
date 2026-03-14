package daemon

import (
	"fmt"
	"testing"
)

func TestRunPeriodicProactiveExtraction_NotDue(t *testing.T) {
	d := &Daemon{
		Scheduler: NewPeriodicScheduler(),
	}
	// Not registered = not due
	result := d.RunPeriodicProactiveExtraction()
	if result != nil {
		t.Errorf("expected nil when not due, got %+v", result)
	}
}

func TestRunPeriodicProactiveExtraction_NoService(t *testing.T) {
	d := &Daemon{
		Scheduler: NewPeriodicScheduler(),
		// ProactiveExtraction not set
	}
	d.Scheduler.Register(TaskProactiveExtraction, true, 0)
	// Force due by setting zero interval (IsDue returns true when lastRun is zero and enabled)
	// Actually need a positive interval for IsDue
	d.Scheduler.Register(TaskProactiveExtraction, true, 1)

	result := d.RunPeriodicProactiveExtraction()
	if result == nil {
		t.Fatal("expected non-nil result when service not configured")
	}
	if result.Error == nil {
		t.Error("expected error when service not configured")
	}
}

func TestRunPeriodicProactiveExtraction_NoFilesAboveThreshold(t *testing.T) {
	svc := &mockProactiveExtractionService{
		scanFunc: func(threshold int) ([]ProactiveExtractionFile, error) {
			return nil, nil // No files above threshold
		},
	}
	d := &Daemon{
		Scheduler:            NewPeriodicScheduler(),
		ProactiveExtraction:  svc,
	}
	d.Scheduler.Register(TaskProactiveExtraction, true, 1)

	result := d.RunPeriodicProactiveExtraction()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Created != 0 {
		t.Errorf("expected 0 created, got %d", result.Created)
	}
	if result.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestRunPeriodicProactiveExtraction_CreatesArchitectIssue(t *testing.T) {
	var createdTitle string
	svc := &mockProactiveExtractionService{
		scanFunc: func(threshold int) ([]ProactiveExtractionFile, error) {
			if threshold != 1200 {
				t.Errorf("expected threshold 1200, got %d", threshold)
			}
			return []ProactiveExtractionFile{
				{Path: "cmd/orch/daemon_loop.go", Lines: 1350},
			}, nil
		},
		hasOpenIssueFunc: func(filePath string) (bool, error) {
			return false, nil
		},
		createIssueFunc: func(filePath string, lines int) (string, error) {
			createdTitle = filePath
			return "orch-go-abc1", nil
		},
	}
	d := &Daemon{
		Scheduler:           NewPeriodicScheduler(),
		ProactiveExtraction: svc,
	}
	d.Scheduler.Register(TaskProactiveExtraction, true, 1)

	result := d.RunPeriodicProactiveExtraction()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Created != 1 {
		t.Errorf("expected 1 created, got %d", result.Created)
	}
	if len(result.CreatedIssues) != 1 || result.CreatedIssues[0] != "orch-go-abc1" {
		t.Errorf("expected created issue orch-go-abc1, got %v", result.CreatedIssues)
	}
	if createdTitle != "cmd/orch/daemon_loop.go" {
		t.Errorf("expected issue for cmd/orch/daemon_loop.go, got %s", createdTitle)
	}
}

func TestRunPeriodicProactiveExtraction_SkipsDuplicateIssues(t *testing.T) {
	svc := &mockProactiveExtractionService{
		scanFunc: func(threshold int) ([]ProactiveExtractionFile, error) {
			return []ProactiveExtractionFile{
				{Path: "cmd/orch/daemon_loop.go", Lines: 1350},
			}, nil
		},
		hasOpenIssueFunc: func(filePath string) (bool, error) {
			return true, nil // Already has an open issue
		},
	}
	d := &Daemon{
		Scheduler:           NewPeriodicScheduler(),
		ProactiveExtraction: svc,
	}
	d.Scheduler.Register(TaskProactiveExtraction, true, 1)

	result := d.RunPeriodicProactiveExtraction()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created != 0 {
		t.Errorf("expected 0 created (dedup), got %d", result.Created)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped (dedup), got %d", result.Skipped)
	}
}

func TestRunPeriodicProactiveExtraction_MultipleFiles(t *testing.T) {
	var created []string
	svc := &mockProactiveExtractionService{
		scanFunc: func(threshold int) ([]ProactiveExtractionFile, error) {
			return []ProactiveExtractionFile{
				{Path: "cmd/orch/daemon_loop.go", Lines: 1350},
				{Path: "cmd/orch/spawn_cmd.go", Lines: 1250},
				{Path: "pkg/daemon/ooda.go", Lines: 1400},
			}, nil
		},
		hasOpenIssueFunc: func(filePath string) (bool, error) {
			// Second file already has an issue
			if filePath == "cmd/orch/spawn_cmd.go" {
				return true, nil
			}
			return false, nil
		},
		createIssueFunc: func(filePath string, lines int) (string, error) {
			created = append(created, filePath)
			return fmt.Sprintf("orch-go-%d", len(created)), nil
		},
	}
	d := &Daemon{
		Scheduler:           NewPeriodicScheduler(),
		ProactiveExtraction: svc,
	}
	d.Scheduler.Register(TaskProactiveExtraction, true, 1)

	result := d.RunPeriodicProactiveExtraction()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created != 2 {
		t.Errorf("expected 2 created, got %d", result.Created)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
	if result.Scanned != 3 {
		t.Errorf("expected 3 scanned, got %d", result.Scanned)
	}
}

func TestRunPeriodicProactiveExtraction_ScanError(t *testing.T) {
	svc := &mockProactiveExtractionService{
		scanFunc: func(threshold int) ([]ProactiveExtractionFile, error) {
			return nil, fmt.Errorf("scan failed")
		},
	}
	d := &Daemon{
		Scheduler:           NewPeriodicScheduler(),
		ProactiveExtraction: svc,
	}
	d.Scheduler.Register(TaskProactiveExtraction, true, 1)

	result := d.RunPeriodicProactiveExtraction()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == nil {
		t.Error("expected error on scan failure")
	}
}

func TestRunPeriodicProactiveExtraction_SkipsFilesAbove1500(t *testing.T) {
	// Files >1500 lines are already handled by the CRITICAL extraction gate.
	// Proactive extraction should only trigger for files between 1200-1500.
	var created []string
	svc := &mockProactiveExtractionService{
		scanFunc: func(threshold int) ([]ProactiveExtractionFile, error) {
			return []ProactiveExtractionFile{
				{Path: "cmd/orch/big_file.go", Lines: 1600},   // >1500 = skip (handled by critical gate)
				{Path: "cmd/orch/medium_file.go", Lines: 1300}, // 1200-1500 = create issue
			}, nil
		},
		hasOpenIssueFunc: func(filePath string) (bool, error) {
			return false, nil
		},
		createIssueFunc: func(filePath string, lines int) (string, error) {
			created = append(created, filePath)
			return "orch-go-x1", nil
		},
	}
	d := &Daemon{
		Scheduler:           NewPeriodicScheduler(),
		ProactiveExtraction: svc,
	}
	d.Scheduler.Register(TaskProactiveExtraction, true, 1)

	result := d.RunPeriodicProactiveExtraction()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created != 1 {
		t.Errorf("expected 1 created (only 1200-1500 range), got %d", result.Created)
	}
	if len(created) != 1 || created[0] != "cmd/orch/medium_file.go" {
		t.Errorf("expected issue for medium_file.go only, got %v", created)
	}
	if result.SkippedCritical != 1 {
		t.Errorf("expected 1 skipped critical, got %d", result.SkippedCritical)
	}
}

// mockProactiveExtractionService implements ProactiveExtractionService for testing.
type mockProactiveExtractionService struct {
	scanFunc         func(threshold int) ([]ProactiveExtractionFile, error)
	hasOpenIssueFunc func(filePath string) (bool, error)
	createIssueFunc  func(filePath string, lines int) (string, error)
}

func (m *mockProactiveExtractionService) ScanFilesAboveThreshold(threshold int) ([]ProactiveExtractionFile, error) {
	if m.scanFunc != nil {
		return m.scanFunc(threshold)
	}
	return nil, nil
}

func (m *mockProactiveExtractionService) HasOpenExtractionIssue(filePath string) (bool, error) {
	if m.hasOpenIssueFunc != nil {
		return m.hasOpenIssueFunc(filePath)
	}
	return false, nil
}

func (m *mockProactiveExtractionService) CreateArchitectIssue(filePath string, lines int) (string, error) {
	if m.createIssueFunc != nil {
		return m.createIssueFunc(filePath, lines)
	}
	return "", nil
}
