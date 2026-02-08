package daemon

import (
	"path/filepath"
	"testing"
	"time"
)

func TestQueueDiagnosticsForIssues_CountsGraceAndSlotPressure(t *testing.T) {
	d := &Daemon{
		Config: Config{GracePeriod: 30 * time.Second},
		Pool:   NewWorkerPool(1),
		firstSeen: map[string]time.Time{
			"issue-a": time.Now().Add(-2 * time.Minute),
			"issue-b": time.Now().Add(-2 * time.Minute),
		},
	}

	diagnostics := d.QueueDiagnosticsForIssues([]Issue{
		{ID: "issue-a", IssueType: "task", Status: "open"},
		{ID: "issue-b", IssueType: "task", Status: "open"},
		{ID: "issue-c", IssueType: "task", Status: "open"},
	})

	if diagnostics.Queued != 3 {
		t.Fatalf("Queued = %d, want 3", diagnostics.Queued)
	}
	if diagnostics.GracePeriod != 1 {
		t.Fatalf("GracePeriod = %d, want 1", diagnostics.GracePeriod)
	}
	if diagnostics.Spawnable != 2 {
		t.Fatalf("Spawnable = %d, want 2", diagnostics.Spawnable)
	}
	if diagnostics.WaitingForSlots != 1 {
		t.Fatalf("WaitingForSlots = %d, want 1", diagnostics.WaitingForSlots)
	}
}

func TestQueueDiagnosticsForIssues_CountsProcessedCache(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewProcessedIssueCache(filepath.Join(tmpDir, "processed-issues.jsonl"), 100, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}
	cache.sessionChecker = func(beadsID string) bool { return false }
	cache.phaseCompleteChecker = func(beadsID string) (bool, error) { return false, nil }
	if err := cache.MarkProcessed("issue-a"); err != nil {
		t.Fatalf("MarkProcessed failed: %v", err)
	}

	d := &Daemon{
		Config:         Config{GracePeriod: 0},
		Pool:           NewWorkerPool(5),
		ProcessedCache: cache,
	}

	diagnostics := d.QueueDiagnosticsForIssues([]Issue{
		{ID: "issue-a", IssueType: "task", Status: "open"},
		{ID: "issue-b", IssueType: "task", Status: "open"},
	})

	if diagnostics.ProcessedCache != 1 {
		t.Fatalf("ProcessedCache = %d, want 1", diagnostics.ProcessedCache)
	}
	if diagnostics.Spawnable != 1 {
		t.Fatalf("Spawnable = %d, want 1", diagnostics.Spawnable)
	}
	if diagnostics.WaitingForSlots != 0 {
		t.Fatalf("WaitingForSlots = %d, want 0", diagnostics.WaitingForSlots)
	}
}

func TestQueueDiagnosticsForIssues_DoesNotMutateFirstSeen(t *testing.T) {
	d := &Daemon{
		Config:    Config{GracePeriod: 30 * time.Second},
		firstSeen: map[string]time.Time{},
	}

	_ = d.QueueDiagnosticsForIssues([]Issue{{ID: "issue-a", IssueType: "task", Status: "open"}})

	if len(d.firstSeen) != 0 {
		t.Fatalf("firstSeen mutated by diagnostics, got %d entries", len(d.firstSeen))
	}
}
