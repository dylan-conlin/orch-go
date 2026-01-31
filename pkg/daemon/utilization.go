// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// UtilizationMetrics tracks the ratio of daemon-spawned vs manual-spawned agents.
// This surfaces where triage discipline is slipping (manual spawns bypassing daemon workflow).
type UtilizationMetrics struct {
	// TotalSpawns is the total number of spawns in the analysis window.
	TotalSpawns int `json:"total_spawns"`

	// DaemonSpawns is the number of spawns triggered by the daemon.
	DaemonSpawns int `json:"daemon_spawns"`

	// ManualSpawns is the number of spawns triggered manually (not by daemon).
	ManualSpawns int `json:"manual_spawns"`

	// DaemonSpawnRate is the percentage of spawns from daemon (0-100).
	// Higher is better - indicates triage workflow is being followed.
	DaemonSpawnRate float64 `json:"daemon_spawn_rate"`

	// TriageBypassed is the count of spawns that explicitly bypassed triage.
	TriageBypassed int `json:"triage_bypassed"`

	// TriageSlipRate is the percentage of spawns that bypassed triage (0-100).
	// Lower is better - high rate indicates discipline slippage.
	TriageSlipRate float64 `json:"triage_slip_rate"`

	// AutoCompletions is the count of daemon auto-completions.
	AutoCompletions int `json:"auto_completions"`

	// AnalysisPeriod describes the time window analyzed (e.g., "Last 7 days").
	AnalysisPeriod string `json:"analysis_period"`

	// DaysAnalyzed is the number of days in the analysis window.
	DaysAnalyzed int `json:"days_analyzed"`
}

// UtilizationEvent represents a parsed event from events.jsonl for utilization tracking.
type UtilizationEvent struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// GetUtilizationMetrics computes daemon utilization metrics from events.jsonl.
// The days parameter controls the analysis window (e.g., 7 for last 7 days).
func GetUtilizationMetrics(days int) (*UtilizationMetrics, error) {
	eventsPath := getEventsPath()

	events, err := parseUtilizationEvents(eventsPath)
	if err != nil {
		return nil, err
	}

	return computeUtilization(events, days), nil
}

// getEventsPath returns the path to events.jsonl.
func getEventsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/events.jsonl"
	}
	return filepath.Join(home, ".orch", "events.jsonl")
}

// parseUtilizationEvents reads events from events.jsonl.
func parseUtilizationEvents(path string) ([]UtilizationEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No events yet - return empty slice
			return []UtilizationEvent{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var events []UtilizationEvent
	scanner := bufio.NewScanner(file)
	// Increase buffer size for potentially long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event UtilizationEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip malformed lines
			continue
		}

		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// computeUtilization aggregates events into utilization metrics.
func computeUtilization(events []UtilizationEvent, days int) *UtilizationMetrics {
	metrics := &UtilizationMetrics{
		AnalysisPeriod: "Last " + formatDays(days),
		DaysAnalyzed:   days,
	}

	// Time window cutoff
	now := time.Now().Unix()
	cutoff := now - int64(days*86400)

	for _, event := range events {
		// Skip events outside the analysis window
		if event.Timestamp < cutoff {
			continue
		}

		switch event.Type {
		case "session.spawned":
			metrics.TotalSpawns++

			// Check if this is a daemon spawn by looking at spawn_source in data
			if data := event.Data; data != nil {
				if source, ok := data["spawn_source"].(string); ok && source == "daemon" {
					metrics.DaemonSpawns++
				}
			}

		case "daemon.spawn":
			// Direct daemon spawn event
			metrics.DaemonSpawns++

		case "session.auto_completed":
			metrics.AutoCompletions++

		case "spawn.triage_bypassed":
			metrics.TriageBypassed++
		}
	}

	// Safeguard: daemon spawns should not exceed total spawns
	// (can happen if spawn fails after daemon event is logged)
	if metrics.DaemonSpawns > metrics.TotalSpawns {
		metrics.DaemonSpawns = metrics.TotalSpawns
	}

	// Calculate manual spawns (total - daemon)
	metrics.ManualSpawns = metrics.TotalSpawns - metrics.DaemonSpawns

	// Calculate rates
	if metrics.TotalSpawns > 0 {
		metrics.DaemonSpawnRate = float64(metrics.DaemonSpawns) / float64(metrics.TotalSpawns) * 100

		// Triage slip rate: percentage of spawns that explicitly bypassed triage
		// Note: TriageBypassed can exceed TotalSpawns if some spawns failed after bypass event was logged
		// We cap at 100% for meaningful display
		metrics.TriageSlipRate = float64(metrics.TriageBypassed) / float64(metrics.TotalSpawns) * 100
		if metrics.TriageSlipRate > 100 {
			metrics.TriageSlipRate = 100
		}
	}

	return metrics
}

// formatDays returns a human-readable string for the days count.
func formatDays(days int) string {
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// GetRecentlyAbandonedIssues returns the beads IDs of issues that were abandoned
// within the last `hours` hours. This is used to clear the SpawnedIssueTracker
// so that abandoned issues can be respawned by the daemon.
//
// The issue is: when an agent is abandoned via `orch abandon`, the daemon's
// SpawnedIssueTracker still has the issue marked as "recently spawned" (6h TTL).
// This blocks the daemon from respawning the issue even after triage:ready is re-added.
//
// This function reads agent.abandoned events from events.jsonl and returns
// the beads IDs that were abandoned recently.
func GetRecentlyAbandonedIssues(hours int) ([]string, error) {
	eventsPath := getEventsPath()

	events, err := parseUtilizationEvents(eventsPath)
	if err != nil {
		return nil, err
	}

	return filterRecentlyAbandoned(events, hours), nil
}

// filterRecentlyAbandoned extracts beads IDs from recent agent.abandoned events.
func filterRecentlyAbandoned(events []UtilizationEvent, hours int) []string {
	// Time window cutoff (convert hours to seconds)
	now := time.Now().Unix()
	cutoff := now - int64(hours*3600)

	var abandonedIDs []string
	seen := make(map[string]bool) // Dedupe in case of multiple abandons for same issue

	for _, event := range events {
		// Only look at agent.abandoned events
		if event.Type != "agent.abandoned" {
			continue
		}

		// Skip events outside the time window
		if event.Timestamp < cutoff {
			continue
		}

		// Extract beads_id from event data
		if data := event.Data; data != nil {
			if beadsID, ok := data["beads_id"].(string); ok && beadsID != "" {
				if !seen[beadsID] {
					seen[beadsID] = true
					abandonedIDs = append(abandonedIDs, beadsID)
				}
			}
		}
	}

	return abandonedIDs
}
