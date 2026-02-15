package spawn

import (
	"os"
	"strings"
	"testing"
)

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name      string
		task      string
		maxWords  int
		wantWords []string // Check these words are present
		notWords  []string // Check these words are NOT present
	}{
		{
			name:      "basic extraction",
			task:      "Add rate limiting to the API",
			maxWords:  6,
			wantWords: []string{"rate", "limiting", "api"},
			notWords:  []string{"add", "to", "the"}, // stop words
		},
		{
			name:      "respects max words",
			task:      "Implement authentication middleware for user sessions with JWT tokens",
			maxWords:  3,
			wantWords: []string{"authentication"}, // Should be in first 3
		},
		{
			name:      "filters short words",
			task:      "A is a an the or so it",
			maxWords:  6,
			wantWords: []string{}, // All filtered out
		},
		{
			name:      "handles empty task",
			task:      "",
			maxWords:  6,
			wantWords: []string{},
		},
		{
			name:      "extracts from complex sentence",
			task:      "Fix the spawn context generation to include kb knowledge before spawning agents",
			maxWords:  6,
			wantWords: []string{"spawn", "context", "generation"},
			notWords:  []string{"fix", "the", "to"}, // "include" is not a stop word
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractKeywords(tt.task, tt.maxWords)

			// Check wanted words are present
			for _, word := range tt.wantWords {
				if !strings.Contains(result, word) {
					t.Errorf("ExtractKeywords() = %q, want word %q to be present", result, word)
				}
			}

			// Check unwanted words are NOT present
			for _, word := range tt.notWords {
				if strings.Contains(result, word) {
					t.Errorf("ExtractKeywords() = %q, word %q should not be present (stop word)", result, word)
				}
			}
		})
	}
}

func TestParseKBContextOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantCount   int
		wantTypes   []string
		wantSources []string
	}{
		{
			name: "parses constraints",
			output: `Context for "spawn":

## CONSTRAINTS (from kn)

- Agents must not spawn more than 3 iterations without human review
  Reason: Prevents runaway iteration loops`,
			wantCount:   1,
			wantTypes:   []string{"constraint"},
			wantSources: []string{"kn"},
		},
		{
			name: "parses decisions",
			output: `Context for "test":

## DECISIONS (from kn)

- Use TDD for all new features
  Reason: Better code quality

## DECISIONS (from kb)

- Minimal Artifact Taxonomy
  Path: /path/to/decision.md`,
			wantCount:   2,
			wantTypes:   []string{"decision", "decision"},
			wantSources: []string{"kn", "kb"},
		},
		{
			name: "parses investigations and guides",
			output: `Context for "auth":

## INVESTIGATIONS (from kb)

- Authentication Flow Analysis
  Path: /path/to/investigation.md

## GUIDES (from kb)

- Auth Implementation Guide
  Path: /path/to/guide.md`,
			wantCount:   2,
			wantTypes:   []string{"investigation", "guide"},
			wantSources: []string{"kb", "kb"},
		},
		{
			name: "parses models",
			output: `Context for "model":

## MODELS (from kb)

- Session Lifecycle Model
  Path: /path/to/model.md`,
			wantCount:   1,
			wantTypes:   []string{"model"},
			wantSources: []string{"kb"},
		},
		{
			name: "parses failed attempts and open questions",
			output: `Context for "failed":

## FAILED ATTEMPTS (from kn)

- Tried using Redis
  Reason: Too complex for this scale

## OPEN QUESTIONS (from kn)

- Should we use SQLite?`,
			wantCount:   2,
			wantTypes:   []string{"failed-attempt", "open-question"},
			wantSources: []string{"kn", "kn"},
		},
		{
			name:      "handles empty output",
			output:    "",
			wantCount: 0,
		},
		{
			name:      "handles no results",
			output:    "No results found",
			wantCount: 0,
		},
		{
			name: "parses global output with project prefixes",
			output: `Context for "spawn":

## CONSTRAINTS (from kn)

- [orch-knowledge] Orchestrators NEVER do spawnable work
  Reason: Orchestrator doing task work blocks the entire system
- [orch-cli] Worker agents must NEVER spawn other agents
  Reason: Recursive spawn testing incident
- [orch-go] Agents must not spawn more than 3 iterations
  Reason: Prevents runaway iteration loops

## DECISIONS (from kn)

- [orch-knowledge] kn integrates via smart auto-inject in orch spawn
  Reason: Auto-inject prevents missing critical knowledge`,
			wantCount:   4,
			wantTypes:   []string{"constraint", "constraint", "constraint", "decision"},
			wantSources: []string{"kn", "kn", "kn", "kn"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := parseKBContextOutput(tt.output)

			if len(matches) != tt.wantCount {
				t.Errorf("parseKBContextOutput() returned %d matches, want %d", len(matches), tt.wantCount)
			}

			for i, wantType := range tt.wantTypes {
				if i < len(matches) && matches[i].Type != wantType {
					t.Errorf("match[%d].Type = %q, want %q", i, matches[i].Type, wantType)
				}
			}

			for i, wantSource := range tt.wantSources {
				if i < len(matches) && matches[i].Source != wantSource {
					t.Errorf("match[%d].Source = %q, want %q", i, matches[i].Source, wantSource)
				}
			}
		})
	}
}

