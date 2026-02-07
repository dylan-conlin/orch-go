// Package materializer provides an SSE-driven state materializer that subscribes
// to OpenCode's SSE event stream and writes agent state changes to state.db in
// real-time.
//
// This makes state.db a near-realtime projection of agent state, eliminating the
// need for subprocess polling to get status/processing information.
//
// Architecture:
//   - Subscribes to OpenCode SSE stream (http://127.0.0.1:4096/event)
//   - Processes session.status events → updates is_processing in state.db
//   - Processes message.part events → updates session_updated_at (last activity)
//   - Reconnects with exponential backoff on disconnect
//   - Gracefully shuts down via context cancellation
//
// Integration: Started as a goroutine by orch serve. Health status is exposed
// via the /health endpoint.
//
// Constraint: SSE busy→idle CANNOT detect true agent completion (known constraint).
// The materializer only updates is_processing and activity timestamps, NOT
// completion status.
package materializer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/state"
)

// ConnectionState represents the current state of the SSE connection.
type ConnectionState int

const (
	// StateDisconnected means the materializer is not connected to SSE.
	StateDisconnected ConnectionState = iota
	// StateConnecting means a connection attempt is in progress.
	StateConnecting
	// StateConnected means the SSE stream is actively being consumed.
	StateConnected
)

// String returns the string representation of a ConnectionState.
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	default:
		return "unknown"
	}
}

// Status represents the materializer's current health status.
type Status struct {
	State             ConnectionState `json:"state"`
	StateStr          string          `json:"state_str"`
	ReconnectAttempts int             `json:"reconnect_attempts"`
	EventsProcessed   int64           `json:"events_processed"`
	LastEventAt       time.Time       `json:"last_event_at,omitempty"`
	LastError         string          `json:"last_error,omitempty"`
	StartedAt         time.Time       `json:"started_at"`
}

// Config holds configuration for the materializer.
type Config struct {
	// ServerURL is the OpenCode server URL (e.g., http://127.0.0.1:4096).
	ServerURL string

	// DBPath overrides the default state.db path. Empty uses default.
	DBPath string

	// InitialReconnectDelay is the starting backoff for reconnection.
	// Defaults to 2 seconds.
	InitialReconnectDelay time.Duration

	// MaxReconnectDelay is the maximum backoff duration.
	// Defaults to 60 seconds.
	MaxReconnectDelay time.Duration
}

// EventHandler processes parsed SSE events and writes to state.db.
// Extracted as an interface for testability.
type EventHandler interface {
	HandleEvent(event opencode.SSEEvent) error
}

// Materializer subscribes to OpenCode SSE events and materializes state into state.db.
type Materializer struct {
	config Config
	db     *state.DB

	// State tracking
	mu                sync.RWMutex
	state             ConnectionState
	reconnectAttempts int
	eventsProcessed   int64
	lastEventAt       time.Time
	lastError         string
	startedAt         time.Time

	// For graceful shutdown
	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a new Materializer with the given configuration.
func New(config Config) *Materializer {
	if config.InitialReconnectDelay == 0 {
		config.InitialReconnectDelay = 2 * time.Second
	}
	if config.MaxReconnectDelay == 0 {
		config.MaxReconnectDelay = 60 * time.Second
	}

	return &Materializer{
		config: config,
		done:   make(chan struct{}),
	}
}

// Start begins the materializer goroutine. It subscribes to the SSE stream
// and processes events until the context is cancelled or Stop() is called.
func (m *Materializer) Start(ctx context.Context) error {
	// Open the state database
	db, err := state.Open(m.config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	m.db = db

	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.startedAt = time.Now()

	go m.run(ctx)

	return nil
}

// Stop stops the materializer gracefully.
func (m *Materializer) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	<-m.done

	// Close the database
	if m.db != nil {
		m.db.Close()
	}
}

// Status returns the current materializer health status.
func (m *Materializer) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Status{
		State:             m.state,
		StateStr:          m.state.String(),
		ReconnectAttempts: m.reconnectAttempts,
		EventsProcessed:   m.eventsProcessed,
		LastEventAt:       m.lastEventAt,
		LastError:         m.lastError,
		StartedAt:         m.startedAt,
	}
}

