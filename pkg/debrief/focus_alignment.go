package debrief

import (
	"fmt"
	"strings"
)

// FocusAlignmentData holds focus alignment information for a debrief.
type FocusAlignmentData struct {
	Verdict      string   `json:"verdict"`        // on-track, drifting, unverified, no-focus
	Goal         string   `json:"goal"`           // The focus goal
	FocusedIssue string   `json:"focused_issue,omitempty"` // Beads ID if focus targets a specific issue
	Reason       string   `json:"reason"`         // Human-readable explanation of verdict
	ActiveWork   []string `json:"active_work,omitempty"`   // Formatted active work items
}

// CollectFocusAlignment builds focus alignment data from a verdict and active work.
// This is a pure data transformation — the caller is responsible for obtaining
// the drift verdict from the focus package.
func CollectFocusAlignment(verdict, goal, focusedIssue, reason string, activeWork []string) *FocusAlignmentData {
	if verdict == "no-focus" || verdict == "" {
		return nil
	}

	return &FocusAlignmentData{
		Verdict:      verdict,
		Goal:         goal,
		FocusedIssue: focusedIssue,
		Reason:       reason,
		ActiveWork:   activeWork,
	}
}

// FormatFocusAlignment formats focus alignment data into debrief lines.
func FormatFocusAlignment(data *FocusAlignmentData) []string {
	if data == nil {
		return nil
	}

	var lines []string

	// Verdict line with indicator
	var indicator string
	switch data.Verdict {
	case "on-track":
		indicator = "ON TRACK"
	case "drifting":
		indicator = "DRIFTING"
	case "unverified":
		indicator = "UNVERIFIED"
	default:
		indicator = strings.ToUpper(data.Verdict)
	}

	lines = append(lines, fmt.Sprintf("**%s** — %s", indicator, data.Reason))

	// Focused issue if present
	if data.FocusedIssue != "" {
		lines = append(lines, fmt.Sprintf("Focused issue: %s", data.FocusedIssue))
	}

	// Active work items
	if len(data.ActiveWork) > 0 {
		for _, work := range data.ActiveWork {
			lines = append(lines, fmt.Sprintf("Active: %s", work))
		}
	}

	return lines
}
