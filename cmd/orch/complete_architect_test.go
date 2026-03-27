package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestInferImplementationSkill(t *testing.T) {
	tests := []struct {
		name      string
		synthesis *verify.Synthesis
		expected  string
	}{
		{
			name: "fix/debug keywords → systematic-debugging",
			synthesis: &verify.Synthesis{
				TLDR: "Fix the nil pointer crash in daemon spawn",
				Next: "Fix the crash in daemon.go",
				NextActions: []string{
					"- Fix nil pointer dereference in spawnAgent()",
				},
			},
			expected: "systematic-debugging",
		},
		{
			name: "implement/add keywords → feature-impl",
			synthesis: &verify.Synthesis{
				TLDR: "Design new auto-create feature",
				Next: "Implement the auto-create pipeline",
				NextActions: []string{
					"- Implement auto-create logic in complete_cmd.go",
				},
			},
			expected: "feature-impl",
		},
		{
			name: "investigate/analyze keywords → investigation",
			synthesis: &verify.Synthesis{
				TLDR: "Need deeper investigation into performance",
				Next: "Investigate the root cause further",
				NextActions: []string{
					"- Investigate why daemon spawns are slow",
				},
			},
			expected: "investigation",
		},
		{
			name: "default fallback → feature-impl",
			synthesis: &verify.Synthesis{
				TLDR: "Designed the new dashboard layout",
				Next: "Apply the design to the dashboard",
				NextActions: []string{
					"- Update dashboard layout",
				},
			},
			expected: "feature-impl",
		},
		{
			name: "refactor keywords → feature-impl",
			synthesis: &verify.Synthesis{
				TLDR: "Refactor extraction logic",
				Next: "Extract the monolithic function",
				NextActions: []string{
					"- Refactor daemon.go into smaller functions",
				},
			},
			expected: "feature-impl",
		},
		{
			name: "bug/error keywords → systematic-debugging",
			synthesis: &verify.Synthesis{
				TLDR: "Identified bug in session cleanup",
				Next: "Debug the session leak",
				NextActions: []string{
					"- Debug why sessions leak after crash",
				},
			},
			expected: "systematic-debugging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferImplementationSkill(tt.synthesis)
			if result != tt.expected {
				t.Errorf("inferImplementationSkill() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildImplementationTitle(t *testing.T) {
	tests := []struct {
		name      string
		synthesis *verify.Synthesis
		beadsID   string
		expected  string
	}{
		{
			name: "uses first next action",
			synthesis: &verify.Synthesis{
				TLDR: "Designed new feature",
				NextActions: []string{
					"- Implement auto-create logic in complete_cmd.go",
					"- Add tests for new feature",
				},
			},
			beadsID:  "orch-go-abc1",
			expected: "Implement auto-create logic in complete_cmd.go (from architect orch-go-abc1)",
		},
		{
			name: "falls back to TLDR when no next actions",
			synthesis: &verify.Synthesis{
				TLDR:        "Designed the auto-create pipeline for architect completions",
				NextActions: nil,
			},
			beadsID:  "orch-go-xyz9",
			expected: "Implement: Designed the auto-create pipeline for architect completions (from architect orch-go-xyz9)",
		},
		{
			name: "strips bullet prefix from action",
			synthesis: &verify.Synthesis{
				NextActions: []string{
					"* Add retry logic to spawn",
				},
			},
			beadsID:  "orch-go-1234",
			expected: "Add retry logic to spawn (from architect orch-go-1234)",
		},
		{
			name: "strips numbered prefix from action",
			synthesis: &verify.Synthesis{
				NextActions: []string{
					"1. Refactor the daemon config",
				},
			},
			beadsID:  "orch-go-5678",
			expected: "Refactor the daemon config (from architect orch-go-5678)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildImplementationTitle(tt.synthesis, tt.beadsID)
			if result != tt.expected {
				t.Errorf("buildImplementationTitle() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsActionableRecommendation(t *testing.T) {
	tests := []struct {
		name           string
		recommendation string
		expected       bool
	}{
		{"implement is actionable", "implement", true},
		{"escalate is actionable", "escalate", true},
		{"spawn is actionable", "spawn", true},
		{"empty is not actionable", "", false},
		{"close is not actionable", "close", false},
		{"done is not actionable", "done", false},
		{"uppercase implement is actionable", "Implement", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isActionableRecommendation(tt.recommendation)
			if result != tt.expected {
				t.Errorf("isActionableRecommendation(%q) = %v, want %v", tt.recommendation, result, tt.expected)
			}
		})
	}
}

func TestBuildImplementationDescription(t *testing.T) {
	synthesis := &verify.Synthesis{
		TLDR: "Designed the caching layer for API responses",
		Next: "**Recommendation:** implement\n\nImplement Redis-based caching.",
		NextActions: []string{
			"- Add Redis client to pkg/cache",
			"- Implement TTL-based invalidation",
		},
	}
	beadsID := "orch-go-abc1"

	desc := buildImplementationDescription(synthesis, beadsID, "", nil)

	// Should contain the architect reference
	if !containsStr(desc, "orch-go-abc1") {
		t.Error("description should reference architect beads ID")
	}
	// Should contain the TLDR
	if !containsStr(desc, "Designed the caching layer") {
		t.Error("description should contain TLDR")
	}
	// Should contain next actions
	if !containsStr(desc, "Redis client") {
		t.Error("description should contain next actions")
	}
}

func TestBuildImplementationDescriptionWithKBContext(t *testing.T) {
	synthesis := &verify.Synthesis{
		TLDR: "Designed enrichment for architect issues",
		NextActions: []string{
			"- Modify buildImplementationDescription in complete_architect.go",
			"- Add kb context extraction",
		},
	}
	beadsID := "orch-go-test1"
	kbContext := "- [decision] No local agent state (2026-01-15)\n- [constraint] Accretion boundaries >1500 lines"

	desc := buildImplementationDescription(synthesis, beadsID, kbContext, nil)

	if !containsStr(desc, "## Relevant Knowledge") {
		t.Error("description should contain Relevant Knowledge section when kbContext is provided")
	}
	if !containsStr(desc, "No local agent state") {
		t.Error("description should include kb context content")
	}
	if !containsStr(desc, "Accretion boundaries") {
		t.Error("description should include all kb context entries")
	}
}

func TestBuildImplementationDescriptionWithTargetFiles(t *testing.T) {
	synthesis := &verify.Synthesis{
		TLDR: "Fix spawn timeout",
		NextActions: []string{
			"- Update timeout in pkg/spawn/kbcontext.go",
		},
	}
	beadsID := "orch-go-test2"
	targetFiles := []string{"pkg/spawn/kbcontext.go", "cmd/orch/complete_architect.go"}

	desc := buildImplementationDescription(synthesis, beadsID, "", targetFiles)

	if !containsStr(desc, "## Target Files") {
		t.Error("description should contain Target Files section when files are provided")
	}
	if !containsStr(desc, "pkg/spawn/kbcontext.go") {
		t.Error("description should list target files")
	}
	if !containsStr(desc, "cmd/orch/complete_architect.go") {
		t.Error("description should list all target files")
	}
}

func TestBuildImplementationDescriptionEmptyEnrichment(t *testing.T) {
	synthesis := &verify.Synthesis{
		TLDR: "Simple change",
	}
	beadsID := "orch-go-test3"

	desc := buildImplementationDescription(synthesis, beadsID, "", nil)

	// Should NOT contain enrichment sections when empty
	if containsStr(desc, "## Relevant Knowledge") {
		t.Error("description should not contain Relevant Knowledge when kbContext is empty")
	}
	if containsStr(desc, "## Target Files") {
		t.Error("description should not contain Target Files when no files provided")
	}
}

func TestExtractTargetFiles(t *testing.T) {
	tests := []struct {
		name     string
		synth    *verify.Synthesis
		expected []string
	}{
		{
			name: "extracts from NextActions",
			synth: &verify.Synthesis{
				NextActions: []string{
					"- Modify buildImplementationDescription in cmd/orch/complete_architect.go",
					"- Update pkg/spawn/kbcontext.go timeout",
				},
			},
			expected: []string{"cmd/orch/complete_architect.go", "pkg/spawn/kbcontext.go"},
		},
		{
			name: "extracts from Delta",
			synth: &verify.Synthesis{
				Delta: "Modified: cmd/orch/work_cmd.go, pkg/verify/synthesis_parser.go",
			},
			expected: []string{"cmd/orch/work_cmd.go", "pkg/verify/synthesis_parser.go"},
		},
		{
			name: "deduplicates across fields",
			synth: &verify.Synthesis{
				Delta:       "Modified: pkg/spawn/kbcontext.go",
				NextActions: []string{"- Fix timeout in pkg/spawn/kbcontext.go"},
			},
			expected: []string{"pkg/spawn/kbcontext.go"},
		},
		{
			name: "empty synthesis returns nil",
			synth: &verify.Synthesis{
				TLDR: "A change with no file references",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTargetFiles(tt.synth)
			if len(result) != len(tt.expected) {
				t.Errorf("extractTargetFiles() returned %d files, want %d: got %v", len(result), len(tt.expected), result)
				return
			}
			for _, exp := range tt.expected {
				found := false
				for _, r := range result {
					if r == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("extractTargetFiles() missing expected file %q, got %v", exp, result)
				}
			}
		})
	}
}

func TestBuildArchitectPhaseTitle(t *testing.T) {
	tests := []struct {
		name     string
		phase    verify.PhaseInfo
		beadsID  string
		expected string
	}{
		{
			name:     "phase with title",
			phase:    verify.PhaseInfo{Number: 1, Title: "Add data parser"},
			beadsID:  "orch-go-abc1",
			expected: "Phase 1: Add data parser (from architect orch-go-abc1)",
		},
		{
			name:     "phase without title",
			phase:    verify.PhaseInfo{Number: 2, Title: ""},
			beadsID:  "orch-go-abc1",
			expected: "Phase 2 implementation (from architect orch-go-abc1)",
		},
		{
			name:     "phase 3",
			phase:    verify.PhaseInfo{Number: 3, Title: "Wire integration tests"},
			beadsID:  "orch-go-xyz9",
			expected: "Phase 3: Wire integration tests (from architect orch-go-xyz9)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildArchitectPhaseTitle(tt.phase, tt.beadsID)
			if result != tt.expected {
				t.Errorf("buildArchitectPhaseTitle() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildArchitectPhaseDescription(t *testing.T) {
	synthesis := &verify.Synthesis{
		TLDR: "Designed three-phase enrichment pipeline",
	}
	phase := verify.PhaseInfo{
		Number:      1,
		Title:       "Add data parser",
		Description: "Extract and normalize incoming data from pkg/pipeline/parser.go.",
	}
	beadsID := "orch-go-abc1"

	desc := buildArchitectPhaseDescription(phase, synthesis, beadsID, "")

	// Should reference the architect
	if !containsStr(desc, "orch-go-abc1") {
		t.Error("description should reference architect beads ID")
	}
	// Should mention multi-phase
	if !containsStr(desc, "multi-phase") {
		t.Error("description should mention multi-phase design")
	}
	// Should contain the TLDR
	if !containsStr(desc, "Designed three-phase enrichment pipeline") {
		t.Error("description should contain architect TLDR")
	}
	// Should contain phase title and description
	if !containsStr(desc, "Phase 1: Add data parser") {
		t.Error("description should contain phase heading")
	}
	if !containsStr(desc, "Extract and normalize") {
		t.Error("description should contain phase description")
	}
}

func TestBuildArchitectPhaseDescriptionWithKBContext(t *testing.T) {
	synthesis := &verify.Synthesis{
		TLDR: "Designed pipeline",
	}
	phase := verify.PhaseInfo{
		Number:      1,
		Title:       "Add parser",
		Description: "Parse data.",
	}
	kbContext := "- [decision] No local agent state"

	desc := buildArchitectPhaseDescription(phase, synthesis, "orch-go-test", kbContext)

	if !containsStr(desc, "## Relevant Knowledge") {
		t.Error("description should contain Relevant Knowledge when kbContext is provided")
	}
	if !containsStr(desc, "No local agent state") {
		t.Error("description should include kb context content")
	}
}

func TestBuildArchitectPhaseDescriptionExtractsTargetFiles(t *testing.T) {
	synthesis := &verify.Synthesis{
		TLDR: "Pipeline design",
	}
	phase := verify.PhaseInfo{
		Number:      1,
		Title:       "Add parser",
		Description: "Modify pkg/pipeline/parser.go to handle edge cases.",
	}

	desc := buildArchitectPhaseDescription(phase, synthesis, "orch-go-test", "")

	if !containsStr(desc, "## Target Files") {
		t.Error("description should contain Target Files when phase references files")
	}
	if !containsStr(desc, "pkg/pipeline/parser.go") {
		t.Error("description should list target files from phase description")
	}
}

// containsStr is defined in review_triage_test.go
