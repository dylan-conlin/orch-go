package orch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/beadsutil"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// FindTmuxWindowByBeadsID searches all tmux sessions for a window matching the
// given beads ID. Returns nil window if not found. Package-level variable for test mocking.
var FindTmuxWindowByBeadsID = func(beadsID string) (*tmux.WindowInfo, string, error) {
	return tmux.FindWindowByBeadsIDAllSessions(beadsID)
}

// IsTmuxPaneActive checks if a tmux pane has an active (non-shell) process.
// Package-level variable for test mocking.
var IsTmuxPaneActive = tmux.IsPaneActive

func SetupBeadsTracking(skillName, task, projectName, beadsIssueFlag string, isOrchestrator, isMetaOrchestrator bool, serverURL string, noTrack bool, workspaceName string, createBeadsFn func(string, string, string, string) (string, error), projectDir string) (string, error) {
	skipBeadsForOrchestrator := isOrchestrator || isMetaOrchestrator
	beadsID, err := determineBeadsID(projectName, skillName, task, beadsIssueFlag, noTrack || skipBeadsForOrchestrator, createBeadsFn, projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to determine beads ID: %w", err)
	}
	if skipBeadsForOrchestrator {
		fmt.Println("Skipping beads tracking (orchestrator session)")
	} else if noTrack {
		fmt.Fprintf(os.Stderr, "⚠️  --no-track is deprecated and will be removed in a future release.\n")
		fmt.Fprintf(os.Stderr, "   Created lightweight beads issue %s instead of synthetic ID.\n", beadsID)
		fmt.Fprintf(os.Stderr, "   Lightweight issues auto-close on completion and skip non-essential verification.\n")
	}
	if !noTrack && !skipBeadsForOrchestrator && beadsIssueFlag != "" {
		if stats, err := verify.GetFixAttemptStats(beadsID); err == nil && stats.IsRetryPattern() {
			warning := verify.FormatRetryWarning(stats)
			if warning != "" {
				fmt.Fprintf(os.Stderr, "\n%s\n", warning)
			}
		}
	}
	if !noTrack && !skipBeadsForOrchestrator && beadsIssueFlag != "" {
		if issue, err := verify.GetIssue(beadsID, projectDir); err == nil {
			if issue.Status == "closed" {
				return "", fmt.Errorf("issue %s is already closed", beadsID)
			}
			if issue.Status == "in_progress" {
				if err := CheckActiveAgent(beadsID, serverURL); err != nil {
					return "", err
				}
				if complete, err := verify.IsPhaseComplete(beadsID, projectDir); err == nil && complete {
					return "", fmt.Errorf("issue %s has Phase: Complete but is not closed. Run 'orch complete %s' first", beadsID, beadsID)
				}
				fmt.Fprintf(os.Stderr, "Warning: issue %s is in_progress but no active agent found. Respawning.\n", beadsID)
			}
		}
	}
	if !skipBeadsForOrchestrator && beadsID != "" {
		if err := verify.UpdateIssueStatus(beadsID, "in_progress", projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
		}
		if workspaceName != "" {
			if err := verify.UpdateIssueAssignee(beadsID, workspaceName, projectDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to set assignee on beads issue: %v\n", err)
			}
		}
	}
	return beadsID, nil
}

func determineBeadsID(projectName, skillName, task, spawnIssue string, spawnNoTrack bool, createBeadsFn func(string, string, string, string) (string, error), projectDir string) (string, error) {
	if spawnIssue != "" {
		return resolveShortBeadsID(spawnIssue)
	}
	if spawnNoTrack {
		// Create a real beads issue with tier:lightweight label instead of synthetic ID.
		// This ensures --no-track agents are visible to orch status/complete/clean.
		beadsID, err := createBeadsFn(projectName, skillName, task, projectDir)
		if err != nil {
			return "", fmt.Errorf("failed to create lightweight beads issue: %w", err)
		}
		// Add tier:lightweight label to distinguish from fully-tracked issues
		if err := beads.FallbackAddLabel(beadsID, "tier:lightweight", projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add tier:lightweight label to %s: %v\n", beadsID, err)
		}
		return beadsID, nil
	}
	beadsID, err := createBeadsFn(projectName, skillName, task, projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to create beads issue: %w", err)
	}
	return beadsID, nil
}

// CreateBeadsIssue creates a beads issue in the specified project directory.
// When dir is empty, falls back to CWD (for same-project spawns).
// For cross-project spawns, pass the target project's directory so the issue
// is created in the target project's .beads/, not the source project's.
func CreateBeadsIssue(projectName, skillName, task, dir string) (string, error) {
	title := fmt.Sprintf("[%s] %s: %s", projectName, skillName, truncate(task, 50))
	socketPath, err := beads.FindSocketPath(dir)
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Create(&beads.CreateArgs{Title: title, IssueType: "task", Priority: 2})
			if err == nil {
				return issue.ID, nil
			}
		}
	}
	issue, err := beads.FallbackCreate(title, "", "task", 2, nil, dir)
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}

