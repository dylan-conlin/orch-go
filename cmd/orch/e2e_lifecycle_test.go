// E2E lifecycle tests for the agent tracking pipeline (orch-go-1100).
//
// Key contract: tracked agents appear in status queries, completed agents do not.
//
// These tests exercise the full pipeline (listTrackedIssuesCLI → filterActiveIssues →
// joinWithReasonCodes → determineAgentStatus) across lifecycle transitions:
//   spawn → active → phase transitions → complete → closed/gone
//
// All tests use mock data — no real beads daemon or OpenCode server required.
package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// TestE2ELifecycle_SingleAgent exercises the complete lifecycle of one agent:
// create → visible in status → phase transitions → complete → gone from status.
func TestE2ELifecycle_SingleAgent(t *testing.T) {
	beadsID := "orch-go-e2e-100"
	sessionID := "sess-e2e-100"

	// === Phase 1: Issue created, agent spawned ===
	// After spawn, beads has an in_progress issue with orch:agent label.
	// listTrackedIssuesCLI should return it.
	t.Run("spawn_visible_in_pipeline", func(t *testing.T) {
		oldFn := fallbackListWithLabelFn
		defer func() { fallbackListWithLabelFn = oldFn }()

		fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
			return []beads.Issue{
				{ID: beadsID, Title: "E2E test agent", Status: "in_progress", Labels: []string{"orch:agent"}},
			}, nil
		}

		issues, err := listTrackedIssuesCLI()
		if err != nil {
			t.Fatalf("listTrackedIssuesCLI: %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("expected 1 tracked issue, got %d", len(issues))
		}
		if issues[0].ID != beadsID {
			t.Errorf("expected beads ID %s, got %s", beadsID, issues[0].ID)
		}
	})

	// === Phase 2: Agent actively working (busy session) ===
	t.Run("active_agent_has_full_status", func(t *testing.T) {
		issues := []beads.Issue{
			{ID: beadsID, Title: "E2E test agent", Status: "in_progress", Labels: []string{"orch:agent"}},
		}
		manifests := map[string]*spawn.AgentManifest{
			beadsID: {
				BeadsID:       beadsID,
				SessionID:     sessionID,
				ProjectDir:    "/tmp/e2e-project",
				WorkspaceName: "og-feat-e2e-test-19feb-abcd",
				Skill:         "feature-impl",
				Tier:          "light",
				Model:         "claude-sonnet-4-5",
				SpawnMode:     "opencode",
			},
		}
		liveness := map[string]opencode.SessionStatusInfo{
			sessionID: {Type: "busy"},
		}
		phases := map[string]string{
			beadsID: "Planning - Reading codebase",
		}

		results := joinWithReasonCodes(issues, manifests, liveness, phases)
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}

		r := results[0]
		if r.Status != "active" {
			t.Errorf("active agent status = %q, want 'active'", r.Status)
		}
		if r.Phase != "Planning - Reading codebase" {
			t.Errorf("phase = %q, want 'Planning - Reading codebase'", r.Phase)
		}
		if r.MissingBinding || r.MissingSession || r.SessionDead || r.MissingPhase {
			t.Error("fully bound active agent should not have any reason code flags")
		}
		if r.Reason != "" {
			t.Errorf("fully bound active agent reason = %q, want empty", r.Reason)
		}
	})

	// === Phase 3: Agent progresses through phases ===
	t.Run("phase_transitions_reflected", func(t *testing.T) {
		phases := []string{
			"Planning - Reading codebase",
			"Implementing - Writing tests",
			"Implementing - Running TDD cycle",
			"Complete - All tests passing, ready for review",
		}

		for _, phase := range phases {
			issues := []beads.Issue{
				{ID: beadsID, Title: "E2E test agent", Status: "in_progress"},
			}
			manifests := map[string]*spawn.AgentManifest{
				beadsID: {BeadsID: beadsID, SessionID: sessionID, ProjectDir: "/tmp"},
			}
			liveness := map[string]opencode.SessionStatusInfo{
				sessionID: {Type: "busy"},
			}
			phaseMap := map[string]string{
				beadsID: phase,
			}

			results := joinWithReasonCodes(issues, manifests, liveness, phaseMap)
			if len(results) != 1 {
				t.Fatalf("phase %q: expected 1 result, got %d", phase, len(results))
			}
			if results[0].Phase != phase {
				t.Errorf("phase not reflected: got %q, want %q", results[0].Phase, phase)
			}
		}
	})

	// === Phase 4: Agent reports Phase: Complete ===
	// determineAgentStatus should return "completed" when phase is complete.
	t.Run("phase_complete_marks_completed", func(t *testing.T) {
		status := determineAgentStatus(false, true, "", "active")
		if status != "completed" {
			t.Errorf("Phase:Complete agent status = %q, want 'completed'", status)
		}
	})

	// === Phase 5: Issue closed → agent disappears from pipeline ===
	t.Run("closed_issue_filtered_out", func(t *testing.T) {
		oldFn := fallbackListWithLabelFn
		defer func() { fallbackListWithLabelFn = oldFn }()

		fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
			return []beads.Issue{
				{ID: beadsID, Title: "E2E test agent", Status: "closed", Labels: []string{"orch:agent"}},
			}, nil
		}

		issues, err := listTrackedIssuesCLI()
		if err != nil {
			t.Fatalf("listTrackedIssuesCLI: %v", err)
		}
		if len(issues) != 0 {
			t.Errorf("closed agent should not appear in tracked issues, got %d", len(issues))
		}
	})

	// === Phase 6: Verify determineAgentStatus with closed issue ===
	t.Run("closed_issue_status_completed", func(t *testing.T) {
		status := determineAgentStatus(true, false, "", "idle")
		if status != "completed" {
			t.Errorf("closed issue status = %q, want 'completed'", status)
		}
	})
}

