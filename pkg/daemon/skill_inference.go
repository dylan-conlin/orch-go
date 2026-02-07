// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"strings"
)

// IsSpawnableType returns true if the issue type can be spawned.
func IsSpawnableType(issueType string) bool {
	switch issueType {
	case "bug", "feature", "task", "investigation", "question":
		return true
	default:
		return false
	}
}

// InferSkill maps issue types to skills.
//
// Bug handling: Defaults to "systematic-debugging" for direct action on bugs.
// Use explicit skill:architect label for complex/recurring bugs that need
// architectural understanding before fixing.
//
// See: .kb/decisions/2026-01-23-investigation-overhead-firefighting-mode.md
func InferSkill(issueType string) (string, error) {
	switch issueType {
	case "bug":
		// Default to systematic-debugging: direct action on bugs
		// Use skill:architect label for complex/recurring bugs
		return "systematic-debugging", nil
	case "feature":
		return "feature-impl", nil
	case "task":
		return "feature-impl", nil
	case "investigation":
		return "investigation", nil
	case "question":
		// Questions spawn investigation skill to answer them
		return "investigation", nil
	default:
		return "", fmt.Errorf("cannot infer skill for issue type: %s", issueType)
	}
}

// InferSkillFromLabels extracts a skill name from skill:* labels.
// Returns the skill name if found (e.g., "research" from "skill:research"),
// or empty string if no skill label is present.
func InferSkillFromLabels(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, "skill:") {
			return strings.TrimPrefix(label, "skill:")
		}
	}
	return ""
}

// InferMCPFromLabels extracts MCP server requirements from needs:* labels.
// Returns the MCP server name if found (e.g., "playwright" from "needs:playwright"),
// or empty string if no MCP-related label is present.
//
// Supported labels:
//   - needs:playwright → returns "playwright" (browser automation for UI verification)
//
// This allows daemon-spawned agents to automatically get browser access when
// working on UI/CSS fixes that require visual verification.
func InferMCPFromLabels(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, "needs:") {
			need := strings.TrimPrefix(label, "needs:")
			// Map needs labels to MCP servers
			switch need {
			case "playwright":
				return "playwright"
				// Future: add more mappings as needed
				// case "browser-use":
				//     return "browser-use"
			}
		}
	}
	return ""
}

// InferSkillFromTitle detects skills from issue title patterns.
// Returns the skill name if a known pattern is matched, or empty string otherwise.
func InferSkillFromTitle(title string) string {
	// Synthesis issues created by kb reflect --create-issue
	if strings.HasPrefix(title, "Synthesize ") && strings.Contains(title, " investigations") {
		return "kb-reflect"
	}
	return ""
}

// InferSkillFromIssue determines the skill to use for an issue.
// Priority order: skill:* label > title pattern > issue type inference > error
// This respects explicit skill assignments via labels while falling back
// to type-based inference for issues without skill labels.
func InferSkillFromIssue(issue *Issue) (string, error) {
	if issue == nil {
		return "", fmt.Errorf("cannot infer skill for nil issue")
	}

	// First, check for explicit skill:* label
	if skill := InferSkillFromLabels(issue.Labels); skill != "" {
		return skill, nil
	}

	// Check for title-based patterns
	if skill := InferSkillFromTitle(issue.Title); skill != "" {
		return skill, nil
	}

	// Fall back to type-based inference
	return InferSkill(issue.IssueType)
}
