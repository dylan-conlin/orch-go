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

	// Ground-truth metrics from git
	NetLinesAdded   int `json:"net_lines_added,omitempty"`
	NetLinesRemoved int `json:"net_lines_removed,omitempty"`
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
	Total                 int              `json:"total"`
	Synthesis             int              `json:"synthesis"`
	Stale                 int              `json:"stale"`
	Promote               int              `json:"promote"`
	Drift                 int              `json:"drift"`
	Agreements            int              `json:"agreements"`
	TopClusters           []ReflectCluster `json:"top_clusters,omitempty"`
	Age                   string           `json:"age,omitempty"`                    // human-readable age like "2h ago"
	SessionOrphans        int              `json:"session_orphans,omitempty"`        // unlinked investigations from last session
	SessionInvestigations int              `json:"session_investigations,omitempty"` // total investigations from last session
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
	Level  string `json:"level"` // "green", "yellow", "red"
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

// PromotionCandidate represents a converged thread ready for promotion.
type PromotionCandidate struct {
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Updated    string `json:"updated"`
	EntryCount int    `json:"entry_count"`
}

// OrientationData holds all data needed to render session orientation.
// The thinking surface (FormatOrientation) renders: threads, briefs, tensions.
// Operational sections (FormatHealth) render: throughput, changelog, models, health, daemon, divergence, etc.
type OrientationData struct {
	// --- Thinking surface (rendered by FormatOrientation) ---
	ActiveThreads      []ActiveThread       `json:"active_threads,omitempty"`
	PromotionReady     []PromotionCandidate `json:"promotion_ready,omitempty"`
	RecentBriefs     []RecentBrief   `json:"recent_briefs,omitempty"`
	UnreadBriefCount int             `json:"unread_brief_count"`
	DigestSummary    *DigestSummary  `json:"digest_summary,omitempty"`
	ClaimEdges       string          `json:"claim_edges,omitempty"` // Pre-formatted claim edges text (filtered to thread-relevant)
	ReadyIssues      []ReadyIssue    `json:"ready_issues,omitempty"`
	ActivePlans      []PlanSummary   `json:"active_plans,omitempty"`
	FocusGoal        string          `json:"focus_goal,omitempty"`
	PreviousSession  *DebriefSummary `json:"previous_session,omitempty"`

	// --- Context (rendered by FormatOrientation) ---
	SessionResume *SessionResume    `json:"session_resume,omitempty"`
	ConfigDrift   []ConfigDriftItem `json:"config_drift,omitempty"`
	UsageWarning  *UsageWarning     `json:"usage_warning,omitempty"`

	// --- Operational (rendered by FormatHealth, not FormatOrientation) ---
	Throughput        Throughput          `json:"throughput"`
	RelevantModels    []ModelFreshness    `json:"relevant_models,omitempty"`
	StaleModels       []ModelFreshness    `json:"stale_models,omitempty"`
	HealthSummary     *HealthSummary      `json:"health_summary,omitempty"`
	DaemonHealth      *DaemonHealthView   `json:"daemon_health,omitempty"`
	Changelog         []ChangelogEntry    `json:"changelog,omitempty"`
	ReflectSummary    *ReflectSummary     `json:"reflect_summary,omitempty"`
	DivergenceAlerts  []DivergenceAlert   `json:"divergence_alerts,omitempty"`
	ExploreCandidates []ExploreCandidate  `json:"explore_candidates,omitempty"`
	AdoptionDrift     []AdoptionDriftItem `json:"adoption_drift,omitempty"`
}

// AdoptionDriftItem surfaces a compositional signal that has drifted below its target.
type AdoptionDriftItem struct {
	Signal    string  `json:"signal"`
	RatePct   float64 `json:"rate_pct"`
	TargetPct float64 `json:"target_pct"`
	Level     string  `json:"level"` // "drift" or "critical"
}

// ExploreCandidate is a recommendation for `orch spawn --explore investigation "..."`.
// Each candidate aggregates one or more signals into a concrete explore question.
type ExploreCandidate struct {
	Question string  `json:"question"` // The explore question to spawn
	Signal   string  `json:"signal"`   // Source signal type
	Score    float64 `json:"score"`    // Urgency score (higher = more urgent)
	Reason   string  `json:"reason"`   // Why this is worth exploring
}

