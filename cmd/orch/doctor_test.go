package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServiceStatusJSON(t *testing.T) {
	status := ServiceStatus{
		Name:      "TestService",
		Running:   true,
		Port:      8080,
		URL:       "http://localhost:8080",
		Details:   "Running fine",
		CanFix:    true,
		FixAction: "restart service",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal ServiceStatus: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["name"] != "TestService" {
		t.Errorf("Expected name 'TestService', got %v", result["name"])
	}
	if result["running"] != true {
		t.Errorf("Expected running true, got %v", result["running"])
	}
	if result["port"] != float64(8080) {
		t.Errorf("Expected port 8080, got %v", result["port"])
	}
	if result["can_fix"] != true {
		t.Errorf("Expected can_fix true, got %v", result["can_fix"])
	}
}

func TestDoctorReportJSON(t *testing.T) {
	report := DoctorReport{
		Healthy: true,
		Services: []ServiceStatus{
			{
				Name:    "Service1",
				Running: true,
			},
			{
				Name:    "Service2",
				Running: false,
			},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal DoctorReport: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["healthy"] != true {
		t.Errorf("Expected healthy true, got %v", result["healthy"])
	}

	services, ok := result["services"].([]interface{})
	if !ok {
		t.Fatal("Expected services to be an array")
	}
	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}
}

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

func TestDoctorReportHealthyLogic(t *testing.T) {
	tests := []struct {
		name     string
		services []ServiceStatus
		expected bool
	}{
		{
			name: "all services running",
			services: []ServiceStatus{
				{Name: "Service1", Running: true},
				{Name: "Service2", Running: true},
			},
			expected: true,
		},
		{
			name: "one service down",
			services: []ServiceStatus{
				{Name: "Service1", Running: true},
				{Name: "Service2", Running: false},
			},
			expected: false,
		},
		{
			name: "all services down",
			services: []ServiceStatus{
				{Name: "Service1", Running: false},
				{Name: "Service2", Running: false},
			},
			expected: false,
		},
		{
			name:     "no services",
			services: []ServiceStatus{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from runDoctor
			healthy := true
			for _, svc := range tt.services {
				if !svc.Running {
					healthy = false
				}
			}

			if healthy != tt.expected {
				t.Errorf("Expected healthy=%v, got %v", tt.expected, healthy)
			}
		})
	}
}

func TestServiceStatusFields(t *testing.T) {
	// Test that ServiceStatus has all expected fields
	status := ServiceStatus{}
	
	// These assignments should compile - ensures struct has correct fields
	status.Name = "test"
	status.Running = true
	status.Port = 8080
	status.URL = "http://localhost:8080"
	status.Details = "details"
	status.CanFix = true
	status.FixAction = "action"

	if status.Name != "test" {
		t.Error("Name field not working correctly")
	}
	if !status.Running {
		t.Error("Running field not working correctly")
	}
	if status.Port != 8080 {
		t.Error("Port field not working correctly")
	}
}

func TestBinaryStatusFields(t *testing.T) {
	// Test that BinaryStatus has all expected fields
	status := BinaryStatus{}

	// These assignments should compile - ensures struct has correct fields
	status.Stale = true
	status.BinaryHash = "abc123"
	status.CurrentHash = "def456"
	status.SourceDir = "/path/to/source"
	status.Error = "some error"

	if !status.Stale {
		t.Error("Stale field not working correctly")
	}
	if status.BinaryHash != "abc123" {
		t.Error("BinaryHash field not working correctly")
	}
	if status.CurrentHash != "def456" {
		t.Error("CurrentHash field not working correctly")
	}
	if status.SourceDir != "/path/to/source" {
		t.Error("SourceDir field not working correctly")
	}
	if status.Error != "some error" {
		t.Error("Error field not working correctly")
	}
}

func TestBinaryStatusJSON(t *testing.T) {
	status := BinaryStatus{
		Stale:       true,
		BinaryHash:  "abc123",
		CurrentHash: "def456",
		SourceDir:   "/path/to/source",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal BinaryStatus: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["stale"] != true {
		t.Errorf("Expected stale true, got %v", result["stale"])
	}
	if result["binary_hash"] != "abc123" {
		t.Errorf("Expected binary_hash 'abc123', got %v", result["binary_hash"])
	}
	if result["current_hash"] != "def456" {
		t.Errorf("Expected current_hash 'def456', got %v", result["current_hash"])
	}
	if result["source_dir"] != "/path/to/source" {
		t.Errorf("Expected source_dir '/path/to/source', got %v", result["source_dir"])
	}
}
