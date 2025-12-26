package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestKBContextResultParsing(t *testing.T) {
	// Test JSON parsing of kb context output
	jsonData := `{
		"constraints": [
			{
				"id": "kn-abc123",
				"type": "constraint",
				"content": "Always use rate limiting",
				"reason": "Prevents API abuse"
			}
		],
		"decisions": [
			{
				"id": "kn-def456",
				"type": "decision",
				"content": "Use Redis for caching",
				"reason": "Better performance"
			}
		],
		"attempts": null,
		"questions": null,
		"investigations": [
			{
				"name": "2025-12-25-inv-test.md",
				"path": "/test/path/inv.md",
				"title": "Test Investigation",
				"type": "investigations",
				"matches": ["line 1", "line 2"]
			}
		]
	}`

	var result KBContextResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify constraints
	if len(result.Constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(result.Constraints))
	}
	if result.Constraints[0].Content != "Always use rate limiting" {
		t.Errorf("Unexpected constraint content: %s", result.Constraints[0].Content)
	}

	// Verify decisions
	if len(result.Decisions) != 1 {
		t.Errorf("Expected 1 decision, got %d", len(result.Decisions))
	}

	// Verify investigations
	if len(result.Investigations) != 1 {
		t.Errorf("Expected 1 investigation, got %d", len(result.Investigations))
	}
	if result.Investigations[0].Title != "Test Investigation" {
		t.Errorf("Unexpected investigation title: %s", result.Investigations[0].Title)
	}
}

func TestWriteContextForSynthesis(t *testing.T) {
	result := &KBContextResult{
		Constraints: []KNEntry{
			{Content: "Must use HTTPS", Reason: "Security requirement"},
			{Content: "Rate limit 100 req/s", Reason: "API stability"},
		},
		Decisions: []KNEntry{
			{Content: "Use PostgreSQL", Reason: "ACID compliance needed"},
		},
		Attempts: []KNEntry{
			{Content: "Tried MongoDB", Result: "Too complex for our use case"},
		},
	}

	var builder strings.Builder
	writeContextForSynthesis(&builder, result, 10)

	output := builder.String()

	// Verify constraints are included
	if !strings.Contains(output, "## CONSTRAINTS") {
		t.Error("Expected CONSTRAINTS section")
	}
	if !strings.Contains(output, "Must use HTTPS") {
		t.Error("Expected constraint content")
	}

	// Verify decisions are included
	if !strings.Contains(output, "## DECISIONS") {
		t.Error("Expected DECISIONS section")
	}
	if !strings.Contains(output, "Use PostgreSQL") {
		t.Error("Expected decision content")
	}

	// Verify attempts are included
	if !strings.Contains(output, "## FAILED ATTEMPTS") {
		t.Error("Expected FAILED ATTEMPTS section")
	}
	if !strings.Contains(output, "Tried MongoDB") {
		t.Error("Expected attempt content")
	}
}

func TestWriteContextForSynthesisLimit(t *testing.T) {
	// Create result with more entries than limit
	result := &KBContextResult{
		Constraints: []KNEntry{
			{Content: "Constraint 1"},
			{Content: "Constraint 2"},
			{Content: "Constraint 3"},
			{Content: "Constraint 4"},
			{Content: "Constraint 5"},
		},
	}

	var builder strings.Builder
	writeContextForSynthesis(&builder, result, 2)

	output := builder.String()

	// Should only include 2 constraints due to limit
	count := strings.Count(output, "Constraint")
	if count != 2 {
		t.Errorf("Expected 2 constraints due to limit, got %d", count)
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"How do we handle rate limiting?", "how-do-we-handle-rate-limiting"},
		{"What's our auth pattern?", "whats-our-auth-pattern"},
		{"Test   Multiple   Spaces", "test-multiple-spaces"},
		{"with_underscores_here", "with-underscores-here"},
		{"UPPERCASE test", "uppercase-test"},
		{"", ""},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateSlugLengthLimit(t *testing.T) {
	// Test that long inputs are truncated
	longInput := strings.Repeat("word ", 20)
	result := generateSlug(longInput)

	if len(result) > 50 {
		t.Errorf("Expected slug to be max 50 chars, got %d", len(result))
	}
}

func TestBuildSynthesisPrompt(t *testing.T) {
	question := "How should we handle errors?"
	context := "## CONSTRAINTS\n- Must log all errors\n"

	prompt := buildSynthesisPrompt(question, context)

	// Verify question is in prompt
	if !strings.Contains(prompt, question) {
		t.Error("Expected question in prompt")
	}

	// Verify context is in prompt
	if !strings.Contains(prompt, context) {
		t.Error("Expected context in prompt")
	}

	// Verify instructions are present
	if !strings.Contains(prompt, "Answer the question directly") {
		t.Error("Expected instructions in prompt")
	}
}

func TestReadArtifactSummary(t *testing.T) {
	// Create a temp file with investigation content
	content := `# Test Investigation

## TLDR
This is a short summary.

## Background
Some background info here.
This should be skipped.

## Conclusion
This is the conclusion.
And another line.

## Extra Section
This should be skipped too.
`
	// Write to temp file
	tmpFile, err := os.CreateTemp("", "test-inv-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Read summary
	summary, err := readArtifactSummary(tmpFile.Name())
	if err != nil {
		t.Fatalf("readArtifactSummary failed: %v", err)
	}

	// Should contain TLDR and Conclusion but not Background or Extra Section
	if !strings.Contains(summary, "## TLDR") {
		t.Error("Expected TLDR section in summary")
	}
	if !strings.Contains(summary, "This is a short summary") {
		t.Error("Expected TLDR content in summary")
	}
	if !strings.Contains(summary, "## Conclusion") {
		t.Error("Expected Conclusion section in summary")
	}

	// Background should NOT be included
	if strings.Contains(summary, "## Background") {
		t.Error("Background section should not be included")
	}
}
