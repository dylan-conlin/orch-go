// Package daemon provides autonomous overnight processing capabilities.
//
// prior_art_dedup.go implements spawn gates that detect when work described
// in an issue has already been completed. This prevents wasted agent spawns
// for duplicate or already-done work.
//
// Two gates:
//   - CommitDedupGate (L6): Checks if referenced beads IDs have git commits.
//     Catches architect follow-ups for work already committed by another agent.
//   - KeywordDedupGate (L7): Checks keyword overlap between issue titles.
//     Catches semantic duplicates where titles differ but work is the same.
//
// See: .kb/investigations/2026-03-26-inv-daemon-duplicate-spawn-detection.md
package daemon

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// beadsIDPattern matches beads issue IDs in text: {project}-{suffix}-{5chars}.
// Examples: orch-go-94bxz, price-watch-a1b2c, orch-go-tw4po
var beadsIDPattern = regexp.MustCompile(`\b[a-z]+-[a-z]+-[a-z0-9]{5}\b`)

// extractReferencedBeadsIDs extracts beads IDs mentioned in the issue description,
// excluding the issue's own ID and deduplicating results.
func extractReferencedBeadsIDs(description, selfID string) []string {
	if description == "" {
		return nil
	}
	matches := beadsIDPattern.FindAllString(description, -1)
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		if m == selfID || seen[m] {
			continue
		}
		seen[m] = true
		result = append(result, m)
	}
	return result
}

// CommitDedupGate checks if work for this issue has already been committed (L6).
// Two checks:
//  1. Does the issue's own beads ID appear in recent git commits?
//     (Agent committed but died before updating beads status.)
//  2. Do beads IDs referenced in the description have commits?
//     (Architect created follow-up for work already done by another agent.)
//     Cross-type references are skipped: a task referencing a completed
//     investigation is follow-up work, not duplication.
type CommitDedupGate struct {
	// HasCommitsFunc checks if a beads ID has associated commits in recent
	// git history. Returns true if commits found. When nil, gate is skipped.
	HasCommitsFunc func(beadsID string) bool

	// GetIssueTypeFunc looks up the issue type for a referenced beads ID.
	// Returns the issue type (e.g., "task", "investigation", "bug") or empty
	// string if unknown. Used by Check 2 to skip cross-type references —
	// a task referencing a completed investigation is follow-up, not duplication.
	// When nil, Check 2 behaves as before (no type filtering).
	GetIssueTypeFunc func(beadsID string) string

	// GetIssueTitleFunc looks up the title for a referenced beads ID.
	// Used by Check 2 to distinguish contextual references from true duplicates:
	// if the new issue's title has low keyword overlap with the referenced issue's
	// title, the reference is contextual (background/follow-on), not duplication.
	// When nil, Check 2 falls back to rejecting any same-type ref with commits
	// (backward-compatible behavior).
	GetIssueTitleFunc func(beadsID string) string
}

func (g *CommitDedupGate) Name() string      { return "commit-dedup" }
func (g *CommitDedupGate) FailMode() FailMode { return FailOpen }

func (g *CommitDedupGate) Check(issue *Issue) GateResult {
	if g.HasCommitsFunc == nil {
		return GateResult{Gate: g.Name(), Verdict: GateAllow}
	}

	// Check 1: Does this issue's own ID have commits?
	if g.HasCommitsFunc(issue.ID) {
		return GateResult{
			Gate:    g.Name(),
			Verdict: GateReject,
			Message: fmt.Sprintf("issue %s already has commits in git history", issue.ID),
		}
	}

	// Check 2: Do referenced beads IDs have commits?
	// Three layers of false-positive prevention:
	//   a) Cross-type references are skipped (task→investigation = follow-up)
	//   b) Title similarity: if titles differ, the reference is contextual
	//   c) Falls back to rejection when title lookup unavailable (backward compat)
	refs := extractReferencedBeadsIDs(issue.Description, issue.ID)
	for _, ref := range refs {
		if g.GetIssueTypeFunc != nil && issue.IssueType != "" {
			refType := g.GetIssueTypeFunc(ref)
			if refType != "" && refType != issue.IssueType {
				continue // Cross-type reference — citation, not duplication
			}
		}
		if g.HasCommitsFunc(ref) {
			// Title similarity check: contextual references (background, follow-on)
			// have dissimilar titles. Only reject when titles indicate same work.
			if g.GetIssueTitleFunc != nil {
				refTitle := g.GetIssueTitleFunc(ref)
				if refTitle != "" && !titlesSuggestDuplication(issue.Title, refTitle) {
					continue // Different work scope — contextual reference
				}
			}
			return GateResult{
				Gate:    g.Name(),
				Verdict: GateReject,
				Message: fmt.Sprintf("referenced issue %s already has commits — work may be done", ref),
			}
		}
	}

	return GateResult{Gate: g.Name(), Verdict: GateAllow}
}

// HasRecentCommitsForBeadsID checks if any git commit in the last 48 hours
// references the given beads ID in its commit message.
// Falls open: returns false on error (git not available, etc).
// Exported for production wiring in daemon_loop.go.
func HasRecentCommitsForBeadsID(beadsID string) bool {
	return hasRecentCommitsForBeadsIDInDir(beadsID, "")
}

