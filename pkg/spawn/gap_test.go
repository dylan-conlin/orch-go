package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeGaps_NilResult(t *testing.T) {
	analysis := AnalyzeGaps(nil, "test", "")

	if !analysis.HasGaps {
		t.Error("expected HasGaps=true for nil result")
	}
	if analysis.ContextQuality != 0 {
		t.Errorf("expected ContextQuality=0 for nil result, got %d", analysis.ContextQuality)
	}
	if len(analysis.Gaps) != 1 {
		t.Errorf("expected 1 gap for nil result, got %d", len(analysis.Gaps))
	}
	if analysis.Gaps[0].Type != GapTypeNoContext {
		t.Errorf("expected GapTypeNoContext, got %s", analysis.Gaps[0].Type)
	}
	if analysis.Gaps[0].Severity != GapSeverityCritical {
		t.Errorf("expected GapSeverityCritical, got %s", analysis.Gaps[0].Severity)
	}
}

func TestAnalyzeGaps_EmptyResult(t *testing.T) {
	result := &KBContextResult{
		Query:      "test",
		HasMatches: false,
		Matches:    []KBContextMatch{},
	}

	analysis := AnalyzeGaps(result, "test", "")

	if !analysis.HasGaps {
		t.Error("expected HasGaps=true for empty result")
	}
	if analysis.ContextQuality != 0 {
		t.Errorf("expected ContextQuality=0 for empty result, got %d", analysis.ContextQuality)
	}
}

func TestAnalyzeGaps_SparseContext(t *testing.T) {
	result := &KBContextResult{
		Query:      "test",
		HasMatches: true,
		Matches: []KBContextMatch{
			{Type: "investigation", Title: "Single investigation"},
		},
	}

	analysis := AnalyzeGaps(result, "test", "")

	if !analysis.HasGaps {
		t.Error("expected HasGaps=true for sparse context")
	}

	// Should have sparse context gap
	foundSparse := false
	for _, gap := range analysis.Gaps {
		if gap.Type == GapTypeSparseContext {
			foundSparse = true
			if gap.Severity != GapSeverityWarning {
				t.Errorf("expected GapSeverityWarning for sparse context, got %s", gap.Severity)
			}
		}
	}
	if !foundSparse {
		t.Error("expected GapTypeSparseContext gap")
	}

	// Should also have no constraints and no decisions gaps
	foundNoConstraints := false
	foundNoDecisions := false
	for _, gap := range analysis.Gaps {
		if gap.Type == GapTypeNoConstraints {
			foundNoConstraints = true
		}
		if gap.Type == GapTypeNoDecisions {
			foundNoDecisions = true
		}
	}
	if !foundNoConstraints {
		t.Error("expected GapTypeNoConstraints gap")
	}
	if !foundNoDecisions {
		t.Error("expected GapTypeNoDecisions gap")
	}
}

func TestAnalyzeGaps_GoodCoverage(t *testing.T) {
	result := &KBContextResult{
		Query:      "test",
		HasMatches: true,
		Matches: []KBContextMatch{
			{Type: "constraint", Title: "C1"},
			{Type: "constraint", Title: "C2"},
			{Type: "decision", Title: "D1"},
			{Type: "decision", Title: "D2"},
			{Type: "investigation", Title: "I1"},
		},
	}

	analysis := AnalyzeGaps(result, "test", "")

	// With good coverage, there should be no gaps
	if analysis.HasGaps {
		t.Errorf("expected no gaps for good coverage, got %d gaps", len(analysis.Gaps))
	}

	// Context quality should be high
	if analysis.ContextQuality < 80 {
		t.Errorf("expected ContextQuality >= 80 for good coverage, got %d", analysis.ContextQuality)
	}

	// Verify stats
	if analysis.MatchStats.TotalMatches != 5 {
		t.Errorf("expected TotalMatches=5, got %d", analysis.MatchStats.TotalMatches)
	}
	if analysis.MatchStats.ConstraintCount != 2 {
		t.Errorf("expected ConstraintCount=2, got %d", analysis.MatchStats.ConstraintCount)
	}
	if analysis.MatchStats.DecisionCount != 2 {
		t.Errorf("expected DecisionCount=2, got %d", analysis.MatchStats.DecisionCount)
	}
	if analysis.MatchStats.InvestigationCount != 1 {
		t.Errorf("expected InvestigationCount=1, got %d", analysis.MatchStats.InvestigationCount)
	}
}

