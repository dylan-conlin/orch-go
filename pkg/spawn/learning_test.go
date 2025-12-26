package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGapTrackerRecordGap(t *testing.T) {
	tracker := &GapTracker{Events: []GapEvent{}}

	// Create a gap analysis with gaps
	analysis := &GapAnalysis{
		HasGaps:        true,
		ContextQuality: 15,
		Query:          "test query",
		Gaps: []Gap{
			{
				Type:        GapTypeNoContext,
				Severity:    GapSeverityCritical,
				Description: "No context found",
			},
		},
	}

	tracker.RecordGap(analysis, "investigation", "test task")

	if len(tracker.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(tracker.Events))
	}

	event := tracker.Events[0]
	if event.Query != "test query" {
		t.Errorf("expected query 'test query', got %q", event.Query)
	}
	if event.Skill != "investigation" {
		t.Errorf("expected skill 'investigation', got %q", event.Skill)
	}
	if event.GapType != string(GapTypeNoContext) {
		t.Errorf("expected gap type %q, got %q", GapTypeNoContext, event.GapType)
	}
	if event.Severity != string(GapSeverityCritical) {
		t.Errorf("expected severity %q, got %q", GapSeverityCritical, event.Severity)
	}
}

func TestGapTrackerRecordGapNoGaps(t *testing.T) {
	tracker := &GapTracker{Events: []GapEvent{}}

	// Analysis with no gaps
	analysis := &GapAnalysis{
		HasGaps:        false,
		ContextQuality: 85,
		Query:          "test query",
		Gaps:           []Gap{},
	}

	tracker.RecordGap(analysis, "investigation", "test task")

	if len(tracker.Events) != 0 {
		t.Errorf("expected 0 events for no-gap analysis, got %d", len(tracker.Events))
	}
}

func TestGapTrackerRecordResolution(t *testing.T) {
	tracker := &GapTracker{
		Events: []GapEvent{
			{
				Timestamp: time.Now().Add(-time.Hour),
				Query:     "test query", // Same query, older event
			},
			{
				Timestamp: time.Now(),
				Query:     "test query", // Same query, newer event
			},
			{
				Timestamp: time.Now(),
				Query:     "different query", // Different query, should not be affected
			},
		},
	}

	tracker.RecordResolution("test query", "added_knowledge", "kn decide added")

	// Should update ALL matching events, not just the most recent
	if tracker.Events[0].Resolution != "added_knowledge" {
		t.Errorf("expected older event resolution 'added_knowledge', got %q", tracker.Events[0].Resolution)
	}
	if tracker.Events[0].ResolutionDetails != "kn decide added" {
		t.Errorf("expected older event resolution details, got %q", tracker.Events[0].ResolutionDetails)
	}
	if tracker.Events[1].Resolution != "added_knowledge" {
		t.Errorf("expected newer event resolution 'added_knowledge', got %q", tracker.Events[1].Resolution)
	}
	if tracker.Events[1].ResolutionDetails != "kn decide added" {
		t.Errorf("expected newer event resolution details, got %q", tracker.Events[1].ResolutionDetails)
	}
	// Different query event should be unchanged
	if tracker.Events[2].Resolution != "" {
		t.Errorf("expected different query event to have empty resolution, got %q", tracker.Events[2].Resolution)
	}
}

