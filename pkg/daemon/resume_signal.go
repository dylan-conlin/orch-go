// Package daemon provides autonomous overnight processing capabilities.
// resume_signal.go handles the resume signal file for manual daemon resumption.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ResumePath returns the path to the resume signal file.
// Default: ~/.orch/daemon-resume.signal
func ResumePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".orch/daemon-resume.signal"
	}
	return filepath.Join(homeDir, ".orch", "daemon-resume.signal")
}

// WriteResumeSignal writes a resume signal file.
// The running daemon will detect this file and resume operation.
func WriteResumeSignal() error {
	resumePath := ResumePath()

	dir := filepath.Dir(resumePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create resume signal directory: %w", err)
	}

	timestamp := time.Now().Format(time.RFC3339)
	if err := os.WriteFile(resumePath, []byte(timestamp), 0644); err != nil {
		return fmt.Errorf("failed to write resume signal: %w", err)
	}

	return nil
}

// CheckAndClearResumeSignal checks if a resume signal exists.
// If it does, it removes the signal file and returns true.
func CheckAndClearResumeSignal() (bool, error) {
	resumePath := ResumePath()

	if _, err := os.Stat(resumePath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check resume signal: %w", err)
	}

	if err := os.Remove(resumePath); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to remove resume signal: %w", err)
	}

	return true, nil
}
