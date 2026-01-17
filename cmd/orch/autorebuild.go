// Package main provides auto-rebuild functionality for the orch CLI.
// When the binary is stale (git hash doesn't match current HEAD), it automatically
// rebuilds and re-executes itself to ensure users always run the latest code.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// shouldAutoRebuild checks if the binary needs to be rebuilt.
// Returns true if rebuild is needed, false otherwise.
func shouldAutoRebuild() bool {
	// Check if disabled via environment variable
	if os.Getenv("ORCH_NO_AUTOREBUILD") == "1" || os.Getenv("ORCH_NO_AUTOREBUILD") == "true" {
		return false
	}

	return shouldAutoRebuildCheck(sourceDir, gitHash)
}

// shouldAutoRebuildCheck is the testable core of shouldAutoRebuild.
// It checks if the binary is stale by comparing the embedded git hash
// against the current HEAD of the source directory.
func shouldAutoRebuildCheck(srcDir, embeddedHash string) bool {
	// Skip if sourceDir not embedded (dev build)
	if srcDir == "" || srcDir == "unknown" {
		return false
	}

	// Skip if gitHash not embedded (dev build)
	if embeddedHash == "" || embeddedHash == "unknown" {
		return false
	}

	// Check if source directory exists
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return false
	}

	// Get current git hash from source directory
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = srcDir
	output, err := cmd.Output()
	if err != nil {
		// Cannot determine current hash, skip rebuild
		return false
	}

	currentHash := strings.TrimSpace(string(output))

	// Compare hashes - if different, rebuild is needed
	return currentHash != embeddedHash
}

// getAutoRebuildLockPath returns the path to the rebuild lock file.
func getAutoRebuildLockPath(srcDir string) string {
	return filepath.Join(srcDir, ".autorebuild.lock")
}

// isRebuildInProgress checks if another rebuild is currently in progress.
// It reads the PID from the lock file and verifies the process is still running.
// If the lock file exists but the process is dead, it removes the stale lock.
func isRebuildInProgress(lockPath string) bool {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		// Lock file doesn't exist or can't be read
		return false
	}

	// Parse PID from lock file
	pidStr := strings.TrimSpace(string(data))
	pid := 0
	if _, err := fmt.Sscanf(pidStr, "%d", &pid); err != nil || pid <= 0 {
		// Invalid PID in lock file - remove stale lock
		os.Remove(lockPath)
		return false
	}

	// Check if process is still running
	// On Unix, sending signal 0 checks if process exists without actually signaling it
	process, err := os.FindProcess(pid)
	if err != nil {
		// Can't find process - remove stale lock
		os.Remove(lockPath)
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to actually check
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist or we don't have permission
		// Common errors indicating process is dead:
		// - syscall.ESRCH: No such process (direct syscall error)
		// - "os: process already finished" (Go's wrapper)
		// - "operation not permitted" could mean process exists but different user
		errStr := err.Error()
		if err == syscall.ESRCH || strings.Contains(errStr, "process already finished") ||
			strings.Contains(errStr, "no such process") {
			os.Remove(lockPath)
			return false
		}
		// EPERM/"operation not permitted" means process exists but we can't signal it
		// This is conservative but safe - treat as in progress
	}

	// Process is still running
	return true
}

// acquireRebuildLock attempts to acquire the rebuild lock.
// Returns a release function and nil on success, or nil and error on failure.
func acquireRebuildLock(lockPath string) (func(), error) {
	// Try to create lock file exclusively
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("rebuild already in progress")
		}
		return nil, err
	}

	// Write PID to lock file for debugging
	fmt.Fprintf(f, "%d\n", os.Getpid())
	f.Close()

	// Return release function
	return func() {
		os.Remove(lockPath)
	}, nil
}

// autoRebuildAndReexec rebuilds the binary and re-executes it.
// This function does not return on success (it replaces the current process).
// On failure, it returns an error and the caller should continue with the stale binary.
func autoRebuildAndReexec() error {
	lockPath := getAutoRebuildLockPath(sourceDir)

	// Check if rebuild is already in progress
	if isRebuildInProgress(lockPath) {
		return fmt.Errorf("rebuild already in progress")
	}

	// Acquire lock
	release, err := acquireRebuildLock(lockPath)
	if err != nil {
		return fmt.Errorf("failed to acquire rebuild lock: %w", err)
	}
	defer release()

	// Print status message
	fmt.Fprintf(os.Stderr, "🔄 Binary is stale, auto-rebuilding...\n")

	// Run make install
	cmd := exec.Command("make", "install")
	cmd.Dir = sourceDir
	cmd.Stdout = os.Stderr // Send build output to stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rebuild failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Rebuild complete, re-executing...\n")

	// Get path to current executable
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Re-resolve the executable path (in case it's a symlink that was updated)
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Re-exec with the same arguments
	// syscall.Exec replaces the current process, so this won't return on success
	return syscall.Exec(executable, os.Args, os.Environ())
}

// hasJSONFlag checks if the --json flag is present in os.Args.
// This is used to suppress auto-rebuild warnings when JSON output is requested,
// since warnings to stderr would break JSON parsers that capture both streams.
func hasJSONFlag() bool {
	for _, arg := range os.Args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

// maybeAutoRebuild checks if auto-rebuild is needed and performs it.
// This should be called at the top of main() before rootCmd.Execute().
// On successful rebuild, this function does not return (process is replaced).
// On failure or no rebuild needed, returns normally.
func maybeAutoRebuild() {
	if !shouldAutoRebuild() {
		return
	}

	// Attempt rebuild - if it fails, continue with stale binary
	if err := autoRebuildAndReexec(); err != nil {
		// Suppress warning if --json flag is present to avoid breaking JSON parsers
		// that capture both stdout and stderr (e.g., orch status --json 2>&1 | jq)
		if !hasJSONFlag() {
			// Log the error but continue with stale binary
			fmt.Fprintf(os.Stderr, "⚠️  Auto-rebuild failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "   Continuing with stale binary. Run 'make install' manually.\n")
		}
	}
}
// test comment Thu Jan  8 15:56:54 PST 2026
