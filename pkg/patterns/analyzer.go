// Package patterns provides behavioral pattern detection for orchestrator actions.
// It analyzes action logs to detect repeated failures and other behavioral patterns
// that indicate the orchestrator is performing futile or incorrect actions.
package patterns

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Analyzer constants.
const (
	// RepetitionThreshold is the number of times an action must occur with the same
	// outcome before it's considered a behavioral pattern worth surfacing.
	RepetitionThreshold = 3

	// ActionLogMaxAge is how long to keep action events (7 days for behavioral patterns).
	// This is shorter than gap tracking because behavioral patterns are more ephemeral.
	ActionLogMaxAge = 7 * 24 * time.Hour

	// MaxActionEvents is the maximum number of action events to store.
	MaxActionEvents = 500
)

// ActionOutcome represents the result of a tool action.
type ActionOutcome string

const (
	// OutcomeSuccess indicates the action completed successfully with meaningful result.
	OutcomeSuccess ActionOutcome = "success"

	// OutcomeEmpty indicates the action completed but returned empty/no data.
	OutcomeEmpty ActionOutcome = "empty"

	// OutcomeError indicates the action failed with an error.
	OutcomeError ActionOutcome = "error"

	// OutcomeTimeout indicates the action timed out.
	OutcomeTimeout ActionOutcome = "timeout"
)

// ActionEvent represents a single tool action and its outcome.
// This is the input format for the pattern analyzer, produced by the action logging subsystem.
type ActionEvent struct {
	// Timestamp when the action occurred.
	Timestamp time.Time `json:"timestamp"`

	// Tool is the name of the tool that was invoked (e.g., "Read", "Bash", "Glob").
	Tool string `json:"tool"`

	// Target is the specific target of the action (e.g., file path, command, pattern).
	Target string `json:"target"`

	// Outcome is the result category of the action.
	Outcome ActionOutcome `json:"outcome"`

	// OutcomeDetail provides additional context about the outcome.
	// For errors, this contains the error message.
	// For empty, this may explain why (e.g., "file not found", "empty file").
	OutcomeDetail string `json:"outcome_detail,omitempty"`

	// WorkspaceDir is the workspace directory where this action occurred.
	WorkspaceDir string `json:"workspace_dir,omitempty"`

	// WorkspaceContext contains relevant workspace metadata (e.g., tier, skill, phase).
	WorkspaceContext map[string]string `json:"workspace_context,omitempty"`

	// SessionID is the OpenCode session ID where this action occurred.
	SessionID string `json:"session_id,omitempty"`
}

// Pattern represents a detected behavioral pattern.
type Pattern struct {
	// Type categorizes the pattern (e.g., "repeated_empty_read", "repeated_error").
	Type string `json:"type"`

	// Description is a human-readable description of the pattern.
	Description string `json:"description"`

	// Severity indicates how significant this pattern is.
	// Values: "info", "warning", "critical"
	Severity string `json:"severity"`

	// Count is how many times this pattern was observed.
	Count int `json:"count"`

	// Events are the actions that form this pattern.
	Events []ActionEvent `json:"events"`

	// Suggestion is a recommended action to address this pattern.
	Suggestion string `json:"suggestion,omitempty"`

	// Context captures common context across pattern events.
	Context map[string]string `json:"context,omitempty"`
}

// ActionLog holds the history of action events for pattern analysis.
type ActionLog struct {
	// Events is the list of recorded action events.
	Events []ActionEvent `json:"events"`

	// LastAnalysis tracks when pattern analysis was last run.
	LastAnalysis time.Time `json:"last_analysis,omitempty"`

	// SuppressedPatterns tracks patterns that have been acknowledged/suppressed.
	SuppressedPatterns []SuppressedPattern `json:"suppressed_patterns,omitempty"`
}

