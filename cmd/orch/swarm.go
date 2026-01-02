// Package main provides the CLI entry point for orch-go.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Swarm command flags
	swarmIssues      []string // Explicit list of issue IDs
	swarmReady       bool     // Spawn from bd ready queue
	swarmConcurrency int      // Maximum concurrent agents
	swarmDetach      bool     // Fire-and-forget mode (don't wait for completion)
	swarmDryRun      bool     // Preview what would be spawned
	swarmDelay       int      // Delay between spawns in seconds
	swarmModel       string   // Model to use for spawned agents
	swarmVerbose     bool     // Verbose output
)

var swarmCmd = &cobra.Command{
	Use:   "swarm",
	Short: "Batch spawn multiple agents with concurrency control",
	Long: `Spawn multiple agents in parallel with WorkerPool-based concurrency control.

The swarm command enables batch processing of multiple issues:
  - Use --issues to spawn from an explicit list of issue IDs
  - Use --ready to spawn from all triage:ready issues (bd ready)
  - Use --concurrency to limit parallel agents (default: 3)
  - Use --detach to return immediately (fire-and-forget mode)

Progress is displayed showing: spawned / active / completed / failed

Examples:
  orch-go swarm --issues proj-123,proj-456,proj-789   # Explicit list
  orch-go swarm --ready                               # From bd ready queue
  orch-go swarm --ready --concurrency 5               # 5 parallel agents
  orch-go swarm --ready --detach                      # Fire-and-forget
  orch-go swarm --ready --dry-run                     # Preview without spawning
  orch-go swarm --issues proj-123 --model flash       # Specific model`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSwarm()
	},
}

func init() {
	swarmCmd.Flags().StringSliceVar(&swarmIssues, "issues", nil, "Comma-separated list of beads issue IDs to spawn")
	swarmCmd.Flags().BoolVar(&swarmReady, "ready", false, "Spawn from bd ready queue (triage:ready issues)")
	swarmCmd.Flags().IntVarP(&swarmConcurrency, "concurrency", "c", 3, "Maximum concurrent agents")
	swarmCmd.Flags().BoolVar(&swarmDetach, "detach", false, "Return immediately without waiting for completion")
	swarmCmd.Flags().BoolVar(&swarmDryRun, "dry-run", false, "Preview what would be spawned without spawning")
	swarmCmd.Flags().IntVar(&swarmDelay, "delay", 5, "Delay between spawns in seconds")
	swarmCmd.Flags().StringVar(&swarmModel, "model", "", "Model to use for spawned agents (opus, sonnet, flash, etc)")
	swarmCmd.Flags().BoolVarP(&swarmVerbose, "verbose", "v", false, "Enable verbose output")

	rootCmd.AddCommand(swarmCmd)
}

// SwarmResult holds the result of spawning a single issue.
type SwarmResult struct {
	IssueID   string
	Title     string
	Skill     string
	SessionID string
	Error     error
	Duration  time.Duration
}

// SwarmProgress tracks overall swarm progress.
type SwarmProgress struct {
	mu        sync.Mutex
	Total     int
	Spawned   int
	Active    int
	Completed int
	Failed    int
	Results   []SwarmResult
}

func (p *SwarmProgress) String() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return fmt.Sprintf("Progress: %d spawned / %d active / %d completed / %d failed (of %d)",
		p.Spawned, p.Active, p.Completed, p.Failed, p.Total)
}

func (p *SwarmProgress) AddSpawned() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Spawned++
	p.Active++
}

func (p *SwarmProgress) AddCompleted(result SwarmResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Active--
	if result.Error != nil {
		p.Failed++
	} else {
		p.Completed++
	}
	p.Results = append(p.Results, result)
}

// swarmAgentTracker tracks a spawned agent for monitoring.
type swarmAgentTracker struct {
	IssueID   string
	Skill     string
	SessionID string
	SpawnTime time.Time
}

