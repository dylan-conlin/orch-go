// stats_cmd.go - Aggregate events.jsonl metrics for orchestration observability
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	statsDays       int
	statsJSONOutput bool
	statsVerbose    bool
)

// SkillCategory represents the type of work a skill does
type SkillCategory string

const (
	// TaskSkill represents skills that complete discrete tasks
	TaskSkill SkillCategory = "task"
	// CoordinationSkill represents skills that coordinate other agents (not meant to complete)
	CoordinationSkill SkillCategory = "coordination"
)

// coordinationSkills lists skills that are coordination roles, not completable tasks.
// These are excluded from the completion rate warning because they're interactive sessions
// designed to run until context exhaustion, not complete discrete tasks.
var coordinationSkills = map[string]bool{
	"orchestrator":      true,
	"meta-orchestrator": true,
}

// getSkillCategory returns the category of a skill
func getSkillCategory(skill string) SkillCategory {
	if coordinationSkills[skill] {
		return CoordinationSkill
	}
	return TaskSkill
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show aggregated agent statistics from events.jsonl",
	Long: `Aggregate events.jsonl to surface orchestration metrics.

Shows:
  - Spawn and completion counts
  - Completion and abandonment rates
  - Average session duration
  - Skill effectiveness breakdown
  - Daemon health metrics

Examples:
  orch stats                  # Show last 7 days
  orch stats --days 1         # Show last 24 hours
  orch stats --days 30        # Show last 30 days
  orch stats --json           # Output as JSON for scripting
  orch stats --verbose        # Show additional metrics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStats()
	},
}

func init() {
	statsCmd.Flags().IntVar(&statsDays, "days", 7, "Number of days to analyze")
	statsCmd.Flags().BoolVar(&statsJSONOutput, "json", false, "Output as JSON")
	statsCmd.Flags().BoolVar(&statsVerbose, "verbose", false, "Show additional metrics")

	rootCmd.AddCommand(statsCmd)
}

// Event represents a parsed event from events.jsonl
type StatsEvent struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// StatsReport contains all aggregated statistics
type StatsReport struct {
	GeneratedAt    string              `json:"generated_at"`
	AnalysisPeriod string              `json:"analysis_period"`
	DaysAnalyzed   int                 `json:"days_analyzed"`
	EventsAnalyzed int                 `json:"events_analyzed"`
	Summary        StatsSummary        `json:"summary"`
	SkillStats     []SkillStatsSummary `json:"skill_stats"`
	DaemonStats    DaemonStatsSummary  `json:"daemon_stats"`
	WaitStats      WaitStatsSummary    `json:"wait_stats,omitempty"`
	SessionStats   SessionStatsSummary `json:"session_stats,omitempty"`
}

// StatsSummary contains core metrics
type StatsSummary struct {
	TotalSpawns        int     `json:"total_spawns"`
	TotalCompletions   int     `json:"total_completions"`
	TotalAbandonments  int     `json:"total_abandonments"`
	CompletionRate     float64 `json:"completion_rate"`
	AbandonmentRate    float64 `json:"abandonment_rate"`
	AvgDurationMinutes float64 `json:"avg_duration_minutes,omitempty"`
	// Task skill specific metrics (excludes coordination skills like orchestrator)
	TaskSpawns         int     `json:"task_spawns"`
	TaskCompletions    int     `json:"task_completions"`
	TaskCompletionRate float64 `json:"task_completion_rate"`
	// Coordination skill metrics (orchestrator, meta-orchestrator)
	CoordinationSpawns         int     `json:"coordination_spawns"`
	CoordinationCompletions    int     `json:"coordination_completions"`
	CoordinationCompletionRate float64 `json:"coordination_completion_rate"`
}

// SkillStatsSummary contains per-skill metrics
type SkillStatsSummary struct {
	Skill          string        `json:"skill"`
	Category       SkillCategory `json:"category"`
	Spawns         int           `json:"spawns"`
	Completions    int           `json:"completions"`
	Abandonments   int           `json:"abandonments"`
	CompletionRate float64       `json:"completion_rate"`
}

// DaemonStatsSummary contains daemon-specific metrics
type DaemonStatsSummary struct {
	DaemonSpawns    int     `json:"daemon_spawns"`
	AutoCompletions int     `json:"auto_completions"`
	TriageBypassed  int     `json:"triage_bypassed"`
	DaemonSpawnRate float64 `json:"daemon_spawn_rate"`
}

// WaitStatsSummary contains wait operation metrics
type WaitStatsSummary struct {
	WaitCompleted int     `json:"wait_completed"`
	WaitTimeouts  int     `json:"wait_timeouts"`
	TimeoutRate   float64 `json:"timeout_rate"`
}

// SessionStatsSummary contains orchestrator session metrics
type SessionStatsSummary struct {
	SessionsStarted int `json:"sessions_started"`
	SessionsEnded   int `json:"sessions_ended"`
	ActiveSessions  int `json:"active_sessions"`
}

func runStats() error {
	// Get events file path
	eventsPath := getEventsPath()

	// Parse events
	events, err := parseEvents(eventsPath, statsDays)
	if err != nil {
		return fmt.Errorf("failed to parse events: %w", err)
	}

	// Aggregate statistics
	report := aggregateStats(events, statsDays)

	// Output
	if statsJSONOutput {
		return outputStatsJSON(report)
	}
	return outputStatsText(report)
}

func getEventsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/events.jsonl"
	}
	return filepath.Join(home, ".orch", "events.jsonl")
}

func parseEvents(path string, days int) ([]StatsEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("events.jsonl not found at %s - no events recorded yet", path)
		}
		return nil, fmt.Errorf("failed to open events file: %w", err)
	}
	defer file.Close()

	cutoff := time.Now().AddDate(0, 0, -days).Unix()

	var events []StatsEvent
	scanner := bufio.NewScanner(file)
	// Increase buffer size for potentially long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event StatsEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip malformed lines
			continue
		}

		// Filter by time window
		if event.Timestamp >= cutoff {
			events = append(events, event)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}

	return events, nil
}

func aggregateStats(events []StatsEvent, days int) *StatsReport {
	report := &StatsReport{
		GeneratedAt:    time.Now().Format(time.RFC3339),
		AnalysisPeriod: fmt.Sprintf("Last %d days", days),
		DaysAnalyzed:   days,
		EventsAnalyzed: len(events),
		SkillStats:     []SkillStatsSummary{},
	}

	// Track spawn times for duration calculation
	spawnTimes := make(map[string]int64)     // session_id -> timestamp
	spawnSkills := make(map[string]string)   // session_id -> skill
	spawnBeadsIDs := make(map[string]string) // session_id -> beads_id
	skillCounts := make(map[string]*SkillStatsSummary)
	var durations []float64

	// Track beads completions to correlate with spawns
	beadsCompletions := make(map[string]int64) // beads_id -> completion timestamp

	for _, event := range events {
		switch event.Type {
		case "session.spawned":
			report.Summary.TotalSpawns++
			spawnTimes[event.SessionID] = event.Timestamp

			// Extract skill from data
			if data := event.Data; data != nil {
				if skill, ok := data["skill"].(string); ok && skill != "" {
					spawnSkills[event.SessionID] = skill
					if _, exists := skillCounts[skill]; !exists {
						skillCounts[skill] = &SkillStatsSummary{
							Skill:    skill,
							Category: getSkillCategory(skill),
						}
					}
					skillCounts[skill].Spawns++
				}
				if beadsID, ok := data["beads_id"].(string); ok && beadsID != "" {
					spawnBeadsIDs[event.SessionID] = beadsID
				}
			}

		case "session.completed":
			report.Summary.TotalCompletions++
			// Calculate duration if we have spawn time
			if spawnTime, ok := spawnTimes[event.SessionID]; ok {
				duration := float64(event.Timestamp-spawnTime) / 60.0 // minutes
				if duration > 0 && duration < 480 {                   // Sanity check: < 8 hours
					durations = append(durations, duration)
				}
			}
			// Update skill completions
			if skill, ok := spawnSkills[event.SessionID]; ok {
				if stats, exists := skillCounts[skill]; exists {
					stats.Completions++
				}
			}

		case "agent.completed":
			report.Summary.TotalCompletions++
			// Extract beads_id for correlation
			if data := event.Data; data != nil {
				if beadsID, ok := data["beads_id"].(string); ok && beadsID != "" {
					beadsCompletions[beadsID] = event.Timestamp
				}
			}
			// Calculate duration by matching beads_id
			if data := event.Data; data != nil {
				if beadsID, ok := data["beads_id"].(string); ok && beadsID != "" {
					// Find spawn with matching beads_id
					for sessionID, spawnBeadsID := range spawnBeadsIDs {
						if spawnBeadsID == beadsID {
							if spawnTime, ok := spawnTimes[sessionID]; ok {
								duration := float64(event.Timestamp-spawnTime) / 60.0
								if duration > 0 && duration < 480 {
									durations = append(durations, duration)
								}
							}
							// Update skill completions
							if skill, ok := spawnSkills[sessionID]; ok {
								if stats, exists := skillCounts[skill]; exists {
									stats.Completions++
								}
							}
							break
						}
					}
				}
			}

		case "agent.abandoned":
			report.Summary.TotalAbandonments++
			// Update skill abandonments
			if data := event.Data; data != nil {
				if beadsID, ok := data["beads_id"].(string); ok && beadsID != "" {
					for sessionID, spawnBeadsID := range spawnBeadsIDs {
						if spawnBeadsID == beadsID {
							if skill, ok := spawnSkills[sessionID]; ok {
								if stats, exists := skillCounts[skill]; exists {
									stats.Abandonments++
								}
							}
							break
						}
					}
				}
			}

		case "daemon.spawn":
			report.DaemonStats.DaemonSpawns++

		case "session.auto_completed":
			report.DaemonStats.AutoCompletions++

		case "spawn.triage_bypassed":
			report.DaemonStats.TriageBypassed++

		case "agent.wait.complete":
			report.WaitStats.WaitCompleted++

		case "agent.wait.timeout":
			report.WaitStats.WaitTimeouts++

		case "session.orchestrator.started", "session.started":
			report.SessionStats.SessionsStarted++

		case "session.orchestrator.ended", "session.ended":
			report.SessionStats.SessionsEnded++
		}
	}

	// Calculate rates
	if report.Summary.TotalSpawns > 0 {
		report.Summary.CompletionRate = float64(report.Summary.TotalCompletions) / float64(report.Summary.TotalSpawns) * 100
		report.Summary.AbandonmentRate = float64(report.Summary.TotalAbandonments) / float64(report.Summary.TotalSpawns) * 100
		report.DaemonStats.DaemonSpawnRate = float64(report.DaemonStats.DaemonSpawns) / float64(report.Summary.TotalSpawns) * 100
	}

	// Calculate average duration
	if len(durations) > 0 {
		var total float64
		for _, d := range durations {
			total += d
		}
		report.Summary.AvgDurationMinutes = total / float64(len(durations))
	}

	// Calculate wait timeout rate
	totalWaits := report.WaitStats.WaitCompleted + report.WaitStats.WaitTimeouts
	if totalWaits > 0 {
		report.WaitStats.TimeoutRate = float64(report.WaitStats.WaitTimeouts) / float64(totalWaits) * 100
	}

	// Calculate active sessions
	report.SessionStats.ActiveSessions = report.SessionStats.SessionsStarted - report.SessionStats.SessionsEnded

	// Calculate per-skill completion rates and aggregate by category
	for _, stats := range skillCounts {
		if stats.Spawns > 0 {
			stats.CompletionRate = float64(stats.Completions) / float64(stats.Spawns) * 100
		}
		report.SkillStats = append(report.SkillStats, *stats)

		// Aggregate by category
		if stats.Category == TaskSkill {
			report.Summary.TaskSpawns += stats.Spawns
			report.Summary.TaskCompletions += stats.Completions
		} else if stats.Category == CoordinationSkill {
			report.Summary.CoordinationSpawns += stats.Spawns
			report.Summary.CoordinationCompletions += stats.Completions
		}
	}

	// Calculate task skill completion rate
	if report.Summary.TaskSpawns > 0 {
		report.Summary.TaskCompletionRate = float64(report.Summary.TaskCompletions) / float64(report.Summary.TaskSpawns) * 100
	}

	// Calculate coordination skill completion rate
	if report.Summary.CoordinationSpawns > 0 {
		report.Summary.CoordinationCompletionRate = float64(report.Summary.CoordinationCompletions) / float64(report.Summary.CoordinationSpawns) * 100
	}

	// Sort skills by spawn count descending
	sort.Slice(report.SkillStats, func(i, j int) bool {
		return report.SkillStats[i].Spawns > report.SkillStats[j].Spawns
	})

	return report
}

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

	return nil
}

func truncateSkill(skill string, maxLen int) string {
	if len(skill) <= maxLen {
		return skill
	}
	return skill[:maxLen-3] + "..."
}
