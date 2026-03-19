package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestHarnessIntegration_AllFourVerdicts verifies the full pipeline produces
// all 4 falsification verdicts with data sources cited.
func TestHarnessIntegration_AllFourVerdicts(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")
	now := time.Now().Unix()

	events := []map[string]interface{}{
		// Spawns
		{"type": "session.spawned", "session_id": "s1", "timestamp": now - 3600, "data": map[string]interface{}{"skill": "feature-impl"}},
		{"type": "session.spawned", "session_id": "s2", "timestamp": now - 7200, "data": map[string]interface{}{"skill": "investigation"}},
		// Gate decisions (new unified events)
		{"type": "spawn.gate_decision", "timestamp": now - 3500, "data": map[string]interface{}{
			"gate_name": "hotspot", "decision": "bypass", "skill": "feature-impl",
		}},
		{"type": "spawn.gate_decision", "timestamp": now - 3500, "data": map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "investigation",
		}},
		// Legacy bypass events
		{"type": "spawn.triage_bypassed", "timestamp": now - 3400, "data": map[string]interface{}{"reason": "manual"}},
		// Completions with full fields
		{"type": "agent.completed", "session_id": "s1", "timestamp": now - 1800, "data": map[string]interface{}{
			"beads_id": "orch-go-1", "skill": "feature-impl", "outcome": "success",
			"duration_seconds": float64(1800), "verification_passed": true, "forced": false,
		}},
	}

	f, _ := os.Create(eventsPath)
	enc := json.NewEncoder(f)
	for _, e := range events {
		enc.Encode(e)
	}
	f.Close()

	parsed, err := parseEvents(eventsPath)
	if err != nil {
		t.Fatal(err)
	}
	resp := buildHarnessResponse(parsed, 7)

	// Verify all 4 falsification verdicts are present
	verdictKeys := []string{"gates_are_ceremony", "gates_are_irrelevant", "soft_harness_is_inert", "framework_is_anecdotal"}
	for _, key := range verdictKeys {
		verdict, ok := resp.FalsificationVerdicts[key]
		if !ok {
			t.Errorf("missing falsification verdict: %s", key)
			continue
		}
		if verdict.Status == "" {
			t.Errorf("verdict %s has empty status", key)
		}
		if verdict.Criterion == "" {
			t.Errorf("verdict %s has empty criterion", key)
		}
		if verdict.Evidence == "" {
			t.Errorf("verdict %s has empty evidence (no data source cited)", key)
		}
		if verdict.Threshold == "" {
			t.Errorf("verdict %s has empty threshold", key)
		}
	}
}

// TestHarnessIntegration_GateDeflectionFromUnifiedEvents verifies gate deflection
// rates are computable from unified events (legacy + new gate_decision).
func TestHarnessIntegration_GateDeflectionFromUnifiedEvents(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")
	now := time.Now().Unix()

	events := []map[string]interface{}{
		// 10 spawns
		{"type": "session.spawned", "session_id": "s1", "timestamp": now - 1000},
		{"type": "session.spawned", "session_id": "s2", "timestamp": now - 900},
		{"type": "session.spawned", "session_id": "s3", "timestamp": now - 800},
		{"type": "session.spawned", "session_id": "s4", "timestamp": now - 700},
		{"type": "session.spawned", "session_id": "s5", "timestamp": now - 600},
		// Legacy hotspot bypasses (3)
		{"type": "spawn.hotspot_bypassed", "timestamp": now - 990},
		{"type": "spawn.hotspot_bypassed", "timestamp": now - 890},
		{"type": "spawn.hotspot_bypassed", "timestamp": now - 790},
		// New gate_decision events (2 blocks, 1 allow)
		{"type": "spawn.gate_decision", "timestamp": now - 690, "data": map[string]interface{}{
			"gate_name": "hotspot", "decision": "block", "skill": "feature-impl",
		}},
		{"type": "spawn.gate_decision", "timestamp": now - 590, "data": map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "investigation",
		}},
	}

	f, _ := os.Create(eventsPath)
	enc := json.NewEncoder(f)
	for _, e := range events {
		enc.Encode(e)
	}
	f.Close()

	parsed, err := parseEvents(eventsPath)
	if err != nil {
		t.Fatal(err)
	}
	resp := buildHarnessResponse(parsed, 7)

	// Find hotspot gate in pipeline
	var hotspot *PipelineComponent
	for _, stage := range resp.Pipeline {
		for i, comp := range stage.Components {
			if comp.Name == "hotspot_gate" {
				hotspot = &stage.Components[i]
				break
			}
		}
	}
	if hotspot == nil {
		t.Fatal("hotspot_gate not found in pipeline")
	}

	// Should combine: 3 legacy bypasses + 1 new block = 4 fire events from 5 spawns
	// (allow events don't count as "fire" for bypass/block counting)
	if hotspot.Bypassed < 3 {
		t.Errorf("expected at least 3 hotspot bypasses (legacy+new), got %d", hotspot.Bypassed)
	}
	if hotspot.Blocked < 1 {
		t.Errorf("expected at least 1 hotspot block, got %d", hotspot.Blocked)
	}
	if hotspot.FireRate == nil {
		t.Error("hotspot fire rate should not be nil")
	}
	if hotspot.MeasurementStatus != "flowing" {
		t.Errorf("expected measurement_status='flowing', got %q", hotspot.MeasurementStatus)
	}
}

