package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// daemonRuntime holds all runtime state for the daemon main loop.
// Grouping these fields together avoids passing 10+ parameters between
// the lifecycle helper functions that make up runDaemonLoop.
type daemonRuntime struct {
	d          *daemon.Daemon
	logger     *events.Logger
	config     daemon.Config
	ctx        context.Context
	cancel     context.CancelFunc
	projectDir string

	// Counters
	processed int
	completed int
	cycles    int

	// Timestamps
	lastSpawn      time.Time
	lastCompletion time.Time
}

// buildDaemonConfig constructs a daemon.Config from CLI flags and user config.
// This consolidates the 30+ flag-to-config assignments that were inline in runDaemonLoop.
func buildDaemonConfig() (daemon.Config, error) {
	// Load user config to get backend setting
	cfg, err := userconfig.Load()
	if err != nil {
		// Non-fatal: use default (opencode) if config can't be loaded
		cfg = userconfig.DefaultConfig()
	}

	// Build configuration from defaults, then override with flags.
	// This ensures recovery settings (RecoveryEnabled, ServerRecoveryEnabled, etc.)
	// get their default values even when not explicitly set via flags.
	config := daemon.DefaultConfig()
	config.PollInterval = time.Duration(daemonPollInterval) * time.Second
	config.MaxAgents = daemonMaxAgents
	config.Label = daemonLabel
	config.SpawnDelay = time.Duration(daemonDelay) * time.Second
	config.DryRun = daemonDryRun
	config.Verbose = daemonVerbose
	config.CrossProject = daemonCrossProject
	config.Backend = cfg.Backend // Use backend from user config
	config.ReflectEnabled = daemonReflectInterval > 0
	config.ReflectInterval = time.Duration(daemonReflectInterval) * time.Minute
	config.ReflectCreateIssues = daemonReflectIssues
	config.CleanupEnabled = daemonCleanupEnabled && daemonCleanupInterval > 0
	config.CleanupInterval = time.Duration(daemonCleanupInterval) * time.Minute
	config.CleanupSessions = daemonCleanupSessions
	config.CleanupSessionsAgeDays = daemonCleanupSessionsAge
	config.CleanupWorkspaces = daemonCleanupWorkspaces
	config.CleanupWorkspacesAgeDays = daemonCleanupWorkspacesAge
	config.CleanupInvestigations = daemonCleanupInvestigations
	config.CleanupPreserveOrchestrator = daemonCleanupPreserveOrch
	config.CleanupServerURL = serverURL // Use global serverURL from root command
	config.SpawnFactualQuestions = daemonSpawnFactualQuestions
	config.DeadSessionDetectionEnabled = daemonDeadSessionDetectionEnabled && daemonDeadSessionDetectionInterval > 0
	config.DeadSessionDetectionInterval = time.Duration(daemonDeadSessionDetectionInterval) * time.Minute
	if daemonMaxDeadSessionRetries > 0 {
		config.MaxDeadSessionRetries = daemonMaxDeadSessionRetries
	}
	config.OrphanReapEnabled = daemonOrphanReapEnabled && daemonOrphanReapInterval > 0
	config.OrphanReapInterval = time.Duration(daemonOrphanReapInterval) * time.Minute
	config.SortMode = daemonSortMode

	return config, nil
}

// initDaemonRuntime creates and initializes the full daemon runtime:
// daemon instance, processed-issue cache, event logger, and signal handler.
func initDaemonRuntime(config daemon.Config) (*daemonRuntime, error) {
	projectDir, err := currentProjectDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	d := daemon.NewWithConfig(config)

	// Initialize ProcessedIssueCache for unified dedup (survives daemon restart)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		cachePath := filepath.Join(homeDir, ".orch", "processed-issues.jsonl")
		cache, cacheErr := daemon.NewProcessedIssueCache(cachePath)
		if cacheErr != nil {
			fmt.Printf("Warning: failed to initialize ProcessedIssueCache: %v\n", cacheErr)
			fmt.Println("  Falling back to in-memory dedup only")
		} else {
			d.ProcessedCache = cache
		}
	}

	logger := events.NewLogger(events.DefaultLogPath())
	d.SetEventLogger(logger)

	ctx, cancel := context.WithCancel(context.Background())

	return &daemonRuntime{
		d:          d,
		logger:     logger,
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
		projectDir: projectDir,
	}, nil
}

