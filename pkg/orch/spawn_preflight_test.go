package orch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
)

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

	data, err := os.ReadFile(logger.CurrentPath())
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

func TestHotspotAutoBypassEmitsGateDecision(t *testing.T) {
	// When CheckHotspot returns HasCriticalHotspot=true but no error (auto-detected
	// prior architect review), spawn_preflight should emit a "bypass" gate_decision
	// event with reason "auto-detected prior architect review".
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")

	logger := events.NewLogger(logPath)
	// Simulate what spawn_preflight.go does for auto-bypass:
	// forceHotspot=false, hotspotResult.HasCriticalHotspot=true, err=nil
	hotspotResult := &gates.HotspotResult{
		HasCriticalHotspot: true,
		CriticalFiles:      []string{"cmd/orch/stats_cmd.go"},
	}
	forceHotspot := false

	// Replicate the spawn_preflight gate decision logic
	if forceHotspot && hotspotResult != nil && hotspotResult.HasCriticalHotspot {
		t.Fatal("should not take forceHotspot path")
	} else if hotspotResult != nil && hotspotResult.HasCriticalHotspot {
		_ = logger.LogGateDecision(events.GateDecisionData{
			GateName:    "hotspot",
			Decision:    "bypass",
			Skill:       "feature-impl",
			BeadsID:     "orch-go-test-auto",
			Reason:      "auto-detected prior architect review",
			TargetFiles: hotspotResult.CriticalFiles,
		})
	} else {
		t.Fatal("should not take allow path")
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Data["decision"] != "bypass" {
		t.Errorf("decision = %v, want %q", event.Data["decision"], "bypass")
	}
	if event.Data["reason"] != "auto-detected prior architect review" {
		t.Errorf("reason = %v, want %q", event.Data["reason"], "auto-detected prior architect review")
	}
	targetFiles, ok := event.Data["target_files"].([]interface{})
	if !ok || len(targetFiles) != 1 {
		t.Errorf("target_files = %v, want [cmd/orch/stats_cmd.go]", event.Data["target_files"])
	}
}

func TestGovernanceGateEmitsWarnEvent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")

	logger := events.NewLogger(logPath)
	// Simulate what spawn_preflight.go does when governance matches
	_ = logger.LogGateDecision(events.GateDecisionData{
		GateName:    "governance",
		Decision:    "warn",
		Skill:       "feature-impl",
		BeadsID:     "orch-go-gov01",
		Reason:      "task references governance-protected paths",
		TargetFiles: []string{"pkg/spawn/gates/", "_precommit.go", "pkg/verify/accretion.go"},
	})

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Data["gate_name"] != "governance" {
		t.Errorf("gate_name = %v, want %q", event.Data["gate_name"], "governance")
	}
	if event.Data["decision"] != "warn" {
		t.Errorf("decision = %v, want %q", event.Data["decision"], "warn")
	}
	targetFiles, ok := event.Data["target_files"].([]interface{})
	if !ok || len(targetFiles) != 3 {
		t.Errorf("target_files should have 3 entries, got %v", event.Data["target_files"])
	}
}

func TestLogGateDecision_AllowDecisions(t *testing.T) {
	// Verify that allow events for gates produce valid spawn.gate_decision events.
	tests := []struct {
		name     string
		gate     string
		decision string
		reason   string
	}{
		{
			name:     "accretion_precommit allow",
			gate:     "accretion_precommit",
			decision: "allow",
			reason:   "staged files within accretion threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "events.jsonl")

			logger := events.NewLogger(logPath)
			err := logger.LogGateDecision(events.GateDecisionData{
				GateName: tt.gate,
				Decision: tt.decision,
				Skill:    "feature-impl",
				BeadsID:  "orch-go-test1",
				Reason:   tt.reason,
			})
			if err != nil {
				t.Fatalf("LogGateDecision() error = %v", err)
			}

			data, err := os.ReadFile(logger.CurrentPath())
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}

			var event events.Event
			if err := json.Unmarshal(data, &event); err != nil {
				t.Fatalf("Failed to unmarshal event: %v", err)
			}

			if event.Type != events.EventTypeSpawnGateDecision {
				t.Errorf("event.Type = %q, want %q", event.Type, events.EventTypeSpawnGateDecision)
			}
			if event.Data["gate_name"] != tt.gate {
				t.Errorf("gate_name = %v, want %q", event.Data["gate_name"], tt.gate)
			}
			if event.Data["decision"] != tt.decision {
				t.Errorf("decision = %v, want %q", event.Data["decision"], tt.decision)
			}
			if event.Data["reason"] != tt.reason {
				t.Errorf("reason = %v, want %q", event.Data["reason"], tt.reason)
			}
		})
	}
}
