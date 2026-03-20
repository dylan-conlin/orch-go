package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestLogDaemonGateDecision_WritesEvent(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")
	logger := events.NewLogger(logPath)

	_ = logger.LogGateDecision(events.GateDecisionData{
		GateName: "ratelimit",
		Decision: "block",
		Skill:    "feature-impl",
		BeadsID:  "orch-go-abc12",
		Reason:   "Rate limited: 10/10 spawns in the last hour",
	})

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.Type != events.EventTypeSpawnGateDecision {
		t.Errorf("Type = %q, want %q", event.Type, events.EventTypeSpawnGateDecision)
	}
	if event.Data["gate_name"] != "ratelimit" {
		t.Errorf("gate_name = %q, want %q", event.Data["gate_name"], "ratelimit")
	}
	if event.Data["decision"] != "block" {
		t.Errorf("decision = %q, want %q", event.Data["decision"], "block")
	}
}

func TestLogDaemonGateDecision_ConcurrencyBlock(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")
	logger := events.NewLogger(logPath)

	_ = logger.LogGateDecision(events.GateDecisionData{
		GateName: "concurrency",
		Decision: "block",
		Skill:    "investigation",
		BeadsID:  "orch-go-xyz99",
		Reason:   "At capacity: 5/5 slots occupied",
	})

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.Data["gate_name"] != "concurrency" {
		t.Errorf("gate_name = %q, want %q", event.Data["gate_name"], "concurrency")
	}
}

func TestLogDaemonGateDecision_GovernanceWarn(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")
	logger := events.NewLogger(logPath)

	_ = logger.LogGateDecision(events.GateDecisionData{
		GateName:    "governance",
		Decision:    "warn",
		Skill:       "feature-impl",
		BeadsID:     "orch-go-gov01",
		Reason:      "task references governance-protected paths",
		TargetFiles: []string{"pkg/spawn/gates/", "_precommit.go", "pkg/verify/accretion.go"},
	})

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.Data["gate_name"] != "governance" {
		t.Errorf("gate_name = %q, want %q", event.Data["gate_name"], "governance")
	}
	if event.Data["decision"] != "warn" {
		t.Errorf("decision = %q, want %q", event.Data["decision"], "warn")
	}
	targetFiles, ok := event.Data["target_files"].([]interface{})
	if !ok || len(targetFiles) != 3 {
		t.Errorf("target_files should have 3 entries, got %v", event.Data["target_files"])
	}
}
