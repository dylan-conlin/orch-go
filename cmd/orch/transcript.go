// transcript.go - Format OpenCode session transcripts
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	// Transcript command flags
	transcriptOutput      string
	transcriptToolOutput  bool
	transcriptMaxMessages int
	transcriptStdout      bool
)

var transcriptCmd = &cobra.Command{
	Use:   "transcript",
	Short: "Work with session transcripts",
	Long: `Work with OpenCode session transcripts.

Commands for formatting and managing transcripts.`,
}

var transcriptFormatCmd = &cobra.Command{
	Use:   "format <input-file>",
	Short: "Convert OpenCode JSON export to readable markdown",
	Long: `Convert OpenCode JSON export to readable markdown.

Takes an OpenCode session export JSON file and converts it to a
human-readable markdown format showing:
- User messages
- Assistant responses (text only)
- Tool calls (summarized)
- Timestamps and token usage

Examples:
  orch transcript format session-transcript.json
  orch transcript format session.json -o readable.md
  orch transcript format session.json --tool-output --stdout`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTranscriptFormat(args[0])
	},
}

func init() {
	transcriptFormatCmd.Flags().StringVarP(&transcriptOutput, "output", "o", "", "Output markdown file path")
	transcriptFormatCmd.Flags().BoolVar(&transcriptToolOutput, "tool-output", false, "Include truncated tool outputs in summary")
	transcriptFormatCmd.Flags().IntVar(&transcriptMaxMessages, "max-messages", 0, "Limit number of messages to include")
	transcriptFormatCmd.Flags().BoolVar(&transcriptStdout, "stdout", false, "Output to stdout instead of file")

	transcriptCmd.AddCommand(transcriptFormatCmd)
	rootCmd.AddCommand(transcriptCmd)
}

// TranscriptData represents an OpenCode session export.
type TranscriptData struct {
	Info     TranscriptInfo      `json:"info"`
	Messages []TranscriptMessage `json:"messages"`
}

// TranscriptInfo contains session metadata.
type TranscriptInfo struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Directory string            `json:"directory"`
	Time      TranscriptTime    `json:"time"`
	Summary   TranscriptSummary `json:"summary"`
}

// TranscriptTime contains timing information.
type TranscriptTime struct {
	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
}

// TranscriptSummary contains session summary stats.
type TranscriptSummary struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Files     int `json:"files"`
}

// TranscriptMessage represents a message in the transcript.
type TranscriptMessage struct {
	Info  TranscriptMessageInfo `json:"info"`
	Parts []TranscriptPart      `json:"parts"`
}

// TranscriptMessageInfo contains message metadata.
type TranscriptMessageInfo struct {
	Role   string           `json:"role"`
	Time   TranscriptTime   `json:"time"`
	Tokens TranscriptTokens `json:"tokens"`
	Cost   float64          `json:"cost"`
}

// TranscriptTokens contains token usage information.
type TranscriptTokens struct {
	Input  int                 `json:"input"`
	Output int                 `json:"output"`
	Cache  TranscriptCacheInfo `json:"cache"`
}

// TranscriptCacheInfo contains cache statistics.
type TranscriptCacheInfo struct {
	Read int `json:"read"`
}

// TranscriptPart represents a part of a message.
type TranscriptPart struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	Tool  string                 `json:"tool,omitempty"`
	State map[string]interface{} `json:"state,omitempty"`
}

