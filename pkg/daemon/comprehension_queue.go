// Package daemon provides autonomous overnight processing capabilities.
// comprehension_queue.go tracks issues that are mechanically complete but
// not yet comprehended by the orchestrator. The daemon adds comprehension:pending
// after auto-completing agents; the orchestrator removes it during review.
// When the queue exceeds a threshold, the daemon pauses spawning.
package daemon

import (
	"fmt"
	"strings"
)

const (
	// LabelComprehensionPending marks work that is mechanically complete but
	// not yet reviewed/comprehended by the orchestrator.
	LabelComprehensionPending = "comprehension:pending"

	// DefaultComprehensionThreshold is the maximum number of pending comprehension
	// items before the daemon pauses spawning. Configurable via config.
	DefaultComprehensionThreshold = 5
)

// ComprehensionQuerier counts issues with the comprehension:pending label.
type ComprehensionQuerier interface {
	// CountPending returns the number of issues with comprehension:pending label.
	CountPending() (int, error)
}

// BeadsComprehensionQuerier implements ComprehensionQuerier via bd CLI.
type BeadsComprehensionQuerier struct{}

// CountPending counts issues with the comprehension:pending label via bd list.
func (q *BeadsComprehensionQuerier) CountPending() (int, error) {
	output, err := runBdCommand("list", "--label", LabelComprehensionPending, "--format", "json")
	if err != nil {
		return 0, fmt.Errorf("bd list --label %s failed: %w", LabelComprehensionPending, err)
	}
	// Count lines — each JSON object is one issue
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" || trimmed == "[]" {
		return 0, nil
	}
	// bd list --format json outputs one JSON object per line (JSONL)
	count := 0
	for _, line := range strings.Split(trimmed, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

// AddComprehensionPending adds the comprehension:pending label to an issue.
func AddComprehensionPending(beadsID string) error {
	_, err := runBdCommand("label", "add", beadsID, LabelComprehensionPending)
	if err != nil {
		return fmt.Errorf("failed to add %s label to %s: %w", LabelComprehensionPending, beadsID, err)
	}
	return nil
}

// AddComprehensionPendingInDir adds the comprehension:pending label in a specific project directory.
func AddComprehensionPendingInDir(beadsID, dir string) error {
	_, err := runBdCommandInDir(dir, "label", "add", beadsID, LabelComprehensionPending)
	if err != nil {
		return fmt.Errorf("failed to add %s label to %s: %w", LabelComprehensionPending, beadsID, err)
	}
	return nil
}

// RemoveComprehensionPending removes the comprehension:pending label from an issue.
func RemoveComprehensionPending(beadsID string) error {
	_, err := runBdCommand("label", "remove", beadsID, LabelComprehensionPending)
	if err != nil {
		return fmt.Errorf("failed to remove %s label from %s: %w", LabelComprehensionPending, beadsID, err)
	}
	return nil
}

// RemoveComprehensionPendingInDir removes the label in a specific project directory.
func RemoveComprehensionPendingInDir(beadsID, dir string) error {
	_, err := runBdCommandInDir(dir, "label", "remove", beadsID, LabelComprehensionPending)
	if err != nil {
		return fmt.Errorf("failed to remove %s label from %s: %w", LabelComprehensionPending, beadsID, err)
	}
	return nil
}

// RunBdListComprehensionPending returns raw bd list output for comprehension:pending items.
func RunBdListComprehensionPending() ([]byte, error) {
	return runBdCommand("list", "--label", LabelComprehensionPending, "--format", "json")
}

// CheckComprehensionThrottle checks if the comprehension queue exceeds the threshold.
// Returns (allowed, count, threshold). If the querier is nil, always allows.
func CheckComprehensionThrottle(querier ComprehensionQuerier, threshold int) (bool, int, int) {
	if querier == nil {
		return true, 0, threshold
	}
	if threshold <= 0 {
		threshold = DefaultComprehensionThreshold
	}
	count, err := querier.CountPending()
	if err != nil {
		// Fail-open: if we can't check, don't block spawning
		return true, 0, threshold
	}
	return count < threshold, count, threshold
}
