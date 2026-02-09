// Package beads provides a Go RPC client for the beads daemon.
// It connects to the beads daemon via Unix socket at .beads/bd.sock
// and provides operations for issue management.
package beads

import (
	"encoding/json"
	"strings"
)

// RPC operation constants matching beads internal/rpc/protocol.go
const (
	OpPing        = "ping"
	OpHealth      = "health"
	OpStatus      = "status"
	OpCreate      = "create"
	OpUpdate      = "update"
	OpClose       = "close"
	OpDelete      = "delete"
	OpList        = "list"
	OpCount       = "count"
	OpShow        = "show"
	OpReady       = "ready"
	OpStale       = "stale"
	OpStats       = "stats"
	OpDepAdd      = "dep_add"
	OpDepRemove   = "dep_remove"
	OpLabelAdd    = "label_add"
	OpLabelRemove = "label_remove"
	OpCommentList = "comment_list"
	OpCommentAdd  = "comment_add"
	OpResolveID   = "resolve_id"
	OpShutdown    = "shutdown"
)

// Request represents an RPC request to the beads daemon.
type Request struct {
	Operation     string          `json:"operation"`
	Args          json.RawMessage `json:"args"`
	Cwd           string          `json:"cwd,omitempty"`
	ClientVersion string          `json:"client_version,omitempty"`
	ExpectedDB    string          `json:"expected_db,omitempty"`
}

// Response represents an RPC response from the beads daemon.
type Response struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// HealthResponse represents the daemon health check response.
type HealthResponse struct {
	Status        string  `json:"status"`
	Version       string  `json:"version"`
	ClientVersion string  `json:"client_version,omitempty"`
	Compatible    bool    `json:"compatible"`
	Uptime        float64 `json:"uptime_seconds"`
	Error         string  `json:"error,omitempty"`
}

// CreateArgs represents arguments for creating an issue.
type CreateArgs struct {
	ID           string   `json:"id,omitempty"`
	Parent       string   `json:"parent,omitempty"`
	CausedBy     string   `json:"caused_by,omitempty"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	IssueType    string   `json:"issue_type"`
	Priority     int      `json:"priority"`
	Labels       []string `json:"labels,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	// Force bypasses duplicate detection - creates issue even if one with same title exists.
	Force bool `json:"force,omitempty"`
}

// CreateResult contains the result of a Create operation.
// If a duplicate was detected and Force was false, Existing will be set
// and Created will be false.
type CreateResult struct {
	Issue    *Issue
	Created  bool   // True if a new issue was created, false if existing returned
	Existing bool   // True if an existing issue was returned due to duplicate detection
	Message  string // Human-readable message about the result
}

// CloseArgs represents arguments for closing an issue.
type CloseArgs struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
	// Force bypasses Phase: Complete checks on the daemon side.
	Force bool `json:"force,omitempty"`
}

// ListArgs represents arguments for listing issues.
type ListArgs struct {
	Query     string   `json:"query,omitempty"`
	Status    string   `json:"status,omitempty"`
	Priority  *int     `json:"priority,omitempty"`
	IssueType string   `json:"issue_type,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	LabelsAny []string `json:"labels_any,omitempty"`
	IDs       []string `json:"ids,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	// Title filters by title text (case-insensitive substring match).
	Title string `json:"title,omitempty"`
	// Parent filters by parent issue ID (shows children of specified issue).
	// Used for listing children of an epic.
	Parent string `json:"parent,omitempty"`
	// ClosedAfter filters issues closed after the given date/time (RFC3339 or YYYY-MM-DD).
	// Used for finding recently closed issues.
	ClosedAfter string `json:"closed_after,omitempty"`
}

// ShowArgs represents arguments for showing an issue.
type ShowArgs struct {
	ID string `json:"id"`
}

// ReadyArgs represents arguments for ready issues query.
type ReadyArgs struct {
	Assignee   string   `json:"assignee,omitempty"`
	Unassigned bool     `json:"unassigned,omitempty"`
	Priority   *int     `json:"priority,omitempty"`
	Type       string   `json:"type,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	SortPolicy string   `json:"sort_policy,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	LabelsAny  []string `json:"labels_any,omitempty"`
}

// CommentListArgs represents arguments for listing comments.
type CommentListArgs struct {
	ID string `json:"id"`
}

// CommentAddArgs represents arguments for adding a comment.
type CommentAddArgs struct {
	ID     string `json:"id"`
	Author string `json:"author"`
	Text   string `json:"text"`
}

