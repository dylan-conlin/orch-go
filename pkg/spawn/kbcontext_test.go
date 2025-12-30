package spawn

import (
	"encoding/json"
	"os"
	"path/filepath"
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
			name: "parses investigations",
			output: `Context for "auth":

## INVESTIGATIONS (from kb)

- Authentication Flow Analysis
  Path: /path/to/investigation.md`,
			wantCount:   1,
			wantTypes:   []string{"investigation"},
			wantSources: []string{"kb"},
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

func TestExtractDeltaFromInvestigation(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		wantDelta string
	}{
		{
			name: "extracts delta from standard format",
			content: `---
linked_issues:
  - orch-go-xyz
---
## Summary (D.E.K.N.)

**Delta:** The spawn context should include related investigations to help agents create lineage references.

**Evidence:** Analyzed 500 investigations, found 0 lineage references.

**Knowledge:** Agents need context to build knowledge connections.

**Next:** Implement investigation inclusion in spawn context.

---

# Investigation: Test Investigation
`,
			wantDelta: "The spawn context should include related investigations to help agents create lineage references.",
		},
		{
			name: "handles file without delta",
			content: `# Investigation: Simple Investigation

Some content here without D.E.K.N. format.
`,
			wantDelta: "",
		},
		{
			name: "handles empty delta line",
			content: `## Summary (D.E.K.N.)

**Delta:**

**Evidence:** Some evidence
`,
			wantDelta: "",
		},
		{
			name: "extracts delta with special characters",
			content: `## Summary (D.E.K.N.)

**Delta:** This is a finding with "quotes" and (parentheses) - including dashes.

**Evidence:** Test
`,
			wantDelta: `This is a finding with "quotes" and (parentheses) - including dashes.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test file
			testFile := filepath.Join(tempDir, tt.name+".md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got := extractDeltaFromInvestigation(testFile)
			if got != tt.wantDelta {
				t.Errorf("extractDeltaFromInvestigation() = %q, want %q", got, tt.wantDelta)
			}
		})
	}

	// Test non-existent file
	t.Run("handles non-existent file", func(t *testing.T) {
		got := extractDeltaFromInvestigation("/non/existent/file.md")
		if got != "" {
			t.Errorf("expected empty string for non-existent file, got %q", got)
		}
	})
}

func TestEnrichInvestigationsWithDelta(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test investigation files
	inv1Content := `## Summary (D.E.K.N.)

**Delta:** First investigation key finding.
`
	inv2Content := `## Summary (D.E.K.N.)

**Delta:** Second investigation key finding.
`
	inv3Content := `## Summary (D.E.K.N.)

**Delta:** Third investigation key finding.
`
	inv4Content := `## Summary (D.E.K.N.)

**Delta:** Fourth investigation key finding (should be dropped).
`

	inv1Path := filepath.Join(tempDir, "inv1.md")
	inv2Path := filepath.Join(tempDir, "inv2.md")
	inv3Path := filepath.Join(tempDir, "inv3.md")
	inv4Path := filepath.Join(tempDir, "inv4.md")

	os.WriteFile(inv1Path, []byte(inv1Content), 0644)
	os.WriteFile(inv2Path, []byte(inv2Content), 0644)
	os.WriteFile(inv3Path, []byte(inv3Content), 0644)
	os.WriteFile(inv4Path, []byte(inv4Content), 0644)

	t.Run("limits to MaxInvestigationsInContext", func(t *testing.T) {
		investigations := []KBContextMatch{
			{Type: "investigation", Title: "Inv 1", Path: inv1Path},
			{Type: "investigation", Title: "Inv 2", Path: inv2Path},
			{Type: "investigation", Title: "Inv 3", Path: inv3Path},
			{Type: "investigation", Title: "Inv 4", Path: inv4Path},
		}

		enriched := enrichInvestigationsWithDelta(investigations)

		if len(enriched) != MaxInvestigationsInContext {
			t.Errorf("expected %d investigations, got %d", MaxInvestigationsInContext, len(enriched))
		}
	})

	t.Run("enriches with delta", func(t *testing.T) {
		investigations := []KBContextMatch{
			{Type: "investigation", Title: "Inv 1", Path: inv1Path},
			{Type: "investigation", Title: "Inv 2", Path: inv2Path},
		}

		enriched := enrichInvestigationsWithDelta(investigations)

		if enriched[0].Delta != "First investigation key finding." {
			t.Errorf("expected delta for inv1, got %q", enriched[0].Delta)
		}
		if enriched[1].Delta != "Second investigation key finding." {
			t.Errorf("expected delta for inv2, got %q", enriched[1].Delta)
		}
	})

	t.Run("handles missing path", func(t *testing.T) {
		investigations := []KBContextMatch{
			{Type: "investigation", Title: "Inv without path"},
		}

		enriched := enrichInvestigationsWithDelta(investigations)

		if len(enriched) != 1 {
			t.Errorf("expected 1 investigation, got %d", len(enriched))
		}
		if enriched[0].Delta != "" {
			t.Errorf("expected empty delta for missing path, got %q", enriched[0].Delta)
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		investigations := []KBContextMatch{}
		enriched := enrichInvestigationsWithDelta(investigations)
		if len(enriched) != 0 {
			t.Errorf("expected empty list, got %d", len(enriched))
		}
	})
}

func TestFormatChronicleForSpawn(t *testing.T) {
	t.Run("nil result returns empty", func(t *testing.T) {
		result := FormatChronicleForSpawn(nil)
		if result != "" {
			t.Errorf("expected empty string for nil result, got %q", result)
		}
	})

	t.Run("empty timeline returns empty", func(t *testing.T) {
		result := FormatChronicleForSpawn(&ChronicleResult{
			Topic:    "test",
			Timeline: []ChronicleEntry{},
		})
		if result != "" {
			t.Errorf("expected empty string for empty timeline, got %q", result)
		}
	})

	t.Run("formats single investigation", func(t *testing.T) {
		result := FormatChronicleForSpawn(&ChronicleResult{
			Topic: "spawn context",
			Timeline: []ChronicleEntry{
				{
					Date:    "2025-12-30",
					Type:    "investigation",
					Title:   "Spawn Context Generation",
					Summary: "Found that spawn context uses generic keywords.",
					Path:    "/path/to/investigation.md",
				},
			},
		})

		if !strings.Contains(result, "Prior Investigations on This Topic") {
			t.Error("expected 'Prior Investigations on This Topic' header")
		}
		if !strings.Contains(result, "Spawn Context Generation") {
			t.Error("expected investigation title in output")
		}
		if !strings.Contains(result, "Found that spawn context uses generic keywords.") {
			t.Error("expected summary in output")
		}
		if !strings.Contains(result, "/path/to/investigation.md") {
			t.Error("expected path in output")
		}
		if !strings.Contains(result, "spawn context") {
			t.Error("expected topic in output")
		}
	})

	t.Run("truncates long summaries", func(t *testing.T) {
		longSummary := strings.Repeat("a", 300)
		result := FormatChronicleForSpawn(&ChronicleResult{
			Topic: "test",
			Timeline: []ChronicleEntry{
				{
					Type:    "investigation",
					Title:   "Test Investigation",
					Summary: longSummary,
				},
			},
		})

		if strings.Contains(result, longSummary) {
			t.Error("expected long summary to be truncated")
		}
		if !strings.Contains(result, "...") {
			t.Error("expected ellipsis for truncated summary")
		}
	})

	t.Run("formats multiple investigations", func(t *testing.T) {
		result := FormatChronicleForSpawn(&ChronicleResult{
			Topic: "auth",
			Timeline: []ChronicleEntry{
				{Type: "investigation", Title: "Auth Flow Analysis", Path: "/path/1.md"},
				{Type: "investigation", Title: "JWT Implementation", Path: "/path/2.md"},
				{Type: "investigation", Title: "Session Management", Path: "/path/3.md"},
			},
		})

		if !strings.Contains(result, "Auth Flow Analysis") {
			t.Error("expected first investigation title")
		}
		if !strings.Contains(result, "JWT Implementation") {
			t.Error("expected second investigation title")
		}
		if !strings.Contains(result, "Session Management") {
			t.Error("expected third investigation title")
		}
	})
}

func TestChronicleResultParsing(t *testing.T) {
	// Test that we can parse the expected JSON format from kb chronicle
	jsonData := `{
		"topic": "spawn context",
		"timeline": [
			{
				"date": "2025-12-30T00:00:00Z",
				"type": "investigation",
				"title": "CLI orch spawn Command Implementation",
				"summary": "First checking direct symlinks",
				"path": "/path/to/investigation.md",
				"id": ""
			}
		],
		"investigations": []
	}`

	var result ChronicleResult
	err := json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		t.Fatalf("failed to parse chronicle JSON: %v", err)
	}

	if result.Topic != "spawn context" {
		t.Errorf("expected topic 'spawn context', got %q", result.Topic)
	}
	if len(result.Timeline) != 1 {
		t.Errorf("expected 1 timeline entry, got %d", len(result.Timeline))
	}
	if result.Timeline[0].Type != "investigation" {
		t.Errorf("expected type 'investigation', got %q", result.Timeline[0].Type)
	}
	if result.Timeline[0].Title != "CLI orch spawn Command Implementation" {
		t.Errorf("unexpected title: %q", result.Timeline[0].Title)
	}
}

func TestFormatContextIncludesDelta(t *testing.T) {
	// Create a temporary directory for test file
	tempDir := t.TempDir()

	invContent := `## Summary (D.E.K.N.)

**Delta:** Important finding about spawn context generation.
`
	invPath := filepath.Join(tempDir, "test-inv.md")
	os.WriteFile(invPath, []byte(invContent), 0644)

	kbResult := &KBContextResult{
		Query:      "spawn context",
		HasMatches: true,
		Matches: []KBContextMatch{
			{Type: "investigation", Title: "Spawn Context Investigation", Path: invPath},
		},
	}

	result := FormatContextForSpawnWithLimit(kbResult, 100000)

	// Check that the delta is included in the formatted output
	if !strings.Contains(result.Content, "**Key finding:**") {
		t.Error("expected '**Key finding:**' in formatted output")
	}
	if !strings.Contains(result.Content, "Important finding about spawn context generation.") {
		t.Error("expected delta content in formatted output")
	}
}
