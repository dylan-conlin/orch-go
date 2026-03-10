// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/group"
)

// OrchEcosystemRepos defines the allowlist of repos that are relevant for orchestration work.
// Used as fallback when groups.yaml doesn't exist (~/.kb/ or ~/.orch/).
// When groups.yaml exists, project group membership replaces this hardcode.
var OrchEcosystemRepos = map[string]bool{
	"orch-go":        true,
	"orch-cli":       true,
	"kb-cli":         true,
	"beads": true,
	"kn":             true,
}

// MinMatchesForLocalSearch is the threshold below which we expand to global search.
// If local search returns fewer matches than this, we try global with ecosystem filter.
// Set to 5 (raised from 3) because projects with rich knowledge bases (e.g., orch-go
// with 280+ investigations) trivially hit a low threshold with single-keyword matches
// on generic terms like "architect", preventing cross-project search for domain-specific content.
const MinMatchesForLocalSearch = 5

// MaxMatchesPerCategory limits results per category to prevent context flood.
const MaxMatchesPerCategory = 20

// MaxKBContextChars limits the total KB context size to prevent token bloat.
// Set to ~80k chars which is approximately 20k tokens (using 4 chars/token ratio).
// This leaves room for other spawn context elements (skills, CLAUDE.md, template).
const MaxKBContextChars = 80000

// ScopedMaxKBContextChars is the reduced character budget for scoped tasks.
// Scoped tasks target specific files/locations and don't need model summaries,
// guides, or investigations. 15k chars ≈ 3,750 tokens — enough for constraints + decisions.
const ScopedMaxKBContextChars = 15000

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
	HasStaleModels    bool     // Whether any served models have stale file references
	CrossRepoModelDir string   // When non-empty, the primary model lives in a different repo than ProjectDir
}

// StalenessResult holds the result of checking a model's staleness.
type StalenessResult struct {
	IsStale      bool     // Whether the model has stale references
	ChangedFiles []string // Files that changed since Last Updated
	DeletedFiles []string // Files that no longer exist
	LastUpdated  string   // The model's Last Updated date
}

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

// RunKBContextCheck runs 'kb context' with tiered search strategy using the current working directory.
// For cross-project spawns (--workdir), use RunKBContextCheckForDir instead.
func RunKBContextCheck(query string) (*KBContextResult, error) {
	return RunKBContextCheckForDir(query, "")
}

