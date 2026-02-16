package timeline

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// ExtractOptions configures timeline extraction.
type ExtractOptions struct {
	ProjectDir string // Project root directory
	SessionID  string // Filter to specific session (empty = all sessions)
	Limit      int    // Limit number of sessions returned (0 = unlimited)
}

// Extract builds the timeline from various data sources.
func Extract(opts ExtractOptions) (*Timeline, error) {
	if opts.ProjectDir == "" {
		return nil, fmt.Errorf("project directory is required")
	}

	var allActions []TimelineAction

	// 1. Extract from event log (.orch/events.jsonl)
	eventActions, err := extractFromEventLog(opts.ProjectDir)
	if err != nil {
		// Don't fail if event log doesn't exist or can't be read
		fmt.Fprintf(os.Stderr, "Warning: failed to extract from event log: %v\n", err)
	} else {
		allActions = append(allActions, eventActions...)
	}

	// 2. Extract from beads comments
	beadsActions, err := extractFromBeads(opts.ProjectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to extract from beads: %v\n", err)
	} else {
		allActions = append(allActions, beadsActions...)
	}

	// 3. Extract from quick decisions (.kb/quick/)
	quickActions, err := extractFromQuickDecisions(opts.ProjectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to extract from quick decisions: %v\n", err)
	} else {
		allActions = append(allActions, quickActions...)
	}

	// 4. Extract from decisions (.kb/decisions/)
	decisionActions, err := extractFromDecisions(opts.ProjectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to extract from decisions: %v\n", err)
	} else {
		allActions = append(allActions, decisionActions...)
	}

	// Filter by session if specified
	if opts.SessionID != "" {
		filtered := make([]TimelineAction, 0)
		for _, action := range allActions {
			if action.SessionID == opts.SessionID {
				filtered = append(filtered, action)
			}
		}
		allActions = filtered
	}

	// Group by session
	timeline := groupBySession(allActions, opts.ProjectDir)

	// Apply limit if specified
	if opts.Limit > 0 && len(timeline.Sessions) > opts.Limit {
		timeline.Sessions = timeline.Sessions[:opts.Limit]
		// Recalculate total
		total := 0
		for _, session := range timeline.Sessions {
			total += session.ActionCount
		}
		timeline.Total = total
	}

	return timeline, nil
}

