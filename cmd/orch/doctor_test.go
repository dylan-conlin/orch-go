package main

import (
	"encoding/json"
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
