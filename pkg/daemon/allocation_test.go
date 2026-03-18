package daemon

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestScoreIssue_BasePriorityOnly(t *testing.T) {
	// With no learning data, scoring should preserve priority ordering
	learning := &events.LearningStore{Skills: map[string]*events.SkillLearning{}}

	p0 := Issue{ID: "a-1", Priority: 0, IssueType: "feature"}
	p2 := Issue{ID: "a-2", Priority: 2, IssueType: "feature"}
	p4 := Issue{ID: "a-3", Priority: 4, IssueType: "feature"}

	s0 := ScoreIssue(p0, learning)
	s2 := ScoreIssue(p2, learning)
	s4 := ScoreIssue(p4, learning)

	if s0.Score <= s2.Score {
		t.Errorf("P0 score (%f) should be > P2 score (%f)", s0.Score, s2.Score)
	}
	if s2.Score <= s4.Score {
		t.Errorf("P2 score (%f) should be > P4 score (%f)", s2.Score, s4.Score)
	}
}

func TestScoreIssue_SkillSuccessRateBoost(t *testing.T) {
	// A high-success-rate skill should score higher than a low-success-rate skill
	// at the same priority level
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       10,
				TotalCompletions: 10,
				SuccessCount:     9,
				SuccessRate:      0.9,
			},
			"systematic-debugging": {
				SpawnCount:       10,
				TotalCompletions: 10,
				SuccessCount:     3,
				SuccessRate:      0.3,
			},
		},
	}

	feature := Issue{ID: "a-1", Priority: 2, IssueType: "feature"} // infers feature-impl
	bug := Issue{ID: "a-2", Priority: 2, IssueType: "bug"}         // infers systematic-debugging

	featureScore := ScoreIssue(feature, learning)
	bugScore := ScoreIssue(bug, learning)

	if featureScore.Score <= bugScore.Score {
		t.Errorf("feature-impl (90%% success) score (%f) should be > systematic-debugging (30%% success) score (%f)",
			featureScore.Score, bugScore.Score)
	}
	// With ReworkCount=0, no ground-truth adjustment — use self-reported rate.
	// At 10 samples (full weight), blended rate = 0.9.
	if featureScore.SkillSuccessRate < 0.89 || featureScore.SkillSuccessRate > 0.91 {
		t.Errorf("SkillSuccessRate = %f, want ~0.9 (self-reported, no rework data)", featureScore.SkillSuccessRate)
	}
}

func TestScoreIssue_UnknownSkillGetsDefaultRate(t *testing.T) {
	// Skills without learning data should get a neutral success rate (0.5)
	learning := &events.LearningStore{Skills: map[string]*events.SkillLearning{}}

	issue := Issue{ID: "a-1", Priority: 2, IssueType: "feature"}
	score := ScoreIssue(issue, learning)

	if score.SkillSuccessRate != DefaultSuccessRate {
		t.Errorf("SkillSuccessRate = %f, want %f (default)", score.SkillSuccessRate, DefaultSuccessRate)
	}
}

func TestScoreIssue_LowSampleSizeBlendedWithDefault(t *testing.T) {
	// With only 1 completion, the success rate should be blended toward the default
	// to avoid overreacting to small sample sizes
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       1,
				TotalCompletions: 1,
				SuccessCount:     1,
				SuccessRate:      1.0,
			},
		},
	}

	issue := Issue{ID: "a-1", Priority: 2, IssueType: "feature"}
	score := ScoreIssue(issue, learning)

	// With 1 sample, should be blended between 1.0 and default (0.5)
	// Not as high as 1.0, not as low as 0.5
	if score.SkillSuccessRate >= 1.0 {
		t.Errorf("SkillSuccessRate = %f, should be blended below 1.0 with 1 sample", score.SkillSuccessRate)
	}
	if score.SkillSuccessRate <= DefaultSuccessRate {
		t.Errorf("SkillSuccessRate = %f, should be above default %f with perfect record", score.SkillSuccessRate, DefaultSuccessRate)
	}
}

func TestScoreIssues_SortedByScore(t *testing.T) {
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       20,
				TotalCompletions: 20,
				SuccessCount:     18,
				SuccessRate:      0.9,
			},
			"systematic-debugging": {
				SpawnCount:       20,
				TotalCompletions: 20,
				SuccessCount:     6,
				SuccessRate:      0.3,
			},
		},
	}

	issues := []Issue{
		{ID: "a-1", Priority: 2, IssueType: "bug"},     // low success skill
		{ID: "a-2", Priority: 2, IssueType: "feature"},  // high success skill
		{ID: "a-3", Priority: 0, IssueType: "bug"},      // high priority, low success skill
	}

	scored := ScoreIssues(issues, learning)

	if len(scored) != 3 {
		t.Fatalf("ScoreIssues() returned %d scores, want 3", len(scored))
	}

	// Should be sorted descending by score
	for i := 1; i < len(scored); i++ {
		if scored[i].Score > scored[i-1].Score {
			t.Errorf("ScoreIssues() not sorted: index %d score (%f) > index %d score (%f)",
				i, scored[i].Score, i-1, scored[i-1].Score)
		}
	}
}

