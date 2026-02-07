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

	// Use a loop for reconnection instead of spawning new goroutines
	// This prevents goroutine leaks on reconnection
	reconnectDelay := 5 * time.Second

	for {
		// Check if we should stop before connecting
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Connect and process events
		err := m.connectAndProcess(ctx)
		if err != nil {
			// Check if context was cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			fmt.Printf("SSE connection error: %v, reconnecting in %v\n", err, reconnectDelay)

			// Wait before reconnecting
			select {
			case <-ctx.Done():
				return
			case <-time.After(reconnectDelay):
				continue
			}
		}
	}
}

// connectAndProcess connects to SSE and processes events until disconnection.
// Returns an error if the connection fails or is lost.
func (m *Monitor) connectAndProcess(ctx context.Context) error {
	events := make(chan SSEEvent, 100)
	errChan := make(chan error, 1)

	// Start SSE connection in a goroutine
	go func() {
		if err := m.sseClient.Connect(events); err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
		close(events)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				// Connection closed, return to trigger reconnection
				return fmt.Errorf("SSE connection closed")
			}
			m.handleEvent(event)
		case err := <-errChan:
			return err
		}
	}
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

		// Clean up session from map to prevent memory leak.
		// Sessions that complete shouldn't stay in memory forever.
		delete(m.sessions, sessionID)
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
