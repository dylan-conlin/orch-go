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

// Tests for kb extract functionality

func TestDetermineTargetDir(t *testing.T) {
	tests := []struct {
		name           string
		artifactPath   string
		targetProject  string
		expectedSuffix string
	}{
		{
			name:           "investigation in .kb/investigations",
			artifactPath:   "/home/user/project/.kb/investigations/2025-01-01-test.md",
			targetProject:  "/home/user/target",
			expectedSuffix: ".kb/investigations",
		},
		{
			name:           "decision in .kb/decisions",
			artifactPath:   "/home/user/project/.kb/decisions/2025-01-01-choice.md",
			targetProject:  "/home/user/target",
			expectedSuffix: ".kb/decisions",
		},
		{
			name:           "nested directory",
			artifactPath:   "/home/user/project/.kb/investigations/simple/2025-01-01-test.md",
			targetProject:  "/home/user/target",
			expectedSuffix: ".kb/investigations/simple",
		},
		{
			name:           "not in .kb directory",
			artifactPath:   "/home/user/project/docs/readme.md",
			targetProject:  "/home/user/target",
			expectedSuffix: ".kb/extracted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := determineTargetDir(tt.artifactPath, tt.targetProject)
			if err != nil {
				t.Fatalf("determineTargetDir failed: %v", err)
			}
			if !strings.HasSuffix(result, tt.expectedSuffix) {
				t.Errorf("Expected suffix %q, got %q", tt.expectedSuffix, result)
			}
		})
	}
}

