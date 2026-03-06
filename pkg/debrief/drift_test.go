package debrief

import (
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestCollectDriftSummary(t *testing.T) {
	today := time.Date(2026, 3, 5, 14, 0, 0, 0, time.UTC)
	todayStr := "2026-03-05"

	events := []spawn.StalenessEvent{
		{
			Timestamp:    todayStr + "T10:00:00Z",
			Model:        ".kb/models/agent-lifecycle/MODEL.md",
			ChangedFiles: []string{"cmd/orch/status_cmd.go"},
			SpawnID:      "spawn-1",
		},
		{
			Timestamp:    todayStr + "T11:00:00Z",
			Model:        ".kb/models/agent-lifecycle/MODEL.md",
			ChangedFiles: []string{"cmd/orch/status_cmd.go", "pkg/verify/review.go"},
			SpawnID:      "spawn-2",
		},
		{
			Timestamp:    todayStr + "T12:00:00Z",
			Model:        ".kb/models/completion-verification/MODEL.md",
			DeletedFiles: []string{"old_file.go"},
			SpawnID:      "spawn-3",
		},
		// Yesterday's event — should be filtered out
		{
			Timestamp: "2026-03-04T23:00:00Z",
			Model:     ".kb/models/agent-lifecycle/MODEL.md",
			SpawnID:   "spawn-old",
		},
	}

	items := CollectDriftSummary(events, today)
	if len(items) != 2 {
		t.Fatalf("expected 2 drift items, got %d", len(items))
	}

	// First should be agent-lifecycle (2 spawns)
	if items[0].SpawnCount != 2 {
		t.Errorf("expected 2 spawns for agent-lifecycle, got %d", items[0].SpawnCount)
	}
	if items[0].Domain != "agent-lifecycle" {
		t.Errorf("expected domain 'agent-lifecycle', got %q", items[0].Domain)
	}
	// Deduplicated changed files: status_cmd.go + review.go = 2
	if items[0].ChangedFiles != 2 {
		t.Errorf("expected 2 changed files, got %d", items[0].ChangedFiles)
	}

	// Second should be completion-verification (1 spawn)
	if items[1].SpawnCount != 1 {
		t.Errorf("expected 1 spawn for completion-verification, got %d", items[1].SpawnCount)
	}
	if items[1].DeletedFiles != 1 {
		t.Errorf("expected 1 deleted file, got %d", items[1].DeletedFiles)
	}
}

func TestCollectDriftSummaryEmpty(t *testing.T) {
	items := CollectDriftSummary(nil, time.Now())
	if items != nil {
		t.Errorf("expected nil for empty events, got %d items", len(items))
	}
}

func TestCollectDriftSummaryNoTodayEvents(t *testing.T) {
	today := time.Date(2026, 3, 5, 14, 0, 0, 0, time.UTC)
	events := []spawn.StalenessEvent{
		{
			Timestamp: "2026-03-04T10:00:00Z",
			Model:     ".kb/models/agent-lifecycle/MODEL.md",
			SpawnID:   "spawn-1",
		},
	}

	items := CollectDriftSummary(events, today)
	if items != nil {
		t.Errorf("expected nil for no today events, got %d items", len(items))
	}
}

func TestFormatDriftSummary(t *testing.T) {
	items := []DriftItem{
		{Domain: "agent-lifecycle", SpawnCount: 3, ChangedFiles: 2, DeletedFiles: 0},
		{Domain: "completion-verification", SpawnCount: 1, ChangedFiles: 0, DeletedFiles: 1},
	}

	lines := FormatDriftSummary(items)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "agent-lifecycle") {
		t.Errorf("expected domain name in line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "3 stale spawn(s)") {
		t.Errorf("expected spawn count in line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "2 changed") {
		t.Errorf("expected changed count in line, got: %s", lines[0])
	}

	if !strings.Contains(lines[1], "1 deleted") {
		t.Errorf("expected deleted count in second line, got: %s", lines[1])
	}
}

func TestFormatDriftSummaryEmpty(t *testing.T) {
	lines := FormatDriftSummary(nil)
	if lines != nil {
		t.Errorf("expected nil for empty items, got %d lines", len(lines))
	}
}

func TestDomainFromModelPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{".kb/models/agent-lifecycle-state-model/MODEL.md", "agent-lifecycle-state-model"},
		{"/Users/dylan/.kb/models/completion-verification/MODEL.md", "completion-verification"},
		{".kb/models/simple.md", "simple"},
		{"unknown/path/file.md", "file"},
	}

	for _, tt := range tests {
		got := domainFromModelPath(tt.path)
		if got != tt.expected {
			t.Errorf("domainFromModelPath(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}
