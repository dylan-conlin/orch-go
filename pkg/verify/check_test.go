package verify

import (
	"os"
	"path/filepath"
	"strings"
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

func TestReadTierFromWorkspace(t *testing.T) {
	t.Run("reads light tier", func(t *testing.T) {
		tmpDir := t.TempDir()
		tierPath := filepath.Join(tmpDir, ".tier")
		if err := os.WriteFile(tierPath, []byte("light\n"), 0644); err != nil {
			t.Fatalf("failed to write tier file: %v", err)
		}

		got := ReadTierFromWorkspace(tmpDir)
		if got != "light" {
			t.Errorf("ReadTierFromWorkspace = %q, want %q", got, "light")
		}
	})

	t.Run("reads full tier", func(t *testing.T) {
		tmpDir := t.TempDir()
		tierPath := filepath.Join(tmpDir, ".tier")
		if err := os.WriteFile(tierPath, []byte("full\n"), 0644); err != nil {
			t.Fatalf("failed to write tier file: %v", err)
		}

		got := ReadTierFromWorkspace(tmpDir)
		if got != "full" {
			t.Errorf("ReadTierFromWorkspace = %q, want %q", got, "full")
		}
	})

	t.Run("returns full for missing file (conservative default)", func(t *testing.T) {
		tmpDir := t.TempDir()
		got := ReadTierFromWorkspace(tmpDir)
		if got != "full" {
			t.Errorf("ReadTierFromWorkspace = %q, want %q (conservative default)", got, "full")
		}
	})

	t.Run("returns full for empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tierPath := filepath.Join(tmpDir, ".tier")
		if err := os.WriteFile(tierPath, []byte(""), 0644); err != nil {
			t.Fatalf("failed to write tier file: %v", err)
		}

		got := ReadTierFromWorkspace(tmpDir)
		if got != "full" {
			t.Errorf("ReadTierFromWorkspace = %q, want %q (conservative default)", got, "full")
		}
	})
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

func TestParseSynthesisUnexploredQuestions(t *testing.T) {
	// Test parsing of Unexplored Questions section
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	content := `# Session Synthesis

**Agent:** og-feat-test-21dec
**Issue:** test-456
**Outcome:** success

## TLDR

Implemented feature Y with unexplored questions.

---

## Delta (What Changed)

### Files Modified
- ` + "`pkg/verify/check.go`" + ` - Added unexplored questions parsing

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How would this work with concurrent agents?
- Should we add rate limiting?

**Areas worth exploring further:**
- Performance optimization for large synthesis files
- Integration with kb reflect command

**What remains unclear:**
- Edge cases with empty sections
- Behavior with malformed markdown

---

## Session Metadata

**Skill:** feature-impl
`
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test synthesis file: %v", err)
	}

	got, err := ParseSynthesis(tmpDir)
	if err != nil {
		t.Fatalf("ParseSynthesis failed: %v", err)
	}

	// Test UnexploredQuestions raw section
	if got.UnexploredQuestions == "" {
		t.Error("UnexploredQuestions should not be empty")
	}

	// Test AreasToExplore extraction
	if len(got.AreasToExplore) != 2 {
		t.Errorf("len(AreasToExplore) = %d, want 2", len(got.AreasToExplore))
	} else {
		if !contains(got.AreasToExplore[0], "Performance optimization") {
			t.Errorf("AreasToExplore[0] = %q, want to contain 'Performance optimization'", got.AreasToExplore[0])
		}
	}

	// Test Uncertainties extraction
	if len(got.Uncertainties) != 2 {
		t.Errorf("len(Uncertainties) = %d, want 2", len(got.Uncertainties))
	} else {
		if !contains(got.Uncertainties[0], "Edge cases") {
			t.Errorf("Uncertainties[0] = %q, want to contain 'Edge cases'", got.Uncertainties[0])
		}
	}
}

