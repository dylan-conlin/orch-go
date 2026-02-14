// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// OrchEcosystemRepos defines the allowlist of repos that are relevant for orchestration work.
// Used for tiered filtering: when global search is needed, results are post-filtered to this set.
var OrchEcosystemRepos = map[string]bool{
	"orch-go":        true,
	"orch-cli":       true,
	"kb-cli":         true,
	"orch-knowledge": true,
	"beads":          true,
	"kn":             true,
}

// MinMatchesForLocalSearch is the threshold below which we expand to global search.
// If local search returns fewer matches than this, we try global with ecosystem filter.
const MinMatchesForLocalSearch = 3

// MaxMatchesPerCategory limits results per category to prevent context flood.
const MaxMatchesPerCategory = 20

// MaxKBContextChars limits the total KB context size to prevent token bloat.
// Set to ~80k chars which is approximately 20k tokens (using 4 chars/token ratio).
// This leaves room for other spawn context elements (skills, CLAUDE.md, template).
const MaxKBContextChars = 80000

// CharsPerToken is a conservative estimate for token calculation.
// Claude typically uses ~4 chars per token for English text.
const CharsPerToken = 4

// maxModelSectionChars limits each injected model section.
// Large models are truncated per section to preserve token budget.
const maxModelSectionChars = 2500

// KBContextMatch represents a match from kb context.
type KBContextMatch struct {
	Type        string // "constraint", "decision", "investigation", "guide"
	Source      string // "kn" or "kb"
	Title       string // Title or description of the match
	Path        string // File path (for kb artifacts)
	Reason      string // Reason (for kn entries)
	FullContent string // Full content line for display
}

// KBContextResult holds the results of a kb context query.
type KBContextResult struct {
	Query      string
	HasMatches bool
	Matches    []KBContextMatch
	RawOutput  string
}

// KBContextFormatResult holds the formatted context and truncation information.
type KBContextFormatResult struct {
	Content           string   // Formatted KB context for SPAWN_CONTEXT.md
	WasTruncated      bool     // Whether context was truncated due to token limit
	OriginalMatches   int      // Original number of matches before truncation
	TruncatedMatches  int      // Number of matches after truncation
	EstimatedTokens   int      // Estimated token count of the formatted content
	OmittedCategories []string // Categories that were partially or fully omitted
	HasInjectedModels bool     // Whether model content (summary/invariants/failures) was injected
	PrimaryModelPath  string   // File path of the first model (when HasInjectedModels is true)
}

// ExtractKeywords extracts meaningful keywords from a task description for kb context query.
// Uses the same stop word filtering as generateSlug but returns more words for better search.
func ExtractKeywords(task string, maxWords int) string {
	// Stop words to exclude
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"for": true, "to": true, "in": true, "on": true, "at": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"this": true, "that": true, "with": true, "from": true, "of": true,
		"add": true, "implement": true, "create": true, "update": true, "fix": true,
		"new": true, "should": true, "can": true, "will": true, "need": true,
	}

	// Extract words (lowercase, alphanumeric only)
	matches := regexAlphanumeric.FindAllString(strings.ToLower(task), -1)

	var words []string
	for _, word := range matches {
		if !stopWords[word] && len(word) > 2 {
			words = append(words, word)
			if maxWords > 0 && len(words) >= maxWords {
				break
			}
		}
	}

	return strings.Join(words, " ")
}

