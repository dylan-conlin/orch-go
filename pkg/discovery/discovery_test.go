package discovery

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestJoinWithReasonCodes_FullyBound(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-100", Title: "Test task", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-100": {
			BeadsID:    "orch-go-100",
			SessionID:  "sess-abc",
			ProjectDir: "/tmp/project",
			Skill:      "feature-impl",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-abc": {Type: "busy"},
	}
	phases := map[string]string{
		"orch-go-100": "Implementing - Adding auth middleware",
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.BeadsID != "orch-go-100" {
		t.Errorf("expected BeadsID orch-go-100, got %s", r.BeadsID)
	}
	if r.SessionID != "sess-abc" {
		t.Errorf("expected SessionID sess-abc, got %s", r.SessionID)
	}
	if r.Status != "active" {
		t.Errorf("expected Status active, got %s", r.Status)
	}
	if r.Phase != "Implementing - Adding auth middleware" {
		t.Errorf("expected Phase 'Implementing - Adding auth middleware', got %q", r.Phase)
	}
	if r.MissingBinding {
		t.Error("expected MissingBinding=false")
	}
	if r.Reason != "" {
		t.Errorf("expected empty Reason, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_MissingBinding(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-200", Title: "Missing manifest", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.MissingBinding {
		t.Error("expected MissingBinding=true")
	}
	if r.Status != "unknown" {
		t.Errorf("expected Status unknown, got %s", r.Status)
	}
	if r.Reason != "missing_binding" {
		t.Errorf("expected Reason missing_binding, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_MissingSession(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-300", Title: "No session", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-300": {
			BeadsID:    "orch-go-300",
			SessionID:  "",
			ProjectDir: "/tmp/project",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.MissingSession {
		t.Error("expected MissingSession=true")
	}
	if r.Status != "unknown" {
		t.Errorf("expected Status unknown, got %s", r.Status)
	}
}

func TestJoinWithReasonCodes_SessionDead(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-400", Title: "Dead session", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-400": {
			BeadsID:    "orch-go-400",
			SessionID:  "sess-dead",
			ProjectDir: "/tmp/project",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-dead": {Type: "idle"},
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.SessionDead {
		t.Error("expected SessionDead=true")
	}
	if r.Status != "idle" {
		t.Errorf("expected Status idle, got %s", r.Status)
	}
}

func TestJoinWithReasonCodes_ClaudeBackendWithPhase(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-1100", Title: "Claude agent with phase", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1100": {
			BeadsID:       "orch-go-1100",
			SessionID:     "",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-feat-my-task-21feb-abcd",
			SpawnTime:     "2026-02-24T10:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}
	phases := map[string]string{
		"orch-go-1100": "Implementing - Working on feature",
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "active" {
		t.Errorf("expected Status active, got %s", r.Status)
	}
	if r.Reason != "phase_reported" {
		t.Errorf("expected Reason phase_reported, got %q", r.Reason)
	}
	if r.SpawnMode != "claude" {
		t.Errorf("expected SpawnMode claude, got %q", r.SpawnMode)
	}
}

func TestJoinWithReasonCodes_ClaudeBackendPhaseComplete(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-1150", Title: "Completed claude agent", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1150": {
			BeadsID:       "orch-go-1150",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-feat-done-21feb-abcd",
			SpawnTime:     "2026-02-24T08:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}
	phases := map[string]string{
		"orch-go-1150": "Complete - All tests passing",
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "completed" {
		t.Errorf("expected Status completed, got %s", r.Status)
	}
	if r.Reason != "phase_complete" {
		t.Errorf("expected Reason phase_complete, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_ClaudeBackendRecentlySpawned(t *testing.T) {
	recentTime := time.Now().Add(-2 * time.Minute).Format(time.RFC3339)
	issues := []beads.Issue{
		{ID: "orch-go-1160", Title: "Just-spawned claude agent", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1160": {
			BeadsID:       "orch-go-1160",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-feat-new-task-24feb-abcd",
			SpawnTime:     recentTime,
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "active" {
		t.Errorf("expected Status active, got %s", r.Status)
	}
	if r.Reason != "recently_spawned" {
		t.Errorf("expected Reason recently_spawned, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_ClaudeBackendNoPhaseStale(t *testing.T) {
	// Mock tmux check to return false
	oldCheck := CheckTmuxWindowAlive
	CheckTmuxWindowAlive = func(workspaceName, projectDir string) bool { return false }
	defer func() { CheckTmuxWindowAlive = oldCheck }()

	issues := []beads.Issue{
		{ID: "orch-go-1200", Title: "Dead claude agent", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1200": {
			BeadsID:       "orch-go-1200",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-feat-dead-task-21feb-efgh",
			SpawnTime:     "2026-02-20T10:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "dead" {
		t.Errorf("expected Status dead, got %s", r.Status)
	}
	if r.Reason != "no_phase_reported" {
		t.Errorf("expected Reason no_phase_reported, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_ClaudeBackendTmuxFallbackAlive(t *testing.T) {
	oldCheck := CheckTmuxWindowAlive
	CheckTmuxWindowAlive = func(workspaceName, projectDir string) bool { return true }
	defer func() { CheckTmuxWindowAlive = oldCheck }()

	issues := []beads.Issue{
		{ID: "orch-go-1250", Title: "Working but no phase", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1250": {
			BeadsID:       "orch-go-1250",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-debug-browser-02mar-abcd",
			SpawnTime:     "2026-02-20T10:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "active" {
		t.Errorf("expected Status active, got %s", r.Status)
	}
	if r.Reason != "tmux_window_alive" {
		t.Errorf("expected Reason tmux_window_alive, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_OpenCodePhaseComplete(t *testing.T) {
	// Bug: OpenCode agents with Phase: Complete but active session showed as "running".
	// Phase: Complete must override OpenCode session liveness for all backends.
	issues := []beads.Issue{
		{ID: "orch-go-500", Title: "Auto-completed OpenCode agent", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-500": {
			BeadsID:   "orch-go-500",
			SessionID: "sess-still-busy",
			ProjectDir: "/tmp/project",
			Skill:     "feature-impl",
		},
	}
	// Session is still busy in OpenCode, but agent reported Phase: Complete
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-still-busy": {Type: "busy"},
	}
	phases := map[string]string{
		"orch-go-500": "Complete - All tests passing, ready for review",
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "completed" {
		t.Errorf("expected Status completed, got %s (Reason: %s)", r.Status, r.Reason)
	}
	if r.Reason != "phase_complete" {
		t.Errorf("expected Reason phase_complete, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_ClaudePhaseCompleteUniversal(t *testing.T) {
	// Verify that Phase: Complete works for Claude agents via the universal check
	// (not just the Claude-specific code path).
	issues := []beads.Issue{
		{ID: "orch-go-510", Title: "Claude agent auto-completed", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-510": {
			BeadsID:       "orch-go-510",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-feat-done-10mar-abcd",
			SpawnTime:     "2026-03-10T08:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}
	phases := map[string]string{
		"orch-go-510": "Complete - Implemented and tested",
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "completed" {
		t.Errorf("expected Status completed, got %s", r.Status)
	}
	if r.Reason != "phase_complete" {
		t.Errorf("expected Reason phase_complete, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_NoSpawnModePhaseComplete(t *testing.T) {
	// Agents with missing SpawnMode (old manifests) that have Phase: Complete
	// should also be detected as completed.
	issues := []beads.Issue{
		{ID: "orch-go-520", Title: "Old manifest agent", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-520": {
			BeadsID:   "orch-go-520",
			SessionID: "sess-old",
			ProjectDir: "/tmp/project",
			// SpawnMode intentionally empty — old manifest format
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-old": {Type: "busy"},
	}
	phases := map[string]string{
		"orch-go-520": "Complete - Done",
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "completed" {
		t.Errorf("expected Status completed, got %s (Reason: %s)", r.Status, r.Reason)
	}
}

func TestJoinWithReasonCodes_EmptyInputs(t *testing.T) {
	results := JoinWithReasonCodes(nil, nil, nil, nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil inputs, got %d", len(results))
	}
}

func TestJoinWithReasonCodes_PhaseTimestamps(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-1400", Title: "Agent with phase timestamp", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1400": {
			BeadsID:   "orch-go-1400",
			SessionID: "sess-1400",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-1400": {Type: "busy"},
	}
	phases := map[string]string{
		"orch-go-1400": "Implementing - working",
	}
	ts := time.Date(2026, 2, 28, 10, 30, 0, 0, time.UTC)
	phaseTimestamps := map[string]time.Time{
		"orch-go-1400": ts,
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, phases, phaseTimestamps)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.PhaseReportedAt == nil {
		t.Fatal("expected PhaseReportedAt to be set")
	}
	if !r.PhaseReportedAt.Equal(ts) {
		t.Errorf("expected PhaseReportedAt %v, got %v", ts, *r.PhaseReportedAt)
	}
}

func TestJoinWithReasonCodes_SessionRetrying(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-450", Title: "Retrying session", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-450": {
			BeadsID:   "orch-go-450",
			SessionID: "sess-retry",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-retry": {Type: "retry", Attempt: 3},
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "retrying" {
		t.Errorf("expected Status retrying, got %s", r.Status)
	}
	if r.Reason != "session_retrying" {
		t.Errorf("expected Reason session_retrying, got %q", r.Reason)
	}
}

func TestUnknownLiveness(t *testing.T) {
	sessionIDs := []string{"sess-1", "sess-2", "sess-3"}
	result := UnknownLiveness(sessionIDs)

	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	for _, id := range sessionIDs {
		info, ok := result[id]
		if !ok {
			t.Errorf("missing entry for %s", id)
			continue
		}
		if info.Type != "unknown" {
			t.Errorf("expected type 'unknown' for %s, got %q", id, info.Type)
		}
	}
}

func TestExtractSessionIDs(t *testing.T) {
	manifests := map[string]*spawn.AgentManifest{
		"id-1": {SessionID: "sess-a"},
		"id-2": {SessionID: ""},
		"id-3": {SessionID: "sess-b"},
	}

	result := ExtractSessionIDs(manifests)

	if len(result) != 2 {
		t.Fatalf("expected 2 session IDs, got %d", len(result))
	}

	found := make(map[string]bool)
	for _, id := range result {
		found[id] = true
	}
	if !found["sess-a"] || !found["sess-b"] {
		t.Error("missing expected session ID")
	}
}

func TestFilterActiveIssues(t *testing.T) {
	issues := []beads.Issue{
		{ID: "id-1", Status: "open"},
		{ID: "id-2", Status: "in_progress"},
		{ID: "id-3", Status: "closed"},
		{ID: "id-4", Status: "open"},
		{ID: "id-5", Status: "resolved"},
	}

	active := FilterActiveIssues(issues)

	if len(active) != 3 {
		t.Fatalf("expected 3 active issues, got %d", len(active))
	}
	ids := make(map[string]bool)
	for _, a := range active {
		ids[a.ID] = true
	}
	if !ids["id-1"] || !ids["id-2"] || !ids["id-4"] {
		t.Errorf("expected id-1, id-2, id-4, got %v", ids)
	}
}

func TestLatestPhaseFromComments(t *testing.T) {
	tests := []struct {
		name     string
		comments []beads.Comment
		want     string
	}{
		{"no comments", nil, ""},
		{"no phase comments", []beads.Comment{{Text: "Started working"}}, ""},
		{"single phase", []beads.Comment{{Text: "Phase: Planning - Reading"}}, "Planning - Reading"},
		{
			"latest phase",
			[]beads.Comment{
				{Text: "Phase: Planning - start"},
				{Text: "other"},
				{Text: "Phase: Complete - done"},
			},
			"Complete - done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LatestPhaseFromComments(tt.comments)
			if got != tt.want {
				t.Errorf("LatestPhaseFromComments() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLatestPhaseWithTimestamp(t *testing.T) {
	comments := []beads.Comment{
		{Text: "Phase: Planning - start", CreatedAt: "2026-02-28T09:00:00Z"},
		{Text: "Phase: Complete - done", CreatedAt: "2026-02-28T11:00:00Z"},
	}

	phase, ts := LatestPhaseWithTimestamp(comments)

	if phase != "Complete - done" {
		t.Errorf("expected 'Complete - done', got %q", phase)
	}
	if ts.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestJoinWithReasonCodes_MultipleAgents(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-601", Title: "Active agent", Status: "in_progress"},
		{ID: "orch-go-602", Title: "Missing binding", Status: "in_progress"},
		{ID: "orch-go-603", Title: "Dead agent", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-601": {BeadsID: "orch-go-601", SessionID: "sess-1", ProjectDir: "/tmp/p1"},
		"orch-go-603": {BeadsID: "orch-go-603", SessionID: "sess-3", ProjectDir: "/tmp/p3"},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-1": {Type: "busy"},
		"sess-3": {Type: "idle"},
	}
	phases := map[string]string{
		"orch-go-601": "Implementing - working",
		"orch-go-603": "Planning - reading",
	}

	results := JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Status != "active" {
		t.Errorf("agent 601: expected active, got %s", results[0].Status)
	}
	if !results[1].MissingBinding {
		t.Error("agent 602: expected MissingBinding=true")
	}
	if !results[2].SessionDead {
		t.Error("agent 603: expected SessionDead=true")
	}
}

func TestJoinWithReasonCodes_OpenCodeDown(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-800", Title: "Task", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-800": {
			BeadsID:   "orch-go-800",
			SessionID: "sess-oc-down",
		},
	}
	liveness := UnknownLiveness([]string{"sess-oc-down"})

	results := JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "unknown" {
		t.Errorf("expected Status unknown, got %s", r.Status)
	}
	if r.Reason != "opencode_unreachable" {
		t.Errorf("expected Reason opencode_unreachable, got %q", r.Reason)
	}
}
