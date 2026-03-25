// Package daemon provides autonomous overnight processing capabilities.
// comprehension_queue.go tracks two-state comprehension lifecycle:
//   - comprehension:unread — daemon completed work, orchestrator hasn't reviewed yet
//   - comprehension:processed — orchestrator reviewed, Dylan hasn't read brief yet
//
// The daemon throttles spawning based on comprehension:unread count.
// orch complete transitions unread → processed.
// Reading the brief removes comprehension:processed.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// LabelComprehensionUnread marks work that is mechanically complete but
	// not yet reviewed/comprehended by the orchestrator.
	LabelComprehensionUnread = "comprehension:unread"

	// LabelComprehensionProcessed marks work that the orchestrator has reviewed
	// but Dylan hasn't read the brief yet.
	LabelComprehensionProcessed = "comprehension:processed"

	// LabelComprehensionPending is the legacy label, kept for backward compatibility
	// during migration. New code should use Unread/Processed.
	LabelComprehensionPending = "comprehension:pending"

	// DefaultComprehensionThreshold is the maximum number of unread comprehension
	// items before the daemon pauses spawning. Configurable via config.
	DefaultComprehensionThreshold = 5
)

// ComprehensionQuerier counts issues with comprehension labels.
type ComprehensionQuerier interface {
	// CountPending returns the number of issues needing orchestrator review
	// (comprehension:unread + legacy comprehension:pending).
	CountPending() (int, error)
}

// BeadsComprehensionQuerier implements ComprehensionQuerier via bd CLI.
type BeadsComprehensionQuerier struct{}

// CountPending counts issues with comprehension:unread label via bd list.
// Also counts legacy comprehension:pending for backward compatibility.
func (q *BeadsComprehensionQuerier) CountPending() (int, error) {
	unread, err := countByLabel(LabelComprehensionUnread)
	if err != nil {
		return 0, fmt.Errorf("count %s failed: %w", LabelComprehensionUnread, err)
	}
	// Also count legacy pending labels during migration
	legacy, _ := countByLabel(LabelComprehensionPending)
	return unread + legacy, nil
}

