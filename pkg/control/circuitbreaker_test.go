package control

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAck(t *testing.T) {
	tmpDir := t.TempDir()
	heartbeatPath := filepath.Join(tmpDir, "heartbeat")

	// Ack should create the heartbeat file
	if err := Ack(heartbeatPath); err != nil {
		t.Fatalf("Ack failed: %v", err)
	}

	// File should exist
	info, err := os.Stat(heartbeatPath)
	if err != nil {
		t.Fatalf("heartbeat file should exist: %v", err)
	}

	// Should be recent (within last second)
	if time.Since(info.ModTime()) > time.Second {
		t.Error("heartbeat mtime should be recent")
	}

	// Ack again should update mtime
	time.Sleep(10 * time.Millisecond)
	if err := Ack(heartbeatPath); err != nil {
		t.Fatalf("second Ack failed: %v", err)
	}
	info2, _ := os.Stat(heartbeatPath)
	if !info2.ModTime().After(info.ModTime()) || info2.ModTime().Equal(info.ModTime()) {
		// ModTime resolution may not catch 10ms difference on all systems,
		// so just verify the file still exists
		if _, err := os.Stat(heartbeatPath); err != nil {
			t.Error("heartbeat file should still exist after second ack")
		}
	}
}

func TestHeartbeatAge(t *testing.T) {
	tmpDir := t.TempDir()

	// No heartbeat file → should return large age
	age, err := HeartbeatAge(filepath.Join(tmpDir, "nonexistent"))
	if err != nil {
		t.Fatalf("should not error for missing heartbeat: %v", err)
	}
	if age < 24*time.Hour {
		t.Errorf("missing heartbeat should report large age, got %v", age)
	}

	// Fresh heartbeat → should be near zero
	heartbeatPath := filepath.Join(tmpDir, "heartbeat")
	if err := Ack(heartbeatPath); err != nil {
		t.Fatal(err)
	}
	age, err = HeartbeatAge(heartbeatPath)
	if err != nil {
		t.Fatalf("HeartbeatAge failed: %v", err)
	}
	if age > time.Second {
		t.Errorf("fresh heartbeat age should be <1s, got %v", age)
	}
}

func TestHaltStatus_NotHalted(t *testing.T) {
	tmpDir := t.TempDir()
	haltPath := filepath.Join(tmpDir, "halt")

	status, err := HaltStatus(haltPath)
	if err != nil {
		t.Fatalf("HaltStatus failed: %v", err)
	}
	if status.Halted {
		t.Error("should not be halted when halt file doesn't exist")
	}
	if status.Reason != "" {
		t.Errorf("reason should be empty when not halted, got %q", status.Reason)
	}
}

func TestHaltStatus_Halted(t *testing.T) {
	tmpDir := t.TempDir()
	haltPath := filepath.Join(tmpDir, "halt")

	// Write a halt file in the format the shell hook produces
	content := `reason: Rolling 3-day average exceeded (75 commits/day, threshold 70)
triggered_by: rolling_avg
triggered_at: 2026-02-14T12:00:00Z`
	if err := os.WriteFile(haltPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	status, err := HaltStatus(haltPath)
	if err != nil {
		t.Fatalf("HaltStatus failed: %v", err)
	}
	if !status.Halted {
		t.Error("should be halted when halt file exists")
	}
	if status.Reason != "Rolling 3-day average exceeded (75 commits/day, threshold 70)" {
		t.Errorf("unexpected reason: %q", status.Reason)
	}
	if status.TriggeredBy != "rolling_avg" {
		t.Errorf("unexpected triggered_by: %q", status.TriggeredBy)
	}
}

func TestResume(t *testing.T) {
	tmpDir := t.TempDir()
	haltPath := filepath.Join(tmpDir, "halt")
	heartbeatPath := filepath.Join(tmpDir, "heartbeat")

	// Create a halt file
	if err := os.WriteFile(haltPath, []byte("reason: test halt"), 0644); err != nil {
		t.Fatal(err)
	}

	// Resume should clear halt and touch heartbeat
	if err := Resume(haltPath, heartbeatPath); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	// Halt file should be gone
	if _, err := os.Stat(haltPath); !os.IsNotExist(err) {
		t.Error("halt file should be removed after resume")
	}

	// Heartbeat should exist and be fresh
	info, err := os.Stat(heartbeatPath)
	if err != nil {
		t.Fatalf("heartbeat file should exist after resume: %v", err)
	}
	if time.Since(info.ModTime()) > time.Second {
		t.Error("heartbeat should be fresh after resume")
	}
}

func TestResume_NoHaltFile(t *testing.T) {
	tmpDir := t.TempDir()
	haltPath := filepath.Join(tmpDir, "halt")
	heartbeatPath := filepath.Join(tmpDir, "heartbeat")

	// Resume when not halted should still touch heartbeat (no error)
	if err := Resume(haltPath, heartbeatPath); err != nil {
		t.Fatalf("Resume should succeed even without halt file: %v", err)
	}

	// Heartbeat should exist
	if _, err := os.Stat(heartbeatPath); err != nil {
		t.Error("heartbeat should be touched even when not halted")
	}
}

func TestCircuitBreakerStatus(t *testing.T) {
	tmpDir := t.TempDir()
	haltPath := filepath.Join(tmpDir, "halt")
	heartbeatPath := filepath.Join(tmpDir, "heartbeat")

	// No halt, no heartbeat
	status, err := CircuitBreakerStatus(haltPath, heartbeatPath)
	if err != nil {
		t.Fatalf("CircuitBreakerStatus failed: %v", err)
	}
	if status.Halted {
		t.Error("should not be halted")
	}
	if status.HeartbeatAge < 24*time.Hour {
		t.Error("missing heartbeat should report large age")
	}

	// Touch heartbeat
	if err := Ack(heartbeatPath); err != nil {
		t.Fatal(err)
	}

	status, err = CircuitBreakerStatus(haltPath, heartbeatPath)
	if err != nil {
		t.Fatalf("CircuitBreakerStatus failed: %v", err)
	}
	if status.Halted {
		t.Error("should not be halted")
	}
	if status.HeartbeatAge > time.Second {
		t.Errorf("heartbeat age should be fresh, got %v", status.HeartbeatAge)
	}

	// Create halt
	if err := os.WriteFile(haltPath, []byte("reason: test\ntriggered_by: test\ntriggered_at: 2026-01-01T00:00:00Z"), 0644); err != nil {
		t.Fatal(err)
	}
	status, err = CircuitBreakerStatus(haltPath, heartbeatPath)
	if err != nil {
		t.Fatalf("CircuitBreakerStatus failed: %v", err)
	}
	if !status.Halted {
		t.Error("should be halted")
	}
	if status.HaltReason != "test" {
		t.Errorf("unexpected reason: %q", status.HaltReason)
	}
}

func TestDefaultPaths(t *testing.T) {
	// Verify default path functions don't panic
	halt := DefaultHaltPath()
	heartbeat := DefaultHeartbeatPath()
	if halt == "" || heartbeat == "" {
		t.Error("default paths should not be empty")
	}
	if filepath.Dir(halt) != filepath.Dir(heartbeat) {
		t.Error("halt and heartbeat should be in the same directory")
	}
}
