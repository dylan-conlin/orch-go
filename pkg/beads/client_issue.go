package beads

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func ensureCreatePersisted(issue *Issue, showFn func(string) (*Issue, error)) error {
	if issue == nil {
		return fmt.Errorf("bd create returned empty issue payload")
	}
	if strings.TrimSpace(issue.ID) == "" {
		return fmt.Errorf("bd create returned empty issue id")
	}

	persisted, err := showFn(issue.ID)
	if err != nil {
		return fmt.Errorf("bd create returned issue %s but it was not persisted (possible JSONL hash mismatch): %w", issue.ID, err)
	}
	if persisted == nil || strings.TrimSpace(persisted.ID) == "" {
		return fmt.Errorf("bd create returned issue %s but read-back was empty (possible JSONL hash mismatch)", issue.ID)
	}

	return nil
}

// Show retrieves a single issue by ID.
// Note: bd show --json returns an array even for a single issue.
// The RPC daemon may return either format (array or single object) depending on version.
// We try array format first (CLI behavior), then fall back to single object (RPC daemon).
// Returns ErrIssueNotFound if the issue doesn't exist.
func (c *Client) Show(id string) (*Issue, error) {
	args := ShowArgs{ID: id}

	resp, err := c.execute(OpShow, args)
	if err != nil {
		// Check if error message indicates issue not found
		if strings.Contains(err.Error(), "no issue found") || strings.Contains(err.Error(), "issue not found") {
			return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
		}
		return nil, err
	}

	// Handle empty or nil data - issue not found
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
	}

	// Try array format first (bd show --json CLI returns array)
	var issues []Issue
	if err := json.Unmarshal(resp.Data, &issues); err == nil {
		if len(issues) == 0 {
			return nil, fmt.Errorf("%w: %s (empty array)", ErrIssueNotFound, id)
		}
		return &issues[0], nil
	}

	// Fall back to single object format (some RPC daemon versions)
	var issue Issue
	if err := json.Unmarshal(resp.Data, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issue (tried array and object): %w", err)
	}

	return &issue, nil
}

// CloseIssue closes an issue with an optional reason.
func (c *Client) CloseIssue(id, reason string) error {
	return c.CloseIssueForce(id, reason, false)
}

// CloseIssueForce closes an issue with an optional reason and force flag.
// When force is true, the daemon bypasses Phase: Complete checks.
func (c *Client) CloseIssueForce(id, reason string, force bool) error {
	args := CloseArgs{
		ID:     id,
		Reason: reason,
		Force:  force,
	}

	_, err := c.execute(OpClose, args)
	return err
}

// Create creates a new issue.
func (c *Client) Create(args *CreateArgs) (*Issue, error) {
	if args == nil {
		return nil, fmt.Errorf("create args required")
	}

	resp, err := c.execute(OpCreate, args)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(resp.Data, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal created issue: %w", err)
	}

	return &issue, nil
}

// Update updates an existing issue.
func (c *Client) Update(args *UpdateArgs) (*Issue, error) {
	if args == nil {
		return nil, fmt.Errorf("update args required")
	}

	resp, err := c.execute(OpUpdate, args)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(resp.Data, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal updated issue: %w", err)
	}

	return &issue, nil
}

// Delete deletes one or more issues.
func (c *Client) Delete(args *DeleteArgs) error {
	if args == nil {
		return fmt.Errorf("delete args required")
	}

	_, err := c.execute(OpDelete, args)
	return err
}

// AddDependency adds a dependency between issues.
func (c *Client) AddDependency(fromID, toID, depType string) error {
	args := DepAddArgs{
		FromID:  fromID,
		ToID:    toID,
		DepType: depType,
	}

	_, err := c.execute(OpDepAdd, args)
	return err
}

// RemoveDependency removes a dependency between issues.
func (c *Client) RemoveDependency(fromID, toID, depType string) error {
	args := DepRemoveArgs{
		FromID:  fromID,
		ToID:    toID,
		DepType: depType,
	}

	_, err := c.execute(OpDepRemove, args)
	return err
}

// FallbackShow retrieves an issue via bd CLI.
// Note: bd show --json always returns an array, even for a single issue.
// We unmarshal the array and return the first element.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
// Returns ErrIssueNotFound if the issue doesn't exist.
func FallbackShow(id string) (*Issue, error) {
	return fallbackShowWithDir(id, DefaultDir)
}