// TestHarnessIntegration_AccretionVelocityPrePostGate verifies accretion velocity
// shows pre/post gate comparison from snapshot events.
func TestHarnessIntegration_AccretionVelocityPrePostGate(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")
	now := time.Now().Unix()

	events := []map[string]interface{}{
		// Pre-gate snapshot (3 weeks ago)
		{"type": "accretion.snapshot", "timestamp": now - 21*86400, "data": map[string]interface{}{
			"directory": "cmd/orch/", "total_lines": float64(10000), "file_count": float64(20),
			"files_over_800": float64(5), "files_over_1500": float64(1), "snapshot_type": "weekly",
		}},
		// Pre-gate snapshot (2 weeks ago) - baseline
		{"type": "accretion.snapshot", "timestamp": now - 14*86400, "data": map[string]interface{}{
			"directory": "cmd/orch/", "total_lines": float64(16131), "file_count": float64(22),
			"files_over_800": float64(7), "files_over_1500": float64(2), "snapshot_type": "weekly",
		}},
		// Post-gate snapshot (1 week ago)
		{"type": "accretion.snapshot", "timestamp": now - 7*86400, "data": map[string]interface{}{
			"directory": "cmd/orch/", "total_lines": float64(16500), "file_count": float64(23),
			"files_over_800": float64(7), "files_over_1500": float64(2), "snapshot_type": "weekly",
		}},
	}

	f, _ := os.Create(eventsPath)
	enc := json.NewEncoder(f)
	for _, e := range events {
		enc.Encode(e)
	}
	f.Close()

	parsed, err := parseEvents(eventsPath)
	if err != nil {
		t.Fatal(err)
	}
	resp := buildHarnessResponse(parsed, 30)

	if resp.AccretionVelocity == nil {
		t.Fatal("expected accretion velocity data, got nil")
	}

	// Baseline = 6131 lines/week (16131 - 10000 over 1 week)
	// Current = 369 lines/week (16500 - 16131 over 1 week)
	// Velocity should show declining trend
	if resp.AccretionVelocity.BaselineWeeklyLines == 0 {
		t.Error("baseline_weekly_lines should not be zero")
	}
	if resp.AccretionVelocity.Trend == "" {
		t.Error("trend should not be empty")
	}
}

// TestHarnessIntegration_CompletionCoverageFieldDetection verifies completion field
// coverage detects all expected fields (skill, outcome, duration_seconds).
func TestHarnessIntegration_CompletionCoverageFieldDetection(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")
	now := time.Now().Unix()

	events := []map[string]interface{}{
		// Full coverage event
		{"type": "agent.completed", "session_id": "b1", "timestamp": now - 100, "data": map[string]interface{}{
			"beads_id": "orch-go-1", "skill": "feature-impl", "outcome": "success",
			"duration_seconds": float64(3600), "verification_passed": true, "forced": false,
		}},
		// Missing skill
		{"type": "agent.completed", "session_id": "b2", "timestamp": now - 90, "data": map[string]interface{}{
			"beads_id": "orch-go-2", "outcome": "forced", "duration_seconds": float64(1200),
			"verification_passed": false, "forced": true,
		}},
		// Missing outcome and duration
		{"type": "agent.completed", "session_id": "b3", "timestamp": now - 80, "data": map[string]interface{}{
			"beads_id": "orch-go-3", "skill": "investigation",
		}},
	}

	f, _ := os.Create(eventsPath)
	enc := json.NewEncoder(f)
	for _, e := range events {
		enc.Encode(e)
	}
	f.Close()

	parsed, err := parseEvents(eventsPath)
	if err != nil {
		t.Fatal(err)
	}
	resp := buildHarnessResponse(parsed, 7)

	cc := resp.CompletionCoverage
	if cc.TotalCompletions != 3 {
		t.Errorf("expected 3 completions, got %d", cc.TotalCompletions)
	}
	if cc.WithSkill != 2 {
		t.Errorf("expected 2 with skill, got %d", cc.WithSkill)
	}
	if cc.WithOutcome != 2 {
		t.Errorf("expected 2 with outcome, got %d", cc.WithOutcome)
	}
	if cc.WithDuration != 2 {
		t.Errorf("expected 2 with duration, got %d", cc.WithDuration)
	}
}