// Issue represents a beads issue.
// Note: Dependencies field uses json.RawMessage because bd show returns
// full Issue objects with dependency_type field for epic children,
// while RPC may return just string IDs. We don't need to parse dependencies
// for orch-go's use cases - just being able to unmarshal the response is enough.
type Issue struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Description  string          `json:"description,omitempty"`
	Status       string          `json:"status"`
	Priority     int             `json:"priority"`
	IssueType    string          `json:"issue_type"`
	CausedBy     string          `json:"caused_by,omitempty"`
	Labels       []string        `json:"labels,omitempty"`
	Dependencies json.RawMessage `json:"dependencies,omitempty"`
	CreatedAt    string          `json:"created_at,omitempty"`
	UpdatedAt    string          `json:"updated_at,omitempty"`
	ClosedAt     string          `json:"closed_at,omitempty"`
	CloseReason  string          `json:"close_reason,omitempty"`
}

// Dependency represents a dependency relationship returned by bd show.
// When an issue has dependencies, bd show returns full Issue objects
// with an additional dependency_type field.
type Dependency struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	IssueType      string `json:"issue_type,omitempty"` // e.g., "question", "task", "bug"
	DependencyType string `json:"dependency_type"`      // e.g., "blocks"
}

// ParseDependencies parses the raw dependencies JSON into a slice of Dependency objects.
// Returns nil if there are no dependencies or if parsing fails.
func (i *Issue) ParseDependencies() []Dependency {
	if len(i.Dependencies) == 0 {
		return nil
	}

	var deps []Dependency
	if err := json.Unmarshal(i.Dependencies, &deps); err != nil {
		return nil
	}
	return deps
}

// BlockingDependency represents a dependency that is blocking this issue.
type BlockingDependency struct {
	ID     string
	Title  string
	Status string
}

// isInferredParentChild detects parent-child relationships when dependency_type is missing.
// This is a workaround for a bug in beads where bd show --json doesn't return dependency_type
// in JSONL-only mode (non-SQLite). The heuristic checks if:
//  1. The issue ID follows child pattern (e.g., "project-123.1" is child of "project-123")
//  2. The dependency ID matches the parent pattern
//
// Example: issue "specs-platform-10.1" with dependency on "specs-platform-10" is parent-child.
func isInferredParentChild(issueID, depID string) bool {
	// Check if issueID is a child of depID
	// Child IDs have format: parentID.N (e.g., "bd-a3f8.1", "bd-a3f8.2")
	if strings.HasPrefix(issueID, depID+".") {
		return true
	}
	return false
}

// GetBlockingDependencies returns a list of dependencies that are blocking this issue.
// Blocking behavior depends on dependency type and issue type:
//   - "blocks": blocks if not closed (open or in_progress)
//   - "parent-child": NEVER blocks (children are independently workable)
//   - Questions: unblock when status is "answered" OR "closed"
//
// The parent-child distinction is critical for epics: an epic closes when its children
// complete, so children must be spawnable while the parent epic is open. If children
// were blocked by their parent, work could never begin (circular dependency).
//
// Questions use a different lifecycle: Open → Investigating → Answered → Closed.
// Dependencies unblock when a question reaches "answered" status, not just "closed",
// because the answer is the gate - closure is just administrative cleanup.
func (i *Issue) GetBlockingDependencies() []BlockingDependency {
	deps := i.ParseDependencies()
	if deps == nil {
		return nil
	}

	var blocking []BlockingDependency
	for _, dep := range deps {
		isBlocking := false

		// Determine dependency type, using inference as fallback
		depType := dep.DependencyType
		if depType == "" && isInferredParentChild(i.ID, dep.ID) {
			depType = "parent-child"
		}

		switch depType {
		case "parent-child":
			// Parent-child: NEVER blocks - children are independently spawnable
			// Epic closes when children complete, so children can't wait for parent
			isBlocking = false
		default:
			// "blocks" and other types: blocks unless resolved
			// Questions unblock when "answered" or "closed"
			// Other types unblock only when "closed"
			if dep.IssueType == "question" {
				// Questions: unblock when answered or closed
				isBlocking = dep.Status != "closed" && dep.Status != "answered"
			} else {
				// Regular issues: unblock only when closed
				isBlocking = dep.Status != "closed"
			}
		}

		if isBlocking {
			blocking = append(blocking, BlockingDependency{
				ID:     dep.ID,
				Title:  dep.Title,
				Status: dep.Status,
			})
		}
	}
	return blocking
}

