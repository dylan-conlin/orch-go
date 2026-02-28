package daemon

import (
	"fmt"
	"testing"
	"time"
)

func TestShouldRunPhaseTimeout(t *testing.T) {
	t.Run("disabled returns false", func(t *testing.T) {
		d := NewWithConfig(Config{PhaseTimeoutEnabled: false})
		if d.ShouldRunPhaseTimeout() {
			t.Error("expected false when disabled")
		}
	})

	t.Run("zero interval returns false", func(t *testing.T) {
		d := NewWithConfig(Config{PhaseTimeoutEnabled: true, PhaseTimeoutInterval: 0})
		if d.ShouldRunPhaseTimeout() {
			t.Error("expected false when interval is zero")
		}
	})

	t.Run("first run returns true", func(t *testing.T) {
		d := NewWithConfig(Config{
			PhaseTimeoutEnabled:  true,
			PhaseTimeoutInterval: 5 * time.Minute,
		})
		if !d.ShouldRunPhaseTimeout() {
			t.Error("expected true on first run (zero lastPhaseTimeout)")
		}
	})

	t.Run("too soon returns false", func(t *testing.T) {
		d := NewWithConfig(Config{
			PhaseTimeoutEnabled:  true,
			PhaseTimeoutInterval: 5 * time.Minute,
		})
		d.lastPhaseTimeout = time.Now().Add(-2 * time.Minute)
		if d.ShouldRunPhaseTimeout() {
			t.Error("expected false when last check was 2min ago (interval 5min)")
		}
	})

	t.Run("interval elapsed returns true", func(t *testing.T) {
		d := NewWithConfig(Config{
			PhaseTimeoutEnabled:  true,
			PhaseTimeoutInterval: 5 * time.Minute,
		})
		d.lastPhaseTimeout = time.Now().Add(-6 * time.Minute)
		if !d.ShouldRunPhaseTimeout() {
			t.Error("expected true when last check was 6min ago (interval 5min)")
		}
	})
}

