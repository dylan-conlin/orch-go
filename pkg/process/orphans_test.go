package process

import (
	"testing"
)

func TestIsOpenCodeProcess(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		match bool
	}{
		{
			name:  "opencode process",
			line:  `57559 1234 bun run --conditions=browser ./src/index.ts /Users/dylanconlin/Documents/personal/orch-go`,
			match: true,
		},
		{
			name:  "opencode server",
			line:  `12345 1 bun run --conditions=browser ./src/index.ts serve --port 4096`,
			match: true,
		},
		{
			name:  "no conditions flag",
			line:  `12345 1 bun run src/index.ts`,
			match: false,
		},
		{
			name:  "other bun project",
			line:  `12345 1 bun run dev`,
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOpenCodeProcess(tt.line)
			if got != tt.match {
				t.Errorf("isOpenCodeProcess(%q) = %v, want %v", tt.line, got, tt.match)
			}
		})
	}
}

func TestIsReapableAgent(t *testing.T) {
	const serverPID = 1000

	tests := []struct {
		name      string
		line      string
		ppid      int
		serverPID int
		want      bool
	}{
		{
			name:      "attach agent",
			line:      `bun run --conditions=browser ./src/index.ts attach http://127.0.0.1:4096 --dir /tmp --session abc123`,
			ppid:      999,
			serverPID: serverPID,
			want:      true,
		},
		{
			name:      "old format attach agent",
			line:      `bun run --conditions=browser ./src/index.ts run --attach http://127.0.0.1:4096 --title my-workspace [beads-id]`,
			ppid:      999,
			serverPID: serverPID,
			want:      true,
		},
		{
			name:      "headless agent (child of server)",
			line:      `bun run --conditions=browser ./src/index.ts /Users/dylanconlin/Documents/personal/orch-go`,
			ppid:      serverPID,
			serverPID: serverPID,
			want:      true,
		},
		{
			name:      "orphan (parent died, reparented to init)",
			line:      `bun run --conditions=browser ./src/index.ts /Users/dylanconlin/Documents/personal/orch-go`,
			ppid:      1,
			serverPID: serverPID,
			want:      true,
		},
		{
			name:      "TUI (child of user shell, no attach)",
			line:      `bun run --conditions=browser ./src/index.ts /Users/dylanconlin/Documents/personal/orch-go`,
			ppid:      5555, // shell PID
			serverPID: serverPID,
			want:      false,
		},
		{
			name:      "TUI when server not running",
			line:      `bun run --conditions=browser ./src/index.ts /Users/dylanconlin/Documents/personal/orch-go`,
			ppid:      5555,
			serverPID: 0, // server not found
			want:      false,
		},
		{
			name:      "orphan when server not running",
			line:      `bun run --conditions=browser ./src/index.ts /Users/dylanconlin/Documents/personal/orch-go`,
			ppid:      1,
			serverPID: 0, // server not found, but PPID=1 means parent died
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isReapableAgent(tt.line, tt.ppid, tt.serverPID)
			if got != tt.want {
				t.Errorf("isReapableAgent(%q, ppid=%d, server=%d) = %v, want %v",
					tt.line, tt.ppid, tt.serverPID, got, tt.want)
			}
		})
	}
}

func TestFindAgentProcesses(t *testing.T) {
	// This test just verifies the function runs without error on the current system.
	agents, err := FindAgentProcesses()
	if err != nil {
		t.Fatalf("FindAgentProcesses() returned error: %v", err)
	}

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
	allAgents, err := FindAgentProcesses()
	if err != nil {
		t.Fatalf("FindAgentProcesses() returned error: %v", err)
	}

	orphans, err := FindOrphanProcesses(map[string]bool{}, map[string]bool{})
	if err != nil {
		t.Fatalf("FindOrphanProcesses() returned error: %v", err)
	}

	// With no active sessions, all agents should be orphans
	if len(orphans) != len(allAgents) {
		t.Errorf("Expected %d orphans with empty active set, got %d", len(allAgents), len(orphans))
	}

	if len(allAgents) > 0 {
		activeTitles := make(map[string]bool)
		activeIDs := make(map[string]bool)
		for _, agent := range allAgents {
			if agent.WorkspaceName != "" {
				activeTitles[agent.WorkspaceName] = true
			}
			if agent.SessionID != "" {
				activeIDs[agent.SessionID] = true
			}
		}

		orphans, err = FindOrphanProcesses(activeTitles, activeIDs)
		if err != nil {
			t.Fatalf("FindOrphanProcesses() with active set returned error: %v", err)
		}

		for _, orphan := range orphans {
			if orphan.WorkspaceName != "" && activeTitles[orphan.WorkspaceName] {
				t.Errorf("Orphan PID %d has workspace name %s that is in active set", orphan.PID, orphan.WorkspaceName)
			}
			if orphan.SessionID != "" && activeIDs[orphan.SessionID] {
				t.Errorf("Orphan PID %d has session ID %s that is in active set", orphan.PID, orphan.SessionID)
			}
		}
	}
}

func TestStartupSweepWithReconciliation(t *testing.T) {
	// Test that the function executes without panicking
	// Note: This is a basic smoke test since the function interacts with system processes
	result := StartupSweepWithReconciliation()

	// Basic result validation
	if result.LedgerTotalEntries < 0 {
		t.Errorf("Expected non-negative LedgerTotalEntries, got %d", result.LedgerTotalEntries)
	}

	if result.LedgerStaleRemoved < 0 {
		t.Errorf("Expected non-negative LedgerStaleRemoved, got %d", result.LedgerStaleRemoved)
	}

	if result.OrphanProcessesFound < 0 {
		t.Errorf("Expected non-negative OrphanProcessesFound, got %d", result.OrphanProcessesFound)
	}

	if result.OrphanProcessesKilled < 0 {
		t.Errorf("Expected non-negative OrphanProcessesKilled, got %d", result.OrphanProcessesKilled)
	}

	// Should not kill more than what was found
	if result.OrphanProcessesKilled > result.OrphanProcessesFound {
		t.Errorf("Killed (%d) more processes than found (%d)", result.OrphanProcessesKilled, result.OrphanProcessesFound)
	}
}

func TestPerformFullReconciliation(t *testing.T) {
	// Test with empty active session maps - should not error
	killed, err := PerformFullReconciliation(map[string]bool{}, map[string]bool{})
	if err != nil {
		t.Errorf("PerformFullReconciliation() returned error: %v", err)
	}

	if killed < 0 {
		t.Errorf("Expected non-negative killed count, got %d", killed)
	}
}
