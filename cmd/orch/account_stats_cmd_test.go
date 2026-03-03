package main

import (
	"testing"
	"time"
)

func TestAggregateAccountStats(t *testing.T) {
	now := time.Now().Unix()
	sevenDaysAgo := now - 7*24*3600

	tests := []struct {
		name     string
		events   []StatsEvent
		days     int
		wantAccs []string // expected account names in output
		wantTotal int
	}{
		{
			name: "groups by account name",
			events: []StatsEvent{
				{Type: "session.spawned", Timestamp: now - 3600, Data: map[string]interface{}{"account": "personal", "skill": "feature-impl"}},
				{Type: "session.spawned", Timestamp: now - 7200, Data: map[string]interface{}{"account": "personal", "skill": "investigation"}},
				{Type: "session.spawned", Timestamp: now - 1800, Data: map[string]interface{}{"account": "work", "skill": "feature-impl"}},
			},
			days:      7,
			wantAccs:  []string{"personal", "work"},
			wantTotal: 3,
		},
		{
			name: "filters by time window",
			events: []StatsEvent{
				{Type: "session.spawned", Timestamp: now - 3600, Data: map[string]interface{}{"account": "personal"}},
				{Type: "session.spawned", Timestamp: sevenDaysAgo - 3600, Data: map[string]interface{}{"account": "personal"}}, // too old
			},
			days:      7,
			wantAccs:  []string{"personal"},
			wantTotal: 1,
		},
		{
			name: "events without account field counted as unknown",
			events: []StatsEvent{
				{Type: "session.spawned", Timestamp: now - 3600, Data: map[string]interface{}{"skill": "feature-impl"}},
				{Type: "session.spawned", Timestamp: now - 1800, Data: map[string]interface{}{"account": "personal"}},
			},
			days:      7,
			wantAccs:  []string{"personal", "(unknown)"},
			wantTotal: 2,
		},
		{
			name: "ignores non-spawn events",
			events: []StatsEvent{
				{Type: "session.spawned", Timestamp: now - 3600, Data: map[string]interface{}{"account": "personal"}},
				{Type: "agent.completed", Timestamp: now - 1800, Data: map[string]interface{}{"account": "personal"}},
			},
			days:      7,
			wantAccs:  []string{"personal"},
			wantTotal: 1,
		},
		{
			name:      "empty events",
			events:    []StatsEvent{},
			days:      7,
			wantAccs:  nil,
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := aggregateAccountStats(tt.events, tt.days)

			if stats.Total != tt.wantTotal {
				t.Errorf("total = %d, want %d", stats.Total, tt.wantTotal)
			}

			if tt.wantAccs == nil {
				if len(stats.Accounts) != 0 {
					t.Errorf("expected no accounts, got %d", len(stats.Accounts))
				}
				return
			}

			gotAccs := make(map[string]bool)
			for _, a := range stats.Accounts {
				gotAccs[a.Name] = true
			}

			for _, want := range tt.wantAccs {
				if !gotAccs[want] {
					t.Errorf("missing account %q in results", want)
				}
			}
		})
	}
}

func TestAccountStatsPercentages(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now - 3600, Data: map[string]interface{}{"account": "personal"}},
		{Type: "session.spawned", Timestamp: now - 3500, Data: map[string]interface{}{"account": "personal"}},
		{Type: "session.spawned", Timestamp: now - 3400, Data: map[string]interface{}{"account": "personal"}},
		{Type: "session.spawned", Timestamp: now - 1800, Data: map[string]interface{}{"account": "work"}},
	}

	stats := aggregateAccountStats(events, 7)

	if stats.Total != 4 {
		t.Fatalf("total = %d, want 4", stats.Total)
	}

	for _, a := range stats.Accounts {
		switch a.Name {
		case "personal":
			if a.Count != 3 {
				t.Errorf("personal count = %d, want 3", a.Count)
			}
			if a.Percent < 74 || a.Percent > 76 {
				t.Errorf("personal percent = %.1f, want ~75", a.Percent)
			}
		case "work":
			if a.Count != 1 {
				t.Errorf("work count = %d, want 1", a.Count)
			}
			if a.Percent < 24 || a.Percent > 26 {
				t.Errorf("work percent = %.1f, want ~25", a.Percent)
			}
		default:
			t.Errorf("unexpected account %q", a.Name)
		}
	}
}

func TestAccountStatsSortedByCount(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now - 3600, Data: map[string]interface{}{"account": "work"}},
		{Type: "session.spawned", Timestamp: now - 3500, Data: map[string]interface{}{"account": "personal"}},
		{Type: "session.spawned", Timestamp: now - 3400, Data: map[string]interface{}{"account": "personal"}},
		{Type: "session.spawned", Timestamp: now - 3300, Data: map[string]interface{}{"account": "personal"}},
	}

	stats := aggregateAccountStats(events, 7)

	if len(stats.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(stats.Accounts))
	}

	// personal (3) should come before work (1)
	if stats.Accounts[0].Name != "personal" {
		t.Errorf("first account = %q, want personal", stats.Accounts[0].Name)
	}
	if stats.Accounts[1].Name != "work" {
		t.Errorf("second account = %q, want work", stats.Accounts[1].Name)
	}
}
