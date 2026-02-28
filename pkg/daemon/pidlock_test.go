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

	// First acquire should succeed
	lock1, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer lock1.Release()

	// Second acquire (same process) should fail because flock is held
	_, err = AcquirePIDLockAt(lockPath)
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

	// Write a stale PID file (no flock held). This simulates a crash
	// where the process died and the kernel released the flock.
	stalePID := 4194300 // Near max PID on most systems
	os.WriteFile(lockPath, []byte(strconv.Itoa(stalePID)), 0644)

	// Should succeed because no flock is held (file content is irrelevant)
	lock, err := AcquirePIDLockAt(lockPath)
	if err != nil {
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

	// Write garbage content (no flock held)
	os.WriteFile(lockPath, []byte("not-a-pid"), 0644)

	// Should succeed - no flock is held regardless of file content
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

	// Release should close fd (releasing flock) and remove the file
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

	// Should be able to acquire again (flock released when fd closed)
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

func TestPIDLock_ReleaseDoubleCallSafe(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	lock, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Double release should not panic or error
	if err := lock.Release(); err != nil {
		t.Fatalf("first release failed: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("second release should not error, got: %v", err)
	}
}

func TestPIDLock_FlockReleasedOnProcessExit(t *testing.T) {
	// This test verifies that a stale PID file (from a crashed process)
	// does NOT prevent lock acquisition, because flock is kernel-managed
	// and released on process exit.
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// Simulate crash: write PID file but don't hold flock
	os.WriteFile(lockPath, []byte("12345"), 0644)

	// Lock acquisition should succeed (no flock held)
	lock, err := AcquirePIDLockAt(lockPath)
	if err != nil {
		t.Fatalf("should acquire lock over stale file: %v", err)
	}
	defer lock.Release()
}

func TestReadPIDFromLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// No file → 0
	if pid := ReadPIDFromLockFileAt(lockPath); pid != 0 {
		t.Errorf("expected 0 for missing file, got %d", pid)
	}

	// Valid PID content
	os.WriteFile(lockPath, []byte("42"), 0644)
	if pid := ReadPIDFromLockFileAt(lockPath); pid != 42 {
		t.Errorf("expected 42, got %d", pid)
	}

	// Invalid content → 0
	os.WriteFile(lockPath, []byte("not-a-pid"), 0644)
	if pid := ReadPIDFromLockFileAt(lockPath); pid != 0 {
		t.Errorf("expected 0 for invalid content, got %d", pid)
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