func TestScoreIssue_InferredFields(t *testing.T) {
	learning := &events.LearningStore{Skills: map[string]*events.SkillLearning{}}

	feature := Issue{ID: "a-1", Priority: 2, IssueType: "feature"}
	score := ScoreIssue(feature, learning)

	if score.InferredSkill != "feature-impl" {
		t.Errorf("InferredSkill = %q, want %q", score.InferredSkill, "feature-impl")
	}
}

func TestScoreIssue_SkillLabelOverridesTypeInference(t *testing.T) {
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"architect": {
				SpawnCount:       10,
				TotalCompletions: 10,
				SuccessCount:     8,
				SuccessRate:      0.8,
			},
		},
	}

	// Issue type says "feature" but label says "skill:architect"
	issue := Issue{ID: "a-1", Priority: 2, IssueType: "feature", Labels: []string{"skill:architect"}}
	score := ScoreIssue(issue, learning)

	if score.InferredSkill != "architect" {
		t.Errorf("InferredSkill = %q, want %q (from label)", score.InferredSkill, "architect")
	}
	// With ReworkCount=0, no ground-truth adjustment — use self-reported rate.
	// At 10 samples (full weight), blended rate = 0.8.
	if score.SkillSuccessRate < 0.79 || score.SkillSuccessRate > 0.81 {
		t.Errorf("SkillSuccessRate = %f, want ~0.8 (self-reported, no rework data)", score.SkillSuccessRate)
	}
}

func TestPrioritizeIssues_WithLearning(t *testing.T) {
	// When learning data is available, PrioritizeIssues should use scored ranking
	d := &Daemon{
		Learning: &events.LearningStore{
			Skills: map[string]*events.SkillLearning{
				"feature-impl": {
					SpawnCount:       20,
					TotalCompletions: 20,
					SuccessCount:     18,
					SuccessRate:      0.9,
				},
				"systematic-debugging": {
					SpawnCount:       20,
					TotalCompletions: 20,
					SuccessCount:     4,
					SuccessRate:      0.2,
				},
			},
		},
	}

	issues := []Issue{
		{ID: "a-1", Priority: 1, IssueType: "bug"},     // P1 bug, low success skill
		{ID: "a-2", Priority: 2, IssueType: "feature"},  // P2 feature, high success skill
	}

	sorted, _, err := d.PrioritizeIssues(issues)
	if err != nil {
		t.Fatalf("PrioritizeIssues() error: %v", err)
	}

	// The high-success feature at P2 should potentially beat the low-success bug at P1
	// because scoring blends priority with success rate
	if len(sorted) != 2 {
		t.Fatalf("PrioritizeIssues() returned %d issues, want 2", len(sorted))
	}

	// With 90% vs 20% success rate and only 1 priority level apart,
	// the feature-impl should rank higher
	if sorted[0].ID != "a-2" {
		t.Errorf("PrioritizeIssues() first issue = %s, want a-2 (high success feature)", sorted[0].ID)
	}
}

func TestPrioritizeIssues_WithoutLearning_FallsBackToPriority(t *testing.T) {
	// Without learning data, should fall back to pure priority sorting
	d := &Daemon{}

	issues := []Issue{
		{ID: "a-2", Priority: 2, IssueType: "feature"},
		{ID: "a-1", Priority: 0, IssueType: "bug"},
	}

	sorted, _, err := d.PrioritizeIssues(issues)
	if err != nil {
		t.Fatalf("PrioritizeIssues() error: %v", err)
	}

	if sorted[0].ID != "a-1" {
		t.Errorf("PrioritizeIssues() first issue = %s, want a-1 (P0)", sorted[0].ID)
	}
}

func TestScoreIssue_HighPriorityStillWinsWithModerateSuccessGap(t *testing.T) {
	// P0 should still beat P3 even if P3's skill has slightly better success rate
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       20,
				TotalCompletions: 20,
				SuccessCount:     14,
				SuccessRate:      0.7,
			},
			"systematic-debugging": {
				SpawnCount:       20,
				TotalCompletions: 20,
				SuccessCount:     10,
				SuccessRate:      0.5,
			},
		},
	}

	p0Bug := Issue{ID: "a-1", Priority: 0, IssueType: "bug"}       // P0, 50% success
	p3Feature := Issue{ID: "a-2", Priority: 3, IssueType: "feature"} // P3, 70% success

	s0 := ScoreIssue(p0Bug, learning)
	s3 := ScoreIssue(p3Feature, learning)

	if s0.Score <= s3.Score {
		t.Errorf("P0 bug score (%f) should beat P3 feature score (%f) — priority dominates moderate success gap",
			s0.Score, s3.Score)
	}
}

