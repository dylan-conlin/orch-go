package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestRunPeriodicTasks_NothingDue(t *testing.T) {
	// All periodic tasks disabled — should return empty result, no events
	config := daemon.DefaultConfig()
	config.ReflectEnabled = false
	config.ReflectModelDriftEnabled = false
	config.KnowledgeHealthEnabled = false
	config.CleanupEnabled = false
	config.RecoveryEnabled = false
	config.OrphanDetectionEnabled = false

	d := daemon.NewWithConfig(config)
	tmpDir := t.TempDir()
	logger := events.NewLogger(filepath.Join(tmpDir, "events.jsonl"))

	result := runPeriodicTasks(d, "12:00:00", false, logger)

	if result.KnowledgeHealthSnapshot != nil {
		t.Error("expected nil KnowledgeHealthSnapshot when task is disabled")
	}
}

func TestRunPeriodicTasks_ReflectionError(t *testing.T) {
	config := daemon.DefaultConfig()
	config.ReflectEnabled = true
	config.ReflectInterval = 1 * time.Millisecond
	config.ReflectModelDriftEnabled = false
	config.KnowledgeHealthEnabled = false
	config.CleanupEnabled = false
	config.RecoveryEnabled = false
	config.OrphanDetectionEnabled = false

	d := daemon.NewWithConfig(config)
	d.SetReflectFunc(func(createIssues bool) (*daemon.ReflectResult, error) {
		return nil, fmt.Errorf("reflect failed")
	})

	tmpDir := t.TempDir()
	logger := events.NewLogger(filepath.Join(tmpDir, "events.jsonl"))

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	runPeriodicTasks(d, "12:00:00", false, logger)

	w.Close()
	os.Stderr = oldStderr
	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	if !strings.Contains(output, "Reflection error") {
		t.Errorf("expected stderr to contain 'Reflection error', got: %s", output)
	}
}

func TestRunPeriodicTasks_CleanupLogsEvent(t *testing.T) {
	config := daemon.DefaultConfig()
	config.ReflectEnabled = false
	config.ReflectModelDriftEnabled = false
	config.KnowledgeHealthEnabled = false
	config.CleanupEnabled = true
	config.CleanupInterval = 1 * time.Millisecond
	config.RecoveryEnabled = false
	config.OrphanDetectionEnabled = false

	d := daemon.NewWithConfig(config)
	d.SetCleanupFunc(func(cfg daemon.Config) (int, string, error) {
		return 3, "Deleted 3 stale sessions", nil
	})

	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	logger := events.NewLogger(eventsPath)

	runPeriodicTasks(d, "12:00:00", false, logger)

	// Verify event was logged
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("failed to read events file: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data[:len(data)-1], &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.Type != "daemon.cleanup" {
		t.Errorf("expected event type daemon.cleanup, got %s", event.Type)
	}

	deleted, ok := event.Data["deleted"].(float64)
	if !ok || int(deleted) != 3 {
		t.Errorf("expected deleted=3, got %v", event.Data["deleted"])
	}
}

func TestRunPeriodicTasks_KnowledgeHealthSnapshot(t *testing.T) {
	config := daemon.DefaultConfig()
	config.ReflectEnabled = false
	config.ReflectModelDriftEnabled = false
	config.KnowledgeHealthEnabled = true
	config.KnowledgeHealthInterval = 1 * time.Millisecond
	config.KnowledgeHealthThreshold = 50
	config.CleanupEnabled = false
	config.RecoveryEnabled = false
	config.OrphanDetectionEnabled = false

	d := daemon.NewWithConfig(config)
	d.SetKnowledgeHealthFunc(func() (*daemon.KnowledgeHealthResult, error) {
		return &daemon.KnowledgeHealthResult{
			TotalActive: 25,
			ByType:      map[string]int{"decision": 10, "constraint": 15},
			Message:     "Knowledge health: 25 active entries",
		}, nil
	})

	tmpDir := t.TempDir()
	logger := events.NewLogger(filepath.Join(tmpDir, "events.jsonl"))

	result := runPeriodicTasks(d, "12:00:00", false, logger)

	if result.KnowledgeHealthSnapshot == nil {
		t.Fatal("expected KnowledgeHealthSnapshot to be set")
	}

	if result.KnowledgeHealthSnapshot.TotalActive != 25 {
		t.Errorf("expected TotalActive=25, got %d", result.KnowledgeHealthSnapshot.TotalActive)
	}
}

func TestRunPeriodicTasks_RecoveryErrorLogsEvent(t *testing.T) {
	// Recovery calls GetActiveAgents() directly (not mockable via setter).
	// In test environment without beads, it returns an error which should
	// produce an error event in the log.
	config := daemon.DefaultConfig()
	config.ReflectEnabled = false
	config.ReflectModelDriftEnabled = false
	config.KnowledgeHealthEnabled = false
	config.CleanupEnabled = false
	config.RecoveryEnabled = true
	config.RecoveryInterval = 1 * time.Millisecond
	config.RecoveryIdleThreshold = 1 * time.Hour
	config.RecoveryRateLimit = 1 * time.Hour
	config.OrphanDetectionEnabled = false

	d := daemon.NewWithConfig(config)

	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	logger := events.NewLogger(eventsPath)

	// Capture stderr to suppress noise
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	runPeriodicTasks(d, "12:00:00", false, logger)

	w.Close()
	os.Stderr = oldStderr

	// Recovery errored — should log an error event
	data, _ := os.ReadFile(eventsPath)
	if len(data) == 0 {
		t.Error("expected error event when recovery fails, got nothing")
		return
	}

	var event events.Event
	if err := json.Unmarshal(data[:len(data)-1], &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.Type != "daemon.recovery" {
		t.Errorf("expected event type daemon.recovery, got %s", event.Type)
	}

	if _, hasError := event.Data["error"]; !hasError {
		t.Error("expected event to have error field")
	}
}

func TestRunPeriodicTasks_OrphanDetectionLogsEvent(t *testing.T) {
	config := daemon.DefaultConfig()
	config.ReflectEnabled = false
	config.ReflectModelDriftEnabled = false
	config.KnowledgeHealthEnabled = false
	config.CleanupEnabled = false
	config.RecoveryEnabled = false
	config.OrphanDetectionEnabled = true
	config.OrphanDetectionInterval = 1 * time.Millisecond
	config.OrphanAgeThreshold = 1 * time.Hour

	d := daemon.NewWithConfig(config)
	// Mock GetActiveAgents to return no agents
	d.SetGetActiveAgentsFunc(func() ([]daemon.ActiveAgent, error) {
		return nil, nil
	})

	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	logger := events.NewLogger(eventsPath)

	runPeriodicTasks(d, "12:00:00", false, logger)

	// Orphan detection ran but found nothing — no event should be logged
	data, _ := os.ReadFile(eventsPath)
	if len(data) > 0 {
		t.Errorf("expected no events when orphan detection finds nothing, got: %s", string(data))
	}
}
