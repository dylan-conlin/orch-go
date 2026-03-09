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

// ActiveThread represents a living thread surfaced during orientation.
type ActiveThread struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Updated     string `json:"updated"`
	EntryCount  int    `json:"entry_count"`
	LatestEntry string `json:"latest_entry"`
}

// HealthAlert represents a health threshold crossing surfaced during orientation.
type HealthAlert struct {
	Message string `json:"message"`
	Level   string `json:"level"` // "warn" or "critical"
}

// HealthSummary holds a lightweight health snapshot for orientation.
type HealthSummary struct {
	OpenIssues    int           `json:"open_issues"`
	BlockedIssues int           `json:"blocked_issues"`
	StaleIssues   int           `json:"stale_issues"`
	BloatedFiles  int           `json:"bloated_files"`
	FixFeatRatio  float64       `json:"fix_feat_ratio"`
	Alerts        []HealthAlert `json:"alerts,omitempty"`
}

// ReflectSummary holds reflection suggestions from kb reflect daemon output.
type ReflectSummary struct {
	Total       int              `json:"total"`
	Synthesis   int              `json:"synthesis"`
	Stale       int              `json:"stale"`
	Promote     int              `json:"promote"`
	Drift       int              `json:"drift"`
	Agreements  int              `json:"agreements"`
	TopClusters []ReflectCluster `json:"top_clusters,omitempty"`
	Age         string           `json:"age,omitempty"` // human-readable age like "2h ago"
	OrphanRate  float64          `json:"orphan_rate,omitempty"`  // percentage 0-100
	OrphanTotal int              `json:"orphan_total,omitempty"` // total investigations counted
}

// ReflectCluster represents a synthesis opportunity cluster.
type ReflectCluster struct {
	Topic string `json:"topic"`
	Count int    `json:"count"`
}

// UsageWarning holds Claude Max usage information when above threshold.
type UsageWarning struct {
	Utilization float64 `json:"utilization"` // percentage 0-100
	Remaining   string  `json:"remaining"`
	ResetTime   string  `json:"reset_time"`
	Level       string  `json:"level"` // "WARNING", "HIGH", "CRITICAL"
}

// ConfigDriftItem represents a single config symlink that has drifted.
type ConfigDriftItem struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

// DaemonHealthSignalView represents a single health signal for orient display.
type DaemonHealthSignalView struct {
	Name   string `json:"name"`
	Level  string `json:"level"`  // "green", "yellow", "red"
	Detail string `json:"detail"`
}

// DaemonHealthView holds daemon health signals for orient display.
type DaemonHealthView struct {
	Signals []DaemonHealthSignalView `json:"signals"`
}

// SessionResume holds session handoff context for resume injection.
type SessionResume struct {
	Content string `json:"content"`
	Source  string `json:"source,omitempty"` // path to handoff file
}

