package beads

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// CLIClient implements BeadsClient using bd CLI commands.
// This client shells out to the bd command-line tool for all operations.
// Use this when the beads daemon is not available or for simpler deployments.
type CLIClient struct {
	// WorkDir is the working directory for bd commands.
	// If empty, uses the current working directory.
	WorkDir string

	// BdPath is the path to the bd executable.
	// If empty, uses "bd" and relies on PATH lookup.
	BdPath string

	// Env is the environment for bd commands.
	// If nil, inherits from os.Environ().
	Env []string
}

// CLIOption is a functional option for configuring CLIClient.
type CLIOption func(*CLIClient)

// WithWorkDir sets the working directory for bd commands.
func WithWorkDir(dir string) CLIOption {
	return func(c *CLIClient) {
		c.WorkDir = dir
	}
}

// WithBdPath sets the path to the bd executable.
func WithBdPath(path string) CLIOption {
	return func(c *CLIClient) {
		c.BdPath = path
	}
}

// WithEnv sets the environment for bd commands.
func WithEnv(env []string) CLIOption {
	return func(c *CLIClient) {
		c.Env = env
	}
}

// NewCLIClient creates a new CLIClient with the given options.
func NewCLIClient(opts ...CLIOption) *CLIClient {
	c := &CLIClient{
		BdPath: "bd",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// bdCommand creates an exec.Cmd for a bd command with proper configuration.
func (c *CLIClient) bdCommand(args ...string) *exec.Cmd {
	cmd := exec.Command(c.BdPath, args...)
	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
	}
	if c.Env != nil {
		cmd.Env = c.Env
	} else {
		cmd.Env = os.Environ()
	}
	return cmd
}

// Ready retrieves issues that are ready for work.
func (c *CLIClient) Ready(args *ReadyArgs) ([]Issue, error) {
	cmdArgs := []string{"ready", "--json"}
	// Note: The CLI 'bd ready' command has limited filtering compared to RPC.
	// For full filtering support, use the RPC Client instead.

	// Handle limit - default to 0 (no limit) to get ALL ready issues
	// bd ready defaults to limit 10, which truncates results
	limit := 0
	if args != nil && args.Limit > 0 {
		limit = args.Limit
	}
	cmdArgs = append(cmdArgs, "--limit", fmt.Sprintf("%d", limit))

	cmd := c.bdCommand(cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd ready failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd ready output: %w", err)
	}

	return issues, nil
}

// Show retrieves a single issue by ID.
func (c *CLIClient) Show(id string) (*Issue, error) {
	cmd := c.bdCommand("show", id, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	// bd show --json always returns an array
	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd show output: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("bd show returned empty array for id: %s", id)
	}

	return &issues[0], nil
}

// List retrieves issues matching the given criteria.
func (c *CLIClient) List(args *ListArgs) ([]Issue, error) {
	cmdArgs := []string{"list", "--json"}
	if args != nil && args.Status != "" {
		cmdArgs = append(cmdArgs, "--status", args.Status)
	}

	cmd := c.bdCommand(cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd list output: %w", err)
	}

	return issues, nil
}

// Stats retrieves beads statistics.
func (c *CLIClient) Stats() (*Stats, error) {
	cmd := c.bdCommand("stats", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd stats failed: %w", err)
	}

	var stats Stats
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse bd stats output: %w", err)
	}

	return &stats, nil
}

// Comments retrieves comments for an issue.
func (c *CLIClient) Comments(id string) ([]Comment, error) {
	cmd := c.bdCommand("comments", id, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd comments failed: %w", err)
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse bd comments output: %w", err)
	}

	return comments, nil
}

// AddComment adds a comment to an issue.
// Note: The CLI client ignores the author parameter as bd CLI uses
// the current user/agent automatically.
func (c *CLIClient) AddComment(id, _, text string) error {
	cmd := c.bdCommand("comment", id, text)
	return cmd.Run()
}

// CloseIssue closes an issue with an optional reason.
func (c *CLIClient) CloseIssue(id, reason string) error {
	args := []string{"close", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	cmd := c.bdCommand(args...)
	return cmd.Run()
}

// Create creates a new issue.
func (c *CLIClient) Create(args *CreateArgs) (*Issue, error) {
	if args == nil {
		return nil, fmt.Errorf("create args required")
	}

	cmdArgs := []string{"create", args.Title, "--json"}
	if args.Description != "" {
		cmdArgs = append(cmdArgs, "--description", args.Description)
	}
	if args.IssueType != "" {
		cmdArgs = append(cmdArgs, "--type", args.IssueType)
	}
	if args.Priority > 0 {
		cmdArgs = append(cmdArgs, "--priority", fmt.Sprintf("%d", args.Priority))
	}
	for _, label := range args.Labels {
		cmdArgs = append(cmdArgs, "--label", label)
	}
	if args.Parent != "" {
		cmdArgs = append(cmdArgs, "--parent", args.Parent)
	}

	cmd := c.bdCommand(cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd create failed: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse bd create output: %w", err)
	}

	return &issue, nil
}

// Update updates an existing issue.
func (c *CLIClient) Update(args *UpdateArgs) (*Issue, error) {
	if args == nil {
		return nil, fmt.Errorf("update args required")
	}

	cmdArgs := []string{"update", args.ID}
	if args.Status != nil {
		cmdArgs = append(cmdArgs, "--status", *args.Status)
	}
	if args.Title != nil {
		cmdArgs = append(cmdArgs, "--title", *args.Title)
	}
	if args.Description != nil {
		cmdArgs = append(cmdArgs, "--description", *args.Description)
	}
	if args.Priority != nil {
		cmdArgs = append(cmdArgs, "--priority", fmt.Sprintf("%d", *args.Priority))
	}
	for _, label := range args.AddLabels {
		cmdArgs = append(cmdArgs, "--add-label", label)
	}
	for _, label := range args.RemoveLabels {
		cmdArgs = append(cmdArgs, "--remove-label", label)
	}

	cmd := c.bdCommand(cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bd update failed: %w: %s", err, string(output))
	}

	// Note: bd update may not return JSON. Fetch the issue separately.
	return c.Show(args.ID)
}

// AddLabel adds a label to an issue.
func (c *CLIClient) AddLabel(id, label string) error {
	cmd := c.bdCommand("label", id, label)
	return cmd.Run()
}

// RemoveLabel removes a label from an issue.
func (c *CLIClient) RemoveLabel(id, label string) error {
	cmd := c.bdCommand("unlabel", id, label)
	return cmd.Run()
}

// ResolveID resolves a partial issue ID to a full ID.
// Note: bd CLI doesn't have a dedicated resolve command, so we use show
// which accepts partial IDs.
func (c *CLIClient) ResolveID(partialID string) (string, error) {
	issue, err := c.Show(partialID)
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}

// Ensure CLIClient implements BeadsClient.
var _ BeadsClient = (*CLIClient)(nil)
