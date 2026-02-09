// Package verify provides attempt tracking for agent work on beads issues.
// This file tracks fix attempts to surface retry patterns and identify flaky issues.

package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// FixAttemptStats tracks how many times an issue has been worked on.
// This helps surface retry patterns to identify flaky issues that may need
// reliability-testing rather than repeated debugging attempts.
type FixAttemptStats struct {
	BeadsID        string    // The beads issue ID
	SpawnCount     int       // Number of times agents were spawned for this issue
	AbandonedCount int       // Number of times agents were abandoned
	CompletedCount int       // Number of times agents completed successfully
	LastAttemptAt  time.Time // When the most recent attempt started
	LastOutcome    string    // "spawned", "completed", "abandoned", or ""
	Skills         []string  // Skills used (may indicate strategy changes)
}

// IsRetryPattern returns true if the stats suggest a flaky issue.
// A retry pattern is when:
// - Multiple spawn attempts exist (indicating respawns after failures)
// - At least one abandon occurred (explicit failure)
// This signals the issue may need reliability-testing instead of more attempts.
func (s *FixAttemptStats) IsRetryPattern() bool {
	// More than one spawn with any abandons suggests retry pattern
	return s.SpawnCount > 1 && s.AbandonedCount > 0
}

// IsPersistentFailure returns true if multiple attempts have failed without success.
func (s *FixAttemptStats) IsPersistentFailure() bool {
	// Multiple spawns with no completions and multiple abandons
	return s.SpawnCount >= 2 && s.CompletedCount == 0 && s.AbandonedCount >= 2
}

// SuggestedAction returns a recommendation based on the attempt pattern.
func (s *FixAttemptStats) SuggestedAction() string {
	if s.IsPersistentFailure() {
		return "reliability-testing"
	}
	if s.IsRetryPattern() {
		return "investigate-root-cause"
	}
	return ""
}

// WarningLevel returns the severity level for display purposes.
// Returns: "critical" (persistent failure), "warning" (retry pattern), "" (normal)
func (s *FixAttemptStats) WarningLevel() string {
	if s.IsPersistentFailure() {
		return "critical"
	}
	if s.IsRetryPattern() {
		return "warning"
	}
	return ""
}

// FixAttemptEvent represents a parsed event from events.jsonl
type FixAttemptEvent struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// GetFixAttemptStats retrieves attempt statistics for a beads issue.
// It scans the events.jsonl file to count spawns, completions, and abandonments.
func GetFixAttemptStats(beadsID string) (*FixAttemptStats, error) {
	return GetFixAttemptStatsFromPath(beadsID, DefaultEventsPath())
}

// GetFixAttemptStatsFromPath retrieves attempt statistics from a specific events file.
// This is useful for testing with a custom events file.
func GetFixAttemptStatsFromPath(beadsID, eventsPath string) (*FixAttemptStats, error) {
	stats := &FixAttemptStats{
		BeadsID: beadsID,
		Skills:  make([]string, 0),
	}

	// Track unique skills
	seenSkills := make(map[string]bool)

	// Scan for relevant events
	err := events.ReadCompactedJSONL(eventsPath, func(line string) error {
		var event FixAttemptEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil // Skip malformed lines
		}

		// Check if this event is for our beads ID
		eventBeadsID := extractBeadsIDFromEvent(event)
		if eventBeadsID != beadsID {
			return nil
		}

		// Update stats based on event type
		switch event.Type {
		case "session.spawned":
			stats.SpawnCount++
			stats.LastOutcome = "spawned"
			if skill := extractSkillFromEvent(event); skill != "" && !seenSkills[skill] {
				seenSkills[skill] = true
				stats.Skills = append(stats.Skills, skill)
			}

		case "agent.completed":
			stats.CompletedCount++
			stats.LastOutcome = "completed"

		case "agent.abandoned":
			stats.AbandonedCount++
			stats.LastOutcome = "abandoned"
		}

		// Track latest timestamp
		eventTime := time.Unix(event.Timestamp, 0)
		if eventTime.After(stats.LastAttemptAt) {
			stats.LastAttemptAt = eventTime
		}

		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			// No events file yet - return empty stats
			return stats, nil
		}
		return nil, err
	}

	return stats, nil
}

// GetAllRetryPatterns scans all events to find issues with retry patterns.
// Returns a slice of stats for issues that show retry patterns, sorted by severity.
func GetAllRetryPatterns() ([]*FixAttemptStats, error) {
	return GetAllRetryPatternsFromPath(DefaultEventsPath())
}

