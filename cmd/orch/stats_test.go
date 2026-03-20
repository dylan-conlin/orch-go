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
	report := aggregateStats(parsed, 7)
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

	report := aggregateStats(events, 7) // includeUntracked=true for backwards compat

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

	report := aggregateStats(events, 7)

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

	report := aggregateStats(events, 7) // includeUntracked=true for backwards compat

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

	report := aggregateStats(events, 7) // includeUntracked=true for backwards compat

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
	report := aggregateStats(events, 7)

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

	report := aggregateStats(events, 7)

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

	report := aggregateStats(events, 7)

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

	report := aggregateStats(events, 7)

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

	report := aggregateStats(events, 7)

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

	report := aggregateStats(events, 7)

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

func TestAggregateStatsVerificationBypassed(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Spawns
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 6200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-2",
		}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 5200, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "test-3",
		}},
		// verification.bypassed events (from --skip-* flags)
		// test-1 skipped test_evidence and git_diff
		{Type: "verification.bypassed", SessionID: "test-1", Timestamp: now - 3000, Data: map[string]interface{}{
			"beads_id": "test-1", "gate": "test_evidence", "reason": "Tests run in CI pipeline", "skill": "feature-impl",
		}},
		{Type: "verification.bypassed", SessionID: "test-1", Timestamp: now - 3000, Data: map[string]interface{}{
			"beads_id": "test-1", "gate": "git_diff", "reason": "Tests run in CI pipeline", "skill": "feature-impl",
		}},
		// test-2 skipped synthesis
		{Type: "verification.bypassed", SessionID: "test-2", Timestamp: now - 2500, Data: map[string]interface{}{
			"beads_id": "test-2", "gate": "synthesis", "reason": "Docs-only change", "skill": "feature-impl",
		}},
		// verification.auto_skipped event
		{Type: "verification.auto_skipped", SessionID: "test-3", Timestamp: now - 2000, Data: map[string]interface{}{
			"beads_id": "test-3", "gate": "test_evidence", "reason": "investigation skill exemption", "skill": "investigation",
		}},
		// Completions
		{Type: "agent.completed", Timestamp: now - 1500, Data: map[string]interface{}{
			"beads_id": "test-1", "verification_passed": true, "skill": "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 1000, Data: map[string]interface{}{
			"beads_id": "test-2", "verification_passed": true, "skill": "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 500, Data: map[string]interface{}{
			"beads_id": "test-3", "verification_passed": true, "skill": "investigation",
		}},
	}

	report := aggregateStats(events, 7)

	// Should count 3 skip-bypassed gate events
	if report.VerificationStats.SkipBypassed != 3 {
		t.Errorf("expected 3 skip-bypassed events, got %d", report.VerificationStats.SkipBypassed)
	}

	// Should count 1 auto-skipped event
	if report.VerificationStats.AutoSkipped != 1 {
		t.Errorf("expected 1 auto-skipped event, got %d", report.VerificationStats.AutoSkipped)
	}

	// Gate bypass breakdown should include gates from verification.bypassed events
	var testEvidenceStats *GateFailureStats
	var gitDiffStats *GateFailureStats
	var synthesisStats *GateFailureStats
	for i := range report.VerificationStats.FailuresByGate {
		gate := &report.VerificationStats.FailuresByGate[i]
		switch gate.Gate {
		case "test_evidence":
			testEvidenceStats = gate
		case "git_diff":
			gitDiffStats = gate
		case "synthesis":
			synthesisStats = gate
		}
	}

	if testEvidenceStats == nil {
		t.Fatal("test_evidence gate not found in breakdown")
	}
	// test_evidence: bypassed 1 time via --skip-*, auto-skipped 1 time
	if testEvidenceStats.BypassCount != 1 {
		t.Errorf("expected test_evidence to have 1 bypass, got %d", testEvidenceStats.BypassCount)
	}
	if testEvidenceStats.AutoSkipCount != 1 {
		t.Errorf("expected test_evidence to have 1 auto-skip, got %d", testEvidenceStats.AutoSkipCount)
	}

	if gitDiffStats == nil {
		t.Fatal("git_diff gate not found in breakdown")
	}
	if gitDiffStats.BypassCount != 1 {
		t.Errorf("expected git_diff to have 1 bypass, got %d", gitDiffStats.BypassCount)
	}

	if synthesisStats == nil {
		t.Fatal("synthesis gate not found in breakdown")
	}
	if synthesisStats.BypassCount != 1 {
		t.Errorf("expected synthesis to have 1 bypass, got %d", synthesisStats.BypassCount)
	}

	// Check bypass reasons are tracked
	if len(report.VerificationStats.BypassReasons) == 0 {
		t.Fatal("expected bypass reasons to be populated")
	}

	// Should have 2 unique gate+reason combos:
	// test_evidence|Tests run in CI pipeline, git_diff|Tests run in CI pipeline, synthesis|Docs-only change
	if len(report.VerificationStats.BypassReasons) != 3 {
		t.Errorf("expected 3 bypass reason entries, got %d", len(report.VerificationStats.BypassReasons))
	}
}

func TestAggregateStatsAbandonmentDeduplication(t *testing.T) {
	// Regression test: orch abandon emits TWO agent.abandoned events per abandonment
	// (one from LifecycleManager, one from telemetry). Stats must deduplicate by beads_id.
	now := time.Now().Unix()

	events := []StatsEvent{
		// Spawn an architect agent
		{Type: "session.spawned", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "architect", "beads_id": "test-arch-1", "workspace": "og-arch-test-1",
		}},
		// Spawn a feature-impl agent
		{Type: "session.spawned", SessionID: "ses_fi1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-fi-1",
		}},
		// Abandon architect: event 1 (from LifecycleManager - no skill field)
		{Type: "agent.abandoned", Timestamp: now - 3600, Data: map[string]interface{}{
			"beads_id": "test-arch-1", "workspace": "og-arch-test-1", "reason": "stuck",
		}},
		// Abandon architect: event 2 (from telemetry - has skill field)
		// In production this would be agent.abandoned.telemetry after the fix,
		// but this tests the dedup safety net for existing events in events.jsonl
		{Type: "agent.abandoned", Timestamp: now - 3600, Data: map[string]interface{}{
			"beads_id": "test-arch-1", "workspace": "og-arch-test-1", "reason": "stuck", "skill": "architect",
		}},
		// Abandon feature-impl: event 1
		{Type: "agent.abandoned", Timestamp: now - 1800, Data: map[string]interface{}{
			"beads_id": "test-fi-1", "reason": "died",
		}},
		// Abandon feature-impl: event 2 (duplicate)
		{Type: "agent.abandoned", Timestamp: now - 1800, Data: map[string]interface{}{
			"beads_id": "test-fi-1", "reason": "died", "skill": "feature-impl",
		}},
	}

	report := aggregateStats(events, 7)

	// Should count 2 unique abandonments (not 4)
	if report.Summary.TotalAbandonments != 2 {
		t.Errorf("expected 2 unique abandonments, got %d (duplicate events not deduplicated)", report.Summary.TotalAbandonments)
	}

	// Check per-skill counts are also deduplicated
	for _, s := range report.SkillStats {
		switch s.Skill {
		case "architect":
			if s.Abandonments != 1 {
				t.Errorf("expected 1 architect abandonment, got %d", s.Abandonments)
			}
		case "feature-impl":
			if s.Abandonments != 1 {
				t.Errorf("expected 1 feature-impl abandonment, got %d", s.Abandonments)
			}
		}
	}
}

