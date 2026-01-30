package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// mockOpenCodeServer creates a test HTTP server that simulates OpenCode API responses.
// It supports configurable number of sessions with mix of tracked/untracked.
type mockOpenCodeServer struct {
	server         *httptest.Server
	sessions       []opencode.Session
	messagesByID   map[string][]opencode.Message
	untrackedRatio float64       // ratio of untracked sessions (0.0 to 1.0)
	apiLatency     time.Duration // simulated latency per API call
	requestCount   int           // tracks total API requests made
}

// newMockOpenCodeServer creates a mock OpenCode server with N sessions.
// untrackedRatio controls what fraction of sessions are untracked (no beads ID).
// apiLatency simulates network latency per API call (use 0 for instant responses).
func newMockOpenCodeServer(sessionCount int, untrackedRatio float64, apiLatency time.Duration) *mockOpenCodeServer {
	m := &mockOpenCodeServer{
		messagesByID:   make(map[string][]opencode.Message),
		untrackedRatio: untrackedRatio,
		apiLatency:     apiLatency,
		requestCount:   0,
	}

	// Generate mock sessions
	now := time.Now()
	baseTime := now.Add(-2 * time.Hour) // All sessions started 2 hours ago

	untrackedCount := int(float64(sessionCount) * untrackedRatio)
	trackedCount := sessionCount - untrackedCount

	// Create tracked sessions (with beads ID in title)
	for i := 0; i < trackedCount; i++ {
		sessionID := fmt.Sprintf("ses_%d", i)
		beadsID := fmt.Sprintf("orch-go-%04x", i)
		skill := "feature-impl"
		if i%3 == 0 {
			skill = "investigation"
		} else if i%5 == 0 {
			skill = "systematic-debugging"
		}

		title := fmt.Sprintf("og-feat-%s-30jan-%s", skill, beadsID)

		// Vary activity times - some idle, some active
		updatedTime := baseTime.Add(time.Duration(i%60) * time.Minute)

		session := opencode.Session{
			ID:        sessionID,
			Directory: "/Users/test/orch-go",
			Title:     title,
			Time: opencode.SessionTime{
				Created: baseTime.Unix() * 1000,
				Updated: updatedTime.Unix() * 1000,
			},
		}
		m.sessions = append(m.sessions, session)

		// Create minimal messages for GetSessionModel calls
		m.messagesByID[sessionID] = []opencode.Message{
			{
				Info: opencode.MessageInfo{
					ID:      fmt.Sprintf("msg_%d", i),
					Role:    "assistant",
					ModelID: "anthropic/claude-sonnet-4-5-20250929",
				},
			},
		}
	}

	// Create untracked sessions (no beads ID)
	for i := 0; i < untrackedCount; i++ {
		sessionID := fmt.Sprintf("ses_untracked_%d", i)
		title := fmt.Sprintf("General discussion %d", i)

		// Untracked sessions use longer idle threshold (2h) so keep them recent
		updatedTime := now.Add(-time.Duration(i%90) * time.Minute)

		session := opencode.Session{
			ID:        sessionID,
			Directory: "/Users/test/orch-go",
			Title:     title,
			Time: opencode.SessionTime{
				Created: baseTime.Unix() * 1000,
				Updated: updatedTime.Unix() * 1000,
			},
		}
		m.sessions = append(m.sessions, session)

		// Create minimal messages
		m.messagesByID[sessionID] = []opencode.Message{
			{
				Info: opencode.MessageInfo{
					ID:      fmt.Sprintf("msg_untracked_%d", i),
					Role:    "assistant",
					ModelID: "anthropic/claude-sonnet-4-5-20250929",
				},
			},
		}
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// GET /session - list all sessions
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Simulate API latency
		if m.apiLatency > 0 {
			time.Sleep(m.apiLatency)
		}
		m.requestCount++

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m.sessions)
	})

	// GET /session/{id}/message - get messages for a session
	mux.HandleFunc("/session/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/session/")
		parts := strings.Split(path, "/")

		if len(parts) < 2 || parts[1] != "message" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// Simulate API latency
		if m.apiLatency > 0 {
			time.Sleep(m.apiLatency)
		}
		m.requestCount++

		sessionID := parts[0]
		messages, ok := m.messagesByID[sessionID]
		if !ok {
			messages = []opencode.Message{} // Return empty for unknown sessions
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	})

	m.server = httptest.NewServer(mux)
	return m
}

// Close shuts down the mock server.
func (m *mockOpenCodeServer) Close() {
	m.server.Close()
}

// URL returns the server URL for use as serverURL parameter.
func (m *mockOpenCodeServer) URL() string {
	return m.server.URL
}

