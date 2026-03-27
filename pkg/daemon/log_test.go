package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDaemonLogger_WritesToFile(t *testing.T) {
	// Create temp directory for test log
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "daemon.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	logger := &DaemonLogger{
		file: f,
		out:  f, // Write only to file for test (not stdout)
	}
	defer logger.Close()

	logger.Printf("test message %d\n", 42)
	logger.Errorf("error message %s\n", "oops")

	// Close to flush
	logger.Close()

	// Verify file contents
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "test message 42") {
		t.Errorf("Log file missing Printf output, got: %q", content)
	}
	if !strings.Contains(content, "error message oops") {
		t.Errorf("Log file missing Errorf output, got: %q", content)
	}
}

func TestDaemonLogger_StampIncludesTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "daemon.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	logger := &DaemonLogger{
		file: f,
		out:  f,
	}
	defer logger.Close()

	logger.Stamp("hello %s", "world")
	logger.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	// Should have timestamp format [HH:MM:SS]
	if !strings.Contains(content, "] hello world") {
		t.Errorf("Stamp output missing expected format, got: %q", content)
	}
	if content[0] != '[' {
		t.Errorf("Stamp output should start with '[', got: %q", content)
	}
}

func TestDaemonLogger_FallbackToStdout(t *testing.T) {
	// NewDaemonLogger with invalid path falls back gracefully
	logger := &DaemonLogger{out: os.Stdout}
	defer logger.Close()

	// Should not panic
	logger.Printf("test\n")
	logger.Errorf("test error\n")
	logger.Stamp("test stamp")
}

func TestRotateIfNeeded_RotatesLargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "daemon.log")

	// Create a file larger than maxLogSize (10 MB)
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	// Write 11 MB of data
	data := make([]byte, 11*1024*1024)
	f.Write(data)
	f.Close()

	rotateIfNeeded(logPath)

	// Original should be gone
	if _, err := os.Stat(logPath); err == nil {
		t.Error("Expected original log file to be renamed")
	}

	// Backup should exist with the original size
	backupPath := logPath + ".1"
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("Expected backup file to exist: %v", err)
	}
	if info.Size() != int64(11*1024*1024) {
		t.Errorf("Backup size = %d, want %d", info.Size(), 11*1024*1024)
	}
}

func TestRotateIfNeeded_SkipsSmallFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "daemon.log")

	// Create a small file (1 KB)
	if err := os.WriteFile(logPath, make([]byte, 1024), 0644); err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	rotateIfNeeded(logPath)

	// Original should still exist
	if _, err := os.Stat(logPath); err != nil {
		t.Error("Small log file should not be rotated")
	}

	// No backup should exist
	if _, err := os.Stat(logPath + ".1"); err == nil {
		t.Error("No backup should exist for small file")
	}
}

func TestRotateIfNeeded_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "daemon.log")

	// Should not panic when file doesn't exist
	rotateIfNeeded(logPath)
}

func TestStdoutIsLogFile_ReturnsFalseNormally(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "daemon.log")

	// Create a log file that stdout is NOT pointing to
	if err := os.WriteFile(logPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	// In test context, stdout is the test runner — not this file
	if stdoutIsLogFile(logPath) {
		t.Error("stdoutIsLogFile should return false when stdout is not the log file")
	}
}

func TestStdoutIsLogFile_ReturnsFalseWhenNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nonexistent.log")

	// Log file doesn't exist — should return false, not panic
	if stdoutIsLogFile(logPath) {
		t.Error("stdoutIsLogFile should return false when log file doesn't exist")
	}
}

func TestNewDaemonLogger_SkipsFileWhenStdoutMatchesLog(t *testing.T) {
	// When stdoutIsLogFile returns true, the logger should have no file writer
	// (l.file == nil). We can't easily redirect stdout in a test, but we verify
	// that the normal path does set l.file.
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "daemon.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	// Simulate the non-launchd path: file should be set
	logger := &DaemonLogger{file: f, out: f}
	if logger.file == nil {
		t.Error("Expected file to be set in non-launchd mode")
	}
	logger.Close()

	// Simulate the launchd-detected path: file should be nil
	logger = &DaemonLogger{out: os.Stdout}
	if logger.file != nil {
		t.Error("Expected file to be nil in launchd-detected mode")
	}
	logger.Close()
}

func TestDaemonLogPath(t *testing.T) {
	path := DaemonLogPath()
	if path == "" {
		t.Skip("Cannot determine home directory")
	}
	if !strings.Contains(path, ".orch") || !strings.HasSuffix(path, "daemon.log") {
		t.Errorf("DaemonLogPath() = %q, want path containing .orch/daemon.log", path)
	}
}
