// Package opencode provides a client for interacting with OpenCode sessions.
package opencode

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SessionState tracks the current state of a session.
type SessionState struct {
	SessionID   string
	Status      string
	WasBusy     bool // Track if session was ever busy (to detect completion)
	LastUpdated time.Time
}

// CompletionHandler is called when a session completes.
type CompletionHandler func(sessionID string)

// Monitor watches SSE events and detects session completions.
type Monitor struct {
	sseClient *SSEClient
	sessions  map[string]*SessionState
	handlers  []CompletionHandler
	mu        sync.RWMutex

	// For graceful shutdown
	cancel context.CancelFunc
	done   chan struct{}
}

// NewMonitor creates a new SSE monitor for the given server URL.
func NewMonitor(serverURL string) *Monitor {
	sseURL := serverURL + "/event"
	return &Monitor{
		sseClient: NewSSEClient(sseURL),
		sessions:  make(map[string]*SessionState),
		handlers:  make([]CompletionHandler, 0),
		done:      make(chan struct{}),
	}
}

// OnCompletion registers a handler to be called when a session completes.
// Multiple handlers can be registered.
func (m *Monitor) OnCompletion(handler CompletionHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// Start begins monitoring SSE events in the background.
// Returns immediately. Call Stop() to stop monitoring.
func (m *Monitor) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	go m.run(ctx)
}

// Stop stops the monitor gracefully.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	<-m.done
}

// run is the main monitoring loop.
func (m *Monitor) run(ctx context.Context) {
	defer close(m.done)

	events := make(chan SSEEvent, 100)
	errChan := make(chan error, 1)

	// Start SSE connection in a goroutine
	go func() {
		if err := m.sseClient.Connect(events); err != nil {
			errChan <- err
		}
		close(events)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				// Connection closed, try to reconnect
				m.reconnect(ctx, events, errChan)
				continue
			}
			m.handleEvent(event)
		case err := <-errChan:
			fmt.Printf("SSE connection error: %v\n", err)
			// Try to reconnect after a delay
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				m.reconnect(ctx, events, errChan)
			}
		}
	}
}

// reconnect attempts to reconnect to the SSE stream.
func (m *Monitor) reconnect(ctx context.Context, events chan SSEEvent, errChan chan error) {
	// Create a new events channel since the old one is closed
	newEvents := make(chan SSEEvent, 100)

	go func() {
		if err := m.sseClient.Connect(newEvents); err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
		close(newEvents)
	}()

	// Forward events from new channel to main loop
	go func() {
		for event := range newEvents {
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
}

// handleEvent processes an SSE event and detects completions.
func (m *Monitor) handleEvent(event SSEEvent) {
	// Only care about session.status events for completion detection
	if event.Event != "session.status" {
		return
	}

	status, sessionID := ParseSessionStatus(event.Data)
	if sessionID == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create session state
	state, exists := m.sessions[sessionID]
	if !exists {
		state = &SessionState{
			SessionID: sessionID,
		}
		m.sessions[sessionID] = state
	}

	// Track if the session was ever busy
	if status == "busy" || status == "running" {
		state.WasBusy = true
	}

	// Detect completion: transition from busy to idle
	wasRunning := state.Status == "busy" || state.Status == "running" || state.WasBusy
	nowIdle := status == "idle"

	// Update state
	prevStatus := state.Status
	state.Status = status
	state.LastUpdated = time.Now()

	// Trigger completion if we transitioned to idle after being busy
	if wasRunning && nowIdle && prevStatus != "idle" {
		// Call handlers outside the lock
		handlers := make([]CompletionHandler, len(m.handlers))
		copy(handlers, m.handlers)

		go func(sid string) {
			for _, handler := range handlers {
				handler(sid)
			}
		}(sessionID)

		// Reset WasBusy for next run in same session
		state.WasBusy = false
	}
}

// GetSessionState returns the current state of a session.
// Returns nil if session is not being tracked.
func (m *Monitor) GetSessionState(sessionID string) *SessionState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.sessions[sessionID]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	return &SessionState{
		SessionID:   state.SessionID,
		Status:      state.Status,
		WasBusy:     state.WasBusy,
		LastUpdated: state.LastUpdated,
	}
}

// ActiveSessions returns a list of currently tracked session IDs.
func (m *Monitor) ActiveSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		result = append(result, id)
	}
	return result
}
