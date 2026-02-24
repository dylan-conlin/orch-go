package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// AgentStatus represents the status of a tracked agent with explicit reason codes
// for any missing or partial data. This is the output of the single-pass query engine.
//
// Design: Every missing field must have an explicit reason code.
// Never return silent empty metadata.
// See: .kb/decisions/2026-02-18-two-lane-agent-discovery.md
type AgentStatus struct {
	// Identity (from beads)
	BeadsID string `json:"beads_id"`
	Title   string `json:"title"`

	// Binding (from workspace manifest)
	SessionID     string `json:"session_id,omitempty"`
	ProjectDir    string `json:"project_dir,omitempty"`
	WorkspaceName string `json:"workspace_name,omitempty"`
	Skill         string `json:"skill,omitempty"`
	Tier          string `json:"tier,omitempty"`
	Model         string `json:"model,omitempty"`
	SpawnMode     string `json:"spawn_mode,omitempty"`

	// Phase (from beads comments)
	Phase string `json:"phase,omitempty"` // "Planning", "Implementing", "Complete", etc.

	// Liveness (from OpenCode)
	Status string `json:"status"` // "active", "idle", "retrying", "unknown"

	// Reason codes for missing/partial data
	MissingBinding bool `json:"missing_binding,omitempty"` // Workspace manifest not found
	MissingSession bool `json:"missing_session,omitempty"` // No session ID in manifest
	SessionDead    bool `json:"session_dead,omitempty"`    // Session exists but idle/errored
	MissingPhase   bool `json:"missing_phase,omitempty"`   // No Phase comment in beads

	// Human-readable explanation for degraded state
	Reason string `json:"reason,omitempty"`
}

// queryTrackedAgents implements the single-pass query engine for tracked work.
// It queries beads (source of truth for what work exists), workspace manifests
// (for binding), and OpenCode (for liveness), then joins with explicit reason codes.
//
// Degraded modes:
//   - OpenCode down → agents shown with status=unknown, reason=opencode_unreachable
//   - Workspace missing → agent shown with reason=missing_binding
//   - No session ID → agent shown with reason=missing_session
//
// This function never returns silent empty metadata.
func queryTrackedAgents(projectDirs []string) ([]AgentStatus, error) {
	// Step 1: Start from beads (source of truth for what work exists)
	issues, err := listTrackedIssues()
	if err != nil {
		return nil, fmt.Errorf("beads query failed: %w", err)
	}
	if len(issues) == 0 {
		return nil, nil
	}

	// Step 2: Batch lookup workspace bindings
	beadsIDs := make([]string, len(issues))
	for i, issue := range issues {
		beadsIDs[i] = issue.ID
	}
	manifests, err := lookupManifestsAcrossProjects(projectDirs, beadsIDs)
	if err != nil {
		// Workspace scan failure is not fatal - agents will show with missing_binding
		log.Printf("Warning: workspace lookup failed: %v", err)
		manifests = make(map[string]*spawn.AgentManifest)
	}

	// Step 3: Extract latest phase from beads comments
	phases := extractLatestPhases(beadsIDs)

	// Step 4: Batch check session liveness
	sessionIDs := extractSessionIDs(manifests)
	var liveness map[string]opencode.SessionStatusInfo
	if len(sessionIDs) > 0 {
		client := opencode.NewClient(opencode.DefaultServerURL)
		liveness, err = client.GetSessionStatusByIDs(sessionIDs)
		if err != nil {
			// OpenCode down: agents shown with status=unknown
			log.Printf("Warning: OpenCode unreachable: %v", err)
			liveness = unknownLiveness(sessionIDs)
		}
	}

	// Step 5: Join with explicit reason codes
	return joinWithReasonCodes(issues, manifests, liveness, phases), nil
}