func TestAggregateStatsAbandonmentRetries(t *testing.T) {
	// Test: same beads_id abandoned multiple times (retries) counts as one abandonment
	now := time.Now().Unix()

	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "architect", "beads_id": "test-retry-1", "workspace": "og-arch-retry-1",
		}},
		// Abandoned 3 times (agent retried and failed)
		{Type: "agent.abandoned", Timestamp: now - 5400, Data: map[string]interface{}{
			"beads_id": "test-retry-1", "reason": "stuck attempt 1",
		}},
		{Type: "agent.abandoned", Timestamp: now - 3600, Data: map[string]interface{}{
			"beads_id": "test-retry-1", "reason": "stuck attempt 2",
		}},
		{Type: "agent.abandoned", Timestamp: now - 1800, Data: map[string]interface{}{
			"beads_id": "test-retry-1", "reason": "stuck attempt 3",
		}},
	}

	report := aggregateStats(events, 7)

	if report.Summary.TotalAbandonments != 1 {
		t.Errorf("expected 1 unique abandonment for retried issue, got %d", report.Summary.TotalAbandonments)
	}
}

func TestAggregateStatsSpawnGateBypasses(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// 10 total spawns (to calculate bypass rates)
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 9000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 8000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-2",
		}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 7000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-3",
		}},
		{Type: "session.spawned", SessionID: "ses_4", Timestamp: now - 6000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-4",
		}},
		{Type: "session.spawned", SessionID: "ses_5", Timestamp: now - 5000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-5",
		}},
		{Type: "session.spawned", SessionID: "ses_6", Timestamp: now - 4000, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "test-6",
		}},
		{Type: "session.spawned", SessionID: "ses_7", Timestamp: now - 3000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-7",
		}},
		{Type: "session.spawned", SessionID: "ses_8", Timestamp: now - 2000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-8",
		}},
		{Type: "session.spawned", SessionID: "ses_9", Timestamp: now - 1500, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-9",
		}},
		{Type: "session.spawned", SessionID: "ses_10", Timestamp: now - 1000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-10",
		}},

		// Triage bypasses (3x)
		{Type: "spawn.triage_bypassed", Timestamp: now - 8500, Data: map[string]interface{}{
			"skill": "feature-impl", "task": "urgent fix", "reason": "urgent production issue",
		}},
		{Type: "spawn.triage_bypassed", Timestamp: now - 7500, Data: map[string]interface{}{
			"skill": "feature-impl", "task": "quick change", "reason": "small config update",
		}},
		{Type: "spawn.triage_bypassed", Timestamp: now - 6500, Data: map[string]interface{}{
			"skill": "feature-impl", "task": "another fix", "reason": "urgent production issue",
		}},

		// Hotspot bypasses (2x)
		{Type: "spawn.hotspot_bypassed", Timestamp: now - 5500, Data: map[string]interface{}{
			"skill": "feature-impl", "task": "refactor gate", "architect_ref": "orch-go-abc",
			"reason": "architect approved extraction plan", "critical_files": []interface{}{"cmd/orch/spawn_cmd.go"},
		}},
		{Type: "spawn.hotspot_bypassed", Timestamp: now - 4500, Data: map[string]interface{}{
			"skill": "systematic-debugging", "task": "fix spawn bug", "architect_ref": "orch-go-def",
			"reason": "architect approved extraction plan", "critical_files": []interface{}{"cmd/orch/stats_cmd.go"},
		}},

		// Verification gate bypasses at spawn time (1x)
		{Type: "spawn.verification_bypassed", Timestamp: now - 3500, Data: map[string]interface{}{
			"reason": "independent parallel work on unrelated feature",
		}},
	}

	report := aggregateStats(events, 7)

	// Check SpawnGateStats
	if report.SpawnGateStats.TotalBypasses != 6 {
		t.Errorf("expected 6 total spawn gate bypasses, got %d", report.SpawnGateStats.TotalBypasses)
	}

	if report.SpawnGateStats.TotalSpawns != 10 {
		t.Errorf("expected 10 total spawns for rate calc, got %d", report.SpawnGateStats.TotalSpawns)
	}

	expectedRate := 60.0 // 6/10 * 100
	if report.SpawnGateStats.BypassRate < expectedRate-0.1 || report.SpawnGateStats.BypassRate > expectedRate+0.1 {
		t.Errorf("expected bypass rate ~%.1f%%, got %.1f%%", expectedRate, report.SpawnGateStats.BypassRate)
	}

	// Check per-gate breakdown
	gateMap := make(map[string]*SpawnGateEntry)
	for i := range report.SpawnGateStats.ByGate {
		gateMap[report.SpawnGateStats.ByGate[i].Gate] = &report.SpawnGateStats.ByGate[i]
	}

	triage, ok := gateMap["triage"]
	if !ok {
		t.Fatal("triage gate not found in SpawnGateStats.ByGate")
	}
	if triage.Bypassed != 3 {
		t.Errorf("expected triage bypassed=3, got %d", triage.Bypassed)
	}

	hotspot, ok := gateMap["hotspot"]
	if !ok {
		t.Fatal("hotspot gate not found in SpawnGateStats.ByGate")
	}
	if hotspot.Bypassed != 2 {
		t.Errorf("expected hotspot bypassed=2, got %d", hotspot.Bypassed)
	}

	verification, ok := gateMap["verification"]
	if !ok {
		t.Fatal("verification gate not found in SpawnGateStats.ByGate")
	}
	if verification.Bypassed != 1 {
		t.Errorf("expected verification bypassed=1, got %d", verification.Bypassed)
	}

	// Check top reasons
	if len(report.SpawnGateStats.TopReasons) == 0 {
		t.Fatal("expected top reasons to be populated")
	}
	// "urgent production issue" should appear 2x, "architect approved extraction plan" 2x
	reasonMap := make(map[string]int)
	for _, r := range report.SpawnGateStats.TopReasons {
		reasonMap[r.Gate+"|"+r.Reason] = r.Count
	}
	if reasonMap["triage|urgent production issue"] != 2 {
		t.Errorf("expected 'urgent production issue' 2x for triage, got %d", reasonMap["triage|urgent production issue"])
	}
}

