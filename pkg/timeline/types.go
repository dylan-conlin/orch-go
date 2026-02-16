// Package timeline provides session timeline extraction and grouping.
package timeline

import (
	"time"
)

// ActionType represents the type of action in the timeline.
type ActionType string

const (
	ActionTypeIssueCreated   ActionType = "issue_created"
	ActionTypeIssueCompleted ActionType = "issue_completed"
	ActionTypeIssueClosed    ActionType = "issue_closed"
	ActionTypeIssueReleased  ActionType = "issue_released"
	ActionTypeAgentSpawned   ActionType = "agent_spawned"
	ActionTypeAgentCompleted ActionType = "agent_completed"
	ActionTypeDecisionMade   ActionType = "decision_made"
	ActionTypeQuickDecision  ActionType = "quick_decision"
	ActionTypeSessionStarted ActionType = "session_started"
	ActionTypeSessionEnded   ActionType = "session_ended"
	ActionTypeSessionLabeled ActionType = "session_labeled"
)

// TimelineAction represents a single action in the timeline.
type TimelineAction struct {
	Type       ActionType             `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	SessionID  string                 `json:"session_id"`
	Title      string                 `json:"title"`
	BeadsID    string                 `json:"beads_id,omitempty"`
	Path       string                 `json:"path,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	ArtifactID string                 `json:"artifact_id,omitempty"` // For split-and-grow: issue → artifact transformation
}

// SessionGroup represents a group of actions for a single session.
type SessionGroup struct {
	SessionID   string           `json:"session_id"`
	Label       string           `json:"label,omitempty"` // Human-readable session name
	StartTime   time.Time        `json:"start_time"`      // First action timestamp
	EndTime     time.Time        `json:"end_time"`        // Last action timestamp
	Actions     []TimelineAction `json:"actions"`         // Chronological actions
	ActionCount int              `json:"action_count"`    // Total actions
}

// Timeline represents the full timeline grouped by session.
type Timeline struct {
	Sessions []SessionGroup `json:"sessions"`
	Total    int            `json:"total"` // Total actions across all sessions
}
