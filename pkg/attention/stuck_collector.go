package attention

import (
	"fmt"
	"net/http"
	"time"
)

// StuckCollector implements the Collector interface for agents that have been running
// for an extended period (>2h) without progress. These are Authority signals that
// require human intervention.
type StuckCollector struct {
	client          *http.Client
	apiURL          string
	stuckThresholdH float64 // Hours after which an agent is considered stuck (default: 2)

	agentSnapshot []AgentAPIItem
	snapshotErr   error
	useSnapshot   bool
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

// NewStuckCollectorWithSnapshot creates a collector that uses a pre-fetched shared snapshot.
// If snapshotErr is non-nil, Collect returns that error without making API requests.
func NewStuckCollectorWithSnapshot(agentSnapshot []AgentAPIItem, snapshotErr error, stuckThresholdH float64) *StuckCollector {
	if stuckThresholdH <= 0 {
		stuckThresholdH = 2.0
	}

	return &StuckCollector{
		stuckThresholdH: stuckThresholdH,
		agentSnapshot:   agentSnapshot,
		snapshotErr:     snapshotErr,
		useSnapshot:     true,
	}
}

// StuckAgentItem represents an agent from the /api/agents response.
// It aliases AgentAPIItem so stuck and verify collectors can share one snapshot.
type StuckAgentItem = AgentAPIItem

// Collect gathers attention items for agents that appear stuck (running >2h).
// These are Authority signals requiring human intervention.
func (c *StuckCollector) Collect(role string) ([]AttentionItem, error) {
	agents, err := c.resolveAgents()
	if err != nil {
		return nil, err
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
			Concern:     Authority,
			Signal:      "stuck",
			Subject:     agent.BeadsID,
			Summary:     summary,
			Priority:    priority,
			Role:        role,
			ActionHint:  fmt.Sprintf("tmux attach -t %s # Review and intervene", agent.ID),
			CollectedAt: now,
			Metadata: map[string]any{
				"agent_id":           agent.ID,
				"phase":              agent.Phase,
				"project":            agent.Project,
				"skill":              agent.Skill,
				"spawned_at":         agent.SpawnedAt,
				"last_activity_at":   agent.LastActivityAt,
				"running_hours":      runningDuration.Hours(),
				"inactivity_minutes": inactivityDuration.Minutes(),
				"is_stalled":         agent.IsStalled,
			},
		}
		items = append(items, item)
	}

	return items, nil
}

func (c *StuckCollector) resolveAgents() ([]AgentAPIItem, error) {
	if c.useSnapshot {
		if c.snapshotErr != nil {
			return nil, c.snapshotErr
		}
		return c.agentSnapshot, nil
	}

	return FetchAgentSnapshot(c.client, c.apiURL)
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
