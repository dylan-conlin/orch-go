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

	if resp.MeasurementCoverage.TotalComponents != 12 {
		t.Errorf("expected 12 total components, got %d", resp.MeasurementCoverage.TotalComponents)
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

func TestAccretionGate_CollectingStatus(t *testing.T) {
	now := time.Now().Unix() - 3600
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now},
		{Type: "accretion.snapshot", Timestamp: now, Data: map[string]interface{}{
			"total_lines": float64(5000), "directory": "pkg/",
		}},
	}

	resp := buildHarnessResponse(events, 30)

	// Find accretion gate in pre_commit stage
	var accretionGate *PipelineComponent
	for _, stage := range resp.Pipeline {
		if stage.Stage == "pre_commit" {
			for i, comp := range stage.Components {
				if comp.Name == "accretion_gate" {
					accretionGate = &stage.Components[i]
					break
				}
			}
		}
	}

	if accretionGate == nil {
		t.Fatal("accretion_gate not found in pipeline")
	}

	if accretionGate.MeasurementStatus != "collecting" {
		t.Errorf("expected 'collecting' measurement status, got '%s'", accretionGate.MeasurementStatus)
	}
	if accretionGate.CollectingSince == "" {
		t.Error("expected CollectingSince to be set")
	}
}

func TestAccretionGate_UnmeasuredWithoutEvents(t *testing.T) {
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: time.Now().Unix() - 3600},
	}

	resp := buildHarnessResponse(events, 30)

	var accretionGate *PipelineComponent
	for _, stage := range resp.Pipeline {
		if stage.Stage == "pre_commit" {
			for i, comp := range stage.Components {
				if comp.Name == "accretion_gate" {
					accretionGate = &stage.Components[i]
					break
				}
			}
		}
	}

	if accretionGate == nil {
		t.Fatal("accretion_gate not found in pipeline")
	}

	if accretionGate.MeasurementStatus != "unmeasured" {
		t.Errorf("expected 'unmeasured' without accretion events, got '%s'", accretionGate.MeasurementStatus)
	}
}

func TestAccretionSnapshot_DirectoriesArrayFormat(t *testing.T) {
	// Two snapshots with directories array format (actual event format)
	// spaced 7 days apart to compute velocity
	now := time.Now().Unix()
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now - 3600},
		{Type: "accretion.snapshot", Timestamp: now - 7*86400, Data: map[string]interface{}{
			"directories": []interface{}{
				map[string]interface{}{"directory": "cmd/orch/", "total_lines": 50000.0, "file_count": 100.0},
				map[string]interface{}{"directory": "pkg/spawn/", "total_lines": 20000.0, "file_count": 50.0},
			},
			"snapshot_type": "baseline",
		}},
		{Type: "accretion.snapshot", Timestamp: now, Data: map[string]interface{}{
			"directories": []interface{}{
				map[string]interface{}{"directory": "cmd/orch/", "total_lines": 52000.0, "file_count": 105.0},
				map[string]interface{}{"directory": "pkg/spawn/", "total_lines": 21000.0, "file_count": 52.0},
			},
			"snapshot_type": "periodic",
		}},
	}

	resp := buildHarnessResponse(events, 30)

	if resp.AccretionVelocity == nil {
		t.Fatal("expected AccretionVelocity to be computed from directories-array snapshots")
	}

	// First snapshot: 70000 lines, second: 73000 lines = +3000 over 7 days = 3000/week
	if resp.AccretionVelocity.CurrentWeeklyLines != 3000 {
		t.Errorf("expected 3000 weekly lines, got %d", resp.AccretionVelocity.CurrentWeeklyLines)
	}
}

func TestAccretionSnapshot_FlatFormat(t *testing.T) {
	// Legacy flat format still works
	now := time.Now().Unix()
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now - 3600},
		{Type: "accretion.snapshot", Timestamp: now - 7*86400, Data: map[string]interface{}{
			"total_lines": 70000.0,
			"directory":   "cmd/orch/",
		}},
		{Type: "accretion.snapshot", Timestamp: now, Data: map[string]interface{}{
			"total_lines": 73000.0,
			"directory":   "cmd/orch/",
		}},
	}

	resp := buildHarnessResponse(events, 30)

	if resp.AccretionVelocity == nil {
		t.Fatal("expected AccretionVelocity from flat-format snapshots")
	}
	if resp.AccretionVelocity.CurrentWeeklyLines != 3000 {
		t.Errorf("expected 3000 weekly lines, got %d", resp.AccretionVelocity.CurrentWeeklyLines)
	}
}

func TestBuildVerdicts_LowFireRate(t *testing.T) {
	// 100 spawns, 2 bypasses total = 2% fire rate < 5% threshold
	verdicts := buildVerdicts(100, 1, 1, nil)
	v := verdicts["gates_are_irrelevant"]
	if v.Status != "confirmed" {
		t.Errorf("expected 'confirmed' for low fire rate, got '%s'", v.Status)
	}
}

