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

func TestFormatOrientation_ActivePlans(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ActivePlans: []PlanSummary{
			{
				Title:    "Ship Dashboard V2",
				Projects: []string{"orch-go", "price-watch"},
				TLDR:     "Multi-project dashboard rewrite.",
				Phases: []PlanPhase{
					{Name: "Phase 1: Foundation", Status: "complete"},
					{Name: "Phase 2: Agent Cards", Status: "in-progress"},
					{Name: "Phase 3: Plan Panel", Status: "ready"},
				},
			},
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Active plans:") {
		t.Error("missing 'Active plans:' section header")
	}
	if !strings.Contains(output, "Ship Dashboard V2") {
		t.Error("missing plan title")
	}
	if !strings.Contains(output, "[orch-go, price-watch]") {
		t.Error("missing projects list")
	}
	if !strings.Contains(output, "Multi-project dashboard rewrite.") {
		t.Error("missing TLDR")
	}
	if !strings.Contains(output, "[x] Phase 1: Foundation") {
		t.Error("missing completed phase marker")
	}
	if !strings.Contains(output, "[>] Phase 2: Agent Cards") {
		t.Error("missing in-progress phase marker")
	}
	if !strings.Contains(output, "[ ] Phase 3: Plan Panel") {
		t.Error("missing ready phase marker")
	}
}

func TestFormatOrientation_NoPlans(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
	}

	output := FormatOrientation(data)

	if strings.Contains(output, "Active plans") {
		t.Error("plans section should not appear when no active plans")
	}
}

func TestFormatOrientation_ActivePlansWithBeadsProgress(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ActivePlans: []PlanSummary{
			{
				Title:    "Gate Signal vs Noise",
				Progress: "1/4 complete",
				Phases: []PlanPhase{
					{Name: "Phase 1: Gate census", Status: "complete", BeadsIDs: []string{"orch-go-a1b2"}},
					{Name: "Phase 2: Fix noise gates", Status: "in-progress", BeadsIDs: []string{"orch-go-c3d4"}},
					{Name: "Phase 3: Retrospective audit", Status: "ready", BeadsIDs: []string{"orch-go-e5f6"}},
					{Name: "Phase 4: Prospective measurement", Status: ""},
				},
			},
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Active plans:") {
		t.Error("missing 'Active plans:' section header")
	}
	if !strings.Contains(output, "Gate Signal vs Noise (1/4 complete)") {
		t.Errorf("missing plan title with progress, got:\n%s", output)
	}
	if !strings.Contains(output, "[x] Phase 1: Gate census") {
		t.Error("missing completed phase marker")
	}
	if !strings.Contains(output, "[>] Phase 2: Fix noise gates") {
		t.Error("missing in-progress phase marker")
	}
	if !strings.Contains(output, "[ ] Phase 3: Retrospective audit") {
		t.Error("missing ready phase marker")
	}
	if !strings.Contains(output, "[ ] Phase 4: Prospective measurement") {
		t.Error("missing unhydrated phase marker")
	}
}

func TestFormatOrientation_ActivePlansNoProgress(t *testing.T) {
	// Plans without beads progress should still work (no progress shown)
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ActivePlans: []PlanSummary{
			{
				Title: "Unhydrated Plan",
				Phases: []PlanPhase{
					{Name: "Phase 1: Setup", Status: ""},
				},
			},
		},
	}

	output := FormatOrientation(data)

	// Title should appear without progress suffix
	if !strings.Contains(output, "- Unhydrated Plan") {
		t.Errorf("missing plan title, got:\n%s", output)
	}
	// Should NOT show "()" or "(0/0 complete)"
	if strings.Contains(output, "( complete)") || strings.Contains(output, "(0/") {
		t.Error("should not show empty progress")
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

func TestFormatOrientation_ActiveThreads(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ActiveThreads: []ActiveThread{
			{
				Name:        "enforcement-comprehension",
				Title:       "How enforcement and comprehension relate",
				Updated:     "2026-03-05",
				EntryCount:  3,
				LatestEntry: "The distinction is clearer now — enforcement gates vs comprehension probes",
			},
			{
				Name:        "daemon-capacity",
				Title:       "Daemon capacity planning",
				Updated:     "2026-03-04",
				EntryCount:  1,
				LatestEntry: "Initial thoughts on scaling",
			},
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Active threads:") {
		t.Error("missing 'Active threads:' section header")
	}
	if !strings.Contains(output, "How enforcement and comprehension relate") {
		t.Error("missing thread title")
	}
	if !strings.Contains(output, "updated 2026-03-05") {
		t.Error("missing thread updated date")
	}
	if !strings.Contains(output, "3 entries") {
		t.Error("missing thread entry count")
	}
	if !strings.Contains(output, "The distinction is clearer now") {
		t.Error("missing thread latest entry preview")
	}
	if !strings.Contains(output, "Daemon capacity planning") {
		t.Error("missing second thread title")
	}
}

func TestFormatOrientation_NoThreads(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
	}

	output := FormatOrientation(data)

	if strings.Contains(output, "Active threads") {
		t.Error("threads section should not appear when no active threads")
	}
}