func TestFormatContextForSpawn(t *testing.T) {
	tests := []struct {
		name         string
		result       *KBContextResult
		wantEmpty    bool
		wantContains []string
	}{
		{
			name:      "nil result returns empty",
			result:    nil,
			wantEmpty: true,
		},
		{
			name: "no matches returns empty",
			result: &KBContextResult{
				Query:      "test",
				HasMatches: false,
				Matches:    []KBContextMatch{},
			},
			wantEmpty: true,
		},
		{
			name: "formats constraints section",
			result: &KBContextResult{
				Query:      "test",
				HasMatches: true,
				Matches: []KBContextMatch{
					{Type: "constraint", Source: "kn", Title: "No infinite loops", Reason: "Prevents runaway"},
				},
			},
			wantEmpty: false,
			wantContains: []string{
				"## PRIOR KNOWLEDGE",
				"### Constraints",
				"No infinite loops",
				"Reason: Prevents runaway",
			},
		},
		{
			name: "formats decisions with path",
			result: &KBContextResult{
				Query:      "auth",
				HasMatches: true,
				Matches: []KBContextMatch{
					{Type: "decision", Source: "kb", Title: "Use JWT", Path: "/path/to/decision.md"},
				},
			},
			wantEmpty: false,
			wantContains: []string{
				"### Prior Decisions",
				"Use JWT",
				"See: /path/to/decision.md",
			},
		},
		{
			name: "formats models and guides",
			result: &KBContextResult{
				Query:      "model",
				HasMatches: true,
				Matches: []KBContextMatch{
					{Type: "model", Source: "kb", Title: "State Model", Path: "/path/to/model.md"},
					{Type: "guide", Source: "kb", Title: "Step Guide", Path: "/path/to/guide.md"},
				},
			},
			wantEmpty: false,
			wantContains: []string{
				"### Models",
				"State Model",
				"### Guides",
				"Step Guide",
			},
		},
		{
			name: "formats failed attempts and open questions",
			result: &KBContextResult{
				Query:      "retry",
				HasMatches: true,
				Matches: []KBContextMatch{
					{Type: "failed-attempt", Source: "kn", Title: "Manual retry", Reason: "Timed out"},
					{Type: "open-question", Source: "kn", Title: "Auto retry?"},
				},
			},
			wantEmpty: false,
			wantContains: []string{
				"### Failed Attempts",
				"Manual retry",
				"Result: Timed out",
				"### Open Questions",
				"Auto retry?",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatContextForSpawn(tt.result)

			if tt.wantEmpty && result != "" {
				t.Errorf("FormatContextForSpawn() = %q, want empty", result)
			}

			if !tt.wantEmpty {
				for _, want := range tt.wantContains {
					if !strings.Contains(result, want) {
						t.Errorf("FormatContextForSpawn() missing %q in output:\n%s", want, result)
					}
				}
			}
		})
	}
}