// BenchmarkStatus_100Sessions benchmarks status command with 100 sessions (zero latency).
// 80% tracked (with beads ID), 20% untracked.
func BenchmarkStatus_100Sessions(b *testing.B) {
	benchmarkStatus(b, 100, 0.2, 0)
}

// BenchmarkStatus_500Sessions benchmarks status command with 500 sessions (zero latency).
// 80% tracked (with beads ID), 20% untracked.
func BenchmarkStatus_500Sessions(b *testing.B) {
	benchmarkStatus(b, 500, 0.2, 0)
}

// BenchmarkStatus_1000Sessions benchmarks status command with 1000 sessions (zero latency).
// 80% tracked (with beads ID), 20% untracked.
func BenchmarkStatus_1000Sessions(b *testing.B) {
	benchmarkStatus(b, 1000, 0.2, 0)
}

// BenchmarkStatus_100Sessions_AllUntracked benchmarks with only untracked sessions (zero latency).
// This tests the performance of Phase 3 discovery (untracked session detection from orch-go-20988).
func BenchmarkStatus_100Sessions_AllUntracked(b *testing.B) {
	benchmarkStatus(b, 100, 1.0, 0)
}

// BenchmarkStatus_500Sessions_AllUntracked benchmarks with only untracked sessions (zero latency).
// This tests the performance of Phase 3 discovery with larger counts.
func BenchmarkStatus_500Sessions_AllUntracked(b *testing.B) {
	benchmarkStatus(b, 500, 1.0, 0)
}

// BenchmarkStatus_100Sessions_WithLatency benchmarks with 1ms API latency per call.
// Simulates realistic network conditions with local OpenCode server.
func BenchmarkStatus_100Sessions_WithLatency(b *testing.B) {
	benchmarkStatus(b, 100, 0.2, 1*time.Millisecond)
}

// BenchmarkStatus_500Sessions_WithLatency benchmarks with 1ms API latency per call.
// Tests performance degradation with larger session counts under realistic conditions.
func BenchmarkStatus_500Sessions_WithLatency(b *testing.B) {
	benchmarkStatus(b, 500, 0.2, 1*time.Millisecond)
}

// BenchmarkStatus_1000Sessions_WithLatency benchmarks with 1ms API latency per call.
// Tests worst-case performance with large session counts and network latency.
func BenchmarkStatus_1000Sessions_WithLatency(b *testing.B) {
	benchmarkStatus(b, 1000, 0.2, 1*time.Millisecond)
}

// benchmarkStatus is the common benchmark implementation.
func benchmarkStatus(b *testing.B, sessionCount int, untrackedRatio float64, apiLatency time.Duration) {
	// Create mock server
	mock := newMockOpenCodeServer(sessionCount, untrackedRatio, apiLatency)
	defer mock.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Suppress stdout during benchmark to avoid polluting benchmark output
		oldStdout := os.Stdout
		devNull, _ := os.Open("/dev/null")
		os.Stdout = devNull

		// Run status command against mock server
		// Errors are expected since we don't have full beads/tmux infrastructure
		_ = runStatus(mock.URL())

		// Restore stdout
		os.Stdout = oldStdout
		devNull.Close()
	}

	// Report API request count for analysis
	b.ReportMetric(float64(mock.requestCount)/float64(b.N), "api-calls/op")
}

// TestMockServerBasicFunctionality verifies the mock server works correctly.
func TestMockServerBasicFunctionality(t *testing.T) {
	mock := newMockOpenCodeServer(10, 0.3, 0)
	defer mock.Close()

	// Test GET /session
	client := opencode.NewClient(mock.URL())
	sessions, err := client.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 10 {
		t.Errorf("Expected 10 sessions, got %d", len(sessions))
	}

	// Count tracked vs untracked
	tracked := 0
	untracked := 0
	for _, s := range sessions {
		if strings.Contains(s.Title, "orch-go-") {
			tracked++
		} else {
			untracked++
		}
	}

	// With 10 sessions and 0.3 untracked ratio, expect 3 untracked, 7 tracked
	expectedUntracked := 3
	expectedTracked := 7
	if untracked != expectedUntracked {
		t.Errorf("Expected %d untracked sessions, got %d", expectedUntracked, untracked)
	}
	if tracked != expectedTracked {
		t.Errorf("Expected %d tracked sessions, got %d", expectedTracked, tracked)
	}

	// Test GET /session/{id}/message
	if len(sessions) > 0 {
		sessionID := sessions[0].ID
		messages, err := client.GetMessages(sessionID)
		if err != nil {
			t.Fatalf("GetMessages failed: %v", err)
		}
		if len(messages) == 0 {
			t.Error("Expected at least one message")
		}
	}
}
