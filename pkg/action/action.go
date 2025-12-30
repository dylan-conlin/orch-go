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

	// IsOrchestrator indicates whether this action was performed by an orchestrator (true)
	// or a worker agent (false). Detection is based on session title pattern and cwd.
	IsOrchestrator bool `json:"is_orchestrator"`

	// BeadsID is the beads issue ID associated with this action (for worker agents).
	// Extracted from session title format: "workspace [beads-id]"
	BeadsID string `json:"beads_id,omitempty"`
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
// We keep targets mostly intact to avoid over-grouping unrelated actions.
// Only normalize very long targets (truncate) and whitespace.
func normalizeTarget(target string) string {
	target = strings.TrimSpace(target)

	// Truncate very long targets (e.g., long bash commands) for readability
	// but keep enough to distinguish different commands
	if len(target) > 80 {
		return target[:77] + "..."
	}

	return target
}

// isExpectedEmpty returns true if the command is expected to produce no output.
// These should not be flagged as "empty result" patterns.
func isExpectedEmpty(tool, target string) bool {
	if tool != "Bash" {
		return false
	}
	target = strings.TrimSpace(target)

	// Commands that are expected to produce no output
	expectedEmptyPrefixes := []string{
		"sleep ",
		"wait",
		"cd ",
		"export ",
		"unset ",
		"true",
		":",         // bash no-op
		"mkdir -p ", // silent on success
		"touch ",    // silent on success
		"rm ",       // silent on success (unless -v)
		"cp ",       // silent on success (unless -v)
		"mv ",       // silent on success (unless -v)
		"chmod ",    // silent on success
		"chown ",    // silent on success
	}

	for _, prefix := range expectedEmptyPrefixes {
		if strings.HasPrefix(target, prefix) {
			return true
		}
	}

	return false
}

// IsWorkerSession determines if a session is a worker based on its title.
// Worker sessions have titles matching the pattern: "workspace [beads-id]"
// e.g., "og-feat-xxx [orch-go-abc]"
func IsWorkerSession(sessionTitle string) bool {
	// Check if title contains '[' and ends with ']'
	if sessionTitle == "" {
		return false
	}
	bracketStart := strings.LastIndex(sessionTitle, "[")
	bracketEnd := strings.LastIndex(sessionTitle, "]")
	// Worker pattern: has '[' followed by ']' at the end
	return bracketStart != -1 && bracketEnd != -1 && bracketEnd > bracketStart && bracketEnd == len(sessionTitle)-1
}

// IsWorkerWorkspace determines if a path is within a worker workspace.
// Worker workspaces are located under .orch/workspace/
func IsWorkerWorkspace(path string) bool {
	if path == "" {
		return false
	}
	return strings.Contains(path, ".orch/workspace/")
}

// ExtractBeadsIDFromTitle extracts the beads ID from a session title.
// Session titles follow format: "workspace-name [beads-id]"
// e.g., "og-feat-add-feature-24dec [orch-go-3anf]" -> "orch-go-3anf"
func ExtractBeadsIDFromTitle(title string) string {
	if title == "" {
		return ""
	}
	// Look for "[beads-id]" pattern at the end
	start := strings.LastIndex(title, "[")
	end := strings.LastIndex(title, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(title[start+1 : end])
}

// DetectOrchestratorStatus determines if an event is from an orchestrator or worker.
// It uses session title and workspace path to make the determination.
// Returns (isOrchestrator, beadsID).
//
// Detection logic:
// - Worker if: session title contains '[' and ends with ']' (e.g., 'og-feat-xxx [orch-go-abc]')
// - Worker if: workspace contains '.orch/workspace/'
// - Otherwise: orchestrator
func DetectOrchestratorStatus(sessionTitle, workspace string) (isOrchestrator bool, beadsID string) {
	// Check if it's a worker session by title pattern
	if IsWorkerSession(sessionTitle) {
		beadsID = ExtractBeadsIDFromTitle(sessionTitle)
		return false, beadsID
	}

	// Check if it's a worker by workspace path
	if IsWorkerWorkspace(workspace) {
		// Try to extract beads ID from title even if pattern doesn't match perfectly
		beadsID = ExtractBeadsIDFromTitle(sessionTitle)
		return false, beadsID
	}

	// Default to orchestrator
	return true, ""
}

// Logger handles action outcome logging to a JSONL file.
type Logger struct {
	Path string
	// SessionTitle is the title of the current session, used for orchestrator/worker detection.
	// If set, Log will automatically populate IsOrchestrator and BeadsID fields.
	SessionTitle string
}

// LoggerPathFunc allows customizing the log path for testing.
var loggerPathFunc = defaultLogPath

// SetLoggerPathFunc sets the function used to determine the default log path.
// This is primarily used for testing.
func SetLoggerPathFunc(f func() string) {
	loggerPathFunc = f
}

// GetLoggerPathFunc returns the current logger path function.
// This is primarily used for testing to save and restore the original function.
func GetLoggerPathFunc() func() string {
	return loggerPathFunc
}

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

// NewLoggerWithSession creates a new action logger with a custom path and session title.
// The session title is used for automatic orchestrator/worker detection.
func NewLoggerWithSession(path, sessionTitle string) *Logger {
	return &Logger{Path: path, SessionTitle: sessionTitle}
}

// NewDefaultLogger creates a new action logger with the default path.
func NewDefaultLogger() *Logger {
	return &Logger{Path: DefaultLogPath()}
}

// NewDefaultLoggerWithSession creates a new action logger with the default path and session title.
// The session title is used for automatic orchestrator/worker detection.
func NewDefaultLoggerWithSession(sessionTitle string) *Logger {
	return &Logger{Path: DefaultLogPath(), SessionTitle: sessionTitle}
}

// SetSessionTitle sets the session title for orchestrator/worker detection.
func (l *Logger) SetSessionTitle(title string) {
	l.SessionTitle = title
}

// Log appends an action event to the JSONL log file.
// If the logger has a SessionTitle set, it will automatically populate
// IsOrchestrator and BeadsID fields using DetectOrchestratorStatus.
func (l *Logger) Log(event ActionEvent) error {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Auto-detect orchestrator status if SessionTitle is available and fields aren't set
	// We check if BeadsID is empty to avoid overwriting explicitly set values
	if l.SessionTitle != "" && event.BeadsID == "" {
		isOrch, beadsID := DetectOrchestratorStatus(l.SessionTitle, event.Workspace)
		event.IsOrchestrator = isOrch
		event.BeadsID = beadsID
	} else if l.SessionTitle == "" && event.BeadsID == "" {
		// No session title available - try to detect from workspace alone
		// If workspace contains .orch/workspace/, it's a worker
		if IsWorkerWorkspace(event.Workspace) {
			event.IsOrchestrator = false
		} else {
			// Default to orchestrator when no information available
			event.IsOrchestrator = true
		}
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
		// Skip expected-empty commands (sleep, cd, export, etc.)
		if e.Outcome == OutcomeEmpty && isExpectedEmpty(e.Tool, e.Target) {
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
