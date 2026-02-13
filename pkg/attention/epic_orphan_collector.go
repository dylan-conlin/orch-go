// Package attention provides types and interfaces for the composable attention architecture.
package attention

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EpicOrphanCollector collects attention signals for orphaned children of force-closed epics.
// It reads from the events.jsonl file to find epic.orphaned events.
type EpicOrphanCollector struct {
	eventsPath string
}

// NewEpicOrphanCollector creates a new collector for epic orphan signals.
func NewEpicOrphanCollector() *EpicOrphanCollector {
	homeDir, _ := os.UserHomeDir()
	return &EpicOrphanCollector{
		eventsPath: filepath.Join(homeDir, ".orch", "events.jsonl"),
	}
}

// epicOrphanedEvent represents an epic.orphaned event from events.jsonl.
type epicOrphanedEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Data      struct {
		EpicID           string   `json:"epic_id"`
		EpicTitle        string   `json:"epic_title"`
		OrphanedChildren []string `json:"orphaned_children"`
		Reason           string   `json:"reason"`
	} `json:"data"`
}

// Collect gathers attention items for orphaned epic children.
func (c *EpicOrphanCollector) Collect(role string) ([]AttentionItem, error) {
	// Read events file
	data, err := os.ReadFile(c.eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No events file yet
		}
		return nil, fmt.Errorf("failed to read events: %w", err)
	}

	// Parse JSONL and find recent epic.orphaned events
	var items []AttentionItem
	lines := strings.Split(string(data), "\n")
	
	// Only look at last 100 events for performance
	startIdx := 0
	if len(lines) > 100 {
		startIdx = len(lines) - 100
	}
	
	// Track which epic orphan events we've already processed (dedup by epic ID)
	seen := make(map[string]bool)
	
	for i := len(lines) - 1; i >= startIdx; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		
		var event epicOrphanedEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		
		if event.Type != "epic.orphaned" {
			continue
		}
		
		// Skip if we've already seen this epic (take most recent)
		if seen[event.Data.EpicID] {
			continue
		}
		seen[event.Data.EpicID] = true
		
		// Skip events older than 7 days
		eventTime := time.Unix(event.Timestamp, 0)
		if time.Since(eventTime) > 7*24*time.Hour {
			continue
		}
		
		// Calculate priority based on role
		priority := 50 // Default medium priority
		switch role {
		case "human":
			priority = 30 // Higher priority for humans - needs manual intervention
		case "orchestrator":
			priority = 40 // Medium-high for orchestrators
		case "daemon":
			priority = 60 // Lower for daemon - can't auto-fix this
		}
		
		item := AttentionItem{
			ID:          fmt.Sprintf("epic-orphan-%s", event.Data.EpicID),
			Source:      "epic-orphan",
			Concern:     Authority, // Requires human decision
			Signal:      "epic-orphaned",
			Subject:     event.Data.EpicID,
			Summary:     fmt.Sprintf("Epic %s closed with %d open children", event.Data.EpicID, len(event.Data.OrphanedChildren)),
			Priority:    priority,
			Role:        role,
			ActionHint:  fmt.Sprintf("Review orphaned children: %s", strings.Join(event.Data.OrphanedChildren, ", ")),
			CollectedAt: time.Now(),
			Metadata: map[string]any{
				"epic_id":           event.Data.EpicID,
				"epic_title":        event.Data.EpicTitle,
				"orphaned_children": event.Data.OrphanedChildren,
				"orphaned_at":       eventTime.Format(time.RFC3339),
			},
		}
		items = append(items, item)
	}
	
	return items, nil
}
