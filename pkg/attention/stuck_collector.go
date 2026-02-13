package attention

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StuckCollector implements the Collector interface for agents that have been running
// for an extended period (>2h) without progress. These are Authority signals that
// require human intervention.
type StuckCollector struct {
	client           *http.Client
	apiURL           string
	stuckThresholdH  float64 // Hours after which an agent is considered stuck (default: 2)
}

// NewStuckCollector creates a new StuckCollector with the given HTTP client and API URL.
// apiURL should be the base URL of the orch server (e.g., "http://localhost:3348").
// stuckThresholdH is the number of hours after which an active agent is considered stuck (default: 2).
func NewStuckCollector(client *http.Client, apiURL string, stuckThresholdH float64) *StuckCollector {
	if stuckThresholdH <= 0 {
		stuckThresholdH = 2.0 // Default: 2 hours
	}
	return &StuckCollector{
		client:          client,
		apiURL:          apiURL,
		stuckThresholdH: stuckThresholdH,
	}
}

// StuckAgentItem represents an agent from the /api/agents response.
// This is a subset of the full response - only fields needed for stuck detection.
type StuckAgentItem struct {
	ID             string `json:"id"`
	BeadsID        string `json:"beads_id"`
	BeadsTitle     string `json:"beads_title"`
	Status         string `json:"status"` // active, idle, dead, completed, awaiting-cleanup
	Phase          string `json:"phase"`
	Task           string `json:"task"`
	Project        string `json:"project"`
	Skill          string `json:"skill"`
	IsStalled      bool   `json:"is_stalled"`       // True if same phase for 15+ min
	SpawnedAt      string `json:"spawned_at"`       // ISO 8601 timestamp
	UpdatedAt      string `json:"updated_at"`       // ISO 8601 timestamp
	LastActivityAt string `json:"last_activity_at"` // ISO 8601 timestamp
	Runtime        string `json:"runtime"`
}

// Collect gathers attention items for agents that appear stuck (running >2h).
// These are Authority signals requiring human intervention.
func (c *StuckCollector) Collect(role string) ([]AttentionItem, error) {
	// Query all agents (since=all to include historical)
	url := fmt.Sprintf("%s/api/agents?since=all", c.apiURL)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("agents API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents response: %w", err)
	}

	// Parse response - agents API returns a raw array
	var agents []StuckAgentItem
	if err := json.Unmarshal(body, &agents); err != nil {
		return nil, fmt.Errorf("failed to decode agents response: %w", err)
	}

	// Filter for stuck agents
	items := make([]AttentionItem, 0)
	now := time.Now()
	stuckThreshold := time.Duration(c.stuckThresholdH * float64(time.Hour))

	for _, agent := range agents {
		// Only consider active agents (not dead, completed, or awaiting-cleanup)
		if agent.Status != "active" && agent.Status != "idle" {
			continue
		}

		// Skip agents without beads ID
		if agent.BeadsID == "" {
			continue
		}

		// Parse spawn time
		spawnedAt, err := time.Parse(time.RFC3339, agent.SpawnedAt)
		if err != nil {
			// Can't determine age, skip
			continue
		}

		// Check if running longer than threshold
		runningDuration := now.Sub(spawnedAt)
		if runningDuration < stuckThreshold {
			continue
		}

		// Check last activity - if recent activity, might not be stuck
		lastActivityAt := spawnedAt // Default to spawn time
		if agent.LastActivityAt != "" {
			if parsed, err := time.Parse(time.RFC3339, agent.LastActivityAt); err == nil {
				lastActivityAt = parsed
			}
		}
		inactivityDuration := now.Sub(lastActivityAt)

		// Agent is stuck if:
		// 1. Running > threshold AND
		// 2. No activity in last 30 minutes (or marked as stalled)
		if inactivityDuration < 30*time.Minute && !agent.IsStalled {
			continue
		}

		priority := calculateStuckPriority(agent, runningDuration, role)

		summary := fmt.Sprintf("Stuck %s: %s (running %s, last activity %s ago)",
			agent.Status,
			truncateStr(agent.Task, 40),
			formatDuration(runningDuration),
			formatDuration(inactivityDuration))

		if agent.Task == "" && agent.BeadsTitle != "" {
			summary = fmt.Sprintf("Stuck %s: %s (running %s, last activity %s ago)",
				agent.Status,
				truncateStr(agent.BeadsTitle, 40),
				formatDuration(runningDuration),
				formatDuration(inactivityDuration))
		}

		item := AttentionItem{
			ID:          fmt.Sprintf("stuck-%s", agent.BeadsID),
			Source:      "agent",
			Concern:     Authority, // Requires human intervention
			Signal:      "stuck",
			Subject:     agent.BeadsID,
			Summary:     summary,
			Priority:    priority,
			Role:        role,
			ActionHint:  fmt.Sprintf("tmux attach -t %s # Review and intervene", agent.ID),
			CollectedAt: now,
			Metadata: map[string]any{
				"agent_id":            agent.ID,
				"phase":               agent.Phase,
				"project":             agent.Project,
				"skill":               agent.Skill,
				"spawned_at":          agent.SpawnedAt,
				"last_activity_at":    agent.LastActivityAt,
				"running_hours":       runningDuration.Hours(),
				"inactivity_minutes":  inactivityDuration.Minutes(),
				"is_stalled":          agent.IsStalled,
			},
		}
		items = append(items, item)
	}

	return items, nil
}

// calculateStuckPriority determines priority based on role and how long stuck.
// Lower numbers = higher priority.
func calculateStuckPriority(agent StuckAgentItem, runningDuration time.Duration, role string) int {
	// Base priority for stuck agents - high because this requires intervention
	basePriority := 20

	// Adjust based on how long running (longer = higher priority)
	hours := runningDuration.Hours()
	if hours > 8 {
		basePriority -= 15 // Very stuck (> 8h)
	} else if hours > 4 {
		basePriority -= 10 // Stuck (> 4h)
	} else if hours > 2 {
		basePriority -= 5 // Just crossed threshold
	}

	// Boost priority if already marked as stalled
	if agent.IsStalled {
		basePriority -= 5
	}

	// Role-aware adjustments
	switch role {
	case "human":
		// Humans are the primary audience for stuck agents
		return basePriority

	case "orchestrator":
		// Orchestrators should know about stuck work
		return basePriority - 5

	case "daemon":
		// Daemon can't auto-fix stuck agents
		return basePriority + 50

	default:
		return basePriority
	}
}

// truncateStr shortens a string to maxLen, adding "..." if truncated.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours >= 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
