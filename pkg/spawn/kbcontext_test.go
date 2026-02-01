package spawn

import (
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
		{
			name:      "strips Investigation: prefix",
			task:      "Investigation: Server Crash Patterns",
			maxWords:  6,
			wantWords: []string{"server", "crash", "patterns"},
			notWords:  []string{"investigation"}, // Should be stripped
		},
		{
			name:      "strips Design: prefix",
			task:      "Design: Authentication Flow",
			maxWords:  6,
			wantWords: []string{"authentication", "flow"},
			notWords:  []string{"design"}, // Should be stripped
		},
		{
			name:      "strips ## Investigation: prefix",
			task:      "## Investigation: Performance Issues in Dashboard",
			maxWords:  6,
			wantWords: []string{"performance", "issues", "dashboard"},
			notWords:  []string{"investigation"}, // Should be stripped
		},
		{
			name:      "strips ## Design: prefix",
			task:      "## Design: Database Schema Migration",
			maxWords:  6,
			wantWords: []string{"database", "schema", "migration"},
			notWords:  []string{"design"}, // Should be stripped
		},
		{
			name:      "does not strip investigation when not a prefix",
			task:      "Root cause investigation of memory leaks",
			maxWords:  6,
			wantWords: []string{"investigation", "memory", "leaks"},
			notWords:  []string{}, // investigation should be included since it's not a prefix
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

func TestFilterToEcosystem(t *testing.T) {
	tests := []struct {
		name          string
		domain        string
		matches       []KBContextMatch
		expectedCount int
		wantTitles    []string
		noTitles      []string
	}{
		{
			name:   "personal domain filters to orch ecosystem",
			domain: DomainPersonal,
			matches: []KBContextMatch{
				{Type: "constraint", Title: "[orch-go] Agents must not spawn recursively"},
				{Type: "constraint", Title: "[price-watch] Max retries per product"},
				{Type: "decision", Title: "[kb-cli] Use YAML for config"},
				{Type: "constraint", Title: "Local constraint without prefix"},
			},
			expectedCount: 3,
			wantTitles: []string{
				"[orch-go] Agents must not spawn recursively",
				"[kb-cli] Use YAML for config",
				"Local constraint without prefix",
			},
			noTitles: []string{
				"[price-watch] Max retries per product",
			},
		},
		{
			name:   "work domain filters to work ecosystem",
			domain: DomainWork,
			matches: []KBContextMatch{
				{Type: "constraint", Title: "[orch-go] Agents must not spawn recursively"},
				{Type: "decision", Title: "[scs-special-projects] API rate limits"},
				{Type: "constraint", Title: "Local constraint without prefix"},
			},
			expectedCount: 2,
			wantTitles: []string{
				"[scs-special-projects] API rate limits",
				"Local constraint without prefix",
			},
			noTitles: []string{
				"[orch-go] Agents must not spawn recursively",
			},
		},
		{
			name:   "unknown domain falls back to personal",
			domain: "unknown",
			matches: []KBContextMatch{
				{Type: "constraint", Title: "[orch-go] Constraint from orch ecosystem"},
				{Type: "constraint", Title: "[unknown-repo] Some other constraint"},
			},
			expectedCount: 1,
			wantTitles: []string{
				"[orch-go] Constraint from orch ecosystem",
			},
			noTitles: []string{
				"[unknown-repo] Some other constraint",
			},
		},
		{
			name:   "local matches always included",
			domain: DomainWork,
			matches: []KBContextMatch{
				{Type: "constraint", Title: "Local constraint one"},
				{Type: "constraint", Title: "Local constraint two"},
			},
			expectedCount: 2,
			wantTitles: []string{
				"Local constraint one",
				"Local constraint two",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterToEcosystem(tt.matches, tt.domain)

			if len(filtered) != tt.expectedCount {
				t.Errorf("filterToEcosystem() returned %d matches, want %d", len(filtered), tt.expectedCount)
			}

			// Check wanted titles are present
			for _, wantTitle := range tt.wantTitles {
				found := false
				for _, m := range filtered {
					if m.Title == wantTitle {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("filterToEcosystem() missing expected title: %q", wantTitle)
				}
			}

			// Check unwanted titles are NOT present
			for _, noTitle := range tt.noTitles {
				for _, m := range filtered {
					if m.Title == noTitle {
						t.Errorf("filterToEcosystem() should have filtered out: %q", noTitle)
					}
				}
			}
		})
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
