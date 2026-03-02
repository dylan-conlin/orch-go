package daemon

import (
	"os"
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
