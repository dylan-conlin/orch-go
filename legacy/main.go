package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
)

// OpenCodeEvent represents an event from opencode's JSON output
type OpenCodeEvent struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionID,omitempty"` // Top-level sessionID in actual opencode output
	Session   *SessionInfo    `json:"session,omitempty"`
	Step      *StepInfo       `json:"step,omitempty"`
	Content   string          `json:"content,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
	Raw       json.RawMessage `json:"-"`
}

// SessionInfo contains session details
type SessionInfo struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// StepInfo contains step details
type StepInfo struct {
	ID string `json:"id"`
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string
	Data  string
}

// Event is a loggable event for events.jsonl
type Event struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Config holds CLI configuration
type Config struct {
	Command   string
	Prompt    string
	SessionID string
	ServerURL string
	Title     string
}

// OpenCodeResult holds the result of processing opencode output
type OpenCodeResult struct {
	SessionID string
	Events    []OpenCodeEvent
}

// SSEClient handles SSE connections
type SSEClient struct {
	URL string
}

// ParseOpenCodeEvent parses a JSON event from opencode output
func ParseOpenCodeEvent(line string) (OpenCodeEvent, error) {
	var event OpenCodeEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return event, err
	}
	return event, nil
}

// ParseSSEEvent parses an SSE formatted event
func ParseSSEEvent(raw string) (event string, data string) {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		}
	}
	return event, data
}

// ParseSessionStatus extracts status and session ID from SSE data
func ParseSessionStatus(data string) (status string, sessionID string) {
	var parsed struct {
		Status    string `json:"status"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return "", ""
	}
	return parsed.Status, parsed.SessionID
}

// LogEvent appends an event to the JSONL log file
func LogEvent(path string, event Event) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for appending
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Encode and write
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

// ExtractSessionID extracts session ID from opencode events
// OpenCode includes sessionID at the top level of each event
func ExtractSessionID(events []string) (string, error) {
	for _, line := range events {
		event, err := ParseOpenCodeEvent(line)
		if err != nil {
			continue
		}
		// sessionID is at top level of each event in actual opencode output
		if event.SessionID != "" {
			return event.SessionID, nil
		}
	}
	return "", fmt.Errorf("no session ID found in output")
}

// BuildSpawnCommand builds the opencode spawn command
func BuildSpawnCommand(serverURL, prompt, title string) *exec.Cmd {
	args := []string{
		"run",
		"--attach", serverURL,
		"--format", "json",
		"--title", title,
		prompt,
	}
	return exec.Command("opencode", args...)
}

// BuildAskCommand builds the opencode ask command
func BuildAskCommand(serverURL, sessionID, prompt string) *exec.Cmd {
	args := []string{
		"run",
		"--attach", serverURL,
		"--session", sessionID,
		"--format", "json",
		prompt,
	}
	return exec.Command("opencode", args...)
}

// NewSSEClient creates a new SSE client
func NewSSEClient(url string) *SSEClient {
	return &SSEClient{URL: url}
}

