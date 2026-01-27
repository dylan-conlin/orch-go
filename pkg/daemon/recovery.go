// Package daemon provides autonomous overnight processing capabilities.
// This file contains stuck agent recovery functionality including:
// - Idle agent recovery (RunPeriodicRecovery)
// - Server restart recovery (RunServerRecovery)
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// ActiveAgent represents an in-progress agent for recovery purposes.
type ActiveAgent struct {
	BeadsID   string    // Beads issue ID
	Phase     string    // Current phase from comments (e.g., "Implementing", "Planning")
	UpdatedAt time.Time // When the agent last reported progress (from phase comment timestamp)
	Title     string    // Issue title
}

// GetActiveAgents returns all agents that are currently in_progress.
// It queries beads for in_progress issues and parses their phase from comments.
func GetActiveAgents() ([]ActiveAgent, error) {
	// Get open/in_progress issues
	openIssues, err := verify.ListOpenIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}

	if len(openIssues) == 0 {
		return nil, nil
	}

	// Filter to only in_progress issues (agents currently working)
	var inProgressIDs []string
	inProgressIssues := make(map[string]*verify.Issue)
	for id, issue := range openIssues {
		if strings.EqualFold(issue.Status, "in_progress") {
			inProgressIDs = append(inProgressIDs, id)
			inProgressIssues[id] = issue
		}
	}

	if len(inProgressIDs) == 0 {
		return nil, nil
	}

	// Fetch comments in batch
	commentMap := verify.GetCommentsBatch(inProgressIDs)

	var agents []ActiveAgent
	for id, issue := range inProgressIssues {
		comments := commentMap[id]

		// Parse phase from comments
		phaseStatus := verify.ParsePhaseFromComments(comments)

		agent := ActiveAgent{
			BeadsID: id,
			Title:   issue.Title,
			Phase:   phaseStatus.Phase,
		}

		// Use phase timestamp if available, otherwise use current time
		if phaseStatus.PhaseReportedAt != nil {
			agent.UpdatedAt = *phaseStatus.PhaseReportedAt
		} else {
			// No phase timestamp means agent hasn't reported - use old time to trigger recovery
			agent.UpdatedAt = time.Now().Add(-24 * time.Hour)
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// ResumeAgentByBeadsID attempts to resume a stuck agent by its beads ID.
// It finds the agent's session and sends a continuation prompt.
func ResumeAgentByBeadsID(beadsID string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)

	// Find workspace by beadsID and read session_id
	var sessionID, agentID, workspacePath string
	workspaceBase := filepath.Join(projectDir, ".orch", "workspace")
	if entries, err := os.ReadDir(workspaceBase); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.Contains(entry.Name(), beadsID) {
				workspacePath = filepath.Join(workspaceBase, entry.Name())
				sessionID = spawn.ReadSessionID(workspacePath)
				agentID = entry.Name()
				break
			}
		}
	}

	// If workspace file doesn't have session_id, try to find via OpenCode API
	serverURL := os.Getenv("OPENCODE_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}

	if sessionID == "" {
		client := opencode.NewClient(serverURL)
		allSessions, err := client.ListSessions(projectDir)
		if err == nil {
			for _, s := range allSessions {
				if strings.Contains(s.Title, beadsID) {
					sessionID = s.ID
					break
				}
			}
		}
	}

	if sessionID == "" {
		return fmt.Errorf("no agent found for beads ID: %s (no workspace file or active session)", beadsID)
	}

	if agentID == "" {
		agentID = beadsID
	}

	// Generate resume prompt
	var prompt string
	if workspacePath != "" {
		contextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
		prompt = fmt.Sprintf(
			"You were paused mid-task. Re-read your spawn context from %s and continue your work. "+
				"Report progress via bd comment %s.",
			contextPath,
			beadsID,
		)
	} else {
		prompt = fmt.Sprintf(
			"You were paused mid-task. Continue your work from where you left off. "+
				"Report progress via bd comment %s.",
			beadsID,
		)
	}

	// Send resume message via OpenCode API
	client := opencode.NewClient(serverURL)
	if err := client.SendMessageAsync(sessionID, prompt, ""); err != nil {
		return fmt.Errorf("failed to send resume prompt: %w", err)
	}

	// Log the resume event
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agent.recovered",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id":   beadsID,
			"agent_id":   agentID,
			"session_id": sessionID,
			"project":    projectName,
			"source":     "daemon_recovery",
		},
	}
	if err := logger.Log(event); err != nil {
		// Don't fail the resume just because logging failed
		fmt.Fprintf(os.Stderr, "Warning: failed to log recovery event: %v\n", err)
	}

	return nil
}

