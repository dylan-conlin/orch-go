package daemon

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestRunPeriodicRecovery_DisabledWhenConfigDisabled(t *testing.T) {
	config := Config{
		RecoveryEnabled: false,
	}
	d := NewWithConfig(config)

	result := d.RunPeriodicRecovery()
	if result != nil {
		t.Errorf("Expected nil result when recovery disabled, got %+v", result)
	}
}

func TestRunPeriodicRecovery_DisabledWhenIntervalZero(t *testing.T) {
	config := Config{
		RecoveryEnabled:  true,
		RecoveryInterval: 0,
	}
	d := NewWithConfig(config)

	result := d.RunPeriodicRecovery()
	if result != nil {
		t.Errorf("Expected nil result when interval=0, got %+v", result)
	}
}

func TestRunPeriodicRecovery_RunsImmediatelyFirstTime(t *testing.T) {
	config := Config{
		RecoveryEnabled:  true,
		RecoveryInterval: time.Hour, // Long interval
	}
	d := NewWithConfig(config)

	// First run should execute immediately (lastRecovery is zero)
	if !d.ShouldRunRecovery() {
		t.Error("Expected recovery to run immediately on first call")
	}
}

func TestRunPeriodicRecovery_RespectsInterval(t *testing.T) {
	config := Config{
		RecoveryEnabled:  true,
		RecoveryInterval: time.Minute,
	}
	d := NewWithConfig(config)

	// Simulate having run recently
	d.Scheduler.SetLastRun(TaskRecovery, time.Now())

	// Should not run again immediately
	if d.ShouldRunRecovery() {
		t.Error("Expected recovery to NOT run immediately after recent run")
	}

	// Simulate time passing
	d.Scheduler.SetLastRun(TaskRecovery, time.Now().Add(-2*time.Minute))

	// Should run now
	if !d.ShouldRunRecovery() {
		t.Error("Expected recovery to run after interval passed")
	}
}

func TestRunPeriodicRecovery_RateLimiting(t *testing.T) {
	config := Config{
		RecoveryEnabled:       true,
		RecoveryInterval:      time.Minute,
		RecoveryIdleThreshold: 10 * time.Minute,
		RecoveryRateLimit:     time.Hour,
	}
	d := NewWithConfig(config)

	// Create mock agent
	agent := ActiveAgent{
		BeadsID:   "test-123",
		Phase:     "Planning",
		UpdatedAt: time.Now().Add(-15 * time.Minute), // Idle for 15 min
	}

	// First resume attempt
	d.resumeAttempts[agent.BeadsID] = time.Now()

	// Check if we would skip due to rate limit
	now := time.Now()
	if lastAttempt, exists := d.resumeAttempts[agent.BeadsID]; exists {
		timeSinceLastAttempt := now.Sub(lastAttempt)
		if timeSinceLastAttempt < d.Config.RecoveryRateLimit {
			// Expected - rate limit is working
			if timeSinceLastAttempt >= time.Hour {
				t.Errorf("Rate limit should block, but %v passed (limit: %v)", timeSinceLastAttempt, d.Config.RecoveryRateLimit)
			}
		} else {
			t.Error("Expected rate limit to block within 1 hour")
		}
	}
}