func TestAnalyzeGaps_NoConstraints(t *testing.T) {
	result := &KBContextResult{
		Query:      "test",
		HasMatches: true,
		Matches: []KBContextMatch{
			{Type: "decision", Title: "D1"},
			{Type: "decision", Title: "D2"},
			{Type: "investigation", Title: "I1"},
		},
	}

	analysis := AnalyzeGaps(result, "test", "")

	// Should have no constraints gap (info level)
	foundNoConstraints := false
	for _, gap := range analysis.Gaps {
		if gap.Type == GapTypeNoConstraints {
			foundNoConstraints = true
			if gap.Severity != GapSeverityInfo {
				t.Errorf("expected GapSeverityInfo for no constraints, got %s", gap.Severity)
			}
		}
	}
	if !foundNoConstraints {
		t.Error("expected GapTypeNoConstraints gap")
	}
}

func TestCalculateContextQuality(t *testing.T) {
	tests := []struct {
		name     string
		stats    MatchStatistics
		minScore int
		maxScore int
	}{
		{
			name:     "no matches",
			stats:    MatchStatistics{TotalMatches: 0},
			minScore: 0,
			maxScore: 0,
		},
		{
			name: "single investigation",
			stats: MatchStatistics{
				TotalMatches:       1,
				InvestigationCount: 1,
			},
			minScore: 10,
			maxScore: 20,
		},
		{
			name: "single constraint",
			stats: MatchStatistics{
				TotalMatches:    1,
				ConstraintCount: 1,
			},
			minScore: 20,
			maxScore: 30,
		},
		{
			name: "comprehensive coverage",
			stats: MatchStatistics{
				TotalMatches:       6,
				ConstraintCount:    2,
				DecisionCount:      2,
				InvestigationCount: 2,
			},
			minScore: 80,
			maxScore: 100,
		},
		{
			name: "maximum coverage",
			stats: MatchStatistics{
				TotalMatches:       10,
				ConstraintCount:    4,
				DecisionCount:      4,
				InvestigationCount: 4, // 4 investigations to hit the cap
			},
			minScore: 100,
			maxScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateContextQuality(tt.stats)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateContextQuality() = %d, want between %d and %d",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestGapAnalysis_FormatGapWarning(t *testing.T) {
	t.Run("no gaps returns empty", func(t *testing.T) {
		analysis := &GapAnalysis{HasGaps: false}
		warning := analysis.FormatGapWarning()
		if warning != "" {
			t.Errorf("expected empty warning for no gaps, got %q", warning)
		}
	})

	t.Run("critical gap shows alert", func(t *testing.T) {
		analysis := &GapAnalysis{
			HasGaps:        true,
			ContextQuality: 0,
			Gaps: []Gap{
				{
					Type:        GapTypeNoContext,
					Severity:    GapSeverityCritical,
					Description: "No context found",
					Suggestion:  "Add kn entries",
				},
			},
		}
		warning := analysis.FormatGapWarning()

		if !strings.Contains(warning, "🚨 CONTEXT GAP DETECTED") {
			t.Error("expected critical alert header")
		}
		if !strings.Contains(warning, "No context found") {
			t.Error("expected gap description")
		}
		if !strings.Contains(warning, "Add kn entries") {
			t.Error("expected suggestion")
		}
	})

	t.Run("warning gap shows warning", func(t *testing.T) {
		analysis := &GapAnalysis{
			HasGaps:        true,
			ContextQuality: 25,
			MatchStats: MatchStatistics{
				TotalMatches:       1,
				InvestigationCount: 1,
			},
			Gaps: []Gap{
				{
					Type:        GapTypeSparseContext,
					Severity:    GapSeverityWarning,
					Description: "Only 1 match found",
					Suggestion:  "Add more context",
				},
			},
		}
		warning := analysis.FormatGapWarning()

		if !strings.Contains(warning, "⚠️  Context coverage warning") {
			t.Error("expected warning header")
		}
		if !strings.Contains(warning, "25/100") {
			t.Error("expected quality score")
		}
	})

	t.Run("shows match breakdown", func(t *testing.T) {
		analysis := &GapAnalysis{
			HasGaps:        true,
			ContextQuality: 40,
			MatchStats: MatchStatistics{
				TotalMatches:       3,
				ConstraintCount:    1,
				DecisionCount:      1,
				InvestigationCount: 1,
			},
			Gaps: []Gap{
				{Type: GapTypeNoDecisions, Severity: GapSeverityInfo, Description: "info gap"},
			},
		}
		warning := analysis.FormatGapWarning()

		if !strings.Contains(warning, "constraints: 1") {
			t.Error("expected constraint count in breakdown")
		}
		if !strings.Contains(warning, "decisions: 1") {
			t.Error("expected decision count in breakdown")
		}
	})
}

func TestGapAnalysis_FormatGapSummary(t *testing.T) {
	t.Run("no gaps returns empty", func(t *testing.T) {
		analysis := &GapAnalysis{HasGaps: false}
		summary := analysis.FormatGapSummary()
		if summary != "" {
			t.Errorf("expected empty summary for no gaps, got %q", summary)
		}
	})

	t.Run("zero quality shows no context message", func(t *testing.T) {
		analysis := &GapAnalysis{
			HasGaps:        true,
			ContextQuality: 0,
		}
		summary := analysis.FormatGapSummary()
		if !strings.Contains(summary, "No prior knowledge found") {
			t.Error("expected no prior knowledge message")
		}
	})

	t.Run("low quality shows limited context", func(t *testing.T) {
		analysis := &GapAnalysis{
			HasGaps:        true,
			ContextQuality: 20,
		}
		summary := analysis.FormatGapSummary()
		if !strings.Contains(summary, "Limited context") {
			t.Error("expected limited context message")
		}
	})

	t.Run("moderate quality returns empty", func(t *testing.T) {
		analysis := &GapAnalysis{
			HasGaps:        true,
			ContextQuality: 50,
			Gaps:           []Gap{{Type: GapTypeNoConstraints, Severity: GapSeverityInfo}},
		}
		summary := analysis.FormatGapSummary()
		if summary != "" {
			t.Errorf("expected empty summary for moderate quality, got %q", summary)
		}
	})
}

func TestGapAnalysis_ShouldWarnAboutGaps(t *testing.T) {
	tests := []struct {
		name     string
		analysis *GapAnalysis
		want     bool
	}{
		{
			name:     "no gaps",
			analysis: &GapAnalysis{HasGaps: false},
			want:     false,
		},
		{
			name: "critical gap always warns",
			analysis: &GapAnalysis{
				HasGaps:        true,
				ContextQuality: 80, // Even with high quality
				Gaps:           []Gap{{Severity: GapSeverityCritical}},
			},
			want: true,
		},
		{
			name: "warning gap with low quality warns",
			analysis: &GapAnalysis{
				HasGaps:        true,
				ContextQuality: 20,
				Gaps:           []Gap{{Severity: GapSeverityWarning}},
			},
			want: true,
		},
		{
			name: "warning gap with moderate quality does not warn",
			analysis: &GapAnalysis{
				HasGaps:        true,
				ContextQuality: 50,
				Gaps:           []Gap{{Severity: GapSeverityWarning}},
			},
			want: false,
		},
		{
			name: "info gap does not warn",
			analysis: &GapAnalysis{
				HasGaps:        true,
				ContextQuality: 10,
				Gaps:           []Gap{{Severity: GapSeverityInfo}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.analysis.ShouldWarnAboutGaps()
			if got != tt.want {
				t.Errorf("ShouldWarnAboutGaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountMatchesByType(t *testing.T) {
	matches := []KBContextMatch{
		{Type: "constraint", Title: "C1"},
		{Type: "constraint", Title: "C2"},
		{Type: "decision", Title: "D1"},
		{Type: "investigation", Title: "I1"},
		{Type: "investigation", Title: "I2"},
		{Type: "investigation", Title: "I3"},
		{Type: "guide", Title: "G1"},
	}

	stats := countMatchesByType(matches)

	if stats.TotalMatches != 7 {
		t.Errorf("TotalMatches = %d, want 7", stats.TotalMatches)
	}
	if stats.ConstraintCount != 2 {
		t.Errorf("ConstraintCount = %d, want 2", stats.ConstraintCount)
	}
	if stats.DecisionCount != 1 {
		t.Errorf("DecisionCount = %d, want 1", stats.DecisionCount)
	}
	if stats.InvestigationCount != 3 {
		t.Errorf("InvestigationCount = %d, want 3", stats.InvestigationCount)
	}
	if stats.GuideCount != 1 {
		t.Errorf("GuideCount = %d, want 1", stats.GuideCount)
	}
}

func TestGapAnalysis_ShouldBlockSpawn(t *testing.T) {
	tests := []struct {
		name      string
		analysis  *GapAnalysis
		threshold int
		want      bool
	}{
		{
			name: "no gaps - should not block",
			analysis: &GapAnalysis{
				HasGaps:        false,
				ContextQuality: 75,
			},
			threshold: 20,
			want:      false,
		},
		{
			name: "quality below default threshold - should block",
			analysis: &GapAnalysis{
				HasGaps:        true,
				ContextQuality: 10,
			},
			threshold: 0, // Use default
			want:      true,
		},
		{
			name: "quality at default threshold - should not block",
			analysis: &GapAnalysis{
				HasGaps:        true,
				ContextQuality: 20,
			},
			threshold: 0, // Use default
			want:      false,
		},
		{
			name: "quality below custom threshold - should block",
			analysis: &GapAnalysis{
				HasGaps:        true,
				ContextQuality: 25,
			},
			threshold: 30,
			want:      true,
		},
		{
			name: "quality above custom threshold - should not block",
			analysis: &GapAnalysis{
				HasGaps:        true,
				ContextQuality: 35,
			},
			threshold: 30,
			want:      false,
		},
		{
			name: "zero quality - should always block",
			analysis: &GapAnalysis{
				HasGaps:        true,
				ContextQuality: 0,
			},
			threshold: 20,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.analysis.ShouldBlockSpawn(tt.threshold)
			if got != tt.want {
				t.Errorf("ShouldBlockSpawn(%d) = %v, want %v", tt.threshold, got, tt.want)
			}
		})
	}
}

func TestGapAnalysis_HasCriticalGaps(t *testing.T) {
	tests := []struct {
		name     string
		analysis *GapAnalysis
		want     bool
	}{
		{
			name: "no gaps",
			analysis: &GapAnalysis{
				HasGaps: false,
				Gaps:    []Gap{},
			},
			want: false,
		},
		{
			name: "only info gaps",
			analysis: &GapAnalysis{
				HasGaps: true,
				Gaps:    []Gap{{Severity: GapSeverityInfo}},
			},
			want: false,
		},
		{
			name: "only warning gaps",
			analysis: &GapAnalysis{
				HasGaps: true,
				Gaps:    []Gap{{Severity: GapSeverityWarning}},
			},
			want: false,
		},
		{
			name: "one critical gap",
			analysis: &GapAnalysis{
				HasGaps: true,
				Gaps:    []Gap{{Severity: GapSeverityCritical}},
			},
			want: true,
		},
		{
			name: "mixed gaps with critical",
			analysis: &GapAnalysis{
				HasGaps: true,
				Gaps: []Gap{
					{Severity: GapSeverityInfo},
					{Severity: GapSeverityWarning},
					{Severity: GapSeverityCritical},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.analysis.HasCriticalGaps()
			if got != tt.want {
				t.Errorf("HasCriticalGaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGapAnalysis_FormatGateBlockMessage(t *testing.T) {
	analysis := &GapAnalysis{
		HasGaps:        true,
		ContextQuality: 5,
		Query:          "test task",
		Gaps: []Gap{
			{
				Type:        GapTypeNoContext,
				Severity:    GapSeverityCritical,
				Description: "No prior knowledge found for query",
				Suggestion:  "Add relevant kn entries",
			},
		},
	}

	msg := analysis.FormatGateBlockMessage()

	// Check that key elements are present
	if msg == "" {
		t.Error("FormatGateBlockMessage() returned empty string")
	}
	if !contains(msg, "SPAWN BLOCKED") {
		t.Error("Message should contain 'SPAWN BLOCKED'")
	}
	if !contains(msg, "5/100") {
		t.Error("Message should contain context quality score")
	}
	if !contains(msg, "--skip-gap-gate") {
		t.Error("Message should mention --skip-gap-gate flag")
	}
}

func TestGapAnalysis_FormatProminentWarning(t *testing.T) {
	analysis := &GapAnalysis{
		HasGaps:        true,
		ContextQuality: 25,
		Query:          "test task",
		MatchStats: MatchStatistics{
			TotalMatches:       2,
			ConstraintCount:    0,
			DecisionCount:      1,
			InvestigationCount: 1,
		},
		Gaps: []Gap{
			{
				Type:        GapTypeNoConstraints,
				Severity:    GapSeverityWarning,
				Description: "No constraints found",
			},
		},
	}

	msg := analysis.FormatProminentWarning()

	// Check that key elements are present
	if msg == "" {
		t.Error("FormatProminentWarning() returned empty string")
	}
	if !contains(msg, "CONTEXT GAP") {
		t.Error("Message should contain 'CONTEXT GAP'")
	}
	if !contains(msg, "25/100") {
		t.Error("Message should contain context quality score")
	}
	if !contains(msg, "2 matches") {
		t.Error("Message should mention match count")
	}
}

func TestGapAnalysis_FormatProminentWarning_NoGaps(t *testing.T) {
	analysis := &GapAnalysis{
		HasGaps:        false,
		ContextQuality: 75,
	}

	msg := analysis.FormatProminentWarning()

	if msg != "" {
		t.Errorf("FormatProminentWarning() should return empty string when no gaps, got: %s", msg)
	}
}

func TestGapAnalysis_ToAPIResponse(t *testing.T) {
	analysis := &GapAnalysis{
		HasGaps:        true,
		ContextQuality: 30,
		Query:          "test query",
		MatchStats: MatchStatistics{
			TotalMatches:       5,
			ConstraintCount:    2,
			DecisionCount:      1,
			InvestigationCount: 2,
		},
		Gaps: []Gap{
			{
				Type:        GapTypeNoConstraints,
				Severity:    GapSeverityWarning,
				Description: "No constraints found",
				Suggestion:  "Add via kn constrain",
			},
		},
	}

	resp := analysis.ToAPIResponse()

	if !resp.HasGaps {
		t.Error("API response should have HasGaps = true")
	}
	if resp.ContextQuality != 30 {
		t.Errorf("ContextQuality = %d, want 30", resp.ContextQuality)
	}
	if resp.Query != "test query" {
		t.Errorf("Query = %s, want 'test query'", resp.Query)
	}
	if resp.MatchStats.TotalMatches != 5 {
		t.Errorf("MatchStats.TotalMatches = %d, want 5", resp.MatchStats.TotalMatches)
	}
	if len(resp.Gaps) != 1 {
		t.Errorf("Gaps length = %d, want 1", len(resp.Gaps))
	}
	if resp.Gaps[0].Type != "no_constraints" {
		t.Errorf("Gaps[0].Type = %s, want 'no_constraints'", resp.Gaps[0].Type)
	}
}

func TestIsWrongProjectMatch(t *testing.T) {
	tests := []struct {
		name          string
		match         KBContextMatch
		absProjectDir string
		globalKBDir   string
		want          bool
	}{
		{
			name:          "no path - cannot determine",
			match:         KBContextMatch{Type: "constraint", Title: "C1"},
			absProjectDir: "/home/user/projects/toolshed",
			globalKBDir:   "/home/user/.kb",
			want:          false,
		},
		{
			name:          "correct project path",
			match:         KBContextMatch{Type: "model", Path: "/home/user/projects/toolshed/.kb/models/pricing/model.md"},
			absProjectDir: "/home/user/projects/toolshed",
			globalKBDir:   "/home/user/.kb",
			want:          false,
		},
		{
			name:          "wrong project path",
			match:         KBContextMatch{Type: "model", Path: "/home/user/projects/orch-go/.kb/models/spawn-architecture/model.md"},
			absProjectDir: "/home/user/projects/toolshed",
			globalKBDir:   "/home/user/.kb",
			want:          true,
		},
		{
			name:          "global kb path - acceptable",
			match:         KBContextMatch{Type: "decision", Path: "/home/user/.kb/decisions/2026-02-01-auth.md"},
			absProjectDir: "/home/user/projects/toolshed",
			globalKBDir:   "/home/user/.kb",
			want:          false,
		},
		{
			name:          "path without .kb - not flagged",
			match:         KBContextMatch{Type: "guide", Path: "/home/user/projects/docs/guide.md"},
			absProjectDir: "/home/user/projects/toolshed",
			globalKBDir:   "/home/user/.kb",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWrongProjectMatch(tt.match, tt.absProjectDir, tt.globalKBDir)
			if got != tt.want {
				t.Errorf("isWrongProjectMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzeGaps_WrongProject(t *testing.T) {
	// Simulate toolshed spawn receiving orch-go knowledge
	result := &KBContextResult{
		Query:      "pricing strategy",
		HasMatches: true,
		Matches: []KBContextMatch{
			{Type: "model", Title: "Spawn Architecture", Path: "/home/user/projects/orch-go/.kb/models/spawn-architecture/model.md"},
			{Type: "model", Title: "Dashboard Architecture", Path: "/home/user/projects/orch-go/.kb/models/dashboard/model.md"},
			{Type: "guide", Title: "Model Selection Guide", Path: "/home/user/projects/orch-go/.kb/guides/model-selection.md"},
			{Type: "decision", Title: "Slide-out panel", Path: "/home/user/projects/orch-go/.kb/decisions/2026-02-01-slide-out.md"},
			{Type: "constraint", Title: "Dashboard max-h-64"},
		},
	}

	analysis := AnalyzeGaps(result, "pricing strategy", "/home/user/projects/toolshed")

	// Should detect wrong-project matches
	if analysis.MatchStats.WrongProjectCount != 4 {
		t.Errorf("expected WrongProjectCount=4, got %d", analysis.MatchStats.WrongProjectCount)
	}

	// Should have a wrong_project gap
	foundWrongProject := false
	for _, gap := range analysis.Gaps {
		if gap.Type == GapTypeWrongProject {
			foundWrongProject = true
			// 4 of 5 matches are wrong-project (>50%), should be critical
			if gap.Severity != GapSeverityCritical {
				t.Errorf("expected GapSeverityCritical for majority wrong-project, got %s", gap.Severity)
			}
		}
	}
	if !foundWrongProject {
		t.Error("expected GapTypeWrongProject gap")
	}

	// Quality score should be very low (only 1 valid match out of 5, and it has no path)
	if analysis.ContextQuality > 20 {
		t.Errorf("expected ContextQuality <= 20 for mostly wrong-project matches, got %d", analysis.ContextQuality)
	}
}

func TestAnalyzeGaps_AllWrongProject(t *testing.T) {
	// All matches from wrong project — quality should be 0
	result := &KBContextResult{
		Query:      "pricing",
		HasMatches: true,
		Matches: []KBContextMatch{
			{Type: "model", Title: "Spawn Architecture", Path: "/home/user/projects/orch-go/.kb/models/spawn/model.md"},
			{Type: "decision", Title: "Some decision", Path: "/home/user/projects/orch-go/.kb/decisions/dec.md"},
			{Type: "investigation", Title: "Some inv", Path: "/home/user/projects/orch-go/.kb/investigations/inv.md"},
		},
	}

	analysis := AnalyzeGaps(result, "pricing", "/home/user/projects/toolshed")

	if analysis.ContextQuality != 0 {
		t.Errorf("expected ContextQuality=0 when all matches wrong-project, got %d", analysis.ContextQuality)
	}
	if analysis.MatchStats.WrongProjectCount != 3 {
		t.Errorf("expected WrongProjectCount=3, got %d", analysis.MatchStats.WrongProjectCount)
	}
}

func TestAnalyzeGaps_NoProjectDir(t *testing.T) {
	// When projectDir is empty, wrong-project detection is skipped
	result := &KBContextResult{
		Query:      "test",
		HasMatches: true,
		Matches: []KBContextMatch{
			{Type: "model", Title: "Some model", Path: "/some/other/project/.kb/models/model.md"},
			{Type: "constraint", Title: "C1"},
			{Type: "decision", Title: "D1"},
		},
	}

	analysis := AnalyzeGaps(result, "test", "")

	// No wrong-project detection without projectDir
	if analysis.MatchStats.WrongProjectCount != 0 {
		t.Errorf("expected WrongProjectCount=0 without projectDir, got %d", analysis.MatchStats.WrongProjectCount)
	}
}

func TestAnalyzeGaps_MixedCorrectAndWrong(t *testing.T) {
	// Use real home dir so global .kb/ detection works
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	globalKBPath := filepath.Join(homeDir, ".kb", "decisions", "global.md")
	projectDir := filepath.Join(homeDir, "projects", "toolshed")

	// Mix of correct-project and wrong-project matches
	result := &KBContextResult{
		Query:      "pricing",
		HasMatches: true,
		Matches: []KBContextMatch{
			// Correct project
			{Type: "model", Title: "Toolshed Architecture", Path: filepath.Join(projectDir, ".kb", "models", "arch", "model.md")},
			{Type: "decision", Title: "Toolshed decision", Path: filepath.Join(projectDir, ".kb", "decisions", "dec.md")},
			// Wrong project
			{Type: "model", Title: "Orch-go model", Path: filepath.Join(homeDir, "projects", "orch-go", ".kb", "models", "model.md")},
			// Global (acceptable)
			{Type: "decision", Title: "Global decision", Path: globalKBPath},
			// No path (benefit of doubt)
			{Type: "constraint", Title: "Some constraint"},
		},
	}

	analysis := AnalyzeGaps(result, "pricing", projectDir)

	if analysis.MatchStats.WrongProjectCount != 1 {
		t.Errorf("expected WrongProjectCount=1, got %d", analysis.MatchStats.WrongProjectCount)
	}
	// 1 of 5 wrong = 20%, should be warning not critical
	foundWrongProject := false
	for _, gap := range analysis.Gaps {
		if gap.Type == GapTypeWrongProject {
			foundWrongProject = true
			if gap.Severity != GapSeverityWarning {
				t.Errorf("expected GapSeverityWarning for minority wrong-project, got %s", gap.Severity)
			}
		}
	}
	if !foundWrongProject {
		t.Error("expected GapTypeWrongProject gap")
	}

	// Quality should still be reasonable since most matches are correct
	if analysis.ContextQuality < 40 {
		t.Errorf("expected ContextQuality >= 40 for mostly-correct matches, got %d", analysis.ContextQuality)
	}
}

func TestCalculateContextQuality_WrongProject(t *testing.T) {
	tests := []struct {
		name     string
		stats    MatchStatistics
		minScore int
		maxScore int
	}{
		{
			name: "all wrong project",
			stats: MatchStatistics{
				TotalMatches:      5,
				ConstraintCount:   2,
				DecisionCount:     2,
				WrongProjectCount: 5,
			},
			minScore: 0,
			maxScore: 0,
		},
		{
			name: "half wrong project reduces score",
			stats: MatchStatistics{
				TotalMatches:       6,
				ConstraintCount:    2,
				DecisionCount:      2,
				InvestigationCount: 2,
				WrongProjectCount:  3,
			},
			minScore: 20,
			maxScore: 55, // Roughly half of the ~100 max
		},
		{
			name: "no wrong project - same as before",
			stats: MatchStatistics{
				TotalMatches:       6,
				ConstraintCount:    2,
				DecisionCount:      2,
				InvestigationCount: 2,
				WrongProjectCount:  0,
			},
			minScore: 80,
			maxScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateContextQuality(tt.stats)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateContextQuality() = %d, want between %d and %d", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

// Helper function for string containment check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
