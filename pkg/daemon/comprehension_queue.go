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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	output, err := runBdCommand("list", "--label", label, "--json")
	if err != nil {
		return 0, err
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" || trimmed == "[]" {
		return 0, nil
	}
	// Parse JSON array and count elements
	var items []json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
		// Fallback: count non-empty lines (for non-JSON output)
		count := 0
		for _, line := range strings.Split(trimmed, "\n") {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		return count, nil
	}
	return len(items), nil
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

// StripAllComprehensionLabels removes all comprehension lifecycle labels from an issue.
// Used by orch complete: completion IS comprehension, so all labels are stripped.
func StripAllComprehensionLabels(beadsID string) {
	runBdCommand("label", "remove", beadsID, LabelComprehensionUnread)
	runBdCommand("label", "remove", beadsID, LabelComprehensionPending)
	runBdCommand("label", "remove", beadsID, LabelComprehensionProcessed)
}

// StripAllComprehensionLabelsInDir removes all comprehension lifecycle labels in a specific directory.
func StripAllComprehensionLabelsInDir(beadsID, dir string) {
	runBdCommandInDir(dir, "label", "remove", beadsID, LabelComprehensionUnread)
	runBdCommandInDir(dir, "label", "remove", beadsID, LabelComprehensionPending)
	runBdCommandInDir(dir, "label", "remove", beadsID, LabelComprehensionProcessed)
}

// RunBdListComprehensionUnread returns raw bd list output for comprehension:unread items.
func RunBdListComprehensionUnread() ([]byte, error) {
	return runBdCommand("list", "--label", LabelComprehensionUnread, "--json")
}

// RunBdListComprehensionProcessed returns raw bd list output for comprehension:processed items.
func RunBdListComprehensionProcessed() ([]byte, error) {
	return runBdCommand("list", "--label", LabelComprehensionProcessed, "--json")
}

// RunBdListComprehensionPending returns raw bd list output for legacy comprehension:pending items.
func RunBdListComprehensionPending() ([]byte, error) {
	return runBdCommand("list", "--label", LabelComprehensionPending, "--json")
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

// DrainComprehensionUnread transitions all comprehension:unread items to processed.
// Used by `orch daemon resume` to clear the review backlog gate.
// Returns the number of items drained.
func DrainComprehensionUnread() int {
	output, err := runBdCommand("list", "--label", LabelComprehensionUnread, "--format", "ids")
	if err != nil {
		return 0
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return 0
	}
	ids := strings.Split(trimmed, "\n")
	drained := 0
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := TransitionToProcessed(id); err == nil {
			drained++
		}
	}
	return drained
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

// ParseBriefSignalCount extracts signal_count from YAML frontmatter in brief content.
// Returns 0 if no frontmatter or signal_count is not found.
func ParseBriefSignalCount(content string) int {
	// Brief must start with "---\n"
	if !strings.HasPrefix(content, "---\n") {
		return 0
	}
	// Find closing "---"
	endIdx := strings.Index(content[4:], "\n---")
	if endIdx < 0 {
		return 0
	}
	frontmatter := content[4 : 4+endIdx]
	for _, line := range strings.Split(frontmatter, "\n") {
		if strings.HasPrefix(line, "signal_count: ") {
			val := strings.TrimPrefix(line, "signal_count: ")
			n := 0
			for _, c := range val {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				} else {
					break
				}
			}
			return n
		}
	}
	return 0
}

// BriefSignal represents a single quality signal parsed from brief frontmatter.
type BriefSignal struct {
	Score    string
	Detected bool
	Evidence string
}

// BriefQueueEntry represents a brief with parsed signal metadata for queue ordering.
type BriefQueueEntry struct {
	BeadsID     string
	SignalCount int
	Signals     map[string]BriefSignal
}

// ParseBriefSignals extracts per-signal detail from YAML frontmatter in brief content.
// Returns a map of signal name -> BriefSignal. Returns empty map if no frontmatter.
func ParseBriefSignals(content string) map[string]BriefSignal {
	signals := make(map[string]BriefSignal)

	if !strings.HasPrefix(content, "---\n") {
		return signals
	}
	endIdx := strings.Index(content[4:], "\n---")
	if endIdx < 0 {
		return signals
	}
	frontmatter := content[4 : 4+endIdx]
	lines := strings.Split(frontmatter, "\n")

	var currentSignal string
	var currentBrief BriefSignal

	inQualitySignals := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect quality_signals: block
		if trimmed == "quality_signals:" {
			inQualitySignals = true
			continue
		}

		if !inQualitySignals {
			continue
		}

		// Signal name line: exactly 2-space indent, ends with ":"
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			// Save previous signal if any
			if currentSignal != "" {
				signals[currentSignal] = currentBrief
			}
			currentSignal = strings.TrimSuffix(trimmed, ":")
			currentBrief = BriefSignal{}
			continue
		}

		// Detail lines: 4-space indent
		if strings.HasPrefix(line, "    ") && currentSignal != "" {
			if strings.HasPrefix(trimmed, "score: ") {
				currentBrief.Score = unquoteYAML(strings.TrimPrefix(trimmed, "score: "))
			} else if strings.HasPrefix(trimmed, "detected: ") {
				currentBrief.Detected = strings.TrimPrefix(trimmed, "detected: ") == "true"
			} else if strings.HasPrefix(trimmed, "evidence: ") {
				currentBrief.Evidence = unquoteYAML(strings.TrimPrefix(trimmed, "evidence: "))
			}
			continue
		}

		// Non-indented line after quality_signals means block ended
		if !strings.HasPrefix(line, "  ") && trimmed != "" {
			// Save last signal
			if currentSignal != "" {
				signals[currentSignal] = currentBrief
			}
			break
		}
	}

	// Save last signal if we reached end of frontmatter
	if currentSignal != "" {
		if _, exists := signals[currentSignal]; !exists {
			signals[currentSignal] = currentBrief
		}
	}

	return signals
}

// unquoteYAML removes surrounding quotes from a YAML string value.
func unquoteYAML(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// OrderBriefsBySignals sorts briefs by signal quality, highest first.
// If prioritySignals is non-nil, briefs with those signals detected sort first.
func OrderBriefsBySignals(briefs []BriefQueueEntry, prioritySignals []string) {
	sort.Slice(briefs, func(i, j int) bool {
		// Priority signal tiebreaker: count how many priority signals each has
		if len(prioritySignals) > 0 {
			iPriority := countPrioritySignals(briefs[i].Signals, prioritySignals)
			jPriority := countPrioritySignals(briefs[j].Signals, prioritySignals)
			if iPriority != jPriority {
				return iPriority > jPriority
			}
		}
		// Fall back to aggregate signal count
		return briefs[i].SignalCount > briefs[j].SignalCount
	})
}

func countPrioritySignals(signals map[string]BriefSignal, priority []string) int {
	count := 0
	for _, name := range priority {
		if sig, ok := signals[name]; ok && sig.Detected {
			count++
		}
	}
	return count
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
