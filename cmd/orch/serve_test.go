package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestHandleAgents(t *testing.T) {
	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()

	// Call the handler
	handleAgents(w, req)

	// Check the response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify it returns valid JSON (even if empty array)
	var agents []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
}

func TestHandleAgentsMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/agents", nil)
	w := httptest.NewRecorder()

	handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleEventsMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/events", nil)
	w := httptest.NewRecorder()

	handleEvents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestAgentAPIResponseJSONFormat(t *testing.T) {
	// Test that AgentAPIResponse serializes correctly to JSON
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

	// Verify the JSON contains expected fields
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

func TestHandleAgentlogMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/agentlog", nil)
	w := httptest.NewRecorder()

	handleAgentlog(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleAgentlogEmptyFile(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	tmpLogPath := filepath.Join(tmpDir, "events.jsonl")

	// Test with non-existent file - should return empty array
	eventList, err := readLastNEvents(tmpLogPath, 100)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Create empty file
	if err := os.WriteFile(tmpLogPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	eventList, err = readLastNEvents(tmpLogPath, 100)
	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}
	if len(eventList) != 0 {
		t.Errorf("Expected empty event list, got %d events", len(eventList))
	}
}

func TestReadLastNEvents(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	tmpLogPath := filepath.Join(tmpDir, "events.jsonl")

	// Create test events
	testEvents := []events.Event{
		{Type: "session.spawned", SessionID: "sess1", Timestamp: time.Now().Unix()},
		{Type: "session.status", SessionID: "sess1", Timestamp: time.Now().Unix()},
		{Type: "session.completed", SessionID: "sess1", Timestamp: time.Now().Unix()},
	}

	// Write events to file
	file, err := os.Create(tmpLogPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	for _, event := range testEvents {
		data, _ := json.Marshal(event)
		file.Write(append(data, '\n'))
	}
	file.Close()

	// Test reading all events
	eventList, err := readLastNEvents(tmpLogPath, 100)
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}
	if len(eventList) != 3 {
		t.Errorf("Expected 3 events, got %d", len(eventList))
	}

	// Test reading last 2 events
	eventList, err = readLastNEvents(tmpLogPath, 2)
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}
	if len(eventList) != 2 {
		t.Errorf("Expected 2 events, got %d", len(eventList))
	}
	if eventList[0].Type != "session.status" {
		t.Errorf("Expected first event to be session.status, got %s", eventList[0].Type)
	}
	if eventList[1].Type != "session.completed" {
		t.Errorf("Expected second event to be session.completed, got %s", eventList[1].Type)
	}
}

func TestHandleAgentlogJSONResponse(t *testing.T) {
	// Note: This test uses the default log path which may or may not exist
	// In production, we'd want to inject the path, but for now we just verify
	// the endpoint returns valid JSON
	req := httptest.NewRequest(http.MethodGet, "/api/agentlog", nil)
	w := httptest.NewRecorder()

	handleAgentlog(w, req)

	resp := w.Result()
	// Should be 200 even if file doesn't exist (returns empty array)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify it returns valid JSON array
	var eventList []events.Event
	if err := json.NewDecoder(resp.Body).Decode(&eventList); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
}

func TestHandleUsageMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/usage", nil)
	w := httptest.NewRecorder()

	handleUsage(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleUsageJSONResponse(t *testing.T) {
	// Test that usage endpoint returns valid JSON
	req := httptest.NewRequest(http.MethodGet, "/api/usage", nil)
	w := httptest.NewRecorder()

	handleUsage(w, req)

	resp := w.Result()
	// Should be 200 even if auth fails (returns error in JSON)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify it returns valid JSON
	var usageResp UsageAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageResp); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}

	// Response should either have data or an error
	// If no auth is configured, we expect an error message
	if usageResp.Error == "" && usageResp.Account == "" && usageResp.FiveHour == 0 && usageResp.Weekly == 0 {
		t.Log("Usage response has no data and no error - auth may be working")
	}
}

func TestUsageAPIResponseJSONFormat(t *testing.T) {
	// Test that UsageAPIResponse serializes correctly to JSON
	usage := &UsageAPIResponse{
		Account:    "test@example.com",
		FiveHour:   45.5,
		Weekly:     72.3,
		WeeklyOpus: 15.0,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("Failed to marshal UsageAPIResponse: %v", err)
	}

	// Verify the JSON contains expected fields
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["account"] != "test@example.com" {
		t.Errorf("Expected account 'test@example.com', got %v", result["account"])
	}
	if result["five_hour_percent"] != 45.5 {
		t.Errorf("Expected five_hour_percent 45.5, got %v", result["five_hour_percent"])
	}
	if result["weekly_percent"] != 72.3 {
		t.Errorf("Expected weekly_percent 72.3, got %v", result["weekly_percent"])
	}
	if result["weekly_opus_percent"] != 15.0 {
		t.Errorf("Expected weekly_opus_percent 15.0, got %v", result["weekly_opus_percent"])
	}
}