// RunKBContextCheckForDir runs 'kb context' with tiered search strategy:
// 1. First query current project (no --global) for targeted results
// 2. If sparse (<3 matches), expand to global search with group-aware filter
// 3. Apply per-category limits to prevent context flood
// projectDir controls which project's groups are used for global search filtering.
// When empty, falls back to os.Getwd().
// Returns nil if no matches found or if kb command fails.
func RunKBContextCheckForDir(query string, projectDir string) (*KBContextResult, error) {
	// Step 1: Try current project first (no --global flag)
	result, err := runKBContextQuery(query, false, projectDir)
	if err != nil {
		return nil, err
	}

	// Step 2: If local search is sparse, expand to global with group-aware filter
	if result == nil || len(result.Matches) < MinMatchesForLocalSearch {
		globalResult, err := runKBContextQuery(query, true, projectDir)
		if err != nil {
			return nil, err
		}

		if globalResult != nil && len(globalResult.Matches) > 0 {
			// Post-filter using project group (falls back to orch ecosystem)
			allowlist := resolveProjectAllowlistForDir(projectDir)
			if allowlist != nil {
				globalResult.Matches = filterToProjectGroup(globalResult.Matches, allowlist)
			}
			// allowlist == nil means ungrouped project: include all global matches
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
// projectDir sets the working directory for the kb command. When non-empty,
// local (non-global) queries search the target project's .kb/ instead of process CWD.
func runKBContextQuery(query string, global bool, projectDir string) (*KBContextResult, error) {
	// Create context with timeout to prevent hangs
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if global {
		cmd = exec.CommandContext(ctx, "kb", "context", "--global", query)
	} else {
		cmd = exec.CommandContext(ctx, "kb", "context", query)
	}

	// Set working directory to target project so local search hits correct .kb/
	if projectDir != "" {
		cmd.Dir = projectDir
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
// Deprecated: Use filterToProjectGroup for group-aware filtering. Kept as fallback.
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

// filterToProjectGroup filters matches using group-aware allowlist.
// Matches without a project prefix (local results) are always included.
func filterToProjectGroup(matches []KBContextMatch, allowlist map[string]bool) []KBContextMatch {
	var filtered []KBContextMatch
	for _, m := range matches {
		project := extractProjectFromMatch(m)
		if project == "" || allowlist[project] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// resolveProjectAllowlist builds an allowlist using the current working directory.
// For cross-project spawns, use resolveProjectAllowlistForDir instead.
func resolveProjectAllowlist() map[string]bool {
	return resolveProjectAllowlistForDir("")
}

// resolveProjectAllowlistForDir builds an allowlist of project names for global search filtering.
// projectDir controls which project's groups are used. When empty, falls back to os.Getwd().
// Tries group-based resolution from groups.yaml first (~/.kb/ primary, ~/.orch/ fallback).
// Falls back to OrchEcosystemRepos if groups.yaml doesn't exist or the current project is ungrouped.
func resolveProjectAllowlistForDir(projectDir string) map[string]bool {
	cfg, err := group.Load()
	if err != nil {
		// groups.yaml doesn't exist or can't be parsed — use hardcoded fallback
		return OrchEcosystemRepos
	}

	// Detect project name from directory
	projectName := detectProjectNameFromDir(projectDir)
	if projectName == "" {
		return OrchEcosystemRepos
	}

	// Get kb projects list for parent inference
	kbProjects := loadKBProjectsMap()
	if kbProjects == nil {
		return OrchEcosystemRepos
	}

	// Resolve groups for current project
	groups := cfg.GroupsForProject(projectName, kbProjects)
	if len(groups) == 0 {
		// Project is ungrouped — no group-based filtering
		// Return nil to signal "don't filter" (include all global matches)
		return nil
	}

	// Build allowlist from all projects in matching groups
	allowlist := make(map[string]bool)
	for _, g := range groups {
		members := cfg.ResolveGroupMembers(g.Name, kbProjects)
		for _, m := range members {
			allowlist[m] = true
		}
	}

	return allowlist
}

// detectCurrentProjectName returns the project name from the current working directory.
// Deprecated: Use detectProjectNameFromDir for explicit directory control.
func detectCurrentProjectName() string {
	return detectProjectNameFromDir("")
}

// detectProjectNameFromDir returns the project name from the given directory.
// Uses .beads/config.yaml issue-prefix if available, otherwise falls back to directory basename.
// When dir is empty, falls back to os.Getwd().
func detectProjectNameFromDir(dir string) string {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return ""
		}
		dir = cwd
	}

	// Try .beads/config.yaml for issue prefix (more reliable)
	configPath := filepath.Join(dir, ".beads", "config.yaml")
	if data, err := os.ReadFile(configPath); err == nil {
		// Simple YAML parsing — look for issue-prefix field
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "issue-prefix:") {
				prefix := strings.TrimSpace(strings.TrimPrefix(line, "issue-prefix:"))
				prefix = strings.Trim(prefix, `"'`)
				if prefix != "" {
					return prefix
				}
			}
		}
	}

	// Fall back to directory basename
	return filepath.Base(dir)
}

// kbProjectEntry matches the JSON format from `kb projects list --json`.
type kbProjectEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// loadKBProjectsMap runs `kb projects list --json` and returns name->path map.
func loadKBProjectsMap() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var projects []kbProjectEntry
	if err := json.Unmarshal(output, &projects); err != nil {
		return nil
	}

	result := make(map[string]string, len(projects))
	for _, p := range projects {
		result[p.Name] = p.Path
	}
	return result
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

// DetectCrossRepoModel checks if the primary model path is in a different git repo
// than the project directory. Returns the model's repo root directory if cross-repo,
// or empty string if same-repo or if detection fails.
//
// This catches the case where `--workdir` points to repo A but the model being probed
// lives in repo B. Without this detection, probe files get created in the wrong repo.
func DetectCrossRepoModel(primaryModelPath, projectDir string) string {
	if primaryModelPath == "" || projectDir == "" {
		return ""
	}

	// Normalize paths
	absModelPath, err := filepath.Abs(primaryModelPath)
	if err != nil {
		return ""
	}
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return ""
	}

	// Check if model path is under project directory
	if strings.HasPrefix(absModelPath, absProjectDir+string(filepath.Separator)) {
		return "" // Same repo, no cross-repo issue
	}

	// Model is outside project dir - find the model's repo root
	// Walk up from the model path to find the nearest .git directory
	modelDir := filepath.Dir(absModelPath)
	for modelDir != "/" && modelDir != "." {
		gitDir := filepath.Join(modelDir, ".git")
		if info, err := os.Stat(gitDir); err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return modelDir
		}
		modelDir = filepath.Dir(modelDir)
	}

	// Couldn't find git root, but model is still outside project dir
	// Return the directory containing .kb/ as a reasonable guess
	modelDir = filepath.Dir(absModelPath)
	for modelDir != "/" && modelDir != "." {
		kbDir := filepath.Join(modelDir, ".kb")
		if info, err := os.Stat(kbDir); err == nil && info.IsDir() {
			return modelDir
		}
		modelDir = filepath.Dir(modelDir)
	}

	return "" // Can't determine model repo
}

func formatModelMatchForSpawn(match KBContextMatch, projectDir string, stalenessMeta *StalenessEventMeta) (string, bool) {
	var sb strings.Builder
	isStale := false

	sb.WriteString(fmt.Sprintf("- %s\n", match.Title))
	if match.Path != "" {
		sb.WriteString(fmt.Sprintf("  - See: %s\n", match.Path))
	}

	if match.Path == "" {
		return sb.String(), isStale
	}

	// Check for staleness
	modelContent, err := os.ReadFile(match.Path)
	if err == nil {
		stalenessResult, err := checkModelStaleness(string(modelContent), projectDir)
		if err == nil && stalenessResult.IsStale {
			isStale = true
			if err := RecordModelStalenessEvent(match.Path, stalenessResult, stalenessMeta); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record staleness event: %v\n", err)
			}
			// Prepend staleness warning
			sb.WriteString("  - **STALENESS WARNING:**\n")
			sb.WriteString(fmt.Sprintf("    This model was last updated %s.\n", stalenessResult.LastUpdated))
			if len(stalenessResult.ChangedFiles) > 0 {
				sb.WriteString(fmt.Sprintf("    Changed files: %s.\n", strings.Join(stalenessResult.ChangedFiles, ", ")))
			}
			if len(stalenessResult.DeletedFiles) > 0 {
				sb.WriteString(fmt.Sprintf("    Deleted files: %s.\n", strings.Join(stalenessResult.DeletedFiles, ", ")))
			}
			sb.WriteString("    Verify model claims about these files against current code.\n")
		}
	}

	sections, err := extractModelSectionsForSpawn(match.Path)
	if err != nil {
		return sb.String(), isStale
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

	return sb.String(), isStale
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

// extractCodeRefs parses file paths from Primary Evidence sections in model content.
// Looks for backtick-quoted file paths (with optional :line or :function suffixes).
// Returns relative file paths without line numbers or function names.
func extractCodeRefs(content string) []string {
	var refs []string
	seen := make(map[string]bool)

	// Look for Primary Evidence section
	lines := strings.Split(content, "\n")
	inPrimaryEvidence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of Primary Evidence section
		if strings.Contains(trimmed, "Primary Evidence") {
			inPrimaryEvidence = true
			continue
		}

		// End of section (next ## heading, or explicit code_refs close marker)
		if inPrimaryEvidence {
			if strings.HasPrefix(trimmed, "##") || strings.HasPrefix(trimmed, "# ") {
				break
			}
			if strings.Contains(trimmed, "<!-- /code_refs") {
				break
			}
		}

		// Extract file paths from backticks
		if inPrimaryEvidence && strings.Contains(trimmed, "`") {
			// Find all backtick-quoted content
			parts := strings.Split(trimmed, "`")
			for i := 1; i < len(parts); i += 2 {
				path := strings.TrimSpace(parts[i])

				// Check if this looks like a file path (has .go, .md, etc.)
				if !strings.Contains(path, ".") {
					continue
				}

				// Remove :line number or :function() suffix
				if idx := strings.Index(path, ":"); idx > 0 {
					path = path[:idx]
				}

				// Skip if already seen
				if seen[path] {
					continue
				}

				// Only include if it looks like a file path (contains / or has valid extension)
				if strings.Contains(path, "/") || hasValidFileExtension(path) {
					refs = append(refs, path)
					seen[path] = true
				}
			}
		}
	}

	return refs
}

// hasValidFileExtension checks if a string ends with a common file extension.
func hasValidFileExtension(path string) bool {
	validExts := []string{".go", ".md", ".yaml", ".yml", ".json", ".toml", ".sh", ".js", ".ts", ".tsx", ".jsx"}
	for _, ext := range validExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// extractLastUpdated parses the "Last Updated:" date from model content.
// Returns the date string (YYYY-MM-DD format) or empty string if not found.
func extractLastUpdated(content string) string {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for **Last Updated:** or **last updated:**
		if strings.HasPrefix(trimmed, "**Last Updated:") || strings.HasPrefix(trimmed, "**last updated:") {
			// Extract the date after the colon
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				date := strings.TrimSpace(parts[1])
				// Remove leading and trailing ** (markdown bold markers)
				date = strings.TrimPrefix(date, "**")
				date = strings.TrimSuffix(date, "**")
				return strings.TrimSpace(date)
			}
		}
	}

	return ""
}