// FallbackShowWithDir retrieves an issue via bd CLI from a specific directory.
// This is used for cross-project agent visibility where the beads issue is in a different
// project than the current working directory.
// If dir is empty, uses DefaultDir if set, otherwise the current working directory.
// Uses getBdPath() to resolve the bd executable location.
func FallbackShowWithDir(id, dir string) (*Issue, error) {
	if dir == "" {
		dir = DefaultDir
	}
	return fallbackShowWithDir(id, dir)
}

func fallbackShowWithDir(id, dir string) (*Issue, error) {
	output, err := runBDOutput(dir, "show", id, "--json")
	if err != nil {
		if IsCLITimeout(err) {
			return nil, fmt.Errorf("bd show timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Check if stderr contains "no issue found" or "no .beads directory" message
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "no issue found") || strings.Contains(stderr, "no .beads directory") {
				return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
			}
			return nil, fmt.Errorf("bd show failed: %w: %s", err, stderr)
		}
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	// Handle empty output - bd show returns exit code 0 but empty output
	// when issue is not found (this is a bd CLI bug but we handle it gracefully)
	if len(output) == 0 || strings.TrimSpace(string(output)) == "" {
		return nil, fmt.Errorf("%w: %s", ErrIssueNotFound, id)
	}

	// bd show returns an array even for a single issue
	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd show output: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("%w: %s (empty array)", ErrIssueNotFound, id)
	}

	return &issues[0], nil
}

// FallbackClose closes an issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackClose(id, reason string) error {
	return FallbackCloseForce(id, reason, false)
}

// FallbackCloseForce closes an issue via bd CLI with optional --force flag.
// When force is true, passes --force to bypass bd's "Phase: Complete" check.
// This is needed when callers (orch complete, daemon) have already verified
// Phase: Complete and the redundant bd gate would reject the close.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackCloseForce(id, reason string, force bool) error {
	args := []string{"close", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	if force {
		args = append(args, "--force")
	}

	output, err := runBDCombinedOutput(DefaultDir, args...)
	if err != nil {
		if IsCLITimeout(err) {
			return fmt.Errorf("bd close timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd close failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackCreate creates an issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackCreate(title, description, issueType string, priority int, labels []string) (*Issue, error) {
	return FallbackCreateWithParentAndCause(title, description, issueType, priority, labels, "", "")
}

// FallbackCreateWithParent creates an issue via bd CLI with an optional parent link.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackCreateWithParent(title, description, issueType string, priority int, labels []string, parent string) (*Issue, error) {
	return FallbackCreateWithParentAndCause(title, description, issueType, priority, labels, parent, "")
}

// FallbackCreateWithParentAndCause creates an issue via bd CLI with optional
// parent and caused-by links.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackCreateWithParentAndCause(title, description, issueType string, priority int, labels []string, parent string, causedBy string) (*Issue, error) {
	args := []string{"create", title, "--json"}
	if description != "" {
		args = append(args, "--description", description)
	}
	if issueType != "" {
		args = append(args, "--type", issueType)
	}
	if parent != "" {
		args = append(args, "--parent", parent)
	}
	if causedBy != "" {
		args = append(args, "--caused-by", causedBy)
	}
	if priority > 0 {
		args = append(args, "--priority", fmt.Sprintf("%d", priority))
	}
	for _, label := range labels {
		args = append(args, "--label", label)
	}

	output, err := runBDOutput(DefaultDir, args...)
	if err != nil {
		if IsCLITimeout(err) {
			return nil, fmt.Errorf("bd create timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd create failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd create failed: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse bd create output: %w", err)
	}

	if err := ensureCreatePersisted(&issue, func(id string) (*Issue, error) {
		return fallbackShowWithDir(id, DefaultDir)
	}); err != nil {
		return nil, err
	}

	return &issue, nil
}

// FallbackUpdate updates an issue via bd CLI.
// Currently supports updating the status field.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackUpdate(id, status string) error {
	args := []string{"update", id}
	if status != "" {
		args = append(args, "--status", status)
	}
	output, err := runBDCombinedOutput(DefaultDir, args...)
	if err != nil {
		if IsCLITimeout(err) {
			return fmt.Errorf("bd update timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd update failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackReopen reopens an issue via bd CLI.
// Uses bd reopen which emits a Reopened event (distinct from simply updating status to open).
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackReopen(id, reason string) error {
	args := []string{"reopen", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	output, err := runBDCombinedOutput(DefaultDir, args...)
	if err != nil {
		if IsCLITimeout(err) {
			return fmt.Errorf("bd reopen timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd reopen failed: %w: %s", err, string(output))
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
	issue, err := FallbackShow(issueID)
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
