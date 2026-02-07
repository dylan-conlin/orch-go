package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestExtractQuestions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "bullet points",
			input: `- What is the performance impact of this change?
- Should we cache these results?
- How does this integrate with the existing API?`,
			expected: []string{
				"What is the performance impact of this change?",
				"Should we cache these results?",
				"How does this integrate with the existing API?",
			},
		},
		{
			name: "with section headers",
			input: `**Areas worth exploring further:**
- Performance optimization strategies
- Alternative caching mechanisms

**What remains unclear:**
- Security implications of the new approach`,
			expected: []string{
				"Performance optimization strategies",
				"Alternative caching mechanisms",
				"Security implications of the new approach",
			},
		},
		{
			name: "numbered list",
			input: `1. What authentication method should we use?
2. How do we handle rate limiting?`,
			expected: []string{
				"What authentication method should we use?",
				"How do we handle rate limiting?",
			},
		},
		{
			name:     "empty section",
			input:    "",
			expected: []string{},
		},
		{
			name: "filters short lines",
			input: `- Short
- This is a proper question that should be included in results`,
			expected: []string{
				"This is a proper question that should be included in results",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractQuestions(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("extractQuestions() count = %d, want %d", len(got), len(tt.expected))
				t.Logf("Got: %v", got)
				t.Logf("Want: %v", tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("extractQuestions()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestFilterRelevantMatches(t *testing.T) {
	matches := []spawn.KBContextMatch{
		{Type: "constraint", Title: "Never use X"},
		{Type: "decision", Title: "Use Y for Z"},
		{Type: "investigation", Title: "How does X work?"},
		{Type: "guide", Title: "Best practices for Y"},
		{Type: "investigation", Title: "Performance analysis"},
	}

	relevant := filterRelevantMatches(matches)

	// Should include constraint, decision, guide (3 items)
	// Should exclude investigations (2 items)
	if len(relevant) != 3 {
		t.Errorf("filterRelevantMatches() count = %d, want 3", len(relevant))
	}

	// Check types
	for _, match := range relevant {
		if match.Type == "investigation" {
			t.Errorf("filterRelevantMatches() included investigation, should exclude")
		}
	}
}

func TestExtractMatchPaths(t *testing.T) {
	matches := []spawn.KBContextMatch{
		{Type: "constraint", Path: "/path/to/constraint.md"},
		{Type: "decision", Path: "/path/to/decision.md"},
		{Type: "constraint", Path: "", Reason: "Never do X"}, // kn entry
	}

	paths := extractMatchPaths(matches)

	if len(paths) != 3 {
		t.Errorf("extractMatchPaths() count = %d, want 3", len(paths))
	}

	// Check kn entry format
	if !strings.HasPrefix(paths[2], "kn: ") {
		t.Errorf("extractMatchPaths()[2] = %q, want to start with 'kn: '", paths[2])
	}
}

func TestExtractMatchTypes(t *testing.T) {
	matches := []spawn.KBContextMatch{
		{Type: "constraint"},
		{Type: "decision"},
		{Type: "constraint"}, // Duplicate
		{Type: "guide"},
	}

	types := extractMatchTypes(matches)

	// Should deduplicate - 3 unique types
	if len(types) != 3 {
		t.Errorf("extractMatchTypes() count = %d, want 3 (deduped)", len(types))
	}

	// Check all types present
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	if !typeSet["constraint"] || !typeSet["decision"] || !typeSet["guide"] {
		t.Errorf("extractMatchTypes() missing expected types, got %v", types)
	}
}

func TestDetectKnowledgeGaps_NoSynthesis(t *testing.T) {
	// Create temporary directory without SYNTHESIS.md
	tmpDir := t.TempDir()

	result, err := DetectKnowledgeGaps(tmpDir, "test-123", "test-skill", tmpDir)
	if err != nil {
		t.Errorf("DetectKnowledgeGaps() should not error when SYNTHESIS.md missing, got %v", err)
	}

	if result.GapsDetected != 0 {
		t.Errorf("DetectKnowledgeGaps() with no SYNTHESIS should have 0 gaps, got %d", result.GapsDetected)
	}
}

func TestDetectKnowledgeGaps_NoUnexploredQuestions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create SYNTHESIS.md without Unexplored Questions section
	synthesisContent := `**Agent:** test-agent
**Issue:** test-123

## TLDR
Completed the task successfully.

## Delta
- Modified file.go

## Evidence
- Tests pass

## Knowledge
- Learned about X

## Next
**Recommendation:** close
`

	if err := os.WriteFile(filepath.Join(tmpDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
	}

	result, err := DetectKnowledgeGaps(tmpDir, "test-123", "test-skill", tmpDir)
	if err != nil {
		t.Errorf("DetectKnowledgeGaps() error = %v", err)
	}

	if result.GapsDetected != 0 {
		t.Errorf("DetectKnowledgeGaps() with no unexplored questions should have 0 gaps, got %d", result.GapsDetected)
	}
}

func TestLogKnowledgeGaps(t *testing.T) {
	// Use temporary home directory for test
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	gaps := []KnowledgeGap{
		{
			Timestamp:   "2026-01-30T12:00:00Z",
			BeadsID:     "test-123",
			Workspace:   "test-workspace",
			Question:    "How does authentication work?",
			KBMatches:   []string{"/path/to/auth-decision.md"},
			MatchTypes:  []string{"decision"},
			SearchQuery: "authentication work",
			Skill:       "feature-impl",
			ProjectDir:  "/test/project",
		},
		{
			Timestamp:   "2026-01-30T12:01:00Z",
			BeadsID:     "test-456",
			Workspace:   "test-workspace-2",
			Question:    "What caching strategy should we use?",
			KBMatches:   []string{"/path/to/caching-guide.md"},
			MatchTypes:  []string{"guide"},
			SearchQuery: "caching strategy",
			Skill:       "architect",
			ProjectDir:  "/test/project",
		},
	}

	err := LogKnowledgeGaps(gaps)
	if err != nil {
		t.Fatalf("LogKnowledgeGaps() error = %v", err)
	}

	// Verify file was created
	gapLogPath := filepath.Join(tmpHome, ".orch", "knowledge-gaps.jsonl")
	if _, err := os.Stat(gapLogPath); os.IsNotExist(err) {
		t.Fatalf("knowledge-gaps.jsonl was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(gapLogPath)
	if err != nil {
		t.Fatalf("Failed to read knowledge-gaps.jsonl: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines in log file, got %d", len(lines))
	}

	// Verify JSON structure of first line
	var firstGap KnowledgeGap
	if err := json.Unmarshal([]byte(lines[0]), &firstGap); err != nil {
		t.Fatalf("Failed to parse first gap: %v", err)
	}

	if firstGap.BeadsID != "test-123" {
		t.Errorf("First gap BeadsID = %q, want %q", firstGap.BeadsID, "test-123")
	}

	if firstGap.Question != "How does authentication work?" {
		t.Errorf("First gap Question = %q, want %q", firstGap.Question, "How does authentication work?")
	}
}

func TestLogKnowledgeGaps_EmptyGaps(t *testing.T) {
	// Use temporary home directory for test
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Should not error on empty gaps
	err := LogKnowledgeGaps([]KnowledgeGap{})
	if err != nil {
		t.Errorf("LogKnowledgeGaps() with empty gaps should not error, got %v", err)
	}

	// File should not be created if no gaps
	gapLogPath := filepath.Join(tmpHome, ".orch", "knowledge-gaps.jsonl")
	if _, err := os.Stat(gapLogPath); err == nil {
		t.Error("knowledge-gaps.jsonl should not be created when no gaps to log")
	}
}

func TestLogKnowledgeGaps_Appends(t *testing.T) {
	// Use temporary home directory for test
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Write first gap
	gap1 := []KnowledgeGap{
		{
			Timestamp:  "2026-01-30T12:00:00Z",
			BeadsID:    "test-1",
			Workspace:  "workspace-1",
			Question:   "Question 1",
			KBMatches:  []string{"/path/1"},
			MatchTypes: []string{"decision"},
		},
	}

	if err := LogKnowledgeGaps(gap1); err != nil {
		t.Fatalf("First LogKnowledgeGaps() error = %v", err)
	}

	// Write second gap
	gap2 := []KnowledgeGap{
		{
			Timestamp:  "2026-01-30T12:01:00Z",
			BeadsID:    "test-2",
			Workspace:  "workspace-2",
			Question:   "Question 2",
			KBMatches:  []string{"/path/2"},
			MatchTypes: []string{"constraint"},
		},
	}

	if err := LogKnowledgeGaps(gap2); err != nil {
		t.Fatalf("Second LogKnowledgeGaps() error = %v", err)
	}

	// Verify both gaps are in file
	gapLogPath := filepath.Join(tmpHome, ".orch", "knowledge-gaps.jsonl")
	data, err := os.ReadFile(gapLogPath)
	if err != nil {
		t.Fatalf("Failed to read knowledge-gaps.jsonl: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines after two appends, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var gap KnowledgeGap
		if err := json.Unmarshal([]byte(line), &gap); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}
