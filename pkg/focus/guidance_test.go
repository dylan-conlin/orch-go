package focus

import (
	"strings"
	"testing"
)

func TestDetectThreadKeyword(t *testing.T) {
	tests := []struct {
		title        string
		wantKeyword  string
		wantThread   string
	}{
		{"Implement session end validation", "session", "Session tooling"},
		{"Update 6 models to use pattern", "model", "Model system"},
		{"Fix dashboard rendering issue", "dashboard", "Dashboard"},
		{"Add spawn retry logic", "spawn", "Spawn system"},
		{"Configure daemon polling interval", "daemon", "Daemon"},
		{"Improve kb context search", "kb", "Knowledge base"},
		{"orch doctor: verify health", "orch", "Orch tooling"},
		{"Add beads integration tests", "beads", "Beads integration"},
		{"Clean up empty templates", "clean", "Cleanup"},
		{"Track escape hatch usage", "escape", "Escape hatch"},
		{"Some random issue title", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			keyword, thread := detectThreadKeyword(tt.title)
			if keyword != tt.wantKeyword {
				t.Errorf("detectThreadKeyword(%q) keyword = %q, want %q", tt.title, keyword, tt.wantKeyword)
			}
			if thread != tt.wantThread {
				t.Errorf("detectThreadKeyword(%q) thread = %q, want %q", tt.title, thread, tt.wantThread)
			}
		})
	}
}

func TestDetectThreadKeywordCaseInsensitive(t *testing.T) {
	// Test case insensitivity
	tests := []string{
		"SESSION validation",
		"Session validation",
		"session validation",
		"fix SESSION issue",
	}

	for _, title := range tests {
		keyword, thread := detectThreadKeyword(title)
		if keyword != "session" {
			t.Errorf("detectThreadKeyword(%q) should match 'session', got %q", title, keyword)
		}
		if thread != "Session tooling" {
			t.Errorf("detectThreadKeyword(%q) should return 'Session tooling', got %q", title, thread)
		}
	}
}

func TestGroupIntoThreads(t *testing.T) {
	issues := []BeadsIssue{
		{ID: "proj-1", Title: "Session start focus guidance"},
		{ID: "proj-2", Title: "Session end validation"},
		{ID: "proj-3", Title: "Session resume protocol"},
		{ID: "proj-4", Title: "Update model template"},
		{ID: "proj-5", Title: "Random unrelated issue"},
	}

	threads := GroupIntoThreads(issues)

	// Should have 3 threads: Session tooling (3), Model system (1), Misc (1)
	if len(threads) != 3 {
		t.Errorf("expected 3 threads, got %d", len(threads))
	}

	// First thread should be Session tooling (largest with 3 issues)
	if threads[0].Name != "Session tooling" {
		t.Errorf("expected first thread to be 'Session tooling', got %q", threads[0].Name)
	}
	if len(threads[0].Issues) != 3 {
		t.Errorf("expected Session tooling to have 3 issues, got %d", len(threads[0].Issues))
	}

	// Check Misc thread exists and has the ungrouped issue
	var miscThread *Thread
	for i := range threads {
		if threads[i].Name == "Misc" {
			miscThread = &threads[i]
			break
		}
	}
	if miscThread == nil {
		t.Error("expected Misc thread to exist")
	}
	if miscThread != nil && len(miscThread.Issues) != 1 {
		t.Errorf("expected Misc to have 1 issue, got %d", len(miscThread.Issues))
	}
}

func TestGroupIntoThreadsEmpty(t *testing.T) {
	threads := GroupIntoThreads(nil)
	if threads != nil {
		t.Errorf("expected nil threads for empty input, got %v", threads)
	}

	threads = GroupIntoThreads([]BeadsIssue{})
	if threads != nil {
		t.Errorf("expected nil threads for empty slice, got %v", threads)
	}
}

