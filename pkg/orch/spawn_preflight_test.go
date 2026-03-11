package orch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestIsArchitectIssue(t *testing.T) {
	tests := []struct {
		name  string
		issue *verify.Issue
		want  bool
	}{
		{
			name: "skill:architect label",
			issue: &verify.Issue{
				Title:     "some task",
				IssueType: "task",
				Labels:    []string{"skill:architect"},
			},
			want: true,
		},
		{
			name: "architect in title",
			issue: &verify.Issue{
				Title:     "[orch-go] architect: design extraction first",
				IssueType: "task",
				Labels:    nil,
			},
			want: true,
		},
		{
			name: "feature-impl issue",
			issue: &verify.Issue{
				Title:     "[orch-go] feature-impl: add hotspot gate",
				IssueType: "task",
				Labels:    []string{"skill:feature-impl"},
			},
			want: false,
		},
		{
			name: "no labels or architect title",
			issue: &verify.Issue{
				Title:     "fix something",
				IssueType: "bug",
				Labels:    nil,
			},
			want: false,
		},
		{
			name: "architect label among others",
			issue: &verify.Issue{
				Title:     "review hotspot area",
				IssueType: "task",
				Labels:    []string{"priority:high", "skill:architect", "area:spawn"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isArchitectIssue(tt.issue)
			if got != tt.want {
				t.Errorf("isArchitectIssue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractSearchTerms(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		contains []string // expected terms that should be in result
	}{
		{
			name:     "single file with full path",
			files:    []string{"pkg/orch/extraction.go"},
			contains: []string{"pkg/orch/extraction.go", "extraction"},
		},
		{
			name:     "basename only",
			files:    []string{"daemon.go"},
			contains: []string{"daemon"},
		},
		{
			name:     "multiple files",
			files:    []string{"cmd/orch/main.go", "pkg/daemon/daemon.go"},
			contains: []string{"cmd/orch/main.go", "main", "pkg/daemon/daemon.go", "daemon"},
		},
		{
			name:     "empty list",
			files:    []string{},
			contains: []string{},
		},
		{
			name:     "empty string in list",
			files:    []string{""},
			contains: []string{},
		},
		{
			name:     "case normalization",
			files:    []string{"Cmd/Orch/Main.Go"},
			contains: []string{"cmd/orch/main.go", "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := extractSearchTerms(tt.files)
			for _, expected := range tt.contains {
				found := false
				for _, term := range terms {
					if term == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("extractSearchTerms(%v) missing expected term %q, got %v", tt.files, expected, terms)
				}
			}
		})
	}
}

func TestLogGateDecision_IncludesBeadsID(t *testing.T) {
	// Override events log path to a temp file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")

	// Directly test the logger (logGateDecision is a thin wrapper)
	logger := events.NewLogger(logPath)
	err := logger.LogGateDecision(events.GateDecisionData{
		GateName: "triage",
		Decision: "allow",
		Skill:    "feature-impl",
		BeadsID:  "orch-go-xyz99",
		Reason:   "daemon-driven spawn",
	})
	if err != nil {
		t.Fatalf("LogGateDecision() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Verify beads_id is in the event data
	if event.Data["beads_id"] != "orch-go-xyz99" {
		t.Errorf("data.beads_id = %v, want %q", event.Data["beads_id"], "orch-go-xyz99")
	}
	// Verify session_id is also set (used for correlation)
	if event.SessionID != "orch-go-xyz99" {
		t.Errorf("event.SessionID = %q, want %q", event.SessionID, "orch-go-xyz99")
	}
}
