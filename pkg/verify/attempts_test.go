package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFixAttemptStats_IsRetryPattern(t *testing.T) {
	tests := []struct {
		name     string
		stats    FixAttemptStats
		expected bool
	}{
		{
			name: "no spawns",
			stats: FixAttemptStats{
				SpawnCount:     0,
				AbandonedCount: 0,
			},
			expected: false,
		},
		{
			name: "single spawn no abandon",
			stats: FixAttemptStats{
				SpawnCount:     1,
				AbandonedCount: 0,
			},
			expected: false,
		},
		{
			name: "single spawn with abandon",
			stats: FixAttemptStats{
				SpawnCount:     1,
				AbandonedCount: 1,
			},
			expected: false, // Need multiple spawns for retry pattern
		},
		{
			name: "multiple spawns with abandon",
			stats: FixAttemptStats{
				SpawnCount:     3,
				AbandonedCount: 2,
			},
			expected: true,
		},
		{
			name: "multiple spawns no abandon",
			stats: FixAttemptStats{
				SpawnCount:     3,
				AbandonedCount: 0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stats.IsRetryPattern(); got != tt.expected {
				t.Errorf("IsRetryPattern() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFixAttemptStats_IsPersistentFailure(t *testing.T) {
	tests := []struct {
		name     string
		stats    FixAttemptStats
		expected bool
	}{
		{
			name: "no failures",
			stats: FixAttemptStats{
				SpawnCount:     1,
				CompletedCount: 1,
				AbandonedCount: 0,
			},
			expected: false,
		},
		{
			name: "single failure",
			stats: FixAttemptStats{
				SpawnCount:     1,
				CompletedCount: 0,
				AbandonedCount: 1,
			},
			expected: false, // Need multiple spawns
		},
		{
			name: "persistent failure pattern",
			stats: FixAttemptStats{
				SpawnCount:     3,
				CompletedCount: 0,
				AbandonedCount: 2,
			},
			expected: true,
		},
		{
			name: "eventual success breaks pattern",
			stats: FixAttemptStats{
				SpawnCount:     3,
				CompletedCount: 1,
				AbandonedCount: 2,
			},
			expected: false, // Has a completion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stats.IsPersistentFailure(); got != tt.expected {
				t.Errorf("IsPersistentFailure() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFixAttemptStats_SuggestedAction(t *testing.T) {
	tests := []struct {
		name     string
		stats    FixAttemptStats
		expected string
	}{
		{
			name: "no pattern",
			stats: FixAttemptStats{
				SpawnCount:     1,
				AbandonedCount: 0,
			},
			expected: "",
		},
		{
			name: "retry pattern suggests investigation",
			stats: FixAttemptStats{
				SpawnCount:     2,
				AbandonedCount: 1,
			},
			expected: "investigate-root-cause",
		},
		{
			name: "persistent failure suggests reliability testing",
			stats: FixAttemptStats{
				SpawnCount:     3,
				AbandonedCount: 2,
				CompletedCount: 0,
			},
			expected: "reliability-testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stats.SuggestedAction(); got != tt.expected {
				t.Errorf("SuggestedAction() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetFixAttemptStatsFromPath(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	eventsPath := filepath.Join(tempDir, "events.jsonl")

	t.Run("missing events file returns empty stats", func(t *testing.T) {
		stats, err := GetFixAttemptStatsFromPath("test-123", eventsPath)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if stats.SpawnCount != 0 {
			t.Errorf("Expected 0 spawns, got %d", stats.SpawnCount)
		}
	})

	t.Run("counts spawns and abandonments correctly", func(t *testing.T) {
		events := `{"type":"session.spawned","timestamp":1703001600,"data":{"beads_id":"test-123","skill":"feature-impl"}}
{"type":"agent.abandoned","timestamp":1703002600,"data":{"beads_id":"test-123","reason":"stuck"}}
{"type":"session.spawned","timestamp":1703003600,"data":{"beads_id":"test-123","skill":"systematic-debugging"}}
{"type":"agent.completed","timestamp":1703004600,"data":{"beads_id":"test-123"}}
{"type":"session.spawned","timestamp":1703005600,"data":{"beads_id":"other-456","skill":"feature-impl"}}
`
		if err := os.WriteFile(eventsPath, []byte(events), 0644); err != nil {
			t.Fatalf("Failed to write events file: %v", err)
		}

		stats, err := GetFixAttemptStatsFromPath("test-123", eventsPath)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if stats.SpawnCount != 2 {
			t.Errorf("SpawnCount = %d, want 2", stats.SpawnCount)
		}
		if stats.AbandonedCount != 1 {
			t.Errorf("AbandonedCount = %d, want 1", stats.AbandonedCount)
		}
		if stats.CompletedCount != 1 {
			t.Errorf("CompletedCount = %d, want 1", stats.CompletedCount)
		}
		if stats.LastOutcome != "completed" {
			t.Errorf("LastOutcome = %s, want completed", stats.LastOutcome)
		}
		if len(stats.Skills) != 2 {
			t.Errorf("Skills count = %d, want 2", len(stats.Skills))
		}
	})

	t.Run("handles malformed events gracefully", func(t *testing.T) {
		events := `{"type":"session.spawned","timestamp":1703001600,"data":{"beads_id":"test-123"}}
not valid json
{"type":"agent.abandoned","timestamp":1703002600,"data":{"beads_id":"test-123"}}
`
		if err := os.WriteFile(eventsPath, []byte(events), 0644); err != nil {
			t.Fatalf("Failed to write events file: %v", err)
		}

		stats, err := GetFixAttemptStatsFromPath("test-123", eventsPath)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should still parse valid events
		if stats.SpawnCount != 1 || stats.AbandonedCount != 1 {
			t.Errorf("Expected 1 spawn and 1 abandon, got %d and %d", stats.SpawnCount, stats.AbandonedCount)
		}
	})
}

func TestGetAllRetryPatternsFromPath(t *testing.T) {
	tempDir := t.TempDir()
	eventsPath := filepath.Join(tempDir, "events.jsonl")

	events := `{"type":"session.spawned","timestamp":1703001600,"data":{"beads_id":"flaky-001"}}
{"type":"agent.abandoned","timestamp":1703002600,"data":{"beads_id":"flaky-001"}}
{"type":"session.spawned","timestamp":1703003600,"data":{"beads_id":"flaky-001"}}
{"type":"agent.abandoned","timestamp":1703004600,"data":{"beads_id":"flaky-001"}}
{"type":"session.spawned","timestamp":1703005600,"data":{"beads_id":"ok-002"}}
{"type":"agent.completed","timestamp":1703006600,"data":{"beads_id":"ok-002"}}
{"type":"session.spawned","timestamp":1703007600,"data":{"beads_id":"retry-003"}}
{"type":"agent.abandoned","timestamp":1703008600,"data":{"beads_id":"retry-003"}}
{"type":"session.spawned","timestamp":1703009600,"data":{"beads_id":"retry-003"}}
`

	if err := os.WriteFile(eventsPath, []byte(events), 0644); err != nil {
		t.Fatalf("Failed to write events file: %v", err)
	}

	patterns, err := GetAllRetryPatternsFromPath(eventsPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should find flaky-001 (persistent failure) and retry-003 (retry pattern)
	// ok-002 should NOT be included (single spawn, completed)
	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(patterns))
	}

	// Check sorting - persistent failures first
	if len(patterns) >= 1 && patterns[0].BeadsID != "flaky-001" {
		t.Errorf("Expected flaky-001 first (persistent failure), got %s", patterns[0].BeadsID)
	}
}

func TestFormatRetryWarning(t *testing.T) {
	t.Run("nil stats returns empty", func(t *testing.T) {
		result := FormatRetryWarning(nil)
		if result != "" {
			t.Errorf("Expected empty string for nil stats")
		}
	})

	t.Run("no retry pattern returns empty", func(t *testing.T) {
		stats := &FixAttemptStats{SpawnCount: 1, AbandonedCount: 0}
		result := FormatRetryWarning(stats)
		if result != "" {
			t.Errorf("Expected empty string for non-retry pattern")
		}
	})

	t.Run("retry pattern shows warning", func(t *testing.T) {
		stats := &FixAttemptStats{
			SpawnCount:     2,
			AbandonedCount: 1,
			CompletedCount: 0,
			Skills:         []string{"feature-impl"},
		}
		result := FormatRetryWarning(stats)
		if !strings.Contains(result, "RETRY PATTERN") {
			t.Errorf("Expected warning to contain 'RETRY PATTERN', got: %s", result)
		}
		if !strings.Contains(result, "investigate-root-cause") {
			t.Errorf("Expected suggestion for investigate-root-cause")
		}
	})

	t.Run("persistent failure shows critical warning", func(t *testing.T) {
		stats := &FixAttemptStats{
			SpawnCount:     3,
			AbandonedCount: 2,
			CompletedCount: 0,
			Skills:         []string{"feature-impl", "systematic-debugging"},
		}
		result := FormatRetryWarning(stats)
		if !strings.Contains(result, "PERSISTENT FAILURE") {
			t.Errorf("Expected warning to contain 'PERSISTENT FAILURE', got: %s", result)
		}
		if !strings.Contains(result, "reliability-testing") {
			t.Errorf("Expected suggestion for reliability-testing")
		}
	})
}

func TestFixAttemptStats_LastAttemptAt(t *testing.T) {
	tempDir := t.TempDir()
	eventsPath := filepath.Join(tempDir, "events.jsonl")

	now := time.Now().Unix()
	events := `{"type":"session.spawned","timestamp":1703001600,"data":{"beads_id":"test-123"}}
{"type":"agent.abandoned","timestamp":` + itoa(int(now)) + `,"data":{"beads_id":"test-123"}}
`
	if err := os.WriteFile(eventsPath, []byte(events), 0644); err != nil {
		t.Fatalf("Failed to write events file: %v", err)
	}

	stats, err := GetFixAttemptStatsFromPath("test-123", eventsPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Last attempt should be the more recent timestamp
	if stats.LastAttemptAt.Unix() != now {
		t.Errorf("LastAttemptAt = %v, want ~%v", stats.LastAttemptAt.Unix(), now)
	}
}
