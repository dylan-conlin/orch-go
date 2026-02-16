package control

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHeartbeatAgeHours(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	oldPath := HeartbeatPath
	HeartbeatPath = filepath.Join(tmpDir, "control-heartbeat")
	defer func() { HeartbeatPath = oldPath }()

	// Test case 1: No heartbeat file exists
	age := HeartbeatAgeHours()
	if age != -1 {
		t.Errorf("Expected age -1 when no heartbeat exists, got %.1f", age)
	}

	// Test case 2: Fresh heartbeat (just created)
	if err := Ack(); err != nil {
		t.Fatalf("Failed to create heartbeat: %v", err)
	}

	age = HeartbeatAgeHours()
	if age < 0 || age > 0.1 {
		t.Errorf("Expected fresh heartbeat age ~0, got %.1f", age)
	}

	// Test case 3: Old heartbeat (simulate 25h old)
	// We can't easily test this without manipulating file mtime,
	// but we can at least verify the function doesn't crash
	info, err := os.Stat(HeartbeatPath)
	if err != nil {
		t.Fatalf("Failed to stat heartbeat: %v", err)
	}
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(HeartbeatPath, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to change heartbeat mtime: %v", err)
	}

	age = HeartbeatAgeHours()
	if age < 24.9 || age > 25.1 {
		t.Errorf("Expected age ~25h for old heartbeat, got %.1f", age)
	}

	// Verify file still exists
	if _, err := os.Stat(HeartbeatPath); err != nil {
		t.Errorf("Heartbeat file should still exist after age check: %v", err)
	}

	_ = info // Use info to avoid unused variable error
}

func TestIsHeartbeatStale(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	oldPath := HeartbeatPath
	HeartbeatPath = filepath.Join(tmpDir, "control-heartbeat")
	defer func() { HeartbeatPath = oldPath }()

	// Test case 1: No heartbeat file (should be stale)
	if !IsHeartbeatStale() {
		t.Error("Expected stale=true when no heartbeat exists")
	}

	// Test case 2: Fresh heartbeat (should not be stale)
	if err := Ack(); err != nil {
		t.Fatalf("Failed to create heartbeat: %v", err)
	}

	if IsHeartbeatStale() {
		t.Error("Expected stale=false for fresh heartbeat")
	}

	// Test case 3: Old heartbeat (>24h, should be stale)
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(HeartbeatPath, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to change heartbeat mtime: %v", err)
	}

	if !IsHeartbeatStale() {
		t.Error("Expected stale=true for 25h old heartbeat")
	}

	// Test case 4: Exactly 24h (boundary case)
	// Note: Due to time precision, we can't test exact 24h reliably.
	// The implementation uses age > 24, so anything at or under 24h is not stale.
	// We'll test 23.5h to avoid boundary issues.
	justUnder24h := time.Now().Add(-23*time.Hour - 30*time.Minute)
	if err := os.Chtimes(HeartbeatPath, justUnder24h, justUnder24h); err != nil {
		t.Fatalf("Failed to change heartbeat mtime: %v", err)
	}

	if IsHeartbeatStale() {
		t.Error("Expected stale=false for 23.5h old heartbeat")
	}

	// Test case 5: Slightly over 24h (should be stale)
	slightlyOver := time.Now().Add(-24*time.Hour - 1*time.Minute)
	if err := os.Chtimes(HeartbeatPath, slightlyOver, slightlyOver); err != nil {
		t.Fatalf("Failed to change heartbeat mtime: %v", err)
	}

	if !IsHeartbeatStale() {
		t.Error("Expected stale=true for 24h1m old heartbeat")
	}
}

func TestAck(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	oldPath := HeartbeatPath
	HeartbeatPath = filepath.Join(tmpDir, "control-heartbeat")
	defer func() { HeartbeatPath = oldPath }()

	// Verify file doesn't exist initially
	if _, err := os.Stat(HeartbeatPath); err == nil {
		t.Error("Heartbeat file should not exist initially")
	}

	// First Ack should create file
	if err := Ack(); err != nil {
		t.Fatalf("First Ack failed: %v", err)
	}

	info1, err := os.Stat(HeartbeatPath)
	if err != nil {
		t.Fatalf("Heartbeat file not created: %v", err)
	}

	// Wait a bit and Ack again - should update mtime
	time.Sleep(100 * time.Millisecond)

	if err := Ack(); err != nil {
		t.Fatalf("Second Ack failed: %v", err)
	}

	info2, err := os.Stat(HeartbeatPath)
	if err != nil {
		t.Fatalf("Heartbeat file disappeared: %v", err)
	}

	// mtime should be updated
	if !info2.ModTime().After(info1.ModTime()) {
		t.Error("Second Ack should update mtime")
	}
}
