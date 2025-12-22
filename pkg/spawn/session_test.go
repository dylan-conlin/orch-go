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