func runSwarm() error {
	// Validate flags - must specify either --issues or --ready
	if len(swarmIssues) == 0 && !swarmReady {
		return fmt.Errorf("must specify either --issues or --ready")
	}
	if len(swarmIssues) > 0 && swarmReady {
		return fmt.Errorf("cannot use both --issues and --ready")
	}

	// Collect issues to process
	issues, err := collectSwarmIssues()
	if err != nil {
		return fmt.Errorf("failed to collect issues: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("No issues to spawn")
		return nil
	}

	// Dry-run mode: just print what would be spawned
	if swarmDryRun {
		return printSwarmDryRun(issues)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, stopping swarm...")
		cancel()
	}()

	// Initialize progress tracking
	progress := &SwarmProgress{
		Total:   len(issues),
		Results: make([]SwarmResult, 0, len(issues)),
	}

	// Create worker pool
	pool := daemon.NewWorkerPool(swarmConcurrency)

	// Print initial status
	fmt.Printf("Starting swarm: %d issues with concurrency %d\n", len(issues), swarmConcurrency)
	fmt.Println()

	// Log the swarm start
	logger := events.NewLogger(events.DefaultLogPath())
	startEvent := events.Event{
		Type:      "swarm.start",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"total":       len(issues),
			"concurrency": swarmConcurrency,
			"detach":      swarmDetach,
			"model":       swarmModel,
		},
	}
	if err := logger.Log(startEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// If detach mode, spawn all and return immediately
	if swarmDetach {
		return runSwarmDetached(ctx, pool, issues, progress, logger)
	}

	// Non-detach mode: spawn and monitor until all complete
	return runSwarmAttached(ctx, pool, issues, progress, logger)
}

// collectSwarmIssues gathers the list of issues to spawn.
func collectSwarmIssues() ([]daemon.Issue, error) {
	if swarmReady {
		// Get issues from bd ready queue using daemon's ListOpenIssues
		// and filter by triage:ready label
		return getSwarmReadyIssues()
	}

	// Explicit issue list - fetch details for each
	issues := make([]daemon.Issue, 0, len(swarmIssues))
	for _, id := range swarmIssues {
		// Get issue details from beads
		issue, err := verify.GetIssue(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get issue %s: %w", id, err)
		}
		issues = append(issues, daemon.Issue{
			ID:        id,
			Title:     issue.Title,
			IssueType: issue.IssueType,
			Status:    issue.Status,
			Priority:  0, // Default priority when not available
		})
	}
	return issues, nil
}

// getSwarmReadyIssues fetches issues from bd list with triage:ready label.
// It uses the beads RPC client when available, falling back to the bd CLI.
func getSwarmReadyIssues() ([]daemon.Issue, error) {
	// Try RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()

			issues, err := client.List(&beads.ListArgs{
				Status: "open",
				Labels: []string{"triage:ready"},
			})
			if err == nil {
				result := make([]daemon.Issue, len(issues))
				for i, issue := range issues {
					result[i] = daemon.Issue{
						ID:        issue.ID,
						Title:     issue.Title,
						IssueType: issue.IssueType,
						Priority:  issue.Priority,
						Labels:    issue.Labels,
					}
				}
				return result, nil
			}
			// Fall through to CLI fallback on RPC error
		}
	}

	// Fallback to CLI - need to fetch and filter
	issues, err := beads.FallbackList("open")
	if err != nil {
		return nil, err
	}

	// Filter for triage:ready label
	var result []daemon.Issue
	for _, issue := range issues {
		for _, label := range issue.Labels {
			if label == "triage:ready" {
				result = append(result, daemon.Issue{
					ID:        issue.ID,
					Title:     issue.Title,
					IssueType: issue.IssueType,
					Priority:  issue.Priority,
					Labels:    issue.Labels,
				})
				break
			}
		}
	}

	return result, nil
}

// printSwarmDryRun shows what would be spawned without actually spawning.
func printSwarmDryRun(issues []daemon.Issue) error {
	fmt.Printf("[DRY-RUN] Would spawn %d agents:\n\n", len(issues))

	for i, issue := range issues {
		skill, err := daemon.InferSkill(issue.IssueType)
		if err != nil {
			skill = "unknown"
		}
		fmt.Printf("%d. %s\n", i+1, issue.ID)
		fmt.Printf("   Title: %s\n", issue.Title)
		fmt.Printf("   Type:  %s\n", issue.IssueType)
		fmt.Printf("   Skill: %s\n", skill)
		fmt.Println()
	}

	fmt.Printf("Concurrency: %d\n", swarmConcurrency)
	fmt.Printf("Model: %s\n", swarmModel)
	fmt.Println("\nNo agents were spawned (dry-run mode).")
	return nil
}

