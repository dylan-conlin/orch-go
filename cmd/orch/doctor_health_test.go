package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/health"
)

func TestCountCommitTypes(t *testing.T) {
	// This test runs against actual git history - it should not fail
	// as long as we're in a git repo (which we are during tests)
	fixes, feats := countCommitTypes()

	// We can't assert exact counts since git history changes,
	// but we can verify the function doesn't panic and returns non-negative values
	if fixes < 0 {
		t.Errorf("Expected non-negative fix count, got %d", fixes)
	}
	if feats < 0 {
		t.Errorf("Expected non-negative feat count, got %d", feats)
	}
}

func TestTrendLabel(t *testing.T) {
	tests := []struct {
		name     string
		trend    health.Trend
		lower    bool
		expected string
	}{
		{"up normal", health.TrendUp, false, "↑"},
		{"up lower-is-better", health.TrendUp, true, "↑ (worsening)"},
		{"down normal", health.TrendDown, false, "↓"},
		{"down lower-is-better", health.TrendDown, true, "↓ (improving)"},
		{"stable", health.TrendStable, false, "→ stable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trendLabel(tt.trend, tt.lower)
			if result != tt.expected {
				t.Errorf("trendLabel(%v, %v) = %q, want %q", tt.trend, tt.lower, result, tt.expected)
			}
		})
	}
}

func TestCollectHealthSnapshot(t *testing.T) {
	// Integration test - runs against real system state
	// Should not panic even if bd is not available
	snap := collectHealthSnapshot()

	// Timestamp should be set
	if snap.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	// Values should be non-negative (bd might not be available in test env)
	if snap.OpenIssues < 0 {
		t.Errorf("Expected non-negative OpenIssues, got %d", snap.OpenIssues)
	}
	if snap.BloatedFiles < 0 {
		t.Errorf("Expected non-negative BloatedFiles, got %d", snap.BloatedFiles)
	}
}

func TestOutputHealthText(t *testing.T) {
	// Verify outputHealthText doesn't panic with various report states
	report := health.Report{
		Current: health.Snapshot{
			OpenIssues:   43,
			BlockedIssues: 15,
			StaleIssues:  8,
			BloatedFiles: 5,
			FixFeatRatio: 1.5,
		},
		SnapshotCount: 10,
		Trends: health.TrendSet{
			OpenIssues:    health.TrendUp,
			BlockedIssues: health.TrendStable,
			StaleIssues:   health.TrendDown,
			BloatedFiles:  health.TrendUp,
			FixFeatRatio:  health.TrendUp,
		},
		Alerts: []health.Alert{
			{Metric: "fix_feat_ratio", Message: "Fix:feat ratio is 1.5", Level: "warn"},
		},
	}

	// Should not panic
	err := outputHealthText(report)
	if err != nil {
		t.Errorf("outputHealthText failed: %v", err)
	}
}

func TestOutputHealthJSON(t *testing.T) {
	report := health.Report{
		Current: health.Snapshot{
			OpenIssues: 10,
		},
		Alerts: []health.Alert{},
	}

	// Should not panic
	err := outputHealthJSON(report)
	if err != nil {
		t.Errorf("outputHealthJSON failed: %v", err)
	}
}
