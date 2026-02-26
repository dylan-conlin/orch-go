// Package main provides knowledge maintenance at completion time.
// This implements Touchpoint 1 from the "Continuous Knowledge Maintenance via
// Orchestration Side Effects" decision: when orch complete runs, the orchestrator
// reviews relevant quick entries and can promote, obsolete, or skip them.
//
// See: .kb/decisions/2026-02-25-continuous-knowledge-maintenance.md
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"golang.org/x/term"
)

// QuickEntry represents a kb quick entry (decision, constraint, attempt, question).
type QuickEntry struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	Reason    string `json:"reason"`
	RefCount  int    `json:"ref_count"`
}

// KnowledgeAction represents what to do with a surfaced quick entry.
type KnowledgeAction int

const (
	ActionSkip     KnowledgeAction = iota
	ActionPromote                  // Promote to formal kb decision
	ActionObsolete                 // Mark as obsolete (work made it outdated)
)

// maxSurfacedEntries limits how many entries are shown during completion review.
const maxSurfacedEntries = 5

// extractCompletionKeywords extracts meaningful keywords from skill name, issue description,
// and completion summary for matching against quick entries during completion review.
func extractCompletionKeywords(skill, issueDesc, summary string) []string {
	seen := make(map[string]bool)
	var keywords []string

	addWords := func(text string) {
		// Replace hyphens and underscores with spaces for tokenization
		text = strings.NewReplacer("-", " ", "_", " ", "/", " ", ".", " ").Replace(text)
		for _, word := range strings.Fields(text) {
			word = strings.ToLower(strings.Trim(word, ".,;:!?()[]{}\"'"))
			if len(word) < 3 {
				continue
			}
			if stopwords[word] {
				continue
			}
			if !seen[word] {
				seen[word] = true
				keywords = append(keywords, word)
			}
		}
	}

	addWords(skill)
	addWords(issueDesc)
	addWords(summary)

	return keywords
}

// scoreEntry calculates a relevance score for a quick entry against keywords.
// Higher scores mean more keyword matches. Returns 0 if no keywords match.
func scoreEntry(entry QuickEntry, keywords []string) int {
	if len(keywords) == 0 {
		return 0
	}

	score := 0
	contentLower := strings.ToLower(entry.Content)
	reasonLower := strings.ToLower(entry.Reason)

	for _, kw := range keywords {
		kw = strings.ToLower(kw)
		if strings.Contains(contentLower, kw) {
			score += 2 // Content matches are weighted higher
		}
		if strings.Contains(reasonLower, kw) {
			score += 1
		}
	}
	return score
}

// filterQuickEntries filters entries by keyword relevance and returns the top N.
func filterQuickEntries(entries []QuickEntry, keywords []string, maxCount int) []QuickEntry {
	if len(keywords) == 0 || len(entries) == 0 {
		return nil
	}

	type scored struct {
		entry QuickEntry
		score int
	}

	var matches []scored
	for _, e := range entries {
		s := scoreEntry(e, keywords)
		if s > 0 {
			matches = append(matches, scored{entry: e, score: s})
		}
	}

	// Sort by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	// Take top N
	if len(matches) > maxCount {
		matches = matches[:maxCount]
	}

	result := make([]QuickEntry, len(matches))
	for i, m := range matches {
		result[i] = m.entry
	}
	return result
}

// formatEntryForReview formats a quick entry for display during completion review.
func formatEntryForReview(entry QuickEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  [%s] %s\n", entry.Type, entry.ID)
	fmt.Fprintf(&b, "  %s\n", entry.Content)
	if entry.Reason != "" {
		fmt.Fprintf(&b, "  Reason: %s\n", entry.Reason)
	}
	return b.String()
}

// parseKnowledgeAction parses user input into a KnowledgeAction.
func parseKnowledgeAction(input string) KnowledgeAction {
	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "p", "promote":
		return ActionPromote
	case "o", "obsolete":
		return ActionObsolete
	default:
		return ActionSkip
	}
}

// loadQuickEntries loads all active quick entries by running kb quick list --json.
func loadQuickEntries() ([]QuickEntry, error) {
	cmd := exec.Command("kb", "quick", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kb quick list failed: %w", err)
	}

	var entries []QuickEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse kb quick list output: %w", err)
	}

	// Filter to active entries only
	var active []QuickEntry
	for _, e := range entries {
		if e.Status == "active" {
			active = append(active, e)
		}
	}

	return active, nil
}

