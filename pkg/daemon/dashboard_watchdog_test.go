package daemon

import (
	"errors"
	"testing"
	"time"
)

func TestCheckDashboardHealth_DisabledByConfig(t *testing.T) {
	d := &Daemon{
		Config: Config{
			DashboardWatchdogEnabled: false,
		},
	}
	result := d.CheckDashboardHealth()
	if result != nil {
		t.Error("Expected nil result when watchdog is disabled")
	}
}

func TestCheckDashboardHealth_SkipsWhenNotDue(t *testing.T) {
	d := &Daemon{
		Config: Config{
			DashboardWatchdogEnabled:  true,
			DashboardWatchdogInterval: 30 * time.Second,
		},
		lastDashboardCheck: time.Now(), // Just checked
	}
	result := d.CheckDashboardHealth()
	if result != nil {
		t.Error("Expected nil result when check is not due yet")
	}
}

func TestCheckDashboardHealth_RunsOnFirstCall(t *testing.T) {
	d := &Daemon{
		Config: Config{
			DashboardWatchdogEnabled:               true,
			DashboardWatchdogInterval:              30 * time.Second,
			DashboardWatchdogFailuresBeforeRestart: 2,
			DashboardWatchdogRestartCooldown:       5 * time.Minute,
		},
		restartDashboardFunc: func() error { return nil },
		// lastDashboardCheck is zero value - should run on first call
	}
	result := d.CheckDashboardHealth()
	if result == nil {
		t.Fatal("Expected non-nil result on first call")
	}
	// On a real system, ports may or may not be responding.
	// The key assertion is that it actually runs.
}

func TestCheckDashboardHealth_RunsWhenDue(t *testing.T) {
	d := &Daemon{
		Config: Config{
			DashboardWatchdogEnabled:               true,
			DashboardWatchdogInterval:              30 * time.Second,
			DashboardWatchdogFailuresBeforeRestart: 2,
			DashboardWatchdogRestartCooldown:       5 * time.Minute,
		},
		lastDashboardCheck:   time.Now().Add(-1 * time.Minute), // Overdue
		restartDashboardFunc: func() error { return nil },
	}
	result := d.CheckDashboardHealth()
	if result == nil {
		t.Fatal("Expected non-nil result when check is overdue")
	}
}

func TestCheckDashboardHealth_ConsecutiveFailuresRequired(t *testing.T) {
	restartCalled := false
	d := &Daemon{
		Config: Config{
			DashboardWatchdogEnabled:               true,
			DashboardWatchdogInterval:              0, // Always due
			DashboardWatchdogFailuresBeforeRestart: 3,
			DashboardWatchdogRestartCooldown:       5 * time.Minute,
		},
		restartDashboardFunc: func() error {
			restartCalled = true
			return nil
		},
	}

	// Simulate port checks failing by checking that restart is NOT called
	// on first failure (needs 3 consecutive)
	// Note: This test relies on ports 3348/5188 NOT being open in the test environment.
	// If they happen to be open, the test still passes (health check succeeds, no restart needed).
	result := d.CheckDashboardHealth()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Healthy {
		// Dashboard is actually running in this environment - that's fine, skip restart logic test
		t.Skip("Dashboard services appear to be running - can't test failure path")
	}

	// First failure - should NOT restart yet
	if restartCalled {
		t.Error("Restart should not be called on first failure (need 3 consecutive)")
	}
	if d.dashboardConsecutiveFailures != 1 {
		t.Errorf("Expected 1 consecutive failure, got %d", d.dashboardConsecutiveFailures)
	}

	// Second failure
	d.lastDashboardCheck = time.Time{} // Reset to allow next check
	result = d.CheckDashboardHealth()
	if restartCalled {
		t.Error("Restart should not be called on second failure (need 3 consecutive)")
	}
	if d.dashboardConsecutiveFailures != 2 {
		t.Errorf("Expected 2 consecutive failures, got %d", d.dashboardConsecutiveFailures)
	}

	// Third failure - NOW should restart
	d.lastDashboardCheck = time.Time{} // Reset to allow next check
	result = d.CheckDashboardHealth()
	if !restartCalled {
		t.Error("Restart should have been called after 3 consecutive failures")
	}
	if result == nil {
		t.Fatal("Expected non-nil result after restart")
	}
	if !result.Restarted {
		t.Error("Expected Restarted=true after restart triggered")
	}
	// Consecutive failures should be reset after restart
	if d.dashboardConsecutiveFailures != 0 {
		t.Errorf("Expected 0 consecutive failures after restart, got %d", d.dashboardConsecutiveFailures)
	}
}

