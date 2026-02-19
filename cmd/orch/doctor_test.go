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

func TestSessionsCrossReferenceReportJSON(t *testing.T) {
	report := SessionsCrossReferenceReport{
		WorkspaceCount:       100,
		SessionCount:         50,
		OrphanedWorkspaces:   3,
		OrphanedSessions:     2,
		ZombieSessions:       1,
		OrphanedWorkspaceIDs: []string{"ws1", "ws2", "ws3"},
		OrphanedSessionIDs:   []string{"ses_1", "ses_2"},
		ZombieSessionIDs:     []string{"ses_zombie"},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal SessionsCrossReferenceReport: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["workspace_count"] != float64(100) {
		t.Errorf("Expected workspace_count 100, got %v", result["workspace_count"])
	}
	if result["session_count"] != float64(50) {
		t.Errorf("Expected session_count 50, got %v", result["session_count"])
	}
	if result["orphaned_workspaces"] != float64(3) {
		t.Errorf("Expected orphaned_workspaces 3, got %v", result["orphaned_workspaces"])
	}
	if result["orphaned_sessions"] != float64(2) {
		t.Errorf("Expected orphaned_sessions 2, got %v", result["orphaned_sessions"])
	}
	if result["zombie_sessions"] != float64(1) {
		t.Errorf("Expected zombie_sessions 1, got %v", result["zombie_sessions"])
	}
}

// =============================================================================
// Tests for ConfigDrift detection
// =============================================================================

func TestConfigDriftReportJSON(t *testing.T) {
	report := ConfigDriftReport{
		Healthy:    false,
		PlistFound: true,
		Drifts: []ConfigDrift{
			{Field: "poll_interval", Expected: "60", Actual: "30"},
			{Field: "reflect_issues", Expected: "false", Actual: "true"},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal ConfigDriftReport: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["healthy"] != false {
		t.Errorf("Expected healthy false, got %v", result["healthy"])
	}
	if result["plist_found"] != true {
		t.Errorf("Expected plist_found true, got %v", result["plist_found"])
	}

	drifts, ok := result["drifts"].([]interface{})
	if !ok {
		t.Fatal("Expected drifts to be an array")
	}
	if len(drifts) != 2 {
		t.Errorf("Expected 2 drifts, got %d", len(drifts))
	}
}

func TestParsePlistValues(t *testing.T) {
	// Sample plist content
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.orch.daemon</string>

    <key>ProgramArguments</key>
    <array>
        <string>/Users/test/bin/orch</string>
        <string>daemon</string>
        <string>run</string>
        <string>--poll-interval</string>
        <string>60</string>
        <string>--max-agents</string>
        <string>3</string>
        <string>--label</string>
        <string>triage:ready</string>
		<string>--verbose</string>
		<string>--reflect-issues=false</string>
		<string>--reflect-open=true</string>
	</array>

    <key>WorkingDirectory</key>
    <string>/Users/test/Documents/personal/orch-go</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/Users/test/.bun/bin:/usr/bin:/bin</string>
    </dict>
</dict>
</plist>`

	values, err := parsePlistValues(plistContent)
	if err != nil {
		t.Fatalf("parsePlistValues() error = %v", err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"poll_interval", "60"},
		{"max_agents", "3"},
		{"label", "triage:ready"},
		{"verbose", "true"},
		{"reflect_issues", "false"},
		{"reflect_open", "true"},
		{"working_directory", "/Users/test/Documents/personal/orch-go"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := values[tt.key]; got != tt.expected {
				t.Errorf("parsePlistValues()[%q] = %q, want %q", tt.key, got, tt.expected)
			}
		})
	}
}

func TestParsePlistValuesWithoutVerbose(t *testing.T) {
	// Plist without --verbose flag
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>ProgramArguments</key>
    <array>
        <string>orch</string>
        <string>daemon</string>
        <string>run</string>
        <string>--poll-interval</string>
        <string>30</string>
    </array>
</dict>
</plist>`

	values, err := parsePlistValues(plistContent)
	if err != nil {
		t.Fatalf("parsePlistValues() error = %v", err)
	}

	// Without --verbose flag, should be false
	if values["verbose"] != "false" {
		t.Errorf("parsePlistValues() verbose = %q, want \"false\"", values["verbose"])
	}

	if values["poll_interval"] != "30" {
		t.Errorf("parsePlistValues() poll_interval = %q, want \"30\"", values["poll_interval"])
	}
}

func TestParsePlistValuesWithReflectIssuesTrue(t *testing.T) {
	// Plist with --reflect-issues=true
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>ProgramArguments</key>
    <array>
        <string>orch</string>
        <string>--reflect-issues=true</string>
        <string>--reflect-open=false</string>
    </array>
</dict>
</plist>`

	values, err := parsePlistValues(plistContent)
	if err != nil {
		t.Fatalf("parsePlistValues() error = %v", err)
	}

	if values["reflect_issues"] != "true" {
		t.Errorf("parsePlistValues() reflect_issues = %q, want \"true\"", values["reflect_issues"])
	}
	if values["reflect_open"] != "false" {
		t.Errorf("parsePlistValues() reflect_open = %q, want \"false\"", values["reflect_open"])
	}
}

func TestConfigDriftFields(t *testing.T) {
	// Test that ConfigDrift has all expected fields
	drift := ConfigDrift{
		Field:    "poll_interval",
		Expected: "60",
		Actual:   "30",
	}

	if drift.Field != "poll_interval" {
		t.Error("Field not working correctly")
	}
	if drift.Expected != "60" {
		t.Error("Expected not working correctly")
	}
	if drift.Actual != "30" {
		t.Error("Actual not working correctly")
	}
}

// =============================================================================
// Tests for Doctor Daemon self-healing functionality
// =============================================================================

func TestParseElapsedTime(t *testing.T) {
	tests := []struct {
		input    string
		expected string // Duration as string for easy comparison
	}{
		{"00:30", "30s"},            // 30 seconds
		{"05:30", "5m30s"},          // 5 minutes 30 seconds
		{"01:30:00", "1h30m0s"},     // 1 hour 30 minutes
		{"02:00:00", "2h0m0s"},      // 2 hours
		{"1-00:00:00", "24h0m0s"},   // 1 day
		{"2-12:30:45", "60h30m45s"}, // 2 days 12 hours 30 min 45 sec
		{"10:00", "10m0s"},          // 10 minutes
		{"", "0s"},                  // empty
		{"invalid", "0s"},           // invalid format
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseElapsedTime(tt.input)
			if result.String() != tt.expected {
				t.Errorf("parseElapsedTime(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDoctorDaemonConfig(t *testing.T) {
	config := DefaultDoctorDaemonConfig()

	if config.PollInterval.Seconds() != 30 {
		t.Errorf("Expected PollInterval 30s, got %v", config.PollInterval)
	}
	if config.OrphanedViteMaxAge.Minutes() != 5 {
		t.Errorf("Expected OrphanedViteMaxAge 5m, got %v", config.OrphanedViteMaxAge)
	}
	if config.LongRunningBdMaxAge.Minutes() != 10 {
		t.Errorf("Expected LongRunningBdMaxAge 10m, got %v", config.LongRunningBdMaxAge)
	}
	if config.LogPath == "" {
		t.Error("Expected LogPath to be set")
	}
}

func TestDoctorDaemonIntervention(t *testing.T) {
	intervention := DoctorDaemonIntervention{
		Type:    "kill_orphan_vite",
		Target:  "PID 12345",
		Reason:  "orphaned vite (PPID=1)",
		Success: true,
	}

	if intervention.Type != "kill_orphan_vite" {
		t.Error("Type field not working correctly")
	}
	if intervention.Target != "PID 12345" {
		t.Error("Target field not working correctly")
	}
	if !intervention.Success {
		t.Error("Success field not working correctly")
	}
}

func TestGetDoctorPlistPath(t *testing.T) {
	path := getDoctorPlistPath()
	if path == "" {
		t.Error("Expected non-empty plist path")
	}
	// Check that path contains expected filename
	expected := "com.orch.doctor.plist"
	found := false
	for i := 0; i <= len(path)-len(expected); i++ {
		if path[i:i+len(expected)] == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected path to contain '%s', got %s", expected, path)
	}
}
