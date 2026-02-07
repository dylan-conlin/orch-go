// dashboard_watchdog.go contains periodic dashboard health monitoring and auto-restart.
//
// The dashboard (orch-dashboard) runs via overmind with three services:
//   - api (orch serve) on port 3348
//   - web (vite dev server) on port 5188
//   - opencode on port 4096
//
// When any of the core services (api, web) die, this watchdog detects the failure
// and automatically runs `orch-dashboard restart` to recover.
package daemon

import (
	"fmt"
	"net"
	"os/exec"
	"time"
)

// DashboardWatchdogResult contains the result of a dashboard health check cycle.
type DashboardWatchdogResult struct {
	// Healthy is true if all checked services are responding.
	Healthy bool
	// Restarted is true if a restart was triggered.
	Restarted bool
	// Error is set if the restart failed.
	Error error
	// Message is a human-readable summary.
	Message string
	// DownServices lists which services were detected as down.
	DownServices []string
}

// CheckDashboardHealth checks if the dashboard services are healthy and
// auto-restarts them if they are down. Uses TCP port checks for speed.
//
// Restart behavior:
//   - Requires consecutive failures before restarting (avoids flapping on transient errors)
//   - Rate-limits restarts to prevent infinite restart loops
//   - Only restarts when core services (api on 3348, web on 5188) are down
//
// Returns nil if the watchdog is not due to run (based on interval tracking).
func (d *Daemon) CheckDashboardHealth() *DashboardWatchdogResult {
	if !d.Config.DashboardWatchdogEnabled {
		return nil
	}

	// Check if enough time has passed since last check
	if !d.lastDashboardCheck.IsZero() && time.Since(d.lastDashboardCheck) < d.Config.DashboardWatchdogInterval {
		return nil
	}

	d.lastDashboardCheck = time.Now()

	// Check core dashboard services via TCP
	var downServices []string
	for _, check := range []struct {
		name string
		port int
	}{
		{"api", 3348},
		{"web", 5188},
	} {
		if !isTCPPortResponding(check.port) {
			downServices = append(downServices, fmt.Sprintf("%s (port %d)", check.name, check.port))
		}
	}

	// All healthy - reset consecutive failure count
	if len(downServices) == 0 {
		d.dashboardConsecutiveFailures = 0
		return &DashboardWatchdogResult{
			Healthy: true,
			Message: "Dashboard services healthy",
		}
	}

	// Increment consecutive failure counter
	d.dashboardConsecutiveFailures++

	// Require consecutive failures before restarting to avoid flapping
	if d.dashboardConsecutiveFailures < d.Config.DashboardWatchdogFailuresBeforeRestart {
		return &DashboardWatchdogResult{
			Healthy:      false,
			DownServices: downServices,
			Message: fmt.Sprintf("Dashboard unhealthy (%d/%d consecutive failures before restart): %v",
				d.dashboardConsecutiveFailures, d.Config.DashboardWatchdogFailuresBeforeRestart, downServices),
		}
	}

	// Check rate limit - don't restart too frequently
	if !d.lastDashboardRestart.IsZero() && time.Since(d.lastDashboardRestart) < d.Config.DashboardWatchdogRestartCooldown {
		remaining := d.Config.DashboardWatchdogRestartCooldown - time.Since(d.lastDashboardRestart)
		return &DashboardWatchdogResult{
			Healthy:      false,
			DownServices: downServices,
			Message: fmt.Sprintf("Dashboard down but restart on cooldown (%s remaining): %v",
				remaining.Round(time.Second), downServices),
		}
	}

	// Attempt restart
	d.lastDashboardRestart = time.Now()
	d.dashboardConsecutiveFailures = 0

	if err := d.restartDashboardFunc(); err != nil {
		return &DashboardWatchdogResult{
			Healthy:      false,
			Restarted:    false,
			Error:        fmt.Errorf("dashboard restart failed: %w", err),
			DownServices: downServices,
			Message:      fmt.Sprintf("Dashboard restart failed: %v (down services: %v)", err, downServices),
		}
	}

	return &DashboardWatchdogResult{
		Healthy:      false, // Was unhealthy (restart was triggered)
		Restarted:    true,
		DownServices: downServices,
		Message:      fmt.Sprintf("Dashboard auto-restarted (was down: %v)", downServices),
	}
}

// isTCPPortResponding performs a quick TCP connect test to check if a service
// is listening on the given port.
func isTCPPortResponding(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// restartDashboard runs `orch-dashboard restart` to restart all dashboard services.
// This is the default restart function used in production.
func restartDashboard() error {
	cmd := exec.Command("orch-dashboard", "restart")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}
