package agent

import (
	"fmt"
	"testing"
	"time"
)

// --- Mock implementations of the interfaces ---

type mockBeadsClient struct {
	labelsRemoved   map[string][]string
	statusUpdated   map[string]string
	assigneeCleared []string
	issuesClosed    map[string]string
	comments        map[string][]string  // beadsID → comments
	trackedIssues   []TrackedIssue       // returned by ListByLabel
	failOn          map[string]error     // key: "operation:beadsID"
}

func newMockBeadsClient() *mockBeadsClient {
	return &mockBeadsClient{
		labelsRemoved:  make(map[string][]string),
		statusUpdated:  make(map[string]string),
		issuesClosed:   make(map[string]string),
		comments:       make(map[string][]string),
		failOn:         make(map[string]error),
	}
}

func (m *mockBeadsClient) AddLabel(beadsID, label string) error {
	if err, ok := m.failOn["add_label:"+beadsID]; ok {
		return err
	}
	return nil
}

func (m *mockBeadsClient) RemoveLabel(beadsID, label string) error {
	if err, ok := m.failOn["remove_label:"+beadsID]; ok {
		return err
	}
	m.labelsRemoved[beadsID] = append(m.labelsRemoved[beadsID], label)
	return nil
}

func (m *mockBeadsClient) UpdateStatus(beadsID, status string) error {
	if err, ok := m.failOn["update_status:"+beadsID]; ok {
		return err
	}
	m.statusUpdated[beadsID] = status
	return nil
}

func (m *mockBeadsClient) SetAssignee(beadsID, assignee string) error {
	return nil
}

func (m *mockBeadsClient) ClearAssignee(beadsID string) error {
	if err, ok := m.failOn["clear_assignee:"+beadsID]; ok {
		return err
	}
	m.assigneeCleared = append(m.assigneeCleared, beadsID)
	return nil
}

func (m *mockBeadsClient) CloseIssue(beadsID, reason string) error {
	if err, ok := m.failOn["close_issue:"+beadsID]; ok {
		return err
	}
	m.issuesClosed[beadsID] = reason
	return nil
}

func (m *mockBeadsClient) GetComments(beadsID string) ([]string, error) {
	if comments, ok := m.comments[beadsID]; ok {
		return comments, nil
	}
	return nil, nil
}

func (m *mockBeadsClient) ListByLabel(label string) ([]TrackedIssue, error) {
	if err, ok := m.failOn["list_by_label:"+label]; ok {
		return nil, err
	}
	return m.trackedIssues, nil
}

type mockOpenCodeClient struct {
	sessions       map[string]bool // sessionID → exists
	deleted        []string
	exported       map[string]string // sessionID → outputPath
	failOn         map[string]error
}

func newMockOpenCodeClient() *mockOpenCodeClient {
	return &mockOpenCodeClient{
		sessions: make(map[string]bool),
		exported: make(map[string]string),
		failOn:   make(map[string]error),
	}
}

func (m *mockOpenCodeClient) SessionExists(sessionID string) (bool, error) {
	if err, ok := m.failOn["session_exists:"+sessionID]; ok {
		return false, err
	}
	return m.sessions[sessionID], nil
}

func (m *mockOpenCodeClient) DeleteSession(sessionID string) error {
	if err, ok := m.failOn["delete_session:"+sessionID]; ok {
		return err
	}
	m.deleted = append(m.deleted, sessionID)
	delete(m.sessions, sessionID)
	return nil
}

func (m *mockOpenCodeClient) ExportActivity(sessionID, outputPath string) error {
	if err, ok := m.failOn["export_activity:"+sessionID]; ok {
		return err
	}
	m.exported[sessionID] = outputPath
	return nil
}

type mockTmuxClient struct {
	windows map[string]bool // name → exists
	killed  []string
	failOn  map[string]error
}

func newMockTmuxClient() *mockTmuxClient {
	return &mockTmuxClient{
		windows: make(map[string]bool),
		failOn:  make(map[string]error),
	}
}

func (m *mockTmuxClient) WindowExists(name string) (bool, error) {
	if err, ok := m.failOn["window_exists:"+name]; ok {
		return false, err
	}
	return m.windows[name], nil
}

func (m *mockTmuxClient) KillWindow(name string) error {
	if err, ok := m.failOn["kill_window:"+name]; ok {
		return err
	}
	m.killed = append(m.killed, name)
	delete(m.windows, name)
	return nil
}

type mockEventLogger struct {
	events []map[string]interface{}
	failOn error
}

func (m *mockEventLogger) Log(eventType string, data map[string]interface{}) error {
	if m.failOn != nil {
		return m.failOn
	}
	data["_type"] = eventType
	m.events = append(m.events, data)
	return nil
}

type mockWorkspaceManager struct {
	archived         []string
	failureReports   map[string]string // path → reason
	existing         map[string]bool
	sessionIDs       map[string]string // path → sessionID
	removed          []string
	workspaces       map[string][]WorkspaceInfo // projectDir → workspaces
	landedArtifacts  map[string]bool            // workspacePath → has artifacts
	failOn           map[string]error
}

func newMockWorkspaceManager() *mockWorkspaceManager {
	return &mockWorkspaceManager{
		failureReports:  make(map[string]string),
		existing:        make(map[string]bool),
		sessionIDs:      make(map[string]string),
		workspaces:      make(map[string][]WorkspaceInfo),
		landedArtifacts: make(map[string]bool),
		failOn:          make(map[string]error),
	}
}

func (m *mockWorkspaceManager) Archive(workspacePath string) error {
	if err, ok := m.failOn["archive:"+workspacePath]; ok {
		return err
	}
	m.archived = append(m.archived, workspacePath)
	return nil
}

func (m *mockWorkspaceManager) WriteFailureReport(workspacePath, reason string) error {
	if err, ok := m.failOn["failure_report:"+workspacePath]; ok {
		return err
	}
	m.failureReports[workspacePath] = reason
	return nil
}

func (m *mockWorkspaceManager) Exists(workspacePath string) bool {
	return m.existing[workspacePath]
}

func (m *mockWorkspaceManager) WriteSessionID(workspacePath, sessionID string) error {
	if err, ok := m.failOn["write_session_id:"+workspacePath]; ok {
		return err
	}
	m.sessionIDs[workspacePath] = sessionID
	return nil
}

func (m *mockWorkspaceManager) Remove(workspacePath string) error {
	if err, ok := m.failOn["remove:"+workspacePath]; ok {
		return err
	}
	m.removed = append(m.removed, workspacePath)
	return nil
}

func (m *mockWorkspaceManager) ScanWorkspaces(projectDir string) ([]WorkspaceInfo, error) {
	if err, ok := m.failOn["scan_workspaces:"+projectDir]; ok {
		return nil, err
	}
	return m.workspaces[projectDir], nil
}

func (m *mockWorkspaceManager) HasLandedArtifacts(workspacePath, projectDir string) (bool, error) {
	if err, ok := m.failOn["has_landed_artifacts:"+workspacePath]; ok {
		return false, err
	}
	return m.landedArtifacts[workspacePath], nil
}

