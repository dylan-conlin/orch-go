// Package attention provides types and interfaces for the composable attention architecture.
// It defines the AttentionItem struct for representing attention signals, the Collector interface
// for signal sources, and the ConcernType enum for categorizing attention types.
package attention

import "time"

// ConcernType categorizes the type of attention signal.
type ConcernType int

const (
	// Observability signals provide state information about the system.
	// Example: "Issue has commits mentioning it"
	Observability ConcernType = iota

	// Actionability signals indicate something can be acted on right now.
	// Example: "Issue ready for work (no blockers)"
	Actionability

	// Authority signals indicate something requires a specific actor.
	// Example: "Agent stuck >2h (needs human intervention)"
	Authority
)

// String returns the string representation of a ConcernType.
func (c ConcernType) String() string {
	switch c {
	case Observability:
		return "Observability"
	case Actionability:
		return "Actionability"
	case Authority:
		return "Authority"
	default:
		return "Unknown"
	}
}

// AttentionItem represents a single attention signal in the unified attention model.
// It is the normalized output produced by signal collectors and consumed by the Work Graph
// and other attention surfaces.
type AttentionItem struct {
	// ID is a unique identifier for this attention item.
	ID string `json:"id"`

	// Source identifies the collector that produced this item.
	// Examples: "beads", "git", "session", "kb"
	Source string `json:"source"`

	// Concern categorizes the type of attention signal.
	Concern ConcernType `json:"concern"`

	// Signal is a human-readable signal type.
	// Examples: "issue-ready", "agent-stuck", "commit-evidence"
	Signal string `json:"signal"`

	// Subject identifies what needs attention.
	// Examples: issue ID, session ID, file path
	Subject string `json:"subject"`

	// Summary is a one-line description of what needs attention.
	Summary string `json:"summary"`

	// Priority is a role-specific priority score.
	// Lower numbers indicate higher priority.
	Priority int `json:"priority"`

	// Role identifies the intended audience for this item.
	// Examples: "human", "orchestrator", "daemon"
	Role string `json:"role"`

	// ActionHint suggests an action to take.
	// Examples: "orch complete X", "review decision"
	ActionHint string `json:"action_hint,omitempty"`

	// CollectedAt is the timestamp when this item was collected.
	CollectedAt time.Time `json:"collected_at"`

	// Metadata holds signal-specific additional data.
	// The structure varies by signal type.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Collector is the interface that signal sources must implement.
// Each collector is responsible for gathering attention signals from a specific source
// (beads, git, sessions, kb, etc.) and normalizing them into AttentionItems.
type Collector interface {
	// Collect gathers attention items for the specified role.
	// The role parameter allows collectors to compute role-aware priority scores.
	// Returns a slice of AttentionItems or an error if collection fails.
	Collect(role string) ([]AttentionItem, error)
}
