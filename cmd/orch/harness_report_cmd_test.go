package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHarnessReportCmd_Flags(t *testing.T) {
	// Verify command exists and has expected flags
	cmd := harnessReportCmd
	if cmd.Use != "report" {
		t.Errorf("expected Use='report', got %q", cmd.Use)
	}

	// Check flags exist
	flags := []string{"days", "json", "verbose"}
	for _, name := range flags {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("expected flag --%s to exist", name)
		}
	}
}

func TestHarnessReportCmd_IsRegistered(t *testing.T) {
	found := false
	for _, cmd := range harnessCmd.Commands() {
		if cmd.Use == "report" {
			found = true
			break
		}
	}
	if !found {
		t.Error("harness report command not registered as subcommand of harness")
	}
}

func TestFormatHarnessText_EmptyReport(t *testing.T) {
	resp := buildEmptyHarnessResponse(7)
	output := formatHarnessText(resp, false)
	if output == "" {
		t.Error("expected non-empty text output for empty report")
	}
	// Should contain key sections
	for _, section := range []string{"GATE DEFLECTION", "FALSIFICATION VERDICTS", "COMPLETION COVERAGE"} {
		if !contains(output, section) {
			t.Errorf("expected output to contain %q", section)
		}
	}
}

func TestFormatHarnessText_WithData(t *testing.T) {
	// Write test events
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")

	now := time.Now().Unix()
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now - 3600},
		{Type: "session.spawned", Timestamp: now - 7200},
		{Type: "session.spawned", Timestamp: now - 10800},
		{Type: "spawn.hotspot_bypassed", Timestamp: now - 3600, Data: map[string]interface{}{"skill": "feature-impl"}},
		{Type: "spawn.triage_bypassed", Timestamp: now - 7200, Data: map[string]interface{}{"skill": "investigation"}},
		{Type: "agent.completed", Timestamp: now - 1800, Data: map[string]interface{}{
			"skill":            "feature-impl",
			"outcome":          "completed",
			"duration_minutes": 45.0,
		}},
		{Type: "agent.completed", Timestamp: now - 900, Data: map[string]interface{}{
			"skill":   "investigation",
			"outcome": "completed",
		}},
	}

	f, err := os.Create(eventsPath)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, e := range events {
		enc.Encode(e)
	}
	f.Close()

	// Parse and build response
	parsed, err := parseEvents(eventsPath)
	if err != nil {
		t.Fatal(err)
	}
	resp := buildHarnessResponse(parsed, 7)

	// Test non-verbose
	output := formatHarnessText(resp, false)
	if !strings.Contains(output, "3 spawns") {
		t.Errorf("expected '3 spawns' in output, got:\n%s", output)
	}
	if !contains(output, "GATE DEFLECTION") {
		t.Errorf("expected GATE DEFLECTION section")
	}

	// Test verbose
	verboseOutput := formatHarnessText(resp, true)
	if !contains(verboseOutput, "MEASUREMENT COVERAGE") {
		t.Errorf("expected MEASUREMENT COVERAGE in verbose output")
	}
}

func TestFormatHarnessJSON(t *testing.T) {
	resp := buildEmptyHarnessResponse(7)
	output, err := formatHarnessJSON(resp)
	if err != nil {
		t.Fatal(err)
	}

	// Should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("output should be valid JSON: %v", err)
	}

	// Should contain key fields
	if _, ok := parsed["falsification_verdicts"]; !ok {
		t.Error("expected falsification_verdicts in JSON output")
	}
}

func TestVerdictSymbol(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"falsified", "✓"},
		{"confirmed", "✗"},
		{"insufficient_data", "…"},
		{"not_measurable", "?"},
		{"unknown", "?"},
	}
	for _, tt := range tests {
		got := verdictSymbol(tt.status)
		if got != tt.want {
			t.Errorf("verdictSymbol(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestVerdictLabel(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"falsified", "FALSIFIED"},
		{"confirmed", "CONFIRMED"},
		{"insufficient_data", "INSUFFICIENT DATA"},
		{"not_measurable", "NOT MEASURABLE"},
	}
	for _, tt := range tests {
		got := verdictLabel(tt.status)
		if got != tt.want {
			t.Errorf("verdictLabel(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

// contains and stringContains are defined in resume_test.go / session_resume_test.go
