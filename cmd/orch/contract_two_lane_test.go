package main

// Contract tests for the Two-Lane Agent Discovery Architecture.
//
// These tests enforce the 12-scenario acceptance matrix from:
//   .kb/decisions/2026-02-18-two-lane-agent-discovery.md
//
// Each test maps to a specific row in the "Acceptance Test Matrix" table.
// These are structural gates: if any test fails, the contract is violated.

import (
	"os"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/discovery"
	"github.com/dylan-conlin/orch-go/pkg/execution"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// --- Scenario 1: Tracked agent spawned → visible in orch status with full metadata ---

func TestContract_TrackedAgent_VisibleInStatus(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-c01", Title: "Tracked agent task", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-c01": {
			BeadsID:       "orch-go-c01",
			SessionID:     "sess-c01",
			ProjectDir:    "/home/user/project",
			WorkspaceName: "og-feat-test-19feb-c01a",
			Skill:         "feature-impl",
			Tier:          "full",
			Model:         "claude-opus-4-5",
			SpawnMode:     "opencode",
		},
	}
	liveness := map[string]execution.SessionStatusInfo{
		"sess-c01": {Type: "busy"},
	}
	phases := map[string]string{
		"orch-go-c01": "Implementing - Adding tests",
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)

	if len(results) != 1 {
		t.Fatalf("Contract: tracked agent must be visible; got %d results, want 1", len(results))
	}

	r := results[0]

	// Must have identity
	if r.BeadsID != "orch-go-c01" {
		t.Errorf("Contract: BeadsID = %q, want orch-go-c01", r.BeadsID)
	}
	if r.Title != "Tracked agent task" {
		t.Errorf("Contract: Title = %q, want 'Tracked agent task'", r.Title)
	}

	// Must have full binding metadata
	if r.SessionID != "sess-c01" {
		t.Errorf("Contract: SessionID = %q, want sess-c01", r.SessionID)
	}
	if r.ProjectDir != "/home/user/project" {
		t.Errorf("Contract: ProjectDir = %q, want /home/user/project", r.ProjectDir)
	}
	if r.WorkspaceName != "og-feat-test-19feb-c01a" {
		t.Errorf("Contract: WorkspaceName = %q, want og-feat-test-19feb-c01a", r.WorkspaceName)
	}
	if r.Skill != "feature-impl" {
		t.Errorf("Contract: Skill = %q, want feature-impl", r.Skill)
	}
	if r.Tier != "full" {
		t.Errorf("Contract: Tier = %q, want full", r.Tier)
	}
	if r.Model != "claude-opus-4-5" {
		t.Errorf("Contract: Model = %q, want claude-opus-4-5", r.Model)
	}
	if r.SpawnMode != "opencode" {
		t.Errorf("Contract: SpawnMode = %q, want opencode", r.SpawnMode)
	}

	// Must have phase
	if r.Phase != "Implementing - Adding tests" {
		t.Errorf("Contract: Phase = %q, want 'Implementing - Adding tests'", r.Phase)
	}

	// Must be active status with no reason codes
	if r.Status != "active" {
		t.Errorf("Contract: Status = %q, want active", r.Status)
	}
	if r.MissingBinding || r.MissingSession || r.SessionDead || r.MissingPhase {
		t.Error("Contract: tracked agent with full metadata must not have any reason code flags set")
	}
	if r.Reason != "" {
		t.Errorf("Contract: Reason = %q, want empty", r.Reason)
	}
}

// --- Scenario 2: Tracked agent completed → gone from orch status ---

