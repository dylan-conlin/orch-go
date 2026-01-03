package shell

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDefaultRunner_Run(t *testing.T) {
	r := New()

	// Test basic command execution
	output, err := r.Run(context.Background(), "echo", "hello", "world")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	expected := "hello world"
	if got := strings.TrimSpace(string(output)); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestDefaultRunner_Output(t *testing.T) {
	r := New()

	// Test stdout-only output
	output, err := r.Output(context.Background(), "echo", "test")
	if err != nil {
		t.Fatalf("Output failed: %v", err)
	}

	if got := strings.TrimSpace(string(output)); got != "test" {
		t.Errorf("expected %q, got %q", "test", got)
	}
}

func TestDefaultRunner_WithDir(t *testing.T) {
	tmpDir := t.TempDir()
	r := New(WithDir(tmpDir))

	// Test that command runs in specified directory
	output, err := r.Output(context.Background(), "pwd")
	if err != nil {
		t.Fatalf("Output failed: %v", err)
	}

	// Resolve symlinks for comparison (macOS uses /private/var/folders)
	expectedDir, _ := filepath.EvalSymlinks(tmpDir)
	gotDir := strings.TrimSpace(string(output))
	gotDir, _ = filepath.EvalSymlinks(gotDir)

	if gotDir != expectedDir {
		t.Errorf("expected dir %q, got %q", expectedDir, gotDir)
	}
}

func TestDefaultRunner_WithEnv(t *testing.T) {
	r := New(WithEnv([]string{"TEST_VAR=test_value"}))

	// Test custom environment variable
	var output []byte
	var err error

	if runtime.GOOS == "windows" {
		output, err = r.Output(context.Background(), "cmd", "/c", "echo", "%TEST_VAR%")
	} else {
		output, err = r.Output(context.Background(), "sh", "-c", "echo $TEST_VAR")
	}

	if err != nil {
		t.Fatalf("Output failed: %v", err)
	}

	if got := strings.TrimSpace(string(output)); got != "test_value" {
		t.Errorf("expected %q, got %q", "test_value", got)
	}
}

func TestDefaultRunner_WithTimeout(t *testing.T) {
	r := New(WithTimeout(100 * time.Millisecond))

	// This command should timeout
	_, err := r.Run(context.Background(), "sleep", "10")
	if err == nil {
		t.Error("expected timeout error, got nil")
	}

	// The error can be a context deadline exceeded, signal killed, or exit code -1
	// depending on OS and timing. Any of these indicate the timeout worked.
	errStr := err.Error()
	isTimeoutRelated := strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "signal: killed") ||
		strings.Contains(errStr, "exit code -1")
	if !isTimeoutRelated {
		t.Errorf("expected timeout-related error, got: %v", err)
	}
}

func TestDefaultRunner_ContextCancellation(t *testing.T) {
	r := New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := r.Run(ctx, "echo", "test")
	if err == nil {
		t.Error("expected context canceled error, got nil")
	}
}

func TestDefaultRunner_ExitError(t *testing.T) {
	r := New()

	// This command should fail with exit code 1
	_, err := r.Run(context.Background(), "sh", "-c", "exit 1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}

	if exitErr.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode)
	}

	if exitErr.Cmd != "sh" {
		t.Errorf("expected cmd %q, got %q", "sh", exitErr.Cmd)
	}
}

func TestDefaultRunner_RunWithStdin(t *testing.T) {
	r := New()

	// Test stdin input
	stdin := []byte("hello from stdin")
	output, err := r.RunWithStdin(context.Background(), stdin, "cat")
	if err != nil {
		t.Fatalf("RunWithStdin failed: %v", err)
	}

	if string(output) != string(stdin) {
		t.Errorf("expected %q, got %q", string(stdin), string(output))
	}
}

func TestDefaultRunner_Start(t *testing.T) {
	r := New()

	cmd, err := r.Start(context.Background(), "sleep", "0.1")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	pid := cmd.Pid()
	if pid <= 0 {
		t.Errorf("expected positive PID, got %d", pid)
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("Wait failed: %v", err)
	}
}

func TestDefaultRunner_CommandNotFound(t *testing.T) {
	r := New()

	_, err := r.Run(context.Background(), "nonexistent_command_xyz")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestExitError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ExitError
		expected string
	}{
		{
			name: "with stderr",
			err: &ExitError{
				Cmd:      "test",
				ExitCode: 1,
				Stderr:   []byte("error message"),
			},
			expected: `command "test" failed with exit code 1: error message`,
		},
		{
			name: "without stderr",
			err: &ExitError{
				Cmd:      "test",
				ExitCode: 2,
				Stderr:   nil,
			},
			expected: `command "test" failed with exit code 2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNew_DefaultOptions(t *testing.T) {
	r := New()

	// Should be able to execute commands with default config
	_, err := r.Output(context.Background(), "echo", "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDefaultRunner_InheritsEnvironment(t *testing.T) {
	// Set a test environment variable
	os.Setenv("SHELL_TEST_VAR", "inherited")
	defer os.Unsetenv("SHELL_TEST_VAR")

	r := New() // No custom env, should inherit

	var output []byte
	var err error

	if runtime.GOOS == "windows" {
		output, err = r.Output(context.Background(), "cmd", "/c", "echo", "%SHELL_TEST_VAR%")
	} else {
		output, err = r.Output(context.Background(), "sh", "-c", "echo $SHELL_TEST_VAR")
	}

	if err != nil {
		t.Fatalf("Output failed: %v", err)
	}

	if got := strings.TrimSpace(string(output)); got != "inherited" {
		t.Errorf("expected %q, got %q", "inherited", got)
	}
}
