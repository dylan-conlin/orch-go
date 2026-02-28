// Package main provides model-impact advisory for orch complete.
// Cross-references a completing agent's SYNTHESIS.md against the model corpus
// (.kb/models/) to surface models that may need updating based on the agent's work.
// This is informational only — it does not block completion.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// ModelInfo represents a model from the .kb/models/ corpus.
type ModelInfo struct {
	DirName  string   // Directory name, e.g., "completion-verification"
	Name     string   // Human-readable name from # Model: header
	Domain   string   // Domain field from model.md
	Keywords []string // Extracted keywords for matching
}

// ModelImpactMatch represents a match between synthesis content and a model.
type ModelImpactMatch struct {
	Model           ModelInfo
	MatchedKeywords []string // Which keywords matched in the synthesis
}

// stopWords are common words filtered out during keyword extraction.
var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "from": true,
	"that": true, "this": true, "model": true, "architecture": true,
	"system": true, "management": true, "integration": true,
}

// regexModelName matches the "# Model: <name>" header in model.md files.
var regexModelName = regexp.MustCompile(`(?m)^#\s+Model:\s*(.+)$`)

// regexDomain matches the "**Domain:** <value>" field in model.md files.
var regexDomain = regexp.MustCompile(`(?m)\*\*Domain:\*\*\s*(.+)$`)

// RunModelImpactAdvisory cross-references a completing agent's SYNTHESIS.md
// against the model corpus and returns formatted advisory text.
// Returns empty string if no matches or if synthesis/models are unavailable.
func RunModelImpactAdvisory(projectDir, workspacePath string) string {
	if projectDir == "" || workspacePath == "" {
		return ""
	}

	// Parse synthesis
	synthesis, err := verify.ParseSynthesis(workspacePath)
	if err != nil || synthesis == nil {
		return ""
	}

	// Build synthesis text for matching
	synthesisText := buildSynthesisSearchText(synthesis)
	if strings.TrimSpace(synthesisText) == "" {
		return ""
	}

	// Discover models
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	models := discoverModels(modelsDir)
	if len(models) == 0 {
		return ""
	}

	// Cross-reference
	matches := matchSynthesisToModels(synthesisText, models)

	return formatModelImpactAdvisory(matches)
}

// buildSynthesisSearchText combines relevant synthesis sections into a single
// searchable text string. Uses TLDR, Delta, Knowledge, and ArchitecturalChoices
// as these describe what changed and what was learned.
func buildSynthesisSearchText(s *verify.Synthesis) string {
	parts := []string{s.TLDR, s.Delta, s.Knowledge, s.ArchitecturalChoices}
	if s.Next != "" {
		parts = append(parts, s.Next)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

// discoverModels scans the .kb/models/ directory for model.md files and
// extracts model info with keywords for matching.
func discoverModels(modelsDir string) []ModelInfo {
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil
	}

	var models []ModelInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()

		// Skip non-model directories
		if dirName == "archived" || dirName == "." || dirName == ".." || strings.HasPrefix(dirName, ".") {
			continue
		}

		modelPath := filepath.Join(modelsDir, dirName, "model.md")
		data, err := os.ReadFile(modelPath)
		if err != nil {
			continue // No model.md, skip
		}
		content := string(data)

		// Extract name and domain
		name := dirName
		if matches := regexModelName.FindStringSubmatch(content); len(matches) >= 2 {
			name = strings.TrimSpace(matches[1])
		}
		domain := ""
		if matches := regexDomain.FindStringSubmatch(content); len(matches) >= 2 {
			domain = strings.TrimSpace(matches[1])
		}

		keywords := extractModelKeywords(dirName, domain)
		if len(keywords) < 2 {
			continue // Need at least 2 keywords for meaningful matching
		}

		models = append(models, ModelInfo{
			DirName:  dirName,
			Name:     name,
			Domain:   domain,
			Keywords: keywords,
		})
	}

	return models
}

// extractModelKeywords builds a keyword set from the model's directory name
// and domain field. Filters out stop words and short words (<=3 chars).
func extractModelKeywords(dirName, domain string) []string {
	seen := make(map[string]bool)
	var keywords []string

	addWord := func(word string) {
		word = strings.ToLower(strings.TrimSpace(word))
		// Remove punctuation
		word = strings.Trim(word, ".,;:!?()[]{}\"'`")
		if len(word) <= 3 || stopWords[word] || seen[word] {
			return
		}
		seen[word] = true
		keywords = append(keywords, word)
	}

	// Extract from directory name (kebab-case)
	for _, part := range strings.Split(dirName, "-") {
		addWord(part)
	}

	// Extract from domain field (slash/comma separated)
	if domain != "" {
		// Split on / and , first, then split each part into words
		for _, segment := range strings.FieldsFunc(domain, func(r rune) bool {
			return r == '/' || r == ','
		}) {
			for _, word := range strings.Fields(segment) {
				addWord(word)
			}
		}
	}

	return keywords
}

// matchSynthesisToModels checks which models are potentially impacted by the
// synthesis content. A model matches if at least 2 of its keywords appear
// in the synthesis text.
func matchSynthesisToModels(synthesisText string, models []ModelInfo) []ModelImpactMatch {
	textLower := strings.ToLower(synthesisText)
	var matches []ModelImpactMatch

	for _, model := range models {
		var matched []string
		for _, kw := range model.Keywords {
			if strings.Contains(textLower, kw) {
				matched = append(matched, kw)
			}
		}
		// Require at least 2 keyword matches to reduce false positives
		if len(matched) >= 2 {
			matches = append(matches, ModelImpactMatch{
				Model:           model,
				MatchedKeywords: matched,
			})
		}
	}

	return matches
}

// formatModelImpactAdvisory formats matched models as a readable advisory block.
// Returns empty string if no matches.
func formatModelImpactAdvisory(matches []ModelImpactMatch) string {
	if len(matches) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("┌─────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  MODEL IMPACT: Synthesis may affect these models            │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")

	for _, m := range matches {
		dirDisplay := m.Model.DirName
		if len(dirDisplay) > 35 {
			dirDisplay = dirDisplay[:32] + "..."
		}
		kwDisplay := strings.Join(m.MatchedKeywords, ", ")
		if len(kwDisplay) > 20 {
			kwDisplay = kwDisplay[:17] + "..."
		}
		line := fmt.Sprintf("│  %-35s (%s)", dirDisplay, kwDisplay)
		for len(line) < 62 {
			line += " "
		}
		sb.WriteString(line + "│\n")
	}

	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
	sb.WriteString("│  Check if these models need updating based on agent's work  │\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}
