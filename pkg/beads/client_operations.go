package beads

import (
	"encoding/json"
	"fmt"
)

// Health performs a health check.
func (c *Client) Health() (*HealthResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected to daemon")
	}

	return c.healthLocked()
}

// Ready retrieves ready issues from the daemon.
func (c *Client) Ready(args *ReadyArgs) ([]Issue, error) {
	if args == nil {
		args = &ReadyArgs{}
	}

	resp, err := c.execute(OpReady, args)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(resp.Data, &issues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ready issues: %w", err)
	}

	return issues, nil
}

// Show retrieves a single issue by ID.
// Note: bd show --json returns an array even for a single issue.
// The RPC daemon may return either format (array or single object) depending on version.
// We try array format first (CLI behavior), then fall back to single object (RPC daemon).
func (c *Client) Show(id string) (*Issue, error) {
	args := ShowArgs{ID: id}

	resp, err := c.execute(OpShow, args)
	if err != nil {
		return nil, err
	}

	// Try array format first (bd show --json CLI returns array)
	var issues []Issue
	if err := json.Unmarshal(resp.Data, &issues); err == nil {
		if len(issues) == 0 {
			return nil, fmt.Errorf("bd show returned empty array for id: %s", id)
		}
		return &issues[0], nil
	}

	// Fall back to single object format (some RPC daemon versions)
	var issue Issue
	if err := json.Unmarshal(resp.Data, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issue (tried array and object): %w", err)
	}

	return &issue, nil
}

// List retrieves issues matching the given criteria.
func (c *Client) List(args *ListArgs) ([]Issue, error) {
	if args == nil {
		args = &ListArgs{}
	}

	resp, err := c.execute(OpList, args)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(resp.Data, &issues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issues: %w", err)
	}

	return issues, nil
}

// Stats retrieves beads statistics.
func (c *Client) Stats() (*Stats, error) {
	resp, err := c.execute(OpStats, nil)
	if err != nil {
		return nil, err
	}

	// RPC returns flat stats (no "summary" wrapper), CLI returns wrapped.
	// Try flat format first (RPC), then wrapped format (CLI fallback compatibility).
	var summary StatsSummary
	if err := json.Unmarshal(resp.Data, &summary); err == nil && summary.TotalIssues > 0 {
		// RPC format: flat StatsSummary
		return &Stats{Summary: summary}, nil
	}

	// Try wrapped format (CLI format)
	var stats Stats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	return &stats, nil
}

// Comments retrieves comments for an issue.
func (c *Client) Comments(id string) ([]Comment, error) {
	args := CommentListArgs{ID: id}

	resp, err := c.execute(OpCommentList, args)
	if err != nil {
		return nil, err
	}

	var comments []Comment
	if err := json.Unmarshal(resp.Data, &comments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal comments: %w", err)
	}

	return comments, nil
}

// AddComment adds a comment to an issue.
func (c *Client) AddComment(id, author, text string) error {
	args := CommentAddArgs{
		ID:     id,
		Author: author,
		Text:   text,
	}

	_, err := c.execute(OpCommentAdd, args)
	return err
}

// CloseIssue closes an issue with an optional reason.
func (c *Client) CloseIssue(id, reason string) error {
	args := CloseArgs{
		ID:     id,
		Reason: reason,
	}

	_, err := c.execute(OpClose, args)
	return err
}

// Create creates a new issue.
func (c *Client) Create(args *CreateArgs) (*Issue, error) {
	if args == nil {
		return nil, fmt.Errorf("create args required")
	}

	resp, err := c.execute(OpCreate, args)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(resp.Data, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal created issue: %w", err)
	}

	return &issue, nil
}

// Update updates an existing issue.
func (c *Client) Update(args *UpdateArgs) (*Issue, error) {
	if args == nil {
		return nil, fmt.Errorf("update args required")
	}

	resp, err := c.execute(OpUpdate, args)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(resp.Data, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal updated issue: %w", err)
	}

	return &issue, nil
}

// Delete deletes one or more issues.
func (c *Client) Delete(args *DeleteArgs) error {
	if args == nil {
		return fmt.Errorf("delete args required")
	}

	_, err := c.execute(OpDelete, args)
	return err
}

// Stale retrieves stale issues.
func (c *Client) Stale(args *StaleArgs) ([]Issue, error) {
	if args == nil {
		args = &StaleArgs{}
	}

	resp, err := c.execute(OpStale, args)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(resp.Data, &issues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stale issues: %w", err)
	}

	return issues, nil
}

// Count counts issues matching the given criteria.
func (c *Client) Count(args *CountArgs) (*CountResponse, error) {
	if args == nil {
		args = &CountArgs{}
	}

	resp, err := c.execute(OpCount, args)
	if err != nil {
		return nil, err
	}

	var countResp CountResponse
	if err := json.Unmarshal(resp.Data, &countResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal count response: %w", err)
	}

	return &countResp, nil
}

// Status retrieves daemon status metadata.
func (c *Client) Status() (*StatusResponse, error) {
	resp, err := c.execute(OpStatus, nil)
	if err != nil {
		return nil, err
	}

	var status StatusResponse
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status response: %w", err)
	}

	return &status, nil
}

// Ping sends a ping to verify the daemon is alive.
func (c *Client) Ping() error {
	_, err := c.execute(OpPing, nil)
	return err
}

// Shutdown sends a graceful shutdown request to the daemon.
func (c *Client) Shutdown() error {
	_, err := c.execute(OpShutdown, nil)
	return err
}

// AddLabel adds a label to an issue.
func (c *Client) AddLabel(id, label string) error {
	args := LabelAddArgs{
		ID:    id,
		Label: label,
	}

	_, err := c.execute(OpLabelAdd, args)
	return err
}

// RemoveLabel removes a label from an issue.
func (c *Client) RemoveLabel(id, label string) error {
	args := LabelRemoveArgs{
		ID:    id,
		Label: label,
	}

	_, err := c.execute(OpLabelRemove, args)
	return err
}

// AddDependency adds a dependency between issues.
func (c *Client) AddDependency(fromID, toID, depType string) error {
	args := DepAddArgs{
		FromID:  fromID,
		ToID:    toID,
		DepType: depType,
	}

	_, err := c.execute(OpDepAdd, args)
	return err
}

// RemoveDependency removes a dependency between issues.
func (c *Client) RemoveDependency(fromID, toID, depType string) error {
	args := DepRemoveArgs{
		FromID:  fromID,
		ToID:    toID,
		DepType: depType,
	}

	_, err := c.execute(OpDepRemove, args)
	return err
}

// ResolveID resolves a partial issue ID to a full ID.
func (c *Client) ResolveID(partialID string) (string, error) {
	args := ResolveIDArgs{ID: partialID}

	resp, err := c.execute(OpResolveID, args)
	if err != nil {
		return "", err
	}

	// The response data is the resolved ID as a string
	var resolvedID string
	if err := json.Unmarshal(resp.Data, &resolvedID); err != nil {
		return "", fmt.Errorf("failed to unmarshal resolved ID: %w", err)
	}

	return resolvedID, nil
}
