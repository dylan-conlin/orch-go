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

	// Parse events (parseEvents now returns all events, filtering happens in aggregateStats)
	parsed, err := parseEvents(eventsPath)
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
		`{"type":"session.spawned","session_id":"ses_1","timestamp":` + itoa(now-3600) + `,"data":{"skill":"feature-impl","beads_id":"test-1"}}`,
		`{"type":"session.spawned","session_id":"ses_2","timestamp":` + itoa(oldTimestamp) + `,"data":{"skill":"investigation","beads_id":"test-2"}}`,
	}

	content := ""
	for _, e := range events {
		content += e + "\n"
	}

	if err := os.WriteFile(eventsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test events: %v", err)
	}

	// parseEvents now returns all events (time filtering happens in aggregateStats)
	parsed, err := parseEvents(eventsPath)
	if err != nil {
		t.Fatalf("parseEvents failed: %v", err)
	}

	// parseEvents returns ALL events now
	if len(parsed) != 2 {
		t.Errorf("expected 2 events (all events), got %d", len(parsed))
	}

	// Time filtering happens in aggregateStats - verify that only recent events are counted
	report := aggregateStats(parsed, 7, true)
	if report.Summary.TotalSpawns != 1 {
		t.Errorf("expected 1 spawn in 7-day window, got %d", report.Summary.TotalSpawns)
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

	report := aggregateStats(events, 7, true) // includeUntracked=true for backwards compat

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
	report := aggregateStats(events, 7, true)

	if report.Summary.TotalSpawns != 0 {
		t.Errorf("expected 0 spawns, got %d", report.Summary.TotalSpawns)
	}

	if report.Summary.CompletionRate != 0 {
		t.Errorf("expected 0%% completion rate, got %.1f%%", report.Summary.CompletionRate)
	}
}

