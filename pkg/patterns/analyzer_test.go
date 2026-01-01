package patterns

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestActionOutcomeConstants(t *testing.T) {
	// Verify outcome constants are distinct
	outcomes := []ActionOutcome{OutcomeSuccess, OutcomeEmpty, OutcomeError, OutcomeTimeout}
	seen := make(map[ActionOutcome]bool)
	for _, o := range outcomes {
		if seen[o] {
			t.Errorf("Duplicate outcome constant: %s", o)
		}
		seen[o] = true
	}
}

func TestActionLogRecordAction(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	event := ActionEvent{
		Tool:    "Read",
		Target:  "/path/to/file.md",
		Outcome: OutcomeSuccess,
	}

	log.RecordAction(event)

	if len(log.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(log.Events))
	}

	// Should have set timestamp
	if log.Events[0].Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestActionLogRecordActionPreservesTimestamp(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	customTime := time.Date(2025, 12, 27, 10, 0, 0, 0, time.UTC)
	event := ActionEvent{
		Timestamp: customTime,
		Tool:      "Read",
		Target:    "/path/to/file.md",
		Outcome:   OutcomeSuccess,
	}

	log.RecordAction(event)

	if !log.Events[0].Timestamp.Equal(customTime) {
		t.Errorf("Expected timestamp %v, got %v", customTime, log.Events[0].Timestamp)
	}
}

func TestDetectRepeatedEmptyReads_BelowThreshold(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add 2 empty reads (below threshold of 3)
	for i := 0; i < 2; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/workspace/SYNTHESIS.md",
			Outcome: OutcomeEmpty,
		})
	}

	patterns := log.DetectPatterns()

	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns below threshold, got %d", len(patterns))
	}
}

func TestDetectRepeatedEmptyReads_AtThreshold(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add exactly threshold number of empty reads
	for i := 0; i < RepetitionThreshold; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/workspace/SYNTHESIS.md",
			Outcome: OutcomeEmpty,
		})
	}

	patterns := log.DetectPatterns()

	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern at threshold, got %d", len(patterns))
	}

	if patterns[0].Type != "repeated_empty_read" {
		t.Errorf("Expected type 'repeated_empty_read', got %q", patterns[0].Type)
	}

	if patterns[0].Count != RepetitionThreshold {
		t.Errorf("Expected count %d, got %d", RepetitionThreshold, patterns[0].Count)
	}
}

func TestDetectRepeatedEmptyReads_WithLightTier(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add empty reads in light-tier workspace
	for i := 0; i < RepetitionThreshold; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/workspace/SYNTHESIS.md",
			Outcome: OutcomeEmpty,
			WorkspaceContext: map[string]string{
				"tier": "light",
			},
		})
	}

	patterns := log.DetectPatterns()

	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	// Light tier SYNTHESIS.md reads should be "info" severity
	if patterns[0].Severity != "info" {
		t.Errorf("Expected severity 'info' for light tier, got %q", patterns[0].Severity)
	}

	// Should have context preserved
	if patterns[0].Context["tier"] != "light" {
		t.Errorf("Expected context tier=light, got %v", patterns[0].Context)
	}
}

func TestDetectRepeatedEmptyReads_StandardTier(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add empty reads in standard (non-light) workspace
	for i := 0; i < RepetitionThreshold; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/workspace/SYNTHESIS.md",
			Outcome: OutcomeEmpty,
			WorkspaceContext: map[string]string{
				"tier": "standard",
			},
		})
	}

	patterns := log.DetectPatterns()

	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	// Standard tier SYNTHESIS.md should be "warning"
	if patterns[0].Severity != "warning" {
		t.Errorf("Expected severity 'warning' for standard tier, got %q", patterns[0].Severity)
	}
}

func TestDetectRepeatedErrors(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add repeated errors
	for i := 0; i < RepetitionThreshold; i++ {
		log.RecordAction(ActionEvent{
			Tool:          "Bash",
			Target:        "make build",
			Outcome:       OutcomeError,
			OutcomeDetail: "exit code 1: no such file or directory",
		})
	}

	patterns := log.DetectPatterns()

	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	if patterns[0].Type != "repeated_error" {
		t.Errorf("Expected type 'repeated_error', got %q", patterns[0].Type)
	}
}

