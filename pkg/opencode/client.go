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
func (c *Client) BuildSpawnCommand(prompt, title string) *exec.Cmd {
	args := []string{
		"run",
		"--attach", c.ServerURL,
		"--format", "json",
		"--title", title,
		prompt,
	}
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
func (c *Client) SendMessageAsync(sessionID, content string) error {
	payload := map[string]any{
		"parts": []map[string]string{{"type": "text", "text": content}},
		"agent": "build",
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
func (c *Client) ListSessions() ([]Session, error) {
	resp, err := http.Get(c.ServerURL + "/session")
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

// CreateSessionRequest represents the request body for creating a new session.
type CreateSessionRequest struct {
	Title     string `json:"title,omitempty"`
	Directory string `json:"directory,omitempty"`
}

// CreateSessionResponse represents the response from creating a new session.
type CreateSessionResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	Directory string `json:"directory,omitempty"`
}

// CreateSession creates a new OpenCode session via HTTP API.
// This is used for headless spawns (no tmux window).
func (c *Client) CreateSession(title, directory string) (*CreateSessionResponse, error) {
	payload := CreateSessionRequest{
		Title:     title,
		Directory: directory,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(c.ServerURL+"/session", "application/json", bytes.NewReader(body))
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
func (c *Client) SendPrompt(sessionID, prompt string) error {
	return c.SendMessageAsync(sessionID, prompt)
}