func TestAggregateStatsSpawnGateMiscalibrationWarning(t *testing.T) {
	now := time.Now().Unix()

	// Scenario: 5 spawns, 4 triage bypasses → 80% bypass rate → miscalibration
	events := []StatsEvent{
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 5000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 4000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-2",
		}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 3000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-3",
		}},
		{Type: "session.spawned", SessionID: "ses_4", Timestamp: now - 2000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-4",
		}},
		{Type: "session.spawned", SessionID: "ses_5", Timestamp: now - 1000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-5",
		}},
		// 4 triage bypasses
		{Type: "spawn.triage_bypassed", Timestamp: now - 4500, Data: map[string]interface{}{
			"skill": "feature-impl", "reason": "manual spawn",
		}},
		{Type: "spawn.triage_bypassed", Timestamp: now - 3500, Data: map[string]interface{}{
			"skill": "feature-impl", "reason": "manual spawn",
		}},
		{Type: "spawn.triage_bypassed", Timestamp: now - 2500, Data: map[string]interface{}{
			"skill": "feature-impl", "reason": "manual spawn",
		}},
		{Type: "spawn.triage_bypassed", Timestamp: now - 1500, Data: map[string]interface{}{
			"skill": "feature-impl", "reason": "manual spawn",
		}},
	}

	report := aggregateStats(events, 7)

	// Triage gate should be flagged as miscalibrated (>50% bypass rate)
	var triageEntry *SpawnGateEntry
	for i := range report.SpawnGateStats.ByGate {
		if report.SpawnGateStats.ByGate[i].Gate == "triage" {
			triageEntry = &report.SpawnGateStats.ByGate[i]
			break
		}
	}

	if triageEntry == nil {
		t.Fatal("triage gate not found")
	}

	expectedBypassRate := 80.0 // 4/5 * 100
	if triageEntry.BypassRate < expectedBypassRate-0.1 || triageEntry.BypassRate > expectedBypassRate+0.1 {
		t.Errorf("expected triage bypass rate ~%.1f%%, got %.1f%%", expectedBypassRate, triageEntry.BypassRate)
	}

	if !triageEntry.Miscalibrated {
		t.Error("expected triage gate to be flagged as miscalibrated (>50% bypass rate)")
	}
}

func TestGateDecisionStats(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Triage gate block
		{Type: "spawn.gate_decision", Timestamp: now - 100, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "block", "skill": "feature-impl",
		}},
		// Triage gate bypass
		{Type: "spawn.gate_decision", Timestamp: now - 90, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "bypass", "skill": "feature-impl",
		}},
		// Hotspot gate block
		{Type: "spawn.gate_decision", Timestamp: now - 80, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "block", "skill": "feature-impl",
			"target_files": []interface{}{"cmd/orch/spawn_cmd.go"},
		}},
		// Hotspot gate block (different skill)
		{Type: "spawn.gate_decision", Timestamp: now - 70, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "block", "skill": "systematic-debugging",
		}},
		// Verification gate block
		{Type: "spawn.gate_decision", Timestamp: now - 60, Data: map[string]interface{}{
			"gate_name": "verification", "decision": "block", "skill": "feature-impl",
		}},
		// Accretion precommit block (no skill — precommit context)
		{Type: "spawn.gate_decision", Timestamp: now - 50, Data: map[string]interface{}{
			"gate_name": "accretion_precommit", "decision": "block",
			"target_files": []interface{}{"cmd/orch/stats_cmd.go"},
		}},
		// Accretion precommit bypass
		{Type: "spawn.gate_decision", Timestamp: now - 40, Data: map[string]interface{}{
			"gate_name": "accretion_precommit", "decision": "bypass",
		}},
	}

	report := aggregateStats(events, 7)

	// Verify totals
	if report.GateDecisionStats.TotalDecisions != 7 {
		t.Errorf("TotalDecisions = %d, want 7", report.GateDecisionStats.TotalDecisions)
	}
	if report.GateDecisionStats.TotalBlocks != 5 {
		t.Errorf("TotalBlocks = %d, want 5", report.GateDecisionStats.TotalBlocks)
	}
	if report.GateDecisionStats.TotalBypasses != 2 {
		t.Errorf("TotalBypasses = %d, want 2", report.GateDecisionStats.TotalBypasses)
	}

	// Verify per-gate breakdown
	if len(report.GateDecisionStats.ByGate) != 4 {
		t.Fatalf("ByGate length = %d, want 4", len(report.GateDecisionStats.ByGate))
	}

	// Find hotspot entry (should have 2 blocks, 0 bypasses)
	var hotspotEntry *GateDecisionEntry
	for i := range report.GateDecisionStats.ByGate {
		if report.GateDecisionStats.ByGate[i].Gate == "hotspot" {
			hotspotEntry = &report.GateDecisionStats.ByGate[i]
			break
		}
	}
	if hotspotEntry == nil {
		t.Fatal("hotspot gate entry not found in ByGate")
	}
	if hotspotEntry.Blocks != 2 {
		t.Errorf("hotspot.Blocks = %d, want 2", hotspotEntry.Blocks)
	}
	if hotspotEntry.Bypasses != 0 {
		t.Errorf("hotspot.Bypasses = %d, want 0", hotspotEntry.Bypasses)
	}

	// Verify top blocked skills (each gate|skill combo is a separate entry)
	// feature-impl blocked by: triage(1), hotspot(1), verification(1) = 3 entries
	// systematic-debugging blocked by: hotspot(1) = 1 entry
	if len(report.GateDecisionStats.TopBlockedSkills) != 4 {
		t.Errorf("TopBlockedSkills length = %d, want 4", len(report.GateDecisionStats.TopBlockedSkills))
	}
	// All entries should have count=1 (each gate|skill combo appears once)
	for _, entry := range report.GateDecisionStats.TopBlockedSkills {
		if entry.Count != 1 {
			t.Errorf("TopBlockedSkills entry %s|%s count = %d, want 1", entry.Gate, entry.Skill, entry.Count)
		}
	}
}

