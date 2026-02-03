package attention

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// GitCollector implements the Collector interface for git-based attention signals.
// It wraps the LikelyDone signal detection and transforms signals into AttentionItems.
type GitCollector struct {
	projectDir string
	client     beads.BeadsClient
}

// NewGitCollector creates a new GitCollector with the given project directory and beads client.
func NewGitCollector(projectDir string, client beads.BeadsClient) *GitCollector {
	return &GitCollector{
		projectDir: projectDir,
		client:     client,
	}
}

// Collect gathers attention items for issues that appear complete (have commits)
// but haven't been formally closed yet. These are observability signals that help
// identify work that may be ready for review/completion.
func (c *GitCollector) Collect(role string) ([]AttentionItem, error) {
	// Collect likely-done signals from git commits
	signals, err := CollectLikelyDoneSignals(c.projectDir, c.client)
	if err != nil {
		return nil, fmt.Errorf("failed to collect git signals: %w", err)
	}

	// Transform signals into attention items
	items := make([]AttentionItem, 0, len(signals.Signals))
	now := time.Now()

	for _, signal := range signals.Signals {
		// Calculate priority based on role
		priority := calculateGitPriority(signal, role)

		item := AttentionItem{
			ID:          fmt.Sprintf("git-%s", signal.IssueID),
			Source:      "git",
			Concern:     Observability,
			Signal:      "likely-done",
			Subject:     signal.IssueID,
			Summary:     fmt.Sprintf("%s: %s (commits: %d)", signal.IssueStatus, signal.IssueTitle, signal.CommitCount),
			Priority:    priority,
			Role:        role,
			ActionHint:  fmt.Sprintf("orch complete %s", signal.IssueID),
			CollectedAt: now,
			Metadata: map[string]any{
				"commit_count":   signal.CommitCount,
				"last_commit_at": signal.LastCommitAt,
				"issue_status":   signal.IssueStatus,
				"reason":         signal.Reason,
				"commit_hashes":  signal.CommitHashes,
			},
		}
		items = append(items, item)
	}

	return items, nil
}

// calculateGitPriority determines priority based on role and signal characteristics.
// Lower numbers = higher priority.
func calculateGitPriority(signal LikelyDoneSignal, role string) int {
	// Base priority for git signals (lower than ready issues since these are observability)
	basePriority := 100

	// Role-aware adjustments
	switch role {
	case "human":
		// Humans care about review/completion - higher priority for issues with more commits
		if signal.CommitCount >= 5 {
			return basePriority - 20 // Many commits = likely significant work
		} else if signal.CommitCount >= 3 {
			return basePriority - 10
		}
		return basePriority

	case "orchestrator":
		// Orchestrators care about unfinished work - prioritize based on status
		if signal.IssueStatus == "in_progress" {
			return basePriority - 15 // In-progress with commits = likely ready to complete
		}
		return basePriority

	case "daemon":
		// Daemons have lower priority for likely-done signals (not actionable without human)
		return basePriority + 50

	default:
		return basePriority
	}
}
