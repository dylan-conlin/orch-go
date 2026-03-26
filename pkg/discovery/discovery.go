// Package discovery provides the canonical backend-aware agent query interface.
//
// This package prevents Class 2 (Multi-Backend Blindness) defects by providing
// a single entry point that dispatches to both OpenCode and Claude CLI backends
// and merges results. Callers should use this package instead of querying
// OpenCode or tmux directly for agent discovery.
//
// See: .kb/investigations/2026-03-03-inv-catalogue-unnamed-defect-classes-orch.md
package discovery

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/execution"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// neverStartedThreshold is the absolute time after spawn beyond which an agent
// with no phase comment is considered never-started, regardless of tmux window state.
// This prevents dead agents from being masked by live tmux windows indefinitely.
const neverStartedThreshold = 10 * time.Minute

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
	Phase           string     `json:"phase,omitempty"`
	PhaseReportedAt *time.Time `json:"phase_reported_at,omitempty"`

	// Liveness (from OpenCode or tmux, depending on backend)
	Status string `json:"status"` // "active", "idle", "retrying", "completed", "dead", "unknown"

	// Composed liveness (from tmux pane or OpenCode session)
	TmuxWindowID string `json:"tmux_window_id,omitempty"` // Tmux window ID for Claude agents, if window exists
	IsProcessing bool   `json:"is_processing,omitempty"`  // True if agent is actively generating output

	// Reason codes for missing/partial data
	MissingBinding bool `json:"missing_binding,omitempty"`
	MissingSession bool `json:"missing_session,omitempty"`
	SessionDead    bool `json:"session_dead,omitempty"`
	MissingPhase   bool `json:"missing_phase,omitempty"`
	NeverStarted   bool `json:"never_started,omitempty"` // True if agent was spawned 10+ min ago with no phase comment

	// Human-readable explanation for degraded state
	Reason string `json:"reason,omitempty"`
}

// CheckTmuxWindow looks up a tmux window for the given workspace and returns
// its window ID (e.g., "@1234") if found. Returns ("", false) if no window exists.
// This is a package-level variable to allow test mocking.
var CheckTmuxWindow = func(workspaceName, projectDir string) (windowID string, alive bool) {
	if workspaceName == "" || projectDir == "" {
		return "", false
	}
	projectName := filepath.Base(projectDir)
	sessionName := tmux.GetWorkersSessionName(projectName)
	window, _ := tmux.FindWindowByWorkspaceName(sessionName, workspaceName)
	if window == nil {
		return "", false
	}
	return window.ID, true
}

// CheckPaneActive checks if a tmux pane has an active (non-shell) process.
// This is a package-level variable to allow test mocking.
var CheckPaneActive = tmux.IsPaneActive

// QueryTrackedAgents implements the single-pass query engine for tracked work.
// It queries beads (source of truth), workspace manifests (binding), OpenCode
// (liveness for headless agents), and beads comments (phase), then joins with
// explicit reason codes.
//
// This is the canonical entry point for agent discovery. Use this instead of
// querying OpenCode or tmux directly.
func QueryTrackedAgents(projectDirs []string) ([]AgentStatus, error) {
	issues, err := ListTrackedIssues(projectDirs)
	if err != nil {
		return nil, fmt.Errorf("beads query failed: %w", err)
	}
	if len(issues) == 0 {
		return nil, nil
	}

	beadsIDs := make([]string, len(issues))
	for i, issue := range issues {
		beadsIDs[i] = issue.ID
	}

	manifests, err := LookupManifestsAcrossProjects(projectDirs, beadsIDs)
	if err != nil {
		log.Printf("Warning: workspace lookup failed: %v", err)
		manifests = make(map[string]*spawn.AgentManifest)
	}

	phases, phaseTimestamps := ExtractLatestPhases(beadsIDs)

	sessionIDs := ExtractSessionIDs(manifests)
	ctx := context.Background()
	var liveness map[string]execution.SessionStatusInfo
	if len(sessionIDs) > 0 {
		client := execution.NewOpenCodeAdapter(execution.DefaultServerURL)
		if !client.IsReachable(ctx) {
			liveness = UnknownLiveness(sessionIDs)
		} else {
			liveness, err = client.GetSessionStatusByIDs(ctx, sessionIDs)
			if err != nil {
				log.Printf("Warning: OpenCode session status query failed: %v", err)
				liveness = UnknownLiveness(sessionIDs)
			}
		}
	}

	return JoinWithReasonCodes(issues, manifests, liveness, phases, phaseTimestamps), nil
}