func TestGateDecisionStats_AllowEvents(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Triage gate allow (daemon-driven, no bypass needed)
		{Type: "spawn.gate_decision", Timestamp: now - 100, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "allow", "skill": "feature-impl",
		}},
		// Hotspot gate allow (no critical hotspot)
		{Type: "spawn.gate_decision", Timestamp: now - 90, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "feature-impl",
		}},
		// Verification gate allow (no unverified work)
		{Type: "spawn.gate_decision", Timestamp: now - 80, Data: map[string]interface{}{
			"gate_name": "verification", "decision": "allow", "skill": "investigation",
		}},
		// Hotspot gate block (for contrast)
		{Type: "spawn.gate_decision", Timestamp: now - 70, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "block", "skill": "feature-impl",
		}},
		// Triage gate bypass (for contrast)
		{Type: "spawn.gate_decision", Timestamp: now - 60, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "bypass", "skill": "feature-impl",
		}},
	}

	report := aggregateStats(events, 7)

	// Verify totals: 3 allows + 1 block + 1 bypass = 5 decisions
	if report.GateDecisionStats.TotalDecisions != 5 {
		t.Errorf("TotalDecisions = %d, want 5", report.GateDecisionStats.TotalDecisions)
	}
	if report.GateDecisionStats.TotalAllows != 3 {
		t.Errorf("TotalAllows = %d, want 3", report.GateDecisionStats.TotalAllows)
	}
	if report.GateDecisionStats.TotalBlocks != 1 {
		t.Errorf("TotalBlocks = %d, want 1", report.GateDecisionStats.TotalBlocks)
	}
	if report.GateDecisionStats.TotalBypasses != 1 {
		t.Errorf("TotalBypasses = %d, want 1", report.GateDecisionStats.TotalBypasses)
	}

	// Verify per-gate: hotspot should have 1 allow + 1 block
	var hotspotEntry *GateDecisionEntry
	for i := range report.GateDecisionStats.ByGate {
		if report.GateDecisionStats.ByGate[i].Gate == "hotspot" {
			hotspotEntry = &report.GateDecisionStats.ByGate[i]
			break
		}
	}
	if hotspotEntry == nil {
		t.Fatal("hotspot gate entry not found in ByGate")
	}
	if hotspotEntry.Allows != 1 {
		t.Errorf("hotspot.Allows = %d, want 1", hotspotEntry.Allows)
	}
	if hotspotEntry.Blocks != 1 {
		t.Errorf("hotspot.Blocks = %d, want 1", hotspotEntry.Blocks)
	}

	// Verify triage: 1 allow + 1 bypass
	var triageEntry *GateDecisionEntry
	for i := range report.GateDecisionStats.ByGate {
		if report.GateDecisionStats.ByGate[i].Gate == "triage" {
			triageEntry = &report.GateDecisionStats.ByGate[i]
			break
		}
	}
	if triageEntry == nil {
		t.Fatal("triage gate entry not found in ByGate")
	}
	if triageEntry.Allows != 1 {
		t.Errorf("triage.Allows = %d, want 1", triageEntry.Allows)
	}
	if triageEntry.Bypasses != 1 {
		t.Errorf("triage.Bypasses = %d, want 1", triageEntry.Bypasses)
	}
}

func TestGateDecisionStats_Empty(t *testing.T) {
	events := []StatsEvent{
		// Only a spawn event, no gate decisions
		{Type: "session.spawned", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"skill": "feature-impl",
		}},
	}

	report := aggregateStats(events, 7)

	if report.GateDecisionStats.TotalDecisions != 0 {
		t.Errorf("TotalDecisions = %d, want 0", report.GateDecisionStats.TotalDecisions)
	}
	if len(report.GateDecisionStats.ByGate) != 0 {
		t.Errorf("ByGate should be empty, got %d entries", len(report.GateDecisionStats.ByGate))
	}
}

