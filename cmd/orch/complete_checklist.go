// Package main provides checklist and changelog UI rendering for the completion pipeline.
// Extracted from complete_actions.go to keep presentation concerns separate from
// action helpers.
package main

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// NotableChangelogEntry represents a notable change from the changelog.
type NotableChangelogEntry struct {
	Commit CommitInfo
	Reason string // Why this is notable (e.g., "BREAKING", "skill-relevant", "behavioral")
}

// detectNotableChangelogEntries checks recent commits across ecosystem repos for
// notable changes that the orchestrator should be aware of.
func detectNotableChangelogEntries(projectDir string, agentSkill string) []string {
	var entries []string

	result, err := GetChangelog(3, "all")
	if err != nil {
		return nil
	}

	for _, dateCommits := range result.CommitsByDate {
		for _, commit := range dateCommits {
			var reasons []string

			if commit.SemanticInfo.IsBreaking {
				reasons = append(reasons, "BREAKING")
			}

			if commit.SemanticInfo.ChangeType == ChangeTypeBehavioral {
				if commit.Category == "skills" || commit.Category == "skill-behavioral" ||
					commit.Category == "cmd" || commit.Category == "pkg" {
					reasons = append(reasons, "behavioral")
				}
			}

			if agentSkill != "" && isSkillRelevantChange(commit, agentSkill) {
				reasons = append(reasons, fmt.Sprintf("relevant to %s", agentSkill))
			}

			if len(reasons) > 0 {
				icon := "📌"
				if commit.SemanticInfo.IsBreaking {
					icon = "🚨"
				} else if strings.Contains(strings.Join(reasons, ","), "relevant to") {
					icon = "🎯"
				}

				entry := fmt.Sprintf("%s [%s] %s (%s)",
					icon,
					commit.Repo,
					truncateString(commit.Subject, 40),
					strings.Join(reasons, ", "))
				entries = append(entries, entry)
			}
		}
	}

	if len(entries) > 5 {
		entries = entries[:5]
	}

	return entries
}

// isSkillRelevantChange checks if a commit affects files related to a specific skill.
func isSkillRelevantChange(commit CommitInfo, skillName string) bool {
	for _, file := range commit.Files {
		if strings.Contains(file, "skills/") {
			if strings.Contains(file, "/"+skillName+"/") ||
				strings.Contains(file, "/"+skillName+".") ||
				strings.HasPrefix(file, "skills/"+skillName+"/") ||
				strings.Contains(file, "/skills/"+skillName+"/") {
				return true
			}
		}

		if strings.Contains(file, "SPAWN_CONTEXT") ||
			strings.Contains(file, "pkg/spawn/") {
			return true
		}

		if strings.Contains(file, "pkg/verify/skill") {
			return true
		}
	}
	return false
}

type verificationChecklistItem struct {
	Label  string
	Status string // passed, pending, skipped
}

func buildVerificationChecklist(
	result verify.VerificationResult,
	issueType string,
	tier string,
	isOrchestrator bool,
	skipConfig verify.SkipConfig,
	gate1Complete bool,
	gate2Complete bool,
) []verificationChecklistItem {
	items := []verificationChecklistItem{}
	appendItem := func(label, status string) {
		if status == "n/a" {
			return
		}
		items = append(items, verificationChecklistItem{Label: label, Status: status})
	}

	gateStatus := func(gate string) string {
		if skipConfig.ShouldSkipGate(gate) {
			return "skipped"
		}
		for _, failed := range result.GatesFailed {
			if failed == gate {
				return "pending"
			}
		}
		return "passed"
	}

	if isOrchestrator {
		appendItem("session handoff", gateStatus(verify.GateSessionHandoff))
		appendItem("handoff content", gateStatus(verify.GateHandoffContent))
		return items
	}

	if issueType != "" && checkpoint.RequiresCheckpoint(issueType) {
		explainStatus := "pending"
		if skipConfig.ExplainBack {
			explainStatus = "skipped"
		} else if gate1Complete {
			explainStatus = "passed"
		}
		appendItem("explain-back (gate1)", explainStatus)
	} else {
		appendItem("explain-back (gate1)", "n/a")
	}

	if issueType != "" && checkpoint.RequiresGate2(issueType) {
		behaviorStatus := "pending"
		if gate2Complete {
			behaviorStatus = "passed"
		}
		appendItem("behavioral verification (gate2)", behaviorStatus)
	} else {
		appendItem("behavioral verification (gate2)", "n/a")
	}

	appendItem("phase complete", gateStatus(verify.GatePhaseComplete))

	if tier == "light" || verify.IsKnowledgeProducingSkill(result.Skill) {
		appendItem("synthesis", "n/a")
	} else {
		appendItem("synthesis", gateStatus(verify.GateSynthesis))
	}

	appendItem("test evidence", gateStatus(verify.GateTestEvidence))
	appendItem("visual verification", gateStatus(verify.GateVisualVerify))
	appendItem("git diff", gateStatus(verify.GateGitDiff))
	appendItem("build", gateStatus(verify.GateBuild))
	appendItem("constraint", gateStatus(verify.GateConstraint))
	appendItem("phase gate", gateStatus(verify.GatePhaseGate))
	appendItem("skill output", gateStatus(verify.GateSkillOutput))
	appendItem("decision patch limit", gateStatus(verify.GateDecisionPatchLimit))
	appendItem("accretion", gateStatus(verify.GateAccretion))
	appendItem("architectural choices", gateStatus(verify.GateArchitecturalChoices))

	return items
}

func printVerificationChecklist(items []verificationChecklistItem, trustTier TrustTier) {
	if len(items) == 0 {
		return
	}

	fmt.Println("\n--- Verification Checklist ---")
	fmt.Printf("  Trust: %s\n", formatTrustTier(trustTier))
	for _, item := range items {
		fmt.Printf("  [%s] %s\n", formatChecklistStatus(item.Status), item.Label)
	}
	fmt.Println("--------------------------------")
}

func formatChecklistStatus(status string) string {
	switch status {
	case "passed":
		return "PASS"
	case "pending":
		return "PEND"
	case "skipped":
		return "SKIP"
	default:
		return "N/A"
	}
}
