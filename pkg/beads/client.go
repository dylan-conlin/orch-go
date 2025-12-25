package beads

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// ClientVersion is the version of this RPC client.
// Should match the bd CLI version for compatibility.
var ClientVersion = "0.1.0"

// Client represents a beads RPC client that connects to the daemon.
type Client struct {
	mu         sync.Mutex
	conn       net.Conn
	socketPath string
	timeout    time.Duration
	cwd        string // Working directory for operations
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithTimeout sets the request timeout duration.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithCwd sets the working directory for operations.
func WithCwd(cwd string) Option {
	return func(c *Client) {
		c.cwd = cwd
	}
}

// NewClient creates a new beads client with the given options.
// The socketPath should point to the .beads/bd.sock file.
func NewClient(socketPath string, opts ...Option) *Client {
	c := &Client{
		socketPath: socketPath,
		timeout:    30 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// FindSocketPath finds the beads socket path for a directory.
// It looks for .beads/bd.sock in the given directory or walks up to find it.
func FindSocketPath(dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Walk up directory tree looking for .beads/bd.sock
	current := dir
	for {
		socketPath := filepath.Join(current, ".beads", "bd.sock")
		if _, err := os.Stat(socketPath); err == nil {
			return socketPath, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root without finding socket
			return "", fmt.Errorf("no beads socket found in %s or parent directories", dir)
		}
		current = parent
	}
}

// Connect attempts to connect to the beads daemon.
// Returns nil if daemon is not running or unhealthy.
func (c *Client) Connect() error {
	return c.ConnectWithTimeout(200 * time.Millisecond)
}

// ConnectWithTimeout attempts to connect to the daemon with custom dial timeout.
func (c *Client) ConnectWithTimeout(dialTimeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if socket exists
	if _, err := os.Stat(c.socketPath); os.IsNotExist(err) {
		return fmt.Errorf("daemon not running: socket not found at %s", c.socketPath)
	}

	// Dial the socket
	conn, err := net.DialTimeout("unix", c.socketPath, dialTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}

	c.conn = conn

	// Perform health check
	health, err := c.healthLocked()
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return fmt.Errorf("health check failed: %w", err)
	}

	if health.Status == "unhealthy" {
		c.conn.Close()
		c.conn = nil
		return fmt.Errorf("daemon unhealthy: %s", health.Error)
	}

	return nil
}

// Close closes the connection to the daemon.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// IsConnected returns true if the client has an active connection.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// Reconnect closes any existing connection and attempts to reconnect.
func (c *Client) Reconnect() error {
	c.Close()
	return c.Connect()
}

// execute sends an RPC request and returns the response.
func (c *Client) execute(operation string, args interface{}) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected to daemon")
	}

	return c.executeLocked(operation, args)
}

// executeLocked performs the actual RPC call (caller must hold lock).
func (c *Client) executeLocked(operation string, args interface{}) (*Response, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal args: %w", err)
	}

	cwd := c.cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	req := Request{
		Operation:     operation,
		Args:          argsJSON,
		ClientVersion: ClientVersion,
		Cwd:           cwd,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Set deadline
	if c.timeout > 0 {
		if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
			return nil, fmt.Errorf("failed to set deadline: %w", err)
		}
	}

	// Write request
	writer := bufio.NewWriter(c.conn)
	if _, err := writer.Write(reqJSON); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}
	if err := writer.WriteByte('\n'); err != nil {
		return nil, fmt.Errorf("failed to write newline: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush: %w", err)
	}

	// Read response
	reader := bufio.NewReader(c.conn)
	respLine, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(respLine, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !resp.Success {
		return &resp, fmt.Errorf("operation failed: %s", resp.Error)
	}

	return &resp, nil
}

// healthLocked performs a health check (caller must hold lock).
func (c *Client) healthLocked() (*HealthResponse, error) {
	resp, err := c.executeLocked(OpHealth, nil)
	if err != nil {
		return nil, err
	}

	var health HealthResponse
	if err := json.Unmarshal(resp.Data, &health); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health response: %w", err)
	}

	return &health, nil
}

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
func (c *Client) Show(id string) (*Issue, error) {
	args := ShowArgs{ID: id}

	resp, err := c.execute(OpShow, args)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(resp.Data, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issue: %w", err)
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

// Fallback functions for when daemon is not available.
// These shell out to the bd CLI as a fallback mechanism.

// FallbackReady retrieves ready issues via bd CLI.
func FallbackReady() ([]Issue, error) {
	cmd := exec.Command("bd", "ready", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd ready failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd ready output: %w", err)
	}

	return issues, nil
}

// FallbackShow retrieves an issue via bd CLI.
func FallbackShow(id string) (*Issue, error) {
	cmd := exec.Command("bd", "show", id, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse bd show output: %w", err)
	}

	return &issue, nil
}

// FallbackList retrieves issues via bd CLI.
func FallbackList(status string) ([]Issue, error) {
	args := []string{"list", "--json"}
	if status != "" {
		args = append(args, "--status", status)
	}

	cmd := exec.Command("bd", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackStats retrieves stats via bd CLI.
func FallbackStats() (*Stats, error) {
	cmd := exec.Command("bd", "stats", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd stats failed: %w", err)
	}

	var stats Stats
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse bd stats output: %w", err)
	}

	return &stats, nil
}

// FallbackComments retrieves comments via bd CLI.
func FallbackComments(id string) ([]Comment, error) {
	cmd := exec.Command("bd", "comments", id, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd comments failed: %w", err)
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse bd comments output: %w", err)
	}

	return comments, nil
}

// FallbackClose closes an issue via bd CLI.
func FallbackClose(id, reason string) error {
	args := []string{"close", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	cmd := exec.Command("bd", args...)
	return cmd.Run()
}

// FallbackCreate creates an issue via bd CLI.
func FallbackCreate(title, description, issueType string, priority int, labels []string) (*Issue, error) {
	args := []string{"create", title, "--json"}
	if description != "" {
		args = append(args, "--description", description)
	}
	if issueType != "" {
		args = append(args, "--type", issueType)
	}
	if priority > 0 {
		args = append(args, "--priority", fmt.Sprintf("%d", priority))
	}
	for _, label := range labels {
		args = append(args, "--label", label)
	}

	cmd := exec.Command("bd", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd create failed: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse bd create output: %w", err)
	}

	return &issue, nil
}

// FallbackAddComment adds a comment via bd CLI.
func FallbackAddComment(id, text string) error {
	cmd := exec.Command("bd", "comment", id, text)
	return cmd.Run()
}
