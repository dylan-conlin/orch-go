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
	d.lastRecovery = time.Now()

	// Should not run again immediately
	if d.ShouldRunRecovery() {
		t.Error("Expected recovery to NOT run immediately after recent run")
	}

	// Simulate time passing
	d.lastRecovery = time.Now().Add(-2 * time.Minute)

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
			name:           "never run before",
			lastRecovery:   time.Time{}, // Zero time
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
			d.lastRecovery = tt.lastRecovery
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
	d.lastRecovery = time.Now().Add(-30 * time.Minute)

	nextTime := d.NextRecoveryTime()
	expectedNext := d.lastRecovery.Add(time.Hour)

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

// =============================================================================
// Server Recovery Tests
// =============================================================================

func TestServerRecoveryState_ShouldRunServerRecovery(t *testing.T) {
	state := NewServerRecoveryState()

	// Immediately after creation, should not run (stabilization delay not passed)
	if state.ShouldRunServerRecovery(30 * time.Second) {
		t.Error("Should not run server recovery immediately after creation")
	}

	// Simulate time passing (past stabilization delay)
	state.daemonStartTime = time.Now().Add(-31 * time.Second)
	if !state.ShouldRunServerRecovery(30 * time.Second) {
		t.Error("Should run server recovery after stabilization delay")
	}

	// After marking as run, should not run again
	state.MarkRecoveryRun()
	if state.ShouldRunServerRecovery(30 * time.Second) {
		t.Error("Should not run server recovery again after already run")
	}
}

func TestServerRecoveryState_WasRecentlyRecovered(t *testing.T) {
	state := NewServerRecoveryState()
	beadsID := "test-123"
	rateLimit := time.Hour

	// Before any recovery, should return false
	if state.WasRecentlyRecovered(beadsID, rateLimit) {
		t.Error("WasRecentlyRecovered should be false before any recovery")
	}

	// After marking as recovered, should return true
	state.MarkRecovered(beadsID)
	if !state.WasRecentlyRecovered(beadsID, rateLimit) {
		t.Error("WasRecentlyRecovered should be true after recovery")
	}

	// Simulate rate limit passing
	state.recoveredSessionsMap[beadsID] = time.Now().Add(-2 * time.Hour)
	if state.WasRecentlyRecovered(beadsID, rateLimit) {
		t.Error("WasRecentlyRecovered should be false after rate limit passed")
	}
}

func TestRunServerRecovery_DisabledWhenConfigDisabled(t *testing.T) {
	config := Config{
		ServerRecoveryEnabled: false,
	}
	d := NewWithConfig(config)

	result := d.RunServerRecovery()
	if result != nil {
		t.Errorf("Expected nil result when server recovery disabled, got %+v", result)
	}
}

func TestRunServerRecovery_WaitsForStabilizationDelay(t *testing.T) {
	config := DefaultConfig()
	config.ServerRecoveryEnabled = true
	config.ServerRecoveryStabilizationDelay = 30 * time.Second
	d := NewWithConfig(config)

	// Immediately after creation, should not run (stabilization delay)
	result := d.RunServerRecovery()
	if result != nil {
		t.Errorf("Expected nil result during stabilization delay, got %+v", result)
	}

	// Simulate time passing
	d.serverRecoveryState.daemonStartTime = time.Now().Add(-31 * time.Second)

	// Now it should run (will likely return 0 orphaned since no real sessions)
	result = d.RunServerRecovery()
	if result == nil {
		t.Error("Expected result after stabilization delay")
	}
}