// =============================================================================
// Server Restart Recovery
// =============================================================================
//
// Server restart recovery handles the case where the OpenCode server crashes or
// restarts, losing all in-memory sessions. Unlike idle agent recovery which
// detects individual stuck agents, this mechanism:
// 1. Detects when daemon starts (or server recovers)
// 2. Finds disk sessions for in_progress beads issues without matching in-memory sessions
// 3. Resumes orphaned sessions with recovery-specific context
// 4. Uses stabilization delay to ensure server is ready

// OrphanedSession represents a session that was orphaned by server restart.
// The session exists on disk but is no longer in OpenCode's in-memory state.
type OrphanedSession struct {
	BeadsID       string // Beads issue ID associated with this session
	SessionID     string // OpenCode session ID (from disk)
	WorkspacePath string // Path to workspace directory
	AgentID       string // Agent/workspace name
	Phase         string // Last reported phase from beads comments
	ProjectDir    string // Project directory
}

// ServerRecoveryResult contains the result of a server recovery operation.
type ServerRecoveryResult struct {
	ResumedCount  int
	SkippedCount  int
	OrphanedCount int // Number of orphaned sessions found
	Error         error
	Message       string
}

// ServerRecoveryState tracks state for server recovery detection.
// This is used by the daemon to determine when server recovery should run.
type ServerRecoveryState struct {
	mu                   sync.Mutex
	daemonStartTime      time.Time // When the daemon started
	lastRecoveryTime     time.Time // When server recovery last ran
	recoveredSessionsMap map[string]time.Time // Sessions we've already recovered (beadsID -> time)
}

// NewServerRecoveryState creates a new server recovery state tracker.
func NewServerRecoveryState() *ServerRecoveryState {
	return &ServerRecoveryState{
		daemonStartTime:      time.Now(),
		recoveredSessionsMap: make(map[string]time.Time),
	}
}

// ShouldRunServerRecovery determines if server recovery should run.
// Returns true if:
// - Daemon just started (within first poll cycle)
// - Hasn't run recovery yet since daemon start
func (s *ServerRecoveryState) ShouldRunServerRecovery(stabilizationDelay time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only run if we haven't run recovery since daemon started
	if !s.lastRecoveryTime.IsZero() {
		fmt.Printf("[DEBUG] ServerRecoveryState.ShouldRunServerRecovery: returning false - already ran at %v\n",
			s.lastRecoveryTime.Format(time.RFC3339))
		return false
	}

	// Wait for stabilization delay after daemon start
	timeSinceStart := time.Since(s.daemonStartTime)
	result := timeSinceStart >= stabilizationDelay
	fmt.Printf("[DEBUG] ServerRecoveryState.ShouldRunServerRecovery: daemonStartTime=%v, timeSinceStart=%v, stabilizationDelay=%v, result=%v\n",
		s.daemonStartTime.Format(time.RFC3339), timeSinceStart.Round(time.Second), stabilizationDelay, result)
	return result
}

// MarkRecoveryRun records that server recovery has run.
func (s *ServerRecoveryState) MarkRecoveryRun() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRecoveryTime = time.Now()
}

// WasRecentlyRecovered checks if a session was recently recovered.
func (s *ServerRecoveryState) WasRecentlyRecovered(beadsID string, rateLimit time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if lastRecovery, exists := s.recoveredSessionsMap[beadsID]; exists {
		return time.Since(lastRecovery) < rateLimit
	}
	return false
}

// MarkRecovered records that a session was recovered.
func (s *ServerRecoveryState) MarkRecovered(beadsID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recoveredSessionsMap[beadsID] = time.Now()
}