func TestBuildVerdicts_NoSpawns(t *testing.T) {
	verdicts := buildVerdicts(0, 0, 0, nil)
	v := verdicts["gates_are_irrelevant"]
	if v.Status != "insufficient_data" {
		t.Errorf("expected 'insufficient_data' with no spawns, got '%s'", v.Status)
	}
}

func TestBuildHarnessResponse_ExplorationMetrics(t *testing.T) {
	now := time.Now().Unix() - 3600
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now},
		{Type: "exploration.decomposed", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-abc1", "parent_skill": "investigation",
			"breadth": 3.0, "subproblems": []interface{}{"sub1", "sub2", "sub3"},
		}},
		{Type: "exploration.judged", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-abc1", "total_findings": 8.0,
			"accepted": 5.0, "contested": 2.0, "rejected": 1.0, "coverage_gaps": 1.0,
		}},
		{Type: "exploration.synthesized", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-abc1", "worker_count": 3.0, "duration_seconds": 450.0,
		}},
		// Second exploration run (partial - no synthesis yet)
		{Type: "exploration.decomposed", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-def2", "parent_skill": "architect",
			"breadth": 2.0,
		}},
		{Type: "exploration.judged", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-def2", "total_findings": 4.0,
			"accepted": 3.0, "contested": 1.0, "rejected": 0.0, "coverage_gaps": 0.0,
		}},
	}

	resp := buildHarnessResponse(events, 30)

	if resp.ExplorationMetrics == nil {
		t.Fatal("expected ExplorationMetrics to be present")
	}

	em := resp.ExplorationMetrics
	if em.TotalRuns != 2 {
		t.Errorf("expected 2 total runs, got %d", em.TotalRuns)
	}
	if em.CompletedRuns != 1 {
		t.Errorf("expected 1 completed run, got %d", em.CompletedRuns)
	}
	if em.TotalFindings != 12 {
		t.Errorf("expected 12 total findings, got %d", em.TotalFindings)
	}
	if em.TotalAccepted != 8 {
		t.Errorf("expected 8 accepted, got %d", em.TotalAccepted)
	}
	if em.TotalContested != 3 {
		t.Errorf("expected 3 contested, got %d", em.TotalContested)
	}
	if em.TotalRejected != 1 {
		t.Errorf("expected 1 rejected, got %d", em.TotalRejected)
	}
	if em.AvgWorkersPerRun != 2.5 {
		t.Errorf("expected 2.5 avg workers, got %f", em.AvgWorkersPerRun)
	}
}

func TestBuildHarnessResponse_ExplorationWithIteration(t *testing.T) {
	now := time.Now().Unix() - 3600
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now},
		{Type: "exploration.decomposed", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-iter1", "parent_skill": "investigation",
			"breadth": 3.0,
		}},
		{Type: "exploration.judged", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-iter1", "total_findings": 6.0,
			"accepted": 3.0, "contested": 1.0, "rejected": 0.0, "coverage_gaps": 2.0,
		}},
		{Type: "exploration.iterated", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-iter1", "iteration": 2.0, "gaps_addressed": 2.0, "new_workers": 2.0,
		}},
		{Type: "exploration.judged", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-iter1", "total_findings": 8.0,
			"accepted": 6.0, "contested": 1.0, "rejected": 1.0, "coverage_gaps": 0.0,
		}},
		{Type: "exploration.synthesized", Timestamp: now, Data: map[string]interface{}{
			"beads_id": "orch-go-iter1", "worker_count": 5.0,
		}},
	}

	resp := buildHarnessResponse(events, 30)

	if resp.ExplorationMetrics == nil {
		t.Fatal("expected ExplorationMetrics to be present")
	}

	em := resp.ExplorationMetrics
	if em.TotalIterations != 1 {
		t.Errorf("expected 1 total iteration, got %d", em.TotalIterations)
	}
	if em.IteratedRuns != 1 {
		t.Errorf("expected 1 iterated run, got %d", em.IteratedRuns)
	}
	if em.TotalRuns != 1 {
		t.Errorf("expected 1 total run, got %d", em.TotalRuns)
	}
	if em.CompletedRuns != 1 {
		t.Errorf("expected 1 completed run, got %d", em.CompletedRuns)
	}
	// Two judge events: 6 + 8 = 14 total findings
	if em.TotalFindings != 14 {
		t.Errorf("expected 14 total findings (two judge rounds), got %d", em.TotalFindings)
	}
}

func TestBuildHarnessResponse_NoExplorationEvents(t *testing.T) {
	now := time.Now().Unix() - 3600
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now},
	}

	resp := buildHarnessResponse(events, 30)

	if resp.ExplorationMetrics != nil {
		t.Error("expected ExplorationMetrics to be nil when no exploration events")
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