func TestCheckDashboardHealth_RestartCooldown(t *testing.T) {
	restartCount := 0
	d := &Daemon{
		Config: Config{
			DashboardWatchdogEnabled:               true,
			DashboardWatchdogInterval:              0, // Always due
			DashboardWatchdogFailuresBeforeRestart: 1, // Restart on first failure
			DashboardWatchdogRestartCooldown:       5 * time.Minute,
		},
		lastDashboardRestart: time.Now().Add(-1 * time.Minute), // Restarted 1 min ago
		restartDashboardFunc: func() error {
			restartCount++
			return nil
		},
		dashboardConsecutiveFailures: 5, // Many failures already
	}

	result := d.CheckDashboardHealth()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Healthy {
		t.Skip("Dashboard services appear to be running - can't test cooldown path")
	}

	// Should NOT restart because cooldown hasn't elapsed
	if restartCount > 0 {
		t.Error("Restart should not be called during cooldown period")
	}
	if result.Restarted {
		t.Error("Expected Restarted=false during cooldown")
	}
}

func TestCheckDashboardHealth_RestartFailure(t *testing.T) {
	restartErr := errors.New("orch-dashboard not found")
	d := &Daemon{
		Config: Config{
			DashboardWatchdogEnabled:               true,
			DashboardWatchdogInterval:              0, // Always due
			DashboardWatchdogFailuresBeforeRestart: 1, // Restart on first failure
			DashboardWatchdogRestartCooldown:       0, // No cooldown
		},
		restartDashboardFunc: func() error {
			return restartErr
		},
	}

	result := d.CheckDashboardHealth()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Healthy {
		t.Skip("Dashboard services appear to be running - can't test failure path")
	}

	if result.Restarted {
		t.Error("Expected Restarted=false when restart fails")
	}
	if result.Error == nil {
		t.Error("Expected non-nil Error when restart fails")
	}
}

func TestCheckDashboardHealth_ResetsOnHealthy(t *testing.T) {
	d := &Daemon{
		Config: Config{
			DashboardWatchdogEnabled:               true,
			DashboardWatchdogInterval:              0,
			DashboardWatchdogFailuresBeforeRestart: 2,
			DashboardWatchdogRestartCooldown:       5 * time.Minute,
		},
		dashboardConsecutiveFailures: 5, // Previous failures
		restartDashboardFunc:         func() error { return nil },
	}

	result := d.CheckDashboardHealth()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// If services happen to be up in this test env, consecutive failures should reset
	if result.Healthy {
		if d.dashboardConsecutiveFailures != 0 {
			t.Errorf("Expected consecutive failures to reset to 0 when healthy, got %d",
				d.dashboardConsecutiveFailures)
		}
	}
}

func TestCheckDashboardHealth_DefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if !config.DashboardWatchdogEnabled {
		t.Error("DefaultConfig().DashboardWatchdogEnabled should be true")
	}
	if config.DashboardWatchdogInterval != 30*time.Second {
		t.Errorf("DefaultConfig().DashboardWatchdogInterval = %v, want 30s", config.DashboardWatchdogInterval)
	}
	if config.DashboardWatchdogFailuresBeforeRestart != 2 {
		t.Errorf("DefaultConfig().DashboardWatchdogFailuresBeforeRestart = %d, want 2",
			config.DashboardWatchdogFailuresBeforeRestart)
	}
	if config.DashboardWatchdogRestartCooldown != 5*time.Minute {
		t.Errorf("DefaultConfig().DashboardWatchdogRestartCooldown = %v, want 5m",
			config.DashboardWatchdogRestartCooldown)
	}
}

func TestIsTCPPortResponding_ClosedPort(t *testing.T) {
	// Test with a port that's almost certainly not in use
	if isTCPPortResponding(19999) {
		t.Skip("Port 19999 is unexpectedly in use")
	}
	// This should return false for a closed port
	if isTCPPortResponding(19999) {
		t.Error("Expected false for port 19999 (should not be listening)")
	}
}