// runSwarmDetached spawns all issues and returns immediately (fire-and-forget).
func runSwarmDetached(ctx context.Context, pool *daemon.WorkerPool, issues []daemon.Issue, progress *SwarmProgress, logger *events.Logger) error {
	var wg sync.WaitGroup

	for _, issue := range issues {
		// Check context before each spawn
		select {
		case <-ctx.Done():
			fmt.Println("\nSwarm interrupted before all spawns")
			return nil
		default:
		}

		// Acquire slot (blocking)
		slot, err := pool.Acquire(ctx)
		if err != nil {
			// Context cancelled
			break
		}
		slot.BeadsID = issue.ID

		wg.Add(1)
		go func(issue daemon.Issue, slot *daemon.Slot) {
			defer wg.Done()
			defer pool.Release(slot)

			// Spawn the agent
			result := spawnSwarmAgent(issue)
			progress.AddSpawned()

			if swarmVerbose {
				if result.Error != nil {
					fmt.Printf("  [FAIL] %s: %v\n", issue.ID, result.Error)
				} else {
					fmt.Printf("  [OK] %s (%s)\n", issue.ID, result.Skill)
				}
			}

			// Log spawn
			logSwarmSpawn(logger, issue, result)
		}(issue, slot)

		// Delay between spawns to avoid rate limits
		if swarmDelay > 0 {
			select {
			case <-ctx.Done():
				break
			case <-time.After(time.Duration(swarmDelay) * time.Second):
			}
		}
	}

	// In detach mode, we don't wait for agents to complete
	fmt.Printf("Spawned %d agents (detach mode - not waiting for completion)\n", progress.Spawned)
	fmt.Println("Use 'orch status' to monitor progress")

	// Log swarm detach
	detachEvent := events.Event{
		Type:      "swarm.detach",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"spawned": progress.Spawned,
			"total":   progress.Total,
		},
	}
	if err := logger.Log(detachEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	return nil
}

// runSwarmAttached spawns issues and waits for all to complete.
func runSwarmAttached(ctx context.Context, pool *daemon.WorkerPool, issues []daemon.Issue, progress *SwarmProgress, logger *events.Logger) error {
	var wg sync.WaitGroup

	// Channel to track spawned agents for monitoring
	spawnedAgents := make(chan swarmAgentTracker, len(issues))

	for _, issue := range issues {
		// Check context before each spawn
		select {
		case <-ctx.Done():
			fmt.Println("\nSwarm interrupted")
			printSwarmSummary(progress)
			return nil
		default:
		}

		// Acquire slot (blocking)
		slot, err := pool.Acquire(ctx)
		if err != nil {
			// Context cancelled
			break
		}
		slot.BeadsID = issue.ID

		wg.Add(1)
		go func(issue daemon.Issue, slot *daemon.Slot) {
			defer wg.Done()
			defer pool.Release(slot)

			// Spawn the agent
			startTime := time.Now()
			result := spawnSwarmAgent(issue)
			progress.AddSpawned()

			// Print progress
			fmt.Printf("[%s] Spawned: %s (%s)\n",
				time.Now().Format("15:04:05"),
				issue.ID,
				result.Skill,
			)
			fmt.Printf("  %s\n", progress.String())

			// Log spawn
			logSwarmSpawn(logger, issue, result)

			// Track for monitoring
			spawnedAgents <- swarmAgentTracker{
				IssueID:   issue.ID,
				Skill:     result.Skill,
				SessionID: result.SessionID,
				SpawnTime: startTime,
			}
		}(issue, slot)

		// Delay between spawns to avoid rate limits
		if swarmDelay > 0 {
			select {
			case <-ctx.Done():
				break
			case <-time.After(time.Duration(swarmDelay) * time.Second):
			}
		}
	}

	// Wait for all spawns to initiate
	wg.Wait()
	close(spawnedAgents)

	// Collect spawned agents for monitoring
	var agents []swarmAgentTracker
	for agent := range spawnedAgents {
		agents = append(agents, agent)
	}

	fmt.Println()
	fmt.Println("All agents spawned. Monitoring for completion...")
	fmt.Println()

	// Monitor agents until all complete (or context cancelled)
	if err := monitorSwarmAgents(ctx, agents, progress, logger); err != nil {
		return err
	}

	// Print final summary
	printSwarmSummary(progress)

	// Log swarm complete
	completeEvent := events.Event{
		Type:      "swarm.complete",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"total":     progress.Total,
			"spawned":   progress.Spawned,
			"completed": progress.Completed,
			"failed":    progress.Failed,
		},
	}
	if err := logger.Log(completeEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	return nil
}

