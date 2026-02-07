// Package action provides action outcome logging and pattern detection.
// This enables behavioral pattern detection by tracking tool invocations
// and their outcomes, allowing the orchestrator to identify repeated futile actions.
//
// The key insight from the prior investigation is that current mechanisms
// track knowledge state (gaps, decisions, constraints) but not action outcomes.
// Tool failures are ephemeral and untracked, making behavioral pattern detection
// impossible. This package addresses that gap by persisting action outcomes.
package action

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Outcome represents the result of a tool action.
type Outcome string

const (
	// OutcomeSuccess indicates the action completed successfully with expected result.
	OutcomeSuccess Outcome = "success"

	// OutcomeEmpty indicates the action succeeded but returned empty/no result.
	// This is distinct from failure - the tool worked but found nothing.
	OutcomeEmpty Outcome = "empty"

	// OutcomeError indicates the action failed with an error.
	OutcomeError Outcome = "error"

	// OutcomeFallback indicates the action failed and a fallback was used.
	// This signals "tried X, fell back to Y" patterns.
	OutcomeFallback Outcome = "fallback"
)

// ActionEvent represents a single tool action with its outcome.
type ActionEvent struct {
	// Timestamp when the action occurred.
	Timestamp time.Time `json:"timestamp"`

	// Tool is the name of the tool invoked (e.g., "Read", "Bash", "Glob").
	Tool string `json:"tool"`

	// Target is what the tool acted on (e.g., file path, command, search pattern).
	Target string `json:"target"`

	// Outcome is the result of the action.
	Outcome Outcome `json:"outcome"`

	// ErrorMessage is set when Outcome is Error.
	ErrorMessage string `json:"error_message,omitempty"`

	// FallbackAction describes what was done instead when Outcome is Fallback.
	FallbackAction string `json:"fallback_action,omitempty"`

	// SessionID links this action to a specific agent session.
	SessionID string `json:"session_id,omitempty"`

	// Workspace links this action to a specific workspace.
	Workspace string `json:"workspace,omitempty"`

	// Context provides additional context about why the action was taken.
	Context string `json:"context,omitempty"`
}

// ActionPattern represents a detected behavioral pattern.
type ActionPattern struct {
	// Tool is the tool that shows the pattern.
	Tool string `json:"tool"`

	// Target is what the tool repeatedly acted on.
	Target string `json:"target"`

	// Outcome is the repeated outcome.
	Outcome Outcome `json:"outcome"`

	// Count is how many times this pattern occurred.
	Count int `json:"count"`

	// FirstSeen is when the pattern was first detected.
	FirstSeen time.Time `json:"first_seen"`

	// LastSeen is when the pattern was most recently seen.
	LastSeen time.Time `json:"last_seen"`

	// Sessions lists the sessions where this pattern occurred.
	Sessions []string `json:"sessions,omitempty"`

	// Workspaces lists the workspaces where this pattern occurred.
	Workspaces []string `json:"workspaces,omitempty"`
}

// PatternKey generates a key for grouping similar actions.
func (e *ActionEvent) PatternKey() string {
	return fmt.Sprintf("%s:%s:%s", e.Tool, normalizeTarget(e.Target), e.Outcome)
}

// normalizeTarget normalizes a target for pattern grouping.
// This helps group similar actions (e.g., different file paths with same suffix).
func normalizeTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return "(empty)"
	}

	// Truncate very long targets (bash commands, etc.)
	const maxLen = 60
	if len(target) > maxLen {
		target = target[:maxLen] + "..."
	}

	// Detect if this is likely a bash command (starts with cd, contains &&, ||, |, etc.)
	if strings.HasPrefix(target, "cd ") ||
		strings.Contains(target, " && ") ||
		strings.Contains(target, " || ") ||
		strings.Contains(target, " | ") {
		// Extract the main command (first command after && or the whole thing)
		cmd := target
		if idx := strings.Index(target, " && "); idx != -1 {
			cmd = strings.TrimSpace(target[idx+4:])
		}
		// Get first word of command
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			cmdName := parts[0]
			// For orch/bd/kb commands, include the subcommand
			if (cmdName == "orch" || cmdName == "bd" || cmdName == "kb" || cmdName == "git") && len(parts) > 1 {
				return cmdName + " " + parts[1]
			}
			return cmdName
		}
	}

	// Detect if this is a file path (starts with / or ~, and looks like a path)
	if (strings.HasPrefix(target, "/") || strings.HasPrefix(target, "~")) &&
		!strings.Contains(target, " ") {
		// It's a file path - extract meaningful part
		base := filepath.Base(target)
		if ext := filepath.Ext(base); ext != "" && len(ext) <= 5 {
			return "*" + ext
		}
		return base
	}

	// For URLs, extract the host
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		// Keep just the host part for grouping
		target = strings.TrimPrefix(target, "http://")
		target = strings.TrimPrefix(target, "https://")
		if idx := strings.Index(target, "/"); idx != -1 {
			target = target[:idx]
		}
		return target
	}

	// For CSS selectors and other targets, return as-is (truncated)
	return target
}