// ListTrackedIssues queries beads for all in-progress issues tagged with orch:agent
// across all known project directories.
func ListTrackedIssues(projectDirs []string) ([]beads.Issue, error) {
	issues, err := listTrackedIssuesLocal()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(issues))
	for _, issue := range issues {
		seen[issue.ID] = true
	}

	for _, dir := range projectDirs {
		if dir == "" {
			continue
		}
		projectIssues, err := listTrackedIssuesForDir(dir)
		if err != nil {
			log.Printf("Warning: failed to list tracked issues for %s: %v", dir, err)
			continue
		}
		for _, issue := range projectIssues {
			if !seen[issue.ID] {
				seen[issue.ID] = true
				issues = append(issues, issue)
			}
		}
	}

	return FilterActiveIssues(issues), nil
}

func listTrackedIssuesLocal() ([]beads.Issue, error) {
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			issues, err := client.List(&beads.ListArgs{
				LabelsAny: []string{"orch:agent"},
				Limit:     beads.IntPtr(0),
			})
			if err == nil {
				return issues, nil
			}
		}
	}
	return beads.FallbackListWithLabel("orch:agent", "")
}

func listTrackedIssuesForDir(dir string) ([]beads.Issue, error) {
	socketPath, err := beads.FindSocketPath(dir)
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			issues, err := client.List(&beads.ListArgs{
				LabelsAny: []string{"orch:agent"},
				Limit:     beads.IntPtr(0),
			})
			if err == nil {
				return issues, nil
			}
		}
	}
	return beads.FallbackListWithLabel("orch:agent", dir)
}

// FilterActiveIssues returns only issues with active statuses (open, in_progress).
func FilterActiveIssues(issues []beads.Issue) []beads.Issue {
	var active []beads.Issue
	for _, issue := range issues {
		switch strings.ToLower(issue.Status) {
		case "open", "in_progress":
			active = append(active, issue)
		}
	}
	return active
}

// LookupManifestsAcrossProjects scans workspace directories across all project dirs
// for manifests matching the given beads IDs.
func LookupManifestsAcrossProjects(projectDirs []string, beadsIDs []string) (map[string]*spawn.AgentManifest, error) {
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

// ExtractSessionIDs collects non-empty session IDs from manifests.
func ExtractSessionIDs(manifests map[string]*spawn.AgentManifest) []string {
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

// ExtractLatestPhases fetches beads comments for each issue and extracts the
// most recent "Phase:" comment.
func ExtractLatestPhases(beadsIDs []string) (map[string]string, map[string]time.Time) {
	phases := make(map[string]string, len(beadsIDs))
	timestamps := make(map[string]time.Time, len(beadsIDs))

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
				phase, ts := LatestPhaseWithTimestamp(comments)
				if phase != "" {
					phases[id] = phase
				}
				if !ts.IsZero() {
					timestamps[id] = ts
				}
			}
			return phases, timestamps
		}
	}

	for _, id := range beadsIDs {
		comments, err := beads.FallbackComments(id, "")
		if err != nil {
			continue
		}
		phase, ts := LatestPhaseWithTimestamp(comments)
		if phase != "" {
			phases[id] = phase
		}
		if !ts.IsZero() {
			timestamps[id] = ts
		}
	}
	return phases, timestamps
}

// LatestPhaseFromComments extracts the phase from the most recent "Phase:" comment.
func LatestPhaseFromComments(comments []beads.Comment) string {
	for i := len(comments) - 1; i >= 0; i-- {
		if strings.HasPrefix(comments[i].Text, "Phase:") {
			return strings.TrimSpace(strings.TrimPrefix(comments[i].Text, "Phase:"))
		}
	}
	return ""
}

// LatestPhaseWithTimestamp extracts the phase and its timestamp from the most recent
// "Phase:" comment.
func LatestPhaseWithTimestamp(comments []beads.Comment) (string, time.Time) {
	for i := len(comments) - 1; i >= 0; i-- {
		if strings.HasPrefix(comments[i].Text, "Phase:") {
			phase := strings.TrimSpace(strings.TrimPrefix(comments[i].Text, "Phase:"))
			var ts time.Time
			if comments[i].CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339, comments[i].CreatedAt); err == nil {
					ts = t
				}
			}
			return phase, ts
		}
	}
	return "", time.Time{}
}

