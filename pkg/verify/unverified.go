// Package verify provides verification-related functionality.
// This file defines the canonical source of truth for "what work is unverified."
package verify

import (
	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
)

// UnverifiedItem represents work that has a verification checkpoint but hasn't
// passed the required verification gates for its tier.
//
// This is the canonical type for unverified work across all consumers:
// - Spawn verification gate (blocks spawns)
// - Daemon verification tracker (pauses autonomous spawning)
// - Review command (displays pending completions)
type UnverifiedItem struct {
	BeadsID   string
	IssueType string
	Title     string
	Tier      int  // 1=feature/bug/decision, 2=investigation/probe, 3=task/question
	Gate1     bool // Comprehension gate
	Gate2     bool // Behavioral gate
}

// ListUnverifiedWork returns all work with checkpoints that hasn't passed the
// required verification gates for its tier. Only considers OPEN issues
// (open/in_progress/blocked). Closed/deferred/tombstone issues are excluded.
//
// This is the single source of truth for verification state. All consumers
// (spawn gate, daemon, review) should use this function to prevent divergent
// counting that leads to inconsistent blocking behavior.
//
// Tier verification requirements:
//   - Tier 1 (feature/bug/decision): both gate1 AND gate2 must be complete
//   - Tier 2 (investigation/probe): gate1 must be complete
//   - Tier 3 (task/question/other): no verification required (never returned)
func ListUnverifiedWork() ([]UnverifiedItem, error) {
	checkpoints, err := checkpoint.ReadCheckpoints()
	if err != nil {
		return nil, err
	}
	if len(checkpoints) == 0 {
		return nil, nil
	}

	// Get the set of open issues to filter against
	openIssues, err := ListOpenIssues()
	if err != nil {
		return nil, err
	}

	// Build map of latest checkpoint per beads ID
	// (checkpoint file is append-only; later entries supersede earlier ones)
	latestCheckpoint := make(map[string]*checkpoint.Checkpoint)
	for i := range checkpoints {
		cp := &checkpoints[i]
		existing, ok := latestCheckpoint[cp.BeadsID]
		if !ok || cp.Timestamp.After(existing.Timestamp) {
			latestCheckpoint[cp.BeadsID] = cp
		}
	}

	var unverified []UnverifiedItem
	for beadsID, cp := range latestCheckpoint {
		// Only consider open issues
		issue, isOpen := openIssues[beadsID]
		if !isOpen {
			continue
		}

		tier := checkpoint.TierForIssueType(issue.IssueType)
		switch tier {
		case 1:
			// Tier 1: requires both gate1 (comprehension) and gate2 (behavioral)
			if !cp.Gate1Complete || !cp.Gate2Complete {
				unverified = append(unverified, UnverifiedItem{
					BeadsID:   beadsID,
					IssueType: issue.IssueType,
					Title:     issue.Title,
					Tier:      tier,
					Gate1:     cp.Gate1Complete,
					Gate2:     cp.Gate2Complete,
				})
			}
		case 2:
			// Tier 2: requires gate1 (comprehension) only
			if !cp.Gate1Complete {
				unverified = append(unverified, UnverifiedItem{
					BeadsID:   beadsID,
					IssueType: issue.IssueType,
					Title:     issue.Title,
					Tier:      tier,
					Gate1:     cp.Gate1Complete,
					Gate2:     cp.Gate2Complete,
				})
			}
			// Tier 3: no verification required - skip
		}
	}

	return unverified, nil
}

// CountUnverifiedWork returns the total count of unverified work items.
// This is a convenience wrapper around ListUnverifiedWork.
func CountUnverifiedWork() (int, error) {
	items, err := ListUnverifiedWork()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}