func TestGapTrackerFindRecurringGaps(t *testing.T) {
	tracker := &GapTracker{Events: []GapEvent{}}

	// Add 3 events for same query (meets threshold)
	for i := 0; i < 3; i++ {
		tracker.Events = append(tracker.Events, GapEvent{
			Timestamp:      time.Now(),
			Query:          "recurring gap",
			GapType:        string(GapTypeNoContext),
			Severity:       string(GapSeverityCritical),
			ContextQuality: 0,
		})
	}

	// Add 2 events for different query (below threshold)
	for i := 0; i < 2; i++ {
		tracker.Events = append(tracker.Events, GapEvent{
			Timestamp:      time.Now(),
			Query:          "infrequent gap",
			GapType:        string(GapTypeSparseContext),
			Severity:       string(GapSeverityWarning),
			ContextQuality: 25,
		})
	}

	suggestions := tracker.FindRecurringGaps()

	if len(suggestions) != 1 {
		t.Errorf("expected 1 recurring gap suggestion, got %d", len(suggestions))
	}

	if len(suggestions) > 0 {
		s := suggestions[0]
		if s.Query != "recurring gap" {
			t.Errorf("expected query 'recurring gap', got %q", s.Query)
		}
		if s.Count != 3 {
			t.Errorf("expected count 3, got %d", s.Count)
		}
		if s.Priority != "high" {
			t.Errorf("expected priority 'high' for critical gaps, got %q", s.Priority)
		}
	}
}

func TestGapTrackerFindRecurringGapsExcludesResolved(t *testing.T) {
	tracker := &GapTracker{Events: []GapEvent{}}

	// Add 4 events for same query, but resolve them all
	for i := 0; i < 4; i++ {
		tracker.Events = append(tracker.Events, GapEvent{
			Timestamp:         time.Now(),
			Query:             "resolved gap",
			GapType:           string(GapTypeNoContext),
			Severity:          string(GapSeverityCritical),
			ContextQuality:    0,
			Resolution:        "added_knowledge",
			ResolutionDetails: "fixed it",
		})
	}

	// Add 3 unresolved events for different query
	for i := 0; i < 3; i++ {
		tracker.Events = append(tracker.Events, GapEvent{
			Timestamp:      time.Now(),
			Query:          "unresolved gap",
			GapType:        string(GapTypeNoContext),
			Severity:       string(GapSeverityCritical),
			ContextQuality: 0,
			// No Resolution set
		})
	}

	suggestions := tracker.FindRecurringGaps()

	// Should only find the unresolved gap, not the resolved one
	if len(suggestions) != 1 {
		t.Errorf("expected 1 recurring gap suggestion, got %d", len(suggestions))
	}

	if len(suggestions) > 0 {
		s := suggestions[0]
		if s.Query != "unresolved gap" {
			t.Errorf("expected query 'unresolved gap', got %q", s.Query)
		}
	}
}

func TestGapTrackerRecordResolutionRemovesFromSuggestions(t *testing.T) {
	tracker := &GapTracker{Events: []GapEvent{}}

	// Add 3 events for same query (meets threshold)
	for i := 0; i < 3; i++ {
		tracker.Events = append(tracker.Events, GapEvent{
			Timestamp:      time.Now(),
			Query:          "test gap",
			GapType:        string(GapTypeNoContext),
			Severity:       string(GapSeverityCritical),
			ContextQuality: 0,
		})
	}

	// Verify gap appears in suggestions before resolution
	suggestions := tracker.FindRecurringGaps()
	if len(suggestions) != 1 {
		t.Errorf("before resolution: expected 1 suggestion, got %d", len(suggestions))
	}

	// Resolve the gap
	tracker.RecordResolution("test gap", "added_knowledge", "kn entry added")

	// Verify gap no longer appears in suggestions
	suggestions = tracker.FindRecurringGaps()
	if len(suggestions) != 0 {
		t.Errorf("after resolution: expected 0 suggestions, got %d", len(suggestions))
	}

	// Verify all events were marked as resolved
	for i, e := range tracker.Events {
		if e.Resolution != "added_knowledge" {
			t.Errorf("event %d: expected resolution 'added_knowledge', got %q", i, e.Resolution)
		}
	}
}

