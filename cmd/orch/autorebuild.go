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
func isRebuildInProgress(lockPath string) bool {
	_, err := os.Stat(lockPath)
	return err == nil
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
		// Log the error but continue with stale binary
		fmt.Fprintf(os.Stderr, "⚠️  Auto-rebuild failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "   Continuing with stale binary. Run 'make install' manually.\n")
	}
}
// test comment Thu Jan  8 15:56:54 PST 2026
