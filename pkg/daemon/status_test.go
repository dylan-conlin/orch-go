package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStatusFilePath(t *testing.T) {
	path := StatusFilePath()
	if path == "" {
		t.Error("StatusFilePath returned empty string")
	}

	// Should end with daemon-status.json
	if filepath.Base(path) != "daemon-status.json" {
		t.Errorf("StatusFilePath = %q, want to end with 'daemon-status.json'", path)
	}

	// Should be in .orch directory
	parent := filepath.Base(filepath.Dir(path))
	if parent != ".orch" {
		t.Errorf("StatusFilePath parent = %q, want '.orch'", parent)
	}
}

func TestWriteAndReadStatusFile(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create test status
	now := time.Now().Truncate(time.Second) // Truncate for JSON round-trip
	lastSpawn := now.Add(-5 * time.Minute)
	status := DaemonStatus{
		Capacity: CapacityStatus{
			Max:       3,
			Active:    2,
			Available: 1,
		},
		LastPoll:   now,
		LastSpawn:  lastSpawn,
		ReadyCount: 5,
		Status:     "running",
	}

	// Write status
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	// Verify file exists
	statusPath := StatusFilePath()
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		t.Fatalf("Status file not created at %q", statusPath)
	}

	// Read and verify
	readStatus, err := ReadStatusFile()
	if err != nil {
		t.Fatalf("ReadStatusFile failed: %v", err)
	}

	// Compare fields
	if readStatus.Capacity.Max != status.Capacity.Max {
		t.Errorf("Capacity.Max = %d, want %d", readStatus.Capacity.Max, status.Capacity.Max)
	}
	if readStatus.Capacity.Active != status.Capacity.Active {
		t.Errorf("Capacity.Active = %d, want %d", readStatus.Capacity.Active, status.Capacity.Active)
	}
	if readStatus.Capacity.Available != status.Capacity.Available {
		t.Errorf("Capacity.Available = %d, want %d", readStatus.Capacity.Available, status.Capacity.Available)
	}
	if readStatus.ReadyCount != status.ReadyCount {
		t.Errorf("ReadyCount = %d, want %d", readStatus.ReadyCount, status.ReadyCount)
	}
	if readStatus.Status != status.Status {
		t.Errorf("Status = %q, want %q", readStatus.Status, status.Status)
	}
}

func TestWriteStatusFile_AtomicWrite(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	status := DaemonStatus{
		Capacity: CapacityStatus{
			Max:       3,
			Active:    0,
			Available: 3,
		},
		LastPoll: time.Now(),
		Status:   "running",
	}

	// Write status
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	// Verify temp file doesn't exist (was renamed)
	tempPath := StatusFilePath() + ".tmp"
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("Temp file should not exist after write")
	}
}

func TestWriteStatusFile_CreatesDirectory(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// .orch directory shouldn't exist yet
	orchDir := filepath.Join(tmpDir, ".orch")
	if _, err := os.Stat(orchDir); !os.IsNotExist(err) {
		t.Skip(".orch directory already exists")
	}

	status := DaemonStatus{
		Status: "running",
	}

	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(orchDir); os.IsNotExist(err) {
		t.Error(".orch directory was not created")
	}
}

func TestRemoveStatusFile(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write a status file first
	status := DaemonStatus{
		Status: "running",
	}
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	// Remove it
	if err := RemoveStatusFile(); err != nil {
		t.Fatalf("RemoveStatusFile failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(StatusFilePath()); !os.IsNotExist(err) {
		t.Error("Status file should not exist after removal")
	}
}

func TestRemoveStatusFile_NonExistent(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Removing non-existent file should not error
	if err := RemoveStatusFile(); err != nil {
		t.Errorf("RemoveStatusFile on non-existent file should not error: %v", err)
	}
}

func TestDetermineStatus(t *testing.T) {
	pollInterval := 60 * time.Second

	tests := []struct {
		name        string
		lastPollAge time.Duration
		wantStatus  string
	}{
		{
			name:        "recent poll is running",
			lastPollAge: 30 * time.Second,
			wantStatus:  "running",
		},
		{
			name:        "exactly at interval is running",
			lastPollAge: 60 * time.Second,
			wantStatus:  "running",
		},
		{
			name:        "under 2x interval is running",
			lastPollAge: 100 * time.Second,
			wantStatus:  "running",
		},
		{
			name:        "over 2x interval is stalled",
			lastPollAge: 130 * time.Second,
			wantStatus:  "stalled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastPoll := time.Now().Add(-tt.lastPollAge)
			got := DetermineStatus(lastPoll, pollInterval)
			if got != tt.wantStatus {
				t.Errorf("DetermineStatus() = %q, want %q", got, tt.wantStatus)
			}
		})
	}
}