// extractFromEventLog reads the events.jsonl file and extracts timeline actions.
func extractFromEventLog(projectDir string) ([]TimelineAction, error) {
	eventLogPath := filepath.Join(projectDir, ".orch", "events.jsonl")
	file, err := os.Open(eventLogPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var actions []TimelineAction
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var event events.Event
		if err := json.Unmarshal([]byte(scanner.Text()), &event); err != nil {
			continue // Skip malformed lines
		}

		// Extract session ID from event data
		sessionID := getSessionIDFromEvent(event)

		// Convert event to timeline action
		switch event.Type {
		case "session.started":
			actions = append(actions, TimelineAction{
				Type:      ActionTypeSessionStarted,
				Timestamp: time.Unix(event.Timestamp, 0),
				SessionID: sessionID,
				Title:     getStringFromData(event.Data, "goal"),
				Metadata:  event.Data,
			})

		case "session.ended":
			actions = append(actions, TimelineAction{
				Type:      ActionTypeSessionEnded,
				Timestamp: time.Unix(event.Timestamp, 0),
				SessionID: sessionID,
				Title:     getStringFromData(event.Data, "goal"),
				Metadata:  event.Data,
			})

		case "session.labeled":
			actions = append(actions, TimelineAction{
				Type:      ActionTypeSessionLabeled,
				Timestamp: time.Unix(event.Timestamp, 0),
				SessionID: sessionID,
				Title:     getStringFromData(event.Data, "label"),
				Metadata:  event.Data,
			})

		case "agent.spawned":
			actions = append(actions, TimelineAction{
				Type:      ActionTypeAgentSpawned,
				Timestamp: time.Unix(event.Timestamp, 0),
				SessionID: sessionID,
				Title:     getStringFromData(event.Data, "title"),
				BeadsID:   getStringFromData(event.Data, "beads_id"),
				Metadata:  event.Data,
			})

		case "agent.completed":
			actions = append(actions, TimelineAction{
				Type:      ActionTypeAgentCompleted,
				Timestamp: time.Unix(event.Timestamp, 0),
				SessionID: sessionID,
				Title:     getStringFromData(event.Data, "title"),
				BeadsID:   getStringFromData(event.Data, "beads_id"),
				Metadata:  event.Data,
			})

		case "issue.created":
			actions = append(actions, TimelineAction{
				Type:      ActionTypeIssueCreated,
				Timestamp: time.Unix(event.Timestamp, 0),
				SessionID: sessionID,
				Title:     getStringFromData(event.Data, "title"),
				BeadsID:   getStringFromData(event.Data, "issue_id"),
				Metadata:  event.Data,
			})

		case "issue.closed":
			actions = append(actions, TimelineAction{
				Type:      ActionTypeIssueClosed,
				Timestamp: time.Unix(event.Timestamp, 0),
				SessionID: sessionID,
				Title:     getStringFromData(event.Data, "title"),
				BeadsID:   getStringFromData(event.Data, "issue_id"),
				Metadata:  event.Data,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return actions, nil
}

// extractFromBeads reads beads comments for Phase reports.
func extractFromBeads(projectDir string) ([]TimelineAction, error) {
	// TODO: Parse .beads/issues.jsonl for comments with Phase: reports
	// This requires parsing beads issue structure
	return []TimelineAction{}, nil
}

// extractFromQuickDecisions reads .kb/quick/ JSONL files.
func extractFromQuickDecisions(projectDir string) ([]TimelineAction, error) {
	quickDir := filepath.Join(projectDir, ".kb", "quick")
	if _, err := os.Stat(quickDir); os.IsNotExist(err) {
		return []TimelineAction{}, nil
	}

	var actions []TimelineAction

	// Read all JSONL files in .kb/quick/
	entries, err := os.ReadDir(quickDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		filePath := filepath.Join(quickDir, entry.Name())
		file, err := os.Open(filePath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var quickEntry map[string]interface{}
			if err := json.Unmarshal([]byte(scanner.Text()), &quickEntry); err != nil {
				continue
			}

			// Extract timestamp
			timestampStr, ok := quickEntry["timestamp"].(string)
			if !ok {
				continue
			}
			timestamp, err := time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				continue
			}

			// Extract content
			content, _ := quickEntry["content"].(string)
			typ, _ := quickEntry["type"].(string)

			actions = append(actions, TimelineAction{
				Type:      ActionTypeQuickDecision,
				Timestamp: timestamp,
				SessionID: getCurrentSessionID(), // Fallback to current session
				Title:     fmt.Sprintf("%s: %s", typ, content),
				Path:      filePath,
				Metadata:  quickEntry,
			})
		}
		file.Close()
	}

	return actions, nil
}

// extractFromDecisions reads .kb/decisions/ markdown files.
func extractFromDecisions(projectDir string) ([]TimelineAction, error) {
	decisionsDir := filepath.Join(projectDir, ".kb", "decisions")
	if _, err := os.Stat(decisionsDir); os.IsNotExist(err) {
		return []TimelineAction{}, nil
	}

	var actions []TimelineAction

	// Read all .md files in .kb/decisions/
	err := filepath.Walk(decisionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Use file modification time as decision timestamp
		timestamp := info.ModTime()

		// Extract title from filename or first line
		title := extractTitleFromDecision(path)

		actions = append(actions, TimelineAction{
			Type:      ActionTypeDecisionMade,
			Timestamp: timestamp,
			SessionID: getCurrentSessionID(), // Fallback to current session
			Title:     title,
			Path:      path,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return actions, nil
}

// groupBySession groups actions by session ID and sorts chronologically.
func groupBySession(actions []TimelineAction, projectDir string) *Timeline {
	// Load session labels
	labels := loadSessionLabels(projectDir)

	// Group by session ID
	sessionMap := make(map[string]*SessionGroup)

	for _, action := range actions {
		sessionID := action.SessionID
		if sessionID == "" {
			sessionID = "unknown"
		}

		group, exists := sessionMap[sessionID]
		if !exists {
			group = &SessionGroup{
				SessionID: sessionID,
				Label:     labels[sessionID],
				Actions:   []TimelineAction{},
			}
			sessionMap[sessionID] = group
		}

		group.Actions = append(group.Actions, action)
	}

	// Convert map to slice and sort by start time (descending - most recent first)
	var sessions []SessionGroup
	for _, group := range sessionMap {
		// Sort actions within each session chronologically (ascending)
		sort.Slice(group.Actions, func(i, j int) bool {
			return group.Actions[i].Timestamp.Before(group.Actions[j].Timestamp)
		})

		// Set start and end times
		if len(group.Actions) > 0 {
			group.StartTime = group.Actions[0].Timestamp
			group.EndTime = group.Actions[len(group.Actions)-1].Timestamp
		}

		group.ActionCount = len(group.Actions)
		sessions = append(sessions, *group)
	}

	// Sort sessions by start time (descending - most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	// Calculate total
	total := 0
	for _, session := range sessions {
		total += session.ActionCount
	}

	return &Timeline{
		Sessions: sessions,
		Total:    total,
	}
}

// loadSessionLabels loads session labels from .orch/session_labels.json.
func loadSessionLabels(projectDir string) map[string]string {
	labelsFile := filepath.Join(projectDir, ".orch", "session_labels.json")
	data, err := os.ReadFile(labelsFile)
	if err != nil {
		return make(map[string]string)
	}

	var labels map[string]string
	if err := json.Unmarshal(data, &labels); err != nil {
		return make(map[string]string)
	}

	return labels
}

// getCurrentSessionID returns the current OpenCode session ID from environment.
func getCurrentSessionID() string {
	return os.Getenv("CLAUDE_SESSION_ID")
}

// getSessionIDFromEvent extracts session ID from event data.
func getSessionIDFromEvent(event events.Event) string {
	if sessionID, ok := event.Data["session_id"].(string); ok {
		return sessionID
	}
	// Fallback to current session if not in event
	return getCurrentSessionID()
}

// getStringFromData safely extracts a string from event data map.
func getStringFromData(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

// extractTitleFromDecision extracts title from decision file.
func extractTitleFromDecision(path string) string {
	// Try to read first line of file for title
	file, err := os.Open(path)
	if err != nil {
		return filepath.Base(path)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Look for markdown heading
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
		// Or return first non-empty line
		if line != "" {
			return line
		}
	}

	return filepath.Base(path)
}
