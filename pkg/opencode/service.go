// Package opencode provides a client for interacting with OpenCode sessions.
package opencode

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/notify"
	"github.com/dylan-conlin/orch-go/pkg/registry"
)

// CompletionService monitors sessions and handles completion actions.
type CompletionService struct {
	monitor   *Monitor
	notifier  *notify.Notifier
	client    *Client
	registry  *registry.Registry
	logger    *events.Logger
	serverURL string

	// Session to workspace/beads mapping
	// This is populated when we detect session activity
	sessionInfo map[string]*SessionCompletionInfo
	mu          sync.RWMutex
}

// SessionCompletionInfo tracks metadata needed for completion handling.
type SessionCompletionInfo struct {
	SessionID string
	Workspace string
	BeadsID   string
	Directory string
}

// NewCompletionService creates a new completion service.
func NewCompletionService(serverURL string) (*CompletionService, error) {
	reg, err := registry.New("")
	if err != nil {
		return nil, fmt.Errorf("failed to open registry: %w", err)
	}

	s := &CompletionService{
		monitor:     NewMonitor(serverURL),
		notifier:    notify.Default(),
		client:      NewClient(serverURL),
		registry:    reg,
		logger:      events.NewLogger(events.DefaultLogPath()),
		serverURL:   serverURL,
		sessionInfo: make(map[string]*SessionCompletionInfo),
	}

	// Register completion handler
	s.monitor.OnCompletion(s.handleCompletion)

	return s, nil
}

// Start begins monitoring for session completions.
func (s *CompletionService) Start() {
	s.monitor.Start()
}

// Stop stops the completion service.
func (s *CompletionService) Stop() {
	s.monitor.Stop()
}

// RegisterSession registers a session with its associated metadata.
// This should be called after spawning a session to enable proper completion handling.
func (s *CompletionService) RegisterSession(info *SessionCompletionInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionInfo[info.SessionID] = info
}

// handleCompletion is called when a session completes.
func (s *CompletionService) handleCompletion(sessionID string) {
	// Try to get session info from our mapping first
	s.mu.RLock()
	info, hasInfo := s.sessionInfo[sessionID]
	s.mu.RUnlock()

	var workspace string
	var beadsID string

	if hasInfo {
		workspace = info.Workspace
		beadsID = info.BeadsID
	} else {
		// Try to get session info from the OpenCode API
		session, err := s.client.GetSession(sessionID)
		if err == nil && session != nil {
			workspace = session.Title

			// Try to find matching agent in registry by workspace name
			agent := s.findAgentByWorkspace(workspace)
			if agent != nil {
				beadsID = agent.BeadsID
			}
		}
	}

	// Send desktop notification
	if err := s.notifier.SessionComplete(sessionID, workspace); err != nil {
		fmt.Printf("Warning: failed to send notification: %v\n", err)
	}

	// DISABLED: Automatic registry completion (2025-12-21)
	// Reason: Monitor's busy→idle detection triggers false positives.
	// Agents go idle during normal operation (loading, thinking, waiting for tools).
	// The first idle transition after spawn (4-6 seconds) incorrectly marks agents complete.
	// Solution: Require explicit `orch complete` command instead of automatic detection.
	// See: .kb/investigations/2025-12-21-inv-agents-being-marked-completed-registry.md
	//
	// if beadsID != "" {
	// 	// Mark agent as completed in registry
	// 	if s.registry.Complete(beadsID) || s.registry.Complete(workspace) {
	// 		if err := s.registry.Save(); err != nil {
	// 			fmt.Printf("Warning: failed to save registry: %v\n", err)
	// 		}
	// 	}
	//
	// 	// Update beads status
	// 	if err := s.updateBeadsPhase(beadsID); err != nil {
	// 		fmt.Printf("Warning: failed to update beads phase: %v\n", err)
	// 	}
	// }

	// Log the completion event
	eventData := map[string]interface{}{
		"session_id": sessionID,
	}
	if workspace != "" {
		eventData["workspace"] = workspace
	}
	if beadsID != "" {
		eventData["beads_id"] = beadsID
	}

	event := events.Event{
		Type:      "session.completed",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := s.logger.Log(event); err != nil {
		fmt.Printf("Warning: failed to log completion: %v\n", err)
	}

	fmt.Printf("Session completed: %s (workspace: %s, beads: %s)\n", sessionID, workspace, beadsID)
}

// findAgentByWorkspace looks up an agent in the registry by workspace name.
func (s *CompletionService) findAgentByWorkspace(workspace string) *registry.Agent {
	// Try exact match first
	agent := s.registry.Find(workspace)
	if agent != nil {
		return agent
	}

	// Try to find by iterating through active agents
	activeAgents := s.registry.ListActive()
	for _, a := range activeAgents {
		if a.ID == workspace {
			return a
		}
	}

	return nil
}

// updateBeadsPhase adds a "Phase: Complete" comment to the beads issue.
// This is a backup for cases where the agent didn't report completion properly.
func (s *CompletionService) updateBeadsPhase(beadsID string) error {
	// Check if Phase: Complete is already reported
	cmd := exec.Command("bd", "comments", beadsID, "--json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get comments: %w", err)
	}

	// Check if Phase: Complete is already in comments
	if containsPhaseComplete(string(output)) {
		// Already has Phase: Complete, skip
		return nil
	}

	// Add Phase: Complete comment
	comment := "Phase: Complete - Session finished (detected via SSE monitor)"
	cmd = exec.Command("bd", "comment", beadsID, comment)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add completion comment: %w", err)
	}

	return nil
}

// containsPhaseComplete checks if the comments JSON contains "Phase: Complete".
func containsPhaseComplete(commentsJSON string) bool {
	// Simple string check - more robust than parsing JSON
	return contains(commentsJSON, "Phase: Complete") ||
		contains(commentsJSON, "Phase: complete") ||
		contains(commentsJSON, "phase: Complete") ||
		contains(commentsJSON, "phase: complete")
}

// contains is a simple case-sensitive substring check.
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0
}

// findSubstring returns the index of substr in s, or -1 if not found.
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// MonitorCmd creates a completion service and runs it until interrupted.
// This is the entry point for the "orch-go monitor" command.
func MonitorCmd(serverURL string) error {
	service, err := NewCompletionService(serverURL)
	if err != nil {
		return err
	}

	fmt.Printf("Starting SSE monitor at %s/event...\n", serverURL)
	fmt.Println("Press Ctrl+C to stop")

	service.Start()

	// Block forever (caller should handle signals)
	select {}
}