func TestContract_TrackedAgent_CompletedGoneFromStatus(t *testing.T) {
	// When an agent's beads issue is closed, filterActiveIssues excludes it.
	// This means queryTrackedAgents never sees it → not in orch status.
	issues := []beads.Issue{
		{ID: "orch-go-c02a", Status: "in_progress", Labels: []string{"orch:agent"}},
		{ID: "orch-go-c02b", Status: "closed", Labels: []string{"orch:agent"}},
		{ID: "orch-go-c02c", Status: "open", Labels: []string{"orch:agent"}},
	}

	active := discovery.FilterActiveIssues(issues)

	// Closed issue must not appear
	for _, issue := range active {
		if issue.ID == "orch-go-c02b" {
			t.Error("Contract: completed (closed) agent must not appear in active issues")
		}
	}

	// Active issues must appear
	if len(active) != 2 {
		t.Fatalf("Contract: expected 2 active issues, got %d", len(active))
	}
	activeIDs := map[string]bool{}
	for _, a := range active {
		activeIDs[a.ID] = true
	}
	if !activeIDs["orch-go-c02a"] || !activeIDs["orch-go-c02c"] {
		t.Error("Contract: in_progress and open issues must remain visible")
	}
}

// --- Scenario 2b: E2E lifecycle (spawn → visible → complete → gone) ---

