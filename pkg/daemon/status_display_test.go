package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetStatusInfo_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	info := GetStatusInfo()
	if info.Running {
		t.Error("Should not be running when no status file exists")
	}
	if info.Status != "stopped" {
		t.Errorf("Status = %q, want 'stopped'", info.Status)
	}
}

func TestGetStatusInfo_StaleFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write status with dead PID
	status := DaemonStatus{
		PID:      999999, // Definitely dead
		Status:   "running",
		LastPoll: time.Now(),
		Capacity: CapacityStatus{Max: 3, Active: 1, Available: 2},
	}
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	info := GetStatusInfo()
	if info.Running {
		t.Error("Should not be running with dead PID")
	}
	if !info.StaleFile {
		t.Error("Should detect stale file")
	}
	if info.PID != 999999 {
		t.Errorf("PID = %d, want 999999", info.PID)
	}
}

func TestGetStatusInfo_LiveProcess(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	now := time.Now().Truncate(time.Second)
	status := DaemonStatus{
		PID:      os.Getpid(), // Current process — definitely alive
		Status:   "running",
		LastPoll: now,
		Capacity: CapacityStatus{Max: 5, Active: 2, Available: 3},
		ReadyCount: 7,
	}
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	info := GetStatusInfo()
	if !info.Running {
		t.Error("Should be running with live PID")
	}
	if info.StaleFile {
		t.Error("Should not detect stale file for live process")
	}
	if info.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", info.PID, os.Getpid())
	}
	if info.Status != "running" {
		t.Errorf("Status = %q, want 'running'", info.Status)
	}
	if info.Capacity.Max != 5 {
		t.Errorf("Capacity.Max = %d, want 5", info.Capacity.Max)
	}
	if info.ReadyCount != 7 {
		t.Errorf("ReadyCount = %d, want 7", info.ReadyCount)
	}
}

func TestFormatStatusInfo_Stopped(t *testing.T) {
	info := StatusInfo{Status: "stopped"}
	result := FormatStatusInfo(info)
	if result != "Daemon: stopped" {
		t.Errorf("FormatStatusInfo = %q, want 'Daemon: stopped'", result)
	}
}

func TestFormatStatusInfo_StaleFile(t *testing.T) {
	info := StatusInfo{StaleFile: true, PID: 12345}
	result := FormatStatusInfo(info)
	if !strings.Contains(result, "stale status file") {
		t.Errorf("FormatStatusInfo should mention stale file, got %q", result)
	}
	if !strings.Contains(result, "12345") {
		t.Errorf("FormatStatusInfo should mention PID, got %q", result)
	}
}

func TestFormatStatusInfo_Running(t *testing.T) {
	info := StatusInfo{
		Running: true,
		PID:     42,
		Status:  "running",
		Capacity: CapacityStatus{
			Max: 3, Active: 1, Available: 2,
		},
		ReadyCount: 5,
		LastPoll:   time.Now().Add(-30 * time.Second),
	}
	result := FormatStatusInfo(info)
	if !strings.Contains(result, "running (PID 42)") {
		t.Errorf("FormatStatusInfo should show running PID, got %q", result)
	}
	if !strings.Contains(result, "1/3 agents active") {
		t.Errorf("FormatStatusInfo should show capacity, got %q", result)
	}
	if !strings.Contains(result, "5 issues") {
		t.Errorf("FormatStatusInfo should show ready count, got %q", result)
	}
}

func TestFormatStatusInfo_Paused(t *testing.T) {
	info := StatusInfo{
		Running: true,
		PID:     42,
		Status:  "paused",
		Capacity: CapacityStatus{Max: 3},
		Verification: &VerificationStatusSnapshot{
			IsPaused:                     true,
			CompletionsSinceVerification: 5,
			Threshold:                    3,
		},
	}
	result := FormatStatusInfo(info)
	if !strings.Contains(result, "PAUSED") {
		t.Errorf("FormatStatusInfo should show PAUSED for verification pause, got %q", result)
	}
}

// --- SIGKILL restart tests: PID lock file fallback ---

