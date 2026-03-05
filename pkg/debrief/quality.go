package debrief

import (
	"fmt"
	"strings"
)

// QualityWarning describes a single quality issue detected in a debrief.
type QualityWarning struct {
	Pattern string // Machine-readable pattern name (e.g., "empty_learned", "action_verb_only")
	Message string // Human-readable advisory message
}

// QualityResult holds the outcome of a debrief quality check.
type QualityResult struct {
	Pass     bool             // True if no critical warnings detected
	Warnings []QualityWarning // Advisory warnings (may be non-empty even when Pass=true)
}

// actionVerbPrefixes are past-tense action verbs that indicate event-log style
// rather than comprehension. These start sentences in "what happened" summaries.
var actionVerbPrefixes = []string{
	"added", "fixed", "implemented", "updated", "refactored",
	"removed", "created", "deleted", "moved", "merged",
	"deployed", "configured", "migrated", "resolved", "completed",
	"built", "shipped", "released", "extracted", "consolidated",
}

// connectiveWords are words/phrases that indicate causal or relational reasoning
// — the connective tissue of comprehension vs event logging.
var connectiveWords = []string{
	"because", "which means", "therefore", "implies",
	"this means", "so that", "in order to", "as a result",
	"the reason", "matters because", "discovered that",
	"realized that", "learned that", "turns out",
	"the insight", "the key", "what this tells",
}

// CheckQuality runs advisory heuristics on a debrief and returns warnings.
// This is non-blocking — it flags patterns but doesn't prevent debrief creation.
func CheckQuality(data *DebriefData) QualityResult {
	var warnings []QualityWarning

	// Check 1: Empty "What We Learned" section
	if len(data.WhatWeLearned) == 0 {
		warnings = append(warnings, QualityWarning{
			Pattern: "empty_learned",
			Message: "\"What We Learned\" is empty — this is the most valuable section of the debrief",
		})
	}

	// Check 2: Action-verb-only in "What We Learned"
	if len(data.WhatWeLearned) > 0 {
		actionCount := 0
		for _, item := range data.WhatWeLearned {
			if IsActionVerbSentence(item) {
				actionCount++
			}
		}
		if actionCount == len(data.WhatWeLearned) {
			warnings = append(warnings, QualityWarning{
				Pattern: "action_verb_only",
				Message: "All \"What We Learned\" items start with action verbs (Added, Fixed, etc.) — these describe what happened, not what was learned",
			})
		}
	}

	// Check 3: Missing connective language in "What We Learned"
	if len(data.WhatWeLearned) > 0 {
		connCount := 0
		for _, item := range data.WhatWeLearned {
			if HasConnectiveLanguage(item) {
				connCount++
			}
		}
		if connCount == 0 {
			warnings = append(warnings, QualityWarning{
				Pattern: "missing_connectives",
				Message: "No connective language found (because, which means, therefore, implies) — insights connect observations to meaning",
			})
		}
	}

	// Pass if no critical warnings (empty_learned or all-action-verb or no-connectives)
	pass := true
	for _, w := range warnings {
		switch w.Pattern {
		case "empty_learned", "action_verb_only", "missing_connectives":
			pass = false
		}
	}

	return QualityResult{
		Pass:     pass,
		Warnings: warnings,
	}
}

// IsActionVerbSentence returns true if the sentence starts with a past-tense
// action verb, indicating it's an event-log summary rather than an insight.
func IsActionVerbSentence(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, prefix := range actionVerbPrefixes {
		if strings.HasPrefix(lower, prefix+" ") || strings.HasPrefix(lower, prefix+".") {
			return true
		}
	}
	return false
}

// HasConnectiveLanguage returns true if the text contains words/phrases that
// indicate causal or relational reasoning.
func HasConnectiveLanguage(s string) bool {
	lower := strings.ToLower(s)
	for _, conn := range connectiveWords {
		if strings.Contains(lower, conn) {
			return true
		}
	}
	return false
}

// ComprehensionPrompt returns the prompt text printed after auto-populating
// facts, guiding the orchestrator toward synthesis.
func ComprehensionPrompt() string {
	return `
Comprehension check — review "What We Learned" before finalizing:

  Thread:   What were you working on? (context for the insight)
  Insight:  What did you learn? (the durable value)
  Position: How does this change your approach? (forward impact)

Use --learned to add insights: orch debrief --learned "X because Y"
`
}

// FormatQualityWarnings renders quality warnings for terminal display.
func FormatQualityWarnings(result QualityResult) string {
	if len(result.Warnings) == 0 {
		return ""
	}

	var b strings.Builder
	if result.Pass {
		b.WriteString("Quality: PASS (with notes)\n")
	} else {
		b.WriteString("Quality: ADVISORY warnings detected\n")
	}
	for _, w := range result.Warnings {
		b.WriteString(fmt.Sprintf("  ⚠ [%s] %s\n", w.Pattern, w.Message))
	}
	return b.String()
}