func TestAggregateStatsEscapeHatch(t *testing.T) {
	now := time.Now().Unix()
	sevenDaysAgo := now - (7 * 24 * 60 * 60) + 3600   // 7 days ago + 1 hour (within 7d window)
	thirtyDaysAgo := now - (25 * 24 * 60 * 60)        // 25 days ago (within 30d, outside 7d)
	veryOld := now - (60 * 24 * 60 * 60)              // 60 days ago (outside 30d)

	events := []StatsEvent{
		// Recent escape hatch spawn with account (within 7d)
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 3600, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1", "spawn_mode": "claude", "usage_account": "account1@example.com",
		}},
		// Older escape hatch spawn with different account (within 30d, outside 7d)
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: thirtyDaysAgo, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-2", "spawn_mode": "claude", "usage_account": "account2@example.com",
		}},
		// Very old escape hatch spawn (outside 30d)
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: veryOld, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-3", "spawn_mode": "claude", "usage_account": "account1@example.com",
		}},
		// Recent regular spawn (not escape hatch)
		{Type: "session.spawned", SessionID: "ses_4", Timestamp: now - 1800, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-4", "spawn_mode": "headless",
		}},
		// Escape hatch without account info
		{Type: "session.spawned", SessionID: "ses_5", Timestamp: sevenDaysAgo, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "test-5", "spawn_mode": "claude",
		}},
	}

	report := aggregateStats(events, 7, true)

	// Test escape hatch totals
	if report.EscapeHatchStats.TotalSpawns != 4 { // All 4 escape hatch spawns
		t.Errorf("expected 4 total escape hatch spawns, got %d", report.EscapeHatchStats.TotalSpawns)
	}

	if report.EscapeHatchStats.Last7DaySpawns != 2 { // ses_1 and ses_5
		t.Errorf("expected 2 escape hatch spawns in last 7d, got %d", report.EscapeHatchStats.Last7DaySpawns)
	}

	if report.EscapeHatchStats.Last30DaySpawns != 3 { // ses_1, ses_2, ses_5
		t.Errorf("expected 3 escape hatch spawns in last 30d, got %d", report.EscapeHatchStats.Last30DaySpawns)
	}

	// Test escape hatch rate (in 7-day window: 2 escape hatch out of 3 spawns)
	// Note: ses_2, ses_3, ses_5 are outside the 7d window but escape hatch still counts them for rate
	// Actually: within 7d window we have ses_1, ses_4, ses_5 (3 total), with ses_1 and ses_5 being escape hatch
	// So rate should be 2/3 = 66.67%
	expectedRate := (2.0 / 3.0) * 100
	if report.EscapeHatchStats.EscapeHatchRate < expectedRate-1 || report.EscapeHatchStats.EscapeHatchRate > expectedRate+1 {
		t.Errorf("expected escape hatch rate ~%.1f%%, got %.1f%%", expectedRate, report.EscapeHatchStats.EscapeHatchRate)
	}

	// Test account breakdown
	if len(report.EscapeHatchStats.ByAccount) < 2 {
		t.Errorf("expected at least 2 accounts in breakdown, got %d", len(report.EscapeHatchStats.ByAccount))
	}

	// Verify account1 has 2 spawns total (ses_1 in 7d, ses_3 outside 30d)
	var account1Found bool
	for _, acct := range report.EscapeHatchStats.ByAccount {
		if acct.Account == "account1@example.com" {
			account1Found = true
			if acct.TotalSpawns != 2 {
				t.Errorf("expected account1 to have 2 total spawns, got %d", acct.TotalSpawns)
			}
			if acct.Last7Days != 1 {
				t.Errorf("expected account1 to have 1 spawn in last 7d, got %d", acct.Last7Days)
			}
		}
	}
	if !account1Found {
		t.Error("account1@example.com not found in breakdown")
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
	_, err := parseEvents("/nonexistent/path/events.jsonl")
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

	report := aggregateStats(events, 7, true) // includeUntracked=true for backwards compat

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

	report := aggregateStats(events, 7, true) // includeUntracked=true for backwards compat

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

func TestIsUntrackedSpawn(t *testing.T) {
	tests := []struct {
		beadsID  string
		expected bool
	}{
		{"orch-go-abc123", false},
		{"orch-go-untracked-abc123", true},
		{"test-untracked-xyz", true},
		{"untracked", true},
		{"", false},
		{"my-feature-impl-task", false},
	}

	for _, tt := range tests {
		got := isUntrackedSpawn(tt.beadsID)
		if got != tt.expected {
			t.Errorf("isUntrackedSpawn(%q) = %v, want %v", tt.beadsID, got, tt.expected)
		}
	}
}

func TestAggregateStatsUntrackedExclusion(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Tracked spawns (should always count)
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "orch-go-abc123"}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 3600, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "orch-go-def456"}},
		// Untracked spawns (should be excluded by default)
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 1800, Data: map[string]interface{}{"skill": "investigation", "beads_id": "orch-go-untracked-ghi789"}},
		{Type: "session.spawned", SessionID: "ses_4", Timestamp: now - 1200, Data: map[string]interface{}{"skill": "investigation", "beads_id": "test-untracked-jkl012"}},
		// Completions
		{Type: "agent.completed", Timestamp: now - 6000, Data: map[string]interface{}{"beads_id": "orch-go-abc123"}},
		{Type: "agent.completed", Timestamp: now - 5000, Data: map[string]interface{}{"beads_id": "orch-go-untracked-ghi789"}},
		// Abandonment
		{Type: "agent.abandoned", Timestamp: now - 600, Data: map[string]interface{}{"beads_id": "test-untracked-jkl012"}},
	}

	// Test with includeUntracked=false (default behavior)
	reportExcluded := aggregateStats(events, 7, false)

	// Should only count 2 tracked spawns
	if reportExcluded.Summary.TotalSpawns != 2 {
		t.Errorf("expected 2 tracked spawns, got %d", reportExcluded.Summary.TotalSpawns)
	}

	// Should only count 1 tracked completion
	if reportExcluded.Summary.TotalCompletions != 1 {
		t.Errorf("expected 1 tracked completion, got %d", reportExcluded.Summary.TotalCompletions)
	}

	// Should count 0 abandonments (the abandonment was untracked)
	if reportExcluded.Summary.TotalAbandonments != 0 {
		t.Errorf("expected 0 tracked abandonments, got %d", reportExcluded.Summary.TotalAbandonments)
	}

	// Should track untracked spawns separately
	if reportExcluded.Summary.UntrackedSpawns != 2 {
		t.Errorf("expected 2 untracked spawns, got %d", reportExcluded.Summary.UntrackedSpawns)
	}

	// Should track untracked completions separately
	if reportExcluded.Summary.UntrackedCompletions != 1 {
		t.Errorf("expected 1 untracked completion, got %d", reportExcluded.Summary.UntrackedCompletions)
	}

	// Completion rate should be 50% (1/2 tracked)
	expectedRate := 50.0
	if reportExcluded.Summary.CompletionRate < expectedRate-0.1 || reportExcluded.Summary.CompletionRate > expectedRate+0.1 {
		t.Errorf("expected completion rate ~%.1f%%, got %.1f%%", expectedRate, reportExcluded.Summary.CompletionRate)
	}

	// Test with includeUntracked=true
	reportIncluded := aggregateStats(events, 7, true)

	// Should count all 4 spawns
	if reportIncluded.Summary.TotalSpawns != 4 {
		t.Errorf("expected 4 total spawns, got %d", reportIncluded.Summary.TotalSpawns)
	}

	// Should count all 2 completions
	if reportIncluded.Summary.TotalCompletions != 2 {
		t.Errorf("expected 2 total completions, got %d", reportIncluded.Summary.TotalCompletions)
	}

	// Should count the 1 abandonment
	if reportIncluded.Summary.TotalAbandonments != 1 {
		t.Errorf("expected 1 total abandonment, got %d", reportIncluded.Summary.TotalAbandonments)
	}
}

