package beads

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ClientVersion is the version of this RPC client.
// Should match the bd CLI version for compatibility.
var ClientVersion = "0.1.0"

// BdPath is the resolved absolute path to the bd executable.
// Set this at startup via ResolveBdPath() to ensure Fallback* functions
// work correctly when running under launchd with minimal PATH.
// If empty, defaults to "bd" (relies on PATH lookup).
var BdPath string

// bdSearchPaths are common locations where bd might be installed.
// These are checked in order when ResolveBdPath can't find bd in PATH.
var bdSearchPaths = []string{
	"$HOME/bin/bd",
	"$HOME/go/bin/bd",
	"$HOME/.bun/bin/bd",
	"$HOME/.local/bin/bd",
	"/usr/local/bin/bd",
	"/opt/homebrew/bin/bd",
}

// ResolveBdPath attempts to find the bd executable and stores its absolute path
// in BdPath. This should be called at startup by processes that may run under
// launchd or other environments with minimal PATH.
//
// Search order:
// 1. Current PATH (via exec.LookPath)
// 2. Common installation locations (~/bin, ~/go/bin, ~/.bun/bin, etc.)
//
// If bd is found, returns the absolute path and sets BdPath.
// If not found, returns an error but BdPath remains empty (fallback to "bd").
func ResolveBdPath() (string, error) {
	// First, try to find bd in current PATH
	path, err := exec.LookPath("bd")
	if err == nil {
		// Got it from PATH - store absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path // Use as-is if Abs fails
		}
		BdPath = absPath
		return BdPath, nil
	}

	// Not in PATH - check common installation locations
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE") // Windows fallback
	}

	for _, searchPath := range bdSearchPaths {
		// Expand $HOME
		expanded := strings.Replace(searchPath, "$HOME", home, 1)
		if _, err := os.Stat(expanded); err == nil {
			BdPath = expanded
			return BdPath, nil
		}
	}

	return "", fmt.Errorf("bd executable not found in PATH or common locations")
}

// getBdPath returns the bd executable path to use.
// Returns BdPath if set, otherwise "bd" (relies on PATH).
func getBdPath() string {
	if BdPath != "" {
		return BdPath
	}
	return "bd"
}

// setupFallbackEnv configures the command environment for CLI fallback.
// Sets BEADS_NO_DAEMON=1 to skip daemon connection attempts, which avoids
// the 5s timeout when running in launchd/minimal environments where the
// daemon socket may not be accessible.
func setupFallbackEnv(cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")
}

// fallbackCmd creates a bd CLI command with timeout for fallback operations.
// If dir is non-empty, runs in that directory; otherwise uses the process CWD.
// The returned cancel function MUST be called by the caller (typically via defer).
// This prevents unkillable lock pileups when bd hangs on JSONL lock.
func fallbackCmd(dir string, args ...string) (*exec.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	cmd := exec.CommandContext(ctx, getBdPath(), args...)
	setupFallbackEnv(cmd)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd, cancel
}

// FallbackReady retrieves ready issues via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackReady(dir string) ([]Issue, error) {
	// Use --limit 0 to get ALL ready issues (bd ready defaults to limit 10)
	cmd, cancel := fallbackCmd(dir, "ready", "--json", "--limit", "0")
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd ready failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd ready failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd ready output: %w", err)
	}

	return issues, nil
}

// FallbackShow retrieves an issue via bd CLI.
// Note: bd show --json always returns an array, even for a single issue.
// We unmarshal the array and return the first element.
// If dir is non-empty, runs in that directory.
func FallbackShow(id, dir string) (*Issue, error) {
	cmd, cancel := fallbackCmd(dir, "show", id, "--json")
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd show failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	// bd show returns an array even for a single issue
	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd show output: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("bd show returned empty array for id: %s", id)
	}

	return &issues[0], nil
}

// FallbackList retrieves issues via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackList(status, dir string) ([]Issue, error) {
	args := []string{"list", "--json", "--limit", "0"}
	if status != "" {
		args = append(args, "--status", status)
	}

	cmd, cancel := fallbackCmd(dir, args...)
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackListWithLabel retrieves issues with a specific label via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackListWithLabel(label, dir string) ([]Issue, error) {
	if label == "" {
		return []Issue{}, nil
	}

	args := []string{"list", "--json", "--limit", "0", "-l", label}

	cmd, cancel := fallbackCmd(dir, args...)
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list -l %s failed: %w: %s", label, err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list -l %s failed: %w", label, err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackListByIDs retrieves specific issues by ID via bd CLI.
// Uses --id and --all flags to fetch issues regardless of status.
// If dir is non-empty, runs in that directory.
func FallbackListByIDs(ids []string, dir string) ([]Issue, error) {
	if len(ids) == 0 {
		return []Issue{}, nil
	}

	// Use --id with comma-separated IDs and --all to include closed issues
	args := []string{"list", "--json", "--all", "--id", strings.Join(ids, ",")}

	cmd, cancel := fallbackCmd(dir, args...)
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list --id failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list --id failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackListByParent retrieves children of a parent issue via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackListByParent(parentID, dir string) ([]Issue, error) {
	if parentID == "" {
		return []Issue{}, nil
	}

	// Use --parent and --all to include closed children
	// Use --limit 0 to get all children
	args := []string{"list", "--json", "--limit", "0", "--parent", parentID}

	cmd, cancel := fallbackCmd(dir, args...)
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd list --parent failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd list --parent failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// FallbackStats retrieves stats via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackStats(dir string) (*Stats, error) {
	cmd, cancel := fallbackCmd(dir, "stats", "--json")
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd stats failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd stats failed: %w", err)
	}

	var stats Stats
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse bd stats output: %w", err)
	}

	return &stats, nil
}

