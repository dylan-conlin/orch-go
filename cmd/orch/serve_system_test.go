package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
	if usageResp.Error == "" && usageResp.Account == "" && usageResp.FiveHour == nil && usageResp.Weekly == nil {
		t.Log("Usage response has no data and no error - auth may be working")
	}
}

func TestUsageAPIResponseJSONFormat(t *testing.T) {
	// Test that UsageAPIResponse serializes correctly to JSON
	fiveHour := 45.5
	weekly := 72.3
	weeklyOpus := 15.0
	usage := &UsageAPIResponse{
		Account:    "test@example.com",
		FiveHour:   &fiveHour,
		Weekly:     &weekly,
		WeeklyOpus: &weeklyOpus,
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

func TestHandleDaemonReturnsCapacityFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/daemon", nil)
	w := httptest.NewRecorder()

	handleDaemon(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var daemonResp DaemonAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&daemonResp); err != nil {
		t.Fatalf("Failed to decode daemon response: %v", err)
	}

	// Verify capacity fields are present in the JSON (even when daemon isn't running)
	data, _ := json.Marshal(daemonResp)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	requiredFields := []string{"running", "capacity_max", "capacity_used", "capacity_free", "ready_count"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("Missing required field %q in daemon API response", field)
		}
	}
}

func TestHandleDaemonMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/daemon", nil)
	w := httptest.NewRecorder()

	handleDaemon(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestDaemonAPIResponseCapacityJSON(t *testing.T) {
	// Test that DaemonAPIResponse correctly serializes capacity fields
	resp := DaemonAPIResponse{
		Running:      true,
		Status:       "running",
		ReadyCount:   72,
		CapacityMax:  3,
		CapacityUsed: 2,
		CapacityFree: 1,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal DaemonAPIResponse: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["capacity_max"] != float64(3) {
		t.Errorf("Expected capacity_max 3, got %v", result["capacity_max"])
	}
	if result["capacity_used"] != float64(2) {
		t.Errorf("Expected capacity_used 2, got %v", result["capacity_used"])
	}
	if result["capacity_free"] != float64(1) {
		t.Errorf("Expected capacity_free 1, got %v", result["capacity_free"])
	}
	if result["ready_count"] != float64(72) {
		t.Errorf("Expected ready_count 72, got %v", result["ready_count"])
	}
}

func TestFormatDurationAgo(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"1 min ago", 1 * time.Minute, "1 min ago"},
		{"5 mins ago", 5 * time.Minute, "5 mins ago"},
		{"1 hour ago", 1 * time.Hour, "1 hour ago"},
		{"2 hours ago", 2 * time.Hour, "2 hours ago"},
		{"1 day ago", 24 * time.Hour, "1 day ago"},
		{"2 days ago", 48 * time.Hour, "2 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDurationAgo(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDurationAgo(%v) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}