// TestE2ELifecycle_MultipleAgents exercises the lifecycle with multiple concurrent agents
// at different lifecycle stages, verifying that active agents are visible and completed
// agents disappear independently.
func TestE2ELifecycle_MultipleAgents(t *testing.T) {
	// Set up 4 agents at different stages:
	// 1. Just spawned (open, planning phase)
	// 2. Actively working (in_progress, implementing phase)
	// 3. Completed but not yet closed (in_progress, Phase: Complete)
	// 4. Fully closed
	allIssues := []beads.Issue{
		{ID: "orch-go-multi-1", Title: "Just spawned", Status: "open", Labels: []string{"orch:agent"}},
		{ID: "orch-go-multi-2", Title: "Actively working", Status: "in_progress", Labels: []string{"orch:agent"}},
		{ID: "orch-go-multi-3", Title: "Phase complete", Status: "in_progress", Labels: []string{"orch:agent"}},
		{ID: "orch-go-multi-4", Title: "Fully closed", Status: "closed", Labels: []string{"orch:agent"}},
	}

	// Step 1: Verify filter removes only closed
	t.Run("filter_keeps_active_removes_closed", func(t *testing.T) {
		oldFn := fallbackListWithLabelFn
		defer func() { fallbackListWithLabelFn = oldFn }()

		fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
			return allIssues, nil
		}

		tracked, err := listTrackedIssuesCLI()
		if err != nil {
			t.Fatalf("listTrackedIssuesCLI: %v", err)
		}

		if len(tracked) != 3 {
			t.Fatalf("expected 3 active issues (open + in_progress), got %d", len(tracked))
		}

		trackedIDs := make(map[string]bool)
		for _, issue := range tracked {
			trackedIDs[issue.ID] = true
		}

		if !trackedIDs["orch-go-multi-1"] {
			t.Error("open issue (just spawned) should be visible")
		}
		if !trackedIDs["orch-go-multi-2"] {
			t.Error("in_progress issue (actively working) should be visible")
		}
		if !trackedIDs["orch-go-multi-3"] {
			t.Error("in_progress issue (phase complete) should be visible")
		}
		if trackedIDs["orch-go-multi-4"] {
			t.Error("closed issue should NOT be visible")
		}
	})

	// Step 2: Join gives correct status for each active agent
	t.Run("join_differentiates_agent_states", func(t *testing.T) {
		activeIssues := filterActiveIssues(allIssues)

		manifests := map[string]*spawn.AgentManifest{
			"orch-go-multi-1": {BeadsID: "orch-go-multi-1", SessionID: "sess-m1", ProjectDir: "/tmp/p1", Skill: "feature-impl"},
			"orch-go-multi-2": {BeadsID: "orch-go-multi-2", SessionID: "sess-m2", ProjectDir: "/tmp/p2", Skill: "feature-impl"},
			"orch-go-multi-3": {BeadsID: "orch-go-multi-3", SessionID: "sess-m3", ProjectDir: "/tmp/p3", Skill: "investigation"},
		}
		liveness := map[string]opencode.SessionStatusInfo{
			"sess-m1": {Type: "busy"},
			"sess-m2": {Type: "busy"},
			"sess-m3": {Type: "idle"}, // Session went idle after Phase: Complete
		}
		phases := map[string]string{
			"orch-go-multi-1": "Planning - Reading codebase",
			"orch-go-multi-2": "Implementing - Writing feature",
			"orch-go-multi-3": "Complete - All tests passing",
		}

		results := joinWithReasonCodes(activeIssues, manifests, liveness, phases)
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}

		resultMap := make(map[string]AgentStatus)
		for _, r := range results {
			resultMap[r.BeadsID] = r
		}

		// Agent 1: Just spawned, active
		if resultMap["orch-go-multi-1"].Status != "active" {
			t.Errorf("just-spawned agent status = %q, want 'active'", resultMap["orch-go-multi-1"].Status)
		}
		if resultMap["orch-go-multi-1"].Phase != "Planning - Reading codebase" {
			t.Errorf("just-spawned agent phase = %q, want 'Planning - Reading codebase'", resultMap["orch-go-multi-1"].Phase)
		}

		// Agent 2: Actively working
		if resultMap["orch-go-multi-2"].Status != "active" {
			t.Errorf("working agent status = %q, want 'active'", resultMap["orch-go-multi-2"].Status)
		}

		// Agent 3: Phase complete, session idle
		if resultMap["orch-go-multi-3"].Status != "idle" {
			t.Errorf("phase-complete idle agent status = %q, want 'idle'", resultMap["orch-go-multi-3"].Status)
		}
		if resultMap["orch-go-multi-3"].Phase != "Complete - All tests passing" {
			t.Errorf("phase-complete agent phase = %q, want 'Complete - All tests passing'", resultMap["orch-go-multi-3"].Phase)
		}
	})

	// Step 3: determineAgentStatus correctly identifies completion states
	t.Run("determine_status_at_each_stage", func(t *testing.T) {
		tests := []struct {
			name         string
			issueClosed  bool
			phaseComp    bool
			sessionState string
			want         string
		}{
			{"just spawned - active", false, false, "active", "active"},
			{"actively working - active", false, false, "active", "active"},
			{"phase complete - session active", false, true, "active", "completed"},
			{"phase complete - session dead", false, true, "dead", "awaiting-cleanup"},
			{"issue closed", true, false, "idle", "completed"},
			{"issue closed overrides everything", true, true, "dead", "completed"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := determineAgentStatus(tt.issueClosed, tt.phaseComp, "", tt.sessionState)
				if got != tt.want {
					t.Errorf("determineAgentStatus(%v, %v, _, %q) = %q, want %q",
						tt.issueClosed, tt.phaseComp, tt.sessionState, got, tt.want)
				}
			})
		}
	})

	// Step 4: Progressive closure - as agents complete, they disappear
	t.Run("progressive_closure", func(t *testing.T) {
		oldFn := fallbackListWithLabelFn
		defer func() { fallbackListWithLabelFn = oldFn }()

		// Start: 3 active, 1 closed
		fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
			return allIssues, nil
		}
		tracked, _ := listTrackedIssuesCLI()
		if len(tracked) != 3 {
			t.Fatalf("start: expected 3, got %d", len(tracked))
		}

		// Close agent 3 (Phase: Complete → orchestrator runs orch complete)
		updatedIssues := []beads.Issue{
			{ID: "orch-go-multi-1", Title: "Just spawned", Status: "open", Labels: []string{"orch:agent"}},
			{ID: "orch-go-multi-2", Title: "Actively working", Status: "in_progress", Labels: []string{"orch:agent"}},
			{ID: "orch-go-multi-3", Title: "Phase complete", Status: "closed", Labels: []string{"orch:agent"}},
			{ID: "orch-go-multi-4", Title: "Fully closed", Status: "closed", Labels: []string{"orch:agent"}},
		}
		fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
			return updatedIssues, nil
		}
		tracked, _ = listTrackedIssuesCLI()
		if len(tracked) != 2 {
			t.Fatalf("after closing agent 3: expected 2, got %d", len(tracked))
		}

		// Close all remaining
		allClosed := []beads.Issue{
			{ID: "orch-go-multi-1", Status: "closed", Labels: []string{"orch:agent"}},
			{ID: "orch-go-multi-2", Status: "closed", Labels: []string{"orch:agent"}},
			{ID: "orch-go-multi-3", Status: "closed", Labels: []string{"orch:agent"}},
			{ID: "orch-go-multi-4", Status: "closed", Labels: []string{"orch:agent"}},
		}
		fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
			return allClosed, nil
		}
		tracked, _ = listTrackedIssuesCLI()
		if len(tracked) != 0 {
			t.Fatalf("after all closed: expected 0, got %d", len(tracked))
		}
	})
}