// Comment represents a comment on a beads issue.
type Comment struct {
	ID        int    `json:"id"`
	IssueID   string `json:"issue_id,omitempty"`
	Author    string `json:"author"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

// StatsSummary represents the summary section of beads statistics.
type StatsSummary struct {
	TotalIssues      int     `json:"total_issues"`
	OpenIssues       int     `json:"open_issues"`
	InProgressIssues int     `json:"in_progress_issues"`
	ClosedIssues     int     `json:"closed_issues"`
	BlockedIssues    int     `json:"blocked_issues"`
	DeferredIssues   int     `json:"deferred_issues"`
	ReadyIssues      int     `json:"ready_issues"`
	TombstoneIssues  int     `json:"tombstone_issues"`
	PinnedIssues     int     `json:"pinned_issues"`
	AvgLeadTimeHours float64 `json:"average_lead_time_hours"`
}

// StatsRecentActivity represents recent activity in beads.
type StatsRecentActivity struct {
	HoursTracked   int `json:"hours_tracked"`
	CommitCount    int `json:"commit_count"`
	IssuesCreated  int `json:"issues_created"`
	IssuesClosed   int `json:"issues_closed"`
	IssuesUpdated  int `json:"issues_updated"`
	IssuesReopened int `json:"issues_reopened"`
	TotalChanges   int `json:"total_changes"`
}

// Stats represents beads statistics.
type Stats struct {
	Summary        StatsSummary        `json:"summary"`
	RecentActivity StatsRecentActivity `json:"recent_activity,omitempty"`
}

// UpdateArgs represents arguments for updating an issue.
type UpdateArgs struct {
	ID                 string   `json:"id"`
	Title              *string  `json:"title,omitempty"`
	Description        *string  `json:"description,omitempty"`
	Status             *string  `json:"status,omitempty"`
	Priority           *int     `json:"priority,omitempty"`
	Design             *string  `json:"design,omitempty"`
	AcceptanceCriteria *string  `json:"acceptance_criteria,omitempty"`
	Notes              *string  `json:"notes,omitempty"`
	Assignee           *string  `json:"assignee,omitempty"`
	ExternalRef        *string  `json:"external_ref,omitempty"`
	EstimatedMinutes   *int     `json:"estimated_minutes,omitempty"`
	IssueType          *string  `json:"issue_type,omitempty"`
	AddLabels          []string `json:"add_labels,omitempty"`
	RemoveLabels       []string `json:"remove_labels,omitempty"`
	SetLabels          []string `json:"set_labels,omitempty"`
}

// DeleteArgs represents arguments for deleting issues.
type DeleteArgs struct {
	IDs     []string `json:"ids"`
	Force   bool     `json:"force,omitempty"`
	DryRun  bool     `json:"dry_run,omitempty"`
	Cascade bool     `json:"cascade,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

// StaleArgs represents arguments for the stale command.
type StaleArgs struct {
	Days   int    `json:"days,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// DepAddArgs represents arguments for adding a dependency.
type DepAddArgs struct {
	FromID  string `json:"from_id"`
	ToID    string `json:"to_id"`
	DepType string `json:"dep_type"`
}

// DepRemoveArgs represents arguments for removing a dependency.
type DepRemoveArgs struct {
	FromID  string `json:"from_id"`
	ToID    string `json:"to_id"`
	DepType string `json:"dep_type,omitempty"`
}

// LabelAddArgs represents arguments for adding a label.
type LabelAddArgs struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// LabelRemoveArgs represents arguments for removing a label.
type LabelRemoveArgs struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ResolveIDArgs represents arguments for resolving a partial ID.
type ResolveIDArgs struct {
	ID string `json:"id"`
}

// CountArgs represents arguments for the count operation.
type CountArgs struct {
	Query     string   `json:"query,omitempty"`
	Status    string   `json:"status,omitempty"`
	Priority  *int     `json:"priority,omitempty"`
	IssueType string   `json:"issue_type,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	LabelsAny []string `json:"labels_any,omitempty"`
	GroupBy   string   `json:"group_by,omitempty"`
}

// CountResponse represents the response for a count operation.
type CountResponse struct {
	Count  int            `json:"count,omitempty"`
	Groups map[string]int `json:"groups,omitempty"`
}

// StatusResponse represents the daemon status metadata.
type StatusResponse struct {
	Version             string  `json:"version"`
	WorkspacePath       string  `json:"workspace_path"`
	DatabasePath        string  `json:"database_path"`
	SocketPath          string  `json:"socket_path"`
	PID                 int     `json:"pid"`
	UptimeSeconds       float64 `json:"uptime_seconds"`
	LastActivityTime    string  `json:"last_activity_time"`
	ExclusiveLockActive bool    `json:"exclusive_lock_active"`
	ExclusiveLockHolder string  `json:"exclusive_lock_holder,omitempty"`
	AutoCommit          bool    `json:"auto_commit"`
	AutoPush            bool    `json:"auto_push"`
	LocalMode           bool    `json:"local_mode"`
	SyncInterval        string  `json:"sync_interval"`
	DaemonMode          string  `json:"daemon_mode"`
}
