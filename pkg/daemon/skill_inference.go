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
	case "bug", "feature", "task", "investigation", "experiment", "question":
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
	case "experiment":
		return "investigation", nil
	case "question":
		return "architect", nil
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

// InferBrowserToolFromLabels extracts browser tool requirements from needs:* labels.
// Returns "playwright-cli" if needs:playwright label is found, or empty string otherwise.
//
// This allows daemon-spawned agents to automatically get browser automation context
// (playwright-cli) when working on UI/CSS fixes that require visual verification.
func InferBrowserToolFromLabels(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, "needs:") {
			need := strings.TrimPrefix(label, "needs:")
			switch need {
			case "playwright":
				return "playwright-cli" // Triggers playwright-cli context injection
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
			"design":               "architect",
			"debug":                "systematic-debugging",
			"investigation":        "investigation",
			"investigate":          "investigation",
			"explore":              "investigation",
			"experiment":           "investigation",
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
		"experiment":    "investigation",
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

// --- Capability classes ---
//
// Each skill belongs to a capability class that determines its model requirements.
// Deep-reasoning skills need sustained analysis and synthesis (opus).
// Implementation skills can use lighter models when effort is bounded.

const (
	// CapabilityDeepReasoning requires models with strong analysis/synthesis.
	// Skills: architect, investigation, systematic-debugging, research, codebase-audit.
	CapabilityDeepReasoning = "deep-reasoning"

	// CapabilityImplementation handles code changes. Model depends on effort.
	// Skills: feature-impl, reliability-testing.
	CapabilityImplementation = "implementation"

	// CapabilityLight handles lightweight tasks with no model requirement.
	// Skills: issue-creation, and unknown/unmapped skills.
	CapabilityLight = "light"
)

// skillCapabilityClass maps skills to their capability class.
var skillCapabilityClass = map[string]string{
	"systematic-debugging": CapabilityDeepReasoning,
	"investigation":        CapabilityDeepReasoning,
	"architect":            CapabilityDeepReasoning,
	"codebase-audit":       CapabilityDeepReasoning,
	"research":             CapabilityDeepReasoning,
	"feature-impl":         CapabilityImplementation,
	"reliability-testing":  CapabilityImplementation,
	"issue-creation":       CapabilityLight,
}

// SkillCapability returns the capability class for a skill.
// Returns CapabilityLight for unknown skills.
func SkillCapability(skill string) string {
	if cap, ok := skillCapabilityClass[skill]; ok {
		return cap
	}
	return CapabilityLight
}

// --- Model routing ---

// ModelRoute is the result of capability-aware model routing.
// Contains both the model alias and a human-readable reason for the choice.
type ModelRoute struct {
	// Model is the model alias (e.g., "opus", "gpt-5.4", or "" for resolve pipeline default).
	Model string
	// Reason explains why this model was chosen (for daemon logs and status display).
	Reason string
}

// RouteModel selects a model based on skill capability class, issue effort labels,
// and escalation state. This replaces the skill-only InferModelFromSkill with
// issue-aware routing.
//
// Routing rules:
//   - Deep-reasoning skills (architect, investigation, debugging, research) → opus
//   - Implementation skills + effort:small or effort:medium → gpt-5.4
//   - Implementation skills + effort:large or no effort label → "" (resolve pipeline default)
//   - Light/unknown skills → "" (resolve pipeline default)
//
// The resolve pipeline (pkg/spawn/resolve.go) still applies account config,
// repo defaults, and per-user overrides on top of this inference.
func RouteModel(skill string, issue *Issue) ModelRoute {
	capability := SkillCapability(skill)

	switch capability {
	case CapabilityDeepReasoning:
		return ModelRoute{
			Model:  "opus",
			Reason: fmt.Sprintf("deep-reasoning skill (%s) requires opus", skill),
		}

	case CapabilityImplementation:
		if issue == nil {
			return ModelRoute{
				Model:  "",
				Reason: fmt.Sprintf("implementation skill (%s), no issue context", skill),
			}
		}

		effort := InferEffortFromLabels(issue.Labels)
		switch effort {
		case "small":
			return ModelRoute{
				Model:  "gpt-5.4",
				Reason: fmt.Sprintf("implementation skill (%s) + effort:small → gpt-5.4", skill),
			}
		case "medium":
			return ModelRoute{
				Model:  "gpt-5.4",
				Reason: fmt.Sprintf("implementation skill (%s) + effort:medium → gpt-5.4", skill),
			}
		case "large":
			return ModelRoute{
				Model:  "",
				Reason: fmt.Sprintf("implementation skill (%s) + effort:large → resolve pipeline default", skill),
			}
		default:
			return ModelRoute{
				Model:  "",
				Reason: fmt.Sprintf("implementation skill (%s), no effort label → resolve pipeline default", skill),
			}
		}

	default: // CapabilityLight or unmapped
		return ModelRoute{
			Model:  "",
			Reason: fmt.Sprintf("light skill (%s) → resolve pipeline default", skill),
		}
	}
}

// InferEffortFromLabels extracts the effort level from labels.
// Returns "small", "medium", "large", or "" if no effort label is present.
func InferEffortFromLabels(labels []string) string {
	for _, label := range labels {
		switch {
		case strings.EqualFold(label, LabelEffortSmall):
			return "small"
		case strings.EqualFold(label, LabelEffortMedium):
			return "medium"
		case strings.EqualFold(label, LabelEffortLarge):
			return "large"
		}
	}
	return ""
}

// InferModelFromSkill returns the model alias for a given skill (without issue context).
// This is the backward-compatible wrapper used by coordination.go for extraction/escalation
// paths where issue context is not relevant (the skill alone determines the model).
//
// For issue-aware routing, use RouteModel instead.
func InferModelFromSkill(skill string) string {
	route := RouteModel(skill, nil)
	return route.Model
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
