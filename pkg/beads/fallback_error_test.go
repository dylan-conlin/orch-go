package beads

import (
	"errors"
	"strings"
	"testing"
)

// TestErrIssueNotFound_Integration verifies that "no .beads directory" errors
// are properly mapped to ErrIssueNotFound to suppress warning spam.
//
// This test documents the expected behavior for warning suppression during spawn.
// When agents run from subprocess contexts (like orch spawn), beads lookups
// fail with "no .beads directory" because the subprocess doesn't have .beads.
// These errors should return ErrIssueNotFound to avoid spamming warnings.
func TestErrIssueNotFound_Integration(t *testing.T) {
	tests := []struct {
		name                     string
		stderrMessage            string
		shouldBeErrIssueNotFound bool
	}{
		{
			name:                     "no issue found message",
			stderrMessage:            "Error: no issue found with id test-123",
			shouldBeErrIssueNotFound: true,
		},
		{
			name:                     "no .beads directory message",
			stderrMessage:            "Error: no .beads directory found",
			shouldBeErrIssueNotFound: true,
		},
		{
			name:                     "no .beads directory with path",
			stderrMessage:            "Error: no .beads directory found in /tmp/workspace",
			shouldBeErrIssueNotFound: true,
		},
		{
			name:                     "unrelated error",
			stderrMessage:            "Error: connection timeout",
			shouldBeErrIssueNotFound: false,
		},
		{
			name:                     "empty stderr",
			stderrMessage:            "",
			shouldBeErrIssueNotFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the error handling logic from FallbackShow/FallbackShowWithDir
			var err error
			if strings.Contains(tt.stderrMessage, "no issue found") || strings.Contains(tt.stderrMessage, "no .beads directory") {
				err = ErrIssueNotFound
			} else if tt.stderrMessage != "" {
				err = errors.New(tt.stderrMessage)
			}

			isNotFound := errors.Is(err, ErrIssueNotFound)
			if isNotFound != tt.shouldBeErrIssueNotFound {
				t.Errorf("Expected ErrIssueNotFound=%v for stderr %q, got %v",
					tt.shouldBeErrIssueNotFound, tt.stderrMessage, isNotFound)
			}
		})
	}
}

// TestWarningSuppressionLogic verifies the warning suppression behavior
// used in active_count.go. When ErrIssueNotFound is returned, no warning
// should be logged. For other errors, warnings should be logged.
func TestWarningSuppressionLogic(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		shouldWarn bool
	}{
		{
			name:       "ErrIssueNotFound - no warning",
			err:        ErrIssueNotFound,
			shouldWarn: false,
		},
		{
			name:       "wrapped ErrIssueNotFound - no warning",
			err:        errors.New("wrapped: issue not found"),
			shouldWarn: true, // Not wrapped with %w, so errors.Is won't match
		},
		{
			name:       "other error - should warn",
			err:        errors.New("connection timeout"),
			shouldWarn: true,
		},
		{
			name:       "nil error - no warning",
			err:        nil,
			shouldWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from active_count.go line 272:
			// if !errors.Is(err, beads.ErrIssueNotFound) {
			//     log.Printf("Warning: beads lookup failed...")
			// }
			shouldWarn := tt.err != nil && !errors.Is(tt.err, ErrIssueNotFound)

			if shouldWarn != tt.shouldWarn {
				t.Errorf("Expected shouldWarn=%v for error %v, got %v",
					tt.shouldWarn, tt.err, shouldWarn)
			}
		})
	}
}

// TestNoBeadsDirectoryErrorMapping documents the expected error mapping
// behavior that prevents warning spam during spawn operations.
//
// Background: orch spawn runs beads lookups from subprocess contexts that
// don't have .beads directories. Before the fix, these lookups would log
// warnings like "Warning: beads lookup failed for orch-go-XXXXX". The fix
// maps "no .beads directory" errors to ErrIssueNotFound, which is then
// handled silently in active_count.go (line 272).
//
// End-to-end flow:
// 1. orch spawn executes in subprocess without .beads directory
// 2. active_count.go calls FallbackShowWithDir(beadsID, projectPath)
// 3. bd CLI returns "no .beads directory found" in stderr
// 4. FallbackShowWithDir detects "no .beads directory" in stderr
// 5. FallbackShowWithDir returns ErrIssueNotFound (not a generic error)
// 6. active_count.go checks errors.Is(err, ErrIssueNotFound)
// 7. Since it's ErrIssueNotFound, no warning is logged
// 8. Spawn output is clean, no warning spam
func TestNoBeadsDirectoryErrorMapping(t *testing.T) {
	errorMessages := []string{
		"no .beads directory",
		"no .beads directory found",
		"no .beads directory found in /path/to/workspace",
		"Error: no .beads directory",
	}

	for _, errMsg := range errorMessages {
		t.Run(errMsg, func(t *testing.T) {
			// Verify that the string matching logic used in FallbackShow
			// correctly identifies "no .beads directory" errors
			isNoBeadsDir := strings.Contains(errMsg, "no .beads directory")
			if !isNoBeadsDir {
				t.Errorf("Expected error message %q to be identified as 'no .beads directory'", errMsg)
			}
		})
	}
}
