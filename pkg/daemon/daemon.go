// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
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

	// Pool is the worker pool for concurrency control.
	// If set, it is used instead of activeCountFunc.
	Pool *WorkerPool

	// listIssuesFunc is used for testing - allows mocking bd list
	listIssuesFunc func() ([]Issue, error)
	// spawnFunc is used for testing - allows mocking orch work
	spawnFunc func(beadsID string) error
	// activeCountFunc is used for testing - allows mocking active agent count
	// Deprecated: Use Pool for concurrency control instead.
	activeCountFunc func() int
	// listCompletedAgentsFunc is used for testing - allows mocking completed agents list
	listCompletedAgentsFunc func(CompletionConfig) ([]CompletedAgent, error)
}

// New creates a new Daemon instance with default configuration.
func New() *Daemon {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new Daemon instance with the given configuration.
func NewWithConfig(config Config) *Daemon {
	d := &Daemon{
		Config:          config,
		listIssuesFunc:  ListReadyIssues,
		spawnFunc:       SpawnWork,
		activeCountFunc: DefaultActiveCount,
	}
	// Initialize worker pool if MaxAgents is set
	if config.MaxAgents > 0 {
		d.Pool = NewWorkerPool(config.MaxAgents)
	}
	return d
}

// NewWithPool creates a new Daemon instance with an explicit worker pool.
// This is useful for sharing a pool across daemon instances or for testing.
func NewWithPool(config Config, pool *WorkerPool) *Daemon {
	return &Daemon{
		Config:          config,
		Pool:            pool,
		listIssuesFunc:  ListReadyIssues,
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

	if d.Config.Verbose {
		fmt.Printf("  DEBUG: Found %d open issues\n", len(issues))
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Priority < issues[j].Priority
	})

	for _, issue := range issues {
		// Skip non-spawnable types
		if !IsSpawnableType(issue.IssueType) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (type %s not spawnable)\n", issue.ID, issue.IssueType)
			}
			continue
		}
		// Skip blocked issues
		if issue.Status == "blocked" {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (blocked)\n", issue.ID)
			}
			continue
		}
		// Skip in_progress issues (already being worked on)
		if issue.Status == "in_progress" {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (already in_progress)\n", issue.ID)
			}
			continue
		}
		// Skip issues without required label (if filter is set)
		if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (missing label %s, has %v)\n", issue.ID, d.Config.Label, issue.Labels)
			}
			continue
		}
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Selected %s (type=%s, labels=%v)\n", issue.ID, issue.IssueType, issue.Labels)
		}
		return &issue, nil
	}

	return nil, nil
}

// AvailableSlots returns the number of agent slots available for spawning.
// Returns a high number if no limit is set.
func (d *Daemon) AvailableSlots() int {
	// Use pool if available
	if d.Pool != nil {
		return d.Pool.Available()
	}
	// Fallback to legacy activeCountFunc
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
	// Use pool if available
	if d.Pool != nil {
		return d.Pool.AtCapacity()
	}
	// Fallback to legacy activeCountFunc
	if d.Config.MaxAgents <= 0 {
		return false // No limit
	}
	return d.activeCountFunc() >= d.Config.MaxAgents
}

// ActiveCount returns the number of currently active agents.
func (d *Daemon) ActiveCount() int {
	if d.Pool != nil {
		return d.Pool.Active()
	}
	return d.activeCountFunc()
}

// PoolStatus returns the current worker pool status for monitoring.
// Returns nil if no pool is configured.
func (d *Daemon) PoolStatus() *PoolStatus {
	if d.Pool == nil {
		return nil
	}
	status := d.Pool.Status()
	return &status
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

// ListReadyIssues retrieves ready issues from beads (open or in_progress, no blockers).
// It uses the beads RPC daemon if available, falling back to the bd CLI if not.
func ListReadyIssues() ([]Issue, error) {
	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()
			beadsIssues, err := client.Ready(nil)
			if err == nil {
				return convertBeadsIssues(beadsIssues), nil
			}
			// Fall through to CLI fallback on Ready() error
		}
		// Fall through to CLI fallback on Connect() error
	}

	// Fallback to CLI if daemon unavailable
	return listReadyIssuesCLI()
}

