package orient

import (
	"fmt"
	"math"
	"strings"
)

// DivergenceThreshold is the minimum gap (0.0-1.0) before an alert fires.
const DivergenceThreshold = 0.20

// OrphanRateThreshold is the orphan rate percentage above which an alert fires.
const OrphanRateThreshold = 30.0

// StaleDecisionThreshold is the fraction of stale decisions above which an alert fires.
const StaleDecisionThreshold = 0.25

// DivergenceInput holds the metrics needed to compute divergence alerts.
type DivergenceInput struct {
	// Activity metrics (self-reported)
	CompletionRate      float64 // completions / spawns (0.0-1.0)
	SelfReportedSuccess float64 // success rate from events.jsonl (0.0-1.0)

	// Ground-truth metrics (external)
	ReworkRate float64 // rework count / completions (0.0-1.0)
	OrphanRate     float64 // investigation orphan percentage (0-100)
	StaleDecisions int     // count of stale decisions from reflect
	TotalDecisions int     // total decisions for stale rate computation

	Days int // time window for context
}

// DivergenceAlert represents a detected divergence between activity and impact metrics.
type DivergenceAlert struct {
	Type    string  `json:"type"`    // "merge_gap", "rework_gap", "orphan_rate", "stale_decisions"
	Message string  `json:"message"` // human-readable description
	Gap     float64 `json:"gap"`     // magnitude of divergence (0.0-1.0)
	Level   string  `json:"level"`   // "warning" or "critical"
}

// ComputeDivergence compares activity metrics against impact metrics and returns
// alerts for sustained gaps. Returns nil when no divergence exceeds thresholds.
// Fails open: missing data (zero values) produces no alerts, not false positives.
func ComputeDivergence(input DivergenceInput) []DivergenceAlert {
	var alerts []DivergenceAlert

	// Rework gap: self-reported success vs (1 - rework rate)
	// Only meaningful when both signals are present
	if input.SelfReportedSuccess > 0 && input.ReworkRate > 0 {
		groundTruthSuccess := 1.0 - input.ReworkRate
		gap := math.Abs(input.SelfReportedSuccess - groundTruthSuccess)
		if gap >= DivergenceThreshold {
			alerts = append(alerts, DivergenceAlert{
				Type:    "rework_gap",
				Message: fmt.Sprintf("%d%% self-reported success but %d%% rework rate", pct(input.SelfReportedSuccess), pct(input.ReworkRate)),
				Gap:     gap,
				Level:   alertLevel(gap),
			})
		}
	}

	// Orphan rate: high investigation orphan rate means work isn't connecting
	if input.OrphanRate >= OrphanRateThreshold {
		alerts = append(alerts, DivergenceAlert{
			Type:    "orphan_rate",
			Message: fmt.Sprintf("%.0f%% investigation orphan rate — work not connecting to knowledge base", input.OrphanRate),
			Gap:     input.OrphanRate / 100.0,
			Level:   alertLevel(input.OrphanRate / 100.0),
		})
	}

	// Stale decisions: many stale decisions means activity isn't following through
	if input.TotalDecisions > 0 {
		staleRate := float64(input.StaleDecisions) / float64(input.TotalDecisions)
		if staleRate >= StaleDecisionThreshold {
			alerts = append(alerts, DivergenceAlert{
				Type:    "stale_decisions",
				Message: fmt.Sprintf("%d/%d decisions stale (%.0f%%) — decisions not being acted on", input.StaleDecisions, input.TotalDecisions, staleRate*100),
				Gap:     staleRate,
				Level:   alertLevel(staleRate),
			})
		}
	}

	return alerts
}

// FormatDivergenceAlerts renders divergence alerts as structured text.
func FormatDivergenceAlerts(alerts []DivergenceAlert) string {
	if len(alerts) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Metric divergence:\n")
	for _, a := range alerts {
		icon := "[!]"
		if a.Level == "critical" {
			icon = "[!!!]"
		}
		b.WriteString(fmt.Sprintf("   %s %s: %s\n", icon, a.Type, a.Message))
	}
	b.WriteString("\n")
	return b.String()
}

// pct converts a 0.0-1.0 fraction to a percentage integer.
func pct(f float64) int {
	return int(f * 100)
}

// alertLevel returns "critical" for gaps >= 40%, "warning" otherwise.
func alertLevel(gap float64) string {
	if gap >= 0.40 {
		return "critical"
	}
	return "warning"
}