func TestFormatOrientation_HealthSummary(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		HealthSummary: &HealthSummary{
			OpenIssues:    45,
			BlockedIssues: 12,
			StaleIssues:   3,
			BloatedFiles:  5,
			FixFeatRatio:  1.2,
			Alerts: []HealthAlert{
				{Message: "Fix:feat ratio is 1.2 — approaching maintenance mode", Level: "warn"},
			},
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Health:") {
		t.Error("missing 'Health:' section header")
	}
	if !strings.Contains(output, "Open: 45") {
		t.Error("missing open issues count")
	}
	if !strings.Contains(output, "Blocked: 12") {
		t.Error("missing blocked issues count")
	}
	if !strings.Contains(output, "Stale: 3") {
		t.Error("missing stale issues count")
	}
	if !strings.Contains(output, "Bloated files: 5") {
		t.Error("missing bloated files count")
	}
	if !strings.Contains(output, "Fix:feat 1.2") {
		t.Error("missing fix:feat ratio")
	}
	if !strings.Contains(output, "approaching maintenance mode") {
		t.Error("missing health alert message")
	}
}

func TestFormatOrientation_HealthSummaryNoAlerts(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		HealthSummary: &HealthSummary{
			OpenIssues:    10,
			BlockedIssues: 2,
			StaleIssues:   0,
			BloatedFiles:  1,
			FixFeatRatio:  0.5,
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Health:") {
		t.Error("missing 'Health:' section header")
	}
	// No alerts - should not have warning markers
	if strings.Contains(output, "warn") || strings.Contains(output, "critical") {
		t.Error("should not show alert levels when no alerts")
	}
}

func TestFormatOrientation_NoHealthSummary(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
	}

	output := FormatOrientation(data)

	if strings.Contains(output, "Health:") {
		t.Error("health section should not appear when nil")
	}
}

func TestHealthSummaryJSON(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		HealthSummary: &HealthSummary{
			OpenIssues:    45,
			BlockedIssues: 12,
			StaleIssues:   3,
			BloatedFiles:  5,
			FixFeatRatio:  1.2,
			Alerts: []HealthAlert{
				{Message: "test alert", Level: "warn"},
			},
		},
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(b)
	if !strings.Contains(jsonStr, `"health_summary"`) {
		t.Error("JSON missing health_summary field")
	}
	if !strings.Contains(jsonStr, `"open_issues":45`) {
		t.Error("JSON missing open_issues")
	}
	if !strings.Contains(jsonStr, `"alerts"`) {
		t.Error("JSON missing alerts")
	}

	// Verify omitempty works when nil
	data2 := &OrientationData{Throughput: Throughput{Days: 1}}
	b2, _ := json.Marshal(data2)
	if strings.Contains(string(b2), "health_summary") {
		t.Error("health_summary should be omitted when nil")
	}
}

func TestActiveThreadJSON(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ActiveThreads: []ActiveThread{
			{Name: "test-thread", Title: "Test", Updated: "2026-03-05", EntryCount: 2, LatestEntry: "latest"},
		},
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(b)
	if !strings.Contains(jsonStr, `"active_threads"`) {
		t.Error("JSON missing active_threads field")
	}
	if !strings.Contains(jsonStr, `"name":"test-thread"`) {
		t.Error("JSON missing thread name")
	}

	// Verify omitempty works
	data2 := &OrientationData{Throughput: Throughput{Days: 1}}
	b2, _ := json.Marshal(data2)
	if strings.Contains(string(b2), "active_threads") {
		t.Error("active_threads should be omitted when nil")
	}
}

