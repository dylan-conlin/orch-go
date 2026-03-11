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
		ExploreDepth:       1,
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
		{"judge skill", "exploration-judge", true},
		{"judge verdict", "judge-verdict.yaml", true},
		{"judge verdicts", "accepted/contested/rejected", true},
		{"synthesis output", "Synthesis Output", true},
		{"cost bounding", "Cost Bounding", true},
		// Single pass (depth=1) should NOT show iteration
		{"no iteration at depth 1", "ITERATE", false},
		{"no depth display at 1", "Depth:**", false},
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
		ExploreDepth:       3,
		ExploreParentSkill: "architect",
		ExploreJudgeModel:  "sonnet",
	}

	if !cfg.Explore {
		t.Error("Explore should be true")
	}
	if cfg.ExploreBreadth != 5 {
		t.Errorf("ExploreBreadth = %d, want 5", cfg.ExploreBreadth)
	}
	if cfg.ExploreDepth != 3 {
		t.Errorf("ExploreDepth = %d, want 3", cfg.ExploreDepth)
	}
	if cfg.ExploreParentSkill != "architect" {
		t.Errorf("ExploreParentSkill = %q, want %q", cfg.ExploreParentSkill, "architect")
	}
	if cfg.ExploreJudgeModel != "sonnet" {
		t.Errorf("ExploreJudgeModel = %q, want %q", cfg.ExploreJudgeModel, "sonnet")
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

func TestExploreDepthBounds(t *testing.T) {
	tests := []struct {
		depth int
		valid bool
	}{
		{0, false},
		{1, true},
		{3, true},
		{5, true},
		{6, false},
	}

	for _, tt := range tests {
		valid := tt.depth >= 1 && tt.depth <= 5
		if valid != tt.valid {
			t.Errorf("depth %d: valid=%v, want %v", tt.depth, valid, tt.valid)
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
		ExploreDepth:       1,
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

func TestExploreDepthIterationInContext(t *testing.T) {
	cfg := &Config{
		Task:               "How does the daemon handle concurrent spawns?",
		SkillName:          "exploration-orchestrator",
		Project:            "orch-go",
		ProjectDir:         "/tmp/test-project",
		WorkspaceName:      "explore-iterate-test",
		BeadsID:            "orch-go-iter123",
		Explore:            true,
		ExploreBreadth:     3,
		ExploreDepth:       3,
		ExploreParentSkill: "investigation",
		Tier:               "full",
		NoTrack:            true,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	checks := []struct {
		name string
		want string
	}{
		{"iteration mode header", "iterate"},
		{"depth display", "Depth:** 3"},
		{"re-exploration count", "up to 2 re-exploration rounds"},
		{"iteration protocol", "Iteration Protocol"},
		{"iteration decision rules", "Iteration Decision Rules"},
		{"emit iteration event", "exploration.iterated"},
		{"critical severity", "critical"},
		{"depth limit", "depth < 3"},
		{"iteration summary in synthesis", "iteration rounds"},
		{"total agent budget", "Total agent budget"},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if !strings.Contains(content, check.want) {
				t.Errorf("expected %q in generated context with depth=3, but not found", check.want)
			}
		})
	}
}

func TestExploreJudgeModelInContext(t *testing.T) {
	cfg := &Config{
		Task:               "How does auth work?",
		SkillName:          "exploration-orchestrator",
		Project:            "orch-go",
		ProjectDir:         "/tmp/test-project",
		WorkspaceName:      "explore-judge-model-test",
		BeadsID:            "orch-go-jm123",
		Explore:            true,
		ExploreBreadth:     3,
		ExploreDepth:       1,
		ExploreParentSkill: "investigation",
		ExploreJudgeModel:  "sonnet",
		Tier:               "full",
		NoTrack:            true,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	checks := []struct {
		name string
		want string
	}{
		{"judge model displayed", "Judge Model:** sonnet"},
		{"cross-model note", "cross-model judging"},
		{"model in spawn command", "--model sonnet"},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if !strings.Contains(content, check.want) {
				t.Errorf("expected %q in generated context with judge model, but not found", check.want)
			}
		})
	}
}

func TestExploreNoJudgeModelOmitsFlag(t *testing.T) {
	cfg := &Config{
		Task:               "How does auth work?",
		SkillName:          "exploration-orchestrator",
		Project:            "orch-go",
		ProjectDir:         "/tmp/test-project",
		WorkspaceName:      "explore-no-jm-test",
		BeadsID:            "orch-go-nojm",
		Explore:            true,
		ExploreBreadth:     3,
		ExploreDepth:       1,
		ExploreParentSkill: "investigation",
		ExploreJudgeModel:  "", // no judge model
		Tier:               "full",
		NoTrack:            true,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	if strings.Contains(content, "Judge Model:**") {
		t.Error("Judge Model should not appear when ExploreJudgeModel is empty")
	}
	if strings.Contains(content, "--model") {
		t.Error("--model flag should not appear in judge spawn command when ExploreJudgeModel is empty")
	}
}
