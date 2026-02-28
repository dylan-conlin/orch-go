package agent

import (
	"fmt"
	"testing"
	"time"
)

// --- Mock implementations of the interfaces ---

type mockBeadsClient struct {
	labelsRemoved  map[string][]string
	statusUpdated  map[string]string
	assigneeCleared []string
	issuesClosed   map[string]string
	failOn         map[string]error // key: "operation:beadsID"
}

func newMockBeadsClient() *mockBeadsClient {
	return &mockBeadsClient{
		labelsRemoved:  make(map[string][]string),
		statusUpdated:  make(map[string]string),
		issuesClosed:   make(map[string]string),
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
	return nil, nil
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
	archived       []string
	failureReports map[string]string // path → reason
	existing       map[string]bool
	failOn         map[string]error
}

func newMockWorkspaceManager() *mockWorkspaceManager {
	return &mockWorkspaceManager{
		failureReports: make(map[string]string),
		existing:       make(map[string]bool),
		failOn:         make(map[string]error),
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
