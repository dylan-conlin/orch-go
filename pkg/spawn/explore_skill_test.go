package spawn

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/skills"
)

func TestExplorationOrchestratorSkillLoads(t *testing.T) {
	loader := skills.DefaultLoader()

	content, err := loader.LoadSkillContent("exploration-orchestrator")
	if err != nil {
		t.Fatalf("Failed to load exploration-orchestrator skill: %v", err)
	}

	if content == "" {
		t.Fatal("exploration-orchestrator skill content is empty")
	}

	metadata, err := skills.ParseSkillMetadata(content)
	if err != nil {
		t.Fatalf("Failed to parse exploration-orchestrator skill metadata: %v", err)
	}

	if metadata.Name != "exploration-orchestrator" {
		t.Errorf("skill name = %q, want %q", metadata.Name, "exploration-orchestrator")
	}

	if metadata.SkillType != "orchestrator" {
		t.Errorf("skill-type = %q, want %q", metadata.SkillType, "orchestrator")
	}

	// Should be detected as orchestrator
	isOrchestrator := metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"
	if !isOrchestrator {
		t.Error("exploration-orchestrator should be detected as orchestrator type")
	}
}