// RunKBContextCheck runs 'kb context' with tiered search strategy:
// 1. First query current project (no --global) for targeted results
// 2. If sparse (<3 matches), expand to global search with orch ecosystem filter
// 3. Apply per-category limits to prevent context flood
// Returns nil if no matches found or if kb command fails.
func RunKBContextCheck(query string) (*KBContextResult, error) {
	// Step 1: Try current project first (no --global flag)
	result, err := runKBContextQuery(query, false)
	if err != nil {
		return nil, err
	}

	// Step 2: If local search is sparse, expand to global with ecosystem filter
	if result == nil || len(result.Matches) < MinMatchesForLocalSearch {
		globalResult, err := runKBContextQuery(query, true)
		if err != nil {
			return nil, err
		}

		if globalResult != nil && len(globalResult.Matches) > 0 {
			// Post-filter to orch ecosystem repos
			globalResult.Matches = filterToOrchEcosystem(globalResult.Matches)
			globalResult.HasMatches = len(globalResult.Matches) > 0

			// Merge with local results if any
			if result != nil && len(result.Matches) > 0 {
				result = mergeResults(result, globalResult)
			} else if globalResult.HasMatches {
				result = globalResult
			}
		}
	}

	// Step 3: Apply per-category limits
	if result != nil && len(result.Matches) > 0 {
		result.Matches = applyPerCategoryLimits(result.Matches, MaxMatchesPerCategory)
		result.HasMatches = len(result.Matches) > 0
		// Regenerate RawOutput to reflect filtered results
		result.RawOutput = formatMatchesForDisplay(result.Matches, result.Query)
	}

	if result == nil || !result.HasMatches {
		return nil, nil
	}

	return result, nil
}

// runKBContextQuery runs a single kb context query with optional --global flag.
// Uses a 5-second timeout to prevent infinite hangs from kb context --global
// scanning large directories like ~/Documents.
func runKBContextQuery(query string, global bool) (*KBContextResult, error) {
	// Create context with timeout to prevent hangs
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if global {
		cmd = exec.CommandContext(ctx, "kb", "context", "--global", query)
	} else {
		cmd = exec.CommandContext(ctx, "kb", "context", query)
	}

	output, err := cmd.Output()
	if err != nil {
		// If kb command fails (not found, no matches, timeout, etc.), return nil
		// This is not an error - just means no context available
		return nil, nil
	}

	outputStr := strings.TrimSpace(string(output))

	// Check for "No results found" or empty output
	if outputStr == "" || strings.Contains(outputStr, "No results found") {
		return nil, nil
	}

	result := &KBContextResult{
		Query:     query,
		RawOutput: outputStr,
	}

	// Parse the output to extract matches
	result.Matches = parseKBContextOutput(outputStr)
	result.HasMatches = len(result.Matches) > 0

	if !result.HasMatches {
		return nil, nil
	}

	return result, nil
}

