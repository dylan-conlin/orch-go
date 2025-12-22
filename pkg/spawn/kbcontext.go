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

// RunKBContextCheck runs 'kb context' with the given query and parses the output.
// Returns nil if no matches found or if kb command fails.
func RunKBContextCheck(query string) (*KBContextResult, error) {
	// Run kb context command
	cmd := exec.Command("kb", "context", query)
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
