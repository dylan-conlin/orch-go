// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// CompletionEvent represents an agent completion event.
type CompletionEvent struct {
	SessionID   string
	BeadsID     string
	CompletedAt time.Time
	Duration    time.Duration // Time from spawn to completion
}

// CompletionHandler is called when an agent completes.
type CompletionHandler func(CompletionEvent)

// CompletionService tracks headless agent sessions and releases slots on completion.
// It bridges the gap between SSE-based completion detection (Monitor) and
// slot management (WorkerPool/CapacityManager).
type CompletionService struct {
	mu sync.RWMutex

	// Session tracking: sessionID → TrackedSession
	sessions map[string]*TrackedSession

	// Handlers for completion events
	handlers []CompletionHandler

	// Pool for releasing slots (optional)
	pool *WorkerPool

	// Monitor for SSE events
	monitor *opencode.Monitor

	// Reconnect settings
	serverURL        string
	reconnectBackoff time.Duration
	maxReconnects    int

	// Shutdown
	cancel context.CancelFunc
	done   chan struct{}
}

// TrackedSession tracks a headless agent session for completion detection.
type TrackedSession struct {
	SessionID  string
	BeadsID    string
	Slot       *Slot
	SpawnedAt  time.Time
	LastStatus string
}

// CompletionServiceConfig configures the CompletionService.
type CompletionServiceConfig struct {
	// ServerURL is the OpenCode server URL (for SSE reconnection)
	ServerURL string

	// Pool is the worker pool for slot management (optional)
	Pool *WorkerPool

	// ReconnectBackoff is the initial backoff duration for reconnection
	ReconnectBackoff time.Duration

	// MaxReconnects is the maximum number of reconnection attempts (-1 = infinite)
	MaxReconnects int
}

// DefaultCompletionServiceConfig returns sensible defaults.
func DefaultCompletionServiceConfig() CompletionServiceConfig {
	return CompletionServiceConfig{
		ServerURL:        "http://127.0.0.1:4096",
		ReconnectBackoff: 5 * time.Second,
		MaxReconnects:    -1, // Infinite reconnects for daemon mode
	}
}

// NewCompletionService creates a new CompletionService.
func NewCompletionService(cfg CompletionServiceConfig) *CompletionService {
	cs := &CompletionService{
		sessions:         make(map[string]*TrackedSession),
		handlers:         make([]CompletionHandler, 0),
		pool:             cfg.Pool,
		serverURL:        cfg.ServerURL,
		reconnectBackoff: cfg.ReconnectBackoff,
		maxReconnects:    cfg.MaxReconnects,
		done:             make(chan struct{}),
	}

	// Create and configure monitor
	cs.monitor = opencode.NewMonitor(cfg.ServerURL)
	cs.monitor.OnCompletion(cs.handleCompletion)

	return cs
}

// OnCompletion registers a handler to be called when an agent completes.
// Multiple handlers can be registered.
func (cs *CompletionService) OnCompletion(handler CompletionHandler) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.handlers = append(cs.handlers, handler)
}

// Track starts tracking a headless session for completion.
// The slot will be released when the session completes (if provided).
func (cs *CompletionService) Track(sessionID, beadsID string, slot *Slot) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.sessions[sessionID] = &TrackedSession{
		SessionID: sessionID,
		BeadsID:   beadsID,
		Slot:      slot,
		SpawnedAt: time.Now(),
	}
}

// Untrack stops tracking a session (e.g., if spawn fails).
func (cs *CompletionService) Untrack(sessionID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.sessions, sessionID)
}

// IsTracked returns true if the session is being tracked.
func (cs *CompletionService) IsTracked(sessionID string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	_, exists := cs.sessions[sessionID]
	return exists
}

// GetTrackedSession returns information about a tracked session.
// Returns nil if session is not tracked.
func (cs *CompletionService) GetTrackedSession(sessionID string) *TrackedSession {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	session, exists := cs.sessions[sessionID]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	return &TrackedSession{
		SessionID:  session.SessionID,
		BeadsID:    session.BeadsID,
		SpawnedAt:  session.SpawnedAt,
		LastStatus: session.LastStatus,
		// Note: Slot is not copied (it's a pointer managed by the pool)
	}
}

// TrackedSessionCount returns the number of sessions being tracked.
func (cs *CompletionService) TrackedSessionCount() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return len(cs.sessions)
}

// TrackedSessions returns all tracked session IDs.
func (cs *CompletionService) TrackedSessions() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]string, 0, len(cs.sessions))
	for id := range cs.sessions {
		result = append(result, id)
	}
	return result
}

// Start begins monitoring SSE events for completions.
// This is non-blocking; call Stop() to stop monitoring.
func (cs *CompletionService) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	cs.cancel = cancel

	// Start the underlying monitor
	cs.monitor.Start()

	// Start a goroutine to handle context cancellation
	go func() {
		<-ctx.Done()
		cs.monitor.Stop()
		close(cs.done)
	}()
}

// Stop stops the completion service gracefully.
func (cs *CompletionService) Stop() {
	if cs.cancel != nil {
		cs.cancel()
	}
	<-cs.done
}

// handleCompletion is called by the Monitor when a session completes.
func (cs *CompletionService) handleCompletion(sessionID string) {
	cs.mu.Lock()

	// Check if we're tracking this session
	session, exists := cs.sessions[sessionID]
	if !exists {
		cs.mu.Unlock()
		return
	}

	// Prepare completion event before releasing lock
	event := CompletionEvent{
		SessionID:   sessionID,
		BeadsID:     session.BeadsID,
		CompletedAt: time.Now(),
		Duration:    time.Since(session.SpawnedAt),
	}

	// Get the slot before removing session
	slot := session.Slot

	// Remove from tracking
	delete(cs.sessions, sessionID)

	// Copy handlers to avoid holding lock during callback
	handlers := make([]CompletionHandler, len(cs.handlers))
	copy(handlers, cs.handlers)

	cs.mu.Unlock()

	// Release the slot (if pool is configured and slot exists)
	if cs.pool != nil && slot != nil {
		cs.pool.Release(slot)
	}

	// Notify handlers
	for _, handler := range handlers {
		handler(event)
	}
}

// ReleaseSlot manually releases a slot for a session.
// Useful for error recovery or manual cleanup.
func (cs *CompletionService) ReleaseSlot(sessionID string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	session, exists := cs.sessions[sessionID]
	if !exists {
		return false
	}

	// Release slot if present
	if cs.pool != nil && session.Slot != nil {
		cs.pool.Release(session.Slot)
	}

	// Remove from tracking
	delete(cs.sessions, sessionID)

	return true
}

// Status returns the current status of the completion service.
type CompletionServiceStatus struct {
	TrackedCount int
	Sessions     []TrackedSessionInfo
}

// TrackedSessionInfo provides information about a tracked session.
type TrackedSessionInfo struct {
	SessionID  string
	BeadsID    string
	SpawnedAt  time.Time
	Duration   time.Duration
	LastStatus string
}

// Status returns the current status of all tracked sessions.
func (cs *CompletionService) Status() CompletionServiceStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	now := time.Now()
	sessions := make([]TrackedSessionInfo, 0, len(cs.sessions))
	for _, s := range cs.sessions {
		sessions = append(sessions, TrackedSessionInfo{
			SessionID:  s.SessionID,
			BeadsID:    s.BeadsID,
			SpawnedAt:  s.SpawnedAt,
			Duration:   now.Sub(s.SpawnedAt),
			LastStatus: s.LastStatus,
		})
	}

	return CompletionServiceStatus{
		TrackedCount: len(cs.sessions),
		Sessions:     sessions,
	}
}