// filterToOrchEcosystem filters matches to only include those from orch ecosystem repos.
// Matches without a project prefix (local results) are always included.
func filterToOrchEcosystem(matches []KBContextMatch) []KBContextMatch {
	var filtered []KBContextMatch
	for _, m := range matches {
		project := extractProjectFromMatch(m)
		// Include if: no project prefix (local), OR project is in ecosystem allowlist
		if project == "" || OrchEcosystemRepos[project] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// extractProjectFromMatch extracts the project name from a match's title or path.
// Returns empty string if no project prefix found.
func extractProjectFromMatch(m KBContextMatch) string {
	// Check for [project] prefix in title (e.g., "[orch-go] Title here")
	if strings.HasPrefix(m.Title, "[") {
		end := strings.Index(m.Title, "]")
		if end > 1 {
			return m.Title[1:end]
		}
	}
	return ""
}

// applyPerCategoryLimits limits the number of matches per category type.
func applyPerCategoryLimits(matches []KBContextMatch, limit int) []KBContextMatch {
	categoryCounts := make(map[string]int)
	var filtered []KBContextMatch

	for _, m := range matches {
		if categoryCounts[m.Type] < limit {
			filtered = append(filtered, m)
			categoryCounts[m.Type]++
		}
	}
	return filtered
}

// mergeResults combines two KBContextResults, deduplicating matches.
func mergeResults(local, global *KBContextResult) *KBContextResult {
	if local == nil {
		return global
	}
	if global == nil {
		return local
	}

	// Create a set of existing titles to avoid duplicates
	seen := make(map[string]bool)
	var merged []KBContextMatch

	// Add local matches first (higher priority)
	for _, m := range local.Matches {
		key := m.Type + ":" + m.Title
		if !seen[key] {
			seen[key] = true
			merged = append(merged, m)
		}
	}

	// Add global matches that aren't duplicates
	for _, m := range global.Matches {
		key := m.Type + ":" + m.Title
		if !seen[key] {
			seen[key] = true
			merged = append(merged, m)
		}
	}

	return &KBContextResult{
		Query:      local.Query,
		HasMatches: len(merged) > 0,
		Matches:    merged,
		RawOutput:  formatMatchesForDisplay(merged, local.Query),
	}
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

// parseKBContextOutput parses the output of 'kb context' command.
func parseKBContextOutput(output string) []KBContextMatch {
	var matches []KBContextMatch

	lines := strings.Split(output, "\n")
	var currentSection string
	var currentSource string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect section headers
		if strings.HasPrefix(line, "## CONSTRAINTS") {
			currentSection = "constraint"
			currentSource = extractSource(line)
			continue
		}
		if strings.HasPrefix(line, "## DECISIONS") {
			currentSection = "decision"
			currentSource = extractSource(line)
			continue
		}
		if strings.HasPrefix(line, "## MODELS") {
			currentSection = "model"
			currentSource = "kb"
			continue
		}
		if strings.HasPrefix(line, "## GUIDES") {
			currentSection = "guide"
			currentSource = "kb"
			continue
		}
		if strings.HasPrefix(line, "## FAILED ATTEMPTS") {
			currentSection = "failed-attempt"
			currentSource = extractSource(line)
			continue
		}
		if strings.HasPrefix(line, "## OPEN QUESTIONS") {
			currentSection = "open-question"
			currentSource = "kn"
			continue
		}
		if strings.HasPrefix(line, "## INVESTIGATIONS") {
			currentSection = "investigation"
			currentSource = "kb"
			continue
		}

		if strings.HasPrefix(line, "## DECISIONS") {
			currentSection = "decision"
			currentSource = extractSource(line)
			continue
		}
		if strings.HasPrefix(line, "## FAILED ATTEMPTS") {
			currentSection = "failed-attempt"
			currentSource = extractSource(line)
			continue
		}
		if strings.HasPrefix(line, "## OPEN QUESTIONS") {
			currentSection = "open-question"
			currentSource = "kn"
			continue
		}
		if strings.HasPrefix(line, "## INVESTIGATIONS") {
			currentSection = "investigation"
			currentSource = extractSource(line)
			continue
		}
		if strings.HasPrefix(line, "## MODELS") {
			currentSection = "model"
			currentSource = extractSource(line)
			continue
		}
		if strings.HasPrefix(line, "## GUIDES") {
			currentSection = "guide"
			currentSource = extractSource(line)
			continue
		}

		if strings.HasPrefix(line, "Context for") {
			continue // Skip the header line
		}

		// Parse entry lines (start with "- ")
		if strings.HasPrefix(line, "- ") {
			entry := strings.TrimPrefix(line, "- ")
			match := KBContextMatch{
				Type:        currentSection,
				Source:      currentSource,
				FullContent: entry,
			}

			// Extract title and path/reason
			if strings.Contains(entry, "Path:") {
				// kb artifact format: "Title\n  Path: /path/to/file"
				parts := strings.SplitN(entry, "Path:", 2)
				match.Title = strings.TrimSpace(parts[0])
				if len(parts) > 1 {
					match.Path = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(entry, "Reason:") {
				// kn entry format: "Description\n  Reason: explanation"
				parts := strings.SplitN(entry, "Reason:", 2)
				match.Title = strings.TrimSpace(parts[0])
				if len(parts) > 1 {
					match.Reason = strings.TrimSpace(parts[1])
				}
			} else {
				match.Title = entry
			}

			matches = append(matches, match)
		}

		// Handle multi-line entries (Path: or Reason: on next line)
		if strings.HasPrefix(line, "Path:") && len(matches) > 0 {
			matches[len(matches)-1].Path = strings.TrimSpace(strings.TrimPrefix(line, "Path:"))
		}
		if strings.HasPrefix(line, "Reason:") && len(matches) > 0 {
			matches[len(matches)-1].Reason = strings.TrimSpace(strings.TrimPrefix(line, "Reason:"))
		}
	}

	return matches
}

// extractSource extracts the source (kn or kb) from a section header.
func extractSource(line string) string {
	if strings.Contains(line, "(from kn)") {
		return "kn"
	}
	if strings.Contains(line, "(from kb)") {
		return "kb"
	}
	return "unknown"
}

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
	content := formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil)

	// Extract primary model path if models exist
	primaryModelPath := ""
	if len(models) > 0 && models[0].Path != "" {
		primaryModelPath = models[0].Path
	}

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
		content = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil)
	}
	if len(filterByType(result.Matches, "open-question")) > len(openQuestions) {
		omittedCategories = append(omittedCategories, "open-question")
	}

	// If still too large, remove failed attempts one at a time
	for len(content) > maxChars && len(failedAttempts) > 0 {
		failedAttempts = failedAttempts[:len(failedAttempts)-1]
		truncatedMatches--
		content = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil)
	}
	if len(filterByType(result.Matches, "failed-attempt")) > len(failedAttempts) {
		omittedCategories = append(omittedCategories, "failed-attempt")
	}

	// If still too large, remove investigations one at a time
	for len(content) > maxChars && len(investigations) > 0 {
		investigations = investigations[:len(investigations)-1]
		truncatedMatches--
		content = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil)
	}
	if len(filterByType(result.Matches, "investigation")) > len(investigations) {
		omittedCategories = append(omittedCategories, "investigation")
	}

	// If still too large, remove guides one at a time
	for len(content) > maxChars && len(guides) > 0 {
		guides = guides[:len(guides)-1]
		truncatedMatches--
		content = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil)
	}
	if len(filterByType(result.Matches, "guide")) > len(guides) {
		omittedCategories = append(omittedCategories, "guide")
	}

	// If still too large, remove models one at a time
	for len(content) > maxChars && len(models) > 0 {
		models = models[:len(models)-1]
		truncatedMatches--
		content = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil)
	}
	if len(filterByType(result.Matches, "model")) > len(models) {
		omittedCategories = append(omittedCategories, "model")
	}

	// If still too large, remove decisions one at a time
	for len(content) > maxChars && len(decisions) > 0 {
		decisions = decisions[:len(decisions)-1]
		truncatedMatches--
		content = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil)
	}
	if len(filterByType(result.Matches, "decision")) > len(decisions) {
		omittedCategories = append(omittedCategories, "decision")
	}

	// If STILL too large, remove constraints one at a time (last resort)
	for len(content) > maxChars && len(constraints) > 0 {
		constraints = constraints[:len(constraints)-1]
		truncatedMatches--
		content = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, nil)
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
		content = formatKBContextContent(result.Query, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions, &truncationNote)
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
	}
}

