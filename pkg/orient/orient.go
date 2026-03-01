package orient

import (
	"fmt"
	"strings"
	"time"
)

// Event is a simplified event from events.jsonl.
type Event struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Throughput holds aggregate throughput metrics for a time window.
type Throughput struct {
	Days           int `json:"days"`
	Spawns         int `json:"spawns"`
	Completions    int `json:"completions"`
	Abandonments   int `json:"abandonments"`
	InProgress     int `json:"in_progress"`
	AvgDurationMin int `json:"avg_duration_min"`
}

// ReadyIssue represents a beads issue ready for work.
type ReadyIssue struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Priority  string    `json:"priority"`
	KBContext []KBEntry `json:"kb_context,omitempty"` // Relevant decisions, constraints, and failed attempts
}

// OrientationData holds all data needed to render session orientation.
type OrientationData struct {
	Throughput     Throughput       `json:"throughput"`
	ReadyIssues    []ReadyIssue     `json:"ready_issues,omitempty"`
	RelevantModels []ModelFreshness `json:"relevant_models,omitempty"`
	StaleModels    []ModelFreshness `json:"stale_models,omitempty"`
	FocusGoal      string           `json:"focus_goal,omitempty"`
}

// ComputeThroughput aggregates events within the given day window.
func ComputeThroughput(events []Event, now time.Time, days int) Throughput {
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	cutoffUnix := cutoff.Unix()

	var tp Throughput
	tp.Days = days
	var totalDuration float64
	var durationCount int

	for _, e := range events {
		if e.Timestamp < cutoffUnix {
			continue
		}
		switch e.Type {
		case "session.spawned":
			tp.Spawns++
		case "agent.completed":
			tp.Completions++
			if e.Data != nil {
				// Check duration_seconds (current event format) first, then duration_minutes (legacy)
				if d, ok := e.Data["duration_seconds"]; ok {
					if df, ok := d.(float64); ok {
						totalDuration += df / 60.0
						durationCount++
					}
				} else if d, ok := e.Data["duration_minutes"]; ok {
					if df, ok := d.(float64); ok {
						totalDuration += df
						durationCount++
					}
				}
			}
		case "agent.abandoned":
			tp.Abandonments++
		}
	}

	if durationCount > 0 {
		tp.AvgDurationMin = int(totalDuration / float64(durationCount))
	}

	return tp
}

// FormatOrientation renders OrientationData as structured text for orchestrator consumption.
func FormatOrientation(data *OrientationData) string {
	var b strings.Builder

	b.WriteString("== SESSION ORIENTATION ==\n\n")

	// Throughput section
	formatThroughput(&b, &data.Throughput)

	// Ready work section
	formatReadyIssues(&b, data.ReadyIssues)

	// Relevant models section
	formatRelevantModels(&b, data.RelevantModels)

	// Stale models section
	formatStaleModels(&b, data.StaleModels)

	// Focus section
	formatFocus(&b, data.FocusGoal)

	return b.String()
}

func formatThroughput(b *strings.Builder, tp *Throughput) {
	if tp.Days == 1 {
		b.WriteString("Last 24h:\n")
	} else {
		b.WriteString(fmt.Sprintf("Last %dd:\n", tp.Days))
	}
	b.WriteString(fmt.Sprintf("   Completions: %d | Abandonments: %d | In-progress: %d\n",
		tp.Completions, tp.Abandonments, tp.InProgress))
	if tp.AvgDurationMin > 0 {
		b.WriteString(fmt.Sprintf("   Avg duration: %d min\n", tp.AvgDurationMin))
	}
	b.WriteString("\n")
}

func formatReadyIssues(b *strings.Builder, issues []ReadyIssue) {
	b.WriteString("Ready to work:\n")
	if len(issues) == 0 {
		b.WriteString("   No issues ready\n")
	} else {
		for _, issue := range issues {
			b.WriteString(fmt.Sprintf("   [%s] %s (%s)\n", issue.Priority, issue.Title, issue.ID))
			for _, entry := range issue.KBContext {
				content := truncateSummary(entry.Content, 80)
				b.WriteString(fmt.Sprintf("      %s: %s\n", entry.Type, content))
			}
		}
	}
	b.WriteString("\n")
}

func formatRelevantModels(b *strings.Builder, models []ModelFreshness) {
	if len(models) == 0 {
		return
	}
	b.WriteString("Relevant models:\n")
	for _, m := range models {
		age := HumanAge(m.AgeDays)
		summary := truncateSummary(m.Summary, 100)
		b.WriteString(fmt.Sprintf("   - %s (updated %s): %s\n", m.Name, age, summary))
	}
	b.WriteString("\n")
}

func formatStaleModels(b *strings.Builder, models []ModelFreshness) {
	if len(models) == 0 {
		return
	}
	b.WriteString("Stale models:\n")
	for _, m := range models {
		age := HumanAge(m.AgeDays)
		probeNote := "no recent probes"
		if m.HasRecentProbes {
			probeNote = "has recent probes"
		}
		b.WriteString(fmt.Sprintf("   - %s (updated %s, %s)\n", m.Name, age, probeNote))
	}
	b.WriteString("\n")
}

func formatFocus(b *strings.Builder, goal string) {
	if goal == "" {
		return
	}
	b.WriteString(fmt.Sprintf("Focus: %s\n", goal))
}

// truncateSummary truncates a summary to maxLen characters, adding "..." if truncated.
func truncateSummary(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Truncate at word boundary
	truncated := s[:maxLen]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}
