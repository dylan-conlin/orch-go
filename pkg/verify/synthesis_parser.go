// Package verify provides verification helpers for agent completion.
// This file contains synthesis parsing logic extracted from check.go.
package verify

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Pre-compiled regex patterns for synthesis parsing
var (
	regexRecommendation  = regexp.MustCompile(`(?m)\*\*Recommendation:\*\*\s*(\w+)`)
	regexNumberedPattern = regexp.MustCompile(`^\d+\.`)
)

// Synthesis represents the content of a SYNTHESIS.md file using the D.E.K.N. structure.
// D.E.K.N. = Delta (what changed), Evidence (what was observed), Knowledge (what was learned), Next (what should happen)
type Synthesis struct {
	// Header fields
	Agent    string // Agent workspace name
	Issue    string // Beads issue ID
	Duration string // Session duration
	Outcome  string // success, partial, blocked, etc.

	// Core D.E.K.N. sections
	TLDR      string // One-sentence summary
	Delta     string // What changed (files created/modified, commits)
	Evidence  string // What was observed (tests run, verification)
	Knowledge string // What was learned (artifacts, decisions, constraints)
	Next      string // What should happen (recommendation, follow-up)

	// Unexplored Questions section (for self-reflection)
	UnexploredQuestions string   // Questions that emerged during session
	AreasToExplore      []string // Areas worth exploring further
	Uncertainties       []string // What remains unclear

	// Architectural Choices section (tradeoff visibility)
	ArchitecturalChoices string // Raw content of the "Architectural Choices" section

	// Parsed fields for easy access
	Recommendation string   // Extracted from Next section (close, continue, escalate)
	NextActions    []string // Follow-up items
}

// ParseSynthesis extracts key information from a SYNTHESIS.md file.
// Supports both the full D.E.K.N. format and simpler formats with just TLDR and Next Actions.
func ParseSynthesis(workspacePath string) (*Synthesis, error) {
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	data, err := os.ReadFile(synthesisPath)
	if err != nil {
		return nil, err
	}
	content := string(data)

	s := &Synthesis{}

	// Parse header fields
	s.Agent = extractHeaderField(content, "Agent")
	s.Issue = extractHeaderField(content, "Issue")
	s.Duration = extractHeaderField(content, "Duration")
	s.Outcome = extractHeaderField(content, "Outcome")

	// Parse TLDR section
	s.TLDR = extractSection(content, "TLDR")

	// Parse D.E.K.N. sections
	// Delta can be "## Delta" or "## Delta (What Changed)"
	s.Delta = extractSectionWithVariant(content, "Delta", "Delta (What Changed)")

	// Evidence can be "## Evidence" or "## Evidence (What Was Observed)"
	s.Evidence = extractSectionWithVariant(content, "Evidence", "Evidence (What Was Observed)")

	// Knowledge can be "## Knowledge" or "## Knowledge (What Was Learned)"
	s.Knowledge = extractSectionWithVariant(content, "Knowledge", "Knowledge (What Was Learned)")

	// Next can be "## Next", "## Next (What Should Happen)", or "## Next Actions"
	s.Next = extractSectionWithVariant(content, "Next", "Next (What Should Happen)")

	// Extract recommendation from Next section
	s.Recommendation = extractRecommendation(s.Next)

	// Parse Next Actions (follow-up items)
	s.NextActions = extractNextActions(content)

	// Parse Architectural Choices section (tradeoff visibility)
	s.ArchitecturalChoices = extractSection(content, "Architectural Choices")

	// Parse Unexplored Questions section
	unexploredSection := extractSection(content, "Unexplored Questions")
	if unexploredSection != "" {
		s.UnexploredQuestions = unexploredSection
		s.AreasToExplore = extractBoldSubsection(unexploredSection, "Areas worth exploring further")
		s.Uncertainties = extractBoldSubsection(unexploredSection, "What remains unclear")
	}

	return s, nil
}

// extractHeaderField extracts a header field like "**Field:** value"
func extractHeaderField(content, field string) string {
	pattern := regexp.MustCompile(`(?m)\*\*` + regexp.QuoteMeta(field) + `:\*\*\s*(.+)$`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractSection extracts content from a markdown section.
// Handles sections that end at the next ## heading or end of file.
func extractSection(content, sectionName string) string {
	// Match section header (with optional parenthetical)
	// Use \n## to match next section, but be careful to capture multi-line content
	pattern := regexp.MustCompile(`(?s)## ` + regexp.QuoteMeta(sectionName) + `(?:\s*\([^)]*\))?\s*\n(.*?)(?:\n---\n|\n## |\z)`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractSectionWithVariant tries multiple section name variants.
func extractSectionWithVariant(content, name1, name2 string) string {
	result := extractSection(content, name1)
	if result == "" {
		result = extractSection(content, name2)
	}
	return result
}

// extractRecommendation extracts the recommendation from the Next section.
// Looks for patterns like "**Recommendation:** close" or just "close" on its own line.
func extractRecommendation(nextSection string) string {
	matches := regexRecommendation.FindStringSubmatch(nextSection)
	if len(matches) >= 2 {
		return strings.ToLower(strings.TrimSpace(matches[1]))
	}
	return ""
}

// extractNextActions extracts follow-up action items from various sections.
func extractNextActions(content string) []string {
	var actions []string

	// Try "## Next Actions" section first
	actionsSection := extractSection(content, "Next Actions")
	if actionsSection != "" {
		actions = append(actions, parseActionItems(actionsSection)...)
	}

	// Also look for follow-up work in Next section
	nextSection := extractSectionWithVariant(content, "Next", "Next (What Should Happen)")
	if nextSection != "" {
		// Look for follow-up subsections with various naming conventions:
		// - "### Follow-up Work" or "### Follow-up Work Identified"
		// - "### Spawn Follow-up" or "### If Spawn Follow-up"
		followUpPatterns := []string{
			`(?s)### Follow-up Work[^\n]*\n(.*?)(?:\n###|\n---|\z)`,
			`(?s)### (?:If )?Spawn Follow-up[^\n]*\n(.*?)(?:\n###|\n---|\z)`,
		}
		for _, pattern := range followUpPatterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(nextSection)
			if len(matches) >= 2 {
				actions = append(actions, parseActionItems(matches[1])...)
			}
		}
	}

	return actions
}

// parseActionItems extracts list items (- item, * item, or 1. item format).
// Note: Uses "* " (asterisk+space) to distinguish bullet points from markdown bold (**text**).
// Note: Only matches non-indented lines to avoid capturing continuation/metadata lines
// that are indented under a parent item.
func parseActionItems(section string) []string {
	var items []string
	lines := strings.Split(section, "\n")

	for _, line := range lines {
		// Skip indented lines - they're continuation/metadata, not separate items
		// Check for indentation BEFORE trimming
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Match bullet points: "- item" or "* item" (with space after marker)
		// Using "* " to avoid matching markdown bold syntax like "**Skill:**"
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || regexNumberedPattern.MatchString(line) {
			items = append(items, line)
		}
	}
	return items
}

// extractBoldSubsection extracts list items from a subsection that starts with **bold header:**
// For example: **Areas worth exploring further:** followed by bullet points.
func extractBoldSubsection(content, header string) []string {
	var items []string

	// Find the bold header and extract content until the next bold header or end
	pattern := regexp.MustCompile(`(?s)\*\*` + regexp.QuoteMeta(header) + `:\*\*\s*\n(.*?)(?:\n\*\*|\n---|\z)`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return items
	}

	subsection := matches[1]
	lines := strings.Split(subsection, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Only extract bullet point items
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			items = append(items, line)
		}
	}

	return items
}
