package spawn

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/skills"
)

func TestExplorationJudgeSkillLoads(t *testing.T) {
	loader := skills.DefaultLoader()

	content, err := loader.LoadSkillContent("exploration-judge")
	if err != nil {
		t.Fatalf("Failed to load exploration-judge skill: %v", err)
	}

	if content == "" {
		t.Fatal("exploration-judge skill content is empty")
	}

	metadata, err := skills.ParseSkillMetadata(content)
	if err != nil {
		t.Fatalf("Failed to parse exploration-judge skill metadata: %v", err)
	}

	if metadata.Name != "exploration-judge" {
		t.Errorf("skill name = %q, want %q", metadata.Name, "exploration-judge")
	}

	if metadata.SkillType != "evaluator" {
		t.Errorf("skill-type = %q, want %q", metadata.SkillType, "evaluator")
	}
}

func TestExplorationJudgeSkillContainsVerdictSchema(t *testing.T) {
	loader := skills.DefaultLoader()

	content, err := loader.LoadSkillContent("exploration-judge")
	if err != nil {
		t.Fatalf("Failed to load exploration-judge skill: %v", err)
	}

	// Verify the skill contains all 5 evaluation dimensions
	dimensions := []string{
		"Grounding",
		"Consistency",
		"Coverage",
		"Relevance",
		"Actionability",
	}
	for _, dim := range dimensions {
		if !strings.Contains(content, dim) {
			t.Errorf("skill content missing evaluation dimension: %s", dim)
		}
	}

	// Verify verdict output format is documented
	verdictMarkers := []string{
		"sub_findings:",
		"contested_findings:",
		"coverage_gaps:",
		"verdict: accepted | contested | rejected",
	}
	for _, marker := range verdictMarkers {
		if !strings.Contains(content, marker) {
			t.Errorf("skill content missing verdict format marker: %s", marker)
		}
	}
}
