package main

import (
	"testing"

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
