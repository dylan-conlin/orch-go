package orient

import (
	"encoding/json"
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

func TestThroughputFromEvents_DurationSeconds(t *testing.T) {
	now := time.Now()
	oneDayAgo := now.Add(-12 * time.Hour)

	// Test with duration_seconds (current event format from LogAgentCompleted)
	events := []Event{
		{Type: "session.spawned", Timestamp: oneDayAgo.Unix()},
		{Type: "session.spawned", Timestamp: oneDayAgo.Unix()},
		{Type: "agent.completed", Timestamp: now.Unix(), Data: map[string]interface{}{
			"duration_seconds": 1800.0, // 30 minutes
		}},
		{Type: "agent.completed", Timestamp: now.Unix(), Data: map[string]interface{}{
			"duration_seconds": 2760.0, // 46 minutes
		}},
	}

	tp := ComputeThroughput(events, now, 1)

	if tp.Spawns != 2 {
		t.Errorf("expected 2 spawns, got %d", tp.Spawns)
	}
	if tp.Completions != 2 {
		t.Errorf("expected 2 completions, got %d", tp.Completions)
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
	if tp.Days != 1 {
		t.Errorf("expected Days=1, got %d", tp.Days)
	}

	// 3-day window includes event exactly at boundary (cutoff uses strict less-than)
	tp3 := ComputeThroughput(events, now, 3)
	if tp3.Spawns != 2 {
		t.Errorf("expected 2 spawns in 3-day window (72h event is at boundary, included), got %d", tp3.Spawns)
	}

	// 2-day window should exclude 72h-old event
	tp2 := ComputeThroughput(events, now, 2)
	if tp2.Spawns != 1 {
		t.Errorf("expected 1 spawn in 2-day window, got %d", tp2.Spawns)
	}
	if tp2.Days != 2 {
		t.Errorf("expected Days=2, got %d", tp2.Days)
	}
}

func TestFormatThroughput_DaysHeader(t *testing.T) {
	tests := []struct {
		days     int
		expected string
	}{
		{1, "Last 24h:"},
		{3, "Last 3d:"},
		{7, "Last 7d:"},
	}

	for _, tc := range tests {
		data := &OrientationData{
			Throughput: Throughput{Days: tc.days},
		}
		output := FormatOrientation(data)
		if !strings.Contains(output, tc.expected) {
			t.Errorf("days=%d: expected %q in output, got:\n%s", tc.days, tc.expected, output)
		}
	}
}

func TestFormatOrientation(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{
			Days:           1,
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
	if !strings.Contains(output, "Last 24h:") {
		t.Error("missing 'Last 24h:' header for days=1")
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

func TestOrientationDataJSON(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{
			Days:           1,
			Completions:    5,
			Abandonments:   2,
			InProgress:     3,
			AvgDurationMin: 25,
		},
		RelevantModels: []ModelFreshness{
			{Name: "test-model", Summary: "A test model.", AgeDays: 1, HasRecentProbes: true},
		},
		StaleModels: []ModelFreshness{
			{Name: "old-model", AgeDays: 30, HasRecentProbes: false},
		},
		FocusGoal: "Ship orient",
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal OrientationData: %v", err)
	}

	// Verify key JSON fields exist
	jsonStr := string(b)
	for _, key := range []string{
		`"completions":5`,
		`"abandonments":2`,
		`"in_progress":3`,
		`"avg_duration_min":25`,
		`"focus_goal":"Ship orient"`,
		`"name":"test-model"`,
		`"name":"old-model"`,
		`"age_days":30`,
		`"has_recent_probes":true`,
	} {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("JSON missing expected key %q in:\n%s", key, jsonStr)
		}
	}

	// Verify ready_issues is omitted when nil
	if strings.Contains(jsonStr, "ready_issues") {
		t.Error("ready_issues should be omitted when nil")
	}

	// Round-trip: unmarshal back
	var decoded OrientationData
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.Throughput.Completions != 5 {
		t.Errorf("round-trip: expected completions=5, got %d", decoded.Throughput.Completions)
	}
	if decoded.FocusGoal != "Ship orient" {
		t.Errorf("round-trip: expected focus 'Ship orient', got %q", decoded.FocusGoal)
	}
	if len(decoded.RelevantModels) != 1 || decoded.RelevantModels[0].Name != "test-model" {
		t.Error("round-trip: relevant models mismatch")
	}
}

func TestOrientationDataJSON_SkipReady(t *testing.T) {
	// Simulates --skip-ready: ReadyIssues is nil
	data := &OrientationData{
		Throughput: Throughput{Days: 1, Completions: 3},
		FocusGoal:  "Test",
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(b)
	if strings.Contains(jsonStr, "ready_issues") {
		t.Error("ready_issues should be omitted with skip-ready")
	}
	if !strings.Contains(jsonStr, `"completions":3`) {
		t.Error("throughput should still be present")
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