func TestAggregateStatsUntrackedSkillBreakdown(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Tracked feature-impl spawns
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "orch-go-abc123"}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 3600, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "orch-go-def456"}},
		// Untracked feature-impl spawn
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 1800, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "orch-go-untracked-ghi789"}},
		// Completions for all
		{Type: "agent.completed", Timestamp: now - 6000, Data: map[string]interface{}{"beads_id": "orch-go-abc123"}},
		{Type: "agent.completed", Timestamp: now - 5000, Data: map[string]interface{}{"beads_id": "orch-go-def456"}},
		{Type: "agent.completed", Timestamp: now - 4000, Data: map[string]interface{}{"beads_id": "orch-go-untracked-ghi789"}},
	}

	// Test with includeUntracked=false
	report := aggregateStats(events, 7, false)

	// Should only have feature-impl in skill stats with 2 spawns (not 3)
	if len(report.SkillStats) != 1 {
		t.Errorf("expected 1 skill, got %d", len(report.SkillStats))
	}

	if report.SkillStats[0].Skill != "feature-impl" {
		t.Errorf("expected feature-impl skill, got %s", report.SkillStats[0].Skill)
	}

	if report.SkillStats[0].Spawns != 2 {
		t.Errorf("expected 2 tracked spawns for feature-impl, got %d", report.SkillStats[0].Spawns)
	}

	if report.SkillStats[0].Completions != 2 {
		t.Errorf("expected 2 tracked completions for feature-impl, got %d", report.SkillStats[0].Completions)
	}

	// Completion rate should be 100% for tracked feature-impl
	if report.SkillStats[0].CompletionRate < 99.9 || report.SkillStats[0].CompletionRate > 100.1 {
		t.Errorf("expected 100%% completion rate for feature-impl, got %.1f%%", report.SkillStats[0].CompletionRate)
	}
}

