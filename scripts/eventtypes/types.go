package eventtypes

// SessionSpawned represents a session.spawned event from events.jsonl.
type SessionSpawned struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Timestamp int64  `json:"timestamp"`
	Data      struct {
		BeadsID                string   `json:"beads_id"`
		Model                  string   `json:"model"`
		Skill                  string   `json:"skill"`
		Task                   string   `json:"task"`
		SpawnMode              string   `json:"spawn_mode"`
		NoTrack                bool     `json:"no_track"`
		GapContextQuality      int      `json:"gap_context_quality"`
		GapHasGaps             bool     `json:"gap_has_gaps"`
		SkipArtifactCheck      bool     `json:"skip_artifact_check"`
		Workspace              string   `json:"workspace"`
		GapMatchTotal          int      `json:"gap_match_total"`
		GapMatchConstraints    int      `json:"gap_match_constraints"`
		GapMatchDecisions      int      `json:"gap_match_decisions"`
		GapMatchInvestigations int      `json:"gap_match_investigations"`
		GapTypes               []string `json:"gap_types"`
	} `json:"data"`
}

// AgentCompleted represents an agent.completed event from events.jsonl.
type AgentCompleted struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Timestamp int64  `json:"timestamp"`
	Data      struct {
		BeadsID            string `json:"beads_id"`
		Outcome            string `json:"outcome"`
		Skill              string `json:"skill"`
		VerificationPassed bool   `json:"verification_passed"`
		TokensInput        int    `json:"tokens_input"`
		TokensOutput       int    `json:"tokens_output"`
		Workspace          string `json:"workspace"`
		Reason             string `json:"reason"`
		Orchestrator       bool   `json:"orchestrator"`
		Untracked          bool   `json:"untracked"`
	} `json:"data"`
}

// AgentAbandoned represents an agent.abandoned event from events.jsonl.
type AgentAbandoned struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Timestamp int64  `json:"timestamp"`
	Data      struct {
		BeadsID   string `json:"beads_id"`
		Reason    string `json:"reason"`
		Workspace string `json:"workspace"`
	} `json:"data"`
}
