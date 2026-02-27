// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// IsSpawnableType returns true if the issue type can be spawned.
func IsSpawnableType(issueType string) bool {
	switch issueType {
	case "bug", "feature", "task", "investigation":
		return true
	default:
		return false
	}
}

// InferSkill maps issue types to skills.
//
// Bug handling: Routes to "systematic-debugging" which includes Phase 1
// (root cause investigation) ensuring understanding before fixing.
// Use explicit skill:architect label when architectural review is needed.
func InferSkill(issueType string) (string, error) {
	switch issueType {
	case "bug":
		return "systematic-debugging", nil
	case "feature":
		return "feature-impl", nil
	case "task":
		return "feature-impl", nil
	case "investigation":
		return "investigation", nil
	default:
		return "", fmt.Errorf("cannot infer skill for issue type: %s", issueType)
	}
}

// InferSkillFromLabels extracts a skill name from skill:* labels.
// Returns the skill name if found (e.g., "research" from "skill:research"),
// or empty string if no skill label is present.
func InferSkillFromLabels(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, "skill:") {
			return strings.TrimPrefix(label, "skill:")
		}
	}
	return ""
}

// InferMCPFromLabels extracts MCP server requirements from needs:* labels.
// Returns the MCP server name if found (e.g., "playwright" from "needs:playwright"),
// or empty string if no MCP-related label is present.
//
// Supported labels:
//   - needs:playwright → returns "playwright" (browser automation for UI verification)
//
// This allows daemon-spawned agents to automatically get browser access when
// working on UI/CSS fixes that require visual verification.
func InferMCPFromLabels(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, "needs:") {
			need := strings.TrimPrefix(label, "needs:")
			// Map needs labels to MCP servers
			switch need {
			case "playwright":
				return "playwright"
				// Future: add more mappings as needed
				// case "browser-use":
				//     return "browser-use"
			}
		}
	}
	return ""
}

// InferSkillFromTitle detects skills from issue title patterns.
// Returns the skill name if a known pattern is matched, or empty string otherwise.
//
// Detection order:
//  1. Colon-prefix pattern: "Architect: Design system" → architect
//  2. First-word keyword: "Investigate Claude Code --worktree" → investigation
//
// The first-word check catches titles that use natural language (no colon prefix)
// like "Design orchestrator diagnostic mode" or "Investigate spawn failures".
func InferSkillFromTitle(title string) string {
	if title == "" {
		return ""
	}

	// Check for "SkillName: ..." colon-prefix pattern
	parts := strings.SplitN(title, ":", 2)
	if len(parts) >= 2 {
		// Extract and normalize the potential skill name
		skillPrefix := strings.ToLower(strings.TrimSpace(parts[0]))

		// Map single-word title prefixes to known skills
		prefixMap := map[string]string{
			"architect":            "architect",
			"design":              "architect",
			"debug":                "systematic-debugging",
			"investigation":        "investigation",
			"investigate":          "investigation",
			"explore":              "investigation",
			"research":             "research",
			"feature":              "feature-impl",
			"implement":            "feature-impl",
			"fix":                  "systematic-debugging",
			"broken":               "systematic-debugging",
			"systematic-debugging": "systematic-debugging",
			"feature-impl":         "feature-impl",
		}

		if skill, ok := prefixMap[skillPrefix]; ok {
			return skill
		}
	}

	// First-word keyword detection for natural language titles
	// e.g., "Investigate Claude Code --worktree flag" → investigation
	// e.g., "Design orchestrator diagnostic mode" → architect
	lower := strings.ToLower(title)
	words := strings.Fields(lower)
	if len(words) == 0 {
		return ""
	}

	firstWordMap := map[string]string{
		"investigate":   "investigation",
		"investigation": "investigation",
		"explore":       "investigation",
		"design":        "architect",
		"architect":     "architect",
		"debug":         "systematic-debugging",
		"fix":           "systematic-debugging",
		"broken":        "systematic-debugging",
	}

	if skill, ok := firstWordMap[words[0]]; ok {
		return skill
	}

	return ""
}

