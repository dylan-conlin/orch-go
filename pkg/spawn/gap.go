// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"strings"
)

// Gap detection thresholds.
const (
	// MinMatchesForGapDetection is the threshold below which we consider results sparse.
	// If fewer than this many matches are found, we flag a potential context gap.
	MinMatchesForGapDetection = 2

	// HighConfidenceMatchThreshold is the number of matches that indicates good coverage.
	// When we have this many or more matches, context coverage is considered adequate.
	HighConfidenceMatchThreshold = 5
)

// GapType categorizes the type of context gap detected.
type GapType string

const (
	// GapTypeNoContext indicates no KB context was found for the task.
	GapTypeNoContext GapType = "no_context"

	// GapTypeSparseContext indicates very few KB context matches were found.
	GapTypeSparseContext GapType = "sparse_context"

	// GapTypeNoConstraints indicates context was found but no constraints.
	GapTypeNoConstraints GapType = "no_constraints"

	// GapTypeNoDecisions indicates context was found but no prior decisions.
	GapTypeNoDecisions GapType = "no_decisions"
)

// GapSeverity indicates how significant the detected gap is.
type GapSeverity string

const (
	// GapSeverityInfo is for informational gaps that might be normal.
	GapSeverityInfo GapSeverity = "info"

	// GapSeverityWarning is for gaps that should be noted but don't block.
	GapSeverityWarning GapSeverity = "warning"

	// GapSeverityCritical is for gaps that likely indicate missing context.
	GapSeverityCritical GapSeverity = "critical"
)

// Gap represents a single detected context gap.
type Gap struct {
	Type        GapType     // The type of gap detected
	Severity    GapSeverity // How significant this gap is
	Description string      // Human-readable description of the gap
	Suggestion  string      // Suggested action to address the gap
}

// GapAnalysis contains the results of analyzing KB context for gaps.
type GapAnalysis struct {
	// HasGaps indicates whether any gaps were detected.
	HasGaps bool

	// Gaps is the list of detected gaps.
	Gaps []Gap

	// ContextQuality is a score from 0-100 indicating context coverage.
	// 0 = no context, 100 = comprehensive context.
	ContextQuality int

	// MatchStats contains statistics about the matches found.
	MatchStats MatchStatistics

	// Query is the original query that was searched.
	Query string
}

// MatchStatistics contains detailed statistics about KB context matches.
type MatchStatistics struct {
	TotalMatches       int
	ConstraintCount    int
	DecisionCount      int
	InvestigationCount int
	GuideCount         int
}

// AnalyzeGaps analyzes a KB context result for potential gaps.
// Returns a GapAnalysis with detected gaps and context quality score.
func AnalyzeGaps(result *KBContextResult, query string) *GapAnalysis {
	analysis := &GapAnalysis{
		Query: query,
		Gaps:  []Gap{},
	}

	// No result at all
	if result == nil || !result.HasMatches || len(result.Matches) == 0 {
		analysis.HasGaps = true
		analysis.ContextQuality = 0
		analysis.Gaps = append(analysis.Gaps, Gap{
			Type:        GapTypeNoContext,
			Severity:    GapSeverityCritical,
			Description: fmt.Sprintf("No prior knowledge found for query %q", query),
			Suggestion:  "Consider running 'kb context' manually to verify, or add relevant kn entries/investigations",
		})
		return analysis
	}

	// Count matches by type
	stats := countMatchesByType(result.Matches)
	analysis.MatchStats = stats

	// Check for sparse context
	if stats.TotalMatches < MinMatchesForGapDetection {
		analysis.HasGaps = true
		analysis.Gaps = append(analysis.Gaps, Gap{
			Type:        GapTypeSparseContext,
			Severity:    GapSeverityWarning,
			Description: fmt.Sprintf("Only %d match(es) found - context may be incomplete", stats.TotalMatches),
			Suggestion:  "Agent may need to discover context during work; consider adding relevant kn entries",
		})
	}

	// Check for missing constraints (often critical for system behavior)
	if stats.ConstraintCount == 0 && stats.TotalMatches > 0 {
		analysis.HasGaps = true
		analysis.Gaps = append(analysis.Gaps, Gap{
			Type:        GapTypeNoConstraints,
			Severity:    GapSeverityInfo,
			Description: "No constraints found - agent may not know system limitations",
			Suggestion:  "If there are constraints for this area, add them via 'kb quick constrain'",
		})
	}

	// Check for missing decisions (important for consistency)
	if stats.DecisionCount == 0 && stats.TotalMatches > 0 {
		analysis.HasGaps = true
		analysis.Gaps = append(analysis.Gaps, Gap{
			Type:        GapTypeNoDecisions,
			Severity:    GapSeverityInfo,
			Description: "No prior decisions found - agent may not know established patterns",
			Suggestion:  "If decisions exist for this area, ensure they're discoverable via 'kb context'",
		})
	}

	// Calculate context quality score
	analysis.ContextQuality = calculateContextQuality(stats)
	analysis.HasGaps = len(analysis.Gaps) > 0

	return analysis
}

