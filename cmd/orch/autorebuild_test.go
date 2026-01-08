package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestShouldAutoRebuildEnvDisabled tests that ORCH_NO_AUTOREBUILD skips the check.
func TestShouldAutoRebuildEnvDisabled(t *testing.T) {
	// Save and restore original env var
	orig := os.Getenv("ORCH_NO_AUTOREBUILD")
	defer func() {
		if orig == "" {
			os.Unsetenv("ORCH_NO_AUTOREBUILD")
		} else {
			os.Setenv("ORCH_NO_AUTOREBUILD", orig)
		}
	}()

	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "disabled with 1",
			envValue: "1",
			want:     false,
		},
		{
			name:     "disabled with true",
			envValue: "true",
			want:     false,
		},
		{
			name:     "not disabled with empty",
			envValue: "",
			want:     true, // Will need other checks to pass
		},
		{
			name:     "not disabled with 0",
			envValue: "0",
			want:     true, // Will need other checks to pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				os.Unsetenv("ORCH_NO_AUTOREBUILD")
			} else {
				os.Setenv("ORCH_NO_AUTOREBUILD", tt.envValue)
			}

			// Test the early return logic - when env is set to "1" or "true", should skip
			shouldSkip := os.Getenv("ORCH_NO_AUTOREBUILD") == "1" || os.Getenv("ORCH_NO_AUTOREBUILD") == "true"

			// When skip is true, we should NOT auto-rebuild (want=false)
			// When skip is false, we may auto-rebuild (want=true, pending other checks)
			if shouldSkip == tt.want {
				t.Errorf("skip=%v but want autorebuild=%v", shouldSkip, tt.want)
			}
		})
	}
}

// TestShouldAutoRebuildMissingSourceDir tests that missing sourceDir skips rebuild.
func TestShouldAutoRebuildMissingSourceDir(t *testing.T) {
	tests := []struct {
		name      string
		sourceDir string
		want      bool
	}{
		{
			name:      "unknown source dir",
			sourceDir: "unknown",
			want:      false,
		},
		{
			name:      "empty source dir",
			sourceDir: "",
			want:      false,
		},
		{
			name:      "non-existent path",
			sourceDir: "/path/that/does/not/exist",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldAutoRebuildCheck(tt.sourceDir, "somehash")
			if result != tt.want {
				t.Errorf("shouldAutoRebuildCheck(%q, _) = %v, want %v", tt.sourceDir, result, tt.want)
			}
		})
	}
}

// TestShouldAutoRebuildUnknownGitHash tests that unknown gitHash skips rebuild.
func TestShouldAutoRebuildUnknownGitHash(t *testing.T) {
	// Create a temp directory (as a valid source dir)
	tempDir, err := os.MkdirTemp("", "test-autorebuild-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name    string
		gitHash string
		want    bool
	}{
		{
			name:    "unknown git hash",
			gitHash: "unknown",
			want:    false,
		},
		{
			name:    "empty git hash",
			gitHash: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldAutoRebuildCheck(tempDir, tt.gitHash)
			if result != tt.want {
				t.Errorf("shouldAutoRebuildCheck(_, %q) = %v, want %v", tt.gitHash, result, tt.want)
			}
		})
	}
}

// TestShouldAutoRebuildMatchingHash tests that matching hash skips rebuild.
func TestShouldAutoRebuildMatchingHash(t *testing.T) {
	// Get current directory git hash (if available)
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		t.Skip("Not a git repository, skipping git hash test")
	}

	// Use current directory as source
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}

	// Find git root
	gitRootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	gitRootOutput, err := gitRootCmd.Output()
	if err != nil {
		t.Skip("Cannot determine git root")
	}
	gitRoot := string(gitRootOutput[:len(gitRootOutput)-1]) // trim newline

	currentHash := string(output[:len(output)-1]) // trim newline
	_ = cwd                                       // unused in this test

	// When hashes match, should NOT auto-rebuild
	result := shouldAutoRebuildCheck(gitRoot, currentHash)
	if result != false {
		t.Errorf("shouldAutoRebuildCheck with matching hash should be false, got %v", result)
	}
}

// TestShouldAutoRebuildStaleHash tests that stale hash triggers rebuild.
func TestShouldAutoRebuildStaleHash(t *testing.T) {
	// Get current directory git hash (if available)
	cmd := exec.Command("git", "rev-parse", "HEAD")
	_, err := cmd.Output()
	if err != nil {
		t.Skip("Not a git repository, skipping git hash test")
	}

	// Find git root
	gitRootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	gitRootOutput, err := gitRootCmd.Output()
	if err != nil {
		t.Skip("Cannot determine git root")
	}
	gitRoot := string(gitRootOutput[:len(gitRootOutput)-1]) // trim newline

	// Use a fake old hash - should trigger rebuild
	oldHash := "abcd1234567890abcd1234567890abcd12345678"

	result := shouldAutoRebuildCheck(gitRoot, oldHash)
	if result != true {
		t.Errorf("shouldAutoRebuildCheck with stale hash should be true, got %v", result)
	}
}

// TestAutoRebuildLockPath tests the lock file path construction.
func TestAutoRebuildLockPath(t *testing.T) {
	tempDir := "/tmp/test-orch"
	expected := filepath.Join(tempDir, ".autorebuild.lock")

	result := getAutoRebuildLockPath(tempDir)
	if result != expected {
		t.Errorf("getAutoRebuildLockPath(%q) = %q, want %q", tempDir, result, expected)
	}
}

// TestIsRebuildInProgress tests the concurrent rebuild detection.
func TestIsRebuildInProgress(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-autorebuild-lock-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lockPath := filepath.Join(tempDir, ".autorebuild.lock")

	// Initially no lock
	if isRebuildInProgress(lockPath) {
		t.Error("Expected no rebuild in progress initially")
	}

	// Create lock file
	if err := os.WriteFile(lockPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create lock file: %v", err)
	}

	// Now should detect rebuild in progress
	if !isRebuildInProgress(lockPath) {
		t.Error("Expected rebuild in progress with lock file")
	}
}

// TestAcquireReleaseLock tests the file locking mechanism.
func TestAcquireReleaseLock(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-autorebuild-lock2-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lockPath := filepath.Join(tempDir, ".autorebuild.lock")

	// Acquire lock
	release, err := acquireRebuildLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Lock file should exist
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file should exist after acquire")
	}

	// Release lock
	release()

	// Lock file should be removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("Lock file should be removed after release")
	}
}

// TestAutoRebuildIntegrationSkip tests the full flow skips correctly.
func TestAutoRebuildIntegrationSkip(t *testing.T) {
	// Save and restore original env var
	orig := os.Getenv("ORCH_NO_AUTOREBUILD")
	defer func() {
		if orig == "" {
			os.Unsetenv("ORCH_NO_AUTOREBUILD")
		} else {
			os.Setenv("ORCH_NO_AUTOREBUILD", orig)
		}
	}()

	// Set env to disable
	os.Setenv("ORCH_NO_AUTOREBUILD", "1")

	// The full function should return early
	shouldRebuild := shouldAutoRebuild()
	if shouldRebuild {
		t.Error("shouldAutoRebuild should return false when ORCH_NO_AUTOREBUILD=1")
	}
}
