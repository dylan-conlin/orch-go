package spawn

import (
	"regexp"
	"strings"
)

// regexSkillPrefix matches "SkillName:" prefix patterns at the start of task titles.
// Many task titles follow the form "Architect: Redesign pricing KPIs" where the skill
// name is a prefix. Stripping this prevents skill names from polluting kb context queries.
var regexSkillPrefix = regexp.MustCompile(`(?i)^(architect|investigation|investigate|debug|debugging|research|audit|feature[- ]?impl|systematic[- ]?debugging|codebase[- ]?audit|design[- ]?session|reliability[- ]?testing|issue[- ]?creation)\s*[:]\s*`)

// ExtractKeywords extracts meaningful keywords from a task description for kb context query.
// Uses the same stop word filtering as generateSlug but returns more words for better search.
// Strips skill name prefixes (e.g., "Architect: Redesign pricing KPIs" → "Redesign pricing KPIs")
// and filters out skill-related terms that would match infrastructure knowledge instead of
// domain-specific topics.
func ExtractKeywords(task string, maxWords int) string {
	// Strip "Skill:" prefix pattern — these match infrastructure knowledge, not domain topics
	cleaned := regexSkillPrefix.ReplaceAllString(task, "")

	// Stop words to exclude — includes common articles, verbs, AND skill/infrastructure
	// terms that match orch-go knowledge entries instead of task-specific domain topics
	stopWords := map[string]bool{
		// Articles and conjunctions
		"the": true, "a": true, "an": true, "and": true, "or": true,
		// Prepositions
		"for": true, "to": true, "in": true, "on": true, "at": true,
		// Be verbs
		"is": true, "are": true, "was": true, "were": true, "be": true,
		// Demonstratives and pronouns
		"this": true, "that": true, "with": true, "from": true, "of": true,
		// Common action verbs (already present)
		"add": true, "implement": true, "create": true, "update": true, "fix": true,
		"new": true, "should": true, "can": true, "will": true, "need": true,
		// Skill names — these match kb entries about the skill itself,
		// not about the domain the task targets
		"architect": true, "investigation": true, "investigate": true,
		"debug": true, "debugging": true, "research": true, "audit": true,
		"feature": true, "impl": true, "systematic": true, "quick": true,
		// Common action verbs used as task prefixes that match infrastructure decisions
		"redesign": true, "refactor": true, "optimize": true, "analyze": true,
	}

	// Extract words (lowercase, alphanumeric only)
	matches := regexAlphanumeric.FindAllString(strings.ToLower(cleaned), -1)

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

// ExtractKeywordsWithContext extracts keywords from both task title AND orientation frame.
// The title provides the primary keywords; the frame provides additional domain-specific terms
// that disambiguate cross-domain spawns (e.g., "pricing KPI" from a frame when the title only says
// "fix kb context query"). Keywords are deduplicated and capped at maxWords.
func ExtractKeywordsWithContext(task, orientationFrame string, maxWords int) string {
	if orientationFrame == "" {
		return ExtractKeywords(task, maxWords)
	}

	// Extract keywords from title first (these get priority)
	titleKeywords := ExtractKeywords(task, maxWords)

	// Extract more keywords from the orientation frame
	// Use a larger pool to find domain-specific terms
	frameKeywords := ExtractKeywords(orientationFrame, maxWords*2)

	if titleKeywords == "" && frameKeywords == "" {
		return ""
	}
	if titleKeywords == "" {
		return ExtractKeywords(orientationFrame, maxWords)
	}
	if frameKeywords == "" {
		return titleKeywords
	}

	// Combine: title keywords first, then frame keywords for additional domain terms
	seen := make(map[string]bool)
	var combined []string

	for _, w := range strings.Fields(titleKeywords) {
		if !seen[w] {
			seen[w] = true
			combined = append(combined, w)
		}
	}
	for _, w := range strings.Fields(frameKeywords) {
		if !seen[w] {
			seen[w] = true
			combined = append(combined, w)
		}
	}

	if len(combined) > maxWords {
		combined = combined[:maxWords]
	}

	return strings.Join(combined, " ")
}

// regexScopedFilePath matches file paths with directory separators and extensions.
// Matches: pkg/spawn/context.go, cmd/orch/main.go, src/components/Dashboard.tsx
// Does not match: URLs (https://...), plain words, package names without extensions.
var regexScopedFilePath = regexp.MustCompile(`(?:^|[\s"'` + "`" + `(])(?:\./)?[a-zA-Z_][\w.-]*/[\w./-]+\.\w{1,5}(?::\d+)?(?:[\s"'` + "`" + `),]|$)`)

// TaskIsScoped detects if a task description targets specific files or code locations.
// Returns true when the task contains file paths (e.g., pkg/spawn/context.go).
// Scoped tasks get reduced kb context to save tokens — models, guides, investigations,
// and open questions are stripped since the agent is working on specific code.
func TaskIsScoped(task string) bool {
	if task == "" {
		return false
	}
	return regexScopedFilePath.MatchString(task)
}

// FilterForScopedTask removes heavyweight kb context categories that aren't needed
// for file-targeted tasks. Keeps constraints (always relevant), decisions (prevent
// re-deciding), and failed attempts (prevent repeating mistakes). Drops models
// (summaries/probes/invariants), guides, investigations, and open questions.
func FilterForScopedTask(matches []KBContextMatch) []KBContextMatch {
	if len(matches) == 0 {
		return nil
	}
	// Categories to keep for scoped tasks
	keepTypes := map[string]bool{
		"constraint":     true,
		"decision":       true,
		"failed-attempt": true,
	}
	var filtered []KBContextMatch
	for _, m := range matches {
		if keepTypes[m.Type] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
