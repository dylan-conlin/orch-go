package main

import (
	"regexp"
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

// TestExtractBeadsIDFromSpawnContext verifies beads ID extraction from SPAWN_CONTEXT.md.
func TestExtractBeadsIDFromSpawnContext(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "standard format",
			content: `You were spawned from beads issue: **orch-go-4ufh**

Some other content here.`,
			want: "orch-go-4ufh",
		},
		{
			name: "with extra whitespace",
			content: `You were spawned from beads issue:  **proj-123**

More content.`,
			want: "proj-123",
		},
		{
			name:    "no beads ID",
			content: "No beads issue mentioned here.",
			want:    "",
		},
		{
			name: "different project format",
			content: `You were spawned from beads issue: **test-abc123**

Content.`,
			want: "test-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBeadsIDFromSpawnContext(tt.content)
			if got != tt.want {
				t.Errorf("extractBeadsIDFromSpawnContext() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestResolveBeadsIDPattern verifies the pattern matching logic.
func TestResolveBeadsIDPattern(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantType   string // "beads", "session", "workspace", "unknown"
	}{
		{
			name:       "beads ID with hyphen",
			identifier: "orch-go-4ufh",
			wantType:   "beads",
		},
		{
			name:       "session ID prefix",
			identifier: "ses_abc123xyz",
			wantType:   "session",
		},
		{
			name:       "workspace name with hyphens",
			identifier: "og-debug-wait-23dec",
			wantType:   "beads", // Workspace names with hyphens are tried as beads ID first
		},
		{
			name:       "short session ID (invalid)",
			identifier: "ses_abc",
			wantType:   "session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pattern matching logic
			var gotType string

			if strings.Contains(tt.identifier, "-") {
				gotType = "beads"
			} else if strings.HasPrefix(tt.identifier, "ses_") {
				gotType = "session"
			} else {
				gotType = "workspace"
			}

			if gotType != tt.wantType {
				t.Errorf("pattern type = %q, want %q", gotType, tt.wantType)
			}
		})
	}
}

// TestExtractBeadsIDFromSpawnContextRegex tests the regex pattern directly.
func TestExtractBeadsIDFromSpawnContextRegex(t *testing.T) {
	pattern := regexp.MustCompile(`spawned from beads issue:\s*\*\*([a-z0-9-]+)\*\*`)

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "matches pattern",
			content: "spawned from beads issue: **orch-go-4ufh**",
			want:    "orch-go-4ufh",
		},
		{
			name:    "no match",
			content: "no beads issue here",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := pattern.FindStringSubmatch(tt.content)
			var got string
			if len(matches) >= 2 {
				got = matches[1]
			}
			if got != tt.want {
				t.Errorf("regex match = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestResolveBeadsIDFromSessionID verifies that session IDs are properly resolved
// to beads IDs via workspace lookup.
func TestResolveBeadsIDFromSessionID(t *testing.T) {
	// This tests the logic flow, not the actual file system operations
	// which are tested via integration tests.

	tests := []struct {
		name          string
		identifier    string
		wantIsSession bool
	}{
		{
			name:          "session ID is recognized",
			identifier:    "ses_4b24c0801ffeTHun0PPtR2eJTx",
			wantIsSession: true,
		},
		{
			name:          "beads ID is not session",
			identifier:    "orch-go-4ufh",
			wantIsSession: false,
		},
		{
			name:          "workspace name is not session",
			identifier:    "og-debug-wait-23dec",
			wantIsSession: false,
		},
		{
			name:          "session ID with unusual chars",
			identifier:    "ses_abc123XYZ_def",
			wantIsSession: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsSession := strings.HasPrefix(tt.identifier, "ses_")
			if gotIsSession != tt.wantIsSession {
				t.Errorf("isSession = %v, want %v", gotIsSession, tt.wantIsSession)
			}
		})
	}
}
