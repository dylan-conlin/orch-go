package launchd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		name     string
		pd       plistData
		expected string
	}{
		{
			name: "StartCalendarInterval hour and minute",
			pd: plistData{
				StartCalendarInterval: map[string]interface{}{
					"Hour":   uint64(4),
					"Minute": uint64(30),
				},
			},
			expected: "04:30",
		},
		{
			name: "StartCalendarInterval hour only",
			pd: plistData{
				StartCalendarInterval: map[string]interface{}{
					"Hour": uint64(12),
				},
			},
			expected: "12:00",
		},
		{
			name: "StartCalendarInterval minute only",
			pd: plistData{
				StartCalendarInterval: map[string]interface{}{
					"Minute": uint64(15),
				},
			},
			expected: "*:15",
		},
		{
			name: "StartCalendarInterval with weekday",
			pd: plistData{
				StartCalendarInterval: map[string]interface{}{
					"Hour":    uint64(9),
					"Minute":  uint64(0),
					"Weekday": uint64(1),
				},
			},
			expected: "09:00 Mon",
		},
		{
			name: "StartInterval seconds",
			pd: plistData{
				StartInterval: 30,
			},
			expected: "every 30s",
		},
		{
			name: "StartInterval minutes",
			pd: plistData{
				StartInterval: 300,
			},
			expected: "every 5m",
		},
		{
			name: "StartInterval hours",
			pd: plistData{
				StartInterval: 7200,
			},
			expected: "every 2h",
		},
		{
			name: "StartInterval days",
			pd: plistData{
				StartInterval: 172800,
			},
			expected: "every 2d",
		},
		{
			name: "WatchPaths",
			pd: plistData{
				WatchPaths: []string{"/some/path"},
			},
			expected: "file-triggered",
		},
		{
			name: "RunAtLoad only",
			pd: plistData{
				RunAtLoad: true,
			},
			expected: "on load",
		},
		{
			name: "manual",
			pd:       plistData{},
			expected: "manual",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseSchedule(tc.pd)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestFormatInterval(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{30, "every 30s"},
		{60, "every 1m"},
		{300, "every 5m"},
		{3600, "every 1h"},
		{7200, "every 2h"},
		{86400, "every 1d"},
		{172800, "every 2d"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatInterval(tc.seconds)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestAgent_Status(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		expected string
	}{
		{
			name:     "not loaded",
			agent:    Agent{Loaded: false},
			expected: "not loaded",
		},
		{
			name:     "running",
			agent:    Agent{Loaded: true, Running: true, PID: 12345},
			expected: "running (PID 12345)",
		},
		{
			name:     "idle",
			agent:    Agent{Loaded: true, Running: false},
			expected: "idle",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.agent.Status()
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestAgent_HasFailure(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		expected bool
	}{
		{
			name:     "not loaded",
			agent:    Agent{Loaded: false, LastExitCode: 1},
			expected: false,
		},
		{
			name:     "loaded with exit 0",
			agent:    Agent{Loaded: true, LastExitCode: 0},
			expected: false,
		},
		{
			name:     "loaded with exit 1",
			agent:    Agent{Loaded: true, LastExitCode: 1},
			expected: true,
		},
		{
			name:     "loaded with exit 78",
			agent:    Agent{Loaded: true, LastExitCode: 78},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.agent.HasFailure()
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestParsePlist(t *testing.T) {
	// Create a temp plist file for testing
	tmpDir := t.TempDir()
	plistPath := filepath.Join(tmpDir, "com.test.agent.plist")

	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.test.agent</string>
    <key>RunAtLoad</key>
    <true/>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>4</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
</dict>
</plist>`

	err := os.WriteFile(plistPath, []byte(plistContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test plist: %v", err)
	}

	agent, err := ParsePlist(plistPath)
	if err != nil {
		t.Fatalf("ParsePlist failed: %v", err)
	}

	if agent.Label != "com.test.agent" {
		t.Errorf("expected label 'com.test.agent', got %q", agent.Label)
	}

	if !agent.RunAtLoad {
		t.Error("expected RunAtLoad to be true")
	}

	if agent.Schedule != "04:00" {
		t.Errorf("expected schedule '04:00', got %q", agent.Schedule)
	}
}

func TestDefaultScanOptions(t *testing.T) {
	opts := DefaultScanOptions()

	// Check directory contains LaunchAgents
	if !filepath.IsAbs(opts.Directory) {
		t.Errorf("expected absolute path, got %q", opts.Directory)
	}

	if !contains(opts.Directory, "LaunchAgents") {
		t.Errorf("expected path to contain 'LaunchAgents', got %q", opts.Directory)
	}

	// Check prefixes
	expectedPrefixes := []string{"com.dylan.", "com.user.", "com.orch.", "com.cdd."}
	for _, prefix := range expectedPrefixes {
		found := false
		for _, p := range opts.Prefixes {
			if p == prefix {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected prefix %q not found in options", prefix)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
