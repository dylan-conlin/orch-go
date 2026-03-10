package opencode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

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