// spawnSwarmAgent spawns a single agent for an issue.
func spawnSwarmAgent(issue daemon.Issue) SwarmResult {
	result := SwarmResult{
		IssueID: issue.ID,
		Title:   issue.Title,
	}

	// Infer skill from issue type
	skill, err := daemon.InferSkill(issue.IssueType)
	if err != nil {
		result.Error = fmt.Errorf("failed to infer skill: %w", err)
		return result
	}
	result.Skill = skill

	// Build spawn command
	args := []string{"spawn", skill, issue.Title, "--issue", issue.ID}
	if swarmModel != "" {
		args = append(args, "--model", swarmModel)
	}

	// Execute spawn
	startTime := time.Now()
	cmd := exec.Command("orch-go", args...)
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = fmt.Errorf("spawn failed: %w: %s", err, string(output))
		return result
	}

	// Try to extract session ID from output (for monitoring)
	result.SessionID = extractSessionIDFromOutput(string(output))

	return result
}

// extractSessionIDFromOutput parses spawn output for session ID.
func extractSessionIDFromOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Session ID:") {
			parts := strings.Split(line, "Session ID:")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// monitorSwarmAgents monitors spawned agents until all complete.
func monitorSwarmAgents(ctx context.Context, agents []swarmAgentTracker, progress *SwarmProgress, logger *events.Logger) error {
	if len(agents) == 0 {
		return nil
	}

	// Check interval for agent completion
	checkInterval := 30 * time.Second
	timeout := 4 * time.Hour // Max wait time

	startTime := time.Now()
	pendingAgents := make(map[string]swarmAgentTracker)
	for _, agent := range agents {
		pendingAgents[agent.IssueID] = agent
	}

	for len(pendingAgents) > 0 {
		// Check timeout
		if time.Since(startTime) > timeout {
			fmt.Printf("\nSwarm monitoring timeout after %s\n", timeout)
			fmt.Printf("Remaining agents: %d\n", len(pendingAgents))
			return nil
		}

		// Check context
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Check each pending agent
		for issueID, agent := range pendingAgents {
			status, err := verify.GetPhaseStatus(issueID)
			if err != nil {
				// Can't get status - might be transient error, continue
				if swarmVerbose {
					fmt.Printf("  [?] %s: could not get status: %v\n", issueID, err)
				}
				continue
			}

			if status.Found && status.Phase == "Complete" {
				// Agent completed
				result := SwarmResult{
					IssueID:  issueID,
					Skill:    agent.Skill,
					Duration: time.Since(agent.SpawnTime),
				}
				progress.AddCompleted(result)
				delete(pendingAgents, issueID)

				fmt.Printf("[%s] Completed: %s (duration: %s)\n",
					time.Now().Format("15:04:05"),
					issueID,
					formatDuration(result.Duration),
				)
				fmt.Printf("  %s\n", progress.String())

				// Log completion
				completionEvent := events.Event{
					Type:      "swarm.agent.complete",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"issue_id": issueID,
						"skill":    agent.Skill,
						"duration": result.Duration.String(),
					},
				}
				if err := logger.Log(completionEvent); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
				}
			}
		}

		// Wait before next check
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(checkInterval):
		}
	}

	return nil
}

// logSwarmSpawn logs a spawn event.
func logSwarmSpawn(logger *events.Logger, issue daemon.Issue, result SwarmResult) {
	eventData := map[string]interface{}{
		"issue_id": issue.ID,
		"title":    issue.Title,
		"skill":    result.Skill,
	}
	if result.Error != nil {
		eventData["error"] = result.Error.Error()
	}
	if result.SessionID != "" {
		eventData["session_id"] = result.SessionID
	}

	event := events.Event{
		Type:      "swarm.spawn",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}
}

// printSwarmSummary prints the final swarm summary.
func printSwarmSummary(progress *SwarmProgress) {
	fmt.Println()
	fmt.Println("=== Swarm Summary ===")
	fmt.Printf("Total:     %d\n", progress.Total)
	fmt.Printf("Spawned:   %d\n", progress.Spawned)
	fmt.Printf("Completed: %d\n", progress.Completed)
	fmt.Printf("Failed:    %d\n", progress.Failed)

	// Print any failures
	if progress.Failed > 0 {
		fmt.Println()
		fmt.Println("Failed agents:")
		for _, result := range progress.Results {
			if result.Error != nil {
				fmt.Printf("  - %s: %v\n", result.IssueID, result.Error)
			}
		}
	}
}
