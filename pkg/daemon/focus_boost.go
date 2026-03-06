package daemon

import (
	"strings"
)

// matchFocusToProject checks if a focus goal matches a project identified by prefix.
// It checks both the prefix directly and the project directory basename from the registry.
func matchFocusToProject(goal, prefix string, projectDirNames map[string]string) bool {
	if goal == "" || prefix == "" {
		return false
	}

	goalLower := strings.ToLower(goal)

	// Check if goal contains the prefix directly (e.g., "orch-go reliability" matches "orch-go")
	if strings.Contains(goalLower, strings.ToLower(prefix)) {
		return true
	}

	// Check if goal contains the project directory basename (e.g., "price-watch" matches "pw" via registry)
	if projectDirNames != nil {
		if dirName, ok := projectDirNames[prefix]; ok {
			if strings.Contains(goalLower, strings.ToLower(dirName)) {
				return true
			}
		}
	}

	return false
}

// applyFocusBoost returns a copy of the issues slice with boosted priorities for
// issues from projects matching the focus goal. The boost subtracts boostAmount
// from the priority (lower number = higher priority), clamping at 0.
//
// Returns the original slice unchanged if goal is empty.
func applyFocusBoost(issues []Issue, goal string, boostAmount int, projectDirNames map[string]string) []Issue {
	if goal == "" || boostAmount <= 0 {
		return issues
	}

	// Copy the slice to avoid modifying the original
	result := make([]Issue, len(issues))
	copy(result, issues)

	for i := range result {
		prefix := projectFromIssueID(result[i].ID)
		if matchFocusToProject(goal, prefix, projectDirNames) {
			result[i].Priority -= boostAmount
			if result[i].Priority < 0 {
				result[i].Priority = 0
			}
		}
	}

	return result
}

