// Package daemon provides autonomous overnight processing capabilities.
// allocation.go implements skill-aware slot scoring for the daemon's Orient phase.
// Instead of first-eligible spawning, it scores candidate issues by skill success
// rate, model fit, and base priority to produce a ranked allocation profile.
package daemon

import (
	"fmt"
	"sort"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

const (
	// DefaultSuccessRate is used when no learning data exists for a skill.
	// 0.5 is neutral — neither boost nor penalty.
	DefaultSuccessRate = 0.5

	// MinSamplesForFullWeight is the sample size at which observed success rate
	// fully replaces the default. Below this, rates are blended toward the default
	// to avoid overreacting to small samples.
	MinSamplesForFullWeight = 10

	// SuccessRateWeight controls how much success rate modulates the base priority score.
	// At 0.4, a skill with 100% success gets a 20% boost and 0% success gets a 20% penalty
	// relative to the neutral default (0.5). Priority still dominates.
	SuccessRateWeight = 0.4

)

// IssueScore holds the computed allocation score for a candidate issue.
type IssueScore struct {
	Issue            Issue
	Score            float64
	SkillSuccessRate float64
	InferredSkill    string
	InferredModel    string
}

// ScoreIssue computes an allocation score for a single issue using learning data.
// Higher score = better candidate for spawning.
//
// Score formula: basePriority * (1 - weight + weight * blendedSuccessRate)
// - basePriority: (maxPriority - issue.Priority) normalized, so P0=4, P4=0
// - blendedSuccessRate: observed rate blended with default based on sample size
// - weight: how much success rate can modulate the base score (±20% at weight=0.4)
func ScoreIssue(issue Issue, learning *events.LearningStore) IssueScore {
	skill := inferSkillForScoring(issue)
	model := InferModelFromSkill(skill)

	successRate := lookupSuccessRate(skill, learning)

	// Base priority: P0=5, P1=4, P2=3, P3=2, P4=1 (add 1 so P4 is nonzero)
	basePriority := float64(5 - issue.Priority)
	if basePriority < 1 {
		basePriority = 1
	}

	// Modulate base priority by success rate
	// At SuccessRateWeight=0.4: multiplier ranges from 0.8 (0% success) to 1.2 (100% success)
	multiplier := 1 - SuccessRateWeight + SuccessRateWeight*successRate*2
	score := basePriority * multiplier

	return IssueScore{
		Issue:            issue,
		Score:            score,
		SkillSuccessRate: successRate,
		InferredSkill:    skill,
		InferredModel:    model,
	}
}

// ScoreIssues scores all candidate issues and returns them sorted by score (descending).
func ScoreIssues(issues []Issue, learning *events.LearningStore) []IssueScore {
	scored := make([]IssueScore, len(issues))
	for i, issue := range issues {
		scored[i] = ScoreIssue(issue, learning)
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	return scored
}

// BlendedSuccessRate blends the observed success rate with the default rate based
// on sample size. This prevents a single success/failure from dominating the score.
//
// Uses a simple weight: w = min(sampleSize, MinSamplesForFullWeight) / MinSamplesForFullWeight
// blended = w * observed + (1 - w) * default
func BlendedSuccessRate(observed float64, sampleSize int) float64 {
	if sampleSize <= 0 {
		return DefaultSuccessRate
	}

	w := float64(sampleSize) / float64(MinSamplesForFullWeight)
	if w > 1.0 {
		w = 1.0
	}

	return w*observed + (1-w)*DefaultSuccessRate
}

// lookupSuccessRate returns the blended success rate for a skill.
// Blends observed success rate with the default rate based on sample size.
func lookupSuccessRate(skill string, learning *events.LearningStore) float64 {
	if learning == nil {
		return DefaultSuccessRate
	}

	sl, ok := learning.Skills[skill]
	if !ok {
		return DefaultSuccessRate
	}

	sampleSize := sl.TotalCompletions + sl.AbandonedCount
	return BlendedSuccessRate(sl.SuccessRate, sampleSize)
}

// MinCompletionsForChannelHealthCheck is the minimum number of completions
// a skill must have before we warn about absent rework signal. Below this
// threshold, zero reworks is expected (not enough volume to judge).
const MinCompletionsForChannelHealthCheck = 10

// ChannelHealthWarning indicates that a feedback channel (rework) appears
// inactive despite sufficient volume to expect signal. Absent signal should
// not be treated as positive — it may mean the channel is broken.
type ChannelHealthWarning struct {
	// Skill is the skill with the absent signal.
	Skill string
	// Completions is the number of completions for this skill.
	Completions int
	// Message is a human-readable warning.
	Message string
}

// CheckChannelHealth examines learning data for skills where rework=0
// alongside high completion volume. This detects a "silent channel" — the
// absence of negative signal is not evidence of quality, it may indicate
// the rework feedback loop is not functioning.
func CheckChannelHealth(learning *events.LearningStore) []ChannelHealthWarning {
	if learning == nil {
		return nil
	}

	var warnings []ChannelHealthWarning
	for name, sl := range learning.Skills {
		if sl.TotalCompletions >= MinCompletionsForChannelHealthCheck && sl.ReworkCount == 0 {
			warnings = append(warnings, ChannelHealthWarning{
				Skill:       name,
				Completions: sl.TotalCompletions,
				Message: fmt.Sprintf(
					"skill %q has %d completions but 0 reworks — rework channel may be inactive (absent signal ≠ positive signal)",
					name, sl.TotalCompletions,
				),
			})
		}
	}
	return warnings
}

// inferSkillForScoring infers the skill for scoring purposes.
// Uses the same inference chain as InferSkillFromIssue but without event logging
// (scoring is speculative — we don't want to log an inference event for every
// candidate in every poll cycle).
func inferSkillForScoring(issue Issue) string {
	// Skill label takes precedence
	if skill := InferSkillFromLabels(issue.Labels); skill != "" {
		return skill
	}
	// Title pattern
	if skill := InferSkillFromTitle(issue.Title); skill != "" {
		return skill
	}
	// Description heuristic
	if skill := InferSkillFromDescription(issue.Description); skill != "" {
		return skill
	}
	// Type-based fallback
	skill, err := InferSkill(issue.IssueType)
	if err != nil {
		return "feature-impl" // safe fallback
	}
	return skill
}
