package beads

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

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
// If dir is empty, uses current working directory.
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
