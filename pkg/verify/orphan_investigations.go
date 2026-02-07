// Package verify provides verification helpers for agent completion.
// This file contains orphan investigation detection logic.
package verify

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// OrphanInvestigation represents an investigation that may have lineage gaps.
// This occurs when an investigation has no prior-work citations but similar-topic
// investigations exist that could have been cited.
type OrphanInvestigation struct {
	// Path is the file path of the investigation.
	Path string `json:"path"`

	// Topic is the primary topic extracted from the filename.
	Topic string `json:"topic"`

	// SimilarInvestigations are investigations on the same topic.
	SimilarInvestigations []string `json:"similar_investigations"`

	// Suggestion is the human-readable recommendation.
	Suggestion string `json:"suggestion"`
}

// OrphanInvestigations represents the result of scanning for lineage gaps.
type OrphanInvestigations struct {
	Orphans      []OrphanInvestigation `json:"orphans,omitempty"`
	TotalScanned int                   `json:"total_scanned"`
}

// LineageMetadataFields are the fields that indicate prior work citation.
// We check for both the new Prior-Work format and legacy Supersedes.
var LineageMetadataFields = []string{
	"## Prior Work",
	"## Prior-Work",
	"**Prior-Work:**",
	"**Supersedes:**",
}

// EmptyLineageIndicators are patterns that indicate unfilled or empty lineage fields.
var EmptyLineageIndicators = []string{
	"[Path to artifact this replaces, if applicable]",
	"N/A",
	"n/a",
	"None",
	"none",
	"TBD",
	"tbd",
}

// DetectOrphanInvestigations scans the .kb directory for investigations with lineage gaps.
// An investigation is considered "orphaned" if:
// 1. It has no meaningful prior-work citations (empty or placeholder values)
// 2. There are other investigations on the same topic that could have been cited
func DetectOrphanInvestigations(projectDir string) (*OrphanInvestigations, error) {
	kbDir := filepath.Join(projectDir, ".kb")

	// Check if .kb directory exists
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		return &OrphanInvestigations{}, nil
	}

	// Scan all investigations
	investigationsDir := filepath.Join(kbDir, "investigations")
	investigations, err := listInvestigations(investigationsDir)
	if err != nil {
		return nil, err
	}

	// Build topic -> investigations map
	topicInvestigations := buildTopicMap(investigations)

	// Find orphans: investigations without prior-work that have topic peers
	var orphans []OrphanInvestigation
	for _, invPath := range investigations {
		if hasLineageCitations(invPath) {
			continue // Has citations, not an orphan
		}

		topic := extractPrimaryTopic(invPath)
		if topic == "" {
			continue // Can't determine topic
		}

		// Get similar investigations (same topic, excluding self)
		similar := getSimilarInvestigations(invPath, topic, topicInvestigations)
		if len(similar) == 0 {
			continue // No similar investigations, lineage gap not applicable
		}

		orphans = append(orphans, OrphanInvestigation{
			Path:                  invPath,
			Topic:                 topic,
			SimilarInvestigations: similar,
			Suggestion:            formatOrphanSuggestion(topic, len(similar)),
		})
	}

	// Sort by number of similar investigations (most similar first)
	sort.Slice(orphans, func(i, j int) bool {
		return len(orphans[i].SimilarInvestigations) > len(orphans[j].SimilarInvestigations)
	})

	return &OrphanInvestigations{
		Orphans:      orphans,
		TotalScanned: len(investigations),
	}, nil
}

// buildTopicMap creates a mapping from topic keywords to investigation paths.
func buildTopicMap(investigations []string) map[string][]string {
	topicMap := make(map[string][]string)

	for _, invPath := range investigations {
		filename := filepath.Base(invPath)
		matches := investigationFilenamePattern.FindStringSubmatch(filename)
		if len(matches) < 3 {
			continue
		}

		topicPart := strings.ToLower(matches[2])

		// Add to all matching topic keywords
		for _, keyword := range TopicKeywords {
			if strings.Contains(topicPart, keyword) {
				topicMap[keyword] = append(topicMap[keyword], invPath)
			}
		}

		// Also add the full topic slug (for more precise matching)
		topicMap[topicPart] = append(topicMap[topicPart], invPath)
	}

	return topicMap
}

