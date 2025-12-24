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

// Client handles OpenCode CLI interactions.
type Client struct {
	ServerURL string
}

// NewClient creates a new OpenCode client.
func NewClient(serverURL string) *Client {
	return &Client{ServerURL: serverURL}
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

// ProcessOutput processes the output from opencode command.
func ProcessOutput(r io.Reader) (*Result, error) {
	result := &Result{}
	scanner := bufio.NewScanner(r)

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
func (c *Client) SendMessageAsync(sessionID, content, model string) error {
	payload := map[string]any{
		"parts": []map[string]string{{"type": "text", "text": content}},
		"agent": "build",
	}
	// Include model if provided (per-message model in OpenCode)
	if model != "" {
		payload["model"] = model
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(c.ServerURL+"/session/"+sessionID+"/prompt_async", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
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

	resp, err := http.DefaultClient.Do(req)
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
	resp, err := http.Get(c.ServerURL + "/session/" + sessionID)
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
	resp, err := http.Get(c.ServerURL + "/session/" + sessionID)
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
func (c *Client) CreateSession(title, directory, model string) (*CreateSessionResponse, error) {
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

	resp, err := http.DefaultClient.Do(req)
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
	resp, err := http.Get(c.ServerURL + "/session/" + sessionID + "/message")
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

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
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
	sseURL := c.ServerURL + "/event"
	resp, err := http.Get(sseURL)
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