// Logger handles action outcome logging to a JSONL file.
type Logger struct {
	Path string
}

// LoggerPathFunc allows customizing the log path for testing.
var loggerPathFunc = defaultLogPath

// defaultLogPath returns the default path for action-log.jsonl.
func defaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/action-log.jsonl"
	}
	return filepath.Join(home, ".orch", "action-log.jsonl")
}

// DefaultLogPath returns the current default log path.
func DefaultLogPath() string {
	return loggerPathFunc()
}

// NewLogger creates a new action logger with a custom path.
func NewLogger(path string) *Logger {
	return &Logger{Path: path}
}

// NewDefaultLogger creates a new action logger with the default path.
func NewDefaultLogger() *Logger {
	return &Logger{Path: DefaultLogPath()}
}

// Log appends an action event to the JSONL log file.
func (l *Logger) Log(event ActionEvent) error {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Ensure directory exists
	dir := filepath.Dir(l.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for appending
	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open action log file: %w", err)
	}
	defer f.Close()

	// Encode and write
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal action event: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write action event: %w", err)
	}

	return nil
}

// LogSuccess logs a successful action.
func (l *Logger) LogSuccess(tool, target string) error {
	return l.Log(ActionEvent{
		Tool:    tool,
		Target:  target,
		Outcome: OutcomeSuccess,
	})
}

// LogEmpty logs an action that succeeded but returned empty.
func (l *Logger) LogEmpty(tool, target string) error {
	return l.Log(ActionEvent{
		Tool:    tool,
		Target:  target,
		Outcome: OutcomeEmpty,
	})
}

// LogError logs a failed action.
func (l *Logger) LogError(tool, target, errMsg string) error {
	return l.Log(ActionEvent{
		Tool:         tool,
		Target:       target,
		Outcome:      OutcomeError,
		ErrorMessage: errMsg,
	})
}

// LogFallback logs an action that fell back to an alternative.
func (l *Logger) LogFallback(tool, target, fallbackAction string) error {
	return l.Log(ActionEvent{
		Tool:           tool,
		Target:         target,
		Outcome:        OutcomeFallback,
		FallbackAction: fallbackAction,
	})
}

// Tracker loads and analyzes action patterns.
type Tracker struct {
	Events []ActionEvent
}

// LoadTracker loads action events from the log file.
func LoadTracker(path string) (*Tracker, error) {
	if path == "" {
		path = DefaultLogPath()
	}

	tracker := &Tracker{Events: []ActionEvent{}}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return tracker, nil
		}
		return nil, fmt.Errorf("failed to open action log: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event ActionEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip malformed lines
		}
		tracker.Events = append(tracker.Events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading action log: %w", err)
	}

	return tracker, nil
}

// PatternThreshold is the minimum occurrences to consider a pattern.
const PatternThreshold = 3

// MaxAge is how long to consider events for pattern detection.
const MaxAge = 7 * 24 * time.Hour // 1 week

// FindPatterns identifies repeated action patterns.
// Only considers events within MaxAge and requires PatternThreshold occurrences.
func (t *Tracker) FindPatterns() []ActionPattern {
	cutoff := time.Now().Add(-MaxAge)

	// Group events by pattern key
	groups := make(map[string][]ActionEvent)
	for _, e := range t.Events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		// Only track non-success outcomes (those are the "futile" actions)
		if e.Outcome == OutcomeSuccess {
			continue
		}
		key := e.PatternKey()
		groups[key] = append(groups[key], e)
	}

	// Build patterns from groups that meet threshold
	patterns := []ActionPattern{}
	for _, events := range groups {
		if len(events) < PatternThreshold {
			continue
		}

		// Collect unique sessions and workspaces
		sessions := make(map[string]bool)
		workspaces := make(map[string]bool)
		var firstSeen, lastSeen time.Time

		for _, e := range events {
			if e.SessionID != "" {
				sessions[e.SessionID] = true
			}
			if e.Workspace != "" {
				workspaces[e.Workspace] = true
			}
			if firstSeen.IsZero() || e.Timestamp.Before(firstSeen) {
				firstSeen = e.Timestamp
			}
			if lastSeen.IsZero() || e.Timestamp.After(lastSeen) {
				lastSeen = e.Timestamp
			}
		}

		// Build unique lists
		sessionList := make([]string, 0, len(sessions))
		for s := range sessions {
			sessionList = append(sessionList, s)
		}
		sort.Strings(sessionList)

		workspaceList := make([]string, 0, len(workspaces))
		for w := range workspaces {
			workspaceList = append(workspaceList, w)
		}
		sort.Strings(workspaceList)

		patterns = append(patterns, ActionPattern{
			Tool:       events[0].Tool,
			Target:     normalizeTarget(events[0].Target),
			Outcome:    events[0].Outcome,
			Count:      len(events),
			FirstSeen:  firstSeen,
			LastSeen:   lastSeen,
			Sessions:   sessionList,
			Workspaces: workspaceList,
		})
	}

	// Sort by count (highest first)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	return patterns
}

