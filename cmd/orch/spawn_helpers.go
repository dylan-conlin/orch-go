// Package main provides helper functions for spawn command registration and design handoff.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// registerOrchestratorSession registers an orchestrator session in the session registry.
// This tracks orchestrator sessions separately from worker agents for cross-project coordination.
func registerOrchestratorSession(cfg *spawn.Config, sessionID, task string) {
	if !cfg.IsOrchestrator && !cfg.IsMetaOrchestrator {
		return // Only register orchestrator sessions
	}

	registry := session.NewRegistry("")
	orchSession := session.OrchestratorSession{
		WorkspaceName: cfg.WorkspaceName,
		SessionID:     sessionID,
		ProjectDir:    cfg.ProjectDir,
		SpawnTime:     time.Now(),
		Goal:          task,
		Status:        "active",
	}
	if err := registry.Register(orchSession); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to register orchestrator session: %v\n", err)
	}
}

// addGapAnalysisToEventData adds gap analysis information to an event data map.
// This enables tracking of context gaps for pattern analysis and dashboard surfacing.
func addGapAnalysisToEventData(eventData map[string]interface{}, gapAnalysis *spawn.GapAnalysis) {
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

// addUsageInfoToEventData adds usage information to an event data map.
// This enables tracking of rate limit patterns and account utilization at spawn time.
func addUsageInfoToEventData(eventData map[string]interface{}, usageInfo *spawn.UsageInfo) {
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

// formatContextQualitySummary formats context quality for spawn summary output.
// Returns a formatted string with visual indicators for gap severity.
// This is the "prominent" surfacing that makes gaps hard to ignore.
func formatContextQualitySummary(gapAnalysis *spawn.GapAnalysis) string {
	if gapAnalysis == nil {
		return "not checked"
	}

	quality := gapAnalysis.ContextQuality

	// Determine visual indicator and label based on quality level
	var indicator, label string
	switch {
	case quality == 0:
		indicator = "🚨"
		label = "CRITICAL - No context"
	case quality < 20:
		indicator = "⚠️"
		label = "poor"
	case quality < 40:
		indicator = "⚠️"
		label = "limited"
	case quality < 60:
		indicator = "📊"
		label = "moderate"
	case quality < 80:
		indicator = "✓"
		label = "good"
	default:
		indicator = "✓"
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

// printSpawnSummaryWithGapWarning prints the spawn summary with prominent gap warnings.
// This ensures gaps are visible in the final output, not just during context gathering.
func printSpawnSummaryWithGapWarning(gapAnalysis *spawn.GapAnalysis) {
	if gapAnalysis == nil || !gapAnalysis.ShouldWarnAboutGaps() {
		return
	}

	// Print a prominent warning box for critical gaps
	if gapAnalysis.HasCriticalGaps() || gapAnalysis.ContextQuality < 20 {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "┌─────────────────────────────────────────────────────────────┐\n")
		fmt.Fprintf(os.Stderr, "│  ⚠️  GAP WARNING: Agent spawned with limited context         │\n")
		fmt.Fprintf(os.Stderr, "├─────────────────────────────────────────────────────────────┤\n")
		fmt.Fprintf(os.Stderr, "│  Agent may compensate by guessing patterns/conventions.    │\n")
		fmt.Fprintf(os.Stderr, "│  Consider: kn decide / kn constrain / kb create            │\n")
		fmt.Fprintf(os.Stderr, "└─────────────────────────────────────────────────────────────┘\n")
	}
}

// registerAgent registers an agent in the registry for tracking and monitoring.
func registerAgent(cfg *spawn.Config, sessionID, tmuxWindow, mode, modelSpec string) {
	agentReg, err := registry.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open agent registry: %v\n", err)
		return
	}

	agent := &registry.Agent{
		ID:         cfg.WorkspaceName,
		BeadsID:    cfg.BeadsID,
		Mode:       mode,
		SessionID:  sessionID,
		TmuxWindow: tmuxWindow,
		Model:      modelSpec,
		ProjectDir: cfg.ProjectDir,
		Skill:      cfg.SkillName,
		Status:     registry.StateActive,
	}

	if err := agentReg.Register(agent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to register agent in registry: %v\n", err)
		return
	}

	if err := agentReg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save agent registry: %v\n", err)
	}
}

// readDesignArtifacts reads design artifacts from a ui-design-session workspace.
// Returns mockup path, prompt path, and design notes from SYNTHESIS.md.
// If the workspace doesn't exist or artifacts are missing, returns empty strings.
func readDesignArtifacts(projectDir, designWorkspace string) (mockupPath, promptPath, designNotes string) {
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", designWorkspace)

	// Check if workspace exists
	if _, err := os.Stat(workspacePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: design workspace not found: %s\n", workspacePath)
		return "", "", ""
	}

	// Look for mockup in screenshots/ directory
	// Convention: approved.png or any .png file
	screenshotsPath := filepath.Join(workspacePath, "screenshots")
	if entries, err := os.ReadDir(screenshotsPath); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".png") {
				mockupPath = filepath.Join(screenshotsPath, entry.Name())
				// Check for corresponding .prompt.md file
				promptName := strings.TrimSuffix(entry.Name(), ".png") + ".prompt.md"
				promptPath = filepath.Join(screenshotsPath, promptName)
				if _, err := os.Stat(promptPath); err != nil {
					promptPath = "" // Prompt file doesn't exist
				}
				break // Use first .png found
			}
		}
	}

	// Read design notes from SYNTHESIS.md
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	if content, err := os.ReadFile(synthesisPath); err == nil {
		// Extract relevant sections from SYNTHESIS.md
		// For now, just include the TLDR and Knowledge sections
		designNotes = extractDesignNotes(string(content))
	}

	return mockupPath, promptPath, designNotes
}

// extractDesignNotes extracts relevant sections from SYNTHESIS.md for design handoff.
// Returns TLDR and Knowledge sections which contain key design insights.
func extractDesignNotes(content string) string {
	var notes strings.Builder

	// Extract TLDR section
	if tldr := extractSection(content, "## TLDR"); tldr != "" {
		notes.WriteString("**Design TLDR:**\n")
		notes.WriteString(tldr)
		notes.WriteString("\n\n")
	}

	// Extract Knowledge section
	if knowledge := extractSection(content, "## Knowledge"); knowledge != "" {
		notes.WriteString("**Design Knowledge:**\n")
		notes.WriteString(knowledge)
	}

	return notes.String()
}

// extractSection extracts content between a section header and the next ## header.
// Returns empty string if section not found.
func extractSection(content, sectionHeader string) string {
	lines := strings.Split(content, "\n")
	var sectionLines []string
	inSection := false

	for _, line := range lines {
		if strings.HasPrefix(line, sectionHeader) {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(line, "##") {
			break // Reached next section
		}
		if inSection {
			sectionLines = append(sectionLines, line)
		}
	}

	if len(sectionLines) == 0 {
		return ""
	}

	return strings.TrimSpace(strings.Join(sectionLines, "\n"))
}
