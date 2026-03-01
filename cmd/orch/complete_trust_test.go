package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestComputeTrustTier_ReviewTierMapping(t *testing.T) {
	noSkip := verify.SkipConfig{}

	tests := []struct {
		name       string
		reviewTier string
		want       TrustTier
	}{
		{"auto → green", spawn.ReviewAuto, TrustGreen},
		{"scan → green", spawn.ReviewScan, TrustGreen},
		{"review → yellow", spawn.ReviewReview, TrustYellow},
		{"deep → red", spawn.ReviewDeep, TrustRed},
		{"empty → yellow", "", TrustYellow},
		{"unknown → yellow", "unknown", TrustYellow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeTrustTier(tt.reviewTier, noSkip, false)
			if got != tt.want {
				t.Errorf("ComputeTrustTier(%q, noSkip, false) = %q, want %q", tt.reviewTier, got, tt.want)
			}
		})
	}
}

func TestComputeTrustTier_ForceEscalatesToRed(t *testing.T) {
	noSkip := verify.SkipConfig{}

	// Even auto/scan should escalate to red when force is used
	tiers := []string{spawn.ReviewAuto, spawn.ReviewScan, spawn.ReviewReview, spawn.ReviewDeep}
	for _, tier := range tiers {
		got := ComputeTrustTier(tier, noSkip, true)
		if got != TrustRed {
			t.Errorf("ComputeTrustTier(%q, noSkip, force=true) = %q, want red", tier, got)
		}
	}
}

func TestComputeTrustTier_SkipFlagsEscalateToRed(t *testing.T) {
	skipConfig := verify.SkipConfig{
		TestEvidence: true,
		Reason:       "Tests run in CI",
	}

	// Even auto/scan should escalate to red when gates are bypassed
	tiers := []string{spawn.ReviewAuto, spawn.ReviewScan, spawn.ReviewReview}
	for _, tier := range tiers {
		got := ComputeTrustTier(tier, skipConfig, false)
		if got != TrustRed {
			t.Errorf("ComputeTrustTier(%q, skip=true, false) = %q, want red", tier, got)
		}
	}
}

func TestComputeTrustTier_SkipAndForceStillRed(t *testing.T) {
	skipConfig := verify.SkipConfig{
		Build:  true,
		Reason: "Build system broken",
	}

	got := ComputeTrustTier(spawn.ReviewAuto, skipConfig, true)
	if got != TrustRed {
		t.Errorf("ComputeTrustTier(auto, skip+force) = %q, want red", got)
	}
}

func TestFormatTrustTier(t *testing.T) {
	tests := []struct {
		tier TrustTier
		want string
	}{
		{TrustGreen, "🟢 GREEN (quick-ack)"},
		{TrustYellow, "🟡 YELLOW (standard review)"},
		{TrustRed, "🔴 RED (deep scrutiny)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			got := formatTrustTier(tt.tier)
			if got != tt.want {
				t.Errorf("formatTrustTier(%q) = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func TestFormatTrustTier_Unknown(t *testing.T) {
	got := formatTrustTier(TrustTier("unknown"))
	if got != "🟡 YELLOW (standard review)" {
		t.Errorf("formatTrustTier(unknown) = %q, want yellow default", got)
	}
}
