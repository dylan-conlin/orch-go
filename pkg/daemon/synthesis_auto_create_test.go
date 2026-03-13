package daemon

import (
	"fmt"
	"testing"
	"time"
)

// mockSynthesisAutoCreateService implements SynthesisAutoCreateService for tests.
type mockSynthesisAutoCreateService struct {
	LoadFunc          func() ([]SynthesisSuggestion, error)
	ModelDirFunc      func(topic string) (bool, error)
	HasOpenIssueFunc  func(topic string) (bool, error)
	CreateIssueFunc   func(topic string, count int, investigations []string) (string, error)
}

func (m *mockSynthesisAutoCreateService) LoadSynthesisSuggestions() ([]SynthesisSuggestion, error) {
	if m.LoadFunc != nil {
		return m.LoadFunc()
	}
	return nil, nil
}

func (m *mockSynthesisAutoCreateService) ModelDirExists(topic string) (bool, error) {
	if m.ModelDirFunc != nil {
		return m.ModelDirFunc(topic)
	}
	return false, nil
}

func (m *mockSynthesisAutoCreateService) HasOpenSynthesisIssue(topic string) (bool, error) {
	if m.HasOpenIssueFunc != nil {
		return m.HasOpenIssueFunc(topic)
	}
	return false, nil
}

func (m *mockSynthesisAutoCreateService) CreateSynthesisIssue(topic string, count int, investigations []string) (string, error) {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(topic, count, investigations)
	}
	return "test-123", nil
}

func TestDaemon_RunPeriodicSynthesisAutoCreate_NotDue(t *testing.T) {
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		SynthesisAutoCreate: &mockSynthesisAutoCreateService{},
	}
	d.Scheduler.SetLastRun(TaskSynthesisAutoCreate, time.Now())

	result := d.RunPeriodicSynthesisAutoCreate()
	if result != nil {
		t.Error("RunPeriodicSynthesisAutoCreate() should return nil when not due")
	}
}

func TestDaemon_RunPeriodicSynthesisAutoCreate_NoSuggestions(t *testing.T) {
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		SynthesisAutoCreate: &mockSynthesisAutoCreateService{
			LoadFunc: func() ([]SynthesisSuggestion, error) {
				return nil, nil
			},
		},
	}

	result := d.RunPeriodicSynthesisAutoCreate()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Created != 0 {
		t.Errorf("Created = %d, want 0", result.Created)
	}
	if result.Message != "Synthesis auto-create: no clusters above threshold" {
		t.Errorf("Message = %q, want 'no clusters above threshold'", result.Message)
	}
}

func TestDaemon_RunPeriodicSynthesisAutoCreate_BelowThreshold(t *testing.T) {
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		SynthesisAutoCreate: &mockSynthesisAutoCreateService{
			LoadFunc: func() ([]SynthesisSuggestion, error) {
				return []SynthesisSuggestion{
					{Topic: "some-topic", Count: 3},
					{Topic: "another-topic", Count: 4},
				}, nil
			},
		},
	}

	result := d.RunPeriodicSynthesisAutoCreate()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Created != 0 {
		t.Errorf("Created = %d, want 0", result.Created)
	}
	if result.Evaluated != 0 {
		t.Errorf("Evaluated = %d, want 0 (below threshold)", result.Evaluated)
	}
}

func TestDaemon_RunPeriodicSynthesisAutoCreate_CreatesIssue(t *testing.T) {
	createCalled := false
	createTopic := ""
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		SynthesisAutoCreate: &mockSynthesisAutoCreateService{
			LoadFunc: func() ([]SynthesisSuggestion, error) {
				return []SynthesisSuggestion{
					{Topic: "spawn-architecture", Count: 7, Investigations: []string{"inv-1", "inv-2"}},
				}, nil
			},
			ModelDirFunc: func(topic string) (bool, error) {
				return false, nil // No model dir
			},
			HasOpenIssueFunc: func(topic string) (bool, error) {
				return false, nil // No existing issue
			},
			CreateIssueFunc: func(topic string, count int, investigations []string) (string, error) {
				createCalled = true
				createTopic = topic
				return "orch-go-abc12", nil
			},
		},
	}

	result := d.RunPeriodicSynthesisAutoCreate()
	if result == nil {
		t.Fatal("expected result")
	}
	if !createCalled {
		t.Error("CreateSynthesisIssue should be called")
	}
	if createTopic != "spawn-architecture" {
		t.Errorf("CreateSynthesisIssue topic = %q, want 'spawn-architecture'", createTopic)
	}
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1", result.Created)
	}
	if len(result.CreatedIssues) != 1 || result.CreatedIssues[0] != "orch-go-abc12" {
		t.Errorf("CreatedIssues = %v, want [orch-go-abc12]", result.CreatedIssues)
	}
	if result.Evaluated != 1 {
		t.Errorf("Evaluated = %d, want 1", result.Evaluated)
	}
}

func TestDaemon_RunPeriodicSynthesisAutoCreate_SkipsModelExists(t *testing.T) {
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		SynthesisAutoCreate: &mockSynthesisAutoCreateService{
			LoadFunc: func() ([]SynthesisSuggestion, error) {
				return []SynthesisSuggestion{
					{Topic: "existing-model", Count: 8},
				}, nil
			},
			ModelDirFunc: func(topic string) (bool, error) {
				return true, nil // Model dir exists
			},
		},
	}

	result := d.RunPeriodicSynthesisAutoCreate()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Created != 0 {
		t.Errorf("Created = %d, want 0", result.Created)
	}
	if result.SkippedModelExists != 1 {
		t.Errorf("SkippedModelExists = %d, want 1", result.SkippedModelExists)
	}
}

