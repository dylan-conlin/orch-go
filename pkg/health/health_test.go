package health

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotAppendAndRead(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "snapshots.jsonl"))

	snap := Snapshot{
		Timestamp:    time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
		OpenIssues:   43,
		BlockedIssues: 15,
		StaleIssues:  8,
		OrphanedIssues: 3,
		BloatedFiles: 5,
		FixFeatRatio: 1.5,
	}

	if err := store.Append(snap); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	snapshots, err := store.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(snapshots) != 1 {
		t.Fatalf("Expected 1 snapshot, got %d", len(snapshots))
	}

	if snapshots[0].OpenIssues != 43 {
		t.Errorf("Expected OpenIssues 43, got %d", snapshots[0].OpenIssues)
	}
	if snapshots[0].FixFeatRatio != 1.5 {
		t.Errorf("Expected FixFeatRatio 1.5, got %f", snapshots[0].FixFeatRatio)
	}
}

func TestMultipleSnapshots(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "snapshots.jsonl"))

	for i := 0; i < 5; i++ {
		snap := Snapshot{
			Timestamp:  time.Date(2026, 3, 1+i, 12, 0, 0, 0, time.UTC),
			OpenIssues: 40 + i*2,
			StaleIssues: i,
			BloatedFiles: 5 + i,
			FixFeatRatio: 1.0 + float64(i)*0.2,
		}
		if err := store.Append(snap); err != nil {
			t.Fatalf("Append %d failed: %v", i, err)
		}
	}

	snapshots, err := store.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(snapshots) != 5 {
		t.Fatalf("Expected 5 snapshots, got %d", len(snapshots))
	}
}

func TestReadRecentN(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "snapshots.jsonl"))

	for i := 0; i < 10; i++ {
		snap := Snapshot{
			Timestamp:  time.Date(2026, 3, 1, i, 0, 0, 0, time.UTC),
			OpenIssues: i * 10,
		}
		store.Append(snap)
	}

	recent, err := store.ReadRecent(3)
	if err != nil {
		t.Fatalf("ReadRecent failed: %v", err)
	}

	if len(recent) != 3 {
		t.Fatalf("Expected 3 snapshots, got %d", len(recent))
	}

	// Should be the last 3
	if recent[0].OpenIssues != 70 {
		t.Errorf("Expected first recent OpenIssues 70, got %d", recent[0].OpenIssues)
	}
	if recent[2].OpenIssues != 90 {
		t.Errorf("Expected last recent OpenIssues 90, got %d", recent[2].OpenIssues)
	}
}

func TestReadRecentMoreThanAvailable(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "snapshots.jsonl"))

	store.Append(Snapshot{Timestamp: time.Now(), OpenIssues: 10})

	recent, err := store.ReadRecent(5)
	if err != nil {
		t.Fatalf("ReadRecent failed: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("Expected 1 snapshot, got %d", len(recent))
	}
}

func TestReadEmptyStore(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "snapshots.jsonl"))

	snapshots, err := store.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll on empty store failed: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("Expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestTrendDirection(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected Trend
	}{
		{"increasing", []float64{1, 2, 3, 4, 5}, TrendUp},
		{"decreasing", []float64{5, 4, 3, 2, 1}, TrendDown},
		{"stable", []float64{3, 3, 3, 3, 3}, TrendStable},
		{"mostly increasing", []float64{1, 3, 2, 4, 5}, TrendUp},
		{"mostly decreasing", []float64{5, 3, 4, 2, 1}, TrendDown},
		{"single value", []float64{5}, TrendStable},
		{"empty", []float64{}, TrendStable},
		{"two values up", []float64{1, 5}, TrendUp},
		{"two values down", []float64{5, 1}, TrendDown},
		{"two values same", []float64{3, 3}, TrendStable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeTrend(tt.values)
			if result != tt.expected {
				t.Errorf("ComputeTrend(%v) = %v, want %v", tt.values, result, tt.expected)
			}
		})
	}
}

func TestHealthReport(t *testing.T) {
	snapshots := []Snapshot{
		{Timestamp: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), OpenIssues: 30, StaleIssues: 2, BloatedFiles: 3, FixFeatRatio: 0.8},
		{Timestamp: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), OpenIssues: 35, StaleIssues: 4, BloatedFiles: 4, FixFeatRatio: 1.2},
		{Timestamp: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), OpenIssues: 40, StaleIssues: 6, BloatedFiles: 5, FixFeatRatio: 1.8},
	}

	report := GenerateReport(snapshots)

	if report.Current.OpenIssues != 40 {
		t.Errorf("Expected current OpenIssues 40, got %d", report.Current.OpenIssues)
	}

	// OpenIssues trending up
	if report.Trends.OpenIssues != TrendUp {
		t.Errorf("Expected OpenIssues trend up, got %v", report.Trends.OpenIssues)
	}

	// StaleIssues trending up
	if report.Trends.StaleIssues != TrendUp {
		t.Errorf("Expected StaleIssues trend up, got %v", report.Trends.StaleIssues)
	}
}

func TestAlertGeneration(t *testing.T) {
	snapshots := []Snapshot{
		{Timestamp: time.Now().Add(-48 * time.Hour), StaleIssues: 2, FixFeatRatio: 0.5, BloatedFiles: 3},
		{Timestamp: time.Now().Add(-24 * time.Hour), StaleIssues: 5, FixFeatRatio: 1.5, BloatedFiles: 5},
		{Timestamp: time.Now(), StaleIssues: 10, FixFeatRatio: 2.5, BloatedFiles: 8},
	}

	report := GenerateReport(snapshots)
	alerts := report.Alerts

	if len(alerts) == 0 {
		t.Fatal("Expected alerts for degrading metrics")
	}

	// Should have alert for fix:feat ratio > 2.0
	hasRatioAlert := false
	for _, a := range alerts {
		if a.Metric == "fix_feat_ratio" {
			hasRatioAlert = true
		}
	}
	if !hasRatioAlert {
		t.Error("Expected alert for fix_feat_ratio > 2.0")
	}
}

func TestStoreFileCreation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "snapshots.jsonl")
	store := NewStore(path)

	snap := Snapshot{Timestamp: time.Now(), OpenIssues: 10}
	if err := store.Append(snap); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Expected snapshot file to be created")
	}
}
