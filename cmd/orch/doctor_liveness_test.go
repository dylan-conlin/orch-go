package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOrchServeWithMockServer(t *testing.T) {
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

	// Note: We can't easily test checkOrchServe with a mock server
	// because it uses a hardcoded port. This test validates the
	// response parsing logic separately.

	// Test parsing a health response
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if health.Status != "ok" {
		t.Errorf("Expected status 'ok', got %s", health.Status)
	}
}

func TestCheckOrchServeServiceStatus(t *testing.T) {
	// Test that checkOrchServe returns a properly structured ServiceStatus
	// This tests the function behavior when the server is not running
	// (which is the common case in tests)
	status := checkOrchServe()

	// Basic structure checks
	if status.Name != "orch serve" {
		t.Errorf("Expected name 'orch serve', got %s", status.Name)
	}
	if status.Port != DefaultServePort {
		t.Errorf("Expected port %d, got %d", DefaultServePort, status.Port)
	}
	if status.URL != "https://localhost:3348" {
		t.Errorf("Expected URL 'https://localhost:3348', got %s", status.URL)
	}
	if !status.CanFix {
		t.Error("Expected CanFix to be true")
	}
	if status.FixAction != "Run: orch serve &" {
		t.Errorf("Expected FixAction 'Run: orch serve &', got %s", status.FixAction)
	}

	// When server is not running, Running should be false and Details should indicate not listening
	// (unless the server happens to be running during tests)
	if !status.Running && status.Details == "" {
		t.Error("Expected Details to be set when server is not running")
	}
}