// printDaemonBanner prints the startup configuration summary.
func printDaemonBanner(config daemon.Config) {
	fmt.Println("Starting daemon...")
	fmt.Printf("  Poll interval:    %s\n", formatDaemonDuration(config.PollInterval))
	fmt.Printf("  Concurrency:      %d (worker pool)\n", config.MaxAgents)
	countMode := "OpenCode sessions"
	if config.Backend == "docker" {
		countMode = "Docker containers"
	}
	fmt.Printf("  Backend:          %s (counting %s)\n", config.Backend, countMode)
	fmt.Printf("  Required label:   %s\n", config.Label)
	sortMode := config.SortMode
	if sortMode == "" {
		sortMode = "priority"
	}
	fmt.Printf("  Sort mode:        %s\n", sortMode)
	fmt.Printf("  Spawn delay:      %s\n", formatDaemonDuration(config.SpawnDelay))
	if config.CrossProject {
		fmt.Println("  Cross-project:    enabled (polling all kb-registered projects)")
	}
	if config.ReflectEnabled {
		fmt.Printf("  Reflect interval:  %s\n", formatDaemonDuration(config.ReflectInterval))
		fmt.Printf("  Reflect issues:    %v\n", config.ReflectCreateIssues)
	} else {
		fmt.Println("  Reflect interval:  disabled")
	}
	if config.CleanupEnabled {
		fmt.Printf("  Cleanup interval:  %s\n", formatDaemonDuration(config.CleanupInterval))
		if config.CleanupSessions {
			fmt.Printf("  Cleanup sessions:  enabled (age: %d days)\n", config.CleanupSessionsAgeDays)
		}
		if config.CleanupWorkspaces {
			fmt.Printf("  Cleanup workspaces: enabled (age: %d days)\n", config.CleanupWorkspacesAgeDays)
		}
		if config.CleanupInvestigations {
			fmt.Printf("  Cleanup investigations: enabled\n")
		}
		fmt.Printf("  Cleanup preserve:  %v (orchestrator sessions/workspaces)\n", config.CleanupPreserveOrchestrator)
	} else {
		fmt.Println("  Cleanup interval:  disabled")
	}
	if config.RecoveryEnabled {
		fmt.Printf("  Recovery interval: %s\n", formatDaemonDuration(config.RecoveryInterval))
		fmt.Printf("  Recovery idle:     %s\n", formatDaemonDuration(config.RecoveryIdleThreshold))
		fmt.Printf("  Recovery rate:     %s (per agent)\n", formatDaemonDuration(config.RecoveryRateLimit))
	} else {
		fmt.Println("  Recovery interval: disabled")
	}
	if config.ServerRecoveryEnabled {
		fmt.Printf("  Server recovery:   enabled\n")
		fmt.Printf("  Server stab delay: %s\n", formatDaemonDuration(config.ServerRecoveryStabilizationDelay))
		fmt.Printf("  Server resume gap: %s\n", formatDaemonDuration(config.ServerRecoveryResumeDelay))
	} else {
		fmt.Println("  Server recovery:   disabled")
	}
	if config.SpawnFactualQuestions {
		fmt.Println("  Factual questions: enabled (spawning investigations for subtype:factual)")
	} else {
		fmt.Println("  Factual questions: disabled")
	}
	if config.OrphanReapEnabled {
		fmt.Printf("  Orphan reaper:     %s\n", formatDaemonDuration(config.OrphanReapInterval))
	} else {
		fmt.Println("  Orphan reaper:     disabled")
	}
	if config.DeadSessionDetectionEnabled {
		fmt.Printf("  Dead session det:  %s\n", formatDaemonDuration(config.DeadSessionDetectionInterval))
	} else {
		fmt.Println("  Dead session det:  disabled")
	}
	fmt.Println()
}

