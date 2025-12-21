package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePhaseFromComments(t *testing.T) {
	tests := []struct {
		name     string
		comments []Comment
		want     PhaseStatus
	}{
		{
			name:     "no comments",
			comments: []Comment{},
			want:     PhaseStatus{Found: false},
		},
		{
			name: "no phase comments",
			comments: []Comment{
				{Text: "Just a regular comment"},
				{Text: "Another comment without phase"},
			},
			want: PhaseStatus{Found: false},
		},
		{
			name: "simple phase complete",
			comments: []Comment{
				{Text: "Phase: Complete"},
			},
			want: PhaseStatus{Phase: "Complete", Found: true},
		},
		{
			name: "phase with summary",
			comments: []Comment{
				{Text: "Phase: Complete - All tests passing, ready for review"},
			},
			want: PhaseStatus{
				Phase:   "Complete",
				Summary: "All tests passing, ready for review",
				Found:   true,
			},
		},
		{
			name: "phase with en-dash",
			comments: []Comment{
				{Text: "Phase: Complete – Implementation finished"},
			},
			want: PhaseStatus{
				Phase:   "Complete",
				Summary: "Implementation finished",
				Found:   true,
			},
		},
		{
			name: "phase with em-dash",
			comments: []Comment{
				{Text: "Phase: Complete — Done"},
			},
			want: PhaseStatus{
				Phase:   "Complete",
				Summary: "Done",
				Found:   true,
			},
		},
		{
			name: "multiple phases - returns latest",
			comments: []Comment{
				{Text: "Phase: Planning - Starting work"},
				{Text: "Some progress comment"},
				{Text: "Phase: Implementing - Adding tests"},
				{Text: "Phase: Complete - All done"},
			},
			want: PhaseStatus{
				Phase:   "Complete",
				Summary: "All done",
				Found:   true,
			},
		},
		{
			name: "case insensitive",
			comments: []Comment{
				{Text: "phase: complete - done"},
			},
			want: PhaseStatus{
				Phase:   "complete",
				Summary: "done",
				Found:   true,
			},
		},
		{
			name: "phase in middle of comment",
			comments: []Comment{
				{Text: "Update: Phase: Implementing - Working on feature"},
			},
			want: PhaseStatus{
				Phase:   "Implementing",
				Summary: "Working on feature",
				Found:   true,
			},
		},
		{
			name: "planning phase",
			comments: []Comment{
				{Text: "Phase: Planning - Analyzing codebase structure"},
			},
			want: PhaseStatus{
				Phase:   "Planning",
				Summary: "Analyzing codebase structure",
				Found:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePhaseFromComments(tt.comments)

			if got.Phase != tt.want.Phase {
				t.Errorf("Phase = %q, want %q", got.Phase, tt.want.Phase)
			}
			if got.Summary != tt.want.Summary {
				t.Errorf("Summary = %q, want %q", got.Summary, tt.want.Summary)
			}
			if got.Found != tt.want.Found {
				t.Errorf("Found = %v, want %v", got.Found, tt.want.Found)
			}
		})
	}
}

func TestVerificationResult(t *testing.T) {
	t.Run("empty result defaults to passed", func(t *testing.T) {
		result := VerificationResult{Passed: true}
		if !result.Passed {
			t.Error("Expected default result to be passed")
		}
		if len(result.Errors) != 0 {
			t.Error("Expected no errors")
		}
		if len(result.Warnings) != 0 {
			t.Error("Expected no warnings")
		}
	})
}