// FindPatternsForSession finds patterns specific to a session.
func (t *Tracker) FindPatternsForSession(sessionID string) []ActionPattern {
	cutoff := time.Now().Add(-MaxAge)

	// Filter to session events
	sessionEvents := []ActionEvent{}
	for _, e := range t.Events {
		if e.SessionID == sessionID && e.Timestamp.After(cutoff) && e.Outcome != OutcomeSuccess {
			sessionEvents = append(sessionEvents, e)
		}
	}

	// Group by pattern key
	groups := make(map[string][]ActionEvent)
	for _, e := range sessionEvents {
		key := e.PatternKey()
		groups[key] = append(groups[key], e)
	}

	// Build patterns
	patterns := []ActionPattern{}
	for _, events := range groups {
		if len(events) < 2 { // Lower threshold for session-specific
			continue
		}

		var firstSeen, lastSeen time.Time
		for _, e := range events {
			if firstSeen.IsZero() || e.Timestamp.Before(firstSeen) {
				firstSeen = e.Timestamp
			}
			if lastSeen.IsZero() || e.Timestamp.After(lastSeen) {
				lastSeen = e.Timestamp
			}
		}

		patterns = append(patterns, ActionPattern{
			Tool:      events[0].Tool,
			Target:    normalizeTarget(events[0].Target),
			Outcome:   events[0].Outcome,
			Count:     len(events),
			FirstSeen: firstSeen,
			LastSeen:  lastSeen,
			Sessions:  []string{sessionID},
		})
	}

	// Sort by count
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	return patterns
}

// Summary returns a brief summary of the action log state.
func (t *Tracker) Summary() string {
	if len(t.Events) == 0 {
		return "No actions tracked yet"
	}

	patterns := t.FindPatterns()
	return fmt.Sprintf("%d action events tracked, %d behavioral patterns detected", len(t.Events), len(patterns))
}

// FormatPatterns formats patterns for display.
func FormatPatterns(patterns []ActionPattern) string {
	if len(patterns) == 0 {
		return "No behavioral patterns detected - actions are effective!\n"
	}

	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔══════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║  🔄 BEHAVIORAL PATTERNS - Repeated futile actions detected                   ║\n")
	sb.WriteString("╠══════════════════════════════════════════════════════════════════════════════╣\n")

	for i, p := range patterns {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("║  ... and %d more patterns                                                   ║\n", len(patterns)-10))
			break
		}

		icon := "○"
		if p.Count >= 5 {
			icon = "●"
		} else if p.Count >= 3 {
			icon = "◐"
		}

		outcomeStr := string(p.Outcome)
		sb.WriteString(fmt.Sprintf("║  %s [%s] %s → %s (%dx)\n", icon, outcomeStr, truncate(p.Tool, 15), truncate(p.Target, 30), p.Count))

		// Show sessions/workspaces if available
		if len(p.Workspaces) > 0 {
			sb.WriteString(fmt.Sprintf("║      Workspaces: %s\n", truncate(strings.Join(p.Workspaces, ", "), 50)))
		}
	}

	sb.WriteString("╠══════════════════════════════════════════════════════════════════════════════╣\n")
	sb.WriteString("║  These patterns may indicate knowledge gaps or system limitations            ║\n")
	sb.WriteString("║  Consider: kn tried \"[action]\" --failed \"[why]\"                              ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════════════════════╝\n")

	return sb.String()
}

// truncate shortens a string to fit display.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SuggestKnEntry returns a suggested kn command based on a pattern.
func (p *ActionPattern) SuggestKnEntry() string {
	switch p.Outcome {
	case OutcomeEmpty:
		return fmt.Sprintf(`kn tried "%s on %s" --failed "Returns empty - target doesn't exist or has no content"`, p.Tool, p.Target)
	case OutcomeError:
		return fmt.Sprintf(`kn tried "%s on %s" --failed "Action fails repeatedly - investigate cause"`, p.Tool, p.Target)
	case OutcomeFallback:
		return fmt.Sprintf(`kn constrain "Avoid %s on %s" --reason "Requires fallback - prefer alternative approach"`, p.Tool, p.Target)
	default:
		return ""
	}
}

// Prune removes old events from the log file.
// Returns the number of events pruned.
func Prune(path string, maxAge time.Duration) (int, error) {
	if path == "" {
		path = DefaultLogPath()
	}

	// Load all events
	tracker, err := LoadTracker(path)
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-maxAge)
	kept := []ActionEvent{}
	pruned := 0

	for _, e := range tracker.Events {
		if e.Timestamp.After(cutoff) {
			kept = append(kept, e)
		} else {
			pruned++
		}
	}

	if pruned == 0 {
		return 0, nil
	}

	// Rewrite file with kept events
	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("failed to create pruned log: %w", err)
	}
	defer f.Close()

	for _, e := range kept {
		data, err := json.Marshal(e)
		if err != nil {
			continue
		}
		f.Write(append(data, '\n'))
	}

	return pruned, nil
}
