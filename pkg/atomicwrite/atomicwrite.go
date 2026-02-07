// Package atomicwrite provides atomic file write operations using the
// temp-file-plus-rename pattern, with proper cleanup of temp files on
// errors and process crashes.
//
// All atomic writes use the convention of appending ".tmp" to the target
// file path. The CleanupStaleTempFiles function removes any orphaned
// .tmp files left behind by interrupted processes.
package atomicwrite

import (
	"fmt"
	"os"
	"path/filepath"
)

// TempSuffix is the suffix appended to target files for temporary writes.
const TempSuffix = ".tmp"

// WriteFile atomically writes data to the named file.
// It writes to a temporary file first, then renames it to the target path.
// If the rename fails, the temp file is cleaned up.
//
// If the process is killed between the temp write and the rename, the
// temp file will be orphaned. Call CleanupStaleTempFiles on startup to
// remove any such orphans.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	tmpPath := path + TempSuffix

	// Write to temp file first
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmpPath, err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Best-effort cleanup on rename failure
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, path, err)
	}

	return nil
}

// WriteFileWithDir atomically writes data to the named file, creating
// the parent directory if it doesn't exist.
func WriteFileWithDir(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	return WriteFile(path, data, perm)
}
