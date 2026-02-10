// Package main provides display and formatting functions for the status command.
// Extracted from status_cmd.go as part of the status_cmd.go refactoring.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/usage"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"golang.org/x/term"

	"github.com/dylan-conlin/orch-go/pkg/account"
)

// Terminal width thresholds for adaptive output
const (
	termWidthWide   = 120 // Full table with all columns
	termWidthNarrow = 100 // Drop TASK column, abbreviate SKILL
	termWidthMin    = 80  // Minimum supported width (vertical card format)
)

// getTerminalWidth returns the current terminal width, or a default if detection fails.
// Returns the width and whether we're outputting to a real terminal.
func getTerminalWidth() (int, bool) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Not a terminal (piped output) - use wide format
		return termWidthWide + 1, false
	}
	return width, true
}

// printSwarmStatus prints the swarm status in human-readable format.
// Adapts output format based on terminal width.
func printSwarmStatus(output StatusOutput, showAll bool) {
	width, _ := getTerminalWidth()
	printSwarmStatusWithWidth(output, showAll, width)
}

// printSwarmStatusWithWidth prints swarm status with explicit width (for testing).
func printSwarmStatusWithWidth(output StatusOutput, showAll bool, termWidth int) {
	// Check for dev mode and warn
	if devModeInfo, err := readDevModeFile(".dev-mode"); err == nil {
		duration := time.Since(devModeInfo.Started).Round(time.Minute)
		fmt.Printf("WARNING: DEV MODE ACTIVE (%s): %s\n", duration, devModeInfo.Reason)
		fmt.Println("   Infrastructure is unprotected. Run 'orch mode ops' when done.")
		fmt.Println()
	}

	// Print infrastructure health section first
	printInfrastructureHealth(output.Infrastructure)

	// Print swarm summary header with processing breakdown
	fmt.Printf("SWARM STATUS: Active: %d", output.Swarm.Active)
	if output.Swarm.Active > 0 {
		fmt.Printf(" (running: %d, idle: %d)", output.Swarm.Processing, output.Swarm.Idle)
	}
	if output.Swarm.Completed > 0 {
		fmt.Printf(", Completed: %d", output.Swarm.Completed)
		if !showAll {
			fmt.Printf(" (use --all to show)")
		}
	}
	if output.Swarm.Phantom > 0 {
		fmt.Printf(", Phantom: %d", output.Swarm.Phantom)
		if !showAll {
			fmt.Printf(" (use --all to show)")
		}
	}
	if output.Swarm.Untracked > 0 {
		fmt.Printf(", Untracked: %d", output.Swarm.Untracked)
		// Untracked sessions are now shown by default (no --all needed)
	}
	fmt.Println()
	// In compact mode, add hint about hidden idle agents
	if !showAll && output.Swarm.Idle > 0 && output.Swarm.Idle > len(output.Agents) {
		hiddenIdle := output.Swarm.Idle - countIdleInList(output.Agents)
		if hiddenIdle > 0 {
			fmt.Printf("  (compact mode: %d idle agents hidden, use --all for full list)\n", hiddenIdle)
		}
	}

	// Print drift metrics if any issues detected
	if output.DriftMetrics != nil {
		driftIssues := output.DriftMetrics.MissingSessionID +
			output.DriftMetrics.MissingTmuxWindow +
			output.DriftMetrics.StaleActive
		if driftIssues > 0 {
			fmt.Printf("  DRIFT: ")
			parts := make([]string, 0, 3)
			if output.DriftMetrics.MissingSessionID > 0 {
				parts = append(parts, fmt.Sprintf("%d missing session_id", output.DriftMetrics.MissingSessionID))
			}
			if output.DriftMetrics.MissingTmuxWindow > 0 {
				parts = append(parts, fmt.Sprintf("%d missing tmux_window", output.DriftMetrics.MissingTmuxWindow))
			}
			if output.DriftMetrics.StaleActive > 0 {
				parts = append(parts, fmt.Sprintf("%d stale active (>2h no update)", output.DriftMetrics.StaleActive))
			}
			fmt.Println(strings.Join(parts, ", "))
		}
	}
	fmt.Println()

	// Print account usage
	if len(output.Accounts) > 0 {
		fmt.Println("ACCOUNTS")
		for _, acc := range output.Accounts {
			activeMarker := ""
			if acc.IsActive {
				activeMarker = " *"
			}
			usageStr := "N/A"
			if acc.UsedPercent > 0 || acc.IsActive {
				usageStr = fmt.Sprintf("%.0f%% used", acc.UsedPercent)
				if acc.ResetTime != "" {
					usageStr += fmt.Sprintf(" (resets in %s)", acc.ResetTime)
				}
			}
			name := acc.Name
			if acc.Email != "" && acc.Name == "current" {
				name = acc.Email
			}
			fmt.Printf("  %-20s %s%s\n", name+":", usageStr, activeMarker)
		}
		fmt.Println()
	}

	// Print orchestrator sessions
	if len(output.OrchestratorSessions) > 0 {
		printOrchestratorSessions(output.OrchestratorSessions, termWidth)
		fmt.Println()
	}

	// Print agents in format appropriate for terminal width
	if len(output.Agents) > 0 {
		fmt.Println("AGENTS")
		if termWidth < termWidthMin {
			printAgentsCardFormat(output.Agents)
		} else if termWidth < termWidthNarrow {
			printAgentsNarrowFormat(output.Agents)
		} else {
			printAgentsWideFormat(output.Agents)
		}
	} else {
		fmt.Println("No active agents")
	}

	// Print synthesis opportunities (if any)
	if output.SynthesisOpportunities != nil && output.SynthesisOpportunities.HasOpportunities() {
		fmt.Println()
		printSynthesisOpportunities(output.SynthesisOpportunities)
	}
}

