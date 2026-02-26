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

	desc := buildImplementationDescription(synthesis, beadsID)

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

// containsStr is defined in review_triage_test.go