// listReadyIssuesCLI retrieves ready issues by shelling out to bd CLI.
func listReadyIssuesCLI() ([]Issue, error) {
	cmd := exec.Command("bd", "ready", "--json")
	cmd.Env = os.Environ() // Inherit env (including BEADS_NO_DAEMON)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run bd ready: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}

// convertBeadsIssues converts beads.Issue slice to daemon.Issue slice.
func convertBeadsIssues(beadsIssues []beads.Issue) []Issue {
	issues := make([]Issue, len(beadsIssues))
	for i, bi := range beadsIssues {
		issues[i] = Issue{
			ID:          bi.ID,
			Title:       bi.Title,
			Description: bi.Description,
			Priority:    bi.Priority,
			Status:      bi.Status,
			IssueType:   bi.IssueType,
			Labels:      bi.Labels,
		}
	}
	return issues
}

// ListOpenIssues is an alias for ListReadyIssues for backward compatibility.
// Deprecated: Use ListReadyIssues instead.
func ListOpenIssues() ([]Issue, error) {
	return ListReadyIssues()
}

// SpawnWork spawns work on a beads issue using orch work command.
// This is the default implementation that shells out to orch.
func SpawnWork(beadsID string) error {
	cmd := exec.Command("orch", "work", beadsID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to spawn work: %w: %s", err, string(output))
	}
	return nil
}

// DefaultActiveCount returns the number of active agents by querying OpenCode API.
// Counts all in-memory sessions, which represent active agents.
func DefaultActiveCount() int {
	// Use OpenCode API to count active sessions
	// The default server URL is used; this works because the daemon runs
	// on the same machine as OpenCode server.
	serverURL := os.Getenv("OPENCODE_URL")
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}

	// Make HTTP request to list sessions
	resp, err := http.Get(serverURL + "/session")
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var sessions []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return 0
	}

	// All active OpenCode sessions are considered active agents
	// The daemon polls at 60s intervals, so this is acceptable overhead
	return len(sessions)
}

// Once processes a single issue from the queue and returns.
// If a worker pool is configured, it acquires a slot before spawning.
// Note: The slot is NOT automatically released when the agent completes.
// Use OnceWithSlot() for explicit slot management, or ReleaseSlot() manually.
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

	// If pool is configured, acquire a slot first
	var slot *Slot
	if d.Pool != nil {
		slot = d.Pool.TryAcquire()
		if slot == nil {
			return &OnceResult{
				Processed: false,
				Issue:     issue,
				Skill:     skill,
				Message:   "At capacity - no slots available",
			}, nil
		}
		slot.BeadsID = issue.ID
	}

	// Spawn the work
	if err := d.spawnFunc(issue.ID); err != nil {
		// Release slot on spawn failure
		if d.Pool != nil && slot != nil {
			d.Pool.Release(slot)
		}
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

// OnceWithSlot processes a single issue and returns the acquired slot.
// The caller is responsible for releasing the slot when the agent completes.
// Returns (result, slot, error). Slot will be nil if no pool is configured or if spawn failed.
func (d *Daemon) OnceWithSlot() (*OnceResult, *Slot, error) {
	issue, err := d.NextIssue()
	if err != nil {
		return nil, nil, err
	}

	if issue == nil {
		return &OnceResult{
			Processed: false,
			Message:   "No spawnable issues in queue",
		}, nil, nil
	}

	skill, err := InferSkill(issue.IssueType)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to infer skill: %w", err)
	}

	// If pool is configured, acquire a slot first
	var slot *Slot
	if d.Pool != nil {
		slot = d.Pool.TryAcquire()
		if slot == nil {
			return &OnceResult{
				Processed: false,
				Issue:     issue,
				Skill:     skill,
				Message:   "At capacity - no slots available",
			}, nil, nil
		}
		slot.BeadsID = issue.ID
	}

	// Spawn the work
	if err := d.spawnFunc(issue.ID); err != nil {
		// Release slot on spawn failure
		if d.Pool != nil && slot != nil {
			d.Pool.Release(slot)
		}
		return &OnceResult{
			Processed: false,
			Issue:     issue,
			Skill:     skill,
			Error:     err,
			Message:   fmt.Sprintf("Failed to spawn: %v", err),
		}, nil, nil
	}

	return &OnceResult{
		Processed: true,
		Issue:     issue,
		Skill:     skill,
		Message:   fmt.Sprintf("Spawned work on %s", issue.ID),
	}, slot, nil
}

