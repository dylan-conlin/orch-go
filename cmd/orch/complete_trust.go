// Package main provides trust calibration tier computation for the completion review.
// The trust tier (green/yellow/red) guides risk-proportional verification pacing:
// the orchestrator invests more review effort for higher-risk completions.
package main

import (
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// TrustTier represents the risk-proportional trust calibration for completion review.
type TrustTier string

const (
	TrustGreen  TrustTier = "green"  // Quick-ack: minimal orchestrator review
	TrustYellow TrustTier = "yellow" // Standard review: read synthesis, check diff
	TrustRed    TrustTier = "red"    // Deep scrutiny: explain-back, behavioral verification
)

// ComputeTrustTier determines the trust calibration tier based on risk signals.
// The tier guides orchestrator review effort:
//   - GREEN: quick ack (low risk — auto/scan review tier, no bypasses)
//   - YELLOW: standard review (medium risk — review tier, no bypasses)
//   - RED: deep scrutiny (high risk — deep tier, bypasses, or force)
//
// Escalation rules:
//   - Force completion (--force) always escalates to RED
//   - Skip flags (--skip-*) always escalate to RED
//   - Review tier provides the baseline: auto/scan→GREEN, review→YELLOW, deep→RED
func ComputeTrustTier(reviewTier string, skipConfig verify.SkipConfig, force bool) TrustTier {
	// Any bypass or force → RED (circumvented verification)
	if force || skipConfig.HasAnySkip() {
		return TrustRed
	}

	switch reviewTier {
	case spawn.ReviewAuto, spawn.ReviewScan:
		return TrustGreen
	case spawn.ReviewDeep:
		return TrustRed
	default: // "review" or unknown
		return TrustYellow
	}
}

// formatTrustTier returns a human-readable display string for the trust tier.
func formatTrustTier(tier TrustTier) string {
	switch tier {
	case TrustGreen:
		return "🟢 GREEN (quick-ack)"
	case TrustYellow:
		return "🟡 YELLOW (standard review)"
	case TrustRed:
		return "🔴 RED (deep scrutiny)"
	default:
		return "🟡 YELLOW (standard review)"
	}
}
