package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// ErrorEvent represents an error event for the API.
type ErrorEvent struct {
	Type       string `json:"type"`                  // "session.error" or "agent.abandoned"
	SessionID  string `json:"session_id,omitempty"`  // Session ID if available
	BeadsID    string `json:"beads_id,omitempty"`    // Beads issue ID if available
	Timestamp  string `json:"timestamp"`             // ISO 8601 timestamp
	Message    string `json:"message,omitempty"`     // Error message or abandon reason
	Workspace  string `json:"workspace,omitempty"`   // Workspace path if available
	Skill      string `json:"skill,omitempty"`       // Skill type if known
	RecurCount int    `json:"recur_count,omitempty"` // How many times this error pattern has occurred
}

// ErrorPattern represents a recurring error pattern.
type ErrorPattern struct {
	Pattern    string   `json:"pattern"`    // Error message pattern (may be truncated/normalized)
	Count      int      `json:"count"`      // Number of occurrences
	LastSeen   string   `json:"last_seen"`  // ISO 8601 timestamp of most recent occurrence
	BeadsIDs   []string `json:"beads_ids"`  // Affected beads issues
	Suggestion string   `json:"suggestion"` // Remediation suggestion
}

// ErrorsAPIResponse is the JSON structure returned by /api/errors.
type ErrorsAPIResponse struct {
	TotalErrors    int            `json:"total_errors"`            // Total error events
	ErrorsLast24h  int            `json:"errors_last_24h"`         // Errors in last 24 hours
	ErrorsLast7d   int            `json:"errors_last_7d"`          // Errors in last 7 days
	AbandonedCount int            `json:"abandoned_count"`         // Total agent.abandoned events
	SessionErrors  int            `json:"session_errors"`          // Total session.error events
	RecentErrors   []ErrorEvent   `json:"recent_errors,omitempty"` // Last 20 error events
	Patterns       []ErrorPattern `json:"patterns,omitempty"`      // Recurring error patterns
	ByType         map[string]int `json:"by_type"`                 // Breakdown by error type
	Error          string         `json:"error,omitempty"`         // Error message if any
}

