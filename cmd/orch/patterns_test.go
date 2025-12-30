package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/action"
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

func TestCollectActionPatterns(t *testing.T) {
	// Create a temp file for action log
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "action-log.jsonl")

	// Override the default log path for testing
	originalFunc := action.GetLoggerPathFunc()
	action.SetLoggerPathFunc(func() string { return testPath })
	defer action.SetLoggerPathFunc(originalFunc)

	// Create a logger and add some test events with futile patterns
	logger := action.NewLogger(testPath)
	now := time.Now()

	// Create a pattern of repeated empty reads (futile action - 5 occurrences)
	for i := 0; i < 5; i++ {
		err := logger.Log(action.ActionEvent{
			Tool:      "Read",
			Target:    "/path/to/SYNTHESIS.md",
			Outcome:   action.OutcomeEmpty,
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
			Workspace: "test-workspace",
		})
		if err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	// Create a pattern of repeated errors (3 occurrences - warning level)
	for i := 0; i < 3; i++ {
		err := logger.Log(action.ActionEvent{
			Tool:         "Bash",
			Target:       "git status",
			Outcome:      action.OutcomeError,
			ErrorMessage: "command failed",
			Timestamp:    now.Add(-time.Duration(i) * time.Minute),
		})
		if err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	// Create events below threshold (only 2 - should not appear)
	for i := 0; i < 2; i++ {
		err := logger.Log(action.ActionEvent{
			Tool:           "Read",
			Target:         "/path/to/other.go",
			Outcome:        action.OutcomeFallback,
			FallbackAction: "used alternative",
			Timestamp:      now.Add(-time.Duration(i) * time.Minute),
		})
		if err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	// Add success events (should never appear as patterns)
	for i := 0; i < 10; i++ {
		err := logger.Log(action.ActionEvent{
			Tool:      "Read",
			Target:    "/path/to/good.go",
			Outcome:   action.OutcomeSuccess,
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
		})
		if err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	// Collect action patterns
	patterns, err := collectActionPatterns()
	if err != nil {
		t.Fatalf("collectActionPatterns failed: %v", err)
	}

	// Should find 2 patterns (5x empty and 3x error)
	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(patterns))
		for _, p := range patterns {
			t.Logf("Pattern: %s (count=%d)", p.Title, p.Count)
		}
	}

	// Find and verify the empty result pattern
	var foundEmpty bool
	for _, p := range patterns {
		if p.Type == PatternTypeFutileAction && p.Count == 5 {
			foundEmpty = true
			if p.Severity != PatternSeverityCritical {
				t.Errorf("Expected critical severity for 5-count pattern, got %s", p.Severity)
			}
			if p.Suggestion == "" {
				t.Error("Expected suggestion for futile action pattern")
			}
		}
	}
	if !foundEmpty {
		t.Error("Expected to find empty result pattern with count 5")
	}

	// Find and verify the error pattern
	var foundError bool
	for _, p := range patterns {
		if p.Type == PatternTypeFutileAction && p.Count == 3 {
			foundError = true
			if p.Severity != PatternSeverityWarning {
				t.Errorf("Expected warning severity for 3-count pattern, got %s", p.Severity)
			}
		}
	}
	if !foundError {
		t.Error("Expected to find error pattern with count 3")
	}
}

func TestCollectActionPatterns_Empty(t *testing.T) {
	// Create a temp file for action log (empty)
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "action-log.jsonl")

	// Override the default log path for testing
	originalFunc := action.GetLoggerPathFunc()
	action.SetLoggerPathFunc(func() string { return testPath })
	defer action.SetLoggerPathFunc(originalFunc)

	// Collect action patterns from empty log
	patterns, err := collectActionPatterns()
	if err != nil {
		t.Fatalf("collectActionPatterns failed: %v", err)
	}

	// Should find no patterns
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns for empty log, got %d", len(patterns))
	}
}

func TestFutileActionPatternType(t *testing.T) {
	// Verify PatternTypeFutileAction is properly defined
	if string(PatternTypeFutileAction) != "futile_action" {
		t.Errorf("Expected PatternTypeFutileAction to be 'futile_action', got %s", PatternTypeFutileAction)
	}
}

func TestParseSuppressDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		// Days
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		// Standard Go durations
		{"1h", time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"24h", 24 * time.Hour, false},
		// Invalid
		{"invalid", 0, true},
		{"d", 0, true},
		{"xd", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSuppressDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSuppressDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseSuppressDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
