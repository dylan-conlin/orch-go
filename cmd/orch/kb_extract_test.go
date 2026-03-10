package main

import (
	"os"
	"strings"
	"testing"
)

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