func TestFormatOrientation_ReflectSummary(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ReflectSummary: &ReflectSummary{
			Total:      112,
			Synthesis:  46,
			Stale:      66,
			Agreements: 0,
			TopClusters: []ReflectCluster{
				{Topic: "context", Count: 7},
				{Topic: "reflect", Count: 4},
				{Topic: "config", Count: 3},
			},
			Age: "2h ago",
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Reflection suggestions:") {
		t.Error("missing 'Reflection suggestions:' section header")
	}
	if !strings.Contains(output, "112 items need attention") {
		t.Error("missing total count")
	}
	if !strings.Contains(output, "from 2h ago") {
		t.Error("missing age")
	}
	if !strings.Contains(output, "46 synthesis opportunities") {
		t.Error("missing synthesis count")
	}
	if !strings.Contains(output, "66 stale decisions") {
		t.Error("missing stale count")
	}
	if !strings.Contains(output, "context(7)") {
		t.Error("missing top cluster")
	}
}

func TestFormatOrientation_ReflectSummaryNil(t *testing.T) {
	data := &OrientationData{Throughput: Throughput{Days: 1}}
	output := FormatOrientation(data)
	if strings.Contains(output, "Reflection suggestions") {
		t.Error("reflect section should not appear when nil")
	}
}

func TestFormatOrientation_ReflectSummaryEmpty(t *testing.T) {
	data := &OrientationData{
		Throughput:     Throughput{Days: 1},
		ReflectSummary: &ReflectSummary{Total: 0},
	}
	output := FormatOrientation(data)
	if strings.Contains(output, "Reflection suggestions") {
		t.Error("reflect section should not appear when total is 0")
	}
}

func TestFormatOrientation_ReflectSummaryOrphanRate(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ReflectSummary: &ReflectSummary{
			Total:       5,
			Synthesis:   5,
			OrphanRate:  52.0,
			OrphanTotal: 196,
		},
	}
	output := FormatOrientation(data)
	if !strings.Contains(output, "Orphan rate: 52.0%") {
		t.Errorf("missing orphan rate, got:\n%s", output)
	}
	if !strings.Contains(output, "196 investigations") {
		t.Errorf("missing orphan total, got:\n%s", output)
	}
}

func TestFormatOrientation_UsageWarning(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		UsageWarning: &UsageWarning{
			Utilization: 92,
			Remaining:   "8%",
			ResetTime:   "2d 4h",
			Level:       "HIGH",
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Usage HIGH: 92%") {
		t.Error("missing usage warning header")
	}
	if !strings.Contains(output, "8% remaining") {
		t.Error("missing remaining percentage")
	}
	if !strings.Contains(output, "Resets in: 2d 4h") {
		t.Error("missing reset time")
	}
}

func TestFormatOrientation_UsageWarningNil(t *testing.T) {
	data := &OrientationData{Throughput: Throughput{Days: 1}}
	output := FormatOrientation(data)
	if strings.Contains(output, "Usage") && strings.Contains(output, "weekly limit") {
		t.Error("usage section should not appear when nil")
	}
}

func TestFormatOrientation_ConfigDrift(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ConfigDrift: []ConfigDriftItem{
			{File: "settings.json", Reason: "not a symlink"},
			{File: "CLAUDE.md", Reason: "points to /tmp/other"},
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Config drift detected:") {
		t.Error("missing 'Config drift detected:' section header")
	}
	if !strings.Contains(output, "settings.json (not a symlink)") {
		t.Error("missing drift item")
	}
	if !strings.Contains(output, "CLAUDE.md (points to /tmp/other)") {
		t.Error("missing second drift item")
	}
	if !strings.Contains(output, "Fix: ln -sf") {
		t.Error("missing fix instructions")
	}
}

func TestFormatOrientation_ConfigDriftEmpty(t *testing.T) {
	data := &OrientationData{Throughput: Throughput{Days: 1}}
	output := FormatOrientation(data)
	if strings.Contains(output, "Config drift") {
		t.Error("config drift section should not appear when empty")
	}
}

func TestFormatOrientation_SessionResume(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		SessionResume: &SessionResume{
			Content: "# Session Handoff\n\nLast session worked on X.\nNext: do Y.",
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Session resumed:") {
		t.Error("missing 'Session resumed:' section header")
	}
	if !strings.Contains(output, "Session Handoff") {
		t.Error("missing handoff content")
	}
	if !strings.Contains(output, "Next: do Y") {
		t.Error("missing handoff continuation")
	}
}

