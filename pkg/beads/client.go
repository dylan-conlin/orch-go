package beads

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/binutil"
)

// DefaultCLITimeout is the maximum time to wait for a bd CLI fallback command
// to complete. This prevents orch complete from hanging indefinitely when the
// beads daemon is unresponsive or the bd CLI gets stuck.
// 30 seconds is generous enough for any single bd operation while preventing
// indefinite hangs that required manual intervention.
const DefaultCLITimeout = 30 * time.Second

// ErrIssueNotFound is returned when a beads issue lookup fails because the issue doesn't exist.
// This is distinct from RPC errors or other failures - it means the issue ID was not found
// in the beads database. Callers can use errors.Is(err, ErrIssueNotFound) to distinguish
// between "not found" and other error conditions.
var ErrIssueNotFound = errors.New("issue not found")

// ClientVersion is the version of this RPC client.
// Should match the bd CLI version for compatibility.
var ClientVersion = "0.1.0"

// DefaultDir is the default directory to search for .beads/bd.sock
// when FindSocketPath is called with an empty string. Set this at
// startup if the process may run from a different working directory.
var DefaultDir string

// BdPath is the resolved absolute path to the bd executable.
// Set this at startup via ResolveBdPath() to ensure Fallback* functions
// work correctly when running under launchd with minimal PATH.
// If empty, defaults to "bd" (relies on PATH lookup).
var BdPath string

// ResolveBdPath attempts to find the bd executable and stores its absolute path
// in BdPath. This should be called at startup by processes that may run under
// launchd or other environments with minimal PATH.
//
// Search order:
// 1. BD_BIN environment variable (if set)
// 2. Current PATH (via exec.LookPath)
// 3. Common installation locations (~/bin, ~/go/bin, ~/.bun/bin, etc.)
//
// If bd is found, returns the absolute path and sets BdPath.
// If not found, returns an error but BdPath remains empty (fallback to "bd").
func ResolveBdPath() (string, error) {
	path, err := binutil.ResolveBinary("bd", "BD_BIN", binutil.CommonSearchPaths("bd"))
	if err != nil {
		return "", err
	}
	BdPath = path
	return BdPath, nil
}

// getBdPath returns the bd executable path to use.
// Returns BdPath if set, otherwise "bd" (relies on PATH).
func getBdPath() string {
	if BdPath != "" {
		return BdPath
	}
	return "bd"
}

