package spawn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpawnTelemetry_Serialization(t *testing.T) {
	telemetry := SpawnTelemetry{
		BeadsID:               "orch-go-xyz",
		WorkspaceName:         "og-inv-test-30dec",
		Skill:                 "investigation",
		Tier:                  "full",
		ContextSizeChars:      45000,
		ContextSizeTokensEst:  11250,
		BehavioralPatternsCount: 3,
		EcosystemInjected:     true,
		ServerContextInjected: false,
		KBContextStats: &KBContextStats{
			Query:              "observability",
			MatchCount:         5,
			WasTruncated:       false,
			ConstraintsCount:   2,
			DecisionsCount:     3,
			InvestigationsCount: 0,
		},
	}

	data, err := json.Marshal(telemetry)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Verify expected JSON field names
	expected := []string{
		"beads_id",
		"workspace_name",
		"skill",
		"tier",
		"context_size_chars",
		"context_size_tokens_est",
		"behavioral_patterns_count",
		"ecosystem_context_injected",
		"server_context_injected",
		"kb_context_stats",
	}
	for _, field := range expected {
		if !strings.Contains(string(data), field) {
			t.Errorf("Expected JSON field %q not found", field)
		}
	}

	// Round-trip verification
	var parsed SpawnTelemetry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if parsed.BeadsID != telemetry.BeadsID {
		t.Errorf("BeadsID = %v, want %v", parsed.BeadsID, telemetry.BeadsID)
	}
	if parsed.ContextSizeChars != telemetry.ContextSizeChars {
		t.Errorf("ContextSizeChars = %v, want %v", parsed.ContextSizeChars, telemetry.ContextSizeChars)
	}
	if parsed.KBContextStats == nil {
		t.Error("KBContextStats should not be nil after unmarshal")
	} else if parsed.KBContextStats.MatchCount != 5 {
		t.Errorf("KBContextStats.MatchCount = %v, want 5", parsed.KBContextStats.MatchCount)
	}
}

func TestSpawnTelemetry_OmitsEmptyKBContext(t *testing.T) {
	telemetry := SpawnTelemetry{
		BeadsID:          "orch-go-xyz",
		WorkspaceName:    "og-inv-test-30dec",
		Skill:            "investigation",
		Tier:             "light",
		ContextSizeChars: 10000,
		// KBContextStats is nil
	}

	data, err := json.Marshal(telemetry)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// kb_context_stats should be omitted when nil
	if strings.Contains(string(data), "kb_context_stats") {
		t.Error("Expected kb_context_stats to be omitted when nil")
	}
}

