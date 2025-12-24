package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPhasesFromFile(t *testing.T) {
	t.Run("no phase block", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "SPAWN_CONTEXT.md")
		content := `# SPAWN_CONTEXT

TASK: Do something

## SKILL GUIDANCE (feature-impl)

Some skill content without phases.
`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		phases, err := ExtractPhasesFromFile(filePath)
		if err != nil {
			t.Fatalf("ExtractPhasesFromFile failed: %v", err)
		}

		if len(phases) != 0 {
			t.Errorf("expected 0 phases, got %d", len(phases))
		}
	})

	t.Run("phase block with required and optional phases", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "SPAWN_CONTEXT.md")
		content := `# SPAWN_CONTEXT

TASK: Do something

## SKILL GUIDANCE (feature-impl)

<!-- SKILL-PHASES -->
<!-- phase: investigation | required: false -->
<!-- phase: design | required: false -->
<!-- phase: implementation | required: true -->
<!-- phase: validation | required: true -->
<!-- phase: complete | required: true -->
<!-- /SKILL-PHASES -->

Some more content.
`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		phases, err := ExtractPhasesFromFile(filePath)
		if err != nil {
			t.Fatalf("ExtractPhasesFromFile failed: %v", err)
		}

		if len(phases) != 5 {
			t.Fatalf("expected 5 phases, got %d", len(phases))
		}

		// Check first optional phase
		if phases[0].Name != "investigation" {
			t.Errorf("phases[0].Name = %q, want %q", phases[0].Name, "investigation")
		}
		if phases[0].Required {
			t.Errorf("phases[0].Required = true, want false")
		}

		// Check first required phase
		if phases[2].Name != "implementation" {
			t.Errorf("phases[2].Name = %q, want %q", phases[2].Name, "implementation")
		}
		if !phases[2].Required {
			t.Errorf("phases[2].Required = false, want true")
		}

		// Check complete phase
		if phases[4].Name != "complete" {
			t.Errorf("phases[4].Name = %q, want %q", phases[4].Name, "complete")
		}
		if !phases[4].Required {
			t.Errorf("phases[4].Required = false, want true")
		}
	})

	t.Run("file not found returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nonexistent.md")

		phases, err := ExtractPhasesFromFile(filePath)
		if err != nil {
			t.Fatalf("expected no error for missing file, got: %v", err)
		}
		if phases != nil {
			t.Errorf("expected nil phases for missing file, got: %v", phases)
		}
	})

	t.Run("phase outside block is ignored", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "SPAWN_CONTEXT.md")
		content := `# SPAWN_CONTEXT

<!-- phase: ignored | required: true -->

<!-- SKILL-PHASES -->
<!-- phase: implementation | required: true -->
<!-- /SKILL-PHASES -->

<!-- phase: also_ignored | required: true -->
`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		phases, err := ExtractPhasesFromFile(filePath)
		if err != nil {
			t.Fatalf("ExtractPhasesFromFile failed: %v", err)
		}

		if len(phases) != 1 {
			t.Fatalf("expected 1 phase (inside block only), got %d", len(phases))
		}

		if phases[0].Name != "implementation" {
			t.Errorf("expected phase from inside block, got %q", phases[0].Name)
		}
	})
}

func TestExtractReportedPhases(t *testing.T) {
	t.Run("extracts phases from comments in order", func(t *testing.T) {
		comments := []Comment{
			{Text: "Phase: Planning - analyzing codebase"},
			{Text: "Some other comment"},
			{Text: "Phase: Implementation - writing code"},
			{Text: "Phase: Validation - running tests"},
			{Text: "Phase: Complete - all done"},
		}

		phases := ExtractReportedPhases(comments)

		expected := []string{"planning", "implementation", "validation", "complete"}
		if len(phases) != len(expected) {
			t.Fatalf("expected %d phases, got %d: %v", len(expected), len(phases), phases)
		}

		for i, exp := range expected {
			if phases[i] != exp {
				t.Errorf("phases[%d] = %q, want %q", i, phases[i], exp)
			}
		}
	})

	t.Run("deduplicates phases", func(t *testing.T) {
		comments := []Comment{
			{Text: "Phase: Planning - first"},
			{Text: "Phase: Planning - duplicate"},
			{Text: "Phase: Implementation - work"},
		}

		phases := ExtractReportedPhases(comments)

		// Should only have 2 unique phases
		if len(phases) != 2 {
			t.Errorf("expected 2 unique phases, got %d: %v", len(phases), phases)
		}
	})

	t.Run("handles various formats", func(t *testing.T) {
		comments := []Comment{
			{Text: "Phase: Planning"},                 // No summary
			{Text: "Phase: Design – with em dash"},    // Em dash
			{Text: "Phase: Implementation—no spaces"}, // No spaces around dash
			{Text: "PHASE: COMPLETE - uppercase"},     // Uppercase
		}

		phases := ExtractReportedPhases(comments)

		expected := []string{"planning", "design", "implementation", "complete"}
		if len(phases) != len(expected) {
			t.Fatalf("expected %d phases, got %d: %v", len(expected), len(phases), phases)
		}
	})

	t.Run("empty comments returns empty slice", func(t *testing.T) {
		phases := ExtractReportedPhases([]Comment{})

		if len(phases) != 0 {
			t.Errorf("expected 0 phases, got %d", len(phases))
		}
	})
}

