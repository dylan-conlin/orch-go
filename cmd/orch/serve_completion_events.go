package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CompletionEvent is the payload pushed to dashboard clients when the daemon
// processes an agent completion.
type CompletionEvent struct {
	// BeadsID is the beads issue ID of the completed agent.
	BeadsID string `json:"beads_id"`
	// Reason is the completion/close reason.
	Reason string `json:"reason,omitempty"`
	// Escalation is the verification escalation level (none, info, review, block, failed).
	Escalation string `json:"escalation,omitempty"`
	// Source identifies who sent the notification (e.g., "daemon", "orch-complete").
	Source string `json:"source,omitempty"`
	// Timestamp is when the completion was processed.
	Timestamp int64 `json:"timestamp"`
}

// completionBroadcaster manages SSE clients subscribed to completion events.
// Same pattern as contextBroadcaster in serve_context.go.
type completionBroadcaster struct {
	mu      sync.RWMutex
	clients map[chan CompletionEvent]struct{}
}

var globalCompletionBroadcaster = &completionBroadcaster{
	clients: make(map[chan CompletionEvent]struct{}),
}

// subscribe registers a client channel for completion events.
func (b *completionBroadcaster) subscribe() chan CompletionEvent {
	ch := make(chan CompletionEvent, 4) // Buffered to prevent blocking broadcaster
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// unsubscribe removes a client channel.
func (b *completionBroadcaster) unsubscribe(ch chan CompletionEvent) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// broadcast sends a completion event to all connected clients.
func (b *completionBroadcaster) broadcast(event CompletionEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		// Non-blocking send: drop if client is behind
		select {
		case ch <- event:
		default:
		}
	}
}

// handleCompletionNotify accepts a POST from the daemon when it processes a completion.
// This enables push-based completion surfacing: the daemon notifies the dashboard
// immediately instead of the dashboard discovering completions via polling.
func handleCompletionNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event CompletionEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Ensure timestamp is set
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}

	// Broadcast to all connected SSE clients
	globalCompletionBroadcaster.broadcast(event)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleCompletionEvents streams completion events via SSE.
// Dashboard clients subscribe to receive real-time notifications when agents complete.
func handleCompletionEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to completion events
	ch := globalCompletionBroadcaster.subscribe()
	defer globalCompletionBroadcaster.unsubscribe(ch)

	// Send connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"source\":\"completion-broadcaster\"}\n\n")
	flusher.Flush()

	ctx := r.Context()

	// Stream completion events to client
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: agent.completed\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}
