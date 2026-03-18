// Package daemon provides autonomous overnight processing capabilities.
// allocation.go implements skill-aware slot scoring for the daemon's Orient phase.
// Instead of first-eligible spawning, it scores candidate issues by skill success
// rate, model fit, and base priority to produce a ranked allocation profile.
package daemon

import (
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

	// GroundTruthWeight controls how much ground-truth (rework rate) influences the
	// adjusted success rate. At 0.3, the adjusted rate is 70% self-reported + 30% ground truth.
	// The lower weight reflects that rework data is sparser than completion data.
	GroundTruthWeight = 0.3
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

// GroundTruthAdjustedRate blends self-reported success rate with ground-truth signal
// (1 - reworkRate) at 70/30 weighting. When no rework data exists (hasReworkData=false),
// returns the self-reported rate unchanged.
//
// Formula: adjustedRate = (1 - GroundTruthWeight) * selfReported + GroundTruthWeight * (1 - reworkRate)
func GroundTruthAdjustedRate(selfReported, reworkRate float64, hasReworkData bool) float64 {
	if !hasReworkData {
		return selfReported
	}
	groundTruthRate := 1.0 - reworkRate
	return (1-GroundTruthWeight)*selfReported + GroundTruthWeight*groundTruthRate
}

// lookupSuccessRate returns the ground-truth-adjusted success rate for a skill.
// Blends self-reported success rate with rework-based ground truth at 70/30,
// then blends with the default rate based on sample size.
func lookupSuccessRate(skill string, learning *events.LearningStore) float64 {
	if learning == nil {
		return DefaultSuccessRate
	}

	sl, ok := learning.Skills[skill]
	if !ok {
		return DefaultSuccessRate
	}

	sampleSize := sl.TotalCompletions + sl.AbandonedCount

	// Start with self-reported rate, blend with ground truth if available
	hasReworkData := sl.ReworkCount > 0 || sl.TotalCompletions >= MinSamplesForFullWeight
	adjustedRate := GroundTruthAdjustedRate(sl.SuccessRate, sl.ReworkRate, hasReworkData)

	return BlendedSuccessRate(adjustedRate, sampleSize)
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