// listTrackedIssues queries beads for all in-progress issues tagged with orch:agent.
// This is the entry point for the tracked work lane.
func listTrackedIssues() ([]beads.Issue, error) {
	// Try RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			issues, err := client.List(&beads.ListArgs{
				LabelsAny: []string{"orch:agent"},
				Limit:     0,
			})
			if err == nil {
				return filterActiveIssues(issues), nil
			}
			// Fall through to CLI fallback
		}
	}

	// CLI fallback
	return listTrackedIssuesCLI()
}

// listTrackedIssuesCLI queries beads via bd CLI for orch:agent tagged issues.
var fallbackListWithLabelFn = beads.FallbackListWithLabel

func listTrackedIssuesCLI() ([]beads.Issue, error) {
	issues, err := fallbackListWithLabelFn("orch:agent")
	if err != nil {
		return nil, err
	}
	return filterActiveIssues(issues), nil
}

// filterActiveIssues returns only issues with active statuses (open, in_progress).
func filterActiveIssues(issues []beads.Issue) []beads.Issue {
	var active []beads.Issue
	for _, issue := range issues {
		switch strings.ToLower(issue.Status) {
		case "open", "in_progress":
			active = append(active, issue)
		}
	}
	return active
}

// lookupManifestsAcrossProjects scans workspace directories across all project dirs
// for manifests matching the given beads IDs.
func lookupManifestsAcrossProjects(projectDirs []string, beadsIDs []string) (map[string]*spawn.AgentManifest, error) {
	if len(projectDirs) == 0 || len(beadsIDs) == 0 {
		return nil, nil
	}

	combined := make(map[string]*spawn.AgentManifest)
	for _, dir := range projectDirs {
		manifests, err := spawn.LookupManifestsByBeadsIDs(dir, beadsIDs)
		if err != nil {
			log.Printf("Warning: failed to scan workspace in %s: %v", dir, err)
			continue
		}
		for id, m := range manifests {
			if _, exists := combined[id]; !exists {
				combined[id] = m
			}
		}
	}
	return combined, nil
}

// extractSessionIDs collects non-empty session IDs from manifests.
func extractSessionIDs(manifests map[string]*spawn.AgentManifest) []string {
	if manifests == nil {
		return nil
	}
	var ids []string
	for _, m := range manifests {
		if m.SessionID != "" {
			ids = append(ids, m.SessionID)
		}
	}
	return ids
}

// extractLatestPhases fetches beads comments for each issue and extracts the
// most recent "Phase:" comment. Returns a map of beadsID → phase string.
// Comment fetch failures are non-fatal: issues with failed fetches simply
// won't appear in the map (and will get MissingPhase=true in the join).
func extractLatestPhases(beadsIDs []string) map[string]string {
	phases := make(map[string]string, len(beadsIDs))

	// Try RPC client first for batch efficiency
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(1))
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			for _, id := range beadsIDs {
				comments, err := client.Comments(id)
				if err != nil {
					continue
				}
				if phase := latestPhaseFromComments(comments); phase != "" {
					phases[id] = phase
				}
			}
			return phases
		}
	}

	// CLI fallback
	for _, id := range beadsIDs {
		comments, err := beads.FallbackComments(id)
		if err != nil {
			continue
		}
		if phase := latestPhaseFromComments(comments); phase != "" {
			phases[id] = phase
		}
	}
	return phases
}

// latestPhaseFromComments extracts the phase from the most recent "Phase:" comment.
// Returns the full phase text (e.g., "Implementing - Adding auth middleware").
// Returns "" if no phase comment found.
func latestPhaseFromComments(comments []beads.Comment) string {
	for i := len(comments) - 1; i >= 0; i-- {
		if strings.HasPrefix(comments[i].Text, "Phase:") {
			return strings.TrimSpace(strings.TrimPrefix(comments[i].Text, "Phase:"))
		}
	}
	return ""
}

// unknownLiveness creates a liveness map where every session has "unknown" status.
// Used as degraded mode when OpenCode is unreachable.
func unknownLiveness(sessionIDs []string) map[string]opencode.SessionStatusInfo {
	result := make(map[string]opencode.SessionStatusInfo, len(sessionIDs))
	for _, id := range sessionIDs {
		result[id] = opencode.SessionStatusInfo{Type: "unknown"}
	}
	return result
}

