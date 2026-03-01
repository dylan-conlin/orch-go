package orch

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestDetermineSpawnTier_TaskScopeSignals(t *testing.T) {
	// Isolate user config lookups
	t.Setenv("HOME", t.TempDir())

	tests := []struct {
		name string
		task string
		want string
	}{
		{
			name: "session scope medium upgrades to full",
			task: "SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])",
			want: spawn.TierFull,
		},
		{
			name: "session scope small keeps light",
			task: "SESSION SCOPE: Small (text edits only)",
			want: spawn.TierLight,
		},
		{
			name: "new package with tests upgrades to full",
			task: "Create new package pkg/graph with comprehensive tests",
			want: spawn.TierFull,
		},
		{
			name: "no scope signals uses skill default",
			task: "Update README wording",
			want: spawn.TierLight,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineSpawnTier("feature-impl", tt.task, false, false)
			if got != tt.want {
				t.Errorf("DetermineSpawnTier() = %q, want %q", got, tt.want)
			}
		})
	}
}
