package atomicwrite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StaleThreshold is how old a .tmp file must be before cleanup considers it
// stale. This prevents removing temp files that are actively being written.
const StaleThreshold = 5 * time.Second

// CleanupStaleTempFiles removes orphaned .tmp files from the given directories.
// A temp file is considered stale if it is older than StaleThreshold.
//
// This should be called on process startup to clean up temp files left
// behind by previous crashes or interrupted processes.
//
// Returns the number of files cleaned up and any errors encountered.
// Errors are collected but do not stop processing of remaining files.
func CleanupStaleTempFiles(dirs ...string) (int, []error) {
	cleaned := 0
	var errs []error
	now := time.Now()

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Directory doesn't exist, nothing to clean
			}
			errs = append(errs, fmt.Errorf("read dir %s: %w", dir, err))
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !strings.HasSuffix(entry.Name(), TempSuffix) {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				errs = append(errs, fmt.Errorf("stat %s: %w", entry.Name(), err))
				continue
			}

			// Only remove files older than threshold to avoid racing with
			// active writes
			age := now.Sub(info.ModTime())
			if age < StaleThreshold {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				errs = append(errs, fmt.Errorf("remove %s: %w", path, err))
				continue
			}
			cleaned++
		}
	}

	return cleaned, errs
}

// CleanupStaleTempFilesInWorkspaces walks workspace directories and removes
// orphaned .tmp files. Each immediate subdirectory of workspaceRoot is treated
// as a workspace directory to scan.
//
// This handles the common case where spawn workspace files (.session_id.tmp,
// .tier.tmp, .spawn_time.tmp, etc.) are orphaned by crashed agents.
func CleanupStaleTempFilesInWorkspaces(workspaceRoot string) (int, []error) {
	entries, err := os.ReadDir(workspaceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, []error{fmt.Errorf("read workspace root %s: %w", workspaceRoot, err)}
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(workspaceRoot, entry.Name()))
		}
	}

	return CleanupStaleTempFiles(dirs...)
}