// Connect establishes SSE connection and sends events to channel
func (c *SSEClient) Connect(events chan<- SSEEvent) error {
	resp, err := http.Get(c.URL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
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

// DetectCompletion checks if events indicate session completion
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

// ParseArgs parses CLI arguments
func ParseArgs(args []string) (Config, error) {
	cfg := Config{
		ServerURL: "http://localhost:4096",
	}

	if len(args) < 2 {
		return cfg, fmt.Errorf("usage: orch-go <command> [args]")
	}

	cfg.Command = args[1]

	switch cfg.Command {
	case "spawn":
		if len(args) < 3 {
			return cfg, fmt.Errorf("usage: orch-go spawn <prompt>")
		}
		cfg.Prompt = strings.Join(args[2:], " ")
		cfg.Title = fmt.Sprintf("orch-go-%d", time.Now().Unix())
	case "ask":
		if len(args) < 4 {
			return cfg, fmt.Errorf("usage: orch-go ask <session-id> <prompt>")
		}
		cfg.SessionID = args[2]
		cfg.Prompt = strings.Join(args[3:], " ")
	case "monitor":
		// No additional args needed
	default:
		return cfg, fmt.Errorf("unknown command: %s", cfg.Command)
	}

	return cfg, nil
}

// FormatNotification formats a notification message
func FormatNotification(sessionID, status string) (title string, body string) {
	title = "OpenCode Session Update"
	body = fmt.Sprintf("Session %s: %s", sessionID, status)
	return title, body
}

// SendNotification sends a macOS desktop notification
func SendNotification(title, body string) error {
	return beeep.Notify(title, body, "")
}

// ProcessOpenCodeOutput processes the output from opencode command
func ProcessOpenCodeOutput(r io.Reader) (*OpenCodeResult, error) {
	result := &OpenCodeResult{}
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		event, err := ParseOpenCodeEvent(line)
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

// GetEventsLogPath returns the path to events.jsonl
func GetEventsLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/events.jsonl"
	}
	return filepath.Join(home, ".orch", "events.jsonl")
}

// RunSpawn executes the spawn command
func RunSpawn(cfg Config) error {
	cmd := BuildSpawnCommand(cfg.ServerURL, cfg.Prompt, cfg.Title)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	result, err := ProcessOpenCodeOutput(stdout)
	if err != nil {
		return fmt.Errorf("failed to process output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("opencode exited with error: %w", err)
	}

	// Log the session creation
	logPath := GetEventsLogPath()
	event := Event{
		Type:      "session.spawned",
		SessionID: result.SessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"prompt": cfg.Prompt,
			"title":  cfg.Title,
		},
	}
	if err := LogEvent(logPath, event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Session ID: %s\n", result.SessionID)
	return nil
}

// RunAsk executes the ask command
func RunAsk(cfg Config) error {
	cmd := BuildAskCommand(cfg.ServerURL, cfg.SessionID, cfg.Prompt)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	result, err := ProcessOpenCodeOutput(stdout)
	if err != nil {
		return fmt.Errorf("failed to process output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("opencode exited with error: %w", err)
	}

	// Log the Q&A
	logPath := GetEventsLogPath()
	event := Event{
		Type:      "session.ask",
		SessionID: cfg.SessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"prompt":      cfg.Prompt,
			"event_count": len(result.Events),
		},
	}
	if err := LogEvent(logPath, event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Q&A complete for session: %s\n", cfg.SessionID)
	return nil
}

// RunMonitor executes the monitor command
func RunMonitor(cfg Config) error {
	sseURL := cfg.ServerURL + "/event"
	client := NewSSEClient(sseURL)

	fmt.Printf("Monitoring SSE events at %s...\n", sseURL)

	events := make(chan SSEEvent, 100)
	errChan := make(chan error, 1)

	go func() {
		if err := client.Connect(events); err != nil {
			errChan <- err
		}
		close(events)
	}()

	logPath := GetEventsLogPath()
	var sessionEvents []SSEEvent
	var currentSession string
	_ = currentSession // used for potential future extensions

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return nil
			}

			// Log every event
			logEvent := Event{
				Type:      event.Event,
				Timestamp: time.Now().Unix(),
				Data:      map[string]interface{}{"raw_data": event.Data},
			}

			// Parse session info if available
			if event.Event == "session.status" || event.Event == "session.created" {
				status, sid := ParseSessionStatus(event.Data)
				if sid != "" {
					logEvent.SessionID = sid
					currentSession = sid
				}
				if status != "" {
					logEvent.Data["status"] = status
				}
			}

			if err := LogEvent(logPath, logEvent); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
			}

			fmt.Printf("[%s] %s\n", event.Event, event.Data)
			sessionEvents = append(sessionEvents, event)

			// Check for completion
			sessionID, completed := DetectCompletion(sessionEvents)
			if completed {
				title, body := FormatNotification(sessionID, "completed")
				if err := SendNotification(title, body); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to send notification: %v\n", err)
				}
				fmt.Printf("\nSession %s completed!\n", sessionID)

				// Log completion
				completionEvent := Event{
					Type:      "session.completed",
					SessionID: sessionID,
					Timestamp: time.Now().Unix(),
				}
				if err := LogEvent(logPath, completionEvent); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log completion: %v\n", err)
				}

				// Reset for next session
				sessionEvents = nil
			}

		case err := <-errChan:
			return fmt.Errorf("SSE connection error: %w", err)
		}
	}
}

func main() {
	cfg, err := ParseArgs(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nCommands:")
		fmt.Fprintln(os.Stderr, "  spawn <prompt>              Spawn a new session")
		fmt.Fprintln(os.Stderr, "  monitor                     Monitor SSE events for completion")
		fmt.Fprintln(os.Stderr, "  ask <session-id> <prompt>   Ask a question to an existing session")
		os.Exit(1)
	}

	var cmdErr error
	switch cfg.Command {
	case "spawn":
		cmdErr = RunSpawn(cfg)
	case "ask":
		cmdErr = RunAsk(cfg)
	case "monitor":
		cmdErr = RunMonitor(cfg)
	}

	if cmdErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", cmdErr)
		os.Exit(1)
	}
}