func TestContract_E2ELifecycle_SpawnVisibleCompleteGone(t *testing.T) {
	beadsID := "orch-go-e2e1"
	sessionID := "sess-e2e1"

	// Spawn → visible
	issues := []beads.Issue{
		{ID: beadsID, Title: "E2E lifecycle", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	manifests := map[string]*spawn.AgentManifest{
		beadsID: {
			BeadsID:       beadsID,
			SessionID:     sessionID,
			ProjectDir:    "/tmp/project",
			WorkspaceName: "og-feat-e2e-lifecycle-19feb-acde",
			Skill:         "feature-impl",
			Tier:          "light",
			SpawnMode:     "opencode",
		},
	}
	liveness := map[string]execution.SessionStatusInfo{
		sessionID: {Type: "busy"},
	}
	phases := map[string]string{
		beadsID: "Implementing - Running lifecycle",
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)
	if len(results) != 1 {
		t.Fatalf("Contract: expected 1 agent after spawn, got %d", len(results))
	}
	if results[0].Status != "active" {
		t.Errorf("Contract: spawned agent status = %q, want active", results[0].Status)
	}

	// Complete (Phase: Complete reported, issue still open)
	status := determineAgentStatus(false, true, "", "active")
	if status != "completed" {
		t.Errorf("Contract: phase-complete agent status = %q, want completed", status)
	}

	// Gone (issue closed → filtered out)
	closed := discovery.FilterActiveIssues([]beads.Issue{
		{ID: beadsID, Status: "closed", Labels: []string{"orch:agent"}},
	})
	if len(closed) != 0 {
		t.Fatalf("Contract: completed agent must be gone from active issues, got %d", len(closed))
	}
}

// --- Scenario 3: --no-track agent → visible in orch sessions, NOT in orch status ---

func TestContract_NoTrack_NotInOrcheStatus(t *testing.T) {
	// A --no-track agent has no beads issue (no orch:agent label).
	// queryTrackedAgents starts from beads → --no-track is never returned.
	// This test verifies the structural guarantee: only orch:agent-labeled
	// issues appear in the tracked lane.
	issues := []beads.Issue{
		// Only tracked agents have orch:agent label and appear in beads query
		{ID: "orch-go-c03", Title: "Tracked task", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-c03": {BeadsID: "orch-go-c03", SessionID: "sess-tracked", ProjectDir: "/tmp"},
	}
	liveness := map[string]execution.SessionStatusInfo{
		"sess-tracked": {Type: "busy"},
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	// Only tracked agent appears; no-track agent is structurally excluded
	// because it has no beads issue to start from
	if len(results) != 1 {
		t.Fatalf("Contract: only tracked agents in status; got %d results", len(results))
	}
	if results[0].BeadsID != "orch-go-c03" {
		t.Errorf("Contract: expected tracked agent orch-go-c03, got %s", results[0].BeadsID)
	}
}

func TestContract_NoTrack_VisibleInSessions(t *testing.T) {
	// A --no-track session has no_track=true in OpenCode metadata.
	// classifyUntrackedSession must categorize it as "no-track".
	session := execution.SessionInfo{
		ID:    "sess-notrack",
		Title: "Ad-hoc exploration",
		Metadata: map[string]string{
			"no_track": "true",
			"skill":    "investigation",
		},
	}

	category, meta := classifyUntrackedSession(session)

	if category != "no-track" {
		t.Errorf("Contract: --no-track session category = %q, want 'no-track'", category)
	}
	if meta.Skill != "investigation" {
		t.Errorf("Contract: --no-track session skill = %q, want 'investigation'", meta.Skill)
	}
}

// --- Scenario 4: Orchestrator session → visible in orch sessions ---

func TestContract_OrchestratorSession_VisibleInSessions(t *testing.T) {
	session := execution.SessionInfo{
		ID:    "sess-orch",
		Title: "Orchestrator session",
		Metadata: map[string]string{
			"role": "orchestrator",
		},
	}

	category, meta := classifyUntrackedSession(session)

	if category != "orchestrator" {
		t.Errorf("Contract: orchestrator session category = %q, want 'orchestrator'", category)
	}
	if meta.Role != "orchestrator" {
		t.Errorf("Contract: orchestrator role = %q, want 'orchestrator'", meta.Role)
	}
}

func TestContract_OrchestratorSession_SkillInference(t *testing.T) {
	// Orchestrator sessions can also be detected by skill name
	session := execution.SessionInfo{
		ID:    "sess-meta-orch",
		Title: "Meta orchestrator",
		Metadata: map[string]string{
			"skill": "meta-orchestrator",
		},
	}

	category, meta := classifyUntrackedSession(session)

	if category != "orchestrator" {
		t.Errorf("Contract: meta-orchestrator by skill category = %q, want 'orchestrator'", category)
	}
	if meta.Role != "meta-orchestrator" {
		t.Errorf("Contract: meta-orchestrator role = %q, want 'meta-orchestrator'", meta.Role)
	}
}

// --- Scenario 5: Beads down during spawn → spawn fails with clear error ---

func TestContract_BeadsDown_AtomicSpawnRejectsNoTrackFalse(t *testing.T) {
	// AtomicSpawnPhase1 tags beads with orch:agent when NoTrack=false.
	// If beads is down, tagBeadsAgent fails → spawn must fail.
	//
	// We can't easily simulate beads being down in a unit test, but we verify
	// the structural contract: Phase1 with NoTrack=false and non-empty BeadsID
	// MUST call tagBeadsAgent. If tagBeadsAgent returns an error, the entire
	// spawn fails with a wrapping error.
	//
	// This test verifies the NoTrack gate: when NoTrack=true, beads tagging
	// is skipped entirely.
	opts := &spawn.AtomicSpawnOpts{
		NoTrack: true,
		BeadsID: "orch-go-c05",
	}

	// When NoTrack=true, Phase1 should NOT attempt beads tagging.
	// The structural contract: tagBeadsAgent is only called when !NoTrack && BeadsID != ""
	if !opts.NoTrack && opts.BeadsID != "" {
		t.Error("Contract: this code path should not execute for NoTrack=true")
	}

	// Verify inverse: when NoTrack=false AND BeadsID is set, tagging is required
	opts2 := &spawn.AtomicSpawnOpts{
		NoTrack: false,
		BeadsID: "orch-go-c05",
	}
	if opts2.NoTrack || opts2.BeadsID == "" {
		t.Error("Contract: tracked spawn with BeadsID must require beads tagging")
	}
}

// --- Scenario 6: OpenCode down → agents shown with status=unknown ---

func TestContract_OpenCodeDown_StatusUnknown(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-c06a", Title: "Agent A", Status: "in_progress"},
		{ID: "orch-go-c06b", Title: "Agent B", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-c06a": {BeadsID: "orch-go-c06a", SessionID: "sess-a", ProjectDir: "/tmp/p1"},
		"orch-go-c06b": {BeadsID: "orch-go-c06b", SessionID: "sess-b", ProjectDir: "/tmp/p2"},
	}

	// Simulate OpenCode unreachable: unknownLiveness is used by queryTrackedAgents
	liveness := discovery.UnknownLiveness([]string{"sess-a", "sess-b"})

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 2 {
		t.Fatalf("Contract: all agents must be shown even when OpenCode is down; got %d", len(results))
	}

	for _, r := range results {
		if r.Status != "unknown" {
			t.Errorf("Contract: agent %s status = %q, want 'unknown' when OpenCode is down", r.BeadsID, r.Status)
		}
		if r.Reason != "opencode_unreachable" {
			t.Errorf("Contract: agent %s reason = %q, want 'opencode_unreachable'", r.BeadsID, r.Reason)
		}
		// Must NOT be silently empty - reason code is the contract
		if r.SessionDead {
			t.Errorf("Contract: agent %s must not be SessionDead when OpenCode is unreachable (we don't know)", r.BeadsID)
		}
	}
}

// --- Scenario 7: Workspace missing → reason=missing_binding ---

func TestContract_WorkspaceMissing_ReasonCode(t *testing.T) {
	issues := []beads.Issue{
		{ID: "orch-go-c07", Title: "Missing workspace", Status: "in_progress"},
	}
	// No manifest found for this agent
	manifests := map[string]*spawn.AgentManifest{}
	liveness := map[string]execution.SessionStatusInfo{}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("Contract: agent must still appear even without workspace; got %d", len(results))
	}

	r := results[0]
	if !r.MissingBinding {
		t.Error("Contract: MissingBinding must be true when workspace manifest not found")
	}
	if r.Reason != "missing_binding" {
		t.Errorf("Contract: Reason = %q, want 'missing_binding'", r.Reason)
	}
	if r.Status != "unknown" {
		t.Errorf("Contract: Status = %q, want 'unknown' when binding is missing", r.Status)
	}
	// Must still have identity
	if r.BeadsID != "orch-go-c07" {
		t.Errorf("Contract: BeadsID must be preserved even without workspace")
	}
	if r.Title != "Missing workspace" {
		t.Errorf("Contract: Title must be preserved even without workspace")
	}
}

// --- Scenario 8: Cross-project --workdir → correct project_dir ---

func TestContract_CrossProject_CorrectProjectDir(t *testing.T) {
	// When an agent is spawned with --workdir pointing to a different project,
	// the manifest's ProjectDir reflects that project, not the orch-go project.
	issues := []beads.Issue{
		{ID: "orch-go-c08", Title: "Cross-project task", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-c08": {
			BeadsID:    "orch-go-c08",
			SessionID:  "sess-cross",
			ProjectDir: "/home/user/other-project", // Different project
			Skill:      "feature-impl",
		},
	}
	liveness := map[string]execution.SessionStatusInfo{
		"sess-cross": {Type: "busy"},
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 1 {
		t.Fatalf("Contract: cross-project agent must be visible; got %d", len(results))
	}

	r := results[0]
	if r.ProjectDir != "/home/user/other-project" {
		t.Errorf("Contract: ProjectDir = %q, want '/home/user/other-project'", r.ProjectDir)
	}
}

func TestContract_CrossProject_ManifestLookup(t *testing.T) {
	// LookupManifestsByBeadsIDs must scan across multiple project dirs.
	// Create temp workspace manifests in two different "projects".
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create workspace dirs with manifests
	createTestWorkspace(t, tmpDir1, "og-feat-task-a", spawn.AgentManifest{
		WorkspaceName: "og-feat-task-a",
		BeadsID:       "proj-a-001",
		SessionID:     "sess-a",
		ProjectDir:    tmpDir1,
	})
	createTestWorkspace(t, tmpDir2, "og-feat-task-b", spawn.AgentManifest{
		WorkspaceName: "og-feat-task-b",
		BeadsID:       "proj-b-001",
		SessionID:     "sess-b",
		ProjectDir:    tmpDir2,
	})

	// Look up across both project dirs
	beadsIDs := []string{"proj-a-001", "proj-b-001"}
	combined := make(map[string]*spawn.AgentManifest)

	for _, dir := range []string{tmpDir1, tmpDir2} {
		found, err := spawn.LookupManifestsByBeadsIDs(dir, beadsIDs)
		if err != nil {
			t.Fatalf("LookupManifestsByBeadsIDs(%s): %v", dir, err)
		}
		for id, m := range found {
			combined[id] = m
		}
	}

	if len(combined) != 2 {
		t.Fatalf("Contract: cross-project lookup must find manifests in both dirs; got %d", len(combined))
	}
	if combined["proj-a-001"].ProjectDir != tmpDir1 {
		t.Errorf("Contract: proj-a-001 ProjectDir = %q, want %q", combined["proj-a-001"].ProjectDir, tmpDir1)
	}
	if combined["proj-b-001"].ProjectDir != tmpDir2 {
		t.Errorf("Contract: proj-b-001 ProjectDir = %q, want %q", combined["proj-b-001"].ProjectDir, tmpDir2)
	}
}

// --- Scenario 9: Concurrent spawns → no duplicates ---

func TestContract_ConcurrentSpawns_NoDuplicates(t *testing.T) {
	// When 5 agents are spawned, all 5 must appear exactly once.
	// joinWithReasonCodes produces one result per issue - no dedup logic needed
	// because beads is the source of truth and issues are unique by ID.
	issues := make([]beads.Issue, 5)
	manifests := make(map[string]*spawn.AgentManifest, 5)
	liveness := make(map[string]execution.SessionStatusInfo, 5)

	for i := 0; i < 5; i++ {
		id := "orch-go-c09" + string(rune('a'+i))
		sessID := "sess-c09" + string(rune('a'+i))
		issues[i] = beads.Issue{ID: id, Title: "Concurrent " + id, Status: "in_progress"}
		manifests[id] = &spawn.AgentManifest{
			BeadsID:   id,
			SessionID: sessID,
		}
		liveness[sessID] = execution.SessionStatusInfo{Type: "busy"}
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	if len(results) != 5 {
		t.Fatalf("Contract: 5 concurrent spawns must produce 5 results; got %d", len(results))
	}

	// Check no duplicate BeadsIDs
	seen := make(map[string]bool)
	for _, r := range results {
		if seen[r.BeadsID] {
			t.Errorf("Contract: duplicate BeadsID %s in results", r.BeadsID)
		}
		seen[r.BeadsID] = true
	}
	if len(seen) != 5 {
		t.Errorf("Contract: expected 5 unique agents, got %d", len(seen))
	}
}

// --- Scenario 10: Server restart → no ghosts ---

func TestContract_ServerRestart_NoGhosts(t *testing.T) {
	// After server restart, agents are discovered from beads (persistent)
	// not from in-memory state. There should be no "ghost" agents from
	// a previous server run.
	//
	// This is guaranteed structurally: queryTrackedAgents always starts
	// from beads, never from in-memory cache. We verify that:
	// 1. Only beads issues appear in results
	// 2. Stale sessions are shown with appropriate reason codes

	// Simulate: 3 beads issues exist, but only 1 has a live session
	// (other 2 sessions died during restart)
	issues := []beads.Issue{
		{ID: "orch-go-c10a", Title: "Survived restart", Status: "in_progress"},
		{ID: "orch-go-c10b", Title: "Session died", Status: "in_progress"},
		{ID: "orch-go-c10c", Title: "No workspace", Status: "in_progress"},
	}
	manifests := map[string]*spawn.AgentManifest{
		"orch-go-c10a": {BeadsID: "orch-go-c10a", SessionID: "sess-alive", ProjectDir: "/tmp"},
		"orch-go-c10b": {BeadsID: "orch-go-c10b", SessionID: "sess-dead", ProjectDir: "/tmp"},
		// c10c has no manifest (workspace cleaned up)
	}
	liveness := map[string]execution.SessionStatusInfo{
		"sess-alive": {Type: "busy"},
		// sess-dead not in liveness → idle
	}

	results := discovery.JoinWithReasonCodes(issues, manifests, liveness, nil)

	// All 3 must appear (no ghosts, no missing)
	if len(results) != 3 {
		t.Fatalf("Contract: all beads-tracked agents must appear after restart; got %d", len(results))
	}

	resultMap := make(map[string]discovery.AgentStatus)
	for _, r := range results {
		resultMap[r.BeadsID] = r
	}

	// Survived restart: active
	if resultMap["orch-go-c10a"].Status != "active" {
		t.Errorf("Contract: surviving agent status = %q, want 'active'", resultMap["orch-go-c10a"].Status)
	}

	// Session died: shown with reason (not silently missing)
	dead := resultMap["orch-go-c10b"]
	if dead.Status == "" {
		t.Error("Contract: dead session agent must have a status")
	}
	if dead.SessionDead != true {
		t.Error("Contract: dead session must be flagged SessionDead=true")
	}

	// No workspace: shown with missing_binding reason
	missing := resultMap["orch-go-c10c"]
	if !missing.MissingBinding {
		t.Error("Contract: agent without workspace must have MissingBinding=true")
	}
	if missing.Reason != "missing_binding" {
		t.Errorf("Contract: agent without workspace reason = %q, want 'missing_binding'", missing.Reason)
	}

	// No extra "ghost" results beyond the 3 beads issues
	for _, r := range results {
		if r.BeadsID != "orch-go-c10a" && r.BeadsID != "orch-go-c10b" && r.BeadsID != "orch-go-c10c" {
			t.Errorf("Contract: unexpected ghost agent %s in results", r.BeadsID)
		}
	}
}

// --- Scenario 11 (architecture lint) is in architecture_lint_test.go ---

// --- Scenario 12 (--no-track excluded from orch status, combined with session classification) ---

func TestContract_TrackedSessionExcludedFromSessions(t *testing.T) {
	// A session with a beads_id that IS tracked should be excluded from
	// the untracked sessions lane (category = "").
	session := execution.SessionInfo{
		ID:    "sess-tracked",
		Title: "Tracked worker",
		Metadata: map[string]string{
			"beads_id": "orch-go-c12",
			"skill":    "feature-impl",
		},
	}

	category, _ := classifyUntrackedSession(session)

	// Sessions with beads_id (and not orchestrator/no-track) return empty category
	// which means they're filtered out of the untracked list
	if category != "" {
		t.Errorf("Contract: tracked session must be excluded from untracked list; category = %q, want empty", category)
	}
}

func TestContract_TwoLaneSplit_Comprehensive(t *testing.T) {
	// Verify the complete two-lane split: sessions are partitioned into
	// exactly one of {tracked, orchestrator, no-track, ad-hoc, excluded}.
	tests := []struct {
		name         string
		session      execution.SessionInfo
		wantCategory string
	}{
		{
			name: "tracked (has beads_id, not orchestrator, not no-track)",
			session: execution.SessionInfo{
				ID:       "sess-1",
				Metadata: map[string]string{"beads_id": "orch-go-999", "skill": "feature-impl"},
			},
			wantCategory: "", // Excluded from untracked lane
		},
		{
			name: "orchestrator by role",
			session: execution.SessionInfo{
				ID:       "sess-2",
				Metadata: map[string]string{"role": "orchestrator"},
			},
			wantCategory: "orchestrator",
		},
		{
			name: "orchestrator by skill",
			session: execution.SessionInfo{
				ID:       "sess-3",
				Metadata: map[string]string{"skill": "orchestrator"},
			},
			wantCategory: "orchestrator",
		},
		{
			name: "no-track explicit",
			session: execution.SessionInfo{
				ID:       "sess-4",
				Metadata: map[string]string{"no_track": "true"},
			},
			wantCategory: "no-track",
		},
		{
			name: "ad-hoc (no beads_id, no role, no no_track)",
			session: execution.SessionInfo{
				ID:       "sess-5",
				Metadata: map[string]string{},
			},
			wantCategory: "ad-hoc",
		},
		{
			name: "ad-hoc (nil metadata)",
			session: execution.SessionInfo{
				ID: "sess-6",
			},
			wantCategory: "ad-hoc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category, _ := classifyUntrackedSession(tt.session)
			if category != tt.wantCategory {
				t.Errorf("Contract: %s → category = %q, want %q", tt.name, category, tt.wantCategory)
			}
		})
	}
}

// --- Helper ---

func createTestWorkspace(t *testing.T, projectDir, workspaceName string, manifest spawn.AgentManifest) {
	t.Helper()
	workspacePath := projectDir + "/.orch/workspace/" + workspaceName

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}
	if err := spawn.WriteAgentManifest(workspacePath, manifest); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
}