func TestAggregateStatsOrchestratorWorkspaceCorrelation(t *testing.T) {
	now := time.Now().Unix()

	// This test verifies that orchestrator completions are correlated via workspace
	// (not beads_id, since orchestrators are untracked by design)
	events := []StatsEvent{
		// Orchestrator spawns (with workspace but untracked beads_id)
		{Type: "session.spawned", SessionID: "orch_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill":     "orchestrator",
			"beads_id":  "orch-go-untracked-123",
			"workspace": "og-orch-test-workspace-1",
		}},
		{Type: "session.spawned", SessionID: "orch_2", Timestamp: now - 3600, Data: map[string]interface{}{
			"skill":     "meta-orchestrator",
			"beads_id":  "orch-go-untracked-456",
			"workspace": "meta-orch-test-workspace-2",
		}},
		// Task skill spawn for comparison
		{Type: "session.spawned", SessionID: "task_1", Timestamp: now - 1800, Data: map[string]interface{}{
			"skill":    "feature-impl",
			"beads_id": "orch-go-abc123",
		}},
		// Orchestrator completions (have workspace and orchestrator flag, no beads_id)
		{Type: "agent.completed", Timestamp: now - 6000, Data: map[string]interface{}{
			"orchestrator": true,
			"workspace":    "og-orch-test-workspace-1",
			"reason":       "Orchestrator session completed",
		}},
		{Type: "agent.completed", Timestamp: now - 5000, Data: map[string]interface{}{
			"orchestrator": true,
			"workspace":    "meta-orch-test-workspace-2",
			"reason":       "Orchestrator session completed",
		}},
		// Task completion (has beads_id)
		{Type: "agent.completed", Timestamp: now - 4000, Data: map[string]interface{}{
			"beads_id": "orch-go-abc123",
		}},
	}

	// Test with includeUntracked=true (to include orchestrators)
	report := aggregateStats(events, 7, true)

	// Should have all 3 spawns
	if report.Summary.TotalSpawns != 3 {
		t.Errorf("expected 3 spawns, got %d", report.Summary.TotalSpawns)
	}

	// Should have all 3 completions (2 orchestrator via workspace, 1 task via beads_id)
	if report.Summary.TotalCompletions != 3 {
		t.Errorf("expected 3 completions, got %d", report.Summary.TotalCompletions)
	}

	// Task skills should have 1 completion
	if report.Summary.TaskCompletions != 1 {
		t.Errorf("expected 1 task completion, got %d", report.Summary.TaskCompletions)
	}

	// Coordination skills should have 2 completions (via workspace correlation)
	if report.Summary.CoordinationCompletions != 2 {
		t.Errorf("expected 2 coordination completions, got %d", report.Summary.CoordinationCompletions)
	}

	// Check coordination completion rate is now 100% (2/2)
	if report.Summary.CoordinationCompletionRate < 99.9 {
		t.Errorf("expected 100%% coordination completion rate, got %.1f%%", report.Summary.CoordinationCompletionRate)
	}

	// Check individual skill stats
	var orchStats, metaOrchStats *SkillStatsSummary
	for i := range report.SkillStats {
		if report.SkillStats[i].Skill == "orchestrator" {
			orchStats = &report.SkillStats[i]
		}
		if report.SkillStats[i].Skill == "meta-orchestrator" {
			metaOrchStats = &report.SkillStats[i]
		}
	}

	if orchStats == nil {
		t.Fatal("orchestrator skill not found in stats")
	}
	if orchStats.Completions != 1 {
		t.Errorf("expected 1 orchestrator completion, got %d", orchStats.Completions)
	}
	if orchStats.CompletionRate < 99.9 {
		t.Errorf("expected 100%% orchestrator completion rate, got %.1f%%", orchStats.CompletionRate)
	}

	if metaOrchStats == nil {
		t.Fatal("meta-orchestrator skill not found in stats")
	}
	if metaOrchStats.Completions != 1 {
		t.Errorf("expected 1 meta-orchestrator completion, got %d", metaOrchStats.Completions)
	}
	if metaOrchStats.CompletionRate < 99.9 {
		t.Errorf("expected 100%% meta-orchestrator completion rate, got %.1f%%", metaOrchStats.CompletionRate)
	}
}

