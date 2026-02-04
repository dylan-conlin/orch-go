package attention

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEpicOrphanCollector_Collect(t *testing.T) {
	// Create temp directory for test events file
	tmpDir, err := os.MkdirTemp("", "epic-orphan-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	eventsPath := filepath.Join(tmpDir, "events.jsonl")

	// Create test events
	events := []struct {
		Type      string `json:"type"`
		Timestamp int64  `json:"timestamp"`
		Data      map[string]any `json:"data,omitempty"`
	}{
		{
			Type:      "session.spawned",
			Timestamp: time.Now().Unix(),
			Data:      map[string]any{"session_id": "test-session"},
		},
		{
			Type:      "epic.orphaned",
			Timestamp: time.Now().Unix(),
			Data: map[string]any{
				"epic_id":           "orch-go-test1",
				"epic_title":        "Test Epic 1",
				"orphaned_children": []string{"orch-go-test1.1", "orch-go-test1.2"},
				"reason":            "Force closed by user",
			},
		},
		{
			Type:      "epic.orphaned",
			Timestamp: time.Now().Add(-8 * 24 * time.Hour).Unix(), // 8 days ago - should be filtered out
			Data: map[string]any{
				"epic_id":           "orch-go-old",
				"epic_title":        "Old Epic",
				"orphaned_children": []string{"orch-go-old.1"},
				"reason":            "Force closed",
			},
		},
	}

	// Write events to file
	f, err := os.Create(eventsPath)
	if err != nil {
		t.Fatalf("failed to create events file: %v", err)
	}
	encoder := json.NewEncoder(f)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			t.Fatalf("failed to write event: %v", err)
		}
	}
	f.Close()

	// Create collector with custom events path
	collector := &EpicOrphanCollector{eventsPath: eventsPath}

	// Collect items
	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() failed: %v", err)
	}

	// Verify results
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	if len(items) > 0 {
		item := items[0]
		if item.Subject != "orch-go-test1" {
			t.Errorf("expected subject 'orch-go-test1', got '%s'", item.Subject)
		}
		if item.Signal != "epic-orphaned" {
			t.Errorf("expected signal 'epic-orphaned', got '%s'", item.Signal)
		}
		if item.Concern != Authority {
			t.Errorf("expected concern Authority, got %v", item.Concern)
		}
		if item.Source != "epic-orphan" {
			t.Errorf("expected source 'epic-orphan', got '%s'", item.Source)
		}
	}
}

func TestEpicOrphanCollector_NoEventsFile(t *testing.T) {
	collector := &EpicOrphanCollector{eventsPath: "/nonexistent/path/events.jsonl"}

	items, err := collector.Collect("human")
	if err != nil {
		t.Errorf("Collect() should not error for missing file, got: %v", err)
	}
	if items != nil && len(items) > 0 {
		t.Errorf("expected nil or empty items, got %d items", len(items))
	}
}

func TestEpicOrphanCollector_PriorityByRole(t *testing.T) {
	// Create temp directory for test events file
	tmpDir, err := os.MkdirTemp("", "epic-orphan-priority-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	eventsPath := filepath.Join(tmpDir, "events.jsonl")

	// Create a single test event
	event := struct {
		Type      string `json:"type"`
		Timestamp int64  `json:"timestamp"`
		Data      map[string]any `json:"data,omitempty"`
	}{
		Type:      "epic.orphaned",
		Timestamp: time.Now().Unix(),
		Data: map[string]any{
			"epic_id":           "orch-go-test",
			"epic_title":        "Test Epic",
			"orphaned_children": []string{"orch-go-test.1"},
			"reason":            "Force closed",
		},
	}

	f, err := os.Create(eventsPath)
	if err != nil {
		t.Fatalf("failed to create events file: %v", err)
	}
	json.NewEncoder(f).Encode(event)
	f.Close()

	collector := &EpicOrphanCollector{eventsPath: eventsPath}

	tests := []struct {
		role         string
		expectedPrio int
	}{
		{"human", 30},
		{"orchestrator", 40},
		{"daemon", 60},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			items, _ := collector.Collect(tt.role)
			if len(items) == 0 {
				t.Fatalf("expected items, got none")
			}
			if items[0].Priority != tt.expectedPrio {
				t.Errorf("expected priority %d for role %s, got %d", tt.expectedPrio, tt.role, items[0].Priority)
			}
		})
	}
}