// joinWithReasonCodes merges beads issues, workspace manifests, session liveness,
// and phase data into AgentStatus structs with explicit reason codes for any missing data.
//
// This is the core of the single-pass query engine. Every missing field gets
// a reason code so failures are never silent.
func joinWithReasonCodes(
	issues []beads.Issue,
	manifests map[string]*spawn.AgentManifest,
	liveness map[string]opencode.SessionStatusInfo,
	phases map[string]string,
) []AgentStatus {
	if len(issues) == 0 {
		return nil
	}

	results := make([]AgentStatus, 0, len(issues))

	for _, issue := range issues {
		agent := AgentStatus{
			BeadsID: issue.ID,
			Title:   issue.Title,
		}

		// Populate phase from beads comments
		if phase, ok := phases[issue.ID]; ok && phase != "" {
			agent.Phase = phase
		} else {
			agent.MissingPhase = true
		}

		// Step 1: Look up workspace manifest binding
		manifest, hasBind := manifests[issue.ID]
		if !hasBind || manifest == nil {
			agent.MissingBinding = true
			agent.Status = "unknown"
			agent.Reason = "missing_binding"
			results = append(results, agent)
			continue
		}

		// Populate from manifest
		agent.SessionID = manifest.SessionID
		agent.ProjectDir = manifest.ProjectDir
		agent.WorkspaceName = manifest.WorkspaceName
		agent.Skill = manifest.Skill
		agent.Tier = manifest.Tier
		agent.Model = manifest.Model
		agent.SpawnMode = manifest.SpawnMode

		// Step 2: Check session ID
		if manifest.SessionID == "" {
			// Claude-backend agents don't have OpenCode sessions.
			// Use phase comments as heartbeat (beads-based liveness).
			// See: .kb/investigations/2026-02-24-design-dashboard-oscillation-tmux-liveness-architectural-analysis.md
			if manifest.SpawnMode == "claude" && manifest.WorkspaceName != "" {
				spawnTime := manifest.ParseSpawnTime()

				if agent.Phase != "" && strings.HasPrefix(agent.Phase, "Complete") {
					agent.Status = "completed"
					agent.Reason = "phase_complete"
				} else if agent.Phase != "" {
					agent.Status = "active"
					agent.Reason = "phase_reported"
				} else if !spawnTime.IsZero() && time.Since(spawnTime) < 5*time.Minute {
					agent.Status = "active"
					agent.Reason = "recently_spawned"
				} else {
					agent.Status = "dead"
					agent.Reason = "no_phase_reported"
				}
				results = append(results, agent)
				continue
			}
			agent.MissingSession = true
			agent.Status = "unknown"
			agent.Reason = "missing_session"
			results = append(results, agent)
			continue
		}

		// Step 3: Check session liveness
		statusInfo, hasLiveness := liveness[manifest.SessionID]
		if !hasLiveness {
			// Session not in liveness map = idle in OpenCode (sessions not actively
			// processing are removed from the in-memory status map)
			agent.SessionDead = true
			agent.Status = "idle"
			agent.Reason = "session_idle"
			results = append(results, agent)
			continue
		}

		switch statusInfo.Type {
		case "busy":
			agent.Status = "active"
			// No reason code needed - fully healthy
		case "idle":
			agent.SessionDead = true
			agent.Status = "idle"
			agent.Reason = "session_idle"
		case "retry":
			agent.Status = "retrying"
			agent.Reason = "session_retrying"
		case "unknown":
			// OpenCode was unreachable - unknownLiveness was used
			agent.Status = "unknown"
			agent.Reason = "opencode_unreachable"
		default:
			agent.Status = "unknown"
			agent.Reason = fmt.Sprintf("unexpected_status_%s", statusInfo.Type)
		}

		results = append(results, agent)
	}

	return results
}