func TestAggregateStatsDeduplicationByBeadsID(t *testing.T) {
	now := time.Now().Unix()

	// This test verifies that multiple completion events for the same beads_id
	// are deduplicated and only counted once.
	events := []StatsEvent{
		// 3 spawns with different beads_ids
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill":    "feature-impl",
			"beads_id": "orch-go-abc1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 3600, Data: map[string]interface{}{
			"skill":    "feature-impl",
			"beads_id": "orch-go-abc2",
		}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 1800, Data: map[string]interface{}{
			"skill":    "investigation",
			"beads_id": "orch-go-abc3",
		}},
		// Multiple completion events for the SAME beads_id (simulates duplicate events)
		// orch-go-abc1 has 3 completion events
		{Type: "agent.completed", Timestamp: now - 6000, Data: map[string]interface{}{"beads_id": "orch-go-abc1"}},
		{Type: "agent.completed", Timestamp: now - 5500, Data: map[string]interface{}{"beads_id": "orch-go-abc1"}},
		{Type: "agent.completed", Timestamp: now - 5000, Data: map[string]interface{}{"beads_id": "orch-go-abc1"}},
		// orch-go-abc2 has 2 completion events
		{Type: "agent.completed", Timestamp: now - 4000, Data: map[string]interface{}{"beads_id": "orch-go-abc2"}},
		{Type: "agent.completed", Timestamp: now - 3500, Data: map[string]interface{}{"beads_id": "orch-go-abc2"}},
		// orch-go-abc3 has 1 completion event (no duplicate)
		{Type: "agent.completed", Timestamp: now - 2000, Data: map[string]interface{}{"beads_id": "orch-go-abc3"}},
	}

	report := aggregateStats(events, 7, true)

	// Should have 3 spawns (unique)
	if report.Summary.TotalSpawns != 3 {
		t.Errorf("expected 3 spawns, got %d", report.Summary.TotalSpawns)
	}

	// Should have 3 completions (deduplicated by beads_id)
	// NOT 6 (the actual number of completion events)
	if report.Summary.TotalCompletions != 3 {
		t.Errorf("expected 3 completions (deduplicated), got %d (should not count %d events)", report.Summary.TotalCompletions, 6)
	}

	// Completion rate should be 100% (3/3)
	expectedRate := 100.0
	if report.Summary.CompletionRate < expectedRate-0.1 {
		t.Errorf("expected completion rate 100%%, got %.1f%%", report.Summary.CompletionRate)
	}

	// Check skill breakdown
	var featureImplStats *SkillStatsSummary
	var invStats *SkillStatsSummary
	for i := range report.SkillStats {
		if report.SkillStats[i].Skill == "feature-impl" {
			featureImplStats = &report.SkillStats[i]
		}
		if report.SkillStats[i].Skill == "investigation" {
			invStats = &report.SkillStats[i]
		}
	}

	if featureImplStats == nil {
		t.Fatal("feature-impl skill not found in stats")
	}
	// feature-impl should have 2 completions (abc1 and abc2), not 5 (3+2 events)
	if featureImplStats.Completions != 2 {
		t.Errorf("expected 2 feature-impl completions (deduplicated), got %d", featureImplStats.Completions)
	}

	if invStats == nil {
		t.Fatal("investigation skill not found in stats")
	}
	if invStats.Completions != 1 {
		t.Errorf("expected 1 investigation completion, got %d", invStats.Completions)
	}
}

