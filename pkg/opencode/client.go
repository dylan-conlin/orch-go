package opencode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
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

func (c *Client) getOpencodeBin() string {
	if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
		return bin
	}
	return "opencode"
}

// ParseEvent parses a JSON event from opencode output.
func ParseEvent(line string) (Event, error) {
	var event Event
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return event, err
	}
	return event, nil
}

// ExtractSessionID extracts session ID from opencode events.
// OpenCode includes sessionID at the top level of each event.
func ExtractSessionID(events []string) (string, error) {
	for _, line := range events {
		event, err := ParseEvent(line)
		if err != nil {
			continue
		}
		if event.SessionID != "" {
			return event.SessionID, nil
		}
	}
	return "", ErrNoSessionID
}

// ExtractSessionIDFromReader reads from a reader until it finds a session ID.
// Returns as soon as a session ID is found, leaving remaining data unread.
// This is useful for headless spawns where we need the session ID quickly
// but don't want to block waiting for the process to complete.
//
// The function is robust to non-JSON content mixed with JSON lines. This handles
// cases where warnings from npm dependencies leak into stdout without a trailing
// newline, e.g.: "[baseline-browser-mapping] warning...{"type":"event",...}
func ExtractSessionIDFromReader(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	// Use large buffer to handle OpenCode's potentially large JSON events
	scanner.Buffer(make([]byte, 0, LargeScannerBufferSize), LargeScannerBufferSize)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Try to extract session ID from this line.
		// Uses findSessionIDInLine which handles both pure JSON lines
		// and lines with non-JSON content prepended (e.g., npm warnings).
		if sessionID := findSessionIDInLine(line); sessionID != "" {
			return sessionID, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanner error: %w", err)
	}
	return "", ErrNoSessionID
}

// findSessionIDInLine attempts to extract a session ID from a line that may contain
// mixed content (non-JSON prefix followed by JSON). This handles cases where
// npm warnings like "[baseline-browser-mapping] ..." leak into stdout without
// a trailing newline, concatenating with the JSON event.
//
// Returns the session ID if found, or empty string if not found.
func findSessionIDInLine(line string) string {
	// First, try parsing the entire line as JSON (most common case)
	event, err := ParseEvent(line)
	if err == nil && event.SessionID != "" {
		return event.SessionID
	}

	// If that failed, look for JSON object start markers and try parsing from there.
	// This handles lines like: '[warn] message{"type":"event","sessionID":"ses_..."}'
	for i := 0; i < len(line); i++ {
		if line[i] == '{' {
			// Try to parse JSON starting from this position
			event, err := ParseEvent(line[i:])
			if err == nil && event.SessionID != "" {
				return event.SessionID
			}
			// If parsing failed, continue looking for next '{'
			// This handles malformed content before valid JSON
		}
	}

	return ""
}

// ProcessOutput processes the output from opencode command.
func ProcessOutput(r io.Reader) (*Result, error) {
	result := &Result{}
	scanner := bufio.NewScanner(r)
	// Use large buffer to handle OpenCode's potentially large JSON events
	scanner.Buffer(make([]byte, 0, LargeScannerBufferSize), LargeScannerBufferSize)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		event, err := ParseEvent(line)
		if err != nil {
			// Skip non-JSON lines
			continue
		}

		result.Events = append(result.Events, event)

		// sessionID is at top level of each event - grab from first event that has it
		if result.SessionID == "" && event.SessionID != "" {
			result.SessionID = event.SessionID
		}
	}

	if err := scanner.Err(); err != nil {
		return result, err
	}

	return result, nil
}

// ProcessOutputWithStreaming processes the output from opencode command
// and streams text content to the provided writer.
func ProcessOutputWithStreaming(r io.Reader, streamTo io.Writer) (*Result, error) {
	result := &Result{}
	scanner := bufio.NewScanner(r)
	// Use large buffer to handle OpenCode's potentially large JSON events
	scanner.Buffer(make([]byte, 0, LargeScannerBufferSize), LargeScannerBufferSize)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		event, err := ParseEvent(line)
		if err != nil {
			// Skip non-JSON lines
			continue
		}

		result.Events = append(result.Events, event)

		// sessionID is at top level of each event - grab from first event that has it
		if result.SessionID == "" && event.SessionID != "" {
			result.SessionID = event.SessionID
		}

		// Stream text content to output
		if event.Type == "text" && event.Content != "" {
			streamTo.Write([]byte(event.Content))
		}
	}

	if err := scanner.Err(); err != nil {
		return result, err
	}

	return result, nil
}

