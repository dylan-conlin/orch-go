// Package daemon provides autonomous overnight processing capabilities.
// This file contains stuck agent recovery functionality.
package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/execution"
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
	openIssues, err := verify.ListOpenIssues("")
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
	commentMap := verify.GetCommentsBatch(inProgressIDs, nil)

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
		client := execution.NewOpenCodeAdapter(serverURL)
		ctx := context.Background()
		allSessions, err := client.ListSessions(ctx, projectDir)
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
	client := execution.NewOpenCodeAdapter(serverURL)
	if err := client.SendMessageAsync(context.Background(), execution.SessionHandle(sessionID), prompt, ""); err != nil {
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
