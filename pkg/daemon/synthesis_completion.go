// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// synthesisTopicPattern matches titles like "Synthesize model investigations" or
// "Synthesize model investigations (11)" and extracts the topic word(s).
var synthesisTopicPattern = regexp.MustCompile(`(?i)^Synthesize\s+(.+?)\s+investigations`)

// ExtractSynthesisTopic extracts the topic from a synthesis issue title.
// Returns empty string if the title is not a synthesis issue.
//
// Examples:
//
//	"Synthesize model investigations (11)" -> "model"
//	"Synthesize daemon investigations" -> "daemon"
//	"Synthesize agent lifecycle investigations (5)" -> "agent lifecycle"
//	"Fix a bug in the code" -> ""
func ExtractSynthesisTopic(title string) string {
	matches := synthesisTopicPattern.FindStringSubmatch(title)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

// IsSynthesisCompleted checks whether a synthesis topic already has a corresponding
// guide or decision in the project's .kb directory.
//
// This prevents spawning agents for synthesis work that has already been completed.
// The check matches topic words against guide/decision filenames using the same
// logic as verify.DetectSynthesisOpportunities (hyphenated variants).
//
// projectDir is the root of the project containing .kb/. If empty, uses cwd.
func IsSynthesisCompleted(topic, projectDir string) bool {
	if topic == "" {
		return false
	}

	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return false
		}
	}

	kbDir := filepath.Join(projectDir, ".kb")
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		return false
	}

	// Normalize topic for matching
	topicLower := strings.ToLower(topic)
	topicWords := strings.Fields(topicLower)

	// Check guides
	guidesDir := filepath.Join(kbDir, "guides")
	if matchesExistingArtifact(guidesDir, topicLower, topicWords) {
		return true
	}

	// Check decisions
	decisionsDir := filepath.Join(kbDir, "decisions")
	if matchesExistingArtifact(decisionsDir, topicLower, topicWords) {
		return true
	}

	return false
}

// matchesExistingArtifact checks if any file in dir matches the topic.
// Matching logic: split filename by hyphens, check if any topic word appears.
// This mirrors the logic in verify.DetectSynthesisOpportunities.
func matchesExistingArtifact(dir, topicLower string, topicWords []string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		nameLower := strings.ToLower(name)

		// Remove date prefix from decision filenames (YYYY-MM-DD-)
		if len(nameLower) > 11 && nameLower[4] == '-' && nameLower[7] == '-' && nameLower[10] == '-' {
			nameLower = nameLower[11:]
		}

		// Split filename into parts for matching
		parts := strings.Split(nameLower, "-")

		// Check if any topic word matches any filename part
		for _, word := range topicWords {
			if len(word) <= 2 {
				continue // Skip short words
			}
			for _, part := range parts {
				if part == word {
					return true
				}
			}
			// Also check if the full topic appears in the filename
			if strings.Contains(nameLower, word) {
				return true
			}
		}
	}

	return false
}

// IsSynthesisIssue returns true if the issue title matches the synthesis pattern.
func IsSynthesisIssue(title string) bool {
	return synthesisTopicPattern.MatchString(title)
}

// CheckSynthesisCompletion checks if a synthesis issue's topic has already been
// completed. Returns a reason string if the spawn should be blocked, or empty
// string if the spawn should proceed.
//
// This is called from the daemon's spawn loop as a defense-in-depth check
// against the kb-cli dedup failure (where JSON parse errors cause "no duplicate"
// to be returned, allowing duplicate synthesis issues to be created).
func CheckSynthesisCompletion(issue *Issue, projectDir string) string {
	if issue == nil {
		return ""
	}

	topic := ExtractSynthesisTopic(issue.Title)
	if topic == "" {
		return "" // Not a synthesis issue
	}

	if IsSynthesisCompleted(topic, projectDir) {
		return fmt.Sprintf("synthesis already completed for topic %q (guide/decision exists)", topic)
	}

	return ""
}