// runSubsystems executes all periodic subsystems (reflection, cleanup, recovery,
// server recovery, dead session detection). Returns early if any subsystem needs
// special handling.
func (rt *daemonRuntime) runSubsystems(timestamp string) {
	// Run periodic reflection if due
	if result := rt.d.RunPeriodicReflection(); result != nil {
		if result.Error != nil {
			fmt.Fprintf(os.Stderr, "[%s] Reflection error: %v\n", timestamp, result.Error)
		} else if result.Suggestions != nil && result.Suggestions.HasSuggestions() {
			fmt.Printf("[%s] Reflection: %s\n", timestamp, result.Suggestions.Summary())
		} else if daemonVerbose {
			fmt.Printf("[%s] Reflection: no suggestions found\n", timestamp)
		}
	}

	// Run periodic cleanup if due
	if result := rt.d.RunPeriodicCleanup(); result != nil {
		data := map[string]interface{}{
			"sessions_deleted":        result.SessionsDeleted,
			"workspaces_archived":     result.WorkspacesArchived,
			"investigations_archived": result.InvestigationsArchived,
			"message":                 result.Message,
		}
		if result.Error != nil {
			data["error"] = result.Error.Error()
		}
		logSubsystemResult(rt.logger, timestamp, daemonVerbose, subsystemResult{
			Name:        "Cleanup",
			EventType:   "daemon.cleanup",
			Error:       result.Error,
			Message:     result.Message,
			HasActivity: result.SessionsDeleted > 0 || result.WorkspacesArchived > 0 || result.InvestigationsArchived > 0,
			Data:        data,
		})
	}

	// Run periodic stuck agent recovery if due
	if result := rt.d.RunPeriodicRecovery(); result != nil {
		data := map[string]interface{}{
			"resumed": result.ResumedCount,
			"skipped": result.SkippedCount,
			"message": result.Message,
		}
		if result.Error != nil {
			data["resumed"] = 0
			data["error"] = result.Error.Error()
		}
		logSubsystemResult(rt.logger, timestamp, daemonVerbose, subsystemResult{
			Name:        "Recovery",
			EventType:   "daemon.recovery",
			Error:       result.Error,
			Message:     result.Message,
			HasActivity: result.ResumedCount > 0,
			Data:        data,
		})
	}

	// Run server restart recovery if due (runs once after daemon startup)
	serverRecoveryResult := rt.d.RunServerRecovery()
	if daemonVerbose {
		if serverRecoveryResult == nil {
			fmt.Printf("[%s] [DEBUG] Server recovery: ShouldRunServerRecovery returned false (recovery not due)\n", timestamp)
		} else {
			fmt.Printf("[%s] [DEBUG] Server recovery: result=%+v\n", timestamp, serverRecoveryResult)
		}
	}
	if serverRecoveryResult != nil {
		data := map[string]interface{}{
			"orphaned": serverRecoveryResult.OrphanedCount,
			"resumed":  serverRecoveryResult.ResumedCount,
			"skipped":  serverRecoveryResult.SkippedCount,
			"message":  serverRecoveryResult.Message,
		}
		if serverRecoveryResult.Error != nil {
			data["orphaned"] = 0
			data["resumed"] = 0
			data["error"] = serverRecoveryResult.Error.Error()
		}
		logSubsystemResult(rt.logger, timestamp, daemonVerbose, subsystemResult{
			Name:        "Server recovery",
			EventType:   "daemon.server_recovery",
			Error:       serverRecoveryResult.Error,
			Message:     serverRecoveryResult.Message,
			HasActivity: serverRecoveryResult.OrphanedCount > 0,
			Data:        data,
		})
	}

	// Run periodic orphan process reaping if due
	if result := rt.d.ReapOrphanProcesses(); result != nil {
		data := map[string]interface{}{
			"found":   result.Found,
			"killed":  result.Killed,
			"message": result.Message,
		}
		if result.Error != nil {
			data["found"] = 0
			data["killed"] = 0
			data["error"] = result.Error.Error()
		}
		logSubsystemResult(rt.logger, timestamp, daemonVerbose, subsystemResult{
			Name:        "Orphan reaper",
			EventType:   "daemon.orphan_reap",
			Error:       result.Error,
			Message:     result.Message,
			HasActivity: result.Killed > 0,
			Data:        data,
		})
	}

	// Run periodic dead session detection if due
	if result := rt.d.RunPeriodicDeadSessionDetection(); result != nil {
		data := map[string]interface{}{
			"detected":  result.DetectedCount,
			"marked":    result.MarkedCount,
			"escalated": result.EscalatedCount,
			"skipped":   result.SkippedCount,
			"message":   result.Message,
		}
		if result.Error != nil {
			data["detected"] = 0
			data["marked"] = 0
			data["error"] = result.Error.Error()
		}
		logSubsystemResult(rt.logger, timestamp, daemonVerbose, subsystemResult{
			Name:        "Dead session detection",
			EventType:   "daemon.dead_session_detection",
			Error:       result.Error,
			Message:     result.Message,
			HasActivity: result.MarkedCount > 0 || result.EscalatedCount > 0,
			Data:        data,
		})
	}
}

