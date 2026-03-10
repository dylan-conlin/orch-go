// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

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
	}

	// Step 4: Normalize ~/.kb/ symlink paths to .kb/global/ project-relative paths
	if result != nil && len(result.Matches) > 0 {
		result.Matches = normalizeGlobalKBPaths(result.Matches, projectDir)
		// Regenerate RawOutput to reflect filtered and normalized results
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