// printOrchestratorSessions prints orchestrator sessions in a table format.
func printOrchestratorSessions(sessions []OrchestratorSessionInfo, termWidth int) {
	fmt.Println("ORCHESTRATOR SESSIONS")

	if termWidth < termWidthMin {
		// Card format for very narrow terminals
		for i, s := range sessions {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("  %s [%s]\n", s.WorkspaceName, s.Status)
			fmt.Printf("    Goal: %s\n", truncate(s.Goal, 50))
			fmt.Printf("    Duration: %s | Project: %s\n", s.Duration, s.Project)
		}
	} else if termWidth < termWidthNarrow {
		// Narrow format - drop goal column
		fmt.Printf("  %-40s %-10s %s\n", "WORKSPACE", "DURATION", "PROJECT")
		fmt.Printf("  %s\n", strings.Repeat("-", 65))
		for _, s := range sessions {
			project := s.Project
			if project == "" {
				project = "-"
			}
			fmt.Printf("  %-40s %-10s %s\n",
				truncate(s.WorkspaceName, 38),
				s.Duration,
				project)
		}
	} else {
		// Wide format - full table
		fmt.Printf("  %-40s %-30s %-10s %s\n", "WORKSPACE", "GOAL", "DURATION", "PROJECT")
		fmt.Printf("  %s\n", strings.Repeat("-", 95))
		for _, s := range sessions {
			project := s.Project
			if project == "" {
				project = "-"
			}
			fmt.Printf("  %-40s %-30s %-10s %s\n",
				truncate(s.WorkspaceName, 38),
				truncate(s.Goal, 28),
				s.Duration,
				project)
		}
	}
}