// handleErrors returns error pattern analysis from ~/.orch/events.jsonl.
func (s *Server) handleErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logPath := events.DefaultLogPath()

	now := time.Now()
	day24h := now.Add(-24 * time.Hour)
	days7 := now.Add(-7 * 24 * time.Hour)

	var allErrors []ErrorEvent
	byType := make(map[string]int)
	patternCounts := make(map[string]*ErrorPattern)

	err := events.ReadCompactedJSONL(logPath, func(line string) error {
		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil
		}

		// Only process error-related events
		if event.Type != events.EventTypeSessionError && event.Type != "agent.abandoned" {
			return nil
		}

		ts := time.Unix(event.Timestamp, 0)
		errorEvent := ErrorEvent{
			Type:      event.Type,
			SessionID: event.SessionID,
			Timestamp: ts.Format(time.RFC3339),
		}

		// Extract details based on event type
		if event.Type == events.EventTypeSessionError {
			if msg, ok := event.Data["error"].(string); ok {
				errorEvent.Message = msg
			}
		} else if event.Type == "agent.abandoned" {
			if reason, ok := event.Data["reason"].(string); ok {
				errorEvent.Message = reason
			}
			if beadsID, ok := event.Data["beads_id"].(string); ok {
				errorEvent.BeadsID = beadsID
			}
			if workspace, ok := event.Data["workspace_path"].(string); ok {
				errorEvent.Workspace = filepath.Base(workspace)
			}
			if agentID, ok := event.Data["agent_id"].(string); ok {
				// Extract skill from agent_id pattern like "og-feat-xxx" or "og-debug-xxx"
				errorEvent.Skill = extractSkillFromAgentID(agentID)
			}
		}

		allErrors = append(allErrors, errorEvent)
		byType[event.Type]++

		// Track patterns for recurring error detection
		patternKey := normalizeErrorMessage(errorEvent.Message)
		if patternKey != "" {
			if p, exists := patternCounts[patternKey]; exists {
				p.Count++
				p.LastSeen = errorEvent.Timestamp
				if errorEvent.BeadsID != "" && !containsString(p.BeadsIDs, errorEvent.BeadsID) {
					p.BeadsIDs = append(p.BeadsIDs, errorEvent.BeadsID)
				}
			} else {
				beadsIDs := []string{}
				if errorEvent.BeadsID != "" {
					beadsIDs = append(beadsIDs, errorEvent.BeadsID)
				}
				patternCounts[patternKey] = &ErrorPattern{
					Pattern:  patternKey,
					Count:    1,
					LastSeen: errorEvent.Timestamp,
					BeadsIDs: beadsIDs,
				}
			}
		}
		return nil
	})

	if err != nil {
		if os.IsNotExist(err) {
			// Return empty response if file doesn't exist
			resp := ErrorsAPIResponse{
				ByType: make(map[string]int),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		resp := ErrorsAPIResponse{Error: fmt.Sprintf("Failed to read events file: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Count errors by time window
	var errorsLast24h, errorsLast7d int
	for _, e := range allErrors {
		ts, _ := time.Parse(time.RFC3339, e.Timestamp)
		if ts.After(day24h) {
			errorsLast24h++
		}
		if ts.After(days7) {
			errorsLast7d++
		}
	}

	// Get recent errors (last 20, most recent first)
	recentErrors := allErrors
	if len(recentErrors) > 20 {
		recentErrors = recentErrors[len(recentErrors)-20:]
	}
	// Reverse to show most recent first
	for i, j := 0, len(recentErrors)-1; i < j; i, j = i+1, j-1 {
		recentErrors[i], recentErrors[j] = recentErrors[j], recentErrors[i]
	}

	// Convert patterns map to slice and sort by count
	var patterns []ErrorPattern
	for _, p := range patternCounts {
		if p.Count >= 2 { // Only include patterns that occurred 2+ times
			p.Suggestion = suggestRemediation(p.Pattern)
			patterns = append(patterns, *p)
		}
	}
	// Sort by count descending
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Count > patterns[i].Count {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}
	// Limit to top 10 patterns
	if len(patterns) > 10 {
		patterns = patterns[:10]
	}

	resp := ErrorsAPIResponse{
		TotalErrors:    len(allErrors),
		ErrorsLast24h:  errorsLast24h,
		ErrorsLast7d:   errorsLast7d,
		AbandonedCount: byType["agent.abandoned"],
		SessionErrors:  byType[events.EventTypeSessionError],
		RecentErrors:   recentErrors,
		Patterns:       patterns,
		ByType:         byType,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode errors: %v", err), http.StatusInternalServerError)
		return
	}
}

// extractSkillFromAgentID extracts the skill type from an agent ID.
// Agent IDs have patterns like "og-feat-xxx", "og-debug-xxx", "og-inv-xxx".
func extractSkillFromAgentID(agentID string) string {
	parts := strings.Split(agentID, "-")
	if len(parts) < 2 {
		return ""
	}
	// Map short prefixes to skill names
	switch parts[1] {
	case "feat":
		return "feature-impl"
	case "debug":
		return "systematic-debugging"
	case "inv":
		return "investigation"
	case "arch":
		return "architect"
	case "work":
		return "design-session"
	default:
		return parts[1]
	}
}

// normalizeErrorMessage normalizes an error message for pattern matching.
// Removes specific identifiers to group similar errors together.
func normalizeErrorMessage(msg string) string {
	if msg == "" {
		return ""
	}
	// Truncate long messages
	if len(msg) > 100 {
		msg = msg[:100]
	}
	// Simple normalization - could be enhanced with regex for IDs, paths, etc.
	return strings.TrimSpace(msg)
}

// containsString checks if a string is in a slice.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// suggestRemediation provides a remediation suggestion based on error pattern.
func suggestRemediation(pattern string) string {
	lower := strings.ToLower(pattern)
	switch {
	case strings.Contains(lower, "stall"):
		return "Check agent for long-running operations or API timeouts"
	case strings.Contains(lower, "timeout"):
		return "Review API response times or increase timeout limits"
	case strings.Contains(lower, "capacity"):
		return "Increase daemon capacity or check for stuck agents"
	case strings.Contains(lower, "daemon"):
		return "Check daemon logs at ~/.orch/daemon.log"
	case strings.Contains(lower, "context"):
		return "Review spawn context for missing or incorrect information"
	case strings.Contains(lower, "connection"):
		return "Check network connectivity or API endpoint availability"
	default:
		return "Review agent workspace for more details"
	}
}
