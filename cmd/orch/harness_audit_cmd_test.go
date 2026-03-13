package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHarnessAuditCmd_Flags(t *testing.T) {
	cmd := harnessAuditCmd
	if cmd.Use != "audit" {
		t.Errorf("expected Use='audit', got %q", cmd.Use)
	}
	for _, name := range []string{"days", "json"} {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("expected flag --%s to exist", name)
		}
	}
}

func TestHarnessAuditCmd_IsRegistered(t *testing.T) {
	found := false
	for _, cmd := range harnessCmd.Commands() {
		if cmd.Use == "audit" {
			found = true
			break
		}
	}
	if !found {
		t.Error("harness audit command not registered as subcommand of harness")
	}
}

func TestBuildGateAudit_Empty(t *testing.T) {
	result := buildGateAudit(nil, 30)
	if result.DaysAnalyzed != 30 {
		t.Errorf("expected days=30, got %d", result.DaysAnalyzed)
	}
	if len(result.Gates) != 0 {
		t.Errorf("expected 0 gates, got %d", len(result.Gates))
	}
	if len(result.Anomalies) != 0 {
		t.Errorf("expected 0 anomalies, got %d", len(result.Anomalies))
	}
}

func TestBuildGateAudit_BasicCounts(t *testing.T) {
	now := time.Now().Unix()
	events := []StatsEvent{
		{Type: "spawn.gate_decision", Timestamp: now - 100, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "allow", "skill": "feature-impl",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 200, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "block", "skill": "feature-impl",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 300, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "bypass", "skill": "investigation",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 400, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "feature-impl",
		}},
	}

	result := buildGateAudit(events, 7)
	if len(result.Gates) != 2 {
		t.Fatalf("expected 2 gates, got %d", len(result.Gates))
	}

	var triage *GateAuditEntry
	for i := range result.Gates {
		if result.Gates[i].Gate == "triage" {
			triage = &result.Gates[i]
		}
	}
	if triage == nil {
		t.Fatal("triage gate not found")
	}
	if triage.Invocations != 3 {
		t.Errorf("triage invocations: expected 3, got %d", triage.Invocations)
	}
	if triage.Blocks != 1 {
		t.Errorf("triage blocks: expected 1, got %d", triage.Blocks)
	}
	if triage.Allows != 1 {
		t.Errorf("triage allows: expected 1, got %d", triage.Allows)
	}
	if triage.Bypasses != 1 {
		t.Errorf("triage bypasses: expected 1, got %d", triage.Bypasses)
	}
	// Fire rate = (blocks+bypasses) / invocations = 2/3 = 66.7%
	expectedFireRate := 2.0 / 3.0 * 100
	if diff := triage.FireRatePct - expectedFireRate; diff > 0.1 || diff < -0.1 {
		t.Errorf("triage fire rate: expected ~%.1f%%, got %.1f%%", expectedFireRate, triage.FireRatePct)
	}
}

func TestBuildGateAudit_CostFromPipelineTiming(t *testing.T) {
	now := time.Now().Unix()
	events := []StatsEvent{
		{Type: "spawn.gate_decision", Timestamp: now - 100, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 200, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow",
		}},
		{Type: "agent.completed", Timestamp: now - 50, Data: map[string]interface{}{
			"pipeline_timing": []interface{}{
				map[string]interface{}{"name": "hotspot", "duration_ms": float64(300), "skipped": false},
				map[string]interface{}{"name": "duplication", "duration_ms": float64(5000), "skipped": false},
			},
		}},
		{Type: "agent.completed", Timestamp: now - 150, Data: map[string]interface{}{
			"pipeline_timing": []interface{}{
				map[string]interface{}{"name": "hotspot", "duration_ms": float64(500), "skipped": false},
			},
		}},
	}

	result := buildGateAudit(events, 7)

	var hotspot *GateAuditEntry
	for i := range result.Gates {
		if result.Gates[i].Gate == "hotspot" {
			hotspot = &result.Gates[i]
		}
	}
	if hotspot == nil {
		t.Fatal("hotspot gate not found")
	}
	if hotspot.MeanCostMs != 400 {
		t.Errorf("hotspot mean cost: expected 400, got %d", hotspot.MeanCostMs)
	}
}