// extractPrimaryTopic extracts the primary topic keyword from an investigation path.
func extractPrimaryTopic(invPath string) string {
	filename := filepath.Base(invPath)
	matches := investigationFilenamePattern.FindStringSubmatch(filename)
	if len(matches) < 3 {
		return ""
	}

	topicPart := strings.ToLower(matches[2])

	// Find the first matching keyword
	for _, keyword := range TopicKeywords {
		if strings.Contains(topicPart, keyword) {
			return keyword
		}
	}

	// If no keyword matches, use the first word of the topic slug
	parts := strings.Split(topicPart, "-")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

// getSimilarInvestigations returns investigations on the same topic, excluding self.
func getSimilarInvestigations(selfPath, topic string, topicMap map[string][]string) []string {
	investigations := topicMap[topic]
	var similar []string

	for _, inv := range investigations {
		if inv != selfPath {
			similar = append(similar, inv)
		}
	}

	return similar
}

// hasLineageCitations checks if an investigation has meaningful prior-work citations.
func hasLineageCitations(invPath string) bool {
	content, err := os.ReadFile(invPath)
	if err != nil {
		return false // Can't read, assume no citations
	}

	contentStr := string(content)

	// Check for in-text citations to other investigations
	if hasInTextCitations(contentStr) {
		return true
	}

	// Check for formal lineage metadata with meaningful values
	if hasFormalLineageCitation(contentStr) {
		return true
	}

	return false
}

// inTextCitationPatterns are regex patterns for detecting in-text citations.
var inTextCitationPatterns = []*regexp.Regexp{
	// Direct path references
	regexp.MustCompile(`\.kb/investigations/\d{4}-\d{2}-\d{2}-[^/]+\.md`),
	// Date-based references like "Jan 26 investigation"
	regexp.MustCompile(`(?i)(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)\s+\d{1,2}\s+investigation`),
	// Prior investigation language
	regexp.MustCompile(`(?i)prior investigation`),
	regexp.MustCompile(`(?i)previous investigation`),
	regexp.MustCompile(`(?i)earlier investigation`),
	// From investigation/finding language
	regexp.MustCompile(`(?i)from .*investigation`),
	// Investigation references with dates
	regexp.MustCompile(`(?i)investigation from \d{4}-\d{2}-\d{2}`),
}

// hasInTextCitations checks for informal in-text citations to other investigations.
func hasInTextCitations(content string) bool {
	for _, pattern := range inTextCitationPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// hasFormalLineageCitation checks for formal lineage metadata with meaningful values.
func hasFormalLineageCitation(content string) bool {
	// Look for Supersedes field with a real .md path
	supersedes := regexp.MustCompile(`(?m)^\*\*Supersedes:\*\*\s*(.+)$`)
	matches := supersedes.FindStringSubmatch(content)
	if len(matches) >= 2 {
		value := strings.TrimSpace(matches[1])
		if isEmptyLineageValue(value) {
			return false
		}
		// Has a non-empty value
		if strings.Contains(value, ".md") {
			return true
		}
	}

	// Look for Prior-Work section with table entries
	if strings.Contains(content, "## Prior Work") || strings.Contains(content, "## Prior-Work") {
		// Check if table has actual entries (not just headers)
		tableRow := regexp.MustCompile(`\|\s*\.kb/investigations/.*\|`)
		if tableRow.MatchString(content) {
			return true
		}
	}

	return false
}

// isEmptyLineageValue checks if a lineage field value is effectively empty.
func isEmptyLineageValue(value string) bool {
	if value == "" {
		return true
	}

	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	for _, indicator := range EmptyLineageIndicators {
		if normalizedValue == strings.ToLower(indicator) {
			return true
		}
	}

	// Check for template placeholder patterns
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		return true
	}

	return false
}

// formatOrphanSuggestion creates a human-readable suggestion for an orphan investigation.
func formatOrphanSuggestion(topic string, similarCount int) string {
	return "Potential lineage gap: " + strconv.Itoa(similarCount) + " " +
		pluralize(similarCount, "investigation") +
		" on '" + topic + "' exists but not cited in Prior-Work"
}

// pluralize returns "item" or "items" based on count.
func pluralize(count int, singular string) string {
	if count == 1 {
		return singular
	}
	return singular + "s"
}

// HasOrphans returns true if there are any orphan investigations.
func (o *OrphanInvestigations) HasOrphans() bool {
	return o != nil && len(o.Orphans) > 0
}