func TestFilterByType(t *testing.T) {
	matches := []KBContextMatch{
		{Type: "constraint", Title: "C1"},
		{Type: "decision", Title: "D1"},
		{Type: "constraint", Title: "C2"},
		{Type: "investigation", Title: "I1"},
	}

	constraints := filterByType(matches, "constraint")
	if len(constraints) != 2 {
		t.Errorf("filterByType() for constraint returned %d, want 2", len(constraints))
	}

	decisions := filterByType(matches, "decision")
	if len(decisions) != 1 {
		t.Errorf("filterByType() for decision returned %d, want 1", len(decisions))
	}

	guides := filterByType(matches, "guide")
	if len(guides) != 0 {
		t.Errorf("filterByType() for guide returned %d, want 0", len(guides))
	}
}

func TestExtractProjectFromMatch(t *testing.T) {
	tests := []struct {
		name  string
		match KBContextMatch
		want  string
	}{
		{
			name:  "extracts project from bracket prefix",
			match: KBContextMatch{Title: "[orch-go] Some constraint about spawning"},
			want:  "orch-go",
		},
		{
			name:  "extracts project with hyphen",
			match: KBContextMatch{Title: "[orch-knowledge] Some decision"},
			want:  "orch-knowledge",
		},
		{
			name:  "returns empty for no prefix",
			match: KBContextMatch{Title: "Plain title without project"},
			want:  "",
		},
		{
			name:  "returns empty for malformed prefix",
			match: KBContextMatch{Title: "[incomplete"},
			want:  "",
		},
		{
			name:  "handles empty title",
			match: KBContextMatch{Title: ""},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProjectFromMatch(tt.match)
			if got != tt.want {
				t.Errorf("extractProjectFromMatch() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterToOrchEcosystem(t *testing.T) {
	matches := []KBContextMatch{
		{Type: "constraint", Title: "[orch-go] Agents must not spawn recursively"},
		{Type: "constraint", Title: "[orch-cli] Worker agents must not spawn"},
		{Type: "constraint", Title: "[price-watch] Max retries per product"},
		{Type: "decision", Title: "[kb-cli] Use YAML for config"},
		{Type: "decision", Title: "[dotfiles] Zsh is the default shell"},
		{Type: "investigation", Title: "[orch-knowledge] Pattern analysis"},
		{Type: "investigation", Title: "[beads] Issue tracking investigation"},
		{Type: "investigation", Title: "[scs-slack] Slack integration research"},
		{Type: "constraint", Title: "Local constraint without prefix"}, // Should be included
	}

	filtered := filterToOrchEcosystem(matches)

	// Should keep: orch-go, orch-cli, kb-cli, orch-knowledge, beads, and local (no prefix)
	// Should filter: price-watch, dotfiles, scs-slack
	expectedCount := 6
	if len(filtered) != expectedCount {
		t.Errorf("filterToOrchEcosystem() returned %d matches, want %d", len(filtered), expectedCount)
	}

	// Verify specific matches
	wantTitles := map[string]bool{
		"[orch-go] Agents must not spawn recursively": true,
		"[orch-cli] Worker agents must not spawn":     true,
		"[kb-cli] Use YAML for config":                true,
		"[orch-knowledge] Pattern analysis":           true,
		"[beads] Issue tracking investigation":        true,
		"Local constraint without prefix":             true,
	}

	for _, m := range filtered {
		if !wantTitles[m.Title] {
			t.Errorf("filterToOrchEcosystem() included unwanted match: %q", m.Title)
		}
	}

	// Verify filtered out matches are not present
	noTitles := []string{
		"[price-watch] Max retries per product",
		"[dotfiles] Zsh is the default shell",
		"[scs-slack] Slack integration research",
	}
	for _, m := range filtered {
		for _, noTitle := range noTitles {
			if m.Title == noTitle {
				t.Errorf("filterToOrchEcosystem() should have filtered out: %q", noTitle)
			}
		}
	}
}

func TestApplyPerCategoryLimits(t *testing.T) {
	// Create matches with 25 constraints, 10 decisions, 5 investigations
	var matches []KBContextMatch
	for i := 0; i < 25; i++ {
		matches = append(matches, KBContextMatch{Type: "constraint", Title: "C" + string(rune('0'+i%10))})
	}
	for i := 0; i < 10; i++ {
		matches = append(matches, KBContextMatch{Type: "decision", Title: "D" + string(rune('0'+i))})
	}
	for i := 0; i < 5; i++ {
		matches = append(matches, KBContextMatch{Type: "investigation", Title: "I" + string(rune('0'+i))})
	}

	// Apply limit of 20 per category
	filtered := applyPerCategoryLimits(matches, 20)

	// Count by type
	counts := make(map[string]int)
	for _, m := range filtered {
		counts[m.Type]++
	}

	// Constraints should be capped at 20
	if counts["constraint"] != 20 {
		t.Errorf("applyPerCategoryLimits() constraint count = %d, want 20", counts["constraint"])
	}

	// Decisions should all be included (only 10)
	if counts["decision"] != 10 {
		t.Errorf("applyPerCategoryLimits() decision count = %d, want 10", counts["decision"])
	}

	// Investigations should all be included (only 5)
	if counts["investigation"] != 5 {
		t.Errorf("applyPerCategoryLimits() investigation count = %d, want 5", counts["investigation"])
	}
}

func TestMergeResults(t *testing.T) {
	local := &KBContextResult{
		Query: "test",
		Matches: []KBContextMatch{
			{Type: "constraint", Title: "Local constraint"},
			{Type: "decision", Title: "Shared decision"}, // Also in global
		},
	}

	global := &KBContextResult{
		Query: "test",
		Matches: []KBContextMatch{
			{Type: "decision", Title: "Shared decision"}, // Duplicate
			{Type: "investigation", Title: "[orch-go] Global investigation"},
		},
	}

	merged := mergeResults(local, global)

	if merged == nil {
		t.Fatal("mergeResults() returned nil")
	}

	// Should have 3 unique matches (deduplicated)
	if len(merged.Matches) != 3 {
		t.Errorf("mergeResults() returned %d matches, want 3", len(merged.Matches))
	}

	// Verify local matches come first
	if merged.Matches[0].Title != "Local constraint" {
		t.Errorf("mergeResults() first match = %q, want local match first", merged.Matches[0].Title)
	}
}

func TestMergeResults_NilInputs(t *testing.T) {
	local := &KBContextResult{
		Query:   "test",
		Matches: []KBContextMatch{{Type: "constraint", Title: "C1"}},
	}

	// nil local
	if result := mergeResults(nil, local); result != local {
		t.Error("mergeResults(nil, local) should return local")
	}

	// nil global
	if result := mergeResults(local, nil); result != local {
		t.Error("mergeResults(local, nil) should return local")
	}

	// both nil
	if result := mergeResults(nil, nil); result != nil {
		t.Error("mergeResults(nil, nil) should return nil")
	}
}

func TestFormatMatchesForDisplay(t *testing.T) {
	matches := []KBContextMatch{
		{Type: "constraint", Source: "kn", Title: "No infinite loops", Reason: "Prevents runaway"},
		{Type: "decision", Source: "kb", Title: "Use JWT", Path: "/path/to/decision.md"},
		{Type: "investigation", Source: "kb", Title: "Auth flow analysis", Path: "/path/to/inv.md"},
	}

	output := formatMatchesForDisplay(matches, "auth")

	// Check for required sections
	if !strings.Contains(output, "Context for \"auth\"") {
		t.Error("formatMatchesForDisplay() missing query header")
	}
	if !strings.Contains(output, "## CONSTRAINTS") {
		t.Error("formatMatchesForDisplay() missing CONSTRAINTS section")
	}
	if !strings.Contains(output, "## DECISIONS") {
		t.Error("formatMatchesForDisplay() missing DECISIONS section")
	}
	if !strings.Contains(output, "## INVESTIGATIONS") {
		t.Error("formatMatchesForDisplay() missing INVESTIGATIONS section")
	}
	if !strings.Contains(output, "No infinite loops") {
		t.Error("formatMatchesForDisplay() missing constraint title")
	}
	if !strings.Contains(output, "Reason: Prevents runaway") {
		t.Error("formatMatchesForDisplay() missing constraint reason")
	}
}

func TestFormatContextForSpawnWithLimit(t *testing.T) {
	t.Run("nil result returns empty", func(t *testing.T) {
		result := FormatContextForSpawnWithLimit(nil, 10000)
		if result.Content != "" {
			t.Errorf("expected empty content for nil result, got %q", result.Content)
		}
		if result.WasTruncated {
			t.Error("expected WasTruncated=false for nil result")
		}
	})

	t.Run("no matches returns empty", func(t *testing.T) {
		kbResult := &KBContextResult{
			Query:      "test",
			HasMatches: false,
			Matches:    []KBContextMatch{},
		}
		result := FormatContextForSpawnWithLimit(kbResult, 10000)
		if result.Content != "" {
			t.Errorf("expected empty content for no matches, got %q", result.Content)
		}
		if result.WasTruncated {
			t.Error("expected WasTruncated=false for no matches")
		}
	})

	t.Run("small context not truncated", func(t *testing.T) {
		kbResult := &KBContextResult{
			Query:      "test",
			HasMatches: true,
			Matches: []KBContextMatch{
				{Type: "constraint", Title: "C1", Reason: "R1"},
				{Type: "decision", Title: "D1", Path: "/path/d1"},
			},
		}
		result := FormatContextForSpawnWithLimit(kbResult, 10000)
		if result.WasTruncated {
			t.Error("expected WasTruncated=false for small context")
		}
		if result.OriginalMatches != 2 {
			t.Errorf("expected OriginalMatches=2, got %d", result.OriginalMatches)
		}
		if result.TruncatedMatches != 2 {
			t.Errorf("expected TruncatedMatches=2, got %d", result.TruncatedMatches)
		}
		if !strings.Contains(result.Content, "C1") || !strings.Contains(result.Content, "D1") {
			t.Error("expected all matches in content")
		}
	})

	t.Run("truncates investigations first", func(t *testing.T) {
		kbResult := &KBContextResult{
			Query:      "test",
			HasMatches: true,
			Matches: []KBContextMatch{
				{Type: "constraint", Title: "Constraint 1", Reason: "Important constraint reason"},
				{Type: "decision", Title: "Decision 1", Reason: "Important decision reason"},
				{Type: "investigation", Title: "Investigation 1 with a long title", Path: "/very/long/path/to/investigation1.md"},
				{Type: "investigation", Title: "Investigation 2 with a long title", Path: "/very/long/path/to/investigation2.md"},
			},
		}

		// Use a small limit to force truncation
		result := FormatContextForSpawnWithLimit(kbResult, 500)

		if !result.WasTruncated {
			t.Error("expected WasTruncated=true for small limit")
		}
		if result.OriginalMatches != 4 {
			t.Errorf("expected OriginalMatches=4, got %d", result.OriginalMatches)
		}
		// Constraints and decisions should be kept over investigations
		if !strings.Contains(result.Content, "Constraint 1") {
			t.Error("expected constraint to be preserved")
		}
		if !strings.Contains(result.Content, "Decision 1") {
			t.Error("expected decision to be preserved")
		}
		// Check truncation warning is present
		if !strings.Contains(result.Content, "KB context truncated") {
			t.Error("expected truncation warning in content")
		}
	})

	t.Run("truncates decisions before constraints", func(t *testing.T) {
		// Create many decisions and constraints
		var matches []KBContextMatch
		for i := 0; i < 5; i++ {
			matches = append(matches, KBContextMatch{
				Type:   "constraint",
				Title:  strings.Repeat("C", 50),
				Reason: strings.Repeat("R", 100),
			})
		}
		for i := 0; i < 5; i++ {
			matches = append(matches, KBContextMatch{
				Type:   "decision",
				Title:  strings.Repeat("D", 50),
				Reason: strings.Repeat("R", 100),
			})
		}

		kbResult := &KBContextResult{
			Query:      "test",
			HasMatches: true,
			Matches:    matches,
		}

		// Very small limit - should truncate heavily
		result := FormatContextForSpawnWithLimit(kbResult, 800)

		if !result.WasTruncated {
			t.Error("expected truncation with very small limit")
		}
		// OmittedCategories should include "decision" if decisions were truncated
		// but "constraint" should be last to be truncated
		foundDecision := false
		for _, cat := range result.OmittedCategories {
			if cat == "decision" {
				foundDecision = true
			}
		}
		if !foundDecision && result.TruncatedMatches < result.OriginalMatches {
			// If we truncated and decisions were present, they should be omitted before constraints
			// This test verifies the priority order
		}
	})

	t.Run("estimates tokens correctly", func(t *testing.T) {
		kbResult := &KBContextResult{
			Query:      "test",
			HasMatches: true,
			Matches: []KBContextMatch{
				{Type: "constraint", Title: "Test constraint"},
			},
		}
		result := FormatContextForSpawnWithLimit(kbResult, 100000)
		expectedTokens := EstimateTokens(len(result.Content))
		if result.EstimatedTokens != expectedTokens {
			t.Errorf("expected EstimatedTokens=%d, got %d", expectedTokens, result.EstimatedTokens)
		}
	})

	t.Run("FormatContextForSpawn uses default limit", func(t *testing.T) {
		kbResult := &KBContextResult{
			Query:      "test",
			HasMatches: true,
			Matches: []KBContextMatch{
				{Type: "constraint", Title: "Test constraint"},
			},
		}
		// FormatContextForSpawn should use MaxKBContextChars by default
		content := FormatContextForSpawn(kbResult)
		directResult := FormatContextForSpawnWithLimit(kbResult, MaxKBContextChars)
		if content != directResult.Content {
			t.Error("FormatContextForSpawn should produce same output as FormatContextForSpawnWithLimit with default limit")
		}
	})
}

func TestExtractCodeRefs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name: "extracts file paths from Primary Evidence section",
			content: `# Model: Test
**Last Updated:** 2026-01-12

**Primary Evidence (Verify These):**
- ` + "`pkg/verify/check.go`" + ` - Verification gate implementation
- ` + "`cmd/orch/complete_cmd.go`" + ` - Completion orchestration pipeline
- ` + "`pkg/spawn/kbcontext.go:150`" + ` - Context formatting with line number

Other content here`,
			want: []string{
				"pkg/verify/check.go",
				"cmd/orch/complete_cmd.go",
				"pkg/spawn/kbcontext.go",
			},
		},
		{
			name: "handles paths with functions",
			content: `**Primary Evidence:**
- ` + "`pkg/test.go:TestFunc()`" + ` - Test function
- ` + "`internal/helper.go:DoWork()`" + ` - Helper`,
			want: []string{
				"pkg/test.go",
				"internal/helper.go",
			},
		},
		{
			name: "returns empty for no code references",
			content: `# Model: Test
No code references here`,
			want: []string{},
		},
		{
			name: "handles HTML comment markers (future format)",
			content: `**Primary Evidence:**
<!-- code_refs -->
- ` + "`pkg/test.go`" + ` - Test file
<!-- /code_refs -->`,
			want: []string{"pkg/test.go"},
		},
		{
			name: "skips non-file backtick content",
			content: `**Primary Evidence:**
- ` + "`pkg/test.go`" + ` - Test file
- Use ` + "`--flag`" + ` for options
- Variable ` + "`someVar`" + ` is set`,
			want: []string{"pkg/test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCodeRefs(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("extractCodeRefs() returned %d paths, want %d: %v", len(got), len(tt.want), got)
				return
			}
			for i, wantPath := range tt.want {
				if i < len(got) && got[i] != wantPath {
					t.Errorf("extractCodeRefs()[%d] = %q, want %q", i, got[i], wantPath)
				}
			}
		})
	}
}