// formatKBContextContent generates the formatted KB context markdown.
// If truncationNote is provided, it's inserted after the query line.
func formatKBContextContent(query string, constraints, decisions, models, guides, investigations, failedAttempts, openQuestions []KBContextMatch, truncationNote *string) string {
	var sb strings.Builder
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
			sb.WriteString(formatModelMatchForSpawn(m))
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

type markdownHeading struct {
	line  int
	level int
	title string
}

type modelSpawnSections struct {
	summary            string
	criticalInvariants string
	whyThisFails       string
}

// hasInjectedModelContent checks whether any model matches have extractable content
// (summary, critical invariants, or why-this-fails sections). When true, spawn context
// should include probe guidance so agents produce lightweight probes instead of full investigations.
func hasInjectedModelContent(models []KBContextMatch) bool {
	for _, m := range models {
		if m.Path == "" {
			continue
		}
		sections, err := extractModelSectionsForSpawn(m.Path)
		if err != nil {
			continue
		}
		if sections.summary != "" || sections.criticalInvariants != "" || sections.whyThisFails != "" {
			return true
		}
	}
	return false
}

func formatModelMatchForSpawn(match KBContextMatch) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("- %s\n", match.Title))
	if match.Path != "" {
		sb.WriteString(fmt.Sprintf("  - See: %s\n", match.Path))
	}

	if match.Path == "" {
		return sb.String()
	}

	sections, err := extractModelSectionsForSpawn(match.Path)
	if err != nil {
		return sb.String()
	}

	hasInjectedContent := false
	if sections.summary != "" {
		hasInjectedContent = true
		sb.WriteString("  - Summary:\n")
		sb.WriteString(indentBlock(sections.summary, "    "))
	}
	if sections.criticalInvariants != "" {
		hasInjectedContent = true
		sb.WriteString("  - Critical Invariants:\n")
		sb.WriteString(indentBlock(sections.criticalInvariants, "    "))
	}
	if sections.whyThisFails != "" {
		hasInjectedContent = true
		sb.WriteString("  - Why This Fails:\n")
		sb.WriteString(indentBlock(sections.whyThisFails, "    "))
	}

	if hasInjectedContent {
		sb.WriteString("  - Your findings should confirm, contradict, or extend the claims above.\n")
	}

	// Inject recent probes from this model's probes/ directory
	if match.Path != "" {
		probes := ListRecentProbes(match.Path, MaxRecentProbes)
		probeContent := FormatProbesForSpawn(probes)
		if probeContent != "" {
			sb.WriteString(probeContent)
		}
	}

	return sb.String()
}