func runTranscriptFormat(inputPath string) error {
	// Read input file
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Parse JSON
	var data TranscriptData
	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Format to markdown
	markdown := formatTranscript(&data, transcriptToolOutput, transcriptMaxMessages)

	// Output
	if transcriptStdout {
		fmt.Println(markdown)
		return nil
	}

	// Determine output path
	outputPath := transcriptOutput
	if outputPath == "" {
		// Default: same name with .md extension
		ext := filepath.Ext(inputPath)
		outputPath = inputPath[:len(inputPath)-len(ext)] + ".md"
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Formatted transcript written to: %s\n", outputPath)
	return nil
}

func formatTranscript(data *TranscriptData, includeToolOutput bool, maxMessages int) string {
	var lines []string

	// Header
	lines = append(lines, "# Session Transcript", "")

	// Session metadata
	lines = append(lines, fmt.Sprintf("**Title:** %s", data.Info.Title))
	lines = append(lines, fmt.Sprintf("**Session ID:** `%s`", data.Info.ID))
	if data.Info.Directory != "" {
		lines = append(lines, fmt.Sprintf("**Directory:** `%s`", data.Info.Directory))
	}
	if data.Info.Time.Created > 0 {
		lines = append(lines, fmt.Sprintf("**Started:** %s", formatTimestamp(data.Info.Time.Created)))
	}
	if data.Info.Time.Updated > 0 {
		lines = append(lines, fmt.Sprintf("**Updated:** %s", formatTimestamp(data.Info.Time.Updated)))
	}

	// Summary stats
	if data.Info.Summary.Additions > 0 || data.Info.Summary.Deletions > 0 || data.Info.Summary.Files > 0 {
		lines = append(lines, fmt.Sprintf("**Changes:** +%d/-%d in %d files",
			data.Info.Summary.Additions, data.Info.Summary.Deletions, data.Info.Summary.Files))
	}

	lines = append(lines, "", "---", "")

	// Format messages
	messages := data.Messages
	if maxMessages > 0 && len(messages) > maxMessages {
		messages = messages[:maxMessages]
	}

	for _, msg := range messages {
		formatted := formatMessage(&msg, includeToolOutput)
		if formatted != "" {
			lines = append(lines, formatted)
		}
	}

	return strings.Join(lines, "\n")
}

func formatMessage(msg *TranscriptMessage, includeToolOutput bool) string {
	// Collect text parts and tool parts
	var textParts []string
	var toolParts []TranscriptPart

	for _, part := range msg.Parts {
		switch part.Type {
		case "text":
			text := strings.TrimSpace(part.Text)
			if text != "" {
				textParts = append(textParts, text)
			}
		case "tool":
			toolParts = append(toolParts, part)
		}
	}

	// Skip message if no content
	if len(textParts) == 0 && len(toolParts) == 0 {
		return ""
	}

	var lines []string

	// Header with role and timestamp
	timestamp := formatTimestamp(msg.Info.Time.Created)
	switch msg.Info.Role {
	case "user":
		lines = append(lines, fmt.Sprintf("## User (%s)", timestamp))
	case "assistant":
		lines = append(lines, fmt.Sprintf("## Assistant (%s)", timestamp))
		// Add token/cost info
		var tokenInfo []string
		if msg.Info.Tokens.Input > 0 {
			tokenInfo = append(tokenInfo, fmt.Sprintf("in:%d", msg.Info.Tokens.Input))
		}
		if msg.Info.Tokens.Output > 0 {
			tokenInfo = append(tokenInfo, fmt.Sprintf("out:%d", msg.Info.Tokens.Output))
		}
		if msg.Info.Tokens.Cache.Read > 0 {
			tokenInfo = append(tokenInfo, fmt.Sprintf("cached:%d", msg.Info.Tokens.Cache.Read))
		}
		if msg.Info.Cost > 0 {
			tokenInfo = append(tokenInfo, fmt.Sprintf("$%.4f", msg.Info.Cost))
		}
		if len(tokenInfo) > 0 {
			lines = append(lines, fmt.Sprintf("*Tokens: %s*", strings.Join(tokenInfo, ", ")))
		}
	default:
		lines = append(lines, fmt.Sprintf("## %s (%s)", strings.Title(msg.Info.Role), timestamp))
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
			lines = append(lines, formatToolSummary(&tool, includeToolOutput))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func formatToolSummary(part *TranscriptPart, includeOutput bool) string {
	toolName := part.Tool
	if toolName == "" {
		toolName = "unknown"
	}

	state := part.State
	status := "unknown"
	if s, ok := state["status"].(string); ok {
		status = s
	}

	// Get input parameters summary
	var inputSummary string
	if input, ok := state["input"].(map[string]interface{}); ok {
		switch toolName {
		case "read":
			if path, ok := input["filePath"].(string); ok {
				inputSummary = fmt.Sprintf(`"%s"`, path)
			}
		case "bash":
			if cmd, ok := input["command"].(string); ok {
				inputSummary = fmt.Sprintf("`%s`", truncate(cmd, 60))
			}
		case "write", "edit":
			if path, ok := input["filePath"].(string); ok {
				inputSummary = fmt.Sprintf(`"%s"`, path)
			}
		case "glob":
			if pattern, ok := input["pattern"].(string); ok {
				inputSummary = fmt.Sprintf(`"%s"`, pattern)
			}
		case "grep":
			if pattern, ok := input["pattern"].(string); ok {
				inputSummary = fmt.Sprintf(`"%s"`, pattern)
			}
		case "todowrite":
			if todos, ok := input["todos"].([]interface{}); ok {
				inputSummary = fmt.Sprintf("(%d items)", len(todos))
			}
		default:
			// Generic: show first parameter
			for key, val := range input {
				if s, ok := val.(string); ok {
					inputSummary = fmt.Sprintf(`%s="%s"`, key, truncate(s, 40))
				}
				break
			}
		}
	}

	// Status indicator
	statusIcon := map[string]string{
		"completed": "OK",
		"error":     "ERR",
		"pending":   "...",
		"running":   "RUN",
	}[status]
	if statusIcon == "" {
		statusIcon = status
	}

	line := fmt.Sprintf("  [%s] %s", statusIcon, toolName)
	if inputSummary != "" {
		line += " " + inputSummary
	}

	// Add error message if error
	if status == "error" {
		if errStr, ok := state["error"].(string); ok && errStr != "" {
			errMsg := truncate(strings.ReplaceAll(errStr, "\n", " "), 80)
			line += fmt.Sprintf("\n        Error: %s", errMsg)
		}
	}

	// Optionally add output summary
	if includeOutput && status == "completed" {
		if output, ok := state["output"].(string); ok && output != "" {
			lineCount := len(strings.Split(strings.TrimSpace(output), "\n"))
			if lineCount > 3 {
				line += fmt.Sprintf("\n        (%d lines of output)", lineCount)
			}
		}
	}

	return line
}

func formatTimestamp(ms int64) string {
	if ms == 0 {
		return ""
	}
	t := time.Unix(ms/1000, 0)
	return t.Format("2006-01-02 15:04:05")
}