// checkModelStaleness checks if a model's referenced files have changed since its Last Updated date.
// Returns a StalenessResult with changed/deleted files, or an empty result if the model can't be checked.
func checkModelStaleness(modelContent string, projectDir string) (*StalenessResult, error) {
	result := &StalenessResult{
		IsStale:      false,
		ChangedFiles: []string{},
		DeletedFiles: []string{},
	}

	// Extract Last Updated date
	lastUpdated := extractLastUpdated(modelContent)
	if lastUpdated == "" {
		// No Last Updated field - can't check staleness
		return result, nil
	}
	result.LastUpdated = lastUpdated

	// Parse date and add 1 day to avoid same-day boundary false positives.
	// git log --since=YYYY-MM-DD includes all commits from midnight of that day,
	// but the model was updated at some point during that day, so earlier same-day
	// commits would appear as "changed" even though the model already accounts for them.
	sinceDate := lastUpdated
	if t, err := time.Parse("2006-01-02", lastUpdated); err == nil {
		sinceDate = t.AddDate(0, 0, 1).Format("2006-01-02")
	}

	// Extract code references
	codeRefs := extractCodeRefs(modelContent)
	if len(codeRefs) == 0 {
		// No code references - can't check staleness
		return result, nil
	}

	// Check each referenced file
	for _, ref := range codeRefs {
		filePath := ref
		if strings.HasPrefix(filePath, "~/") {
			// Expand tilde to home directory
			home, err := os.UserHomeDir()
			if err == nil {
				filePath = filepath.Join(home, filePath[2:])
			}
		} else if !strings.HasPrefix(filePath, "/") {
			// Make relative paths absolute
			filePath = fmt.Sprintf("%s/%s", projectDir, ref)
		}

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			result.DeletedFiles = append(result.DeletedFiles, ref)
			result.IsStale = true
			continue
		}

		// Check if file changed since Last Updated using git log
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "git", "log", "--since="+sinceDate, "--oneline", "--", ref)
		cmd.Dir = projectDir
		output, err := cmd.Output()
		if err != nil {
			// Git command failed - might not be in a git repo, or file not tracked
			// This is not an error condition - just skip this file
			continue
		}

		// If output is non-empty, file has commits since Last Updated
		if len(strings.TrimSpace(string(output))) > 0 {
			result.ChangedFiles = append(result.ChangedFiles, ref)
			result.IsStale = true
		}
	}

	return result, nil
}
