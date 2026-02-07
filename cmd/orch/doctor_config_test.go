package main

import (
	"encoding/json"
	"testing"
)

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
