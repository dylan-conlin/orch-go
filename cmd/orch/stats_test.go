package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseEvents(t *testing.T) {
	// Create a temporary events file
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")

	// Create test events - timestamps within last 7 days
	now := time.Now().Unix()
	events := []string{
		`{"type":"session.spawned","session_id":"ses_1","timestamp":` + itoa(now-3600) + `,"data":{"skill":"feature-impl","beads_id":"test-1"}}`,
		`{"type":"session.spawned","session_id":"ses_2","timestamp":` + itoa(now-7200) + `,"data":{"skill":"investigation","beads_id":"test-2"}}`,
		`{"type":"agent.completed","timestamp":` + itoa(now-1800) + `,"data":{"beads_id":"test-1"}}`,
		`{"type":"agent.abandoned","timestamp":` + itoa(now-600) + `,"data":{"beads_id":"test-2"}}`,
		`{"type":"daemon.spawn","timestamp":` + itoa(now-300) + `,"data":{"beads_id":"test-3","skill":"feature-impl"}}`,
	}

	content := ""
	for _, e := range events {
		content += e + "\n"
	}

	if err := os.WriteFile(eventsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test events: %v", err)
	}

	// Parse events
	parsed, err := parseEvents(eventsPath, 7)
	if err != nil {
		t.Fatalf("parseEvents failed: %v", err)
	}

	if len(parsed) != 5 {
		t.Errorf("expected 5 events, got %d", len(parsed))
	}
}

func TestParseEventsTimeFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")

	now := time.Now().Unix()
	oldTimestamp := now - (10 * 24 * 60 * 60) // 10 days ago

	events := []string{
		`{"type":"session.spawned","session_id":"ses_1","timestamp":` + itoa(now-3600) + `,"data":{"skill":"feature-impl"}}`,
		`{"type":"session.spawned","session_id":"ses_2","timestamp":` + itoa(oldTimestamp) + `,"data":{"skill":"investigation"}}`,
	}

	content := ""
	for _, e := range events {
		content += e + "\n"
	}

	if err := os.WriteFile(eventsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test events: %v", err)
	}

	// Parse with 7 day window - should only get 1 event
	parsed, err := parseEvents(eventsPath, 7)
	if err != nil {
		t.Fatalf("parseEvents failed: %v", err)
	}

	if len(parsed) != 1 {
		t.Errorf("expected 1 event (recent only), got %d", len(parsed))
	}
}

func TestAggregateStats(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "test-1"}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 3600, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "test-2"}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 1800, Data: map[string]interface{}{"skill": "investigation", "beads_id": "test-3"}},
		{Type: "agent.completed", Timestamp: now - 6000, Data: map[string]interface{}{"beads_id": "test-1"}},
		{Type: "agent.completed", Timestamp: now - 2400, Data: map[string]interface{}{"beads_id": "test-2"}},
		{Type: "agent.abandoned", Timestamp: now - 600, Data: map[string]interface{}{"beads_id": "test-3"}},
		{Type: "daemon.spawn", Timestamp: now - 500, Data: map[string]interface{}{}},
		{Type: "agent.wait.complete", Timestamp: now - 400, Data: map[string]interface{}{}},
		{Type: "agent.wait.timeout", Timestamp: now - 300, Data: map[string]interface{}{}},
	}

	report := aggregateStats(events, 7)

	// Test core metrics
	if report.Summary.TotalSpawns != 3 {
		t.Errorf("expected 3 spawns, got %d", report.Summary.TotalSpawns)
	}

	if report.Summary.TotalCompletions != 2 {
		t.Errorf("expected 2 completions, got %d", report.Summary.TotalCompletions)
	}

	if report.Summary.TotalAbandonments != 1 {
		t.Errorf("expected 1 abandonment, got %d", report.Summary.TotalAbandonments)
	}

	// Test completion rate calculation
	expectedRate := (2.0 / 3.0) * 100
	if report.Summary.CompletionRate < expectedRate-0.1 || report.Summary.CompletionRate > expectedRate+0.1 {
		t.Errorf("expected completion rate ~%.1f%%, got %.1f%%", expectedRate, report.Summary.CompletionRate)
	}

	// Test daemon stats
	if report.DaemonStats.DaemonSpawns != 1 {
		t.Errorf("expected 1 daemon spawn, got %d", report.DaemonStats.DaemonSpawns)
	}

	// Test wait stats
	if report.WaitStats.WaitCompleted != 1 {
		t.Errorf("expected 1 wait completed, got %d", report.WaitStats.WaitCompleted)
	}

	if report.WaitStats.WaitTimeouts != 1 {
		t.Errorf("expected 1 wait timeout, got %d", report.WaitStats.WaitTimeouts)
	}

	// Test skill stats
	if len(report.SkillStats) != 2 {
		t.Errorf("expected 2 skills, got %d", len(report.SkillStats))
	}

	// Find feature-impl skill stats
	var featureImplStats *SkillStatsSummary
	for i := range report.SkillStats {
		if report.SkillStats[i].Skill == "feature-impl" {
			featureImplStats = &report.SkillStats[i]
			break
		}
	}

	if featureImplStats == nil {
		t.Fatal("feature-impl skill not found in stats")
	}

	if featureImplStats.Spawns != 2 {
		t.Errorf("expected 2 feature-impl spawns, got %d", featureImplStats.Spawns)
	}

	if featureImplStats.Completions != 2 {
		t.Errorf("expected 2 feature-impl completions, got %d", featureImplStats.Completions)
	}
}

