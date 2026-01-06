// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"strings"
)

// IsSpawnableType returns true if the issue type can be spawned.
func IsSpawnableType(issueType string) bool {
	switch issueType {
	case "bug", "feature", "task", "investigation":
		return true
	default:
		return false
	}
}

// InferSkill maps issue types to skills.
//
// Bug handling: Defaults to "architect" (understand before fixing) rather than
// "systematic-debugging". This implements the "Premise Before Solution" principle -
// most bugs reported as vague symptoms need understanding before patching.
// Use explicit skill:systematic-debugging label for isolated bugs with clear cause.
//
// See: orch hotspot (58 areas with fix churn from tactical fixes without strategic thinking)
func InferSkill(issueType string) (string, error) {
	switch issueType {
	case "bug":
		// Default to architect: understand before fixing
		// Use skill:systematic-debugging label for clear, isolated bugs
		return "architect", nil
	case "feature":
		return "feature-impl", nil
	case "task":
		return "feature-impl", nil
	case "investigation":
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
