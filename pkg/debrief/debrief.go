// Package debrief provides session debrief generation and auto-population.
//
// A debrief is a durable artifact at .kb/sessions/YYYY-MM-DD-debrief.md that
// captures what happened during an orchestrator session. It auto-populates from
// events.jsonl, git log, beads, and other sources, then allows inline overrides
// via --changed and --next flags.
package debrief

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// SessionEvent is a simplified event from events.jsonl for debrief consumption.
type SessionEvent struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// InFlightIssue represents an in-progress beads issue.
type InFlightIssue struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// DebriefData holds all data needed to render a session debrief.
type DebriefData struct {
	Date         string   `json:"date"`
	Duration     string   `json:"duration,omitempty"`
	Focus        string   `json:"focus"`
	WhatHappened []string `json:"what_happened,omitempty"`
	WhatChanged  []string `json:"what_changed,omitempty"`
	InFlight     []string `json:"in_flight,omitempty"`
	WhatsNext    []string `json:"whats_next,omitempty"`
}

// DebriefFilePath returns the file path for a debrief on the given date.
func DebriefFilePath(projectDir, date string) string {
	return filepath.Join(projectDir, ".kb", "sessions", date+"-debrief.md")
}

// RenderDebrief renders a DebriefData into markdown matching the template format.
func RenderDebrief(data *DebriefData) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("# Session Debrief: %s\n\n", data.Date))
	b.WriteString(fmt.Sprintf("**Date:** %s\n", data.Date))
	if data.Duration != "" {
		b.WriteString(fmt.Sprintf("**Duration:** %s\n", data.Duration))
	}
	b.WriteString(fmt.Sprintf("**Focus:** %s\n", data.Focus))
	b.WriteString("\n---\n\n")

	// What Happened
	b.WriteString("## What Happened\n\n")
	writeList(&b, data.WhatHappened)

	// What Changed
	b.WriteString("## What Changed\n\n")
	writeList(&b, data.WhatChanged)

	// What's In Flight
	b.WriteString("## What's In Flight\n\n")
	writeList(&b, data.InFlight)

	// What's Next
	b.WriteString("## What's Next\n\n")
	writeListNumbered(&b, data.WhatsNext)

	return b.String()
}

func writeList(b *strings.Builder, items []string) {
	if len(items) == 0 {
		b.WriteString("- (none)\n")
	} else {
		for _, item := range items {
			b.WriteString(fmt.Sprintf("- %s\n", item))
		}
	}
	b.WriteString("\n")
}

func writeListNumbered(b *strings.Builder, items []string) {
	if len(items) == 0 {
		b.WriteString("- (none)\n")
	} else {
		for i, item := range items {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
		}
	}
	b.WriteString("\n")
}

// CollectWhatHappened produces one line per event with skill, title, and context.
// Deduplicates completions and abandonments by beads_id.
func CollectWhatHappened(events []SessionEvent) []string {
	if len(events) == 0 {
		return nil
	}

	var lines []string
	seen := make(map[string]bool)

	for _, e := range events {
		switch e.Type {
		case "agent.completed":
			beadsID, _ := e.Data["beads_id"].(string)
			if beadsID != "" && seen[beadsID] {
				continue
			}
			if beadsID != "" {
				seen[beadsID] = true
			}
			skill, _ := e.Data["skill"].(string)
			reason, _ := e.Data["reason"].(string)
			lines = append(lines, formatCompletionLine(beadsID, skill, reason))

		case "session.spawned":
			beadsID, _ := e.Data["beads_id"].(string)
			if beadsID != "" && seen[beadsID] {
				continue
			}
			if beadsID != "" {
				seen[beadsID] = true
			}
			skill, _ := e.Data["skill"].(string)
			task, _ := e.Data["task"].(string)
			lines = append(lines, formatSpawnLine(skill, task))

		case "agent.abandoned":
			beadsID, _ := e.Data["beads_id"].(string)
			if beadsID != "" && seen[beadsID] {
				continue
			}
			if beadsID != "" {
				seen[beadsID] = true
			}
			reason, _ := e.Data["reason"].(string)
			lines = append(lines, formatAbandonLine(beadsID, reason))
		}
	}

	return lines
}

