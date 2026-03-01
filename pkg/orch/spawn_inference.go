package orch

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// InferSkillFromIssueType maps issue types to appropriate skills.
func InferSkillFromIssueType(issueType string) (string, error) {
	switch issueType {
	case "bug":
		return "architect", nil
	case "feature":
		return "feature-impl", nil
	case "task":
		return "feature-impl", nil
	case "investigation":
		return "investigation", nil
	case "epic":
		return "", fmt.Errorf("cannot spawn work on epic issues - epics are decomposed into sub-issues")
	case "":
		return "", fmt.Errorf("issue type is empty")
	default:
		return "", fmt.Errorf("unknown issue type: %s", issueType)
	}
}

// DetermineSpawnTier determines the spawn tier based on flags, config, task scope signals,
// and skill defaults.
func DetermineSpawnTier(skillName, task string, lightFlag, fullFlag bool) string {
	if lightFlag {
		return spawn.TierLight
	}
	if fullFlag {
		return spawn.TierFull
	}
	cfg, err := userconfig.Load()
	if err == nil && cfg.GetDefaultTier() != "" {
		return cfg.GetDefaultTier()
	}
	if inferredTier := inferTierFromTask(task); inferredTier != "" {
		return inferredTier
	}
	return spawn.DefaultTierForSkill(skillName)
}

func inferTierFromTask(task string) string {
	if task == "" {
		return ""
	}
	if scope := spawn.ParseScopeFromTask(task); scope != "" {
		switch scope {
		case "medium", "large", "full", "4-6h", "4-6h+", "2-4h":
			return spawn.TierFull
		}
	}
	lower := strings.ToLower(task)
	score := 0
	if containsAny(lower, []string{
		"create package", "new package", "create module", "new module",
		"new pkg/", "create pkg/", "new package/", "create package/",
	}) {
		score += 2
	}
	if containsAny(lower, []string{
		"comprehensive tests", "test suite", "integration tests",
		"unit tests", "tests for", "add tests",
	}) {
		score++
	}
	if score >= 2 {
		return spawn.TierFull
	}
	return ""
}

func containsAny(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

// validateModeModelCombo checks for known invalid mode+model combinations.
func validateModeModelCombo(backend string, resolvedModel model.ModelSpec) error {
	if backend == "opencode" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "opus") {
		return fmt.Errorf("Warning: opencode backend with opus model may fail (auth blocked). Recommendation: Use --model sonnet (default) or let auto-selection use claude backend")
	}
	return nil
}

// inferSkillFromBeadsIssue infers skill from a beads issue using labels, title, then type.
func inferSkillFromBeadsIssue(issue *beads.Issue) string {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "skill:") {
			return strings.TrimPrefix(label, "skill:")
		}
	}
	if strings.HasPrefix(issue.Title, "Synthesize ") && strings.Contains(issue.Title, " investigations") {
		return "kb-reflect"
	}
	skill, err := InferSkillFromIssueType(issue.IssueType)
	if err != nil {
		return "feature-impl"
	}
	return skill
}

// inferMCPFromBeadsIssue extracts MCP server requirements from issue labels.
func inferMCPFromBeadsIssue(issue *beads.Issue) string {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "needs:") {
			need := strings.TrimPrefix(label, "needs:")
			switch need {
			case "playwright":
				return "playwright"
			}
		}
	}
	return ""
}