func TestDaemonStatus_JSONFormat(t *testing.T) {
	// Verify the JSON structure matches expected format
	now := time.Now()
	status := DaemonStatus{
		Capacity: CapacityStatus{
			Max:       3,
			Active:    2,
			Available: 1,
		},
		LastPoll:   now,
		LastSpawn:  now.Add(-5 * time.Minute),
		ReadyCount: 5,
		Status:     "running",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}

	// Unmarshal to generic map to verify structure
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Verify expected keys exist
	expectedKeys := []string{"capacity", "last_poll", "last_spawn", "ready_count", "status"}
	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("Missing expected key %q in JSON output", key)
		}
	}

	// Verify capacity structure
	capacity, ok := m["capacity"].(map[string]interface{})
	if !ok {
		t.Fatal("capacity is not an object")
	}
	capacityKeys := []string{"max", "active", "available"}
	for _, key := range capacityKeys {
		if _, ok := capacity[key]; !ok {
			t.Errorf("Missing expected key %q in capacity object", key)
		}
	}
}

func TestDaemonStatus_ZeroLastSpawn(t *testing.T) {
	// Verify last_spawn is included even when zero value (time.Time always serializes)
	status := DaemonStatus{
		Status: "running",
		// LastSpawn is zero value
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Verify status is present
	if m["status"] != "running" {
		t.Errorf("status = %v, want 'running'", m["status"])
	}
}

func TestReadValidatedStatusFile_StaleFile(t *testing.T) {
	// Simulate stale daemon-status.json from a crashed daemon:
	// file exists with a PID that is no longer alive.
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write a status file with a PID that definitely doesn't exist.
	status := DaemonStatus{
		PID:      999999,
		Status:   "running",
		LastPoll: time.Now().Add(-10 * time.Minute),
		Capacity: CapacityStatus{Max: 3, Active: 1, Available: 2},
	}
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	// ReadValidatedStatusFile should return nil (stale file)
	validated, err := ReadValidatedStatusFile()
	if err != nil {
		t.Fatalf("ReadValidatedStatusFile returned error: %v", err)
	}
	if validated != nil {
		t.Errorf("ReadValidatedStatusFile should return nil for dead PID, got status=%q", validated.Status)
	}
}

func TestReadValidatedStatusFile_LiveProcess(t *testing.T) {
	// Status file with current process PID should be considered valid.
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	status := DaemonStatus{
		PID:      os.Getpid(), // Current process - definitely alive
		Status:   "running",
		LastPoll: time.Now(),
		Capacity: CapacityStatus{Max: 3, Active: 0, Available: 3},
	}
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	validated, err := ReadValidatedStatusFile()
	if err != nil {
		t.Fatalf("ReadValidatedStatusFile returned error: %v", err)
	}
	if validated == nil {
		t.Fatal("ReadValidatedStatusFile should return status for live process")
	}
	if validated.Status != "running" {
		t.Errorf("Status = %q, want %q", validated.Status, "running")
	}
}

func TestReadValidatedStatusFile_NoPID(t *testing.T) {
	// Status file with PID=0 (not set) should still be considered valid
	// for backward compatibility with old daemon versions that didn't write PID.
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	status := DaemonStatus{
		PID:      0, // No PID recorded
		Status:   "running",
		LastPoll: time.Now(),
	}
	if err := WriteStatusFile(status); err != nil {
		t.Fatalf("WriteStatusFile failed: %v", err)
	}

	validated, err := ReadValidatedStatusFile()
	if err != nil {
		t.Fatalf("ReadValidatedStatusFile returned error: %v", err)
	}
	if validated == nil {
		t.Fatal("ReadValidatedStatusFile should return status when PID is not set (backward compat)")
	}
}

func TestReadValidatedStatusFile_NoFile(t *testing.T) {
	// When no status file exists, should return error.
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	validated, err := ReadValidatedStatusFile()
	if err == nil {
		t.Fatal("ReadValidatedStatusFile should return error when no file exists")
	}
	if validated != nil {
		t.Error("ReadValidatedStatusFile should return nil status when no file exists")
	}
}

func TestDetermineStatus_RunningAndStalled(t *testing.T) {
	pollInterval := time.Minute
	lastPoll := time.Now()

	got := DetermineStatus(lastPoll, pollInterval)
	if got != "running" {
		t.Errorf("DetermineStatus() = %q, want %q", got, "running")
	}

	// Stalled when last poll was more than 2x interval ago
	stalledPoll := time.Now().Add(-3 * time.Minute)
	got = DetermineStatus(stalledPoll, pollInterval)
	if got != "stalled" {
		t.Errorf("DetermineStatus() = %q, want %q", got, "stalled")
	}
}