func TestDetectRepeatedErrors_CriticalSeverity(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add many errors (2x threshold)
	for i := 0; i < RepetitionThreshold*2; i++ {
		log.RecordAction(ActionEvent{
			Tool:          "Bash",
			Target:        "npm install",
			Outcome:       OutcomeError,
			OutcomeDetail: "ENOENT: permission denied",
		})
	}

	patterns := log.DetectPatterns()

	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	if patterns[0].Severity != "critical" {
		t.Errorf("Expected severity 'critical' for 2x threshold, got %q", patterns[0].Severity)
	}
}

func TestNormalizeActionKey(t *testing.T) {
	tests := []struct {
		tool   string
		target string
		want   string
	}{
		{"Read", "/path/to/file.md", "read:to/file.md"},
		{"read", "/path/to/file.md", "read:to/file.md"},
		{"Bash", "make build", "bash:make build"},
		{"Read", "/a/b/c/d/e.txt", "read:d/e.txt"},
		{"Read", "file.md", "read:file.md"},
	}

	for _, tt := range tests {
		got := normalizeActionKey(tt.tool, tt.target)
		if got != tt.want {
			t.Errorf("normalizeActionKey(%q, %q) = %q, want %q", tt.tool, tt.target, got, tt.want)
		}
	}
}

func TestNormalizeErrorType(t *testing.T) {
	tests := []struct {
		detail string
		want   string
	}{
		{"no such file or directory", "file_not_found"},
		{"ENOENT: no such file or directory", "file_not_found"},
		{"permission denied", "permission_denied"},
		{"operation timeout", "timeout"},
		{"dial tcp: connection refused", "connection_refused"},
		{"some other error", "some other error"},
		{"this is a very long error message that should be truncated to fifty characters", "this is a very long error message that should be t"},
	}

	for _, tt := range tests {
		got := normalizeErrorType(tt.detail)
		if got != tt.want {
			t.Errorf("normalizeErrorType(%q) = %q, want %q", tt.detail, got, tt.want)
		}
	}
}

func TestExtractCommonContext(t *testing.T) {
	events := []ActionEvent{
		{WorkspaceContext: map[string]string{"tier": "light", "skill": "feature-impl"}},
		{WorkspaceContext: map[string]string{"tier": "light", "skill": "investigation"}},
		{WorkspaceContext: map[string]string{"tier": "light", "phase": "complete"}},
	}

	common := extractCommonContext(events)

	// Only "tier" is common across all events
	if common["tier"] != "light" {
		t.Errorf("Expected tier=light, got %v", common["tier"])
	}

	if _, exists := common["skill"]; exists {
		t.Error("skill should not be in common context (differs across events)")
	}
}

func TestSuppressPattern(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Create pattern events
	for i := 0; i < RepetitionThreshold; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/workspace/SYNTHESIS.md",
			Outcome: OutcomeEmpty,
		})
	}

	patterns := log.DetectPatterns()
	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	// Suppress the pattern
	log.SuppressPattern(patterns[0], "Known issue", 24*time.Hour)

	// Should now return no patterns
	patterns = log.DetectPatterns()
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns after suppression, got %d", len(patterns))
	}
}

func TestSuppressPatternExpiration(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Create pattern events
	for i := 0; i < RepetitionThreshold; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/workspace/SYNTHESIS.md",
			Outcome: OutcomeEmpty,
		})
	}

	patterns := log.DetectPatterns()
	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	// Suppress with short duration, then manually set ExpiresAt to past
	log.SuppressPattern(patterns[0], "Known issue", 1*time.Hour)

	// Verify suppression is active
	patterns = log.DetectPatterns()
	if len(patterns) != 0 {
		t.Fatalf("Expected 0 patterns with active suppression, got %d", len(patterns))
	}

	// Manually set expiration to past to simulate expiration
	log.SuppressedPatterns[0].ExpiresAt = time.Now().Add(-1 * time.Hour)

	// Prune should remove expired suppression
	log.pruneExpiredSuppressions()

	// Should see the pattern again
	patterns = log.DetectPatterns()
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern after suppression expired, got %d", len(patterns))
	}
}

