// Package verify provides verification helpers for agent completion.
// This file handles knowledge gap detection - cross-checking agent questions against existing kb.
package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// KnowledgeGap represents a detected gap where an agent surfaced a question that kb already answers.
type KnowledgeGap struct {
	Timestamp   string   `json:"timestamp"`
	BeadsID     string   `json:"beads_id,omitempty"`
	Workspace   string   `json:"workspace"`
	Question    string   `json:"question"`
	KBMatches   []string `json:"kb_matches"`   // Paths to kb artifacts that could have answered this
	MatchTypes  []string `json:"match_types"`  // Types of matches (constraint, decision, guide, etc.)
	SearchQuery string   `json:"search_query"` // Keywords used for kb context search
	Skill       string   `json:"skill,omitempty"`
	ProjectDir  string   `json:"project_dir,omitempty"`
}

// KnowledgeGapResult holds the outcome of gap detection for a workspace.
type KnowledgeGapResult struct {
	GapsDetected int
	Gaps         []KnowledgeGap
}

// DetectKnowledgeGaps analyzes SYNTHESIS.md for open questions and cross-checks against kb.
// Returns gaps where kb has relevant knowledge that the agent didn't surface.
//
// Algorithm:
// 1. Parse SYNTHESIS.md for 'Unexplored Questions' or 'Open Questions' section
// 2. Extract key terms from each question
// 3. Run kb context with those terms
// 4. If kb returns constraints/decisions/guides → log as surfacing gap
//
// projectDir should be the directory where kb context should be run (target project).
func DetectKnowledgeGaps(workspacePath, beadsID, skill, projectDir string) (*KnowledgeGapResult, error) {
	result := &KnowledgeGapResult{
		Gaps: []KnowledgeGap{},
	}

	// Parse SYNTHESIS.md
	synthesis, err := ParseSynthesis(workspacePath)
	if err != nil {
		// No SYNTHESIS.md or parse error - not a failure, just no gaps to detect
		return result, nil
	}

	// No unexplored questions section - no gaps to detect
	if synthesis.UnexploredQuestions == "" {
		return result, nil
	}

	// Extract questions from the unexplored questions section
	questions := extractQuestions(synthesis.UnexploredQuestions)
	if len(questions) == 0 {
		return result, nil
	}

	// Determine workspace name for logging
	workspaceName := filepath.Base(workspacePath)

	// Cross-check each question against kb
	for _, question := range questions {
		// Extract keywords from question (using spawn's ExtractKeywords)
		keywords := spawn.ExtractKeywords(question, 5) // Max 5 keywords per question
		if keywords == "" {
			continue
		}

		// Query kb context with these keywords
		// Use projectDir for kb context search to ensure we search the right project
		kbResult, err := spawn.RunKBContextCheckWithDomain(keywords, spawn.DomainPersonal, projectDir)
		if err != nil {
			// KB query failed - not a fatal error, just skip this question
			continue
		}

		// No kb matches - not a gap (agent correctly identified an unexplored area)
		if kbResult == nil || !kbResult.HasMatches {
			continue
		}

		// Filter matches to only include high-value knowledge types
		// (constraints, decisions, guides) - investigations are expected to be unknown
		relevantMatches := filterRelevantMatches(kbResult.Matches)
		if len(relevantMatches) == 0 {
			continue
		}

		// This is a gap - kb has relevant knowledge that agent didn't surface
		gap := KnowledgeGap{
			Timestamp:   time.Now().Format(time.RFC3339),
			BeadsID:     beadsID,
			Workspace:   workspaceName,
			Question:    question,
			KBMatches:   extractMatchPaths(relevantMatches),
			MatchTypes:  extractMatchTypes(relevantMatches),
			SearchQuery: keywords,
			Skill:       skill,
			ProjectDir:  projectDir,
		}

		result.Gaps = append(result.Gaps, gap)
	}

	result.GapsDetected = len(result.Gaps)
	return result, nil
}

// LogKnowledgeGaps appends detected gaps to ~/.orch/knowledge-gaps.jsonl
func LogKnowledgeGaps(gaps []KnowledgeGap) error {
	if len(gaps) == 0 {
		return nil
	}

	// Ensure ~/.orch directory exists
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	orchDir := filepath.Join(homeDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		return fmt.Errorf("failed to create .orch directory: %w", err)
	}

	gapLogPath := filepath.Join(orchDir, "knowledge-gaps.jsonl")

	// Open file for appending (create if doesn't exist)
	f, err := os.OpenFile(gapLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open knowledge-gaps.jsonl: %w", err)
	}
	defer f.Close()

	// Write each gap as a JSON line
	encoder := json.NewEncoder(f)
	for _, gap := range gaps {
		if err := encoder.Encode(gap); err != nil {
			return fmt.Errorf("failed to write gap to log: %w", err)
		}
	}

	return nil
}

// extractQuestions extracts individual questions from the Unexplored Questions section.
// Handles both bullet point format and plain text questions.
func extractQuestions(unexploredSection string) []string {
	var questions []string
	lines := strings.Split(unexploredSection, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip section headers (bold text like "**Areas worth exploring:**")
		if strings.HasPrefix(line, "**") && strings.HasSuffix(line, ":**") {
			continue
		}

		// Strip bullet point markers
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")

		// Skip numbered list markers (e.g., "1. ")
		if len(line) > 3 && line[0] >= '0' && line[0] <= '9' && line[1] == '.' && line[2] == ' ' {
			line = line[3:]
		}

		line = strings.TrimSpace(line)

		// Only include lines that are actually questions or substantive statements
		// (at least 15 characters to filter out noise)
		if len(line) >= 15 {
			questions = append(questions, line)
		}
	}

	return questions
}

// filterRelevantMatches filters kb matches to only include high-value knowledge types.
// Constraints, decisions, and guides are relevant. Investigations are not (they're exploratory).
func filterRelevantMatches(matches []spawn.KBContextMatch) []spawn.KBContextMatch {
	var relevant []spawn.KBContextMatch
	for _, match := range matches {
		if match.Type == "constraint" || match.Type == "decision" || match.Type == "guide" {
			relevant = append(relevant, match)
		}
	}
	return relevant
}

// extractMatchPaths extracts file paths from kb matches.
func extractMatchPaths(matches []spawn.KBContextMatch) []string {
	var paths []string
	for _, match := range matches {
		if match.Path != "" {
			paths = append(paths, match.Path)
		} else if match.Reason != "" {
			// For kn entries, use the reason as identifier
			paths = append(paths, fmt.Sprintf("kn: %s", match.Reason))
		}
	}
	return paths
}

// extractMatchTypes extracts match types from kb matches.
func extractMatchTypes(matches []spawn.KBContextMatch) []string {
	var types []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if !seen[match.Type] {
			types = append(types, match.Type)
			seen[match.Type] = true
		}
	}
	return types
}