// processCompletions handles auto-closing agents that report Phase: Complete.
// Returns the number of agents completed this cycle.
func (rt *daemonRuntime) processCompletions(timestamp string) {
	completionConfig := daemon.CompletionConfig{
		ProjectDir: rt.projectDir,
		ServerURL:  serverURL,
		DryRun:     false,
		Verbose:    daemonVerbose,
	}
	completionResult, err := rt.d.CompletionOnce(completionConfig)
	if err != nil && daemonVerbose {
		fmt.Fprintf(os.Stderr, "[%s] Completion processing error: %v\n", timestamp, err)
	} else if completionResult != nil {
		completedThisCycle := 0
		for _, cr := range completionResult.Processed {
			if cr.Processed {
				completedThisCycle++
				rt.completed++
				rt.lastCompletion = time.Now()
				fmt.Printf("[%s] Auto-completed: %s (escalation=%s)\n",
					timestamp, cr.BeadsID, cr.Escalation)

				// Release the pool slot immediately on completion.
				if rt.d.Pool != nil && rt.d.Pool.ReleaseByBeadsID(cr.BeadsID) {
					if daemonVerbose {
						fmt.Printf("[%s] Released pool slot for %s\n", timestamp, cr.BeadsID)
					}
				}

				logDaemonEvent(rt.logger, "daemon.complete", map[string]interface{}{
					"beads_id":   cr.BeadsID,
					"reason":     cr.CloseReason,
					"escalation": cr.Escalation.String(),
					"source":     "daemon_auto_complete",
				})
			} else if cr.Error != nil && daemonVerbose {
				fmt.Printf("[%s] Completion blocked: %s - %v (escalation=%s)\n",
					timestamp, cr.BeadsID, cr.Error, cr.Escalation)
			}
		}
		if completedThisCycle > 0 && daemonVerbose {
			fmt.Printf("[%s] Auto-completed %d agent(s) this cycle\n", timestamp, completedThisCycle)
		}
	}
}

// processFactualQuestions spawns investigations for factual questions if enabled.
func (rt *daemonRuntime) processFactualQuestions(timestamp string) {
	if !rt.config.SpawnFactualQuestions {
		return
	}
	factualSpawned := rt.d.ProcessFactualQuestions()
	if factualSpawned > 0 {
		rt.processed += factualSpawned
		rt.lastSpawn = time.Now()
		fmt.Printf("[%s] Spawned %d investigation(s) for factual questions\n", timestamp, factualSpawned)
	} else if daemonVerbose && !rt.d.AtCapacity() {
		fmt.Printf("[%s] No factual questions to spawn\n", timestamp)
	}
}

// writeStatus writes the daemon status file with current state.
func (rt *daemonRuntime) writeStatus(timestamp string, pollTime time.Time) {
	readyIssues, _ := daemon.ListReadyIssuesWithLabel(rt.config.Label)
	readyCount := len(readyIssues)

	status := daemon.DaemonStatus{
		Capacity: daemon.CapacityStatus{
			Max:       rt.config.MaxAgents,
			Active:    rt.d.ActiveCount(),
			Available: rt.d.AvailableSlots(),
		},
		LastPoll:       pollTime,
		LastSpawn:      rt.lastSpawn,
		LastCompletion: rt.lastCompletion,
		ReadyCount:     readyCount,
		Status:         daemon.DetermineStatus(pollTime, rt.config.PollInterval),
	}
	if err := daemon.WriteStatusFile(status); err != nil && daemonVerbose {
		fmt.Fprintf(os.Stderr, "Warning: failed to write status file: %v\n", err)
	}
}

