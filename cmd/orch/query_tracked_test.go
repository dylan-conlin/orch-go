package main

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/discovery"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// These tests exercise the canonical discovery.JoinWithReasonCodes function
// to verify status derivation. The local queryTrackedAgents delegates to
// discovery.QueryTrackedAgents, so these tests cover the core join logic.

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

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)

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
	if r.ProjectDir != "/tmp/project" {
		t.Errorf("expected ProjectDir /tmp/project, got %s", r.ProjectDir)
	}
	if r.Phase != "Implementing - Adding auth middleware" {
		t.Errorf("expected Phase 'Implementing - Adding auth middleware', got %q", r.Phase)
	}
	if r.MissingBinding {
		t.Error("expected MissingBinding=false")
	}
	if r.MissingSession {
		t.Error("expected MissingSession=false")
	}
	if r.SessionDead {
		t.Error("expected SessionDead=false")
	}
	if r.MissingPhase {
		t.Error("expected MissingPhase=false")
	}
	if r.Reason != "" {
		t.Errorf("expected empty Reason, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_MissingBinding(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-200", Title: "Missing manifest", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	manifests := map[string]*spawn.AgentManifest{} // empty - no workspace manifest found
	liveness := map[string]opencode.SessionStatusInfo{}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

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
	if r.SessionID != "" {
		t.Errorf("expected empty SessionID, got %s", r.SessionID)
	}
}

func TestJoinWithReasonCodes_MissingSession(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-300", Title: "No session", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-300": {
			BeadsID:    "orch-go-300",
			SessionID:  "", // No session ID in manifest (e.g., claude backend)
			ProjectDir: "/tmp/project",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

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
	if r.Reason != "missing_session" {
		t.Errorf("expected Reason missing_session, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_SessionDead(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-400", Title: "Dead session", Status: "in_progress", Labels: []string{"orch:agent"}},
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

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

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
	if r.Reason != "session_idle" {
		t.Errorf("expected Reason session_idle, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_SessionRetrying(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-450", Title: "Retrying session", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-450": {
			BeadsID:    "orch-go-450",
			SessionID:  "sess-retry",
			ProjectDir: "/tmp/project",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-retry": {Type: "retry", Attempt: 3},
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

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

func TestJoinWithReasonCodes_SessionNotInLiveness(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-500", Title: "Unknown session", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-500": {
			BeadsID:    "orch-go-500",
			SessionID:  "sess-unknown",
			ProjectDir: "/tmp/project",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{} // session not in map

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.SessionDead {
		t.Error("expected SessionDead=true for session not in liveness map")
	}
	if r.Status != "idle" {
		t.Errorf("expected Status idle, got %s", r.Status)
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
		// orch-go-602 intentionally missing
		"orch-go-603": {BeadsID: "orch-go-603", SessionID: "sess-3", ProjectDir: "/tmp/p3"},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-1": {Type: "busy"},
		"sess-3": {Type: "idle"},
	}
	phases := map[string]string{
		"orch-go-601": "Implementing - working",
		"orch-go-603": "Planning - reading code",
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Status != "active" {
		t.Errorf("agent 601: expected active, got %s", results[0].Status)
	}
	if results[0].Phase != "Implementing - working" {
		t.Errorf("agent 601: expected Phase 'Implementing - working', got %q", results[0].Phase)
	}
	if !results[1].MissingBinding {
		t.Error("agent 602: expected MissingBinding=true")
	}
	if !results[1].MissingPhase {
		t.Error("agent 602: expected MissingPhase=true (no phase in map)")
	}
	if !results[2].SessionDead {
		t.Error("agent 603: expected SessionDead=true")
	}
	if results[2].Phase != "Planning - reading code" {
		t.Errorf("agent 603: expected Phase 'Planning - reading code', got %q", results[2].Phase)
	}
}

func TestJoinWithReasonCodes_EmptyInputs(t *testing.T) {
	results := discovery.JoinWithReasonCodes(nil, nil, nil, nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil inputs, got %d", len(results))
	}
}

func TestJoinWithReasonCodes_PreservesIssueMetadata(t *testing.T) {
	issues := []beads.Issue{
		{
			ID:        "orch-go-700",
			Title:     "My task title",
			Status:    "in_progress",
			IssueType: "feature",
			Labels:    []string{"orch:agent", "area:core"},
		},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-700": {
			BeadsID:       "orch-go-700",
			SessionID:     "sess-meta",
			ProjectDir:    "/tmp/project",
			Skill:         "feature-impl",
			WorkspaceName: "og-feat-my-task-19feb-abcd",
			Tier:          "light",
			Model:         "claude-opus-4-5",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-meta": {Type: "busy"},
	}
	phases := map[string]string{
		"orch-go-700": "Complete - All tests passing",
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Title != "My task title" {
		t.Errorf("expected Title 'My task title', got %q", r.Title)
	}
	if r.Skill != "feature-impl" {
		t.Errorf("expected Skill feature-impl, got %q", r.Skill)
	}
	if r.WorkspaceName != "og-feat-my-task-19feb-abcd" {
		t.Errorf("expected WorkspaceName og-feat-my-task-19feb-abcd, got %q", r.WorkspaceName)
	}
	if r.Tier != "light" {
		t.Errorf("expected Tier light, got %q", r.Tier)
	}
	if r.Model != "claude-opus-4-5" {
		t.Errorf("expected Model claude-opus-4-5, got %q", r.Model)
	}
	if r.Phase != "Complete - All tests passing" {
		t.Errorf("expected Phase 'Complete - All tests passing', got %q", r.Phase)
	}
}

func TestUnknownLiveness(t *testing.T) {
	sessionIDs := []string{"sess-1", "sess-2", "sess-3"}
	result := discovery.UnknownLiveness(sessionIDs)

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
		"id-2": {SessionID: ""}, // No session (claude backend)
		"id-3": {SessionID: "sess-b"},
	}

	result := discovery.ExtractSessionIDs(manifests)

	if len(result) != 2 {
		t.Fatalf("expected 2 session IDs, got %d", len(result))
	}

	found := make(map[string]bool)
	for _, id := range result {
		found[id] = true
	}
	if !found["sess-a"] {
		t.Error("missing sess-a")
	}
	if !found["sess-b"] {
		t.Error("missing sess-b")
	}
}

func TestJoinWithReasonCodes_OpenCodeDown(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-800", Title: "Task", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-800": {
			BeadsID:    "orch-go-800",
			SessionID:  "sess-oc-down",
			ProjectDir: "/tmp/project",
		},
	}
	liveness := discovery.UnknownLiveness([]string{"sess-oc-down"})

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "unknown" {
		t.Errorf("expected Status unknown when OpenCode down, got %s", r.Status)
	}
	if r.Reason != "opencode_unreachable" {
		t.Errorf("expected Reason opencode_unreachable, got %q", r.Reason)
	}
	if r.SessionDead {
		t.Error("should not be SessionDead when status is unknown")
	}
}

func TestJoinWithReasonCodes_MissingPhase(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-900", Title: "No phase yet", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-900": {
			BeadsID:    "orch-go-900",
			SessionID:  "sess-nophase",
			ProjectDir: "/tmp/project",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-nophase": {Type: "busy"},
	}
	phases := map[string]string{}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Phase != "" {
		t.Errorf("expected empty Phase, got %q", r.Phase)
	}
	if !r.MissingPhase {
		t.Error("expected MissingPhase=true when no phase comment exists")
	}
	if r.Status != "active" {
		t.Errorf("expected Status active, got %s", r.Status)
	}
}

func TestLatestPhaseFromComments(t *testing.T) {
	tests := []struct {
		name     string
		comments []beads.Comment
		want     string
	}{
		{
			name:     "no comments",
			comments: nil,
			want:     "",
		},
		{
			name: "no phase comments",
			comments: []beads.Comment{
				{Text: "Started working on this"},
				{Text: "Found a bug in the process"},
			},
			want: "",
		},
		{
			name: "single phase comment",
			comments: []beads.Comment{
				{Text: "Phase: Planning - Reading codebase"},
			},
			want: "Planning - Reading codebase",
		},
		{
			name: "multiple phase comments returns latest",
			comments: []beads.Comment{
				{Text: "Phase: Planning - Reading codebase"},
				{Text: "Some other comment"},
				{Text: "Phase: Implementing - Adding feature"},
				{Text: "Phase: Complete - All tests pass"},
			},
			want: "Complete - All tests pass",
		},
		{
			name: "phase comment with extra whitespace",
			comments: []beads.Comment{
				{Text: "Phase:  Implementing - stuff "},
			},
			want: "Implementing - stuff",
		},
		{
			name: "non-phase comments interspersed",
			comments: []beads.Comment{
				{Text: "Phase: Planning - start"},
				{Text: "BLOCKED: need API key"},
				{Text: "Phase: Implementing - resumed"},
				{Text: "Self-review passed"},
			},
			want: "Implementing - resumed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := discovery.LatestPhaseFromComments(tt.comments)
			if got != tt.want {
				t.Errorf("LatestPhaseFromComments() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLatestPhaseWithTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		comments  []beads.Comment
		wantPhase string
		wantTime  bool
	}{
		{
			name:      "no comments",
			comments:  nil,
			wantPhase: "",
			wantTime:  false,
		},
		{
			name: "phase with valid timestamp",
			comments: []beads.Comment{
				{Text: "Phase: Planning - Reading code", CreatedAt: "2026-02-28T10:30:00Z"},
			},
			wantPhase: "Planning - Reading code",
			wantTime:  true,
		},
		{
			name: "phase with empty timestamp",
			comments: []beads.Comment{
				{Text: "Phase: Implementing - Adding feature", CreatedAt: ""},
			},
			wantPhase: "Implementing - Adding feature",
			wantTime:  false,
		},
		{
			name: "multiple phases returns latest with timestamp",
			comments: []beads.Comment{
				{Text: "Phase: Planning - start", CreatedAt: "2026-02-28T09:00:00Z"},
				{Text: "Phase: Implementing - work", CreatedAt: "2026-02-28T10:00:00Z"},
				{Text: "Phase: Complete - done", CreatedAt: "2026-02-28T11:00:00Z"},
			},
			wantPhase: "Complete - done",
			wantTime:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, ts := discovery.LatestPhaseWithTimestamp(tt.comments)
			if phase != tt.wantPhase {
				t.Errorf("phase = %q, want %q", phase, tt.wantPhase)
			}
			if tt.wantTime && ts.IsZero() {
				t.Error("expected non-zero timestamp")
			}
			if !tt.wantTime && !ts.IsZero() {
				t.Error("expected zero timestamp")
			}
		})
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

	active := discovery.FilterActiveIssues(issues)

	if len(active) != 3 {
		t.Fatalf("expected 3 active issues, got %d", len(active))
	}
	ids := make(map[string]bool)
	for _, a := range active {
		ids[a.ID] = true
	}
	if !ids["id-1"] || !ids["id-2"] || !ids["id-4"] {
		t.Errorf("expected id-1, id-2, id-4 to be active, got %v", ids)
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
			Skill:         "feature-impl",
			SpawnTime:     "2026-02-24T10:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}
	phases := map[string]string{
		"orch-go-1100": "Implementing - Working on feature",
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "active" {
		t.Errorf("expected Status active for claude agent with phase, got %s", r.Status)
	}
	if r.Reason != "phase_reported" {
		t.Errorf("expected Reason phase_reported, got %q", r.Reason)
	}
	if r.MissingSession {
		t.Error("claude-backend agent should not be marked MissingSession")
	}
	if r.SpawnMode != "claude" {
		t.Errorf("expected SpawnMode claude, got %q", r.SpawnMode)
	}
	if r.Phase != "Implementing - Working on feature" {
		t.Errorf("expected Phase 'Implementing - Working on feature', got %q", r.Phase)
	}
}

func TestJoinWithReasonCodes_ClaudeBackendPhaseComplete(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-1150", Title: "Completed claude agent", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1150": {
			BeadsID:       "orch-go-1150",
			SessionID:     "",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-feat-done-task-21feb-abcd",
			Skill:         "feature-impl",
			SpawnTime:     "2026-02-24T08:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}
	phases := map[string]string{
		"orch-go-1150": "Complete - All tests passing, ready for review",
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "completed" {
		t.Errorf("expected Status completed for claude agent with Phase: Complete, got %s", r.Status)
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
			SessionID:     "",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-feat-new-task-24feb-abcd",
			Skill:         "feature-impl",
			SpawnTime:     recentTime,
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "active" {
		t.Errorf("expected Status active for recently spawned claude agent, got %s", r.Status)
	}
	if r.Reason != "recently_spawned" {
		t.Errorf("expected Reason recently_spawned, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_ClaudeBackendNoPhaseStale(t *testing.T) {
	// Mock tmux check to return false (window not alive)
	oldCheck := discovery.CheckTmuxWindowAlive
	discovery.CheckTmuxWindowAlive = func(workspaceName, projectDir string) bool { return false }
	defer func() { discovery.CheckTmuxWindowAlive = oldCheck }()

	issues := []beads.Issue{
		{ID: "orch-go-1200", Title: "Dead claude agent", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1200": {
			BeadsID:       "orch-go-1200",
			SessionID:     "",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-feat-dead-task-21feb-efgh",
			Skill:         "investigation",
			SpawnTime:     "2026-02-20T10:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "dead" {
		t.Errorf("expected Status dead for claude agent without phase and stale spawn, got %s", r.Status)
	}
	if r.Reason != "no_phase_reported" {
		t.Errorf("expected Reason no_phase_reported, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_ClaudeBackendTmuxFallbackAlive(t *testing.T) {
	oldCheck := discovery.CheckTmuxWindowAlive
	discovery.CheckTmuxWindowAlive = func(workspaceName, projectDir string) bool {
		if workspaceName != "og-debug-browser-02mar-abcd" {
			t.Errorf("unexpected workspaceName: %s", workspaceName)
		}
		if projectDir != "/tmp/project" {
			t.Errorf("unexpected projectDir: %s", projectDir)
		}
		return true
	}
	defer func() { discovery.CheckTmuxWindowAlive = oldCheck }()

	issues := []beads.Issue{
		{ID: "orch-go-1250", Title: "Working but no phase", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1250": {
			BeadsID:       "orch-go-1250",
			SessionID:     "",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-debug-browser-02mar-abcd",
			Skill:         "systematic-debugging",
			SpawnTime:     "2026-02-20T10:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "active" {
		t.Errorf("expected Status active for claude agent with tmux window alive, got %s", r.Status)
	}
	if r.Reason != "tmux_window_alive" {
		t.Errorf("expected Reason tmux_window_alive, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_ClaudeBackendTmuxFallbackNotCheckedWhenPhaseExists(t *testing.T) {
	oldCheck := discovery.CheckTmuxWindowAlive
	tmuxCalled := false
	discovery.CheckTmuxWindowAlive = func(workspaceName, projectDir string) bool {
		tmuxCalled = true
		return true
	}
	defer func() { discovery.CheckTmuxWindowAlive = oldCheck }()

	issues := []beads.Issue{
		{ID: "orch-go-1260", Title: "Agent with phase", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1260": {
			BeadsID:       "orch-go-1260",
			SessionID:     "",
			ProjectDir:    "/tmp/project",
			SpawnMode:     "claude",
			WorkspaceName: "og-feat-has-phase-02mar-efgh",
			Skill:         "feature-impl",
			SpawnTime:     "2026-02-20T10:00:00Z",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}
	phases := map[string]string{
		"orch-go-1260": "Implementing - Working on feature",
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "active" {
		t.Errorf("expected Status active, got %s", results[0].Status)
	}
	if results[0].Reason != "phase_reported" {
		t.Errorf("expected Reason phase_reported, got %q", results[0].Reason)
	}
	if tmuxCalled {
		t.Error("tmux check should NOT be called when phase is present -- tmux is only a fallback")
	}
}

func TestJoinWithReasonCodes_NonClaudeNoSession(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-1300", Title: "Non-claude no session", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1300": {
			BeadsID:    "orch-go-1300",
			SessionID:  "",
			ProjectDir: "/tmp/project",
			SpawnMode:  "opencode",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.MissingSession {
		t.Error("expected MissingSession=true for non-claude agent")
	}
	if r.Status != "unknown" {
		t.Errorf("expected Status unknown, got %s", r.Status)
	}
	if r.Reason != "missing_session" {
		t.Errorf("expected Reason missing_session, got %q", r.Reason)
	}
}

func TestJoinWithReasonCodes_PhaseTimestamps(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-1400", Title: "Agent with phase timestamp", Status: "in_progress"},
		{ID: "orch-go-1401", Title: "Agent without timestamp", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-1400": {
			BeadsID:   "orch-go-1400",
			SessionID: "sess-1400",
		},
		"orch-go-1401": {
			BeadsID:   "orch-go-1401",
			SessionID: "sess-1401",
		},
	}
	liveness := map[string]opencode.SessionStatusInfo{
		"sess-1400": {Type: "busy"},
		"sess-1401": {Type: "busy"},
	}
	phases := map[string]string{
		"orch-go-1400": "Implementing - working",
		"orch-go-1401": "Planning - reading",
	}
	ts := time.Date(2026, 2, 28, 10, 30, 0, 0, time.UTC)
	phaseTimestamps := map[string]time.Time{
		"orch-go-1400": ts,
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases, phaseTimestamps)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r1 := results[0]
	if r1.BeadsID == "orch-go-1401" {
		r1 = results[1]
	}
	if r1.PhaseReportedAt == nil {
		t.Fatal("expected PhaseReportedAt to be set for orch-go-1400")
	}
	if !r1.PhaseReportedAt.Equal(ts) {
		t.Errorf("expected PhaseReportedAt %v, got %v", ts, *r1.PhaseReportedAt)
	}

	r2 := results[1]
	if r2.BeadsID == "orch-go-1400" {
		r2 = results[0]
	}
	if r2.PhaseReportedAt != nil {
		t.Errorf("expected nil PhaseReportedAt for orch-go-1401, got %v", *r2.PhaseReportedAt)
	}
}
