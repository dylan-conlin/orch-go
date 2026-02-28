package agent

import (
	"testing"
	"time"
)

func TestIsActiveForConcurrency_RunningAlwaysCounts(t *testing.T) {
	// Running agents always count toward concurrency limit
	if !IsActiveForConcurrency("running", time.Now(), "") {
		t.Error("running agent should count toward concurrency limit")
	}
}

func TestIsActiveForConcurrency_IdleNeverCounts(t *testing.T) {
	// Idle agents should NOT count, even if recently active
	recentActivity := time.Now().Add(-5 * time.Minute) // 5 minutes ago
	if IsActiveForConcurrency("idle", recentActivity, "") {
		t.Error("idle agent should NOT count toward concurrency limit, even if recently active")
	}
}

func TestIsActiveForConcurrency_IdleOldNeverCounts(t *testing.T) {
	// Idle agents with old activity should definitely not count
	oldActivity := time.Now().Add(-2 * time.Hour)
	if IsActiveForConcurrency("idle", oldActivity, "") {
		t.Error("idle agent with old activity should NOT count toward concurrency limit")
	}
}

func TestIsActiveForConcurrency_PhaseCompleteNeverCounts(t *testing.T) {
	// Phase: Complete agents never count, even if "running"
	if IsActiveForConcurrency("running", time.Now(), "Complete") {
		t.Error("Phase: Complete agent should NOT count, even if running")
	}
}

func TestIsActiveForConcurrency_PhaseCompleteCaseInsensitive(t *testing.T) {
	// Phase matching should be case-insensitive
	if IsActiveForConcurrency("running", time.Now(), "complete") {
		t.Error("Phase: complete (lowercase) should NOT count")
	}
	if IsActiveForConcurrency("running", time.Now(), "COMPLETE") {
		t.Error("Phase: COMPLETE (uppercase) should NOT count")
	}
}

func TestIsActiveForConcurrency_DeadDoesNotCount(t *testing.T) {
	// Dead/unknown status should not count
	if IsActiveForConcurrency("dead", time.Now(), "") {
		t.Error("dead agent should NOT count toward concurrency limit")
	}
}

func TestIsActiveForConcurrency_ManyIdleAgentsDontBlockSpawns(t *testing.T) {
	// Simulates the repro scenario: 15 idle agents should result in 0 active for concurrency
	activeCount := 0
	for i := 0; i < 15; i++ {
		// Mix of recent and old idle agents
		var lastActivity time.Time
		if i%2 == 0 {
			lastActivity = time.Now().Add(-30 * time.Minute) // recent idle
		} else {
			lastActivity = time.Now().Add(-3 * time.Hour) // old idle
		}
		if IsActiveForConcurrency("idle", lastActivity, "") {
			activeCount++
		}
	}
	if activeCount != 0 {
		t.Errorf("15 idle agents should count as 0 active for concurrency, got %d", activeCount)
	}
}
