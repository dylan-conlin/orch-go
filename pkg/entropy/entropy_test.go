package entropy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestClassifyCommits(t *testing.T) {
	commits := []commitEntry{
		{message: "fix: resolve crash in spawn", timestamp: time.Now()},
		{message: "feat: add entropy command", timestamp: time.Now()},
		{message: "fix: correct hotspot threshold", timestamp: time.Now()},
		{message: "refactor: extract utility", timestamp: time.Now()},
		{message: "feat: new dashboard panel", timestamp: time.Now()},
		{message: "chore: update deps", timestamp: time.Now()},
	}

	result := classifyCommits(commits)

	if result.Fixes != 2 {
		t.Errorf("expected 2 fixes, got %d", result.Fixes)
	}
	if result.Features != 2 {
		t.Errorf("expected 2 features, got %d", result.Features)
	}
	if result.Other != 2 {
		t.Errorf("expected 2 other, got %d", result.Other)
	}
	if result.Total != 6 {
		t.Errorf("expected 6 total, got %d", result.Total)
	}
}

func TestFixFeatRatio(t *testing.T) {
	tests := []struct {
		name     string
		fixes    int
		features int
		want     float64
	}{
		{"healthy", 1, 3, 0.333},
		{"degrading", 3, 4, 0.75},
		{"spiral", 5, 3, 1.667},
		{"no features", 5, 0, 0}, // guard: no division by zero
		{"no commits", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &CommitClassification{Fixes: tt.fixes, Features: tt.features, Total: tt.fixes + tt.features}
			ratio := cc.FixFeatRatio()
			// Allow 0.01 tolerance
			if diff := ratio - tt.want; diff > 0.01 || diff < -0.01 {
				t.Errorf("FixFeatRatio() = %.3f, want %.3f", ratio, tt.want)
			}
		})
	}
}

func TestVelocity(t *testing.T) {
	now := time.Now()
	commits := []commitEntry{
		{message: "fix: a", timestamp: now},
		{message: "feat: b", timestamp: now.Add(-24 * time.Hour)},
		{message: "feat: c", timestamp: now.Add(-48 * time.Hour)},
	}

	v := calculateVelocity(commits, 7)
	// 3 commits over 7 days = ~0.43/day
	if v.CommitsPerDay < 0.4 || v.CommitsPerDay > 0.5 {
		t.Errorf("expected ~0.43 commits/day, got %.2f", v.CommitsPerDay)
	}
	if v.WindowDays != 7 {
		t.Errorf("expected 7 day window, got %d", v.WindowDays)
	}
}

func TestHealthLevel(t *testing.T) {
	tests := []struct {
		ratio float64
		want  string
	}{
		{0.2, "healthy"},
		{0.5, "degrading"},
		{0.8, "degrading"},
		{1.0, "spiral"},
		{1.5, "spiral"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := healthLevel(tt.ratio)
			if got != tt.want {
				t.Errorf("healthLevel(%.1f) = %s, want %s", tt.ratio, got, tt.want)
			}
		})
	}
}

func TestCountBloatedFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a small file
	os.WriteFile(filepath.Join(dir, "small.go"), make([]byte, 100), 0644)

	// Create a large file (>800 lines)
	var lines []byte
	for i := 0; i < 900; i++ {
		lines = append(lines, []byte("line content here\n")...)
	}
	os.WriteFile(filepath.Join(dir, "big.go"), lines, 0644)

	count, files := countBloatedFiles(dir, 800)
	if count != 1 {
		t.Errorf("expected 1 bloated file, got %d", count)
	}
	if len(files) != 1 || files[0].Path != "big.go" {
		t.Errorf("expected big.go in bloated files, got %v", files)
	}
}

func TestAggregateEventsFromFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")

	now := time.Now()
	logger := events.NewLogger(logPath)

	// Log some events
	logger.Log(events.Event{
		Type:      events.EventTypeSessionSpawned,
		Timestamp: now.Unix(),
		Data:      map[string]interface{}{"skill": "feature-impl"},
	})
	logger.Log(events.Event{
		Type:      events.EventTypeAgentCompleted,
		Timestamp: now.Unix(),
		Data:      map[string]interface{}{"outcome": "success"},
	})
	logger.Log(events.Event{
		Type:      events.EventTypeAgentAbandonedTelemetry,
		Timestamp: now.Unix(),
		Data:      map[string]interface{}{"reason": "stuck"},
	})
	logger.Log(events.Event{
		Type:      events.EventTypeVerificationBypassed,
		Timestamp: now.Unix(),
		Data:      map[string]interface{}{"gate": "test_evidence"},
	})

	stats := aggregateEvents(logPath, 7)
	if stats.Spawns != 1 {
		t.Errorf("expected 1 spawn, got %d", stats.Spawns)
	}
	if stats.Completions != 1 {
		t.Errorf("expected 1 completion, got %d", stats.Completions)
	}
	if stats.Abandonments != 1 {
		t.Errorf("expected 1 abandonment, got %d", stats.Abandonments)
	}
	if stats.Bypasses != 1 {
		t.Errorf("expected 1 bypass, got %d", stats.Bypasses)
	}
}

func TestGenerateRecommendations(t *testing.T) {
	// Spiral-level report
	report := &Report{
		CommitClassification: CommitClassification{Fixes: 10, Features: 5, Total: 15},
		Velocity:             Velocity{CommitsPerDay: 50},
		BloatedFileCount:     5,
		EventStats:           EventStats{Bypasses: 10, Abandonments: 8, Spawns: 10},
	}

	recs := generateRecommendations(report)
	if len(recs) == 0 {
		t.Error("expected recommendations for unhealthy report")
	}

	// Check that high fix:feat ratio triggers a recommendation
	found := false
	for _, r := range recs {
		if r.Severity == "critical" {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one critical recommendation for spiral-level metrics")
	}
}

func TestReportJSON(t *testing.T) {
	report := &Report{
		GeneratedAt:          time.Now(),
		WindowDays:           28,
		CommitClassification: CommitClassification{Fixes: 3, Features: 7, Total: 10},
		HealthLevel:          "healthy",
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["health_level"] != "healthy" {
		t.Errorf("expected health_level=healthy, got %v", parsed["health_level"])
	}
}
