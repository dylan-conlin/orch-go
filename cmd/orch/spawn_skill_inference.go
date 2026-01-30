// Package main provides skill inference logic for spawn commands.
package main

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// InferSkillFromIssueType infers the appropriate skill based on the issue type.
// Returns an error for issue types that cannot be spawned (e.g., epics).
//
// Bug handling: Defaults to "systematic-debugging" for direct action on bugs.
// Use explicit skill:architect label for complex/recurring bugs that need
// architectural understanding before fixing.
//
// See: .kb/decisions/2026-01-23-investigation-overhead-firefighting-mode.md
func InferSkillFromIssueType(issueType string) (string, error) {
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
	case "epic":
		return "", fmt.Errorf("cannot spawn work on epic issues - epics are decomposed into sub-issues")
	case "":
		return "", fmt.Errorf("issue type is empty")
	default:
		return "", fmt.Errorf("unknown issue type: %s", issueType)
	}
}

// inferSkillFromBeadsIssue infers skill from a beads issue using labels, title, then type.
func inferSkillFromBeadsIssue(issue *beads.Issue) string {
	// Check for skill:* labels first
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "skill:") {
			return strings.TrimPrefix(label, "skill:")
		}
	}

	// Check for title patterns (e.g., synthesis issues)
	if strings.HasPrefix(issue.Title, "Synthesize ") && strings.Contains(issue.Title, " investigations") {
		return "kb-reflect"
	}

	// Fall back to type-based inference
	skill, err := InferSkillFromIssueType(issue.IssueType)
	if err != nil {
		return "feature-impl" // Default fallback
	}
	return skill
}

// inferMCPFromBeadsIssue extracts MCP server requirements from issue labels.
// Returns the MCP server name if found (e.g., "playwright" from "needs:playwright"),
// or empty string if no MCP-related label is present.
//
// This allows daemon-spawned agents to automatically get browser access when
// working on UI/CSS fixes that require visual verification.
func inferMCPFromBeadsIssue(issue *beads.Issue) string {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "needs:") {
			need := strings.TrimPrefix(label, "needs:")
			// Map needs labels to MCP servers
			switch need {
			case "playwright":
				return "playwright"
				// Future: add more mappings as needed
			}
		}
	}
	return ""
}
