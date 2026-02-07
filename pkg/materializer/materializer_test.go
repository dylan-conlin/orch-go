package materializer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/state"
)

// testDB creates a temporary state database for testing.
func testDB(t *testing.T) *state.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-state.db")
	db, err := state.Open(path)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// testAgent creates a minimal agent for testing.
func testAgent(name, sessionID string) *state.Agent {
	return &state.Agent{
		WorkspaceName: name,
		BeadsID:       "orch-go-" + name,
		SessionID:     sessionID,
		Mode:          "opencode",
		ProjectDir:    "/Users/test/orch-go",
		SpawnTime:     time.Now().UnixMilli(),
	}
}

// sseServer creates a test SSE server that streams the given events.
func sseServer(t *testing.T, events string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/event" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", 500)
			return
		}
		// Write events
		io.WriteString(w, events)
		flusher.Flush()
	}))
}

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state ConnectionState
		want  string
	}{
		{StateDisconnected, "disconnected"},
		{StateConnecting, "connecting"},
		{StateConnected, "connected"},
		{ConnectionState(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ConnectionState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestNew_DefaultConfig(t *testing.T) {
	m := New(Config{ServerURL: "http://localhost:4096"})
	if m.config.InitialReconnectDelay != 2*time.Second {
		t.Errorf("InitialReconnectDelay = %v, want 2s", m.config.InitialReconnectDelay)
	}
	if m.config.MaxReconnectDelay != 60*time.Second {
		t.Errorf("MaxReconnectDelay = %v, want 60s", m.config.MaxReconnectDelay)
	}
}

func TestNew_CustomConfig(t *testing.T) {
	m := New(Config{
		ServerURL:             "http://localhost:4096",
		InitialReconnectDelay: 5 * time.Second,
		MaxReconnectDelay:     30 * time.Second,
	})
	if m.config.InitialReconnectDelay != 5*time.Second {
		t.Errorf("InitialReconnectDelay = %v, want 5s", m.config.InitialReconnectDelay)
	}
	if m.config.MaxReconnectDelay != 30*time.Second {
		t.Errorf("MaxReconnectDelay = %v, want 30s", m.config.MaxReconnectDelay)
	}
}

func TestHandleSessionStatus_Busy(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-mat-busy", "session-busy-1")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	m := &Materializer{db: db}

	// Simulate busy event
	data := `{"type":"session.status","properties":{"sessionID":"session-busy-1","status":{"type":"busy"}}}`
	m.handleSessionStatus(data)

	got, err := db.GetAgent("og-feat-mat-busy")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if !got.IsProcessing {
		t.Error("IsProcessing should be true after busy event")
	}
	if got.SessionUpdatedAt == 0 {
		t.Error("SessionUpdatedAt should be set after busy event")
	}
}

func TestHandleSessionStatus_Idle(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-mat-idle", "session-idle-1")
	agent.IsProcessing = true
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	m := &Materializer{db: db}

	// Simulate idle event
	data := `{"type":"session.status","properties":{"sessionID":"session-idle-1","status":{"type":"idle"}}}`
	m.handleSessionStatus(data)

	got, err := db.GetAgent("og-feat-mat-idle")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if got.IsProcessing {
		t.Error("IsProcessing should be false after idle event")
	}
}

func TestHandleSessionStatus_OldFormat(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-mat-old", "session-old-1")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	m := &Materializer{db: db}

	// Simulate old format busy event
	data := `{"status":"busy","session_id":"session-old-1"}`
	m.handleSessionStatus(data)

	got, err := db.GetAgent("og-feat-mat-old")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if !got.IsProcessing {
		t.Error("IsProcessing should be true after old-format busy event")
	}
}

func TestHandleSessionStatus_UnknownSession(t *testing.T) {
	db := testDB(t)
	m := &Materializer{db: db}

	// Should not error even if session is not in state.db
	data := `{"type":"session.status","properties":{"sessionID":"unknown-session","status":{"type":"busy"}}}`
	m.handleSessionStatus(data)
	// No crash = success
}

func TestHandleSessionStatus_EmptySessionID(t *testing.T) {
	db := testDB(t)
	m := &Materializer{db: db}

	// Empty session ID should be silently ignored
	data := `{"type":"session.status","properties":{"status":{"type":"busy"}}}`
	m.handleSessionStatus(data)
	// No crash = success
}

func TestHandleMessagePart_UpdatesActivity(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-mat-msg", "session-msg-1")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	m := &Materializer{db: db}

	// Simulate message.part event
	data := `{"sessionID":"session-msg-1","messageID":"msg-1","type":"text"}`
	m.handleMessagePart(data)

	got, err := db.GetAgent("og-feat-mat-msg")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if got.SessionUpdatedAt == 0 {
		t.Error("SessionUpdatedAt should be set after message.part event")
	}
}

func TestHandleMessagePart_NestedProperties(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-mat-nested", "session-nested-1")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	m := &Materializer{db: db}

	// Simulate nested format message.part event
	data := `{"type":"message.part.updated","properties":{"sessionID":"session-nested-1","partID":"part-1"}}`
	m.handleMessagePart(data)

	got, err := db.GetAgent("og-feat-mat-nested")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if got.SessionUpdatedAt == 0 {
		t.Error("SessionUpdatedAt should be set after nested message.part event")
	}
}

func TestHandleEvent_SessionStatus(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-mat-evt", "session-evt-1")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	m := &Materializer{db: db}

	event := opencode.SSEEvent{
		Event: "session.status",
		Data:  `{"type":"session.status","properties":{"sessionID":"session-evt-1","status":{"type":"busy"}}}`,
	}
	m.handleEvent(event)

	got, err := db.GetAgent("og-feat-mat-evt")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if !got.IsProcessing {
		t.Error("IsProcessing should be true after handleEvent with session.status")
	}

	// Check counter incremented
	status := m.Status()
	if status.EventsProcessed != 1 {
		t.Errorf("EventsProcessed = %d, want 1", status.EventsProcessed)
	}
}

func TestHandleEvent_MessagePart(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-mat-msgp", "session-msgp-1")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	m := &Materializer{db: db}

	event := opencode.SSEEvent{
		Event: "message.part.updated",
		Data:  `{"sessionID":"session-msgp-1","messageID":"msg-1","type":"text"}`,
	}
	m.handleEvent(event)

	got, err := db.GetAgent("og-feat-mat-msgp")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if got.SessionUpdatedAt == 0 {
		t.Error("SessionUpdatedAt should be set after handleEvent with message.part.updated")
	}
}

func TestHandleEvent_IgnoresUnknownEvents(t *testing.T) {
	db := testDB(t)
	m := &Materializer{db: db}

	event := opencode.SSEEvent{
		Event: "some.unknown.event",
		Data:  `{}`,
	}
	m.handleEvent(event)

	// Counter should still increment
	status := m.Status()
	if status.EventsProcessed != 1 {
		t.Errorf("EventsProcessed = %d, want 1", status.EventsProcessed)
	}
}

func TestStatus_Initial(t *testing.T) {
	m := New(Config{ServerURL: "http://localhost:4096"})
	status := m.Status()

	if status.State != StateDisconnected {
		t.Errorf("initial state = %v, want disconnected", status.State)
	}
	if status.StateStr != "disconnected" {
		t.Errorf("initial state_str = %q, want 'disconnected'", status.StateStr)
	}
	if status.EventsProcessed != 0 {
		t.Errorf("initial events_processed = %d, want 0", status.EventsProcessed)
	}
	if status.ReconnectAttempts != 0 {
		t.Errorf("initial reconnect_attempts = %d, want 0", status.ReconnectAttempts)
	}
}

func TestSetState(t *testing.T) {
	m := New(Config{ServerURL: "http://localhost:4096"})
	m.setState(StateConnected)

	status := m.Status()
	if status.State != StateConnected {
		t.Errorf("state = %v, want connected", status.State)
	}
}

func TestStartAndStop(t *testing.T) {
	// Create an SSE server that sends one event then closes
	sseEvents := "event: session.status\ndata: {\"type\":\"session.status\",\"properties\":{\"sessionID\":\"test-123\",\"status\":{\"type\":\"busy\"}}}\n\n"
	server := sseServer(t, sseEvents)
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "test-state.db")

	// Pre-populate the database with an agent
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open state db: %v", err)
	}
	agent := testAgent("og-feat-mat-start", "test-123")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}
	db.Close()

	m := New(Config{
		ServerURL:             server.URL,
		DBPath:                dbPath,
		InitialReconnectDelay: 100 * time.Millisecond,
		MaxReconnectDelay:     500 * time.Millisecond,
	})

	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for events to be processed
	time.Sleep(500 * time.Millisecond)

	m.Stop()

	// Verify the event was processed
	status := m.Status()
	if status.EventsProcessed == 0 {
		t.Error("Expected at least one event to be processed")
	}

	// Verify state was materialized
	db2, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen state db: %v", err)
	}
	defer db2.Close()

	got, err := db2.GetAgent("og-feat-mat-start")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if !got.IsProcessing {
		t.Error("IsProcessing should be true after SSE busy event")
	}
}

