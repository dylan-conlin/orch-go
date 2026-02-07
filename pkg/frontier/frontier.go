// Package frontier provides decidability state calculations for the orch ecosystem.
// It calculates the "frontier" of work - what's ready, what's blocked and what would
// unblock the most work if completed.
package frontier

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// Issue represents a beads issue with dependency information.
type Issue struct {
	ID             string       `json:"id"`
	Title          string       `json:"title"`
	Status         string       `json:"status"`
	IssueType      string       `json:"issue_type"`
	Priority       int          `json:"priority"`
	Dependencies   []Dependency `json:"dependencies,omitempty"`
	Dependents     []Dependency `json:"dependents,omitempty"`
	BlockedBy      []string     `json:"blocked_by,omitempty"`
	BlockedByCount int          `json:"blocked_by_count,omitempty"`
	DependentCount int          `json:"dependent_count,omitempty"`
}

// Dependency represents a dependency relationship.
type Dependency struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	IssueType      string `json:"issue_type"`
	DependencyType string `json:"dependency_type"`
}

// BlockedIssue represents a blocked issue with leverage information.
type BlockedIssue struct {
	Issue         *Issue   // The blocked issue
	WouldUnblock  []string // IDs of issues that would be unblocked
	TotalLeverage int      // Total count of transitively unblocked issues
}

// ActiveAgent represents an agent currently working on an issue.
type ActiveAgent struct {
	BeadsID string // Issue being worked on
	Runtime string // Duration running
}

// FrontierState represents the current decidability state.
type FrontierState struct {
	Ready   []*Issue       // Issues ready to work on (no blockers)
	Blocked []*BlockedIssue // Blocked issues sorted by leverage
	Active  []*ActiveAgent // Currently active agents
}

// CalculateFrontier computes the frontier state from beads data.
// It fetches ready issues, blocked issues, and calculates leverage for each blocked issue.
func CalculateFrontier() (*FrontierState, error) {
	// Get ready issues
	ready, err := getReadyIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to get ready issues: %w", err)
	}

	// Get blocked issues with full dependency information
	blocked, err := getBlockedIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked issues: %w", err)
	}

	// Build a map of all open issues for leverage calculation
	allOpen, err := getAllOpenIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to get all open issues: %w", err)
	}

	// Calculate leverage for each blocked issue
	blockedWithLeverage := calculateAllLeverage(blocked, allOpen)

	// Sort by leverage (descending)
	sort.Slice(blockedWithLeverage, func(i, j int) bool {
		return blockedWithLeverage[i].TotalLeverage > blockedWithLeverage[j].TotalLeverage
	})

	return &FrontierState{
		Ready:   ready,
		Blocked: blockedWithLeverage,
		Active:  nil, // Active agents are populated separately via registry
	}, nil
}

// getReadyIssues fetches issues that are ready to work on from beads.
func getReadyIssues() ([]*Issue, error) {
	// Use --sandbox to force JSONL-only mode, avoiding SQLite foreign key issues
	cmd := exec.Command("bd", "--sandbox", "ready", "--json", "--limit", "0")
	if beads.DefaultDir != "" {
		cmd.Dir = beads.DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd ready failed: %w", err)
	}

	var issues []*Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd ready output: %w", err)
	}

	return issues, nil
}

// getBlockedIssues fetches issues that are blocked.
func getBlockedIssues() ([]*Issue, error) {
	cmd := exec.Command("bd", "--sandbox", "blocked", "--json")
	if beads.DefaultDir != "" {
		cmd.Dir = beads.DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd blocked failed: %w", err)
	}

	var issues []*Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd blocked output: %w", err)
	}

	return issues, nil
}

