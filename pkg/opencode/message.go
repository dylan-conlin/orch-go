package opencode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TokenStats represents aggregated token usage for a session.
type TokenStats struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	CacheReadTokens int `json:"cache_read_tokens,omitempty"`
	TotalTokens     int `json:"total_tokens"`
}

// LastActivity represents the most recent activity from a session.
type LastActivity struct {
	Text      string
	Timestamp int64
}

// SessionEnrichment contains model, processing status, and token stats.
type SessionEnrichment struct {
	Model        string
	IsProcessing bool
	Tokens       *TokenStats
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

// GetLastMessage returns the last message in a session.
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

// GetSessionEnrichment fetches messages once and extracts model, processing status, and tokens.
func (c *Client) GetSessionEnrichment(sessionID string) SessionEnrichment {
	messages, err := c.GetMessages(sessionID)
	if err != nil || len(messages) == 0 {
		return SessionEnrichment{}
	}
	var result SessionEnrichment
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Info.Role == "assistant" && messages[i].Info.ModelID != "" {
			result.Model = messages[i].Info.ModelID
			break
		}
	}
	lastMsg := messages[len(messages)-1]
	if lastMsg.Info.Role == "assistant" {
		result.IsProcessing = lastMsg.Info.Finish == "" && lastMsg.Info.Time.Completed == 0
	} else if lastMsg.Info.Role == "user" {
		createdAt := time.Unix(lastMsg.Info.Time.Created/1000, 0)
		result.IsProcessing = time.Since(createdAt) < 30*time.Second
	}
	stats := AggregateTokens(messages)
	result.Tokens = &stats
	return result
}

// GetSessionModel extracts the model ID from a session's most recent assistant message.
func (c *Client) GetSessionModel(sessionID string) string {
	messages, err := c.GetMessages(sessionID)
	if err != nil || len(messages) == 0 {
		return ""
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Info.Role == "assistant" && messages[i].Info.ModelID != "" {
			return messages[i].Info.ModelID
		}
	}
	return ""
}

// GetSessionTokens fetches messages for a session and returns aggregated token stats.
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

// AggregateTokens calculates total token usage from a slice of messages.
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