// DetectCrossRepo checks if a spawn targets a different project than CWD.
// Returns the source project name (CWD basename) if cross-repo, empty string otherwise.
// Pure function for testability — callers pass CWD and resolved project dir.
func DetectCrossRepo(cwd, projectDir string) string {
	if cwd == "" || projectDir == "" {
		return ""
	}
	cwdProject := filepath.Base(cwd)
	targetProject := filepath.Base(projectDir)
	if cwdProject == targetProject {
		return ""
	}
	return cwdProject
}

// ApplyCrossRepoLabels adds cross-repo traceability metadata to a beads issue.
// Adds tier:light label, cross-repo:<source> label, and a back-reference comment.
// The dir parameter specifies the target project directory where the issue lives.
func ApplyCrossRepoLabels(beadsID, sourceProject, dir string) {
	if err := beads.FallbackAddLabel(beadsID, "tier:light", dir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add tier:light label: %v\n", err)
	}
	if err := beads.FallbackAddLabel(beadsID, "cross-repo:"+sourceProject, dir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add cross-repo label: %v\n", err)
	}
	if err := beads.FallbackAddComment(beadsID, fmt.Sprintf("Cross-repo spawn from %s", sourceProject), dir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add cross-repo comment: %v\n", err)
	}
}

// IssueExistsInProject checks if a beads issue exists in a specific project's beads.
// Tries socket client first (fast), falls back to CLI (reliable).
func IssueExistsInProject(beadsID, projectDir string) bool {
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()
			_, err = client.Show(beadsID)
			return err == nil
		}
	}
	_, err = beads.FallbackShow(beadsID, projectDir)
	return err == nil
}

// ResolveCrossRepoBeadsDir determines if BEADS_DIR injection is needed for a
// cross-repo spawn. Returns the .beads/ path to inject, or "" if no override needed.
//
// When an agent spawns in a different project (via --workdir), bd commands default
// to the agent's CWD. If the issue lives in the target project's beads, that works
// naturally. If the issue lives in the source (CWD) project, BEADS_DIR must be set
// so bd can find it.
//
// The issueExists parameter allows injection of a test double.
func ResolveCrossRepoBeadsDir(beadsID, cwd, projectDir string, issueExists func(string, string) bool) string {
	cwdBeadsDir := filepath.Join(cwd, ".beads")
	targetBeadsDir := filepath.Join(projectDir, ".beads")
	if cwdBeadsDir == targetBeadsDir {
		return "" // Same project, no override needed
	}
	// If the issue exists in the target project's beads, no BEADS_DIR needed.
	// The agent's CWD will be the target, so bd's CWD fallback handles it.
	if issueExists(beadsID, projectDir) {
		return ""
	}
	// Issue must be in CWD's project — inject BEADS_DIR so agent can reach it.
	return cwdBeadsDir
}

// CheckActiveAgent checks whether an in_progress issue has a live agent in either
// OpenCode (headless) or tmux (Claude CLI). Returns a non-nil error describing
// the active agent if one is found, nil if no active agent exists.
//
// This is the core dedup check for the manual spawn path. It prevents duplicate
// spawns when the daemon has already picked up an issue via triage:ready.
func CheckActiveAgent(beadsID, serverURL string) error {
	// Check OpenCode sessions (headless backend)
	client := opencode.NewClient(serverURL)
	sessions, _ := client.ListSessions("")
	for _, s := range sessions {
		if strings.Contains(s.Title, beadsID) {
			if client.IsSessionActive(s.ID, 30*time.Minute) {
				return fmt.Errorf("issue %s is already in_progress with active agent (session %s). Use 'orch send %s' to interact or 'orch abandon %s' to restart", beadsID, s.ID, s.ID, beadsID)
			}
			fmt.Fprintf(os.Stderr, "Note: found stale session %s for issue %s (no activity in 30m)\n", shortID(s.ID), beadsID)
		}
	}
	// Check tmux windows (Claude CLI backend)
	// The default backend spawns agents in tmux, not OpenCode.
	// Without this check, manual spawns race with daemon-spawned
	// tmux agents because they're invisible to the OpenCode query above.
	if window, sessionName, err := FindTmuxWindowByBeadsID(beadsID); err == nil && window != nil {
		if IsTmuxPaneActive(window.ID) {
			return fmt.Errorf("issue %s is already in_progress with active agent (tmux %s:%s). Use 'orch abandon %s' to restart", beadsID, sessionName, window.Name, beadsID)
		}
		fmt.Fprintf(os.Stderr, "Note: found tmux window for %s but pane is idle (agent exited)\n", beadsID)
	}
	return nil
}

// resolveShortBeadsID delegates to beadsutil.ResolveShortIDSimple.
func resolveShortBeadsID(id string) (string, error) {
	return beadsutil.ResolveShortIDSimple(id)
}