// FallbackComments retrieves comments via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackComments(id, dir string) ([]Comment, error) {
	cmd, cancel := fallbackCmd(dir, "comments", id, "--json")
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd comments failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd comments failed: %w", err)
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse bd comments output: %w", err)
	}

	return comments, nil
}

// FallbackClose closes an issue via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackClose(id, reason, dir string) error {
	args := []string{"close", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	cmd, cancel := fallbackCmd(dir, args...)
	defer cancel()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd close failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackForceClose closes an issue via bd CLI with --force flag.
// This bypasses bd's Phase: Complete and pinned checks, used when orch complete
// has already verified (or explicitly skipped) those gates.
// If dir is non-empty, runs in that directory.
func FallbackForceClose(id, reason, dir string) error {
	args := []string{"close", id, "--force"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	cmd, cancel := fallbackCmd(dir, args...)
	defer cancel()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd close --force failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackCreate creates an issue via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackCreate(title, description, issueType string, priority int, labels []string, dir string) (*Issue, error) {
	args := []string{"create", title, "--json"}
	if description != "" {
		args = append(args, "--description", description)
	}
	if issueType != "" {
		args = append(args, "--type", issueType)
	}
	if priority > 0 {
		args = append(args, "--priority", fmt.Sprintf("%d", priority))
	}
	for _, label := range labels {
		args = append(args, "--label", label)
	}

	cmd, cancel := fallbackCmd(dir, args...)
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd create failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd create failed: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse bd create output: %w", err)
	}

	return &issue, nil
}

// FallbackAddComment adds a comment via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackAddComment(id, text, dir string) error {
	cmd, cancel := fallbackCmd(dir, "comments", "add", id, text)
	defer cancel()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd comments add failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackUpdate updates an issue via bd CLI.
// Currently supports updating the status field.
// If dir is non-empty, runs in that directory.
func FallbackUpdate(id, status, dir string) error {
	args := []string{"update", id}
	if status != "" {
		args = append(args, "--status", status)
	}
	cmd, cancel := fallbackCmd(dir, args...)
	defer cancel()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd update failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackUpdateAssignee updates the assignee of an issue via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackUpdateAssignee(id, assignee, dir string) error {
	args := []string{"update", id, "--assignee", assignee}
	cmd, cancel := fallbackCmd(dir, args...)
	defer cancel()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd update assignee failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackAddLabel adds a label to an issue via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackAddLabel(id, label, dir string) error {
	cmd, cancel := fallbackCmd(dir, "update", id, "--add-label", label)
	defer cancel()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd add-label failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackRemoveLabel removes a label from an issue via bd CLI.
// If dir is non-empty, runs in that directory.
func FallbackRemoveLabel(id, label, dir string) error {
	cmd, cancel := fallbackCmd(dir, "update", id, "--remove-label", label)
	defer cancel()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd remove-label failed: %w: %s", err, string(output))
	}
	return nil
}

// CheckBlockingDependencies checks if an issue has any blocking dependencies.
// Returns a list of blocking dependencies if any exist, or nil if the issue can be worked on.
// Uses RPC client if available, falls back to CLI otherwise.
func CheckBlockingDependencies(issueID string) ([]BlockingDependency, error) {
	// Try RPC client first
	socketPath, err := FindSocketPath("")
	if err == nil {
		client := NewClient(socketPath, WithAutoReconnect(2))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Show(issueID)
			if err == nil {
				return issue.GetBlockingDependencies(), nil
			}
			// Fall through to CLI on error
		}
	}

	// Fallback to CLI
	issue, err := FallbackShow(issueID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", issueID, err)
	}

	return issue.GetBlockingDependencies(), nil
}

// BlockingDependencyError represents an error when an issue is blocked by dependencies.
type BlockingDependencyError struct {
	IssueID      string
	Blockers     []BlockingDependency
	ForceMessage string
}

func (e *BlockingDependencyError) Error() string {
	if len(e.Blockers) == 0 {
		return fmt.Sprintf("issue %s has unknown blockers", e.IssueID)
	}

	var blockerStrs []string
	for _, b := range e.Blockers {
		blockerStrs = append(blockerStrs, fmt.Sprintf("%s (%s)", b.ID, b.Status))
	}

	return fmt.Sprintf("%s is blocked by: %s\n%s",
		e.IssueID,
		strings.Join(blockerStrs, ", "),
		e.ForceMessage)
}