func TestAggregateStatsDeduplicationMixedEventTypes(t *testing.T) {
	now := time.Now().Unix()

	// Test deduplication when both session.completed and agent.completed exist
	// for the same beads_id
	events := []StatsEvent{
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill":    "feature-impl",
			"beads_id": "orch-go-mixed1",
		}},
		// Both event types for the same beads_id
		{Type: "session.completed", SessionID: "ses_1", Timestamp: now - 5000, Data: map[string]interface{}{
			"beads_id": "orch-go-mixed1",
		}},
		{Type: "agent.completed", Timestamp: now - 4000, Data: map[string]interface{}{
			"beads_id": "orch-go-mixed1",
		}},
	}

	report := aggregateStats(events, 7, true)

	// Should have 1 spawn
	if report.Summary.TotalSpawns != 1 {
		t.Errorf("expected 1 spawn, got %d", report.Summary.TotalSpawns)
	}

	// Should have 1 completion (deduplicated across event types)
	if report.Summary.TotalCompletions != 1 {
		t.Errorf("expected 1 completion (deduplicated across event types), got %d", report.Summary.TotalCompletions)
	}
}

func TestAggregateStatsVerification(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Spawn for 3 agents
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 6200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-2",
		}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 5200, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "test-3",
		}},
		// Verification failures (these don't create attempts, just track gate failures)
		{Type: "verification.failed", Timestamp: now - 4000, Data: map[string]interface{}{
			"beads_id":     "test-1",
			"gates_failed": []interface{}{"test_evidence", "git_diff"},
			"skill":        "feature-impl",
		}},
		{Type: "verification.failed", Timestamp: now - 3000, Data: map[string]interface{}{
			"beads_id":     "test-2",
			"gates_failed": []interface{}{"test_evidence"},
			"skill":        "feature-impl",
		}},
		// Completions - test-1 passed first try, test-2 was forced, test-3 passed
		{Type: "agent.completed", Timestamp: now - 2000, Data: map[string]interface{}{
			"beads_id":            "test-1",
			"verification_passed": true,
			"forced":              false,
			"skill":               "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 1500, Data: map[string]interface{}{
			"beads_id":            "test-2",
			"verification_passed": false,
			"forced":              true,
			"gates_bypassed":      []interface{}{"test_evidence"},
			"skill":               "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 1000, Data: map[string]interface{}{
			"beads_id":            "test-3",
			"verification_passed": true,
			"forced":              false,
			"skill":               "investigation",
		}},
	}

	report := aggregateStats(events, 7, true)

	// Test verification stats
	if report.VerificationStats.TotalAttempts != 3 {
		t.Errorf("expected 3 verification attempts, got %d", report.VerificationStats.TotalAttempts)
	}

	if report.VerificationStats.PassedFirstTry != 2 {
		t.Errorf("expected 2 passed first try, got %d", report.VerificationStats.PassedFirstTry)
	}

	if report.VerificationStats.Bypassed != 1 {
		t.Errorf("expected 1 bypassed, got %d", report.VerificationStats.Bypassed)
	}

	// Check pass rate (2/3 = 66.67%)
	expectedPassRate := (2.0 / 3.0) * 100
	if report.VerificationStats.PassRate < expectedPassRate-0.1 || report.VerificationStats.PassRate > expectedPassRate+0.1 {
		t.Errorf("expected pass rate ~%.1f%%, got %.1f%%", expectedPassRate, report.VerificationStats.PassRate)
	}

	// Check bypass rate (1/3 = 33.33%)
	expectedBypassRate := (1.0 / 3.0) * 100
	if report.VerificationStats.BypassRate < expectedBypassRate-0.1 || report.VerificationStats.BypassRate > expectedBypassRate+0.1 {
		t.Errorf("expected bypass rate ~%.1f%%, got %.1f%%", expectedBypassRate, report.VerificationStats.BypassRate)
	}
}

