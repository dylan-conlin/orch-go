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

	// Verification holds verification tracking information.
	// Omitted if verification tracking is disabled (threshold = 0).
	Verification *VerificationStatusSnapshot `json:"verification,omitempty"`

	// SpawnFailures holds spawn failure tracking information.
	// Surfaced to enable health card alerting when daemon stops spawning due to failures.
	SpawnFailures *SpawnFailureSnapshot `json:"spawn_failures,omitempty"`

	// CompletionFailures holds completion processing failure tracking information.
	// Surfaced to enable health card alerting when daemon stops processing completions.
	CompletionFailures *CompletionFailureSnapshot `json:"completion_failures,omitempty"`

	// KnowledgeHealth holds the latest knowledge health check snapshot.
	// Surfaced to enable dashboard visibility into quick entry accumulation.
	KnowledgeHealth *KnowledgeHealthSnapshot `json:"knowledge_health,omitempty"`
}

// VerificationStatusSnapshot is a snapshot of verification tracking state.
type VerificationStatusSnapshot struct {
	// IsPaused indicates whether the daemon is paused due to verification threshold.
	IsPaused bool `json:"is_paused"`

	// CompletionsSinceVerification is the count of auto-completions since last human verification.
	CompletionsSinceVerification int `json:"completions_since_verification"`

	// Threshold is the maximum auto-completions allowed before pausing.
	Threshold int `json:"threshold"`

	// LastVerification is when the last human verification occurred.
	LastVerification time.Time `json:"last_verification"`

	// RemainingBeforePause is how many more completions are allowed before pause.
	RemainingBeforePause int `json:"remaining_before_pause"`
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
// Returns "paused" if verification pause is active, "stalled" if it appears stuck,
// or "running" if healthy.
func DetermineStatus(lastPoll time.Time, pollInterval time.Duration, verificationPaused bool) string {
	// Check verification pause first (takes precedence over stalled)
	if verificationPaused {
		return "paused"
	}

	// If last poll was more than 2x poll interval ago, consider stalled
	stalledThreshold := pollInterval * 2
	if time.Since(lastPoll) > stalledThreshold {
		return "stalled"
	}

	return "running"
}