// ComputeThroughput aggregates events within the given day window.
// If projectPrefix is non-empty, only events whose data.beads_id starts
// with that prefix are counted (scoping metrics to the current project).
func ComputeThroughput(events []Event, now time.Time, days int, projectPrefix string) Throughput {
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
		if projectPrefix != "" && !eventMatchesProject(e, projectPrefix) {
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

// eventMatchesProject checks if an event's beads_id starts with the given project prefix.
func eventMatchesProject(e Event, prefix string) bool {
	if e.Data == nil {
		return false
	}
	beadsID, ok := e.Data["beads_id"].(string)
	if !ok || beadsID == "" {
		return false
	}
	return strings.HasPrefix(beadsID, prefix+"-")
}

// FormatOrientation renders the thinking surface: threads, briefs, tensions.
// Operational sections (throughput, health, models, daemon, etc.) are in FormatHealth.
func FormatOrientation(data *OrientationData) string {
	var b strings.Builder

	b.WriteString("== SESSION ORIENTATION ==\n\n")

	// Context (surface problems and resume state first)
	formatSessionResume(&b, data.SessionResume)
	formatConfigDrift(&b, data.ConfigDrift)
	formatUsageWarning(&b, data.UsageWarning)

	// Element 1: Threads (primary frame)
	formatActiveThreads(&b, data.ActiveThreads)

	// Element 1b: Promotion-ready threads (converged but not yet artifacts)
	formatPromotionReady(&b, data.PromotionReady)

	// Element 2: Recent Briefs (what was learned)
	b.WriteString(FormatRecentBriefs(data.RecentBriefs, data.UnreadBriefCount))

	// Element 3: Active Tensions (filtered knowledge edges)
	if data.ClaimEdges != "" {
		b.WriteString(data.ClaimEdges)
	}

	return b.String()
}

// FormatHealth renders operational sections moved out of the thinking surface.
// Includes: throughput, changelog, models, health summary, daemon health, divergence,
// adoption drift, explore candidates, reflection suggestions.
func FormatHealth(data *OrientationData) string {
	var b strings.Builder

	b.WriteString("== OPERATIONAL HEALTH ==\n\n")

	// Throughput
	formatThroughput(&b, &data.Throughput)

	// Changelog since last session
	sinceDate := ""
	if data.PreviousSession != nil {
		sinceDate = data.PreviousSession.Date
	}
	b.WriteString(FormatChangelog(data.Changelog, sinceDate))

	// Divergence alerts
	b.WriteString(FormatDivergenceAlerts(data.DivergenceAlerts))

	// Previous session
	b.WriteString(FormatPreviousSession(data.PreviousSession))

	// Relevant models
	formatRelevantModels(&b, data.RelevantModels)

	// Stale models
	formatStaleModels(&b, data.StaleModels)

	// Health summary
	formatHealthSummary(&b, data.HealthSummary)

	// Adoption drift
	formatAdoptionDrift(&b, data.AdoptionDrift)

	// Daemon health
	formatDaemonHealth(&b, data.DaemonHealth)

	// Explore candidates
	formatExploreCandidates(&b, data.ExploreCandidates)

	// Reflection suggestions
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
	if tp.NetLinesAdded > 0 || tp.NetLinesRemoved > 0 {
		b.WriteString(fmt.Sprintf("   Net lines: %+d\n", tp.NetLinesAdded-tp.NetLinesRemoved))
	}
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
		if plan.Progress != "" {
			b.WriteString(fmt.Sprintf(" (%s)", plan.Progress))
		}
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

func formatPromotionReady(b *strings.Builder, candidates []PromotionCandidate) {
	if len(candidates) == 0 {
		return
	}
	b.WriteString("Ready to promote:\n")
	for _, c := range candidates {
		b.WriteString(fmt.Sprintf("   - %s (converged %s, %d entries)\n", c.Title, c.Updated, c.EntryCount))
		b.WriteString(fmt.Sprintf("     orch thread promote %s --as model|decision\n", c.Slug))
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

func formatAdoptionDrift(b *strings.Builder, items []AdoptionDriftItem) {
	if len(items) == 0 {
		return
	}
	b.WriteString("Adoption drift:\n")
	for _, item := range items {
		icon := "[!]"
		if item.Level == "critical" {
			icon = "[!!!]"
		}
		b.WriteString(fmt.Sprintf("   %s %s: %.0f%% (target %.0f%%)\n",
			icon, item.Signal, item.RatePct, item.TargetPct))
	}
	b.WriteString("   Run: orch harness adoption\n")
	b.WriteString("\n")
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
	if r.SessionOrphans > 0 {
		b.WriteString(fmt.Sprintf("   Session orphans: %d unlinked investigations (of %d produced)\n", r.SessionOrphans, r.SessionInvestigations))
	} else if r.SessionInvestigations > 0 {
		b.WriteString(fmt.Sprintf("   Session orphans: 0 (all %d investigations linked)\n", r.SessionInvestigations))
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

func formatExploreCandidates(b *strings.Builder, candidates []ExploreCandidate) {
	if len(candidates) == 0 {
		return
	}
	b.WriteString("Explore candidates:\n")
	for _, c := range candidates {
		b.WriteString(fmt.Sprintf("   [%s] %s\n", c.Signal, c.Question))
		b.WriteString(fmt.Sprintf("     %s\n", c.Reason))
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
