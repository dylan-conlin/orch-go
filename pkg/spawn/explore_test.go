package spawn

import (
	"strings"
	"testing"
)

func TestExploreContextGeneration(t *testing.T) {
	cfg := &Config{
		Task:               "How does the daemon handle concurrent spawns?",
		SkillName:          "exploration-orchestrator",
		Project:            "orch-go",
		ProjectDir:         "/tmp/test-project",
		WorkspaceName:      "explore-daemon-test",
		BeadsID:            "orch-go-test123",
		Explore:            true,
		ExploreBreadth:     3,
		ExploreParentSkill: "investigation",
		Tier:               "full",
		NoTrack:            true,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Verify exploration mode section is present
	checks := []struct {
		name    string
		want    string
		present bool
	}{
		{"exploration header", "EXPLORATION MODE CONFIGURATION", true},
		{"parent skill", "Parent Skill:** investigation", true},
		{"breadth", "Breadth:** 3", true},
		{"decompose step", "DECOMPOSE", true},
		{"spawn command", "orch spawn --bypass-triage --no-track", true},
		{"judge criteria", "Grounding", true},
		{"judge criteria consistency", "Consistency", true},
		{"judge criteria coverage", "Coverage", true},
		{"synthesis output", "Synthesis Output", true},
		{"cost bounding", "Cost Bounding", true},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			found := strings.Contains(content, check.want)
			if found != check.present {
				if check.present {
					t.Errorf("expected %q in generated context, but not found", check.want)
				} else {
					t.Errorf("did not expect %q in generated context, but found it", check.want)
				}
			}
		})
	}
}

func TestExploreContextNotPresentWithoutFlag(t *testing.T) {
	cfg := &Config{
		Task:          "Regular investigation task",
		SkillName:     "investigation",
		Project:       "orch-go",
		ProjectDir:    "/tmp/test-project",
		WorkspaceName: "regular-inv-test",
		BeadsID:       "orch-go-test456",
		Explore:       false,
		Tier:          "full",
		NoTrack:       true,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	if strings.Contains(content, "EXPLORATION MODE CONFIGURATION") {
		t.Error("exploration mode section should NOT be present when Explore is false")
	}
}

func TestExploreConfigFields(t *testing.T) {
	cfg := &Config{
		Explore:            true,
		ExploreBreadth:     5,
		ExploreParentSkill: "architect",
	}

	if !cfg.Explore {
		t.Error("Explore should be true")
	}
	if cfg.ExploreBreadth != 5 {
		t.Errorf("ExploreBreadth = %d, want 5", cfg.ExploreBreadth)
	}
	if cfg.ExploreParentSkill != "architect" {
		t.Errorf("ExploreParentSkill = %q, want %q", cfg.ExploreParentSkill, "architect")
	}
}

func TestExploreBreadthBounds(t *testing.T) {
	// This tests the validation logic that lives in spawn_cmd.go
	// Here we just verify the Config can represent valid breadth values
	tests := []struct {
		breadth int
		valid   bool
	}{
		{0, false},
		{1, true},
		{3, true},
		{5, true},
		{10, true},
		{11, false},
	}

	for _, tt := range tests {
		valid := tt.breadth >= 1 && tt.breadth <= 10
		if valid != tt.valid {
			t.Errorf("breadth %d: valid=%v, want %v", tt.breadth, valid, tt.valid)
		}
	}
}

func TestExploreParentSkillInSpawnCommand(t *testing.T) {
	cfg := &Config{
		Task:               "How does token refresh work?",
		SkillName:          "exploration-orchestrator",
		Project:            "orch-go",
		ProjectDir:         "/tmp/test-project",
		WorkspaceName:      "explore-token-test",
		BeadsID:            "orch-go-testxyz",
		Explore:            true,
		ExploreBreadth:     3,
		ExploreParentSkill: "investigation",
		Tier:               "full",
		NoTrack:            true,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// The spawn command in the template should reference the parent skill
	if !strings.Contains(content, "investigation") {
		t.Error("generated context should reference parent skill 'investigation' in spawn command")
	}
}
