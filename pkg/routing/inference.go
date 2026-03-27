package routing

import (
	"fmt"
	"strings"
)

// InferSkill maps issue types to skills.
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
func InferSkillFromLabels(labels []string) string {
	for _, label := range labels {
		lower := strings.ToLower(label)
		if strings.HasPrefix(lower, "skill:") {
			return strings.TrimSpace(label[len("skill:"):])
		}
	}
	return ""
}

// InferSkillFromTitle detects skills from issue title patterns.
func InferSkillFromTitle(title string) string {
	if title == "" {
		return ""
	}

	parts := strings.SplitN(title, ":", 2)
	if len(parts) >= 2 {
		skillPrefix := strings.ToLower(strings.TrimSpace(parts[0]))
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

	words := strings.Fields(strings.ToLower(title))
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
func InferSkillFromDescription(description string) string {
	if description == "" {
		return ""
	}

	lower := strings.ToLower(description)

	investigationKeywords := []string{
		"audit", "analyze", "correlate", "investigate", "understand",
		"how does", "how do", "what is", "why does", "why do",
	}
	for _, keyword := range investigationKeywords {
		if strings.Contains(lower, keyword) {
			return "investigation"
		}
	}

	researchKeywords := []string{
		"compare", "evaluate", "research", "best practice",
		"which should", "what should we use", "evaluate options",
	}
	for _, keyword := range researchKeywords {
		if strings.Contains(lower, keyword) {
			return "research"
		}
	}

	debugKeywords := []string{"fix", "broken", "error", "crash", "fails", "failing"}
	hasDebugKeyword := false
	for _, keyword := range debugKeywords {
		if strings.Contains(lower, keyword) {
			hasDebugKeyword = true
			break
		}
	}

	if hasDebugKeyword {
		causeIndicators := []string{
			"error:", "exception:", "stack trace", "at line",
			"in function", "returns", "expected", "actual",
			"reproduc", "when i", "steps:", "stacktrace",
		}
		for _, indicator := range causeIndicators {
			if strings.Contains(lower, indicator) {
				return "systematic-debugging"
			}
		}
	}

	return ""
}

// InferSkillForIssue determines the skill to use for an issue.
func InferSkillForIssue(issueType, title, description string, labels []string) (string, error) {
	if skill := InferSkillFromLabels(labels); skill != "" {
		return skill, nil
	}
	if skill := InferSkillFromTitle(title); skill != "" {
		return skill, nil
	}
	if skill := InferSkillFromDescription(description); skill != "" {
		return skill, nil
	}
	return InferSkill(issueType)
}