// printAgentsWideFormat prints agents in full table format (>120 chars).
// Columns: SOURCE, BEADS ID, STATUS, PHASE, TASK, SKILL, RUNTIME, TOKENS, RISK
func printAgentsWideFormat(agents []AgentInfo) {
	// Check if any agent has risk to show RISK column
	hasRisk := false
	for _, agent := range agents {
		if agent.ContextRisk != nil && agent.ContextRisk.IsAtRisk() {
			hasRisk = true
			break
		}
	}

	if hasRisk {
		fmt.Printf("  %-3s %-18s %-8s %-8s %-12s %-20s %-25s %-12s %-7s %-16s %s\n", "SRC", "BEADS ID", "MODE", "MODEL", "STATUS", "PHASE", "TASK", "SKILL", "RUNTIME", "TOKENS", "RISK")
		fmt.Printf("  %s\n", strings.Repeat("-", 150))
	} else {
		fmt.Printf("  %-3s %-18s %-8s %-20s %-8s %-12s %-23s %-12s %-8s %s\n", "SRC", "BEADS ID", "MODE", "MODEL", "STATUS", "PHASE", "TASK", "SKILL", "RUNTIME", "TOKENS")
		fmt.Printf("  %s\n", strings.Repeat("-", 140))
	}

	for _, agent := range agents {
		source := agent.Source
		if source == "" {
			source = "-"
		}
		beadsID := formatBeadsIDForDisplay(agent.BeadsID)
		if beadsID == "" {
			// For untracked sessions, show truncated session ID to enable `orch tail --session`
			if agent.IsUntracked && agent.SessionID != "" {
				beadsID = truncateSessionIDForStatus(agent.SessionID)
			} else {
				beadsID = "-"
			}
		}
		mode := agent.Mode
		if mode == "" {
			mode = "-"
		}
		modelDisplay := formatModelForDisplay(agent.Model)
		phase := agent.Phase
		if phase == "" {
			phase = "-"
		}
		task := agent.Task
		if task == "" {
			task = "-"
		}
		skill := agent.Skill
		if skill == "" {
			skill = "-"
		}
		status := getAgentStatus(agent)
		tokens := formatTokenStatsCompact(agent.Tokens)

		if hasRisk {
			risk := formatContextRisk(agent.ContextRisk)
			fmt.Printf("  %-3s %-18s %-8s %-8s %-12s %-20s %-25s %-12s %-7s %-16s %s\n",
				source,
				beadsID,
				mode,
				modelDisplay,
				status,
				truncate(phase, 10),
				truncate(task, 23),
				truncate(skill, 10),
				agent.Runtime,
				tokens,
				risk)
		} else {
			fmt.Printf("  %-3s %-18s %-8s %-20s %-8s %-12s %-23s %-12s %-8s %s\n",
				source,
				beadsID,
				mode,
				modelDisplay,
				status,
				truncate(phase, 10),
				truncate(task, 21),
				truncate(skill, 10),
				agent.Runtime,
				tokens)
		}
	}
}

// formatContextRisk returns a formatted string for context exhaustion risk.
func formatContextRisk(risk *verify.ContextExhaustionRisk) string {
	if risk == nil || !risk.IsAtRisk() {
		return ""
	}
	emoji := risk.FormatRiskEmoji()
	status := risk.FormatRiskStatus()
	if emoji != "" {
		return emoji + " " + status
	}
	return status
}

// printAgentsNarrowFormat prints agents in narrow format (80-100 chars).
// Drops TASK column, abbreviates SKILL and MODEL.
// Columns: SOURCE, BEADS ID, MODE, MODEL, STATUS, PHASE, SKILL, RUNTIME, TOKENS
func printAgentsNarrowFormat(agents []AgentInfo) {
	fmt.Printf("  %-3s %-18s %-8s %-8s %-8s %-10s %-8s %-8s %s\n", "SRC", "BEADS ID", "MODE", "MODEL", "STATUS", "PHASE", "SKILL", "RUNTIME", "TOKENS")
	fmt.Printf("  %s\n", strings.Repeat("-", 98))

	for _, agent := range agents {
		source := agent.Source
		if source == "" {
			source = "-"
		}
		beadsID := formatBeadsIDForDisplay(agent.BeadsID)
		if beadsID == "" {
			// For untracked sessions, show truncated session ID
			if agent.IsUntracked && agent.SessionID != "" {
				beadsID = truncateSessionIDForStatus(agent.SessionID)
			} else {
				beadsID = "-"
			}
		}
		mode := agent.Mode
		if mode == "" {
			mode = "-"
		}
		modelDisplay := formatModelForDisplay(agent.Model)
		phase := agent.Phase
		if phase == "" {
			phase = "-"
		}
		skill := abbreviateSkill(agent.Skill)
		if skill == "" {
			skill = "-"
		}
		status := getAgentStatus(agent)
		tokens := formatTokenStatsCompact(agent.Tokens)

		fmt.Printf("  %-3s %-18s %-8s %-8s %-8s %-10s %-8s %-8s %s\n",
			source,
			beadsID,
			truncate(mode, 7),
			truncate(modelDisplay, 7),
			status,
			truncate(phase, 9),
			truncate(skill, 7),
			agent.Runtime,
			tokens)
	}
}