func TestStartAndStop_GracefulShutdown(t *testing.T) {
	// Server that sends events then closes (simulating server shutdown)
	sseEvents := "event: session.status\ndata: {\"type\":\"session.status\",\"properties\":{\"sessionID\":\"test-shutdown\",\"status\":{\"type\":\"busy\"}}}\n\n"
	server := sseServer(t, sseEvents)

	dbPath := filepath.Join(t.TempDir(), "test-state.db")

	m := New(Config{
		ServerURL:             server.URL,
		DBPath:                dbPath,
		InitialReconnectDelay: 100 * time.Millisecond,
	})

	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give it time to process events
	time.Sleep(200 * time.Millisecond)

	// Close the server first so SSE connection drops
	server.Close()

	// Stop should complete within a reasonable time
	done := make(chan struct{})
	go func() {
		m.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK - shutdown completed
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds")
	}

	status := m.Status()
	if status.State != StateDisconnected {
		t.Errorf("state after stop = %v, want disconnected", status.State)
	}
}

func TestReconnect_IncreasesAttemptCount(t *testing.T) {
	// Server that immediately closes connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/event" {
			http.NotFound(w, r)
			return
		}
		// Return 200 but immediately close
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
	}))
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "test-state.db")

	m := New(Config{
		ServerURL:             server.URL,
		DBPath:                dbPath,
		InitialReconnectDelay: 50 * time.Millisecond,
		MaxReconnectDelay:     200 * time.Millisecond,
	})

	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Let it attempt a few reconnections
	time.Sleep(400 * time.Millisecond)

	status := m.Status()
	m.Stop()

	if status.ReconnectAttempts == 0 {
		t.Error("Expected reconnect attempts > 0 after server disconnect")
	}
}

