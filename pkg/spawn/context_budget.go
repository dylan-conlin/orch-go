package spawn

import "strings"

var kbLeastRelevantTrimOrder = []string{
	"### Open Questions",
	"### Failed Attempts (DO NOT repeat)",
	"### Related Investigations",
	"### Guides (procedural knowledge)",
	"### Models (synthesized understanding)",
	"### Prior Decisions",
}

const kbConstraintsSectionHeader = "### Constraints (MUST respect)"

func applySpawnContextBudget(cfg *Config, content string) (string, error) {
	budgetTokens := cfg.EffectiveContextBudget()
	if EstimateTokens(len(content)) <= budgetTokens {
		return content, nil
	}

	data := buildContextData(cfg)
	current := content
	currentKB := data.KBContext

	// 1) Keep skill instructions and task/issue context.
	// 2) Trim non-constraint KB entries from least relevant to most relevant.
	for EstimateTokens(len(current)) > budgetTokens {
		updatedKB, removed := removeLeastRelevantKBEntry(currentKB)
		if !removed {
			break
		}

		currentKB = updatedKB
		data.KBContext = currentKB

		var err error
		current, err = renderSpawnContextTemplate(data)
		if err != nil {
			return "", err
		}
	}

	// 3) If still over budget, trim constraints as a last KB-context resort.
	for EstimateTokens(len(current)) > budgetTokens {
		updatedKB, removed := removeLastTopLevelEntryFromSection(currentKB, kbConstraintsSectionHeader)
		if !removed {
			break
		}

		currentKB = updatedKB
		data.KBContext = currentKB

		var err error
		current, err = renderSpawnContextTemplate(data)
		if err != nil {
			return "", err
		}
	}

	// Remove any remaining KB wrapper text if the context still exceeds budget.
	if EstimateTokens(len(current)) > budgetTokens && strings.TrimSpace(currentKB) != "" {
		data.KBContext = ""
		var err error
		current, err = renderSpawnContextTemplate(data)
		if err != nil {
			return "", err
		}
	}

	if EstimateTokens(len(current)) > budgetTokens {
		current = hardTruncateToBudget(current, budgetTokens)
	}

	return current, nil
}

func removeLeastRelevantKBEntry(kbContext string) (string, bool) {
	for _, sectionHeader := range kbLeastRelevantTrimOrder {
		updated, removed := removeLastTopLevelEntryFromSection(kbContext, sectionHeader)
		if removed {
			return updated, true
		}
	}

	return kbContext, false
}

func removeLastTopLevelEntryFromSection(content, sectionHeader string) (string, bool) {
	if strings.TrimSpace(content) == "" {
		return content, false
	}

	lines := strings.Split(content, "\n")
	start := findSectionStartLine(lines, sectionHeader)
	if start == -1 {
		return content, false
	}

	end := findSectionEndLine(lines, start+1)
	if end <= start+1 {
		return content, false
	}

	entryStart := -1
	for i := start + 1; i < end; i++ {
		if strings.HasPrefix(lines[i], "- ") {
			entryStart = i
		}
	}

	if entryStart == -1 {
		return content, false
	}

	entryEnd := end
	for i := entryStart + 1; i < end; i++ {
		if strings.HasPrefix(lines[i], "- ") {
			entryEnd = i
			break
		}
	}

	updated := make([]string, 0, len(lines)-(entryEnd-entryStart))
	updated = append(updated, lines[:entryStart]...)
	updated = append(updated, lines[entryEnd:]...)

	return strings.Join(updated, "\n"), true
}

func findSectionStartLine(lines []string, sectionHeader string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == sectionHeader {
			return i
		}
	}

	return -1
}

func findSectionEndLine(lines []string, from int) int {
	for i := from; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "### ") || strings.HasPrefix(trimmed, "**IMPORTANT:**") {
			return i
		}
	}

	return len(lines)
}

func hardTruncateToBudget(content string, budgetTokens int) string {
	maxChars := budgetTokens * CharsPerToken
	if maxChars <= 0 || len(content) <= maxChars {
		return content
	}

	marker := "\n\n...[truncated to fit context budget]...\n"
	if len(marker) >= maxChars {
		return content[:maxChars]
	}

	keepChars := maxChars - len(marker)
	trimmed := strings.TrimSpace(content[:keepChars])
	if breakAt := strings.LastIndex(trimmed, "\n"); breakAt > keepChars/2 {
		trimmed = strings.TrimSpace(trimmed[:breakAt])
	}

	return trimmed + marker
}
