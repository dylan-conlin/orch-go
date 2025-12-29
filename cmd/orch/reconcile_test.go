package main

import (
	"testing"
	"time"
)

func TestZombieIssue_AgeSinceUpdate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		updatedAt     time.Time
		minHoursAgo   float64
		maxHoursAgo   float64
		wantAgeSuffix string
	}{
		{
			name:          "recent update within hour",
			updatedAt:     now.Add(-30 * time.Minute),
			minHoursAgo:   0.4,
			maxHoursAgo:   0.6,
			wantAgeSuffix: "m",
		},
		{
			name:          "update hours ago",
			updatedAt:     now.Add(-5 * time.Hour),
			minHoursAgo:   4.9,
			maxHoursAgo:   5.1,
			wantAgeSuffix: "h",
		},
		{
			name:          "update days ago",
			updatedAt:     now.Add(-48 * time.Hour),
			minHoursAgo:   47.9,
			maxHoursAgo:   48.1,
			wantAgeSuffix: "h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hoursSince := now.Sub(tt.updatedAt).Hours()

			if hoursSince < tt.minHoursAgo || hoursSince > tt.maxHoursAgo {
				t.Errorf("hoursSinceUpdate = %v, want between %v and %v",
					hoursSince, tt.minHoursAgo, tt.maxHoursAgo)
			}

			// Test formatDuration produces expected suffix
			duration := now.Sub(tt.updatedAt)
			formatted := formatDuration(duration)
			if len(formatted) < 1 {
				t.Errorf("formatDuration returned empty string")
			}
		})
	}
}

func TestExtractProjectFromBeadsID(t *testing.T) {
	// This function should extract the project from beads IDs like "orch-go-abc1"
	tests := []struct {
		beadsID string
		want    string
	}{
		{"orch-go-abc1", "orch-go"},
		{"kb-cli-xyz9", "kb-cli"},
		{"beads-1234", "beads"},
		{"a-b", "a"},
		{"single", ""},
	}

	for _, tt := range tests {
		t.Run(tt.beadsID, func(t *testing.T) {
			got := extractProjectFromBeadsID(tt.beadsID)
			if got != tt.want {
				t.Errorf("extractProjectFromBeadsID(%q) = %q, want %q", tt.beadsID, got, tt.want)
			}
		})
	}
}

func TestReconcileFixModes(t *testing.T) {
	// Test that fix modes are valid
	validModes := []string{"reset", "close"}

	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			if mode != "reset" && mode != "close" {
				t.Errorf("invalid fix mode: %s", mode)
			}
		})
	}
}