func TestGateEffectivenessStats(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// --- Spawns ---
		// Gated spawns (have corresponding gate_decision events)
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "gated-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 6200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "gated-2",
		}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 5200, Data: map[string]interface{}{
			"skill": "systematic-debugging", "beads_id": "gated-3",
		}},
		// Ungated spawns (no gate_decision events)
		{Type: "session.spawned", SessionID: "ses_4", Timestamp: now - 4200, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "ungated-1",
		}},
		{Type: "session.spawned", SessionID: "ses_5", Timestamp: now - 3200, Data: map[string]interface{}{
			"skill": "architect", "beads_id": "ungated-2",
		}},
		// Blocked spawn (gate blocked, then escalated to architect)
		{Type: "session.spawned", SessionID: "ses_6", Timestamp: now - 2200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "blocked-1",
		}},

		// --- Gate decisions ---
		// gated-1: bypass (allowed through)
		{Type: "spawn.gate_decision", Timestamp: now - 7100, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "bypass", "skill": "feature-impl", "beads_id": "gated-1",
		}},
		// gated-2: bypass
		{Type: "spawn.gate_decision", Timestamp: now - 6100, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "bypass", "skill": "feature-impl", "beads_id": "gated-2",
		}},
		// gated-3: bypass
		{Type: "spawn.gate_decision", Timestamp: now - 5100, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "bypass", "skill": "systematic-debugging", "beads_id": "gated-3",
		}},
		// blocked-1: block
		{Type: "spawn.gate_decision", Timestamp: now - 2100, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "block", "skill": "feature-impl", "beads_id": "blocked-1",
		}},

		// --- Architect escalation for blocked work ---
		{Type: "daemon.architect_escalation", Timestamp: now - 2000, Data: map[string]interface{}{
			"issue_id": "blocked-1", "hotspot_file": "cmd/orch/stats_cmd.go",
			"hotspot_type": "CRITICAL", "escalated": true,
		}},

		// --- Completions ---
		// gated-1: completed, verification passed
		{Type: "agent.completed", Timestamp: now - 5000, Data: map[string]interface{}{
			"beads_id": "gated-1", "verification_passed": true, "skill": "feature-impl",
		}},
		// gated-2: completed, verification failed (forced)
		{Type: "agent.completed", Timestamp: now - 4000, Data: map[string]interface{}{
			"beads_id": "gated-2", "verification_passed": false, "forced": true, "skill": "feature-impl",
		}},
		// gated-3: abandoned
		{Type: "agent.abandoned", Timestamp: now - 3000, Data: map[string]interface{}{
			"beads_id": "gated-3",
		}},
		// ungated-1: completed, verification passed
		{Type: "agent.completed", Timestamp: now - 2500, Data: map[string]interface{}{
			"beads_id": "ungated-1", "verification_passed": true, "skill": "investigation",
		}},
		// ungated-2: abandoned
		{Type: "agent.abandoned", Timestamp: now - 1500, Data: map[string]interface{}{
			"beads_id": "ungated-2",
		}},
		// blocked-1: eventually completed after architect redirect
		{Type: "agent.completed", Timestamp: now - 1000, Data: map[string]interface{}{
			"beads_id": "blocked-1", "verification_passed": true, "skill": "feature-impl",
		}},
	}

	report := aggregateStats(events, 7)
	ge := report.GateEffectivenessStats

	// Total evaluations = 4 gate_decision events
	if ge.TotalEvaluations != 4 {
		t.Errorf("TotalEvaluations = %d, want 4", ge.TotalEvaluations)
	}

	// 1 block, 3 bypasses
	if ge.TotalBlocks != 1 {
		t.Errorf("TotalBlocks = %d, want 1", ge.TotalBlocks)
	}
	if ge.TotalBypasses != 3 {
		t.Errorf("TotalBypasses = %d, want 3", ge.TotalBypasses)
	}

	// Block rate = 1/4 = 25%
	expectedBlockRate := 25.0
	if ge.BlockRate < expectedBlockRate-0.1 || ge.BlockRate > expectedBlockRate+0.1 {
		t.Errorf("BlockRate = %.1f%%, want ~%.1f%%", ge.BlockRate, expectedBlockRate)
	}

	// Blocked outcomes
	if ge.BlockedOutcomes.EscalatedToArchitect != 1 {
		t.Errorf("EscalatedToArchitect = %d, want 1", ge.BlockedOutcomes.EscalatedToArchitect)
	}
	if ge.BlockedOutcomes.EventuallyCompleted != 1 {
		t.Errorf("EventuallyCompleted = %d, want 1", ge.BlockedOutcomes.EventuallyCompleted)
	}
	if ge.BlockedOutcomes.StillPending != 0 {
		t.Errorf("StillPending = %d, want 0", ge.BlockedOutcomes.StillPending)
	}

	// Architect escalations
	if ge.ArchitectEscalations != 1 {
		t.Errorf("ArchitectEscalations = %d, want 1", ge.ArchitectEscalations)
	}

	// Gated quality: 4 spawns with gate decisions (gated-1, gated-2, gated-3, blocked-1)
	if ge.GatedCompletion.TotalSpawns != 4 {
		t.Errorf("GatedCompletion.TotalSpawns = %d, want 4", ge.GatedCompletion.TotalSpawns)
	}
	// 3 completions (gated-1, gated-2, blocked-1)
	if ge.GatedCompletion.Completions != 3 {
		t.Errorf("GatedCompletion.Completions = %d, want 3", ge.GatedCompletion.Completions)
	}
	// 1 abandonment (gated-3)
	if ge.GatedCompletion.Abandonments != 1 {
		t.Errorf("GatedCompletion.Abandonments = %d, want 1", ge.GatedCompletion.Abandonments)
	}
	// Completion rate = 3/4 = 75%
	expectedGatedRate := 75.0
	if ge.GatedCompletion.CompletionRate < expectedGatedRate-0.1 || ge.GatedCompletion.CompletionRate > expectedGatedRate+0.1 {
		t.Errorf("GatedCompletion.CompletionRate = %.1f%%, want ~%.1f%%", ge.GatedCompletion.CompletionRate, expectedGatedRate)
	}
	// 2 verification passed (gated-1, blocked-1)
	if ge.GatedCompletion.VerificationPassed != 2 {
		t.Errorf("GatedCompletion.VerificationPassed = %d, want 2", ge.GatedCompletion.VerificationPassed)
	}
	// Verification rate = 2/3 = 66.7% (of completions)
	expectedVerifRate := (2.0 / 3.0) * 100
	if ge.GatedCompletion.VerificationRate < expectedVerifRate-0.1 || ge.GatedCompletion.VerificationRate > expectedVerifRate+0.1 {
		t.Errorf("GatedCompletion.VerificationRate = %.1f%%, want ~%.1f%%", ge.GatedCompletion.VerificationRate, expectedVerifRate)
	}

	// Ungated quality: 2 spawns without gate decisions (ungated-1, ungated-2)
	if ge.UngatedCompletion.TotalSpawns != 2 {
		t.Errorf("UngatedCompletion.TotalSpawns = %d, want 2", ge.UngatedCompletion.TotalSpawns)
	}
	if ge.UngatedCompletion.Completions != 1 {
		t.Errorf("UngatedCompletion.Completions = %d, want 1", ge.UngatedCompletion.Completions)
	}
	if ge.UngatedCompletion.Abandonments != 1 {
		t.Errorf("UngatedCompletion.Abandonments = %d, want 1", ge.UngatedCompletion.Abandonments)
	}
	// Completion rate = 1/2 = 50%
	expectedUngatedRate := 50.0
	if ge.UngatedCompletion.CompletionRate < expectedUngatedRate-0.1 || ge.UngatedCompletion.CompletionRate > expectedUngatedRate+0.1 {
		t.Errorf("UngatedCompletion.CompletionRate = %.1f%%, want ~%.1f%%", ge.UngatedCompletion.CompletionRate, expectedUngatedRate)
	}
	// 1 verification passed (ungated-1)
	if ge.UngatedCompletion.VerificationPassed != 1 {
		t.Errorf("UngatedCompletion.VerificationPassed = %d, want 1", ge.UngatedCompletion.VerificationPassed)
	}
	// Verification rate = 1/1 = 100%
	if ge.UngatedCompletion.VerificationRate < 99.9 {
		t.Errorf("UngatedCompletion.VerificationRate = %.1f%%, want 100%%", ge.UngatedCompletion.VerificationRate)
	}
}

func TestGateEffectivenessStats_Empty(t *testing.T) {
	events := []StatsEvent{
		{Type: "session.spawned", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
	}

	report := aggregateStats(events, 7)

	if report.GateEffectivenessStats.TotalEvaluations != 0 {
		t.Errorf("TotalEvaluations = %d, want 0", report.GateEffectivenessStats.TotalEvaluations)
	}
}

func TestGateEffectivenessStats_AllowWithBeadsID(t *testing.T) {
	// Verify that "allow" gate decisions with beads_id correctly classify spawns as gated.
	// This is the most common path — most spawns pass gates without being blocked.
	now := time.Now().Unix()

	events := []StatsEvent{
		// Two spawns: one with gate_decision (allow), one without
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "gated-allow-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 6200, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "ungated-1",
		}},
		// Gate decisions: all "allow" with beads_id (typical daemon spawn path)
		{Type: "spawn.gate_decision", Timestamp: now - 7100, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "allow", "skill": "feature-impl", "beads_id": "gated-allow-1",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 7100, Data: map[string]interface{}{
			"gate_name": "verification", "decision": "allow", "skill": "feature-impl", "beads_id": "gated-allow-1",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 7100, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "feature-impl", "beads_id": "gated-allow-1",
		}},
		// Both complete
		{Type: "agent.completed", Timestamp: now - 3000, Data: map[string]interface{}{
			"beads_id": "gated-allow-1", "verification_passed": true, "skill": "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 2000, Data: map[string]interface{}{
			"beads_id": "ungated-1", "verification_passed": true, "skill": "investigation",
		}},
	}

	report := aggregateStats(events, 7)
	ge := report.GateEffectivenessStats

	// 3 allows, 0 blocks, 0 bypasses
	if ge.TotalAllows != 3 {
		t.Errorf("TotalAllows = %d, want 3", ge.TotalAllows)
	}
	if ge.TotalBlocks != 0 {
		t.Errorf("TotalBlocks = %d, want 0", ge.TotalBlocks)
	}

	// gated-allow-1 should be classified as gated
	if ge.GatedCompletion.TotalSpawns != 1 {
		t.Errorf("GatedCompletion.TotalSpawns = %d, want 1", ge.GatedCompletion.TotalSpawns)
	}
	if ge.GatedCompletion.Completions != 1 {
		t.Errorf("GatedCompletion.Completions = %d, want 1", ge.GatedCompletion.Completions)
	}

	// ungated-1 should be classified as ungated
	if ge.UngatedCompletion.TotalSpawns != 1 {
		t.Errorf("UngatedCompletion.TotalSpawns = %d, want 1", ge.UngatedCompletion.TotalSpawns)
	}
	if ge.UngatedCompletion.Completions != 1 {
		t.Errorf("UngatedCompletion.Completions = %d, want 1", ge.UngatedCompletion.Completions)
	}
}

