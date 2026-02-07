package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// newTestServer creates a Server with minimal dependencies for unit testing.
// Tests that need specific fields can override them after creation.
func newTestServer() *Server {
	return &Server{
		ServerURL:       "http://127.0.0.1:4096",
		SourceDir:       os.TempDir(),
		Version:         "test",
		BeadsCache:      newBeadsCache(),
		BeadsStatsCache: newBeadsStatsCache(),
		KBHealthCache:   newKBHealthCache(),
		WorkspaceCache:  &globalWorkspaceCacheType{ttl: 30 * time.Second},
	}
}

func TestServeStatusWithMockServer(t *testing.T) {
	// Create a mock server that responds to /health
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Parse the port from the test server URL
	// The URL is in format http://127.0.0.1:PORT (httptest.Server always uses 127.0.0.1)
	var testPort int
	_, err := fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &testPort)
	if err != nil {
		t.Fatalf("Failed to parse test server port: %v", err)
	}

	// Call runServeStatus with the test port
	// This should succeed without error
	err = runServeStatus(testPort)
	if err != nil {
		t.Errorf("Expected no error from runServeStatus, got: %v", err)
	}
}

func TestServeStatusWithNoServer(t *testing.T) {
	// Use a port that is unlikely to be in use
	unusedPort := 59999

	// Call runServeStatus with the unused port
	// This should NOT return an error (it prints status and returns nil)
	err := runServeStatus(unusedPort)
	if err != nil {
		t.Errorf("Expected no error from runServeStatus (should print 'not running'), got: %v", err)
	}
}

func TestDefaultServePort(t *testing.T) {
	// Verify the default port constant
	if DefaultServePort != 3348 {
		t.Errorf("Expected DefaultServePort to be 3348, got %d", DefaultServePort)
	}
}

func TestHandleErrorsMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/errors", nil)
	w := httptest.NewRecorder()

	newTestServer().handleErrors(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleErrorsJSONResponse(t *testing.T) {
	// Test that errors endpoint returns valid JSON
	req := httptest.NewRequest(http.MethodGet, "/api/errors", nil)
	w := httptest.NewRecorder()

	newTestServer().handleErrors(w, req)

	resp := w.Result()
	// Should be 200 even if events file doesn't exist
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify it returns valid JSON
	var errorsResp ErrorsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorsResp); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}

	// Verify ByType map is initialized (not nil)
	if errorsResp.ByType == nil {
		t.Error("Expected ByType to be initialized map, got nil")
	}
}

func TestErrorsAPIResponseJSONFormat(t *testing.T) {
	// Test that ErrorsAPIResponse serializes correctly to JSON
	errors := &ErrorsAPIResponse{
		TotalErrors:    5,
		ErrorsLast24h:  2,
		ErrorsLast7d:   4,
		AbandonedCount: 3,
		SessionErrors:  2,
		RecentErrors: []ErrorEvent{
			{
				Type:      "agent.abandoned",
				BeadsID:   "test-abc123",
				Timestamp: "2025-12-26T12:00:00Z",
				Message:   "Stalled during execution",
				Skill:     "feature-impl",
			},
		},
		Patterns: []ErrorPattern{
			{
				Pattern:    "Stalled during",
				Count:      3,
				LastSeen:   "2025-12-26T12:00:00Z",
				BeadsIDs:   []string{"test-abc1", "test-abc2", "test-abc3"},
				Suggestion: "Check agent for long-running operations or API timeouts",
			},
		},
		ByType: map[string]int{
			"agent.abandoned": 3,
			"session.error":   2,
		},
	}

	data, err := json.Marshal(errors)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorsAPIResponse: %v", err)
	}

	// Verify the JSON contains expected fields
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["total_errors"] != float64(5) {
		t.Errorf("Expected total_errors 5, got %v", result["total_errors"])
	}
	if result["errors_last_24h"] != float64(2) {
		t.Errorf("Expected errors_last_24h 2, got %v", result["errors_last_24h"])
	}
	if result["abandoned_count"] != float64(3) {
		t.Errorf("Expected abandoned_count 3, got %v", result["abandoned_count"])
	}

	// Check patterns array exists
	if _, ok := result["patterns"]; !ok {
		t.Error("Expected patterns field in JSON")
	}

	// Check by_type map
	byType, ok := result["by_type"].(map[string]interface{})
	if !ok {
		t.Error("Expected by_type map in JSON")
	} else {
		if byType["agent.abandoned"] != float64(3) {
			t.Errorf("Expected by_type[agent.abandoned] 3, got %v", byType["agent.abandoned"])
		}
	}
}

