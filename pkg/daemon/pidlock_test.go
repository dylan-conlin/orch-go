package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAcquirePIDLock_Success(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	lock, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer lock.Release()

	// Verify PID file contains our PID
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("failed to read PID file: %v", err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("PID file content is not a number: %q", string(data))
	}
	if pid != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), pid)
	}
}

func TestAcquirePIDLock_DoubleAcquireFails(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// Write our own PID to simulate a running daemon
	// (since we ARE running, kill(pid,0) will report alive)
	currentPID := os.Getpid()
	os.WriteFile(lockPath, []byte(strconv.Itoa(currentPID)), 0644)

	_, err := AcquirePIDLockAt(lockPath)
	if err == nil {
		t.Fatal("expected error for double acquire, got nil")
	}
	if !isErrDaemonAlreadyRunning(err) {
		t.Errorf("expected ErrDaemonAlreadyRunning, got: %v", err)
	}
}

func TestAcquirePIDLock_StalePIDCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// Write a PID that definitely doesn't exist (PID 2 is typically kernel, but
	// use a very high PID that's almost certainly not running)
	stalePID := 4194300 // Near max PID on most systems
	os.WriteFile(lockPath, []byte(strconv.Itoa(stalePID)), 0644)

	// Should succeed because the stale process isn't alive
	lock, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		// If by extreme coincidence this PID is alive, skip the test
		if isErrDaemonAlreadyRunning(err) {
			t.Skipf("PID %d unexpectedly alive, skipping stale PID test", stalePID)
		}
		t.Fatalf("expected no error, got: %v", err)
	}
	defer lock.Release()

	// Verify it now contains our PID
	data, _ := os.ReadFile(lockPath)
	pid, _ := strconv.Atoi(string(data))
	if pid != os.Getpid() {
		t.Errorf("expected PID %d after stale cleanup, got %d", os.Getpid(), pid)
	}
}

func TestAcquirePIDLock_InvalidPIDContent(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// Write garbage content
	os.WriteFile(lockPath, []byte("not-a-pid"), 0644)

	// Should succeed - invalid content is treated as stale
	lock, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		t.Fatalf("expected no error for invalid PID content, got: %v", err)
	}
	defer lock.Release()
}

func TestPIDLock_Release(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	lock, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Release should remove the file
	if err := lock.Release(); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// File should be gone
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("PID file should be removed after release")
	}
}

func TestPIDLock_ReleaseAllowsReacquire(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// Acquire and release
	lock1, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	if err := lock1.Release(); err != nil {
		t.Fatalf("release failed: %v", err)
	}

	// Should be able to acquire again
	lock2, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		t.Fatalf("reacquire after release failed: %v", err)
	}
	defer lock2.Release()
}

func TestPIDLock_ReleaseNilSafe(t *testing.T) {
	var lock *PIDLock
	if err := lock.Release(); err != nil {
		t.Errorf("nil release should not error, got: %v", err)
	}
}

func TestPIDLock_ReleaseOnlyOwnPID(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	lock, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Overwrite the PID file with a different PID (simulating another process taking over)
	os.WriteFile(lockPath, []byte("99999"), 0644)

	// Release should NOT remove the file since it's not our PID
	lock.Release()

	// File should still exist
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("PID file should NOT be removed when PID doesn't match")
	}
}

func TestIsProcessAlive(t *testing.T) {
	// Current process should be alive
	if !isProcessAlive(os.Getpid()) {
		t.Error("current process should be reported as alive")
	}

	// PID 0 should not be alive
	if isProcessAlive(0) {
		t.Error("PID 0 should not be reported as alive")
	}

	// Negative PID should not be alive
	if isProcessAlive(-1) {
		t.Error("negative PID should not be reported as alive")
	}
}

// isErrDaemonAlreadyRunning checks if an error wraps ErrDaemonAlreadyRunning.
func isErrDaemonAlreadyRunning(err error) bool {
	return errors.Is(err, ErrDaemonAlreadyRunning)
}
