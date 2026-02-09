package main

import (
	"math"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/action"
)

func TestComputeDiscoveredWorkStatsCountsWorkerAndOrchestratorCreates(t *testing.T) {
	now := time.Now().Unix()
	events := []StatsEvent{
		{
			Type:      "session.spawned",
			SessionID: "worker-1",
			Timestamp: now - 300,
			Data: map[string]interface{}{
				"skill":    "feature-impl",
				"beads_id": "orch-go-111",
			},
		},
		{
			Type:      "session.spawned",
			SessionID: "worker-2",
			Timestamp: now - 250,
			Data: map[string]interface{}{
				"skill":    "investigation",
				"beads_id": "orch-go-222",
			},
		},
		{
			Type:      "session.spawned",
			SessionID: "orch-1",
			Timestamp: now - 200,
			Data: map[string]interface{}{
				"skill":    "orchestrator",
				"beads_id": "open",
			},
		},
	}

	actionEvents := []action.ActionEvent{
		{Timestamp: time.Unix(now-150, 0), Tool: "Bash", Outcome: action.OutcomeSuccess, Target: `bd create "A"`, SessionID: "worker-1"},
		{Timestamp: time.Unix(now-140, 0), Tool: "Bash", Outcome: action.OutcomeSuccess, Target: `cd /tmp && bd create "B"`, SessionID: "worker-1"},
		{Timestamp: time.Unix(now-130, 0), Tool: "Bash", Outcome: action.OutcomeSuccess, Target: `cd /tmp && bd create "C"`, SessionID: "worker-2"},
		{Timestamp: time.Unix(now-120, 0), Tool: "Bash", Outcome: action.OutcomeSuccess, Target: `bd create "D"`, SessionID: "orch-1"},
		{Timestamp: time.Unix(now-110, 0), Tool: "Bash", Outcome: action.OutcomeSuccess, Target: `bd create "E"`, SessionID: "unknown"},
		{Timestamp: time.Unix(now-105, 0), Tool: "Bash", Outcome: action.OutcomeSuccess, Target: `grep -c "bd create" ~/.orch/action-log.jsonl`, SessionID: "worker-2"},
		{Timestamp: time.Unix(now-100, 0), Tool: "Bash", Outcome: action.OutcomeError, Target: `bd create "F"`, SessionID: "worker-2"},
	}

	stats := computeDiscoveredWorkStats(events, actionEvents, 7, false)

	if stats.WorkerSessions != 2 {
		t.Fatalf("expected 2 worker sessions, got %d", stats.WorkerSessions)
	}
	if stats.WorkerSessionsWithIssueCreation != 2 {
		t.Fatalf("expected 2 worker creator sessions, got %d", stats.WorkerSessionsWithIssueCreation)
	}
	if stats.WorkerIssuesCreated != 3 {
		t.Fatalf("expected 3 worker issues created, got %d", stats.WorkerIssuesCreated)
	}
	if stats.OrchestratorIssuesCreated != 2 {
		t.Fatalf("expected 2 orchestrator issues created, got %d", stats.OrchestratorIssuesCreated)
	}
	if math.Abs(stats.WorkerIssueCreationRate-100.0) > 0.0001 {
		t.Fatalf("expected worker issue creation rate 100.0, got %.4f", stats.WorkerIssueCreationRate)
	}
	if math.Abs(stats.WorkerIssueShare-60.0) > 0.0001 {
		t.Fatalf("expected worker issue share 60.0, got %.4f", stats.WorkerIssueShare)
	}
}

func TestComputeDiscoveredWorkStatsExcludeUntrackedByDefault(t *testing.T) {
	now := time.Now().Unix()
	events := []StatsEvent{
		{
			Type:      "session.spawned",
			SessionID: "worker-tracked",
			Timestamp: now - 300,
			Data: map[string]interface{}{
				"skill":    "feature-impl",
				"beads_id": "orch-go-111",
			},
		},
		{
			Type:      "session.spawned",
			SessionID: "worker-untracked",
			Timestamp: now - 250,
			Data: map[string]interface{}{
				"skill":    "feature-impl",
				"beads_id": "orch-go-untracked-222",
			},
		},
	}

	actionEvents := []action.ActionEvent{
		{Timestamp: time.Unix(now-100, 0), Tool: "Bash", Outcome: action.OutcomeSuccess, Target: `bd create "A"`, SessionID: "worker-untracked"},
	}

	excluded := computeDiscoveredWorkStats(events, actionEvents, 7, false)
	if excluded.WorkerSessions != 1 {
		t.Fatalf("expected 1 tracked worker session, got %d", excluded.WorkerSessions)
	}
	if excluded.WorkerIssuesCreated != 0 {
		t.Fatalf("expected 0 worker issues when untracked excluded, got %d", excluded.WorkerIssuesCreated)
	}
	if excluded.OrchestratorIssuesCreated != 1 {
		t.Fatalf("expected 1 orchestrator/unknown issue when untracked excluded, got %d", excluded.OrchestratorIssuesCreated)
	}

	included := computeDiscoveredWorkStats(events, actionEvents, 7, true)
	if included.WorkerSessions != 2 {
		t.Fatalf("expected 2 worker sessions when including untracked, got %d", included.WorkerSessions)
	}
	if included.WorkerIssuesCreated != 1 {
		t.Fatalf("expected 1 worker issue when including untracked, got %d", included.WorkerIssuesCreated)
	}
	if included.OrchestratorIssuesCreated != 0 {
		t.Fatalf("expected 0 orchestrator/unknown issues when including untracked, got %d", included.OrchestratorIssuesCreated)
	}
}
