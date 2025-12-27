package action

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActionEvent_PatternKey(t *testing.T) {
	tests := []struct {
		name     string
		event    ActionEvent
		expected string
	}{
		{
			name: "basic event",
			event: ActionEvent{
				Tool:    "Read",
				Target:  "/path/to/file.md",
				Outcome: OutcomeEmpty,
			},
			expected: "Read:*.md:empty",
		},
		{
			name: "event without extension",
			event: ActionEvent{
				Tool:    "Bash",
				Target:  "ls -la",
				Outcome: OutcomeError,
			},
			expected: "Bash:ls -la:error",
		},
		{
			name: "file without extension",
			event: ActionEvent{
				Tool:    "Read",
				Target:  "/path/to/README",
				Outcome: OutcomeSuccess,
			},
			expected: "Read:README:success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.PatternKey()
			if got != tt.expected {
				t.Errorf("PatternKey() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNormalizeTarget(t *testing.T) {
	tests := []struct {
		target   string
		expected string
	}{
		{"/path/to/file.md", "*.md"},
		{"/path/to/file.go", "*.go"},
		{"/path/to/README", "README"},
		{"some command", "some command"},
		{"  whitespace  ", "whitespace"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := normalizeTarget(tt.target)
			if got != tt.expected {
				t.Errorf("normalizeTarget(%q) = %q, want %q", tt.target, got, tt.expected)
			}
		})
	}
}

func TestLogger_Log(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "action-log.jsonl")

	logger := NewLogger(logPath)

	// Log some events
	err := logger.LogSuccess("Read", "/path/to/file.go")
	if err != nil {
		t.Fatalf("LogSuccess failed: %v", err)
	}

	err = logger.LogEmpty("Read", "/path/to/missing.md")
	if err != nil {
		t.Fatalf("LogEmpty failed: %v", err)
	}

	err = logger.LogError("Bash", "git status", "command failed")
	if err != nil {
		t.Fatalf("LogError failed: %v", err)
	}

	err = logger.LogFallback("Read", "/path/to/file.md", "Used bd show instead")
	if err != nil {
		t.Fatalf("LogFallback failed: %v", err)
	}

	// Load and verify
	tracker, err := LoadTracker(logPath)
	if err != nil {
		t.Fatalf("LoadTracker failed: %v", err)
	}

	if len(tracker.Events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(tracker.Events))
	}

	// Check each event
	expectedOutcomes := []Outcome{OutcomeSuccess, OutcomeEmpty, OutcomeError, OutcomeFallback}
	for i, expected := range expectedOutcomes {
		if tracker.Events[i].Outcome != expected {
			t.Errorf("Event %d: expected outcome %q, got %q", i, expected, tracker.Events[i].Outcome)
		}
	}
}

func TestTracker_FindPatterns(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "action-log.jsonl")

	logger := NewLogger(logPath)
	now := time.Now()

	// Log events that should create a pattern (3+ empty outcomes on same target type)
	for i := 0; i < 4; i++ {
		event := ActionEvent{
			Tool:      "Read",
			Target:    "/path/to/SYNTHESIS.md",
			Outcome:   OutcomeEmpty,
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := logger.Log(event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Log some success events (should not create pattern)
	for i := 0; i < 5; i++ {
		event := ActionEvent{
			Tool:      "Read",
			Target:    "/path/to/file.go",
			Outcome:   OutcomeSuccess,
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
		}
		if err := logger.Log(event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Log events below threshold (only 2)
	for i := 0; i < 2; i++ {
		event := ActionEvent{
			Tool:      "Bash",
			Target:    "some command",
			Outcome:   OutcomeError,
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := logger.Log(event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Load and find patterns
	tracker, err := LoadTracker(logPath)
	if err != nil {
		t.Fatalf("LoadTracker failed: %v", err)
	}

	patterns := tracker.FindPatterns()

	// Should find 1 pattern (the 4 empty reads)
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(patterns))
	}

	if len(patterns) > 0 {
		p := patterns[0]
		if p.Tool != "Read" {
			t.Errorf("Expected Tool 'Read', got %q", p.Tool)
		}
		if p.Outcome != OutcomeEmpty {
			t.Errorf("Expected Outcome 'empty', got %q", p.Outcome)
		}
		if p.Count != 4 {
			t.Errorf("Expected Count 4, got %d", p.Count)
		}
	}
}

func TestTracker_FindPatternsForSession(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "action-log.jsonl")

	logger := NewLogger(logPath)
	now := time.Now()

	// Log events for session-1
	for i := 0; i < 3; i++ {
		event := ActionEvent{
			Tool:      "Read",
			Target:    "/path/to/file.md",
			Outcome:   OutcomeEmpty,
			SessionID: "session-1",
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := logger.Log(event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Log events for session-2
	for i := 0; i < 2; i++ {
		event := ActionEvent{
			Tool:      "Read",
			Target:    "/path/to/other.md",
			Outcome:   OutcomeEmpty,
			SessionID: "session-2",
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
		}
		if err := logger.Log(event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	tracker, err := LoadTracker(logPath)
	if err != nil {
		t.Fatalf("LoadTracker failed: %v", err)
	}

	// Find patterns for session-1
	patterns1 := tracker.FindPatternsForSession("session-1")
	if len(patterns1) != 1 {
		t.Errorf("Expected 1 pattern for session-1, got %d", len(patterns1))
	}
	if len(patterns1) > 0 && patterns1[0].Count != 3 {
		t.Errorf("Expected count 3, got %d", patterns1[0].Count)
	}

	// Find patterns for session-2
	patterns2 := tracker.FindPatternsForSession("session-2")
	if len(patterns2) != 1 {
		t.Errorf("Expected 1 pattern for session-2, got %d", len(patterns2))
	}
	if len(patterns2) > 0 && patterns2[0].Count != 2 {
		t.Errorf("Expected count 2, got %d", patterns2[0].Count)
	}
}

func TestFormatPatterns(t *testing.T) {
	patterns := []ActionPattern{
		{
			Tool:       "Read",
			Target:     "*.md",
			Outcome:    OutcomeEmpty,
			Count:      5,
			Workspaces: []string{"workspace-1", "workspace-2"},
		},
		{
			Tool:    "Bash",
			Target:  "git status",
			Outcome: OutcomeError,
			Count:   3,
		},
	}

	output := FormatPatterns(patterns)

	// Should contain key elements
	if !contains(output, "BEHAVIORAL PATTERNS") {
		t.Error("Output should contain header")
	}
	if !contains(output, "Read") {
		t.Error("Output should contain 'Read'")
	}
	if !contains(output, "*.md") {
		t.Error("Output should contain '*.md'")
	}
	if !contains(output, "5x") {
		t.Error("Output should contain '5x'")
	}
}

func TestFormatPatterns_Empty(t *testing.T) {
	output := FormatPatterns(nil)

	if !contains(output, "No behavioral patterns detected") {
		t.Error("Output should indicate no patterns")
	}
}

func TestActionPattern_SuggestKnEntry(t *testing.T) {
	tests := []struct {
		name     string
		pattern  ActionPattern
		expected string
	}{
		{
			name: "empty outcome",
			pattern: ActionPattern{
				Tool:    "Read",
				Target:  "*.md",
				Outcome: OutcomeEmpty,
			},
			expected: `kn tried "Read on *.md" --failed "Returns empty - target doesn't exist or has no content"`,
		},
		{
			name: "error outcome",
			pattern: ActionPattern{
				Tool:    "Bash",
				Target:  "git status",
				Outcome: OutcomeError,
			},
			expected: `kn tried "Bash on git status" --failed "Action fails repeatedly - investigate cause"`,
		},
		{
			name: "fallback outcome",
			pattern: ActionPattern{
				Tool:    "Read",
				Target:  "*.md",
				Outcome: OutcomeFallback,
			},
			expected: `kn constrain "Avoid Read on *.md" --reason "Requires fallback - prefer alternative approach"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pattern.SuggestKnEntry()
			if got != tt.expected {
				t.Errorf("SuggestKnEntry() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPrune(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "action-log.jsonl")

	logger := NewLogger(logPath)
	now := time.Now()

	// Log old events
	for i := 0; i < 5; i++ {
		event := ActionEvent{
			Tool:      "Read",
			Target:    "/old/file.md",
			Outcome:   OutcomeSuccess,
			Timestamp: now.Add(-48 * time.Hour), // 2 days ago
		}
		if err := logger.Log(event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Log recent events
	for i := 0; i < 3; i++ {
		event := ActionEvent{
			Tool:      "Read",
			Target:    "/new/file.md",
			Outcome:   OutcomeSuccess,
			Timestamp: now.Add(-1 * time.Hour), // 1 hour ago
		}
		if err := logger.Log(event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Prune events older than 24 hours
	pruned, err := Prune(logPath, 24*time.Hour)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if pruned != 5 {
		t.Errorf("Expected 5 pruned, got %d", pruned)
	}

	// Verify remaining events
	tracker, err := LoadTracker(logPath)
	if err != nil {
		t.Fatalf("LoadTracker failed: %v", err)
	}

	if len(tracker.Events) != 3 {
		t.Errorf("Expected 3 remaining events, got %d", len(tracker.Events))
	}
}

func TestLoadTracker_NonExistent(t *testing.T) {
	tracker, err := LoadTracker("/nonexistent/path/action-log.jsonl")
	if err != nil {
		t.Fatalf("LoadTracker should not error on nonexistent file: %v", err)
	}

	if len(tracker.Events) != 0 {
		t.Error("Expected empty tracker for nonexistent file")
	}
}

func TestTracker_Summary(t *testing.T) {
	tracker := &Tracker{Events: []ActionEvent{}}

	summary := tracker.Summary()
	if !contains(summary, "No actions tracked yet") {
		t.Error("Empty tracker should show no actions message")
	}

	// Add some events
	now := time.Now()
	for i := 0; i < 3; i++ {
		tracker.Events = append(tracker.Events, ActionEvent{
			Tool:      "Read",
			Target:    "/file.md",
			Outcome:   OutcomeEmpty,
			Timestamp: now,
		})
	}

	summary = tracker.Summary()
	if !contains(summary, "3 action events") {
		t.Errorf("Summary should show event count, got: %s", summary)
	}
}

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

func TestDefaultLogPath(t *testing.T) {
	// Save original
	original := loggerPathFunc
	defer func() { loggerPathFunc = original }()

	// Test default behavior
	path := DefaultLogPath()
	if path == "" {
		t.Error("DefaultLogPath should not be empty")
	}
	if !contains(path, "action-log.jsonl") {
		t.Errorf("DefaultLogPath should contain 'action-log.jsonl', got: %s", path)
	}

	// Test with custom path func
	loggerPathFunc = func() string { return "/custom/path.jsonl" }
	if DefaultLogPath() != "/custom/path.jsonl" {
		t.Error("DefaultLogPath should use custom func")
	}
}

func TestLogger_LogWithSession(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "action-log.jsonl")

	logger := NewLogger(logPath)

	event := ActionEvent{
		Tool:      "Read",
		Target:    "/path/to/file.md",
		Outcome:   OutcomeEmpty,
		SessionID: "test-session-123",
		Workspace: "my-workspace",
		Context:   "checking for synthesis file",
	}

	if err := logger.Log(event); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	tracker, err := LoadTracker(logPath)
	if err != nil {
		t.Fatalf("LoadTracker failed: %v", err)
	}

	if len(tracker.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(tracker.Events))
	}

	e := tracker.Events[0]
	if e.SessionID != "test-session-123" {
		t.Errorf("Expected SessionID 'test-session-123', got %q", e.SessionID)
	}
	if e.Workspace != "my-workspace" {
		t.Errorf("Expected Workspace 'my-workspace', got %q", e.Workspace)
	}
	if e.Context != "checking for synthesis file" {
		t.Errorf("Expected Context 'checking for synthesis file', got %q", e.Context)
	}
}

func TestLogger_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nested", "dir", "action-log.jsonl")

	logger := NewLogger(logPath)

	// Should create nested directories
	if err := logger.LogSuccess("Read", "/file.go"); err != nil {
		t.Fatalf("LogSuccess should create directories: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file should have been created")
	}
}
