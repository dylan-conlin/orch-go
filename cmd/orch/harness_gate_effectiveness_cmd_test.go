package main

import (
	"testing"
	"time"
)

func TestBuildGateEffectiveness_Empty(t *testing.T) {
	result := buildGateEffectiveness(nil, 30)
	if result.TotalSpawns != 0 {
		t.Errorf("expected 0 spawns, got %d", result.TotalSpawns)
	}
	if result.Verdict == "" {
		t.Error("expected a verdict even with no data")
	}
}

func TestBuildGateEffectiveness_CohortClassification(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Agent A: spawned, triage allowed, completed with verification
		{Type: "session.spawned", SessionID: "s1", Timestamp: now - 100, Data: map[string]interface{}{
			"beads_id": "agent-a", "skill": "feature-impl", "workspace": "ws-a",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 99, Data: map[string]interface{}{
			"beads_id": "agent-a", "gate_name": "triage", "decision": "allow",
		}},
		{Type: "agent.completed", Timestamp: now - 50, Data: map[string]interface{}{
			"beads_id": "agent-a", "verification_passed": true,
		}},

		// Agent B: spawned, triage bypassed, completed without verification
		{Type: "session.spawned", SessionID: "s2", Timestamp: now - 100, Data: map[string]interface{}{
			"beads_id": "agent-b", "skill": "feature-impl", "workspace": "ws-b",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 99, Data: map[string]interface{}{
			"beads_id": "agent-b", "gate_name": "triage", "decision": "bypass",
		}},
		{Type: "agent.completed", Timestamp: now - 50, Data: map[string]interface{}{
			"beads_id": "agent-b", "verification_passed": false,
		}},

		// Agent C: spawned, hotspot blocked, abandoned
		{Type: "session.spawned", SessionID: "s3", Timestamp: now - 100, Data: map[string]interface{}{
			"beads_id": "agent-c", "skill": "feature-impl", "workspace": "ws-c",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 99, Data: map[string]interface{}{
			"beads_id": "agent-c", "gate_name": "hotspot", "decision": "block",
		}},
		{Type: "agent.abandoned", Timestamp: now - 50, Data: map[string]interface{}{
			"beads_id": "agent-c",
		}},

		// Agent D: spawned, triage allowed, hotspot allowed, completed verified
		{Type: "session.spawned", SessionID: "s4", Timestamp: now - 100, Data: map[string]interface{}{
			"beads_id": "agent-d", "skill": "investigation", "workspace": "ws-d",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 99, Data: map[string]interface{}{
			"beads_id": "agent-d", "gate_name": "triage", "decision": "allow",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 98, Data: map[string]interface{}{
			"beads_id": "agent-d", "gate_name": "hotspot", "decision": "allow",
		}},
		{Type: "agent.completed", Timestamp: now - 50, Data: map[string]interface{}{
			"beads_id": "agent-d", "verification_passed": true,
		}},
		{Type: "accretion.delta", Timestamp: now - 49, Data: map[string]interface{}{
			"beads_id": "agent-d", "net_delta": float64(150), "risk_files": float64(2),
		}},
	}

	result := buildGateEffectiveness(events, 1)

	// Check totals
	if result.TotalSpawns != 4 {
		t.Errorf("expected 4 spawns, got %d", result.TotalSpawns)
	}

	// Find triage gate
	var triageGate *GateEffPerGate
	for i := range result.PerGate {
		if result.PerGate[i].Gate == "triage" {
			triageGate = &result.PerGate[i]
			break
		}
	}
	if triageGate == nil {
		t.Fatal("expected triage gate in results")
	}

	// Triage: 2 allowed (agent-a, agent-d), 1 bypassed (agent-b), 0 blocked
	if triageGate.Allowed.Count != 2 {
		t.Errorf("triage allowed: expected 2, got %d", triageGate.Allowed.Count)
	}
	if triageGate.Bypassed.Count != 1 {
		t.Errorf("triage bypassed: expected 1, got %d", triageGate.Bypassed.Count)
	}
	if triageGate.Blocked.Count != 0 {
		t.Errorf("triage blocked: expected 0, got %d", triageGate.Blocked.Count)
	}

	// Find hotspot gate
	var hotspotGate *GateEffPerGate
	for i := range result.PerGate {
		if result.PerGate[i].Gate == "hotspot" {
			hotspotGate = &result.PerGate[i]
			break
		}
	}
	if hotspotGate == nil {
		t.Fatal("expected hotspot gate in results")
	}

	// Hotspot: 1 allowed (agent-d), 1 blocked (agent-c)
	if hotspotGate.Allowed.Count != 1 {
		t.Errorf("hotspot allowed: expected 1, got %d", hotspotGate.Allowed.Count)
	}
	if hotspotGate.Blocked.Count != 1 {
		t.Errorf("hotspot blocked: expected 1, got %d", hotspotGate.Blocked.Count)
	}

	// Overall: enforced = agent-a, agent-c, agent-d (no bypasses)
	// bypassed = agent-b (bypassed triage)
	if result.Overall.Enforced.Count != 3 {
		t.Errorf("enforced: expected 3, got %d", result.Overall.Enforced.Count)
	}
	if result.Overall.Bypassed.Count != 1 {
		t.Errorf("bypassed: expected 1, got %d", result.Overall.Bypassed.Count)
	}

	// Verification rates
	// Enforced: 2 completed (a, d) both verified, 1 abandoned (c) = 100% verif rate
	if result.Overall.Enforced.VerificationRate != 100.0 {
		t.Errorf("enforced verification rate: expected 100%%, got %.1f%%", result.Overall.Enforced.VerificationRate)
	}
	// Bypassed: 1 completed (b) not verified = 0% verif rate
	if result.Overall.Bypassed.VerificationRate != 0.0 {
		t.Errorf("bypassed verification rate: expected 0%%, got %.1f%%", result.Overall.Bypassed.VerificationRate)
	}

	// Verdict should exist
	if result.Verdict == "" {
		t.Error("expected non-empty verdict")
	}
}

func TestBuildGateEffectiveness_AccretionCorrelation(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		{Type: "session.spawned", SessionID: "s1", Timestamp: now - 100, Data: map[string]interface{}{
			"beads_id": "a1", "skill": "feature-impl", "workspace": "ws-1",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 99, Data: map[string]interface{}{
			"beads_id": "a1", "gate_name": "triage", "decision": "allow",
		}},
		{Type: "agent.completed", Timestamp: now - 50, Data: map[string]interface{}{
			"beads_id": "a1", "verification_passed": true,
		}},
		{Type: "accretion.delta", Timestamp: now - 49, Data: map[string]interface{}{
			"beads_id": "a1", "net_delta": float64(200), "risk_files": float64(3),
		}},

		{Type: "session.spawned", SessionID: "s2", Timestamp: now - 100, Data: map[string]interface{}{
			"beads_id": "a2", "skill": "feature-impl", "workspace": "ws-2",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 99, Data: map[string]interface{}{
			"beads_id": "a2", "gate_name": "triage", "decision": "bypass",
		}},
		{Type: "agent.completed", Timestamp: now - 50, Data: map[string]interface{}{
			"beads_id": "a2", "verification_passed": true,
		}},
		{Type: "accretion.delta", Timestamp: now - 49, Data: map[string]interface{}{
			"beads_id": "a2", "net_delta": float64(500), "risk_files": float64(5),
		}},
	}

	result := buildGateEffectiveness(events, 1)

	// Enforced agent should have lower accretion
	if result.Overall.Enforced.AvgNetDelta != 200 {
		t.Errorf("enforced avg delta: expected 200, got %.0f", result.Overall.Enforced.AvgNetDelta)
	}
	if result.Overall.Bypassed.AvgNetDelta != 500 {
		t.Errorf("bypassed avg delta: expected 500, got %.0f", result.Overall.Bypassed.AvgNetDelta)
	}
	if result.Overall.Enforced.RiskFilesTouched != 3 {
		t.Errorf("enforced risk files: expected 3, got %d", result.Overall.Enforced.RiskFilesTouched)
	}
}

func TestDecisionSeverity(t *testing.T) {
	if decisionSeverity("block") <= decisionSeverity("bypass") {
		t.Error("block should be more severe than bypass")
	}
	if decisionSeverity("bypass") <= decisionSeverity("allow") {
		t.Error("bypass should be more severe than allow")
	}
	if decisionSeverity("allow") <= decisionSeverity("") {
		t.Error("allow should be more severe than empty")
	}
}

func TestBuildCohort_Empty(t *testing.T) {
	c := buildCohort(nil)
	if c.Count != 0 {
		t.Errorf("expected 0 count, got %d", c.Count)
	}
	if c.CompletionRate != 0 {
		t.Errorf("expected 0 completion rate, got %f", c.CompletionRate)
	}
}

func TestGenerateVerdict_InsufficientData(t *testing.T) {
	overall := GateEffOverall{
		Enforced: GateEffCohort{Count: 5, CompletionRate: 80},
		Bypassed: GateEffCohort{Count: 3, CompletionRate: 90},
	}
	verdict, caveats := generateVerdict(overall, nil, 100)
	if verdict != "INSUFFICIENT DATA — need more gate interactions for reliable comparison" {
		t.Errorf("expected insufficient data verdict, got: %s", verdict)
	}
	if len(caveats) == 0 {
		t.Error("expected caveats about small sample")
	}
}

func TestFormatGateEffText_NoError(t *testing.T) {
	result := buildGateEffectiveness(nil, 30)
	text := formatGateEffText(result)
	if text == "" {
		t.Error("expected non-empty text output")
	}
	if !containsStr(text, "GATE EFFECTIVENESS") {
		t.Error("expected header in output")
	}
	if !containsStr(text, "VERDICT") {
		t.Error("expected verdict section in output")
	}
}

// containsStr is defined in review_triage_test.go
