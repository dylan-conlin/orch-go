package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDecisionReferences(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "single relative path",
			content: `Investigation addresses .kb/decisions/2026-01-09-dashboard-reliability.md
and provides a fix.`,
			expected: []string{".kb/decisions/2026-01-09-dashboard-reliability.md"},
		},
		{
			name: "multiple decision references",
			content: `This patches .kb/decisions/2025-12-20-auth-pattern.md and also
references .kb/decisions/2026-01-05-rate-limiting.md for context.`,
			expected: []string{
				".kb/decisions/2025-12-20-auth-pattern.md",
				".kb/decisions/2026-01-05-rate-limiting.md",
			},
		},
		{
			name: "absolute path",
			content: `Addresses /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-03-logging.md
with structured logging improvements.`,
			expected: []string{"/Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-03-logging.md"},
		},
		{
			name:     "no decision references",
			content:  `This investigation doesn't reference any decisions.`,
			expected: []string{},
		},
		{
			name: "decision in parentheses",
			content: `This addresses the issue (see .kb/decisions/2026-01-09-dashboard-reliability.md)
for more details.`,
			expected: []string{".kb/decisions/2026-01-09-dashboard-reliability.md"},
		},
		{
			name: "duplicate references filtered",
			content: `First mention: .kb/decisions/2026-01-09-foo.md
Second mention: .kb/decisions/2026-01-09-foo.md`,
			expected: []string{".kb/decisions/2026-01-09-foo.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := findDecisionReferences(tt.content)
			if len(refs) != len(tt.expected) {
				t.Errorf("Expected %d references, got %d: %v", len(tt.expected), len(refs), refs)
				return
			}
			for i, expected := range tt.expected {
				if refs[i] != expected {
					t.Errorf("Expected ref[%d]=%s, got %s", i, expected, refs[i])
				}
			}
		})
	}
}

func TestHasBlocksFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "has blocks frontmatter",
			content: `---
status: active
blocks:
  - keywords:
      - coaching plugin
---
# Decision`,
			expected: true,
		},
		{
			name: "no blocks frontmatter",
			content: `---
status: active
---
# Decision`,
			expected: false,
		},
		{
			name: "no frontmatter",
			content: `# Decision

Some content.`,
			expected: false,
		},
		{
			name: "blocks in content but not frontmatter",
			content: `---
status: active
---
# Decision

blocks: this is not in frontmatter`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test-decision.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			result, err := hasBlocksFrontmatter(testFile)
			if err != nil {
				t.Fatalf("hasBlocksFrontmatter() returned error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("hasBlocksFrontmatter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindDecisionsWithoutBlocksFrontmatter(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	kbDir := filepath.Join(projectDir, ".kb")
	decisionsDir := filepath.Join(kbDir, "decisions")
	workspaceDir := filepath.Join(tempDir, "workspace")

	// Create directory structure
	os.MkdirAll(decisionsDir, 0755)
	os.MkdirAll(workspaceDir, 0755)

	// Create decision with blocks
	decisionWithBlocks := filepath.Join(decisionsDir, "2026-01-28-has-blocks.md")
	os.WriteFile(decisionWithBlocks, []byte(`---
status: active
blocks:
  - keywords:
      - test
---
# Decision with blocks`), 0644)

	// Create decision without blocks
	decisionWithoutBlocks := filepath.Join(decisionsDir, "2026-01-28-no-blocks.md")
	os.WriteFile(decisionWithoutBlocks, []byte(`---
status: active
---
# Decision without blocks`), 0644)

	tests := []struct {
		name              string
		synthesisContent  string
		expectedCount     int
		expectedFilenames []string
	}{
		{
			name:              "no decision references",
			synthesisContent:  "No decisions referenced here.",
			expectedCount:     0,
			expectedFilenames: []string{},
		},
		{
			name:              "decision with blocks",
			synthesisContent:  "References .kb/decisions/2026-01-28-has-blocks.md",
			expectedCount:     0,
			expectedFilenames: []string{},
		},
		{
			name:              "decision without blocks",
			synthesisContent:  "References .kb/decisions/2026-01-28-no-blocks.md",
			expectedCount:     1,
			expectedFilenames: []string{"2026-01-28-no-blocks.md"},
		},
		{
			name: "mixed decisions",
			synthesisContent: `References .kb/decisions/2026-01-28-has-blocks.md and
.kb/decisions/2026-01-28-no-blocks.md`,
			expectedCount:     1,
			expectedFilenames: []string{"2026-01-28-no-blocks.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create SYNTHESIS.md
			synthesisPath := filepath.Join(workspaceDir, "SYNTHESIS.md")
			os.WriteFile(synthesisPath, []byte(tt.synthesisContent), 0644)

			// Run detection
			results, err := FindDecisionsWithoutBlocksFrontmatter(workspaceDir, projectDir)
			if err != nil {
				t.Fatalf("FindDecisionsWithoutBlocksFrontmatter() returned error: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			// Check filenames
			for i, expectedFilename := range tt.expectedFilenames {
				if i >= len(results) {
					t.Errorf("Missing expected result: %s", expectedFilename)
					continue
				}
				if results[i].Filename != expectedFilename {
					t.Errorf("Expected filename %s, got %s", expectedFilename, results[i].Filename)
				}
			}
		})
	}
}

func TestNormalizeDecisionPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		projectDir  string
		expectation func(result string) bool
	}{
		{
			name:       "absolute path unchanged",
			path:       "/Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-09-foo.md",
			projectDir: "/Users/dylanconlin/Documents/personal/orch-go",
			expectation: func(result string) bool {
				return result == "/Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-09-foo.md"
			},
		},
		{
			name:       "relative path becomes absolute",
			path:       ".kb/decisions/2026-01-09-foo.md",
			projectDir: "/Users/dylanconlin/Documents/personal/orch-go",
			expectation: func(result string) bool {
				// Should be joined with project dir if file exists
				return filepath.IsAbs(result) || result == "2026-01-09-foo.md"
			},
		},
		{
			name:       "basename fallback for non-existent",
			path:       ".kb/decisions/2026-01-09-nonexistent.md",
			projectDir: "/tmp",
			expectation: func(result string) bool {
				return result == "2026-01-09-nonexistent.md"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeDecisionPath(tt.path, tt.projectDir)
			if !tt.expectation(result) {
				t.Errorf("normalizeDecisionPath(%s, %s) = %s, expectation failed",
					tt.path, tt.projectDir, result)
			}
		})
	}
}

func TestVerifyDecisionPatchCount(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	kbDir := filepath.Join(projectDir, ".kb")
	investigationsDir := filepath.Join(kbDir, "investigations")
	decisionsDir := filepath.Join(kbDir, "decisions")
	workspaceDir := filepath.Join(tempDir, "workspace")

	// Create directory structure
	os.MkdirAll(investigationsDir, 0755)
	os.MkdirAll(decisionsDir, 0755)
	os.MkdirAll(workspaceDir, 0755)

	// Create a decision file
	decisionPath := filepath.Join(decisionsDir, "2026-01-09-foo-decision.md")
	os.WriteFile(decisionPath, []byte("# Decision: Foo\n\nWe decided to do foo."), 0644)

	tests := []struct {
		name                  string
		synthesisContent      string
		investigationContents map[string]string // filename -> content
		expectPassed          bool
		expectError           bool
		expectWarning         bool
	}{
		{
			name:             "no decision references",
			synthesisContent: "This investigation doesn't reference any decisions.",
			expectPassed:     true,
			expectError:      false,
			expectWarning:    false,
		},
		{
			name:             "first patch (no existing patches)",
			synthesisContent: "Addresses .kb/decisions/2026-01-09-foo-decision.md",
			expectPassed:     true,
			expectError:      false,
			expectWarning:    false,
		},
		{
			name:             "second patch (one existing patch)",
			synthesisContent: "Addresses .kb/decisions/2026-01-09-foo-decision.md",
			investigationContents: map[string]string{
				"2026-01-08-patch-1.md": "Patches .kb/decisions/2026-01-09-foo-decision.md",
			},
			expectPassed:  true,
			expectError:   false,
			expectWarning: true, // Warning on 2nd patch
		},
		{
			name:             "third patch (two existing patches)",
			synthesisContent: "Addresses .kb/decisions/2026-01-09-foo-decision.md",
			investigationContents: map[string]string{
				"2026-01-08-patch-1.md": "Patches .kb/decisions/2026-01-09-foo-decision.md",
				"2026-01-08-patch-2.md": "Also patches 2026-01-09-foo-decision.md",
			},
			expectPassed:  true,
			expectError:   false,
			expectWarning: true,
		},
		{
			name:             "fourth patch (three existing patches - BLOCKED)",
			synthesisContent: "Addresses .kb/decisions/2026-01-09-foo-decision.md",
			investigationContents: map[string]string{
				"2026-01-08-patch-1.md": "Patches .kb/decisions/2026-01-09-foo-decision.md",
				"2026-01-08-patch-2.md": "Also patches 2026-01-09-foo-decision.md",
				"2026-01-08-patch-3.md": "Third patch to 2026-01-09-foo-decision.md",
			},
			expectPassed:  false,
			expectError:   true,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean investigations directory
			os.RemoveAll(investigationsDir)
			os.MkdirAll(investigationsDir, 0755)

			// Create investigation files
			for filename, content := range tt.investigationContents {
				invPath := filepath.Join(investigationsDir, filename)
				os.WriteFile(invPath, []byte(content), 0644)
			}

			// Create SYNTHESIS.md
			synthesisPath := filepath.Join(workspaceDir, "SYNTHESIS.md")
			os.WriteFile(synthesisPath, []byte(tt.synthesisContent), 0644)

			// Run verification
			result := VerifyDecisionPatchCount(workspaceDir, projectDir)

			// Check result
			if tt.expectPassed {
				if result != nil && !result.Passed {
					t.Errorf("Expected passed=true, got passed=false with errors: %v", result.Errors)
				}
			} else {
				if result == nil || result.Passed {
					t.Errorf("Expected passed=false, got passed=true")
				}
			}

			if tt.expectError {
				if result == nil || len(result.Errors) == 0 {
					t.Errorf("Expected errors, got none")
				}
			}

			if tt.expectWarning {
				if result == nil || len(result.Warnings) == 0 {
					t.Errorf("Expected warnings, got none")
				}
			}
		})
	}
}
