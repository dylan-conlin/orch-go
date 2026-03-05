package main

import (
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// AgentAPIResponse is the JSON structure returned by /api/agents.
type AgentAPIResponse struct {
	ID                   string               `json:"id"`
	SessionID            string               `json:"session_id,omitempty"`
	BeadsID              string               `json:"beads_id,omitempty"`
	BeadsTitle           string               `json:"beads_title,omitempty"`
	Skill                string               `json:"skill,omitempty"`
	Tier                 string               `json:"tier,omitempty"`
	Status               string               `json:"status"`                      // "active", "idle", "dead", "completed", "awaiting-cleanup"
	Phase                string               `json:"phase,omitempty"`             // "Planning", "Implementing", "Complete", etc.
	PhaseReportedAt      string               `json:"phase_reported_at,omitempty"` // ISO 8601 timestamp when phase was reported
	Task                 string               `json:"task,omitempty"`              // Task description from beads issue
	Project              string               `json:"project,omitempty"`           // Project name (orch-go, skillc, etc.)
	Runtime              string               `json:"runtime,omitempty"`
	Window               string               `json:"window,omitempty"`
	IsProcessing         bool                 `json:"is_processing,omitempty"` // True if actively generating response
	IsStale              bool                 `json:"is_stale,omitempty"`      // True if agent is older than beadsFetchThreshold (beads data not fetched)
	IsStalled            bool                 `json:"is_stalled,omitempty"`    // True if active agent has same phase for 15+ minutes (advisory)
	IsUnresponsive       bool                 `json:"is_unresponsive,omitempty"` // True if no phase update for 30+ minutes
	SpawnedAt            string               `json:"spawned_at,omitempty"`    // ISO 8601 timestamp
	UpdatedAt            string               `json:"updated_at,omitempty"`    // ISO 8601 timestamp
	Synthesis            *SynthesisResponse   `json:"synthesis,omitempty"`
	CloseReason          string               `json:"close_reason,omitempty"`          // Beads close reason, fallback when synthesis is null
	GapAnalysis          *GapAPIResponse      `json:"gap_analysis,omitempty"`          // Context gap analysis from spawn time
	Tokens               *opencode.TokenStats `json:"tokens,omitempty"`                // Token usage for the session
	InvestigationPath    string               `json:"investigation_path,omitempty"`    // Path to investigation file from beads comments
	ProjectDir           string               `json:"project_dir,omitempty"`           // Project directory for the agent
	SynthesisContent     string               `json:"synthesis_content,omitempty"`     // Raw SYNTHESIS.md content for inline rendering
	InvestigationContent string               `json:"investigation_content,omitempty"` // Raw investigation file content for inline rendering
	CurrentActivity      string               `json:"current_activity,omitempty"`      // Last activity text from session messages
	LastActivityAt       string               `json:"last_activity_at,omitempty"`      // ISO 8601 timestamp of last activity
	EscalationLevel      string                        `json:"escalation_level,omitempty"`      // none, info, review, block, failed — server-computed escalation
	Reason               string                        `json:"reason,omitempty"`                // Reason code for degraded/partial state (from query engine)
	ContextRisk          *verify.ContextExhaustionRisk `json:"context_risk,omitempty"`          // Context exhaustion risk assessment
}

// GapAPIResponse represents gap analysis data for the API.
type GapAPIResponse struct {
	HasGaps        bool `json:"has_gaps"`
	ContextQuality int  `json:"context_quality"`
	ShouldWarn     bool `json:"should_warn"`
	MatchCount     int  `json:"match_count,omitempty"`
	Constraints    int  `json:"constraints,omitempty"`
	Decisions      int  `json:"decisions,omitempty"`
	Investigations int  `json:"investigations,omitempty"`
}

// SynthesisResponse is a condensed version of verify.Synthesis for the API.
// Uses the D.E.K.N. structure: Delta, Evidence, Knowledge, Next.
type SynthesisResponse struct {
	// Header fields
	TLDR           string `json:"tldr,omitempty"`
	Outcome        string `json:"outcome,omitempty"`        // success, partial, blocked, failed
	Recommendation string `json:"recommendation,omitempty"` // close, continue, escalate

	// Condensed sections
	DeltaSummary string   `json:"delta_summary,omitempty"` // e.g., "3 files created, 2 modified, 5 commits"
	NextActions  []string `json:"next_actions,omitempty"`  // Follow-up items
}
