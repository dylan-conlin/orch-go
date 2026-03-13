package daemonconfig

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestSuggestDowngrades_NoLearning(t *testing.T) {
	cfg := ComplianceConfig{Default: ComplianceStrict}
	suggestions := SuggestDowngrades(&cfg, nil)
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions with nil learning, got %d", len(suggestions))
	}
}

func TestSuggestDowngrades_EmptyLearning(t *testing.T) {
	cfg := ComplianceConfig{Default: ComplianceStrict}
	learning := &events.LearningStore{Skills: make(map[string]*events.SkillLearning)}
	suggestions := SuggestDowngrades(&cfg, learning)
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions with empty learning, got %d", len(suggestions))
	}
}

func TestSuggestDowngrades_HighSuccessRate(t *testing.T) {
	cfg := ComplianceConfig{Default: ComplianceStrict}
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       20,
				TotalCompletions: 18,
				SuccessCount:     17,
				AbandonedCount:   2,
				SuccessRate:      0.85, // 17/(18+2)
			},
		},
	}
	suggestions := SuggestDowngrades(&cfg, learning)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	s := suggestions[0]
	if s.Skill != "feature-impl" {
		t.Errorf("expected skill=feature-impl, got %q", s.Skill)
	}
	if s.CurrentLevel != ComplianceStrict {
		t.Errorf("expected current=strict, got %v", s.CurrentLevel)
	}
	if s.SuggestedLevel != ComplianceStandard {
		t.Errorf("expected suggested=standard, got %v", s.SuggestedLevel)
	}
}

func TestSuggestDowngrades_InsufficientSamples(t *testing.T) {
	cfg := ComplianceConfig{Default: ComplianceStrict}
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       5,
				TotalCompletions: 4,
				SuccessCount:     4,
				AbandonedCount:   0,
				SuccessRate:      1.0, // perfect but too few samples
			},
		},
	}
	suggestions := SuggestDowngrades(&cfg, learning)
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions with insufficient samples, got %d", len(suggestions))
	}
}

func TestSuggestDowngrades_LowSuccessRate(t *testing.T) {
	cfg := ComplianceConfig{Default: ComplianceStrict}
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       20,
				TotalCompletions: 15,
				SuccessCount:     8,
				AbandonedCount:   5,
				SuccessRate:      0.4, // 8/(15+5) — too low
			},
		},
	}
	suggestions := SuggestDowngrades(&cfg, learning)
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions with low success rate, got %d", len(suggestions))
	}
}

func TestSuggestDowngrades_AlreadyAtLowest(t *testing.T) {
	cfg := ComplianceConfig{
		Default: ComplianceStrict,
		Skills:  map[string]ComplianceLevel{"feature-impl": ComplianceAutonomous},
	}
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       30,
				TotalCompletions: 28,
				SuccessCount:     27,
				AbandonedCount:   1,
				SuccessRate:      0.93,
			},
		},
	}
	suggestions := SuggestDowngrades(&cfg, learning)
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions when already autonomous, got %d", len(suggestions))
	}
}

func TestSuggestDowngrades_NeverUpgrades(t *testing.T) {
	// Even if success rate drops, should never suggest upgrading (stricter)
	cfg := ComplianceConfig{
		Default: ComplianceStrict,
		Skills:  map[string]ComplianceLevel{"feature-impl": ComplianceRelaxed},
	}
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       20,
				TotalCompletions: 15,
				SuccessCount:     5,
				AbandonedCount:   5,
				SuccessRate:      0.25, // terrible rate
			},
		},
	}
	suggestions := SuggestDowngrades(&cfg, learning)
	// Should NOT suggest upgrading to strict — only downgrades
	for _, s := range suggestions {
		if s.SuggestedLevel < s.CurrentLevel {
			t.Errorf("auto-adjuster suggested UPGRADE from %v to %v — violates safety asymmetry",
				s.CurrentLevel, s.SuggestedLevel)
		}
	}
}

func TestSuggestDowngrades_StepByStep(t *testing.T) {
	// Should only suggest one level down, not jump from strict to autonomous
	cfg := ComplianceConfig{Default: ComplianceStrict}
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       50,
				TotalCompletions: 48,
				SuccessCount:     47,
				AbandonedCount:   1,
				SuccessRate:      0.96,
			},
		},
	}
	suggestions := SuggestDowngrades(&cfg, learning)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	// Should go strict -> standard, not strict -> autonomous
	if suggestions[0].SuggestedLevel != ComplianceStandard {
		t.Errorf("expected standard (one step down), got %v", suggestions[0].SuggestedLevel)
	}
}

func TestApplyDowngrades(t *testing.T) {
	cfg := ComplianceConfig{Default: ComplianceStrict}
	suggestions := []DowngradeSuggestion{
		{Skill: "feature-impl", CurrentLevel: ComplianceStrict, SuggestedLevel: ComplianceStandard},
	}
	applied := ApplyDowngrades(&cfg, suggestions)
	if applied != 1 {
		t.Errorf("expected 1 applied, got %d", applied)
	}
	if cfg.Skills["feature-impl"] != ComplianceStandard {
		t.Errorf("expected feature-impl=standard after apply, got %v", cfg.Skills["feature-impl"])
	}
}

func TestApplyDowngrades_InitializesSkillsMap(t *testing.T) {
	cfg := ComplianceConfig{Default: ComplianceStrict}
	// Skills map is nil
	suggestions := []DowngradeSuggestion{
		{Skill: "investigation", CurrentLevel: ComplianceStrict, SuggestedLevel: ComplianceStandard},
	}
	applied := ApplyDowngrades(&cfg, suggestions)
	if applied != 1 {
		t.Errorf("expected 1 applied, got %d", applied)
	}
	if cfg.Skills == nil {
		t.Fatal("expected Skills map to be initialized")
	}
	if cfg.Skills["investigation"] != ComplianceStandard {
		t.Errorf("expected investigation=standard, got %v", cfg.Skills["investigation"])
	}
}
