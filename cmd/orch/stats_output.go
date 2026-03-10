// stats_output.go - Text and JSON output formatting for stats command
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func outputStatsJSON(report *StatsReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func outputStatsText(report *StatsReport) error {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("📊 ORCHESTRATION STATISTICS")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Period: %s  |  Events analyzed: %d\n", report.AnalysisPeriod, report.EventsAnalyzed)
	fmt.Println(strings.Repeat("-", 70))

	// Core metrics
	fmt.Println()
	fmt.Println("🎯 CORE METRICS")
	fmt.Printf("  Spawns:        %d\n", report.Summary.TotalSpawns)
	fmt.Printf("  Completions:   %d (%.1f%%)\n", report.Summary.TotalCompletions, report.Summary.CompletionRate)
	fmt.Printf("  Abandonments:  %d (%.1f%%)\n", report.Summary.TotalAbandonments, report.Summary.AbandonmentRate)
	if report.Summary.AvgDurationMinutes > 0 {
		fmt.Printf("  Avg Duration:  %.0f minutes\n", report.Summary.AvgDurationMinutes)
	}

	// Task vs Coordination breakdown
	fmt.Println()
	fmt.Println("📋 COMPLETION BY CATEGORY")
	fmt.Printf("  Task Skills:         %d/%d spawns (%.1f%%) ← main metric\n",
		report.Summary.TaskCompletions, report.Summary.TaskSpawns, report.Summary.TaskCompletionRate)
	fmt.Printf("  Coordination Skills: %d/%d spawns (%.1f%%) [interactive sessions]\n",
		report.Summary.CoordinationCompletions, report.Summary.CoordinationSpawns, report.Summary.CoordinationCompletionRate)

	// Daemon metrics
	fmt.Println()
	fmt.Println("🤖 DAEMON HEALTH")
	fmt.Printf("  Daemon spawns:    %d (%.1f%% of all spawns)\n", report.DaemonStats.DaemonSpawns, report.DaemonStats.DaemonSpawnRate)
	fmt.Printf("  Auto-completions: %d\n", report.DaemonStats.AutoCompletions)
	fmt.Printf("  Triage bypassed:  %d\n", report.DaemonStats.TriageBypassed)

	// Wait metrics (if any)
	if report.WaitStats.WaitCompleted > 0 || report.WaitStats.WaitTimeouts > 0 {
		fmt.Println()
		fmt.Println("⏱️  WAIT OPERATIONS")
		fmt.Printf("  Completed: %d\n", report.WaitStats.WaitCompleted)
		fmt.Printf("  Timeouts:  %d (%.1f%% timeout rate)\n", report.WaitStats.WaitTimeouts, report.WaitStats.TimeoutRate)
	}

	// Session metrics (if verbose or has activity)
	if statsVerbose || report.SessionStats.SessionsStarted > 0 {
		fmt.Println()
		fmt.Println("📝 ORCHESTRATOR SESSIONS")
		fmt.Printf("  Started:  %d\n", report.SessionStats.SessionsStarted)
		fmt.Printf("  Ended:    %d\n", report.SessionStats.SessionsEnded)
		fmt.Printf("  Active:   %d\n", report.SessionStats.ActiveSessions)
	}

	// Escape hatch metrics (if any escape hatch spawns exist)
	if report.EscapeHatchStats.TotalSpawns > 0 {
		fmt.Println()
		fmt.Println("🚪 ESCAPE HATCH (--backend claude)")
		fmt.Printf("  Total:     %d spawns (all time)\n", report.EscapeHatchStats.TotalSpawns)
		fmt.Printf("  Last 7d:   %d spawns\n", report.EscapeHatchStats.Last7DaySpawns)
		fmt.Printf("  Last 30d:  %d spawns\n", report.EscapeHatchStats.Last30DaySpawns)
		if report.EscapeHatchStats.EscapeHatchRate > 0 {
			fmt.Printf("  Rate:      %.1f%% of spawns (in analysis window)\n", report.EscapeHatchStats.EscapeHatchRate)
		}

		// Show account breakdown (if more than one account or verbose)
		if len(report.EscapeHatchStats.ByAccount) > 1 || statsVerbose {
			fmt.Println()
			fmt.Println("  By Account:")
			for _, acct := range report.EscapeHatchStats.ByAccount {
				if acct.Account == "unknown" {
					fmt.Printf("    %-35s %4d total (%d 7d, %d 30d)\n", "(no account info)", acct.TotalSpawns, acct.Last7Days, acct.Last30Days)
				} else {
					// Truncate long email addresses
					displayAcct := acct.Account
					if len(displayAcct) > 35 {
						displayAcct = displayAcct[:32] + "..."
					}
					fmt.Printf("    %-35s %4d total (%d 7d, %d 30d)\n", displayAcct, acct.TotalSpawns, acct.Last7Days, acct.Last30Days)
				}
			}
		}
	}

	// Skill breakdown
	if len(report.SkillStats) > 0 {
		fmt.Println()
		fmt.Println("🎭 SKILL BREAKDOWN")
		fmt.Println("  (C) = Coordination skill (excluded from completion rate warning)")
		fmt.Printf("  %-25s %8s %8s %8s %10s\n", "Skill", "Spawns", "Complete", "Abandon", "Rate")
		fmt.Println("  " + strings.Repeat("-", 62))

		// Show top 10 skills by default, all if verbose
		limit := 10
		if statsVerbose {
			limit = len(report.SkillStats)
		}

		for i, skill := range report.SkillStats {
			if i >= limit {
				remaining := len(report.SkillStats) - limit
				fmt.Printf("  ... and %d more skills (use --verbose to show all)\n", remaining)
				break
			}
			// Mark coordination skills with (C) indicator
			skillName := truncateSkill(skill.Skill, 22)
			if skill.Category == CoordinationSkill {
				skillName = skillName + " (C)"
			}
			fmt.Printf("  %-25s %8d %8d %8d %9.1f%%\n",
				skillName,
				skill.Spawns,
				skill.Completions,
				skill.Abandonments,
				skill.CompletionRate,
			)
		}
	}

	// Verification stats (if any completion attempts or bypass events exist)
	hasVerificationData := report.VerificationStats.TotalAttempts > 0 ||
		report.VerificationStats.SkipBypassed > 0 ||
		report.VerificationStats.AutoSkipped > 0
	if hasVerificationData {
		fmt.Println()
		fmt.Println("✅ VERIFICATION GATES")
		if report.VerificationStats.TotalAttempts > 0 {
			fmt.Printf("  Total attempts:     %d\n", report.VerificationStats.TotalAttempts)
			fmt.Printf("  Passed 1st try:     %d (%.1f%%)\n", report.VerificationStats.PassedFirstTry, report.VerificationStats.PassRate)
			fmt.Printf("  Bypassed (--force): %d (%.1f%%)\n", report.VerificationStats.Bypassed, report.VerificationStats.BypassRate)
		}
		if report.VerificationStats.SkipBypassed > 0 {
			fmt.Printf("  Skipped (--skip-*): %d gate bypass events\n", report.VerificationStats.SkipBypassed)
		}
		if report.VerificationStats.AutoSkipped > 0 {
			fmt.Printf("  Auto-skipped:       %d (skill-class/file exemptions)\n", report.VerificationStats.AutoSkipped)
		}

		// Gate breakdown (if there are any gate-level stats)
		if len(report.VerificationStats.FailuresByGate) > 0 {
			fmt.Println()
			fmt.Println("  Gate Breakdown:")
			fmt.Printf("  %-25s %8s %8s %10s %10s\n", "Gate", "Failed", "Bypassed", "AutoSkip", "Fail Rate")
			fmt.Println("  " + strings.Repeat("-", 65))
			for _, gate := range report.VerificationStats.FailuresByGate {
				fmt.Printf("  %-25s %8d %8d %10d %9.1f%%\n",
					gate.Gate,
					gate.FailCount,
					gate.BypassCount,
					gate.AutoSkipCount,
					gate.FailRate,
				)
			}
		}

		// Bypass reasons (if any --skip-* bypasses with reasons exist)
		if len(report.VerificationStats.BypassReasons) > 0 {
			fmt.Println()
			fmt.Println("  Bypass Reasons (--skip-*):")
			for _, br := range report.VerificationStats.BypassReasons {
				reason := br.Reason
				if reason == "" {
					reason = "(no reason)"
				}
				if len(reason) > 50 {
					reason = reason[:47] + "..."
				}
				fmt.Printf("    %-20s %dx  %s\n", br.Gate, br.Count, reason)
			}
		}

		// Skill breakdown (if verbose and there's skill-level data)
		if statsVerbose && len(report.VerificationStats.BySkill) > 0 {
			fmt.Println()
			fmt.Println("  By Skill:")
			fmt.Printf("  %-25s %8s %8s %8s %10s\n", "Skill", "Attempts", "Passed", "Bypassed", "Pass Rate")
			fmt.Println("  " + strings.Repeat("-", 62))
			for _, sv := range report.VerificationStats.BySkill {
				fmt.Printf("  %-25s %8d %8d %8d %9.1f%%\n",
					truncateSkill(sv.Skill, 22),
					sv.TotalAttempts,
					sv.PassedFirstTry,
					sv.Bypassed,
					sv.PassRate,
				)
			}
		}
	}

	// Spawn gate bypasses (unified view of all spawn-level gate bypasses)
	if report.SpawnGateStats.TotalBypasses > 0 {
		fmt.Println()
		fmt.Println("🚧 SPAWN GATE BYPASSES")
		fmt.Printf("  Total bypasses: %d / %d spawns (%.1f%%)\n",
			report.SpawnGateStats.TotalBypasses,
			report.SpawnGateStats.TotalSpawns,
			report.SpawnGateStats.BypassRate)
		fmt.Println()
		fmt.Printf("  %-20s %8s %10s %s\n", "Gate", "Bypassed", "Rate", "Status")
		fmt.Println("  " + strings.Repeat("-", 55))
		for _, gate := range report.SpawnGateStats.ByGate {
			status := "OK"
			if gate.Miscalibrated {
				status = "MISCALIBRATED"
			}
			fmt.Printf("  %-20s %8d %9.1f%% %s\n",
				gate.Gate, gate.Bypassed, gate.BypassRate, status)
		}

		// Top reasons (if any)
		if len(report.SpawnGateStats.TopReasons) > 0 {
			fmt.Println()
			fmt.Println("  Top Bypass Reasons:")
			limit := 5
			if len(report.SpawnGateStats.TopReasons) < limit {
				limit = len(report.SpawnGateStats.TopReasons)
			}
			for i := 0; i < limit; i++ {
				r := report.SpawnGateStats.TopReasons[i]
				reason := r.Reason
				if len(reason) > 50 {
					reason = reason[:47] + "..."
				}
				fmt.Printf("    %dx  [%s] %s\n", r.Count, r.Gate, reason)
			}
		}

		// Miscalibration warnings
		for _, gate := range report.SpawnGateStats.ByGate {
			if gate.Miscalibrated {
				fmt.Printf("\n  ⚠️  %s gate bypassed >50%% of spawns — review if gate is too strict\n", gate.Gate)
			}
		}
	}

	// Override reasons (if any override events with reasons exist)
	if report.OverrideStats.TotalOverrides > 0 {
		fmt.Println()
		fmt.Println("🔓 OVERRIDE REASONS")
		fmt.Printf("  Total overrides with reasons: %d\n", report.OverrideStats.TotalOverrides)
		fmt.Println()
		for _, entry := range report.OverrideStats.ByType {
			fmt.Printf("  %s (%d):\n", entry.Type, entry.Count)
			if len(entry.Reasons) > 0 {
				for _, reason := range entry.Reasons {
					r := reason.Reason
					if len(r) > 55 {
						r = r[:52] + "..."
					}
					fmt.Printf("    %dx  %s\n", reason.Count, r)
				}
			} else {
				fmt.Println("    (no reasons recorded)")
			}
		}
	}

	// Behavioral health (coaching metrics)
	if len(report.CoachingStats) > 0 {
		fmt.Println()
		fmt.Println("🧠 BEHAVIORAL HEALTH (coaching metrics)")

		// Separate orchestrator and worker metrics
		orchestratorMetrics := []string{"frame_collapse", "completion_backlog", "behavioral_variation", "circular_pattern"}
		workerMetrics := []string{"tool_failure_rate", "context_usage", "session_timeout", "spawn_depth_exceeded"}

		// Display orchestrator metrics
		hasOrchestratorMetrics := false
		for _, metricType := range orchestratorMetrics {
			if _, exists := report.CoachingStats[metricType]; exists {
				hasOrchestratorMetrics = true
				break
			}
		}

		if hasOrchestratorMetrics {
			fmt.Println("  Orchestrator:")
			for _, metricType := range orchestratorMetrics {
				if summary, exists := report.CoachingStats[metricType]; exists {
					timeSince := time.Since(summary.LastSeen)
					var timeStr string
					if timeSince < time.Minute {
						timeStr = "just now"
					} else if timeSince < time.Hour {
						timeStr = fmt.Sprintf("%dm ago", int(timeSince.Minutes()))
					} else if timeSince < 24*time.Hour {
						timeStr = fmt.Sprintf("%dh ago", int(timeSince.Hours()))
					} else {
						timeStr = fmt.Sprintf("%dd ago", int(timeSince.Hours()/24))
					}

					// Add warning emoji for recent events
					warningEmoji := ""
					if timeSince < 30*time.Minute {
						warningEmoji = " ⚠️"
					} else if summary.Count == 0 {
						warningEmoji = " ✅"
					}

					fmt.Printf("    %-25s %d events (last: %s)%s\n",
						metricType+":", summary.Count, timeStr, warningEmoji)
				}
			}
		}

		// Display worker metrics
		hasWorkerMetrics := false
		for _, metricType := range workerMetrics {
			if _, exists := report.CoachingStats[metricType]; exists {
				hasWorkerMetrics = true
				break
			}
		}

		if hasWorkerMetrics {
			if hasOrchestratorMetrics {
				fmt.Println()
			}
			fmt.Println("  Workers:")
			for _, metricType := range workerMetrics {
				if summary, exists := report.CoachingStats[metricType]; exists {
					timeSince := time.Since(summary.LastSeen)
					var timeStr string
					if timeSince < time.Minute {
						timeStr = "just now"
					} else if timeSince < time.Hour {
						timeStr = fmt.Sprintf("%dm ago", int(timeSince.Minutes()))
					} else if timeSince < 24*time.Hour {
						timeStr = fmt.Sprintf("%dh ago", int(timeSince.Hours()))
					} else {
						timeStr = fmt.Sprintf("%dd ago", int(timeSince.Hours()/24))
					}

					fmt.Printf("    %-25s %d events (last: %s)\n",
						metricType+":", summary.Count, timeStr)
				}
			}
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))

	// Quick health assessment (based on task skill rate, not overall)
	// Coordination skills (orchestrator, meta-orchestrator) are interactive sessions,
	// not completable tasks, so they're excluded from the health check.
	if report.Summary.TaskSpawns > 0 && report.Summary.TaskCompletionRate < 80 {
		fmt.Println("⚠️  WARNING: Task skill completion rate below 80% - investigate failure patterns")
	} else if report.Summary.TaskSpawns > 0 && report.Summary.TaskCompletionRate >= 95 {
		fmt.Println("✅ HEALTHY: Task skill completion rate at 95%+")
	}

	// Verification health check
	if report.VerificationStats.TotalAttempts > 0 && report.VerificationStats.BypassRate > 50 {
		fmt.Println("⚠️  WARNING: >50% of completions bypassed verification - gates may be miscalibrated")
	}

	return nil
}

func truncateSkill(skill string, maxLen int) string {
	if len(skill) <= maxLen {
		return skill
	}
	return skill[:maxLen-3] + "..."
}
