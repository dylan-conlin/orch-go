package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleHarnessReport_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/harness", nil)
	w := httptest.NewRecorder()
	handleHarnessReport(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleHarnessReport_EmptyEvents(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/harness", nil)
	w := httptest.NewRecorder()
	handleHarnessReport(w, req)

	// Should return 200 with empty data (no events file is fine)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp HarnessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Pipeline) != 4 {
		t.Errorf("expected 4 pipeline stages, got %d", len(resp.Pipeline))
	}

	if len(resp.FalsificationVerdicts) != 4 {
		t.Errorf("expected 4 falsification verdicts, got %d", len(resp.FalsificationVerdicts))
	}

	if resp.MeasurementCoverage.TotalComponents != 13 {
		t.Errorf("expected 13 total components, got %d", resp.MeasurementCoverage.TotalComponents)
	}
}

func TestBuildHarnessResponse_WithEvents(t *testing.T) {
	now := time.Now().Unix() - 3600 // 1 hour ago — within any reasonable window
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now},
		{Type: "session.spawned", Timestamp: now},
		{Type: "session.spawned", Timestamp: now},
		{Type: "spawn.triage_bypassed", Timestamp: now},
		{Type: "spawn.hotspot_bypassed", Timestamp: now, Data: map[string]interface{}{"reason": "test"}},
		{Type: "spawn.hotspot_bypassed", Timestamp: now},
		{Type: "spawn.gate_decision", Timestamp: now, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "block",
		}},
		{Type: "agent.completed", Timestamp: now, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success", "duration_seconds": 2700.0,
		}},
		{Type: "agent.completed", Timestamp: now, Data: map[string]interface{}{
			"skill": "investigation",
		}},
	}

	resp := buildHarnessResponse(events, 30)

	if resp.TotalSpawns != 3 {
		t.Errorf("expected 3 total spawns, got %d", resp.TotalSpawns)
	}

	// Check spawn stage
	spawnStage := resp.Pipeline[0]
	if spawnStage.Stage != "spawn" {
		t.Errorf("expected first stage 'spawn', got '%s'", spawnStage.Stage)
	}

	// Triage gate: 1 bypass out of 3 spawns
	triageGate := spawnStage.Components[0]
	if triageGate.Name != "triage_gate" {
		t.Errorf("expected triage_gate, got %s", triageGate.Name)
	}
	if triageGate.Bypassed != 1 {
		t.Errorf("expected 1 triage bypass, got %d", triageGate.Bypassed)
	}

	// Hotspot gate: 2 legacy bypasses + 1 gate_decision block = 3 total fires
	hotspotGate := spawnStage.Components[1]
	if hotspotGate.Blocked != 1 {
		t.Errorf("expected 1 hotspot block, got %d", hotspotGate.Blocked)
	}
	if hotspotGate.Bypassed != 2 {
		t.Errorf("expected 2 hotspot bypasses, got %d", hotspotGate.Bypassed)
	}
	if hotspotGate.MeasurementStatus != "flowing" {
		t.Errorf("expected 'flowing' measurement, got '%s'", hotspotGate.MeasurementStatus)
	}

	// Completion coverage
	if resp.CompletionCoverage.TotalCompletions != 2 {
		t.Errorf("expected 2 completions, got %d", resp.CompletionCoverage.TotalCompletions)
	}
	if resp.CompletionCoverage.WithSkill != 2 {
		t.Errorf("expected 2 with skill, got %d", resp.CompletionCoverage.WithSkill)
	}
	if resp.CompletionCoverage.WithDuration != 1 {
		t.Errorf("expected 1 with duration, got %d", resp.CompletionCoverage.WithDuration)
	}

	// Falsification: gates_are_irrelevant should be falsified (high fire rate)
	if v, ok := resp.FalsificationVerdicts["gates_are_irrelevant"]; ok {
		if v.Status != "falsified" {
			t.Errorf("expected gates_are_irrelevant to be 'falsified', got '%s'", v.Status)
		}
	} else {
		t.Error("missing gates_are_irrelevant verdict")
	}
}

func TestBuildVerdicts_LowFireRate(t *testing.T) {
	// 100 spawns, 2 bypasses total = 2% fire rate < 5% threshold
	verdicts := buildVerdicts(100, 1, 1, 0, nil)
	v := verdicts["gates_are_irrelevant"]
	if v.Status != "confirmed" {
		t.Errorf("expected 'confirmed' for low fire rate, got '%s'", v.Status)
	}
}

func TestBuildVerdicts_NoSpawns(t *testing.T) {
	verdicts := buildVerdicts(0, 0, 0, 0, nil)
	v := verdicts["gates_are_irrelevant"]
	if v.Status != "insufficient_data" {
		t.Errorf("expected 'insufficient_data' with no spawns, got '%s'", v.Status)
	}
}

func TestStatusFromCount(t *testing.T) {
	if statusFromCount(0) != "unmeasured" {
		t.Error("expected unmeasured for 0")
	}
	if statusFromCount(5) != "flowing" {
		t.Error("expected flowing for 5")
	}
}