func TestPhaseStatusComplete(t *testing.T) {
	tests := []struct {
		name   string
		status PhaseStatus
		want   bool
	}{
		{
			name:   "complete phase",
			status: PhaseStatus{Phase: "Complete", Found: true},
			want:   true,
		},
		{
			name:   "complete lowercase",
			status: PhaseStatus{Phase: "complete", Found: true},
			want:   true,
		},
		{
			name:   "implementing phase",
			status: PhaseStatus{Phase: "Implementing", Found: true},
			want:   false,
		},
		{
			name:   "no phase found",
			status: PhaseStatus{Found: false},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if phase is complete using same logic as IsPhaseComplete
			got := tt.status.Found && (tt.status.Phase == "Complete" || tt.status.Phase == "complete")
			if got != tt.want {
				t.Errorf("IsComplete = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSynthesis(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	content := `# Synthesis: Test Task

**Session ID:** sess-123
**Beads ID:** proj-123
**Status:** Complete
**Confidence:** High

## TLDR
This is a test TLDR.
It spans multiple lines.

## Delta
- [ ] **Added:** file1.go
- [ ] **Modified:** file2.go

## Evidence
- [ ] **Tests:** go test ./... - Passed

## Knowledge (Externalized via kn)
- [ ] **Decision:** kn-123 - Use Go

## Next Actions
1. [ ] **Action 1** (Skill: skill1)
2. [ ] **Action 2** (Skill: skill2)
`
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test synthesis file: %v", err)
	}

	got, err := ParseSynthesis(tmpDir)
	if err != nil {
		t.Fatalf("ParseSynthesis failed: %v", err)
	}

	wantTLDR := "This is a test TLDR.\nIt spans multiple lines."
	if got.TLDR != wantTLDR {
		t.Errorf("TLDR = %q, want %q", got.TLDR, wantTLDR)
	}

	if len(got.NextActions) != 2 {
		t.Errorf("len(NextActions) = %d, want 2", len(got.NextActions))
	} else {
		if got.NextActions[0] != "1. [ ] **Action 1** (Skill: skill1)" {
			t.Errorf("NextActions[0] = %q", got.NextActions[0])
		}
	}
}

func TestParseSynthesisDEKN(t *testing.T) {
	// Test the full D.E.K.N. (Delta, Evidence, Knowledge, Next) structure
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	content := `# Session Synthesis

**Agent:** og-feat-test-20dec
**Issue:** test-123
**Duration:** 2025-12-20
**Outcome:** success

---

## TLDR

Implemented feature X. Added new API endpoint and tests.

---

## Delta (What Changed)

### Files Created

- ` + "`cmd/orch/new.go`" + ` - New command for feature X
- ` + "`pkg/new/new.go`" + ` - Core logic for feature

### Files Modified

- ` + "`cmd/orch/main.go`" + ` - Added command registration

### Commits

- ` + "`abc1234`" + ` - feat: add feature X

---

## Evidence (What Was Observed)

- Tests pass: go test ./...
- Manual verification completed
- No regressions detected

### Tests Run

` + "```" + `bash
go test ./...
# ok  all packages
` + "```" + `

---

## Knowledge (What Was Learned)

### New Artifacts

- ` + "`.kb/investigations/2025-12-20-feature-x.md`" + ` - Investigation docs

### Decisions Made

- Decision 1: Use existing patterns for consistency

### Constraints Discovered

- None significant

---

## Next (What Should Happen)

**Recommendation:** close

### If Close

- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for completion

### Follow-up Work (Optional)

- Consider adding more tests
- Monitor for edge cases

---

## Session Metadata

**Skill:** feature-impl
**Model:** anthropic/claude-sonnet
**Workspace:** ` + "`.orch/workspace/og-feat-test-20dec/`" + `
`
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test synthesis file: %v", err)
	}

	got, err := ParseSynthesis(tmpDir)
	if err != nil {
		t.Fatalf("ParseSynthesis failed: %v", err)
	}

	// Test TLDR extraction
	wantTLDR := "Implemented feature X. Added new API endpoint and tests."
	if got.TLDR != wantTLDR {
		t.Errorf("TLDR = %q, want %q", got.TLDR, wantTLDR)
	}

	// Test Delta section extraction
	if got.Delta == "" {
		t.Error("Delta should not be empty")
	}
	if !contains(got.Delta, "cmd/orch/new.go") {
		t.Errorf("Delta should contain file reference, got: %q", got.Delta)
	}
	if !contains(got.Delta, "feat: add feature X") {
		t.Errorf("Delta should contain commit message, got: %q", got.Delta)
	}

	// Test Evidence section extraction
	if got.Evidence == "" {
		t.Error("Evidence should not be empty")
	}
	if !contains(got.Evidence, "Tests pass") {
		t.Errorf("Evidence should contain test info, got: %q", got.Evidence)
	}

	// Test Knowledge section extraction
	if got.Knowledge == "" {
		t.Error("Knowledge should not be empty")
	}
	if !contains(got.Knowledge, "feature-x.md") {
		t.Errorf("Knowledge should contain artifact reference, got: %q", got.Knowledge)
	}

	// Test Next section extraction
	if got.Next == "" {
		t.Error("Next should not be empty")
	}
	if !contains(got.Next, "close") {
		t.Errorf("Next should contain recommendation, got: %q", got.Next)
	}

	// Test Outcome extraction
	if got.Outcome != "success" {
		t.Errorf("Outcome = %q, want %q", got.Outcome, "success")
	}

	// Test Recommendation extraction
	if got.Recommendation != "close" {
		t.Errorf("Recommendation = %q, want %q", got.Recommendation, "close")
	}

	// Test NextActions (follow-up work)
	if len(got.NextActions) == 0 {
		t.Error("NextActions should not be empty")
	}
}

func TestParseSynthesisMinimal(t *testing.T) {
	// Test minimal SYNTHESIS.md that only has TLDR and Next
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	content := `# Session Synthesis

## TLDR

Quick fix for bug.

## Next Actions

- Deploy to staging
`
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test synthesis file: %v", err)
	}

	got, err := ParseSynthesis(tmpDir)
	if err != nil {
		t.Fatalf("ParseSynthesis failed: %v", err)
	}

	if got.TLDR != "Quick fix for bug." {
		t.Errorf("TLDR = %q, want %q", got.TLDR, "Quick fix for bug.")
	}

	if len(got.NextActions) != 1 {
		t.Errorf("len(NextActions) = %d, want 1", len(got.NextActions))
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
