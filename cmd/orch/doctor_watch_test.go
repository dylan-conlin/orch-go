package main

import (
	"testing"
)

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