// --- Helper to build a standard test LifecycleManager ---

func testManager() (*lifecycleManager, *mockBeadsClient, *mockOpenCodeClient, *mockTmuxClient, *mockEventLogger, *mockWorkspaceManager) {
	bc := newMockBeadsClient()
	oc := newMockOpenCodeClient()
	tc := newMockTmuxClient()
	el := &mockEventLogger{}
	wm := newMockWorkspaceManager()
	mgr := NewLifecycleManager(bc, oc, tc, el, wm)
	return mgr.(*lifecycleManager), bc, oc, tc, el, wm
}

func testAgent() AgentRef {
	return AgentRef{
		BeadsID:       "orch-go-test1",
		WorkspaceName: "og-feat-test-task-27feb-abcd",
		WorkspacePath: "/tmp/test-workspace",
		SessionID:     "session-123",
		ProjectDir:    "/tmp/project",
		SpawnMode:     "opencode",
	}
}

// --- Abandon Tests ---

func TestAbandon_HappyPath(t *testing.T) {
	mgr, bc, oc, tc, el, wm := testManager()
	agent := testAgent()

	// Set up existing resources
	oc.sessions[agent.SessionID] = true
	tc.windows[agent.WorkspaceName] = true
	wm.existing[agent.WorkspacePath] = true

	event, err := mgr.Abandon(agent, "stuck in loop")
	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	// Verify transition metadata
	if event.Transition != TransitionAbandon {
		t.Errorf("expected transition %s, got %s", TransitionAbandon, event.Transition)
	}
	if event.FromState != StateActive {
		t.Errorf("expected from state %s, got %s", StateActive, event.FromState)
	}
	if event.ToState != StateAbandoned {
		t.Errorf("expected to state %s, got %s", StateAbandoned, event.ToState)
	}
	if event.Reason != "stuck in loop" {
		t.Errorf("expected reason 'stuck in loop', got %q", event.Reason)
	}
	if !event.Success {
		t.Error("expected Success=true")
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	// CRITICAL: orch:agent label must be removed (fixes ghost agent bug)
	labels, ok := bc.labelsRemoved[agent.BeadsID]
	if !ok {
		t.Fatal("orch:agent label was NOT removed — ghost agent bug still present")
	}
	found := false
	for _, l := range labels {
		if l == "orch:agent" {
			found = true
		}
	}
	if !found {
		t.Fatal("orch:agent label was NOT in removed labels — ghost agent bug still present")
	}

	// CRITICAL: assignee must be cleared
	assigneeCleared := false
	for _, id := range bc.assigneeCleared {
		if id == agent.BeadsID {
			assigneeCleared = true
		}
	}
	if !assigneeCleared {
		t.Fatal("assignee was NOT cleared — abandoned agent still appears assigned")
	}

	// Verify status reset to open (for respawn)
	if bc.statusUpdated[agent.BeadsID] != "open" {
		t.Errorf("expected status reset to 'open', got %q", bc.statusUpdated[agent.BeadsID])
	}

	// Verify session deleted
	if len(oc.deleted) != 1 || oc.deleted[0] != agent.SessionID {
		t.Errorf("expected session %s deleted, got %v", agent.SessionID, oc.deleted)
	}

	// Verify tmux window killed
	if len(tc.killed) != 1 || tc.killed[0] != agent.WorkspaceName {
		t.Errorf("expected window %s killed, got %v", agent.WorkspaceName, tc.killed)
	}

	// Verify failure report written
	if wm.failureReports[agent.WorkspacePath] != "stuck in loop" {
		t.Errorf("expected failure report with reason, got %q", wm.failureReports[agent.WorkspacePath])
	}

	// Verify event logged
	if len(el.events) != 1 {
		t.Fatalf("expected 1 event logged, got %d", len(el.events))
	}
	if el.events[0]["_type"] != "agent.abandoned" {
		t.Errorf("expected event type 'agent.abandoned', got %q", el.events[0]["_type"])
	}
}

func TestAbandon_RemovesOrchAgentLabel_Critical(t *testing.T) {
	// This is THE critical test for the ghost agent bug fix.
	// Without removing orch:agent, abandoned agents appear in `bd list -l orch:agent`
	// and count as active agents.
	mgr, bc, _, _, _, _ := testManager()
	agent := testAgent()
	agent.SessionID = "" // No session (claude-mode)

	_, err := mgr.Abandon(agent, "")
	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	labels := bc.labelsRemoved[agent.BeadsID]
	for _, l := range labels {
		if l == "orch:agent" {
			return // Test passes
		}
	}
	t.Fatal("GHOST AGENT BUG: orch:agent label not removed on abandon")
}

func TestAbandon_ClearsAssignee_Critical(t *testing.T) {
	// Without clearing assignee, the abandoned agent still appears "owned"
	// by the dead workspace name, preventing respawn inference.
	mgr, bc, _, _, _, _ := testManager()
	agent := testAgent()
	agent.SessionID = ""

	_, err := mgr.Abandon(agent, "")
	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	for _, id := range bc.assigneeCleared {
		if id == agent.BeadsID {
			return // Test passes
		}
	}
	t.Fatal("Assignee not cleared on abandon")
}

func TestAbandon_NoSession_SkipsSessionCleanup(t *testing.T) {
	mgr, bc, oc, _, _, _ := testManager()
	agent := testAgent()
	agent.SessionID = "" // Claude-mode agent, no OpenCode session

	event, err := mgr.Abandon(agent, "no session")
	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	if !event.Success {
		t.Error("expected Success=true when no session")
	}

	// Session operations should be skipped
	if len(oc.deleted) != 0 {
		t.Errorf("expected no sessions deleted, got %v", oc.deleted)
	}

	// Beads operations should still happen
	if bc.statusUpdated[agent.BeadsID] != "open" {
		t.Error("beads status should still be reset to open")
	}
}

func TestAbandon_NoReason_SkipsFailureReport(t *testing.T) {
	mgr, _, _, _, _, wm := testManager()
	agent := testAgent()
	wm.existing[agent.WorkspacePath] = true

	event, err := mgr.Abandon(agent, "")
	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	if !event.Success {
		t.Error("expected Success=true")
	}

	// No failure report when reason is empty
	if _, exists := wm.failureReports[agent.WorkspacePath]; exists {
		t.Error("failure report should NOT be written when reason is empty")
	}
}

func TestAbandon_BeadsLabelRemovalFails_ReturnsCriticalError(t *testing.T) {
	mgr, bc, _, _, _, _ := testManager()
	agent := testAgent()

	// Make the critical operation fail
	bc.failOn["remove_label:"+agent.BeadsID] = fmt.Errorf("beads socket down")

	event, err := mgr.Abandon(agent, "test")
	if err != nil {
		t.Fatalf("Abandon() should not return error (effects are tracked), got: %v", err)
	}

	// The transition should report critical failure
	if event.Success {
		t.Error("expected Success=false when critical beads operation fails")
	}
	if !event.HasCriticalFailure() {
		t.Error("expected HasCriticalFailure()=true")
	}
}

func TestAbandon_TmuxKillFails_NonCriticalWarning(t *testing.T) {
	mgr, _, _, tc, _, _ := testManager()
	agent := testAgent()
	tc.windows[agent.WorkspaceName] = true
	tc.failOn["kill_window:"+agent.WorkspaceName] = fmt.Errorf("window not found")

	event, err := mgr.Abandon(agent, "test")
	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	// Tmux failure is non-critical — transition should still succeed
	if !event.Success {
		t.Error("expected Success=true — tmux kill is non-critical")
	}

	// But should have a warning
	if len(event.Warnings) == 0 {
		t.Error("expected warning about tmux failure")
	}
}

func TestAbandon_SessionDeleteFails_NonCriticalWarning(t *testing.T) {
	mgr, _, oc, _, _, _ := testManager()
	agent := testAgent()
	oc.sessions[agent.SessionID] = true
	oc.failOn["delete_session:"+agent.SessionID] = fmt.Errorf("API error")

	event, err := mgr.Abandon(agent, "test")
	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	// Session deletion failure is non-critical
	if !event.Success {
		t.Error("expected Success=true — session delete is non-critical")
	}
	if len(event.Warnings) == 0 {
		t.Error("expected warning about session deletion failure")
	}
}

func TestAbandon_EventLogFails_NonCriticalWarning(t *testing.T) {
	mgr, _, _, _, el, _ := testManager()
	agent := testAgent()
	el.failOn = fmt.Errorf("disk full")

	event, err := mgr.Abandon(agent, "test")
	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	// Event logging failure is non-critical
	if !event.Success {
		t.Error("expected Success=true — event logging is non-critical")
	}
}

func TestAbandon_EffectsOrder(t *testing.T) {
	// Verify effects are present for all subsystems
	mgr, _, oc, tc, _, wm := testManager()
	agent := testAgent()
	oc.sessions[agent.SessionID] = true
	tc.windows[agent.WorkspaceName] = true
	wm.existing[agent.WorkspacePath] = true

	event, err := mgr.Abandon(agent, "test reason")
	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	// Check that we have effects from all subsystems
	subsystems := make(map[string]bool)
	for _, e := range event.Effects {
		subsystems[e.Subsystem] = true
	}

	// Critical: beads operations
	if !subsystems["beads"] {
		t.Error("missing beads effects")
	}

	// Non-critical: cleanup operations
	if !subsystems["opencode"] {
		t.Error("missing opencode effects")
	}
	if !subsystems["tmux"] {
		t.Error("missing tmux effects")
	}
	if !subsystems["events"] {
		t.Error("missing events effects")
	}
	if !subsystems["workspace"] {
		t.Error("missing workspace effects")
	}
}

func TestAbandon_TimestampSet(t *testing.T) {
	mgr, _, _, _, _, _ := testManager()
	agent := testAgent()

	before := time.Now()
	event, err := mgr.Abandon(agent, "")
	after := time.Now()

	if err != nil {
		t.Fatalf("Abandon() returned error: %v", err)
	}

	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Error("timestamp should be within test execution window")
	}
}

