package orch

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestBuildSpawnConfig_ReviewTierInferred(t *testing.T) {
	tests := []struct {
		name      string
		skill     string
		issueType string
		isBug     bool
		wantTier  string
	}{
		{
			name:     "feature-impl defaults to review",
			skill:    "feature-impl",
			wantTier: spawn.ReviewReview,
		},
		{
			name:     "capture-knowledge defaults to auto",
			skill:    "capture-knowledge",
			wantTier: spawn.ReviewAuto,
		},
		{
			name:     "investigation defaults to scan",
			skill:    "investigation",
			wantTier: spawn.ReviewScan,
		},
		{
			name:     "debug-with-playwright defaults to deep",
			skill:    "debug-with-playwright",
			wantTier: spawn.ReviewDeep,
		},
		{
			name:      "investigation with feature issue type elevated to review",
			skill:     "investigation",
			issueType: "feature",
			wantTier:  spawn.ReviewReview,
		},
		{
			name:     "bug flag infers bug issue type and elevates",
			skill:    "investigation",
			isBug:    true,
			wantTier: spawn.ReviewReview,
		},
		{
			name:     "unknown skill defaults to review",
			skill:    "custom-skill",
			wantTier: spawn.ReviewReview,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &SpawnContext{
				SkillName:    tt.skill,
				IssueType:    tt.issueType,
				IsBug:        tt.isBug,
				ResolvedModel: model.ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"},
			}
			cfg := BuildSpawnConfig(ctx, "", "tdd", "tests", "", "", false, false, "")
			if cfg.ReviewTier != tt.wantTier {
				t.Errorf("ReviewTier = %q, want %q", cfg.ReviewTier, tt.wantTier)
			}
		})
	}
}

func TestBuildSpawnConfig_ReviewTierExplicit(t *testing.T) {
	// Explicit --review-tier flag should override inference
	ctx := &SpawnContext{
		SkillName:    "capture-knowledge", // would default to auto
		ReviewTier:   spawn.ReviewDeep,     // explicit override
		ResolvedModel: model.ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"},
	}
	cfg := BuildSpawnConfig(ctx, "", "tdd", "tests", "", "", false, false, "")
	if cfg.ReviewTier != spawn.ReviewDeep {
		t.Errorf("ReviewTier = %q, want %q (explicit override)", cfg.ReviewTier, spawn.ReviewDeep)
	}
}

func TestBuildSpawnConfig_ReviewTierIssueTypeFromBug(t *testing.T) {
	// When IssueType is empty but IsBug is true, should use "bug" as issue type
	ctx := &SpawnContext{
		SkillName:    "probe",
		IsBug:        true,
		ResolvedModel: model.ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"},
	}
	cfg := BuildSpawnConfig(ctx, "", "tdd", "tests", "", "", false, false, "")
	// probe defaults to scan, but bug issue type has minimum of review
	if cfg.ReviewTier != spawn.ReviewReview {
		t.Errorf("ReviewTier = %q, want %q (bug issue elevates)", cfg.ReviewTier, spawn.ReviewReview)
	}
}
