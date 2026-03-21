// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeUtilization(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name               string
		events             []UtilizationEvent
		days               int
		wantTotalSpawns    int
		wantDaemonSpawns   int
		wantManualSpawns   int
		wantDaemonRate     float64
		wantTriageBypassed int
	}{
		{
			name: "all daemon spawns",
			events: []UtilizationEvent{
				{Type: "session.spawned", Timestamp: now - 3600},
				{Type: "session.spawned", Timestamp: now - 7200},
				{Type: "daemon.spawn", Timestamp: now - 3600},
				{Type: "daemon.spawn", Timestamp: now - 7200},
			},
			days:               7,
			wantTotalSpawns:    2,
			wantDaemonSpawns:   2,
			wantManualSpawns:   0,
			wantDaemonRate:     100,
			wantTriageBypassed: 0,
		},
		{
			name: "mixed spawns",
			events: []UtilizationEvent{
				{Type: "session.spawned", Timestamp: now - 3600},
				{Type: "session.spawned", Timestamp: now - 7200},
				{Type: "session.spawned", Timestamp: now - 10800},
				{Type: "daemon.spawn", Timestamp: now - 3600},
				{Type: "spawn.triage_bypassed", Timestamp: now - 7200},
				{Type: "spawn.triage_bypassed", Timestamp: now - 10800},
			},
			days:               7,
			wantTotalSpawns:    3,
			wantDaemonSpawns:   1,
			wantManualSpawns:   2,
			wantDaemonRate:     33.33333333333333,
			wantTriageBypassed: 2,
		},
		{
			name: "events outside window excluded",
			events: []UtilizationEvent{
				{Type: "session.spawned", Timestamp: now - 3600},      // within 7 days
				{Type: "session.spawned", Timestamp: now - 864000},    // outside 7 days (10 days ago)
				{Type: "daemon.spawn", Timestamp: now - 3600},         // within 7 days
				{Type: "daemon.spawn", Timestamp: now - 864000},       // outside 7 days
				{Type: "spawn.triage_bypassed", Timestamp: now - 864000}, // outside 7 days
			},
			days:               7,
			wantTotalSpawns:    1,
			wantDaemonSpawns:   1,
			wantManualSpawns:   0,
			wantDaemonRate:     100,
			wantTriageBypassed: 0,
		},
		{
			name: "auto completions tracked",
			events: []UtilizationEvent{
				{Type: "session.spawned", Timestamp: now - 3600},
				{Type: "daemon.spawn", Timestamp: now - 3600},
				{Type: "session.auto_completed", Timestamp: now - 1800},
			},
			days:               7,
			wantTotalSpawns:    1,
			wantDaemonSpawns:   1,
			wantManualSpawns:   0,
			wantDaemonRate:     100,
			wantTriageBypassed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeUtilization(tt.events, tt.days)

			if result.TotalSpawns != tt.wantTotalSpawns {
				t.Errorf("TotalSpawns = %d, want %d", result.TotalSpawns, tt.wantTotalSpawns)
			}
			if result.DaemonSpawns != tt.wantDaemonSpawns {
				t.Errorf("DaemonSpawns = %d, want %d", result.DaemonSpawns, tt.wantDaemonSpawns)
			}
			if result.ManualSpawns != tt.wantManualSpawns {
				t.Errorf("ManualSpawns = %d, want %d", result.ManualSpawns, tt.wantManualSpawns)
			}
			if result.DaemonSpawnRate != tt.wantDaemonRate {
				t.Errorf("DaemonSpawnRate = %f, want %f", result.DaemonSpawnRate, tt.wantDaemonRate)
			}
			if result.TriageBypassed != tt.wantTriageBypassed {
				t.Errorf("TriageBypassed = %d, want %d", result.TriageBypassed, tt.wantTriageBypassed)
			}
		})
	}
}

func TestTriageSlipRateCapped(t *testing.T) {
	// Test that triage slip rate is capped at 100%
	now := time.Now().Unix()
	events := []UtilizationEvent{
		{Type: "session.spawned", Timestamp: now - 3600},
		{Type: "spawn.triage_bypassed", Timestamp: now - 3600},
		{Type: "spawn.triage_bypassed", Timestamp: now - 3500}, // More bypasses than spawns
		{Type: "spawn.triage_bypassed", Timestamp: now - 3400},
	}

	result := computeUtilization(events, 7)

	if result.TriageSlipRate > 100 {
		t.Errorf("TriageSlipRate = %f, want <= 100", result.TriageSlipRate)
	}
}

func TestFormatDays(t *testing.T) {
	tests := []struct {
		days int
		want string
	}{
		{1, "1 day"},
		{7, "7 days"},
		{30, "30 days"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatDays(tt.days); got != tt.want {
				t.Errorf("formatDays(%d) = %q, want %q", tt.days, got, tt.want)
			}
		})
	}
}

func TestParseUtilizationEvents(t *testing.T) {
	// Create a temp file with test events
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")

	now := time.Now().Unix()
	events := []UtilizationEvent{
		{Type: "session.spawned", SessionID: "test-1", Timestamp: now - 3600},
		{Type: "daemon.spawn", Timestamp: now - 3600},
	}

	f, err := os.Create(eventsPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	for _, e := range events {
		data, _ := json.Marshal(e)
		f.Write(append(data, '\n'))
	}
	f.Close()

	// Parse the file
	parsed, err := parseUtilizationEvents(eventsPath, 0)
	if err != nil {
		t.Fatalf("parseUtilizationEvents failed: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("len(parsed) = %d, want 2", len(parsed))
	}
}

func TestParseUtilizationEvents_MissingFile(t *testing.T) {
	// Non-existent file should return empty slice, not error
	events, err := parseUtilizationEvents("/nonexistent/events.jsonl", 0)
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Expected empty slice, got %d events", len(events))
	}
}

func TestDaemonSpawnsCannotExceedTotal(t *testing.T) {
	// Edge case: daemon.spawn events without corresponding session.spawned
	// (can happen if spawn fails after daemon event is logged)
	now := time.Now().Unix()
	events := []UtilizationEvent{
		{Type: "daemon.spawn", Timestamp: now - 3600},
		{Type: "daemon.spawn", Timestamp: now - 3500},
		// No session.spawned events (spawns failed)
	}

	result := computeUtilization(events, 7)

	// Daemon spawns should be capped at total spawns (0)
	if result.DaemonSpawns > result.TotalSpawns {
		t.Errorf("DaemonSpawns (%d) > TotalSpawns (%d)", result.DaemonSpawns, result.TotalSpawns)
	}
}