func TestDaemon_RunPeriodicSynthesisAutoCreate_SkipsDedup(t *testing.T) {
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		SynthesisAutoCreate: &mockSynthesisAutoCreateService{
			LoadFunc: func() ([]SynthesisSuggestion, error) {
				return []SynthesisSuggestion{
					{Topic: "dedup-topic", Count: 6},
				}, nil
			},
			ModelDirFunc: func(topic string) (bool, error) {
				return false, nil // No model dir
			},
			HasOpenIssueFunc: func(topic string) (bool, error) {
				return true, nil // Open issue exists
			},
		},
	}

	result := d.RunPeriodicSynthesisAutoCreate()
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

func TestDaemon_RunPeriodicSynthesisAutoCreate_MultipleClusters(t *testing.T) {
	createCount := 0
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		SynthesisAutoCreate: &mockSynthesisAutoCreateService{
			LoadFunc: func() ([]SynthesisSuggestion, error) {
				return []SynthesisSuggestion{
					{Topic: "new-topic-1", Count: 5},       // Above threshold, no model, no issue
					{Topic: "existing-model", Count: 10},    // Model exists, skip
					{Topic: "has-open-issue", Count: 7},     // Open issue exists, skip
					{Topic: "below-threshold", Count: 3},    // Below threshold
					{Topic: "new-topic-2", Count: 6},        // Above threshold, no model, no issue
				}, nil
			},
			ModelDirFunc: func(topic string) (bool, error) {
				return topic == "existing-model", nil
			},
			HasOpenIssueFunc: func(topic string) (bool, error) {
				return topic == "has-open-issue", nil
			},
			CreateIssueFunc: func(topic string, count int, investigations []string) (string, error) {
				createCount++
				return fmt.Sprintf("orch-go-new%d", createCount), nil
			},
		},
	}

	result := d.RunPeriodicSynthesisAutoCreate()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Created != 2 {
		t.Errorf("Created = %d, want 2", result.Created)
	}
	if result.SkippedModelExists != 1 {
		t.Errorf("SkippedModelExists = %d, want 1", result.SkippedModelExists)
	}
	if result.SkippedDedup != 1 {
		t.Errorf("SkippedDedup = %d, want 1", result.SkippedDedup)
	}
	if result.Evaluated != 4 {
		t.Errorf("Evaluated = %d, want 4 (4 above threshold)", result.Evaluated)
	}
}

func TestDaemon_RunPeriodicSynthesisAutoCreate_LoadError(t *testing.T) {
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		SynthesisAutoCreate: &mockSynthesisAutoCreateService{
			LoadFunc: func() ([]SynthesisSuggestion, error) {
				return nil, fmt.Errorf("file not found")
			},
		},
	}

	result := d.RunPeriodicSynthesisAutoCreate()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Error == nil {
		t.Error("expected error")
	}
}

func TestDaemon_RunPeriodicSynthesisAutoCreate_ServiceNotConfigured(t *testing.T) {
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		// SynthesisAutoCreate is nil
	}

	result := d.RunPeriodicSynthesisAutoCreate()
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Error == nil {
		t.Error("expected error for unconfigured service")
	}
}

func TestDaemon_RunPeriodicSynthesisAutoCreate_UpdatesScheduler(t *testing.T) {
	cfg := Config{
		SynthesisAutoCreateEnabled:   true,
		SynthesisAutoCreateInterval:  2 * time.Hour,
		SynthesisAutoCreateThreshold: 5,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		SynthesisAutoCreate: &mockSynthesisAutoCreateService{
			LoadFunc: func() ([]SynthesisSuggestion, error) {
				return nil, nil
			},
		},
	}

	before := d.Scheduler.LastRunTime(TaskSynthesisAutoCreate)
	if !before.IsZero() {
		t.Fatal("expected zero LastRunTime before first run")
	}

	d.RunPeriodicSynthesisAutoCreate()

	after := d.Scheduler.LastRunTime(TaskSynthesisAutoCreate)
	if after.IsZero() {
		t.Error("expected non-zero LastRunTime after run")
	}
}

func TestTopicToSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Spawn Architecture", "spawn-architecture"},
		{"daemon-autonomous-operation", "daemon-autonomous-operation"},
		{"Agent Lifecycle & Recovery", "agent-lifecycle-recovery"},
		{"  spaced  topic  ", "spaced-topic"},
		{"UPPER_CASE_TOPIC", "upper-case-topic"},
		{"topic/with/slashes", "topicwithslashes"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		got := TopicToSlug(tt.input)
		if got != tt.want {
			t.Errorf("TopicToSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDefaultConfig_IncludesSynthesisAutoCreate(t *testing.T) {
	config := DefaultConfig()

	if !config.SynthesisAutoCreateEnabled {
		t.Error("DefaultConfig().SynthesisAutoCreateEnabled should be true")
	}
	if config.SynthesisAutoCreateInterval != 2*time.Hour {
		t.Errorf("DefaultConfig().SynthesisAutoCreateInterval = %v, want 2h", config.SynthesisAutoCreateInterval)
	}
	if config.SynthesisAutoCreateThreshold != 5 {
		t.Errorf("DefaultConfig().SynthesisAutoCreateThreshold = %d, want 5", config.SynthesisAutoCreateThreshold)
	}
}