// InferSkillFromDescription detects skills from issue description text.
// Uses keyword/pattern matching as a fallback heuristic after label and title checks.
// Returns the skill name if a known pattern is matched, or empty string otherwise.
func InferSkillFromDescription(description string) string {
	if description == "" {
		return ""
	}

	lower := strings.ToLower(description)

	// Investigation signals: audit/analyze/correlate/investigate/understand/how does
	investigationKeywords := []string{
		"audit", "analyze", "correlate", "investigate", "understand",
		"how does", "how do", "what is", "why does", "why do",
	}
	for _, keyword := range investigationKeywords {
		if strings.Contains(lower, keyword) {
			return "investigation"
		}
	}

	// Research signals: compare/evaluate/research/best practice
	researchKeywords := []string{
		"compare", "evaluate", "research", "best practice",
		"which should", "what should we use", "evaluate options",
	}
	for _, keyword := range researchKeywords {
		if strings.Contains(lower, keyword) {
			return "research"
		}
	}

	// Debugging signals: fix/broken/error/crash/fails
	// When cause is described → systematic-debugging
	// When vague → architect (let InferSkill decide based on type)
	debugKeywords := []string{"fix", "broken", "error", "crash", "fails", "failing"}
	hasDebugKeyword := false
	for _, keyword := range debugKeywords {
		if strings.Contains(lower, keyword) {
			hasDebugKeyword = true
			break
		}
	}

	if hasDebugKeyword {
		// Check if cause is described (has specific error messages, stack traces, or detailed symptoms)
		causeIndicators := []string{
			"error:", "exception:", "stack trace", "at line",
			"in function", "returns", "expected", "actual",
			"reproduc", "when i", "steps:", "stacktrace",
		}
		hasCauseDescription := false
		for _, indicator := range causeIndicators {
			if strings.Contains(lower, indicator) {
				hasCauseDescription = true
				break
			}
		}

		if hasCauseDescription {
			return "systematic-debugging"
		}
		// Vague bug - return empty to fall back to type-based inference (systematic-debugging)
	}

	return ""
}

// InferSkillFromIssue determines the skill to use for an issue.
// Priority order: skill:* label > title pattern > description heuristic > issue type inference > error
// This respects explicit skill assignments via labels while falling back
// to description and type-based inference for issues without skill labels.
//
// Logs a spawn.skill_inferred event to events.jsonl for post-hoc accuracy analysis.
func InferSkillFromIssue(issue *Issue) (string, error) {
	if issue == nil {
		return "", fmt.Errorf("cannot infer skill for nil issue")
	}

	// Track which method was used for inference
	hadSkillLabel := false
	hadTitleMatch := false
	usedDescriptionHeuristic := false
	inferredSkill := ""

	// First, check for explicit skill:* label
	if skill := InferSkillFromLabels(issue.Labels); skill != "" {
		inferredSkill = skill
		hadSkillLabel = true
	} else if skill := InferSkillFromTitle(issue.Title); skill != "" {
		// Check for title-based patterns
		inferredSkill = skill
		hadTitleMatch = true
	} else if skill := InferSkillFromDescription(issue.Description); skill != "" {
		// Check for description-based heuristics
		inferredSkill = skill
		usedDescriptionHeuristic = true
	} else {
		// Fall back to type-based inference
		skill, err := InferSkill(issue.IssueType)
		if err != nil {
			return "", err
		}
		inferredSkill = skill
	}

	// Log inference event for accuracy tracking
	logSkillInference(issue.ID, inferredSkill, issue.IssueType, issue.Title,
		hadSkillLabel, hadTitleMatch, usedDescriptionHeuristic)

	return inferredSkill, nil
}

// skillModelMapping maps skill names to model aliases.
// Skills requiring deep reasoning get opus; implementation skills get sonnet.
var skillModelMapping = map[string]string{
	"systematic-debugging": "opus",
	"investigation":        "opus",
	"architect":            "opus",
	"codebase-audit":       "opus",
	"research":             "opus",
}

// InferModelFromSkill returns the appropriate model alias for a given skill.
// Deep reasoning skills (debugging, investigation, architecture) → opus.
// Implementation skills (feature-impl, issue-creation) → empty string (use resolve pipeline defaults).
// Unknown skills → empty string (use resolve pipeline defaults).
//
// Returns empty string when the skill has no explicit model requirement.
// This allows the resolve pipeline (pkg/spawn/resolve.go) to respect user config
// default_model instead of the daemon overriding it with a hardcoded default.
// Only skills with explicit requirements in skillModelMapping get a model override.
func InferModelFromSkill(skill string) string {
	if model, ok := skillModelMapping[skill]; ok {
		return model
	}
	return "" // Let resolve pipeline handle default model selection
}

// logSkillInference logs a skill inference event to events.jsonl.
// This is a separate function to allow testing InferSkillFromIssue without filesystem dependencies.
func logSkillInference(issueID, inferredSkill, issueType, title string,
	hadSkillLabel, hadTitleMatch, usedDescriptionHeuristic bool) {
	// Use default logger to append to ~/.orch/events.jsonl
	logger := events.NewDefaultLogger()
	_ = logger.LogSkillInferred(events.SkillInferredData{
		IssueID:                  issueID,
		InferredSkill:            inferredSkill,
		IssueType:                issueType,
		Title:                    title,
		HadSkillLabel:            hadSkillLabel,
		HadTitleMatch:            hadTitleMatch,
		UsedDescriptionHeuristic: usedDescriptionHeuristic,
	})
}