// OrientationData holds all data needed to render session orientation.
type OrientationData struct {
	Throughput      Throughput       `json:"throughput"`
	PreviousSession *DebriefSummary  `json:"previous_session,omitempty"`
	ReadyIssues     []ReadyIssue     `json:"ready_issues,omitempty"`
	ActivePlans     []PlanSummary    `json:"active_plans,omitempty"`
	ActiveThreads   []ActiveThread   `json:"active_threads,omitempty"`
	RelevantModels  []ModelFreshness `json:"relevant_models,omitempty"`
	StaleModels     []ModelFreshness `json:"stale_models,omitempty"`
	HealthSummary   *HealthSummary   `json:"health_summary,omitempty"`
	DaemonHealth    *DaemonHealthView `json:"daemon_health,omitempty"`
	Changelog       []ChangelogEntry `json:"changelog,omitempty"`
	FocusGoal       string           `json:"focus_goal,omitempty"`
	ReflectSummary  *ReflectSummary  `json:"reflect_summary,omitempty"`
	UsageWarning    *UsageWarning    `json:"usage_warning,omitempty"`
	ConfigDrift     []ConfigDriftItem `json:"config_drift,omitempty"`
	SessionResume   *SessionResume   `json:"session_resume,omitempty"`
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

	// Session resume (first — sets context for everything else)
	formatSessionResume(&b, data.SessionResume)

	// Config drift (surface problems early)
	formatConfigDrift(&b, data.ConfigDrift)

	// Usage warning (surface before work planning)
	formatUsageWarning(&b, data.UsageWarning)

	// Throughput section
	formatThroughput(&b, &data.Throughput)

	// Last session insight — prominent comprehension thread from prior session
	b.WriteString(FormatLastSessionInsight(data.PreviousSession))

	// Previous session section (from debrief)
	b.WriteString(FormatPreviousSession(data.PreviousSession))

	// Changelog since last session
	sinceDate := ""
	if data.PreviousSession != nil {
		sinceDate = data.PreviousSession.Date
	}
	b.WriteString(FormatChangelog(data.Changelog, sinceDate))

	// Ready work section
	formatReadyIssues(&b, data.ReadyIssues)

	// Active plans section
	formatActivePlans(&b, data.ActivePlans)

	// Active threads section
	formatActiveThreads(&b, data.ActiveThreads)

	// Relevant models section
	formatRelevantModels(&b, data.RelevantModels)

	// Stale models section
	formatStaleModels(&b, data.StaleModels)

	// Health summary section
	formatHealthSummary(&b, data.HealthSummary)

	// Daemon health signals
	formatDaemonHealth(&b, data.DaemonHealth)

	// Focus section
	formatFocus(&b, data.FocusGoal)

	// Reflection suggestions (last — informational, not urgent)
	formatReflectSummary(&b, data.ReflectSummary)

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

func formatActivePlans(b *strings.Builder, plans []PlanSummary) {
	if len(plans) == 0 {
		return
	}
	b.WriteString("Active plans:\n")
	for _, plan := range plans {
		b.WriteString(fmt.Sprintf("   - %s", plan.Title))
		if len(plan.Projects) > 0 {
			b.WriteString(fmt.Sprintf(" [%s]", strings.Join(plan.Projects, ", ")))
		}
		b.WriteString("\n")
		if plan.TLDR != "" {
			tldr := truncateSummary(plan.TLDR, 100)
			b.WriteString(fmt.Sprintf("     %s\n", tldr))
		}
		for _, phase := range plan.Phases {
			marker := " "
			switch phase.Status {
			case "complete":
				marker = "x"
			case "in-progress":
				marker = ">"
			case "blocked":
				marker = "!"
			}
			b.WriteString(fmt.Sprintf("     [%s] %s\n", marker, phase.Name))
		}
	}
	b.WriteString("\n")
}

func formatActiveThreads(b *strings.Builder, threads []ActiveThread) {
	if len(threads) == 0 {
		return
	}
	b.WriteString("Active threads:\n")
	for _, t := range threads {
		b.WriteString(fmt.Sprintf("   - %s (updated %s, %d entries)\n", t.Title, t.Updated, t.EntryCount))
		if t.LatestEntry != "" {
			preview := truncateSummary(t.LatestEntry, 100)
			b.WriteString(fmt.Sprintf("     > %s\n", preview))
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

func formatHealthSummary(b *strings.Builder, h *HealthSummary) {
	if h == nil {
		return
	}
	b.WriteString("Health:\n")
	b.WriteString(fmt.Sprintf("   Open: %d | Blocked: %d | Stale: %d | Bloated files: %d\n",
		h.OpenIssues, h.BlockedIssues, h.StaleIssues, h.BloatedFiles))
	b.WriteString(fmt.Sprintf("   Fix:feat %.1f (28d)\n", h.FixFeatRatio))
	for _, alert := range h.Alerts {
		icon := "!"
		if alert.Level == "critical" {
			icon = "!!!"
		}
		b.WriteString(fmt.Sprintf("   [%s] %s\n", icon, alert.Message))
	}
	b.WriteString("\n")
}

func formatDaemonHealth(b *strings.Builder, dh *DaemonHealthView) {
	if dh == nil || len(dh.Signals) == 0 {
		return
	}

	// Only show section if there are non-green signals
	hasNonGreen := false
	for _, sig := range dh.Signals {
		if sig.Level != "green" {
			hasNonGreen = true
			break
		}
	}

	if !hasNonGreen {
		return
	}

	b.WriteString("Daemon health:\n")
	for _, sig := range dh.Signals {
		if sig.Level == "green" {
			continue
		}
		icon := levelIcon(sig.Level)
		b.WriteString(fmt.Sprintf("   %s %s: %s\n", icon, sig.Name, sig.Detail))
	}
	b.WriteString("\n")
}

func levelIcon(level string) string {
	switch level {
	case "red":
		return "[!!!]"
	case "yellow":
		return "[!]"
	default:
		return ""
	}
}

func formatFocus(b *strings.Builder, goal string) {
	if goal == "" {
		return
	}
	b.WriteString(fmt.Sprintf("Focus: %s\n", goal))
}

func formatReflectSummary(b *strings.Builder, r *ReflectSummary) {
	if r == nil || r.Total == 0 {
		return
	}
	b.WriteString("Reflection suggestions:\n")
	b.WriteString(fmt.Sprintf("   %d items need attention", r.Total))
	if r.Age != "" {
		b.WriteString(fmt.Sprintf(" (from %s)", r.Age))
	}
	b.WriteString("\n")

	if r.Agreements > 0 {
		b.WriteString(fmt.Sprintf("   - %d broken agreements\n", r.Agreements))
	}
	if r.Synthesis > 0 {
		b.WriteString(fmt.Sprintf("   - %d synthesis opportunities\n", r.Synthesis))
	}
	if r.Stale > 0 {
		b.WriteString(fmt.Sprintf("   - %d stale decisions\n", r.Stale))
	}
	if r.Promote > 0 {
		b.WriteString(fmt.Sprintf("   - %d promotion candidates\n", r.Promote))
	}
	if r.Drift > 0 {
		b.WriteString(fmt.Sprintf("   - %d potential drifts\n", r.Drift))
	}

	if len(r.TopClusters) > 0 {
		b.WriteString("   Top clusters:")
		for _, c := range r.TopClusters {
			b.WriteString(fmt.Sprintf(" %s(%d)", c.Topic, c.Count))
		}
		b.WriteString("\n")
	}
	if r.OrphanTotal > 0 {
		b.WriteString(fmt.Sprintf("   Orphan rate: %.1f%% (%d investigations)\n", r.OrphanRate, r.OrphanTotal))
	}
	b.WriteString("\n")
}

func formatUsageWarning(b *strings.Builder, u *UsageWarning) {
	if u == nil {
		return
	}
	b.WriteString(fmt.Sprintf("Usage %s: %.0f%% of weekly limit used (%s remaining)\n",
		u.Level, u.Utilization, u.Remaining))
	if u.ResetTime != "" {
		b.WriteString(fmt.Sprintf("   Resets in: %s\n", u.ResetTime))
	}
	b.WriteString("\n")
}

func formatConfigDrift(b *strings.Builder, items []ConfigDriftItem) {
	if len(items) == 0 {
		return
	}
	b.WriteString("Config drift detected:\n")
	for _, item := range items {
		b.WriteString(fmt.Sprintf("   - %s (%s)\n", item.File, item.Reason))
	}
	b.WriteString("   Fix: ln -sf ~/.claude/<file> ~/.claude-personal/<file>\n")
	b.WriteString("\n")
}

func formatSessionResume(b *strings.Builder, r *SessionResume) {
	if r == nil || r.Content == "" {
		return
	}
	b.WriteString("Session resumed:\n")
	// Indent each line of the handoff content
	for _, line := range strings.Split(strings.TrimSpace(r.Content), "\n") {
		b.WriteString(fmt.Sprintf("   %s\n", line))
	}
	b.WriteString("\n")
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
