package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestBuildBriefFromSynthesis(t *testing.T) {
	s := &verify.Synthesis{
		TLDR:                "Added headless completion mode to orch complete",
		Knowledge:           "The headless mode skips interactive gates and generates briefs automatically.",
		Delta:               "Modified complete_cmd.go, created complete_brief.go",
		UnexploredQuestions: "Does the brief quality degrade without human review?",
	}

	brief := buildBriefFromSynthesis("orch-go-abc12", s)

	if !strings.Contains(brief, "# Brief: orch-go-abc12") {
		t.Error("Brief missing title with beads ID")
	}
	if !strings.Contains(brief, "## Frame") {
		t.Error("Brief missing Frame section")
	}
	if !strings.Contains(brief, s.TLDR) {
		t.Error("Brief Frame should contain TLDR")
	}
	if !strings.Contains(brief, "## Resolution") {
		t.Error("Brief missing Resolution section")
	}
	if !strings.Contains(brief, s.Knowledge) {
		t.Error("Brief Resolution should contain Knowledge")
	}
	if !strings.Contains(brief, "## Tension") {
		t.Error("Brief missing Tension section")
	}
	if !strings.Contains(brief, s.UnexploredQuestions) {
		t.Error("Brief Tension should contain UnexploredQuestions")
	}
}

func TestBuildBriefFallbacks(t *testing.T) {
	t.Run("Delta fallback when no Knowledge", func(t *testing.T) {
		s := &verify.Synthesis{
			TLDR:  "Test",
			Delta: "Changed files X and Y",
		}
		brief := buildBriefFromSynthesis("test-123", s)
		if !strings.Contains(brief, s.Delta) {
			t.Error("Resolution should fall back to Delta")
		}
	})

	t.Run("Next fallback when no UnexploredQuestions", func(t *testing.T) {
		s := &verify.Synthesis{
			TLDR: "Test",
			Next: "Follow up with integration testing",
		}
		brief := buildBriefFromSynthesis("test-123", s)
		if !strings.Contains(brief, s.Next) {
			t.Error("Tension should fall back to Next")
		}
	})

	t.Run("Empty synthesis produces placeholder text", func(t *testing.T) {
		s := &verify.Synthesis{}
		brief := buildBriefFromSynthesis("test-123", s)
		if !strings.Contains(brief, "(No TLDR in SYNTHESIS.md)") {
			t.Error("Empty TLDR should show placeholder")
		}
		if !strings.Contains(brief, "(No Knowledge, Delta, or Evidence in SYNTHESIS.md)") {
			t.Error("Empty Resolution should show placeholder")
		}
		if !strings.Contains(brief, "(No open questions or next actions in SYNTHESIS.md)") {
			t.Error("Empty Tension should show placeholder")
		}
	})
}

func TestGenerateHeadlessBrief(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace with SYNTHESIS.md
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-test-agent")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}

	synthesisContent := `# SYNTHESIS

**Agent:** og-test-agent
**Issue:** orch-go-test1

## TLDR

Implemented the test feature successfully.

## Knowledge (What Was Learned)

The test framework supports parallel execution.

## Next (What Should Happen)

**Recommendation:** close

Follow up with load testing.
`
	if err := os.WriteFile(filepath.Join(workspacePath, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create briefs directory target
	projectDir := tmpDir
	briefsDir := filepath.Join(projectDir, ".kb", "briefs")

	target := CompletionTarget{
		BeadsID:        "orch-go-test1",
		WorkspacePath:  workspacePath,
		WorkProjectDir: projectDir,
	}

	err := generateHeadlessBrief(target)
	if err != nil {
		t.Fatalf("generateHeadlessBrief failed: %v", err)
	}

	// Verify brief was created
	briefPath := filepath.Join(briefsDir, "orch-go-test1.md")
	data, err := os.ReadFile(briefPath)
	if err != nil {
		t.Fatalf("Brief file not created: %v", err)
	}

	brief := string(data)
	if !strings.Contains(brief, "# Brief: orch-go-test1") {
		t.Error("Brief missing title")
	}
	if !strings.Contains(brief, "Implemented the test feature successfully.") {
		t.Error("Brief missing TLDR content")
	}
	if !strings.Contains(brief, "The test framework supports parallel execution.") {
		t.Error("Brief missing Knowledge content")
	}
}

func TestGenerateHeadlessBriefNoSynthesis(t *testing.T) {
	tmpDir := t.TempDir()

	// Workspace without SYNTHESIS.md
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "og-test-agent")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatal(err)
	}

	target := CompletionTarget{
		BeadsID:        "orch-go-test1",
		WorkspacePath:  workspacePath,
		WorkProjectDir: tmpDir,
	}

	err := generateHeadlessBrief(target)
	if err == nil {
		t.Error("Expected error when SYNTHESIS.md is missing")
	}
	if !strings.Contains(err.Error(), "SYNTHESIS.md") {
		t.Errorf("Error should mention SYNTHESIS.md, got: %v", err)
	}
}