// hasRecentCommitsForBeadsIDInDir checks git log in a specific directory.
func hasRecentCommitsForBeadsIDInDir(beadsID, dir string) bool {
	if beadsID == "" {
		return false
	}
	cmd := exec.Command("git", "log", "--oneline", "-1", "--since=48 hours ago",
		"--grep", beadsID)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false // fail-open
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// titlesSuggestDuplication checks if two issue titles indicate the same work
// scope using keyword overlap. Returns true when overlap is high enough to
// suggest duplication rather than a contextual/follow-on reference.
//
// Thresholds: overlap coefficient >= 0.7 AND >= 3 common keywords.
// Higher coefficient than KeywordDedupGate (0.5) because follow-on work
// naturally shares domain vocabulary with prior work — a 0.5 threshold
// would reject legitimate follow-on references in the same area.
// At 0.7, titles must share 70% of the shorter title's keywords,
// which indicates near-identical scope rather than domain adjacency.
func titlesSuggestDuplication(titleA, titleB string) bool {
	kwA := extractKeywords(titleA)
	kwB := extractKeywords(titleB)
	common := countCommon(kwA, kwB)
	if common < 3 {
		return false
	}
	return overlapCoefficient(kwA, kwB) >= 0.7
}

// --- Keyword Dedup Gate ---

// KeywordDedupGate checks for keyword overlap between the issue title and
// recently spawned issues (L7). This catches semantic duplicates where two
// issues describe the same work with different titles.
//
// Example: "Fix failing spawn exploration judge flag test" vs
// "Fix unrelated pkg/spawn explore judge model test failure" share keywords
// {fix, spawn, judge, test} — high overlap indicating same work.
type KeywordDedupGate struct {
	// FindOverlapFunc checks if any recently-spawned issue has significant
	// keyword overlap with the given title. Returns (found, matchedIssueID).
	// When nil, gate is skipped.
	FindOverlapFunc func(title, selfID string) (bool, string)
}

func (g *KeywordDedupGate) Name() string      { return "keyword-dedup" }
func (g *KeywordDedupGate) FailMode() FailMode { return FailOpen }

func (g *KeywordDedupGate) Check(issue *Issue) GateResult {
	if g.FindOverlapFunc == nil {
		return GateResult{Gate: g.Name(), Verdict: GateAllow}
	}

	found, matchedID := g.FindOverlapFunc(issue.Title, issue.ID)
	if found {
		return GateResult{
			Gate:    g.Name(),
			Verdict: GateReject,
			Message: fmt.Sprintf("title has significant keyword overlap with recently spawned %s", matchedID),
		}
	}

	return GateResult{Gate: g.Name(), Verdict: GateAllow}
}

// FindKeywordOverlap checks if any issue in the spawn tracker has significant
// keyword overlap with the given title. Returns (found, matchedIssueID).
//
// Matching criteria: overlap coefficient >= 0.5 AND at least 3 common keywords.
// This dual threshold prevents false positives from:
//   - Short titles with coincidental word matches (< 3 common keywords)
//   - Long titles with low-percentage overlap (< 0.5 coefficient)
func FindKeywordOverlap(tracker *SpawnedIssueTracker, title, selfID string) (bool, string) {
	if tracker == nil || title == "" {
		return false, ""
	}

	targetKW := extractKeywords(title)
	if len(targetKW) < 3 {
		// Title too short for meaningful keyword comparison
		return false, ""
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	now := time.Now()
	for trackedTitle, issueID := range tracker.spawnedTitles {
		if issueID == selfID {
			continue
		}
		// Only check entries still within TTL
		spawnTime, exists := tracker.spawned[issueID]
		if !exists || now.Sub(spawnTime) > tracker.TTL {
			continue
		}

		trackedKW := extractKeywords(trackedTitle)
		common := countCommon(targetKW, trackedKW)
		if common < 3 {
			continue
		}
		coefficient := overlapCoefficient(targetKW, trackedKW)
		if coefficient >= 0.5 {
			return true, issueID
		}
	}
	return false, ""
}

// extractKeywords extracts significant keywords from a title for comparison.
// Words shorter than 3 characters are dropped (articles, prepositions).
// Punctuation is stripped. Results are lowercased.
func extractKeywords(title string) map[string]bool {
	words := strings.Fields(strings.ToLower(title))
	keywords := make(map[string]bool)
	for _, w := range words {
		// Strip punctuation
		w = strings.Trim(w, ".,;:!?()[]{}\"'`")
		// Split on slashes for paths like "pkg/spawn"
		parts := strings.Split(w, "/")
		for _, p := range parts {
			if len(p) >= 3 {
				keywords[p] = true
			}
		}
	}
	return keywords
}

// countCommon counts the number of keys present in both maps.
func countCommon(a, b map[string]bool) int {
	count := 0
	for k := range a {
		if b[k] {
			count++
		}
	}
	return count
}

// overlapCoefficient computes |A ∩ B| / min(|A|, |B|).
// Returns 0 if either set is empty.
// This measures how much the smaller set is contained in the larger,
// which is more appropriate than Jaccard for title comparison where
// one issue's title may be more verbose than the other.
func overlapCoefficient(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	common := countCommon(a, b)
	minSize := len(a)
	if len(b) < minSize {
		minSize = len(b)
	}
	return float64(common) / float64(minSize)
}