// Client represents a beads RPC client that connects to the daemon.
type Client struct {
	mu            sync.Mutex
	conn          net.Conn
	socketPath    string
	timeout       time.Duration
	cwd           string // Working directory for operations
	autoReconnect bool   // Whether to automatically reconnect on connection errors
	maxRetries    int    // Maximum number of reconnection attempts
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

// WithAutoReconnect enables automatic reconnection on connection errors.
// maxRetries specifies the maximum number of reconnection attempts (0 = no retries).
func WithAutoReconnect(maxRetries int) Option {
	return func(c *Client) {
		c.autoReconnect = true
		c.maxRetries = maxRetries
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
// If dir is empty, uses DefaultDir if set, otherwise uses current working directory.
func FindSocketPath(dir string) (string, error) {
	if dir == "" {
		if DefaultDir != "" {
			dir = DefaultDir
		} else {
			var err error
			dir, err = os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get working directory: %w", err)
			}
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

// Do initializes a client for projectDir and executes fn.
// It centralizes socket lookup and client option setup used by RPC-first callsites.
// If projectDir is empty, DefaultDir is used for WithCwd when set.
func Do(projectDir string, fn func(*Client) error, opts ...Option) error {
	effectiveDir := projectDir
	if effectiveDir == "" {
		effectiveDir = DefaultDir
	}

	socketPath, err := FindSocketPath(projectDir)
	if err != nil {
		return err
	}

	clientOpts := make([]Option, 0, len(opts)+1)
	if effectiveDir != "" {
		clientOpts = append(clientOpts, WithCwd(effectiveDir))
	}
	clientOpts = append(clientOpts, opts...)

	client := NewClient(socketPath, clientOpts...)
	return fn(client)
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
	return c.connectLocked(dialTimeout)
}

// connectLocked performs the actual connection (caller must hold lock).
func (c *Client) connectLocked(dialTimeout time.Duration) error {
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
// If autoReconnect is enabled, it will attempt to reconnect on connection errors.
func (c *Client) execute(operation string, args interface{}) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		if !c.autoReconnect {
			return nil, fmt.Errorf("not connected to daemon")
		}
		// Try to connect if autoReconnect is enabled
		if err := c.connectLocked(200 * time.Millisecond); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	resp, err := c.executeLocked(operation, args)
	if err != nil && c.autoReconnect && isConnectionError(err) {
		// Connection error, attempt to reconnect and retry
		for attempt := 0; attempt <= c.maxRetries; attempt++ {
			// Close existing connection
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}

			// Wait with exponential backoff before reconnecting
			if attempt > 0 {
				backoff := time.Duration(1<<uint(attempt-1)) * 100 * time.Millisecond
				if backoff > 2*time.Second {
					backoff = 2 * time.Second
				}
				time.Sleep(backoff)
			}

			// Try to reconnect
			if err := c.connectLocked(200 * time.Millisecond); err != nil {
				continue // Retry
			}

			// Retry the operation
			resp, err = c.executeLocked(operation, args)
			if err == nil || !isConnectionError(err) {
				break
			}
		}
	}

	return resp, err
}

// isConnectionError returns true if the error indicates a connection problem
// that might be resolved by reconnecting.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "failed to read response") ||
		strings.Contains(errStr, "failed to write request") ||
		strings.Contains(errStr, "i/o timeout")
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
// Note: bd show --json returns an array even for a single issue.
// The RPC daemon may return either format (array or single object) depending on version.
// We try array format first (CLI behavior), then fall back to single object (RPC daemon).
// Returns ErrIssueNotFound if the issue doesn't exist.
func (c *Client) Show(id string) (*Issue, error) {
	args := ShowArgs{ID: id}

	resp, err := c.execute(OpShow, args)
	if err != nil {
		// Check if error message indicates issue not found
		if strings.Contains(err.Error(), "no issue found") || strings.Contains(err.Error(), "issue not found") {
			return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
		}
		return nil, err
	}

	// Handle empty or nil data - issue not found
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
	}

	// Try array format first (bd show --json CLI returns array)
	var issues []Issue
	if err := json.Unmarshal(resp.Data, &issues); err == nil {
		if len(issues) == 0 {
			return nil, fmt.Errorf("%w: %s (empty array)", ErrIssueNotFound, id)
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
	return c.CloseIssueForce(id, reason, false)
}

// CloseIssueForce closes an issue with an optional reason and force flag.
// When force is true, the daemon bypasses Phase: Complete checks.
func (c *Client) CloseIssueForce(id, reason string, force bool) error {
	args := CloseArgs{
		ID:     id,
		Reason: reason,
		Force:  force,
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

// Fallback functions for when daemon is not available.
// These shell out to the bd CLI as a fallback mechanism.

// setupFallbackEnv configures the command environment for CLI fallback.
// Sets BEADS_NO_DAEMON=1 to skip daemon connection attempts, which avoids
// the 5s timeout when running in launchd/minimal environments where the
// daemon socket may not be accessible.
func setupFallbackEnv(cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")
}

// FallbackReady retrieves ready issues via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackReady() ([]Issue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	// Use --limit 0 to get ALL ready issues (bd ready defaults to limit 10)
	cmd := exec.CommandContext(ctx, getBdPath(), "ready", "--json", "--limit", "0")
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd ready timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd ready failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd ready failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd ready output: %w", err)
	}

	return issues, nil
}

// FallbackShow retrieves an issue via bd CLI.
// Note: bd show --json always returns an array, even for a single issue.
// We unmarshal the array and return the first element.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
// Returns ErrIssueNotFound if the issue doesn't exist.
func FallbackShow(id string) (*Issue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, getBdPath(), "show", id, "--json")
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd show timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Check if stderr contains "no issue found" or "no .beads directory" message
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "no issue found") || strings.Contains(stderr, "no .beads directory") {
				return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
			}
			return nil, fmt.Errorf("bd show failed: %w: %s", err, stderr)
		}
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	// Handle empty output - bd show returns exit code 0 but empty output
	// when issue is not found (this is a bd CLI bug but we handle it gracefully)
	if len(output) == 0 || strings.TrimSpace(string(output)) == "" {
		return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
	}

	// bd show returns an array even for a single issue
	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd show output: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("%w: %s (empty array)", ErrIssueNotFound, id)
	}

	return &issues[0], nil
}

// FallbackShowWithDir retrieves an issue via bd CLI from a specific directory.
// This is used for cross-project agent visibility where the beads issue is in a different
// project than the current working directory.
// If dir is empty, uses DefaultDir if set, otherwise the current working directory.
// Uses getBdPath() to resolve the bd executable location.
func FallbackShowWithDir(id, dir string) (*Issue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, getBdPath(), "show", id, "--json")
	setupFallbackEnv(cmd)
	if dir != "" {
		cmd.Dir = dir
	} else if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd show timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Check if stderr contains "no issue found" or "no .beads directory" message
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "no issue found") || strings.Contains(stderr, "no .beads directory") {
				return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
			}
			return nil, fmt.Errorf("bd show failed: %w: %s", err, stderr)
		}
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	// Handle empty output - bd show returns exit code 0 but empty output
	// when issue is not found (this is a bd CLI bug but we handle it gracefully)
	if len(output) == 0 || strings.TrimSpace(string(output)) == "" {
		return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
	}

	// bd show returns an array even for a single issue
	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd show output: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("%w: %s (empty array)", ErrIssueNotFound, id)
	}

	return &issues[0], nil
}