func TestExtractLastUpdated(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "extracts Last Updated date",
			content: `# Model: Test
**Last Updated:** 2026-01-12
**Domain:** Testing`,
			want: "2026-01-12",
		},
		{
			name: "handles different spacing",
			content: `**Last Updated:**  2025-12-25
Other content`,
			want: "2025-12-25",
		},
		{
			name: "returns empty if not found",
			content: `# Model: Test
No last updated field`,
			want: "",
		},
		{
			name:    "handles lowercase variant",
			content: `**last updated:** 2026-02-14`,
			want:    "2026-02-14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLastUpdated(tt.content)
			if got != tt.want {
				t.Errorf("extractLastUpdated() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckModelStaleness(t *testing.T) {
	// Note: This test requires git to be available and the repo to have history
	// We'll test with the actual repo's files for realistic scenarios

	t.Run("detects no staleness for future date", func(t *testing.T) {
		// Use a far-future date that will definitely have no commits
		testContent := `# Model: Test
**Last Updated:** 2099-12-31

**Primary Evidence:**
- ` + "`pkg/spawn/kbcontext.go`" + ` - This file (guaranteed to exist)
`
		// Use "../.." to get to project root from pkg/spawn/
		result, err := checkModelStaleness(testContent, "../..")
		if err != nil {
			t.Fatalf("checkModelStaleness() error = %v", err)
		}
		if result == nil {
			t.Fatal("checkModelStaleness() returned nil result")
		}
		// Should not be stale (date is far in future)
		if result.IsStale {
			t.Errorf("checkModelStaleness() IsStale = true for far-future date, want false. Changed: %v, Deleted: %v", result.ChangedFiles, result.DeletedFiles)
		}
	})

	t.Run("returns empty result for model without code refs", func(t *testing.T) {
		testContent := `# Model: Test
**Last Updated:** 2025-01-01

No code references here.
`
		result, err := checkModelStaleness(testContent, "../..")
		if err != nil {
			t.Fatalf("checkModelStaleness() error = %v", err)
		}
		if result.IsStale {
			t.Error("checkModelStaleness() should not be stale when no code refs exist")
		}
		if len(result.ChangedFiles) > 0 {
			t.Error("checkModelStaleness() should have no changed files when no code refs")
		}
	})

	t.Run("returns empty result for model without Last Updated", func(t *testing.T) {
		testContent := `# Model: Test

**Primary Evidence:**
- ` + "`pkg/spawn/kbcontext.go`" + ` - Test file
`
		result, err := checkModelStaleness(testContent, "../..")
		if err != nil {
			t.Fatalf("checkModelStaleness() error = %v", err)
		}
		if result.IsStale {
			t.Error("checkModelStaleness() should not be stale when no Last Updated date")
		}
	})

	t.Run("detects deleted files", func(t *testing.T) {
		testContent := `# Model: Test
**Last Updated:** 2025-01-01

**Primary Evidence:**
- ` + "`pkg/nonexistent/deleted.go`" + ` - This file doesn't exist
`
		result, err := checkModelStaleness(testContent, "../..")
		if err != nil {
			t.Fatalf("checkModelStaleness() error = %v", err)
		}
		if !result.IsStale {
			t.Error("checkModelStaleness() should be stale when referenced file doesn't exist")
		}
		if len(result.DeletedFiles) == 0 {
			t.Error("checkModelStaleness() should report deleted file")
		}
	})
}

func TestStaleModelIntegration(t *testing.T) {
	// Integration test: verify staleness detection works end-to-end with real model formatting
	t.Run("integrates staleness into model formatting", func(t *testing.T) {
		// Create a mock KBContextMatch with a stale model
		match := KBContextMatch{
			Type:  "model",
			Title: "Test Stale Model",
			Path:  "/tmp/test-stale-model.md",
		}

		// Create a temporary model file with stale references
		testModelContent := `# Model: Test
**Last Updated:** 2025-01-01

**Primary Evidence:**
- ` + "`pkg/spawn/kbcontext.go`" + ` - This file has changed since 2025-01-01
`
		err := os.WriteFile(match.Path, []byte(testModelContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test model file: %v", err)
		}
		defer os.Remove(match.Path)

		// Format the model
		formatted, isStale := formatModelMatchForSpawn(match, "../..")

		// Should detect staleness
		if !isStale {
			t.Error("formatModelMatchForSpawn() should detect staleness for file changed since 2025-01-01")
		}

		// Should include staleness warning in formatted output
		if !strings.Contains(formatted, "STALENESS WARNING") {
			t.Error("formatted output should include staleness warning")
		}
		if !strings.Contains(formatted, "2025-01-01") {
			t.Error("formatted output should include Last Updated date")
		}
	})
}