// countByLabel counts issues with a given label via bd list.
func countByLabel(label string) (int, error) {
	output, err := runBdCommand("list", "--label", label, "--format", "json")
	if err != nil {
		return 0, err
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" || trimmed == "[]" {
		return 0, nil
	}
	count := 0
	for _, line := range strings.Split(trimmed, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

// AddComprehensionUnread adds the comprehension:unread label to an issue.
func AddComprehensionUnread(beadsID string) error {
	_, err := runBdCommand("label", "add", beadsID, LabelComprehensionUnread)
	if err != nil {
		return fmt.Errorf("failed to add %s label to %s: %w", LabelComprehensionUnread, beadsID, err)
	}
	return nil
}

// AddComprehensionUnreadInDir adds the comprehension:unread label in a specific project directory.
func AddComprehensionUnreadInDir(beadsID, dir string) error {
	_, err := runBdCommandInDir(dir, "label", "add", beadsID, LabelComprehensionUnread)
	if err != nil {
		return fmt.Errorf("failed to add %s label to %s: %w", LabelComprehensionUnread, beadsID, err)
	}
	return nil
}

// TransitionToProcessed transitions an issue from unread to processed.
// Removes comprehension:unread (and legacy pending), adds comprehension:processed.
func TransitionToProcessed(beadsID string) error {
	// Remove unread
	runBdCommand("label", "remove", beadsID, LabelComprehensionUnread)
	// Remove legacy pending if present
	runBdCommand("label", "remove", beadsID, LabelComprehensionPending)
	// Add processed
	_, err := runBdCommand("label", "add", beadsID, LabelComprehensionProcessed)
	if err != nil {
		return fmt.Errorf("failed to add %s label to %s: %w", LabelComprehensionProcessed, beadsID, err)
	}
	return nil
}

// TransitionToProcessedInDir transitions an issue from unread to processed in a specific directory.
func TransitionToProcessedInDir(beadsID, dir string) error {
	// Remove unread
	runBdCommandInDir(dir, "label", "remove", beadsID, LabelComprehensionUnread)
	// Remove legacy pending if present
	runBdCommandInDir(dir, "label", "remove", beadsID, LabelComprehensionPending)
	// Add processed
	_, err := runBdCommandInDir(dir, "label", "add", beadsID, LabelComprehensionProcessed)
	if err != nil {
		return fmt.Errorf("failed to add %s label to %s: %w", LabelComprehensionProcessed, beadsID, err)
	}
	return nil
}

// RemoveComprehensionProcessed removes the comprehension:processed label from an issue.
func RemoveComprehensionProcessed(beadsID string) error {
	_, err := runBdCommand("label", "remove", beadsID, LabelComprehensionProcessed)
	if err != nil {
		return fmt.Errorf("failed to remove %s label from %s: %w", LabelComprehensionProcessed, beadsID, err)
	}
	return nil
}

// RemoveComprehensionProcessedInDir removes comprehension:processed in a specific project directory.
func RemoveComprehensionProcessedInDir(beadsID, dir string) error {
	_, err := runBdCommandInDir(dir, "label", "remove", beadsID, LabelComprehensionProcessed)
	if err != nil {
		return fmt.Errorf("failed to remove %s label from %s: %w", LabelComprehensionProcessed, beadsID, err)
	}
	return nil
}

// RunBdListComprehensionUnread returns raw bd list output for comprehension:unread items.
func RunBdListComprehensionUnread() ([]byte, error) {
	return runBdCommand("list", "--label", LabelComprehensionUnread, "--format", "json")
}

// RunBdListComprehensionProcessed returns raw bd list output for comprehension:processed items.
func RunBdListComprehensionProcessed() ([]byte, error) {
	return runBdCommand("list", "--label", LabelComprehensionProcessed, "--format", "json")
}

// RunBdListComprehensionPending returns raw bd list output for legacy comprehension:pending items.
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

// BriefFeedback represents quality feedback on a comprehension brief.
type BriefFeedback struct {
	BeadsID   string    `json:"beads_id"`
	Rating    string    `json:"rating"` // "shallow" or "good"
	Timestamp time.Time `json:"timestamp"`
}

// RecordBriefFeedback writes feedback for a brief to the feedback log.
func RecordBriefFeedback(beadsID, rating, projectDir string) error {
	if rating != "shallow" && rating != "good" {
		return fmt.Errorf("invalid rating %q: must be 'shallow' or 'good'", rating)
	}

	feedbackDir := filepath.Join(projectDir, ".kb", "briefs", "feedback")
	if err := os.MkdirAll(feedbackDir, 0755); err != nil {
		return fmt.Errorf("failed to create feedback dir: %w", err)
	}

	feedbackPath := filepath.Join(feedbackDir, beadsID+".txt")
	content := fmt.Sprintf("rating: %s\ntimestamp: %s\n", rating, time.Now().Format(time.RFC3339))
	if err := os.WriteFile(feedbackPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write feedback: %w", err)
	}

	return nil
}

// ReadBriefFeedback reads feedback for a brief, if any.
func ReadBriefFeedback(beadsID, projectDir string) (string, error) {
	feedbackPath := filepath.Join(projectDir, ".kb", "briefs", "feedback", beadsID+".txt")
	data, err := os.ReadFile(feedbackPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "rating: ") {
			return strings.TrimPrefix(line, "rating: "), nil
		}
	}
	return "", nil
}

// Backward compatibility aliases for callers that still use the old names.

// AddComprehensionPending adds the comprehension:unread label (backward compat).
func AddComprehensionPending(beadsID string) error {
	return AddComprehensionUnread(beadsID)
}

// AddComprehensionPendingInDir adds the comprehension:unread label in a dir (backward compat).
func AddComprehensionPendingInDir(beadsID, dir string) error {
	return AddComprehensionUnreadInDir(beadsID, dir)
}

// RemoveComprehensionPending transitions to processed (backward compat for orch complete).
func RemoveComprehensionPending(beadsID string) error {
	return TransitionToProcessed(beadsID)
}

// RemoveComprehensionPendingInDir transitions to processed in dir (backward compat for orch complete).
func RemoveComprehensionPendingInDir(beadsID, dir string) error {
	return TransitionToProcessedInDir(beadsID, dir)
}