// formatCompletionLine renders a single completion event.
// Format: "Completed: `skill` (beads_id) — reason summary"
func formatCompletionLine(beadsID, skill, reason string) string {
	var parts []string
	parts = append(parts, "Completed:")
	if skill != "" {
		parts = append(parts, fmt.Sprintf("`%s`", skill))
	}
	if beadsID != "" {
		parts = append(parts, fmt.Sprintf("(%s)", beadsID))
	}
	line := strings.Join(parts, " ")
	if reason != "" {
		line += " — " + truncate(reason, 120)
	}
	return line
}

// formatSpawnLine renders a single spawn event.
// Format: "Spawned: `skill` — task description"
func formatSpawnLine(skill, task string) string {
	var parts []string
	parts = append(parts, "Spawned:")
	if skill != "" {
		parts = append(parts, fmt.Sprintf("`%s`", skill))
	}
	line := strings.Join(parts, " ")
	if task != "" {
		line += " — " + truncate(task, 120)
	}
	return line
}

// formatAbandonLine renders a single abandon event.
// Format: "Abandoned: beads_id — reason"
func formatAbandonLine(beadsID, reason string) string {
	line := "Abandoned:"
	if beadsID != "" {
		line += " " + beadsID
	}
	if reason != "" {
		line += " — " + reason
	}
	return line
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// CollectWhatChanged extracts the reason/explain-back text from agent.completed
// events. These describe what each agent actually accomplished.
// Deduplicates by beads_id.
func CollectWhatChanged(events []SessionEvent) []string {
	if len(events) == 0 {
		return nil
	}

	var lines []string
	seen := make(map[string]bool)

	for _, e := range events {
		if e.Type != "agent.completed" {
			continue
		}
		beadsID, _ := e.Data["beads_id"].(string)
		if beadsID != "" && seen[beadsID] {
			continue
		}
		if beadsID != "" {
			seen[beadsID] = true
		}
		reason, _ := e.Data["reason"].(string)
		if reason == "" {
			continue
		}
		lines = append(lines, reason)
	}

	return lines
}

// CollectInFlight formats in-flight issues into debrief lines.
func CollectInFlight(issues []InFlightIssue) []string {
	if len(issues) == 0 {
		return nil
	}

	var lines []string
	for _, issue := range issues {
		lines = append(lines, fmt.Sprintf("%s: %s (%s)", issue.ID, issue.Title, issue.Status))
	}
	return lines
}

// FilterEventsToday returns only events from today (since midnight local time).
func FilterEventsToday(events []SessionEvent, now time.Time) []SessionEvent {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cutoff := today.Unix()

	var filtered []SessionEvent
	for _, e := range events {
		if e.Timestamp >= cutoff {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// MaxReasonableDuration is the maximum session duration that makes sense for a
// daily debrief. Durations exceeding this are from stale session data and should
// be omitted rather than showing nonsensical values like "~954h".
const MaxReasonableDuration = 24 * time.Hour

// FormatDuration formats a session duration for debrief display.
// Returns empty string for zero, negative, sub-minute, or stale (>24h) durations.
func FormatDuration(d time.Duration) string {
	if d <= 0 || d >= MaxReasonableDuration {
		return ""
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("~%dh", hours)
	}
	mins := int(d.Minutes())
	if mins > 0 {
		return fmt.Sprintf("~%dm", mins)
	}
	return ""
}

// ParseMultiValue splits a semicolon-delimited string into trimmed values.
// If no semicolons, returns the whole string as a single item.
// Returns nil for empty input.
func ParseMultiValue(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	if !strings.Contains(input, ";") {
		return []string{input}
	}

	parts := strings.Split(input, ";")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
