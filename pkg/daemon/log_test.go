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

func TestDaemonLogPath(t *testing.T) {
	path := DaemonLogPath()
	if path == "" {
		t.Skip("Cannot determine home directory")
	}
	if !strings.Contains(path, ".orch") || !strings.HasSuffix(path, "daemon.log") {
		t.Errorf("DaemonLogPath() = %q, want path containing .orch/daemon.log", path)
	}
}