// GetLastActivity extracts the last meaningful activity from session messages.
func (c *Client) GetLastActivity(sessionID string) (*LastActivity, error) {
	messages, err := c.GetMessages(sessionID)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
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
	var activityText string
	for _, part := range lastAssistantMsg.Parts {
		switch part.Type {
		case "tool-invocation", "tool":
			activityText = "Using tool: " + extractToolName(part.Text)
			break
		case "text":
			if part.Text != "" && activityText == "" {
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
	timestamp := lastAssistantMsg.Info.Time.Completed
	if timestamp == 0 {
		timestamp = lastAssistantMsg.Info.Time.Created
	}
	return &LastActivity{Text: activityText, Timestamp: timestamp}, nil
}

func extractToolName(text string) string {
	if text == "" {
		return "unknown"
	}
	if len(text) > 50 {
		return text[:50] + "..."
	}
	return text
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	for i := maxLen - 3; i > 0; i-- {
		if text[i] == ' ' {
			return text[:i] + "..."
		}
	}
	return text[:maxLen-3] + "..."
}

// ExtractRecentText extracts the most recent text content from messages.
func ExtractRecentText(messages []Message, lines int) []string {
	var result []string
	for i := len(messages) - 1; i >= 0 && len(result) < lines; i-- {
		msg := messages[i]
		for j := len(msg.Parts) - 1; j >= 0 && len(result) < lines; j-- {
			part := msg.Parts[j]
			if part.Type == "text" && part.Text != "" {
				textLines := strings.Split(part.Text, "\n")
				for k := len(textLines) - 1; k >= 0 && len(result) < lines; k-- {
					line := textLines[k]
					if line != "" || len(result) > 0 {
						result = append([]string{line}, result...)
					}
				}
			}
		}
	}
	if len(result) > lines {
		result = result[len(result)-lines:]
	}
	return result
}

// SendMessageAsync sends a message to an existing session asynchronously.
func (c *Client) SendMessageAsync(sessionID, content, model string) error {
	payload := map[string]any{
		"parts": []map[string]string{{"type": "text", "text": content}},
		"agent": "build",
	}
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

// SendPrompt sends a prompt to a session via HTTP API (async).
func (c *Client) SendPrompt(sessionID, prompt, model string) error {
	return c.SendMessageAsync(sessionID, prompt, model)
}

// SendMessageWithStreaming sends a message to a session and streams the response.
func (c *Client) SendMessageWithStreaming(sessionID, content string, streamTo io.Writer) error {
	if err := c.SendMessageAsync(sessionID, content, ""); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	sseClient := &http.Client{
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
	var messageIDSeen = make(map[string]bool)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		eventBuffer.WriteString(line)
		if line == "\n" && eventBuffer.Len() > 1 {
			raw := eventBuffer.String()
			eventType, data := ParseSSEEvent(raw)
			eventBuffer.Reset()
			if data == "" {
				continue
			}
			var eventData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &eventData); err != nil {
				continue
			}
			eventSessionID := ""
			if props, ok := eventData["properties"].(map[string]interface{}); ok {
				if sid, ok := props["sessionID"].(string); ok {
					eventSessionID = sid
				}
			}
			if sid, ok := eventData["sessionID"].(string); ok && eventSessionID == "" {
				eventSessionID = sid
			}
			if eventSessionID != "" && eventSessionID != sessionID {
				continue
			}
			if eventType == "session.error" {
				if props, ok := eventData["properties"].(map[string]interface{}); ok {
					if sid, ok := props["sessionID"].(string); ok && sid == sessionID {
						if errorObj, ok := props["error"].(map[string]interface{}); ok {
							if msg, ok := errorObj["message"].(string); ok {
								return fmt.Errorf("session error: %s", msg)
							}
						}
						return fmt.Errorf("session error occurred")
					}
				}
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
			if eventType == "session.status" {
				status, sid := ParseSessionStatus(data)
				if sid == sessionID {
					if status == "busy" || status == "running" {
						sessionWasBusy = true
					}
					if sessionWasBusy && status == "idle" {
						return nil
					}
				}
				continue
			}
			if eventType == "message.part" {
				if props, ok := eventData["properties"].(map[string]interface{}); ok {
					if sid, ok := props["sessionID"].(string); ok && sid != sessionID {
						continue
					}
					messageID := ""
					if mid, ok := props["messageID"].(string); ok {
						messageID = mid
					}
					if part, ok := props["part"].(map[string]interface{}); ok {
						if partType, ok := part["type"].(string); ok && partType == "text" {
							if text, ok := part["text"].(string); ok && text != "" {
								streamTo.Write([]byte(text))
							}
						}
					}
					if messageID != "" {
						messageIDSeen[messageID] = true
					}
				}
			}
		}
	}
}

// ExportSessionTranscript fetches all messages for a session and formats them as markdown.
func (c *Client) ExportSessionTranscript(sessionID string) (string, error) {
	session, err := c.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}
	messages, err := c.GetMessages(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}
	if len(messages) == 0 {
		return "", nil
	}
	return FormatMessagesAsTranscript(session, messages), nil
}

// FormatMessagesAsTranscript converts session info and messages to a markdown transcript.
func FormatMessagesAsTranscript(session *Session, messages []Message) string {
	var lines []string
	lines = append(lines, "# Session Transcript", "")
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
	if session.Summary.Additions > 0 || session.Summary.Deletions > 0 || session.Summary.Files > 0 {
		lines = append(lines, fmt.Sprintf("**Changes:** +%d/-%d in %d files",
			session.Summary.Additions, session.Summary.Deletions, session.Summary.Files))
	}
	lines = append(lines, "", "---", "")
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
	if len(textParts) == 0 && len(toolParts) == 0 {
		return ""
	}
	var lines []string
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
	for _, text := range textParts {
		lines = append(lines, text, "")
	}
	if len(toolParts) > 0 {
		lines = append(lines, "**Tools:**")
		for _, tool := range toolParts {
			toolDesc := formatToolDescription(&tool)
			lines = append(lines, fmt.Sprintf("  - %s", toolDesc))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// formatToolDescription formats a tool part into a human-readable string.
func formatToolDescription(tool *MessagePart) string {
	if tool.Tool == "" {
		return tool.Type
	}
	result := tool.Tool
	if tool.State != nil && tool.State.Title != "" {
		result = fmt.Sprintf("%s: %s", result, tool.State.Title)
	} else if tool.State != nil && len(tool.State.Input) > 0 {
		switch tool.Tool {
		case "read":
			if filePath, ok := tool.State.Input["filePath"].(string); ok {
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
				if len(command) > 60 {
					command = command[:57] + "..."
				}
				result = fmt.Sprintf("%s: %s", result, command)
			}
		case "grep", "glob":
			if pattern, ok := tool.State.Input["pattern"].(string); ok {
				result = fmt.Sprintf("%s: %s", result, pattern)
			}
		}
	}
	return result
}
