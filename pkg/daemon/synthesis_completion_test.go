package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSynthesisTopic_BasicTitle(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Synthesize model investigations (11)", "model"},
		{"Synthesize daemon investigations", "daemon"},
		{"Synthesize agent lifecycle investigations (5)", "agent lifecycle"},
		{"Synthesize dashboard investigations (3)", "dashboard"},
		{"synthesize MODEL investigations (7)", "MODEL"},
		{"Fix a bug in the code", ""},
		{"Add new feature", ""},
		{"", ""},
		{"Synthesize investigations", ""},
		{"Synthesize  investigations", ""},
	}

	for _, tt := range tests {
		got := ExtractSynthesisTopic(tt.title)
		if got != tt.want {
			t.Errorf("ExtractSynthesisTopic(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}

func TestIsSynthesisIssue(t *testing.T) {
	tests := []struct {
		title string
		want  bool
	}{
		{"Synthesize model investigations (11)", true},
		{"Synthesize daemon investigations", true},
		{"synthesize agent investigations (5)", true},
		{"Fix a bug in the code", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsSynthesisIssue(tt.title)
		if got != tt.want {
			t.Errorf("IsSynthesisIssue(%q) = %v, want %v", tt.title, got, tt.want)
		}
	}
}

func TestIsSynthesisCompleted_WithGuide(t *testing.T) {
	// Set up temp project directory with .kb/guides/model-selection.md
	dir := t.TempDir()
	guidesDir := filepath.Join(dir, ".kb", "guides")
	if err := os.MkdirAll(guidesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(guidesDir, "model-selection.md"), []byte("# Model Selection Guide"), 0644); err != nil {
		t.Fatal(err)
	}

	// "model" should match "model-selection.md" (hyphenated variant)
	if !IsSynthesisCompleted("model", dir) {
		t.Error("Expected IsSynthesisCompleted to return true for 'model' with model-selection.md guide")
	}

	// "selection" should also match
	if !IsSynthesisCompleted("selection", dir) {
		t.Error("Expected IsSynthesisCompleted to return true for 'selection' with model-selection.md guide")
	}

	// "daemon" should not match
	if IsSynthesisCompleted("daemon", dir) {
		t.Error("Expected IsSynthesisCompleted to return false for 'daemon' with no daemon guide")
	}
}

func TestIsSynthesisCompleted_WithDecision(t *testing.T) {
	dir := t.TempDir()
	decisionsDir := filepath.Join(dir, ".kb", "decisions")
	if err := os.MkdirAll(decisionsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decisionsDir, "2026-01-06-daemon-architecture.md"), []byte("# Daemon Architecture"), 0644); err != nil {
		t.Fatal(err)
	}

	// "daemon" should match the decision (date prefix stripped, then matches part)
	if !IsSynthesisCompleted("daemon", dir) {
		t.Error("Expected IsSynthesisCompleted to return true for 'daemon' with daemon decision")
	}

	// "architecture" should also match
	if !IsSynthesisCompleted("architecture", dir) {
		t.Error("Expected IsSynthesisCompleted to return true for 'architecture' with daemon-architecture decision")
	}
}

func TestIsSynthesisCompleted_NoKBDir(t *testing.T) {
	dir := t.TempDir()

	// No .kb directory - should return false
	if IsSynthesisCompleted("model", dir) {
		t.Error("Expected IsSynthesisCompleted to return false when no .kb directory exists")
	}
}

func TestIsSynthesisCompleted_EmptyTopic(t *testing.T) {
	dir := t.TempDir()

	if IsSynthesisCompleted("", dir) {
		t.Error("Expected IsSynthesisCompleted to return false for empty topic")
	}
}

func TestIsSynthesisCompleted_ShortWordSkipped(t *testing.T) {
	dir := t.TempDir()
	guidesDir := filepath.Join(dir, ".kb", "guides")
	if err := os.MkdirAll(guidesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(guidesDir, "an-example.md"), []byte("# An Example"), 0644); err != nil {
		t.Fatal(err)
	}

	// "an" (2 chars) should be skipped as too short
	if IsSynthesisCompleted("an", dir) {
		t.Error("Expected IsSynthesisCompleted to return false for short word 'an'")
	}
}

func TestCheckSynthesisCompletion_NonSynthesisIssue(t *testing.T) {
	issue := &Issue{Title: "Fix a bug", ID: "test-1"}
	result := CheckSynthesisCompletion(issue, t.TempDir())
	if result != "" {
		t.Errorf("Expected empty string for non-synthesis issue, got %q", result)
	}
}

func TestCheckSynthesisCompletion_CompletedSynthesis(t *testing.T) {
	dir := t.TempDir()
	guidesDir := filepath.Join(dir, ".kb", "guides")
	if err := os.MkdirAll(guidesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(guidesDir, "model-selection.md"), []byte("# Guide"), 0644); err != nil {
		t.Fatal(err)
	}

	issue := &Issue{
		Title: "Synthesize model investigations (11)",
		ID:    "orch-go-test1",
	}
	result := CheckSynthesisCompletion(issue, dir)
	if result == "" {
		t.Error("Expected non-empty reason for completed synthesis, got empty string")
	}
	if result != `synthesis already completed for topic "model" (guide/decision exists)` {
		t.Errorf("Unexpected reason: %q", result)
	}
}

func TestCheckSynthesisCompletion_IncompleteSynthesis(t *testing.T) {
	dir := t.TempDir()
	// Create .kb directory with no matching guides
	guidesDir := filepath.Join(dir, ".kb", "guides")
	if err := os.MkdirAll(guidesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(guidesDir, "something-else.md"), []byte("# Guide"), 0644); err != nil {
		t.Fatal(err)
	}

	issue := &Issue{
		Title: "Synthesize daemon investigations (5)",
		ID:    "orch-go-test2",
	}
	result := CheckSynthesisCompletion(issue, dir)
	if result != "" {
		t.Errorf("Expected empty string for incomplete synthesis, got %q", result)
	}
}

func TestCheckSynthesisCompletion_NilIssue(t *testing.T) {
	result := CheckSynthesisCompletion(nil, t.TempDir())
	if result != "" {
		t.Errorf("Expected empty string for nil issue, got %q", result)
	}
}

// TestCheckSynthesisCompletion_PolysemousModel tests the specific bug case:
// "Synthesize model investigations (11)" should be blocked when
// .kb/guides/model-selection.md exists (the actual completed synthesis).
func TestCheckSynthesisCompletion_PolysemousModel(t *testing.T) {
	dir := t.TempDir()
	guidesDir := filepath.Join(dir, ".kb", "guides")
	if err := os.MkdirAll(guidesDir, 0755); err != nil {
		t.Fatal(err)
	}
	// This is the actual guide that was created on Jan 6
	if err := os.WriteFile(filepath.Join(guidesDir, "model-selection.md"), []byte("# Model Selection Guide\n\n326 lines of synthesis"), 0644); err != nil {
		t.Fatal(err)
	}

	// This is the actual issue title that kept getting spawned
	issue := &Issue{
		Title: "Synthesize model investigations (11)",
		ID:    "orch-go-bn6io",
	}
	result := CheckSynthesisCompletion(issue, dir)
	if result == "" {
		t.Fatal("REGRESSION: Synthesis completion not detected for the 'model' polysemous keyword case. " +
			"This was the original bug - 'Synthesize model investigations (11)' kept spawning even though " +
			".kb/guides/model-selection.md already existed.")
	}
}
