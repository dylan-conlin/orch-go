package main

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestSortPatterns(t *testing.T) {
	patterns := []DetectedPattern{
		{Type: PatternTypeRecurringGap, Severity: PatternSeverityInfo, Title: "Info 1", Count: 3},
		{Type: PatternTypePersistentFailure, Severity: PatternSeverityCritical, Title: "Critical 1", Count: 5},
		{Type: PatternTypeRetry, Severity: PatternSeverityWarning, Title: "Warning 1", Count: 2},
		{Type: PatternTypeRecurringGap, Severity: PatternSeverityInfo, Title: "Info 2", Count: 10},
		{Type: PatternTypeEmptyContext, Severity: PatternSeverityCritical, Title: "Critical 2", Count: 3},
	}

	sortPatterns(patterns)

	// Critical should be first
	if patterns[0].Severity != PatternSeverityCritical {
		t.Errorf("Expected first pattern to be critical, got %s", patterns[0].Severity)
	}
	if patterns[1].Severity != PatternSeverityCritical {
		t.Errorf("Expected second pattern to be critical, got %s", patterns[1].Severity)
	}

	// Within critical, higher count should be first
	if patterns[0].Count < patterns[1].Count {
		t.Errorf("Expected critical patterns sorted by count (higher first)")
	}

	// Warning should be after critical
	if patterns[2].Severity != PatternSeverityWarning {
		t.Errorf("Expected third pattern to be warning, got %s", patterns[2].Severity)
	}

	// Info should be last
	if patterns[3].Severity != PatternSeverityInfo || patterns[4].Severity != PatternSeverityInfo {
		t.Errorf("Expected info patterns to be last")
	}

	// Within info, higher count should be first
	if patterns[3].Count < patterns[4].Count {
		t.Errorf("Expected info patterns sorted by count (higher first)")
	}
}

func TestPatternType(t *testing.T) {
	tests := []struct {
		patternType PatternType
		expected    string
	}{
		{PatternTypeRetry, "retry"},
		{PatternTypePersistentFailure, "persistent_failure"},
		{PatternTypeEmptyContext, "empty_context"},
		{PatternTypeRecurringGap, "recurring_gap"},
		{PatternTypeContextDrift, "context_drift"},
	}

	for _, tt := range tests {
		if string(tt.patternType) != tt.expected {
			t.Errorf("PatternType %v expected %s, got %s", tt.patternType, tt.expected, string(tt.patternType))
		}
	}
}

func TestPatternSeverity(t *testing.T) {
	tests := []struct {
		severity PatternSeverity
		expected string
	}{
		{PatternSeverityCritical, "critical"},
		{PatternSeverityWarning, "warning"},
		{PatternSeverityInfo, "info"},
	}

	for _, tt := range tests {
		if string(tt.severity) != tt.expected {
			t.Errorf("PatternSeverity %v expected %s, got %s", tt.severity, tt.expected, string(tt.severity))
		}
	}
}

func TestCollectGapPatterns(t *testing.T) {
	// Mock the tracker path for testing
	originalFunc := spawn.TrackerPathFunc()
	defer func() { spawn.SetTrackerPathFunc(func() string { return originalFunc }) }()

	// Create a temp file with test data
	tempDir := t.TempDir()
	testPath := tempDir + "/gap-tracker.json"

	// Write test tracker data
	spawn.SetTrackerPathFunc(func() string { return testPath })

	// Create a tracker with test events
	tracker := &spawn.GapTracker{
		Events: []spawn.GapEvent{
			{
				Timestamp:      time.Now().Add(-1 * time.Hour),
				Query:          "test query",
				GapType:        string(spawn.GapTypeNoContext),
				Severity:       string(spawn.GapSeverityCritical),
				Skill:          "investigation",
				ContextQuality: 0,
			},
			{
				Timestamp:      time.Now().Add(-2 * time.Hour),
				Query:          "test query",
				GapType:        string(spawn.GapTypeNoContext),
				Severity:       string(spawn.GapSeverityCritical),
				Skill:          "investigation",
				ContextQuality: 0,
			},
			{
				Timestamp:      time.Now().Add(-3 * time.Hour),
				Query:          "test query",
				GapType:        string(spawn.GapTypeNoContext),
				Severity:       string(spawn.GapSeverityCritical),
				Skill:          "investigation",
				ContextQuality: 0,
			},
		},
	}

	if err := tracker.Save(); err != nil {
		t.Fatalf("Failed to save test tracker: %v", err)
	}

	// Collect patterns
	patterns, err := collectGapPatterns()
	if err != nil {
		t.Fatalf("collectGapPatterns failed: %v", err)
	}

	// Should have at least one pattern (the recurring "test query" gap)
	if len(patterns) == 0 {
		t.Errorf("Expected at least one pattern, got none")
	}

	// Find the pattern for "test query"
	var found bool
	for _, p := range patterns {
		if p.Query == "test query" {
			found = true
			if p.Type != PatternTypeEmptyContext {
				t.Errorf("Expected PatternTypeEmptyContext, got %s", p.Type)
			}
			if p.Severity != PatternSeverityCritical {
				t.Errorf("Expected PatternSeverityCritical, got %s", p.Severity)
			}
			if p.Count != 3 {
				t.Errorf("Expected count 3, got %d", p.Count)
			}
		}
	}

	if !found {
		t.Errorf("Expected to find pattern for 'test query'")
	}
}

func TestDetectedPatternFields(t *testing.T) {
	now := time.Now()
	pattern := DetectedPattern{
		Type:        PatternTypePersistentFailure,
		Severity:    PatternSeverityCritical,
		Title:       "Test Pattern",
		Description: "Test description",
		Count:       5,
		Query:       "test",
		BeadsID:     "test-123",
		Suggestion:  "Do something",
		Details:     []string{"detail 1", "detail 2"},
		FirstSeen:   now.Add(-24 * time.Hour),
		LastSeen:    now,
	}

	if pattern.Type != PatternTypePersistentFailure {
		t.Errorf("Expected persistent_failure type")
	}
	if pattern.Severity != PatternSeverityCritical {
		t.Errorf("Expected critical severity")
	}
	if pattern.Count != 5 {
		t.Errorf("Expected count 5, got %d", pattern.Count)
	}
	if len(pattern.Details) != 2 {
		t.Errorf("Expected 2 details, got %d", len(pattern.Details))
	}
}