func TestFormatOrientation_SessionResumeNil(t *testing.T) {
	data := &OrientationData{Throughput: Throughput{Days: 1}}
	output := FormatOrientation(data)
	if strings.Contains(output, "Session resumed") {
		t.Error("session resume section should not appear when nil")
	}
}

func TestReflectSummaryJSON(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ReflectSummary: &ReflectSummary{
			Total:     10,
			Synthesis: 5,
			Stale:     5,
		},
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(b)
	if !strings.Contains(jsonStr, `"reflect_summary"`) {
		t.Error("JSON missing reflect_summary field")
	}
	if !strings.Contains(jsonStr, `"total":10`) {
		t.Error("JSON missing total")
	}

	// Verify omitempty
	data2 := &OrientationData{Throughput: Throughput{Days: 1}}
	b2, _ := json.Marshal(data2)
	if strings.Contains(string(b2), "reflect_summary") {
		t.Error("reflect_summary should be omitted when nil")
	}
}

func TestConfigDriftJSON(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		ConfigDrift: []ConfigDriftItem{
			{File: "settings.json", Reason: "not a symlink"},
		},
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(b)
	if !strings.Contains(jsonStr, `"config_drift"`) {
		t.Error("JSON missing config_drift field")
	}
	if !strings.Contains(jsonStr, `"file":"settings.json"`) {
		t.Error("JSON missing file field")
	}

	// Verify omitempty
	data2 := &OrientationData{Throughput: Throughput{Days: 1}}
	b2, _ := json.Marshal(data2)
	if strings.Contains(string(b2), "config_drift") {
		t.Error("config_drift should be omitted when nil")
	}
}

func TestSessionResumeJSON(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		SessionResume: &SessionResume{
			Content: "handoff content",
			Source:  "/path/to/handoff.md",
		},
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(b)
	if !strings.Contains(jsonStr, `"session_resume"`) {
		t.Error("JSON missing session_resume field")
	}
	if !strings.Contains(jsonStr, `"content":"handoff content"`) {
		t.Error("JSON missing content field")
	}

	// Verify omitempty
	data2 := &OrientationData{Throughput: Throughput{Days: 1}}
	b2, _ := json.Marshal(data2)
	if strings.Contains(string(b2), "session_resume") {
		t.Error("session_resume should be omitted when nil")
	}
}

func TestUsageWarningJSON(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		UsageWarning: &UsageWarning{
			Utilization: 85,
			Level:       "WARNING",
		},
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(b)
	if !strings.Contains(jsonStr, `"usage_warning"`) {
		t.Error("JSON missing usage_warning field")
	}

	// Verify omitempty
	data2 := &OrientationData{Throughput: Throughput{Days: 1}}
	b2, _ := json.Marshal(data2)
	if strings.Contains(string(b2), "usage_warning") {
		t.Error("usage_warning should be omitted when nil")
	}
}

func TestFormatOrientation_SectionOrder(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1, Completions: 1},
		SessionResume: &SessionResume{Content: "resume content"},
		ConfigDrift:   []ConfigDriftItem{{File: "test", Reason: "drift"}},
		UsageWarning:  &UsageWarning{Utilization: 90, Remaining: "10%", Level: "HIGH"},
		ReflectSummary: &ReflectSummary{Total: 5, Synthesis: 5},
		FocusGoal:     "Test focus",
	}

	output := FormatOrientation(data)

	// Session resume should come before throughput
	resumeIdx := strings.Index(output, "Session resumed:")
	throughputIdx := strings.Index(output, "Last 24h:")
	if resumeIdx > throughputIdx {
		t.Error("session resume should appear before throughput")
	}

	// Config drift before throughput
	driftIdx := strings.Index(output, "Config drift detected:")
	if driftIdx > throughputIdx {
		t.Error("config drift should appear before throughput")
	}

	// Usage warning before throughput
	usageIdx := strings.Index(output, "Usage HIGH:")
	if usageIdx > throughputIdx {
		t.Error("usage warning should appear before throughput")
	}

	// Reflect summary should come after focus
	reflectIdx := strings.Index(output, "Reflection suggestions:")
	focusIdx := strings.Index(output, "Focus:")
	if reflectIdx < focusIdx {
		t.Error("reflection suggestions should appear after focus")
	}
}

