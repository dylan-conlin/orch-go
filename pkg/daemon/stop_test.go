package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestStopDaemon_NoPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	err := StopDaemonAt(lockPath, StopOptions{})
	if err == nil {
		t.Fatal("expected error when no PID file exists")
	}
	if err != ErrNoDaemonRunning {
		t.Errorf("expected ErrNoDaemonRunning, got: %v", err)
	}
}

func TestStopDaemon_StalePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// Write a PID that doesn't exist (near max PID)
	os.WriteFile(lockPath, []byte("4194300"), 0644)

	err := StopDaemonAt(lockPath, StopOptions{})
	if err == nil {
		t.Fatal("expected error for stale PID")
	}
	if err != ErrNoDaemonRunning {
		t.Errorf("expected ErrNoDaemonRunning, got: %v", err)
	}
}

func TestStopDaemon_InvalidPIDContent(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	os.WriteFile(lockPath, []byte("not-a-pid"), 0644)

	err := StopDaemonAt(lockPath, StopOptions{})
	if err == nil {
		t.Fatal("expected error for invalid PID content")
	}
	if err != ErrNoDaemonRunning {
		t.Errorf("expected ErrNoDaemonRunning, got: %v", err)
	}
}

func TestStopDaemon_SendsSignalToLiveProcess(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// Write our own PID - we're alive, so the stop function should attempt to signal us.
	// We use a short timeout and expect a timeout error since we can't actually
	// kill ourselves in a test.
	os.WriteFile(lockPath, []byte(strconv.Itoa(os.Getpid())), 0644)

	// Use very short timeout to avoid blocking test
	err := StopDaemonAt(lockPath, StopOptions{
		TimeoutMs: 100,
		// Skip the actual kill signal to avoid killing the test process
		SignalFunc: func(pid int) error { return nil },
	})

	// Should timeout waiting for the process to exit (since we're still alive)
	if err == nil {
		t.Fatal("expected timeout error when process doesn't exit")
	}
	if err != ErrStopTimeout {
		t.Errorf("expected ErrStopTimeout, got: %v", err)
	}
}

func TestStopDaemon_EmptyPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	os.WriteFile(lockPath, []byte(""), 0644)

	err := StopDaemonAt(lockPath, StopOptions{})
	if err != ErrNoDaemonRunning {
		t.Errorf("expected ErrNoDaemonRunning, got: %v", err)
	}
}
