// Package daemon provides autonomous overnight processing capabilities.
// This file provides daemon status display formatting for the `orch daemon status` command.
package daemon

import (
	"fmt"
	"time"
)

// StatusInfo contains the daemon status information for display.
// This is the result of querying daemon state with PID liveness validation.
type StatusInfo struct {
	// Running indicates whether the daemon process is alive.
	Running bool

	// PID is the daemon process ID (0 if not running).
	PID int

	// Status is the operational state: "running", "stalled", "paused", or "stopped".
	Status string

	// Capacity holds worker pool capacity info.
	Capacity CapacityStatus

	// LastPoll is the last poll timestamp.
	LastPoll time.Time

	// LastSpawn is the last successful spawn timestamp.
	LastSpawn time.Time

	// LastCompletion is the last auto-completion timestamp.
	LastCompletion time.Time

	// ReadyCount is the number of ready issues.
	ReadyCount int

	// StaleFile indicates the status file exists but the daemon process is dead.
	StaleFile bool

	// Verification holds verification pause information.
	Verification *VerificationStatusSnapshot

	// CompletionFailures holds completion processing failure tracking information.
	CompletionFailures *CompletionFailureSnapshot
}

// GetStatusInfo reads the daemon status file with PID liveness validation.
// Returns StatusInfo with Running=false if daemon is not running.
// Detects stale status files from crashed daemons.
func GetStatusInfo() StatusInfo {
	info := StatusInfo{
		Status: "stopped",
	}

	// Try to read status file
	status, err := ReadStatusFile()
	if err != nil {
		// No status file — daemon not running
		return info
	}

	// Check PID liveness
	if status.PID > 0 && !isProcessAlive(status.PID) {
		// Stale file from crashed daemon
		info.StaleFile = true
		info.PID = status.PID
		return info
	}

	// Daemon is alive
	info.Running = true
	info.PID = status.PID
	info.Status = status.Status
	info.Capacity = status.Capacity
	info.LastPoll = status.LastPoll
	info.LastSpawn = status.LastSpawn
	info.LastCompletion = status.LastCompletion
	info.ReadyCount = status.ReadyCount
	info.Verification = status.Verification
	info.CompletionFailures = status.CompletionFailures

	return info
}

// FormatStatusInfo formats StatusInfo for terminal display.
func FormatStatusInfo(info StatusInfo) string {
	if info.StaleFile {
		return fmt.Sprintf("Daemon: stopped (stale status file from PID %d — process is dead)", info.PID)
	}

	if !info.Running {
		return "Daemon: stopped"
	}

	result := fmt.Sprintf("Daemon: %s (PID %d)\n", info.Status, info.PID)
	result += fmt.Sprintf("  Capacity:     %d/%d agents active (%d available)\n",
		info.Capacity.Active, info.Capacity.Max, info.Capacity.Available)
	result += fmt.Sprintf("  Ready queue:  %d issues\n", info.ReadyCount)

	if !info.LastPoll.IsZero() {
		result += fmt.Sprintf("  Last poll:    %s (%s ago)\n",
			info.LastPoll.Format("15:04:05"), formatDuration(time.Since(info.LastPoll)))
	}
	if !info.LastSpawn.IsZero() {
		result += fmt.Sprintf("  Last spawn:   %s (%s ago)\n",
			info.LastSpawn.Format("15:04:05"), formatDuration(time.Since(info.LastSpawn)))
	}
	if !info.LastCompletion.IsZero() {
		result += fmt.Sprintf("  Last complete: %s (%s ago)\n",
			info.LastCompletion.Format("15:04:05"), formatDuration(time.Since(info.LastCompletion)))
	}

	if info.Verification != nil && info.Verification.IsPaused {
		result += fmt.Sprintf("  Verification: PAUSED (%d/%d unverified)\n",
			info.Verification.CompletionsSinceVerification, info.Verification.Threshold)
	}

	if info.CompletionFailures != nil && info.CompletionFailures.ConsecutiveFailures > 0 {
		result += fmt.Sprintf("  Completion failures: %d consecutive (%s)\n",
			info.CompletionFailures.ConsecutiveFailures, info.CompletionFailures.LastFailureReason)
	}

	return result
}

// formatDuration formats a duration for human display.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m"
		}
		return fmt.Sprintf("%dm", mins)
	}
	hours := int(d.Hours())
	if hours == 1 {
		return "1h"
	}
	return fmt.Sprintf("%dh", hours)
}