// UnknownLiveness creates a liveness map where every session has "unknown" status.
// Used as degraded mode when OpenCode is unreachable.
func UnknownLiveness(sessionIDs []string) map[string]execution.SessionStatusInfo {
	result := make(map[string]execution.SessionStatusInfo, len(sessionIDs))
	for _, id := range sessionIDs {
		result[id] = execution.SessionStatusInfo{Type: "unknown"}
	}
	return result
}

// JoinWithReasonCodes merges beads issues, workspace manifests, session liveness,
// and phase data into AgentStatus structs with explicit reason codes for any missing data.
//
// This is the core of the backend-aware query engine. It routes by SpawnMode:
// - Claude CLI agents: use phase comments + tmux fallback for liveness
// - OpenCode agents: use OpenCode session status for liveness
func JoinWithReasonCodes(
	issues []beads.Issue,
	manifests map[string]*spawn.AgentManifest,
	liveness map[string]execution.SessionStatusInfo,
	phases map[string]string,
	phaseTimestamps ...map[string]time.Time,
) []AgentStatus {
	if len(issues) == 0 {
		return nil
	}

	var tsMap map[string]time.Time
	if len(phaseTimestamps) > 0 {
		tsMap = phaseTimestamps[0]
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

		// Populate phase timestamp
		if tsMap != nil {
			if ts, ok := tsMap[issue.ID]; ok {
				agent.PhaseReportedAt = &ts
			}
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

		// Phase: Complete overrides all backend-specific liveness checks.
		// An agent that reported Phase: Complete is done regardless of whether
		// its underlying session (OpenCode or tmux) is still technically alive.
		if agent.Phase != "" && strings.HasPrefix(agent.Phase, "Complete") {
			agent.Status = "completed"
			agent.Reason = "phase_complete"
			results = append(results, agent)
			continue
		}

		// Step 2: Route by spawn backend
		// Claude-backend agents use tmux pane liveness as the primary signal.
		// Phase is metadata (categorization), NOT a liveness signal — a stale phase
		// comment from a dead agent must not mask its death.
		// Signal priority:
		//   1. Recently spawned (<5 min) → active (grace period, tmux may not be ready)
		//   2. Never started (>10 min, no phase) → dead (overrides tmux)
		//   3. Tmux window + active pane → active (primary liveness signal)
		//   4. Tmux window + idle pane → dead (process exited, shell remains)
		//   5. No tmux window → dead (regardless of phase)
		if manifest.SpawnMode == "claude" && manifest.WorkspaceName != "" {
			spawnTime := manifest.ParseSpawnTime()

			if !spawnTime.IsZero() && time.Since(spawnTime) < 5*time.Minute {
				agent.Status = "active"
				agent.Reason = "recently_spawned"
			} else if !spawnTime.IsZero() && time.Since(spawnTime) >= neverStartedThreshold && agent.Phase == "" {
				// Agent allocated 10+ min ago, never reported phase.
				// Override tmux window check — a live window does not mean a live agent.
				agent.Status = "dead"
				agent.Reason = "never_started"
				agent.NeverStarted = true
			} else if windowID, alive := CheckTmuxWindow(manifest.WorkspaceName, manifest.ProjectDir); alive {
				// Window exists — check if pane has an active process
				agent.TmuxWindowID = windowID
				if CheckPaneActive(windowID) {
					agent.Status = "active"
					agent.Reason = "tmux_pane_active"
					agent.IsProcessing = true
				} else {
					agent.Status = "dead"
					agent.Reason = "tmux_pane_idle"
				}
			} else {
				agent.Status = "dead"
				agent.Reason = "no_tmux_window"
			}
			results = append(results, agent)
			continue
		}

		// Step 3: Check session ID for non-Claude agents
		if manifest.SessionID == "" {
			agent.MissingSession = true
			agent.Status = "unknown"
			agent.Reason = "missing_session"
			results = append(results, agent)
			continue
		}

		// Step 4: Check session liveness via OpenCode
		statusInfo, hasLiveness := liveness[manifest.SessionID]
		if !hasLiveness {
			agent.SessionDead = true
			agent.Status = "idle"
			agent.Reason = "session_idle"
			results = append(results, agent)
			continue
		}

		switch statusInfo.Type {
		case "busy":
			agent.Status = "active"
			agent.IsProcessing = true
		case "idle":
			agent.SessionDead = true
			agent.Status = "idle"
			agent.Reason = "session_idle"
		case "retry":
			agent.Status = "retrying"
			agent.IsProcessing = true
			agent.Reason = "session_retrying"
		case "unknown":
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
