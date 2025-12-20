package opencode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SSEClient handles SSE connections to OpenCode.
type SSEClient struct {
	URL string
}

// NewSSEClient creates a new SSE client.
func NewSSEClient(url string) *SSEClient {
	return &SSEClient{URL: url}
}

// Connect establishes SSE connection and sends events to channel.
// This is a blocking call that reads events until the connection is closed.
func (c *SSEClient) Connect(events chan<- SSEEvent) error {
	resp, err := http.Get(c.URL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	return ReadSSEStream(resp.Body, events)
}

// ReadSSEStream reads SSE events from a reader and sends them to a channel.
// This is useful for testing with mock readers.
func ReadSSEStream(r io.Reader, events chan<- SSEEvent) error {
	reader := bufio.NewReader(r)
	var eventBuffer strings.Builder

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
			if eventType != "" {
				events <- SSEEvent{Event: eventType, Data: data}
			}
			eventBuffer.Reset()
		}
	}
}

// ParseSSEEvent parses an SSE formatted event.
// Returns the event type and data.
func ParseSSEEvent(raw string) (eventType string, data string) {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		}
	}
	return eventType, data
}

// ParseSessionStatus extracts status and session ID from SSE data.
func ParseSessionStatus(data string) (status string, sessionID string) {
	var parsed SessionStatus
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return "", ""
	}
	return parsed.Status, parsed.SessionID
}

// DetectCompletion checks if events indicate session completion.
// A session is complete when it transitions from busy to idle.
func DetectCompletion(events []SSEEvent) (sessionID string, completed bool) {
	var lastStatus string
	var lastSessionID string

	for _, event := range events {
		if event.Event == "session.status" {
			status, sid := ParseSessionStatus(event.Data)
			if sid != "" {
				lastSessionID = sid
			}
			lastStatus = status
		}
	}

	// Session is complete when it transitions to idle
	if lastStatus == "idle" && lastSessionID != "" {
		return lastSessionID, true
	}

	return lastSessionID, false
}
