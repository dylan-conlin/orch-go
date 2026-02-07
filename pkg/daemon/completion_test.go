package daemon

import (
	"sync"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/internal/testutil"
)

func TestCompletionService_Track(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999" // Fake server, no actual SSE
	cs := NewCompletionService(cfg)

	// Track a session
	cs.Track("session-1", "beads-123", nil)

	if !cs.IsTracked("session-1") {
		t.Error("session-1 should be tracked")
	}

	if cs.IsTracked("session-2") {
		t.Error("session-2 should not be tracked")
	}

	// Verify tracked count
	if cs.TrackedSessionCount() != 1 {
		t.Errorf("expected 1 tracked session, got %d", cs.TrackedSessionCount())
	}
}

func TestCompletionService_TrackWithSlot(t *testing.T) {
	pool := NewWorkerPool(3)
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cfg.Pool = pool
	cs := NewCompletionService(cfg)

	// Acquire a slot
	slot := pool.TryAcquire()
	if slot == nil {
		t.Fatal("failed to acquire slot")
	}

	// Track session with slot
	cs.Track("session-1", "beads-123", slot)

	// Verify pool state
	if pool.Active() != 1 {
		t.Errorf("expected 1 active worker, got %d", pool.Active())
	}

	// Manually release via completion service
	if !cs.ReleaseSlot("session-1") {
		t.Error("ReleaseSlot should return true for tracked session")
	}

	// Verify slot was released
	if pool.Active() != 0 {
		t.Errorf("expected 0 active workers after release, got %d", pool.Active())
	}

	// Session should no longer be tracked
	if cs.IsTracked("session-1") {
		t.Error("session-1 should no longer be tracked after release")
	}
}

func TestCompletionService_Untrack(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cs := NewCompletionService(cfg)

	cs.Track("session-1", "beads-123", nil)
	if !cs.IsTracked("session-1") {
		t.Error("session-1 should be tracked")
	}

	cs.Untrack("session-1")
	if cs.IsTracked("session-1") {
		t.Error("session-1 should not be tracked after untrack")
	}
}

func TestCompletionService_GetTrackedSession(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cs := NewCompletionService(cfg)

	cs.Track("session-1", "beads-123", nil)

	session := cs.GetTrackedSession("session-1")
	if session == nil {
		t.Fatal("expected tracked session, got nil")
	}

	if session.SessionID != "session-1" {
		t.Errorf("expected SessionID 'session-1', got '%s'", session.SessionID)
	}

	if session.BeadsID != "beads-123" {
		t.Errorf("expected BeadsID 'beads-123', got '%s'", session.BeadsID)
	}

	// Non-existent session
	notFound := cs.GetTrackedSession("session-2")
	if notFound != nil {
		t.Error("expected nil for non-existent session")
	}
}

func TestCompletionService_TrackedSessions(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cs := NewCompletionService(cfg)

	cs.Track("session-1", "beads-1", nil)
	cs.Track("session-2", "beads-2", nil)
	cs.Track("session-3", "beads-3", nil)

	sessions := cs.TrackedSessions()
	if len(sessions) != 3 {
		t.Errorf("expected 3 tracked sessions, got %d", len(sessions))
	}

	// Check all sessions are in the list
	sessionSet := make(map[string]bool)
	for _, s := range sessions {
		sessionSet[s] = true
	}

	for _, expected := range []string{"session-1", "session-2", "session-3"} {
		if !sessionSet[expected] {
			t.Errorf("expected session '%s' in tracked list", expected)
		}
	}
}

func TestCompletionService_Status(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cs := NewCompletionService(cfg)

	cs.Track("session-1", "beads-1", nil)
	cs.Track("session-2", "beads-2", nil)

	status := cs.Status()
	if status.TrackedCount != 2 {
		t.Errorf("expected TrackedCount 2, got %d", status.TrackedCount)
	}

	if len(status.Sessions) != 2 {
		t.Errorf("expected 2 sessions in status, got %d", len(status.Sessions))
	}
}

func TestCompletionService_OnCompletionHandler(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cs := NewCompletionService(cfg)

	var receivedEvent CompletionEvent
	var handlerCalled bool
	var mu sync.Mutex

	cs.OnCompletion(func(event CompletionEvent) {
		mu.Lock()
		defer mu.Unlock()
		handlerCalled = true
		receivedEvent = event
	})

	// Track a session
	cs.Track("session-1", "beads-123", nil)

	// Simulate completion (calling the internal handler directly)
	cs.handleCompletion("session-1")

	// Wait for async handler to execute
	testutil.WaitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return handlerCalled
	}, "completion handler to be called")

	mu.Lock()
	defer mu.Unlock()

	if receivedEvent.SessionID != "session-1" {
		t.Errorf("expected SessionID 'session-1', got '%s'", receivedEvent.SessionID)
	}

	if receivedEvent.BeadsID != "beads-123" {
		t.Errorf("expected BeadsID 'beads-123', got '%s'", receivedEvent.BeadsID)
	}

	// Session should no longer be tracked after completion
	if cs.IsTracked("session-1") {
		t.Error("session-1 should no longer be tracked after completion")
	}
}