func TestPruneOldEvents(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add old event
	oldEvent := ActionEvent{
		Timestamp: time.Now().Add(-ActionLogMaxAge - time.Hour),
		Tool:      "Read",
		Target:    "/old/file.md",
		Outcome:   OutcomeSuccess,
	}
	log.Events = append(log.Events, oldEvent)

	// Add recent event
	log.RecordAction(ActionEvent{
		Tool:    "Read",
		Target:  "/recent/file.md",
		Outcome: OutcomeSuccess,
	})

	log.pruneOldEvents()

	if len(log.Events) != 1 {
		t.Errorf("Expected 1 event after pruning, got %d", len(log.Events))
	}

	if log.Events[0].Target != "/recent/file.md" {
		t.Error("Expected recent event to remain, got old event")
	}
}

func TestPruneMaxEvents(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add more than max events
	for i := 0; i < MaxActionEvents+100; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/file.md",
			Outcome: OutcomeSuccess,
		})
	}

	log.pruneOldEvents()

	if len(log.Events) != MaxActionEvents {
		t.Errorf("Expected %d events after pruning, got %d", MaxActionEvents, len(log.Events))
	}
}

func TestActionLogSaveLoad(t *testing.T) {
	// Use temp directory
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "action-log.json")

	// Override path function
	originalFunc := logPathFunc
	logPathFunc = func() string { return testPath }
	defer func() { logPathFunc = originalFunc }()

	// Create and save log
	log := &ActionLog{Events: []ActionEvent{}}
	log.RecordAction(ActionEvent{
		Tool:    "Read",
		Target:  "/test/file.md",
		Outcome: OutcomeSuccess,
	})

	err := log.Save()
	if err != nil {
		t.Fatalf("Failed to save log: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Fatal("Log file was not created")
	}

	// Load and verify
	loaded, err := LoadLog()
	if err != nil {
		t.Fatalf("Failed to load log: %v", err)
	}

	if len(loaded.Events) != 1 {
		t.Errorf("Expected 1 event after load, got %d", len(loaded.Events))
	}

	if loaded.Events[0].Target != "/test/file.md" {
		t.Errorf("Expected target '/test/file.md', got %q", loaded.Events[0].Target)
	}
}

func TestLoadLogNonExistent(t *testing.T) {
	// Use temp directory
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "nonexistent", "action-log.json")

	// Override path function
	originalFunc := logPathFunc
	logPathFunc = func() string { return testPath }
	defer func() { logPathFunc = originalFunc }()

	// Should return empty log, not error
	log, err := LoadLog()
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}

	if log == nil || len(log.Events) != 0 {
		t.Error("Expected empty log for non-existent file")
	}
}

func TestFormatPatterns(t *testing.T) {
	patterns := []Pattern{
		{
			Type:        "repeated_empty_read",
			Description: "Read SYNTHESIS.md returned empty 3 times",
			Severity:    "warning",
			Count:       3,
			Suggestion:  "Check if file should exist",
			Context:     map[string]string{"tier": "light"},
		},
	}

	output := FormatPatterns(patterns)

	// Check for key elements
	if !containsString(output, "Read SYNTHESIS.md returned empty 3 times") {
		t.Errorf("Expected output to contain description, got: %s", output)
	}
	if !containsString(output, "warning") {
		t.Error("Expected output to contain severity")
	}
	if !containsString(output, "tier=light") {
		t.Error("Expected output to contain context")
	}
}

func TestFormatPatternsEmpty(t *testing.T) {
	output := FormatPatterns([]Pattern{})

	if !containsString(output, "No behavioral patterns detected") {
		t.Error("Expected empty message")
	}
}

func TestSummary(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Empty log
	summary := log.Summary()
	if !containsString(summary, "No actions logged") {
		t.Errorf("Expected empty summary, got: %s", summary)
	}

	// Add events
	for i := 0; i < 5; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/file.md",
			Outcome: OutcomeSuccess,
		})
	}

	summary = log.Summary()
	if !containsString(summary, "5 action events") {
		t.Errorf("Expected '5 action events' in summary, got: %s", summary)
	}
}

func TestGetRecentEvents(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add events
	for i := 0; i < 10; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/file.md",
			Outcome: OutcomeSuccess,
		})
	}

	// Get fewer than total
	recent := log.GetRecentEvents(3)
	if len(recent) != 3 {
		t.Errorf("Expected 3 recent events, got %d", len(recent))
	}

	// Get more than total
	recent = log.GetRecentEvents(20)
	if len(recent) != 10 {
		t.Errorf("Expected 10 events (all), got %d", len(recent))
	}
}

