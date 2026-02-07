package attention

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AgentCollector implements the Collector interface for agent-based attention signals.
// It queries the /api/agents endpoint and surfaces awaiting-cleanup agents as verify
// attention items. This surfaces completed agents needing 'orch complete' in the
// Work Graph tree with VERIFY badges.
type AgentCollector struct {
	client *http.Client
	apiURL string
}

// AgentAPIItem represents an agent from the /api/agents response.
// This is a subset of the full response - only fields needed for attention signals.
type AgentAPIItem struct {
	ID         string `json:"id"`
	BeadsID    string `json:"beads_id"`
	BeadsTitle string `json:"beads_title"`
	Status     string `json:"status"`
	Phase      string `json:"phase"`
	Task       string `json:"task"`
	Project    string `json:"project"`
	Skill      string `json:"skill"`
	UpdatedAt  string `json:"updated_at"`
}

// AgentAPIListResponse is the response structure used for test mocks.
// The actual /api/agents endpoint returns a raw array, but we keep this
// for backwards compatibility with existing tests.
type AgentAPIListResponse struct {
	Agents []AgentAPIItem `json:"agents"`
}

// NewAgentCollector creates a new AgentCollector with the given HTTP client and API URL.
// apiURL should be the base URL of the orch server (e.g., "http://localhost:3348").
func NewAgentCollector(client *http.Client, apiURL string) *AgentCollector {
	return &AgentCollector{
		client: client,
		apiURL: apiURL,
	}
}

// Collect gathers attention items for awaiting-cleanup agents.
// These are agents that have reported Phase: Complete but their session has died,
// indicating they need 'orch complete' to close out the work.
func (c *AgentCollector) Collect(role string) ([]AttentionItem, error) {
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

	// Parse response - try raw array first (actual API format), then wrapped format (test mocks)
	var agents []AgentAPIItem
	if err := json.Unmarshal(body, &agents); err != nil {
		// Try wrapped format for test compatibility
		var wrapped AgentAPIListResponse
		if err2 := json.Unmarshal(body, &wrapped); err2 != nil {
			return nil, fmt.Errorf("failed to decode agents response: %w", err)
		}
		agents = wrapped.Agents
	}

	// Filter for awaiting-cleanup agents and transform to attention items
	items := make([]AttentionItem, 0)
	now := time.Now()

	for _, agent := range agents {
		if agent.Status != "awaiting-cleanup" {
			continue
		}

		// Skip agents without beads ID (orphaned sessions)
		if agent.BeadsID == "" {
			continue
		}

		// Calculate priority based on role
		priority := calculateAgentPriority(agent, role)

		// Build summary
		summary := fmt.Sprintf("Phase: Complete - %s", truncate(agent.Task, 60))
		if agent.Task == "" && agent.BeadsTitle != "" {
			summary = fmt.Sprintf("Phase: Complete - %s", truncate(agent.BeadsTitle, 60))
		}

		item := AttentionItem{
			ID:          fmt.Sprintf("agent-%s", agent.BeadsID),
			Source:      "agent",
			Concern:     Actionability, // Awaiting-cleanup is actionable
			Signal:      "verify",
			Subject:     agent.BeadsID,
			Summary:     summary,
			Priority:    priority,
			Role:        role,
			ActionHint:  fmt.Sprintf("orch complete %s", agent.BeadsID),
			CollectedAt: now,
			Metadata: map[string]any{
				"agent_id":   agent.ID,
				"phase":      agent.Phase,
				"project":    agent.Project,
				"skill":      agent.Skill,
				"updated_at": agent.UpdatedAt,
			},
		}
		items = append(items, item)
	}

	return items, nil
}

// calculateAgentPriority determines priority based on role.
// Lower numbers = higher priority.
func calculateAgentPriority(agent AgentAPIItem, role string) int {
	// Base priority for verify signals - high because work is done, just needs completion
	basePriority := 50

	// Role-aware adjustments
	switch role {
	case "human":
		// Humans are the primary audience for verify signals
		return basePriority

	case "orchestrator":
		// Orchestrators care about completing work - high priority
		return basePriority - 10

	case "daemon":
		// Daemons might auto-complete in the future
		return basePriority + 20

	default:
		return basePriority
	}
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