func TestAggregateStatsEmptyEvents(t *testing.T) {
	events := []StatsEvent{}
	report := aggregateStats(events, 7)

	if report.Summary.TotalSpawns != 0 {
		t.Errorf("expected 0 spawns, got %d", report.Summary.TotalSpawns)
	}

	if report.Summary.CompletionRate != 0 {
		t.Errorf("expected 0%% completion rate, got %.1f%%", report.Summary.CompletionRate)
	}
}

func TestTruncateSkill(t *testing.T) {
	tests := []struct {
		skill  string
		maxLen int
		want   string
	}{
		{"feature-impl", 25, "feature-impl"},
		{"a-very-long-skill-name-here", 15, "a-very-long-..."},
		{"short", 10, "short"},
	}

	for _, tt := range tests {
		got := truncateSkill(tt.skill, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateSkill(%q, %d) = %q, want %q", tt.skill, tt.maxLen, got, tt.want)
		}
	}
}

func TestParseEventsFileNotFound(t *testing.T) {
	_, err := parseEvents("/nonexistent/path/events.jsonl", 7)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// Helper function to convert int64 to string
func itoa(n int64) string {
	return fmt.Sprintf("%d", n)
}

func TestGetSkillCategory(t *testing.T) {
	tests := []struct {
		skill    string
		expected SkillCategory
	}{
		{"feature-impl", TaskSkill},
		{"investigation", TaskSkill},
		{"systematic-debugging", TaskSkill},
		{"architect", TaskSkill},
		{"orchestrator", CoordinationSkill},
		{"meta-orchestrator", CoordinationSkill},
		{"unknown-skill", TaskSkill}, // default to task
	}

	for _, tt := range tests {
		got := getSkillCategory(tt.skill)
		if got != tt.expected {
			t.Errorf("getSkillCategory(%q) = %q, want %q", tt.skill, got, tt.expected)
		}
	}
}

func TestAggregateStatsCategoryBreakdown(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Task skills (should count toward TaskCompletionRate)
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "test-1"}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 3600, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "test-2"}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 2400, Data: map[string]interface{}{"skill": "investigation", "beads_id": "test-3"}},
		// Coordination skills (should count toward CoordinationCompletionRate)
		{Type: "session.spawned", SessionID: "ses_4", Timestamp: now - 1800, Data: map[string]interface{}{"skill": "orchestrator", "beads_id": "test-4"}},
		{Type: "session.spawned", SessionID: "ses_5", Timestamp: now - 1200, Data: map[string]interface{}{"skill": "meta-orchestrator", "beads_id": "test-5"}},
		// Completions - 2 task completions, 0 coordination completions
		{Type: "agent.completed", Timestamp: now - 6000, Data: map[string]interface{}{"beads_id": "test-1"}},
		{Type: "agent.completed", Timestamp: now - 5000, Data: map[string]interface{}{"beads_id": "test-2"}},
		// Abandonments - 1 task abandonment
		{Type: "agent.abandoned", Timestamp: now - 600, Data: map[string]interface{}{"beads_id": "test-3"}},
	}

	report := aggregateStats(events, 7)

	// Test task skill metrics
	if report.Summary.TaskSpawns != 3 {
		t.Errorf("expected 3 task spawns, got %d", report.Summary.TaskSpawns)
	}
	if report.Summary.TaskCompletions != 2 {
		t.Errorf("expected 2 task completions, got %d", report.Summary.TaskCompletions)
	}
	expectedTaskRate := (2.0 / 3.0) * 100
	if report.Summary.TaskCompletionRate < expectedTaskRate-0.1 || report.Summary.TaskCompletionRate > expectedTaskRate+0.1 {
		t.Errorf("expected task completion rate ~%.1f%%, got %.1f%%", expectedTaskRate, report.Summary.TaskCompletionRate)
	}

	// Test coordination skill metrics
	if report.Summary.CoordinationSpawns != 2 {
		t.Errorf("expected 2 coordination spawns, got %d", report.Summary.CoordinationSpawns)
	}
	if report.Summary.CoordinationCompletions != 0 {
		t.Errorf("expected 0 coordination completions, got %d", report.Summary.CoordinationCompletions)
	}
	if report.Summary.CoordinationCompletionRate != 0 {
		t.Errorf("expected 0%% coordination completion rate, got %.1f%%", report.Summary.CoordinationCompletionRate)
	}

	// Verify skill categories are set correctly in SkillStats
	for _, skill := range report.SkillStats {
		expected := getSkillCategory(skill.Skill)
		if skill.Category != expected {
			t.Errorf("skill %q has category %q, want %q", skill.Skill, skill.Category, expected)
		}
	}
}

