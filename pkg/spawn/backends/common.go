package backends

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// FormatSessionTitle formats the session title to include beads ID for matching.
// Format: "workspace-name [beads-id]" (e.g., "og-debug-orch-status-23dec [orch-go-v4mw]")
// This allows extractBeadsIDFromTitle to find agents in orch status.
func FormatSessionTitle(workspaceName, beadsID string) string {
	if beadsID == "" {
		return workspaceName
	}
	return fmt.Sprintf("%s [%s]", workspaceName, beadsID)
}

// AddGapAnalysisToEventData adds gap analysis information to an event data map.
// This enables tracking of context gaps for pattern analysis and dashboard surfacing.
func AddGapAnalysisToEventData(eventData map[string]interface{}, gapAnalysis *spawn.GapAnalysis) {
	if gapAnalysis == nil {
		return
	}

	eventData["gap_has_gaps"] = gapAnalysis.HasGaps
	eventData["gap_context_quality"] = gapAnalysis.ContextQuality

	if gapAnalysis.HasGaps {
		eventData["gap_should_warn"] = gapAnalysis.ShouldWarnAboutGaps()
		eventData["gap_match_total"] = gapAnalysis.MatchStats.TotalMatches
		eventData["gap_match_constraints"] = gapAnalysis.MatchStats.ConstraintCount
		eventData["gap_match_decisions"] = gapAnalysis.MatchStats.DecisionCount
		eventData["gap_match_investigations"] = gapAnalysis.MatchStats.InvestigationCount

		// Capture gap types for pattern analysis
		var gapTypes []string
		for _, gap := range gapAnalysis.Gaps {
			gapTypes = append(gapTypes, string(gap.Type))
		}
		if len(gapTypes) > 0 {
			eventData["gap_types"] = gapTypes
		}
	}
}

// AddUsageInfoToEventData adds usage information to an event data map.
// This enables tracking of rate limit patterns and account utilization at spawn time.
func AddUsageInfoToEventData(eventData map[string]interface{}, usageInfo *spawn.UsageInfo) {
	if usageInfo == nil {
		return
	}

	eventData["usage_5h_used"] = usageInfo.FiveHourUsed
	eventData["usage_weekly_used"] = usageInfo.SevenDayUsed
	if usageInfo.AccountEmail != "" {
		eventData["usage_account"] = usageInfo.AccountEmail
	}
	if usageInfo.AutoSwitched {
		eventData["usage_auto_switched"] = true
		eventData["usage_switch_reason"] = usageInfo.SwitchReason
	}
}

// FormatContextQualitySummary formats context quality for spawn summary output.
// Returns a formatted string with visual indicators for gap severity.
// This is the "prominent" surfacing that makes gaps hard to ignore.
func FormatContextQualitySummary(gapAnalysis *spawn.GapAnalysis) string {
	if gapAnalysis == nil {
		return "not checked"
	}

	quality := gapAnalysis.ContextQuality

	// Determine visual indicator and label based on quality level
	var indicator, label string
	switch {
	case quality == 0:
		indicator = "!"
		label = "CRITICAL - No context"
	case quality < 20:
		indicator = "!"
		label = "poor"
	case quality < 40:
		indicator = "!"
		label = "limited"
	case quality < 60:
		indicator = "-"
		label = "moderate"
	case quality < 80:
		indicator = "+"
		label = "good"
	default:
		indicator = "+"
		label = "excellent"
	}

	// Format the summary line
	summary := fmt.Sprintf("%s %d/100 (%s)", indicator, quality, label)

	// Add match breakdown for transparency
	if gapAnalysis.MatchStats.TotalMatches > 0 {
		summary += fmt.Sprintf(" - %d matches", gapAnalysis.MatchStats.TotalMatches)
		if gapAnalysis.MatchStats.ConstraintCount > 0 {
			summary += fmt.Sprintf(" (%d constraints)", gapAnalysis.MatchStats.ConstraintCount)
		}
	}

	return summary
}

// PrintSpawnSummaryWithGapWarning prints the spawn summary with prominent gap warnings.
// This ensures gaps are visible in the final output, not just during context gathering.
func PrintSpawnSummaryWithGapWarning(gapAnalysis *spawn.GapAnalysis) {
	if gapAnalysis == nil || !gapAnalysis.ShouldWarnAboutGaps() {
		return
	}

	// Print a prominent warning box for critical gaps
	if gapAnalysis.HasCriticalGaps() || gapAnalysis.ContextQuality < 20 {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "+-------------------------------------------------------------+\n")
		fmt.Fprintf(os.Stderr, "|  GAP WARNING: Agent spawned with limited context           |\n")
		fmt.Fprintf(os.Stderr, "+-------------------------------------------------------------+\n")
		fmt.Fprintf(os.Stderr, "|  Agent may compensate by guessing patterns/conventions.    |\n")
		fmt.Fprintf(os.Stderr, "|  Consider: kn decide / kn constrain / kb create            |\n")
		fmt.Fprintf(os.Stderr, "+-------------------------------------------------------------+\n")
	}
}

