// Package verify provides verification helpers for agent completion.
package verify

import (
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// ReproVerificationResult represents the result of reproduction verification for bugs.
type ReproVerificationResult struct {
	IsBug      bool     // Whether the issue is a bug type
	Repro      string   // The reproduction steps/evidence from the issue
	HasRepro   bool     // Whether reproduction info was found
	Errors     []string // Error messages
	Warnings   []string // Warning messages
}

// reproductionPatterns defines patterns that might contain reproduction information
// in issue descriptions before the dedicated repro field is available.
var reproductionPatterns = []*regexp.Regexp{
	// Explicit repro sections
	regexp.MustCompile(`(?is)##\s*Repro(?:duction)?(?:\s*Steps?)?\s*\n(.*?)(?:\n##|\z)`),
	regexp.MustCompile(`(?is)##\s*Steps\s+to\s+Reproduce\s*\n(.*?)(?:\n##|\z)`),
	regexp.MustCompile(`(?is)\*\*Repro(?:duction)?:\*\*\s*(.*?)(?:\n\n|\n\*\*|\z)`),
	regexp.MustCompile(`(?is)\*\*Steps\s+to\s+Reproduce:\*\*\s*(.*?)(?:\n\n|\n\*\*|\z)`),
	// Command-style reproduction (commonly found in bug reports)
	regexp.MustCompile("(?m)^```(?:bash|sh)?\\n(.*?)```"),
	// Simple "To reproduce:" pattern
	regexp.MustCompile(`(?is)To\s+reproduce:\s*(.*?)(?:\n\n|\z)`),
}

// IsBugType returns true if the issue type indicates a bug.
func IsBugType(issueType string) bool {
	issueType = strings.ToLower(strings.TrimSpace(issueType))
	return issueType == "bug" || issueType == "defect" || issueType == "bugfix"
}

// ExtractReproFromIssue extracts reproduction steps from a beads issue.
// First checks for a dedicated repro field (when available from beads),
// then falls back to parsing the description.
func ExtractReproFromIssue(issue *beads.Issue) (string, bool) {
	if issue == nil {
		return "", false
	}

	// TODO: When beads adds the repro field, check issue.Repro first
	// For now, parse from description

	// Try each pattern to find reproduction steps in the description
	for _, pattern := range reproductionPatterns {
		matches := pattern.FindStringSubmatch(issue.Description)
		if len(matches) >= 2 {
			repro := strings.TrimSpace(matches[1])
			if repro != "" {
				return repro, true
			}
		}
	}

	// No structured repro found - return the full description as context
	// This allows the orchestrator to manually verify based on the bug description
	if issue.Description != "" {
		return issue.Description, true
	}

	return "", false
}

// GetReproForCompletion retrieves reproduction information for a beads issue.
// Returns ReproVerificationResult with repro info if the issue is a bug type.
// Returns nil if the issue is not a bug type (no verification needed).
func GetReproForCompletion(beadsID string) (*ReproVerificationResult, error) {
	result := &ReproVerificationResult{}

	// Get issue details
	issue, err := GetIssue(beadsID, "")
	if err != nil {
		return nil, err
	}

	// Check if this is a bug type
	result.IsBug = IsBugType(issue.IssueType)
	if !result.IsBug {
		// Not a bug - no reproduction verification needed
		return nil, nil
	}

	// Extract reproduction info
	repro, hasRepro := ExtractReproFromIssue(&beads.Issue{
		ID:          issue.ID,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      issue.Status,
		IssueType:   issue.IssueType,
	})

	result.Repro = repro
	result.HasRepro = hasRepro

	if !hasRepro {
		result.Warnings = append(result.Warnings,
			"No explicit reproduction steps found in bug issue",
			"Using issue title/description for verification context",
		)
		// Use title as fallback
		result.Repro = "Bug: " + issue.Title
		result.HasRepro = true
	}

	return result, nil
}
