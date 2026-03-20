package daemon

import (
	"fmt"
	"testing"
	"time"
)

func TestCachedAgentDiscoverer_CachesGetActiveAgents(t *testing.T) {
	callCount := 0
	inner := &mockAgentDiscoverer{
		GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
			callCount++
			return []ActiveAgent{
				{BeadsID: "test-1", Phase: "Implementing", UpdatedAt: time.Now()},
			}, nil
		},
	}

	cached := newCachedAgentDiscoverer(inner)

	// First call should hit inner
	agents1, err := cached.GetActiveAgents()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents1) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents1))
	}
	if callCount != 1 {
		t.Fatalf("expected 1 inner call, got %d", callCount)
	}

	// Second call should return cached result
	agents2, err := cached.GetActiveAgents()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents2) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents2))
	}
	if callCount != 1 {
		t.Fatalf("expected still 1 inner call after cache hit, got %d", callCount)
	}
}

func TestCachedAgentDiscoverer_CachesError(t *testing.T) {
	callCount := 0
	inner := &mockAgentDiscoverer{
		GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
			callCount++
			return nil, fmt.Errorf("beads unavailable")
		},
	}

	cached := newCachedAgentDiscoverer(inner)

	_, err1 := cached.GetActiveAgents()
	if err1 == nil {
		t.Fatal("expected error")
	}

	_, err2 := cached.GetActiveAgents()
	if err2 == nil {
		t.Fatal("expected cached error")
	}
	if callCount != 1 {
		t.Fatalf("expected 1 inner call (error cached), got %d", callCount)
	}
}

func TestCachedAgentDiscoverer_DelegatesSessionChecks(t *testing.T) {
	inner := &mockAgentDiscoverer{
		HasExistingSessionFunc: func(beadsID string) bool {
			return beadsID == "test-1"
		},
		HasExistingSessionOrErrorFunc: func(beadsID string) (bool, error) {
			if beadsID == "test-1" {
				return true, nil
			}
			return false, nil
		},
	}

	cached := newCachedAgentDiscoverer(inner)

	if !cached.HasExistingSession("test-1") {
		t.Error("expected HasExistingSession to delegate to inner")
	}
	if cached.HasExistingSession("test-2") {
		t.Error("expected false for unknown agent")
	}

	found, err := cached.HasExistingSessionOrError("test-1")
	if err != nil || !found {
		t.Error("expected HasExistingSessionOrError to delegate to inner")
	}
}

func TestDaemon_BeginEndCycle(t *testing.T) {
	callCount := 0
	mock := &mockAgentDiscoverer{
		GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
			callCount++
			return []ActiveAgent{{BeadsID: "a1"}}, nil
		},
	}

	d := &Daemon{
		Agents:         mock,
		resumeAttempts: make(map[string]time.Time),
	}

	// Before cycle: calls go directly to mock
	d.Agents.GetActiveAgents()
	if callCount != 1 {
		t.Fatalf("expected 1 call before cycle, got %d", callCount)
	}

	// Begin cycle: wraps with cache
	d.BeginCycle()

	d.Agents.GetActiveAgents()
	if callCount != 2 {
		t.Fatalf("expected 2 calls (first cache miss), got %d", callCount)
	}

	d.Agents.GetActiveAgents()
	if callCount != 2 {
		t.Fatalf("expected still 2 calls (cache hit), got %d", callCount)
	}

	// End cycle: restores original
	d.EndCycle()

	d.Agents.GetActiveAgents()
	if callCount != 3 {
		t.Fatalf("expected 3 calls after cycle end, got %d", callCount)
	}
}
