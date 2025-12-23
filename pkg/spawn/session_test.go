package spawn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReadSessionID(t *testing.T) {
	// Create temp directory as workspace
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sessionID := "session-abc123"

	// Write session ID
	if err := WriteSessionID(tmpDir, sessionID); err != nil {
		t.Fatalf("WriteSessionID failed: %v", err)
	}

	// Verify file exists
	sessionFile := filepath.Join(tmpDir, SessionIDFilename)
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Fatalf("session ID file not created")
	}

	// Read session ID
	readID := ReadSessionID(tmpDir)
	if readID != sessionID {
		t.Errorf("ReadSessionID returned %q, want %q", readID, sessionID)
	}
}

func TestWriteSessionID_EmptyID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Writing empty ID should succeed but not create file
	if err := WriteSessionID(tmpDir, ""); err != nil {
		t.Fatalf("WriteSessionID with empty ID failed: %v", err)
	}

	// File should not exist
	sessionFile := filepath.Join(tmpDir, SessionIDFilename)
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Errorf("session ID file should not exist for empty ID")
	}
}

func TestReadSessionID_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Reading from non-existent file should return empty string
	readID := ReadSessionID(tmpDir)
	if readID != "" {
		t.Errorf("ReadSessionID returned %q for non-existent file, want empty string", readID)
	}
}

func TestSessionIDPath(t *testing.T) {
	workspace := "/some/workspace/path"
	expected := filepath.Join(workspace, SessionIDFilename)
	got := SessionIDPath(workspace)
	if got != expected {
		t.Errorf("SessionIDPath returned %q, want %q", got, expected)
	}
}

func TestWriteReadTier(t *testing.T) {
	// Create temp directory as workspace
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tier := TierLight

	// Write tier
	if err := WriteTier(tmpDir, tier); err != nil {
		t.Fatalf("WriteTier failed: %v", err)
	}

	// Verify file exists
	tierFile := filepath.Join(tmpDir, TierFilename)
	if _, err := os.Stat(tierFile); os.IsNotExist(err) {
		t.Fatalf("tier file not created")
	}

	// Read tier
	readTier := ReadTier(tmpDir)
	if readTier != tier {
		t.Errorf("ReadTier returned %q, want %q", readTier, tier)
	}
}

func TestWriteTier_EmptyTier(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Writing empty tier should succeed but not create file
	if err := WriteTier(tmpDir, ""); err != nil {
		t.Fatalf("WriteTier with empty tier failed: %v", err)
	}

	// File should not exist
	tierFile := filepath.Join(tmpDir, TierFilename)
	if _, err := os.Stat(tierFile); !os.IsNotExist(err) {
		t.Errorf("tier file should not exist for empty tier")
	}
}

func TestReadTier_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Reading from non-existent file should return TierFull (conservative default)
	readTier := ReadTier(tmpDir)
	if readTier != TierFull {
		t.Errorf("ReadTier returned %q for non-existent file, want %q (conservative default)", readTier, TierFull)
	}
}

func TestTierPath(t *testing.T) {
	workspace := "/some/workspace/path"
	expected := filepath.Join(workspace, TierFilename)
	got := TierPath(workspace)
	if got != expected {
		t.Errorf("TierPath returned %q, want %q", got, expected)
	}
}
