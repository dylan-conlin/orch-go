// Package main provides the synthesis checkpoint trigger for orch complete.
// When an agent completes, this checks for synthesis opportunities related to
// the completed work's topic area and surfaces them as an advisory.
//
// Uses cached reflect suggestions from the daemon to avoid shelling out to kb
// reflect on every completion. Topics matching the agent's keywords are surfaced
// with action suggestions.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

// CompletionSynthesisThreshold is the minimum investigation count for a topic
// to be surfaced during completion. Lower than SynthesisWarningThreshold (10)
// because completion-time suggestions are contextually targeted.
const CompletionSynthesisThreshold = 3

// findMatchingSynthesisTopics finds synthesis suggestions whose topic matches
// any of the provided keywords. Returns topics at or above CompletionSynthesisThreshold.
func findMatchingSynthesisTopics(suggestions *daemon.ReflectSuggestions, keywords []string) []daemon.SynthesisSuggestion {
	if suggestions == nil || len(suggestions.Synthesis) == 0 || len(keywords) == 0 {
		return nil
	}

	var matches []daemon.SynthesisSuggestion
	for _, s := range suggestions.Synthesis {
		if s.Count < CompletionSynthesisThreshold {
			continue
		}
		topicLower := strings.ToLower(s.Topic)
		for _, kw := range keywords {
			kwLower := strings.ToLower(kw)
			if strings.Contains(topicLower, kwLower) || strings.Contains(kwLower, topicLower) {
				matches = append(matches, s)
				break
			}
		}
	}

	return matches
}

// formatSynthesisCheckpointAdvisory formats matching synthesis topics into an
// advisory string for display during orch complete.
func formatSynthesisCheckpointAdvisory(topics []daemon.SynthesisSuggestion) string {
	if len(topics) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("┌─────────────────────────────────────────────────────────────┐\n")
	b.WriteString("│  📚 SYNTHESIS CHECKPOINT                                    │\n")
	b.WriteString("├─────────────────────────────────────────────────────────────┤\n")

	for _, topic := range topics {
		line := fmt.Sprintf("│  • %s: %d investigations", topic.Topic, topic.Count)
		// Pad to box width
		if len(line) < 62 {
			line += strings.Repeat(" ", 62-len(line))
		}
		line += "│\n"
		b.WriteString(line)

		// For high-count topics, suggest chronicle
		if topic.Count >= SynthesisWarningThreshold {
			suggestion := fmt.Sprintf("│    → kb chronicle \"%s\"", topic.Topic)
			if len(suggestion) < 62 {
				suggestion += strings.Repeat(" ", 62-len(suggestion))
			}
			suggestion += "│\n"
			b.WriteString(suggestion)
		}
	}

	b.WriteString("├─────────────────────────────────────────────────────────────┤\n")
	b.WriteString("│  Related topics may benefit from synthesis/consolidation    │\n")
	b.WriteString("└─────────────────────────────────────────────────────────────┘\n")

	return b.String()
}

// suggestionsAreFresh checks if cached reflect suggestions are recent enough
// to be useful. Returns false if nil or older than SuggestionFreshnessHours.
func suggestionsAreFresh(suggestions *daemon.ReflectSuggestions) bool {
	if suggestions == nil {
		return false
	}
	return time.Since(suggestions.Timestamp).Hours() <= float64(SuggestionFreshnessHours)
}

// RunSynthesisCheckpoint loads cached reflect suggestions, finds topics matching
// the completed agent's work area, and returns a formatted advisory string.
// Returns empty string if no relevant synthesis opportunities are found.
func RunSynthesisCheckpoint(skillName, issueTitle, phaseSummary string) string {
	// Extract keywords from the completed work
	keywords := extractCompletionKeywords(skillName, issueTitle, phaseSummary)
	if len(keywords) == 0 {
		return ""
	}

	// Load cached suggestions (from daemon's periodic reflect)
	suggestions, err := daemon.LoadSuggestions()
	if err != nil || !suggestionsAreFresh(suggestions) {
		return ""
	}

	// Find matching topics
	matches := findMatchingSynthesisTopics(suggestions, keywords)
	if len(matches) == 0 {
		return ""
	}

	return formatSynthesisCheckpointAdvisory(matches)
}
