package main

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestBuildPostCompletionFailureComment(t *testing.T) {
	tests := []struct {
		name        string
		failureType string
		description string
		want        string
	}{
		{
			name:        "verification failure",
			failureType: spawn.FailureTypeVerification,
			description: "Agent claimed tests pass but feature doesn't work",
			want:        "POST-COMPLETION-FAILURE: verification - Agent claimed tests pass but feature doesn't work",
		},
		{
			name:        "implementation failure",
			failureType: spawn.FailureTypeImplementation,
			description: "Code has a bug in the handler",
			want:        "POST-COMPLETION-FAILURE: implementation - Code has a bug in the handler",
		},
		{
			name:        "spec failure",
			failureType: spawn.FailureTypeSpec,
			description: "Spec was wrong about the API format",
			want:        "POST-COMPLETION-FAILURE: spec - Spec was wrong about the API format",
		},
		{
			name:        "integration failure",
			failureType: spawn.FailureTypeIntegration,
			description: "Works in isolation but fails with the dashboard",
			want:        "POST-COMPLETION-FAILURE: integration - Works in isolation but fails with the dashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPostCompletionFailureComment(tt.failureType, tt.description)
			if got != tt.want {
				t.Errorf("buildPostCompletionFailureComment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildBdReopenCommand(t *testing.T) {
	tests := []struct {
		name    string
		beadsID string
		reason  string
		wantCmd string
	}{
		{
			name:    "reopen with reason",
			beadsID: "orch-go-1234",
			reason:  "Feature doesn't work",
			wantCmd: "reopen orch-go-1234 --reason Feature doesn't work",
		},
		{
			name:    "reopen without reason",
			beadsID: "orch-go-5678",
			reason:  "",
			wantCmd: "reopen orch-go-5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildBdReopenArgs(tt.beadsID, tt.reason)
			got := strings.Join(args, " ")
			if got != tt.wantCmd {
				t.Errorf("buildBdReopenArgs() = %q, want %q", got, tt.wantCmd)
			}
		})
	}
}

func TestValidateFailureType(t *testing.T) {
	tests := []struct {
		name        string
		failureType string
		wantValid   bool
	}{
		{name: "verification", failureType: "verification", wantValid: true},
		{name: "implementation", failureType: "implementation", wantValid: true},
		{name: "spec", failureType: "spec", wantValid: true},
		{name: "integration", failureType: "integration", wantValid: true},
		{name: "empty", failureType: "", wantValid: false},
		{name: "unknown", failureType: "unknown", wantValid: false},
		{name: "typo", failureType: "verificaton", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidFailureType(tt.failureType)
			if got != tt.wantValid {
				t.Errorf("isValidFailureType(%q) = %v, want %v", tt.failureType, got, tt.wantValid)
			}
		})
	}
}

func TestBuildReworkSummary(t *testing.T) {
	tests := []struct {
		name        string
		beadsID     string
		failureType string
		description string
		wantParts   []string // Substrings that must be present
	}{
		{
			name:        "basic summary",
			beadsID:     "orch-go-1234",
			failureType: "implementation",
			description: "Button click doesn't close the dialog",
			wantParts:   []string{"orch-go-1234", "implementation", "Button click", "systematic-debugging"},
		},
		{
			name:        "verification summary with skill suggestion",
			beadsID:     "orch-go-5678",
			failureType: "verification",
			description: "Agent claimed it was done but feature is broken",
			wantParts:   []string{"orch-go-5678", "verification", "reliability-testing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildReworkSummary(tt.beadsID, tt.failureType, tt.description)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("buildReworkSummary() missing %q in output:\n%s", part, got)
				}
			}
		})
	}
}
