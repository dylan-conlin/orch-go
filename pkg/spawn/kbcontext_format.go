package spawn

import (
	"fmt"
	"strings"
)

// FormatContextForSpawn formats kb context matches for inclusion in SPAWN_CONTEXT.md.
// This is a convenience wrapper around FormatContextForSpawnWithLimit that uses
// the default MaxKBContextChars limit.
func FormatContextForSpawn(result *KBContextResult) string {
	formatResult := FormatContextForSpawnWithLimit(result, MaxKBContextChars)
	return formatResult.Content
}

// FormatContextForSpawnWithLimit formats kb context with a character limit to prevent token bloat.
// Returns detailed information about the formatting including truncation status.
// Priority order for truncation: investigations (lowest) → decisions → constraints (highest).
func FormatContextForSpawnWithLimit(result *KBContextResult, maxChars int) *KBContextFormatResult {
	return FormatContextForSpawnWithLimitAndMeta(result, maxChars, ".", nil)
}

// FormatContextForSpawnWithLimitAndMeta formats kb context with a character limit and staleness metadata.
// projectDir controls staleness checks for model references. When stalenessMeta is provided,
// stale model detections will be recorded for daemon consumption.
func FormatContextForSpawnWithLimitAndMeta(result *KBContextResult, maxChars int, projectDir string, stalenessMeta *StalenessEventMeta) *KBContextFormatResult {
	emptyResult := &KBContextFormatResult{
		Content:          "",
		WasTruncated:     false,
		OriginalMatches:  0,
		TruncatedMatches: 0,
		EstimatedTokens:  0,
	}

	if result == nil || !result.HasMatches {
		return emptyResult
	}

	originalMatchCount := len(result.Matches)

	// Group by type for prioritized truncation
	constraints := filterByType(result.Matches, "constraint")
	decisions := filterByType(result.Matches, "decision")
	models := filterByType(result.Matches, "model")
	guides := filterByType(result.Matches, "guide")
	investigations := filterByType(result.Matches, "investigation")
	failedAttempts := filterByType(result.Matches, "failed-attempt")
	openQuestions := filterByType(result.Matches, "open-question")

	// Try to format with all matches first
	content, hasStaleModels := formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil, projectDir, stalenessMeta)

	// Extract primary model path if models exist
	primaryModelPath := ""
	if len(models) > 0 && models[0].Path != "" {
		primaryModelPath = models[0].Path
	}

	// Detect cross-repo model situation
	crossRepoModelDir := DetectCrossRepoModel(primaryModelPath, projectDir)

	// Check if we need to truncate
	if len(content) <= maxChars {
		return &KBContextFormatResult{
			Content:           content,
			WasTruncated:      false,
			OriginalMatches:   originalMatchCount,
			TruncatedMatches:  originalMatchCount,
			EstimatedTokens:   EstimateTokens(len(content)),
			HasInjectedModels: hasInjectedModelContent(models),
			PrimaryModelPath:  primaryModelPath,
			HasStaleModels:    hasStaleModels,
			CrossRepoModelDir: crossRepoModelDir,
		}
	}

	// Need to truncate - apply priority-based reduction
	// Priority: constraints (keep most) > decisions > models > guides > investigations > failed attempts > open questions (drop first)
	var omittedCategories []string
	truncatedMatches := originalMatchCount

	// First, try removing open questions one at a time
	for len(content) > maxChars && len(openQuestions) > 0 {
		openQuestions = openQuestions[:len(openQuestions)-1]
		truncatedMatches--
		content, hasStaleModels = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil, projectDir, stalenessMeta)
	}
	if len(filterByType(result.Matches, "open-question")) > len(openQuestions) {
		omittedCategories = append(omittedCategories, "open-question")
	}

	// If still too large, remove failed attempts one at a time
	for len(content) > maxChars && len(failedAttempts) > 0 {
		failedAttempts = failedAttempts[:len(failedAttempts)-1]
		truncatedMatches--
		content, hasStaleModels = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil, projectDir, stalenessMeta)
	}
	if len(filterByType(result.Matches, "failed-attempt")) > len(failedAttempts) {
		omittedCategories = append(omittedCategories, "failed-attempt")
	}

	// If still too large, remove investigations one at a time
	for len(content) > maxChars && len(investigations) > 0 {
		investigations = investigations[:len(investigations)-1]
		truncatedMatches--
		content, hasStaleModels = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil, projectDir, stalenessMeta)
	}
	if len(filterByType(result.Matches, "investigation")) > len(investigations) {
		omittedCategories = append(omittedCategories, "investigation")
	}

	// If still too large, remove guides one at a time
	for len(content) > maxChars && len(guides) > 0 {
		guides = guides[:len(guides)-1]
		truncatedMatches--
		content, hasStaleModels = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil, projectDir, stalenessMeta)
	}
	if len(filterByType(result.Matches, "guide")) > len(guides) {
		omittedCategories = append(omittedCategories, "guide")
	}

	// If still too large, remove models one at a time
	for len(content) > maxChars && len(models) > 0 {
		models = models[:len(models)-1]
		truncatedMatches--
		content, hasStaleModels = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil, projectDir, stalenessMeta)
	}
	if len(filterByType(result.Matches, "model")) > len(models) {
		omittedCategories = append(omittedCategories, "model")
	}

	// If still too large, remove decisions one at a time
	for len(content) > maxChars && len(decisions) > 0 {
		decisions = decisions[:len(decisions)-1]
		truncatedMatches--
		content, hasStaleModels = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil, projectDir, stalenessMeta)
	}
	if len(filterByType(result.Matches, "decision")) > len(decisions) {
		omittedCategories = append(omittedCategories, "decision")
	}

	// If STILL too large, remove constraints one at a time (last resort)
	for len(content) > maxChars && len(constraints) > 0 {
		constraints = constraints[:len(constraints)-1]
		truncatedMatches--
		content, hasStaleModels = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil, projectDir, stalenessMeta)
	}
	if len(filterByType(result.Matches, "constraint")) > len(constraints) {
		omittedCategories = append(omittedCategories, "constraint")
	}

	// Add truncation warning to content
	omittedCount := originalMatchCount - truncatedMatches
	if omittedCount > 0 {
		estimatedMaxTokens := EstimateTokens(maxChars)
		truncationNote := fmt.Sprintf("⚠️ **KB context truncated:** %d of %d matches omitted to stay within token budget (~%dk tokens).\n\n",
			omittedCount, originalMatchCount, estimatedMaxTokens/1000)
		content, hasStaleModels = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, &truncationNote, projectDir, stalenessMeta)
	}

	return &KBContextFormatResult{
		Content:           content,
		WasTruncated:      omittedCount > 0,
		OriginalMatches:   originalMatchCount,
		TruncatedMatches:  truncatedMatches,
		EstimatedTokens:   EstimateTokens(len(content)),
		OmittedCategories: omittedCategories,
		HasInjectedModels: hasInjectedModelContent(models),
		PrimaryModelPath:  primaryModelPath,
		HasStaleModels:    hasStaleModels,
		CrossRepoModelDir: crossRepoModelDir,
	}
}

