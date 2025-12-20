// Package question provides extraction of pending questions from agent output.
//
// This package handles multiple question patterns found in OpenCode agent output:
// 1. AskUserQuestion tool invocations (structured, highest priority)
// 2. Lines ending with '?' (natural language fallback)
package question

import (
	"regexp"
	"strings"
)

// askUserQuestionPattern matches the AskUserQuestion tool parameter format.
// Example: <parameter name="questions">[{"question": "Should I proceed?", ...}]</parameter>
var askUserQuestionPattern = regexp.MustCompile(`<parameter name="questions">\s*\[\s*\{\s*"question":\s*"([^"]+)"`)

// extractFromAskUserQuestion extracts a question from AskUserQuestion tool invocation.
//
// Looks for pattern:
// <parameter name="questions">[{"question": "...", ...}]</parameter>
//
// Returns the first question found, or empty string if none.
func extractFromAskUserQuestion(text string) string {
	if text == "" {
		return ""
	}

	match := askUserQuestionPattern.FindStringSubmatch(text)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return ""
}

// extractFromQuestionMarks extracts the most recent question ending with '?'.
//
// Searches from bottom to top to get the most recent question.
// When a line ending with '?' is found, looks backward to capture
// preceding lines that are part of the same multi-line question.
//
// Returns the most recent question (single or multi-line), or empty string.
func extractFromQuestionMarks(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")

	// Find the line ending with '?', searching from bottom to top
	questionEndIdx := -1
	for idx := len(lines) - 1; idx >= 0; idx-- {
		if strings.HasSuffix(strings.TrimSpace(lines[idx]), "?") {
			questionEndIdx = idx
			break
		}
	}

	if questionEndIdx == -1 {
		return ""
	}

	// Collect the question line and any preceding lines that are part of it
	questionLines := []string{strings.TrimSpace(lines[questionEndIdx])}

	// Look backward for additional lines that are part of the question
	for idx := questionEndIdx - 1; idx >= 0; idx-- {
		line := strings.TrimSpace(lines[idx])

		// Stop at blank lines (question boundary)
		if line == "" {
			break
		}

		// Stop at option markers (not part of question)
		if strings.HasPrefix(line, "❯") {
			break
		}

		// Stop at numbered options (1., 2., etc.)
		if len(line) > 0 && line[0] >= '0' && line[0] <= '9' {
			break
		}

		// This line is part of the question - prepend it
		questionLines = append([]string{line}, questionLines...)
	}

	// Join lines with space and return
	return strings.Join(questionLines, " ")
}

// Extract extracts a pending question from agent output.
//
// Handles multiple question patterns (in priority order):
// 1. AskUserQuestion tool invocations (highest priority)
// 2. Lines ending with '?' (fallback)
//
// Returns the extracted question text, or empty string if none found.
func Extract(text string) string {
	if text == "" {
		return ""
	}

	// Try AskUserQuestion pattern first (more structured)
	if question := extractFromAskUserQuestion(text); question != "" {
		return question
	}

	// Fall back to question mark pattern
	return extractFromQuestionMarks(text)
}