func TestClearEvents(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add events
	for i := 0; i < 5; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/file.md",
			Outcome: OutcomeSuccess,
		})
	}

	log.ClearEvents()

	if len(log.Events) != 0 {
		t.Errorf("Expected 0 events after clear, got %d", len(log.Events))
	}
}

func TestMultiplePatternTypes(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add empty reads
	for i := 0; i < RepetitionThreshold; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/workspace/SYNTHESIS.md",
			Outcome: OutcomeEmpty,
		})
	}

	// Add errors for different target
	for i := 0; i < RepetitionThreshold; i++ {
		log.RecordAction(ActionEvent{
			Tool:          "Bash",
			Target:        "make test",
			Outcome:       OutcomeError,
			OutcomeDetail: "exit code 1",
		})
	}

	patterns := log.DetectPatterns()

	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(patterns))
	}

	// Check both types are present
	types := make(map[string]bool)
	for _, p := range patterns {
		types[p.Type] = true
	}

	if !types["repeated_empty_read"] {
		t.Error("Expected repeated_empty_read pattern")
	}
	if !types["repeated_error"] {
		t.Error("Expected repeated_error pattern")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestEventMatchesProject(t *testing.T) {
	tests := []struct {
		name       string
		event      ActionEvent
		projectDir string
		want       bool
	}{
		{
			name:       "empty project dir matches all",
			event:      ActionEvent{Target: "/any/path/file.go"},
			projectDir: "",
			want:       true,
		},
		{
			name:       "target within project matches",
			event:      ActionEvent{Target: "/Users/dev/orch-go/pkg/spawn/context.go"},
			projectDir: "/Users/dev/orch-go",
			want:       true,
		},
		{
			name:       "target outside project does not match",
			event:      ActionEvent{Target: "/Users/dev/price-watch/main.go"},
			projectDir: "/Users/dev/orch-go",
			want:       false,
		},
		{
			name:       "workspace dir within project matches",
			event:      ActionEvent{Target: "/other/path", WorkspaceDir: "/Users/dev/orch-go/.orch/workspace/og-test"},
			projectDir: "/Users/dev/orch-go",
			want:       true,
		},
		{
			name:       "neither target nor workspace matches",
			event:      ActionEvent{Target: "/other/path", WorkspaceDir: "/other/workspace"},
			projectDir: "/Users/dev/orch-go",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eventMatchesProject(tt.event, tt.projectDir)
			if got != tt.want {
				t.Errorf("eventMatchesProject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectPatternsForProject(t *testing.T) {
	log := &ActionLog{Events: []ActionEvent{}}

	// Add events for project A (orch-go)
	for i := 0; i < RepetitionThreshold; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/Users/dev/orch-go/pkg/spawn/context.go",
			Outcome: OutcomeEmpty,
		})
	}

	// Add events for project B (price-watch)
	for i := 0; i < RepetitionThreshold+1; i++ {
		log.RecordAction(ActionEvent{
			Tool:    "Read",
			Target:  "/Users/dev/price-watch/main.go",
			Outcome: OutcomeEmpty,
		})
	}

	t.Run("unfiltered returns all patterns", func(t *testing.T) {
		patterns := log.DetectPatternsForProject("")
		if len(patterns) != 2 {
			t.Errorf("Expected 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("filtering by project A returns only project A patterns", func(t *testing.T) {
		patterns := log.DetectPatternsForProject("/Users/dev/orch-go")
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern for orch-go, got %d", len(patterns))
		}
		if len(patterns) > 0 && !containsString(patterns[0].Events[0].Target, "orch-go") {
			t.Errorf("Expected pattern to be from orch-go, got: %s", patterns[0].Events[0].Target)
		}
	})

	t.Run("filtering by project B returns only project B patterns", func(t *testing.T) {
		patterns := log.DetectPatternsForProject("/Users/dev/price-watch")
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern for price-watch, got %d", len(patterns))
		}
		if len(patterns) > 0 && !containsString(patterns[0].Events[0].Target, "price-watch") {
			t.Errorf("Expected pattern to be from price-watch, got: %s", patterns[0].Events[0].Target)
		}
	})

	t.Run("filtering by non-existent project returns empty", func(t *testing.T) {
		patterns := log.DetectPatternsForProject("/Users/dev/nonexistent")
		if len(patterns) != 0 {
			t.Errorf("Expected 0 patterns for non-existent project, got %d", len(patterns))
		}
	})
}
