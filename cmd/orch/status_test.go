package main

import (
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

	// Register agents with session IDs (headless agents)
	agents := []*registry.Agent{
		{ID: "agent-1", BeadsID: "beads-1", SessionID: "ses_abc123", Skill: "investigation"},
		{ID: "agent-2", BeadsID: "beads-2", SessionID: "ses_def456", Skill: "feature-impl"},
		{ID: "agent-3", BeadsID: "beads-3", Skill: "debugging"}, // no session ID
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
}
