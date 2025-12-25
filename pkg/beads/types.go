// Package beads provides a Go RPC client for the beads daemon.
// It connects to the beads daemon via Unix socket at .beads/bd.sock
// and provides operations for issue management.
package beads

import "encoding/json"

// RPC operation constants matching beads internal/rpc/protocol.go
const (
	OpPing        = "ping"
	OpHealth      = "health"
	OpCreate      = "create"
	OpClose       = "close"
	OpList        = "list"
	OpShow        = "show"
	OpReady       = "ready"
	OpStats       = "stats"
	OpCommentList = "comment_list"
	OpCommentAdd  = "comment_add"
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
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	IssueType    string   `json:"issue_type"`
	Priority     int      `json:"priority"`
	Labels       []string `json:"labels,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// CloseArgs represents arguments for closing an issue.
type CloseArgs struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
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
type Issue struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Status       string   `json:"status"`
	Priority     int      `json:"priority"`
	IssueType    string   `json:"issue_type"`
	Labels       []string `json:"labels,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
	ClosedAt     string   `json:"closed_at,omitempty"`
	CloseReason  string   `json:"close_reason,omitempty"`
}

// Comment represents a comment on a beads issue.
type Comment struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

// Stats represents beads statistics.
type Stats struct {
	Total    int            `json:"total"`
	Open     int            `json:"open"`
	Closed   int            `json:"closed"`
	Blocked  int            `json:"blocked"`
	Ready    int            `json:"ready"`
	ByStatus map[string]int `json:"by_status,omitempty"`
	ByType   map[string]int `json:"by_type,omitempty"`
}