func extractModelSectionsForSpawn(path string) (*modelSpawnSections, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	headings := collectMarkdownHeadings(lines)

	sections := &modelSpawnSections{
		summary: truncateModelSection(extractSectionByHeading(lines, headings, func(title string) bool {
			return strings.HasPrefix(title, "summary") || title == "what this is" || strings.HasPrefix(title, "executive summary")
		})),
		criticalInvariants: truncateModelSection(extractSectionByHeading(lines, headings, func(title string) bool { return strings.HasPrefix(title, "critical invariants") })),
		whyThisFails:       truncateModelSection(extractSectionByHeading(lines, headings, func(title string) bool { return strings.HasPrefix(title, "why this fails") })),
	}

	return sections, nil
}

func collectMarkdownHeadings(lines []string) []markdownHeading {
	headings := make([]markdownHeading, 0)
	inCodeFence := false

	for i, raw := range lines {
		line := strings.TrimSpace(raw)

		if strings.HasPrefix(line, "```") {
			inCodeFence = !inCodeFence
			continue
		}
		if inCodeFence {
			continue
		}

		level, title, ok := parseMarkdownHeading(line)
		if !ok {
			continue
		}

		headings = append(headings, markdownHeading{
			line:  i,
			level: level,
			title: normalizeHeading(title),
		})
	}

	return headings
}

func parseMarkdownHeading(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}

	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}

	if level < 2 || level > 6 {
		return 0, "", false
	}

	if len(line) == level || line[level] != ' ' {
		return 0, "", false
	}

	title := strings.TrimSpace(line[level:])
	if title == "" {
		return 0, "", false
	}

	return level, title, true
}

func normalizeHeading(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

func extractSectionByHeading(lines []string, headings []markdownHeading, matcher func(string) bool) string {
	for idx, heading := range headings {
		if !matcher(heading.title) {
			continue
		}

		startLine := heading.line + 1
		endLine := len(lines)

		for next := idx + 1; next < len(headings); next++ {
			if headings[next].level <= heading.level {
				endLine = headings[next].line
				break
			}
		}

		if startLine >= len(lines) || startLine >= endLine {
			return ""
		}

		content := strings.TrimSpace(strings.Join(lines[startLine:endLine], "\n"))
		return content
	}

	return ""
}

func truncateModelSection(content string) string {
	content = strings.TrimSpace(content)
	if content == "" || len(content) <= maxModelSectionChars {
		return content
	}

	truncated := strings.TrimSpace(content[:maxModelSectionChars])
	if lastBreak := strings.LastIndexAny(truncated, "\n "); lastBreak > maxModelSectionChars/2 {
		truncated = strings.TrimSpace(truncated[:lastBreak])
	}

	return truncated + "\n... [truncated]"
}

func indentBlock(content, indent string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = indent
			continue
		}
		lines[i] = indent + line
	}

	return strings.Join(lines, "\n") + "\n"
}