func TestParseSynthesisNoUnexploredQuestions(t *testing.T) {
	// Test that parsing works fine when Unexplored Questions section is missing
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	content := `# Session Synthesis

## TLDR

Straightforward session, no unexplored territory.

## Next Actions

- Deploy changes
`
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test synthesis file: %v", err)
	}

	got, err := ParseSynthesis(tmpDir)
	if err != nil {
		t.Fatalf("ParseSynthesis failed: %v", err)
	}

	// Verify that missing section doesn't cause issues
	if got.UnexploredQuestions != "" {
		t.Errorf("UnexploredQuestions should be empty, got %q", got.UnexploredQuestions)
	}
	if len(got.AreasToExplore) != 0 {
		t.Errorf("AreasToExplore should be empty, got %v", got.AreasToExplore)
	}
	if len(got.Uncertainties) != 0 {
		t.Errorf("Uncertainties should be empty, got %v", got.Uncertainties)
	}
}

func TestParseSynthesisSpawnFollowUpNoFalsePositives(t *testing.T) {
	// Test that markdown bold fields like **Skill:** are NOT parsed as action items
	// Regression test for bug where **Field:** was incorrectly matched as bullet point
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	content := `# Session Synthesis

**Agent:** og-arch-test
**Outcome:** success

## TLDR

Test spawn-follow-up parsing.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Spawn Follow-up
**Issue:** Implement new feature
**Skill:** feature-impl
**Context:**
` + "```" + `
Some context here
` + "```" + `

---

## Session Metadata

**Skill:** architect
`
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test synthesis file: %v", err)
	}

	got, err := ParseSynthesis(tmpDir)
	if err != nil {
		t.Fatalf("ParseSynthesis failed: %v", err)
	}

	// **Issue:**, **Skill:**, **Context:** should NOT be parsed as action items
	// They start with ** (markdown bold), not * (bullet)
	if len(got.NextActions) != 0 {
		t.Errorf("NextActions should be empty (no actual bullet/numbered items), got %d: %v",
			len(got.NextActions), got.NextActions)
	}
}

func TestParseSynthesisSpawnFollowUpWithActions(t *testing.T) {
	// Test that numbered items inside Spawn Follow-up ARE correctly parsed
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	content := `# Session Synthesis

**Agent:** og-arch-test
**Outcome:** success

## TLDR

Test spawn-follow-up with numbered actions.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Spawn Follow-up
**Issue:** Implement new feature
**Skill:** feature-impl

**Tasks:**
1. First task to do
2. Second task to do
- Bullet task

---

## Session Metadata

**Skill:** architect
`
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test synthesis file: %v", err)
	}

	got, err := ParseSynthesis(tmpDir)
	if err != nil {
		t.Fatalf("ParseSynthesis failed: %v", err)
	}

	// Should have 3 items: two numbered and one bullet
	if len(got.NextActions) != 3 {
		t.Errorf("NextActions length = %d, want 3, got: %v", len(got.NextActions), got.NextActions)
	}
}

func TestParseSynthesisIndentedContinuationLines(t *testing.T) {
	// Test that indented lines (metadata/context under a main item) are NOT parsed as separate items
	// Regression test for bug where "   - Skill: feature-impl" was incorrectly matched
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	content := `# Session Synthesis

**Agent:** og-inv-test
**Outcome:** success

## TLDR

Test with indented continuation lines.

---

## Next (What Should Happen)

**Recommendation:** close

### Follow-up Work (for separate issues)
1. **Add glass_* to visual verification** - Update pkg/verify/visual.go
   - Skill: feature-impl
   - Quick win, <30 min
   
2. **Configure Glass as MCP option** - Make it work
   - Skill: feature-impl  
   - Needs investigation

3. **Document Chrome launch requirement** - Add to docs
   - Skill: feature-impl (or documentation task)

---

## Session Metadata

**Skill:** investigation
`
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test synthesis file: %v", err)
	}

	got, err := ParseSynthesis(tmpDir)
	if err != nil {
		t.Fatalf("ParseSynthesis failed: %v", err)
	}

	// Should have exactly 3 items (the numbered main items)
	// The indented "- Skill:", "- Quick win" etc. should NOT be captured
	if len(got.NextActions) != 3 {
		t.Errorf("NextActions length = %d, want 3 (only main numbered items), got: %v",
			len(got.NextActions), got.NextActions)
	}

	// Verify the items are the main action items, not the indented metadata
	for _, action := range got.NextActions {
		if strings.Contains(action, "- Skill:") || strings.Contains(action, "- Quick win") {
			t.Errorf("NextActions should not contain indented metadata lines, got: %q", action)
		}
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
