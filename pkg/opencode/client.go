package opencode

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/binutil"
)

// DefaultServerURL is the default OpenCode server URL.
// Uses 127.0.0.1 instead of localhost to avoid IPv6 resolution issues on macOS.
// On macOS, localhost can resolve to IPv6 ::1 while the server binds to IPv4,
// causing "connection refused" errors.
const DefaultServerURL = "http://127.0.0.1:4096"

// DefaultHTTPTimeout is the default timeout for HTTP requests to the OpenCode API.
// This prevents hangs when OpenCode is in a bad state (e.g., redirect loop).
const DefaultHTTPTimeout = 10 * time.Second

// LargeScannerBufferSize is the buffer size for scanning JSON events from opencode output.
// OpenCode JSON events can be very large (especially tool outputs with file contents),
// so we use 1MB instead of the default 64KB (bufio.MaxScanTokenSize) to prevent ErrTooLong.
const LargeScannerBufferSize = 1024 * 1024 // 1MB

// ClientInterface defines the operations available on an OpenCode client.
// Use this interface for dependency injection and testability.
// The concrete *Client type satisfies this interface.
type ClientInterface interface {
	// Session CRUD
	ListSessions(directory string) ([]Session, error)
	ListSessionsWithOpts(directory string, opts *ListSessionsOpts) ([]Session, error)
	ListDiskSessions(directory string) ([]Session, error)
	GetSession(sessionID string) (*Session, error)
	CreateSession(title, directory, model, variant string, isWorker bool) (*CreateSessionResponse, error)
	DeleteSession(sessionID string) error

	// Session queries
	SessionExists(sessionID string) bool
	IsSessionActive(sessionID string, maxIdleTime time.Duration) bool
	IsSessionProcessing(sessionID string) bool
	FindRecentSession(projectDir string) (string, error)
	FindRecentSessionWithRetry(projectDir string, maxAttempts int, initialDelay time.Duration) (string, error)

	// Messages
	GetMessages(sessionID string) ([]Message, error)
	GetLastMessage(sessionID string) (*Message, error)
	SendMessageAsync(sessionID, content, model string) error
	SendPrompt(sessionID, prompt, model string) error
	SendMessageWithStreaming(sessionID, content string, streamTo io.Writer) error

	// Session metadata
	GetSessionModel(sessionID string) string
	GetSessionTokens(sessionID string) (*TokenStats, error)
	GetSessionEnrichment(sessionID string) SessionEnrichment
	GetLastActivity(sessionID string) (*LastActivity, error)
	UpdateSessionTitle(sessionID, newTitle string) error
	ExportSessionTranscript(sessionID string) (string, error)

	// CLI command builders
	BuildSpawnCommand(prompt, title, model, variant string) *exec.Cmd
	BuildAskCommand(sessionID, prompt string) *exec.Cmd

	// MCP
	MCPStatus() (map[string]MCPServerStatus, error)
	MCPConnect(name string) error
	MCPDisconnect(name string) error
}

// Compile-time assertion that *Client implements ClientInterface.
var _ ClientInterface = (*Client)(nil)

// Client handles OpenCode CLI interactions.
type Client struct {
	ServerURL  string
	httpClient *http.Client
}

// NewClient creates a new OpenCode client with default timeout.
func NewClient(serverURL string) *Client {
	return &Client{
		ServerURL: serverURL,
		httpClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
			// Limit redirects to prevent redirect loops from hanging
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects (max 10)")
				}
				return nil
			},
		},
	}
}

// NewClientWithTimeout creates a new OpenCode client with custom timeout.
func NewClientWithTimeout(serverURL string, timeout time.Duration) *Client {
	return &Client{
		ServerURL: serverURL,
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects (max 10)")
				}
				return nil
			},
		},
	}
}

// OpencodePath is the resolved absolute path to the opencode executable.
// Set this at startup via ResolveOpencodePath() to ensure spawning works
// correctly when running under launchd with minimal PATH.
// If empty, defaults to "opencode" (relies on PATH lookup).
var OpencodePath string

// ResolveOpencodePath attempts to find the opencode executable and stores its absolute path
// in OpencodePath. This should be called at startup by processes that may run under
// launchd or other environments with minimal PATH.
//
// Search order:
// 1. OPENCODE_BIN environment variable (if set)
// 2. Current PATH (via exec.LookPath)
// 3. Common installation locations (~/bin, ~/go/bin, ~/.bun/bin, etc.)
//
// If opencode is found, returns the absolute path and sets OpencodePath.
// If not found, returns an error but OpencodePath remains empty (fallback to "opencode").
func ResolveOpencodePath() (string, error) {
	path, err := binutil.ResolveBinary("opencode", "OPENCODE_BIN", binutil.CommonSearchPaths("opencode"))
	if err != nil {
		return "", err
	}
	OpencodePath = path
	return OpencodePath, nil
}

// getOpencodeBin returns the opencode executable path to use.
// Returns OpencodePath if set, otherwise checks OPENCODE_BIN env var, otherwise "opencode".
func (c *Client) getOpencodeBin() string {
	if OpencodePath != "" {
		return OpencodePath
	}
	if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
		return bin
	}
	return "opencode"
}

// BuildSpawnCommand builds the opencode spawn command.
// variant specifies extended thinking mode: "high" (16k tokens), "max" (32k tokens), or "" (disabled).
func (c *Client) BuildSpawnCommand(prompt, title, model, variant string) *exec.Cmd {
	args := []string{
		"run",
		"--attach", c.ServerURL,
		"--format", "json",
	}

	// Add --model flag only if model is provided
	if model != "" {
		args = append(args, "--model", model)
	}

	// Add --variant flag only if variant is provided
	if variant != "" {
		args = append(args, "--variant", variant)
	}

	args = append(args, "--title", title, prompt)
	return exec.Command(c.getOpencodeBin(), args...)
}

// BuildAskCommand builds the opencode ask command.
func (c *Client) BuildAskCommand(sessionID, prompt string) *exec.Cmd {
	args := []string{
		"run",
		"--attach", c.ServerURL,
		"--session", sessionID,
		"--format", "json",
		prompt,
	}
	return exec.Command(c.getOpencodeBin(), args...)
}

// parseModelSpec parses a model string in "provider/modelID" format
// and returns a map suitable for the OpenCode API.
// Returns nil if the model string is empty or invalid.
func parseModelSpec(model string) map[string]string {
	if model == "" {
		return nil
	}
	// Expected format: "provider/modelID" (e.g., "google/gemini-2.5-flash")
	idx := strings.Index(model, "/")
	if idx <= 0 || idx >= len(model)-1 {
		// Invalid format - no slash or empty provider/model
		return nil
	}
	return map[string]string{
		"providerID": model[:idx],
		"modelID":    model[idx+1:],
	}
}
