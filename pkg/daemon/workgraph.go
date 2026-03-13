// Package daemon provides autonomous overnight processing capabilities.
// workgraph.go implements per-cycle functional computation of work graph signals
// for deduplication and removal. Computed fresh each cycle (no local state).
// Three signals: title similarity, file-target overlap, investigation chains.
package daemon

import (
	"fmt"
	"regexp"
	"strings"
)

// WorkGraph holds the computed signals from a single daemon cycle.
// This is a pure computation from the ready queue + recent completions.
// Respects the No Local Agent State constraint — computed fresh each cycle.
type WorkGraph struct {
	// TitleDuplicates are pairs of issues with similar titles.
	TitleDuplicates []TitleDuplicate
	// FileOverlaps are pairs of issues targeting the same files.
	FileOverlaps []FileOverlap
	// InvestigationChains are issues that reference prior investigations.
	InvestigationChains []InvestigationChain
}

// TitleDuplicate represents two issues with near-duplicate titles.
type TitleDuplicate struct {
	IssueA     string
	IssueB     string
	Similarity float64
}

// FileOverlap represents two issues targeting overlapping files.
type FileOverlap struct {
	IssueA      string
	IssueB      string
	SharedFiles []string
}

// InvestigationChain represents an issue that references prior investigations.
type InvestigationChain struct {
	IssueID          string
	ReferencedIssues []string
}

// RemovalCandidate describes a potential removal or deduplication action.
type RemovalCandidate struct {
	IssueID string
	PairID  string // The other issue in the pair (if applicable)
	Reason  string // title_duplicate, file_overlap, investigation_superseded
	Detail  string // Human-readable description
}

const (
	// TitleSimilarityThreshold is the minimum similarity score to flag
	// two issues as near-duplicates. Set conservatively to reduce false positives.
	TitleSimilarityThreshold = 0.65
)

// issueIDPattern matches beads-style issue IDs like "proj-123", "orch-go-abc".
var issueIDPattern = regexp.MustCompile(`\b([a-z][\w]+-[a-z0-9]+)\b`)

// filePathPattern matches Go-style file paths like "pkg/daemon/daemon.go", "cmd/orch/main.go".
var filePathPattern = regexp.MustCompile(`\b((?:pkg|cmd|internal|web|src)/[\w/.-]+\.(?:go|ts|tsx|svelte|js|jsx))\b`)

// ComputeWorkGraph computes work graph signals from the ready queue and recent completions.
// readyIssues: current queue of spawnable issues.
// recentCompletions: recently closed issues (for detecting rework of already-done work).
func ComputeWorkGraph(readyIssues []Issue, recentCompletions []Issue) WorkGraph {
	graph := WorkGraph{}

	// Combine ready + recent for title comparison (but only flag ready issues as candidates)
	allIssues := make([]Issue, 0, len(readyIssues)+len(recentCompletions))
	allIssues = append(allIssues, readyIssues...)
	allIssues = append(allIssues, recentCompletions...)

	// 1. Title similarity detection
	graph.TitleDuplicates = detectTitleDuplicates(allIssues)

	// 2. File-target overlap (only among ready issues — completed issues are done)
	graph.FileOverlaps = detectFileOverlaps(readyIssues)

	// 3. Investigation chain detection
	graph.InvestigationChains = detectInvestigationChains(readyIssues)

	return graph
}

// RemovalCandidates returns actionable removal/dedup suggestions from the graph.
func (g *WorkGraph) RemovalCandidates() []RemovalCandidate {
	var candidates []RemovalCandidate

	for _, dup := range g.TitleDuplicates {
		candidates = append(candidates, RemovalCandidate{
			IssueID: dup.IssueA,
			PairID:  dup.IssueB,
			Reason:  "title_duplicate",
			Detail: fmt.Sprintf("Issues %s and %s have %.0f%% title similarity — possible duplicate",
				dup.IssueA, dup.IssueB, dup.Similarity*100),
		})
	}

	for _, overlap := range g.FileOverlaps {
		if len(overlap.SharedFiles) >= 2 {
			candidates = append(candidates, RemovalCandidate{
				IssueID: overlap.IssueA,
				PairID:  overlap.IssueB,
				Reason:  "file_overlap",
				Detail: fmt.Sprintf("Issues %s and %s target %d shared files — consider sequencing",
					overlap.IssueA, overlap.IssueB, len(overlap.SharedFiles)),
			})
		}
	}

	return candidates
}

