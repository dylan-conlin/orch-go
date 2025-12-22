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