// getAllOpenIssues fetches all open issues with full dependency info.
func getAllOpenIssues() (map[string]*Issue, error) {
	cmd := exec.Command("bd", "--sandbox", "list", "--status", "open", "--json", "--limit", "0")
	if beads.DefaultDir != "" {
		cmd.Dir = beads.DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	var issues []*Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	// For each issue, fetch full dependency info via bd show
	result := make(map[string]*Issue)
	for _, issue := range issues {
		fullIssue, err := getIssueWithDeps(issue.ID)
		if err != nil {
			// Skip issues we can't fetch
			continue
		}
		result[fullIssue.ID] = fullIssue
	}

	return result, nil
}

// getIssueWithDeps fetches a single issue with full dependency information.
func getIssueWithDeps(id string) (*Issue, error) {
	cmd := exec.Command("bd", "--sandbox", "show", id, "--json")
	if beads.DefaultDir != "" {
		cmd.Dir = beads.DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	// bd show --json returns an array
	var issues []*Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd show output: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("no issue found for id: %s", id)
	}

	return issues[0], nil
}

// calculateAllLeverage computes leverage for all blocked issues.
func calculateAllLeverage(blocked []*Issue, allOpen map[string]*Issue) []*BlockedIssue {
	result := make([]*BlockedIssue, 0, len(blocked))

	for _, issue := range blocked {
		wouldUnblock, totalLeverage := calculateLeverage(issue.ID, allOpen)
		result = append(result, &BlockedIssue{
			Issue:         issue,
			WouldUnblock:  wouldUnblock,
			TotalLeverage: totalLeverage,
		})
	}

	return result
}

// calculateLeverage calculates what completing an issue would unblock.
// Returns the list of directly unblocked issue IDs and the total leverage count.
func calculateLeverage(issueID string, allOpen map[string]*Issue) ([]string, int) {
	// Find issues that depend on this issue
	var directlyUnblocks []string
	visited := make(map[string]bool)

	// Find all issues where this issue is a blocker
	for id, issue := range allOpen {
		if issue.Dependencies == nil {
			continue
		}
		for _, dep := range issue.Dependencies {
			// Check if this issue blocks the other issue
			if dep.ID == issueID && isBlockingDependency(dep) {
				// Check if this is the ONLY blocker
				if isOnlyBlocker(issue, issueID, allOpen) {
					directlyUnblocks = append(directlyUnblocks, id)
				}
			}
		}
	}

	// Calculate total leverage (including transitive unblocks)
	totalLeverage := 0
	var queue []string
	queue = append(queue, directlyUnblocks...)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true
		totalLeverage++

		// Find what this issue would transitively unblock
		for id, issue := range allOpen {
			if visited[id] || issue.Dependencies == nil {
				continue
			}
			for _, dep := range issue.Dependencies {
				if dep.ID == current && isBlockingDependency(dep) {
					if isOnlyBlocker(issue, current, allOpen) {
						queue = append(queue, id)
					}
				}
			}
		}
	}

	return directlyUnblocks, totalLeverage
}

// isBlockingDependency checks if a dependency type causes blocking.
func isBlockingDependency(dep Dependency) bool {
	// parent-child dependencies don't block (children are independently workable)
	if dep.DependencyType == "parent-child" {
		return false
	}
	// "blocks" and other types cause blocking unless the dependency is resolved
	return !isResolved(dep)
}

// isResolved checks if a dependency is resolved (closed or answered for questions).
func isResolved(dep Dependency) bool {
	status := strings.ToLower(dep.Status)
	if status == "closed" {
		return true
	}
	// Questions are resolved when answered
	if dep.IssueType == "question" && status == "answered" {
		return true
	}
	return false
}

// isOnlyBlocker checks if the given issueID is the only active blocker for an issue.
func isOnlyBlocker(issue *Issue, blockerID string, allOpen map[string]*Issue) bool {
	if issue.Dependencies == nil {
		return false
	}

	otherBlockers := 0
	for _, dep := range issue.Dependencies {
		if dep.ID == blockerID {
			continue // Skip the blocker we're checking
		}
		if isBlockingDependency(dep) {
			otherBlockers++
		}
	}

	return otherBlockers == 0
}

// FormatLeverage returns a human-readable description of what an issue would unblock.
func FormatLeverage(bi *BlockedIssue) string {
	if bi.TotalLeverage == 0 {
		return ""
	}

	if len(bi.WouldUnblock) == 0 {
		return fmt.Sprintf("unblocks %d (transitive)", bi.TotalLeverage)
	}

	if len(bi.WouldUnblock) == 1 {
		return fmt.Sprintf("unblocks: %s", bi.WouldUnblock[0])
	}

	// Show first few IDs + count
	if len(bi.WouldUnblock) <= 3 {
		return fmt.Sprintf("unblocks: %s", strings.Join(bi.WouldUnblock, ", "))
	}

	return fmt.Sprintf("unblocks: %s... (+%d more)",
		strings.Join(bi.WouldUnblock[:2], ", "),
		len(bi.WouldUnblock)-2)
}