func TestScoreIssue_GroundTruthAdjustedRate(t *testing.T) {
	// When rework data is available, the allocation score should use
	// ground-truth-adjusted rate: 0.7 * selfReported + 0.3 * (1 - reworkRate)
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       20,
				TotalCompletions: 20,
				SuccessCount:     18,
				SuccessRate:      0.9,     // Self-reported: 90%
				ReworkCount:      6,
				ReworkRate:        0.3,     // 30% rework → ground truth = 70%
			},
			"investigation": {
				SpawnCount:       20,
				TotalCompletions: 20,
				SuccessCount:     19,
				SuccessRate:      0.95,    // Self-reported: 95%
				ReworkCount:      0,
				ReworkRate:        0.0,     // No rework data — channel unused
			},
		},
	}

	feature := Issue{ID: "a-1", Priority: 2, IssueType: "feature"}
	featureScore := ScoreIssue(feature, learning)

	// Expected adjusted rate: 0.7 * 0.9 + 0.3 * (1 - 0.3) = 0.63 + 0.21 = 0.84
	// (after blending with sample size, which at 20 samples is nearly full weight)
	// The adjusted rate should be lower than pure self-reported (0.9)
	if featureScore.SkillSuccessRate >= 0.9 {
		t.Errorf("SkillSuccessRate = %f, should be below 0.9 with 30%% rework rate", featureScore.SkillSuccessRate)
	}

	investigation := Issue{ID: "a-2", Priority: 2, IssueType: "task", Labels: []string{"skill:investigation"}}
	invScore := ScoreIssue(investigation, learning)

	// Investigation with ReworkCount=0: no ground truth adjustment, stays at self-reported 0.95
	if invScore.SkillSuccessRate < 0.94 || invScore.SkillSuccessRate > 0.96 {
		t.Errorf("SkillSuccessRate = %f, want ~0.95 (self-reported, no rework data)", invScore.SkillSuccessRate)
	}
}

func TestGroundTruthAdjustedRate(t *testing.T) {
	tests := []struct {
		name           string
		selfReported   float64
		reworkRate     float64
		hasReworkData  bool
		wantMin        float64
		wantMax        float64
	}{
		{"no rework data uses self-reported only", 0.9, 0.0, false, 0.89, 0.91},
		{"zero rework boosts rate", 0.8, 0.0, true, 0.85, 0.87},
		{"high rework penalizes rate", 0.9, 0.5, true, 0.77, 0.79},
		{"perfect self-report with 100% rework", 1.0, 1.0, true, 0.69, 0.71},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroundTruthAdjustedRate(tt.selfReported, tt.reworkRate, tt.hasReworkData)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("GroundTruthAdjustedRate(%f, %f, %v) = %f, want in [%f, %f]",
					tt.selfReported, tt.reworkRate, tt.hasReworkData, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestLookupSuccessRate_ZeroReworkNotTreatedAsGroundTruth(t *testing.T) {
	// Bug: when TotalCompletions >= MinSamplesForFullWeight but ReworkCount == 0,
	// the old logic treated this as "rework data exists" and blended in
	// groundTruth = 1.0 - 0.0 = 1.0, inflating the rate by ~7.3pp.
	// Fix: hasReworkData should require ReworkCount > 0.
	learning := &events.LearningStore{
		Skills: map[string]*events.SkillLearning{
			"feature-impl": {
				SpawnCount:       15,
				TotalCompletions: 15,
				SuccessCount:     12,
				AbandonedCount:   0,
				SuccessRate:      0.8,
				ReworkCount:      0, // No rework data — channel unused
				ReworkRate:       0.0,
			},
		},
	}

	issue := Issue{ID: "a-1", Priority: 2, IssueType: "feature"}
	score := ScoreIssue(issue, learning)

	// With no rework data (ReworkCount=0), ground truth should NOT be applied.
	// Self-reported rate = 0.8, fully weighted at 15 samples.
	// Blended = 0.8 (since 15 >= MinSamplesForFullWeight).
	//
	// BUG behavior: hasReworkData=true → adjusted = 0.7*0.8 + 0.3*1.0 = 0.86
	// CORRECT behavior: hasReworkData=false → adjusted = 0.8 (self-reported only)
	if score.SkillSuccessRate > 0.81 {
		t.Errorf("SkillSuccessRate = %f, want ~0.8 (self-reported only, no rework data); "+
			"got inflated rate suggesting ground truth was incorrectly applied", score.SkillSuccessRate)
	}
	if score.SkillSuccessRate < 0.79 {
		t.Errorf("SkillSuccessRate = %f, want ~0.8", score.SkillSuccessRate)
	}
}

func TestBlendedSuccessRate(t *testing.T) {
	tests := []struct {
		name        string
		observed    float64
		sampleSize  int
		wantMin     float64
		wantMax     float64
	}{
		{"no samples returns default", 0.0, 0, DefaultSuccessRate, DefaultSuccessRate},
		{"1 sample blended", 1.0, 1, DefaultSuccessRate, 1.0},
		{"20 samples nearly observed", 0.9, 20, 0.85, 0.95},
		{"high sample fully observed", 0.8, 100, 0.79, 0.81},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlendedSuccessRate(tt.observed, tt.sampleSize)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("BlendedSuccessRate(%f, %d) = %f, want in [%f, %f]",
					tt.observed, tt.sampleSize, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}