func TestRunPeriodicPhaseTimeout(t *testing.T) {
	t.Run("returns nil when not due", func(t *testing.T) {
		d := NewWithConfig(Config{PhaseTimeoutEnabled: false})
		result := d.RunPeriodicPhaseTimeout()
		if result != nil {
			t.Error("expected nil when disabled")
		}
	})

	t.Run("detects unresponsive agents", func(t *testing.T) {
		d := NewWithConfig(Config{
			PhaseTimeoutEnabled:   true,
			PhaseTimeoutInterval:  1 * time.Minute,
			PhaseTimeoutThreshold: 30 * time.Minute,
		})

		oldTime := time.Now().Add(-45 * time.Minute)
		recentTime := time.Now().Add(-10 * time.Minute)

		d.Agents = &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "orch-go-aaa", Phase: "Implementing", UpdatedAt: oldTime, Title: "Old agent"},
					{BeadsID: "orch-go-bbb", Phase: "Planning", UpdatedAt: recentTime, Title: "Recent agent"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool {
				// Both agents have sessions
				return beadsID == "orch-go-aaa" || beadsID == "orch-go-bbb"
			},
		}

		result := d.RunPeriodicPhaseTimeout()
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		if result.UnresponsiveCount != 1 {
			t.Errorf("expected 1 unresponsive, got %d", result.UnresponsiveCount)
		}
		if len(result.Agents) != 1 {
			t.Fatalf("expected 1 agent in result, got %d", len(result.Agents))
		}
		if result.Agents[0].BeadsID != "orch-go-aaa" {
			t.Errorf("expected unresponsive agent orch-go-aaa, got %s", result.Agents[0].BeadsID)
		}
	})

	t.Run("skips Phase Complete agents", func(t *testing.T) {
		d := NewWithConfig(Config{
			PhaseTimeoutEnabled:   true,
			PhaseTimeoutInterval:  1 * time.Minute,
			PhaseTimeoutThreshold: 30 * time.Minute,
		})

		oldTime := time.Now().Add(-45 * time.Minute)

		d.Agents = &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "orch-go-ccc", Phase: "Complete", UpdatedAt: oldTime, Title: "Completed agent"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return true },
		}

		result := d.RunPeriodicPhaseTimeout()
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.UnresponsiveCount != 0 {
			t.Errorf("expected 0 unresponsive (Complete agents should be skipped), got %d", result.UnresponsiveCount)
		}
	})

	t.Run("skips agents without sessions", func(t *testing.T) {
		d := NewWithConfig(Config{
			PhaseTimeoutEnabled:   true,
			PhaseTimeoutInterval:  1 * time.Minute,
			PhaseTimeoutThreshold: 30 * time.Minute,
		})

		oldTime := time.Now().Add(-45 * time.Minute)

		d.Agents = &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "orch-go-ddd", Phase: "Implementing", UpdatedAt: oldTime, Title: "No session agent"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		}

		result := d.RunPeriodicPhaseTimeout()
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.UnresponsiveCount != 0 {
			t.Errorf("expected 0 unresponsive (no-session agents handled by orphan detection), got %d", result.UnresponsiveCount)
		}
	})

	t.Run("handles agent discovery error", func(t *testing.T) {
		d := NewWithConfig(Config{
			PhaseTimeoutEnabled:   true,
			PhaseTimeoutInterval:  1 * time.Minute,
			PhaseTimeoutThreshold: 30 * time.Minute,
		})

		d.Agents = &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return nil, fmt.Errorf("mock discovery error")
			},
		}

		result := d.RunPeriodicPhaseTimeout()
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Error == nil {
			t.Error("expected error in result")
		}
	})

	t.Run("updates lastPhaseTimeout timestamp", func(t *testing.T) {
		d := NewWithConfig(Config{
			PhaseTimeoutEnabled:   true,
			PhaseTimeoutInterval:  1 * time.Minute,
			PhaseTimeoutThreshold: 30 * time.Minute,
		})

		d.Agents = &mockAgentDiscoverer{}

		before := time.Now()
		d.RunPeriodicPhaseTimeout()
		after := time.Now()

		if d.lastPhaseTimeout.Before(before) || d.lastPhaseTimeout.After(after) {
			t.Errorf("lastPhaseTimeout not updated correctly: got %v, expected between %v and %v",
				d.lastPhaseTimeout, before, after)
		}
	})
}

func TestPhaseTimeoutSnapshot(t *testing.T) {
	result := &PhaseTimeoutResult{
		UnresponsiveCount: 2,
		Agents: []UnresponsiveAgent{
			{BeadsID: "orch-go-111", Title: "Agent 1", Phase: "Planning", IdleDuration: 45 * time.Minute},
			{BeadsID: "orch-go-222", Title: "Agent 2", Phase: "Implementing", IdleDuration: 60 * time.Minute},
		},
	}

	snapshot := result.Snapshot()
	if snapshot.UnresponsiveCount != 2 {
		t.Errorf("expected UnresponsiveCount 2, got %d", snapshot.UnresponsiveCount)
	}
	if snapshot.LastCheck.IsZero() {
		t.Error("expected non-zero LastCheck")
	}
}

func TestLastPhaseTimeoutTime(t *testing.T) {
	d := NewWithConfig(Config{})
	if !d.LastPhaseTimeoutTime().IsZero() {
		t.Error("expected zero time before any run")
	}
}

func TestNextPhaseTimeoutTime(t *testing.T) {
	t.Run("disabled returns zero", func(t *testing.T) {
		d := NewWithConfig(Config{PhaseTimeoutEnabled: false})
		if !d.NextPhaseTimeoutTime().IsZero() {
			t.Error("expected zero time when disabled")
		}
	})

	t.Run("never run returns now", func(t *testing.T) {
		d := NewWithConfig(Config{
			PhaseTimeoutEnabled:  true,
			PhaseTimeoutInterval: 5 * time.Minute,
		})
		next := d.NextPhaseTimeoutTime()
		if time.Until(next) > 1*time.Second {
			t.Error("expected next time to be approximately now for first run")
		}
	})
}