// ReleaseSlot releases a previously acquired slot.
// Safe to call with nil slot.
func (d *Daemon) ReleaseSlot(slot *Slot) {
	if d.Pool != nil && slot != nil {
		d.Pool.Release(slot)
	}
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

// =============================================================================
// Completion Processing (polls for Phase: Complete agents and closes issues)
// =============================================================================

// CompletionConfig holds configuration for the completion processing loop.
type CompletionConfig struct {
	// PollInterval is the time between polling cycles.
	PollInterval time.Duration

	// DryRun shows what would be processed without actually closing issues.
	DryRun bool

	// Verbose enables detailed output.
	Verbose bool

	// WorkspaceDir is the base directory for agent workspaces.
	// Defaults to .orch/workspace/ relative to project root.
	WorkspaceDir string

	// ProjectDir is the project root directory.
	// Used to locate workspaces and verify constraints.
	ProjectDir string
}

// DefaultCompletionConfig returns sensible defaults for completion configuration.
func DefaultCompletionConfig() CompletionConfig {
	return CompletionConfig{
		PollInterval: 60 * time.Second,
		DryRun:       false,
		Verbose:      false,
	}
}

// CompletedAgent represents an agent that has reported Phase: Complete
// but whose beads issue is still open/in_progress.
type CompletedAgent struct {
	BeadsID       string
	Title         string
	Status        string // open or in_progress
	PhaseSummary  string // Summary from "Phase: Complete - <summary>"
	WorkspacePath string // Path to agent workspace (if found)
}

// CompletionResult contains the result of processing a completion.
type CompletionResult struct {
	BeadsID      string
	Processed    bool
	CloseReason  string
	Error        error
	Verification verify.VerificationResult
}

// CompletionLoopResult contains the results of a completion loop iteration.
type CompletionLoopResult struct {
	Processed []CompletionResult
	Errors    []error
}

// ListCompletedAgents finds all agents that have reported Phase: Complete
// but whose beads issues are still open or in_progress.
func (d *Daemon) ListCompletedAgents(config CompletionConfig) ([]CompletedAgent, error) {
	if d.listCompletedAgentsFunc != nil {
		return d.listCompletedAgentsFunc(config)
	}
	return ListCompletedAgentsDefault(config)
}

// ListCompletedAgentsDefault is the default implementation that queries beads.
func ListCompletedAgentsDefault(config CompletionConfig) ([]CompletedAgent, error) {
	// Get all open/in_progress issues
	openIssues, err := verify.ListOpenIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}

	if len(openIssues) == 0 {
		return nil, nil
	}

	// Collect beads IDs for batch comment fetch
	var beadsIDs []string
	for id := range openIssues {
		beadsIDs = append(beadsIDs, id)
	}

	// Fetch comments for all issues in batch
	commentMap := verify.GetCommentsBatch(beadsIDs)

	var completed []CompletedAgent

	for id, issue := range openIssues {
		comments, ok := commentMap[id]
		if !ok {
			continue
		}

		// Parse phase from comments
		phaseStatus := verify.ParsePhaseFromComments(comments)
		if !phaseStatus.Found {
			continue
		}

		// Check if Phase: Complete
		if !strings.EqualFold(phaseStatus.Phase, "Complete") {
			continue
		}

		// Found a completed agent - look for its workspace
		workspacePath := findWorkspaceForIssue(id, config.WorkspaceDir, config.ProjectDir)

		completed = append(completed, CompletedAgent{
			BeadsID:       id,
			Title:         issue.Title,
			Status:        issue.Status,
			PhaseSummary:  phaseStatus.Summary,
			WorkspacePath: workspacePath,
		})
	}

	return completed, nil
}

// findWorkspaceForIssue tries to find the workspace directory for a beads issue.
// It scans .orch/workspace/ for directories that might match the issue.
func findWorkspaceForIssue(beadsID, workspaceDir, projectDir string) string {
	if workspaceDir == "" && projectDir != "" {
		workspaceDir = filepath.Join(projectDir, ".orch", "workspace")
	}
	if workspaceDir == "" {
		// Try current directory
		cwd, _ := os.Getwd()
		workspaceDir = filepath.Join(cwd, ".orch", "workspace")
	}

	// Check if workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return ""
	}

	// Scan workspace directories for SPAWN_CONTEXT.md that references this beads ID
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wsPath := filepath.Join(workspaceDir, entry.Name())
		spawnContext := filepath.Join(wsPath, "SPAWN_CONTEXT.md")

		// Check if SPAWN_CONTEXT.md exists and references this beads ID
		data, err := os.ReadFile(spawnContext)
		if err != nil {
			continue
		}

		// Look for beads ID in spawn context (e.g., "bd comment <id>" or "--issue <id>")
		if strings.Contains(string(data), beadsID) {
			return wsPath
		}
	}

	return ""
}

