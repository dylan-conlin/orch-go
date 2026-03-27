package routing

import "strings"

var areaKeywordMap = []struct {
	area     string
	keywords []string
}{
	{area: "dashboard", keywords: []string{"web/", "dashboard", "work graph", "serve_beads.go"}},
	{area: "spawn", keywords: []string{"pkg/spawn/", "cmd/orch/spawn", "spawn_", "/spawn/"}},
	{area: "beads", keywords: []string{"pkg/beads/", ".beads/", "beads"}},
	{area: "kb", keywords: []string{".kb/", "cmd/kb/", "pkg/kb"}},
	{area: "skill", keywords: []string{"skills/src/", "skillc", "skill.md"}},
	{area: "opencode", keywords: []string{"pkg/opencode/", "opencode", "claude code"}},
	{area: "cli", keywords: []string{"cmd/orch/", "cmd/desktop/", " cli "}},
}

// InferAreaFromText detects an area label from title/description file path keywords.
func InferAreaFromText(title, description string) string {
	text := " " + strings.ToLower(title+" "+description) + " "
	for _, candidate := range areaKeywordMap {
		for _, keyword := range candidate.keywords {
			if strings.Contains(text, keyword) {
				return candidate.area
			}
		}
	}
	return ""
}

// EnrichRoutingLabels adds inferred skill/area labels while preserving explicit labels.
func EnrichRoutingLabels(labels []string, issueType, title, description string) []string {
	if issueType == "" {
		return cloneLabels(labels)
	}

	enriched := cloneLabels(labels)
	if !hasPrefixedLabel(enriched, "skill:") {
		if skill, err := InferSkillForIssue(issueType, title, description, enriched); err == nil && skill != "" {
			enriched = appendUniqueLabel(enriched, "skill:"+skill)
		}
	}
	if !hasPrefixedLabel(enriched, "area:") {
		if area := InferAreaFromText(title, description); area != "" {
			enriched = appendUniqueLabel(enriched, "area:"+area)
		}
	}
	return enriched
}

func cloneLabels(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	cloned := make([]string, len(labels))
	copy(cloned, labels)
	return cloned
}

func hasPrefixedLabel(labels []string, prefix string) bool {
	prefix = strings.ToLower(prefix)
	for _, label := range labels {
		if strings.HasPrefix(strings.ToLower(label), prefix) {
			return true
		}
	}
	return false
}

func appendUniqueLabel(labels []string, label string) []string {
	for _, existing := range labels {
		if strings.EqualFold(existing, label) {
			return labels
		}
	}
	return append(labels, label)
}