func TestGateEffectivenessStats_BlockedStillPending(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Spawn that gets blocked but never completed
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "blocked-pending",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 7100, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "block", "skill": "feature-impl", "beads_id": "blocked-pending",
		}},
		// No completion or abandonment event
	}

	report := aggregateStats(events, 7)
	ge := report.GateEffectivenessStats

	if ge.TotalBlocks != 1 {
		t.Errorf("TotalBlocks = %d, want 1", ge.TotalBlocks)
	}
	if ge.BlockedOutcomes.StillPending != 1 {
		t.Errorf("StillPending = %d, want 1", ge.BlockedOutcomes.StillPending)
	}
	if ge.BlockedOutcomes.EventuallyCompleted != 0 {
		t.Errorf("EventuallyCompleted = %d, want 0", ge.BlockedOutcomes.EventuallyCompleted)
	}
}

func TestGateEffectivenessStats_PrecommitBlocksWithoutBeadsID(t *testing.T) {
	// Pre-commit gate_decision events lack beads_id (pre-commit hooks don't know
	// which agent is committing). Verify they're counted in gate decisions but
	// don't create phantom blocked outcomes.
	now := time.Now().Unix()

	events := []StatsEvent{
		// A spawn with a beads_id
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "test-1",
		}},
		// Pre-commit gate_decision: block with no beads_id (structural limitation)
		{Type: "spawn.gate_decision", Timestamp: now - 3600, Data: map[string]interface{}{
			"gate_name": "accretion_precommit", "decision": "block",
			"reason": "file exceeds accretion threshold",
			"target_files": []interface{}{"cmd/orch/stats_test.go"},
		}},
		// Pre-commit gate_decision: bypass with no beads_id
		{Type: "spawn.gate_decision", Timestamp: now - 3500, Data: map[string]interface{}{
			"gate_name": "accretion_precommit", "decision": "bypass",
			"reason": "FORCE_ACCRETION=1",
		}},
		// Spawn gate decisions WITH beads_id (normal spawn path)
		{Type: "spawn.gate_decision", Timestamp: now - 7100, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "allow", "skill": "feature-impl", "beads_id": "test-1",
		}},
		// Completion
		{Type: "agent.completed", Timestamp: now - 1800, Data: map[string]interface{}{
			"beads_id": "test-1", "verification_passed": true, "skill": "feature-impl",
		}},
	}

	report := aggregateStats(events, 7)
	ge := report.GateEffectivenessStats

	// Gate decisions should count all events (with and without beads_id)
	if report.GateDecisionStats.TotalDecisions != 3 {
		t.Errorf("TotalDecisions = %d, want 3", report.GateDecisionStats.TotalDecisions)
	}

	// Blocks should include the precommit block
	if report.GateDecisionStats.TotalBlocks != 1 {
		t.Errorf("TotalBlocks = %d, want 1", report.GateDecisionStats.TotalBlocks)
	}

	// But blocked outcomes should be 0 (precommit block has no beads_id to correlate)
	if ge.BlockedOutcomes.StillPending != 0 {
		t.Errorf("StillPending = %d, want 0 (precommit blocks can't correlate)", ge.BlockedOutcomes.StillPending)
	}
	if ge.BlockedOutcomes.EventuallyCompleted != 0 {
		t.Errorf("EventuallyCompleted = %d, want 0", ge.BlockedOutcomes.EventuallyCompleted)
	}

	// Gated cohort should include test-1 (has gate_decision with beads_id)
	if ge.GatedCompletion.TotalSpawns != 1 {
		t.Errorf("GatedCompletion.TotalSpawns = %d, want 1", ge.GatedCompletion.TotalSpawns)
	}
}

