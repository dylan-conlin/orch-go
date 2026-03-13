package daemon

import (
	"fmt"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

// LearningRefreshResult holds the outcome of a periodic learning refresh.
type LearningRefreshResult struct {
	Error             error
	Message           string
	DowngradesApplied int
	Suggestions       []daemonconfig.DowngradeSuggestion
}

// RunPeriodicLearningRefresh recomputes learning metrics from events.jsonl
// and auto-adjusts compliance levels downward for skills with sustained
// high success rates. Returns nil if not due.
//
// Safety asymmetry: only downgrades (less strict), never upgrades.
func (d *Daemon) RunPeriodicLearningRefresh() *LearningRefreshResult {
	if !d.Scheduler.IsDue(TaskLearningRefresh) {
		return nil
	}
	d.Scheduler.MarkRun(TaskLearningRefresh)

	// Recompute learning from events.jsonl
	learning, err := events.ComputeLearning(events.DefaultLogPath())
	if err != nil {
		return &LearningRefreshResult{
			Error:   err,
			Message: fmt.Sprintf("Failed to compute learning: %v", err),
		}
	}

	// Update daemon's learning store (used by allocation scoring)
	d.Learning = learning

	// Evaluate compliance downgrades
	suggestions := daemonconfig.SuggestDowngrades(&d.Config.Compliance, learning)
	if len(suggestions) == 0 {
		skillCount := len(learning.Skills)
		return &LearningRefreshResult{
			Message: fmt.Sprintf("Learning refreshed: %d skills tracked, no compliance adjustments needed", skillCount),
		}
	}

	// Apply downgrades
	applied := daemonconfig.ApplyDowngrades(&d.Config.Compliance, suggestions)

	return &LearningRefreshResult{
		Message:           fmt.Sprintf("Learning refreshed: applied %d compliance downgrade(s)", applied),
		DowngradesApplied: applied,
		Suggestions:       suggestions,
	}
}