// --- BeginSpawn Tests ---

func TestBeginSpawn_HappyPath_Tracked(t *testing.T) {
	mgr, bc, _, _, _, _ := testManager()

	input := SpawnInput{
		BeadsID:       "proj-123",
		WorkspaceName: "og-feat-test-27feb-abc1",
		WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
		ProjectDir:    "/tmp/proj",
		SpawnMode:     "opencode",
	}

	handle, err := mgr.BeginSpawn(input)
	if err != nil {
		t.Fatalf("BeginSpawn() returned error: %v", err)
	}

	if handle == nil {
		t.Fatal("expected non-nil handle")
	}

	// Verify beads was tagged
	// The mock doesn't track adds directly, but the effect should be recorded
	if len(handle.Event().Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(handle.Event().Effects))
	}
	if handle.Event().Effects[0].Subsystem != "beads" {
		t.Errorf("expected beads subsystem, got %q", handle.Event().Effects[0].Subsystem)
	}
	if handle.Event().Effects[0].Operation != "add_label" {
		t.Errorf("expected add_label operation, got %q", handle.Event().Effects[0].Operation)
	}
	if !handle.Event().Effects[0].Success {
		t.Error("expected beads add_label to succeed")
	}

	// Verify agent ref
	if handle.Agent.BeadsID != "proj-123" {
		t.Errorf("BeadsID: got %q, want %q", handle.Agent.BeadsID, "proj-123")
	}

	// Verify rollback removes label
	handle.SafeRollback()
	labels := bc.labelsRemoved["proj-123"]
	found := false
	for _, l := range labels {
		if l == "orch:agent" {
			found = true
		}
	}
	if !found {
		t.Error("rollback should remove orch:agent label")
	}
}

func TestBeginSpawn_HappyPath_NoTrack(t *testing.T) {
	mgr, _, _, _, _, _ := testManager()

	input := SpawnInput{
		WorkspaceName: "og-feat-test-27feb-abc1",
		WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
		ProjectDir:    "/tmp/proj",
		SpawnMode:     "opencode",
		NoTrack:       true,
	}

	handle, err := mgr.BeginSpawn(input)
	if err != nil {
		t.Fatalf("BeginSpawn() returned error: %v", err)
	}

	// No effects when NoTrack (beads tagging skipped)
	if len(handle.Event().Effects) != 0 {
		t.Errorf("expected 0 effects for NoTrack spawn, got %d", len(handle.Event().Effects))
	}
}

