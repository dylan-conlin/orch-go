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
	Directory  string // Project directory for x-opencode-directory header
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

// NewClientWithDirectory creates a new OpenCode client with a specific project directory.
// The directory is used for the x-opencode-directory header in all API calls.
func NewClientWithDirectory(serverURL, directory string) *Client {
	c := NewClient(serverURL)
	c.Directory = directory
	return c
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

// setDirectoryHeader adds the x-opencode-directory header to a request.
// This header tells OpenCode which project directory to use for the session.
// It's required for all API calls to ensure sessions are stored in the correct location.
func (c *Client) setDirectoryHeader(req *http.Request) {
	if c.Directory != "" {
		req.Header.Set("x-opencode-directory", c.Directory)
	}
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
func ExtractSessionIDFromReader(r io.Reader) (string, error) {
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
		if event.SessionID != "" {
			return event.SessionID, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanner error: %w", err)
	}
	return "", ErrNoSessionID
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
	c.setDirectoryHeader(req)
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
	c.setDirectoryHeader(req)
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
	c.setDirectoryHeader(req)
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

// StaleSessionThreshold is the maximum time since last session update before
// a session is considered "stale" (dead/zombie). Active agents update their
// session state constantly (every tool call, every message part). A session
// with no updates for 3 minutes is effectively dead.
const StaleSessionThreshold = 3 * time.Minute

// IsSessionProcessing checks if a session is actively processing (has a pending assistant response).
// This is the most reliable signal for detecting truly active agents because it checks:
// 1. Whether the session has recent activity (within StaleSessionThreshold)
// 2. Whether the last assistant message has finished (finish != "" and completed != 0)
//
// Returns true only if the session is both recently active AND has an incomplete assistant message.
// Returns false if the session is stale (no updates in 3 minutes) even if the last message
// appears incomplete - this handles zombie sessions that were killed mid-execution.
func (c *Client) IsSessionProcessing(sessionID string) bool {
	// First check if the session is stale - if so, it cannot be processing
	// This handles zombie sessions that have pending tool calls but are dead
	session, err := c.GetSession(sessionID)
	if err != nil {
		return false
	}
	updatedAt := time.Unix(session.Time.Updated/1000, 0)
	if time.Since(updatedAt) > StaleSessionThreshold {
		return false // Stale session - no activity in 3 minutes means dead
	}

	messages, err := c.GetMessages(sessionID)
	if err != nil || len(messages) == 0 {
		return false
	}

	// Find the last message
	lastMsg := messages[len(messages)-1]

	// Session is processing if:
	// 1. Session is not stale (checked above) AND
	// 2. Last message is from assistant AND
	// 3. It hasn't finished yet (finish is empty and completed is 0)
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

// CreateSessionOptions contains optional configuration for CreateSession.
type CreateSessionOptions struct {
	// MCPConfigContent is JSON config content for enabling MCP servers.
	// Passed via x-opencode-env-OPENCODE_CONFIG_CONTENT header.
	MCPConfigContent string
}

// CreateSession creates a new OpenCode session via HTTP API.
// This is used for headless spawns (no tmux window).
func (c *Client) CreateSession(title, directory, model string) (*CreateSessionResponse, error) {
	return c.CreateSessionWithOptions(title, directory, model, nil)
}

// CreateSessionWithOptions creates a new OpenCode session with additional options.
// This allows enabling MCP servers via MCPConfigContent.
func (c *Client) CreateSessionWithOptions(title, directory, model string, opts *CreateSessionOptions) (*CreateSessionResponse, error) {
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

	// Set ORCH_WORKER=1 header to signal this is an orch-managed worker session
	// This allows the session-context plugin to skip loading orchestrator skill
	req.Header.Set("x-opencode-env-ORCH_WORKER", "1")

	// Set MCP config content if provided
	// This enables specific MCP servers for this session
	if opts != nil && opts.MCPConfigContent != "" {
		req.Header.Set("x-opencode-env-OPENCODE_CONFIG_CONTENT", opts.MCPConfigContent)
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
	c.setDirectoryHeader(req)
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

// FindRecentSession finds the most recent session for a given project directory and title.
func (c *Client) FindRecentSession(projectDir, title string) (string, error) {
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

	// Find the most recent session for this directory and title
	var mostRecent *Session
	now := time.Now().UnixMilli()
	for i := range sessions {
		s := &sessions[i]
		if s.Directory != projectDir {
			continue
		}
		// If title is provided, match it
		if title != "" && s.Title != title {
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
		return "", fmt.Errorf("no sessions found for directory: %s (title: %s)", projectDir, title)
	}

	return mostRecent.ID, nil
}

// FindRecentSessionWithRetry retries finding a recent session with exponential backoff.
// This handles the race condition where OpenCode TUI starts before registering with the API.
// Returns the session ID if found, or empty string with no error if not found after retries.
func (c *Client) FindRecentSessionWithRetry(projectDir, title string, maxAttempts int, initialDelay time.Duration) (string, error) {
	delay := initialDelay
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		sessionID, err := c.FindRecentSession(projectDir, title)
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
	c.setDirectoryHeader(req)

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

// WaitForMessage polls GetMessages until the session has at least one message.
// This is used to verify that a prompt was actually delivered after SendPrompt,
// addressing the race condition where SendPrompt returns 200 but the message
// isn't actually processed by the session.
//
// Returns nil if a message is found within the timeout.
// Returns ErrMessageDeliveryTimeout if the timeout is exceeded.
// The interval parameter controls how often to poll (recommended: 200-500ms).
func (c *Client) WaitForMessage(sessionID string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		messages, err := c.GetMessages(sessionID)
		if err != nil {
			// Transient error - keep trying
			time.Sleep(interval)
			continue
		}

		if len(messages) > 0 {
			return nil
		}

		time.Sleep(interval)
	}

	return ErrMessageDeliveryTimeout
}

// SendPromptWithVerification sends a prompt and verifies it was delivered.
// This combines SendPrompt with WaitForMessage to provide reliable message delivery.
// If the initial send doesn't result in a message within the timeout, it retries once.
//
// Parameters:
// - sessionID: The session to send the prompt to
// - prompt: The message content
// - model: Optional model specification (can be empty)
// - timeout: How long to wait for message verification (recommended: 5s)
// - interval: How often to poll for the message (recommended: 300ms)
//
// Returns nil on success, or an error if delivery fails after retry.
func (c *Client) SendPromptWithVerification(sessionID, prompt, model string, timeout, interval time.Duration) error {
	// First attempt
	if err := c.SendPrompt(sessionID, prompt, model); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}

	// Wait for message to appear
	if err := c.WaitForMessage(sessionID, timeout, interval); err == nil {
		return nil // Success!
	}

	// First attempt timed out - retry once
	// The session may have been in a transient state during first send
	if err := c.SendPrompt(sessionID, prompt, model); err != nil {
		return fmt.Errorf("retry send failed: %w", err)
	}

	// Wait again after retry
	if err := c.WaitForMessage(sessionID, timeout, interval); err != nil {
		return fmt.Errorf("message delivery failed after retry: %w", err)
	}

	return nil
}