func TestBuildGateAudit_AnomalyZeroFires(t *testing.T) {
	now := time.Now().Unix()
	events := []StatsEvent{
		{Type: "spawn.gate_decision", Timestamp: now - 100, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 200, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow",
		}},
	}

	result := buildGateAudit(events, 30)
	found := false
	for _, a := range result.Anomalies {
		if a.Gate == "hotspot" && a.Type == "zero_fires" {
			found = true
		}
	}
	if !found {
		t.Error("expected zero_fires anomaly for hotspot gate")
	}
}

func TestBuildGateAudit_AnomalyHighCost(t *testing.T) {
	now := time.Now().Unix()
	events := []StatsEvent{
		{Type: "spawn.gate_decision", Timestamp: now - 100, Data: map[string]interface{}{
			"gate_name": "duplication", "decision": "allow",
		}},
		{Type: "agent.completed", Timestamp: now - 50, Data: map[string]interface{}{
			"pipeline_timing": []interface{}{
				map[string]interface{}{"name": "duplication", "duration_ms": float64(1500), "skipped": false},
			},
		}},
	}

	result := buildGateAudit(events, 30)
	found := false
	for _, a := range result.Anomalies {
		if a.Gate == "duplication" && a.Type == "high_cost" {
			found = true
		}
	}
	if !found {
		t.Error("expected high_cost anomaly for duplication gate (>1000ms)")
	}
}

func TestBuildGateAudit_AnomalyLowCoverage(t *testing.T) {
	now := time.Now().Unix()
	events := make([]StatsEvent, 0)
	for i := 0; i < 10; i++ {
		events = append(events, StatsEvent{
			Type: "session.spawned", Timestamp: now - int64(i*100),
			Data: map[string]interface{}{"skill": "feature-impl"},
		})
	}
	for i := 0; i < 4; i++ {
		events = append(events, StatsEvent{
			Type: "spawn.gate_decision", Timestamp: now - int64(i*100),
			Data: map[string]interface{}{"gate_name": "triage", "decision": "allow"},
		})
	}

	result := buildGateAudit(events, 7)
	found := false
	for _, a := range result.Anomalies {
		if a.Gate == "triage" && a.Type == "low_coverage" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected low_coverage anomaly for triage gate; anomalies: %+v", result.Anomalies)
	}
}

func TestBuildGateAudit_OutsideWindowIgnored(t *testing.T) {
	now := time.Now().Unix()
	old := now - 86400*40 // 40 days ago
	events := []StatsEvent{
		{Type: "spawn.gate_decision", Timestamp: old, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "block",
		}},
	}

	result := buildGateAudit(events, 7)
	if len(result.Gates) != 0 {
		t.Error("expected events outside window to be ignored")
	}
}

func TestFormatGateAuditText(t *testing.T) {
	audit := &GateAuditResult{
		DaysAnalyzed: 30,
		TotalSpawns:  50,
		Gates: []GateAuditEntry{
			{Gate: "triage", Invocations: 40, Blocks: 2, Bypasses: 5, Allows: 33, FireRatePct: 17.5, CoveragePct: 80.0, MeanCostMs: 0},
			{Gate: "hotspot", Invocations: 40, Blocks: 0, Bypasses: 0, Allows: 40, FireRatePct: 0.0, CoveragePct: 80.0, MeanCostMs: 315},
		},
		Anomalies: []GateAnomaly{
			{Gate: "hotspot", Type: "zero_fires", Message: "0 fires in 30d"},
		},
	}
	output := formatGateAuditText(audit)
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	for _, expected := range []string{"GATE AUDIT", "triage", "hotspot", "ANOMALIES", "zero_fires"} {
		if !contains(output, expected) {
			t.Errorf("expected output to contain %q", expected)
		}
	}
}

func TestFormatGateAuditJSON(t *testing.T) {
	audit := &GateAuditResult{
		DaysAnalyzed: 7,
		Gates: []GateAuditEntry{
			{Gate: "triage", Invocations: 10},
		},
	}
	output, err := formatGateAuditJSON(audit)
	if err != nil {
		t.Fatal(err)
	}
	var parsed GateAuditResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.DaysAnalyzed != 7 {
		t.Errorf("expected days=7 in JSON, got %d", parsed.DaysAnalyzed)
	}
}