func TestBeginSpawn_InvalidInput(t *testing.T) {
	mgr, _, _, _, _, _ := testManager()

	input := SpawnInput{
		// Missing required fields
		BeadsID: "proj-123",
	}

	_, err := mgr.BeginSpawn(input)
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestBeginSpawn_BeadsTagFails_ReturnsError(t *testing.T) {
	mgr, bc, _, _, _, _ := testManager()
	bc.failOn["add_label:proj-123"] = fmt.Errorf("beads daemon unreachable")

	input := SpawnInput{
		BeadsID:       "proj-123",
		WorkspaceName: "og-feat-test-27feb-abc1",
		WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
		ProjectDir:    "/tmp/proj",
		SpawnMode:     "opencode",
	}

	_, err := mgr.BeginSpawn(input)
	if err == nil {
		t.Error("expected error when beads tag fails")
	}
}

// --- ActivateSpawn Tests ---

func TestActivateSpawn_HappyPath(t *testing.T) {
	mgr, _, _, _, el, wm := testManager()

	input := SpawnInput{
		BeadsID:       "proj-123",
		WorkspaceName: "og-feat-test-27feb-abc1",
		WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
		ProjectDir:    "/tmp/proj",
		SpawnMode:     "opencode",
		NoTrack:       true, // skip beads for simplicity
	}

	handle, err := mgr.BeginSpawn(input)
	if err != nil {
		t.Fatalf("BeginSpawn() returned error: %v", err)
	}

	event, err := mgr.ActivateSpawn(handle, "session-456")
	if err != nil {
		t.Fatalf("ActivateSpawn() returned error: %v", err)
	}

	// Verify transition metadata
	if event.Transition != TransitionSpawn {
		t.Errorf("Transition: got %q, want %q", event.Transition, TransitionSpawn)
	}
	if event.FromState != StateSpawning {
		t.Errorf("FromState: got %q, want %q", event.FromState, StateSpawning)
	}
	if event.ToState != StateActive {
		t.Errorf("ToState: got %q, want %q", event.ToState, StateActive)
	}
	if event.Agent.SessionID != "session-456" {
		t.Errorf("SessionID: got %q, want %q", event.Agent.SessionID, "session-456")
	}
	if !event.Success {
		t.Error("expected Success=true")
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	// Verify session ID written to workspace
	if wm.sessionIDs["/tmp/.orch/workspace/og-feat-test-27feb-abc1"] != "session-456" {
		t.Errorf("session ID not written to workspace")
	}

	// Verify event logged
	if len(el.events) != 1 {
		t.Fatalf("expected 1 event logged, got %d", len(el.events))
	}
	if el.events[0]["_type"] != "session.spawned" {
		t.Errorf("expected event type 'session.spawned', got %q", el.events[0]["_type"])
	}
}

func TestActivateSpawn_EmptySessionID(t *testing.T) {
	mgr, _, _, _, el, wm := testManager()

	input := SpawnInput{
		BeadsID:       "proj-123",
		WorkspaceName: "og-feat-test-27feb-abc1",
		WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
		ProjectDir:    "/tmp/proj",
		SpawnMode:     "claude",
		NoTrack:       true,
	}

	handle, _ := mgr.BeginSpawn(input)
	event, err := mgr.ActivateSpawn(handle, "")
	if err != nil {
		t.Fatalf("ActivateSpawn() returned error: %v", err)
	}

	// Empty session ID means no workspace write for session ID
	if len(wm.sessionIDs) != 0 {
		t.Error("should not write session ID when empty")
	}

	// Event should still be logged
	if len(el.events) != 1 {
		t.Errorf("expected 1 event logged, got %d", len(el.events))
	}

	if !event.Success {
		t.Error("expected Success=true")
	}
}

func TestActivateSpawn_NilHandle(t *testing.T) {
	mgr, _, _, _, _, _ := testManager()

	_, err := mgr.ActivateSpawn(nil, "session-123")
	if err == nil {
		t.Error("expected error for nil handle")
	}
}

func TestActivateSpawn_SessionIDWriteFails_NonCritical(t *testing.T) {
	mgr, _, _, _, _, wm := testManager()
	wm.failOn["write_session_id:/tmp/.orch/workspace/og-feat-test-27feb-abc1"] = fmt.Errorf("disk full")

	input := SpawnInput{
		BeadsID:       "proj-123",
		WorkspaceName: "og-feat-test-27feb-abc1",
		WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
		ProjectDir:    "/tmp/proj",
		SpawnMode:     "opencode",
		NoTrack:       true,
	}

	handle, _ := mgr.BeginSpawn(input)
	event, err := mgr.ActivateSpawn(handle, "session-456")
	if err != nil {
		t.Fatalf("ActivateSpawn() should not return error, got: %v", err)
	}

	// Session ID write failure is non-critical (session already running)
	if !event.Success {
		t.Error("expected Success=true — session ID write is non-critical")
	}
	if len(event.Warnings) == 0 {
		t.Error("expected warning about session ID write failure")
	}
}

func TestSpawnFullLifecycle_BeginThenActivate(t *testing.T) {
	mgr, _, _, _, el, wm := testManager()

	input := SpawnInput{
		BeadsID:       "proj-123",
		WorkspaceName: "og-feat-test-27feb-abc1",
		WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
		ProjectDir:    "/tmp/proj",
		SpawnMode:     "opencode",
		NoTrack:       true,
	}

	// Phase 1: BeginSpawn
	handle, err := mgr.BeginSpawn(input)
	if err != nil {
		t.Fatalf("BeginSpawn() error: %v", err)
	}

	// Simulate: caller creates session here (not lifecycle's concern)

	// Phase 2: ActivateSpawn
	event, err := mgr.ActivateSpawn(handle, "session-789")
	if err != nil {
		t.Fatalf("ActivateSpawn() error: %v", err)
	}

	// Full lifecycle should produce a single coherent TransitionEvent
	if event.Transition != TransitionSpawn {
		t.Errorf("Transition: got %q, want %q", event.Transition, TransitionSpawn)
	}
	if event.FromState != StateSpawning {
		t.Errorf("FromState: got %q, want %q", event.FromState, StateSpawning)
	}
	if event.ToState != StateActive {
		t.Errorf("ToState: got %q, want %q", event.ToState, StateActive)
	}
	if !event.Success {
		t.Error("expected successful transition")
	}

	// Effects from both phases should be in the event
	// Phase 2: workspace write + event log
	subsystems := make(map[string]bool)
	for _, e := range event.Effects {
		subsystems[e.Subsystem] = true
	}

	if !subsystems["workspace"] {
		t.Error("missing workspace effect from Phase 2")
	}
	if !subsystems["events"] {
		t.Error("missing events effect from Phase 2")
	}

	// Verify session ID written
	if wm.sessionIDs["/tmp/.orch/workspace/og-feat-test-27feb-abc1"] != "session-789" {
		t.Error("session ID not written")
	}

	// Verify event logged
	if len(el.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(el.events))
	}
	if el.events[0]["session_id"] != "session-789" {
		t.Errorf("logged session_id: got %q, want %q", el.events[0]["session_id"], "session-789")
	}
}

// --- Complete Tests ---

func TestComplete_HappyPath(t *testing.T) {
	mgr, bc, oc, tc, el, wm := testManager()
	agent := testAgent()

	// Set up existing resources
	oc.sessions[agent.SessionID] = true
	tc.windows[agent.WorkspaceName] = true
	wm.existing[agent.WorkspacePath] = true

	event, err := mgr.Complete(agent, "All tests passing, deliverables verified")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	// Verify transition metadata
	if event.Transition != TransitionComplete {
		t.Errorf("expected transition %s, got %s", TransitionComplete, event.Transition)
	}
	if event.FromState != StatePhaseComplete {
		t.Errorf("expected from state %s, got %s", StatePhaseComplete, event.FromState)
	}
	if event.ToState != StateCompleted {
		t.Errorf("expected to state %s, got %s", StateCompleted, event.ToState)
	}
	if event.Reason != "All tests passing, deliverables verified" {
		t.Errorf("expected reason, got %q", event.Reason)
	}
	if !event.Success {
		t.Error("expected Success=true")
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	// CRITICAL: beads issue must be closed
	if reason, ok := bc.issuesClosed[agent.BeadsID]; !ok {
		t.Fatal("beads issue was NOT closed")
	} else if reason != "All tests passing, deliverables verified" {
		t.Errorf("close reason: got %q", reason)
	}

	// CRITICAL: orch:agent label must be removed (prevents ghost agents)
	labels, ok := bc.labelsRemoved[agent.BeadsID]
	if !ok {
		t.Fatal("orch:agent label was NOT removed — ghost agent bug")
	}
	found := false
	for _, l := range labels {
		if l == "orch:agent" {
			found = true
		}
	}
	if !found {
		t.Fatal("orch:agent label was NOT in removed labels")
	}

	// Verify session deleted
	if len(oc.deleted) != 1 || oc.deleted[0] != agent.SessionID {
		t.Errorf("expected session %s deleted, got %v", agent.SessionID, oc.deleted)
	}

	// Verify tmux window killed
	if len(tc.killed) != 1 || tc.killed[0] != agent.WorkspaceName {
		t.Errorf("expected window %s killed, got %v", agent.WorkspaceName, tc.killed)
	}

	// Verify workspace archived
	if len(wm.archived) != 1 || wm.archived[0] != agent.WorkspacePath {
		t.Errorf("expected workspace archived, got %v", wm.archived)
	}

	// Verify event logged
	if len(el.events) != 1 {
		t.Fatalf("expected 1 event logged, got %d", len(el.events))
	}
	if el.events[0]["_type"] != "agent.completed" {
		t.Errorf("expected event type 'agent.completed', got %q", el.events[0]["_type"])
	}
}

func TestComplete_ClosesBeadsIssue_Critical(t *testing.T) {
	mgr, bc, _, _, _, _ := testManager()
	agent := testAgent()

	_, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if _, ok := bc.issuesClosed[agent.BeadsID]; !ok {
		t.Fatal("beads issue was NOT closed — primary Complete operation missing")
	}
}

func TestComplete_RemovesOrchAgentLabel(t *testing.T) {
	mgr, bc, _, _, _, _ := testManager()
	agent := testAgent()

	_, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	labels := bc.labelsRemoved[agent.BeadsID]
	for _, l := range labels {
		if l == "orch:agent" {
			return // Test passes
		}
	}
	t.Fatal("GHOST AGENT BUG: orch:agent label not removed on complete")
}

func TestComplete_NoSession_SkipsSessionCleanup(t *testing.T) {
	mgr, bc, oc, _, _, _ := testManager()
	agent := testAgent()
	agent.SessionID = "" // Claude-mode agent, no OpenCode session

	event, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if !event.Success {
		t.Error("expected Success=true when no session")
	}

	// Session operations should be skipped
	if len(oc.deleted) != 0 {
		t.Errorf("expected no sessions deleted, got %v", oc.deleted)
	}

	// Beads operations should still happen
	if _, ok := bc.issuesClosed[agent.BeadsID]; !ok {
		t.Error("beads issue should still be closed")
	}
}

func TestComplete_BeadsCloseFails_ReturnsCriticalError(t *testing.T) {
	mgr, bc, _, _, _, _ := testManager()
	agent := testAgent()

	bc.failOn["close_issue:"+agent.BeadsID] = fmt.Errorf("beads socket down")

	event, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() should not return error (effects are tracked), got: %v", err)
	}

	if event.Success {
		t.Error("expected Success=false when critical beads close fails")
	}
	if !event.HasCriticalFailure() {
		t.Error("expected HasCriticalFailure()=true")
	}
}

func TestComplete_TmuxKillFails_NonCriticalWarning(t *testing.T) {
	mgr, _, _, tc, _, _ := testManager()
	agent := testAgent()
	tc.windows[agent.WorkspaceName] = true
	tc.failOn["kill_window:"+agent.WorkspaceName] = fmt.Errorf("window not found")

	event, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	// Tmux failure is non-critical
	if !event.Success {
		t.Error("expected Success=true — tmux kill is non-critical")
	}
	if len(event.Warnings) == 0 {
		t.Error("expected warning about tmux failure")
	}
}

func TestComplete_ArchiveFails_NonCriticalWarning(t *testing.T) {
	mgr, _, _, _, _, wm := testManager()
	agent := testAgent()
	wm.existing[agent.WorkspacePath] = true
	wm.failOn["archive:"+agent.WorkspacePath] = fmt.Errorf("permission denied")

	event, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if !event.Success {
		t.Error("expected Success=true — archive is non-critical")
	}
	if len(event.Warnings) == 0 {
		t.Error("expected warning about archive failure")
	}
}

func TestComplete_SessionDeleteFails_NonCriticalWarning(t *testing.T) {
	mgr, _, oc, _, _, _ := testManager()
	agent := testAgent()
	oc.sessions[agent.SessionID] = true
	oc.failOn["delete_session:"+agent.SessionID] = fmt.Errorf("API error")

	event, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if !event.Success {
		t.Error("expected Success=true — session delete is non-critical")
	}
	if len(event.Warnings) == 0 {
		t.Error("expected warning about session deletion failure")
	}
}

func TestComplete_EventLogFails_NonCriticalWarning(t *testing.T) {
	mgr, _, _, _, el, _ := testManager()
	agent := testAgent()
	el.failOn = fmt.Errorf("disk full")

	event, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if !event.Success {
		t.Error("expected Success=true — event logging is non-critical")
	}
}

func TestComplete_EffectsOrder(t *testing.T) {
	mgr, _, oc, tc, _, wm := testManager()
	agent := testAgent()
	oc.sessions[agent.SessionID] = true
	tc.windows[agent.WorkspaceName] = true
	wm.existing[agent.WorkspacePath] = true

	event, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	// Check that we have effects from all subsystems
	subsystems := make(map[string]bool)
	for _, e := range event.Effects {
		subsystems[e.Subsystem] = true
	}

	if !subsystems["beads"] {
		t.Error("missing beads effects")
	}
	if !subsystems["opencode"] {
		t.Error("missing opencode effects")
	}
	if !subsystems["tmux"] {
		t.Error("missing tmux effects")
	}
	if !subsystems["events"] {
		t.Error("missing events effects")
	}
	if !subsystems["workspace"] {
		t.Error("missing workspace effects")
	}

	// Verify beads close_issue is the first effect (critical path first)
	if event.Effects[0].Subsystem != "beads" || event.Effects[0].Operation != "close_issue" {
		t.Errorf("first effect should be beads/close_issue, got %s/%s",
			event.Effects[0].Subsystem, event.Effects[0].Operation)
	}
}

func TestComplete_NoWorkspace_SkipsArchive(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()
	agent := testAgent()
	agent.WorkspacePath = ""
	agent.WorkspaceName = ""

	event, err := mgr.Complete(agent, "done")
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if !event.Success {
		t.Error("expected Success=true")
	}

	// Workspace archive should be skipped
	if len(wm.archived) != 0 {
		t.Errorf("expected no archives, got %v", wm.archived)
	}

	// Beads operations should still happen
	if _, ok := bc.issuesClosed[agent.BeadsID]; !ok {
		t.Error("beads issue should still be closed")
	}
}

func TestComplete_TimestampSet(t *testing.T) {
	mgr, _, _, _, _, _ := testManager()
	agent := testAgent()

	before := time.Now()
	event, err := mgr.Complete(agent, "done")
	after := time.Now()

	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Error("timestamp should be within test execution window")
	}
}

