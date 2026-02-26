// Package main provides auto-creation of implementation issues when architect agents complete.
// When an architect agent's SYNTHESIS.md recommends action (not "close"), this creates
// a triage:ready implementation issue with inferred skill, closing the architect→implement gap.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// regexNumberedPrefix matches "1. ", "2. ", etc. at the start of a string.
var regexNumberedPrefix = regexp.MustCompile(`^\d+\.\s+`)

// maybeAutoCreateImplementationIssue checks if a completed architect agent's synthesis
// recommends action, and if so, creates a triage:ready implementation issue.
// Returns the created issue ID or empty string if no issue was created.
func maybeAutoCreateImplementationIssue(skillName, beadsID, workspacePath string) string {
	// Only for architect skill
	if skillName != "architect" {
		return ""
	}

	// Parse synthesis
	if workspacePath == "" {
		return ""
	}
	synthesis, err := verify.ParseSynthesis(workspacePath)
	if err != nil || synthesis == nil {
		return ""
	}

	// Check if recommendation is actionable
	if !isActionableRecommendation(synthesis.Recommendation) {
		return ""
	}

	// Build the implementation issue
	title := buildImplementationTitle(synthesis, beadsID)
	description := buildImplementationDescription(synthesis, beadsID)
	skill := inferImplementationSkill(synthesis)

	// Labels: triage:ready for daemon pickup + skill hint
	labels := []string{"triage:ready"}
	if skill != "" {
		labels = append(labels, "skill:"+skill)
	}

	// Create the issue
	issue, err := beads.FallbackCreate(title, description, "task", 2, labels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to auto-create implementation issue: %v\n", err)
		return ""
	}

	fmt.Printf("\n┌─────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│  🔄 AUTO-CREATED IMPLEMENTATION ISSUE                       │\n")
	fmt.Printf("├─────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│  Issue: %-50s │\n", issue.ID)
	fmt.Printf("│  Skill: %-50s │\n", skill)
	fmt.Printf("│  From:  %-50s │\n", beadsID+" (architect)")
	fmt.Printf("└─────────────────────────────────────────────────────────────┘\n")

	return issue.ID
}

// isActionableRecommendation returns true if the synthesis recommendation
// indicates follow-up work is needed (not just closing the issue).
func isActionableRecommendation(recommendation string) bool {
	r := strings.ToLower(strings.TrimSpace(recommendation))
	switch r {
	case "implement", "escalate", "spawn", "continue", "fix", "refactor":
		return true
	default:
		return false
	}
}

// inferImplementationSkill determines the appropriate skill for the follow-up
// implementation based on synthesis content.
func inferImplementationSkill(synthesis *verify.Synthesis) string {
	// Combine relevant text for keyword analysis
	text := strings.ToLower(synthesis.TLDR + " " + synthesis.Next + " " + strings.Join(synthesis.NextActions, " "))

	// Debug/fix signals → systematic-debugging
	debugKeywords := []string{"fix", "debug", "bug", "crash", "error", "broken", "failing"}
	for _, kw := range debugKeywords {
		if strings.Contains(text, kw) {
			return "systematic-debugging"
		}
	}

	// Investigation signals → investigation
	investigationKeywords := []string{"investigate", "analyze", "understand", "explore", "root cause"}
	for _, kw := range investigationKeywords {
		if strings.Contains(text, kw) {
			return "investigation"
		}
	}

	// Default: feature-impl covers implement, refactor, add, create, extract, etc.
	return "feature-impl"
}

// buildImplementationTitle creates a concise title for the implementation issue.
// Uses the first next action if available, otherwise falls back to TLDR.
func buildImplementationTitle(synthesis *verify.Synthesis, beadsID string) string {
	suffix := fmt.Sprintf(" (from architect %s)", beadsID)

	if len(synthesis.NextActions) > 0 {
		action := cleanActionItem(synthesis.NextActions[0])
		return action + suffix
	}

	if synthesis.TLDR != "" {
		return "Implement: " + synthesis.TLDR + suffix
	}

	return "Implementation follow-up" + suffix
}

// buildImplementationDescription creates a detailed description for the implementation issue,
// including context from the architect's synthesis.
func buildImplementationDescription(synthesis *verify.Synthesis, beadsID string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Auto-created from architect review %s.\n\n", beadsID))

	if synthesis.TLDR != "" {
		b.WriteString("## Architect Summary\n")
		b.WriteString(synthesis.TLDR)
		b.WriteString("\n\n")
	}

	if len(synthesis.NextActions) > 0 {
		b.WriteString("## Recommended Actions\n")
		for _, action := range synthesis.NextActions {
			b.WriteString(action)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if synthesis.Next != "" {
		b.WriteString("## Architect Next Section\n")
		b.WriteString(synthesis.Next)
		b.WriteString("\n")
	}

	return b.String()
}

// cleanActionItem strips bullet/numbered prefixes from an action item string.
func cleanActionItem(item string) string {
	item = strings.TrimSpace(item)
	item = strings.TrimPrefix(item, "- ")
	item = strings.TrimPrefix(item, "* ")
	item = regexNumberedPrefix.ReplaceAllString(item, "")
	return item
}