// LogSpawnEvent logs the session.spawned event with common metadata.
func LogSpawnEvent(sessionID string, req *SpawnRequest, mode string, extraData map[string]interface{}) error {
	logger := events.NewLogger(events.DefaultLogPath())

	eventData := map[string]interface{}{
		"skill":               req.SkillName,
		"task":                req.Task,
		"workspace":           req.Config.WorkspaceName,
		"beads_id":            req.BeadsID,
		"spawn_mode":          mode,
		"no_track":            req.Config.NoTrack,
		"skip_artifact_check": req.Config.SkipArtifactCheck,
	}

	// Add session_id for non-inline backends
	if sessionID != "" {
		eventData["session_id"] = sessionID
	}

	// Add model if present
	if req.Config.Model != "" {
		eventData["model"] = req.Config.Model
	}

	// Add MCP if present
	if req.Config.MCP != "" {
		eventData["mcp"] = req.Config.MCP
	}

	// Add extra data (e.g., retry_attempts, window info for tmux)
	for k, v := range extraData {
		eventData[k] = v
	}

	// Add gap analysis data
	AddGapAnalysisToEventData(eventData, req.Config.GapAnalysis)

	// Add usage info data
	AddUsageInfoToEventData(eventData, req.Config.UsageInfo)

	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}

	return logger.Log(event)
}

// PrintSpawnSummary prints the common spawn summary output.
func PrintSpawnSummary(result *Result, req *SpawnRequest) {
	// Print spawn summary with prominent gap warning if needed
	PrintSpawnSummaryWithGapWarning(req.Config.GapAnalysis)

	switch result.SpawnMode {
	case "inline":
		fmt.Printf("Spawned agent:\n")
		fmt.Printf("  Session ID: %s\n", result.SessionID)
		fmt.Printf("  Workspace:  %s\n", req.Config.WorkspaceName)
		fmt.Printf("  Beads ID:   %s\n", req.BeadsID)
		fmt.Printf("  Context:    %s\n", FormatContextQualitySummary(req.Config.GapAnalysis))

	case "headless":
		fmt.Printf("Spawned agent (headless):\n")
		fmt.Printf("  Session ID: %s\n", result.SessionID)
		fmt.Printf("  Workspace:  %s\n", req.Config.WorkspaceName)
		fmt.Printf("  Beads ID:   %s\n", req.BeadsID)
		fmt.Printf("  Model:      %s\n", req.Config.Model)
		printAccountProvenance(req)
		if req.Config.MCP != "" {
			fmt.Printf("  MCP:        %s\n", req.Config.MCP)
		}
		if req.Config.NoTrack {
			fmt.Printf("  Tracking:   disabled (--no-track)\n")
		}
		fmt.Printf("  Context:    %s\n", FormatContextQualitySummary(req.Config.GapAnalysis))

	case "tmux":
		if result.TmuxInfo != nil {
			fmt.Printf("Spawned agent in tmux:\n")
			fmt.Printf("  Session:    %s\n", result.TmuxInfo.SessionName)
			if result.SessionID != "" {
				fmt.Printf("  Session ID: %s\n", result.SessionID)
			}
			fmt.Printf("  Window:     %s\n", result.TmuxInfo.WindowTarget)
			fmt.Printf("  Window ID:  %s\n", result.TmuxInfo.WindowID)
			fmt.Printf("  Workspace:  %s\n", req.Config.WorkspaceName)
			fmt.Printf("  Beads ID:   %s\n", req.BeadsID)
			fmt.Printf("  Model:      %s\n", req.Config.Model)
			printAccountProvenance(req)
			if req.Config.MCP != "" {
				fmt.Printf("  MCP:        %s\n", req.Config.MCP)
			}
			if req.Config.NoTrack {
				fmt.Printf("  Tracking:   disabled (--no-track)\n")
			}
			fmt.Printf("  Context:    %s\n", FormatContextQualitySummary(req.Config.GapAnalysis))
		}
	}
}

// printAccountProvenance prints the account selection provenance line.
// Only prints when an account is set (Claude backend with account routing).
func printAccountProvenance(req *SpawnRequest) {
	if req.Config.Account == "" {
		return
	}
	acctSetting := req.Config.ResolvedSettings.Account
	provenance := fmt.Sprintf("source: %s", acctSetting.Source)
	if acctSetting.Detail != "" {
		provenance += fmt.Sprintf(", detail: %s", acctSetting.Detail)
	}
	fmt.Printf("  Account:    %s (%s)\n", req.Config.Account, provenance)
}
