// Package verify provides verification helpers for agent completion.
// This file contains synthesis opportunity detection logic.
package verify

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SynthesisOpportunity represents a topic with multiple investigations but no synthesis.
type SynthesisOpportunity struct {
	Topic              string   // The topic keyword (e.g., "daemon", "dashboard")
	InvestigationCount int      // Number of investigations on this topic
	InvestigationPaths []string // Paths to the investigations (for reference)
}

// SynthesisOpportunities represents the result of scanning for synthesis opportunities.
type SynthesisOpportunities struct {
	Opportunities []SynthesisOpportunity `json:"opportunities,omitempty"`
	TotalScanned  int                    `json:"total_scanned"` // Total investigations scanned
}

// MinInvestigationsForSynthesis is the threshold for surfacing a synthesis opportunity.
// From the "Coherence Over Patches" principle: 3+ investigations on topic without Guide signals needs synthesis.
const MinInvestigationsForSynthesis = 3

// TopicKeywords are the primary domain keywords to track for synthesis opportunities.
// These are high-level topic areas that warrant synthesis when multiple investigations exist.
//
// NOTE: Meta-topics (investigation, synthesis, artifact, skill) are intentionally excluded.
// These are about the orchestration system itself, not domain topics worth synthesizing.
// Multiple investigations about "how we do investigations" don't need a synthesis -
// they're process improvements, not domain knowledge accumulation.
var TopicKeywords = []string{
	// Core orchestration concepts
	"daemon",
	"dashboard",
	"spawn",
	"status",
	"complete",
	"review",
	"session",
	"registry",
	"workspace",
	// Infrastructure
	"opencode",
	"tmux",
	"beads",
	"server",
	"api",
	"sse",
	// Agent lifecycle
	"agent",
	"headless",
	"verification",
	"context",
	"tokens",
}

// Pre-compiled regex for investigation filename parsing.
// Investigation filenames follow: YYYY-MM-DD-{type}-{topic}.md
// where type is inv-, design-, audit-, debug-, research-, reliability-
var investigationFilenamePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(inv|design|audit|debug|research|reliability)-(.+)\.md$`)

// DetectSynthesisOpportunities scans the .kb directory for synthesis opportunities.
// It finds topics with 3+ investigations but no corresponding Guide or Decision.
func DetectSynthesisOpportunities(projectDir string) (*SynthesisOpportunities, error) {
	kbDir := filepath.Join(projectDir, ".kb")

	// Check if .kb directory exists
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		return &SynthesisOpportunities{}, nil
	}

	// Get existing guides and decisions (these count as synthesis)
	existingGuides, err := listGuides(kbDir)
	if err != nil {
		return nil, err
	}
	existingDecisions, err := listDecisions(kbDir)
	if err != nil {
		return nil, err
	}

	// Build a set of topics that already have synthesis
	synthesizedTopics := make(map[string]bool)
	for _, guide := range existingGuides {
		// Extract topic from guide filename (e.g., "daemon.md" -> "daemon")
		topic := strings.TrimSuffix(filepath.Base(guide), ".md")
		synthesizedTopics[topic] = true
		// Also add hyphenated variants (e.g., "tmux-spawn-guide" -> "tmux", "spawn")
		for _, part := range strings.Split(topic, "-") {
			if len(part) > 2 { // Skip short words like "a", "is", etc.
				synthesizedTopics[part] = true
			}
		}
	}
	for _, decision := range existingDecisions {
		// Extract topic from decision filename (e.g., "2025-12-21-daemon-architecture.md")
		base := strings.TrimSuffix(filepath.Base(decision), ".md")
		// Remove date prefix
		if len(base) > 11 {
			base = base[11:] // Skip "YYYY-MM-DD-"
		}
		for _, part := range strings.Split(base, "-") {
			if len(part) > 2 {
				synthesizedTopics[part] = true
			}
		}
	}

	// Scan investigations directory
	investigationsDir := filepath.Join(kbDir, "investigations")
	investigations, err := listInvestigations(investigationsDir)
	if err != nil {
		return nil, err
	}

	// Count investigations per topic keyword
	topicCounts := make(map[string][]string) // topic -> list of investigation paths

	for _, invPath := range investigations {
		filename := filepath.Base(invPath)
		matches := investigationFilenamePattern.FindStringSubmatch(filename)
		if len(matches) < 3 {
			continue
		}

		// Extract the topic part (everything after the type prefix)
		topicPart := matches[2]

		// Check each keyword against the topic part
		for _, keyword := range TopicKeywords {
			if strings.Contains(strings.ToLower(topicPart), keyword) {
				topicCounts[keyword] = append(topicCounts[keyword], invPath)
			}
		}
	}

	// Build opportunities list
	var opportunities []SynthesisOpportunity
	for topic, paths := range topicCounts {
		// Skip if topic already has synthesis
		if synthesizedTopics[topic] {
			continue
		}

		// Only surface if threshold met
		if len(paths) >= MinInvestigationsForSynthesis {
			opportunities = append(opportunities, SynthesisOpportunity{
				Topic:              topic,
				InvestigationCount: len(paths),
				InvestigationPaths: paths,
			})
		}
	}

	// Sort by count descending
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].InvestigationCount > opportunities[j].InvestigationCount
	})

	return &SynthesisOpportunities{
		Opportunities: opportunities,
		TotalScanned:  len(investigations),
	}, nil
}

// listGuides returns a list of guide file paths in the .kb/guides directory.
func listGuides(kbDir string) ([]string, error) {
	guidesDir := filepath.Join(kbDir, "guides")
	return listMarkdownFiles(guidesDir)
}

// listDecisions returns a list of decision file paths in the .kb/decisions directory.
func listDecisions(kbDir string) ([]string, error) {
	decisionsDir := filepath.Join(kbDir, "decisions")
	return listMarkdownFiles(decisionsDir)
}

// listInvestigations returns a list of investigation file paths.
// Checks both .kb/investigations/ and .kb/investigations/simple/.
func listInvestigations(investigationsDir string) ([]string, error) {
	var investigations []string

	// Main investigations directory
	main, err := listMarkdownFiles(investigationsDir)
	if err == nil {
		investigations = append(investigations, main...)
	}

	// Simple investigations subdirectory
	simpleDir := filepath.Join(investigationsDir, "simple")
	simple, err := listMarkdownFiles(simpleDir)
	if err == nil {
		investigations = append(investigations, simple...)
	}

	return investigations, nil
}

// listMarkdownFiles returns a list of .md file paths in a directory.
func listMarkdownFiles(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	return files, nil
}

// HasOpportunities returns true if there are any synthesis opportunities.
func (s *SynthesisOpportunities) HasOpportunities() bool {
	return s != nil && len(s.Opportunities) > 0
}