func TestAggregateStatsVerificationGateBreakdown(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Spawns
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 6200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-2",
		}},
		// Verification failures - test_evidence fails twice, git_diff once, visual_verification once
		{Type: "verification.failed", Timestamp: now - 4000, Data: map[string]interface{}{
			"beads_id":     "test-1",
			"gates_failed": []interface{}{"test_evidence", "git_diff"},
			"skill":        "feature-impl",
		}},
		{Type: "verification.failed", Timestamp: now - 3500, Data: map[string]interface{}{
			"beads_id":     "test-2",
			"gates_failed": []interface{}{"test_evidence", "visual_verification"},
			"skill":        "feature-impl",
		}},
		// Completions - both bypassed with different gates
		{Type: "agent.completed", Timestamp: now - 2000, Data: map[string]interface{}{
			"beads_id":            "test-1",
			"verification_passed": false,
			"forced":              true,
			"gates_bypassed":      []interface{}{"test_evidence", "git_diff"},
			"skill":               "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 1000, Data: map[string]interface{}{
			"beads_id":            "test-2",
			"verification_passed": false,
			"forced":              true,
			"gates_bypassed":      []interface{}{"test_evidence"},
			"skill":               "feature-impl",
		}},
	}

	report := aggregateStats(events, 7, true)

	// Check gate breakdown
	if len(report.VerificationStats.FailuresByGate) == 0 {
		t.Fatal("expected gate breakdown, got none")
	}

	// Find test_evidence gate stats
	var testEvidenceStats *GateFailureStats
	var gitDiffStats *GateFailureStats
	for i := range report.VerificationStats.FailuresByGate {
		gate := &report.VerificationStats.FailuresByGate[i]
		if gate.Gate == "test_evidence" {
			testEvidenceStats = gate
		}
		if gate.Gate == "git_diff" {
			gitDiffStats = gate
		}
	}

	if testEvidenceStats == nil {
		t.Fatal("test_evidence gate not found in breakdown")
	}

	// test_evidence: failed 2 times, bypassed 2 times (both completions bypassed it)
	if testEvidenceStats.FailCount != 2 {
		t.Errorf("expected test_evidence to have 2 failures, got %d", testEvidenceStats.FailCount)
	}
	if testEvidenceStats.BypassCount != 2 {
		t.Errorf("expected test_evidence to have 2 bypasses, got %d", testEvidenceStats.BypassCount)
	}

	if gitDiffStats == nil {
		t.Fatal("git_diff gate not found in breakdown")
	}

	// git_diff: failed 1 time, bypassed 1 time
	if gitDiffStats.FailCount != 1 {
		t.Errorf("expected git_diff to have 1 failure, got %d", gitDiffStats.FailCount)
	}
	if gitDiffStats.BypassCount != 1 {
		t.Errorf("expected git_diff to have 1 bypass, got %d", gitDiffStats.BypassCount)
	}
}

func TestAggregateStatsVerificationBySkill(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Spawns for different skills
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 6200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-2",
		}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 5200, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "test-3",
		}},
		// Completions
		{Type: "agent.completed", Timestamp: now - 2000, Data: map[string]interface{}{
			"beads_id":            "test-1",
			"verification_passed": true,
			"forced":              false,
			"skill":               "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 1500, Data: map[string]interface{}{
			"beads_id":            "test-2",
			"verification_passed": false,
			"forced":              true,
			"skill":               "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 1000, Data: map[string]interface{}{
			"beads_id":            "test-3",
			"verification_passed": true,
			"forced":              false,
			"skill":               "investigation",
		}},
	}

	report := aggregateStats(events, 7, true)

	// Check skill breakdown
	if len(report.VerificationStats.BySkill) != 2 {
		t.Errorf("expected 2 skills in verification breakdown, got %d", len(report.VerificationStats.BySkill))
	}

	// Find feature-impl stats
	var featureImplStats *SkillVerificationStats
	var invStats *SkillVerificationStats
	for i := range report.VerificationStats.BySkill {
		sv := &report.VerificationStats.BySkill[i]
		if sv.Skill == "feature-impl" {
			featureImplStats = sv
		}
		if sv.Skill == "investigation" {
			invStats = sv
		}
	}

	if featureImplStats == nil {
		t.Fatal("feature-impl not found in skill breakdown")
	}
	if featureImplStats.TotalAttempts != 2 {
		t.Errorf("expected feature-impl to have 2 attempts, got %d", featureImplStats.TotalAttempts)
	}
	if featureImplStats.PassedFirstTry != 1 {
		t.Errorf("expected feature-impl to have 1 passed first try, got %d", featureImplStats.PassedFirstTry)
	}
	if featureImplStats.Bypassed != 1 {
		t.Errorf("expected feature-impl to have 1 bypassed, got %d", featureImplStats.Bypassed)
	}
	// Pass rate should be 50%
	if featureImplStats.PassRate < 49.9 || featureImplStats.PassRate > 50.1 {
		t.Errorf("expected feature-impl pass rate 50%%, got %.1f%%", featureImplStats.PassRate)
	}

	if invStats == nil {
		t.Fatal("investigation not found in skill breakdown")
	}
	if invStats.TotalAttempts != 1 {
		t.Errorf("expected investigation to have 1 attempt, got %d", invStats.TotalAttempts)
	}
	if invStats.PassedFirstTry != 1 {
		t.Errorf("expected investigation to have 1 passed first try, got %d", invStats.PassedFirstTry)
	}
	// Pass rate should be 100%
	if invStats.PassRate < 99.9 {
		t.Errorf("expected investigation pass rate 100%%, got %.1f%%", invStats.PassRate)
	}
}