func TestGateEffectivenessStats_ProspectiveBaseline(t *testing.T) {
	// Simulates the prospective measurement scenario: a mix of daemon-spawned
	// (gated, with beads_id) and manual spawns (some without beads_id).
	// Verifies the gated vs ungated comparison produces correct metrics
	// for the baseline period.
	now := time.Now().Unix()

	events := []StatsEvent{
		// --- Daemon spawns (all have beads_id and gate_decision events) ---
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 86400, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "daemon-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 82800, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "daemon-2",
		}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 79200, Data: map[string]interface{}{
			"skill": "systematic-debugging", "beads_id": "daemon-3",
		}},
		// Gate decisions for daemon spawns (triage allow + hotspot allow per spawn)
		{Type: "spawn.gate_decision", Timestamp: now - 86300, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "allow", "skill": "feature-impl", "beads_id": "daemon-1",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 86300, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "feature-impl", "beads_id": "daemon-1",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 82700, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "allow", "skill": "feature-impl", "beads_id": "daemon-2",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 82700, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "bypass", "skill": "feature-impl", "beads_id": "daemon-2",
			"reason": "architect reviewed extraction plan",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 79100, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "allow", "skill": "systematic-debugging", "beads_id": "daemon-3",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 79100, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "systematic-debugging", "beads_id": "daemon-3",
		}},

		// --- Manual spawns (no gate_decision with beads_id) ---
		{Type: "session.spawned", SessionID: "ses_4", Timestamp: now - 75600, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "manual-1",
		}},
		{Type: "session.spawned", SessionID: "ses_5", Timestamp: now - 72000, Data: map[string]interface{}{
			"skill": "architect", "beads_id": "manual-2",
		}},

		// --- Completions ---
		{Type: "agent.completed", Timestamp: now - 80000, Data: map[string]interface{}{
			"beads_id": "daemon-1", "verification_passed": true, "skill": "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 77000, Data: map[string]interface{}{
			"beads_id": "daemon-2", "verification_passed": false, "forced": true, "skill": "feature-impl",
		}},
		{Type: "agent.completed", Timestamp: now - 74000, Data: map[string]interface{}{
			"beads_id": "daemon-3", "verification_passed": true, "skill": "systematic-debugging",
		}},
		{Type: "agent.completed", Timestamp: now - 70000, Data: map[string]interface{}{
			"beads_id": "manual-1", "verification_passed": true, "skill": "investigation",
		}},
		{Type: "agent.abandoned", Timestamp: now - 68000, Data: map[string]interface{}{
			"beads_id": "manual-2",
		}},
	}

	report := aggregateStats(events, 7)
	ge := report.GateEffectivenessStats

	// 6 gate decision events total (2 per daemon spawn)
	if report.GateDecisionStats.TotalDecisions != 6 {
		t.Errorf("TotalDecisions = %d, want 6", report.GateDecisionStats.TotalDecisions)
	}

	// Gated cohort: 3 daemon spawns (all have gate_decision events with beads_id)
	if ge.GatedCompletion.TotalSpawns != 3 {
		t.Errorf("GatedCompletion.TotalSpawns = %d, want 3", ge.GatedCompletion.TotalSpawns)
	}
	if ge.GatedCompletion.Completions != 3 {
		t.Errorf("GatedCompletion.Completions = %d, want 3", ge.GatedCompletion.Completions)
	}
	if ge.GatedCompletion.VerificationPassed != 2 {
		t.Errorf("GatedCompletion.VerificationPassed = %d, want 2", ge.GatedCompletion.VerificationPassed)
	}
	// Verification rate = 2/3 = 66.7%
	expectedVerifRate := (2.0 / 3.0) * 100
	if ge.GatedCompletion.VerificationRate < expectedVerifRate-0.1 || ge.GatedCompletion.VerificationRate > expectedVerifRate+0.1 {
		t.Errorf("GatedCompletion.VerificationRate = %.1f%%, want ~%.1f%%", ge.GatedCompletion.VerificationRate, expectedVerifRate)
	}

	// Ungated cohort: 2 manual spawns (no gate_decision events)
	if ge.UngatedCompletion.TotalSpawns != 2 {
		t.Errorf("UngatedCompletion.TotalSpawns = %d, want 2", ge.UngatedCompletion.TotalSpawns)
	}
	if ge.UngatedCompletion.Completions != 1 {
		t.Errorf("UngatedCompletion.Completions = %d, want 1", ge.UngatedCompletion.Completions)
	}
	if ge.UngatedCompletion.Abandonments != 1 {
		t.Errorf("UngatedCompletion.Abandonments = %d, want 1", ge.UngatedCompletion.Abandonments)
	}

	// Duration should be calculable for gated completions
	if ge.GatedCompletion.AvgDurationMinutes <= 0 {
		t.Errorf("GatedCompletion.AvgDurationMinutes = %.1f, want > 0", ge.GatedCompletion.AvgDurationMinutes)
	}
}

func TestGateAccuracyBaseline(t *testing.T) {
	// Tests the baseline snapshot generation — the data structure that
	// records gate accuracy metrics at a point in time for future comparison.
	now := time.Now().Unix()

	events := []StatsEvent{
		// Spawns
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7200, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "b-1",
		}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 6200, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "b-2",
		}},
		// Gate decisions
		{Type: "spawn.gate_decision", Timestamp: now - 7100, Data: map[string]interface{}{
			"gate_name": "triage", "decision": "allow", "beads_id": "b-1",
		}},
		{Type: "spawn.gate_decision", Timestamp: now - 7100, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "beads_id": "b-1",
		}},
		// Completions
		{Type: "agent.completed", Timestamp: now - 3600, Data: map[string]interface{}{
			"beads_id": "b-1", "verification_passed": true,
		}},
		{Type: "agent.completed", Timestamp: now - 3000, Data: map[string]interface{}{
			"beads_id": "b-2", "verification_passed": true,
		}},
	}

	report := aggregateStats(events, 7)
	baseline := extractGateAccuracyBaseline(report)

	if baseline.TotalSpawns != 2 {
		t.Errorf("TotalSpawns = %d, want 2", baseline.TotalSpawns)
	}
	if baseline.GatedSpawns != 1 {
		t.Errorf("GatedSpawns = %d, want 1", baseline.GatedSpawns)
	}
	if baseline.UngatedSpawns != 1 {
		t.Errorf("UngatedSpawns = %d, want 1", baseline.UngatedSpawns)
	}
	if baseline.GateDecisions != 2 {
		t.Errorf("GateDecisions = %d, want 2", baseline.GateDecisions)
	}
	if baseline.GatedVerificationRate < 99.9 {
		t.Errorf("GatedVerificationRate = %.1f%%, want 100%%", baseline.GatedVerificationRate)
	}
	if baseline.UngatedVerificationRate < 99.9 {
		t.Errorf("UngatedVerificationRate = %.1f%%, want 100%%", baseline.UngatedVerificationRate)
	}
	if baseline.SnapshotTime == "" {
		t.Error("SnapshotTime should not be empty")
	}
}