func TestGroupIntoThreadsSorting(t *testing.T) {
	// Create issues that should result in predictable ordering
	issues := []BeadsIssue{
		{ID: "1", Title: "Model thing 1"},
		{ID: "2", Title: "Model thing 2"},
		{ID: "3", Title: "Model thing 3"},
		{ID: "4", Title: "Session thing"},
		{ID: "5", Title: "Dashboard thing"},
	}

	threads := GroupIntoThreads(issues)

	// Model system has 3 issues, should be first
	if threads[0].Name != "Model system" {
		t.Errorf("expected first thread to be 'Model system' (3 issues), got %q with %d issues",
			threads[0].Name, len(threads[0].Issues))
	}
}

func TestGenerateThreadNotes(t *testing.T) {
	issues := []BeadsIssue{
		{ID: "1", Title: "A very short title"},
	}
	notes := generateThreadNotes(issues)
	if notes != "A very short title" {
		t.Errorf("expected full title for short notes, got %q", notes)
	}

	// Test truncation
	longTitle := "This is a very long title that should be truncated because it exceeds fifty characters"
	issues = []BeadsIssue{
		{ID: "1", Title: longTitle},
	}
	notes = generateThreadNotes(issues)
	if len(notes) > 50 {
		t.Errorf("expected notes to be truncated to ~50 chars, got %d chars", len(notes))
	}
	if !strings.HasSuffix(notes, "...") {
		t.Errorf("expected truncated notes to end with '...', got %q", notes)
	}
}

func TestFormatFocusGuidance(t *testing.T) {
	guidance := &FocusGuidance{
		TotalIssues: 3,
		ThreadCount: 2,
		Threads: []Thread{
			{Name: "Session tooling", Issues: []BeadsIssue{{ID: "proj-1"}, {ID: "proj-2"}}, Notes: "Session work"},
			{Name: "Model system", Issues: []BeadsIssue{{ID: "proj-3"}}, Notes: "Model work"},
		},
		PromptText: "What's nagging you?",
	}

	output := FormatFocusGuidance(guidance)

	// Check key elements are present
	if !strings.Contains(output, "Focus Guidance") {
		t.Error("expected output to contain 'Focus Guidance'")
	}
	if !strings.Contains(output, "3 ready issues") {
		t.Error("expected output to contain '3 ready issues'")
	}
	if !strings.Contains(output, "2 threads") {
		t.Error("expected output to contain '2 threads'")
	}
	if !strings.Contains(output, "Session tooling") {
		t.Error("expected output to contain 'Session tooling'")
	}
	if !strings.Contains(output, "proj-1, proj-2") {
		t.Error("expected output to contain 'proj-1, proj-2'")
	}
	if !strings.Contains(output, "What's nagging you?") {
		t.Error("expected output to contain prompt text")
	}
}

func TestFormatFocusGuidanceEmpty(t *testing.T) {
	guidance := &FocusGuidance{
		TotalIssues: 0,
		ThreadCount: 0,
		PromptText:  "No ready issues found.",
	}

	output := FormatFocusGuidance(guidance)
	if !strings.Contains(output, "No ready issues found.") {
		t.Errorf("expected output to contain 'No ready issues found.', got %q", output)
	}
}

func TestMaxThreadsCapping(t *testing.T) {
	// Create more than MaxThreads distinct keywords
	issues := []BeadsIssue{
		{ID: "1", Title: "Session thing"},
		{ID: "2", Title: "Model thing"},
		{ID: "3", Title: "Dashboard thing"},
		{ID: "4", Title: "Spawn thing"},
		{ID: "5", Title: "Daemon thing"},
		{ID: "6", Title: "kb thing"},
		{ID: "7", Title: "orch thing"},
		{ID: "8", Title: "beads thing"},
		{ID: "9", Title: "doctor thing"}, // groups with orch
		{ID: "10", Title: "clean thing"},
	}

	threads := GroupIntoThreads(issues)

	if len(threads) > MaxThreads {
		t.Errorf("expected at most %d threads, got %d", MaxThreads, len(threads))
	}
}
