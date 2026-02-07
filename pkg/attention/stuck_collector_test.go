package attention

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStuckCollector_Collect(t *testing.T) {
	now := time.Now()
	// Create timestamps
	threeHoursAgo := now.Add(-3 * time.Hour).Format(time.RFC3339)
	oneHourAgo := now.Add(-1 * time.Hour).Format(time.RFC3339)
	tenMinutesAgo := now.Add(-10 * time.Minute).Format(time.RFC3339)
	fiveHoursAgo := now.Add(-5 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name        string
		agents      []StuckAgentItem
		role        string
		expectCount int
		expectError bool
	}{
		{
			name: "finds stuck agent (>2h, no recent activity)",
			agents: []StuckAgentItem{
				{
					ID:             "session-1",
					BeadsID:        "orch-go-123",
					BeadsTitle:     "Test feature",
					Status:         "active",
					Phase:          "Implementing",
					Task:           "Implement test feature",
					Project:        "orch-go",
					Skill:          "feature-impl",
					IsStalled:      true, // Marked as stalled
					SpawnedAt:      threeHoursAgo,
					UpdatedAt:      threeHoursAgo,
					LastActivityAt: oneHourAgo, // No recent activity
				},
			},
			role:        "human",
			expectCount: 1,
		},
		{
			name: "skips agent with recent activity",
			agents: []StuckAgentItem{
				{
					ID:             "session-1",
					BeadsID:        "orch-go-123",
					Status:         "active",
					Phase:          "Implementing",
					Task:           "Active task",
					SpawnedAt:      threeHoursAgo,
					LastActivityAt: tenMinutesAgo, // Recent activity
					IsStalled:      false,
				},
			},
			role:        "human",
			expectCount: 0, // Recent activity, not stuck
		},
		{
			name: "skips agent under threshold",
			agents: []StuckAgentItem{
				{
					ID:             "session-1",
					BeadsID:        "orch-go-123",
					Status:         "active",
					Phase:          "Implementing",
					Task:           "New task",
					SpawnedAt:      oneHourAgo, // Only 1 hour old
					LastActivityAt: oneHourAgo,
					IsStalled:      false,
				},
			},
			role:        "human",
			expectCount: 0, // Under 2h threshold
		},
		{
			name: "filters out non-active agents",
			agents: []StuckAgentItem{
				{
					ID:        "session-1",
					BeadsID:   "orch-go-123",
					Status:    "completed", // Not active
					SpawnedAt: threeHoursAgo,
				},
				{
					ID:        "session-2",
					BeadsID:   "orch-go-456",
					Status:    "dead", // Not active
					SpawnedAt: threeHoursAgo,
				},
				{
					ID:        "session-3",
					BeadsID:   "orch-go-789",
					Status:    "awaiting-cleanup", // Not active
					SpawnedAt: threeHoursAgo,
				},
			},
			role:        "orchestrator",
			expectCount: 0, // None are active
		},
		{
			name: "skips agents without beads ID",
			agents: []StuckAgentItem{
				{
					ID:             "session-1",
					BeadsID:        "", // No beads ID
					Status:         "active",
					SpawnedAt:      threeHoursAgo,
					LastActivityAt: oneHourAgo,
					IsStalled:      true,
				},
			},
			role:        "human",
			expectCount: 0,
		},
		{
			name:        "returns empty when no agents",
			agents:      []StuckAgentItem{},
			role:        "daemon",
			expectCount: 0,
		},
		{
			name: "marks very stuck agents (>4h) with higher priority",
			agents: []StuckAgentItem{
				{
					ID:             "session-1",
					BeadsID:        "orch-go-123",
					BeadsTitle:     "Very stuck task",
					Status:         "idle",
					Phase:          "Planning",
					Task:           "Very stuck task",
					SpawnedAt:      fiveHoursAgo,
					LastActivityAt: fiveHoursAgo,
					IsStalled:      true,
				},
			},
			role:        "human",
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/agents" {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				json.NewEncoder(w).Encode(tt.agents)
			}))
			defer server.Close()

			// Create collector with 2h threshold
			collector := NewStuckCollector(server.Client(), server.URL, 2.0)

			// Collect
			items, err := collector.Collect(tt.role)
			if (err != nil) != tt.expectError {
				t.Errorf("Collect() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if len(items) != tt.expectCount {
				t.Errorf("Collect() returned %d items, expected %d", len(items), tt.expectCount)
			}

			// Verify item properties for stuck agent
			if tt.expectCount > 0 && len(items) > 0 {
				item := items[0]
				if item.Signal != "stuck" {
					t.Errorf("item.Signal = %q, expected 'stuck'", item.Signal)
				}
				if item.Source != "agent" {
					t.Errorf("item.Source = %q, expected 'agent'", item.Source)
				}
				if item.Concern != Authority {
					t.Errorf("item.Concern = %v, expected Authority", item.Concern)
				}
				if item.ActionHint == "" {
					t.Error("item.ActionHint should not be empty")
				}
			}
		})
	}
}

