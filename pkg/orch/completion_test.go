package orch

import (
	"testing"
	"time"
)

// TestDetectCompletionBacklog tests the completion backlog detection logic.
// This is a placeholder test structure that will be fully implemented in orch-go-k5v
// when the actual detection logic is added.
func TestDetectCompletionBacklog(t *testing.T) {
	tests := []struct {
		name      string
		agents    []AgentInfo
		threshold time.Duration
		want      []string
	}{
		{
			name:      "empty agents list",
			agents:    []AgentInfo{},
			threshold: 10 * time.Minute,
			want:      nil,
		},
		{
			name: "no agents in backlog",
			agents: []AgentInfo{
				{
					BeadsID:         "orch-go-abc",
					Phase:           "Planning",
					PhaseReportedAt: time.Now().Add(-15 * time.Minute),
					Status:          "active",
				},
				{
					BeadsID:         "orch-go-xyz",
					Phase:           "Complete",
					PhaseReportedAt: time.Now().Add(-5 * time.Minute), // Within threshold
					Status:          "active",
				},
			},
			threshold: 10 * time.Minute,
			want:      nil,
		},
		{
			name: "one agent in backlog",
			agents: []AgentInfo{
				{
					BeadsID:         "orch-go-abc",
					Phase:           "Complete",
					PhaseReportedAt: time.Now().Add(-15 * time.Minute), // Beyond threshold
					Status:          "active",
				},
				{
					BeadsID:         "orch-go-xyz",
					Phase:           "Planning",
					PhaseReportedAt: time.Now().Add(-20 * time.Minute),
					Status:          "active",
				},
			},
			threshold: 10 * time.Minute,
			want:      []string{"orch-go-abc"},
		},
		{
			name: "multiple agents in backlog",
			agents: []AgentInfo{
				{
					BeadsID:         "orch-go-abc",
					Phase:           "Complete",
					PhaseReportedAt: time.Now().Add(-15 * time.Minute),
					Status:          "active",
				},
				{
					BeadsID:         "orch-go-def",
					Phase:           "Complete",
					PhaseReportedAt: time.Now().Add(-25 * time.Minute),
					Status:          "idle",
				},
				{
					BeadsID:         "orch-go-ghi",
					Phase:           "Planning",
					PhaseReportedAt: time.Now().Add(-30 * time.Minute),
					Status:          "active",
				},
			},
			threshold: 10 * time.Minute,
			want:      []string{"orch-go-abc", "orch-go-def"},
		},
		{
			name: "completed agents should be excluded",
			agents: []AgentInfo{
				{
					BeadsID:         "orch-go-abc",
					Phase:           "Complete",
					PhaseReportedAt: time.Now().Add(-15 * time.Minute),
					Status:          "completed", // Already closed by orch complete
				},
			},
			threshold: 10 * time.Minute,
			want:      nil,
		},
		{
			name: "case insensitive phase matching",
			agents: []AgentInfo{
				{
					BeadsID:         "orch-go-abc",
					Phase:           "complete", // lowercase
					PhaseReportedAt: time.Now().Add(-15 * time.Minute),
					Status:          "active",
				},
				{
					BeadsID:         "orch-go-def",
					Phase:           "COMPLETE", // uppercase
					PhaseReportedAt: time.Now().Add(-20 * time.Minute),
					Status:          "idle",
				},
			},
			threshold: 10 * time.Minute,
			want:      []string{"orch-go-abc", "orch-go-def"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// NOTE: This test is a placeholder. When the actual implementation is added
			// in orch-go-k5v, uncomment the assertions below and remove this skip.
			t.Skip("Placeholder test - will be implemented in orch-go-k5v")

			// Uncomment when implementation is complete:
			// got := DetectCompletionBacklog(tt.agents, tt.threshold)
			// if !equalStringSlices(got, tt.want) {
			// 	t.Errorf("DetectCompletionBacklog() = %v, want %v", got, tt.want)
			// }
		})
	}
}

// equalStringSlices compares two string slices for equality (order-independent).
// This will be used when the actual tests are enabled in orch-go-k5v.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Create maps for O(n) comparison
	aMap := make(map[string]bool)
	for _, v := range a {
		aMap[v] = true
	}

	for _, v := range b {
		if !aMap[v] {
			return false
		}
	}

	return true
}
