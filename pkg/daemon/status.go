// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DaemonStatus represents the current state of the daemon.
// This is written to ~/.orch/daemon-status.json on each poll cycle
// to enable serve.go to expose daemon health without IPC.
type DaemonStatus struct {
	// PID is the process ID of the running daemon.
	PID int `json:"pid,omitempty"`

	// Capacity holds the agent pool capacity information.
	Capacity CapacityStatus `json:"capacity"`

	// LastPoll is the timestamp of the last poll cycle.
	LastPoll time.Time `json:"last_poll"`

	// LastSpawn is the timestamp of the last successful spawn.
	// Zero value if no spawns have occurred in this daemon run.
	LastSpawn time.Time `json:"last_spawn"`

	// LastCompletion is the timestamp of the last auto-completion.
	// Zero value if no completions have occurred in this daemon run.
	LastCompletion time.Time `json:"last_completion,omitempty"`

	// ReadyCount is the number of issues ready to be processed.
	ReadyCount int `json:"ready_count"`

	// Status indicates the daemon's operational state.
	// Values: "running", "stalled", "paused"
	Status string `json:"status"`

	// CompletionFailures holds completion processing failure tracking information.
	// Surfaced to enable health card alerting when daemon stops processing completions.
	CompletionFailures *CompletionFailureSnapshot `json:"completion_failures,omitempty"`


	// PhaseTimeout holds the latest phase timeout detection snapshot.
	// Surfaced to show unresponsive agent count in daemon status and dashboard.
	PhaseTimeout *PhaseTimeoutSnapshot `json:"phase_timeout,omitempty"`

	// QuestionDetection holds the latest QUESTION phase detection snapshot.
	// Surfaced to show agents waiting for user input in daemon status and dashboard.
	QuestionDetection *QuestionDetectionSnapshot `json:"question_detection,omitempty"`

	// AgreementCheck holds the latest agreement check snapshot.
	// Surfaced to show agreement health in daemon status and dashboard.
	AgreementCheck *AgreementCheckSnapshot `json:"agreement_check,omitempty"`

	// BeadsHealth holds the latest beads health snapshot.
	// Surfaced to show issue/code health trends in daemon status and dashboard.
	BeadsHealth *BeadsHealthSnapshot `json:"beads_health,omitempty"`

	// Comprehension holds the comprehension queue snapshot.
	// Surfaced so sketchybar widget can read queue depth without a slow bd CLI call.
	Comprehension *ComprehensionSnapshot `json:"comprehension,omitempty"`

}

// CapacityStatus holds agent pool capacity information.
type CapacityStatus struct {
	// Max is the maximum number of concurrent agents.
	Max int `json:"max"`

	// Active is the number of currently active agents.
	Active int `json:"active"`

	// Available is the number of slots available for spawning.
	Available int `json:"available"`
}

// ComprehensionSnapshot is a point-in-time snapshot of the comprehension queue.
type ComprehensionSnapshot struct {
	// Count is the number of comprehension:unread items (plus legacy pending).
	Count int `json:"count"`

	// Threshold is the configured maximum before spawning pauses.
	Threshold int `json:"threshold"`
}

// StatusFilePath returns the path to the daemon status file.
// Default: ~/.orch/daemon-status.json
func StatusFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home not available
		return ".orch/daemon-status.json"
	}
	return filepath.Join(homeDir, ".orch", "daemon-status.json")
}

// WriteStatusFile writes the daemon status to the status file atomically.
// Uses temp file + rename pattern to ensure atomic writes.
func WriteStatusFile(status DaemonStatus) error {
	statusPath := StatusFilePath()

	// Ensure directory exists
	dir := filepath.Dir(statusPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create status directory: %w", err)
	}

	// Marshal status to JSON with indentation for readability
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	// Write to temp file first (atomic write pattern)
	tempPath := statusPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp status file: %w", err)
	}

	// Rename temp file to final path (atomic on most filesystems)
	if err := os.Rename(tempPath, statusPath); err != nil {
		// Clean up temp file on rename failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename status file: %w", err)
	}

	return nil
}

// ReadStatusFile reads the daemon status from the status file.
// Returns an error if the file doesn't exist or can't be parsed.
func ReadStatusFile() (*DaemonStatus, error) {
	statusPath := StatusFilePath()

	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read status file: %w", err)
	}

	var status DaemonStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status file: %w", err)
	}

	return &status, nil
}

// ReadValidatedStatusFile reads the daemon status file and validates that the
// daemon process is actually alive. Returns nil, nil if the file exists but
// the daemon process is dead (stale file from unclean shutdown).
// Falls back to PID lock file when status file is missing or stale,
// returning a minimal "starting" status for the SIGKILL restart window.
func ReadValidatedStatusFile() (*DaemonStatus, error) {
	status, err := ReadStatusFile()
	if err != nil {
		// No status file — check PID lock as fallback (SIGKILL restart window)
		if running, pid := IsDaemonRunningFromLock(); running {
			return &DaemonStatus{
				PID:      pid,
				Status:   "starting",
				LastPoll: time.Now(),
			}, nil
		}
		return nil, err
	}

	// If PID is recorded and the process is dead, check for restarted daemon
	if status.PID > 0 && !isProcessAlive(status.PID) {
		if running, pid := IsDaemonRunningFromLock(); running && pid != status.PID {
			return &DaemonStatus{
				PID:      pid,
				Status:   "starting",
				LastPoll: time.Now(),
			}, nil
		}
		return nil, nil
	}

	return status, nil
}

// RemoveStatusFile removes the daemon status file.
// Called when the daemon shuts down cleanly.
func RemoveStatusFile() error {
	statusPath := StatusFilePath()
	if err := os.Remove(statusPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove status file: %w", err)
	}
	return nil
}

// DetermineStatus determines the daemon status based on operational metrics.
// Returns "stalled" if it appears stuck, or "running" if healthy.
func DetermineStatus(lastPoll time.Time, pollInterval time.Duration, _ bool) string {
	// If last poll was more than 2x poll interval ago, consider stalled
	stalledThreshold := pollInterval * 2
	if time.Since(lastPoll) > stalledThreshold {
		return "stalled"
	}

	return "running"
}