// detectTitleDuplicates finds pairs of issues with similar titles.
func detectTitleDuplicates(issues []Issue) []TitleDuplicate {
	var duplicates []TitleDuplicate

	for i := 0; i < len(issues); i++ {
		for j := i + 1; j < len(issues); j++ {
			sim := TitleSimilarity(issues[i].Title, issues[j].Title)
			if sim >= TitleSimilarityThreshold {
				duplicates = append(duplicates, TitleDuplicate{
					IssueA:     issues[i].ID,
					IssueB:     issues[j].ID,
					Similarity: sim,
				})
			}
		}
	}

	return duplicates
}

// TitleSimilarity computes Jaccard similarity between two issue titles
// after normalization (lowercase, common prefix removal, tokenization).
func TitleSimilarity(a, b string) float64 {
	tokensA := tokenize(a)
	tokensB := tokenize(b)

	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}

	setA := make(map[string]bool, len(tokensA))
	for _, t := range tokensA {
		setA[t] = true
	}

	setB := make(map[string]bool, len(tokensB))
	for _, t := range tokensB {
		setB[t] = true
	}

	intersection := 0
	for t := range setA {
		if setB[t] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// tokenize normalizes and tokenizes a title for similarity comparison.
func tokenize(title string) []string {
	title = strings.ToLower(title)
	// Remove common prefixes that don't carry meaning
	for _, prefix := range []string{"fix:", "feat:", "chore:", "refactor:", "docs:", "test:"} {
		title = strings.TrimPrefix(title, prefix)
	}
	title = strings.TrimSpace(title)

	words := strings.Fields(title)
	// Filter stop words
	var tokens []string
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?()[]{}\"'`-—")
		if len(w) < 2 {
			continue
		}
		if isStopWord(w) {
			continue
		}
		tokens = append(tokens, w)
	}
	return tokens
}

// isStopWord returns true for common English stop words.
func isStopWord(w string) bool {
	switch w {
	case "the", "is", "at", "in", "on", "to", "for", "of", "and", "or",
		"an", "by", "as", "it", "be", "do", "if", "so", "no", "up",
		"with", "from", "that", "this", "into", "when", "than", "also":
		return true
	}
	return false
}

// detectFileOverlaps finds pairs of issues that reference the same source files.
func detectFileOverlaps(issues []Issue) []FileOverlap {
	// Extract file references from each issue
	type issueFiles struct {
		id    string
		files map[string]bool
	}

	var entries []issueFiles
	for _, issue := range issues {
		files := extractFileReferences(issue.Title + " " + issue.Description)
		if len(files) > 0 {
			entries = append(entries, issueFiles{id: issue.ID, files: files})
		}
	}

	var overlaps []FileOverlap
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			var shared []string
			for f := range entries[i].files {
				if entries[j].files[f] {
					shared = append(shared, f)
				}
			}
			if len(shared) > 0 {
				overlaps = append(overlaps, FileOverlap{
					IssueA:      entries[i].id,
					IssueB:      entries[j].id,
					SharedFiles: shared,
				})
			}
		}
	}

	return overlaps
}

// extractFileReferences finds file paths mentioned in text.
func extractFileReferences(text string) map[string]bool {
	matches := filePathPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	files := make(map[string]bool, len(matches))
	for _, m := range matches {
		files[m] = true
	}
	return files
}

// detectInvestigationChains finds issues that reference other issue IDs,
// indicating investigation follow-ups or chains.
func detectInvestigationChains(issues []Issue) []InvestigationChain {
	// Build set of all issue IDs for self-reference filtering
	issueIDs := make(map[string]bool, len(issues))
	for _, issue := range issues {
		issueIDs[issue.ID] = true
	}

	var chains []InvestigationChain
	for _, issue := range issues {
		refs := extractIssueReferences(issue.Title+" "+issue.Description, issue.ID)
		if len(refs) > 0 {
			chains = append(chains, InvestigationChain{
				IssueID:          issue.ID,
				ReferencedIssues: refs,
			})
		}
	}

	return chains
}

// extractIssueReferences finds issue ID references in text, excluding selfID.
func extractIssueReferences(text, selfID string) []string {
	matches := issueIDPattern.FindAllString(text, -1)
	seen := make(map[string]bool)
	var refs []string
	for _, m := range matches {
		if m == selfID {
			continue
		}
		// Filter out common false positives (file extensions, package paths)
		if strings.Contains(m, "/") || strings.Contains(m, ".") {
			continue
		}
		if !seen[m] {
			refs = append(refs, m)
			seen[m] = true
		}
	}
	return refs
}