// FindOrphanedSessions finds sessions that were orphaned by server restart.
// It queries:
// 1. Beads for in_progress issues
// 2. Workspaces for session IDs
// 3. Compares against OpenCode's current in-memory sessions
// Returns sessions that exist on disk but aren't in memory.
func FindOrphanedSessions(serverURL string) ([]OrphanedSession, error) {
	fmt.Printf("[DEBUG] FindOrphanedSessions: starting with serverURL=%s\n", serverURL)

	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	fmt.Printf("[DEBUG] FindOrphanedSessions: projectDir=%s\n", projectDir)

	// Get in_progress beads issues
	openIssues, err := verify.ListOpenIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}
	fmt.Printf("[DEBUG] FindOrphanedSessions: found %d open issues\n", len(openIssues))

	// Filter to in_progress issues
	var inProgressIDs []string
	inProgressIssues := make(map[string]*verify.Issue)
	for id, issue := range openIssues {
		if strings.EqualFold(issue.Status, "in_progress") {
			inProgressIDs = append(inProgressIDs, id)
			inProgressIssues[id] = issue
		}
	}
	fmt.Printf("[DEBUG] FindOrphanedSessions: %d issues are in_progress\n", len(inProgressIDs))

	if len(inProgressIDs) == 0 {
		fmt.Printf("[DEBUG] FindOrphanedSessions: no in_progress issues, returning nil\n")
		return nil, nil // No in-progress issues means no orphaned sessions
	}

	// Get beads comments to check phase (skip Phase: Complete)
	commentMap := verify.GetCommentsBatch(inProgressIDs)

	// Get current in-memory sessions from OpenCode
	client := opencode.NewClient(serverURL)
	inMemorySessions, err := client.ListSessions(projectDir)
	if err != nil {
		fmt.Printf("[DEBUG] FindOrphanedSessions: ListSessions error (treating as empty): %v\n", err)
		// Server might not be responding - treat as no in-memory sessions
		inMemorySessions = nil
	}
	fmt.Printf("[DEBUG] FindOrphanedSessions: %d in-memory sessions\n", len(inMemorySessions))

	// Build set of in-memory session IDs for fast lookup
	inMemorySessionIDs := make(map[string]bool)
	for _, s := range inMemorySessions {
		inMemorySessionIDs[s.ID] = true
	}

	// Find workspaces with sessions that aren't in memory
	var orphaned []OrphanedSession
	workspaceBase := filepath.Join(projectDir, ".orch", "workspace")

	for beadsID, issue := range inProgressIssues {
		// Check if this issue has Phase: Complete (skip it)
		comments := commentMap[beadsID]
		phaseStatus := verify.ParsePhaseFromComments(comments)
		if strings.EqualFold(phaseStatus.Phase, "complete") {
			fmt.Printf("[DEBUG] FindOrphanedSessions: skipping %s - Phase: Complete\n", beadsID)
			continue // Skip - agent finished but issue not closed yet
		}

		// Find workspace for this beads ID
		entries, err := os.ReadDir(workspaceBase)
		if err != nil {
			fmt.Printf("[DEBUG] FindOrphanedSessions: skipping %s - cannot read workspace dir: %v\n", beadsID, err)
			continue // No workspace dir
		}

		foundInWorkspace := false
		for _, entry := range entries {
			if entry.IsDir() && strings.Contains(entry.Name(), beadsID) {
				workspacePath := filepath.Join(workspaceBase, entry.Name())
				sessionID := spawn.ReadSessionID(workspacePath)
				if sessionID == "" {
					fmt.Printf("[DEBUG] FindOrphanedSessions: %s - workspace %s has no session_id\n", beadsID, entry.Name())
					continue // No session ID in workspace
				}

				// Check if session is in memory
				if inMemorySessionIDs[sessionID] {
					fmt.Printf("[DEBUG] FindOrphanedSessions: %s - session %s is in memory (not orphaned)\n", beadsID, sessionID)
					foundInWorkspace = true
					break // Session is still active - not orphaned
				}

				// Found an orphaned session
				fmt.Printf("[DEBUG] FindOrphanedSessions: %s - ORPHANED session %s found\n", beadsID, sessionID)
				orphaned = append(orphaned, OrphanedSession{
					BeadsID:       beadsID,
					SessionID:     sessionID,
					WorkspacePath: workspacePath,
					AgentID:       entry.Name(),
					Phase:         phaseStatus.Phase,
					ProjectDir:    projectDir,
				})
				foundInWorkspace = true
				break // Found workspace for this beadsID
			}
		}

		// If no workspace found with session, try to find session via disk query
		// This handles cases where session exists on disk but workspace was cleaned
		if !foundInWorkspace {
			fmt.Printf("[DEBUG] FindOrphanedSessions: %s - no workspace match, trying disk sessions\n", beadsID)
			// Try listing disk sessions to find one matching this beads ID
			diskSessions, err := client.ListDiskSessions(projectDir)
			if err != nil {
				fmt.Printf("[DEBUG] FindOrphanedSessions: %s - ListDiskSessions error: %v\n", beadsID, err)
			} else {
				fmt.Printf("[DEBUG] FindOrphanedSessions: %s - checking %d disk sessions\n", beadsID, len(diskSessions))
				for _, ds := range diskSessions {
					if strings.Contains(ds.Title, beadsID) {
						// Found disk session - check if in memory
						if !inMemorySessionIDs[ds.ID] {
							fmt.Printf("[DEBUG] FindOrphanedSessions: %s - ORPHANED disk session %s found\n", beadsID, ds.ID)
							orphaned = append(orphaned, OrphanedSession{
								BeadsID:    beadsID,
								SessionID:  ds.ID,
								AgentID:    beadsID, // Use beadsID as fallback
								Phase:      phaseStatus.Phase,
								ProjectDir: projectDir,
							})
						} else {
							fmt.Printf("[DEBUG] FindOrphanedSessions: %s - disk session %s is in memory (not orphaned)\n", beadsID, ds.ID)
						}
						break
					}
				}
			}
		}

		_ = issue // Suppress unused variable warning
	}

	fmt.Printf("[DEBUG] FindOrphanedSessions: returning %d orphaned sessions\n", len(orphaned))
	return orphaned, nil
}