func TestSkillInferenceStats(t *testing.T) {
	now := time.Now().Unix()

	events := []StatsEvent{
		// Skill inferences (daemon inferred skills for issues)
		{Type: "spawn.skill_inferred", SessionID: "issue-1", Timestamp: now - 7200, Data: map[string]interface{}{
			"issue_id": "issue-1", "inferred_skill": "feature-impl", "issue_type": "feature",
			"had_skill_label": false, "had_title_match": false, "used_description_heuristic": false,
		}},
		{Type: "spawn.skill_inferred", SessionID: "issue-2", Timestamp: now - 7100, Data: map[string]interface{}{
			"issue_id": "issue-2", "inferred_skill": "systematic-debugging", "issue_type": "bug",
			"had_skill_label": false, "had_title_match": true, "used_description_heuristic": false,
		}},
		{Type: "spawn.skill_inferred", SessionID: "issue-3", Timestamp: now - 7000, Data: map[string]interface{}{
			"issue_id": "issue-3", "inferred_skill": "investigation", "issue_type": "task",
			"had_skill_label": true, "had_title_match": false, "used_description_heuristic": false,
		}},
		{Type: "spawn.skill_inferred", SessionID: "issue-4", Timestamp: now - 6900, Data: map[string]interface{}{
			"issue_id": "issue-4", "inferred_skill": "feature-impl", "issue_type": "task",
			"had_skill_label": false, "had_title_match": false, "used_description_heuristic": true,
		}},
		// Spawns
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7000, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "issue-1"}},
		{Type: "session.spawned", SessionID: "ses_2", Timestamp: now - 6900, Data: map[string]interface{}{"skill": "systematic-debugging", "beads_id": "issue-2"}},
		{Type: "session.spawned", SessionID: "ses_3", Timestamp: now - 6800, Data: map[string]interface{}{"skill": "investigation", "beads_id": "issue-3"}},
		{Type: "session.spawned", SessionID: "ses_4", Timestamp: now - 6700, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "issue-4"}},
		// Outcomes: issue-1 completed, issue-2 completed, issue-3 abandoned, issue-4 completed
		{Type: "agent.completed", Timestamp: now - 3600, Data: map[string]interface{}{"beads_id": "issue-1", "skill": "feature-impl"}},
		{Type: "agent.completed", Timestamp: now - 3500, Data: map[string]interface{}{"beads_id": "issue-2", "skill": "systematic-debugging"}},
		{Type: "agent.abandoned", Timestamp: now - 3400, Data: map[string]interface{}{"beads_id": "issue-3"}},
		{Type: "agent.completed", Timestamp: now - 3300, Data: map[string]interface{}{"beads_id": "issue-4", "skill": "feature-impl"}},
	}

	report := aggregateStats(events, 7)
	si := report.SkillInferenceStats

	// 4 inferences, 3 completed, 1 abandoned
	if si.TotalInferences != 4 {
		t.Errorf("TotalInferences = %d, want 4", si.TotalInferences)
	}
	if si.Completed != 3 {
		t.Errorf("Completed = %d, want 3", si.Completed)
	}
	if si.Abandoned != 1 {
		t.Errorf("Abandoned = %d, want 1", si.Abandoned)
	}
	if si.CompletionRate < 74.9 || si.CompletionRate > 75.1 {
		t.Errorf("CompletionRate = %.1f%%, want 75.0%%", si.CompletionRate)
	}

	// By method: type=1 completed (issue-1), title=1 completed (issue-2),
	// label=1 abandoned (issue-3), description=1 completed (issue-4)
	if len(si.ByMethod) == 0 {
		t.Fatal("expected ByMethod to be populated")
	}
	methodMap := make(map[string]InferenceMethodStats)
	for _, m := range si.ByMethod {
		methodMap[m.Method] = m
	}
	if ms, ok := methodMap["type"]; !ok {
		t.Error("expected 'type' method in ByMethod")
	} else {
		if ms.Inferences != 1 {
			t.Errorf("type method: Inferences = %d, want 1", ms.Inferences)
		}
		if ms.Completed != 1 {
			t.Errorf("type method: Completed = %d, want 1", ms.Completed)
		}
		if ms.CompletionRate < 99.9 {
			t.Errorf("type method: CompletionRate = %.1f%%, want 100%%", ms.CompletionRate)
		}
	}
	if ms, ok := methodMap["label"]; !ok {
		t.Error("expected 'label' method in ByMethod")
	} else {
		if ms.Abandoned != 1 {
			t.Errorf("label method: Abandoned = %d, want 1", ms.Abandoned)
		}
		if ms.CompletionRate > 0.1 {
			t.Errorf("label method: CompletionRate = %.1f%%, want 0%%", ms.CompletionRate)
		}
	}

	// By skill: feature-impl=2 completed, systematic-debugging=1 completed, investigation=1 abandoned
	if len(si.BySkill) == 0 {
		t.Fatal("expected BySkill to be populated")
	}
	skillMap := make(map[string]InferenceSkillStats)
	for _, s := range si.BySkill {
		skillMap[s.Skill] = s
	}
	if ss, ok := skillMap["feature-impl"]; !ok {
		t.Error("expected 'feature-impl' in BySkill")
	} else {
		if ss.Inferences != 2 {
			t.Errorf("feature-impl: Inferences = %d, want 2", ss.Inferences)
		}
		if ss.Completed != 2 {
			t.Errorf("feature-impl: Completed = %d, want 2", ss.Completed)
		}
		if ss.CompletionRate < 99.9 {
			t.Errorf("feature-impl: CompletionRate = %.1f%%, want 100%%", ss.CompletionRate)
		}
	}
	if ss, ok := skillMap["investigation"]; !ok {
		t.Error("expected 'investigation' in BySkill")
	} else {
		if ss.Abandoned != 1 {
			t.Errorf("investigation: Abandoned = %d, want 1", ss.Abandoned)
		}
	}
}

func TestSkillInferenceStatsNoOutcome(t *testing.T) {
	now := time.Now().Unix()

	// Inference with no completion or abandonment — should not be counted
	events := []StatsEvent{
		{Type: "spawn.skill_inferred", SessionID: "issue-1", Timestamp: now - 7200, Data: map[string]interface{}{
			"issue_id": "issue-1", "inferred_skill": "feature-impl", "issue_type": "feature",
			"had_skill_label": false, "had_title_match": false, "used_description_heuristic": false,
		}},
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: now - 7000, Data: map[string]interface{}{"skill": "feature-impl", "beads_id": "issue-1"}},
		// No completion or abandonment
	}

	report := aggregateStats(events, 7)
	if report.SkillInferenceStats.TotalInferences != 0 {
		t.Errorf("TotalInferences = %d, want 0 (no outcomes yet)", report.SkillInferenceStats.TotalInferences)
	}
}

func TestCollectNValueMetrics(t *testing.T) {
	tmpDir := t.TempDir()

	// Create events.jsonl with known line count
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	eventsContent := `{"type":"session.spawned","timestamp":1}
{"type":"agent.completed","timestamp":2}
{"type":"daemon.spawn","timestamp":3}
`
	os.WriteFile(eventsPath, []byte(eventsContent), 0644)

	// Create KB directory with .md files
	kbDir := filepath.Join(tmpDir, ".kb")
	os.MkdirAll(filepath.Join(kbDir, "models"), 0755)
	os.WriteFile(filepath.Join(kbDir, "model.md"), []byte("# Model"), 0644)
	os.WriteFile(filepath.Join(kbDir, "models", "sub.md"), []byte("# Sub"), 0644)
	os.WriteFile(filepath.Join(kbDir, "readme.txt"), []byte("not counted"), 0644)

	metrics := collectNValueMetrics(eventsPath)

	if metrics.EventCount != 3 {
		t.Errorf("EventCount = %d, want 3", metrics.EventCount)
	}
	// KB count depends on cwd, so we test the helpers directly
}

func TestCountFilesRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "a.md"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.md"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "sub", "c.md"), []byte("c"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "d.txt"), []byte("d"), 0644)

	count := countFilesRecursive(tmpDir, ".md")
	if count != 3 {
		t.Errorf("countFilesRecursive = %d, want 3", count)
	}
}

func TestCountDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "ws-1"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "ws-2"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "archived"), 0755) // should be excluded
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("not a dir"), 0644)

	count := countDirectories(tmpDir)
	if count != 2 {
		t.Errorf("countDirectories = %d, want 2 (excluding archived)", count)
	}
}

func TestCountDirectories_EmptyDir(t *testing.T) {
	count := countDirectories("/nonexistent/path")
	if count != 0 {
		t.Errorf("countDirectories on nonexistent = %d, want 0", count)
	}
}

func TestCountFilesRecursive_EmptyDir(t *testing.T) {
	count := countFilesRecursive("/nonexistent/path", ".md")
	if count != 0 {
		t.Errorf("countFilesRecursive on nonexistent = %d, want 0", count)
	}
}
