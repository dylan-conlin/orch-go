// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// VerificationTracker tracks completions since last human verification
// and manages pause state when threshold is reached.
//
// Human verification is defined as a manual `orch complete` invocation
// (not daemon marking ready-for-review). This enforces the verifiability-first
// constraint by pausing autonomous operation after N agents are marked
// ready-for-review without human review.
type VerificationTracker struct {
	mu sync.RWMutex

	// completionsSinceVerification tracks how many agents have been marked
	// ready-for-review since the last human verification.
	completionsSinceVerification int

	// lastVerification is when the last human verification occurred.
	// This is when Dylan manually ran `orch complete`.
	lastVerification time.Time

	// isPaused indicates whether the daemon is paused due to reaching
	// the verification threshold.
	isPaused bool

	// threshold is the maximum number of agents that can be marked ready-for-review
	// before pausing for human verification. Default is 3.
	threshold int
}

// NewVerificationTracker creates a new VerificationTracker with the given threshold.
// If threshold is 0, verification tracking is disabled (never pauses).
func NewVerificationTracker(threshold int) *VerificationTracker {
	return &VerificationTracker{
		threshold:                    threshold,
		lastVerification:             time.Now(), // Start with current time
		isPaused:                     false,
		completionsSinceVerification: 0,
	}
}

// RecordCompletion increments the completion counter.
// This should be called when the daemon marks an issue as ready-for-review.
// Returns true if the threshold was reached and daemon should pause.
func (vt *VerificationTracker) RecordCompletion() bool {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	// If threshold is 0, verification tracking is disabled - don't count
	if vt.threshold == 0 {
		return false
	}

	vt.completionsSinceVerification++

	// Check if we've reached the threshold
	if vt.completionsSinceVerification >= vt.threshold {
		vt.isPaused = true
		return true
	}

	return false
}

// RecordHumanVerification resets the completion counter and unpauses the daemon.
// This should be called when Dylan manually runs `orch complete`.
func (vt *VerificationTracker) RecordHumanVerification() {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	vt.completionsSinceVerification = 0
	vt.lastVerification = time.Now()
	vt.isPaused = false
}

// IsPaused returns true if the daemon is paused due to verification threshold.
func (vt *VerificationTracker) IsPaused() bool {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	return vt.isPaused
}

// Status returns the current verification tracking status.
func (vt *VerificationTracker) Status() VerificationStatus {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	return VerificationStatus{
		CompletionsSinceVerification: vt.completionsSinceVerification,
		LastVerification:             vt.lastVerification,
		IsPaused:                     vt.isPaused,
		Threshold:                    vt.threshold,
	}
}

// SeedFromBacklog sets the completion counter to reflect existing
// unverified backlog. Call after construction, before entering the
// main loop, to make the tracker aware of work completed before
// this daemon session started.
func (vt *VerificationTracker) SeedFromBacklog(unverifiedCount int) {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	vt.completionsSinceVerification = unverifiedCount
	if vt.threshold > 0 && unverifiedCount >= vt.threshold {
		vt.isPaused = true
	}
}

// Resume manually unpauses the daemon without resetting the counter.
// Dylan can resume after reviewing completed work even if counter hasn't reset.
// This allows for "I've reviewed, continue" without requiring manual orch complete.
func (vt *VerificationTracker) Resume() {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	// Reset counter and unpause
	vt.completionsSinceVerification = 0
	vt.lastVerification = time.Now()
	vt.isPaused = false
}

// VerificationStatus contains the current verification tracking status.
type VerificationStatus struct {
	CompletionsSinceVerification int
	LastVerification             time.Time
	IsPaused                     bool
	Threshold                    int
}

// IsEnabled returns true if verification tracking is enabled (threshold > 0).
func (vs VerificationStatus) IsEnabled() bool {
	return vs.Threshold > 0
}

// RemainingBeforePause returns how many more completions are allowed before pause.
// Returns -1 if already paused or if tracking is disabled.
func (vs VerificationStatus) RemainingBeforePause() int {
	if !vs.IsEnabled() {
		return -1
	}
	if vs.IsPaused {
		return 0
	}
	remaining := vs.Threshold - vs.CompletionsSinceVerification
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ResumePath returns the path to the resume signal file.
// Default: ~/.orch/daemon-resume.signal
func ResumePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home not available
		return ".orch/daemon-resume.signal"
	}
	return filepath.Join(homeDir, ".orch", "daemon-resume.signal")
}

// VerificationPath returns the path to the human verification signal file.
// Default: ~/.orch/daemon-verification.signal
// This signal is written by `orch complete` to notify the daemon that human
// verification has occurred, which resets the completion counter.
func VerificationPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home not available
		return ".orch/daemon-verification.signal"
	}
	return filepath.Join(homeDir, ".orch", "daemon-verification.signal")
}

// ReadVerificationSignal reads the last human verification timestamp from the signal file.
// Returns zero time if the signal file does not exist or is empty.
func ReadVerificationSignal() (time.Time, error) {
	verificationPath := VerificationPath()
	data, err := os.ReadFile(verificationPath)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to read verification signal: %w", err)
	}

	ts := strings.TrimSpace(string(data))
	if ts == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse verification signal: %w", err)
	}

	return parsed, nil
}

// WriteResumeSignal writes a resume signal file.
// The running daemon will detect this file and resume operation.
func WriteResumeSignal() error {
	resumePath := ResumePath()

	// Ensure directory exists
	dir := filepath.Dir(resumePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create resume signal directory: %w", err)
	}

	// Write signal file with timestamp
	timestamp := time.Now().Format(time.RFC3339)
	if err := os.WriteFile(resumePath, []byte(timestamp), 0644); err != nil {
		return fmt.Errorf("failed to write resume signal: %w", err)
	}

	return nil
}

// WriteVerificationSignal writes a human verification signal file.
// This is called by `orch complete` to notify the daemon that human verification
// has occurred, which should reset the completion counter and unpause the daemon.
func WriteVerificationSignal() error {
	verificationPath := VerificationPath()

	// Ensure directory exists
	dir := filepath.Dir(verificationPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create verification signal directory: %w", err)
	}

	// Write signal file with timestamp
	timestamp := time.Now().Format(time.RFC3339)
	if err := os.WriteFile(verificationPath, []byte(timestamp), 0644); err != nil {
		return fmt.Errorf("failed to write verification signal: %w", err)
	}

	return nil
}

// CheckAndClearResumeSignal checks if a resume signal exists.
// If it does, it removes the signal file and returns true.
// This should be called by the daemon loop to detect resume requests.
func CheckAndClearResumeSignal() (bool, error) {
	resumePath := ResumePath()

	// Check if signal file exists
	if _, err := os.Stat(resumePath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check resume signal: %w", err)
	}

	// Signal exists - remove it atomically
	if err := os.Remove(resumePath); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to remove resume signal: %w", err)
	}

	return true, nil
}

// CheckAndClearVerificationSignal checks if a human verification signal exists.
// If it does, it removes the signal file and returns true.
// This should be called by the daemon loop to detect when `orch complete` has run,
// indicating that human verification has occurred.
func CheckAndClearVerificationSignal() (bool, error) {
	verificationPath := VerificationPath()

	// Check if signal file exists
	if _, err := os.Stat(verificationPath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check verification signal: %w", err)
	}

	// Signal exists - remove it atomically
	if err := os.Remove(verificationPath); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to remove verification signal: %w", err)
	}

	return true, nil
}