// formatKBContextContent generates the formatted KB context markdown.
// If truncationNote is provided, it's inserted after the query line.
// Returns the formatted content and whether any models were stale.
func formatKBContextContent(query string, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions []KBContextMatch, truncationNote *string, projectDir string, stalenessMeta *StalenessEventMeta) (string, bool) {
	var sb strings.Builder
	hasStaleModels := false

	sb.WriteString("## PRIOR KNOWLEDGE (from kb context)\n\n")
	sb.WriteString(fmt.Sprintf("**Query:** %q\n\n", query))

	if truncationNote != nil {
		sb.WriteString(*truncationNote)
	}

	if len(constraints) > 0 {
		sb.WriteString("### Constraints (MUST respect)\n")
		for _, m := range constraints {
			sb.WriteString(fmt.Sprintf("- %s", m.Title))
			if m.Reason != "" {
				sb.WriteString(fmt.Sprintf("\n  - Reason: %s", m.Reason))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(decisions) > 0 {
		sb.WriteString("### Prior Decisions\n")
		for _, m := range decisions {
			sb.WriteString(fmt.Sprintf("- %s", m.Title))
			if m.Reason != "" {
				sb.WriteString(fmt.Sprintf("\n  - Reason: %s", m.Reason))
			}
			if m.Path != "" {
				sb.WriteString(fmt.Sprintf("\n  - See: %s", m.Path))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(models) > 0 {
		sb.WriteString("### Models (synthesized understanding)\n")
		for _, m := range models {
			modelContent, isStale := formatModelMatchForSpawn(m, projectDir, stalenessMeta)
			sb.WriteString(modelContent)
			if isStale {
				hasStaleModels = true
			}
		}
		sb.WriteString("\n")
	}

	if len(guides) > 0 {
		sb.WriteString("### Guides (procedural knowledge)\n")
		for _, m := range guides {
			sb.WriteString(fmt.Sprintf("- %s", m.Title))
			if m.Path != "" {
				sb.WriteString(fmt.Sprintf("\n  - See: %s", m.Path))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(investigations) > 0 {
		sb.WriteString("### Related Investigations\n")
		for _, m := range investigations {
			sb.WriteString(fmt.Sprintf("- %s", m.Title))
			if m.Path != "" {
				sb.WriteString(fmt.Sprintf("\n  - See: %s", m.Path))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(failedAttempts) > 0 {
		sb.WriteString("### Failed Attempts (DO NOT repeat)\n")
		for _, m := range failedAttempts {
			sb.WriteString(fmt.Sprintf("- %s", m.Title))
			if m.Reason != "" {
				sb.WriteString(fmt.Sprintf("\n  - Result: %s", m.Reason))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(openQuestions) > 0 {
		sb.WriteString("### Open Questions\n")
		for _, m := range openQuestions {
			sb.WriteString(fmt.Sprintf("- %s", m.Title))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.\n\n")

	return sb.String(), hasStaleModels
}

// formatMatchesForDisplay regenerates a display-friendly output from matches.
func formatMatchesForDisplay(matches []KBContextMatch, query string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Context for %q:\n\n", query))

	// Group by type
	byType := make(map[string][]KBContextMatch)
	for _, m := range matches {
		byType[m.Type] = append(byType[m.Type], m)
	}

	// Output in consistent order
	typeOrder := []string{"constraint", "decision", "model", "guide", "investigation", "failed-attempt", "open-question"}
	typeHeaders := map[string]string{
		"constraint":     "## CONSTRAINTS",
		"decision":       "## DECISIONS",
		"model":          "## MODELS",
		"guide":          "## GUIDES",
		"investigation":  "## INVESTIGATIONS",
		"failed-attempt": "## FAILED ATTEMPTS",
		"open-question":  "## OPEN QUESTIONS",
	}

	for _, t := range typeOrder {
		if ms, ok := byType[t]; ok && len(ms) > 0 {
			// Determine source annotation
			source := "(from kb)"
			if len(ms) > 0 && ms[0].Source == "kn" {
				source = "(from kn)"
			}
			sb.WriteString(fmt.Sprintf("%s %s\n\n", typeHeaders[t], source))
			for _, m := range ms {
				sb.WriteString(fmt.Sprintf("- %s\n", m.Title))
				if m.Reason != "" {
					sb.WriteString(fmt.Sprintf("  Reason: %s\n", m.Reason))
				}
				if m.Path != "" {
					sb.WriteString(fmt.Sprintf("  Path: %s\n", m.Path))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func filterByType(matches []KBContextMatch, matchType string) []KBContextMatch {
	var filtered []KBContextMatch
	for _, m := range matches {
		if m.Type == matchType {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