// --- ForceComplete Tests ---

func TestForceComplete_HappyPath(t *testing.T) {
	mgr, bc, oc, tc, el, wm := testManager()
	agent := testAgent()

	oc.sessions[agent.SessionID] = true
	tc.windows[agent.WorkspaceName] = true
	wm.existing[agent.WorkspacePath] = true

	event, err := mgr.ForceComplete(agent, "GC: orphaned agent with Phase: Complete")
	if err != nil {
		t.Fatalf("ForceComplete() returned error: %v", err)
	}

	// Verify transition metadata — key difference from Complete: from StateOrphaned
	if event.Transition != TransitionForceComplete {
		t.Errorf("expected transition %s, got %s", TransitionForceComplete, event.Transition)
	}
	if event.FromState != StateOrphaned {
		t.Errorf("expected from state %s, got %s", StateOrphaned, event.FromState)
	}
	if event.ToState != StateCompleted {
		t.Errorf("expected to state %s, got %s", StateCompleted, event.ToState)
	}
	if !event.Success {
		t.Error("expected Success=true")
	}

	// CRITICAL: beads issue must be closed
	if _, ok := bc.issuesClosed[agent.BeadsID]; !ok {
		t.Fatal("beads issue was NOT closed")
	}

	// orch:agent label must be removed
	labels := bc.labelsRemoved[agent.BeadsID]
	found := false
	for _, l := range labels {
		if l == "orch:agent" {
			found = true
		}
	}
	if !found {
		t.Fatal("orch:agent label was NOT removed")
	}

	// Session deleted
	if len(oc.deleted) != 1 || oc.deleted[0] != agent.SessionID {
		t.Errorf("expected session %s deleted, got %v", agent.SessionID, oc.deleted)
	}

	// Tmux window killed
	if len(tc.killed) != 1 || tc.killed[0] != agent.WorkspaceName {
		t.Errorf("expected window %s killed, got %v", agent.WorkspaceName, tc.killed)
	}

	// Workspace archived
	if len(wm.archived) != 1 || wm.archived[0] != agent.WorkspacePath {
		t.Errorf("expected workspace archived, got %v", wm.archived)
	}

	// Event logged
	if len(el.events) != 1 {
		t.Fatalf("expected 1 event logged, got %d", len(el.events))
	}
	if el.events[0]["_type"] != "agent.force_completed" {
		t.Errorf("expected event type 'agent.force_completed', got %q", el.events[0]["_type"])
	}
}

