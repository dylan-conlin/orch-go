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
	URL        string
	httpClient *http.Client
}

// NewSSEClient creates a new SSE client.
// SSE clients have no timeout since they're meant for long-running streams,
// but do have redirect limiting to prevent redirect loops from hanging.
func NewSSEClient(url string) *SSEClient {
	return &SSEClient{
		URL: url,
		httpClient: &http.Client{
			// No timeout - SSE is meant to be long-running
			// But limit redirects to prevent redirect loops
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects (max 10)")
				}
				return nil
			},
		},
	}
}

// Connect establishes SSE connection and sends events to channel.
// This is a blocking call that reads events until the connection is closed.
func (c *SSEClient) Connect(events chan<- SSEEvent) error {
	resp, err := c.httpClient.Get(c.URL)
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
	// If event type not in SSE prefix, try to extract from JSON data
	if eventType == "" && data != "" {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(data), &result); err == nil {
			if typ, ok := result["type"].(string); ok {
				eventType = typ
			}
		}
	}
	return eventType, data
}

// ParseSessionStatus extracts status and session ID from SSE data.
func ParseSessionStatus(data string) (status string, sessionID string) {
	// Try old format: {"status":"idle","session_id":"..."}
	var old struct {
		Status    string `json:"status"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(data), &old); err == nil && old.Status != "" && old.SessionID != "" {
		return old.Status, old.SessionID
	}
	// Try new format: {"type":"session.status","properties":{"sessionID":"...","status":{"type":"..."}}}
	var new struct {
		Type       string `json:"type"`
		Properties struct {
			SessionID string `json:"sessionID"`
			Status    struct {
				Type string `json:"type"`
			} `json:"status"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(data), &new); err == nil && new.Type == "session.status" && new.Properties.SessionID != "" && new.Properties.Status.Type != "" {
		return new.Properties.Status.Type, new.Properties.SessionID
	}
	// Fallback: parse as map to extract any known fields
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err == nil {
		// Old format fields
		if s, ok := m["status"].(string); ok && s != "" {
			status = s
		}
		if sid, ok := m["session_id"].(string); ok && sid != "" {
			sessionID = sid
		}
		if status != "" && sessionID != "" {
			return status, sessionID
		}
		// New format fields
		if props, ok := m["properties"].(map[string]interface{}); ok {
			if sid, ok := props["sessionID"].(string); ok && sid != "" {
				sessionID = sid
			}
			if statusObj, ok := props["status"].(map[string]interface{}); ok {
				if s, ok := statusObj["type"].(string); ok && s != "" {
					status = s
				}
			}
		}
	}
	return status, sessionID
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

// ParseSessionError extracts error details from a session.error event.
// Returns sessionID and error message if found.
func ParseSessionError(data string) (sessionID string, errMsg string) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return "", ""
	}

	// Try new format: {"type":"session.error","properties":{"sessionID":"...","error":{"message":"..."}}}
	if props, ok := m["properties"].(map[string]interface{}); ok {
		if sid, ok := props["sessionID"].(string); ok {
			sessionID = sid
		}
		if errorObj, ok := props["error"].(map[string]interface{}); ok {
			if msg, ok := errorObj["message"].(string); ok {
				errMsg = msg
			}
		}
	}

	// Fallback: check top-level fields
	if sessionID == "" {
		if sid, ok := m["sessionID"].(string); ok {
			sessionID = sid
		}
	}
	if errMsg == "" {
		if errorObj, ok := m["error"].(map[string]interface{}); ok {
			if msg, ok := errorObj["message"].(string); ok {
				errMsg = msg
			}
		}
	}

	return sessionID, errMsg
}
