package main

import (
	"encoding/json"
	"fmt"
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

func TestBuildGateAudit_RemovedGatesFiltered(t *testing.T) {
	now := time.Now().Unix()
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now - 100},
		{Type: "spawn.gate_decision", Timestamp: now - 50, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "allow",
		}},
		// These gates were removed from the spawn pipeline — should be filtered
		{Type: "spawn.gate_decision", Timestamp: now - 60, Data: map[string]interface{}{
			"gate_name": "verification", "decision": "allow",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 70, Data: map[string]interface{}{
			"gate_name": "drain", "decision": "allow",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 80, Data: map[string]interface{}{
			"gate_name": "concurrency", "decision": "allow",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 90, Data: map[string]interface{}{
			"gate_name": "ratelimit", "decision": "allow",
		}},
	}

	result := buildGateAudit(events, 7)
	if len(result.Gates) != 1 {
		t.Errorf("expected 1 gate (triage only), got %d", len(result.Gates))
	}
	if len(result.Gates) > 0 && result.Gates[0].Gate != "triage" {
		t.Errorf("expected triage gate, got %s", result.Gates[0].Gate)
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

func TestBuildGateAudit_DeadSignal_ZeroEventsOverManyCompletions(t *testing.T) {
	now := time.Now().Unix()
	var events []StatsEvent

	// 15 completions — enough to trigger dead signal detection (threshold=10)
	for i := 0; i < 15; i++ {
		events = append(events, StatsEvent{
			Type: "session.spawned", Timestamp: now - int64(i*100),
			Data: map[string]interface{}{"skill": "feature-impl"},
		})
		events = append(events, StatsEvent{
			Type: "agent.completed", Timestamp: now - int64(i*100),
			Data: map[string]interface{}{"beads_id": fmt.Sprintf("test-%d", i)},
		})
	}
	// No verification.failed, verification.bypassed,
	// duplication.detected, or spawn.gate_decision events at all.

	result := buildGateAudit(events, 30)

	if len(result.DeadSignals) == 0 {
		t.Fatal("expected dead signal entries for channels with zero events over 15 completions")
	}

	channelsSeen := make(map[string]bool)
	for _, ds := range result.DeadSignals {
		channelsSeen[ds.Channel] = true
		if ds.Completions != 15 {
			t.Errorf("channel %s: expected 15 completions, got %d", ds.Channel, ds.Completions)
		}
		if ds.Events != 0 {
			t.Errorf("channel %s: expected 0 events, got %d", ds.Channel, ds.Events)
		}
	}

	for _, ch := range []string{"verification", "duplication"} {
		if !channelsSeen[ch] {
			t.Errorf("expected dead signal for channel %q", ch)
		}
	}
}

func TestBuildGateAudit_DeadSignal_NotFlaggedWithFewCompletions(t *testing.T) {
	now := time.Now().Unix()
	var events []StatsEvent

	// Only 3 completions — below threshold
	for i := 0; i < 3; i++ {
		events = append(events, StatsEvent{
			Type: "agent.completed", Timestamp: now - int64(i*100),
			Data: map[string]interface{}{"beads_id": fmt.Sprintf("test-%d", i)},
		})
	}

	result := buildGateAudit(events, 30)
	if len(result.DeadSignals) != 0 {
		t.Errorf("expected no dead signals with only 3 completions, got %d", len(result.DeadSignals))
	}
}

func TestBuildGateAudit_DeadSignal_NotFlaggedWhenEventsExist(t *testing.T) {
	now := time.Now().Unix()
	var events []StatsEvent

	// 15 completions
	for i := 0; i < 15; i++ {
		events = append(events, StatsEvent{
			Type: "agent.completed", Timestamp: now - int64(i*100),
			Data: map[string]interface{}{"beads_id": fmt.Sprintf("test-%d", i)},
		})
	}
	// Add verification events — this channel should NOT be dead
	events = append(events, StatsEvent{
		Type: "verification.failed", Timestamp: now - 50,
		Data: map[string]interface{}{"gate": "test_evidence"},
	})
	events = append(events, StatsEvent{
		Type: "verification.bypassed", Timestamp: now - 60,
		Data: map[string]interface{}{"gate": "git_diff"},
	})

	result := buildGateAudit(events, 30)

	for _, ds := range result.DeadSignals {
		if ds.Channel == "verification" {
			t.Error("verification channel should NOT be flagged as dead when it has events")
		}
	}
}

func TestBuildGateAudit_DeadSignal_InAnomalies(t *testing.T) {
	now := time.Now().Unix()
	var events []StatsEvent

	for i := 0; i < 15; i++ {
		events = append(events, StatsEvent{
			Type: "agent.completed", Timestamp: now - int64(i*100),
			Data: map[string]interface{}{"beads_id": fmt.Sprintf("test-%d", i)},
		})
	}

	result := buildGateAudit(events, 30)

	deadSignalAnomalies := 0
	for _, a := range result.Anomalies {
		if a.Type == "dead_signal" {
			deadSignalAnomalies++
		}
	}
	if deadSignalAnomalies == 0 {
		t.Error("expected dead_signal anomalies to appear in anomalies list")
	}
}

func TestBuildGateAudit_DeadSignal_TextOutput(t *testing.T) {
	audit := &GateAuditResult{
		DaysAnalyzed: 30,
		TotalSpawns:  20,
		DeadSignals: []DeadSignalEntry{
			{Channel: "verification", Events: 0, Completions: 20, Description: "verification.failed + verification.bypassed"},
		},
		Anomalies: []GateAnomaly{
			{Gate: "verification", Type: "dead_signal", Message: "0 events over 20 completions — channel may be dead, not clean"},
		},
	}
	output := formatGateAuditText(audit)
	if !contains(output, "DEAD SIGNALS") {
		t.Error("expected text output to contain DEAD SIGNALS section")
	}
	if !contains(output, "verification") {
		t.Error("expected text output to contain verification channel")
	}
}
