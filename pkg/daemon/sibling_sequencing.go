// Package daemon provides autonomous overnight processing capabilities.
// sibling_sequencing.go implements cross-issue coordination for same-project
// test-vs-implementation ordering. When the daemon sees a test-like issue
// alongside implementation siblings from the same project, it defers the
// test issue so implementations land first (preventing build breakage from
// tests referencing types that don't exist yet).
//
// Discovered: scrape project, 2026-03-18. Test agent (scrape-9w3) wrote tests
// for ghIssue/ghPullRequest/anthropicRequest types that hadn't been implemented
// yet by sibling agents (scrape-52p, scrape-gdh).
package daemon

import (
	"fmt"
	"strings"
)

// testTitlePatterns are substrings that indicate a test-focused issue.
// Matched against lowercased title + description.
var testTitlePatterns = []string{
	"write tests",
	"add tests",
	"test coverage",
	"table-driven",
	"test-driven",
	"unit test",
	"integration test",
	"tests for ",
	"testing ",
}

// isTestLikeIssue returns true if the issue appears to be primarily about
// writing tests based on title and description keywords. This is a heuristic —
// false negatives are acceptable (test issue spawns normally), but false
// positives would incorrectly defer implementation work.
//
// Investigations and questions are always exempt: they produce knowledge
// artifacts (not code), so deferring them behind implementation siblings
// is meaningless. This prevents false positives from investigations that
// discuss testing topics (e.g., "property-based testing as verification layer").
func isTestLikeIssue(issue Issue) bool {
	if issue.IssueType == "investigation" || issue.IssueType == "question" {
		return false
	}
	text := strings.ToLower(issue.Title + " " + issue.Description)
	for _, p := range testTitlePatterns {
		if strings.Contains(text, p) {
			return true
		}
	}
	return false
}

// SiblingExistsFunc checks whether a sibling issue actually exists in beads.
// Used to protect against ghost issues that appear in the ready queue but
// don't exist when queried directly (e.g., via bd show).
// When nil, all siblings in allIssues are trusted.
type SiblingExistsFunc func(id string) bool

// ShouldDeferTestIssue determines whether a test-like issue should be deferred
// because same-project implementation siblings are still pending (open or
// in_progress). Returns (true, reason) if the issue should be skipped this cycle.
//
// The allIssues slice should include both open and in_progress issues from the
// same ListReadyIssues query — this is already the case since beads Ready()
// returns both statuses.
//
// siblingExists is an optional validator that confirms a sibling issue actually
// exists in beads. When non-nil, siblings that fail validation are skipped
// (ghost issue protection). When nil, all siblings are trusted.
//
// Logic:
//   - If issue is not test-like → no deferral
//   - If same-project siblings exist that are NOT test-like and are open or
//     in_progress → defer (let implementations complete first)
//   - If all same-project siblings are also test-like or are closed → no deferral
//   - If blocking sibling is a ghost (siblingExists returns false) → skip it
func ShouldDeferTestIssue(issue Issue, allIssues []Issue, siblingExists SiblingExistsFunc) (bool, string) {
	if !isTestLikeIssue(issue) {
		return false, ""
	}

	issueProject := projectFromIssueID(issue.ID)
	for _, other := range allIssues {
		if other.ID == issue.ID {
			continue
		}
		if projectFromIssueID(other.ID) != issueProject {
			continue
		}
		// Sibling is implementation-like (not test) and still active
		if !isTestLikeIssue(other) && (other.Status == "open" || other.Status == "in_progress") {
			// Verify sibling exists if validator provided (ghost issue protection)
			if siblingExists != nil && !siblingExists(other.ID) {
				continue
			}
			return true, fmt.Sprintf("test issue deferred: implementation sibling %s (%s) pending in same project", other.ID, other.Status)
		}
	}
	return false, ""
}
