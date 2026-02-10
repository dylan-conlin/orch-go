package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// TestExtractSkillFromTitle_StatusContext tests skill extraction for status display.
func TestExtractSkillFromTitle_StatusContext(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		wantSkill string
	}{
		{
			name:      "feature-impl from -feat-",
			title:     "og-feat-add-feature-19dec",
			wantSkill: "feature-impl",
		},
		{
			name:      "investigation from -inv-",
			title:     "og-inv-explore-codebase-19dec",
			wantSkill: "investigation",
		},
		{
			name:      "systematic-debugging from -debug-",
			title:     "og-debug-fix-bug-19dec",
			wantSkill: "systematic-debugging",
		},
		{
			name:      "architect from -arch-",
			title:     "og-arch-design-system-19dec",
			wantSkill: "architect",
		},
		{
			name:      "no matching pattern",
			title:     "random-session-name",
			wantSkill: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSkillFromTitle(tt.title)
			if got != tt.wantSkill {
				t.Errorf("extractSkillFromTitle(%q) = %q, want %q", tt.title, got, tt.wantSkill)
			}
		})
	}
}

// TestGetAgentStatus tests agent status determination.
func TestGetAgentStatus(t *testing.T) {
	tests := []struct {
		name     string
		agent    AgentInfo
		expected string
	}{
		{
			name:     "completed takes precedence",
			agent:    AgentInfo{IsCompleted: true, IsPhantom: true, IsProcessing: true},
			expected: "completed",
		},
		{
			name:     "phantom takes precedence over processing",
			agent:    AgentInfo{IsPhantom: true, IsProcessing: true},
			expected: "phantom",
		},
		{
			name:     "processing/running",
			agent:    AgentInfo{IsProcessing: true},
			expected: "running",
		},
		{
			name:     "default is idle",
			agent:    AgentInfo{},
			expected: "idle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAgentStatus(tt.agent)
			if got != tt.expected {
				t.Errorf("getAgentStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestComputeIsPhantom(t *testing.T) {
	tests := []struct {
		name        string
		agent       AgentInfo
		issue       *verify.Issue
		issueExists bool
		expected    bool
	}{
		{
			name:        "open issue and no runtime => phantom",
			agent:       AgentInfo{BeadsID: "orch-go-abcd"},
			issue:       &verify.Issue{ID: "orch-go-abcd", Status: "open"},
			issueExists: true,
			expected:    true,
		},
		{
			name:        "runtime session => not phantom",
			agent:       AgentInfo{BeadsID: "orch-go-abcd", SessionID: "ses_123"},
			issue:       &verify.Issue{ID: "orch-go-abcd", Status: "open"},
			issueExists: true,
			expected:    false,
		},
		{
			name:        "runtime tmux window => not phantom",
			agent:       AgentInfo{BeadsID: "orch-go-abcd", Window: "workers:1"},
			issue:       &verify.Issue{ID: "orch-go-abcd", Status: "open"},
			issueExists: true,
			expected:    false,
		},
		{
			name:        "closed issue => not phantom",
			agent:       AgentInfo{BeadsID: "orch-go-abcd"},
			issue:       &verify.Issue{ID: "orch-go-abcd", Status: "closed"},
			issueExists: true,
			expected:    false,
		},
		{
			name:        "missing issue => not phantom",
			agent:       AgentInfo{BeadsID: "orch-go-abcd"},
			issue:       nil,
			issueExists: false,
			expected:    false,
		},
		{
			name:        "no-track beads id => not phantom",
			agent:       AgentInfo{BeadsID: "orch-go-untracked-1768090360"},
			issue:       nil,
			issueExists: false,
			expected:    false,
		},
		{
			name:        "explicit IsUntracked => not phantom",
			agent:       AgentInfo{BeadsID: "orch-go-abcd", IsUntracked: true},
			issue:       &verify.Issue{ID: "orch-go-abcd", Status: "open"},
			issueExists: true,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeIsPhantom(tt.agent, tt.issue, tt.issueExists)
			if got != tt.expected {
				t.Errorf("computeIsPhantom() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsIdleWithWork tests detection of idle agents that have done meaningful work.
func TestIsIdleWithWork(t *testing.T) {
	tests := []struct {
		name     string
		agent    AgentInfo
		expected bool
	}{
		{
			name:     "idle with high tokens (>50K) is idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", Tokens: &opencode.TokenStats{TotalTokens: 60000}},
			expected: true,
		},
		{
			name:     "idle with tokens at threshold (50K) is idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", Tokens: &opencode.TokenStats{TotalTokens: 50000}},
			expected: true,
		},
		{
			name:     "idle with low tokens (<50K) is NOT idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", Tokens: &opencode.TokenStats{TotalTokens: 30000}},
			expected: false,
		},
		{
			name:     "idle at Phase Testing is idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", Phase: "Testing"},
			expected: true,
		},
		{
			name:     "idle at Phase Implementing is idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", Phase: "Implementing"},
			expected: true,
		},
		{
			name:     "idle at Phase Validation is idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", Phase: "Validation"},
			expected: true,
		},
		{
			name:     "idle at Phase Planning is NOT idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", Phase: "Planning"},
			expected: false,
		},
		{
			name:     "idle with no phase and no tokens is NOT idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc"},
			expected: false,
		},
		{
			name:     "running agent is NOT idle-with-work (already visible)",
			agent:    AgentInfo{BeadsID: "orch-go-abc", IsProcessing: true, Tokens: &opencode.TokenStats{TotalTokens: 200000}},
			expected: false,
		},
		{
			name:     "completed agent is NOT idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", IsCompleted: true, Tokens: &opencode.TokenStats{TotalTokens: 200000}},
			expected: false,
		},
		{
			name:     "phantom agent is NOT idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", IsPhantom: true, Tokens: &opencode.TokenStats{TotalTokens: 200000}},
			expected: false,
		},
		{
			name:     "untracked agent is NOT idle-with-work",
			agent:    AgentInfo{BeadsID: "orch-go-abc", IsUntracked: true, Tokens: &opencode.TokenStats{TotalTokens: 200000}},
			expected: false,
		},
		{
			name:     "tokens computed from input+output when TotalTokens is 0",
			agent:    AgentInfo{BeadsID: "orch-go-abc", Tokens: &opencode.TokenStats{InputTokens: 40000, OutputTokens: 15000}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIdleWithWork(tt.agent)
			if got != tt.expected {
				t.Errorf("isIdleWithWork() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestFilterAgentsForDisplay_IdleWithWork tests that idle agents with meaningful work
// are surfaced in compact mode (the original bug).
func TestFilterAgentsForDisplay_IdleWithWork(t *testing.T) {
	tests := []struct {
		name         string
		agents       []AgentInfo
		showAll      bool
		wantCount    int
		wantBeadsIDs []string // Expected beads IDs in the result
	}{
		{
			name: "compact mode shows idle agent with high tokens",
			agents: []AgentInfo{
				{BeadsID: "orch-go-idle-work", Phase: "Testing", Tokens: &opencode.TokenStats{TotalTokens: 263000}},
			},
			showAll:      false,
			wantCount:    1,
			wantBeadsIDs: []string{"orch-go-idle-work"},
		},
		{
			name: "compact mode hides idle agent with no work",
			agents: []AgentInfo{
				{BeadsID: "orch-go-idle-nowork", Phase: "Planning", Tokens: &opencode.TokenStats{TotalTokens: 5000}},
			},
			showAll:   false,
			wantCount: 0,
		},
		{
			name: "compact mode shows running + idle-with-work agents",
			agents: []AgentInfo{
				{BeadsID: "orch-go-running", IsProcessing: true},
				{BeadsID: "orch-go-idle-work", Phase: "Implementing", Tokens: &opencode.TokenStats{TotalTokens: 100000}},
				{BeadsID: "orch-go-idle-nowork"},
			},
			showAll:      false,
			wantCount:    2,
			wantBeadsIDs: []string{"orch-go-running", "orch-go-idle-work"},
		},
		{
			name: "all mode shows all agents regardless",
			agents: []AgentInfo{
				{BeadsID: "orch-go-running", IsProcessing: true},
				{BeadsID: "orch-go-idle-work", Phase: "Implementing", Tokens: &opencode.TokenStats{TotalTokens: 100000}},
				{BeadsID: "orch-go-idle-nowork"},
			},
			showAll:      true,
			wantCount:    3,
			wantBeadsIDs: []string{"orch-go-running", "orch-go-idle-work", "orch-go-idle-nowork"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterAgentsForDisplay(tt.agents, tt.showAll, "")
			if len(got) != tt.wantCount {
				t.Errorf("filterAgentsForDisplay() returned %d agents, want %d", len(got), tt.wantCount)
				for _, a := range got {
					t.Logf("  got: %s (processing=%v, phase=%s)", a.BeadsID, a.IsProcessing, a.Phase)
				}
			}
			for _, wantID := range tt.wantBeadsIDs {
				found := false
				for _, a := range got {
					if a.BeadsID == wantID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected agent %s in result, but not found", wantID)
				}
			}
		})
	}
}

// TestGetAgentStatus_IdleWithWork tests that idle-with-work agents show attention indicator.
func TestGetAgentStatus_IdleWithWork(t *testing.T) {
	// An idle agent with high tokens and active phase should show "idle ⚠"
	agent := AgentInfo{
		BeadsID: "orch-go-abc",
		Phase:   "Testing",
		Tokens:  &opencode.TokenStats{TotalTokens: 263000},
	}
	status := getAgentStatus(agent)
	if status != "idle ⚠" {
		t.Errorf("getAgentStatus() = %q, want %q for idle-with-work agent", status, "idle ⚠")
	}

	// A plain idle agent without work should still show "idle"
	plainAgent := AgentInfo{BeadsID: "orch-go-xyz"}
	plainStatus := getAgentStatus(plainAgent)
	if plainStatus != "idle" {
		t.Errorf("getAgentStatus() = %q, want %q for plain idle agent", plainStatus, "idle")
	}
}

func TestComputeSwarmStatus(t *testing.T) {
	agents := []AgentInfo{
		{BeadsID: "orch-go-1", IsProcessing: true},
		{BeadsID: "orch-go-2"},
		{BeadsID: "orch-go-3", IsPhantom: true},
		{BeadsID: "orch-go-4", IsCompleted: true, IsPhantom: true},
		{SessionID: "ses_x", IsUntracked: true, IsProcessing: true},
		{SessionID: "ses_y", IsUntracked: true},
	}

	swarm := computeSwarmStatus(agents)

	if swarm.Active != 2 {
		t.Fatalf("Active = %d, want %d", swarm.Active, 2)
	}
	if swarm.Processing != 2 {
		t.Fatalf("Processing = %d, want %d", swarm.Processing, 2)
	}
	if swarm.Idle != 1 {
		t.Fatalf("Idle = %d, want %d", swarm.Idle, 1)
	}
	if swarm.Phantom != 1 {
		t.Fatalf("Phantom = %d, want %d", swarm.Phantom, 1)
	}
	if swarm.Completed != 1 {
		t.Fatalf("Completed = %d, want %d", swarm.Completed, 1)
	}
	if swarm.Untracked != 2 {
		t.Fatalf("Untracked = %d, want %d", swarm.Untracked, 2)
	}
}