func TestGetStatusInfo_SIGKILLRestart_StaleStatusLiveLock(t *testing.T) {
	// Simulates: old daemon SIGKILL'd (status file has dead PID),
	// new daemon started (PID lock has current/live PID).
	// Expected: status = "starting", not "stopped (stale)".
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write stale status file from old daemon (dead PID)
	status := DaemonStatus{
		PID:      999999, // Dead PID
		Status:   "running",
		LastPoll: time.Now().Add(-10 * time.Minute),
		Capacity: CapacityStatus{Max: 3, Active: 1, Available: 2},
	}
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	// Write PID lock file with live PID (simulates new daemon after restart)
	lockPath := filepath.Join(tmpDir, ".orch", "daemon.pid")
	if err := os.WriteFile(lockPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		t.Fatalf("failed to write PID lock: %v", err)
	}

	info := GetStatusInfo()
	if !info.Running {
		t.Error("Should be running — new daemon detected via PID lock")
	}
	if info.StaleFile {
		t.Error("Should not be stale — new daemon is starting")
	}
	if info.Status != "starting" {
		t.Errorf("Status = %q, want 'starting'", info.Status)
	}
	if info.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d (from lock file)", info.PID, os.Getpid())
	}
}

func TestGetStatusInfo_SIGKILLRestart_NoStatusLiveLock(t *testing.T) {
	// Simulates: stale status file was already cleaned up,
	// but new daemon is running (PID lock has live PID).
	// Expected: status = "starting", not "stopped".
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// No status file exists, but PID lock file has live PID
	orchDir := filepath.Join(tmpDir, ".orch")
	os.MkdirAll(orchDir, 0755)
	lockPath := filepath.Join(orchDir, "daemon.pid")
	if err := os.WriteFile(lockPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		t.Fatalf("failed to write PID lock: %v", err)
	}

	info := GetStatusInfo()
	if !info.Running {
		t.Error("Should be running — daemon detected via PID lock")
	}
	if info.Status != "starting" {
		t.Errorf("Status = %q, want 'starting'", info.Status)
	}
	if info.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", info.PID, os.Getpid())
	}
}

func TestGetStatusInfo_TrulyStopped_StaleStatusDeadLock(t *testing.T) {
	// Simulates: daemon crashed, no restart. Both status file and PID lock
	// have dead PIDs. Expected: stale file detected, truly stopped.
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write stale status file
	status := DaemonStatus{
		PID:    999999,
		Status: "running",
	}
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	// Write PID lock file with same dead PID
	lockPath := filepath.Join(tmpDir, ".orch", "daemon.pid")
	if err := os.WriteFile(lockPath, []byte("999999"), 0644); err != nil {
		t.Fatalf("failed to write PID lock: %v", err)
	}

	info := GetStatusInfo()
	if info.Running {
		t.Error("Should not be running — both status and lock have dead PID")
	}
	if !info.StaleFile {
		t.Error("Should detect stale file")
	}
}

func TestGetStatusInfo_TrulyStopped_StaleStatusNoLock(t *testing.T) {
	// Simulates: daemon stopped gracefully (lock removed), but status
	// file cleanup failed. Expected: stale file detected.
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write stale status file with dead PID, no lock file
	status := DaemonStatus{
		PID:    999999,
		Status: "running",
	}
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	info := GetStatusInfo()
	if info.Running {
		t.Error("Should not be running — no lock file, dead status PID")
	}
	if !info.StaleFile {
		t.Error("Should detect stale file")
	}
}

func TestFormatStatusInfo_Starting(t *testing.T) {
	info := StatusInfo{
		Running: true,
		PID:     42,
		Status:  "starting",
	}
	result := FormatStatusInfo(info)
	if !strings.Contains(result, "starting") {
		t.Errorf("FormatStatusInfo should show starting, got %q", result)
	}
	if !strings.Contains(result, "42") {
		t.Errorf("FormatStatusInfo should show PID, got %q", result)
	}
	// Should NOT show capacity (not yet available during startup)
	if strings.Contains(result, "agents active") {
		t.Errorf("FormatStatusInfo should not show capacity during startup, got %q", result)
	}
}

func TestIsDaemonRunningFromLock_LiveProcess(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// Write current PID
	os.WriteFile(lockPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

	running, pid := IsDaemonRunningFromLockAt(lockPath)
	if !running {
		t.Error("Should detect running daemon from lock file")
	}
	if pid != os.Getpid() {
		t.Errorf("PID = %d, want %d", pid, os.Getpid())
	}
}

func TestIsDaemonRunningFromLock_DeadProcess(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	// Write dead PID
	os.WriteFile(lockPath, []byte("999999"), 0644)

	running, _ := IsDaemonRunningFromLockAt(lockPath)
	if running {
		t.Error("Should not detect running daemon for dead PID")
	}
}

func TestIsDaemonRunningFromLock_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.pid")

	running, _ := IsDaemonRunningFromLockAt(lockPath)
	if running {
		t.Error("Should not detect running daemon when no lock file")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5m"},
		{1 * time.Minute, "1m"},
		{2 * time.Hour, "2h"},
		{1 * time.Hour, "1h"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