// SuppressedPattern represents a pattern that has been acknowledged and suppressed.
type SuppressedPattern struct {
	// PatternKey uniquely identifies the pattern (tool+target+outcome combination).
	PatternKey string `json:"pattern_key"`

	// SuppressedAt is when the pattern was suppressed.
	SuppressedAt time.Time `json:"suppressed_at"`

	// Reason explains why the pattern was suppressed.
	Reason string `json:"reason,omitempty"`

	// ExpiresAt is when the suppression expires (0 for permanent).
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// logPathFunc is a variable to allow testing with custom paths.
var logPathFunc = defaultLogPath

// defaultLogPath returns the default path for the action log file.
func defaultLogPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".orch", "action-log.json")
}

// LogPath returns the path for the action log file.
func LogPath() string {
	return logPathFunc()
}

// LoadLog loads the action log from disk.
// Returns an empty log if file doesn't exist.
func LoadLog() (*ActionLog, error) {
	path := LogPath()
	if path == "" {
		return &ActionLog{Events: []ActionEvent{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ActionLog{Events: []ActionEvent{}}, nil
		}
		return nil, fmt.Errorf("failed to read action log: %w", err)
	}

	var log ActionLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("failed to parse action log: %w", err)
	}

	return &log, nil
}

// Save saves the action log to disk.
func (l *ActionLog) Save() error {
	path := LogPath()
	if path == "" {
		return fmt.Errorf("could not determine log path")
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prune old events before saving
	l.pruneOldEvents()

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

// pruneOldEvents removes events older than ActionLogMaxAge and caps at MaxActionEvents.
func (l *ActionLog) pruneOldEvents() {
	cutoff := time.Now().Add(-ActionLogMaxAge)

	// Filter out old events
	kept := []ActionEvent{}
	for _, e := range l.Events {
		if e.Timestamp.After(cutoff) {
			kept = append(kept, e)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(kept, func(i, j int) bool {
		return kept[i].Timestamp.After(kept[j].Timestamp)
	})

	// Cap at max events
	if len(kept) > MaxActionEvents {
		kept = kept[:MaxActionEvents]
	}

	l.Events = kept

	// Also prune expired suppressions
	l.pruneExpiredSuppressions()
}

// pruneExpiredSuppressions removes suppressions that have expired.
func (l *ActionLog) pruneExpiredSuppressions() {
	now := time.Now()
	kept := []SuppressedPattern{}
	for _, sp := range l.SuppressedPatterns {
		// Keep if no expiration or not yet expired
		if sp.ExpiresAt.IsZero() || sp.ExpiresAt.After(now) {
			kept = append(kept, sp)
		}
	}
	l.SuppressedPatterns = kept
}

// RecordAction adds a new action event to the log.
func (l *ActionLog) RecordAction(event ActionEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	l.Events = append(l.Events, event)
}

// DetectPatterns analyzes the action log and returns detected behavioral patterns.
// Only returns patterns that haven't been suppressed.
func (l *ActionLog) DetectPatterns() []Pattern {
	patterns := []Pattern{}

	// Detect repeated empty reads
	patterns = append(patterns, l.detectRepeatedEmptyReads()...)

	// Detect repeated errors
	patterns = append(patterns, l.detectRepeatedErrors()...)

	// Filter out suppressed patterns
	patterns = l.filterSuppressedPatterns(patterns)

	// Sort by count (highest first), then severity
	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i].Count != patterns[j].Count {
			return patterns[i].Count > patterns[j].Count
		}
		return severityOrder(patterns[i].Severity) < severityOrder(patterns[j].Severity)
	})

	return patterns
}

// detectRepeatedEmptyReads finds cases where the same file/target was read multiple times
// with empty results, especially when the workspace context indicates this was expected
// to fail (e.g., SYNTHESIS.md in a light-tier workspace).
func (l *ActionLog) detectRepeatedEmptyReads() []Pattern {
	// Group by normalized key: tool + normalized target
	groups := make(map[string][]ActionEvent)
	for _, e := range l.Events {
		if e.Outcome != OutcomeEmpty {
			continue
		}
		// Only consider Read tool for now
		if e.Tool != "Read" && e.Tool != "read" {
			continue
		}
		key := normalizeActionKey(e.Tool, e.Target)
		groups[key] = append(groups[key], e)
	}

	patterns := []Pattern{}
	for key, events := range groups {
		if len(events) < RepetitionThreshold {
			continue
		}

		// Extract common context
		commonContext := extractCommonContext(events)

		// Determine severity based on context
		severity := "warning"
		suggestion := fmt.Sprintf("File %q has been read %d times with empty result. Consider checking if this file should exist.", events[0].Target, len(events))

		// Special handling for SYNTHESIS.md in light-tier workspaces
		if strings.HasSuffix(events[0].Target, "SYNTHESIS.md") {
			if tier, ok := commonContext["tier"]; ok && tier == "light" {
				severity = "info"
				suggestion = "SYNTHESIS.md reads in light-tier workspaces are expected to be empty. Consider skipping this check for light-tier agents."
			} else {
				severity = "warning"
				suggestion = "SYNTHESIS.md should exist in non-light-tier workspaces. Check if agent completed synthesis phase."
			}
		}

		patterns = append(patterns, Pattern{
			Type:        "repeated_empty_read",
			Description: fmt.Sprintf("Read %q returned empty %d times", events[0].Target, len(events)),
			Severity:    severity,
			Count:       len(events),
			Events:      events,
			Suggestion:  suggestion,
			Context:     commonContext,
		})

		// Store pattern key for suppression reference
		_ = key // Used for pattern identification
	}

	return patterns
}

// detectRepeatedErrors finds cases where the same action failed repeatedly with errors.
func (l *ActionLog) detectRepeatedErrors() []Pattern {
	// Group by tool + normalized target + error type
	groups := make(map[string][]ActionEvent)
	for _, e := range l.Events {
		if e.Outcome != OutcomeError {
			continue
		}
		// Include error type in key for more specific patterns
		key := fmt.Sprintf("%s:%s", normalizeActionKey(e.Tool, e.Target), normalizeErrorType(e.OutcomeDetail))
		groups[key] = append(groups[key], e)
	}

	patterns := []Pattern{}
	for _, events := range groups {
		if len(events) < RepetitionThreshold {
			continue
		}

		// Extract common context
		commonContext := extractCommonContext(events)

		// Determine severity based on repetition count
		severity := "warning"
		if len(events) >= RepetitionThreshold*2 {
			severity = "critical"
		}

		patterns = append(patterns, Pattern{
			Type:        "repeated_error",
			Description: fmt.Sprintf("%s on %q failed %d times with similar error", events[0].Tool, events[0].Target, len(events)),
			Severity:    severity,
			Count:       len(events),
			Events:      events,
			Suggestion:  fmt.Sprintf("This action has failed %d times. Consider investigating the root cause or using a different approach.", len(events)),
			Context:     commonContext,
		})
	}

	return patterns
}

// normalizeActionKey creates a normalized key for grouping similar actions.
func normalizeActionKey(tool, target string) string {
	// Normalize tool name to lowercase
	tool = strings.ToLower(tool)

	// For file paths, extract the filename and normalize the path
	if strings.Contains(target, "/") {
		// Keep the last two path components for context
		parts := strings.Split(target, "/")
		if len(parts) > 2 {
			target = strings.Join(parts[len(parts)-2:], "/")
		}
	}

	return fmt.Sprintf("%s:%s", tool, target)
}

// normalizeErrorType extracts a normalized error type from an error message.
func normalizeErrorType(errorDetail string) string {
	// Common error patterns
	if strings.Contains(errorDetail, "no such file or directory") {
		return "file_not_found"
	}
	if strings.Contains(errorDetail, "permission denied") {
		return "permission_denied"
	}
	if strings.Contains(errorDetail, "timeout") {
		return "timeout"
	}
	if strings.Contains(errorDetail, "connection refused") {
		return "connection_refused"
	}

	// Default: use first 50 chars of error
	if len(errorDetail) > 50 {
		return errorDetail[:50]
	}
	return errorDetail
}

// extractCommonContext finds workspace context values that are common across all events.
func extractCommonContext(events []ActionEvent) map[string]string {
	if len(events) == 0 {
		return nil
	}

	// Start with the first event's context
	common := make(map[string]string)
	if events[0].WorkspaceContext != nil {
		for k, v := range events[0].WorkspaceContext {
			common[k] = v
		}
	}

	// Remove any key that differs across events
	for _, e := range events[1:] {
		for k, v := range common {
			if e.WorkspaceContext == nil || e.WorkspaceContext[k] != v {
				delete(common, k)
			}
		}
	}

	return common
}

// filterSuppressedPatterns removes patterns that have been suppressed.
func (l *ActionLog) filterSuppressedPatterns(patterns []Pattern) []Pattern {
	suppressedKeys := make(map[string]bool)
	for _, sp := range l.SuppressedPatterns {
		suppressedKeys[sp.PatternKey] = true
	}

	filtered := []Pattern{}
	for _, p := range patterns {
		key := patternKey(p)
		if !suppressedKeys[key] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// patternKey generates a unique key for a pattern for suppression matching.
func patternKey(p Pattern) string {
	if len(p.Events) == 0 {
		return p.Type
	}
	return fmt.Sprintf("%s:%s:%s", p.Type, p.Events[0].Tool, p.Events[0].Target)
}

// SuppressPattern adds a pattern to the suppression list.
func (l *ActionLog) SuppressPattern(p Pattern, reason string, duration time.Duration) {
	key := patternKey(p)

	sp := SuppressedPattern{
		PatternKey:   key,
		SuppressedAt: time.Now().UTC(),
		Reason:       reason,
	}

	if duration > 0 {
		sp.ExpiresAt = time.Now().UTC().Add(duration)
	}

	// Update existing or add new
	for i, existing := range l.SuppressedPatterns {
		if existing.PatternKey == key {
			l.SuppressedPatterns[i] = sp
			return
		}
	}
	l.SuppressedPatterns = append(l.SuppressedPatterns, sp)
}

// severityOrder returns numeric ordering for severities.
func severityOrder(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "warning":
		return 1
	case "info":
		return 2
	default:
		return 3
	}
}

// FormatPatterns formats detected patterns for display.
func FormatPatterns(patterns []Pattern) string {
	if len(patterns) == 0 {
		return "No behavioral patterns detected.\n"
	}

	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("==================================================================================\n")
	sb.WriteString("  BEHAVIORAL PATTERNS DETECTED\n")
	sb.WriteString("==================================================================================\n")

	for i, p := range patterns {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("\n  ... and %d more patterns\n", len(patterns)-10))
			break
		}

		icon := "○"
		if p.Severity == "critical" {
			icon = "●"
		} else if p.Severity == "warning" {
			icon = "◐"
		}

		sb.WriteString(fmt.Sprintf("\n  %s [%s] %s\n", icon, p.Severity, p.Description))

		// Show context if available
		if len(p.Context) > 0 {
			sb.WriteString("     Context:")
			for k, v := range p.Context {
				sb.WriteString(fmt.Sprintf(" %s=%s", k, v))
			}
			sb.WriteString("\n")
		}

		if p.Suggestion != "" {
			sb.WriteString(fmt.Sprintf("     -> %s\n", p.Suggestion))
		}
	}

	sb.WriteString("\n==================================================================================\n")
	sb.WriteString("  Run 'orch patterns suppress <index>' to suppress a pattern\n")
	sb.WriteString("==================================================================================\n")

	return sb.String()
}

// Summary returns a brief summary of the action log state.
func (l *ActionLog) Summary() string {
	if len(l.Events) == 0 {
		return "No actions logged yet"
	}

	patterns := l.DetectPatterns()
	return fmt.Sprintf("%d action events logged, %d behavioral patterns detected", len(l.Events), len(patterns))
}

// GetRecentEvents returns the most recent N events.
func (l *ActionLog) GetRecentEvents(n int) []ActionEvent {
	// Events are already sorted newest first by pruneOldEvents
	if len(l.Events) <= n {
		return l.Events
	}
	return l.Events[:n]
}

// ClearEvents removes all events from the log (useful for testing or reset).
func (l *ActionLog) ClearEvents() {
	l.Events = []ActionEvent{}
}
