package opencode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// BuildSpawnCommand builds the opencode spawn command.
func (c *Client) BuildSpawnCommand(prompt, title string) *exec.Cmd {
	args := []string{
		"run",
		"--attach", c.ServerURL,
		"--format", "json",
		"--title", title,
		prompt,
	}
	return exec.Command("opencode", args...)
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
	return exec.Command("opencode", args...)
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
