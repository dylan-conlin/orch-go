package main

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

func TestSynthesisWarningThreshold(t *testing.T) {
	// Verify the threshold constant matches kb-cli's SynthesisIssueThreshold
	if SynthesisWarningThreshold != 10 {
		t.Errorf("SynthesisWarningThreshold = %d, want 10 (to match kb-cli)", SynthesisWarningThreshold)
	}
}

func TestSuggestionFreshnessHours(t *testing.T) {
	// Verify the freshness check is 24 hours
	if SuggestionFreshnessHours != 24 {
		t.Errorf("SuggestionFreshnessHours = %d, want 24", SuggestionFreshnessHours)
	}
}

func TestFilterHighCountSynthesis(t *testing.T) {
	tests := []struct {
		name      string
		synthesis []daemon.SynthesisSuggestion
		wantCount int
	}{
		{
			name: "filters below threshold",
			synthesis: []daemon.SynthesisSuggestion{
				{Topic: "low", Count: 3},
				{Topic: "medium", Count: 9},
				{Topic: "high", Count: 10},
				{Topic: "veryhigh", Count: 50},
			},
			wantCount: 2, // high and veryhigh
		},
		{
			name:      "empty list",
			synthesis: []daemon.SynthesisSuggestion{},
			wantCount: 0,
		},
		{
			name: "all below threshold",
			synthesis: []daemon.SynthesisSuggestion{
				{Topic: "a", Count: 3},
				{Topic: "b", Count: 5},
				{Topic: "c", Count: 9},
			},
			wantCount: 0,
		},
		{
			name: "all above threshold",
			synthesis: []daemon.SynthesisSuggestion{
				{Topic: "a", Count: 10},
				{Topic: "b", Count: 20},
				{Topic: "c", Count: 30},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var highCount []daemon.SynthesisSuggestion
			for _, s := range tt.synthesis {
				if s.Count >= SynthesisWarningThreshold {
					highCount = append(highCount, s)
				}
			}
			if len(highCount) != tt.wantCount {
				t.Errorf("filtered count = %d, want %d", len(highCount), tt.wantCount)
			}
		})
	}
}

func TestSuggestionFreshnessCheck(t *testing.T) {
	tests := []struct {
		name      string
		timestamp time.Time
		wantFresh bool
	}{
		{
			name:      "fresh - 1 hour ago",
			timestamp: time.Now().Add(-1 * time.Hour),
			wantFresh: true,
		},
		{
			name:      "fresh - 23 hours ago",
			timestamp: time.Now().Add(-23 * time.Hour),
			wantFresh: true,
		},
		{
			name:      "stale - 25 hours ago",
			timestamp: time.Now().Add(-25 * time.Hour),
			wantFresh: false,
		},
		{
			name:      "stale - 48 hours ago",
			timestamp: time.Now().Add(-48 * time.Hour),
			wantFresh: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isFresh := time.Since(tt.timestamp).Hours() <= SuggestionFreshnessHours
			if isFresh != tt.wantFresh {
				t.Errorf("freshness = %v, want %v", isFresh, tt.wantFresh)
			}
		})
	}
}
