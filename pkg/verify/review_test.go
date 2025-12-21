package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetAgentReview(t *testing.T) {
	// Create a temporary workspace directory
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-21dec")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace directory: %v", err)
	}

	// Create a SYNTHESIS.md file
	synthesisContent := `# Session Synthesis

**Agent:** og-feat-test-21dec
**Issue:** test-123
**Duration:** 15m
**Outcome:** success

---

## TLDR

Implemented the review feature for orch complete command.

---

## Next (What Should Happen)

**Recommendation:** close

`
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	if err := os.WriteFile(synthesisPath, []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write synthesis file: %v", err)
	}

	// Test GetAgentReview (note: can't test beads comments without actual beads)
	review, err := GetAgentReview("test-123", workspacePath, tmpDir)
	if err != nil {
		t.Fatalf("GetAgentReview failed: %v", err)
	}

	// Verify workspace name extraction
	if review.WorkspaceName != "og-feat-test-21dec" {
		t.Errorf("WorkspaceName = %q, want %q", review.WorkspaceName, "og-feat-test-21dec")
	}

	// Verify synthesis parsing
	if !review.SynthesisExists {
		t.Error("SynthesisExists should be true")
	}

	if review.Outcome != "success" {
		t.Errorf("Outcome = %q, want %q", review.Outcome, "success")
	}

	if review.Recommendation != "close" {
		t.Errorf("Recommendation = %q, want %q", review.Recommendation, "close")
	}

	// Verify TLDR
	wantTLDR := "Implemented the review feature for orch complete command."
	if review.TLDR != wantTLDR {
		t.Errorf("TLDR = %q, want %q", review.TLDR, wantTLDR)
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name          string
		workspaceName string
		wantKeywords  []string
	}{
		{
			name:          "typical workspace name",
			workspaceName: "og-feat-implement-review-21dec",
			wantKeywords:  []string{"implement", "review"},
		},
		{
			name:          "workspace with orch",
			workspaceName: "og-debug-fix-orch-spawn-20dec",
			wantKeywords:  []string{"orch", "spawn"},
		},
		{
			name:          "workspace with short keywords",
			workspaceName: "og-feat-add-ui-21dec",
			wantKeywords:  []string{"add"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKeywords(tt.workspaceName)

			for _, want := range tt.wantKeywords {
				found := false
				for _, keyword := range got {
					if keyword == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("extractKeywords(%q) missing keyword %q, got %v", tt.workspaceName, want, got)
				}
			}
		})
	}
}

func TestFormatAgentReview(t *testing.T) {
	review := &AgentReview{
		WorkspaceName:   "og-feat-test-21dec",
		BeadsID:         "test-123",
		Skill:           "feature-impl",
		Status:          "Phase: Complete",
		TLDR:            "Implemented a new feature for testing the review command.",
		Outcome:         "success",
		Recommendation:  "close",
		FilesCreated:    2,
		FilesModified:   1,
		Commits:         []CommitInfo{{Hash: "abc1234", Message: "feat: add review"}},
		SynthesisExists: true,
		Comments: []Comment{
			{Text: "Phase: Planning - Starting work"},
			{Text: "Phase: Complete - All done"},
		},
	}

	output := FormatAgentReview(review)

	// Check for expected sections
	if !strings.Contains(output, "AGENT REVIEW: og-feat-test-21dec") {
		t.Error("Output should contain agent review header")
	}

	if !strings.Contains(output, "Beads:  test-123") {
		t.Error("Output should contain beads ID")
	}

	if !strings.Contains(output, "Skill:  feature-impl") {
		t.Error("Output should contain skill")
	}

	if !strings.Contains(output, "TLDR:") {
		t.Error("Output should contain TLDR section")
	}

	if !strings.Contains(output, "DELTA:") {
		t.Error("Output should contain DELTA section")
	}

	if !strings.Contains(output, "+2 created, 1 modified") {
		t.Error("Output should contain file stats")
	}

	if !strings.Contains(output, "abc1234") {
		t.Error("Output should contain commit hash")
	}

	if !strings.Contains(output, "BEADS COMMENTS:") {
		t.Error("Output should contain beads comments section")
	}

	if !strings.Contains(output, "ARTIFACTS:") {
		t.Error("Output should contain artifacts section")
	}

	if !strings.Contains(output, "SYNTHESIS.md") {
		t.Error("Output should mention SYNTHESIS.md")
	}
}

func TestFormatAgentReviewMissingSynthesis(t *testing.T) {
	review := &AgentReview{
		WorkspaceName:   "og-feat-test-21dec",
		BeadsID:         "test-123",
		SynthesisExists: false,
	}

	output := FormatAgentReview(review)

	if !strings.Contains(output, "SYNTHESIS.md (missing)") {
		t.Error("Output should indicate missing SYNTHESIS.md")
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  int // number of lines expected
	}{
		{
			name:  "short text",
			text:  "Short text",
			width: 70,
			want:  1,
		},
		{
			name:  "long text",
			text:  "This is a much longer text that should be wrapped across multiple lines when the width is exceeded.",
			width: 40,
			want:  3,
		},
		{
			name:  "text with newlines",
			text:  "First line\nSecond line\nThird line",
			width: 70,
			want:  1, // newlines are replaced with spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.text, tt.width)
			lines := strings.Split(got, "\n")
			if len(lines) != tt.want {
				t.Errorf("wrapText(%q, %d) = %d lines, want %d lines", tt.text, tt.width, len(lines), tt.want)
			}
		})
	}
}

func TestFindInvestigationFile(t *testing.T) {
	// Create a temporary project structure
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-implement-review-21dec")
	investigationsDir := filepath.Join(tmpDir, ".kb", "investigations")

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace directory: %v", err)
	}
	if err := os.MkdirAll(investigationsDir, 0755); err != nil {
		t.Fatalf("failed to create investigations directory: %v", err)
	}

	// Create an investigation file
	invFile := filepath.Join(investigationsDir, "2025-12-21-inv-implement-review.md")
	if err := os.WriteFile(invFile, []byte("# Investigation"), 0644); err != nil {
		t.Fatalf("failed to write investigation file: %v", err)
	}

	path, found := findInvestigationFile(workspacePath)
	if !found {
		t.Error("findInvestigationFile should find the investigation file")
	}
	if !strings.HasSuffix(path, "2025-12-21-inv-implement-review.md") {
		t.Errorf("findInvestigationFile returned wrong path: %s", path)
	}
}