// FallbackList retrieves issues via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
// Uses --limit 0 to get ALL issues (bd list defaults to 50 most recent).
func FallbackList(status string) ([]Issue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	// Use --limit 0 to get ALL issues. Without this, bd list returns only
	// the 50 most recent issues, which can miss in_progress issues when
	// the repo has many recent closed issues (discovered in orch-go-20942).
	args := []string{"list", "--json", "--limit", "0"}
	if status != "" {
		args = append(args, "--status", status)
	}

	cmd := exec.CommandContext(ctx, getBdPath(), args...)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd list timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackListByIDs retrieves specific issues by ID via bd CLI.
// Uses --id and --all flags to fetch issues regardless of status.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackListByIDs(ids []string) ([]Issue, error) {
	if len(ids) == 0 {
		return []Issue{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	// Use --id with comma-separated IDs and --all to include closed issues
	args := []string{"list", "--json", "--all", "--id", strings.Join(ids, ",")}

	cmd := exec.CommandContext(ctx, getBdPath(), args...)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd list --id timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list --id failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list --id failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackListByParent retrieves children of a parent issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackListByParent(parentID string) ([]Issue, error) {
	if parentID == "" {
		return []Issue{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	// Use --parent and --all to include closed children
	// Use --limit 0 to get all children
	args := []string{"list", "--json", "--limit", "0", "--parent", parentID}

	cmd := exec.CommandContext(ctx, getBdPath(), args...)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd list --parent timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list --parent failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list --parent failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackStats retrieves stats via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackStats() (*Stats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, getBdPath(), "stats", "--json")
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd stats timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd stats failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd stats failed: %w", err)
	}

	var stats Stats
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse bd stats output: %w", err)
	}

	return &stats, nil
}

// FallbackComments retrieves comments via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackComments(id string) ([]Comment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, getBdPath(), "comments", id, "--json")
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd comments timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd comments failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd comments failed: %w", err)
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse bd comments output: %w", err)
	}

	return comments, nil
}

// FallbackClose closes an issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackClose(id, reason string) error {
	return FallbackCloseForce(id, reason, false)
}

// FallbackCloseForce closes an issue via bd CLI with optional --force flag.
// When force is true, passes --force to bypass bd's "Phase: Complete" check.
// This is needed when callers (orch complete, daemon) have already verified
// Phase: Complete and the redundant bd gate would reject the close.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackCloseForce(id, reason string, force bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	args := []string{"close", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	if force {
		args = append(args, "--force")
	}

	cmd := exec.CommandContext(ctx, getBdPath(), args...)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("bd close timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd close failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackCreate creates an issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackCreate(title, description, issueType string, priority int, labels []string) (*Issue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

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

	cmd := exec.CommandContext(ctx, getBdPath(), args...)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd create timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd create failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd create failed: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse bd create output: %w", err)
	}

	return &issue, nil
}

// FallbackAddComment adds a comment via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackAddComment(id, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, getBdPath(), "comments", "add", id, text)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("bd comments add timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd comments add failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackUpdate updates an issue via bd CLI.
// Currently supports updating the status field.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackUpdate(id, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	args := []string{"update", id}
	if status != "" {
		args = append(args, "--status", status)
	}
	cmd := exec.CommandContext(ctx, getBdPath(), args...)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("bd update timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd update failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackRemoveLabel removes a label from an issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackRemoveLabel(id, label string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, getBdPath(), "update", id, "--remove-label", label)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("bd remove-label timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd remove-label failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackAddLabel adds a label to an issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackAddLabel(id, label string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, getBdPath(), "update", id, "--add-label", label)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("bd add-label timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd add-label failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackReopen reopens an issue via bd CLI.
// Uses bd reopen which emits a Reopened event (distinct from simply updating status to open).
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackReopen(id, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	args := []string{"reopen", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	cmd := exec.CommandContext(ctx, getBdPath(), args...)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("bd reopen timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd reopen failed: %w: %s", err, string(output))
	}
	return nil
}

// CheckBlockingDependencies checks if an issue has any blocking dependencies.
// Returns a list of blocking dependencies if any exist, or nil if the issue can be worked on.
// Uses RPC client if available, falls back to CLI otherwise.
func CheckBlockingDependencies(issueID string) ([]BlockingDependency, error) {
	// Try RPC client first
	socketPath, err := FindSocketPath("")
	if err == nil {
		client := NewClient(socketPath, WithAutoReconnect(2))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Show(issueID)
			if err == nil {
				return issue.GetBlockingDependencies(), nil
			}
			// Fall through to CLI on error
		}
	}

	// Fallback to CLI
	issue, err := FallbackShow(issueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", issueID, err)
	}

	return issue.GetBlockingDependencies(), nil
}

// BlockingDependencyError represents an error when an issue is blocked by dependencies.
type BlockingDependencyError struct {
	IssueID      string
	Blockers     []BlockingDependency
	ForceMessage string
}

func (e *BlockingDependencyError) Error() string {
	if len(e.Blockers) == 0 {
		return fmt.Sprintf("issue %s has unknown blockers", e.IssueID)
	}

	var blockerStrs []string
	for _, b := range e.Blockers {
		blockerStrs = append(blockerStrs, fmt.Sprintf("%s (%s)", b.ID, b.Status))
	}

	return fmt.Sprintf("%s is blocked by: %s\n%s",
		e.IssueID,
		strings.Join(blockerStrs, ", "),
		e.ForceMessage)
}
