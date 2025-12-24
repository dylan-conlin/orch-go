package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPatternToGlob(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "no variables",
			pattern: ".kb/investigations/*.md",
			want:    ".kb/investigations/*.md",
		},
		{
			name:    "date variable",
			pattern: ".kb/investigations/{date}-inv-*.md",
			want:    ".kb/investigations/*-inv-*.md",
		},
		{
			name:    "workspace variable",
			pattern: ".orch/workspace/{workspace}/SYNTHESIS.md",
			want:    ".orch/workspace/*/SYNTHESIS.md",
		},
		{
			name:    "beads variable",
			pattern: ".beads/issues/{beads}.json",
			want:    ".beads/issues/*.json",
		},
		{
			name:    "multiple variables",
			pattern: ".kb/investigations/{date}-{workspace}.md",
			want:    ".kb/investigations/*-*.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PatternToGlob(tt.pattern)
			if got != tt.want {
				t.Errorf("PatternToGlob(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestExtractConstraintsFromFile(t *testing.T) {
	t.Run("no constraint block", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "SPAWN_CONTEXT.md")
		content := `# SPAWN_CONTEXT

TASK: Do something

## SKILL GUIDANCE (investigation)

Some skill content without constraints.
`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		constraints, err := ExtractConstraintsFromFile(filePath)
		if err != nil {
			t.Fatalf("ExtractConstraintsFromFile failed: %v", err)
		}

		if len(constraints) != 0 {
			t.Errorf("expected 0 constraints, got %d", len(constraints))
		}
	})

	t.Run("constraint block with required and optional", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "SPAWN_CONTEXT.md")
		content := `# SPAWN_CONTEXT

TASK: Do something

## SKILL GUIDANCE (investigation)

<!-- SKILL-CONSTRAINTS -->
<!-- required: .kb/investigations/{date}-inv-*.md | Investigation file with findings -->
<!-- required: .orch/workspace/{workspace}/SYNTHESIS.md | Session synthesis document -->
<!-- optional: .kb/decisions/{date}-*.md | Promoted decision -->
<!-- /SKILL-CONSTRAINTS -->

Some more content.
`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		constraints, err := ExtractConstraintsFromFile(filePath)
		if err != nil {
			t.Fatalf("ExtractConstraintsFromFile failed: %v", err)
		}

		if len(constraints) != 3 {
			t.Fatalf("expected 3 constraints, got %d", len(constraints))
		}

		// Check first required constraint
		if constraints[0].Type != ConstraintRequired {
			t.Errorf("constraints[0].Type = %q, want %q", constraints[0].Type, ConstraintRequired)
		}
		if constraints[0].Pattern != ".kb/investigations/{date}-inv-*.md" {
			t.Errorf("constraints[0].Pattern = %q, want %q", constraints[0].Pattern, ".kb/investigations/{date}-inv-*.md")
		}
		if constraints[0].Description != "Investigation file with findings" {
			t.Errorf("constraints[0].Description = %q, want %q", constraints[0].Description, "Investigation file with findings")
		}

		// Check optional constraint
		if constraints[2].Type != ConstraintOptional {
			t.Errorf("constraints[2].Type = %q, want %q", constraints[2].Type, ConstraintOptional)
		}
	})

	t.Run("file not found returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nonexistent.md")

		constraints, err := ExtractConstraintsFromFile(filePath)
		if err != nil {
			t.Fatalf("expected no error for missing file, got: %v", err)
		}
		if constraints != nil {
			t.Errorf("expected nil constraints for missing file, got: %v", constraints)
		}
	})

	t.Run("constraint outside block is ignored", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "SPAWN_CONTEXT.md")
		content := `# SPAWN_CONTEXT

<!-- required: should-be-ignored.md | Outside block -->

<!-- SKILL-CONSTRAINTS -->
<!-- required: .kb/investigations/{date}-inv-*.md | Inside block -->
<!-- /SKILL-CONSTRAINTS -->

<!-- required: also-ignored.md | After block -->
`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		constraints, err := ExtractConstraintsFromFile(filePath)
		if err != nil {
			t.Fatalf("ExtractConstraintsFromFile failed: %v", err)
		}

		if len(constraints) != 1 {
			t.Fatalf("expected 1 constraint (inside block only), got %d", len(constraints))
		}

		if constraints[0].Pattern != ".kb/investigations/{date}-inv-*.md" {
			t.Errorf("expected pattern from inside block, got %q", constraints[0].Pattern)
		}
	})
}