// processSpawns runs the inner spawn loop, spawning issues until the queue is
// empty or the daemon reaches capacity. Returns the number spawned this cycle.
func (rt *daemonRuntime) processSpawns(timestamp string) int {
	spawnedThisCycle := 0
	skippedThisCycle := make(map[string]bool)

	for {
		// Check for interrupt
		select {
		case <-rt.ctx.Done():
			return spawnedThisCycle
		default:
		}

		// Check capacity
		if rt.d.AtCapacity() {
			if daemonVerbose {
				fmt.Printf("[%s] At capacity, stopping this cycle\n", timestamp)
			}
			break
		}

		// Use cross-project or single-project polling based on config
		if rt.config.CrossProject {
			if rt.spawnCrossProject(timestamp, skippedThisCycle, &spawnedThisCycle) {
				break // No more issues or fatal error
			}
		} else {
			if rt.spawnSingleProject(timestamp, skippedThisCycle, &spawnedThisCycle) {
				break // No more issues or fatal error
			}
		}

		// Delay before next spawn to avoid rate limits
		select {
		case <-rt.ctx.Done():
			return spawnedThisCycle
		case <-time.After(rt.config.SpawnDelay):
		}
	}

	return spawnedThisCycle
}

// spawnCrossProject attempts to spawn one issue from any kb-registered project.
// Returns true if the spawn loop should break (no more issues or fatal error).
func (rt *daemonRuntime) spawnCrossProject(timestamp string, skippedThisCycle map[string]bool, spawnedThisCycle *int) bool {
	cpResult, err := rt.d.CrossProjectOnceExcluding(skippedThisCycle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return true
	}

	if !cpResult.Processed {
		// Check if this is a spawn failure
		if cpResult.Issue != nil && cpResult.Error != nil {
			skipKey := fmt.Sprintf("%s:%s", cpResult.Project.Path, cpResult.Issue.ID)
			skippedThisCycle[skipKey] = true
			fmt.Fprintf(os.Stderr, "[%s] [%s] Skipping %s: %v\n",
				timestamp, cpResult.ProjectName, cpResult.Issue.ID, cpResult.Error)
			return false // Continue trying other issues
		}

		// No more issues across all projects
		if daemonVerbose && *spawnedThisCycle == 0 {
			fmt.Printf("[%s] %s\n", timestamp, cpResult.Message)
		}
		return true
	}

	rt.processed++
	*spawnedThisCycle++
	rt.lastSpawn = time.Now()
	fmt.Printf("[%s] [%s] Spawned: %s (%s) - %s\n",
		timestamp,
		cpResult.ProjectName,
		cpResult.Issue.ID,
		cpResult.Skill,
		cpResult.Issue.Title,
	)

	logDaemonEvent(rt.logger, "daemon.spawn", map[string]interface{}{
		"beads_id": cpResult.Issue.ID,
		"skill":    cpResult.Skill,
		"title":    cpResult.Issue.Title,
		"project":  cpResult.ProjectName,
		"count":    rt.processed,
	})

	return false
}

// spawnSingleProject attempts to spawn one issue from the current project.
// Returns true if the spawn loop should break (no more issues or fatal error).
func (rt *daemonRuntime) spawnSingleProject(timestamp string, skippedThisCycle map[string]bool, spawnedThisCycle *int) bool {
	result, err := rt.d.OnceExcluding(skippedThisCycle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return true
	}

	if !result.Processed {
		// Check if this is a spawn failure (not queue empty or capacity)
		if result.Issue != nil && result.Error != nil {
			skippedThisCycle[result.Issue.ID] = true
			fmt.Fprintf(os.Stderr, "[%s] Skipping %s: %v\n",
				timestamp, result.Issue.ID, result.Error)
			return false // Continue trying other issues
		}

		// No more issues or non-issue-specific error
		if daemonVerbose && *spawnedThisCycle == 0 {
			fmt.Printf("[%s] %s\n", timestamp, result.Message)
		}
		return true
	}

	rt.processed++
	*spawnedThisCycle++
	rt.lastSpawn = time.Now()
	fmt.Printf("[%s] Spawned: %s (%s) - %s\n",
		timestamp,
		result.Issue.ID,
		result.Skill,
		result.Issue.Title,
	)

	logDaemonEvent(rt.logger, "daemon.spawn", map[string]interface{}{
		"beads_id": result.Issue.ID,
		"skill":    result.Skill,
		"title":    result.Issue.Title,
		"count":    rt.processed,
	})

	return false
}

// stopMessage returns the standard daemon stop summary.
func (rt *daemonRuntime) stopMessage() string {
	return fmt.Sprintf("Daemon stopped. Spawned %d, completed %d, cycles %d.",
		rt.processed, rt.completed, rt.cycles)
}
