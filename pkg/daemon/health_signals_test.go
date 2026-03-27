package daemon

import (
	"testing"
	"time"
)

func TestComputeDaemonHealth_AllGreen(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    0,
			Available: 3,
		},
		LastPoll:       now.Add(-30 * time.Second),
		LastSpawn:      now.Add(-5 * time.Minute),
		LastCompletion: now.Add(-10 * time.Minute),
		ReadyCount:     5,
		// Note: Verification field removed — review backlog managed by comprehension gate
		PhaseTimeout: &PhaseTimeoutSnapshot{
			UnresponsiveCount: 0,
			LastCheck:         now,
		},
		QuestionDetection: &QuestionDetectionSnapshot{
			QuestionCount: 0,
			LastCheck:     now,
		},
	}

	summary := ComputeDaemonHealth(status, now)

	if len(summary.Signals) != 7 {
		t.Fatalf("expected 7 signals, got %d", len(summary.Signals))
	}

	for _, sig := range summary.Signals {
		if sig.Level != "green" {
			t.Errorf("signal %q: expected green, got %s (detail: %s)", sig.Name, sig.Level, sig.Detail)
		}
	}
}

func TestComputeDaemonHealth_StalledDaemon(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    0,
			Available: 3,
		},
		LastPoll:   now.Add(-15 * time.Minute),
		ReadyCount: 5,
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Daemon Liveness")
	if sig == nil {
		t.Fatal("expected Daemon Liveness signal")
	}
	if sig.Level != "red" {
		t.Errorf("expected red for 15min stale poll, got %s", sig.Level)
	}
}

func TestComputeDaemonHealth_YellowLiveness(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    0,
			Available: 3,
		},
		LastPoll:   now.Add(-5 * time.Minute),
		ReadyCount: 5,
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Daemon Liveness")
	if sig == nil {
		t.Fatal("expected Daemon Liveness signal")
	}
	if sig.Level != "yellow" {
		t.Errorf("expected yellow for 5min stale poll, got %s", sig.Level)
	}
}

func TestComputeDaemonHealth_CapacitySaturated(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    3,
			Available: 0,
		},
		LastPoll:   now.Add(-30 * time.Second),
		ReadyCount: 10,
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Capacity")
	if sig == nil {
		t.Fatal("expected Capacity signal")
	}
	if sig.Level != "red" {
		t.Errorf("expected red for saturated capacity with queue, got %s", sig.Level)
	}
}

func TestComputeDaemonHealth_CapacityYellow(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       5,
			Active:    4,
			Available: 1,
		},
		LastPoll:   now.Add(-30 * time.Second),
		ReadyCount: 5,
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Capacity")
	if sig == nil {
		t.Fatal("expected Capacity signal")
	}
	if sig.Level != "yellow" {
		t.Errorf("expected yellow for 80%% capacity, got %s", sig.Level)
	}
}

func TestComputeDaemonHealth_LargeQueue(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    0,
			Available: 3,
		},
		LastPoll:   now.Add(-30 * time.Second),
		ReadyCount: 75,
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Queue Depth")
	if sig == nil {
		t.Fatal("expected Queue Depth signal")
	}
	if sig.Level != "red" {
		t.Errorf("expected red for 75 ready issues, got %s", sig.Level)
	}
}

func TestComputeDaemonHealth_MediumQueue(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    0,
			Available: 3,
		},
		LastPoll:   now.Add(-30 * time.Second),
		ReadyCount: 30,
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Queue Depth")
	if sig == nil {
		t.Fatal("expected Queue Depth signal")
	}
	if sig.Level != "yellow" {
		t.Errorf("expected yellow for 30 ready issues, got %s", sig.Level)
	}
}

// Removed: VerificationTracker was removed from Daemon
// TestComputeDaemonHealth_VerificationPaused and TestComputeDaemonHealth_VerificationLow
// tested VerificationStatusSnapshot which no longer exists.

func TestComputeDaemonHealth_UnresponsiveAgents(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    2,
			Available: 1,
		},
		LastPoll:   now.Add(-30 * time.Second),
		ReadyCount: 5,
		PhaseTimeout: &PhaseTimeoutSnapshot{
			UnresponsiveCount: 2,
			LastCheck:         now,
		},
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Unresponsive")
	if sig == nil {
		t.Fatal("expected Unresponsive signal")
	}
	if sig.Level != "red" {
		t.Errorf("expected red for 2 unresponsive, got %s", sig.Level)
	}
}

func TestComputeDaemonHealth_QuestionsWaiting(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    1,
			Available: 2,
		},
		LastPoll:   now.Add(-30 * time.Second),
		ReadyCount: 5,
		QuestionDetection: &QuestionDetectionSnapshot{
			QuestionCount: 3,
			LastCheck:     now,
		},
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Questions")
	if sig == nil {
		t.Fatal("expected Questions signal")
	}
	if sig.Level != "red" {
		t.Errorf("expected red for 3 questions, got %s", sig.Level)
	}
}

func TestComputeDaemonHealth_NilStatus(t *testing.T) {
	summary := ComputeDaemonHealth(nil, time.Now())
	if summary != nil {
		t.Error("expected nil for nil status")
	}
}

func TestComputeDaemonHealth_NoVerification(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    0,
			Available: 3,
		},
		LastPoll:   now.Add(-30 * time.Second),
		ReadyCount: 5,
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Evidence Check")
	if sig == nil {
		t.Fatal("expected Evidence Check signal")
	}
	// No verification data = green (not configured)
	if sig.Level != "green" {
		t.Errorf("expected green for no verification config, got %s", sig.Level)
	}
}

func TestComputeDaemonHealth_ComprehensionThreshold(t *testing.T) {
	now := time.Now()
	status := &DaemonStatus{
		PID:    1234,
		Status: "running",
		Capacity: CapacityStatus{
			Max:       3,
			Active:    0,
			Available: 3,
		},
		LastPoll:   now.Add(-30 * time.Second),
		ReadyCount: 5,
		Comprehension: &ComprehensionSnapshot{
			Count:     5,
			Threshold: 5,
		},
	}

	summary := ComputeDaemonHealth(status, now)
	sig := findSignal(summary, "Comprehension")
	if sig == nil {
		t.Fatal("expected Comprehension signal")
	}
	if sig.Level != "red" {
		t.Errorf("expected red for comprehension threshold, got %s", sig.Level)
	}
}

func findSignal(summary *DaemonHealthSummary, name string) *DaemonHealthSignal {
	if summary == nil {
		return nil
	}
	for i := range summary.Signals {
		if summary.Signals[i].Name == name {
			return &summary.Signals[i]
		}
	}
	return nil
}
