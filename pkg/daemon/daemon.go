// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Config holds configuration for the daemon.
type Config struct {
	// PollInterval is the time between polling cycles (0 = run once).
	PollInterval time.Duration

	// MaxAgents is the maximum number of concurrent agents (0 = no limit).
	MaxAgents int

	// Label filters issues to only those with this label (empty = no filter).
	Label string

	// SpawnDelay is the delay between spawns to avoid rate limits.
	SpawnDelay time.Duration

	// DryRun shows what would be processed without spawning.
	DryRun bool

	// Verbose enables detailed output.
	Verbose bool
}

// DefaultConfig returns sensible defaults for daemon configuration.
func DefaultConfig() Config {
	return Config{
		PollInterval: time.Minute,
		MaxAgents:    3,
		Label:        "triage:ready",
		SpawnDelay:   10 * time.Second,
		DryRun:       false,
		Verbose:      false,
	}
}

// Issue represents a beads issue for processing.
type Issue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Status      string   `json:"status"`
	IssueType   string   `json:"issue_type"`
	Labels      []string `json:"labels"`
}

// HasLabel checks if an issue has a specific label.
func (i *Issue) HasLabel(label string) bool {
	for _, l := range i.Labels {
		if strings.EqualFold(l, label) {
			return true
		}
	}
	return false
}

// PreviewResult contains the result of a preview operation.
type PreviewResult struct {
	Issue   *Issue
	Skill   string
	Message string
}

// OnceResult contains the result of processing one issue.
type OnceResult struct {
	Processed bool
	Issue     *Issue
	Skill     string
	Message   string
	Error     error
}

// Daemon manages autonomous issue processing.
type Daemon struct {
	// Config holds the daemon configuration.
	Config Config

	// listIssuesFunc is used for testing - allows mocking bd list
	listIssuesFunc func() ([]Issue, error)
	// spawnFunc is used for testing - allows mocking orch work
	spawnFunc func(beadsID string) error
	// activeCountFunc is used for testing - allows mocking active agent count
	activeCountFunc func() int
}

// New creates a new Daemon instance with default configuration.
func New() *Daemon {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new Daemon instance with the given configuration.
func NewWithConfig(config Config) *Daemon {
	return &Daemon{
		Config:          config,
		listIssuesFunc:  ListOpenIssues,
		spawnFunc:       SpawnWork,
		activeCountFunc: DefaultActiveCount,
	}
}

// NextIssue returns the next spawnable issue from the queue.
// Returns nil if no spawnable issues are available.
// Issues are sorted by priority (0 = highest priority).
// If a label filter is configured, only issues with that label are considered.
func (d *Daemon) NextIssue() (*Issue, error) {
	issues, err := d.listIssuesFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Priority < issues[j].Priority
	})

	for _, issue := range issues {
		// Skip non-spawnable types
		if !IsSpawnableType(issue.IssueType) {
			continue
		}
		// Skip blocked issues
		if issue.Status == "blocked" {
			continue
		}
		// Skip issues without required label (if filter is set)
		if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
			continue
		}
		return &issue, nil
	}

	return nil, nil
}

// AvailableSlots returns the number of agent slots available for spawning.
// Returns a high number if no limit is set.
func (d *Daemon) AvailableSlots() int {
	if d.Config.MaxAgents <= 0 {
		return 100 // No limit
	}
	active := d.activeCountFunc()
	available := d.Config.MaxAgents - active
	if available < 0 {
		return 0
	}
	return available
}

// AtCapacity returns true if the daemon cannot spawn more agents.
func (d *Daemon) AtCapacity() bool {
	if d.Config.MaxAgents <= 0 {
		return false // No limit
	}
	return d.activeCountFunc() >= d.Config.MaxAgents
}

// Preview shows what would be processed next without actually processing.
func (d *Daemon) Preview() (*PreviewResult, error) {
	issue, err := d.NextIssue()
	if err != nil {
		return nil, err
	}

	if issue == nil {
		return &PreviewResult{
			Message: "No spawnable issues in queue",
		}, nil
	}

	skill, err := InferSkill(issue.IssueType)
	if err != nil {
		return nil, fmt.Errorf("failed to infer skill: %w", err)
	}

	return &PreviewResult{
		Issue: issue,
		Skill: skill,
	}, nil
}

// IsSpawnableType returns true if the issue type can be spawned.
func IsSpawnableType(issueType string) bool {
	switch issueType {
	case "bug", "feature", "task", "investigation":
		return true
	default:
		return false
	}
}

// InferSkill maps issue types to skills.
func InferSkill(issueType string) (string, error) {
	switch issueType {
	case "bug":
		return "systematic-debugging", nil
	case "feature":
		return "feature-impl", nil
	case "task":
		return "feature-impl", nil
	case "investigation":
		return "investigation", nil
	default:
		return "", fmt.Errorf("cannot infer skill for issue type: %s", issueType)
	}
}

// FormatPreview formats an issue for preview display.
func FormatPreview(issue *Issue) string {
	return fmt.Sprintf(`Issue:    %s
Title:    %s
Type:     %s
Priority: P%d
Status:   %s
Description: %s`,
		issue.ID,
		issue.Title,
		issue.IssueType,
		issue.Priority,
		issue.Status,
		truncate(issue.Description, 100),
	)
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ListOpenIssues retrieves open issues from beads.
// This is the default implementation that shells out to bd.
func ListOpenIssues() ([]Issue, error) {
	cmd := exec.Command("bd", "list", "--status", "open", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run bd list: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}

// SpawnWork spawns work on a beads issue using orch-go work command.
// This is the default implementation that shells out to orch-go.
func SpawnWork(beadsID string) error {
	cmd := exec.Command("orch-go", "work", beadsID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to spawn work: %w: %s", err, string(output))
	}
	return nil
}

// DefaultActiveCount returns the number of active agents from the registry.
// This reads the registry file directly to avoid circular imports.
func DefaultActiveCount() int {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0
	}
	registryPath := fmt.Sprintf("%s/.orch/agent-registry.json", home)

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return 0 // Registry doesn't exist or can't be read
	}

	var registry struct {
		Agents []struct {
			Status string `json:"status"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return 0
	}

	count := 0
	for _, a := range registry.Agents {
		if a.Status == "active" {
			count++
		}
	}
	return count
}

// Once processes a single issue from the queue and returns.
func (d *Daemon) Once() (*OnceResult, error) {
	issue, err := d.NextIssue()
	if err != nil {
		return nil, err
	}

	if issue == nil {
		return &OnceResult{
			Processed: false,
			Message:   "No spawnable issues in queue",
		}, nil
	}

	skill, err := InferSkill(issue.IssueType)
	if err != nil {
		return nil, fmt.Errorf("failed to infer skill: %w", err)
	}

	// Spawn the work
	if err := d.spawnFunc(issue.ID); err != nil {
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Error:     err,
			Message:   fmt.Sprintf("Failed to spawn: %v", err),
		}, nil
	}

	return &OnceResult{
		Processed: true,
		Issue:     issue,
		Skill:     skill,
		Message:   fmt.Sprintf("Spawned work on %s", issue.ID),
	}, nil
}

// Run processes issues in a loop until the queue is empty or maxIterations is reached.
// Returns a slice of results for each processed issue.
func (d *Daemon) Run(maxIterations int) ([]*OnceResult, error) {
	var results []*OnceResult

	for i := 0; i < maxIterations; i++ {
		result, err := d.Once()
		if err != nil {
			return results, err
		}

		// Queue is empty
		if !result.Processed {
			break
		}

		results = append(results, result)
	}

	return results, nil
}
