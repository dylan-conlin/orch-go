// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"errors"
	"fmt"
	"syscall"
	"time"
)

// ErrNoDaemonRunning indicates no daemon process was found.
var ErrNoDaemonRunning = errors.New("no daemon running")

// ErrStopTimeout indicates the daemon didn't exit within the timeout.
var ErrStopTimeout = errors.New("daemon did not stop within timeout")

// StopOptions configures daemon stop behavior.
type StopOptions struct {
	// TimeoutMs is the maximum time to wait for the daemon to exit, in milliseconds.
	// Default: 10000 (10 seconds).
	TimeoutMs int

	// SignalFunc overrides the default signal sending function.
	// Used for testing to avoid actually killing processes.
	// Default: sends SIGTERM via syscall.Kill.
	SignalFunc func(pid int) error
}

// StopDaemon stops the running daemon by sending SIGTERM and waiting for exit.
// Uses the default PID lock path.
func StopDaemon(opts StopOptions) error {
	return StopDaemonAt(PIDLockPath(), opts)
}

// StopDaemonAt stops the daemon using the PID from the given lock file path.
// Returns ErrNoDaemonRunning if no daemon is running.
// Returns ErrStopTimeout if the daemon doesn't exit within the timeout.
func StopDaemonAt(lockPath string, opts StopOptions) error {
	// Apply defaults
	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = 10000
	}
	if opts.SignalFunc == nil {
		opts.SignalFunc = func(pid int) error {
			return syscall.Kill(pid, syscall.SIGTERM)
		}
	}

	// Read PID from lock file
	pid := ReadPIDFromLockFileAt(lockPath)
	if pid <= 0 {
		return ErrNoDaemonRunning
	}

	// Check if process is actually alive
	if !isProcessAlive(pid) {
		return ErrNoDaemonRunning
	}

	// Send SIGTERM
	if err := opts.SignalFunc(pid); err != nil {
		return fmt.Errorf("failed to send signal to daemon (PID %d): %w", pid, err)
	}

	// Wait for process to exit
	timeout := time.Duration(opts.TimeoutMs) * time.Millisecond
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		if !isProcessAlive(pid) {
			return nil
		}
		time.Sleep(pollInterval)
	}

	return ErrStopTimeout
}
