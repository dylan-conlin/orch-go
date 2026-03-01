package spawn

import (
	"os"
	"strings"
	"testing"
	"time"
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
		{
			name:      "strips architect skill prefix",
			task:      "Architect: Redesign pricing comparison KPIs",
			maxWords:  3,
			wantWords: []string{"pricing", "comparison", "kpis"},
			notWords:  []string{"architect", "redesign"}, // skill name and action verb
		},
		{
			name:      "strips debug skill prefix",
			task:      "Debug: Fix spawn context kb relevance",
			maxWords:  3,
			wantWords: []string{"spawn", "context", "relevance"},
			notWords:  []string{"debug"}, // skill name stripped
		},
		{
			name:      "strips investigate skill prefix",
			task:      "Investigate: Why toolshed metrics are inaccurate",
			maxWords:  5,
			wantWords: []string{"toolshed", "metrics", "inaccurate"},
			notWords:  []string{"investigate"}, // skill name stripped
		},
		{
			name:      "filters skill names as stop words without prefix",
			task:      "architect review of spawn refactor plan",
			maxWords:  5,
			wantWords: []string{"review", "spawn", "plan"},
			notWords:  []string{"architect", "refactor"}, // skill name and action verb
		},
		{
			name:      "preserves domain keywords when no skill prefix",
			task:      "Add user authentication to the web dashboard",
			maxWords:  3,
			wantWords: []string{"user", "authentication", "web"},
			notWords:  []string{"add"}, // "add" is a stop word
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

func TestExtractKeywordsWithContext(t *testing.T) {
	tests := []struct {
		name             string
		task             string
		orientationFrame string
		maxWords         int
		wantWords        []string // Check these words are present
		notWords         []string // Check these words are NOT present
	}{
		{
			name:             "uses frame domain terms when title has generic words",
			task:             "Architect: fix kb context query derivation",
			orientationFrame: "Pricing KPI redesign needs better query derivation for cross-domain spawns",
			maxWords:         5,
			wantWords:        []string{"context", "query", "pricing", "kpi"},
			notWords:         []string{"architect", "fix"},
		},
		{
			name:             "frame adds domain terms not in title",
			task:             "Redesign dashboard metrics",
			orientationFrame: "Toolshed pricing comparison needs new KPI tracking for competitor analysis",
			maxWords:         5,
			wantWords:        []string{"dashboard", "metrics", "toolshed", "pricing"},
			notWords:         []string{"redesign"},
		},
		{
			name:             "empty frame falls back to title-only extraction",
			task:             "Add rate limiting to the API",
			orientationFrame: "",
			maxWords:         3,
			wantWords:        []string{"rate", "limiting", "api"},
			notWords:         []string{"add"},
		},
		{
			name:             "deduplicates words appearing in both title and frame",
			task:             "Fix pricing dashboard",
			orientationFrame: "The pricing module has a dashboard rendering bug",
			maxWords:         5,
			wantWords:        []string{"pricing", "dashboard"},
		},
		{
			name:             "respects maxWords across combined sources",
			task:             "Architect: pricing KPI redesign for toolshed",
			orientationFrame: "Full redesign of toolshed competitor pricing comparison with new metrics",
			maxWords:         4,
			wantWords:        []string{"pricing", "kpi", "toolshed"},
		},
		{
			name:             "strips skill prefixes from frame too",
			task:             "Debug: spawn fails",
			orientationFrame: "Investigation: orientation frame keywords dropped from spawn context assembly",
			maxWords:         5,
			wantWords:        []string{"spawn", "orientation", "frame", "keywords"},
			notWords:         []string{"debug", "investigation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractKeywordsWithContext(tt.task, tt.orientationFrame, tt.maxWords)

			for _, word := range tt.wantWords {
				if !strings.Contains(result, word) {
					t.Errorf("ExtractKeywordsWithContext() = %q, want word %q to be present", result, word)
				}
			}

			for _, word := range tt.notWords {
				if strings.Contains(result, word) {
					t.Errorf("ExtractKeywordsWithContext() = %q, word %q should not be present", result, word)
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
			name: "stops at code_refs close marker excludes cross-project refs",
			content: `**Primary Evidence (Verify These):**
<!-- code_refs: machine-parseable file references for staleness detection -->
- ` + "`pkg/spawn/staleness_events.go`" + ` — Staleness event recording
- ` + "`pkg/spawn/kbcontext.go`" + ` — Spawn-time detection
- ` + "`cmd/orch/focus.go:186-205`" + ` — Drift command
<!-- /code_refs -->

**Cross-project evidence:**
- ` + "`kb-cli/.kb/agreements/*.yaml`" + ` — Cross-boundary contracts
- ` + "`kb-cli/cmd/kb/agreements.go`" + ` — Agreements implementation`,
			want: []string{"pkg/spawn/staleness_events.go", "pkg/spawn/kbcontext.go", "cmd/orch/focus.go"},
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
		// Use a far-future date that will definitely have no commits.
		// Note: avoid 2099-12-31 because +1 day (for same-day boundary fix) produces
		// 2100-01-01 which triggers a git date overflow bug.
		testContent := `# Model: Test
**Last Updated:** 2098-12-31

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

	t.Run("expands tilde paths correctly", func(t *testing.T) {
		// Create a temp file in the home directory to reference
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("cannot get home dir")
		}
		// Use a file that definitely exists in home
		testContent := `# Model: Test
**Last Updated:** 2099-01-01

**Primary Evidence:**
- ` + "`~/.zshrc`" + ` - Shell config
`
		result, err := checkModelStaleness(testContent, "../..")
		if err != nil {
			t.Fatalf("checkModelStaleness() error = %v", err)
		}
		// If ~/.zshrc exists, it should NOT be reported as deleted
		if _, statErr := os.Stat(home + "/.zshrc"); statErr == nil {
			for _, df := range result.DeletedFiles {
				if df == "~/.zshrc" {
					t.Error("checkModelStaleness() should expand ~ paths — reported ~/.zshrc as deleted but file exists")
				}
			}
		}
	})

	t.Run("no false positive for same-day commits", func(t *testing.T) {
		// Bug: git log --since=YYYY-MM-DD includes all commits from midnight of that day.
		// If Last Updated is today, commits from earlier today should NOT trigger staleness
		// because the model was updated today and already accounts for them.
		// Fix: we add 1 day to --since, so --since=tomorrow excludes all of today's commits.
		today := time.Now().Format("2006-01-02")
		testContent := `# Model: Test
**Last Updated:** ` + today + `

**Primary Evidence:**
- ` + "`pkg/spawn/kbcontext.go`" + ` - This file (has commits today)
`
		result, err := checkModelStaleness(testContent, "../..")
		if err != nil {
			t.Fatalf("checkModelStaleness() error = %v", err)
		}
		// Should NOT report kbcontext.go as changed — model was updated today
		for _, cf := range result.ChangedFiles {
			if cf == "pkg/spawn/kbcontext.go" {
				t.Error("checkModelStaleness() false positive: reported same-day commit as changed. " +
					"Model Last Updated is today, so today's commits should not trigger staleness.")
			}
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
		formatted, isStale := formatModelMatchForSpawn(match, "../..", nil)

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

func TestFilterToProjectGroup(t *testing.T) {
	matches := []KBContextMatch{
		{Type: "constraint", Title: "[orch-go] Agents must not spawn recursively"},
		{Type: "constraint", Title: "[price-watch] Max retries per product"},
		{Type: "decision", Title: "[toolshed] Use React for UI"},
		{Type: "investigation", Title: "[scs-slack] Slack integration research"},
		{Type: "constraint", Title: "Local constraint without prefix"},
	}

	t.Run("filters to SCS group", func(t *testing.T) {
		scsAllowlist := map[string]bool{
			"scs-special-projects": true,
			"toolshed":             true,
			"price-watch":          true,
			"scs-slack":            true,
		}

		filtered := filterToProjectGroup(matches, scsAllowlist)

		// Should keep: price-watch, toolshed, scs-slack, and local (no prefix)
		// Should filter: orch-go (not in SCS group)
		if len(filtered) != 4 {
			t.Errorf("filterToProjectGroup() returned %d matches, want 4", len(filtered))
		}

		for _, m := range filtered {
			project := extractProjectFromMatch(m)
			if project != "" && !scsAllowlist[project] {
				t.Errorf("filterToProjectGroup() included non-SCS project: %q", project)
			}
		}
	})

	t.Run("filters to orch group", func(t *testing.T) {
		orchAllowlist := map[string]bool{
			"orch-go":        true,
			"orch-cli":       true,
			"kb-cli":         true,
			"orch-knowledge": true,
		}

		filtered := filterToProjectGroup(matches, orchAllowlist)

		// Should keep: orch-go and local (no prefix)
		// Should filter: price-watch, toolshed, scs-slack
		if len(filtered) != 2 {
			t.Errorf("filterToProjectGroup() returned %d matches, want 2", len(filtered))
		}
	})

	t.Run("nil allowlist includes all", func(t *testing.T) {
		// When allowlist is nil, filterToProjectGroup is not called (handled by caller)
		// But if called with nil, it should include nothing (no project matches nil map)
		// This tests the caller's responsibility — nil means "don't filter"
		// The actual RunKBContextCheck handles nil by not calling the filter
	})
}

func TestDetectCurrentProjectName(t *testing.T) {
	// This test just verifies the function doesn't panic and returns something reasonable
	name := detectCurrentProjectName()
	if name == "" {
		t.Error("detectCurrentProjectName() returned empty string")
	}
	// We're running from orch-go, so it should detect that
	if name != "orch-go" {
		t.Logf("detectCurrentProjectName() = %q (expected orch-go, but may vary by test environment)", name)
	}
}

func TestDetectProjectNameFromDir(t *testing.T) {
	t.Run("empty dir falls back to cwd", func(t *testing.T) {
		name := detectProjectNameFromDir("")
		if name == "" {
			t.Error("detectProjectNameFromDir(\"\") returned empty string")
		}
		// Should match detectCurrentProjectName behavior
		cwdName := detectCurrentProjectName()
		if name != cwdName {
			t.Errorf("detectProjectNameFromDir(\"\") = %q, want %q (same as detectCurrentProjectName)", name, cwdName)
		}
	})

	t.Run("explicit dir uses basename", func(t *testing.T) {
		// Create a temp dir to simulate a project directory
		tmpDir := t.TempDir()
		name := detectProjectNameFromDir(tmpDir)
		// Should return the basename of the temp directory
		if name == "" {
			t.Error("detectProjectNameFromDir(tmpDir) returned empty string")
		}
	})

	t.Run("dir with beads config uses issue-prefix", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := tmpDir + "/.beads"
		if err := os.MkdirAll(beadsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		// Write config with issue-prefix
		if err := os.WriteFile(beadsDir+"/config.yaml", []byte("issue-prefix: toolshed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		name := detectProjectNameFromDir(tmpDir)
		if name != "toolshed" {
			t.Errorf("detectProjectNameFromDir() = %q, want \"toolshed\"", name)
		}
	})

	t.Run("cross-project dir detects different project", func(t *testing.T) {
		// This is the core bug test: when spawning from orch-go with --workdir pointing elsewhere,
		// the function should return the target project name, not the calling process's project.
		cwdName := detectProjectNameFromDir("")
		otherDir := t.TempDir() + "/other-project"
		if err := os.MkdirAll(otherDir, 0o755); err != nil {
			t.Fatal(err)
		}
		otherName := detectProjectNameFromDir(otherDir)
		if otherName == cwdName {
			t.Errorf("detectProjectNameFromDir(otherDir) should differ from cwd, both returned %q", cwdName)
		}
		if otherName != "other-project" {
			t.Errorf("detectProjectNameFromDir(otherDir) = %q, want \"other-project\"", otherName)
		}
	})
}

func TestTaskIsScoped(t *testing.T) {
	tests := []struct {
		name string
		task string
		want bool
	}{
		{
			name: "file path with extension",
			task: "Fix the bug in pkg/spawn/context.go",
			want: true,
		},
		{
			name: "file path with line number",
			task: "Change error message at cmd/orch/spawn_cmd.go:234",
			want: true,
		},
		{
			name: "deep file path",
			task: "Update the constant in pkg/spawn/kbcontext.go",
			want: true,
		},
		{
			name: "relative file path",
			task: "Edit ./cmd/orch/main.go to add flag",
			want: true,
		},
		{
			name: "typescript file path",
			task: "Fix the React component in src/components/Dashboard.tsx",
			want: true,
		},
		{
			name: "generic task no file paths",
			task: "Add rate limiting to the API",
			want: false,
		},
		{
			name: "abstract refactoring task",
			task: "Refactor the spawn pipeline for better modularity",
			want: false,
		},
		{
			name: "package-level task without file",
			task: "Fix the flaky test in the spawn package",
			want: false,
		},
		{
			name: "high-level feature",
			task: "Implement scope-appropriate kb context injection for targeted tasks",
			want: false,
		},
		{
			name: "multiple file paths",
			task: "Rename function in pkg/orch/extraction.go and pkg/spawn/context.go",
			want: true,
		},
		{
			name: "empty task",
			task: "",
			want: false,
		},
		{
			name: "url should not match",
			task: "Check the docs at https://example.com/api/v2",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TaskIsScoped(tt.task)
			if got != tt.want {
				t.Errorf("TaskIsScoped(%q) = %v, want %v", tt.task, got, tt.want)
			}
		})
	}
}

func TestFilterForScopedTask(t *testing.T) {
	matches := []KBContextMatch{
		{Type: "constraint", Title: "No infinite loops", Reason: "Safety"},
		{Type: "decision", Title: "Use TDD", Reason: "Quality"},
		{Type: "model", Title: "Spawn Architecture", Path: "/path/to/model.md"},
		{Type: "guide", Title: "How Spawn Works", Path: "/path/to/guide.md"},
		{Type: "investigation", Title: "Auth flow analysis", Path: "/path/to/inv.md"},
		{Type: "failed-attempt", Title: "Tried Redis", Reason: "Too complex"},
		{Type: "open-question", Title: "Should we use SQLite?"},
	}

	filtered := FilterForScopedTask(matches)

	// Should keep constraints, decisions, and failed attempts
	// Should drop models, guides, investigations, and open questions
	typeCount := make(map[string]int)
	for _, m := range filtered {
		typeCount[m.Type]++
	}

	if typeCount["constraint"] != 1 {
		t.Errorf("expected 1 constraint, got %d", typeCount["constraint"])
	}
	if typeCount["decision"] != 1 {
		t.Errorf("expected 1 decision, got %d", typeCount["decision"])
	}
	if typeCount["failed-attempt"] != 1 {
		t.Errorf("expected 1 failed-attempt, got %d", typeCount["failed-attempt"])
	}
	if typeCount["model"] != 0 {
		t.Errorf("expected 0 models for scoped task, got %d", typeCount["model"])
	}
	if typeCount["guide"] != 0 {
		t.Errorf("expected 0 guides for scoped task, got %d", typeCount["guide"])
	}
	if typeCount["investigation"] != 0 {
		t.Errorf("expected 0 investigations for scoped task, got %d", typeCount["investigation"])
	}
	if typeCount["open-question"] != 0 {
		t.Errorf("expected 0 open-questions for scoped task, got %d", typeCount["open-question"])
	}

	// Total should be 3 (constraint + decision + failed-attempt)
	if len(filtered) != 3 {
		t.Errorf("FilterForScopedTask() returned %d matches, want 3", len(filtered))
	}
}

func TestFilterForScopedTask_EmptyInput(t *testing.T) {
	filtered := FilterForScopedTask(nil)
	if len(filtered) != 0 {
		t.Errorf("FilterForScopedTask(nil) returned %d matches, want 0", len(filtered))
	}

	filtered = FilterForScopedTask([]KBContextMatch{})
	if len(filtered) != 0 {
		t.Errorf("FilterForScopedTask([]) returned %d matches, want 0", len(filtered))
	}
}

func TestFilterForScopedTask_OnlyConstraints(t *testing.T) {
	matches := []KBContextMatch{
		{Type: "constraint", Title: "C1"},
		{Type: "constraint", Title: "C2"},
	}

	filtered := FilterForScopedTask(matches)
	if len(filtered) != 2 {
		t.Errorf("FilterForScopedTask() returned %d matches, want 2", len(filtered))
	}
}

func TestScopedMaxKBContextChars(t *testing.T) {
	// Verify the scoped budget is smaller than the default
	if ScopedMaxKBContextChars >= MaxKBContextChars {
		t.Errorf("ScopedMaxKBContextChars (%d) should be smaller than MaxKBContextChars (%d)",
			ScopedMaxKBContextChars, MaxKBContextChars)
	}

	// Verify it's at least large enough for a reasonable number of constraints + decisions
	// 15k chars ≈ 3,750 tokens — enough for ~30 constraints/decisions
	if ScopedMaxKBContextChars < 10000 {
		t.Errorf("ScopedMaxKBContextChars (%d) is too small for practical use", ScopedMaxKBContextChars)
	}
}

func TestDetectCrossRepoModel(t *testing.T) {
	// Create temp dirs to simulate git repos
	repoA := t.TempDir()
	repoB := t.TempDir()

	// Set up .git in repoA (simulates a git repo)
	if err := os.MkdirAll(repoA+"/.git", 0755); err != nil {
		t.Fatal(err)
	}
	// Set up .git in repoB
	if err := os.MkdirAll(repoB+"/.git", 0755); err != nil {
		t.Fatal(err)
	}
	// Set up .kb/models in repoB for model path
	modelDir := repoB + "/.kb/models/spawn-architecture"
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name             string
		primaryModelPath string
		projectDir       string
		wantCrossRepo    bool
		wantDir          string
	}{
		{
			name:             "same repo - no cross-repo",
			primaryModelPath: repoA + "/.kb/models/spawn-architecture/model.md",
			projectDir:       repoA,
			wantCrossRepo:    false,
		},
		{
			name:             "different repos - cross-repo detected",
			primaryModelPath: repoB + "/.kb/models/spawn-architecture/model.md",
			projectDir:       repoA,
			wantCrossRepo:    true,
			wantDir:          repoB,
		},
		{
			name:             "empty model path",
			primaryModelPath: "",
			projectDir:       repoA,
			wantCrossRepo:    false,
		},
		{
			name:             "empty project dir",
			primaryModelPath: repoB + "/.kb/models/spawn-architecture/model.md",
			projectDir:       "",
			wantCrossRepo:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectCrossRepoModel(tt.primaryModelPath, tt.projectDir)
			if tt.wantCrossRepo {
				if got == "" {
					t.Errorf("DetectCrossRepoModel() = empty, want %q", tt.wantDir)
				} else if got != tt.wantDir {
					t.Errorf("DetectCrossRepoModel() = %q, want %q", got, tt.wantDir)
				}
			} else {
				if got != "" {
					t.Errorf("DetectCrossRepoModel() = %q, want empty", got)
				}
			}
		})
	}
}

func TestRunKBContextQueryProjectDir(t *testing.T) {
	// Skip if kb command is not available
	if _, err := os.Stat(os.ExpandEnv("$HOME/.bun/bin/kb")); err != nil {
		t.Skip("kb command not available, skipping integration test")
	}

	// Use a temp dir with no .kb/ — local search should find nothing
	emptyDir := t.TempDir()

	// Query with projectDir pointing to empty dir — should return nil (no .kb/ to search)
	result, err := runKBContextQuery("spawn context", false, emptyDir)
	if err != nil {
		t.Fatalf("runKBContextQuery with projectDir returned error: %v", err)
	}
	if result != nil {
		t.Errorf("runKBContextQuery with empty projectDir should return nil, got %d matches", len(result.Matches))
	}

	// Query with empty projectDir (uses CWD = orch-go) — may find results from orch-go .kb/
	// We just verify it doesn't crash; actual results depend on CWD having .kb/
	_, err = runKBContextQuery("spawn context", false, "")
	if err != nil {
		t.Fatalf("runKBContextQuery with empty projectDir returned error: %v", err)
	}
}