func TestHandleEvent_MultipleSessionsBusyIdle(t *testing.T) {
	db := testDB(t)

	// Create two agents with different sessions
	agent1 := testAgent("og-feat-mat-multi1", "session-multi-1")
	agent2 := testAgent("og-feat-mat-multi2", "session-multi-2")
	if err := db.InsertAgent(agent1); err != nil {
		t.Fatalf("InsertAgent 1 failed: %v", err)
	}
	if err := db.InsertAgent(agent2); err != nil {
		t.Fatalf("InsertAgent 2 failed: %v", err)
	}

	m := &Materializer{db: db}

	// Session 1 goes busy
	m.handleEvent(opencode.SSEEvent{
		Event: "session.status",
		Data:  `{"type":"session.status","properties":{"sessionID":"session-multi-1","status":{"type":"busy"}}}`,
	})

	// Session 2 goes busy
	m.handleEvent(opencode.SSEEvent{
		Event: "session.status",
		Data:  `{"type":"session.status","properties":{"sessionID":"session-multi-2","status":{"type":"busy"}}}`,
	})

	// Verify both are processing
	got1, _ := db.GetAgent("og-feat-mat-multi1")
	got2, _ := db.GetAgent("og-feat-mat-multi2")
	if !got1.IsProcessing {
		t.Error("Agent 1 should be processing")
	}
	if !got2.IsProcessing {
		t.Error("Agent 2 should be processing")
	}

	// Session 1 goes idle
	m.handleEvent(opencode.SSEEvent{
		Event: "session.status",
		Data:  `{"type":"session.status","properties":{"sessionID":"session-multi-1","status":{"type":"idle"}}}`,
	})

	// Only session 1 should be idle
	got1, _ = db.GetAgent("og-feat-mat-multi1")
	got2, _ = db.GetAgent("og-feat-mat-multi2")
	if got1.IsProcessing {
		t.Error("Agent 1 should NOT be processing after idle event")
	}
	if !got2.IsProcessing {
		t.Error("Agent 2 should still be processing")
	}
}

