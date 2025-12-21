package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/registry"
)

// TestFormatDuration tests the formatDuration function.
// Note: formatDuration is defined in wait.go
func TestFormatDurationForStatus(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes and seconds", 5*time.Minute + 23*time.Second, "5m 23s"},
		{"hours and minutes", 1*time.Hour + 2*time.Minute, "1h 2m"},
		{"zero", 0, "0s"},
		{"just minutes", 10 * time.Minute, "10m"},
		{"just hours", 3 * time.Hour, "3h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

// TestSwarmStatusSerialization tests that SwarmStatus serializes correctly to JSON.
func TestSwarmStatusSerialization(t *testing.T) {
	status := SwarmStatus{
		Active:    3,
		Queued:    2,
		Completed: 7,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal SwarmStatus: %v", err)
	}

	var parsed SwarmStatus
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal SwarmStatus: %v", err)
	}

	if parsed.Active != 3 || parsed.Queued != 2 || parsed.Completed != 7 {
		t.Errorf("SwarmStatus mismatch: got %+v, want {Active:3 Queued:2 Completed:7}", parsed)
	}
}

// TestAccountUsageSerialization tests that AccountUsage serializes correctly to JSON.
func TestAccountUsageSerialization(t *testing.T) {
	usage := AccountUsage{
		Name:        "personal",
		Email:       "user@example.com",
		UsedPercent: 45.5,
		ResetTime:   "2h",
		IsActive:    true,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("Failed to marshal AccountUsage: %v", err)
	}

	var parsed AccountUsage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal AccountUsage: %v", err)
	}

	if parsed.Name != "personal" || parsed.Email != "user@example.com" ||
		parsed.UsedPercent != 45.5 || parsed.ResetTime != "2h" || !parsed.IsActive {
		t.Errorf("AccountUsage mismatch: got %+v", parsed)
	}
}

// TestAgentInfoSerialization tests that AgentInfo serializes correctly to JSON.
func TestAgentInfoSerialization(t *testing.T) {
	agent := AgentInfo{
		SessionID: "ses_abc123",
		BeadsID:   "proj-1",
		Skill:     "investigation",
		Account:   "personal",
		Runtime:   "5m23s",
		Title:     "Test Agent",
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Failed to marshal AgentInfo: %v", err)
	}

	var parsed AgentInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal AgentInfo: %v", err)
	}

	if parsed.SessionID != "ses_abc123" || parsed.BeadsID != "proj-1" ||
		parsed.Skill != "investigation" || parsed.Account != "personal" ||
		parsed.Runtime != "5m23s" || parsed.Title != "Test Agent" {
		t.Errorf("AgentInfo mismatch: got %+v", parsed)
	}
}

// TestStatusOutputSerialization tests the full StatusOutput serialization.
func TestStatusOutputSerialization(t *testing.T) {
	output := StatusOutput{
		Swarm: SwarmStatus{
			Active:    3,
			Queued:    2,
			Completed: 7,
		},
		Accounts: []AccountUsage{
			{Name: "personal", Email: "user@example.com", UsedPercent: 45.5, ResetTime: "2h", IsActive: true},
			{Name: "work", Email: "user@company.com", UsedPercent: 12.0, ResetTime: "5h", IsActive: false},
		},
		Agents: []AgentInfo{
			{SessionID: "ses_abc123", BeadsID: "proj-1", Skill: "investigation", Account: "personal", Runtime: "5m23s"},
			{SessionID: "ses_def456", BeadsID: "proj-2", Skill: "feature-impl", Account: "personal", Runtime: "2m11s"},
		},
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal StatusOutput: %v", err)
	}

	// Verify it's valid JSON
	var parsed StatusOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal StatusOutput: %v", err)
	}

	if parsed.Swarm.Active != 3 {
		t.Errorf("Swarm.Active = %d, want 3", parsed.Swarm.Active)
	}
	if len(parsed.Accounts) != 2 {
		t.Errorf("len(Accounts) = %d, want 2", len(parsed.Accounts))
	}
	if len(parsed.Agents) != 2 {
		t.Errorf("len(Agents) = %d, want 2", len(parsed.Agents))
	}
}

