// Package beads provides area label functionality for issue organization.
// Area labels help group related work (e.g., area:dashboard, area:spawn).
package beads

import (
	"fmt"
	"strings"
)

// AreaLabelPrefix is the prefix for area labels.
const AreaLabelPrefix = "area:"

// KnownAreas defines the valid area values for the project.
// These are derived from the label taxonomy in the design investigation.
// See: .kb/investigations/2026-02-05-inv-design-label-based-issue-grouping.md
var KnownAreas = []string{
	"dashboard", // Work Graph, Activity Feed, WIP section
	"spawn",     // Agent spawning, tier system, workspace
	"beads",     // Issue tracking, labels, dependencies
	"cli",       // orch commands, completion, status
	"skill",     // Skill system, skillc, templates
	"kb",        // Knowledge artifacts, investigations
	"opencode",  // Fork maintenance, session management
	"daemon",    // Autonomous processing, triage workflow
	"config",    // Configuration, project settings
}

// areaKeywords maps keywords to area labels for suggestion.
// This enables keyword-based inference from title/description.
// More specific terms should be listed first as they're more reliable signals.
var areaKeywords = map[string][]string{
	"dashboard": {"dashboard", "work graph", "workgraph", "activity feed", "wip section", "web ui"},
	"spawn":     {"spawn", "agent spawn", "workspace", "spawncontext", "spawn context"},
	"beads":     {"beads", "bd create", "bd list", "bd show", "issue tracking", "issue label", "dependency graph"},
	"cli":       {"cli", "orch command", "orch status", "terminal", "flag", "completion"},
	"skill":     {"skill", "skillc", "skill template", "worker skill", "procedure"},
	"kb":        {"kb ", "knowledge base", "investigation", "decision record", "kb model"},
	"opencode":  {"opencode", "opencode session", "fork upstream", "session management"},
	"daemon":    {"daemon", "orch daemon", "autonomous", "triage", "ready queue"},
	"config":    {"config", "configuration", "config yaml", "settings", "orch config"},
}

// HasAreaLabel checks if the given labels include an area: label.
func HasAreaLabel(labels []string) bool {
	for _, label := range labels {
		if strings.HasPrefix(label, AreaLabelPrefix) {
			return true
		}
	}
	return false
}

// GetAreaLabel returns the area: label from a list of labels, or empty string if none.
func GetAreaLabel(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, AreaLabelPrefix) {
			return label
		}
	}
	return ""
}

// SuggestAreaLabel attempts to infer an area label from title and description.
// Uses keyword matching to suggest the most relevant area.
// Returns empty string if no area can be confidently inferred.
func SuggestAreaLabel(title, description string) string {
	text := strings.ToLower(title + " " + description)

	// Count matches for each area using weighted scoring
	// Specific terms get higher weight than generic ones
	type match struct {
		area  string
		score int
	}
	var matches []match

	for area, keywords := range areaKeywords {
		score := 0
		for _, keyword := range keywords {
			// Use word boundary matching for more accurate results
			// Keywords with spaces are phrase matches (higher value)
			occurrences := countKeywordOccurrences(text, keyword)
			if occurrences > 0 {
				// Give higher weight to multi-word phrases and specific terms
				weight := 1
				if strings.Contains(keyword, " ") {
					weight = 3 // Phrases like "work graph" are more specific
				} else if len(keyword) <= 3 {
					weight = 1 // Short keywords like "bd " are common
				} else {
					weight = 2 // Regular keywords
				}
				score += occurrences * weight
			}
		}
		if score > 0 {
			matches = append(matches, match{area: area, score: score})
		}
	}

	// Return the area with the highest score, or empty if no matches
	if len(matches) == 0 {
		return ""
	}

	// Find the best match
	best := matches[0]
	for _, m := range matches[1:] {
		if m.score > best.score {
			best = m
		}
	}

	return AreaLabelPrefix + best.area
}

// countKeywordOccurrences counts how many times a keyword appears in text.
// Uses word-boundary-aware matching for single words.
func countKeywordOccurrences(text, keyword string) int {
	// For phrases (contains space), use substring matching
	if strings.Contains(keyword, " ") {
		return strings.Count(text, keyword)
	}

	// For single words, count word occurrences
	// This prevents "issue" from matching in "tissue" or partial matches
	count := 0
	words := strings.Fields(text)
	for _, word := range words {
		// Clean punctuation from word
		cleaned := strings.Trim(word, ".,;:!?\"'()[]{}")
		if cleaned == keyword || strings.HasPrefix(cleaned, keyword) {
			count++
		}
	}
	return count
}

// FormatAreaLabelWarning returns a warning message for missing area labels.
// Returns empty string if labels already include an area label.
func FormatAreaLabelWarning(labels []string, suggestedArea string) string {
	if HasAreaLabel(labels) {
		return ""
	}

	var msg strings.Builder
	msg.WriteString("\n")
	msg.WriteString("Warning: Issue missing area: label for work grouping\n")
	msg.WriteString("  Available areas: ")
	for i, area := range KnownAreas {
		if i > 0 {
			msg.WriteString(", ")
		}
		msg.WriteString("area:")
		msg.WriteString(area)
	}
	msg.WriteString("\n")

	if suggestedArea != "" {
		msg.WriteString(fmt.Sprintf("  Suggested: %s (based on title/description)\n", suggestedArea))
		msg.WriteString(fmt.Sprintf("  Add with: bd update <issue-id> --add-label %s\n", suggestedArea))
	} else {
		msg.WriteString("  Add with: bd update <issue-id> --add-label area:<name>\n")
	}

	msg.WriteString("\n")
	return msg.String()
}

// FormatAreaLabelSuggestion returns a suggestion message for the inferred area label.
// Returns empty string if no area can be inferred.
func FormatAreaLabelSuggestion(title, description string) string {
	suggested := SuggestAreaLabel(title, description)
	if suggested == "" {
		return ""
	}
	return fmt.Sprintf("Suggested area label: %s (add with --label %s)", suggested, suggested)
}

// ValidateAreaLabel checks if the given area value is in the known areas list.
func ValidateAreaLabel(area string) bool {
	// Strip prefix if present
	if strings.HasPrefix(area, AreaLabelPrefix) {
		area = strings.TrimPrefix(area, AreaLabelPrefix)
	}

	for _, known := range KnownAreas {
		if area == known {
			return true
		}
	}
	return false
}

// ListAreaLabels returns all known area labels in full format (area:name).
func ListAreaLabels() []string {
	labels := make([]string, len(KnownAreas))
	for i, area := range KnownAreas {
		labels[i] = AreaLabelPrefix + area
	}
	return labels
}