func TestVerifyConstraints(t *testing.T) {
	t.Run("required constraint satisfied", func(t *testing.T) {
		// Create test directory structure
		projectDir := t.TempDir()
		kbDir := filepath.Join(projectDir, ".kb", "investigations")
		if err := os.MkdirAll(kbDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		// Create matching file
		filePath := filepath.Join(kbDir, "2025-12-23-inv-test.md")
		if err := os.WriteFile(filePath, []byte("# Investigation"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		constraints := []Constraint{
			{Type: ConstraintRequired, Pattern: ".kb/investigations/{date}-inv-*.md", Description: "Investigation file"},
		}

		result := VerifyConstraints(constraints, projectDir)

		if !result.Passed {
			t.Errorf("expected verification to pass, got errors: %v", result.Errors)
		}
		if len(result.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result.Results))
		}
		if !result.Results[0].Matched {
			t.Error("expected constraint to match")
		}
		if len(result.Results[0].MatchedFiles) != 1 {
			t.Errorf("expected 1 matched file, got %d", len(result.Results[0].MatchedFiles))
		}
	})

	t.Run("required constraint not satisfied", func(t *testing.T) {
		projectDir := t.TempDir()
		// Create .kb dir but no investigation file
		if err := os.MkdirAll(filepath.Join(projectDir, ".kb", "investigations"), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		constraints := []Constraint{
			{Type: ConstraintRequired, Pattern: ".kb/investigations/{date}-inv-*.md", Description: "Investigation file"},
		}

		result := VerifyConstraints(constraints, projectDir)

		if result.Passed {
			t.Error("expected verification to fail")
		}
		if len(result.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(result.Errors))
		}
	})

	t.Run("optional constraint not satisfied adds warning", func(t *testing.T) {
		projectDir := t.TempDir()

		constraints := []Constraint{
			{Type: ConstraintOptional, Pattern: ".kb/decisions/{date}-*.md", Description: "Optional decision"},
		}

		result := VerifyConstraints(constraints, projectDir)

		if !result.Passed {
			t.Error("expected verification to pass (optional constraint)")
		}
		if len(result.Warnings) != 1 {
			t.Errorf("expected 1 warning, got %d", len(result.Warnings))
		}
	})

	t.Run("multiple constraints mixed results", func(t *testing.T) {
		projectDir := t.TempDir()
		kbDir := filepath.Join(projectDir, ".kb", "investigations")
		if err := os.MkdirAll(kbDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		// Create investigation file but not synthesis
		if err := os.WriteFile(filepath.Join(kbDir, "2025-12-23-inv-test.md"), []byte("# Inv"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		constraints := []Constraint{
			{Type: ConstraintRequired, Pattern: ".kb/investigations/{date}-inv-*.md", Description: "Investigation"},
			{Type: ConstraintRequired, Pattern: ".orch/workspace/*/SYNTHESIS.md", Description: "Synthesis"},
			{Type: ConstraintOptional, Pattern: ".kb/decisions/*.md", Description: "Decision"},
		}

		result := VerifyConstraints(constraints, projectDir)

		if result.Passed {
			t.Error("expected verification to fail (missing synthesis)")
		}
		if len(result.Errors) != 1 {
			t.Errorf("expected 1 error (missing synthesis), got %d: %v", len(result.Errors), result.Errors)
		}
		if len(result.Warnings) != 1 {
			t.Errorf("expected 1 warning (optional decision), got %d", len(result.Warnings))
		}
	})

	t.Run("no constraints passes", func(t *testing.T) {
		projectDir := t.TempDir()

		result := VerifyConstraints(nil, projectDir)

		if !result.Passed {
			t.Error("expected empty constraints to pass")
		}
	})
}

func TestExtractConstraints(t *testing.T) {
	t.Run("extracts from workspace SPAWN_CONTEXT.md", func(t *testing.T) {
		workspace := t.TempDir()
		content := `TASK: Test

<!-- SKILL-CONSTRAINTS -->
<!-- required: test.md | Test file -->
<!-- /SKILL-CONSTRAINTS -->
`
		if err := os.WriteFile(filepath.Join(workspace, "SPAWN_CONTEXT.md"), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		constraints, err := ExtractConstraints(workspace)
		if err != nil {
			t.Fatalf("ExtractConstraints failed: %v", err)
		}

		if len(constraints) != 1 {
			t.Errorf("expected 1 constraint, got %d", len(constraints))
		}
	})
}

func TestVerifyConstraintsForCompletion(t *testing.T) {
	t.Run("end to end verification", func(t *testing.T) {
		// Setup workspace with SPAWN_CONTEXT.md
		workspace := t.TempDir()
		spawnContext := `TASK: Create investigation

<!-- SKILL-CONSTRAINTS -->
<!-- required: .kb/investigations/{date}-inv-*.md | Investigation file -->
<!-- optional: .kb/decisions/{date}-*.md | Decision file -->
<!-- /SKILL-CONSTRAINTS -->
`
		if err := os.WriteFile(filepath.Join(workspace, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
			t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
		}

		// Setup project dir with matching file
		projectDir := t.TempDir()
		kbDir := filepath.Join(projectDir, ".kb", "investigations")
		if err := os.MkdirAll(kbDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(kbDir, "2025-12-23-inv-test.md"), []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to write investigation: %v", err)
		}

		result, err := VerifyConstraintsForCompletion(workspace, projectDir)
		if err != nil {
			t.Fatalf("VerifyConstraintsForCompletion failed: %v", err)
		}

		if !result.Passed {
			t.Errorf("expected verification to pass, got errors: %v", result.Errors)
		}
		// Should have 1 warning for optional decision
		if len(result.Warnings) != 1 {
			t.Errorf("expected 1 warning for optional constraint, got %d", len(result.Warnings))
		}
	})

	t.Run("no constraints in workspace", func(t *testing.T) {
		workspace := t.TempDir()
		// SPAWN_CONTEXT without constraints
		if err := os.WriteFile(filepath.Join(workspace, "SPAWN_CONTEXT.md"), []byte("TASK: Simple task\n"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		projectDir := t.TempDir()

		result, err := VerifyConstraintsForCompletion(workspace, projectDir)
		if err != nil {
			t.Fatalf("VerifyConstraintsForCompletion failed: %v", err)
		}

		if !result.Passed {
			t.Error("expected verification to pass when no constraints")
		}
	})

	t.Run("missing workspace SPAWN_CONTEXT.md", func(t *testing.T) {
		workspace := t.TempDir()
		projectDir := t.TempDir()

		result, err := VerifyConstraintsForCompletion(workspace, projectDir)
		if err != nil {
			t.Fatalf("VerifyConstraintsForCompletion failed: %v", err)
		}

		if !result.Passed {
			t.Error("expected verification to pass when no SPAWN_CONTEXT.md")
		}
	})
}

func TestConstraintWithSimpleFolder(t *testing.T) {
	// Test the simple/ subfolder pattern from the investigation template
	t.Run("simple subfolder pattern", func(t *testing.T) {
		projectDir := t.TempDir()

		// Create simple subfolder with investigation
		simpleDir := filepath.Join(projectDir, ".kb", "investigations", "simple")
		if err := os.MkdirAll(simpleDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		// Note: the simple folder uses YYYY-MM-DD- prefix pattern
		if err := os.WriteFile(filepath.Join(simpleDir, "2025-12-23-test-topic.md"), []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		// Pattern that matches simple subfolder
		constraints := []Constraint{
			{Type: ConstraintRequired, Pattern: ".kb/investigations/simple/{date}-*.md", Description: "Simple investigation"},
		}

		result := VerifyConstraints(constraints, projectDir)

		if !result.Passed {
			t.Errorf("expected verification to pass, got errors: %v", result.Errors)
		}
	})
}