func TestAggregateStatsCoordinationExcludedFromOverallRate(t *testing.T) {
	now := time.Now().Unix()

	// Scenario: All coordination skills with 0% completion, all task skills with 100% completion
	// Overall rate will be 50%, but TaskCompletionRate should be 100%
	events := []StatsEvent{
		// 2 task spawns, both complete
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "test-1"}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 3600, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "test-2"}},
		{Type: "agent.completed", Timestamp: now - 6000, Data: map[string]interface{}{"beads_id": "test-1"}},
		{Type: "agent.completed", Timestamp: now - 5000, Data: map[string]interface{}{"beads_id": "test-2"}},
		// 2 coordination spawns, none complete (as expected for interactive sessions)
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 1800, Data: map[string]interface{}{"skill": "orchestrator", "beads_id": "test-3"}},
		{Type: "session.spawned", SessionID: "ses_4", Timestamp: now - 1200, Data: map[string]interface{}{"skill": "meta-orchestrator", "beads_id": "test-4"}},
	}

	report := aggregateStats(events, 7)

	// Overall rate includes coordination skills (2/4 = 50%)
	expectedOverall := 50.0
	if report.Summary.CompletionRate < expectedOverall-0.1 || report.Summary.CompletionRate > expectedOverall+0.1 {
		t.Errorf("expected overall completion rate ~%.1f%%, got %.1f%%", expectedOverall, report.Summary.CompletionRate)
	}

	// Task rate excludes coordination skills (2/2 = 100%)
	expectedTask := 100.0
	if report.Summary.TaskCompletionRate < expectedTask-0.1 || report.Summary.TaskCompletionRate > expectedTask+0.1 {
		t.Errorf("expected task completion rate ~%.1f%%, got %.1f%%", expectedTask, report.Summary.TaskCompletionRate)
	}

	// Coordination rate is 0% (0/2)
	if report.Summary.CoordinationCompletionRate != 0 {
		t.Errorf("expected coordination completion rate 0%%, got %.1f%%", report.Summary.CoordinationCompletionRate)
	}
}
