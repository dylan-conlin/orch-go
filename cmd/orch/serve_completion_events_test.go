package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCompletionBroadcaster_SubscribeUnsubscribe(t *testing.T) {
	b := &completionBroadcaster{
		clients: make(map[chan CompletionEvent]struct{}),
	}

	ch := b.subscribe()
	b.mu.RLock()
	if len(b.clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(b.clients))
	}
	b.mu.RUnlock()

	b.unsubscribe(ch)
	b.mu.RLock()
	if len(b.clients) != 0 {
		t.Fatalf("expected 0 clients, got %d", len(b.clients))
	}
	b.mu.RUnlock()
}

func TestCompletionBroadcaster_Broadcast(t *testing.T) {
	b := &completionBroadcaster{
		clients: make(map[chan CompletionEvent]struct{}),
	}

	ch := b.subscribe()
	defer b.unsubscribe(ch)

	event := CompletionEvent{
		BeadsID:    "orch-go-test1",
		Reason:     "Phase: Complete",
		Escalation: "none",
		Source:     "daemon",
		Timestamp:  time.Now().Unix(),
	}
	b.broadcast(event)

	select {
	case received := <-ch:
		if received.BeadsID != "orch-go-test1" {
			t.Errorf("expected beads_id orch-go-test1, got %s", received.BeadsID)
		}
		if received.Source != "daemon" {
			t.Errorf("expected source daemon, got %s", received.Source)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for broadcast event")
	}
}

func TestCompletionBroadcaster_NonBlocking(t *testing.T) {
	b := &completionBroadcaster{
		clients: make(map[chan CompletionEvent]struct{}),
	}

	// Create a subscriber with buffer size 4
	ch := b.subscribe()
	defer b.unsubscribe(ch)

	// Fill the buffer
	for i := 0; i < 4; i++ {
		b.broadcast(CompletionEvent{BeadsID: "fill"})
	}

	// This should not block even though the channel buffer is full
	done := make(chan bool, 1)
	go func() {
		b.broadcast(CompletionEvent{BeadsID: "overflow"})
		done <- true
	}()

	select {
	case <-done:
		// Good - broadcast didn't block
	case <-time.After(100 * time.Millisecond):
		t.Fatal("broadcast blocked on full channel")
	}
}

func TestHandleCompletionNotify(t *testing.T) {
	// Save and restore global broadcaster
	saved := globalCompletionBroadcaster
	globalCompletionBroadcaster = &completionBroadcaster{
		clients: make(map[chan CompletionEvent]struct{}),
	}
	defer func() { globalCompletionBroadcaster = saved }()

	// Subscribe to receive events
	ch := globalCompletionBroadcaster.subscribe()
	defer globalCompletionBroadcaster.unsubscribe(ch)

	// POST a completion event
	event := CompletionEvent{
		BeadsID:    "orch-go-abc1",
		Reason:     "Phase: Complete - all tests passing",
		Escalation: "none",
		Source:     "daemon",
	}
	body, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/api/notify/completion", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCompletionNotify(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}

	// Check broadcast was received
	select {
	case received := <-ch:
		if received.BeadsID != "orch-go-abc1" {
			t.Errorf("expected beads_id orch-go-abc1, got %s", received.BeadsID)
		}
		if received.Timestamp == 0 {
			t.Error("expected timestamp to be set")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestHandleCompletionNotify_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/notify/completion", nil)
	w := httptest.NewRecorder()

	handleCompletionNotify(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleCompletionNotify_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/notify/completion", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	handleCompletionNotify(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleCompletionEvents_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/events/completion", nil)
	w := httptest.NewRecorder()

	handleCompletionEvents(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
