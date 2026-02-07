package opencode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// ParseEvent parses a JSON event from opencode output.
func ParseEvent(line string) (Event, error) {
	var event Event
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return event, err
	}
	return event, nil
}

// ExtractSessionID extracts session ID from opencode events.
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
func ExtractSessionIDFromReader(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, LargeScannerBufferSize), LargeScannerBufferSize)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if sessionID := findSessionIDInLine(line); sessionID != "" {
			return sessionID, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanner error: %w", err)
	}
	return "", ErrNoSessionID
}

// findSessionIDInLine attempts to extract a session ID from a line that may contain mixed content.
func findSessionIDInLine(line string) string {
	event, err := ParseEvent(line)
	if err == nil && event.SessionID != "" {
		return event.SessionID
	}
	for i := 0; i < len(line); i++ {
		if line[i] == '{' {
			event, err := ParseEvent(line[i:])
			if err == nil && event.SessionID != "" {
				return event.SessionID
			}
		}
	}
	return ""
}

// ProcessOutput processes the output from opencode command.
func ProcessOutput(r io.Reader) (*Result, error) {
	result := &Result{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, LargeScannerBufferSize), LargeScannerBufferSize)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		event, err := ParseEvent(line)
		if err != nil {
			continue
		}
		result.Events = append(result.Events, event)
		if result.SessionID == "" && event.SessionID != "" {
			result.SessionID = event.SessionID
		}
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}

// ProcessOutputWithStreaming processes the output from opencode command and streams text content.
func ProcessOutputWithStreaming(r io.Reader, streamTo io.Writer) (*Result, error) {
	result := &Result{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, LargeScannerBufferSize), LargeScannerBufferSize)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		event, err := ParseEvent(line)
		if err != nil {
			continue
		}
		result.Events = append(result.Events, event)
		if result.SessionID == "" && event.SessionID != "" {
			result.SessionID = event.SessionID
		}
		if event.Type == "text" && event.Content != "" {
			streamTo.Write([]byte(event.Content))
		}
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}