// printAgentsCardFormat prints agents in vertical card format (<80 chars).
// Each agent is a multi-line block for readability on very narrow terminals.
func printAgentsCardFormat(agents []AgentInfo) {
	for i, agent := range agents {
		if i > 0 {
			fmt.Println()
		}
		source := agent.Source
		if source == "" {
			source = "-"
		}
		beadsID := formatBeadsIDForDisplay(agent.BeadsID)
		if beadsID == "" {
			// For untracked sessions, show truncated session ID
			if agent.IsUntracked && agent.SessionID != "" {
				beadsID = truncateSessionIDForStatus(agent.SessionID)
			} else {
				beadsID = "-"
			}
		}
		modelDisplay := formatModelForDisplay(agent.Model)
		phase := agent.Phase
		if phase == "" {
			phase = "-"
		}
		task := agent.Task
		if task == "" {
			task = "-"
		}
		skill := agent.Skill
		if skill == "" {
			skill = "-"
		}
		status := getAgentStatus(agent)
		riskStr := formatContextRisk(agent.ContextRisk)

		if riskStr != "" {
			fmt.Printf("  [%s] %s [%s] %s\n", source, beadsID, status, riskStr)
		} else {
			fmt.Printf("  [%s] %s [%s]\n", source, beadsID, status)
		}
		fmt.Printf("    Model: %s | Phase: %s | Skill: %s\n", modelDisplay, phase, skill)
		fmt.Printf("    Task: %s\n", truncate(task, 50))
		fmt.Printf("    Runtime: %s | Tokens: %s\n", agent.Runtime, formatTokenStats(agent.Tokens))
		if agent.ContextRisk != nil && agent.ContextRisk.Reason != "" {
			fmt.Printf("    Risk: %s\n", agent.ContextRisk.Reason)
		}
	}
}

// countIdleInList counts the number of idle agents in a list.
// Used to calculate how many idle agents are hidden in compact mode.
func countIdleInList(agents []AgentInfo) int {
	count := 0
	for _, agent := range agents {
		if !agent.IsProcessing && !agent.IsPhantom && !agent.IsCompleted {
			count++
		}
	}
	return count
}

// getAgentStatus returns a status string based on agent state.
func getAgentStatus(agent AgentInfo) string {
	if agent.IsCompleted {
		return "completed"
	}
	if agent.IsPhantom {
		return "phantom"
	}
	if agent.IsUntracked {
		if agent.IsProcessing {
			return "untracked*" // Running untracked session
		}
		return "untracked"
	}
	if agent.IsProcessing {
		return "running"
	}
	if isIdleWithWork(agent) {
		return "idle ⚠"
	}
	return "idle"
}

// getAccountUsage fetches usage info for all configured accounts.
func getAccountUsage() []AccountUsage {
	var accounts []AccountUsage

	// Get current account usage
	currentUsage := usage.FetchUsage()
	if currentUsage.Error == "" && currentUsage.SevenDay != nil {
		current := AccountUsage{
			Name:        "current",
			Email:       currentUsage.Email,
			UsedPercent: currentUsage.SevenDay.Utilization,
			IsActive:    true,
		}
		if currentUsage.SevenDay.ResetsAt != nil {
			current.ResetTime = currentUsage.SevenDay.TimeUntilReset()
		}
		accounts = append(accounts, current)
	}

	// Try to get saved accounts info (without switching)
	cfg, err := account.LoadConfig()
	if err == nil {
		for name, acc := range cfg.Accounts {
			if acc.Source == "saved" {
				// Check if this is the current account (by email match)
				isCurrentAccount := false
				for i := range accounts {
					if accounts[i].Email == acc.Email {
						accounts[i].Name = name // Update name to the saved account name
						isCurrentAccount = true
						break
					}
				}
				if !isCurrentAccount {
					// Add as a saved account (no live usage data without switching)
					accounts = append(accounts, AccountUsage{
						Name:     name,
						Email:    acc.Email,
						IsActive: false,
					})
				}
			}
		}
	}

	return accounts
}

// printSynthesisOpportunities prints the synthesis opportunities section.
// Only shown when there are opportunities (3+ investigations on a topic without synthesis).
func printSynthesisOpportunities(opps *verify.SynthesisOpportunities) {
	fmt.Println("SYNTHESIS OPPORTUNITIES")
	for _, opp := range opps.Opportunities {
		fmt.Printf("  %d investigations on '%s' without synthesis\n", opp.InvestigationCount, opp.Topic)
	}
}

