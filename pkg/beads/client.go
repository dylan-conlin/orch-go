package beads

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/binutil"
	_ "modernc.org/sqlite"
)

// DefaultCLITimeout is the maximum time to wait for a bd CLI fallback command
// to complete. This prevents orch complete from hanging indefinitely when the
// beads daemon is unresponsive or the bd CLI gets stuck.
// 10 seconds is long enough for normal calls while failing fast under contention.
const (
	DefaultCLITimeout       = 10 * time.Second
	defaultMaxBDSubprocess  = 12
	bdSubprocessLimitEnvVar = "ORCH_BD_MAX_CONCURRENT"
	bdDisableSandboxEnvVar  = "ORCH_BD_DISABLE_SANDBOX"
	bdStaleGracePeriod      = 30 * time.Second
)

var (
	maxBDSubprocesses = resolveBDSubprocessLimit()
	bdSubprocessSem   = make(chan struct{}, maxBDSubprocesses)
	useBDSandboxMode  = resolveBDSandboxMode()
)

// BDSubprocessLimit returns the configured hard cap for concurrent bd CLI subprocesses.
func BDSubprocessLimit() int {
	return maxBDSubprocesses
}

func resolveBDSubprocessLimit() int {
	raw := strings.TrimSpace(os.Getenv(bdSubprocessLimitEnvVar))
	if raw == "" {
		return defaultMaxBDSubprocess
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		log.Printf("event=bd_subprocess_limit_invalid component=beads env=%q value=%q default=%d", bdSubprocessLimitEnvVar, raw, defaultMaxBDSubprocess)
		return defaultMaxBDSubprocess
	}

	return limit
}

func resolveBDSandboxMode() bool {
	raw := strings.TrimSpace(os.Getenv(bdDisableSandboxEnvVar))
	if raw == "" {
		return true
	}

	disable, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("event=bd_sandbox_flag_invalid component=beads env=%q value=%q default=true", bdDisableSandboxEnvVar, raw)
		return true
	}

	return !disable
}

func prependSandboxArg(args []string) []string {
	includeSandbox := useBDSandboxMode && !hasCLIArg(args, "--sandbox")
	includeQuiet := os.Getenv("ORCH_DEBUG") == "" && !hasCLIArg(args, "--quiet") && !hasCLIArg(args, "-q")

	if !includeSandbox && !includeQuiet {
		return args
	}

	extra := 0
	if includeSandbox {
		extra++
	}
	if includeQuiet {
		extra++
	}

	cmdArgs := make([]string, 0, len(args)+extra)
	if includeSandbox {
		cmdArgs = append(cmdArgs, "--sandbox")
	}
	if includeQuiet {
		cmdArgs = append(cmdArgs, "--quiet")
	}
	cmdArgs = append(cmdArgs, args...)
	return cmdArgs
}

// acquireBdSubprocessSlot enforces a hard cap across all bd CLI subprocesses.
// Logs when the cap is reached so stampedes are visible in server logs.
func acquireBdSubprocessSlot(ctx context.Context, operation string) (func(), error) {
	select {
	case bdSubprocessSem <- struct{}{}:
		return func() { <-bdSubprocessSem }, nil
	default:
		if os.Getenv("ORCH_DEBUG") != "" {
			log.Printf("event=bd_subprocess_cap_hit component=beads operation=%q inflight=%d cap=%d", operation, len(bdSubprocessSem), cap(bdSubprocessSem))
		}
	}

	select {
	case bdSubprocessSem <- struct{}{}:
		return func() { <-bdSubprocessSem }, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("bd subprocess slot acquire timeout: %w", ctx.Err())
	}
}

func IsCLITimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

func runBDCommand(workDir, bdPath string, env []string, combined bool, args ...string) ([]byte, error) {
	acquireCtx, acquireCancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer acquireCancel()

	operation := "bd"
	if len(args) > 0 {
		operation = "bd " + args[0]
	}

	release, err := acquireBdSubprocessSlot(acquireCtx, operation)
	if err != nil {
		return nil, err
	}
	defer release()

	execCtx, execCancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer execCancel()

	if bdPath == "" {
		bdPath = getBdPath()
	}

	resolvedWorkDir, err := resolveBDWorkDir(workDir)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(execCtx, bdPath, prependSandboxArg(args)...)
	if env != nil {
		cmd.Env = env
	} else {
		setupFallbackEnv(cmd)
	}
	if resolvedWorkDir != "" {
		cmd.Dir = resolvedWorkDir
	}

	var output []byte
	if combined {
		output, err = cmd.CombinedOutput()
	} else {
		output, err = cmd.Output()
	}
	if err != nil && errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		log.Printf("event=bd_subprocess_timeout component=beads operation=%q timeout=%s", operation, DefaultCLITimeout)
	}

	if shouldRetryWithAllowStale(resolvedWorkDir, args, err, output) {
		retryArgs := append(append([]string{}, args...), "--allow-stale")
		retryCmd := exec.CommandContext(execCtx, bdPath, prependSandboxArg(retryArgs)...)
		if env != nil {
			retryCmd.Env = env
		} else {
			setupFallbackEnv(retryCmd)
		}
		if resolvedWorkDir != "" {
			retryCmd.Dir = resolvedWorkDir
		}

		if combined {
			return retryCmd.CombinedOutput()
		}
		return retryCmd.Output()
	}

	return output, err
}