func TestCompletionService_CompletionReleasesSlot(t *testing.T) {
	pool := NewWorkerPool(3)
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cfg.Pool = pool
	cs := NewCompletionService(cfg)

	// Acquire multiple slots
	slot1 := pool.TryAcquire()
	slot2 := pool.TryAcquire()
	slot1.BeadsID = "beads-1"
	slot2.BeadsID = "beads-2"

	cs.Track("session-1", "beads-1", slot1)
	cs.Track("session-2", "beads-2", slot2)

	if pool.Active() != 2 {
		t.Errorf("expected 2 active workers, got %d", pool.Active())
	}

	// Simulate completion of session-1
	cs.handleCompletion("session-1")

	// Slot should be released
	if pool.Active() != 1 {
		t.Errorf("expected 1 active worker after completion, got %d", pool.Active())
	}

	// Session-1 no longer tracked, session-2 still tracked
	if cs.IsTracked("session-1") {
		t.Error("session-1 should no longer be tracked")
	}
	if !cs.IsTracked("session-2") {
		t.Error("session-2 should still be tracked")
	}
}

func TestCompletionService_CompletionForUntrackedSession(t *testing.T) {
	pool := NewWorkerPool(3)
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cfg.Pool = pool
	cs := NewCompletionService(cfg)

	var handlerCalled bool
	cs.OnCompletion(func(event CompletionEvent) {
		handlerCalled = true
	})

	// Don't track any session, but try to complete one
	cs.handleCompletion("unknown-session")

	// For negative tests, use Eventually returning false (short timeout is acceptable)
	if testutil.Eventually(func() bool { return handlerCalled }, 50*testutil.DefaultInterval) {
		t.Error("handler should not be called for untracked sessions")
	}
}

func TestCompletionService_ReleaseSlotNonExistent(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cs := NewCompletionService(cfg)

	// Try to release a non-existent session
	if cs.ReleaseSlot("non-existent") {
		t.Error("ReleaseSlot should return false for non-existent session")
	}
}

func TestCompletionService_MultipleHandlers(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cs := NewCompletionService(cfg)

	var handler1Called, handler2Called bool
	var mu sync.Mutex

	cs.OnCompletion(func(event CompletionEvent) {
		mu.Lock()
		defer mu.Unlock()
		handler1Called = true
	})

	cs.OnCompletion(func(event CompletionEvent) {
		mu.Lock()
		defer mu.Unlock()
		handler2Called = true
	})

	cs.Track("session-1", "beads-1", nil)
	cs.handleCompletion("session-1")

	// Wait for both async handlers to execute
	testutil.WaitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return handler1Called && handler2Called
	}, "both handlers to be called")

	mu.Lock()
	defer mu.Unlock()

	if !handler1Called {
		t.Error("handler1 should have been called")
	}
	if !handler2Called {
		t.Error("handler2 should have been called")
	}
}

func TestCompletionService_ConcurrentTracking(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()
	cfg.ServerURL = "http://127.0.0.1:9999"
	cs := NewCompletionService(cfg)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently track sessions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sessionID := "session-" + string(rune('A'+id%26)) + string(rune('0'+id/26))
			cs.Track(sessionID, "beads-"+sessionID, nil)
		}(i)
	}

	wg.Wait()

	// Should have tracked all sessions
	count := cs.TrackedSessionCount()
	if count != numGoroutines {
		t.Errorf("expected %d tracked sessions, got %d", numGoroutines, count)
	}
}

func TestDefaultCompletionServiceConfig(t *testing.T) {
	cfg := DefaultCompletionServiceConfig()

	if cfg.ServerURL != "http://localhost:4096" {
		t.Errorf("expected default server URL 'http://localhost:4096', got '%s'", cfg.ServerURL)
	}

	if cfg.ReconnectBackoff != 5*time.Second {
		t.Errorf("expected default backoff 5s, got %v", cfg.ReconnectBackoff)
	}

	if cfg.MaxReconnects != -1 {
		t.Errorf("expected default max reconnects -1, got %d", cfg.MaxReconnects)
	}
}