func TestGapTrackerAnalyzePatterns(t *testing.T) {
	now := time.Now()
	tracker := &GapTracker{
		Events: []GapEvent{
			{Timestamp: now, Query: "auth", Skill: "feature-impl", Severity: string(GapSeverityCritical), ContextQuality: 10},
			{Timestamp: now.Add(-2 * 24 * time.Hour), Query: "auth", Skill: "investigation", Severity: string(GapSeverityWarning), ContextQuality: 30},
			{Timestamp: now.Add(-10 * 24 * time.Hour), Query: "auth", Skill: "feature-impl", Severity: string(GapSeverityInfo), ContextQuality: 40},
		},
	}

	analyses := tracker.AnalyzePatterns()

	if len(analyses) != 1 {
		t.Errorf("expected 1 topic analysis, got %d", len(analyses))
	}

	if len(analyses) > 0 {
		a := analyses[0]
		if a.Topic != "auth" {
			t.Errorf("expected topic 'auth', got %q", a.Topic)
		}
		if a.TotalGaps != 3 {
			t.Errorf("expected 3 total gaps, got %d", a.TotalGaps)
		}
		if a.RecentGaps != 2 {
			t.Errorf("expected 2 recent gaps, got %d", a.RecentGaps)
		}
		if a.CriticalGaps != 1 {
			t.Errorf("expected 1 critical gap, got %d", a.CriticalGaps)
		}
		if len(a.Skills) != 2 {
			t.Errorf("expected 2 skills, got %d", len(a.Skills))
		}
	}
}

func TestGapTrackerGetSkillGapRates(t *testing.T) {
	tracker := &GapTracker{
		Events: []GapEvent{
			{Skill: "feature-impl"},
			{Skill: "feature-impl"},
			{Skill: "investigation"},
			{Skill: ""},
		},
	}

	rates := tracker.GetSkillGapRates()

	if rates["feature-impl"] != 2 {
		t.Errorf("expected feature-impl rate 2, got %d", rates["feature-impl"])
	}
	if rates["investigation"] != 1 {
		t.Errorf("expected investigation rate 1, got %d", rates["investigation"])
	}
	if _, exists := rates[""]; exists {
		t.Error("expected empty skill to not be in rates")
	}
}

func TestGapTrackerRecordImprovement(t *testing.T) {
	tracker := &GapTracker{
		Events: []GapEvent{
			{Query: "auth"},
			{Query: "auth"},
		},
	}

	tracker.RecordImprovement("kn_entry", "auth", "kn-123")

	if len(tracker.Improvements) != 1 {
		t.Errorf("expected 1 improvement, got %d", len(tracker.Improvements))
	}

	imp := tracker.Improvements[0]
	if imp.Type != "kn_entry" {
		t.Errorf("expected type 'kn_entry', got %q", imp.Type)
	}
	if imp.Query != "auth" {
		t.Errorf("expected query 'auth', got %q", imp.Query)
	}
	if imp.Reference != "kn-123" {
		t.Errorf("expected reference 'kn-123', got %q", imp.Reference)
	}
	if imp.GapCountBefore != 2 {
		t.Errorf("expected gap count before 2, got %d", imp.GapCountBefore)
	}
}

func TestGapTrackerMeasureImprovementEffectiveness(t *testing.T) {
	now := time.Now()
	tracker := &GapTracker{
		Events: []GapEvent{
			{Timestamp: now.Add(-2 * time.Hour), Query: "auth"},    // Before improvement
			{Timestamp: now.Add(-time.Hour), Query: "auth"},        // Before improvement
			{Timestamp: now.Add(time.Hour), Query: "auth"},         // After improvement
			{Timestamp: now.Add(2 * time.Hour), Query: "database"}, // Different topic
		},
		Improvements: []ImprovementRecord{
			{
				Timestamp:      now,
				Type:           "kn_entry",
				Query:          "auth",
				Reference:      "kn-123",
				GapCountBefore: 2,
			},
		},
	}

	results := tracker.MeasureImprovementEffectiveness()

	if len(results) != 1 {
		t.Errorf("expected 1 improvement result, got %d", len(results))
	}

	if results[0].GapCountAfter != 1 {
		t.Errorf("expected 1 gap after improvement, got %d", results[0].GapCountAfter)
	}
}

func TestGapTrackerSaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override trackerPathFunc for testing
	testPath := filepath.Join(tmpDir, "test-tracker.json")
	originalFunc := trackerPathFunc
	trackerPathFunc = func() string { return testPath }
	defer func() { trackerPathFunc = originalFunc }()

	// Create and save tracker
	tracker := &GapTracker{
		Events: []GapEvent{
			{
				Timestamp:      time.Now().UTC(),
				Query:          "test query",
				GapType:        string(GapTypeNoContext),
				Severity:       string(GapSeverityCritical),
				Skill:          "investigation",
				Task:           "test task",
				ContextQuality: 0,
			},
		},
	}

	if err := tracker.Save(); err != nil {
		t.Fatalf("failed to save tracker: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Error("tracker file was not created")
	}

	// Load and verify
	loaded, err := LoadTracker()
	if err != nil {
		t.Fatalf("failed to load tracker: %v", err)
	}

	if len(loaded.Events) != 1 {
		t.Errorf("expected 1 event after load, got %d", len(loaded.Events))
	}

	if loaded.Events[0].Query != "test query" {
		t.Errorf("expected query 'test query', got %q", loaded.Events[0].Query)
	}
}

func TestGapTrackerPruneOldEvents(t *testing.T) {
	now := time.Now()
	tracker := &GapTracker{
		Events: []GapEvent{
			{Timestamp: now, Query: "recent"},
			{Timestamp: now.Add(-31 * 24 * time.Hour), Query: "old"}, // Older than max age
		},
	}

	tracker.pruneOldEvents()

	if len(tracker.Events) != 1 {
		t.Errorf("expected 1 event after pruning, got %d", len(tracker.Events))
	}

	if tracker.Events[0].Query != "recent" {
		t.Errorf("expected recent event to remain, got %q", tracker.Events[0].Query)
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Test Query", "test query"},
		{"  multiple   spaces  ", "multiple spaces"},
		{"UPPERCASE", "uppercase"},
		{"normal", "normal"},
	}

	for _, tc := range tests {
		result := normalizeQuery(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeQuery(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestDetermineSuggestion(t *testing.T) {
	tests := []struct {
		name         string
		events       []GapEvent
		expectedType string
		containsWord string
	}{
		{
			name: "no_context_gaps",
			events: []GapEvent{
				{GapType: string(GapTypeNoContext)},
			},
			expectedType: "add_knowledge",
			containsWord: "foundational",
		},
		{
			name: "no_constraints_gaps",
			events: []GapEvent{
				{GapType: string(GapTypeNoConstraints)},
			},
			expectedType: "add_knowledge",
			containsWord: "constraints",
		},
		{
			name: "no_decisions_gaps",
			events: []GapEvent{
				{GapType: string(GapTypeNoDecisions)},
			},
			expectedType: "create_issue",
			containsWord: "patterns",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suggestionType, suggestion, _ := determineSuggestion("test", tc.events)
			if suggestionType != tc.expectedType {
				t.Errorf("expected type %q, got %q", tc.expectedType, suggestionType)
			}
			if tc.containsWord != "" && !strings.Contains(strings.ToLower(suggestion), strings.ToLower(tc.containsWord)) {
				t.Errorf("expected suggestion to contain %q, got %q", tc.containsWord, suggestion)
			}
		})
	}
}

func TestDetermineTrend(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		events   []GapEvent
		expected string
	}{
		{
			name:     "insufficient_data",
			events:   []GapEvent{{Timestamp: now}, {Timestamp: now}},
			expected: "insufficient_data",
		},
		{
			name: "increasing",
			events: []GapEvent{
				{Timestamp: now},
				{Timestamp: now.Add(-time.Hour)},
				{Timestamp: now.Add(-2 * time.Hour)},
				{Timestamp: now.Add(-3 * time.Hour)},
				{Timestamp: now.Add(-10 * 24 * time.Hour)}, // Old event
			},
			expected: "increasing",
		},
		{
			name: "decreasing",
			events: []GapEvent{
				{Timestamp: now.Add(-10 * 24 * time.Hour)},
				{Timestamp: now.Add(-11 * 24 * time.Hour)},
				{Timestamp: now.Add(-12 * 24 * time.Hour)},
				{Timestamp: now.Add(-13 * 24 * time.Hour)},
			},
			expected: "decreasing",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := determineTrend(tc.events)
			if result != tc.expected {
				t.Errorf("expected trend %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestFormatSuggestions(t *testing.T) {
	suggestions := []LearningSuggestion{
		{
			Type:       "add_knowledge",
			Priority:   "high",
			Query:      "authentication",
			Count:      5,
			Suggestion: "Add foundational knowledge",
			Command:    `kn decide "auth" --reason "TODO"`,
		},
	}

	output := FormatSuggestions(suggestions)

	if output == "" {
		t.Error("expected non-empty output")
	}

	// Check for key elements
	expectedElements := []string{"LEARNING SUGGESTIONS", "authentication", "5x", "high"}
	for _, elem := range expectedElements {
		if !strings.Contains(strings.ToLower(output), strings.ToLower(elem)) {
			t.Errorf("expected output to contain %q", elem)
		}
	}
}

func TestFormatSuggestionsEmpty(t *testing.T) {
	output := FormatSuggestions([]LearningSuggestion{})

	if !strings.Contains(strings.ToLower(output), "no recurring gaps") {
		t.Errorf("expected empty message, got %q", output)
	}
}

func TestGapTrackerSummary(t *testing.T) {
	tracker := &GapTracker{Events: []GapEvent{}}

	summary := tracker.Summary()
	if summary != "No gaps tracked yet" {
		t.Errorf("expected empty summary, got %q", summary)
	}

	// Add some events
	tracker.Events = []GapEvent{
		{Query: "test"},
		{Query: "test"},
		{Query: "test"},
	}

	summary = tracker.Summary()
	if !strings.Contains(strings.ToLower(summary), "3 gap events") {
		t.Errorf("expected summary to contain event count, got %q", summary)
	}
	if !strings.Contains(strings.ToLower(summary), "1 recurring") {
		t.Errorf("expected summary to contain recurring count, got %q", summary)
	}
}

func TestLoadTrackerNoFile(t *testing.T) {
	// Use non-existent path
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "nonexistent", "tracker.json")
	originalFunc := trackerPathFunc
	trackerPathFunc = func() string { return testPath }
	defer func() { trackerPathFunc = originalFunc }()

	tracker, err := LoadTracker()
	if err != nil {
		t.Errorf("expected no error for missing file, got %v", err)
	}
	if tracker == nil {
		t.Error("expected empty tracker, got nil")
	}
	if len(tracker.Events) != 0 {
		t.Errorf("expected 0 events for new tracker, got %d", len(tracker.Events))
	}
}

func TestParseShellCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "simple_command",
			input:    "echo hello",
			expected: []string{"echo", "hello"},
		},
		{
			name:     "double_quoted_string",
			input:    `kn decide "auth" --reason "test reason"`,
			expected: []string{"kn", "decide", "auth", "--reason", "test reason"},
		},
		{
			name:     "single_quoted_string",
			input:    `kn decide 'auth config' --reason 'test reason'`,
			expected: []string{"kn", "decide", "auth config", "--reason", "test reason"},
		},
		{
			name:     "mixed_quotes",
			input:    `kn decide "auth" --reason 'test reason with "quotes"'`,
			expected: []string{"kn", "decide", "auth", "--reason", `test reason with "quotes"`},
		},
		{
			name:     "complex_reason_with_colons",
			input:    `kn decide "auth" --reason "Used by: investigation, feature-impl. Occurred 5 times"`,
			expected: []string{"kn", "decide", "auth", "--reason", "Used by: investigation, feature-impl. Occurred 5 times"},
		},
		{
			name:     "bd_create_command",
			input:    `bd create "Establish patterns for auth" -d "Used by: investigation. Occurred 3 times. Tasks: analyze flow"`,
			expected: []string{"bd", "create", "Establish patterns for auth", "-d", "Used by: investigation. Occurred 3 times. Tasks: analyze flow"},
		},
		{
			name:     "orch_spawn_command",
			input:    `orch spawn investigation "why does auth lack context"`,
			expected: []string{"orch", "spawn", "investigation", "why does auth lack context"},
		},
		{
			name:     "multiple_spaces",
			input:    "echo    hello    world",
			expected: []string{"echo", "hello", "world"},
		},
		{
			name:     "tabs_as_separators",
			input:    "echo\thello\tworld",
			expected: []string{"echo", "hello", "world"},
		},
		{
			name:    "empty_command",
			input:   "",
			wantErr: true,
		},
		{
			name:    "only_whitespace",
			input:   "   \t   ",
			wantErr: true,
		},
		{
			name:    "unterminated_double_quote",
			input:   `echo "hello`,
			wantErr: true,
		},
		{
			name:    "unterminated_single_quote",
			input:   `echo 'hello`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseShellCommand(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q, got none", tc.input)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tc.input, err)
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("length mismatch for %q: got %d, expected %d\n  got: %v\n  expected: %v",
					tc.input, len(result), len(tc.expected), result, tc.expected)
				return
			}

			for i, arg := range result {
				if arg != tc.expected[i] {
					t.Errorf("argument %d mismatch for %q: got %q, expected %q",
						i, tc.input, arg, tc.expected[i])
				}
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid commands
		{
			name:    "valid_kn_decide",
			input:   `kn decide "auth" --reason "This is a valid reason with 20+ characters"`,
			wantErr: false,
		},
		{
			name:    "valid_kn_constrain",
			input:   `kn constrain "no magic numbers" --reason "Code quality requires consistent style"`,
			wantErr: false,
		},
		{
			name:    "valid_kn_question",
			input:   `kn question "should we use JWT?"`,
			wantErr: false,
		},
		{
			name:    "valid_bd_create",
			input:   `bd create "Fix auth bug" -d "description"`,
			wantErr: false,
		},
		{
			name:    "valid_orch_spawn",
			input:   `orch spawn investigation "analyze auth"`,
			wantErr: false,
		},
		{
			name:    "unknown_command_valid",
			input:   `echo "hello world"`,
			wantErr: false,
		},

		// Invalid commands
		{
			name:    "empty_command",
			input:   "",
			wantErr: true,
		},
		{
			name:    "kn_no_subcommand",
			input:   "kn",
			wantErr: true,
		},
		{
			name:    "kn_decide_no_description",
			input:   "kn decide",
			wantErr: true,
		},
		{
			name:    "kn_decide_reason_no_value",
			input:   `kn decide "test" --reason`,
			wantErr: true,
		},
		{
			name:    "bd_no_subcommand",
			input:   "bd",
			wantErr: true,
		},
		{
			name:    "bd_create_no_title",
			input:   "bd create",
			wantErr: true,
		},
		{
			name:    "bd_create_d_no_value",
			input:   `bd create "title" -d`,
			wantErr: true,
		},
		{
			name:    "orch_no_subcommand",
			input:   "orch",
			wantErr: true,
		},
		{
			name:    "orch_spawn_no_skill",
			input:   "orch spawn",
			wantErr: true,
		},
		{
			name:    "orch_spawn_no_task",
			input:   "orch spawn investigation",
			wantErr: true,
		},
		{
			name:    "unterminated_quote",
			input:   `kn decide "test`,
			wantErr: true,
		},
		{
			name:    "kn_reason_too_short",
			input:   `kn decide "auth" --reason "short"`,
			wantErr: true,
		},
		{
			name:    "kn_reason_exactly_20_chars",
			input:   `kn decide "auth" --reason "exactly twenty chars"`,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCommand(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q, got none", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tc.input, err)
				}
			}
		})
	}
}

func TestDetermineSuggestionGeneratesValidCommands(t *testing.T) {
	// Test that all generated commands can be parsed and validated
	tests := []struct {
		name   string
		events []GapEvent
	}{
		{
			name: "no_context_gaps",
			events: []GapEvent{
				{GapType: string(GapTypeNoContext), Skill: "investigation", Task: "analyze auth"},
			},
		},
		{
			name: "no_constraints_gaps",
			events: []GapEvent{
				{GapType: string(GapTypeNoConstraints), Skill: "feature-impl", Task: "add feature"},
			},
		},
		{
			name: "no_decisions_gaps",
			events: []GapEvent{
				{GapType: string(GapTypeNoDecisions), Skill: "architect", Task: "design system"},
			},
		},
		{
			name: "default_gaps",
			events: []GapEvent{
				{GapType: string(GapTypeSparseContext), Skill: "investigation"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, command := determineSuggestion("test query", tc.events)

			if command == "" {
				t.Error("expected non-empty command")
				return
			}

			// Command should be parseable
			parts, err := ParseShellCommand(command)
			if err != nil {
				t.Errorf("generated command should be parseable: %q, error: %v", command, err)
				return
			}

			// Command should be valid
			if err := ValidateCommand(command); err != nil {
				t.Errorf("generated command should be valid: %q, error: %v", command, err)
				return
			}

			// Command should have at least 3 parts (executable, subcommand, arg)
			if len(parts) < 3 {
				t.Errorf("command should have at least 3 parts: %q, got %d parts", command, len(parts))
			}
		})
	}
}

func TestGenerateReasonFromGaps(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		events       []GapEvent
		wantContains []string
		wantNotEmpty bool
	}{
		{
			name:         "empty_events",
			query:        "auth",
			events:       []GapEvent{},
			wantContains: []string{"No context available"},
		},
		{
			name:  "with_skill",
			query: "auth",
			events: []GapEvent{
				{Skill: "investigation", Task: "analyze auth flow"},
			},
			wantContains: []string{"Used by: investigation", "Occurred 1 times"},
		},
		{
			name:  "with_multiple_skills",
			query: "database",
			events: []GapEvent{
				{Skill: "investigation", Task: "check db schema"},
				{Skill: "feature-impl", Task: "add db migration"},
				{Skill: "investigation", Task: "another task"},
			},
			wantContains: []string{"feature-impl", "investigation", "Occurred 3 times"},
		},
		{
			name:  "with_tasks",
			query: "config",
			events: []GapEvent{
				{Task: "update config parser"},
				{Task: "fix config loading"},
			},
			wantContains: []string{"Tasks:", "Occurred 2 times"},
		},
		{
			name:  "long_task_truncated",
			query: "api",
			events: []GapEvent{
				{Task: "this is a very long task description that should be truncated to prevent overly long output"},
			},
			wantContains: []string{"..."},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := generateReasonFromGaps(tc.query, tc.events)

			if result == "" {
				t.Error("expected non-empty result")
			}

			for _, want := range tc.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got %q", want, result)
				}
			}
		})
	}
}

func TestGenerateReasonFromGapsMinLength(t *testing.T) {
	// Test that generateReasonFromGaps always produces at least MinReasonLength chars
	tests := []struct {
		name   string
		query  string
		events []GapEvent
	}{
		{
			name:   "no_skill_no_task_3_events",
			query:  "auth",
			events: []GapEvent{{}, {}, {}},
		},
		{
			name:   "no_skill_no_task_1_event",
			query:  "x",
			events: []GapEvent{{}},
		},
		{
			name:   "short_query",
			query:  "a",
			events: []GapEvent{{}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := generateReasonFromGaps(tc.query, tc.events)

			if len(result) < MinReasonLength {
				t.Errorf("expected reason length >= %d, got %d: %q", MinReasonLength, len(result), result)
			}
		})
	}
}
