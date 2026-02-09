package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestDetermineSpawnTierWithGlobalDefault(t *testing.T) {
	tests := []struct {
		name          string
		skill         string
		lightFlag     bool
		fullFlag      bool
		globalDefault string
		want          string
	}{
		{
			name:          "light flag overrides everything",
			skill:         "investigation",
			lightFlag:     true,
			fullFlag:      false,
			globalDefault: spawn.TierFull,
			want:          spawn.TierLight,
		},
		{
			name:          "full flag overrides everything",
			skill:         "systematic-debugging",
			lightFlag:     false,
			fullFlag:      true,
			globalDefault: spawn.TierLight,
			want:          spawn.TierFull,
		},
		{
			name:          "skill default full beats global light",
			skill:         "investigation",
			lightFlag:     false,
			fullFlag:      false,
			globalDefault: spawn.TierLight,
			want:          spawn.TierFull,
		},
		{
			name:          "skill default light beats global full",
			skill:         "systematic-debugging",
			lightFlag:     false,
			fullFlag:      false,
			globalDefault: spawn.TierFull,
			want:          spawn.TierLight,
		},
		{
			name:          "unknown skill uses global full fallback",
			skill:         "unknown-skill",
			lightFlag:     false,
			fullFlag:      false,
			globalDefault: spawn.TierFull,
			want:          spawn.TierFull,
		},
		{
			name:          "unknown skill uses global light fallback",
			skill:         "unknown-skill",
			lightFlag:     false,
			fullFlag:      false,
			globalDefault: spawn.TierLight,
			want:          spawn.TierLight,
		},
		{
			name:          "unknown skill without global fallback is full",
			skill:         "unknown-skill",
			lightFlag:     false,
			fullFlag:      false,
			globalDefault: "",
			want:          spawn.TierFull,
		},
		{
			name:          "unknown skill with invalid global fallback is full",
			skill:         "unknown-skill",
			lightFlag:     false,
			fullFlag:      false,
			globalDefault: "invalid",
			want:          spawn.TierFull,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineSpawnTierWithGlobalDefault(tt.skill, tt.lightFlag, tt.fullFlag, tt.globalDefault)
			if got != tt.want {
				t.Fatalf("determineSpawnTierWithGlobalDefault(%q, light=%v, full=%v, global=%q) = %q, want %q", tt.skill, tt.lightFlag, tt.fullFlag, tt.globalDefault, got, tt.want)
			}
		})
	}
}
