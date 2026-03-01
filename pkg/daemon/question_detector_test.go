package daemon

import (
	"testing"
	"time"
)

func TestShouldRunQuestionDetection_Disabled(t *testing.T) {
	d := NewWithConfig(Config{PhaseTimeoutEnabled: false})
	if d.ShouldRunQuestionDetection() {
		t.Error("Should not run when phase timeout disabled")
	}
}

func TestShouldRunQuestionDetection_ZeroInterval(t *testing.T) {
	d := NewWithConfig(Config{PhaseTimeoutEnabled: true, PhaseTimeoutInterval: 0})
	if d.ShouldRunQuestionDetection() {
		t.Error("Should not run with zero interval")
	}
}

func TestShouldRunQuestionDetection_NeverRun(t *testing.T) {
	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: 5 * time.Minute,
	})
	if !d.ShouldRunQuestionDetection() {
		t.Error("Should run when never run before")
	}
}

func TestShouldRunQuestionDetection_IntervalElapsed(t *testing.T) {
	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: 5 * time.Minute,
	})
	d.lastQuestionDetection = time.Now().Add(-10 * time.Minute)
	if !d.ShouldRunQuestionDetection() {
		t.Error("Should run when interval elapsed")
	}
}

func TestShouldRunQuestionDetection_IntervalNotElapsed(t *testing.T) {
	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: 5 * time.Minute,
	})
	d.lastQuestionDetection = time.Now().Add(-1 * time.Minute)
	if d.ShouldRunQuestionDetection() {
		t.Error("Should not run when interval not elapsed")
	}
}

func TestRunPeriodicQuestionDetection_NotDue(t *testing.T) {
	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: 5 * time.Minute,
	})
	d.lastQuestionDetection = time.Now()
	result := d.RunPeriodicQuestionDetection()
	if result != nil {
		t.Error("Should return nil when not due")
	}
}

func newMockDiscovererWithAgents(agents []ActiveAgent) *mockAgentDiscoverer {
	return &mockAgentDiscoverer{
		GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
			return agents, nil
		},
	}
}

func TestRunPeriodicQuestionDetection_DetectsQuestionAgent(t *testing.T) {
	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: 5 * time.Minute,
	})
	d.Agents = newMockDiscovererWithAgents([]ActiveAgent{
		{BeadsID: "test-123", Phase: "QUESTION - Should we use JWT?", Title: "Add auth", UpdatedAt: time.Now()},
		{BeadsID: "test-456", Phase: "Implementing - Adding feature", Title: "Feature X", UpdatedAt: time.Now()},
	})

	result := d.RunPeriodicQuestionDetection()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Error != nil {
		t.Fatalf("Unexpected error: %v", result.Error)
	}
	if result.TotalQuestions != 1 {
		t.Errorf("Expected 1 total question, got %d", result.TotalQuestions)
	}
	if len(result.NewQuestions) != 1 {
		t.Fatalf("Expected 1 new question, got %d", len(result.NewQuestions))
	}
	if result.NewQuestions[0].BeadsID != "test-123" {
		t.Errorf("Expected beads ID test-123, got %s", result.NewQuestions[0].BeadsID)
	}
	if result.NewQuestions[0].Question != "Should we use JWT?" {
		t.Errorf("Expected question text 'Should we use JWT?', got %q", result.NewQuestions[0].Question)
	}
}

func TestRunPeriodicQuestionDetection_NoDuplicateNotification(t *testing.T) {
	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: 5 * time.Minute,
	})
	d.Agents = newMockDiscovererWithAgents([]ActiveAgent{
		{BeadsID: "test-123", Phase: "QUESTION - Should we use JWT?", Title: "Add auth", UpdatedAt: time.Now()},
	})

	// First run: should detect new question
	result1 := d.RunPeriodicQuestionDetection()
	if len(result1.NewQuestions) != 1 {
		t.Fatalf("First run: expected 1 new question, got %d", len(result1.NewQuestions))
	}

	// Reset timer so it runs again
	d.lastQuestionDetection = time.Time{}

	// Second run: same agent still in QUESTION, should NOT be new
	result2 := d.RunPeriodicQuestionDetection()
	if result2 == nil {
		t.Fatal("Expected non-nil result on second run")
	}
	if len(result2.NewQuestions) != 0 {
		t.Errorf("Second run: expected 0 new questions (already notified), got %d", len(result2.NewQuestions))
	}
	if result2.TotalQuestions != 1 {
		t.Errorf("Second run: expected 1 total question, got %d", result2.TotalQuestions)
	}
}

func TestRunPeriodicQuestionDetection_RenotifiesAfterLeaving(t *testing.T) {
	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: 5 * time.Minute,
	})

	// First: agent in QUESTION phase
	d.Agents = newMockDiscovererWithAgents([]ActiveAgent{
		{BeadsID: "test-123", Phase: "QUESTION - Should we use JWT?", Title: "Add auth", UpdatedAt: time.Now()},
	})
	result1 := d.RunPeriodicQuestionDetection()
	if len(result1.NewQuestions) != 1 {
		t.Fatalf("Expected 1 new question, got %d", len(result1.NewQuestions))
	}

	// Agent leaves QUESTION phase
	d.lastQuestionDetection = time.Time{}
	d.Agents = newMockDiscovererWithAgents([]ActiveAgent{
		{BeadsID: "test-123", Phase: "Implementing - Resumed work", Title: "Add auth", UpdatedAt: time.Now()},
	})
	result2 := d.RunPeriodicQuestionDetection()
	if result2.TotalQuestions != 0 {
		t.Errorf("Expected 0 questions after phase change, got %d", result2.TotalQuestions)
	}

	// Agent re-enters QUESTION phase
	d.lastQuestionDetection = time.Time{}
	d.Agents = newMockDiscovererWithAgents([]ActiveAgent{
		{BeadsID: "test-123", Phase: "QUESTION - Different question now?", Title: "Add auth", UpdatedAt: time.Now()},
	})
	result3 := d.RunPeriodicQuestionDetection()
	if len(result3.NewQuestions) != 1 {
		t.Errorf("Expected re-notification after leaving and re-entering QUESTION, got %d", len(result3.NewQuestions))
	}
}

func TestRunPeriodicQuestionDetection_BareQuestionPhase(t *testing.T) {
	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: 5 * time.Minute,
	})
	d.Agents = newMockDiscovererWithAgents([]ActiveAgent{
		{BeadsID: "test-123", Phase: "QUESTION", Title: "Add auth", UpdatedAt: time.Now()},
	})

	result := d.RunPeriodicQuestionDetection()
	if result.TotalQuestions != 1 {
		t.Errorf("Expected 1 question for bare 'QUESTION' phase, got %d", result.TotalQuestions)
	}
	if len(result.NewQuestions) != 1 {
		t.Fatalf("Expected 1 new question, got %d", len(result.NewQuestions))
	}
	if result.NewQuestions[0].Question != "" {
		t.Errorf("Expected empty question text for bare phase, got %q", result.NewQuestions[0].Question)
	}
}
