// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// ErrDaemonAlreadyRunning indicates another daemon instance is active.
var ErrDaemonAlreadyRunning = errors.New("daemon already running")

// PIDLock manages a PID file to ensure only one daemon instance runs at a time.
// The lock is acquired by writing the current process PID to the lock file.
// Stale lock files (from crashed processes) are detected and cleaned up.
type PIDLock struct {
	path string
	pid  int
}

// PIDLockPath returns the path to the daemon PID lock file.
// Default: ~/.orch/daemon.pid
func PIDLockPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".orch/daemon.pid"
	}
	return filepath.Join(homeDir, ".orch", "daemon.pid")
}

// AcquirePIDLock attempts to acquire the daemon PID lock.
// Returns ErrDaemonAlreadyRunning (wrapped with the existing PID) if another
// daemon instance is alive. Cleans up stale PID files from crashed processes.
func AcquirePIDLock() (*PIDLock, error) {
	return AcquirePIDLockAt(PIDLockPath())
}

// AcquirePIDLockAt attempts to acquire a PID lock at the given path.
// This variant is used for testing with custom paths.
func AcquirePIDLockAt(lockPath string) (*PIDLock, error) {
	// Ensure directory exists
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create PID lock directory: %w", err)
	}

	// Check for existing PID file
	data, err := os.ReadFile(lockPath)
	if err == nil {
		// PID file exists - check if process is alive
		pidStr := strings.TrimSpace(string(data))
		if existingPID, parseErr := strconv.Atoi(pidStr); parseErr == nil {
			if isProcessAlive(existingPID) {
				return nil, fmt.Errorf("%w: PID %d", ErrDaemonAlreadyRunning, existingPID)
			}
			// Process is dead - stale PID file, will be overwritten
		}
		// PID file exists but content is invalid or process is dead - overwrite
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read PID lock file: %w", err)
	}

	// Write current PID
	currentPID := os.Getpid()
	pidContent := strconv.Itoa(currentPID)
	if err := os.WriteFile(lockPath, []byte(pidContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write PID lock file: %w", err)
	}

	// Verify we actually wrote our PID (guard against race with another daemon starting)
	verifyData, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to verify PID lock file: %w", err)
	}
	if strings.TrimSpace(string(verifyData)) != pidContent {
		return nil, fmt.Errorf("%w: another instance acquired the lock during startup", ErrDaemonAlreadyRunning)
	}

	return &PIDLock{
		path: lockPath,
		pid:  currentPID,
	}, nil
}

// Release removes the PID lock file.
// Only removes the file if it still contains our PID (safety check).
func (l *PIDLock) Release() error {
	if l == nil {
		return nil
	}

	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already cleaned up
		}
		return fmt.Errorf("failed to read PID lock for release: %w", err)
	}

	// Only remove if it's still our PID
	pidStr := strings.TrimSpace(string(data))
	if pid, err := strconv.Atoi(pidStr); err == nil && pid == l.pid {
		if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove PID lock file: %w", err)
		}
	}

	return nil
}

// IsProcessAlive checks if a process with the given PID is running.
// Uses kill(pid, 0) which checks for process existence without sending a signal.
func IsProcessAlive(pid int) bool {
	return isProcessAlive(pid)
}

// isProcessAlive is the unexported implementation.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	// nil means process exists and we have permission to signal it
	// EPERM means process exists but we don't have permission (still alive)
	// ESRCH means process does not exist
	return err == nil || errors.Is(err, syscall.EPERM)
}
