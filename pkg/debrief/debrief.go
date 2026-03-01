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

// HealthData captures session health indicators.
type HealthData struct {
	Checkpoint     string `json:"checkpoint"`
	FrameCollapse  string `json:"frame_collapse"`
	DiscoveredWork string `json:"discovered_work"`
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
	Health       HealthData `json:"health"`
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

	// Session Health
	b.WriteString("## Session Health\n\n")
	b.WriteString(fmt.Sprintf("- **Checkpoint discipline:** %s\n", data.Health.Checkpoint))
	b.WriteString(fmt.Sprintf("- **Frame collapse:** %s\n", data.Health.FrameCollapse))
	b.WriteString(fmt.Sprintf("- **Discovered work:** %s\n", data.Health.DiscoveredWork))

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

// CollectWhatHappened summarizes session events into human-readable lines.
// Deduplicates by beads_id and limits detail to keep output readable.
func CollectWhatHappened(events []SessionEvent) []string {
	if len(events) == 0 {
		return nil
	}

	var lines []string
	completionSet := make(map[string]string) // beadsID -> skill
	spawnCount := 0
	abandonSet := make(map[string]string) // beadsID -> reason

	for _, e := range events {
		switch e.Type {
		case "agent.completed":
			beadsID, _ := e.Data["beads_id"].(string)
			skill, _ := e.Data["skill"].(string)
			if beadsID != "" {
				if _, exists := completionSet[beadsID]; !exists || skill != "" {
					completionSet[beadsID] = skill
				}
			}
		case "session.spawned":
			spawnCount++
		case "agent.abandoned":
			beadsID, _ := e.Data["beads_id"].(string)
			reason, _ := e.Data["reason"].(string)
			if beadsID != "" {
				if _, exists := abandonSet[beadsID]; !exists {
					abandonSet[beadsID] = reason
				}
			}
		}
	}

	if len(completionSet) > 0 {
		lines = append(lines, formatAgentSummary("Completed", completionSet))
	}
	if spawnCount > 0 {
		lines = append(lines, fmt.Sprintf("Spawned %d agent(s)", spawnCount))
	}
	if len(abandonSet) > 0 {
		lines = append(lines, formatAgentSummary("Abandoned", abandonSet))
	}

	return lines
}

// formatAgentSummary formats a map of beadsID->skill/reason into a concise summary.
// Shows count and up to maxDetail individual items.
func formatAgentSummary(verb string, agents map[string]string) string {
	const maxDetail = 5
	count := len(agents)

	var details []string
	for id, extra := range agents {
		if len(details) >= maxDetail {
			break
		}
		if extra != "" {
			details = append(details, fmt.Sprintf("%s (%s)", id, extra))
		} else {
			details = append(details, id)
		}
	}

	if count <= maxDetail {
		return fmt.Sprintf("%s %d: %s", verb, count, strings.Join(details, ", "))
	}
	return fmt.Sprintf("%s %d: %s, ... (+%d more)", verb, count, strings.Join(details, ", "), count-maxDetail)
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
