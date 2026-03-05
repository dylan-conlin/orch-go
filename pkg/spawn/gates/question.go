package gates

import (
	"fmt"
	"os"
	"strings"
)

// maxDepthWalk is the maximum depth for transitive dependency traversal.
const maxDepthWalk = 10

// OpenQuestion represents an open question found in the transitive dependency chain.
type OpenQuestion struct {
	IssueID string   // The question issue ID
	Title   string   // Question title
	Status  string   // Current status (open, in_progress)
	Path    []string // Dependency path from root to this question
}

// OpenQuestionResult contains the result of checking for open questions
// in an issue's transitive dependency chain.
type OpenQuestionResult struct {
	Questions []OpenQuestion
}

// HasOpenQuestions returns true if any open questions were found.
func (r *OpenQuestionResult) HasOpenQuestions() bool {
	return len(r.Questions) > 0
}

// OpenQuestionChecker checks an issue's transitive dependency chain for open questions.
// Returns nil result if no questions found or check is unavailable.
type OpenQuestionChecker func(issueID string) (*OpenQuestionResult, error)

// IssueSummary is a minimal issue representation for dependency walking.
// Decouples the gate from the beads client.
type IssueSummary struct {
	ID        string
	IssueType string
	Status    string
	Title     string
	Deps      []DepSummary
}

// DepSummary is a minimal dependency representation for walking.
type DepSummary struct {
	ID     string
	Type   string // "blocks", "parent-child", "relates_to"
	Status string
}

// IssueFetcher retrieves an issue summary by ID. Injected to decouple from beads.
type IssueFetcher func(issueID string) (*IssueSummary, error)

// CheckOpenQuestions checks if an issue has transitive dependencies on open questions.
// This is a WARNING-ONLY gate — it never blocks spawn.
// daemonDriven spawns suppress output but still return results.
func CheckOpenQuestions(issueID string, daemonDriven bool, checker OpenQuestionChecker) (*OpenQuestionResult, error) {
	if checker == nil || issueID == "" {
		return nil, nil
	}

	result, err := checker(issueID)
	if err != nil {
		// Log warning but don't block spawn on infrastructure error
		if !daemonDriven {
			fmt.Fprintf(os.Stderr, "Warning: open question check failed: %v\n", err)
		}
		return nil, nil
	}

	if result == nil || !result.HasOpenQuestions() {
		return result, nil
	}

	// Daemon-driven spawns stay silent but return result for telemetry
	if daemonDriven {
		return result, nil
	}

	showOpenQuestionsWarning(result)
	return result, nil
}

// showOpenQuestionsWarning displays open questions as a non-blocking warning.
func showOpenQuestionsWarning(result *OpenQuestionResult) {
	fmt.Fprintf(os.Stderr, "\n⚠️  Provisional work: %d open question(s) in dependency chain\n", len(result.Questions))
	for _, q := range result.Questions {
		pathStr := strings.Join(q.Path, " → ")
		fmt.Fprintf(os.Stderr, "   ? %s: %s (%s)\n", q.IssueID, q.Title, q.Status)
		fmt.Fprintf(os.Stderr, "     path: %s\n", pathStr)
	}
	fmt.Fprintf(os.Stderr, "   Work may need revision when questions are answered.\n\n")
}

// BuildOpenQuestionChecker creates an OpenQuestionChecker from an IssueFetcher.
// Walks the transitive dependency graph (blocks-type only) up to maxDepthWalk,
// collecting any open question-type issues.
func BuildOpenQuestionChecker(fetcher IssueFetcher) OpenQuestionChecker {
	return func(issueID string) (*OpenQuestionResult, error) {
		result := &OpenQuestionResult{}
		visited := make(map[string]bool)
		walkDeps(fetcher, issueID, []string{issueID}, visited, result, 0)
		return result, nil
	}
}

// walkDeps recursively walks blocking dependencies looking for open questions.
func walkDeps(fetcher IssueFetcher, issueID string, path []string, visited map[string]bool, result *OpenQuestionResult, depth int) {
	if depth >= maxDepthWalk {
		return
	}

	issue, err := fetcher(issueID)
	if err != nil {
		return // Can't fetch — skip this branch
	}

	for _, dep := range issue.Deps {
		depID := dep.ID
		if visited[depID] {
			continue // Cycle protection
		}

		// Only walk "blocks" type dependencies
		if dep.Type != "blocks" {
			continue
		}

		// Skip closed/answered deps — they're resolved
		if dep.Status == "closed" || dep.Status == "answered" {
			continue
		}

		visited[depID] = true
		depPath := append(append([]string{}, path...), depID)

		// Fetch the dep to check its type
		depIssue, fetchErr := fetcher(depID)
		if fetchErr != nil {
			continue
		}

		// Check if this dep is an open question
		if depIssue.IssueType == "question" {
			result.Questions = append(result.Questions, OpenQuestion{
				IssueID: depID,
				Title:   depIssue.Title,
				Status:  depIssue.Status,
				Path:    depPath,
			})
		}

		// Continue walking deeper
		walkDeps(fetcher, depID, depPath, visited, result, depth+1)
	}
}
