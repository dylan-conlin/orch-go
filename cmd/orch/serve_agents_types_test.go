package main

import (
	"encoding/json"
	"testing"
)

func TestAgentAPIResponseJSONFormat(t *testing.T) {
	synthesis := &SynthesisResponse{
		TLDR:           "Test synthesis summary",
		Outcome:        "success",
		Recommendation: "close",
		DeltaSummary:   "2 files created, 1 modified",
		NextActions:    []string{"- Review changes", "- Update docs"},
	}

	agent := &AgentAPIResponse{
		Synthesis: synthesis,
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Failed to marshal AgentAPIResponse: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	synthData, ok := result["synthesis"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected synthesis field in JSON")
	}

	if synthData["tldr"] != "Test synthesis summary" {
		t.Errorf("Expected tldr 'Test synthesis summary', got %v", synthData["tldr"])
	}
	if synthData["outcome"] != "success" {
		t.Errorf("Expected outcome 'success', got %v", synthData["outcome"])
	}
	if synthData["recommendation"] != "close" {
		t.Errorf("Expected recommendation 'close', got %v", synthData["recommendation"])
	}
}

func TestAgentAPIResponseStallReason(t *testing.T) {
	tests := []struct {
		name         string
		stallReason  string
		isStalled    bool
		expectReason bool // expect stall_reason in JSON output
	}{
		{"empty omitted when not stalled", "", false, false},
		{"phase_stall", "phase_stall", true, true},
		{"token_stall", "token_stall", true, true},
		{"never_started", "never_started", true, true},
		{"spawn_stale", "spawn_stale", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := AgentAPIResponse{
				ID:          "test-agent",
				Status:      "active",
				IsStalled:   tt.isStalled,
				StallReason: tt.stallReason,
			}

			data, err := json.Marshal(agent)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			_, exists := result["stall_reason"]
			if tt.expectReason && !exists {
				t.Errorf("Expected stall_reason in JSON, got: %s", string(data))
			}
			if !tt.expectReason && exists {
				t.Errorf("Expected stall_reason omitted from JSON, got: %s", string(data))
			}
			if tt.expectReason {
				if got := result["stall_reason"].(string); got != tt.stallReason {
					t.Errorf("Expected stall_reason=%q, got %q", tt.stallReason, got)
				}
			}
		})
	}
}

func TestAgentAPIResponseEscalationLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expectIn bool // expect escalation_level in JSON output
	}{
		{"empty omitted", "", false},
		{"none included", "none", true},
		{"info included", "info", true},
		{"review included", "review", true},
		{"block included", "block", true},
		{"failed included", "failed", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := AgentAPIResponse{
				ID:              "test-agent",
				Status:          "completed",
				EscalationLevel: tt.level,
			}

			data, err := json.Marshal(agent)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			_, exists := result["escalation_level"]
			if tt.expectIn && !exists {
				t.Errorf("Expected escalation_level in JSON, got: %s", string(data))
			}
			if !tt.expectIn && exists {
				t.Errorf("Expected escalation_level omitted from JSON, got: %s", string(data))
			}
			if tt.expectIn {
				if got := result["escalation_level"].(string); got != tt.level {
					t.Errorf("Expected escalation_level=%q, got %q", tt.level, got)
				}
			}
		})
	}
}