func TestHandleEvent_DoesNotUpdateCompletedAgent(t *testing.T) {
	db := testDB(t)
	agent := testAgent("og-feat-mat-completed", "session-completed-1")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}
	// Mark as completed
	if err := db.UpdateCompleted("og-feat-mat-completed"); err != nil {
		t.Fatalf("UpdateCompleted failed: %v", err)
	}

	m := &Materializer{db: db}

	// Try to set processing on completed agent
	m.handleEvent(opencode.SSEEvent{
		Event: "session.status",
		Data:  `{"type":"session.status","properties":{"sessionID":"session-completed-1","status":{"type":"busy"}}}`,
	})

	// Should NOT have updated is_processing (SQL WHERE clause filters completed agents)
	got, _ := db.GetAgent("og-feat-mat-completed")
	if got.IsProcessing {
		t.Error("Completed agent should NOT have is_processing updated")
	}
}

// TestParseSSEEventIntegration verifies the materializer handles real SSE format correctly.
func TestParseSSEEventIntegration(t *testing.T) {
	// This tests the full SSE parsing + handling pipeline using ReadSSEStream
	db := testDB(t)
	agent := testAgent("og-feat-mat-parse", "session-parse-1")
	if err := db.InsertAgent(agent); err != nil {
		t.Fatalf("InsertAgent failed: %v", err)
	}

	m := &Materializer{db: db}

	// Simulate SSE stream with multiple events (each event ends with \n\n)
	sseData := "" +
		"event: session.status\ndata: {\"type\":\"session.status\",\"properties\":{\"sessionID\":\"session-parse-1\",\"status\":{\"type\":\"busy\"}}}\n\n" +
		"event: message.part.updated\ndata: {\"sessionID\":\"session-parse-1\",\"messageID\":\"msg-1\",\"type\":\"text\"}\n\n" +
		"event: session.status\ndata: {\"type\":\"session.status\",\"properties\":{\"sessionID\":\"session-parse-1\",\"status\":{\"type\":\"idle\"}}}\n\n"

	events := make(chan opencode.SSEEvent, 10)
	go func() {
		opencode.ReadSSEStream(strings.NewReader(sseData), events)
		close(events)
	}()

	for event := range events {
		m.handleEvent(event)
	}

	// After busy -> message -> idle, agent should be not processing
	got, _ := db.GetAgent("og-feat-mat-parse")
	if got.IsProcessing {
		t.Error("Agent should NOT be processing after idle event")
	}
	if got.SessionUpdatedAt == 0 {
		t.Error("SessionUpdatedAt should be set")
	}

	// Should have processed 3 events
	status := m.Status()
	if status.EventsProcessed != 3 {
		t.Errorf("EventsProcessed = %d, want 3", status.EventsProcessed)
	}
}