func TestForceComplete_BeadsCloseFails_CriticalError(t *testing.T) {
	mgr, bc, _, _, _, _ := testManager()
	agent := testAgent()
	bc.failOn["close_issue:"+agent.BeadsID] = fmt.Errorf("beads down")

	event, err := mgr.ForceComplete(agent, "GC")
	if err != nil {
		t.Fatalf("ForceComplete() should not return error, got: %v", err)
	}
	if event.Success {
		t.Error("expected Success=false when critical close fails")
	}
}

func TestForceComplete_NoSession_SkipsSessionCleanup(t *testing.T) {
	mgr, _, oc, _, _, _ := testManager()
	agent := testAgent()
	agent.SessionID = "" // Claude-mode agent

	event, err := mgr.ForceComplete(agent, "GC")
	if err != nil {
		t.Fatalf("ForceComplete() returned error: %v", err)
	}
	if !event.Success {
		t.Error("expected Success=true")
	}
	if len(oc.deleted) != 0 {
		t.Error("should not delete sessions for claude-mode agent")
	}
}

// --- ForceAbandon Tests ---

func TestForceAbandon_HappyPath(t *testing.T) {
	mgr, bc, oc, tc, el, wm := testManager()
	agent := testAgent()

	oc.sessions[agent.SessionID] = true
	tc.windows[agent.WorkspaceName] = true
	wm.existing[agent.WorkspacePath] = true

	event, err := mgr.ForceAbandon(agent)
	if err != nil {
		t.Fatalf("ForceAbandon() returned error: %v", err)
	}

	// Verify transition metadata — key difference from Abandon: from StateOrphaned
	if event.Transition != TransitionForceAbandon {
		t.Errorf("expected transition %s, got %s", TransitionForceAbandon, event.Transition)
	}
	if event.FromState != StateOrphaned {
		t.Errorf("expected from state %s, got %s", StateOrphaned, event.FromState)
	}
	if event.ToState != StateAbandoned {
		t.Errorf("expected to state %s, got %s", StateAbandoned, event.ToState)
	}
	if !event.Success {
		t.Error("expected Success=true")
	}

	// CRITICAL: orch:agent label removed (ghost agent fix)
	labels := bc.labelsRemoved[agent.BeadsID]
	found := false
	for _, l := range labels {
		if l == "orch:agent" {
			found = true
		}
	}
	if !found {
		t.Fatal("orch:agent label was NOT removed")
	}

	// Assignee cleared
	assigneeCleared := false
	for _, id := range bc.assigneeCleared {
		if id == agent.BeadsID {
			assigneeCleared = true
		}
	}
	if !assigneeCleared {
		t.Fatal("assignee was NOT cleared")
	}

	// Status reset to open (for respawn)
	if bc.statusUpdated[agent.BeadsID] != "open" {
		t.Errorf("expected status 'open', got %q", bc.statusUpdated[agent.BeadsID])
	}

	// Session deleted
	if len(oc.deleted) != 1 {
		t.Errorf("expected 1 session deleted, got %d", len(oc.deleted))
	}

	// Tmux window killed
	if len(tc.killed) != 1 {
		t.Errorf("expected 1 window killed, got %d", len(tc.killed))
	}

	// Event logged as force_abandoned
	if len(el.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(el.events))
	}
	if el.events[0]["_type"] != "agent.force_abandoned" {
		t.Errorf("expected 'agent.force_abandoned', got %q", el.events[0]["_type"])
	}
}

func TestForceAbandon_BeadsLabelRemoveFails_CriticalError(t *testing.T) {
	mgr, bc, _, _, _, _ := testManager()
	agent := testAgent()
	bc.failOn["remove_label:"+agent.BeadsID] = fmt.Errorf("beads down")

	event, err := mgr.ForceAbandon(agent)
	if err != nil {
		t.Fatalf("ForceAbandon() should not return error, got: %v", err)
	}
	if event.Success {
		t.Error("expected Success=false when critical label removal fails")
	}
}