// GetAllRetryPatternsFromPath scans events from a specific file.
func GetAllRetryPatternsFromPath(eventsPath string) ([]*FixAttemptStats, error) {
	// Build stats for all beads IDs
	statsMap := make(map[string]*FixAttemptStats)

	err := events.ReadCompactedJSONL(eventsPath, func(line string) error {
		var event FixAttemptEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil
		}

		beadsID := extractBeadsIDFromEvent(event)
		if beadsID == "" {
			return nil
		}

		stats, exists := statsMap[beadsID]
		if !exists {
			stats = &FixAttemptStats{
				BeadsID: beadsID,
				Skills:  make([]string, 0),
			}
			statsMap[beadsID] = stats
		}

		switch event.Type {
		case "session.spawned":
			stats.SpawnCount++
			stats.LastOutcome = "spawned"
			if skill := extractSkillFromEvent(event); skill != "" {
				// Simple dedup check
				found := false
				for _, s := range stats.Skills {
					if s == skill {
						found = true
						break
					}
				}
				if !found {
					stats.Skills = append(stats.Skills, skill)
				}
			}
		case "agent.completed":
			stats.CompletedCount++
			stats.LastOutcome = "completed"
		case "agent.abandoned":
			stats.AbandonedCount++
			stats.LastOutcome = "abandoned"
		}

		eventTime := time.Unix(event.Timestamp, 0)
		if eventTime.After(stats.LastAttemptAt) {
			stats.LastAttemptAt = eventTime
		}

		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No events file
		}
		return nil, err
	}

	// Filter to only retry patterns and sort by severity
	result := make([]*FixAttemptStats, 0)
	for _, stats := range statsMap {
		if stats.IsRetryPattern() {
			result = append(result, stats)
		}
	}

	// Sort: persistent failures first, then by spawn count
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsPersistentFailure() != result[j].IsPersistentFailure() {
			return result[i].IsPersistentFailure()
		}
		return result[i].SpawnCount > result[j].SpawnCount
	})

	return result, nil
}

// DefaultEventsPath returns the default path to events.jsonl.
func DefaultEventsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/events.jsonl"
	}
	return filepath.Join(home, ".orch", "events.jsonl")
}

// extractBeadsIDFromEvent extracts the beads ID from an event's data.
func extractBeadsIDFromEvent(event FixAttemptEvent) string {
	if event.Data == nil {
		return ""
	}
	if beadsID, ok := event.Data["beads_id"].(string); ok && beadsID != "" {
		return beadsID
	}
	return ""
}

// extractSkillFromEvent extracts the skill name from an event's data.
func extractSkillFromEvent(event FixAttemptEvent) string {
	if event.Data == nil {
		return ""
	}
	if skill, ok := event.Data["skill"].(string); ok {
		return skill
	}
	return ""
}

// FormatRetryWarning formats a warning message for retry patterns.
// Used when spawning an agent for an issue with retry history.
func FormatRetryWarning(stats *FixAttemptStats) string {
	if stats == nil || !stats.IsRetryPattern() {
		return ""
	}

	var sb strings.Builder

	level := stats.WarningLevel()
	if level == "critical" {
		sb.WriteString("🚨 PERSISTENT FAILURE PATTERN DETECTED\n")
		sb.WriteString("┌─────────────────────────────────────────────────────────────┐\n")
		sb.WriteString("│  This issue has failed multiple times without success.      │\n")
		sb.WriteString("│  Consider using 'reliability-testing' skill instead.        │\n")
		sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")
	} else {
		sb.WriteString("⚠️  RETRY PATTERN DETECTED\n")
		sb.WriteString("┌─────────────────────────────────────────────────────────────┐\n")
		sb.WriteString("│  This issue has been respawned after previous failure(s).   │\n")
		sb.WriteString("│  Consider investigating root cause before more attempts.    │\n")
		sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")
	}

	sb.WriteString("\n")
	sb.WriteString("History:\n")
	sb.WriteString("  Spawns:     " + formatCount(stats.SpawnCount) + "\n")
	sb.WriteString("  Abandoned:  " + formatCount(stats.AbandonedCount) + "\n")
	sb.WriteString("  Completed:  " + formatCount(stats.CompletedCount) + "\n")

	if len(stats.Skills) > 0 {
		sb.WriteString("  Skills:     " + strings.Join(stats.Skills, ", ") + "\n")
	}

	if action := stats.SuggestedAction(); action != "" {
		sb.WriteString("\n")
		sb.WriteString("Suggested: Use skill '" + action + "' to address underlying issue\n")
	}

	return sb.String()
}

func formatCount(n int) string {
	if n == 0 {
		return "0"
	}
	return strings.Repeat("█", n) + " " + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