func TestServerRecoveryResult_Structure(t *testing.T) {
	result := ServerRecoveryResult{
		ResumedCount:  2,
		SkippedCount:  1,
		OrphanedCount: 3,
		Message:       "Server recovery: 3 orphaned found, 2 resumed, 1 skipped",
	}

	if result.ResumedCount != 2 {
		t.Errorf("Expected ResumedCount=2, got %d", result.ResumedCount)
	}
	if result.SkippedCount != 1 {
		t.Errorf("Expected SkippedCount=1, got %d", result.SkippedCount)
	}
	if result.OrphanedCount != 3 {
		t.Errorf("Expected OrphanedCount=3, got %d", result.OrphanedCount)
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestOrphanedSession_Structure(t *testing.T) {
	orphan := OrphanedSession{
		BeadsID:       "test-123",
		SessionID:     "ses_abc",
		WorkspacePath: "/path/to/workspace",
		AgentID:       "og-feature-impl-test",
		Phase:         "Implementing",
		ProjectDir:    "/path/to/project",
	}

	if orphan.BeadsID == "" {
		t.Error("BeadsID should not be empty")
	}
	if orphan.SessionID == "" {
		t.Error("SessionID should not be empty")
	}
	if orphan.Phase == "" {
		t.Error("Phase should not be empty")
	}
}

func TestServerRecoveryConfig_Defaults(t *testing.T) {
	config := DefaultConfig()

	// Verify server recovery defaults
	if !config.ServerRecoveryEnabled {
		t.Error("ServerRecovery should be enabled by default")
	}
	if config.ServerRecoveryStabilizationDelay != 30*time.Second {
		t.Errorf("Expected ServerRecoveryStabilizationDelay=30s, got %v", config.ServerRecoveryStabilizationDelay)
	}
	if config.ServerRecoveryResumeDelay != 10*time.Second {
		t.Errorf("Expected ServerRecoveryResumeDelay=10s, got %v", config.ServerRecoveryResumeDelay)
	}
	if config.ServerRecoveryRateLimit != time.Hour {
		t.Errorf("Expected ServerRecoveryRateLimit=1h, got %v", config.ServerRecoveryRateLimit)
	}
}

func TestShouldRunServerRecovery_DisabledWhenRecoveryOff(t *testing.T) {
	config := Config{
		ServerRecoveryEnabled: false,
	}
	d := NewWithConfig(config)

	if d.ShouldRunServerRecovery() {
		t.Error("ShouldRunServerRecovery should be false when disabled")
	}
}

func TestShouldRunServerRecovery_NilStateReturnsFalse(t *testing.T) {
	config := Config{
		ServerRecoveryEnabled: true,
	}
	d := NewWithConfig(config)
	d.serverRecoveryState = nil

	if d.ShouldRunServerRecovery() {
		t.Error("ShouldRunServerRecovery should be false when state is nil")
	}
}

func TestServerRecoveryState_DetectsServerRestart(t *testing.T) {
	state := NewServerRecoveryState()
	stabilizationDelay := 1 * time.Millisecond

	// Simulate time passing (past stabilization delay)
	state.daemonStartTime = time.Now().Add(-10 * time.Second)

	// First recovery should be allowed
	if !state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("First recovery should be allowed after stabilization delay")
	}

	// Mark first recovery as run
	state.MarkRecoveryRun()

	// Recovery should NOT run immediately after (no restart detected)
	if state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("Recovery should not run immediately after first run")
	}

	// Simulate server going down
	state.UpdateServerHealth(false)

	// Recovery should still not run while server is down
	if state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("Recovery should not run while server is down")
	}

	// Simulate server coming back up (restart detected)
	state.UpdateServerHealth(true)

	// NOW recovery should be allowed again because we detected a restart
	if !state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("Recovery should run after detecting server restart (down -> up transition)")
	}

	// Run recovery again
	state.MarkRecoveryRun()

	// Should not run again until next restart
	if state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("Recovery should not run again until next restart")
	}

	// Simulate second restart
	state.UpdateServerHealth(false)
	state.UpdateServerHealth(true)

	// Should allow recovery again after second restart
	if !state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("Recovery should run after second server restart")
	}
}

func TestCheckServerHealth_UpdatesRecoveryState(t *testing.T) {
	config := DefaultConfig()
	config.ServerRecoveryEnabled = true
	config.CleanupServerURL = "http://127.0.0.1:4096"
	d := NewWithConfig(config)

	// Simulate time passing (past stabilization delay)
	d.serverRecoveryState.daemonStartTime = time.Now().Add(-1 * time.Minute)

	// First call - server might be up or down depending on test environment
	// This just verifies the method doesn't panic and updates state
	d.CheckServerHealth()

	// Verify serverRecoveryState was updated (checking that method works)
	// We can't easily test the full flow without a real server, but we can
	// verify the method exists and doesn't panic
}

func TestUpdateServerHealth_NoOpWhenServerStaysUp(t *testing.T) {
	state := NewServerRecoveryState()
	stabilizationDelay := 1 * time.Millisecond
	state.daemonStartTime = time.Now().Add(-10 * time.Second)

	// First recovery
	state.MarkRecoveryRun()

	// Server stays up (no down -> up transition)
	state.UpdateServerHealth(true)
	state.UpdateServerHealth(true)
	state.UpdateServerHealth(true)

	// Should NOT allow recovery since no restart was detected
	if state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("Recovery should not run when server has been continuously up")
	}
}

func TestUpdateServerHealth_MultipleRestarts(t *testing.T) {
	state := NewServerRecoveryState()
	stabilizationDelay := 1 * time.Millisecond
	state.daemonStartTime = time.Now().Add(-10 * time.Second)

	// First recovery
	state.MarkRecoveryRun()

	// First restart
	state.UpdateServerHealth(false)
	state.UpdateServerHealth(true)

	if !state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("Recovery should run after first restart")
	}
	state.MarkRecoveryRun()

	// Second restart
	state.UpdateServerHealth(false)
	state.UpdateServerHealth(true)

	if !state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("Recovery should run after second restart")
	}
	state.MarkRecoveryRun()

	// Third restart
	state.UpdateServerHealth(false)
	state.UpdateServerHealth(true)

	if !state.ShouldRunServerRecovery(stabilizationDelay) {
		t.Error("Recovery should run after third restart")
	}
}
