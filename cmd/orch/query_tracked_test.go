package main

import (
	"testing"

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

	results := joinWithReasonCodes(issues, manifests, liveness, phases)

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

	results := joinWithReasonCodes(issues, manifests, liveness, nil)

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

	results := joinWithReasonCodes(issues, manifests, liveness, nil)

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

	results := joinWithReasonCodes(issues, manifests, liveness, nil)

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

	results := joinWithReasonCodes(issues, manifests, liveness, nil)

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
	// Session ID exists in manifest but not in liveness map.
	// This means OpenCode returned successfully but the session wasn't in the status map.
	// In OpenCode, sessions not in the /session/status map are idle.
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

	results := joinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	// Sessions not in the status map are idle in OpenCode
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

	results := joinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Results should be in same order as issues
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
	results := joinWithReasonCodes(nil, nil, nil, nil)
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

	results := joinWithReasonCodes(issues, manifests, liveness, phases)

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
	result := unknownLiveness(sessionIDs)

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

	result := extractSessionIDs(manifests)

	// Should only include non-empty session IDs
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
	// When OpenCode is down, liveness map is nil (unknownLiveness was called by queryTrackedAgents).
	// But the join function itself doesn't know if OpenCode is down. It just works with
	// the liveness map it receives. When liveness is nil, sessions are treated as unknown.
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
	// Simulate OpenCode down: unknownLiveness was called, returning "unknown" type
	liveness := unknownLiveness([]string{"sess-oc-down"})

	results := joinWithReasonCodes(issues, manifests, liveness, nil)

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
	// Should NOT be marked as SessionDead - we don't know
	if r.SessionDead {
		t.Error("should not be SessionDead when status is unknown")
	}
}

func TestJoinWithReasonCodes_MissingPhase(t *testing.T) {
	// When no phase data exists for an agent, MissingPhase should be true.
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
	// No phase for this agent
	phases := map[string]string{}

	results := joinWithReasonCodes(issues, manifests, liveness, phases)

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
	// Agent should still be active (phase is independent of liveness)
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
			got := latestPhaseFromComments(tt.comments)
			if got != tt.want {
				t.Errorf("latestPhaseFromComments() = %q, want %q", got, tt.want)
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

	active := filterActiveIssues(issues)

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

func TestListTrackedIssuesCLIFiltersClosed(t *testing.T) {
	oldFn := fallbackListWithLabelFn
	defer func() { fallbackListWithLabelFn = oldFn }()

	fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
		if label != "orch:agent" {
			t.Fatalf("expected label orch:agent, got %s", label)
		}
		return []beads.Issue{
			{ID: "id-open", Status: "open"},
			{ID: "id-progress", Status: "in_progress"},
			{ID: "id-closed", Status: "closed"},
			{ID: "id-closed-upper", Status: "Closed"},
		}, nil
	}

	issues, err := listTrackedIssuesCLI()
	if err != nil {
		t.Fatalf("listTrackedIssuesCLI returned error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 active issues, got %d", len(issues))
	}
	ids := map[string]bool{}
	for _, issue := range issues {
		ids[issue.ID] = true
	}
	if !ids["id-open"] || !ids["id-progress"] {
		t.Fatalf("expected open and in_progress issues, got %v", ids)
	}
}
