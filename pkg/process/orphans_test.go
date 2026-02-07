package process

import (
	"testing"
)

func TestFindAgentProcesses(t *testing.T) {
	// This test just verifies the function runs without error on the current system.
	// It doesn't assert specific processes since those are environment-dependent.
	agents, err := FindAgentProcesses()
	if err != nil {
		t.Fatalf("FindAgentProcesses() returned error: %v", err)
	}

	// Verify all returned agents have valid PIDs
	for _, agent := range agents {
		if agent.PID <= 0 {
			t.Errorf("Agent has invalid PID: %d", agent.PID)
		}
		if agent.Command == "" {
			t.Errorf("Agent PID %d has empty command", agent.PID)
		}
	}

	t.Logf("Found %d agent processes", len(agents))
}

func TestFindOrphanProcesses(t *testing.T) {
	// Test with empty active sessions - all agents should be reported as orphans
	allAgents, err := FindAgentProcesses()
	if err != nil {
		t.Fatalf("FindAgentProcesses() returned error: %v", err)
	}

	orphans, err := FindOrphanProcesses(map[string]bool{})
	if err != nil {
		t.Fatalf("FindOrphanProcesses() returned error: %v", err)
	}

	// With no active sessions, all agents should be orphans
	if len(orphans) != len(allAgents) {
		t.Errorf("Expected %d orphans with empty active set, got %d", len(allAgents), len(orphans))
	}

	// Test with active sessions matching workspace names
	if len(allAgents) > 0 {
		active := make(map[string]bool)
		for _, agent := range allAgents {
			if agent.WorkspaceName != "" {
				active[agent.WorkspaceName] = true
			}
		}

		orphans, err = FindOrphanProcesses(active)
		if err != nil {
			t.Fatalf("FindOrphanProcesses() with active set returned error: %v", err)
		}

		// With all workspace names active, no agents with workspace names should be orphans
		for _, orphan := range orphans {
			if orphan.WorkspaceName != "" && active[orphan.WorkspaceName] {
				t.Errorf("Orphan PID %d has workspace name %s that is in active set", orphan.PID, orphan.WorkspaceName)
			}
		}
	}
}
