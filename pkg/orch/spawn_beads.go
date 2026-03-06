package orch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func SetupBeadsTracking(skillName, task, projectName, beadsIssueFlag string, isOrchestrator, isMetaOrchestrator bool, serverURL string, noTrack bool, workspaceName string, createBeadsFn func(string, string, string) (string, error), projectDir string) (string, error) {
	skipBeadsForOrchestrator := isOrchestrator || isMetaOrchestrator
	beadsID, err := determineBeadsID(projectName, skillName, task, beadsIssueFlag, noTrack || skipBeadsForOrchestrator, createBeadsFn)
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
				client := opencode.NewClient(serverURL)
				sessions, _ := client.ListSessions("")
				for _, s := range sessions {
					if strings.Contains(s.Title, beadsID) {
						if client.IsSessionActive(s.ID, 30*time.Minute) {
							return "", fmt.Errorf("issue %s is already in_progress with active agent (session %s). Use 'orch send %s' to interact or 'orch abandon %s' to restart", beadsID, s.ID, s.ID, beadsID)
						}
						fmt.Fprintf(os.Stderr, "Note: found stale session %s for issue %s (no activity in 30m)\n", shortID(s.ID), beadsID)
					}
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

func determineBeadsID(projectName, skillName, task, spawnIssue string, spawnNoTrack bool, createBeadsFn func(string, string, string) (string, error)) (string, error) {
	if spawnIssue != "" {
		return resolveShortBeadsID(spawnIssue)
	}
	if spawnNoTrack {
		// Create a real beads issue with tier:lightweight label instead of synthetic ID.
		// This ensures --no-track agents are visible to orch status/complete/clean.
		beadsID, err := createBeadsFn(projectName, skillName, task)
		if err != nil {
			return "", fmt.Errorf("failed to create lightweight beads issue: %w", err)
		}
		// Add tier:lightweight label to distinguish from fully-tracked issues
		if err := beads.FallbackAddLabel(beadsID, "tier:lightweight", ""); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add tier:lightweight label to %s: %v\n", beadsID, err)
		}
		return beadsID, nil
	}
	beadsID, err := createBeadsFn(projectName, skillName, task)
	if err != nil {
		return "", fmt.Errorf("failed to create beads issue: %w", err)
	}
	return beadsID, nil
}

func CreateBeadsIssue(projectName, skillName, task string) (string, error) {
	title := fmt.Sprintf("[%s] %s: %s", projectName, skillName, truncate(task, 50))
	socketPath, err := beads.FindSocketPath("")
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
	issue, err := beads.FallbackCreate(title, "", "task", 2, nil, "")
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
func ApplyCrossRepoLabels(beadsID, sourceProject string) {
	if err := beads.FallbackAddLabel(beadsID, "tier:light", ""); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add tier:light label: %v\n", err)
	}
	if err := beads.FallbackAddLabel(beadsID, "cross-repo:"+sourceProject, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add cross-repo label: %v\n", err)
	}
	if err := beads.FallbackAddComment(beadsID, fmt.Sprintf("Cross-repo spawn from %s", sourceProject), ""); err != nil {
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

func resolveShortBeadsID(id string) (string, error) {
	if strings.Contains(id, "-") {
		return id, nil
	}
	projectDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)
	return fmt.Sprintf("%s-%s", projectName, id), nil
}