// TestE2ELifecycle_DegradedModes exercises the lifecycle when infrastructure is partially
// unavailable (OpenCode down, workspace missing), verifying agents are still visible
// with appropriate reason codes rather than silently disappearing.
func TestE2ELifecycle_DegradedModes(t *testing.T) {
	// Agent exists in beads but workspace is missing (manifest not found)
	t.Run("missing_workspace_still_visible", func(t *testing.T) {
		oldFn := fallbackListWithLabelFn
		defer func() { fallbackListWithLabelFn = oldFn }()

		fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
			return []beads.Issue{
				{ID: "orch-go-deg-1", Title: "Missing workspace", Status: "in_progress", Labels: []string{"orch:agent"}},
			}, nil
		}

		// Pipeline: issue exists, but no manifest
		issues, _ := listTrackedIssuesCLI()
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue from pipeline, got %d", len(issues))
		}

		// Join with empty manifests
		results := joinWithReasonCodes(issues, map[string]*spawn.AgentManifest{}, nil, nil)
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].MissingBinding {
			t.Error("expected MissingBinding=true for agent without workspace")
		}
		if results[0].Reason != "missing_binding" {
			t.Errorf("reason = %q, want 'missing_binding'", results[0].Reason)
		}
		// Key: agent is still visible, just degraded
		if results[0].BeadsID != "orch-go-deg-1" {
			t.Error("agent identity must be preserved even in degraded mode")
		}
	})

	// Agent exists with manifest but OpenCode is unreachable
	t.Run("opencode_down_still_visible", func(t *testing.T) {
		issues := []beads.Issue{
			{ID: "orch-go-deg-2", Title: "OpenCode down", Status: "in_progress"},
		}
		manifests := map[string]*spawn.AgentManifest{
			"orch-go-deg-2": {BeadsID: "orch-go-deg-2", SessionID: "sess-oc-down", ProjectDir: "/tmp"},
		}
		// Simulate OpenCode unreachable
		liveness := unknownLiveness([]string{"sess-oc-down"})

		results := joinWithReasonCodes(issues, manifests, liveness, nil)
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Status != "unknown" {
			t.Errorf("status = %q, want 'unknown' when OpenCode is down", results[0].Status)
		}
		if results[0].Reason != "opencode_unreachable" {
			t.Errorf("reason = %q, want 'opencode_unreachable'", results[0].Reason)
		}
	})

	// Degraded agent still disappears when closed
	t.Run("degraded_agent_still_disappears_on_close", func(t *testing.T) {
		oldFn := fallbackListWithLabelFn
		defer func() { fallbackListWithLabelFn = oldFn }()

		// Agent was degraded (missing workspace) but still visible
		fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
			return []beads.Issue{
				{ID: "orch-go-deg-3", Title: "Degraded then closed", Status: "in_progress", Labels: []string{"orch:agent"}},
			}, nil
		}
		issues, _ := listTrackedIssuesCLI()
		if len(issues) != 1 {
			t.Fatalf("degraded agent should be visible: got %d", len(issues))
		}

		// Now close it - should disappear regardless of degraded state
		fallbackListWithLabelFn = func(label string) ([]beads.Issue, error) {
			return []beads.Issue{
				{ID: "orch-go-deg-3", Title: "Degraded then closed", Status: "closed", Labels: []string{"orch:agent"}},
			}, nil
		}
		issues, _ = listTrackedIssuesCLI()
		if len(issues) != 0 {
			t.Errorf("closed agent should disappear even if previously degraded, got %d", len(issues))
		}
	})
}

// TestE2ELifecycle_LatestPhaseExtraction verifies that phase comments are correctly
// extracted from beads comments, with the latest phase winning.
func TestE2ELifecycle_LatestPhaseExtraction(t *testing.T) {
	t.Run("latest_phase_wins", func(t *testing.T) {
		comments := []beads.Comment{
			{Text: "Phase: Planning - Reading codebase"},
			{Text: "Found interesting pattern in auth module"},
			{Text: "Phase: Implementing - Writing tests"},
			{Text: "Self-review passed"},
			{Text: "Phase: Complete - All tests passing, 3 files changed"},
		}

		phase := latestPhaseFromComments(comments)
		if phase != "Complete - All tests passing, 3 files changed" {
			t.Errorf("phase = %q, want 'Complete - All tests passing, 3 files changed'", phase)
		}
	})

	t.Run("no_phase_comments_returns_empty", func(t *testing.T) {
		comments := []beads.Comment{
			{Text: "Started working on the feature"},
			{Text: "Found a bug in adjacent code"},
		}

		phase := latestPhaseFromComments(comments)
		if phase != "" {
			t.Errorf("phase = %q, want empty for no phase comments", phase)
		}
	})
}
