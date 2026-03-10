package opencode

import (
	"fmt"
	"strings"
	"time"
)

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
