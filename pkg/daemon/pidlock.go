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

// PIDLock manages daemon singleton enforcement using flock(2).
// The lock is acquired by opening the lock file and calling flock(LOCK_EX|LOCK_NB).
// The kernel automatically releases the lock when the process exits (even on crash),
// eliminating both the TOCTOU race and stale lock cleanup problems of the old
// read-check-write PID file approach.
//
// The PID is written into the locked file as a secondary artifact for status reporting
// and daemon stop commands — it is NOT the locking mechanism.
type PIDLock struct {
	path string
	pid  int
	file *os.File // held open to maintain flock
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
// daemon instance holds the flock.
func AcquirePIDLock() (*PIDLock, error) {
	return AcquirePIDLockAt(PIDLockPath())
}

// AcquirePIDLockAt attempts to acquire a PID lock at the given path using flock(2).
// This variant is used for testing with custom paths.
//
// The lock is atomic: flock(LOCK_EX|LOCK_NB) either succeeds immediately or fails
// with EWOULDBLOCK if another process holds the lock. No TOCTOU race is possible.
// The kernel releases the lock automatically when the file descriptor is closed
// (including on crash/SIGKILL), so stale lock files are never a problem.
func AcquirePIDLockAt(lockPath string) (*PIDLock, error) {
	// Ensure directory exists
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create PID lock directory: %w", err)
	}

	// Open lock file (create if needed, writable for PID content)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open PID lock file: %w", err)
	}

	// Non-blocking exclusive lock — this is the actual singleton enforcement.
	// LOCK_EX: exclusive lock (only one holder)
	// LOCK_NB: non-blocking (fail immediately if held by another process)
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		// Read existing PID for error message (best-effort)
		existingPID := readPIDFromFile(f)
		f.Close()
		return nil, fmt.Errorf("%w: PID %d", ErrDaemonAlreadyRunning, existingPID)
	}

	// Write our PID into the locked file (for status reporting, not for locking)
	currentPID := os.Getpid()
	f.Truncate(0)
	f.Seek(0, 0)
	fmt.Fprintf(f, "%d", currentPID)
	f.Sync()

	return &PIDLock{
		path: lockPath,
		pid:  currentPID,
		file: f,
	}, nil
}

// Release closes the file descriptor (which releases the flock) and removes the lock file.
func (l *PIDLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}

	// Close the file descriptor — this releases the flock atomically
	l.file.Close()
	l.file = nil

	// Remove the PID file (best-effort cleanup)
	os.Remove(l.path)

	return nil
}

// ReadPIDFromLockFile reads the PID from the lock file at the default path.
// Returns 0 if the file doesn't exist or contains invalid content.
// Used by daemon stop/status commands to find the running daemon's PID.
func ReadPIDFromLockFile() int {
	return ReadPIDFromLockFileAt(PIDLockPath())
}

// ReadPIDFromLockFileAt reads the PID from a lock file at the given path.
func ReadPIDFromLockFileAt(lockPath string) int {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return 0
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0
	}
	return pid
}

// IsDaemonRunningFromLock checks the PID lock file for a running daemon process.
// Returns (true, pid) if the lock file contains a PID for a live process.
// Returns (false, 0) if the lock file doesn't exist, is empty, or has a dead PID.
// Used as a fallback when the status file is missing or stale (e.g., after SIGKILL
// where the deferred status file cleanup was skipped).
func IsDaemonRunningFromLock() (bool, int) {
	return IsDaemonRunningFromLockAt(PIDLockPath())
}

// IsDaemonRunningFromLockAt checks a PID lock file at the given path.
func IsDaemonRunningFromLockAt(lockPath string) (bool, int) {
	pid := ReadPIDFromLockFileAt(lockPath)
	if pid <= 0 {
		return false, 0
	}
	if !isProcessAlive(pid) {
		return false, 0
	}
	return true, pid
}

// readPIDFromFile reads the PID from an already-open file.
func readPIDFromFile(f *os.File) int {
	f.Seek(0, 0)
	buf := make([]byte, 32)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return 0
	}
	pidStr := strings.TrimSpace(string(buf[:n]))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0
	}
	return pid
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
