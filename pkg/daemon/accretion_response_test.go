package daemon

import (
	"fmt"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestRunPeriodicAccretionResponse_NotDue(t *testing.T) {
	d := &Daemon{
		Scheduler: NewPeriodicScheduler(),
	}
	result := d.RunPeriodicAccretionResponse()
	if result != nil {
		t.Errorf("expected nil when not due, got %+v", result)
	}
}

func TestRunPeriodicAccretionResponse_NoService(t *testing.T) {
	d := &Daemon{
		Scheduler: NewPeriodicScheduler(),
	}
	d.Scheduler.Register(TaskAccretionResponse, true, 1)

	result := d.RunPeriodicAccretionResponse()
	if result == nil {
		t.Fatal("expected non-nil result when service not configured")
	}
	if result.Error == nil {
		t.Error("expected error when service not configured")
	}
}

func TestRunPeriodicAccretionResponse_NoEvents(t *testing.T) {
	svc := &mockAccretionResponseService{
		readEventsFunc: func() ([]events.AccretionDeltaData, error) {
			return nil, nil
		},
	}
	d := &Daemon{
		Scheduler:          NewPeriodicScheduler(),
		AccretionResponse:  svc,
	}
	d.Scheduler.Register(TaskAccretionResponse, true, 1)

	result := d.RunPeriodicAccretionResponse()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Created != 0 {
		t.Errorf("expected 0 created, got %d", result.Created)
	}
}

func TestRunPeriodicAccretionResponse_BelowThreshold(t *testing.T) {
	// File grew 100 lines across 2 events — below both thresholds (200 lines, 3 events)
	svc := &mockAccretionResponseService{
		readEventsFunc: func() ([]events.AccretionDeltaData, error) {
			return []events.AccretionDeltaData{
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/main.go", NetDelta: 50}}},
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/main.go", NetDelta: 50}}},
			}, nil
		},
	}
	d := &Daemon{
		Scheduler:         NewPeriodicScheduler(),
		AccretionResponse: svc,
	}
	d.Scheduler.Register(TaskAccretionResponse, true, 1)

	result := d.RunPeriodicAccretionResponse()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created != 0 {
		t.Errorf("expected 0 created (below threshold), got %d", result.Created)
	}
}

func TestRunPeriodicAccretionResponse_AboveThresholdButTooFewEvents(t *testing.T) {
	// File grew 300 lines but only across 2 events — needs >=3
	svc := &mockAccretionResponseService{
		readEventsFunc: func() ([]events.AccretionDeltaData, error) {
			return []events.AccretionDeltaData{
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/main.go", NetDelta: 150}}},
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/main.go", NetDelta: 150}}},
			}, nil
		},
	}
	d := &Daemon{
		Scheduler:         NewPeriodicScheduler(),
		AccretionResponse: svc,
	}
	d.Scheduler.Register(TaskAccretionResponse, true, 1)

	result := d.RunPeriodicAccretionResponse()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created != 0 {
		t.Errorf("expected 0 created (too few events), got %d", result.Created)
	}
}

func TestRunPeriodicAccretionResponse_CreatesIssue(t *testing.T) {
	// File grew 250 lines across 3 events — triggers issue creation
	svc := &mockAccretionResponseService{
		readEventsFunc: func() ([]events.AccretionDeltaData, error) {
			return []events.AccretionDeltaData{
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/big_file.go", NetDelta: 80}}},
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/big_file.go", NetDelta: 90}}},
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/big_file.go", NetDelta: 80}}},
			}, nil
		},
		hasOpenIssueFunc: func(filePath string) (bool, error) {
			return false, nil
		},
		createIssueFunc: func(filePath string, netDelta, eventCount int) (string, error) {
			return "orch-go-abc12", nil
		},
	}
	d := &Daemon{
		Scheduler:         NewPeriodicScheduler(),
		AccretionResponse: svc,
	}
	d.Scheduler.Register(TaskAccretionResponse, true, 1)

	result := d.RunPeriodicAccretionResponse()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created != 1 {
		t.Errorf("expected 1 created, got %d", result.Created)
	}
	if len(result.CreatedIssues) != 1 || result.CreatedIssues[0] != "orch-go-abc12" {
		t.Errorf("expected created issue orch-go-abc12, got %v", result.CreatedIssues)
	}
}

func TestRunPeriodicAccretionResponse_SkipsDuplicates(t *testing.T) {
	svc := &mockAccretionResponseService{
		readEventsFunc: func() ([]events.AccretionDeltaData, error) {
			return []events.AccretionDeltaData{
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/big_file.go", NetDelta: 100}}},
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/big_file.go", NetDelta: 100}}},
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/big_file.go", NetDelta: 100}}},
			}, nil
		},
		hasOpenIssueFunc: func(filePath string) (bool, error) {
			return true, nil // Already has open issue
		},
	}
	d := &Daemon{
		Scheduler:         NewPeriodicScheduler(),
		AccretionResponse: svc,
	}
	d.Scheduler.Register(TaskAccretionResponse, true, 1)

	result := d.RunPeriodicAccretionResponse()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created != 0 {
		t.Errorf("expected 0 created (dedup), got %d", result.Created)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
}

func TestRunPeriodicAccretionResponse_MultipleFiles(t *testing.T) {
	// Two files cross threshold, one doesn't
	var created []string
	svc := &mockAccretionResponseService{
		readEventsFunc: func() ([]events.AccretionDeltaData, error) {
			return []events.AccretionDeltaData{
				{FileDeltas: []events.FileDelta{
					{Path: "cmd/orch/a.go", NetDelta: 80},
					{Path: "cmd/orch/b.go", NetDelta: 80},
					{Path: "cmd/orch/c.go", NetDelta: 10},
				}},
				{FileDeltas: []events.FileDelta{
					{Path: "cmd/orch/a.go", NetDelta: 80},
					{Path: "cmd/orch/b.go", NetDelta: 80},
					{Path: "cmd/orch/c.go", NetDelta: 10},
				}},
				{FileDeltas: []events.FileDelta{
					{Path: "cmd/orch/a.go", NetDelta: 80},
					{Path: "cmd/orch/b.go", NetDelta: 80},
					{Path: "cmd/orch/c.go", NetDelta: 10},
				}},
			}, nil
		},
		hasOpenIssueFunc: func(filePath string) (bool, error) {
			return false, nil
		},
		createIssueFunc: func(filePath string, netDelta, eventCount int) (string, error) {
			created = append(created, filePath)
			return "orch-go-" + filePath, nil
		},
	}
	d := &Daemon{
		Scheduler:         NewPeriodicScheduler(),
		AccretionResponse: svc,
	}
	d.Scheduler.Register(TaskAccretionResponse, true, 1)

	result := d.RunPeriodicAccretionResponse()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created != 2 {
		t.Errorf("expected 2 created, got %d", result.Created)
	}
	if len(created) != 2 {
		t.Errorf("expected 2 files created, got %v", created)
	}
}

func TestRunPeriodicAccretionResponse_NegativeDeltasOffset(t *testing.T) {
	// File grew 300 in one event but shrank 150 in another — net 150 across 3 events, below 200
	svc := &mockAccretionResponseService{
		readEventsFunc: func() ([]events.AccretionDeltaData, error) {
			return []events.AccretionDeltaData{
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/main.go", NetDelta: 300}}},
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/main.go", NetDelta: -150}}},
				{FileDeltas: []events.FileDelta{{Path: "cmd/orch/main.go", NetDelta: 0}}},
			}, nil
		},
	}
	d := &Daemon{
		Scheduler:         NewPeriodicScheduler(),
		AccretionResponse: svc,
	}
	d.Scheduler.Register(TaskAccretionResponse, true, 1)

	result := d.RunPeriodicAccretionResponse()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created != 0 {
		t.Errorf("expected 0 created (net delta below threshold after shrinkage), got %d", result.Created)
	}
}

func TestRunPeriodicAccretionResponse_ReadEventsError(t *testing.T) {
	svc := &mockAccretionResponseService{
		readEventsFunc: func() ([]events.AccretionDeltaData, error) {
			return nil, fmt.Errorf("read failed")
		},
	}
	d := &Daemon{
		Scheduler:         NewPeriodicScheduler(),
		AccretionResponse: svc,
	}
	d.Scheduler.Register(TaskAccretionResponse, true, 1)

	result := d.RunPeriodicAccretionResponse()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == nil {
		t.Error("expected error when read fails")
	}
}

// --- Mock ---

type mockAccretionResponseService struct {
	readEventsFunc   func() ([]events.AccretionDeltaData, error)
	hasOpenIssueFunc func(filePath string) (bool, error)
	createIssueFunc  func(filePath string, netDelta, eventCount int) (string, error)
}

func (m *mockAccretionResponseService) ReadRecentAccretionDeltas() ([]events.AccretionDeltaData, error) {
	if m.readEventsFunc != nil {
		return m.readEventsFunc()
	}
	return nil, nil
}

func (m *mockAccretionResponseService) HasOpenExtractionIssue(filePath string) (bool, error) {
	if m.hasOpenIssueFunc != nil {
		return m.hasOpenIssueFunc(filePath)
	}
	return false, nil
}

func (m *mockAccretionResponseService) CreateAccretionIssue(filePath string, netDelta, eventCount int) (string, error) {
	if m.createIssueFunc != nil {
		return m.createIssueFunc(filePath, netDelta, eventCount)
	}
	return "mock-id", nil
}
