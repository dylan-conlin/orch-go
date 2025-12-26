package spawn

import (
	"strings"
	"testing"
)

func TestAnalyzeGaps_NilResult(t *testing.T) {
	analysis := AnalyzeGaps(nil, "test")

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

	analysis := AnalyzeGaps(result, "test")

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

	analysis := AnalyzeGaps(result, "test")

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

	analysis := AnalyzeGaps(result, "test")

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

	analysis := AnalyzeGaps(result, "test")

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