func TestGetProjectName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/my-project/.kb/decisions/foo.md", "my-project"},
		{"/Users/dylan/orch-go/.kb/investigations/bar.md", "orch-go"},
		{"/path/to/skillc/.kb/decisions/test.md", "skillc"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := getProjectName(tt.path)
			if result != tt.expected {
				t.Errorf("getProjectName(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestAddLineageHeader(t *testing.T) {
	t.Run("content without frontmatter", func(t *testing.T) {
		content := "# My Decision\n\nSome content here."
		result := addLineageHeader(content, "/source/project/.kb/decisions/test.md", "source-project")

		// Should have lineage comment at the beginning
		if !strings.HasPrefix(result, "<!-- Lineage metadata") {
			t.Error("Expected lineage comment at beginning")
		}
		if !strings.Contains(result, "extracted-from: /source/project/.kb/decisions/test.md") {
			t.Error("Expected extracted-from path")
		}
		if !strings.Contains(result, "source-project: source-project") {
			t.Error("Expected source-project")
		}
		// Original content should still be there
		if !strings.Contains(result, "# My Decision") {
			t.Error("Expected original content to be preserved")
		}
	})

	t.Run("content with YAML frontmatter", func(t *testing.T) {
		content := `---
date: "2025-01-01"
status: "Accepted"
---

# My Decision

Some content here.`
		result := addLineageHeader(content, "/source/.kb/decisions/test.md", "source")

		// Should start with frontmatter
		if !strings.HasPrefix(result, "---") {
			t.Error("Expected frontmatter to be preserved at beginning")
		}
		// Lineage should be after frontmatter
		if !strings.Contains(result, "<!-- Lineage metadata") {
			t.Error("Expected lineage comment")
		}
		// Original content should be preserved
		if !strings.Contains(result, "# My Decision") {
			t.Error("Expected original content to be preserved")
		}
	})
}

func TestAddExtractedToReference(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test-source-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	originalContent := "# Original Content\n\nSome text here."
	if _, err := tmpFile.WriteString(originalContent); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Add extracted-to reference
	err = addExtractedToReference(tmpFile.Name(), "/target/.kb/decisions/test.md", "target-project")
	if err != nil {
		t.Fatalf("addExtractedToReference failed: %v", err)
	}

	// Read back and verify
	newContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(newContent)
	if !strings.Contains(content, "<!-- extracted-to:") {
		t.Error("Expected extracted-to comment")
	}
	if !strings.Contains(content, "/target/.kb/decisions/test.md") {
		t.Error("Expected target path in extracted-to comment")
	}
	if !strings.Contains(content, "project: target-project") {
		t.Error("Expected project name in extracted-to comment")
	}
	// Original content should be preserved
	if !strings.Contains(content, "# Original Content") {
		t.Error("Expected original content to be preserved")
	}
}

func TestResolveArtifactPath(t *testing.T) {
	t.Run("absolute path unchanged", func(t *testing.T) {
		path := "/absolute/path/to/file.md"
		result, err := resolveArtifactPath(path)
		if err != nil {
			t.Fatalf("resolveArtifactPath failed: %v", err)
		}
		if result != path {
			t.Errorf("Expected %q, got %q", path, result)
		}
	})

	t.Run("relative path resolved", func(t *testing.T) {
		path := "relative/file.md"
		result, err := resolveArtifactPath(path)
		if err != nil {
			t.Fatalf("resolveArtifactPath failed: %v", err)
		}
		if !strings.HasSuffix(result, path) {
			t.Errorf("Expected suffix %q in %q", path, result)
		}
		if !strings.HasPrefix(result, "/") {
			t.Error("Expected absolute path")
		}
	})
}

// Tests for ecosystem filtering (--global post-filter)

func TestExtractProjectNameFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/Users/dylan/Documents/personal/orch-go/.kb/investigations/foo.md", "orch-go"},
		{"/Users/dylan/orch-cli/.kb/decisions/bar.md", "orch-cli"},
		{"/home/user/beads/.kb/investigations/test.md", "beads"},
		{"/path/to/skillc/.kb/decisions/decision.md", "skillc"},
		{"/path/to/random-project/.kb/investigations/inv.md", "random-project"},
		// Fallback case - no .kb/ in path
		{"/path/to/dir/file.md", "dir"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractProjectNameFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("extractProjectNameFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFilterArtifactsToEcosystem(t *testing.T) {
	artifacts := []KBArtifact{
		{Name: "inv1", Path: "/Users/dylan/orch-go/.kb/investigations/inv1.md", Title: "Orch-go Investigation"},
		{Name: "inv2", Path: "/Users/dylan/random-project/.kb/investigations/inv2.md", Title: "Random Investigation"},
		{Name: "inv3", Path: "/Users/dylan/beads/.kb/investigations/inv3.md", Title: "Beads Investigation"},
		{Name: "inv4", Path: "/Users/dylan/some-app/.kb/investigations/inv4.md", Title: "Some App Investigation"},
		{Name: "inv5", Path: "/Users/dylan/skillc/.kb/decisions/dec1.md", Title: "Skillc Decision"},
	}

	filtered := filterArtifactsToEcosystem(artifacts)

	// Should only include orch-go, beads, skillc (ecosystem repos)
	if len(filtered) != 3 {
		t.Errorf("Expected 3 artifacts after filtering, got %d", len(filtered))
	}

	// Check that correct ones are included
	names := make(map[string]bool)
	for _, a := range filtered {
		names[a.Name] = true
	}

	if !names["inv1"] {
		t.Error("Expected inv1 (orch-go) to be included")
	}
	if !names["inv3"] {
		t.Error("Expected inv3 (beads) to be included")
	}
	if !names["inv5"] {
		t.Error("Expected inv5 (skillc) to be included")
	}
	if names["inv2"] {
		t.Error("Expected inv2 (random-project) to be excluded")
	}
	if names["inv4"] {
		t.Error("Expected inv4 (some-app) to be excluded")
	}
}

func TestFilterArtifactsToEcosystemEmpty(t *testing.T) {
	// Empty input should return empty
	result := filterArtifactsToEcosystem([]KBArtifact{})
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}

	// Nil input should return nil
	result = filterArtifactsToEcosystem(nil)
	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

func TestFilterToEcosystem(t *testing.T) {
	input := &KBContextResult{
		Constraints: []KNEntry{{Content: "constraint1"}},
		Decisions:   []KNEntry{{Content: "decision1"}},
		Attempts:    []KNEntry{{Content: "attempt1"}},
		Questions:   []KNEntry{{Content: "question1"}},
		Investigations: []KBArtifact{
			{Name: "inv1", Path: "/Users/dylan/orch-go/.kb/investigations/inv1.md"},
			{Name: "inv2", Path: "/Users/dylan/random/.kb/investigations/inv2.md"},
		},
		KBDecisions: []KBArtifact{
			{Name: "dec1", Path: "/Users/dylan/beads/.kb/decisions/dec1.md"},
			{Name: "dec2", Path: "/Users/dylan/other/.kb/decisions/dec2.md"},
		},
	}

	result := filterToEcosystem(input)

	// kn entries should be unchanged (not filtered)
	if len(result.Constraints) != 1 {
		t.Error("Constraints should not be filtered")
	}
	if len(result.Decisions) != 1 {
		t.Error("Decisions should not be filtered")
	}
	if len(result.Attempts) != 1 {
		t.Error("Attempts should not be filtered")
	}
	if len(result.Questions) != 1 {
		t.Error("Questions should not be filtered")
	}

	// Investigations should be filtered to ecosystem only
	if len(result.Investigations) != 1 {
		t.Errorf("Expected 1 investigation after filtering, got %d", len(result.Investigations))
	}
	if result.Investigations[0].Name != "inv1" {
		t.Error("Expected inv1 (orch-go) to remain")
	}

	// KBDecisions should be filtered to ecosystem only
	if len(result.KBDecisions) != 1 {
		t.Errorf("Expected 1 decision after filtering, got %d", len(result.KBDecisions))
	}
	if result.KBDecisions[0].Name != "dec1" {
		t.Error("Expected dec1 (beads) to remain")
	}
}

func TestFilterToEcosystemNil(t *testing.T) {
	result := filterToEcosystem(nil)
	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}
