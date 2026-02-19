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
