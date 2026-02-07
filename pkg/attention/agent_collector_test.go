package attention

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentCollector_Collect(t *testing.T) {
	tests := []struct {
		name        string
		agents      []AgentAPIItem
		role        string
		expectCount int
		expectError bool
	}{
		{
			name: "collects awaiting-cleanup agents",
			agents: []AgentAPIItem{
				{
					ID:         "session-1",
					BeadsID:    "orch-go-123",
					BeadsTitle: "Test feature",
					Status:     "awaiting-cleanup",
					Phase:      "Complete",
					Task:       "Implement test feature",
					Project:    "orch-go",
					Skill:      "feature-impl",
					UpdatedAt:  "2026-02-03T10:00:00Z",
				},
				{
					ID:         "session-2",
					BeadsID:    "orch-go-456",
					BeadsTitle: "Another task",
					Status:     "awaiting-cleanup",
					Phase:      "Complete",
					Task:       "Fix bug",
					Project:    "orch-go",
					Skill:      "systematic-debugging",
					UpdatedAt:  "2026-02-03T11:00:00Z",
				},
			},
			role:        "human",
			expectCount: 2,
		},
		{
			name: "filters out non-awaiting-cleanup agents",
			agents: []AgentAPIItem{
				{
					ID:      "session-1",
					BeadsID: "orch-go-123",
					Status:  "awaiting-cleanup",
					Phase:   "Complete",
					Task:    "Done task",
					Project: "orch-go",
				},
				{
					ID:      "session-2",
					BeadsID: "orch-go-456",
					Status:  "active",
					Phase:   "Implementing",
					Task:    "Active task",
					Project: "orch-go",
				},
				{
					ID:      "session-3",
					BeadsID: "orch-go-789",
					Status:  "dead",
					Phase:   "Planning",
					Task:    "Dead task",
					Project: "orch-go",
				},
			},
			role:        "orchestrator",
			expectCount: 1, // Only awaiting-cleanup
		},
		{
			name: "skips agents without beads ID",
			agents: []AgentAPIItem{
				{
					ID:      "session-1",
					BeadsID: "", // No beads ID
					Status:  "awaiting-cleanup",
					Phase:   "Complete",
					Task:    "Orphan task",
				},
				{
					ID:      "session-2",
					BeadsID: "orch-go-123",
					Status:  "awaiting-cleanup",
					Phase:   "Complete",
					Task:    "Valid task",
					Project: "orch-go",
				},
			},
			role:        "human",
			expectCount: 1, // Only the one with beads ID
		},
		{
			name:        "returns empty when no awaiting-cleanup agents",
			agents:      []AgentAPIItem{},
			role:        "daemon",
			expectCount: 0,
		},
		{
			name: "uses beads_title when task is empty",
			agents: []AgentAPIItem{
				{
					ID:         "session-1",
					BeadsID:    "orch-go-123",
					BeadsTitle: "Title from beads",
					Status:     "awaiting-cleanup",
					Phase:      "Complete",
					Task:       "", // Empty task
					Project:    "orch-go",
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
				resp := AgentAPIListResponse{Agents: tt.agents}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			// Create collector
			collector := NewAgentCollector(server.Client(), server.URL)

			// Collect
			items, err := collector.Collect(tt.role)
			if (err != nil) != tt.expectError {
				t.Errorf("Collect() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if len(items) != tt.expectCount {
				t.Errorf("Collect() returned %d items, expected %d", len(items), tt.expectCount)
			}

			// Verify item properties for first awaiting-cleanup agent
			if tt.expectCount > 0 && len(items) > 0 {
				item := items[0]
				if item.Signal != "verify" {
					t.Errorf("item.Signal = %q, expected 'verify'", item.Signal)
				}
				if item.Source != "agent" {
					t.Errorf("item.Source = %q, expected 'agent'", item.Source)
				}
				if item.Concern != Actionability {
					t.Errorf("item.Concern = %v, expected Actionability", item.Concern)
				}
				if item.ActionHint == "" {
					t.Error("item.ActionHint should not be empty")
				}
			}
		})
	}
}

func TestAgentCollector_Collect_HTTPError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	collector := NewAgentCollector(server.Client(), server.URL)

	items, err := collector.Collect("human")
	if err == nil {
		t.Error("Collect() should return error for HTTP 500")
	}
	if items != nil {
		t.Error("Collect() should return nil items on error")
	}
}

func TestAgentCollector_Collect_UsesSharedSnapshot(t *testing.T) {
	collector := NewAgentCollectorWithSnapshot([]AgentAPIItem{
		{
			ID:         "session-1",
			BeadsID:    "orch-go-123",
			BeadsTitle: "Snapshot task",
			Status:     "awaiting-cleanup",
			Phase:      "Complete",
			Task:       "Complete from snapshot",
			Project:    "orch-go",
			Skill:      "feature-impl",
			UpdatedAt:  "2026-02-03T10:00:00Z",
		},
	}, nil)

	items, err := collector.Collect("human")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Collect() returned %d items, expected 1", len(items))
	}
	if items[0].Subject != "orch-go-123" {
		t.Errorf("item.Subject = %q, expected %q", items[0].Subject, "orch-go-123")
	}
}

func TestAgentCollector_Collect_SharedSnapshotError(t *testing.T) {
	snapshotErr := errors.New("snapshot unavailable")
	collector := NewAgentCollectorWithSnapshot(nil, snapshotErr)

	items, err := collector.Collect("human")
	if !errors.Is(err, snapshotErr) {
		t.Fatalf("Collect() error = %v, expected snapshot error", err)
	}
	if items != nil {
		t.Error("Collect() should return nil items on shared snapshot error")
	}
}

func TestAgentCollector_Collect_InvalidJSON(t *testing.T) {
	// Create test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	collector := NewAgentCollector(server.Client(), server.URL)

	items, err := collector.Collect("human")
	if err == nil {
		t.Error("Collect() should return error for invalid JSON")
	}
	if items != nil {
		t.Error("Collect() should return nil items on error")
	}
}

func TestCalculateAgentPriority(t *testing.T) {
	agent := AgentAPIItem{
		ID:      "test",
		BeadsID: "test-123",
		Status:  "awaiting-cleanup",
	}

	tests := []struct {
		role           string
		expectPriority int
	}{
		{"human", 50},
		{"orchestrator", 40}, // Lower priority number = higher priority
		{"daemon", 70},
		{"unknown", 50}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			priority := calculateAgentPriority(agent, tt.role)
			if priority != tt.expectPriority {
				t.Errorf("calculateAgentPriority(%q) = %d, expected %d", tt.role, priority, tt.expectPriority)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"}, // string <= maxLen, no truncation
		{"ab", 2, "ab"},   // string <= maxLen, no truncation
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}