// ResumeOrphanedAgent resumes an orphaned agent with recovery-specific context.
// Unlike ResumeAgentByBeadsID, this includes context about the server restart.
func ResumeOrphanedAgent(orphan OrphanedSession, serverURL string) error {
	projectName := filepath.Base(orphan.ProjectDir)

	// Generate recovery-specific resume prompt
	var prompt string
	if orphan.WorkspacePath != "" {
		contextPath := filepath.Join(orphan.WorkspacePath, "SPAWN_CONTEXT.md")
		prompt = fmt.Sprintf(
			"⚠️ SERVER RECOVERY: The OpenCode server was restarted and your session was interrupted. "+
				"Your previous in-memory state may be lost, but your workspace and conversation history are preserved.\n\n"+
				"**Recovery steps:**\n"+
				"1. Re-read your spawn context from %s\n"+
				"2. Review your last messages to understand where you stopped\n"+
				"3. Validate any in-progress work before continuing\n"+
				"4. Report progress via bd comment %s\n\n"+
				"Continue your work from where you left off.",
			contextPath,
			orphan.BeadsID,
		)
	} else {
		prompt = fmt.Sprintf(
			"⚠️ SERVER RECOVERY: The OpenCode server was restarted and your session was interrupted. "+
				"Your previous in-memory state may be lost.\n\n"+
				"**Recovery steps:**\n"+
				"1. Review your last messages to understand where you stopped\n"+
				"2. Validate any in-progress work before continuing\n"+
				"3. Report progress via bd comment %s\n\n"+
				"Continue your work from where you left off.",
			orphan.BeadsID,
		)
	}

	// Send resume message via OpenCode API
	client := opencode.NewClient(serverURL)
	if err := client.SendMessageAsync(orphan.SessionID, prompt, ""); err != nil {
		return fmt.Errorf("failed to send recovery prompt: %w", err)
	}

	// Log the recovery event with server_recovery source
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "agent.recovered",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id":   orphan.BeadsID,
			"agent_id":   orphan.AgentID,
			"session_id": orphan.SessionID,
			"project":    projectName,
			"phase":      orphan.Phase,
			"source":     "server_recovery",
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log server recovery event: %v\n", err)
	}

	return nil
}