func TestKBContextStats_Serialization(t *testing.T) {
	stats := KBContextStats{
		Query:               "telemetry spawn",
		MatchCount:          7,
		WasTruncated:        true,
		ConstraintsCount:    1,
		DecisionsCount:      2,
		InvestigationsCount: 4,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	expected := []string{
		"query",
		"match_count",
		"was_truncated",
		"constraints_count",
		"decisions_count",
		"investigations_count",
	}
	for _, field := range expected {
		if !strings.Contains(string(data), field) {
			t.Errorf("Expected JSON field %q not found", field)
		}
	}
}

func TestCollectSpawnTelemetry(t *testing.T) {
	cfg := &Config{
		BeadsID:       "orch-go-abc",
		WorkspaceName: "og-feat-test-30dec",
		SkillName:     "feature-impl",
		Tier:          "light",
		ProjectDir:    "/test/project",
	}

	// Simulate generated context
	generatedContext := "TASK: Test task\nSome context here with sufficient content."

	telemetry := CollectSpawnTelemetry(cfg, generatedContext, nil)

	if telemetry.BeadsID != cfg.BeadsID {
		t.Errorf("BeadsID = %v, want %v", telemetry.BeadsID, cfg.BeadsID)
	}
	if telemetry.WorkspaceName != cfg.WorkspaceName {
		t.Errorf("WorkspaceName = %v, want %v", telemetry.WorkspaceName, cfg.WorkspaceName)
	}
	if telemetry.Skill != cfg.SkillName {
		t.Errorf("Skill = %v, want %v", telemetry.Skill, cfg.SkillName)
	}
	if telemetry.Tier != cfg.Tier {
		t.Errorf("Tier = %v, want %v", telemetry.Tier, cfg.Tier)
	}
	if telemetry.ContextSizeChars != len(generatedContext) {
		t.Errorf("ContextSizeChars = %v, want %v", telemetry.ContextSizeChars, len(generatedContext))
	}
	// Token estimate should be chars / 4 (using CharsPerToken constant)
	expectedTokens := len(generatedContext) / CharsPerToken
	if telemetry.ContextSizeTokensEst != expectedTokens {
		t.Errorf("ContextSizeTokensEst = %v, want %v", telemetry.ContextSizeTokensEst, expectedTokens)
	}
}

func TestCollectSpawnTelemetry_WithKBContext(t *testing.T) {
	cfg := &Config{
		BeadsID:       "orch-go-abc",
		WorkspaceName: "og-inv-test-30dec",
		SkillName:     "investigation",
		Tier:          "full",
		ProjectDir:    "/test/project",
	}

	kbResult := &KBContextFormatResult{
		Content:          "## PRIOR KNOWLEDGE...",
		WasTruncated:     true,
		OriginalMatches:  10,
		TruncatedMatches: 7,
		EstimatedTokens:  500,
	}

	generatedContext := "TASK: Test task"

	telemetry := CollectSpawnTelemetry(cfg, generatedContext, kbResult)

	if telemetry.KBContextStats == nil {
		t.Fatal("KBContextStats should not be nil when kbResult provided")
	}
	if telemetry.KBContextStats.WasTruncated != kbResult.WasTruncated {
		t.Errorf("WasTruncated = %v, want %v", telemetry.KBContextStats.WasTruncated, kbResult.WasTruncated)
	}
	if telemetry.KBContextStats.MatchCount != kbResult.TruncatedMatches {
		t.Errorf("MatchCount = %v, want %v (truncated count)", telemetry.KBContextStats.MatchCount, kbResult.TruncatedMatches)
	}
}

func TestCollectSpawnTelemetry_WithGapAnalysis(t *testing.T) {
	cfg := &Config{
		BeadsID:       "orch-go-abc",
		WorkspaceName: "og-inv-test-30dec",
		SkillName:     "investigation",
		Tier:          "full",
		ProjectDir:    "/test/project",
		GapAnalysis: &GapAnalysis{
			Query: "telemetry spawn",
			MatchStats: MatchStatistics{
				TotalMatches:       8,
				ConstraintCount:    2,
				DecisionCount:      4,
				InvestigationCount: 2,
			},
		},
	}

	generatedContext := "TASK: Test task"

	// Don't pass kbResult - should fall back to GapAnalysis
	telemetry := CollectSpawnTelemetry(cfg, generatedContext, nil)

	if telemetry.KBContextStats == nil {
		t.Fatal("KBContextStats should not be nil when GapAnalysis provided")
	}
	if telemetry.KBContextStats.Query != "telemetry spawn" {
		t.Errorf("Query = %v, want 'telemetry spawn'", telemetry.KBContextStats.Query)
	}
	if telemetry.KBContextStats.MatchCount != 8 {
		t.Errorf("MatchCount = %v, want 8", telemetry.KBContextStats.MatchCount)
	}
	if telemetry.KBContextStats.ConstraintsCount != 2 {
		t.Errorf("ConstraintsCount = %v, want 2", telemetry.KBContextStats.ConstraintsCount)
	}
	if telemetry.KBContextStats.DecisionsCount != 4 {
		t.Errorf("DecisionsCount = %v, want 4", telemetry.KBContextStats.DecisionsCount)
	}
	if telemetry.KBContextStats.InvestigationsCount != 2 {
		t.Errorf("InvestigationsCount = %v, want 2", telemetry.KBContextStats.InvestigationsCount)
	}
}

func TestLogSpawnTelemetry(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")

	telemetry := SpawnTelemetry{
		BeadsID:              "orch-go-test",
		WorkspaceName:        "og-feat-test-30dec",
		Skill:                "feature-impl",
		Tier:                 "light",
		ContextSizeChars:     20000,
		ContextSizeTokensEst: 5000,
	}

	err := LogSpawnTelemetry(logPath, telemetry)
	if err != nil {
		t.Fatalf("LogSpawnTelemetry() error = %v", err)
	}

	// Read and verify the logged event
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Should contain the event type
	if !strings.Contains(string(data), "spawn.telemetry") {
		t.Error("Expected event type 'spawn.telemetry'")
	}

	// Should contain telemetry data
	if !strings.Contains(string(data), "orch-go-test") {
		t.Error("Expected beads_id in logged event")
	}
	if !strings.Contains(string(data), "og-feat-test-30dec") {
		t.Error("Expected workspace_name in logged event")
	}

	// Parse and verify structure
	var event struct {
		Type      string          `json:"type"`
		Timestamp int64           `json:"timestamp"`
		Data      SpawnTelemetry  `json:"data"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to parse logged event: %v", err)
	}

	if event.Type != "spawn.telemetry" {
		t.Errorf("Event type = %v, want spawn.telemetry", event.Type)
	}
	if event.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}
	if event.Data.BeadsID != telemetry.BeadsID {
		t.Errorf("Data.BeadsID = %v, want %v", event.Data.BeadsID, telemetry.BeadsID)
	}
}
