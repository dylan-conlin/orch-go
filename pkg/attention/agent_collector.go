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

	agentSnapshot []AgentAPIItem
	snapshotErr   error
	useSnapshot   bool
}

// AgentAPIItem represents an agent from the /api/agents response.
// This is a subset of the full response - only fields needed for attention signals.
type AgentAPIItem struct {
	ID             string `json:"id"`
	BeadsID        string `json:"beads_id"`
	BeadsTitle     string `json:"beads_title"`
	Status         string `json:"status"`
	Phase          string `json:"phase"`
	Task           string `json:"task"`
	Project        string `json:"project"`
	Skill          string `json:"skill"`
	IsStalled      bool   `json:"is_stalled"`
	SpawnedAt      string `json:"spawned_at"`
	UpdatedAt      string `json:"updated_at"`
	LastActivityAt string `json:"last_activity_at"`
	Runtime        string `json:"runtime"`
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

// NewAgentCollectorWithSnapshot creates a collector that uses a pre-fetched shared snapshot.
// If snapshotErr is non-nil, Collect returns that error without making API requests.
func NewAgentCollectorWithSnapshot(agentSnapshot []AgentAPIItem, snapshotErr error) *AgentCollector {
	return &AgentCollector{
		agentSnapshot: agentSnapshot,
		snapshotErr:   snapshotErr,
		useSnapshot:   true,
	}
}

// FetchAgentSnapshot fetches all agents once so multiple collectors can share results.
func FetchAgentSnapshot(client *http.Client, apiURL string) ([]AgentAPIItem, error) {
	if client == nil {
		return nil, fmt.Errorf("agents API client is nil")
	}
	if apiURL == "" {
		return nil, fmt.Errorf("agents API URL is empty")
	}

	url := fmt.Sprintf("%s/api/agents?since=all", apiURL)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("agents API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents response: %w", err)
	}

	var agents []AgentAPIItem
	if err := json.Unmarshal(body, &agents); err != nil {
		var wrapped AgentAPIListResponse
		if err2 := json.Unmarshal(body, &wrapped); err2 != nil {
			return nil, fmt.Errorf("failed to decode agents response: %w", err)
		}
		agents = wrapped.Agents
	}

	return agents, nil
}

// Collect gathers attention items for awaiting-cleanup agents.
// These are agents that have reported Phase: Complete but their session has died,
// indicating they need 'orch complete' to close out the work.
func (c *AgentCollector) Collect(role string) ([]AttentionItem, error) {
	agents, err := c.resolveAgents()
	if err != nil {
		return nil, err
	}

	return collectAgentAttentionItems(agents, role), nil
}

func (c *AgentCollector) resolveAgents() ([]AgentAPIItem, error) {
	if c.useSnapshot {
		if c.snapshotErr != nil {
			return nil, c.snapshotErr
		}
		return c.agentSnapshot, nil
	}

	return FetchAgentSnapshot(c.client, c.apiURL)
}

func collectAgentAttentionItems(agents []AgentAPIItem, role string) []AttentionItem {
	items := make([]AttentionItem, 0)
	now := time.Now()

	for _, agent := range agents {
		if agent.Status != "awaiting-cleanup" {
			continue
		}

		if agent.BeadsID == "" {
			continue
		}

		priority := calculateAgentPriority(agent, role)

		summary := fmt.Sprintf("Phase: Complete - %s", truncate(agent.Task, 60))
		if agent.Task == "" && agent.BeadsTitle != "" {
			summary = fmt.Sprintf("Phase: Complete - %s", truncate(agent.BeadsTitle, 60))
		}

		item := AttentionItem{
			ID:          fmt.Sprintf("agent-%s", agent.BeadsID),
			Source:      "agent",
			Concern:     Actionability,
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

	return items
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
