package orient

import (
	"strings"
	"testing"
	"time"
)

func TestThroughputFromEvents(t *testing.T) {
	now := time.Now()
	oneDayAgo := now.Add(-12 * time.Hour)

	events := []Event{
		{Type: "session.spawned", Timestamp: oneDayAgo.Unix()},
		{Type: "session.spawned", Timestamp: oneDayAgo.Unix()},
		{Type: "session.spawned", Timestamp: oneDayAgo.Unix()},
		{Type: "agent.completed", Timestamp: now.Unix(), Data: map[string]interface{}{
			"duration_minutes": 30.0,
		}},
		{Type: "agent.completed", Timestamp: now.Unix(), Data: map[string]interface{}{
			"duration_minutes": 46.0,
		}},
		{Type: "agent.abandoned", Timestamp: now.Unix()},
	}

	tp := ComputeThroughput(events, now, 1)

	if tp.Spawns != 3 {
		t.Errorf("expected 3 spawns, got %d", tp.Spawns)
	}
	if tp.Completions != 2 {
		t.Errorf("expected 2 completions, got %d", tp.Completions)
	}
	if tp.Abandonments != 1 {
		t.Errorf("expected 1 abandonment, got %d", tp.Abandonments)
	}
	if tp.AvgDurationMin != 38 {
		t.Errorf("expected avg duration 38 min, got %d", tp.AvgDurationMin)
	}
}

func TestThroughputFromEvents_Empty(t *testing.T) {
	tp := ComputeThroughput(nil, time.Now(), 1)
	if tp.Spawns != 0 || tp.Completions != 0 || tp.Abandonments != 0 {
		t.Errorf("expected all zeros for empty events")
	}
}

func TestThroughputFromEvents_FiltersByDays(t *testing.T) {
	now := time.Now()
	threeDaysAgo := now.Add(-72 * time.Hour)

	events := []Event{
		{Type: "session.spawned", Timestamp: threeDaysAgo.Unix()},
		{Type: "session.spawned", Timestamp: now.Unix()},
	}

	tp := ComputeThroughput(events, now, 1)
	if tp.Spawns != 1 {
		t.Errorf("expected 1 spawn in 1-day window, got %d", tp.Spawns)
	}
}

func TestFormatOrientation(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{
			Spawns:         3,
			Completions:    2,
			Abandonments:   1,
			InProgress:     3,
			AvgDurationMin: 38,
		},
		ReadyIssues: []ReadyIssue{
			{ID: "orch-go-abc1", Title: "Fix spawn bug", Priority: "P1"},
			{ID: "orch-go-def2", Title: "Add model drift", Priority: "P2"},
		},
		RelevantModels: []ModelFreshness{
			{Name: "spawn-architecture", Summary: "Spawn uses dual modes.", AgeDays: 2},
			{Name: "completion-verification", Summary: "Verify before close.", AgeDays: 1},
		},
		StaleModels: []ModelFreshness{
			{Name: "coaching-plugin", AgeDays: 33, HasRecentProbes: false},
		},
		FocusGoal: "Ship orient command",
	}

	output := FormatOrientation(data)

	// Check all sections present
	if !strings.Contains(output, "SESSION ORIENTATION") {
		t.Error("missing SESSION ORIENTATION header")
	}
	if !strings.Contains(output, "Completions: 2") {
		t.Error("missing completions count")
	}
	if !strings.Contains(output, "Abandonments: 1") {
		t.Error("missing abandonments count")
	}
	if !strings.Contains(output, "In-progress: 3") {
		t.Error("missing in-progress count")
	}
	if !strings.Contains(output, "38 min") {
		t.Error("missing avg duration")
	}
	if !strings.Contains(output, "orch-go-abc1") {
		t.Error("missing ready issue ID")
	}
	if !strings.Contains(output, "Fix spawn bug") {
		t.Error("missing ready issue title")
	}
	if !strings.Contains(output, "spawn-architecture") {
		t.Error("missing relevant model name")
	}
	if !strings.Contains(output, "Spawn uses dual modes.") {
		t.Error("missing model summary")
	}
	if !strings.Contains(output, "coaching-plugin") {
		t.Error("missing stale model name")
	}
	if !strings.Contains(output, "33d ago") {
		t.Error("missing stale model age")
	}
	if !strings.Contains(output, "Ship orient command") {
		t.Error("missing focus goal")
	}
}

func TestFormatOrientation_NoFocus(t *testing.T) {
	data := &OrientationData{
		Throughput:     Throughput{},
		ReadyIssues:    nil,
		RelevantModels: nil,
		StaleModels:    nil,
		FocusGoal:      "",
	}

	output := FormatOrientation(data)

	// Focus section should not appear
	if strings.Contains(output, "Focus") {
		t.Error("focus section should not appear when no focus set")
	}
	// Should still have header
	if !strings.Contains(output, "SESSION ORIENTATION") {
		t.Error("missing SESSION ORIENTATION header")
	}
}

func TestFormatOrientation_NoReadyIssues(t *testing.T) {
	data := &OrientationData{
		Throughput:  Throughput{Spawns: 1, Completions: 1},
		ReadyIssues: nil,
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "No issues ready") {
		t.Error("should indicate no ready issues")
	}
}

func TestFormatOrientation_NoStaleModels(t *testing.T) {
	data := &OrientationData{
		Throughput:  Throughput{},
		StaleModels: nil,
	}

	output := FormatOrientation(data)

	// Stale models section should not appear when empty
	if strings.Contains(output, "Stale models") {
		t.Error("stale models section should not appear when none are stale")
	}
}

func TestTruncateSummary(t *testing.T) {
	short := "Short summary."
	if got := truncateSummary(short, 100); got != short {
		t.Errorf("truncateSummary should not truncate short text, got %q", got)
	}

	long := strings.Repeat("word ", 100) // 500 chars
	got := truncateSummary(long, 100)
	if len(got) > 103 { // 100 + "..."
		t.Errorf("truncateSummary should truncate to ~100 chars, got %d chars", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("truncated summary should end with ...")
	}
}