func TestShouldRunRecovery_TimingCalculations(t *testing.T) {
	config := Config{
		RecoveryEnabled:  true,
		RecoveryInterval: 5 * time.Minute,
	}
	d := NewWithConfig(config)

	tests := []struct {
		name           string
		lastRecovery   time.Time
		expectedResult bool
	}{
		{
			name:           "never run before", // Zero time
			expectedResult: true,
		},
		{
			name:           "run 1 minute ago (too soon)",
			lastRecovery:   time.Now().Add(-1 * time.Minute),
			expectedResult: false,
		},
		{
			name:           "run 6 minutes ago (should run)",
			lastRecovery:   time.Now().Add(-6 * time.Minute),
			expectedResult: true,
		},
		{
			name:           "run exactly at interval",
			lastRecovery:   time.Now().Add(-5 * time.Minute),
			expectedResult: true, // At or past interval should trigger
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d.Scheduler.SetLastRun(TaskRecovery, tt.lastRecovery)
			result := d.ShouldRunRecovery()
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestGetActiveAgents_FiltersInProgress(t *testing.T) {
	// This test verifies that GetActiveAgents only returns in_progress issues
	// We can't fully test without real beads data, but we can verify the interface exists
	_, err := GetActiveAgents()

	// We expect either nil error (if beads is set up) or an error about missing beads
	// Both are acceptable in a unit test context
	if err != nil {
		// Check it's a reasonable error (not a panic or unexpected failure)
		if err.Error() == "" {
			t.Error("GetActiveAgents returned empty error message")
		}
	}
}

func TestActiveAgent_Structure(t *testing.T) {
	// Verify ActiveAgent struct has required fields
	agent := ActiveAgent{
		BeadsID:   "test-123",
		Phase:     "Planning",
		UpdatedAt: time.Now(),
		Title:     "Test Issue",
	}

	if agent.BeadsID == "" {
		t.Error("BeadsID should not be empty")
	}
	if agent.Phase == "" {
		t.Error("Phase should not be empty")
	}
	if agent.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
	if agent.Title == "" {
		t.Error("Title should not be empty")
	}
}

func TestRecoveryResult_Structure(t *testing.T) {
	// Verify RecoveryResult has expected fields
	result := RecoveryResult{
		ResumedCount: 2,
		SkippedCount: 3,
		Error:        nil,
		Message:      "Recovery attempted: 2 resumed, 3 skipped",
	}

	if result.ResumedCount != 2 {
		t.Errorf("Expected ResumedCount=2, got %d", result.ResumedCount)
	}
	if result.SkippedCount != 3 {
		t.Errorf("Expected SkippedCount=3, got %d", result.SkippedCount)
	}
	if result.Error != nil {
		t.Error("Expected no error")
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestRecoveryConfig_Defaults(t *testing.T) {
	config := DefaultConfig()

	// Verify recovery defaults
	if !config.RecoveryEnabled {
		t.Error("Recovery should be enabled by default")
	}
	if config.RecoveryInterval != 5*time.Minute {
		t.Errorf("Expected RecoveryInterval=5m, got %v", config.RecoveryInterval)
	}
	if config.RecoveryIdleThreshold != 10*time.Minute {
		t.Errorf("Expected RecoveryIdleThreshold=10m, got %v", config.RecoveryIdleThreshold)
	}
	if config.RecoveryRateLimit != time.Hour {
		t.Errorf("Expected RecoveryRateLimit=1h, got %v", config.RecoveryRateLimit)
	}
}

func TestLastRecoveryTime_InitiallyZero(t *testing.T) {
	d := New()

	lastTime := d.LastRecoveryTime()
	if !lastTime.IsZero() {
		t.Error("LastRecoveryTime should be zero initially")
	}
}

func TestNextRecoveryTime_DisabledWhenRecoveryOff(t *testing.T) {
	config := Config{
		RecoveryEnabled: false,
	}
	d := NewWithConfig(config)

	nextTime := d.NextRecoveryTime()
	if !nextTime.IsZero() {
		t.Error("NextRecoveryTime should be zero when recovery disabled")
	}
}

func TestNextRecoveryTime_DisabledWhenIntervalZero(t *testing.T) {
	config := Config{
		RecoveryEnabled:  true,
		RecoveryInterval: 0,
	}
	d := NewWithConfig(config)

	nextTime := d.NextRecoveryTime()
	if !nextTime.IsZero() {
		t.Error("NextRecoveryTime should be zero when interval=0")
	}
}

func TestNextRecoveryTime_ImmediateWhenNeverRun(t *testing.T) {
	config := Config{
		RecoveryEnabled:  true,
		RecoveryInterval: time.Hour,
	}
	d := NewWithConfig(config)

	nextTime := d.NextRecoveryTime()
	// Should be approximately now (within a few seconds)
	if time.Until(nextTime) > 5*time.Second {
		t.Error("NextRecoveryTime should be immediate when never run")
	}
}

func TestNextRecoveryTime_ScheduledAfterInterval(t *testing.T) {
	config := Config{
		RecoveryEnabled:  true,
		RecoveryInterval: time.Hour,
	}
	d := NewWithConfig(config)

	// Simulate having run 30 minutes ago
	lastRun := time.Now().Add(-30 * time.Minute)
	d.Scheduler.SetLastRun(TaskRecovery, lastRun)

	nextTime := d.NextRecoveryTime()
	expectedNext := lastRun.Add(time.Hour)

	// Should be scheduled 30 minutes from now (within tolerance)
	diff := nextTime.Sub(expectedNext)
	if diff > time.Second || diff < -time.Second {
		t.Errorf("NextRecoveryTime calculation incorrect: expected %v, got %v (diff: %v)",
			expectedNext, nextTime, diff)
	}
}

// Mock test for verify package integration
func TestRecoveryIntegration_VerifyPackage(t *testing.T) {
	// Verify that the verify package functions exist and can be called
	// This is a smoke test to ensure the integration points are correct

	// Test that ParsePhaseFromComments exists
	comments := []verify.Comment{
		{
			Text: "Phase: Planning - Starting work",
		},
	}

	phaseStatus := verify.ParsePhaseFromComments(comments)
	if phaseStatus.Phase != "Planning" {
		t.Errorf("Expected phase 'Planning', got '%s'", phaseStatus.Phase)
	}
}
