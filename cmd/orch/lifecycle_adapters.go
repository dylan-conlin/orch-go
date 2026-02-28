// Package main provides adapter implementations that bridge real packages
// (beads, opencode, tmux, events, spawn) to the pkg/agent lifecycle interfaces.
// These adapters enable LifecycleManager to coordinate real infrastructure
// without pkg/agent depending on those packages directly.
package main

import (
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// Compile-time interface compliance checks.
var (
	_ agent.BeadsClient      = (*beadsAdapter)(nil)
	_ agent.OpenCodeClient   = (*openCodeAdapter)(nil)
	_ agent.TmuxClient       = (*tmuxAdapter)(nil)
	_ agent.EventLogger      = (*eventLoggerAdapter)(nil)
	_ agent.WorkspaceManager = (*workspaceAdapter)(nil)
)

// --- BeadsClient adapter ---

// beadsAdapter wraps beads.CLIClient to implement agent.BeadsClient.
type beadsAdapter struct {
	client *beads.CLIClient
}

func newBeadsAdapter(workDir string) *beadsAdapter {
	var opts []beads.CLIOption
	if workDir != "" {
		opts = append(opts, beads.WithWorkDir(workDir))
	}
	return &beadsAdapter{client: beads.NewCLIClient(opts...)}
}

func (a *beadsAdapter) AddLabel(beadsID, label string) error {
	return a.client.AddLabel(beadsID, label)
}

func (a *beadsAdapter) RemoveLabel(beadsID, label string) error {
	return a.client.RemoveLabel(beadsID, label)
}

func (a *beadsAdapter) UpdateStatus(beadsID, status string) error {
	_, err := a.client.Update(&beads.UpdateArgs{ID: beadsID, Status: &status})
	return err
}

func (a *beadsAdapter) SetAssignee(beadsID, assignee string) error {
	return beads.FallbackUpdateAssignee(beadsID, assignee)
}

func (a *beadsAdapter) ClearAssignee(beadsID string) error {
	return beads.FallbackUpdateAssignee(beadsID, "")
}

func (a *beadsAdapter) CloseIssue(beadsID, reason string) error {
	return a.client.CloseIssue(beadsID, reason)
}

func (a *beadsAdapter) GetComments(beadsID string) ([]string, error) {
	comments, err := a.client.Comments(beadsID)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(comments))
	for i, c := range comments {
		result[i] = c.Text
	}
	return result, nil
}

// --- OpenCodeClient adapter ---

// openCodeAdapter wraps opencode.Client to implement agent.OpenCodeClient.
type openCodeAdapter struct {
	client *opencode.Client
}

func newOpenCodeAdapter(serverURL string) *openCodeAdapter {
	return &openCodeAdapter{client: opencode.NewClient(serverURL)}
}

func (a *openCodeAdapter) SessionExists(sessionID string) (bool, error) {
	_, err := a.client.GetSession(sessionID)
	if err != nil {
		// GetSession returns error for both "not found" and actual API errors.
		// Treat all errors as "not existing" since the lifecycle manager
		// skips session deletion when SessionExists returns false.
		return false, nil
	}
	return true, nil
}

func (a *openCodeAdapter) DeleteSession(sessionID string) error {
	return a.client.DeleteSession(sessionID)
}

func (a *openCodeAdapter) ExportActivity(sessionID, outputPath string) error {
	transcript, err := a.client.ExportSessionTranscript(sessionID)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(transcript), 0644)
}

// --- TmuxClient adapter ---

// tmuxAdapter bridges pkg/tmux functions to agent.TmuxClient.
// Uses workspace name to find and kill windows across all tmux sessions.
type tmuxAdapter struct{}

func (a *tmuxAdapter) WindowExists(name string) (bool, error) {
	window, _, err := tmux.FindWindowByWorkspaceNameAllSessions(name)
	if err != nil {
		return false, err
	}
	return window != nil, nil
}

func (a *tmuxAdapter) KillWindow(name string) error {
	window, _, err := tmux.FindWindowByWorkspaceNameAllSessions(name)
	if err != nil {
		return err
	}
	if window == nil {
		return nil
	}
	return tmux.KillWindowByID(window.ID)
}

// --- EventLogger adapter ---

// eventLoggerAdapter wraps events.Logger to implement agent.EventLogger.
type eventLoggerAdapter struct {
	logger *events.Logger
}

func newEventLoggerAdapter() *eventLoggerAdapter {
	return &eventLoggerAdapter{logger: events.NewLogger(events.DefaultLogPath())}
}

func (a *eventLoggerAdapter) Log(eventType string, data map[string]interface{}) error {
	return a.logger.Log(events.Event{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	})
}

// --- WorkspaceManager adapter ---

// workspaceAdapter bridges filesystem + spawn operations to agent.WorkspaceManager.
type workspaceAdapter struct {
	projectDir string
	agentName  string
	beadsID    string
}

func (a *workspaceAdapter) Archive(workspacePath string) error {
	_, err := archiveWorkspace(workspacePath, a.projectDir)
	return err
}

func (a *workspaceAdapter) WriteFailureReport(workspacePath, reason string) error {
	_ = spawn.EnsureFailureReportTemplate(a.projectDir)
	_, err := spawn.WriteFailureReport(workspacePath, a.agentName, a.beadsID, reason, "")
	return err
}

func (a *workspaceAdapter) Exists(workspacePath string) bool {
	_, err := os.Stat(workspacePath)
	return err == nil
}

func (a *workspaceAdapter) WriteSessionID(workspacePath, sessionID string) error {
	return spawn.WriteSessionID(workspacePath, sessionID)
}

func (a *workspaceAdapter) Remove(workspacePath string) error {
	return os.RemoveAll(workspacePath)
}

// --- Factory ---

// buildLifecycleManager constructs a LifecycleManager with real adapters.
func buildLifecycleManager(projectDir, serverURL, agentName, beadsID string) agent.LifecycleManager {
	return agent.NewLifecycleManager(
		newBeadsAdapter(projectDir),
		newOpenCodeAdapter(serverURL),
		&tmuxAdapter{},
		newEventLoggerAdapter(),
		&workspaceAdapter{projectDir: projectDir, agentName: agentName, beadsID: beadsID},
	)
}