func TestVerifyPhaseGates(t *testing.T) {
	t.Run("all required phases reported", func(t *testing.T) {
		requiredPhases := []Phase{
			{Name: "implementation", Required: true},
			{Name: "validation", Required: true},
			{Name: "complete", Required: true},
		}

		comments := []Comment{
			{Text: "Phase: Implementation - code written"},
			{Text: "Phase: Validation - tests pass"},
			{Text: "Phase: Complete - done"},
		}

		result := VerifyPhaseGates(requiredPhases, comments)

		if !result.Passed {
			t.Errorf("expected verification to pass, got errors: %v", result.Errors)
		}
		if len(result.MissingPhases) != 0 {
			t.Errorf("expected no missing phases, got: %v", result.MissingPhases)
		}
	})

	t.Run("missing required phase fails", func(t *testing.T) {
		requiredPhases := []Phase{
			{Name: "investigation", Required: true},
			{Name: "implementation", Required: true},
			{Name: "complete", Required: true},
		}

		comments := []Comment{
			{Text: "Phase: Implementation - code written"},
			{Text: "Phase: Complete - done"},
		}

		result := VerifyPhaseGates(requiredPhases, comments)

		if result.Passed {
			t.Error("expected verification to fail")
		}
		if len(result.MissingPhases) != 1 {
			t.Fatalf("expected 1 missing phase, got %d", len(result.MissingPhases))
		}
		if result.MissingPhases[0] != "investigation" {
			t.Errorf("expected missing phase 'investigation', got %q", result.MissingPhases[0])
		}
	})

	t.Run("optional phases don't affect result", func(t *testing.T) {
		requiredPhases := []Phase{
			{Name: "investigation", Required: false}, // Optional
			{Name: "design", Required: false},        // Optional
			{Name: "implementation", Required: true},
			{Name: "complete", Required: true},
		}

		comments := []Comment{
			{Text: "Phase: Implementation - code written"},
			{Text: "Phase: Complete - done"},
		}

		result := VerifyPhaseGates(requiredPhases, comments)

		if !result.Passed {
			t.Errorf("expected verification to pass (optional phases not required), got errors: %v", result.Errors)
		}
	})

	t.Run("no phases defined passes", func(t *testing.T) {
		result := VerifyPhaseGates(nil, []Comment{{Text: "some comment"}})

		if !result.Passed {
			t.Error("expected empty phases to pass")
		}
	})

	t.Run("case insensitive phase matching", func(t *testing.T) {
		requiredPhases := []Phase{
			{Name: "Implementation", Required: true},
			{Name: "COMPLETE", Required: true},
		}

		comments := []Comment{
			{Text: "Phase: implementation - code written"},
			{Text: "Phase: Complete - done"},
		}

		result := VerifyPhaseGates(requiredPhases, comments)

		if !result.Passed {
			t.Errorf("expected case-insensitive matching to pass, got errors: %v", result.Errors)
		}
	})

	t.Run("multiple missing phases reported", func(t *testing.T) {
		requiredPhases := []Phase{
			{Name: "investigation", Required: true},
			{Name: "design", Required: true},
			{Name: "implementation", Required: true},
			{Name: "complete", Required: true},
		}

		comments := []Comment{
			{Text: "Phase: Complete - done"},
		}

		result := VerifyPhaseGates(requiredPhases, comments)

		if result.Passed {
			t.Error("expected verification to fail")
		}
		if len(result.MissingPhases) != 3 {
			t.Errorf("expected 3 missing phases, got %d: %v", len(result.MissingPhases), result.MissingPhases)
		}
	})
}

func TestExtractPhases(t *testing.T) {
	t.Run("extracts from workspace SPAWN_CONTEXT.md", func(t *testing.T) {
		workspace := t.TempDir()
		content := `TASK: Test

<!-- SKILL-PHASES -->
<!-- phase: implementation | required: true -->
<!-- phase: complete | required: true -->
<!-- /SKILL-PHASES -->
`
		if err := os.WriteFile(filepath.Join(workspace, "SPAWN_CONTEXT.md"), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		phases, err := ExtractPhases(workspace)
		if err != nil {
			t.Fatalf("ExtractPhases failed: %v", err)
		}

		if len(phases) != 2 {
			t.Errorf("expected 2 phases, got %d", len(phases))
		}
	})
}

// Note: TestVerifyPhaseGatesForCompletion requires mocking beads comments,
// which is complex. The integration is tested via the individual functions above.