// run is the main materializer loop with reconnection logic.
func (m *Materializer) run(ctx context.Context) {
	defer close(m.done)

	delay := m.config.InitialReconnectDelay

	for {
		select {
		case <-ctx.Done():
			m.setState(StateDisconnected)
			return
		default:
		}

		m.setState(StateConnecting)
		err := m.connectAndProcess(ctx)

		select {
		case <-ctx.Done():
			m.setState(StateDisconnected)
			return
		default:
		}

		// Connection failed or lost
		m.setState(StateDisconnected)

		m.mu.Lock()
		m.reconnectAttempts++
		if err != nil {
			m.lastError = err.Error()
		}
		attempt := m.reconnectAttempts
		m.mu.Unlock()

		fmt.Printf("[materializer] SSE connection error: %v, reconnecting in %v (attempt %d)\n",
			err, delay, attempt)

		// Wait with backoff
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		// Exponential backoff
		delay = delay * 2
		if delay > m.config.MaxReconnectDelay {
			delay = m.config.MaxReconnectDelay
		}
	}
}

// connectAndProcess establishes SSE connection and processes events.
func (m *Materializer) connectAndProcess(ctx context.Context) error {
	sseURL := m.config.ServerURL + "/event"
	sseClient := opencode.NewSSEClient(sseURL)

	events := make(chan opencode.SSEEvent, 100)
	errChan := make(chan error, 1)

	go func() {
		if err := sseClient.Connect(events); err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
		close(events)
	}()

	connectionEstablished := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-events:
			if !ok {
				return fmt.Errorf("SSE connection closed")
			}

			if !connectionEstablished {
				m.setState(StateConnected)
				// Reset backoff on successful connection
				m.mu.Lock()
				if m.reconnectAttempts > 0 {
					fmt.Printf("[materializer] SSE reconnected after %d attempts\n", m.reconnectAttempts)
				}
				m.reconnectAttempts = 0
				m.mu.Unlock()
				connectionEstablished = true
			}

			m.handleEvent(event)

		case err := <-errChan:
			return err
		}
	}
}

// handleEvent processes a single SSE event and writes to state.db.
func (m *Materializer) handleEvent(event opencode.SSEEvent) {
	m.mu.Lock()
	m.eventsProcessed++
	m.lastEventAt = time.Now()
	m.mu.Unlock()

	switch event.Event {
	case "session.status":
		m.handleSessionStatus(event.Data)

	case "message.part.updated", "message.part.created":
		m.handleMessagePart(event.Data)
	}
}

// handleSessionStatus processes session.status events to update is_processing.
func (m *Materializer) handleSessionStatus(data string) {
	status, sessionID := opencode.ParseSessionStatus(data)
	if sessionID == "" {
		return
	}

	isProcessing := status == "busy" || status == "running"

	if err := m.db.UpdateProcessingBySessionID(sessionID, isProcessing); err != nil {
		fmt.Printf("[materializer] failed to update processing for session %s: %v\n", sessionID, err)
	}
}

// handleMessagePart processes message.part events to update activity timestamps
// and extract token counts.
func (m *Materializer) handleMessagePart(data string) {
	// Parse to extract session ID and optional token info
	var part struct {
		SessionID string `json:"sessionID"`
		MessageID string `json:"messageID"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal([]byte(data), &part); err != nil {
		return
	}

	if part.SessionID == "" {
		// Try nested properties format
		var nested struct {
			Properties struct {
				SessionID string `json:"sessionID"`
			} `json:"properties"`
		}
		if err := json.Unmarshal([]byte(data), &nested); err == nil {
			part.SessionID = nested.Properties.SessionID
		}
	}

	if part.SessionID == "" {
		return
	}

	// Update last activity timestamp
	if err := m.db.UpdateSessionActivity(part.SessionID); err != nil {
		fmt.Printf("[materializer] failed to update activity for session %s: %v\n", part.SessionID, err)
	}
}

// setState updates the connection state thread-safely.
func (m *Materializer) setState(s ConnectionState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = s
}