// ProcessCompletion verifies and closes a single completed agent.
// It runs the same verification as `orch complete` and closes the beads issue.
func (d *Daemon) ProcessCompletion(agent CompletedAgent, config CompletionConfig) CompletionResult {
	result := CompletionResult{
		BeadsID: agent.BeadsID,
	}

	// Determine tier from workspace if available
	tier := ""
	if agent.WorkspacePath != "" {
		tier = verify.ReadTierFromWorkspace(agent.WorkspacePath)
	}

	// Run full verification
	verificationResult, err := verify.VerifyCompletionFull(
		agent.BeadsID,
		agent.WorkspacePath,
		config.ProjectDir,
		tier,
	)
	if err != nil {
		result.Error = fmt.Errorf("verification failed: %w", err)
		result.Verification = verificationResult
		return result
	}

	result.Verification = verificationResult

	// Check if verification passed
	if !verificationResult.Passed {
		result.Error = fmt.Errorf("verification failed: %s", strings.Join(verificationResult.Errors, "; "))
		return result
	}

	// Build close reason from phase summary
	closeReason := "Phase: Complete"
	if agent.PhaseSummary != "" {
		closeReason = fmt.Sprintf("Phase: Complete - %s", agent.PhaseSummary)
	}

	// Close the issue (unless dry run)
	if !config.DryRun {
		if err := verify.CloseIssue(agent.BeadsID, closeReason); err != nil {
			result.Error = fmt.Errorf("failed to close issue: %w", err)
			return result
		}
	}

	result.Processed = true
	result.CloseReason = closeReason
	return result
}

// CompletionOnce runs a single iteration of the completion loop.
// It finds all Phase: Complete agents and processes their completions.
func (d *Daemon) CompletionOnce(config CompletionConfig) (*CompletionLoopResult, error) {
	result := &CompletionLoopResult{}

	// Find completed agents
	completed, err := d.ListCompletedAgents(config)
	if err != nil {
		return nil, fmt.Errorf("failed to list completed agents: %w", err)
	}

	if len(completed) == 0 {
		return result, nil
	}

	// Process each completed agent
	logger := events.NewDefaultLogger()

	for _, agent := range completed {
		if config.Verbose {
			fmt.Printf("  Processing completion for %s: %s\n", agent.BeadsID, agent.Title)
		}

		compResult := d.ProcessCompletion(agent, config)
		result.Processed = append(result.Processed, compResult)

		if compResult.Error != nil {
			result.Errors = append(result.Errors, compResult.Error)
			if config.Verbose {
				fmt.Printf("    Error: %v\n", compResult.Error)
			}
		} else if compResult.Processed {
			// Log successful auto-completion
			if err := logger.LogAutoCompleted(agent.BeadsID, compResult.CloseReason); err != nil && config.Verbose {
				fmt.Printf("    Warning: failed to log completion event: %v\n", err)
			}
			if config.Verbose {
				fmt.Printf("    Closed: %s\n", compResult.CloseReason)
			}
		}
	}

	return result, nil
}

// CompletionLoop runs the completion processing loop continuously.
// It polls for Phase: Complete agents and closes their issues.
// The loop continues until the context is cancelled.
func (d *Daemon) CompletionLoop(ctx context.Context, config CompletionConfig) error {
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	// Run immediately on first call
	if _, err := d.CompletionOnce(config); err != nil && config.Verbose {
		fmt.Printf("Completion loop error: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := d.CompletionOnce(config); err != nil && config.Verbose {
				fmt.Printf("Completion loop error: %v\n", err)
			}
		}
	}
}

// PreviewCompletions shows what agents would be completed without actually closing them.
func (d *Daemon) PreviewCompletions(config CompletionConfig) ([]CompletedAgent, error) {
	return d.ListCompletedAgents(config)
}