// countMatchesByType counts matches in each category.
func countMatchesByType(matches []KBContextMatch) MatchStatistics {
	stats := MatchStatistics{}
	for _, m := range matches {
		switch m.Type {
		case "constraint":
			stats.ConstraintCount++
		case "decision":
			stats.DecisionCount++
		case "investigation":
			stats.InvestigationCount++
		case "guide":
			stats.GuideCount++
		}
		stats.TotalMatches++
	}
	return stats
}

// calculateContextQuality returns a 0-100 score based on match statistics.
// Scoring:
// - Base points for having any matches
// - Bonus points for constraints (most important)
// - Bonus points for decisions
// - Bonus points for investigations
// - Capped at 100
func calculateContextQuality(stats MatchStatistics) int {
	if stats.TotalMatches == 0 {
		return 0
	}

	score := 0

	// Base points: 10 per match, up to 50
	basePoints := stats.TotalMatches * 10
	if basePoints > 50 {
		basePoints = 50
	}
	score += basePoints

	// Constraint bonus: 15 points for having any, +5 per additional up to 25
	if stats.ConstraintCount > 0 {
		constraintBonus := 15 + (stats.ConstraintCount-1)*5
		if constraintBonus > 25 {
			constraintBonus = 25
		}
		score += constraintBonus
	}

	// Decision bonus: 10 points for having any, +3 per additional up to 15
	if stats.DecisionCount > 0 {
		decisionBonus := 10 + (stats.DecisionCount-1)*3
		if decisionBonus > 15 {
			decisionBonus = 15
		}
		score += decisionBonus
	}

	// Investigation bonus: 5 points for having any, +2 per additional up to 10
	if stats.InvestigationCount > 0 {
		investigationBonus := 5 + (stats.InvestigationCount-1)*2
		if investigationBonus > 10 {
			investigationBonus = 10
		}
		score += investigationBonus
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// FormatGapWarning formats gap analysis as a warning message for display.
// Returns empty string if no significant gaps.
func (g *GapAnalysis) FormatGapWarning() string {
	if !g.HasGaps {
		return ""
	}

	var sb strings.Builder

	// Count gaps by severity
	criticalCount := 0
	warningCount := 0
	for _, gap := range g.Gaps {
		switch gap.Severity {
		case GapSeverityCritical:
			criticalCount++
		case GapSeverityWarning:
			warningCount++
		}
	}

	// Header based on severity
	if criticalCount > 0 {
		sb.WriteString("🚨 CONTEXT GAP DETECTED:\n")
	} else if warningCount > 0 {
		sb.WriteString("⚠️  Context coverage warning:\n")
	} else {
		sb.WriteString("ℹ️  Context notes:\n")
	}

	// Quality indicator
	sb.WriteString(fmt.Sprintf("   Context quality: %d/100", g.ContextQuality))
	if g.ContextQuality < 30 {
		sb.WriteString(" (poor)")
	} else if g.ContextQuality < 60 {
		sb.WriteString(" (limited)")
	} else if g.ContextQuality < 80 {
		sb.WriteString(" (moderate)")
	} else {
		sb.WriteString(" (good)")
	}
	sb.WriteString("\n")

	// Match breakdown if any matches
	if g.MatchStats.TotalMatches > 0 {
		sb.WriteString(fmt.Sprintf("   Matches: %d total (constraints: %d, decisions: %d, investigations: %d)\n",
			g.MatchStats.TotalMatches,
			g.MatchStats.ConstraintCount,
			g.MatchStats.DecisionCount,
			g.MatchStats.InvestigationCount))
	}

	// List critical and warning gaps
	for _, gap := range g.Gaps {
		if gap.Severity == GapSeverityCritical || gap.Severity == GapSeverityWarning {
			icon := "⚠️"
			if gap.Severity == GapSeverityCritical {
				icon = "🚨"
			}
			sb.WriteString(fmt.Sprintf("   %s %s\n", icon, gap.Description))
			sb.WriteString(fmt.Sprintf("      → %s\n", gap.Suggestion))
		}
	}

	return sb.String()
}

// FormatGapSummary returns a brief one-line summary suitable for spawn context header.
func (g *GapAnalysis) FormatGapSummary() string {
	if !g.HasGaps {
		return ""
	}

	if g.ContextQuality == 0 {
		return "⚠️ No prior knowledge found - agent starting without historical context"
	}

	if g.ContextQuality < 30 {
		return fmt.Sprintf("⚠️ Limited context (%d/100) - agent may need to discover patterns during work", g.ContextQuality)
	}

	return ""
}

// ShouldWarnAboutGaps returns true if gaps are significant enough to warrant warning.
func (g *GapAnalysis) ShouldWarnAboutGaps() bool {
	if !g.HasGaps {
		return false
	}

	// Always warn for critical gaps
	for _, gap := range g.Gaps {
		if gap.Severity == GapSeverityCritical {
			return true
		}
	}

	// Warn for warning-level gaps only if context quality is low
	if g.ContextQuality < 30 {
		for _, gap := range g.Gaps {
			if gap.Severity == GapSeverityWarning {
				return true
			}
		}
	}

	return false
}

// Gap gating thresholds.
const (
	// DefaultGateThreshold is the default context quality score below which spawn is blocked.
	// A quality score of 20 indicates very sparse or no context.
	DefaultGateThreshold = 20
)

// ShouldBlockSpawn returns true if the context quality is too low and spawn should be blocked.
// Threshold can be customized; default is DefaultGateThreshold.
func (g *GapAnalysis) ShouldBlockSpawn(threshold int) bool {
	if threshold <= 0 {
		threshold = DefaultGateThreshold
	}
	return g.ContextQuality < threshold
}

// HasCriticalGaps returns true if there are any critical-severity gaps.
func (g *GapAnalysis) HasCriticalGaps() bool {
	for _, gap := range g.Gaps {
		if gap.Severity == GapSeverityCritical {
			return true
		}
	}
	return false
}

// FormatGateBlockMessage returns a message explaining why spawn was blocked due to gaps.
func (g *GapAnalysis) FormatGateBlockMessage() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔══════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║  🛑  SPAWN BLOCKED - CONTEXT GAP DETECTED                                    ║\n")
	sb.WriteString("╠══════════════════════════════════════════════════════════════════════════════╣\n")
	sb.WriteString(fmt.Sprintf("║  Context quality: %d/100 (below threshold)                                   ║\n", g.ContextQuality))
	sb.WriteString("║                                                                              ║\n")

	// List critical gaps
	for _, gap := range g.Gaps {
		if gap.Severity == GapSeverityCritical {
			sb.WriteString(fmt.Sprintf("║  🚨 %s\n", truncateWithPadding(gap.Description, 72)))
		}
	}

	sb.WriteString("║                                                                              ║\n")
	sb.WriteString("║  To fix:                                                                     ║\n")
	sb.WriteString("║    1. Add relevant knowledge:  kb quick decide / kb quick constrain           ║\n")
	sb.WriteString("║    2. Or use --skip-gap-gate to proceed anyway (documents bypass)            ║\n")
	sb.WriteString("║                                                                              ║\n")
	sb.WriteString("║  Why gate? Agents without context compensate by guessing, creating          ║\n")
	sb.WriteString("║  inconsistency. Better to add knowledge first.                               ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════════════════════╝\n")

	return sb.String()
}

// truncateWithPadding truncates a string and pads to ensure consistent formatting.
func truncateWithPadding(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s + strings.Repeat(" ", maxLen-len(s))
}

// FormatProminentWarning formats gap analysis as a highly visible warning box.
// This is more prominent than FormatGapWarning and designed to be hard to ignore.
func (g *GapAnalysis) FormatProminentWarning() string {
	if !g.HasGaps {
		return ""
	}

	var sb strings.Builder

	// Determine severity level for header
	header := "⚠️  CONTEXT GAP WARNING"
	if g.HasCriticalGaps() {
		header = "🚨 CRITICAL CONTEXT GAP"
	}

	sb.WriteString("\n")
	sb.WriteString("┌──────────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString(fmt.Sprintf("│  %s                                                           │\n", header))
	sb.WriteString("├──────────────────────────────────────────────────────────────────────────────┤\n")

	// Quality score with visual bar
	qualityBar := g.formatQualityBar()
	sb.WriteString(fmt.Sprintf("│  Context quality: [%s] %d/100                              │\n", qualityBar, g.ContextQuality))

	// Match breakdown
	if g.MatchStats.TotalMatches > 0 {
		sb.WriteString(fmt.Sprintf("│  Found: %d matches (constraints: %d, decisions: %d, investigations: %d)     │\n",
			g.MatchStats.TotalMatches,
			g.MatchStats.ConstraintCount,
			g.MatchStats.DecisionCount,
			g.MatchStats.InvestigationCount))
	} else {
		sb.WriteString("│  Found: 0 matches - no prior knowledge                                       │\n")
	}

	sb.WriteString("├──────────────────────────────────────────────────────────────────────────────┤\n")

	// List gaps with severity indicators
	for _, gap := range g.Gaps {
		icon := "○"
		if gap.Severity == GapSeverityCritical {
			icon = "●"
		} else if gap.Severity == GapSeverityWarning {
			icon = "◐"
		}
		sb.WriteString(fmt.Sprintf("│  %s %s\n", icon, truncateWithPadding(gap.Description, 72)))
	}

	sb.WriteString("├──────────────────────────────────────────────────────────────────────────────┤\n")
	sb.WriteString("│  Agent may need to compensate by discovering patterns during work.          │\n")
	sb.WriteString("│  Consider adding knowledge first: kb quick decide / kb quick constrain       │\n")
	sb.WriteString("└──────────────────────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}

// formatQualityBar returns a visual bar representation of context quality.
func (g *GapAnalysis) formatQualityBar() string {
	// 10-char bar showing quality level
	filled := g.ContextQuality / 10
	if filled > 10 {
		filled = 10
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
}

// GapAPIResponse represents gap analysis data for API responses.
type GapAPIResponse struct {
	HasGaps        bool           `json:"has_gaps"`
	ContextQuality int            `json:"context_quality"`
	Query          string         `json:"query,omitempty"`
	Gaps           []GapAPIDetail `json:"gaps,omitempty"`
	MatchStats     MatchStatsAPI  `json:"match_stats,omitempty"`
	ShouldWarn     bool           `json:"should_warn"`
	ShouldBlock    bool           `json:"should_block"`
}

// GapAPIDetail represents a single gap for API responses.
type GapAPIDetail struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// MatchStatsAPI represents match statistics for API responses.
type MatchStatsAPI struct {
	TotalMatches       int `json:"total"`
	ConstraintCount    int `json:"constraints"`
	DecisionCount      int `json:"decisions"`
	InvestigationCount int `json:"investigations"`
}

// ToAPIResponse converts GapAnalysis to a JSON-serializable API response.
func (g *GapAnalysis) ToAPIResponse() GapAPIResponse {
	resp := GapAPIResponse{
		HasGaps:        g.HasGaps,
		ContextQuality: g.ContextQuality,
		Query:          g.Query,
		ShouldWarn:     g.ShouldWarnAboutGaps(),
		ShouldBlock:    g.ShouldBlockSpawn(DefaultGateThreshold),
	}

	resp.MatchStats = MatchStatsAPI{
		TotalMatches:       g.MatchStats.TotalMatches,
		ConstraintCount:    g.MatchStats.ConstraintCount,
		DecisionCount:      g.MatchStats.DecisionCount,
		InvestigationCount: g.MatchStats.InvestigationCount,
	}

	for _, gap := range g.Gaps {
		resp.Gaps = append(resp.Gaps, GapAPIDetail{
			Type:        string(gap.Type),
			Severity:    string(gap.Severity),
			Description: gap.Description,
			Suggestion:  gap.Suggestion,
		})
	}

	return resp
}
