// Package verify provides verification-related functionality.
// This file defines the canonical source of truth for "what work is unverified."
package verify

import (
	"fmt"
	"sort"
	"strings"

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
	return ListUnverifiedWorkWithDir("")
}

// ListUnverifiedWorkWithDir returns unverified work scoped to a project directory.
// If projectDir is empty, uses the default beads directory.
func ListUnverifiedWorkWithDir(projectDir string) ([]UnverifiedItem, error) {
	checkpoints, err := checkpoint.ReadCheckpoints()
	if err != nil {
		return nil, err
	}
	if len(checkpoints) == 0 {
		return nil, nil
	}

	// Get the set of open issues to filter against
	openIssues, err := ListOpenIssuesWithDir(projectDir)
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

// CountUnverifiedWorkWithDir returns the count of unverified work scoped to a project directory.
// If projectDir is empty, uses the default beads directory.
func CountUnverifiedWorkWithDir(projectDir string) (int, error) {
	items, err := ListUnverifiedWorkWithDir(projectDir)
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

// ProjectBreakdown groups unverified items by project name extracted from beads IDs.
// Returns a map from project name to count, sorted by count descending.
func ProjectBreakdown(items []UnverifiedItem) map[string]int {
	counts := make(map[string]int)
	for _, item := range items {
		project := projectFromBeadsID(item.BeadsID)
		counts[project]++
	}
	return counts
}

// FormatProjectBreakdown returns a parenthesized per-project count string.
// Example: " (orch-go: 4, toolshed: 3, opencode: 3)"
// Returns empty string if items is empty.
func FormatProjectBreakdown(items []UnverifiedItem) string {
	if len(items) == 0 {
		return ""
	}

	counts := ProjectBreakdown(items)

	// Sort by count descending, then by project name for stability
	type pc struct {
		project string
		count   int
	}
	sorted := make([]pc, 0, len(counts))
	for p, c := range counts {
		sorted = append(sorted, pc{p, c})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].count != sorted[j].count {
			return sorted[i].count > sorted[j].count
		}
		return sorted[i].project < sorted[j].project
	})

	parts := make([]string, len(sorted))
	for i, s := range sorted {
		parts[i] = fmt.Sprintf("%s: %d", s.project, s.count)
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// projectFromBeadsID extracts the project name from a beads ID.
// Beads IDs follow the format: project-xxxx (e.g., "orch-go-3anf", "pw-ed7h").
// The last hyphen-separated segment is the hash; everything before it is the project.
func projectFromBeadsID(beadsID string) string {
	if beadsID == "" {
		return "unknown"
	}
	lastHyphen := strings.LastIndex(beadsID, "-")
	if lastHyphen <= 0 {
		return beadsID
	}
	return beadsID[:lastHyphen]
}