func TestExtractSkillFromAgentID(t *testing.T) {
	tests := []struct {
		agentID  string
		expected string
	}{
		{"og-feat-test-26dec", "feature-impl"},
		{"og-debug-something-26dec", "systematic-debugging"},
		{"og-inv-investigation-26dec", "investigation"},
		{"og-arch-design-26dec", "architect"},
		{"og-work-session-26dec", "design-session"},
		{"og-unknown-test", "unknown"},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.agentID, func(t *testing.T) {
			result := extractSkillFromAgentID(tt.agentID)
			if result != tt.expected {
				t.Errorf("extractSkillFromAgentID(%q) = %q, want %q", tt.agentID, result, tt.expected)
			}
		})
	}
}

func TestNormalizeErrorMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple error", "Simple error"},
		{"  Trimmed  ", "Trimmed"},
		{"", ""},
		{
			// Long message should be truncated to 100 chars
			"This is a very long error message that exceeds the 100 character limit and should be truncated for pattern matching purposes",
			"This is a very long error message that exceeds the 100 character limit and should be truncated for p",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeErrorMessage(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeErrorMessage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		slice    []string
		s        string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := containsString(tt.slice, tt.s)
			if result != tt.expected {
				t.Errorf("containsString(%v, %q) = %v, want %v", tt.slice, tt.s, result, tt.expected)
			}
		})
	}
}

func TestSuggestRemediation(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{"Agent stalled during execution", "Check agent for long-running operations or API timeouts"},
		{"Connection timeout occurred", "Review API response times or increase timeout limits"},
		{"At capacity limit", "Increase daemon capacity or check for stuck agents"},
		{"Daemon not responding", "Check daemon logs at ~/.orch/daemon.log"},
		{"Missing context information", "Review spawn context for missing or incorrect information"},
		{"Connection refused", "Check network connectivity or API endpoint availability"},
		{"Unknown error", "Review agent workspace for more details"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := suggestRemediation(tt.pattern)
			if result != tt.expected {
				t.Errorf("suggestRemediation(%q) = %q, want %q", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestHandleErrorsWithTestData(t *testing.T) {
	// Create a temporary events file with test data
	tmpDir := t.TempDir()
	tmpLogPath := filepath.Join(tmpDir, "events.jsonl")

	now := time.Now()
	testEvents := []events.Event{
		{
			Type:      "session.error",
			SessionID: "sess_123",
			Timestamp: now.Add(-1 * time.Hour).Unix(),
			Data:      map[string]interface{}{"error": "Connection timeout"},
		},
		{
			Type:      "agent.abandoned",
			SessionID: "",
			Timestamp: now.Add(-2 * time.Hour).Unix(),
			Data: map[string]interface{}{
				"beads_id":       "test-abc123",
				"reason":         "Stalled during lunch",
				"agent_id":       "og-feat-test-26dec",
				"workspace_path": "/path/to/workspace/og-feat-test-26dec",
			},
		},
		{
			Type:      "session.spawned", // Non-error event, should be skipped
			SessionID: "sess_456",
			Timestamp: now.Unix(),
			Data:      map[string]interface{}{"title": "test spawn"},
		},
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

	// Note: handleErrors uses events.DefaultLogPath() which we can't easily override
	// This test verifies the parsing logic through helper functions instead
	t.Run("verify error event types are recognized", func(t *testing.T) {
		if events.EventTypeSessionError != "session.error" {
			t.Errorf("Expected session.error, got %s", events.EventTypeSessionError)
		}
	})

	t.Run("verify skill extraction from agent abandoned events", func(t *testing.T) {
		skill := extractSkillFromAgentID("og-feat-test-26dec")
		if skill != "feature-impl" {
			t.Errorf("Expected feature-impl, got %s", skill)
		}
	})
}
