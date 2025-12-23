// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SessionIDFilename is the name of the file storing the session ID in the workspace.
const SessionIDFilename = ".session_id"

// WriteSessionID writes the OpenCode session ID to the workspace directory.
// Uses atomic write (temp file + rename) to prevent partial reads.
// The workspace directory must already exist.
func WriteSessionID(workspacePath, sessionID string) error {
	if sessionID == "" {
		return nil // Nothing to write
	}

	sessionFile := filepath.Join(workspacePath, SessionIDFilename)
	tmpFile := sessionFile + ".tmp"

	// Write to temp file first
	if err := os.WriteFile(tmpFile, []byte(sessionID+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write session ID temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, sessionFile); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename session ID file: %w", err)
	}

	return nil
}

// ReadSessionID reads the OpenCode session ID from the workspace directory.
// Returns empty string if the file doesn't exist or is empty.
func ReadSessionID(workspacePath string) string {
	sessionFile := filepath.Join(workspacePath, SessionIDFilename)
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SessionIDPath returns the path to the session ID file for a workspace.
func SessionIDPath(workspacePath string) string {
	return filepath.Join(workspacePath, SessionIDFilename)
}

// TierFilename is the name of the file storing the spawn tier in the workspace.
const TierFilename = ".tier"

// WriteTier writes the spawn tier to the workspace directory.
// Uses atomic write (temp file + rename) to prevent partial reads.
// The workspace directory must already exist.
func WriteTier(workspacePath, tier string) error {
	if tier == "" {
		return nil // Nothing to write
	}

	tierFile := filepath.Join(workspacePath, TierFilename)
	tmpFile := tierFile + ".tmp"

	// Write to temp file first
	if err := os.WriteFile(tmpFile, []byte(tier+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write tier temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, tierFile); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename tier file: %w", err)
	}

	return nil
}

// ReadTier reads the spawn tier from the workspace directory.
// Returns empty string if the file doesn't exist or is empty.
// Returns TierFull as the default if the file is missing (conservative default).
func ReadTier(workspacePath string) string {
	tierFile := filepath.Join(workspacePath, TierFilename)
	data, err := os.ReadFile(tierFile)
	if err != nil {
		// Conservative default: return TierFull for old workspaces without tier file
		return TierFull
	}
	tier := strings.TrimSpace(string(data))
	if tier == "" {
		return TierFull
	}
	return tier
}

// TierPath returns the path to the tier file for a workspace.
func TierPath(workspacePath string) string {
	return filepath.Join(workspacePath, TierFilename)
}