func TestFormatThroughput_NetLinesDisplay(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{
			Days:            1,
			Completions:     5,
			Abandonments:    1,
			InProgress:      3,
			AvgDurationMin:  30,
			NetLinesAdded:   200,
			NetLinesRemoved: 50,
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Completions: 5") {
		t.Error("missing completions line")
	}
	if !strings.Contains(output, "Net lines: +150") {
		t.Errorf("missing or wrong net lines, got:\n%s", output)
	}
	if strings.Contains(output, "Merged:") {
		t.Error("should not show Merged line (removed)")
	}
}

func TestFormatThroughput_NoNetLinesWhenZero(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{
			Days:        1,
			Completions: 5,
			InProgress:  2,
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Completions: 5") {
		t.Error("missing completions")
	}
	if strings.Contains(output, "Net lines") {
		t.Error("should not show Net lines when zero")
	}
}

func TestFormatOrientation_DaemonHealthNonGreen(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		DaemonHealth: &DaemonHealthView{
			Signals: []DaemonHealthSignalView{
				{Name: "Daemon Liveness", Level: "green", Detail: "polling normally"},
				{Name: "Capacity", Level: "red", Detail: "3/3 slots used, 10 queued"},
				{Name: "Queue Depth", Level: "yellow", Detail: "30 issues ready"},
				{Name: "Evidence Check", Level: "green", Detail: "4 completions before pause"},
				{Name: "Unresponsive", Level: "green", Detail: "all agents responsive"},
				{Name: "Questions", Level: "yellow", Detail: "2 agent(s) waiting for input"},
			},
		},
	}

	output := FormatOrientation(data)

	if !strings.Contains(output, "Daemon health:") {
		t.Error("missing 'Daemon health:' section header")
	}
	if !strings.Contains(output, "[!!!] Capacity: 3/3 slots used, 10 queued") {
		t.Error("missing red capacity signal")
	}
	if !strings.Contains(output, "[!] Queue Depth: 30 issues ready") {
		t.Error("missing yellow queue depth signal")
	}
	if !strings.Contains(output, "[!] Questions: 2 agent(s) waiting for input") {
		t.Error("missing yellow questions signal")
	}
	// Green signals should not appear
	if strings.Contains(output, "Daemon Liveness") {
		t.Error("green Daemon Liveness should not appear")
	}
	if strings.Contains(output, "Evidence Check") {
		t.Error("green Evidence Check should not appear")
	}
}

func TestFormatOrientation_DaemonHealthAllGreen(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		DaemonHealth: &DaemonHealthView{
			Signals: []DaemonHealthSignalView{
				{Name: "Daemon Liveness", Level: "green", Detail: "polling normally"},
				{Name: "Capacity", Level: "green", Detail: "0/3 slots used"},
				{Name: "Queue Depth", Level: "green", Detail: "5 issues ready"},
				{Name: "Evidence Check", Level: "green", Detail: "4 completions before pause"},
				{Name: "Unresponsive", Level: "green", Detail: "all agents responsive"},
				{Name: "Questions", Level: "green", Detail: "no pending questions"},
			},
		},
	}

	output := FormatOrientation(data)
	if strings.Contains(output, "Daemon health:") {
		t.Error("daemon health section should not appear when all signals are green")
	}
}

func TestFormatOrientation_DaemonHealthNil(t *testing.T) {
	data := &OrientationData{Throughput: Throughput{Days: 1}}
	output := FormatOrientation(data)
	if strings.Contains(output, "Daemon health") {
		t.Error("daemon health section should not appear when nil")
	}
}

func TestDaemonHealthJSON(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		DaemonHealth: &DaemonHealthView{
			Signals: []DaemonHealthSignalView{
				{Name: "Daemon Liveness", Level: "green", Detail: "polling normally"},
			},
		},
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(b)
	if !strings.Contains(jsonStr, `"daemon_health"`) {
		t.Error("JSON missing daemon_health field")
	}
	if !strings.Contains(jsonStr, `"Daemon Liveness"`) {
		t.Error("JSON missing signal name")
	}

	// Verify omitempty
	data2 := &OrientationData{Throughput: Throughput{Days: 1}}
	b2, _ := json.Marshal(data2)
	if strings.Contains(string(b2), "daemon_health") {
		t.Error("daemon_health should be omitted when nil")
	}
}