func TestForceAbandon_NoSession_SkipsSessionCleanup(t *testing.T) {
	mgr, _, oc, _, _, _ := testManager()
	agent := testAgent()
	agent.SessionID = "" // Claude-mode agent

	event, err := mgr.ForceAbandon(agent)
	if err != nil {
		t.Fatalf("ForceAbandon() returned error: %v", err)
	}
	if !event.Success {
		t.Error("expected Success=true")
	}
	if len(oc.deleted) != 0 {
		t.Error("should not delete sessions for claude-mode agent")
	}
}

func TestForceAbandon_WritesFailureReport(t *testing.T) {
	mgr, _, _, _, _, wm := testManager()
	agent := testAgent()
	wm.existing[agent.WorkspacePath] = true

	_, err := mgr.ForceAbandon(agent)
	if err != nil {
		t.Fatalf("ForceAbandon() returned error: %v", err)
	}

	// ForceAbandon should write a failure report explaining GC action
	if _, ok := wm.failureReports[agent.WorkspacePath]; !ok {
		t.Error("expected failure report to be written for GC abandonment")
	}
}

// --- DetectOrphans Tests ---

func TestDetectOrphans_NoTrackedAgents_EmptyResult(t *testing.T) {
	mgr, _, _, _, _, _ := testManager()

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d", len(result.Orphans))
	}
}

func TestDetectOrphans_ActiveOpenCodeSession_NotOrphaned(t *testing.T) {
	mgr, bc, oc, _, _, wm := testManager()

	// Agent tracked in beads with in_progress status
	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "proj-001", Status: "in_progress", Labels: []string{"orch:agent"}},
	}

	// Workspace exists with session ID
	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{
		{
			Name:      "og-feat-test-27feb-abc1",
			Path:      "/tmp/proj/.orch/workspace/og-feat-test-27feb-abc1",
			BeadsID:   "proj-001",
			SessionID: "session-100",
			SpawnMode: "opencode",
			SpawnTime: time.Now().Add(-1 * time.Hour),
		},
	}

	// OpenCode session is alive
	oc.sessions["session-100"] = true

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	if len(result.Orphans) != 0 {
		t.Errorf("agent with live session should NOT be orphaned, got %d orphans", len(result.Orphans))
	}
	if result.Scanned != 1 {
		t.Errorf("expected 1 scanned, got %d", result.Scanned)
	}
}

func TestDetectOrphans_ClaudeAgentWithTmuxWindow_NotOrphaned(t *testing.T) {
	mgr, bc, _, tc, _, wm := testManager()

	// Claude-mode agent (no OpenCode session, uses tmux)
	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "proj-002", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{
		{
			Name:      "og-feat-test-27feb-def2",
			Path:      "/tmp/proj/.orch/workspace/og-feat-test-27feb-def2",
			BeadsID:   "proj-002",
			SessionID: "", // No OpenCode session
			SpawnMode: "claude",
			SpawnTime: time.Now().Add(-2 * time.Hour),
		},
	}

	// Tmux window exists — Claude agent is alive
	tc.windows["og-feat-test-27feb-def2"] = true

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	if len(result.Orphans) != 0 {
		t.Errorf("Claude agent with live tmux window should NOT be orphaned, got %d orphans", len(result.Orphans))
	}
}

func TestDetectOrphans_DeadSessionNoPhase_Orphaned(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()

	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "proj-003", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{
		{
			Name:      "og-feat-test-27feb-ghi3",
			Path:      "/tmp/proj/.orch/workspace/og-feat-test-27feb-ghi3",
			BeadsID:   "proj-003",
			SessionID: "session-dead",
			SpawnMode: "opencode",
			SpawnTime: time.Now().Add(-2 * time.Hour),
		},
	}

	// Session does NOT exist, no tmux window, no phase
	// (oc.sessions is empty, tc.windows is empty, bc.comments is empty)

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	if len(result.Orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(result.Orphans))
	}
	orphan := result.Orphans[0]
	if orphan.Agent.BeadsID != "proj-003" {
		t.Errorf("expected BeadsID 'proj-003', got %q", orphan.Agent.BeadsID)
	}
	if orphan.Reason != "no_live_execution" {
		t.Errorf("expected reason 'no_live_execution', got %q", orphan.Reason)
	}
	if orphan.ShouldRetry {
		t.Error("agent without triage:ready label should not be retried")
	}
}

func TestDetectOrphans_PhaseComplete_ShouldNotRetry(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()

	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "proj-004", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	bc.comments["proj-004"] = []string{
		"Phase: Planning - started",
		"Phase: Complete - All tests passing",
	}
	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{
		{
			Name:      "og-feat-test-27feb-jkl4",
			Path:      "/tmp/proj/.orch/workspace/og-feat-test-27feb-jkl4",
			BeadsID:   "proj-004",
			SessionID: "session-gone",
			SpawnMode: "opencode",
			SpawnTime: time.Now().Add(-3 * time.Hour),
		},
	}

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	if len(result.Orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(result.Orphans))
	}
	orphan := result.Orphans[0]
	if orphan.LastPhase != "Complete" {
		t.Errorf("expected LastPhase 'Complete', got %q", orphan.LastPhase)
	}
	if orphan.ShouldRetry {
		t.Error("Phase: Complete agent should NOT be retried")
	}
}

func TestDetectOrphans_TriageReadyLabel_ShouldRetry(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()

	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "proj-005", Status: "in_progress", Labels: []string{"orch:agent", "triage:ready"}},
	}
	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{
		{
			Name:      "og-feat-test-27feb-mno5",
			Path:      "/tmp/proj/.orch/workspace/og-feat-test-27feb-mno5",
			BeadsID:   "proj-005",
			SessionID: "",
			SpawnMode: "opencode",
			SpawnTime: time.Now().Add(-1 * time.Hour),
		},
	}

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	if len(result.Orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(result.Orphans))
	}
	if !result.Orphans[0].ShouldRetry {
		t.Error("agent with triage:ready label should be retried")
	}
}

func TestDetectOrphans_UnderThreshold_NotOrphaned(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()

	// Agent spawned recently (5 min ago), no session but under threshold
	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "proj-006", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{
		{
			Name:      "og-feat-test-27feb-pqr6",
			Path:      "/tmp/proj/.orch/workspace/og-feat-test-27feb-pqr6",
			BeadsID:   "proj-006",
			SessionID: "session-gone",
			SpawnMode: "opencode",
			SpawnTime: time.Now().Add(-5 * time.Minute), // Only 5 minutes old
		},
	}

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	if len(result.Orphans) != 0 {
		t.Errorf("agent under threshold should NOT be orphaned, got %d orphans", len(result.Orphans))
	}
}

