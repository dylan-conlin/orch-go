// Package opencode provides a client for interacting with OpenCode sessions.
package opencode

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/notify"
)

// CompletionService monitors sessions and handles completion actions.
type CompletionService struct {
	monitor   *Monitor
	notifier  *notify.Notifier
	client    ClientInterface
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
	s := &CompletionService{
		monitor:     NewMonitor(serverURL),
		notifier:    notify.Default(),
		client:      NewClient(serverURL),
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
			// Try to extract beads ID from session title (format: "workspace [beads-id]")
			beadsID = extractBeadsIDFromTitle(session.Title)
		}
	}

	// Send desktop notification
	if err := s.notifier.SessionComplete(sessionID, workspace); err != nil {
		fmt.Printf("Warning: failed to send notification: %v\n", err)
	}

	// NOTE: Automatic registry completion was disabled (2025-12-21)
	// Reason: Monitor's busy→idle detection triggers false positives.
	// Agents go idle during normal operation (loading, thinking, waiting for tools).
	// The first idle transition after spawn (4-6 seconds) incorrectly marks agents complete.
	// Solution: Require explicit `orch complete` command instead of automatic detection.
	// See: .kb/investigations/2025-12-21-inv-agents-being-marked-completed-registry.md

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

// extractBeadsIDFromTitle extracts beads ID from session title.
// Format: "workspace [beads-id]" -> "beads-id"
func extractBeadsIDFromTitle(title string) string {
	// Look for "[...]" at the end of the title
	start := strings.LastIndex(title, "[")
	end := strings.LastIndex(title, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(title[start+1 : end])
}

// updateBeadsPhase adds a "Phase: Complete" comment to the beads issue.
// This is a backup for cases where the agent didn't report completion properly.
// It uses the beads RPC client when available, falling back to the bd CLI.
func (s *CompletionService) updateBeadsPhase(beadsID string) error {
	var comments []beads.Comment
	var err error

	// Try RPC client first for getting comments
	socketPath, socketErr := beads.FindSocketPath("")
	if socketErr == nil {
		client := beads.NewClient(socketPath)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()

			comments, err = client.Comments(beadsID)
			if err != nil {
				// Fall through to CLI fallback on RPC error
				comments, err = beads.FallbackComments(beadsID)
			}
		} else {
			comments, err = beads.FallbackComments(beadsID)
		}
	} else {
		comments, err = beads.FallbackComments(beadsID)
	}

	if err != nil {
		return fmt.Errorf("failed to get comments: %w", err)
	}

	// Check if Phase: Complete is already in comments
	for _, c := range comments {
		if containsPhaseComplete(c.Text) {
			// Already has Phase: Complete, skip
			return nil
		}
	}

	// Add Phase: Complete comment
	comment := "Phase: Complete - Session finished (detected via SSE monitor)"

	// Try RPC client first for adding comment
	if socketErr == nil {
		client := beads.NewClient(socketPath)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			if err := client.AddComment(beadsID, "", comment); err == nil {
				return nil
			}
			// Fall through to CLI fallback on RPC error
		}
	}

	// Fallback to CLI
	return beads.FallbackAddComment(beadsID, comment)
}

// containsPhaseComplete checks if the comments JSON contains "Phase: Complete".
func containsPhaseComplete(commentsJSON string) bool {
	// Simple string check - more robust than parsing JSON
	return containsStr(commentsJSON, "Phase: Complete") ||
		containsStr(commentsJSON, "Phase: complete") ||
		containsStr(commentsJSON, "phase: Complete") ||
		containsStr(commentsJSON, "phase: complete")
}

// containsStr is a simple case-sensitive substring check.
func containsStr(s, substr string) bool {
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
