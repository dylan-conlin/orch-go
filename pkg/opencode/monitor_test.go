package opencode

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/internal/testutil"
)

// TestMonitorDetectsCompletion verifies the monitor detects busy->idle transitions.
func TestMonitorDetectsCompletion(t *testing.T) {
	// Create a mock SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		// Send busy status
		w.Write([]byte("event: session.status\ndata: {\"status\":\"busy\",\"session_id\":\"ses_test123\"}\n\n"))
		flusher.Flush()

		time.Sleep(50 * time.Millisecond)

		// Send idle status (completion)
		w.Write([]byte("event: session.status\ndata: {\"status\":\"idle\",\"session_id\":\"ses_test123\"}\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	// Create monitor
	monitor := NewMonitor(server.URL)

	// Track completions
	var completedSessions []string
	var mu sync.Mutex
	done := make(chan struct{})

	monitor.OnCompletion(func(sessionID string) {
		mu.Lock()
		completedSessions = append(completedSessions, sessionID)
		mu.Unlock()
		close(done)
	})

	// Start monitoring
	monitor.Start()
	defer monitor.Stop()

	// Wait for completion or timeout
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for completion")
	}

	// Verify completion was detected
	mu.Lock()
	defer mu.Unlock()

	if len(completedSessions) != 1 {
		t.Errorf("Expected 1 completion, got %d", len(completedSessions))
	}

	if len(completedSessions) > 0 && completedSessions[0] != "ses_test123" {
		t.Errorf("Expected session ID 'ses_test123', got '%s'", completedSessions[0])
	}
}

// TestMonitorHandleEvent tests the event handling logic directly.
func TestMonitorHandleEvent(t *testing.T) {
	monitor := &Monitor{
		sessions: make(map[string]*SessionState),
		handlers: make([]CompletionHandler, 0),
		done:     make(chan struct{}),
	}

	// Track completions
	var completions []string
	var mu sync.Mutex
	monitor.OnCompletion(func(sessionID string) {
		mu.Lock()
		completions = append(completions, sessionID)
		mu.Unlock()
	})

	// Test: busy status creates session state
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"busy","session_id":"ses_abc"}`,
	})

	state := monitor.GetSessionState("ses_abc")
	if state == nil {
		t.Fatal("Session state not created")
	}
	if state.Status != "busy" {
		t.Errorf("Expected status 'busy', got '%s'", state.Status)
	}
	if !state.WasBusy {
		t.Error("Expected WasBusy to be true")
	}

	// Test: transition to idle triggers completion
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"idle","session_id":"ses_abc"}`,
	})

	// Wait for async handler to execute
	testutil.WaitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(completions) >= 1
	}, "completion to be recorded")

	mu.Lock()
	if len(completions) != 1 {
		t.Errorf("Expected 1 completion, got %d", len(completions))
	}
	if len(completions) > 0 && completions[0] != "ses_abc" {
		t.Errorf("Expected completion for 'ses_abc', got '%s'", completions[0])
	}
	mu.Unlock()
}

// TestMonitorNewFormatSSE tests handling of new SSE format.
func TestMonitorNewFormatSSE(t *testing.T) {
	monitor := &Monitor{
		sessions: make(map[string]*SessionState),
		handlers: make([]CompletionHandler, 0),
		done:     make(chan struct{}),
	}

	var completions []string
	var mu sync.Mutex
	monitor.OnCompletion(func(sessionID string) {
		mu.Lock()
		completions = append(completions, sessionID)
		mu.Unlock()
	})

	// New format: {"type":"session.status","properties":{"sessionID":"...","status":{"type":"..."}}}
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"type":"session.status","properties":{"sessionID":"ses_new","status":{"type":"running"}}}`,
	})

	state := monitor.GetSessionState("ses_new")
	if state == nil {
		t.Fatal("Session state not created for new format")
	}
	if !state.WasBusy {
		t.Error("Expected WasBusy to be true for 'running' status")
	}

	// Transition to idle
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"type":"session.status","properties":{"sessionID":"ses_new","status":{"type":"idle"}}}`,
	})

	// Wait for async handler to execute
	testutil.WaitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(completions) >= 1
	}, "completion for new format to be recorded")

	mu.Lock()
	if len(completions) != 1 {
		t.Errorf("Expected 1 completion for new format, got %d", len(completions))
	}
	mu.Unlock()
}

// TestMonitorIgnoresNonStatusEvents tests that non-status events are ignored.
func TestMonitorIgnoresNonStatusEvents(t *testing.T) {
	monitor := &Monitor{
		sessions: make(map[string]*SessionState),
		handlers: make([]CompletionHandler, 0),
		done:     make(chan struct{}),
	}

	// Send a message.updated event (should be ignored)
	monitor.handleEvent(SSEEvent{
		Event: "message.updated",
		Data:  `{"content":"working..."}`,
	})

	// Should not create any session state
	if len(monitor.sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(monitor.sessions))
	}
}