// abbreviateSkill returns a shortened version of skill names for narrow displays.
func abbreviateSkill(skill string) string {
	abbreviations := map[string]string{
		"feature-impl":         "feat",
		"investigation":        "inv",
		"systematic-debugging": "debug",
		"architect":            "arch",
		"codebase-audit":       "audit",
		"reliability-testing":  "rel-test",
		"issue-creation":       "issue",
		"design-session":       "design",
		"research":             "research",
	}
	if abbr, ok := abbreviations[skill]; ok {
		return abbr
	}
	return skill
}

// truncateSessionIDForStatus shortens a session ID for status display (ses_xxx... format).
// Shows first 16 chars to enable copy-paste for `orch tail --session`.
func truncateSessionIDForStatus(id string) string {
	if len(id) <= 16 {
		return id
	}
	return id[:16] + "..."
}

// formatModelForDisplay formats a model spec for compact display.
// Shortens common model names (e.g., "gemini-3-flash-preview" -> "flash3", "claude-opus-4-6" -> "opus-4.6")
func formatModelForDisplay(model string) string {
	if model == "" {
		return "-"
	}

	// Map full model IDs to short display names
	// Includes both bare and prefixed versions (e.g., "anthropic/claude-...")
	modelAbbreviations := map[string]string{
		"gemini-3-flash-preview":               "flash3",
		"gemini-2.5-flash":                     "flash-2.5",
		"gemini-2.5-pro":                       "pro-2.5",
		"claude-opus-4-6":                      "opus-4.6",
		"anthropic/claude-opus-4-6":            "opus-4.6",
		"claude-opus-4-5-20251101":             "opus-4.5", // Legacy
		"claude-sonnet-4-5-20250929":           "sonnet-4.5",
		"claude-haiku-4-5-20251001":            "haiku-4.5",
		"anthropic/claude-opus-4-5-20251101":   "opus-4.5", // Legacy
		"anthropic/claude-sonnet-4-5-20250929": "sonnet-4.5",
		"anthropic/claude-haiku-4-5-20251001":  "haiku-4.5",
		"gpt-5":                                "gpt5",
		"gpt-5.2":                              "gpt5.2",
		"gpt-5.2-codex":                        "gpt5.2-codex",
		"gpt-5.3-codex":                        "gpt5.3-codex",
		"gpt-5-mini":                           "gpt5-mini",
		"o3":                                   "o3",
		"o3-mini":                              "o3-mini",
		"deepseek-chat":                        "deepseek",
		"deepseek-reasoner":                    "deepseek-r1",
	}

	if abbr, ok := modelAbbreviations[model]; ok {
		return abbr
	}

	// For unknown models, truncate to 18 chars
	return truncate(model, 18)
}

// formatTokenCount formats a token count with K/M suffixes for readability.
func formatTokenCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}
	if count < 1000000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(count)/1000000)
}

// formatTokenStats returns a formatted string of token usage.
func formatTokenStats(tokens *opencode.TokenStats) string {
	if tokens == nil {
		return "-"
	}
	// Format: "in:X out:Y (cache:Z)"
	result := fmt.Sprintf("in:%s out:%s", formatTokenCount(tokens.InputTokens), formatTokenCount(tokens.OutputTokens))
	if tokens.CacheReadTokens > 0 {
		result += fmt.Sprintf(" (cache:%s)", formatTokenCount(tokens.CacheReadTokens))
	}
	return result
}

// formatTokenStatsCompact returns a compact formatted string of token usage for table display.
// Shows total tokens with input/output breakdown: "12.5K (8K/4K)"
func formatTokenStatsCompact(tokens *opencode.TokenStats) string {
	if tokens == nil {
		return "-"
	}
	total := tokens.TotalTokens
	if total == 0 {
		total = tokens.InputTokens + tokens.OutputTokens
	}
	if total == 0 {
		return "-"
	}
	// Format: "total (in/out)" for quick scanning
	return fmt.Sprintf("%s (%s/%s)",
		formatTokenCount(total),
		formatTokenCount(tokens.InputTokens),
		formatTokenCount(tokens.OutputTokens))
}
