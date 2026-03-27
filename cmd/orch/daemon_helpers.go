package main

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/focus"
)

// daemonConfigFromFlags builds a Config starting from DefaultConfig(),
// overriding with CLI flag values. All daemon paths (run, once, dry-run,
// preview) MUST use this function instead of constructing Config directly.
func daemonConfigFromFlags() daemon.Config {
	config := daemon.DefaultConfig()

	// Override with CLI flags
	config.PollInterval = time.Duration(daemonPollInterval) * time.Second
	config.MaxAgents = daemonMaxAgents
	config.Label = daemonLabel
	config.SpawnDelay = time.Duration(daemonDelay) * time.Second
	config.DryRun = daemonDryRun
	config.Verbose = daemonVerbose
	config.ReflectEnabled = daemonReflectInterval > 0
	config.ReflectInterval = time.Duration(daemonReflectInterval) * time.Minute
	config.ReflectCreateIssues = daemonReflectIssues
	config.ReflectOpenEnabled = daemonReflectOpen
	config.CleanupEnabled = daemonCleanupEnabled && daemonCleanupInterval > 0
	config.CleanupInterval = time.Duration(daemonCleanupInterval) * time.Minute
	config.CleanupAgeDays = daemonCleanupAge
	config.CleanupPreserveOrchestrator = daemonCleanupPreserveOrch
	config.CleanupServerURL = serverURL
	config.OrphanDetectionEnabled = daemonOrphanDetectionInterval > 0
	config.OrphanDetectionInterval = time.Duration(daemonOrphanDetectionInterval) * time.Minute
	config.OrphanAgeThreshold = time.Duration(daemonOrphanAgeThreshold) * time.Minute
	config.PhaseTimeoutEnabled = daemonPhaseTimeoutInterval > 0
	config.PhaseTimeoutInterval = time.Duration(daemonPhaseTimeoutInterval) * time.Minute
	config.PhaseTimeoutThreshold = time.Duration(daemonPhaseTimeoutThreshold) * time.Minute
	config.AgreementCheckEnabled = daemonAgreementCheckInterval > 0
	config.AgreementCheckInterval = time.Duration(daemonAgreementCheckInterval) * time.Minute
	config.BeadsHealthEnabled = daemonBeadsHealthInterval > 0
	config.BeadsHealthInterval = time.Duration(daemonBeadsHealthInterval) * time.Minute

	return config
}

// wireFocusBoost loads the current focus and wires it into the daemon for
// priority boosting. Gracefully degrades if focus can't be loaded.
func wireFocusBoost(d *daemon.Daemon) {
	store, err := focus.New("")
	if err != nil {
		return
	}
	f := store.Get()
	if f == nil {
		return
	}
	d.FocusGoal = f.Goal
	d.FocusBoostAmount = 1 // default: boost by 1 priority level
	d.ProjectDirNames = daemon.BuildProjectDirNames(d.ProjectRegistry)
}

// checkDaemonStuck returns true if the daemon appears stuck:
// all slots full with no spawns or completions for stuckThreshold,
// and last notification was more than cooldown ago.
func checkDaemonStuck(lastSpawn, lastCompletion, lastNotification time.Time, stuckThreshold, cooldown time.Duration) bool {
	if lastSpawn.IsZero() || lastCompletion.IsZero() {
		return false
	}
	return time.Since(lastSpawn) > stuckThreshold &&
		time.Since(lastCompletion) > stuckThreshold &&
		time.Since(lastNotification) > cooldown
}

// formatDaemonDuration formats a duration for daemon display.
func formatDaemonDuration(d time.Duration) string {
	if d == 0 {
		return "0 (run once)"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return d.String()
}

func truncateDaemonString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
