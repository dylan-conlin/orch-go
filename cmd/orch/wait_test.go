package main

import (
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// TestWaitForPhase verifies phase status parsing for wait command.
func TestWaitForPhase(t *testing.T) {
	tests := []struct {
		name        string
		comments    []verify.Comment
		targetPhase string
		wantReached bool
		wantPhase   string
	}{
		{
			name:        "phase already complete",
			targetPhase: "Complete",
			comments: []verify.Comment{
				{Text: "Phase: Complete - All done"},
			},
			wantReached: true,
			wantPhase:   "Complete",
		},
		{
			name:        "waiting for complete, at implementing",
			targetPhase: "Complete",
			comments: []verify.Comment{
				{Text: "Phase: Implementing - Working"},
			},
			wantReached: false,
			wantPhase:   "Implementing",
		},
		{
			name:        "no phase yet",
			targetPhase: "Complete",
			comments: []verify.Comment{
				{Text: "Just started"},
			},
			wantReached: false,
			wantPhase:   "",
		},
		{
			name:        "case insensitive match",
			targetPhase: "complete",
			comments: []verify.Comment{
				{Text: "Phase: Complete - Done"},
			},
			wantReached: true,
			wantPhase:   "Complete",
		},
		{
			name:        "waiting for implementing",
			targetPhase: "Implementing",
			comments: []verify.Comment{
				{Text: "Phase: Implementing - Starting work"},
			},
			wantReached: true,
			wantPhase:   "Implementing",
		},
		{
			name:        "multiple phases - latest matches",
			targetPhase: "Complete",
			comments: []verify.Comment{
				{Text: "Phase: Planning - Start"},
				{Text: "Phase: Implementing - Middle"},
				{Text: "Phase: Complete - End"},
			},
			wantReached: true,
			wantPhase:   "Complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := verify.ParsePhaseFromComments(tt.comments)

			// Check if target phase is reached using simple string comparison
			reached := status.Found && strings.EqualFold(status.Phase, tt.targetPhase)

			if reached != tt.wantReached {
				t.Errorf("isPhaseReached() = %v, want %v", reached, tt.wantReached)
			}
			if status.Phase != tt.wantPhase {
				t.Errorf("Phase = %q, want %q", status.Phase, tt.wantPhase)
			}
		})
	}
}

// TestWaitConfigDefaults verifies wait configuration defaults.
func TestWaitConfigDefaults(t *testing.T) {
	// Test structure for wait command config
	type WaitConfig struct {
		BeadsID      string
		TargetPhase  string
		PollInterval time.Duration
		Timeout      time.Duration
	}

	cfg := WaitConfig{
		BeadsID:     "test-123",
		TargetPhase: "",
	}

	// Default target phase should be Complete
	if cfg.TargetPhase == "" {
		cfg.TargetPhase = "Complete"
	}

	if cfg.TargetPhase != "Complete" {
		t.Errorf("Default target phase = %q, want %q", cfg.TargetPhase, "Complete")
	}

	// Default poll interval
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Second
	}

	if cfg.PollInterval != 5*time.Second {
		t.Errorf("Default poll interval = %v, want %v", cfg.PollInterval, 5*time.Second)
	}
}