func TestAggregateStatsReopenedCount(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Initial spawns
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 6200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-2",
		}},
		// First completions
		{Type: "agent.completed", Timestamp: now - 5000, Data: map[string]interface{}{"beads_id": "test-1"}},
		{Type: "agent.completed", Timestamp: now - 4500, Data: map[string]interface{}{"beads_id": "test-2"}},
		// Issue reopened events (issues were closed and reopened for another attempt)
		{Type: "issue.reopened", Timestamp: now - 4000, Data: map[string]interface{}{
			"beads_id":        "test-1",
			"previous_status": "closed",
			"attempt_number":  2,
		}},
		{Type: "issue.reopened", Timestamp: now - 3000, Data: map[string]interface{}{
			"beads_id":        "test-1",
			"previous_status": "closed",
			"attempt_number":  3,
		}},
		{Type: "issue.reopened", Timestamp: now - 2000, Data: map[string]interface{}{
			"beads_id":        "test-2",
			"previous_status": "closed",
			"attempt_number":  2,
		}},
	}

	report := aggregateStats(events, 7, true)

	// Should have 3 reopened events
	if report.AttemptStats.ReopenedCount != 3 {
		t.Errorf("expected 3 reopened events, got %d", report.AttemptStats.ReopenedCount)
	}
}

func TestAggregateStatsReopenedCountTimeFiltering(t *testing.T) {
	now := time.Now().Unix()
	oldTimestamp := now - (10 * 24 * 60 * 60) // 10 days ago (outside 7-day window)

	events := []StatsEvent{
		// Reopen within 7-day window
		{Type: "issue.reopened", Timestamp: now - 3600, Data: map[string]interface{}{
			"beads_id":        "test-1",
			"previous_status": "closed",
			"attempt_number":  2,
		}},
		// Reopen outside 7-day window (should not be counted)
		{Type: "issue.reopened", Timestamp: oldTimestamp, Data: map[string]interface{}{
			"beads_id":        "test-2",
			"previous_status": "closed",
			"attempt_number":  2,
		}},
	}

	report := aggregateStats(events, 7, true)

	// Should only count the reopen within the time window
	if report.AttemptStats.ReopenedCount != 1 {
		t.Errorf("expected 1 reopened event in 7-day window, got %d", report.AttemptStats.ReopenedCount)
	}
}

func TestAggregateStatsReopenedCountEmpty(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Only spawns and completions, no reopens
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
		{Type: "agent.completed", Timestamp: now - 5000, Data: map[string]interface{}{"beads_id": "test-1"}},
	}

	report := aggregateStats(events, 7, true)

	// Should have 0 reopened events
	if report.AttemptStats.ReopenedCount != 0 {
		t.Errorf("expected 0 reopened events, got %d", report.AttemptStats.ReopenedCount)
	}
}