// TestMonitorMultipleSessions tests handling of multiple concurrent sessions.
func TestMonitorMultipleSessions(t *testing.T) {
	monitor := &Monitor{
		sessions: make(map[string]*SessionState),
		handlers: make([]CompletionHandler, 0),
		done:     make(chan struct{}),
	}

	var completions []string
	var mu sync.Mutex
	monitor.OnCompletion(func(sessionID string) {
		mu.Lock()
		completions = append(completions, sessionID)
		mu.Unlock()
	})

	// Session 1: busy
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"busy","session_id":"ses_1"}`,
	})

	// Session 2: busy
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"busy","session_id":"ses_2"}`,
	})

	// Session 1: idle (completes)
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"idle","session_id":"ses_1"}`,
	})

	// Wait for first completion
	testutil.WaitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(completions) >= 1
	}, "first completion")

	// Session 2 should still be busy
	state2 := monitor.GetSessionState("ses_2")
	if state2 == nil || state2.Status != "busy" {
		t.Error("Session 2 should still be busy")
	}

	// Session 2: idle (completes)
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"idle","session_id":"ses_2"}`,
	})

	// Wait for second completion
	testutil.WaitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(completions) >= 2
	}, "second completion")

	mu.Lock()
	if len(completions) != 2 {
		t.Errorf("Expected 2 completions, got %d", len(completions))
	}
	mu.Unlock()
}

// TestMonitorActiveSessions tests listing active sessions.
func TestMonitorActiveSessions(t *testing.T) {
	monitor := &Monitor{
		sessions: make(map[string]*SessionState),
		handlers: make([]CompletionHandler, 0),
		done:     make(chan struct{}),
	}

	// Add some sessions
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"busy","session_id":"ses_a"}`,
	})
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"idle","session_id":"ses_b"}`,
	})

	sessions := monitor.ActiveSessions()
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

// TestMonitorNoCompletionForDirectIdle tests that direct idle (no busy) doesn't trigger completion.
func TestMonitorNoCompletionForDirectIdle(t *testing.T) {
	monitor := &Monitor{
		sessions: make(map[string]*SessionState),
		handlers: make([]CompletionHandler, 0),
		done:     make(chan struct{}),
	}

	var completions []string
	var mu sync.Mutex
	monitor.OnCompletion(func(sessionID string) {
		mu.Lock()
		completions = append(completions, sessionID)
		mu.Unlock()
	})

	// Send idle directly without busy first
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"idle","session_id":"ses_idle_only"}`,
	})

	// For negative tests, we can't use WaitFor since condition should never be true.
	// A short timeout with Eventually returning false is the correct approach.
	if testutil.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(completions) > 0
	}, 100*time.Millisecond) {
		t.Errorf("Expected 0 completions for direct idle, got some")
	}

	mu.Lock()
	if len(completions) != 0 {
		t.Errorf("Expected 0 completions for direct idle, got %d", len(completions))
	}
	mu.Unlock()
}

// TestMonitorSessionCleanupAfterCompletion tests that sessions are removed from the map after completion.
// This prevents memory leaks in long-running daemons.
func TestMonitorSessionCleanupAfterCompletion(t *testing.T) {
	monitor := &Monitor{
		sessions: make(map[string]*SessionState),
		handlers: make([]CompletionHandler, 0),
		done:     make(chan struct{}),
	}

	var completions []string
	var mu sync.Mutex
	monitor.OnCompletion(func(sessionID string) {
		mu.Lock()
		completions = append(completions, sessionID)
		mu.Unlock()
	})

	// Session goes busy then idle (completes)
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"busy","session_id":"ses_cleanup"}`,
	})

	// Verify session is in the map
	if len(monitor.sessions) != 1 {
		t.Fatalf("Expected 1 session in map, got %d", len(monitor.sessions))
	}

	// Transition to idle (triggers completion)
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"idle","session_id":"ses_cleanup"}`,
	})

	// Wait for async handler and session cleanup
	testutil.WaitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(completions) >= 1
	}, "completion to be recorded")

	// Verify completion was triggered
	mu.Lock()
	if len(completions) != 1 {
		t.Errorf("Expected 1 completion, got %d", len(completions))
	}
	mu.Unlock()

	// CRITICAL: Verify session was removed from the map to prevent memory leak
	// Session cleanup happens synchronously in handleEvent, so no additional wait needed
	if len(monitor.sessions) != 0 {
		t.Errorf("Expected 0 sessions in map after completion (memory leak!), got %d", len(monitor.sessions))
	}
}

// TestMonitorSessionCleanupMultipleSessions tests cleanup with multiple sessions.
func TestMonitorSessionCleanupMultipleSessions(t *testing.T) {
	monitor := &Monitor{
		sessions: make(map[string]*SessionState),
		handlers: make([]CompletionHandler, 0),
		done:     make(chan struct{}),
	}

	// Create 3 sessions
	for i := 1; i <= 3; i++ {
		monitor.handleEvent(SSEEvent{
			Event: "session.status",
			Data:  `{"status":"busy","session_id":"ses_` + string(rune('a'+i-1)) + `"}`,
		})
	}

	if len(monitor.sessions) != 3 {
		t.Fatalf("Expected 3 sessions, got %d", len(monitor.sessions))
	}

	// Complete first session
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"idle","session_id":"ses_a"}`,
	})

	// Session cleanup happens synchronously in handleEvent, verify immediately
	if len(monitor.sessions) != 2 {
		t.Errorf("Expected 2 sessions after first completion, got %d", len(monitor.sessions))
	}

	// Complete second session
	monitor.handleEvent(SSEEvent{
		Event: "session.status",
		Data:  `{"status":"idle","session_id":"ses_b"}`,
	})

	// Session cleanup happens synchronously
	if len(monitor.sessions) != 1 {
		t.Errorf("Expected 1 session after second completion, got %d", len(monitor.sessions))
	}

	// Third session still busy - should remain in map
	state := monitor.GetSessionState("ses_c")
	if state == nil || state.Status != "busy" {
		t.Error("Session ses_c should still be busy")
	}
}
