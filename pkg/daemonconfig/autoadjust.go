package daemonconfig

import "github.com/dylan-conlin/orch-go/pkg/events"

const (
	// MinSamplesForDowngrade is the minimum number of completions+abandonments
	// before auto-downgrade is considered. Prevents overreacting to small samples.
	MinSamplesForDowngrade = 10

	// DowngradeSuccessRateThreshold is the minimum success rate needed to
	// suggest a compliance level downgrade. Set high to be conservative.
	DowngradeSuccessRateThreshold = 0.80
)

// DowngradeSuggestion describes a suggested compliance level relaxation
// for a specific skill based on sustained success rates.
type DowngradeSuggestion struct {
	Skill          string
	CurrentLevel   ComplianceLevel
	SuggestedLevel ComplianceLevel
	SuccessRate    float64
	SampleSize     int
}

// SuggestDowngrades analyzes learning data and returns suggested compliance
// level downgrades for skills with sustained high success rates.
//
// Safety asymmetry: only suggests downgrades (less strict), never upgrades.
// Steps one level at a time (strict -> standard -> relaxed -> autonomous).
func SuggestDowngrades(cfg *ComplianceConfig, learning *events.LearningStore) []DowngradeSuggestion {
	if learning == nil || len(learning.Skills) == 0 {
		return nil
	}

	var suggestions []DowngradeSuggestion

	for skill, sl := range learning.Skills {
		sampleSize := sl.TotalCompletions + sl.AbandonedCount
		if sampleSize < MinSamplesForDowngrade {
			continue
		}

		if sl.SuccessRate < DowngradeSuccessRateThreshold {
			continue
		}

		// Determine current level for this skill
		currentLevel := cfg.Resolve(skill, "")

		// Cannot downgrade past autonomous
		if currentLevel >= ComplianceAutonomous {
			continue
		}

		// Step one level down
		suggestedLevel := currentLevel + 1

		suggestions = append(suggestions, DowngradeSuggestion{
			Skill:          skill,
			CurrentLevel:   currentLevel,
			SuggestedLevel: suggestedLevel,
			SuccessRate:    sl.SuccessRate,
			SampleSize:     sampleSize,
		})
	}

	return suggestions
}

// ApplyDowngrades applies the given downgrade suggestions to the compliance config.
// Returns the number of downgrades applied.
func ApplyDowngrades(cfg *ComplianceConfig, suggestions []DowngradeSuggestion) int {
	applied := 0
	for _, s := range suggestions {
		if cfg.Skills == nil {
			cfg.Skills = make(map[string]ComplianceLevel)
		}
		cfg.Skills[s.Skill] = s.SuggestedLevel
		applied++
	}
	return applied
}
