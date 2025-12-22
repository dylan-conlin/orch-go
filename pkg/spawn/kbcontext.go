// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// OrchEcosystemRepos defines the allowlist of repos that are relevant for orchestration work.
// Used for tiered filtering: when global search is needed, results are post-filtered to this set.
var OrchEcosystemRepos = map[string]bool{
	"orch-go":        true,
	"orch-cli":       true,
	"kb-cli":         true,
	"orch-knowledge": true,
	"beads":          true,
	"kn":             true,
}

// MinMatchesForLocalSearch is the threshold below which we expand to global search.
// If local search returns fewer matches than this, we try global with ecosystem filter.
const MinMatchesForLocalSearch = 3

// MaxMatchesPerCategory limits results per category to prevent context flood.
const MaxMatchesPerCategory = 20

// KBContextMatch represents a match from kb context.
type KBContextMatch struct {
	Type        string // "constraint", "decision", "investigation", "guide"
	Source      string // "kn" or "kb"
	Title       string // Title or description of the match
	Path        string // File path (for kb artifacts)
	Reason      string // Reason (for kn entries)
	FullContent string // Full content line for display
}

// KBContextResult holds the results of a kb context query.
type KBContextResult struct {
	Query      string
	HasMatches bool
	Matches    []KBContextMatch
	RawOutput  string
}

// ExtractKeywords extracts meaningful keywords from a task description for kb context query.
// Uses the same stop word filtering as generateSlug but returns more words for better search.
func ExtractKeywords(task string, maxWords int) string {
	// Stop words to exclude
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"for": true, "to": true, "in": true, "on": true, "at": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"this": true, "that": true, "with": true, "from": true, "of": true,
		"add": true, "implement": true, "create": true, "update": true, "fix": true,
		"new": true, "should": true, "can": true, "will": true, "need": true,
	}

	// Extract words (lowercase, alphanumeric only)
	re := regexp.MustCompile(`[a-zA-Z0-9]+`)
	matches := re.FindAllString(strings.ToLower(task), -1)

	var words []string
	for _, word := range matches {
		if !stopWords[word] && len(word) > 2 {
			words = append(words, word)
			if maxWords > 0 && len(words) >= maxWords {
				break
			}
		}
	}

	return strings.Join(words, " ")
}

// RunKBContextCheck runs 'kb context' with tiered search strategy:
// 1. First query current project (no --global) for targeted results
// 2. If sparse (<3 matches), expand to global search with orch ecosystem filter
// 3. Apply per-category limits to prevent context flood
// Returns nil if no matches found or if kb command fails.
func RunKBContextCheck(query string) (*KBContextResult, error) {
	// Step 1: Try current project first (no --global flag)
	result, err := runKBContextQuery(query, false)
	if err != nil {
		return nil, err
	}

	// Step 2: If local search is sparse, expand to global with ecosystem filter
	if result == nil || len(result.Matches) < MinMatchesForLocalSearch {
		globalResult, err := runKBContextQuery(query, true)
		if err != nil {
			return nil, err
		}

		if globalResult != nil && len(globalResult.Matches) > 0 {
			// Post-filter to orch ecosystem repos
			globalResult.Matches = filterToOrchEcosystem(globalResult.Matches)
			globalResult.HasMatches = len(globalResult.Matches) > 0

			// Merge with local results if any
			if result != nil && len(result.Matches) > 0 {
				result = mergeResults(result, globalResult)
			} else if globalResult.HasMatches {
				result = globalResult
			}
		}
	}

	// Step 3: Apply per-category limits
	if result != nil && len(result.Matches) > 0 {
		result.Matches = applyPerCategoryLimits(result.Matches, MaxMatchesPerCategory)
		result.HasMatches = len(result.Matches) > 0
		// Regenerate RawOutput to reflect filtered results
		result.RawOutput = formatMatchesForDisplay(result.Matches, result.Query)
	}

	if result == nil || !result.HasMatches {
		return nil, nil
	}

	return result, nil
}

// runKBContextQuery runs a single kb context query with optional --global flag.
func runKBContextQuery(query string, global bool) (*KBContextResult, error) {
	var cmd *exec.Cmd
	if global {
		cmd = exec.Command("kb", "context", "--global", query)
	} else {
		cmd = exec.Command("kb", "context", query)
	}

	output, err := cmd.Output()
	if err != nil {
		// If kb command fails (not found, no matches, etc.), return nil
		// This is not an error - just means no context available
		return nil, nil
	}

	outputStr := strings.TrimSpace(string(output))

	// Check for "No results found" or empty output
	if outputStr == "" || strings.Contains(outputStr, "No results found") {
		return nil, nil
	}

	result := &KBContextResult{
		Query:     query,
		RawOutput: outputStr,
	}

	// Parse the output to extract matches
	result.Matches = parseKBContextOutput(outputStr)
	result.HasMatches = len(result.Matches) > 0

	if !result.HasMatches {
		return nil, nil
	}

	return result, nil
}