// TestHarnessIntegration_PipelineFourStages verifies the pipeline has all 4 stages
// with correct component assignment.
func TestHarnessIntegration_PipelineFourStages(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")
	os.WriteFile(eventsPath, []byte(""), 0644)

	parsed, _ := parseEvents(eventsPath)
	resp := buildHarnessResponse(parsed, 7)

	if len(resp.Pipeline) != 4 {
		t.Fatalf("expected 4 pipeline stages, got %d", len(resp.Pipeline))
	}

	expectedStages := map[string][]string{
		"spawn":      {"triage_gate", "hotspot_gate"},
		"authoring":  {"claude_md", "spawn_context", "kb_knowledge"},
		"pre_commit": {"accretion_gate", "build_gate"},
		"completion": {"verification_pipeline", "explain_back"},
	}

	for _, stage := range resp.Pipeline {
		expected, ok := expectedStages[stage.Stage]
		if !ok {
			t.Errorf("unexpected pipeline stage: %s", stage.Stage)
			continue
		}
		componentNames := make(map[string]bool)
		for _, comp := range stage.Components {
			componentNames[comp.Name] = true
		}
		for _, name := range expected {
			if !componentNames[name] {
				t.Errorf("stage %s missing component %s", stage.Stage, name)
			}
		}
	}
}

// TestHarnessIntegration_APIEndpoint verifies the /api/harness endpoint returns
// valid JSON with all required fields.
func TestHarnessIntegration_APIEndpoint(t *testing.T) {
	// Create a test request
	req := httptest.NewRequest("GET", "/api/harness?days=7", nil)
	w := httptest.NewRecorder()

	handleHarnessReport(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var data HarnessResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify all 4 falsification verdicts present
	for _, key := range []string{"gates_are_ceremony", "gates_are_irrelevant", "soft_harness_is_inert", "framework_is_anecdotal"} {
		if _, ok := data.FalsificationVerdicts[key]; !ok {
			t.Errorf("API response missing verdict: %s", key)
		}
	}

	// Verify pipeline stages present
	if len(data.Pipeline) != 4 {
		t.Errorf("expected 4 pipeline stages, got %d", len(data.Pipeline))
	}
}

// TestHarnessIntegration_GateDecisionFieldName verifies the event parser reads
// "gate_name" (not "gate") from spawn.gate_decision events.
func TestHarnessIntegration_GateDecisionFieldName(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")
	now := time.Now().Unix()

	events := []map[string]interface{}{
		{"type": "session.spawned", "session_id": "s1", "timestamp": now - 100},
		{"type": "spawn.gate_decision", "timestamp": now - 99, "data": map[string]interface{}{
			"gate_name": "hotspot", "decision": "block", "skill": "feature-impl",
		}},
	}

	f, _ := os.Create(eventsPath)
	enc := json.NewEncoder(f)
	for _, e := range events {
		enc.Encode(e)
	}
	f.Close()

	parsed, _ := parseEvents(eventsPath)
	resp := buildHarnessResponse(parsed, 7)

	// Should have hotspot gate with 1 block
	for _, stage := range resp.Pipeline {
		for _, comp := range stage.Components {
			if comp.Name == "hotspot_gate" {
				if comp.Blocked != 1 {
					t.Errorf("expected 1 hotspot block, got %d (possible gate_name vs gate field bug)", comp.Blocked)
				}
				return
			}
		}
	}
	t.Error("hotspot_gate not found in pipeline")
}