func resolveBDWorkDir(workDir string) (string, error) {
	dir := strings.TrimSpace(workDir)
	if dir == "" {
		dir = strings.TrimSpace(DefaultDir)
	}
	if dir == "" {
		return "", nil
	}

	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("bd workdir is not a directory: %s", dir)
		}
		return dir, nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to inspect bd workdir %s: %w", dir, err)
	}

	fallbackDir, fallbackErr := findNearestBeadsProjectDir(dir)
	if fallbackErr != nil {
		return "", fmt.Errorf("bd workdir %s does not exist and no project root fallback was found", dir)
	}

	log.Printf("event=bd_workdir_fallback component=beads from=%q to=%q", dir, fallbackDir)
	return fallbackDir, nil
}

func findNearestBeadsProjectDir(dir string) (string, error) {
	current := filepath.Clean(dir)

	for {
		info, err := os.Stat(current)
		if err == nil && info.IsDir() {
			beadsDir := filepath.Join(current, ".beads")
			beadsInfo, beadsErr := os.Stat(beadsDir)
			if beadsErr == nil && beadsInfo.IsDir() {
				return current, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("no beads project root found for %s", dir)
}

func shouldRetryWithAllowStale(workDir string, args []string, err error, output []byte) bool {
	if hasCLIArg(args, "--allow-stale") {
		return false
	}

	if !isOutOfSyncFailure(err, output) {
		return false
	}

	recent, recentErr := importedRecently(workDir, bdStaleGracePeriod)
	if recentErr == nil {
		if recent {
			log.Printf("event=bd_stale_grace_retry component=beads source=last_import_time grace=%s", bdStaleGracePeriod)
		}
		return recent
	}

	hotJSONL, hotErr := jsonlUpdatedRecently(workDir, bdStaleGracePeriod)
	if hotErr != nil {
		return false
	}
	if hotJSONL {
		log.Printf("event=bd_stale_grace_retry component=beads source=jsonl_mtime grace=%s", bdStaleGracePeriod)
	}

	return hotJSONL
}

func hasCLIArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func isOutOfSyncFailure(err error, output []byte) bool {
	if err != nil {
		return isOutOfSyncText(joinOutOfSyncText(err, output))
	}

	msg, ok := outputErrorMessage(output)
	if !ok {
		return false
	}

	return isOutOfSyncText(strings.ToLower(msg))
}

func joinOutOfSyncText(err error, output []byte) string {
	var joined strings.Builder
	joined.WriteString(strings.ToLower(err.Error()))
	if len(output) > 0 {
		joined.WriteString(" ")
		joined.WriteString(strings.ToLower(string(output)))
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		stderr := strings.TrimSpace(strings.ToLower(string(exitErr.Stderr)))
		if stderr != "" {
			joined.WriteString(" ")
			joined.WriteString(stderr)
		}
	}

	return joined.String()
}

func outputErrorMessage(output []byte) (string, bool) {
	if len(output) == 0 {
		return "", false
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(output, &payload); err != nil {
		return "", false
	}

	raw, ok := payload["error"]
	if !ok {
		return "", false
	}

	var msg string
	if err := json.Unmarshal(raw, &msg); err != nil {
		return "", false
	}

	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "", false
	}

	return msg, true
}

func isOutOfSyncText(errText string) bool {
	return strings.Contains(errText, "database out of sync with jsonl") ||
		strings.Contains(errText, "out of sync with jsonl") ||
		strings.Contains(errText, "run 'bd sync --import-only'")
}

func importedRecently(workDir string, within time.Duration) (bool, error) {
	if within <= 0 {
		return false, nil
	}

	dbPath, err := findBeadsDBPath(workDir)
	if err != nil {
		return false, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var raw string
	err = db.QueryRow(`SELECT value FROM metadata WHERE key = 'last_import_time'`).Scan(&raw)
	if err != nil {
		return false, err
	}

	importedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
	if err != nil {
		return false, err
	}

	age := time.Since(importedAt)
	if age < 0 {
		return false, nil
	}

	return age < within, nil
}

func jsonlUpdatedRecently(workDir string, within time.Duration) (bool, error) {
	if within <= 0 {
		return false, nil
	}

	jsonlPath, err := findBeadsJSONLPath(workDir)
	if err != nil {
		return false, err
	}

	info, err := os.Stat(jsonlPath)
	if err != nil {
		return false, err
	}

	age := time.Since(info.ModTime())
	if age < 0 {
		return false, nil
	}

	return age < within, nil
}

func findBeadsDBPath(dir string) (string, error) {
	if dir == "" {
		if DefaultDir != "" {
			dir = DefaultDir
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			dir = cwd
		}
	}

	current := dir
	for {
		dbPath := filepath.Join(current, ".beads", "beads.db")
		if _, err := os.Stat(dbPath); err == nil {
			return dbPath, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no beads database found in %s or parent directories", dir)
		}

		current = parent
	}
}

func findBeadsJSONLPath(dir string) (string, error) {
	if dir == "" {
		if DefaultDir != "" {
			dir = DefaultDir
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			dir = cwd
		}
	}

	current := dir
	for {
		jsonlPath := filepath.Join(current, ".beads", "issues.jsonl")
		if _, err := os.Stat(jsonlPath); err == nil {
			return jsonlPath, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no beads JSONL found in %s or parent directories", dir)
		}

		current = parent
	}
}

func runBDOutput(workDir string, args ...string) ([]byte, error) {
	return runBDCommand(workDir, getBdPath(), nil, false, args...)
}

func runBDCombinedOutput(workDir string, args ...string) ([]byte, error) {
	return runBDCommand(workDir, getBdPath(), nil, true, args...)
}

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

// setupFallbackEnv configures the command environment for CLI fallback.
// Sets BEADS_NO_DAEMON=1 to skip daemon connection attempts, which avoids
// the 5s timeout when running in launchd/minimal environments where the
// daemon socket may not be accessible.
func setupFallbackEnv(cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")
}