// executePromote runs kb promote for a quick entry.
func executePromote(entryID string) error {
	cmd := exec.Command("kb", "promote", entryID, "--no-editor")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// executeObsolete runs kb quick obsolete for a quick entry.
func executeObsolete(entryID, reason string) error {
	cmd := exec.Command("kb", "quick", "obsolete", entryID, "--reason", reason)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunKnowledgeMaintenance surfaces relevant quick entries during completion review
// and prompts the orchestrator to promote, obsolete, or skip each one.
//
// This is Touchpoint 1 from the "Continuous Knowledge Maintenance" decision:
// the orchestrator reviews relevant quick entries at the moment of highest context.
//
// Parameters:
//   - skill: the skill name of the completed agent
//   - issueDesc: the issue title/description from beads
//   - summary: the Phase: Complete summary text
//   - stdout: output writer for display
//   - stdin: input reader for prompts
//
// Returns nil on success or if no entries are relevant. Non-fatal by design.
func RunKnowledgeMaintenance(skill, issueDesc, summary string, stdout io.Writer, stdin io.Reader) error {
	// Extract keywords from the completed work
	keywords := extractCompletionKeywords(skill, issueDesc, summary)
	if len(keywords) == 0 {
		return nil
	}

	// Load quick entries
	entries, err := loadQuickEntries()
	if err != nil {
		fmt.Fprintf(stdout, "Warning: could not load quick entries: %v\n", err)
		return nil // Non-fatal
	}

	if len(entries) == 0 {
		return nil
	}

	// Filter to relevant entries
	relevant := filterQuickEntries(entries, keywords, maxSurfacedEntries)
	if len(relevant) == 0 {
		return nil
	}

	// Check if stdin is a terminal for interactive prompting
	if f, ok := stdin.(*os.File); ok {
		if !term.IsTerminal(int(f.Fd())) {
			// Non-interactive: just show what would be reviewed
			fmt.Fprintf(stdout, "\n📚 %d relevant quick entries found (non-interactive, skipping review)\n", len(relevant))
			return nil
		}
	}

	// Display header
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "┌─────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(stdout, "│  📚 KNOWLEDGE MAINTENANCE                                   │")
	fmt.Fprintf(stdout, "│  %d relevant quick entries for review                        │\n", len(relevant))
	fmt.Fprintln(stdout, "│  [p]romote  [o]bsolete  [Enter] skip  [q]uit review         │")
	fmt.Fprintln(stdout, "└─────────────────────────────────────────────────────────────┘")

	reader := bufio.NewReader(stdin)
	promoted := 0
	obsoleted := 0

	for i, entry := range relevant {
		fmt.Fprintf(stdout, "\n(%d/%d)\n", i+1, len(relevant))
		fmt.Fprint(stdout, formatEntryForReview(entry))
		fmt.Fprint(stdout, "  Action [p/o/Enter/q]: ")

		response, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		response = strings.TrimSpace(response)

		// Quit review early
		if strings.ToLower(response) == "q" || strings.ToLower(response) == "quit" {
			fmt.Fprintln(stdout, "  Ending knowledge review")
			break
		}

		action := parseKnowledgeAction(response)
		switch action {
		case ActionPromote:
			if err := executePromote(entry.ID); err != nil {
				fmt.Fprintf(stdout, "  ⚠️  Promote failed: %v\n", err)
			} else {
				fmt.Fprintln(stdout, "  ✓ Promoted to formal decision")
				promoted++
			}
		case ActionObsolete:
			fmt.Fprint(stdout, "  Reason (min 20 chars): ")
			reason, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			reason = strings.TrimSpace(reason)
			if len(reason) < 20 {
				fmt.Fprintln(stdout, "  ⚠️  Reason too short (need 20+ chars), skipping")
				continue
			}
			if err := executeObsolete(entry.ID, reason); err != nil {
				fmt.Fprintf(stdout, "  ⚠️  Obsolete failed: %v\n", err)
			} else {
				fmt.Fprintln(stdout, "  ✓ Marked obsolete")
				obsoleted++
			}
		case ActionSkip:
			// Do nothing
		}
	}

	// Summary
	if promoted > 0 || obsoleted > 0 {
		fmt.Fprintf(stdout, "\nKnowledge maintenance: %d promoted, %d obsoleted, %d skipped\n",
			promoted, obsoleted, len(relevant)-promoted-obsoleted)
	}

	return nil
}