func TestDetectOrphans_ClosedIssues_Filtered(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()

	// Include both in_progress and closed issues
	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "proj-007", Status: "in_progress", Labels: []string{"orch:agent"}},
		{BeadsID: "proj-008", Status: "closed", Labels: []string{"orch:agent"}},
	}
	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{
		{
			Name:      "og-feat-test-27feb-stu7",
			Path:      "/tmp/proj/.orch/workspace/og-feat-test-27feb-stu7",
			BeadsID:   "proj-007",
			SpawnMode: "opencode",
			SpawnTime: time.Now().Add(-2 * time.Hour),
		},
	}

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	// Only in_progress issue should be scanned
	if result.Scanned != 1 {
		t.Errorf("expected 1 scanned (only in_progress), got %d", result.Scanned)
	}
}

func TestDetectOrphans_BeadsQueryFails_ReturnsError(t *testing.T) {
	mgr, bc, _, _, _, _ := testManager()
	bc.failOn["list_by_label:orch:agent"] = fmt.Errorf("beads unreachable")

	_, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err == nil {
		t.Error("expected error when beads query fails")
	}
}

func TestDetectOrphans_NoWorkspaceMatch_Orphaned(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()

	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "proj-009", Status: "in_progress", Labels: []string{"orch:agent"}},
	}
	// No matching workspace
	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{}

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	if len(result.Orphans) != 1 {
		t.Fatalf("expected 1 orphan (no workspace), got %d", len(result.Orphans))
	}
	if result.Orphans[0].Reason != "no_workspace" {
		t.Errorf("expected reason 'no_workspace', got %q", result.Orphans[0].Reason)
	}
}

func TestDetectOrphans_MultipleProjects(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()

	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "proj-010", Status: "in_progress", Labels: []string{"orch:agent"}},
		{BeadsID: "proj-011", Status: "in_progress", Labels: []string{"orch:agent"}},
	}

	// Workspace in first project
	wm.workspaces["/tmp/proj1"] = []WorkspaceInfo{
		{
			Name:      "og-feat-a-27feb-aaa1",
			Path:      "/tmp/proj1/.orch/workspace/og-feat-a-27feb-aaa1",
			BeadsID:   "proj-010",
			SpawnMode: "opencode",
			SpawnTime: time.Now().Add(-2 * time.Hour),
		},
	}
	// Workspace in second project
	wm.workspaces["/tmp/proj2"] = []WorkspaceInfo{
		{
			Name:      "og-feat-b-27feb-bbb2",
			Path:      "/tmp/proj2/.orch/workspace/og-feat-b-27feb-bbb2",
			BeadsID:   "proj-011",
			SpawnMode: "opencode",
			SpawnTime: time.Now().Add(-2 * time.Hour),
		},
	}

	result, err := mgr.DetectOrphans([]string{"/tmp/proj1", "/tmp/proj2"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}
	if result.Scanned != 2 {
		t.Errorf("expected 2 scanned, got %d", result.Scanned)
	}
	if len(result.Orphans) != 2 {
		t.Errorf("expected 2 orphans, got %d", len(result.Orphans))
	}
}

func TestDetectOrphans_LandedArtifacts_DetectedAndNotRetried(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()

	// Agent is in_progress with orch:agent label
	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "orch-go-crashed1", Status: "in_progress", Labels: []string{"orch:agent", "triage:ready"}},
	}
	// No Phase: Complete comment
	bc.comments["orch-go-crashed1"] = []string{"Phase: Implementing - Working on feature"}

	// Workspace exists, spawned 2h ago
	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{
		{
			Name:      "og-feat-crashed-task-01mar-abc1",
			Path:      "/tmp/proj/.orch/workspace/og-feat-crashed-task-01mar-abc1",
			BeadsID:   "orch-go-crashed1",
			SessionID: "session-dead",
			SpawnMode: "claude",
			SpawnTime: time.Now().Add(-2 * time.Hour),
		},
	}

	// Agent has landed artifacts (committed work since baseline)
	wm.landedArtifacts["/tmp/proj/.orch/workspace/og-feat-crashed-task-01mar-abc1"] = true

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}

	if len(result.Orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(result.Orphans))
	}

	orphan := result.Orphans[0]
	if !orphan.HasLandedArtifacts {
		t.Error("expected HasLandedArtifacts=true for crashed agent with committed work")
	}
	if orphan.ShouldRetry {
		t.Error("orphan with landed artifacts should NOT be retried (needs review instead)")
	}
	if orphan.LastPhase != "Implementing" {
		t.Errorf("expected last phase 'Implementing', got %q", orphan.LastPhase)
	}
}

func TestDetectOrphans_NoLandedArtifacts_ShouldRetry(t *testing.T) {
	mgr, bc, _, _, _, wm := testManager()

	// Agent is in_progress, has triage:ready (eligible for respawn)
	bc.trackedIssues = []TrackedIssue{
		{BeadsID: "orch-go-empty1", Status: "in_progress", Labels: []string{"orch:agent", "triage:ready"}},
	}
	bc.comments["orch-go-empty1"] = []string{"Phase: Planning - Starting work"}

	wm.workspaces["/tmp/proj"] = []WorkspaceInfo{
		{
			Name:      "og-feat-empty-task-01mar-def2",
			Path:      "/tmp/proj/.orch/workspace/og-feat-empty-task-01mar-def2",
			BeadsID:   "orch-go-empty1",
			SpawnMode: "claude",
			SpawnTime: time.Now().Add(-2 * time.Hour),
		},
	}

	// No landed artifacts
	wm.landedArtifacts["/tmp/proj/.orch/workspace/og-feat-empty-task-01mar-def2"] = false

	result, err := mgr.DetectOrphans([]string{"/tmp/proj"}, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectOrphans() returned error: %v", err)
	}

	if len(result.Orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(result.Orphans))
	}

	orphan := result.Orphans[0]
	if orphan.HasLandedArtifacts {
		t.Error("expected HasLandedArtifacts=false for agent with no committed work")
	}
	if !orphan.ShouldRetry {
		t.Error("orphan without landed artifacts and with triage:ready should be retried")
	}
}

func TestFlagLandedArtifacts_AddsLabelAndLogsEvent(t *testing.T) {
	mgr, bc, _, _, el, _ := testManager()

	agentRef := AgentRef{
		BeadsID:       "orch-go-crashed1",
		WorkspaceName: "og-feat-crashed-task-01mar-abc1",
	}

	err := mgr.FlagLandedArtifacts(agentRef)
	if err != nil {
		t.Fatalf("FlagLandedArtifacts() returned error: %v", err)
	}

	// Verify label was added
	// Note: mockBeadsClient.AddLabel doesn't track additions, but it would return error on failure
	_ = bc // label addition succeeded (no error)

	// Verify event was logged
	if len(el.events) != 1 {
		t.Fatalf("expected 1 event logged, got %d", len(el.events))
	}
	if el.events[0]["_type"] != "agent.crashed_with_artifacts" {
		t.Errorf("expected event type 'agent.crashed_with_artifacts', got %q", el.events[0]["_type"])
	}
	if el.events[0]["beads_id"] != "orch-go-crashed1" {
		t.Errorf("expected beads_id 'orch-go-crashed1', got %q", el.events[0]["beads_id"])
	}
}