// TestCompletedTodayCount tests counting completed agents from today.
func TestCompletedTodayCount(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register some agents
	agent1 := &registry.Agent{ID: "agent-1", BeadsID: "beads-1"}
	agent2 := &registry.Agent{ID: "agent-2", BeadsID: "beads-2"}
	agent3 := &registry.Agent{ID: "agent-3", BeadsID: "beads-3"}

	for _, a := range []*registry.Agent{agent1, agent2, agent3} {
		if err := reg.Register(a); err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}
	}

	// Complete some agents
	reg.Complete("agent-1")
	reg.Complete("agent-2")

	if err := reg.Save(); err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Count completed today
	completed := reg.ListCompleted()
	if len(completed) != 2 {
		t.Errorf("Expected 2 completed agents, got %d", len(completed))
	}

	// Verify timestamps exist
	for _, a := range completed {
		if a.CompletedAt == "" {
			t.Errorf("Agent %s has no CompletedAt timestamp", a.ID)
		}
	}
}

// TestAgentBySessionLookup tests building the session-to-agent lookup map.
func TestAgentBySessionLookup(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := registry.New(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register agents with session IDs (like headless agents)
	agents := []*registry.Agent{
		{ID: "agent-1", BeadsID: "beads-1", SessionID: "ses_abc123", Skill: "investigation"},
		{ID: "agent-2", BeadsID: "beads-2", SessionID: "ses_def456", Skill: "feature-impl"},
		{ID: "agent-3", BeadsID: "beads-3", WindowID: "@100", Skill: "debugging"}, // tmux agent, no session ID
	}

	for _, a := range agents {
		if err := reg.Register(a); err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}
	}

	if err := reg.Save(); err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Build lookup map
	agentBySession := make(map[string]*registry.Agent)
	for _, a := range reg.ListActive() {
		if a.SessionID != "" {
			agentBySession[a.SessionID] = a
		}
	}

	// Verify lookup
	if len(agentBySession) != 2 {
		t.Errorf("Expected 2 agents in lookup, got %d", len(agentBySession))
	}

	if a, ok := agentBySession["ses_abc123"]; !ok {
		t.Error("ses_abc123 not found in lookup")
	} else if a.Skill != "investigation" {
		t.Errorf("Wrong agent for ses_abc123: got skill %s", a.Skill)
	}

	if a, ok := agentBySession["ses_def456"]; !ok {
		t.Error("ses_def456 not found in lookup")
	} else if a.Skill != "feature-impl" {
		t.Errorf("Wrong agent for ses_def456: got skill %s", a.Skill)
	}

	// Verify tmux agent is not in lookup
	if _, ok := agentBySession["@100"]; ok {
		t.Error("tmux window ID should not be in session lookup")
	}
}

// TestPrintSwarmStatusFormat tests the printSwarmStatus function output format.
func TestPrintSwarmStatusFormat(t *testing.T) {
	// This test verifies the output doesn't panic and produces reasonable output.
	// We can't easily capture stdout in Go tests, so we just verify it doesn't error.
	output := StatusOutput{
		Swarm: SwarmStatus{Active: 0, Queued: 0, Completed: 0},
		Accounts: []AccountUsage{
			{Name: "current", Email: "test@example.com", UsedPercent: 50.0, ResetTime: "3h", IsActive: true},
		},
		Agents: nil,
	}

	// This should not panic
	printSwarmStatus(output)

	// Test with agents
	output.Swarm.Active = 2
	output.Agents = []AgentInfo{
		{SessionID: "ses_123", BeadsID: "proj-1", Skill: "test", Runtime: "5m"},
		{SessionID: "ses_456", Runtime: "1m"}, // minimal info
	}
	printSwarmStatus(output)
}

// TestStatusJSONFlag tests the --json flag behavior.
func TestStatusJSONFlag(t *testing.T) {
	// Verify the flag variable exists and has correct default
	if statusJSON {
		t.Error("statusJSON should default to false")
	}

	// Set and verify
	statusJSON = true
	if !statusJSON {
		t.Error("statusJSON should be true after setting")
	}

	// Reset for other tests
	statusJSON = false
}
