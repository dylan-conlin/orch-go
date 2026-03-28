// Package main provides headless brief generation for orch complete --headless.
// Reads SYNTHESIS.md from the workspace and produces a brief in .kb/briefs/<beads-id>.md.
//
// Brief format follows the Frame/Resolution/Tension structure used in existing briefs.
// Source verification: briefs are built from SYNTHESIS.md fields (TLDR, Knowledge, Next,
// UnexploredQuestions) which are themselves derived from the agent's work artifacts.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/compose"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// generateHeadlessBrief reads SYNTHESIS.md from the workspace, constructs a brief,
// and writes it to .kb/briefs/<beads-id>.md in the work project directory.
func generateHeadlessBrief(target CompletionTarget) error {
	synthesis, err := verify.ParseSynthesis(target.WorkspacePath)
	if err != nil {
		return fmt.Errorf("cannot read SYNTHESIS.md: %w", err)
	}

	// Classify brief category from issue metadata
	var issueType, skill string
	if target.Issue != nil {
		issueType = target.Issue.IssueType
		skill = inferSkillForBrief(target.Issue)
	}
	category := compose.ClassifyBriefCategory(issueType, skill, target.Identifier)

	brief := buildBriefFromSynthesis(target.BeadsID, synthesis, category)

	briefDir := filepath.Join(target.WorkProjectDir, ".kb", "briefs")
	if err := os.MkdirAll(briefDir, 0755); err != nil {
		return fmt.Errorf("cannot create briefs directory: %w", err)
	}

	briefPath := filepath.Join(briefDir, target.BeadsID+".md")
	if err := os.WriteFile(briefPath, []byte(brief), 0644); err != nil {
		return fmt.Errorf("cannot write brief: %w", err)
	}

	fmt.Printf("Brief generated: %s\n", briefPath)
	return nil
}

// buildBriefFromSynthesis constructs the brief markdown from parsed SYNTHESIS fields.
// Maps SYNTHESIS sections to brief structure:
//   - YAML frontmatter with quality signals (beads_id, signal_count, individual signals)
//   - Frame: TLDR (what the agent was doing and why)
//   - Resolution: Knowledge + Delta (what was learned/changed)
//   - Tension: UnexploredQuestions or Next (what remains open)
func buildBriefFromSynthesis(beadsID string, s *verify.Synthesis, category string) string {
	var b strings.Builder

	// Compute quality signals and prepend as YAML frontmatter
	quality := verify.ComputeSynthesisQuality(s)
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("beads_id: %s\n", beadsID))
	if category != "" {
		b.WriteString(fmt.Sprintf("category: %s\n", category))
	}
	b.WriteString("quality_signals:\n")
	for _, sig := range quality.Signals {
		b.WriteString(fmt.Sprintf("  %s:\n", sig.Name))
		b.WriteString(fmt.Sprintf("    score: %q\n", sig.Score))
		b.WriteString(fmt.Sprintf("    detected: %v\n", sig.Detected))
		// Truncate evidence to 80 chars for frontmatter readability
		evidence := sig.Evidence
		if len(evidence) > 80 {
			evidence = evidence[:80] + "..."
		}
		b.WriteString(fmt.Sprintf("    evidence: %q\n", evidence))
	}
	b.WriteString(fmt.Sprintf("signal_count: %d\n", quality.SignalCount))
	b.WriteString(fmt.Sprintf("signal_total: %d\n", quality.Total))
	b.WriteString("---\n\n")

	b.WriteString(fmt.Sprintf("# Brief: %s\n\n", beadsID))

	// Frame: use TLDR as the framing sentence
	b.WriteString("## Frame\n\n")
	if s.TLDR != "" {
		b.WriteString(s.TLDR + "\n")
	} else {
		b.WriteString("(No TLDR in SYNTHESIS.md)\n")
	}

	// Resolution: combine Knowledge and Delta for what was done
	b.WriteString("\n## Resolution\n\n")
	if s.Knowledge != "" {
		b.WriteString(s.Knowledge + "\n")
	} else if s.Delta != "" {
		b.WriteString(s.Delta + "\n")
	} else if s.Evidence != "" {
		b.WriteString(s.Evidence + "\n")
	} else {
		b.WriteString("(No Knowledge, Delta, or Evidence in SYNTHESIS.md)\n")
	}

	// Tension: use UnexploredQuestions or Next for what's still open
	b.WriteString("\n## Tension\n\n")
	if s.UnexploredQuestions != "" {
		b.WriteString(s.UnexploredQuestions + "\n")
	} else if s.Next != "" {
		b.WriteString(s.Next + "\n")
	} else {
		b.WriteString("(No open questions or next actions in SYNTHESIS.md)\n")
	}

	return b.String()
}

// inferSkillForBrief resolves the skill for an issue using the same inference
// chain as daemon spawn: label > title > issue type.
func inferSkillForBrief(issue *verify.Issue) string {
	if skill := daemon.InferSkillFromLabels(issue.Labels); skill != "" {
		return skill
	}
	if skill := daemon.InferSkillFromTitle(issue.Title); skill != "" {
		return skill
	}
	skill, _ := daemon.InferSkill(issue.IssueType)
	return skill
}