// BuildSpawnCommand builds the opencode spawn command.
func (c *Client) BuildSpawnCommand(prompt, title, model string) *exec.Cmd {
	args := []string{
		"run",
		"--attach", c.ServerURL,
		"--format", "json",
	}

	// Add --model flag only if model is provided
	if model != "" {
		args = append(args, "--model", model)
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

// SendMessageAsync sends a message to an existing session asynchronously.
// The model parameter is optional - if empty, OpenCode will use the default model.
// Model should be in "provider/modelID" format (e.g., "google/gemini-2.5-flash").
func (c *Client) SendMessageAsync(sessionID, content, model string) error {
	payload := map[string]any{
		"parts": []map[string]string{{"type": "text", "text": content}},
		"agent": "build",
	}
	// Include model if provided (per-message model in OpenCode)
	// OpenCode expects model as an object with providerID and modelID fields
	if model != "" {
		modelObj := parseModelSpec(model)
		if modelObj != nil {
			payload["model"] = modelObj
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.ServerURL+"/session/"+sessionID+"/prompt_async", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
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

// ListSessions fetches all sessions from the OpenCode API.
// If directory is provided, it passes it via x-opencode-directory header.
func (c *Client) ListSessions(directory string) ([]Session, error) {
	req, err := http.NewRequest("GET", c.ServerURL+"/session", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if directory != "" {
		req.Header.Set("x-opencode-directory", directory)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return sessions, nil
}

// GetSession fetches a single session by ID from the OpenCode API.
func (c *Client) GetSession(sessionID string) (*Session, error) {
	req, err := http.NewRequest("GET", c.ServerURL+"/session/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	return &session, nil
}

// SessionExists checks if a session exists in OpenCode (in-memory).
// Returns true if the session is accessible via the API, false otherwise.
// NOTE: This returns true for any persisted session, not just actively running ones.
// For liveness checks, use IsSessionActive() instead.
func (c *Client) SessionExists(sessionID string) bool {
	req, err := http.NewRequest("GET", c.ServerURL+"/session/"+sessionID, nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Session exists if we get 200 OK
	return resp.StatusCode == http.StatusOK
}

// IsSessionActive checks if a session is actively running (updated within maxIdleTime).
// This is more reliable than SessionExists() for liveness detection because OpenCode
// persists sessions to disk, so SessionExists() returns true for historical sessions.
// maxIdleTime is the maximum time since last update to consider a session "active".
func (c *Client) IsSessionActive(sessionID string, maxIdleTime time.Duration) bool {
	session, err := c.GetSession(sessionID)
	if err != nil {
		return false
	}

	updatedAt := time.Unix(session.Time.Updated/1000, 0)
	idleTime := time.Since(updatedAt)
	return idleTime <= maxIdleTime
}

// IsSessionProcessing checks if a session is actively processing (has a pending assistant response).
// This is the most reliable signal for detecting truly active agents because it checks
// whether the last assistant message has finished (finish != "" and completed != 0).
// Returns true if the session is currently generating a response, false if idle.
func (c *Client) IsSessionProcessing(sessionID string) bool {
	messages, err := c.GetMessages(sessionID)
	if err != nil || len(messages) == 0 {
		return false
	}

	// Find the last message
	lastMsg := messages[len(messages)-1]

	// Session is processing if:
	// 1. Last message is from assistant AND
	// 2. It hasn't finished yet (finish is empty and completed is 0)
	if lastMsg.Info.Role == "assistant" {
		return lastMsg.Info.Finish == "" && lastMsg.Info.Time.Completed == 0
	}

	// If last message is from user, check if there's an assistant response being generated
	// This happens when user sends a message and assistant hasn't started responding yet
	// In this case, we consider the session as processing (waiting for response)
	if lastMsg.Info.Role == "user" {
		// If the user message was sent recently (within last 30 seconds), consider it processing
		createdAt := time.Unix(lastMsg.Info.Time.Created/1000, 0)
		return time.Since(createdAt) < 30*time.Second
	}

	return false
}

// GetLastMessage returns the last message in a session, or nil if the session has no messages.
func (c *Client) GetLastMessage(sessionID string) (*Message, error) {
	messages, err := c.GetMessages(sessionID)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return &messages[len(messages)-1], nil
}

// LastActivity represents the most recent activity from a session.
// This is used to populate agent activity on initial dashboard load.
type LastActivity struct {
	Text      string // Activity description (truncated for display)
	Timestamp int64  // Unix timestamp in milliseconds
}

// GetLastActivity extracts the last meaningful activity from session messages.
// It looks for the most recent assistant message and extracts a summary of what
// the agent is doing (tool use, text generation, etc.).
// Returns nil if no activity can be extracted.
func (c *Client) GetLastActivity(sessionID string) (*LastActivity, error) {
	messages, err := c.GetMessages(sessionID)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}

	// Find the last assistant message (most relevant for activity)
	var lastAssistantMsg *Message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Info.Role == "assistant" {
			lastAssistantMsg = &messages[i]
			break
		}
	}

	if lastAssistantMsg == nil {
		return nil, nil
	}

	// Extract activity from message parts
	// Priority: tool invocation > text > reasoning
	var activityText string
	for _, part := range lastAssistantMsg.Parts {
		switch part.Type {
		case "tool-invocation", "tool":
			// Tool use is the most informative activity
			activityText = "Using tool: " + extractToolName(part.Text)
			break
		case "text":
			if part.Text != "" && activityText == "" {
				// Truncate long text
				activityText = truncateText(part.Text, 80)
			}
		case "reasoning":
			if activityText == "" {
				activityText = "Thinking..."
			}
		}
	}

	if activityText == "" {
		return nil, nil
	}

	// Use message completion time if available, otherwise created time
	timestamp := lastAssistantMsg.Info.Time.Completed
	if timestamp == 0 {
		timestamp = lastAssistantMsg.Info.Time.Created
	}

	return &LastActivity{
		Text:      activityText,
		Timestamp: timestamp,
	}, nil
}

// extractToolName tries to extract a tool name from tool invocation text.
// Tool text can be JSON or plain text depending on OpenCode version.
func extractToolName(text string) string {
	// Try to extract tool name - the text might be structured or plain
	if text == "" {
		return "unknown"
	}
	// Truncate for display
	if len(text) > 50 {
		return text[:50] + "..."
	}
	return text
}

// truncateText truncates text to maxLen characters with ellipsis.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	// Find last space before maxLen to avoid cutting words
	for i := maxLen - 3; i > 0; i-- {
		if text[i] == ' ' {
			return text[:i] + "..."
		}
	}
	return text[:maxLen-3] + "..."
}

// CreateSessionRequest represents the request body for creating a new session.
type CreateSessionRequest struct {
	Title     string `json:"title,omitempty"`
	Directory string `json:"directory,omitempty"`
	Model     string `json:"model,omitempty"`
}

// CreateSessionResponse represents the response from creating a new session.
type CreateSessionResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	Directory string `json:"directory,omitempty"`
}

// CreateSession creates a new OpenCode session via HTTP API.
// This is used for headless spawns (no tmux window).
// Set isWorker=true for worker spawns to enable ORCH_WORKER detection.
func (c *Client) CreateSession(title, directory, model string, isWorker bool) (*CreateSessionResponse, error) {
	payload := CreateSessionRequest{
		Title:     title,
		Directory: directory,
		Model:     model,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request to set custom headers
	req, err := http.NewRequest("POST", c.ServerURL+"/session", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// OpenCode expects directory via x-opencode-directory header
	if directory != "" {
		req.Header.Set("x-opencode-directory", directory)
	}

	// Set ORCH_WORKER=1 header for worker sessions to signal orch-managed workers
	// This allows the session-context plugin to skip loading orchestrator skill
	if isWorker {
		req.Header.Set("x-opencode-env-ORCH_WORKER", "1")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	var result CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// SendPrompt sends a prompt to a session via HTTP API (async).
// This is used for headless spawns to send the initial prompt.
// The model parameter is optional - if empty, OpenCode will use the default model.
func (c *Client) SendPrompt(sessionID, prompt, model string) error {
	return c.SendMessageAsync(sessionID, prompt, model)
}

// GetMessages fetches all messages for a session from the OpenCode API.
func (c *Client) GetMessages(sessionID string) ([]Message, error) {
	req, err := http.NewRequest("GET", c.ServerURL+"/session/"+sessionID+"/message", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var messages []Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	return messages, nil
}

// FindRecentSession finds the most recent session for a given project directory.
// It matches by directory and creation time only (within 30 seconds).
// Title matching is not used because OpenCode session titles are set to the first
// prompt text, not the workspace name, making title matching unreliable.
func (c *Client) FindRecentSession(projectDir string) (string, error) {
	req, err := http.NewRequest("GET", c.ServerURL+"/session", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-opencode-directory", projectDir)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return "", err
	}

	// Find the most recent session for this directory (within 30 seconds)
	var mostRecent *Session
	now := time.Now().UnixMilli()
	for i := range sessions {
		s := &sessions[i]
		if s.Directory != projectDir {
			continue
		}
		// Only match sessions created in the last 30 seconds
		if now-s.Time.Created > 30*1000 {
			continue
		}
		if mostRecent == nil || s.Time.Created > mostRecent.Time.Created {
			mostRecent = s
		}
	}

	if mostRecent == nil {
		return "", fmt.Errorf("no sessions found for directory: %s (within last 30s)", projectDir)
	}

	return mostRecent.ID, nil
}

// FindRecentSessionWithRetry retries finding a recent session with exponential backoff.
// This handles the race condition where OpenCode TUI starts before registering with the API.
// Returns the session ID if found, or empty string with error if not found after retries.
func (c *Client) FindRecentSessionWithRetry(projectDir string, maxAttempts int, initialDelay time.Duration) (string, error) {
	delay := initialDelay
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		sessionID, err := c.FindRecentSession(projectDir)
		if err == nil {
			return sessionID, nil
		}
		lastErr = err

		// Don't sleep after the last attempt
		if attempt < maxAttempts {
			time.Sleep(delay)
			delay = delay * 2 // Exponential backoff
		}
	}

	// Return empty string (not an error) since tmux window_id is sufficient for monitoring
	return "", lastErr
}

// ExtractRecentText extracts the most recent text content from messages.
// It returns up to `lines` worth of text from the most recent messages.
// The text is extracted from message parts of type "text".
func ExtractRecentText(messages []Message, lines int) []string {
	var result []string

	// Process messages in reverse order (most recent first)
	for i := len(messages) - 1; i >= 0 && len(result) < lines; i-- {
		msg := messages[i]

		// Extract text from parts
		for j := len(msg.Parts) - 1; j >= 0 && len(result) < lines; j-- {
			part := msg.Parts[j]
			if part.Type == "text" && part.Text != "" {
				// Split text into lines and add in reverse
				textLines := strings.Split(part.Text, "\n")
				for k := len(textLines) - 1; k >= 0 && len(result) < lines; k-- {
					line := textLines[k]
					if line != "" || len(result) > 0 { // Skip leading empty lines
						result = append([]string{line}, result...)
					}
				}
			}
		}
	}

	// Trim to requested line count
	if len(result) > lines {
		result = result[len(result)-lines:]
	}

	return result
}

// ListDiskSessions lists all sessions stored on disk for a given directory.
// This differs from ListSessions in that it returns ALL sessions for the directory,
// not just in-memory sessions. The x-opencode-directory header is required.
// Returns the list of sessions and the total count.
func (c *Client) ListDiskSessions(directory string) ([]Session, error) {
	if directory == "" {
		return nil, fmt.Errorf("directory is required for ListDiskSessions")
	}

	req, err := http.NewRequest("GET", c.ServerURL+"/session", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-opencode-directory", directory)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch disk sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return sessions, nil
}

// DeleteSession deletes an OpenCode session by ID.
// Returns nil on success, error on failure.
func (c *Client) DeleteSession(sessionID string) error {
	req, err := http.NewRequest("DELETE", c.ServerURL+"/session/"+sessionID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	defer resp.Body.Close()

	// Accept 200 OK or 204 No Content as success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete session: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendMessageWithStreaming sends a message to a session and streams the response.
// It sends the message via the async API, then connects to SSE to stream text events
// until the session becomes idle. Text content is written to the provided writer.
func (c *Client) SendMessageWithStreaming(sessionID, content string, streamTo io.Writer) error {
	// Send the message via async API first (no model specified for Q&A)
	if err := c.SendMessageAsync(sessionID, content, ""); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Connect to SSE and stream the response
	// Use a client without timeout for SSE - it's a long-running stream
	sseClient := &http.Client{
		// No timeout - SSE is meant to be long-running
		// But still limit redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects (max 10)")
			}
			return nil
		},
	}
	sseURL := c.ServerURL + "/event"
	resp, err := sseClient.Get(sseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE: %w", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	var eventBuffer strings.Builder
	var sessionWasBusy bool
	var messageIDSeen = make(map[string]bool) // Track message IDs to avoid duplicates

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		eventBuffer.WriteString(line)

		// Empty line signals end of event
		if line == "\n" && eventBuffer.Len() > 1 {
			raw := eventBuffer.String()
			eventType, data := ParseSSEEvent(raw)
			eventBuffer.Reset()

			if data == "" {
				continue
			}

			// Parse the JSON data
			var eventData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &eventData); err != nil {
				continue
			}

			// Check if this event is for our session
			eventSessionID := ""
			if props, ok := eventData["properties"].(map[string]interface{}); ok {
				if sid, ok := props["sessionID"].(string); ok {
					eventSessionID = sid
				}
			}
			// Also check top-level sessionID
			if sid, ok := eventData["sessionID"].(string); ok && eventSessionID == "" {
				eventSessionID = sid
			}

			// Skip events from other sessions
			if eventSessionID != "" && eventSessionID != sessionID {
				continue
			}

			// Handle session error events
			if eventType == "session.error" {
				if props, ok := eventData["properties"].(map[string]interface{}); ok {
					// Check if this error is for our session
					if sid, ok := props["sessionID"].(string); ok && sid == sessionID {
						// Extract error details
						if errorObj, ok := props["error"].(map[string]interface{}); ok {
							if msg, ok := errorObj["message"].(string); ok {
								return fmt.Errorf("session error: %s", msg)
							}
						}
						return fmt.Errorf("session error occurred")
					}
				}
				// Also check top-level sessionID for older format
				if sid, ok := eventData["sessionID"].(string); ok && sid == sessionID {
					if errorObj, ok := eventData["error"].(map[string]interface{}); ok {
						if msg, ok := errorObj["message"].(string); ok {
							return fmt.Errorf("session error: %s", msg)
						}
					}
					return fmt.Errorf("session error occurred")
				}
				continue
			}

			// Handle session status events
			if eventType == "session.status" {
				status, sid := ParseSessionStatus(data)
				if sid == sessionID {
					if status == "busy" || status == "running" {
						sessionWasBusy = true
					}
					// Completion: session was busy and is now idle
					if sessionWasBusy && status == "idle" {
						return nil
					}
				}
				continue
			}

			// Handle message.part events (text streaming)
			if eventType == "message.part" {
				if props, ok := eventData["properties"].(map[string]interface{}); ok {
					// Check this is for our session
					if sid, ok := props["sessionID"].(string); ok && sid != sessionID {
						continue
					}

					// Get message ID to track what we've seen
					messageID := ""
					if mid, ok := props["messageID"].(string); ok {
						messageID = mid
					}

					// Get the part data
					if part, ok := props["part"].(map[string]interface{}); ok {
						if partType, ok := part["type"].(string); ok && partType == "text" {
							if text, ok := part["text"].(string); ok && text != "" {
								// Write text to output
								streamTo.Write([]byte(text))
							}
						}
					}

					// Track message ID if provided
					if messageID != "" {
						messageIDSeen[messageID] = true
					}
				}
			}
		}
	}
}

// TokenStats represents aggregated token usage for a session.
type TokenStats struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	CacheReadTokens int `json:"cache_read_tokens,omitempty"`
	TotalTokens     int `json:"total_tokens"` // input + output + reasoning
}

// AggregateTokens calculates total token usage from a slice of messages.
// It sums up input, output, reasoning, and cache tokens across all messages.
func AggregateTokens(messages []Message) TokenStats {
	var stats TokenStats
	for _, msg := range messages {
		if msg.Info.Tokens == nil {
			continue
		}
		stats.InputTokens += msg.Info.Tokens.Input
		stats.OutputTokens += msg.Info.Tokens.Output
		stats.ReasoningTokens += msg.Info.Tokens.Reasoning
		if msg.Info.Tokens.Cache != nil {
			stats.CacheReadTokens += msg.Info.Tokens.Cache.Read
		}
	}
	stats.TotalTokens = stats.InputTokens + stats.OutputTokens + stats.ReasoningTokens
	return stats
}

// GetSessionTokens fetches messages for a session and returns aggregated token stats.
// Returns nil if session doesn't exist or has no messages.
func (c *Client) GetSessionTokens(sessionID string) (*TokenStats, error) {
	messages, err := c.GetMessages(sessionID)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	stats := AggregateTokens(messages)
	return &stats, nil
}

// ExportSessionTranscript fetches all messages for a session and formats them as markdown.
// This is useful for preserving conversation history before deleting a session.
// Returns the markdown transcript and any error encountered.
func (c *Client) ExportSessionTranscript(sessionID string) (string, error) {
	// Get session info
	session, err := c.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	// Get all messages
	messages, err := c.GetMessages(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}

	if len(messages) == 0 {
		return "", nil // No messages to export
	}

	return FormatMessagesAsTranscript(session, messages), nil
}

// FormatMessagesAsTranscript converts session info and messages to a markdown transcript.
func FormatMessagesAsTranscript(session *Session, messages []Message) string {
	var lines []string

	// Header
	lines = append(lines, "# Session Transcript", "")

	// Session metadata
	lines = append(lines, fmt.Sprintf("**Title:** %s", session.Title))
	lines = append(lines, fmt.Sprintf("**Session ID:** `%s`", session.ID))
	if session.Directory != "" {
		lines = append(lines, fmt.Sprintf("**Directory:** `%s`", session.Directory))
	}
	if session.Time.Created > 0 {
		t := time.Unix(session.Time.Created/1000, 0)
		lines = append(lines, fmt.Sprintf("**Started:** %s", t.Format("2006-01-02 15:04:05")))
	}
	if session.Time.Updated > 0 {
		t := time.Unix(session.Time.Updated/1000, 0)
		lines = append(lines, fmt.Sprintf("**Updated:** %s", t.Format("2006-01-02 15:04:05")))
	}

	// Summary stats
	if session.Summary.Additions > 0 || session.Summary.Deletions > 0 || session.Summary.Files > 0 {
		lines = append(lines, fmt.Sprintf("**Changes:** +%d/-%d in %d files",
			session.Summary.Additions, session.Summary.Deletions, session.Summary.Files))
	}

	lines = append(lines, "", "---", "")

	// Format messages
	for _, msg := range messages {
		formatted := formatMessageToMarkdown(&msg)
		if formatted != "" {
			lines = append(lines, formatted)
		}
	}

	return strings.Join(lines, "\n")
}

// formatMessageToMarkdown formats a single message to markdown.
func formatMessageToMarkdown(msg *Message) string {
	// Collect text parts and tool parts
	var textParts []string
	var toolParts []MessagePart

	for _, part := range msg.Parts {
		switch part.Type {
		case "text":
			text := strings.TrimSpace(part.Text)
			if text != "" {
				textParts = append(textParts, text)
			}
		case "tool", "tool-invocation":
			toolParts = append(toolParts, part)
		}
	}

	// Skip message if no content
	if len(textParts) == 0 && len(toolParts) == 0 {
		return ""
	}

	var lines []string

	// Header with role and timestamp
	var timestamp string
	if msg.Info.Time.Created > 0 {
		t := time.Unix(msg.Info.Time.Created/1000, 0)
		timestamp = t.Format("2006-01-02 15:04:05")
	}

	switch msg.Info.Role {
	case "user":
		lines = append(lines, fmt.Sprintf("## User (%s)", timestamp))
	case "assistant":
		lines = append(lines, fmt.Sprintf("## Assistant (%s)", timestamp))
		// Add token/cost info
		if msg.Info.Tokens != nil {
			var tokenInfo []string
			if msg.Info.Tokens.Input > 0 {
				tokenInfo = append(tokenInfo, fmt.Sprintf("in:%d", msg.Info.Tokens.Input))
			}
			if msg.Info.Tokens.Output > 0 {
				tokenInfo = append(tokenInfo, fmt.Sprintf("out:%d", msg.Info.Tokens.Output))
			}
			if msg.Info.Tokens.Cache != nil && msg.Info.Tokens.Cache.Read > 0 {
				tokenInfo = append(tokenInfo, fmt.Sprintf("cached:%d", msg.Info.Tokens.Cache.Read))
			}
			if msg.Info.Cost > 0 {
				tokenInfo = append(tokenInfo, fmt.Sprintf("$%.4f", msg.Info.Cost))
			}
			if len(tokenInfo) > 0 {
				lines = append(lines, fmt.Sprintf("*Tokens: %s*", strings.Join(tokenInfo, ", ")))
			}
		}
	default:
		role := msg.Info.Role
		if role != "" {
			role = strings.ToUpper(role[:1]) + role[1:]
		}
		lines = append(lines, fmt.Sprintf("## %s (%s)", role, timestamp))
	}

	lines = append(lines, "")

	// Add text content
	for _, text := range textParts {
		lines = append(lines, text, "")
	}

	// Add tool summaries
	if len(toolParts) > 0 {
		lines = append(lines, "**Tools:**")
		for _, tool := range toolParts {
			// Format tool details for better debugging
			toolDesc := formatToolDescription(&tool)
			lines = append(lines, fmt.Sprintf("  - %s", toolDesc))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// formatToolDescription formats a tool part into a human-readable string.
// Shows tool name, title/description, and key parameters for debugging.
func formatToolDescription(tool *MessagePart) string {
	// If no tool details available, fall back to type
	if tool.Tool == "" {
		return tool.Type
	}

	// Start with tool name
	result := tool.Tool

	// Add title/description if available (most useful for bash commands)
	if tool.State != nil && tool.State.Title != "" {
		result = fmt.Sprintf("%s: %s", result, tool.State.Title)
	} else if tool.State != nil && len(tool.State.Input) > 0 {
		// If no title, try to extract a useful parameter
		// For common tools, show the most relevant parameter
		switch tool.Tool {
		case "read":
			if filePath, ok := tool.State.Input["filePath"].(string); ok {
				// Show just filename, not full path, to keep it concise
				filename := filePath
				if idx := strings.LastIndex(filePath, "/"); idx >= 0 && idx < len(filePath)-1 {
					filename = filePath[idx+1:]
				}
				result = fmt.Sprintf("%s: %s", result, filename)
			}
		case "edit", "write":
			if filePath, ok := tool.State.Input["filePath"].(string); ok {
				filename := filePath
				if idx := strings.LastIndex(filePath, "/"); idx >= 0 && idx < len(filePath)-1 {
					filename = filePath[idx+1:]
				}
				result = fmt.Sprintf("%s: %s", result, filename)
			}
		case "bash":
			if command, ok := tool.State.Input["command"].(string); ok {
				// Truncate long commands to 60 chars
				if len(command) > 60 {
					command = command[:57] + "..."
				}
				result = fmt.Sprintf("%s: %s", result, command)
			}
		case "grep", "glob":
			if pattern, ok := tool.State.Input["pattern"].(string); ok {
				result = fmt.Sprintf("%s: %s", result, pattern)
			}
		default:
			// For other tools, just show the tool name
		}
	}

	return result
}
