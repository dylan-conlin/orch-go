package process

import (
	"testing"
)

func TestIsOpenCodeAgentLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		match bool
	}{
		{
			name:  "headless agent",
			line:  `57559 bun run --conditions=browser ./src/index.ts /Users/dylanconlin/Documents/personal/orch-go`,
			match: true,
		},
		{
			name:  "attach agent",
			line:  `12345 bun run --conditions=browser ./src/index.ts attach http://127.0.0.1:4096 --dir /tmp --session abc123`,
			match: true,
		},
		{
			name:  "old format agent",
			line:  `12345 bun run --conditions=browser ./src/index.ts run --attach http://127.0.0.1:4096 --title my-workspace [beads-id]`,
			match: true,
		},
		{
			name:  "opencode server (excluded)",
			line:  `12345 bun run --conditions=browser ./src/index.ts serve --port 4096`,
			match: false,
		},
		{
			name:  "other bun project with src/index.ts (no --conditions=browser)",
			line:  `12345 bun run src/index.ts`,
			match: false,
		},
		{
			name:  "other bun project",
			line:  `12345 bun run dev`,
			match: false,
		},
		{
			name:  "chrome-devtools-mcp bun",
			line:  `12345 bun run --watch src/index.ts`,
			match: false,
		},
		{
			name:  "system process with 'bun' in path",
			line:  `12345 /System/Library/CoreServices/SafariSupport.bundle/Contents/MacOS/SafariBookmarksSyncAgent`,
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAgent := isOpenCodeAgentLine(tt.line)
			if isAgent != tt.match {
				t.Errorf("isOpenCodeAgentLine(%q) = %v, want %v", tt.line, isAgent, tt.match)
			}
		})
	}
}

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

	orphans, err := FindOrphanProcesses(map[string]bool{}, map[string]bool{})
	if err != nil {
		t.Fatalf("FindOrphanProcesses() returned error: %v", err)
	}

	// With no active sessions, all agents should be orphans
	if len(orphans) != len(allAgents) {
		t.Errorf("Expected %d orphans with empty active set, got %d", len(allAgents), len(orphans))
	}

	// Test with active sessions matching workspace names or session IDs
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

		// With all workspace names and session IDs active, matched agents should not be orphans
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
