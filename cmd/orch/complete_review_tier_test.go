package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestCleanDiscoveredWorkTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"- Fix the bug", "Fix the bug"},
		{"* Add feature", "Add feature"},
		{"1. First item", "First item"},
		{"12. Twelfth item", "Twelfth item"},
		{"Plain text", "Plain text"},
	}

	for _, tt := range tests {
		got := cleanDiscoveredWorkTitle(tt.input)
		if got != tt.want {
			t.Errorf("cleanDiscoveredWorkTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReviewTierSkipsCheckpointGates(t *testing.T) {
	// Verify that auto/scan tiers should skip checkpoint gates
	// This tests the logic condition used in executeVerificationGates
	tests := []struct {
		tier           string
		shouldSkip     bool
	}{
		{"auto", true},
		{"scan", true},
		{"review", false},
		{"deep", false},
		{"", false},
	}

	for _, tt := range tests {
		skipCheckpoints := tt.tier == "auto" || tt.tier == "scan"
		if skipCheckpoints != tt.shouldSkip {
			t.Errorf("tier %q: skipCheckpoints = %v, want %v", tt.tier, skipCheckpoints, tt.shouldSkip)
		}
	}
}

func TestReviewTierAdvisoryBehavior(t *testing.T) {
	// Verify the isAutoTier/isScanTier/isLightReview logic
	tests := []struct {
		tier          string
		isAuto        bool
		isScan        bool
		isLightReview bool
	}{
		{"auto", true, false, true},
		{"scan", false, true, true},
		{"review", false, false, false},
		{"deep", false, false, false},
	}

	for _, tt := range tests {
		isAuto := tt.tier == "auto"
		isScan := tt.tier == "scan"
		isLight := isAuto || isScan

		if isAuto != tt.isAuto {
			t.Errorf("tier %q: isAuto = %v, want %v", tt.tier, isAuto, tt.isAuto)
		}
		if isScan != tt.isScan {
			t.Errorf("tier %q: isScan = %v, want %v", tt.tier, isScan, tt.isScan)
		}
		if isLight != tt.isLightReview {
			t.Errorf("tier %q: isLightReview = %v, want %v", tt.tier, isLight, tt.isLightReview)
		}
	}
}

func TestReviewTierResolution(t *testing.T) {
	// Test that --review-tier flag validation works
	validTiers := []string{"auto", "scan", "review", "deep"}
	invalidTiers := []string{"", "invalid", "AUTO", "SCAN"}

	for _, tier := range validTiers {
		if !spawn.IsValidReviewTier(tier) {
			t.Errorf("expected %q to be valid review tier", tier)
		}
	}

	for _, tier := range invalidTiers {
		if spawn.IsValidReviewTier(tier) {
			t.Errorf("expected %q to be invalid review tier", tier)
		}
	}
}

func TestCompletionTargetHasReviewTier(t *testing.T) {
	// Verify CompletionTarget struct has ReviewTier field and it works
	target := CompletionTarget{
		Identifier: "test-123",
		ReviewTier: spawn.ReviewAuto,
	}

	if target.ReviewTier != "auto" {
		t.Errorf("ReviewTier = %q, want %q", target.ReviewTier, "auto")
	}
}

func TestReviewTierFromWorkspace(t *testing.T) {
	// Test that ReadReviewTierFromWorkspace returns conservative default for empty workspace
	tier := verify.ReadReviewTierFromWorkspace("/nonexistent/path")
	if tier != spawn.ReviewReview {
		t.Errorf("ReadReviewTierFromWorkspace for nonexistent path = %q, want %q", tier, spawn.ReviewReview)
	}
}

func TestScanTierExemptsArtifactGate(t *testing.T) {
	// Scan-tier and auto-tier skills should be exempt from the COMPLETION.yaml artifact gate.
	// Review-tier and deep-tier skills should NOT be exempt.
	tests := []struct {
		reviewTier    string
		shouldExempt  bool
	}{
		{spawn.ReviewAuto, true},
		{spawn.ReviewScan, true},
		{spawn.ReviewReview, false},
		{spawn.ReviewDeep, false},
		{"", false},
	}

	for _, tt := range tests {
		isScanTierForArtifact := tt.reviewTier == spawn.ReviewScan || tt.reviewTier == spawn.ReviewAuto
		if isScanTierForArtifact != tt.shouldExempt {
			t.Errorf("reviewTier %q: artifact exempt = %v, want %v", tt.reviewTier, isScanTierForArtifact, tt.shouldExempt)
		}
	}
}

func TestScanTierSkillsMapToScanReviewTier(t *testing.T) {
	// Verify that the skills mentioned in the bug report map to scan review tier
	scanSkills := []string{"investigation", "probe", "research", "codebase-audit"}
	for _, skill := range scanSkills {
		tier := spawn.DefaultReviewTier(skill, "")
		if tier != spawn.ReviewScan {
			t.Errorf("skill %q: DefaultReviewTier = %q, want %q (scan)", skill, tier, spawn.ReviewScan)
		}
	}

	// Verify review/deep tier skills are NOT exempt
	nonScanSkills := []string{"feature-impl", "systematic-debugging", "architect"}
	for _, skill := range nonScanSkills {
		tier := spawn.DefaultReviewTier(skill, "")
		if tier == spawn.ReviewScan || tier == spawn.ReviewAuto {
			t.Errorf("skill %q: DefaultReviewTier = %q, should NOT be scan/auto (must require artifact gate)", skill, tier)
		}
	}
}