func TestStuckCollector_Collect_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	collector := NewStuckCollector(server.Client(), server.URL, 2.0)

	items, err := collector.Collect("human")
	if err == nil {
		t.Error("Collect() should return error for HTTP 500")
	}
	if items != nil {
		t.Error("Collect() should return nil items on error")
	}
}

func TestStuckCollector_Collect_UsesSharedSnapshot(t *testing.T) {
	now := time.Now()
	collector := NewStuckCollectorWithSnapshot([]AgentAPIItem{
		{
			ID:             "session-1",
			BeadsID:        "orch-go-123",
			Status:         "active",
			Task:           "Shared snapshot task",
			SpawnedAt:      now.Add(-3 * time.Hour).Format(time.RFC3339),
			LastActivityAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
			IsStalled:      true,
		},
	}, nil, 2.0)

	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Collect() returned %d items, expected 1", len(items))
	}
	if items[0].Signal != "stuck" {
		t.Errorf("item.Signal = %q, expected %q", items[0].Signal, "stuck")
	}
}

func TestStuckCollector_Collect_SharedSnapshotError(t *testing.T) {
	snapshotErr := errors.New("snapshot unavailable")
	collector := NewStuckCollectorWithSnapshot(nil, snapshotErr, 2.0)

	items, err := collector.Collect("human")
	if !errors.Is(err, snapshotErr) {
		t.Fatalf("Collect() error = %v, expected snapshot error", err)
	}
	if items != nil {
		t.Error("Collect() should return nil items on shared snapshot error")
	}
}

func TestStuckCollector_Collect_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	collector := NewStuckCollector(server.Client(), server.URL, 2.0)

	items, err := collector.Collect("human")
	if err == nil {
		t.Error("Collect() should return error for invalid JSON")
	}
	if items != nil {
		t.Error("Collect() should return nil items on error")
	}
}

func TestCalculateStuckPriority(t *testing.T) {

	tests := []struct {
		name           string
		runningHours   float64
		isStalled      bool
		role           string
		expectPriority int
	}{
		// 2h running: base 20 - 5 = 15
		{"2h human", 2.5, false, "human", 15},
		// 2h running: 15 - 5 (orchestrator) = 10
		{"2h orchestrator", 2.5, false, "orchestrator", 10},
		// 5h running: base 20 - 10 = 10
		{"5h human", 5.0, false, "human", 10},
		// 9h running: base 20 - 15 = 5
		{"9h human", 9.0, false, "human", 5},
		// Stalled: -5 bonus
		{"3h stalled human", 3.0, true, "human", 10}, // 20 - 5 (3h) - 5 (stalled)
		// Daemon: +50
		{"3h daemon", 3.0, false, "daemon", 65}, // 20 - 5 (3h) + 50
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := StuckAgentItem{
				ID:        "test",
				BeadsID:   "test-123",
				Status:    "active",
				IsStalled: tt.isStalled,
			}
			runningDuration := time.Duration(tt.runningHours * float64(time.Hour))

			priority := calculateStuckPriority(agent, runningDuration, tt.role)
			if priority != tt.expectPriority {
				// Calculate what we expect
				t.Errorf("calculateStuckPriority(hours=%v, stalled=%v, role=%q) = %d, expected %d",
					tt.runningHours, tt.isStalled, tt.role, priority, tt.expectPriority)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Minute, "30m"},
		{1 * time.Hour, "1h 0m"},
		{2*time.Hour + 30*time.Minute, "2h 30m"},
		{25 * time.Hour, "1d 1h"},
		{48 * time.Hour, "2d 0h"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, expected %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"ab", 2, "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateStr(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateStr(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestNewStuckCollector_DefaultThreshold(t *testing.T) {
	collector := NewStuckCollector(nil, "", 0) // 0 threshold
	if collector.stuckThresholdH != 2.0 {
		t.Errorf("default threshold = %f, expected 2.0", collector.stuckThresholdH)
	}

	collector2 := NewStuckCollector(nil, "", -1) // negative threshold
	if collector2.stuckThresholdH != 2.0 {
		t.Errorf("default threshold for negative = %f, expected 2.0", collector2.stuckThresholdH)
	}
}