// filterToOrchEcosystem filters matches to only include those from orch ecosystem repos.
// Matches without a project prefix (local results) are always included.
func filterToOrchEcosystem(matches []KBContextMatch) []KBContextMatch {
	var filtered []KBContextMatch
	for _, m := range matches {
		project := extractProjectFromMatch(m)
		// Include if: no project prefix (local), OR project is in ecosystem allowlist
		if project == "" || OrchEcosystemRepos[project] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// extractProjectFromMatch extracts the project name from a match's title or path.
// Returns empty string if no project prefix found.
func extractProjectFromMatch(m KBContextMatch) string {
	// Check for [project] prefix in title (e.g., "[orch-go] Title here")
	if strings.HasPrefix(m.Title, "[") {
		end := strings.Index(m.Title, "]")
		if end > 1 {
			return m.Title[1:end]
		}
	}
	return ""
}

// applyPerCategoryLimits limits the number of matches per category type.
func applyPerCategoryLimits(matches []KBContextMatch, limit int) []KBContextMatch {
	categoryCounts := make(map[string]int)
	var filtered []KBContextMatch

	for _, m := range matches {
		if categoryCounts[m.Type] < limit {
			filtered = append(filtered, m)
			categoryCounts[m.Type]++
		}
	}
	return filtered
}

// mergeResults combines two KBContextResults, deduplicating matches.
func mergeResults(local, global *KBContextResult) *KBContextResult {
	if local == nil {
		return global
	}
	if global == nil {
		return local
	}

	// Create a set of existing titles to avoid duplicates
	seen := make(map[string]bool)
	var merged []KBContextMatch

	// Add local matches first (higher priority)
	for _, m := range local.Matches {
		key := m.Type + ":" + m.Title
		if !seen[key] {
			seen[key] = true
			merged = append(merged, m)
		}
	}

	// Add global matches that aren't duplicates
	for _, m := range global.Matches {
		key := m.Type + ":" + m.Title
		if !seen[key] {
			seen[key] = true
			merged = append(merged, m)
		}
	}

	return &KBContextResult{
		Query:      local.Query,
		HasMatches: len(merged) > 0,
		Matches:    merged,
		RawOutput:  formatMatchesForDisplay(merged, local.Query),
	}
}

// formatMatchesForDisplay regenerates a display-friendly output from matches.
func formatMatchesForDisplay(matches []KBContextMatch, query string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Context for %q:\n\n", query))

	// Group by type
	byType := make(map[string][]KBContextMatch)
	for _, m := range matches {
		byType[m.Type] = append(byType[m.Type], m)
	}

	// Output in consistent order
	typeOrder := []string{"constraint", "decision", "investigation", "guide"}
	typeHeaders := map[string]string{
		"constraint":    "## CONSTRAINTS",
		"decision":      "## DECISIONS",
		"investigation": "## INVESTIGATIONS",
		"guide":         "## GUIDES",
	}

	for _, t := range typeOrder {
		if ms, ok := byType[t]; ok && len(ms) > 0 {
			// Determine source annotation
			source := "(from kb)"
			if len(ms) > 0 && ms[0].Source == "kn" {
				source = "(from kn)"
			}
			sb.WriteString(fmt.Sprintf("%s %s\n\n", typeHeaders[t], source))
			for _, m := range ms {
				sb.WriteString(fmt.Sprintf("- %s\n", m.Title))
				if m.Reason != "" {
					sb.WriteString(fmt.Sprintf("  Reason: %s\n", m.Reason))
				}
				if m.Path != "" {
					sb.WriteString(fmt.Sprintf("  Path: %s\n", m.Path))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// parseKBContextOutput parses the output of 'kb context' command.
func parseKBContextOutput(output string) []KBContextMatch {
	var matches []KBContextMatch

	lines := strings.Split(output, "\n")
	var currentSection string
	var currentSource string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect section headers
		if strings.HasPrefix(line, "## CONSTRAINTS") {
			currentSection = "constraint"
			currentSource = extractSource(line)
			continue
		}
		if strings.HasPrefix(line, "## DECISIONS") {
			currentSection = "decision"
			currentSource = extractSource(line)
			continue
		}
		if strings.HasPrefix(line, "## INVESTIGATIONS") {
			currentSection = "investigation"
			currentSource = "kb"
			continue
		}
		if strings.HasPrefix(line, "## GUIDES") {
			currentSection = "guide"
			currentSource = "kb"
			continue
		}
		if strings.HasPrefix(line, "Context for") {
			continue // Skip the header line
		}

		// Parse entry lines (start with "- ")
		if strings.HasPrefix(line, "- ") {
			entry := strings.TrimPrefix(line, "- ")
			match := KBContextMatch{
				Type:        currentSection,
				Source:      currentSource,
				FullContent: entry,
			}

			// Extract title and path/reason
			if strings.Contains(entry, "Path:") {
				// kb artifact format: "Title\n  Path: /path/to/file"
				parts := strings.SplitN(entry, "Path:", 2)
				match.Title = strings.TrimSpace(parts[0])
				if len(parts) > 1 {
					match.Path = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(entry, "Reason:") {
				// kn entry format: "Description\n  Reason: explanation"
				parts := strings.SplitN(entry, "Reason:", 2)
				match.Title = strings.TrimSpace(parts[0])
				if len(parts) > 1 {
					match.Reason = strings.TrimSpace(parts[1])
				}
			} else {
				match.Title = entry
			}

			matches = append(matches, match)
		}

		// Handle multi-line entries (Path: or Reason: on next line)
		if strings.HasPrefix(line, "Path:") && len(matches) > 0 {
			matches[len(matches)-1].Path = strings.TrimSpace(strings.TrimPrefix(line, "Path:"))
		}
		if strings.HasPrefix(line, "Reason:") && len(matches) > 0 {
			matches[len(matches)-1].Reason = strings.TrimSpace(strings.TrimPrefix(line, "Reason:"))
		}
	}

	return matches
}

// extractSource extracts the source (kn or kb) from a section header.
func extractSource(line string) string {
	if strings.Contains(line, "(from kn)") {
		return "kn"
	}
	if strings.Contains(line, "(from kb)") {
		return "kb"
	}
	return "unknown"
}

// DisplayContextAndPrompt displays kb context results and prompts for acknowledgment.
// Returns true if user acknowledges (y) or no matches, false if user declines (n).
func DisplayContextAndPrompt(result *KBContextResult) bool {
	if result == nil || !result.HasMatches {
		return true
	}

	fmt.Printf("\n=== KB Context Found for: %q ===\n\n", result.Query)
	fmt.Println(result.RawOutput)
	fmt.Println()
	fmt.Println("=== End KB Context ===")

	fmt.Printf("Found %d relevant context entries.\n", len(result.Matches))
	fmt.Print("Include this context in SPAWN_CONTEXT.md? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// On error, default to yes (include context)
		return true
	}

	response = strings.TrimSpace(strings.ToLower(response))
	// Default to yes if empty response
	if response == "" || response == "y" || response == "yes" {
		return true
	}

	return false
}

// FormatContextForSpawn formats kb context matches for inclusion in SPAWN_CONTEXT.md.
func FormatContextForSpawn(result *KBContextResult) string {
	if result == nil || !result.HasMatches {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## PRIOR KNOWLEDGE (from kb context)\n\n")
	sb.WriteString(fmt.Sprintf("**Query:** %q\n\n", result.Query))

	// Group by type for cleaner output
	constraints := filterByType(result.Matches, "constraint")
	decisions := filterByType(result.Matches, "decision")
	investigations := filterByType(result.Matches, "investigation")

	if len(constraints) > 0 {
		sb.WriteString("### Constraints (MUST respect)\n")
		for _, m := range constraints {
			sb.WriteString(fmt.Sprintf("- %s", m.Title))
			if m.Reason != "" {
				sb.WriteString(fmt.Sprintf("\n  - Reason: %s", m.Reason))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(decisions) > 0 {
		sb.WriteString("### Prior Decisions\n")
		for _, m := range decisions {
			sb.WriteString(fmt.Sprintf("- %s", m.Title))
			if m.Reason != "" {
				sb.WriteString(fmt.Sprintf("\n  - Reason: %s", m.Reason))
			}
			if m.Path != "" {
				sb.WriteString(fmt.Sprintf("\n  - See: %s", m.Path))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(investigations) > 0 {
		sb.WriteString("### Related Investigations\n")
		for _, m := range investigations {
			sb.WriteString(fmt.Sprintf("- %s", m.Title))
			if m.Path != "" {
				sb.WriteString(fmt.Sprintf("\n  - See: %s", m.Path))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.\n\n")

	return sb.String()
}

func filterByType(matches []KBContextMatch, matchType string) []KBContextMatch {
	var filtered []KBContextMatch
	for _, m := range matches {
		if m.Type == matchType {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
