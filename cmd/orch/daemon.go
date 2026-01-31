// Package main provides the CLI entry point for orch-go.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Autonomous overnight processing",
	Long: `Daemon commands for autonomous overnight processing.

The daemon processes beads issues from the queue, spawning agents
for each issue in priority order.

Subcommands:
  run      Process issues continuously with polling
  once     Process a single issue and exit
  preview  Show what would be processed next without processing
  reflect  Run kb reflect analysis and store suggestions`,
}

var daemonRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Process issues continuously with polling",
	Long: `Process beads issues in priority order, spawning agents for each.

The daemon uses a worker pool pattern to control concurrency. It polls for
ready issues at the specified interval, respecting the concurrency limit
and only processing issues with the required label.

Runs continuously until interrupted with Ctrl+C. Use --poll-interval=0
to run once and exit (legacy behavior).

Cross-project mode (--cross-project) polls all kb-registered projects for
issues, using a shared global capacity pool. Projects must be registered
with 'kb projects add' to be included.

Examples:
  orch-go daemon run                        # Continuous polling (default 60s)
  orch-go daemon run --poll-interval 30     # Poll every 30 seconds
  orch-go daemon run --poll-interval 0      # Run once and exit
  orch-go daemon run --concurrency 5        # Allow up to 5 concurrent agents
  orch-go daemon run --max-agents 5         # Same as --concurrency (alias)
  orch-go daemon run --label triage:ready   # Only process issues with this label
  orch-go daemon run --dry-run              # Preview without spawning
  orch-go daemon run --cross-project        # Poll all kb-registered projects`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonLoop()
	},
}

var daemonOnceCmd = &cobra.Command{
	Use:   "once",
	Short: "Process a single issue and exit",
	Long: `Process the next issue from the queue and exit.

Useful for testing or manual step-by-step processing.

Examples:
  orch-go daemon once`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonOnce()
	},
}

var daemonPreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Show what would be processed next without processing",
	Long: `Preview the next issue that would be processed by the daemon.

Shows issue details and inferred skill without actually spawning an agent.

Examples:
  orch-go daemon preview
  orch-go daemon preview --cross-project   # Preview from all kb-registered projects`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonPreview()
	},
}

var daemonReflectCmd = &cobra.Command{
	Use:   "reflect",
	Short: "Run kb reflect analysis and store suggestions",
	Long: `Run knowledge reflection analysis and store suggestions for SessionStart hook.

This command runs 'kb reflect --format json' to detect:
- Investigation clusters needing synthesis
- kn entries worth promoting to decisions
- Stale decisions with no citations
- Constraints that may conflict with code

Results are stored in ~/.orch/reflect-suggestions.json and surfaced
by the SessionStart hook at next session start.

Examples:
  orch-go daemon reflect`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonReflect()
	},
}

var (
	// Daemon flags
	daemonDelay                        int    // Delay between spawns in seconds
	daemonDryRun                       bool   // Preview mode - show what would be processed without spawning
	daemonPollInterval                 int    // Poll interval in seconds (0 = run once)
	daemonMaxAgents                    int    // Maximum concurrent agents (0 = no limit)
	daemonLabel                        string // Filter issues by label
	daemonVerbose                      bool   // Enable verbose output
	daemonReflect                      bool   // Run reflection analysis after processing (on exit)
	daemonReflectInterval              int    // Periodic reflection interval in minutes (0 = disabled)
	daemonReflectIssues                bool   // Create beads issues for synthesis opportunities
	daemonCleanupEnabled               bool   // Enable periodic session cleanup
	daemonCleanupInterval              int    // Cleanup interval in minutes (0 = disabled)
	daemonCleanupSessions              bool   // Clean stale OpenCode sessions
	daemonCleanupSessionsAge           int    // Session age threshold in days
	daemonCleanupWorkspaces            bool   // Archive stale completed workspaces
	daemonCleanupWorkspacesAge         int    // Workspace age threshold in days
	daemonCleanupInvestigations        bool   // Archive empty investigation files
	daemonCleanupPreserveOrch          bool   // Preserve orchestrator sessions/workspaces during cleanup
	daemonCrossProject                 bool   // Poll all kb-registered projects for issues
	daemonSpawnFactualQuestions        bool   // Spawn investigations for factual questions (subtype:factual label)
	daemonDeadSessionDetectionEnabled  bool   // Enable dead session detection
	daemonDeadSessionDetectionInterval int    // Dead session detection interval in minutes (0 = disabled)
)

func init() {
	daemonCmd.AddCommand(daemonRunCmd)
	daemonCmd.AddCommand(daemonOnceCmd)
	daemonCmd.AddCommand(daemonPreviewCmd)
	daemonCmd.AddCommand(daemonReflectCmd)

	// Spawn delay between issues
	daemonRunCmd.Flags().IntVar(&daemonDelay, "delay", 10, "Delay between spawns in seconds")
	daemonRunCmd.Flags().BoolVar(&daemonDryRun, "dry-run", false, "Preview mode - show what would be processed without spawning")

	// New flags for continuous polling
	daemonRunCmd.Flags().IntVar(&daemonPollInterval, "poll-interval", 60, "Poll interval in seconds (0 = run once and exit)")
	daemonRunCmd.Flags().IntVarP(&daemonMaxAgents, "concurrency", "c", 3, "Maximum concurrent agents (0 = no limit)")
	daemonRunCmd.Flags().IntVar(&daemonMaxAgents, "max-agents", 3, "Maximum concurrent agents (alias for --concurrency)")
	daemonRunCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
	daemonRunCmd.Flags().BoolVarP(&daemonVerbose, "verbose", "v", false, "Enable verbose output")
	daemonRunCmd.Flags().BoolVar(&daemonReflect, "reflect", true, "Run kb reflect analysis on exit (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonReflectInterval, "reflect-interval", 60, "Periodic reflection interval in minutes (0 = disabled, default: 60)")
	daemonRunCmd.Flags().BoolVar(&daemonReflectIssues, "reflect-issues", true, "Create beads issues for synthesis opportunities (default: true)")
	daemonRunCmd.Flags().BoolVar(&daemonCleanupEnabled, "cleanup-enabled", true, "Enable periodic cleanup (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonCleanupInterval, "cleanup-interval", 360, "Cleanup interval in minutes (0 = disabled, default: 360 = 6 hours)")
	daemonRunCmd.Flags().BoolVar(&daemonCleanupSessions, "cleanup-sessions", true, "Clean stale OpenCode sessions (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonCleanupSessionsAge, "cleanup-sessions-age", 7, "Session age threshold in days (default: 7)")
	daemonRunCmd.Flags().BoolVar(&daemonCleanupWorkspaces, "cleanup-workspaces", true, "Archive stale completed workspaces (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonCleanupWorkspacesAge, "cleanup-workspaces-age", 7, "Workspace age threshold in days (default: 7)")
	daemonRunCmd.Flags().BoolVar(&daemonCleanupInvestigations, "cleanup-investigations", true, "Archive empty investigation files (default: true)")
	daemonRunCmd.Flags().BoolVar(&daemonCleanupPreserveOrch, "cleanup-preserve-orchestrator", true, "Preserve orchestrator sessions/workspaces during cleanup (default: true)")
	// Mark max-agents as hidden since --concurrency is the preferred name
	daemonRunCmd.Flags().MarkHidden("max-agents")

	// Cross-project mode
	daemonRunCmd.Flags().BoolVar(&daemonCrossProject, "cross-project", false, "Poll all kb-registered projects for issues")

	// Factual questions spawning
	daemonRunCmd.Flags().BoolVar(&daemonSpawnFactualQuestions, "spawn-factual-questions", false, "Spawn investigations for factual questions (subtype:factual label)")

	// Dead session detection
	daemonRunCmd.Flags().BoolVar(&daemonDeadSessionDetectionEnabled, "dead-session-detection", true, "Enable dead session detection (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonDeadSessionDetectionInterval, "dead-session-interval", 10, "Dead session detection interval in minutes (0 = disabled, default: 10)")

	// Add label filter to preview and once commands (share the same variable)
	daemonPreviewCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
	daemonPreviewCmd.Flags().BoolVar(&daemonCrossProject, "cross-project", false, "Preview issues from all kb-registered projects")
	daemonOnceCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
	daemonOnceCmd.Flags().BoolVar(&daemonCrossProject, "cross-project", false, "Process one issue from all kb-registered projects")
}

func runDaemonLoop() error {
	// Handle dry-run mode
	if daemonDryRun {
		return runDaemonDryRun()
	}

	// Get current directory for completion processing
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

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

	d := daemon.NewWithConfig(config)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, stopping daemon...")
		cancel()
	}()

	logger := events.NewLogger(events.DefaultLogPath())
	// Inject event logger into daemon for dedup telemetry
	d.SetEventLogger(logger)
	processed := 0
	completed := 0 // Track auto-completed agents
	cycles := 0
	var lastSpawn time.Time      // Track last successful spawn
	var lastCompletion time.Time // Track last auto-completion

	// Ensure reflection runs on exit if enabled
	if daemonReflect {
		defer runReflectionAnalysis(daemonVerbose)
	}

	// Clean up status file on shutdown
	defer daemon.RemoveStatusFile()

	fmt.Println("Starting daemon...")
	fmt.Printf("  Poll interval:    %s\n", formatDaemonDuration(config.PollInterval))
	fmt.Printf("  Concurrency:      %d (worker pool)\n", config.MaxAgents)
	countMode := "OpenCode sessions"
	if config.Backend == "docker" {
		countMode = "Docker containers"
	}
	fmt.Printf("  Backend:          %s (counting %s)\n", config.Backend, countMode)
	fmt.Printf("  Required label:   %s\n", config.Label)
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
	if config.DeadSessionDetectionEnabled {
		fmt.Printf("  Dead session det:  %s\n", formatDaemonDuration(config.DeadSessionDetectionInterval))
	} else {
		fmt.Println("  Dead session det:  disabled")
	}
	fmt.Println()

	// Main polling loop
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
			return nil
		default:
		}

		cycles++
		timestamp := time.Now().Format("15:04:05")
		pollTime := time.Now()

		// Check server health and update recovery state FIRST.
		// This enables detection of server restarts (down -> up transitions).
		serverAvailable := d.CheckServerHealth()
		if daemonVerbose {
			fmt.Printf("[%s] Server health: available=%v\n", timestamp, serverAvailable)
		}

		// Reconcile pool with actual OpenCode sessions.
		// This prevents stale capacity counts when agents complete without
		// the daemon knowing (overnight runs, crashes, manual kills).
		// Must happen before status write so status shows accurate counts.
		if freed := d.ReconcileWithOpenCode(); freed > 0 && daemonVerbose {
			fmt.Printf("[%s] Reconciled: freed %d stale slots\n", timestamp, freed)
		}

		// Run periodic reflection if due
		if result := d.RunPeriodicReflection(); result != nil {
			if result.Error != nil {
				fmt.Fprintf(os.Stderr, "[%s] Reflection error: %v\n", timestamp, result.Error)
			} else if result.Suggestions != nil && result.Suggestions.HasSuggestions() {
				fmt.Printf("[%s] Reflection: %s\n", timestamp, result.Suggestions.Summary())
			} else if daemonVerbose {
				fmt.Printf("[%s] Reflection: no suggestions found\n", timestamp)
			}
		}

		// Run periodic cleanup if due
		if result := d.RunPeriodicCleanup(); result != nil {
			if result.Error != nil {
				fmt.Fprintf(os.Stderr, "[%s] Cleanup error: %v\n", timestamp, result.Error)
				// Log the cleanup error
				event := events.Event{
					Type:      "daemon.cleanup",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"sessions_deleted":        result.SessionsDeleted,
						"workspaces_archived":     result.WorkspacesArchived,
						"investigations_archived": result.InvestigationsArchived,
						"error":                   result.Error.Error(),
						"message":                 result.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log cleanup error event: %v\n", err)
				}
			} else if result.SessionsDeleted > 0 || result.WorkspacesArchived > 0 || result.InvestigationsArchived > 0 {
				fmt.Printf("[%s] Cleanup: %s\n", timestamp, result.Message)
				// Log the successful cleanup
				event := events.Event{
					Type:      "daemon.cleanup",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"sessions_deleted":        result.SessionsDeleted,
						"workspaces_archived":     result.WorkspacesArchived,
						"investigations_archived": result.InvestigationsArchived,
						"message":                 result.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log cleanup event: %v\n", err)
				}
			} else if daemonVerbose {
				fmt.Printf("[%s] Cleanup: %s\n", timestamp, result.Message)
			}
		}

		// Run periodic stuck agent recovery if due
		if result := d.RunPeriodicRecovery(); result != nil {
			if result.Error != nil {
				fmt.Fprintf(os.Stderr, "[%s] Recovery error: %v\n", timestamp, result.Error)
				// Log the recovery error
				event := events.Event{
					Type:      "daemon.recovery",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"resumed": 0,
						"skipped": result.SkippedCount,
						"error":   result.Error.Error(),
						"message": result.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log recovery error event: %v\n", err)
				}
			} else if result.ResumedCount > 0 {
				fmt.Printf("[%s] Recovery: %s\n", timestamp, result.Message)
				// Log the successful recovery
				event := events.Event{
					Type:      "daemon.recovery",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"resumed": result.ResumedCount,
						"skipped": result.SkippedCount,
						"message": result.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log recovery event: %v\n", err)
				}
			} else if daemonVerbose {
				fmt.Printf("[%s] Recovery: no stuck agents found\n", timestamp)
			}
		}

		// Run server restart recovery if due (runs once after daemon startup)
		// This handles the case where OpenCode server was restarted and sessions
		// were lost from memory but persist on disk.
		serverRecoveryResult := d.RunServerRecovery()
		if daemonVerbose {
			if serverRecoveryResult == nil {
				fmt.Printf("[%s] [DEBUG] Server recovery: ShouldRunServerRecovery returned false (recovery not due)\n", timestamp)
			} else {
				fmt.Printf("[%s] [DEBUG] Server recovery: result=%+v\n", timestamp, serverRecoveryResult)
			}
		}
		if serverRecoveryResult != nil {
			if serverRecoveryResult.Error != nil {
				fmt.Fprintf(os.Stderr, "[%s] Server recovery error: %v\n", timestamp, serverRecoveryResult.Error)
				// Log the server recovery error
				event := events.Event{
					Type:      "daemon.server_recovery",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"orphaned": 0,
						"resumed":  0,
						"skipped":  serverRecoveryResult.SkippedCount,
						"error":    serverRecoveryResult.Error.Error(),
						"message":  serverRecoveryResult.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log server recovery error event: %v\n", err)
				}
			} else if serverRecoveryResult.OrphanedCount > 0 {
				fmt.Printf("[%s] Server recovery: %s\n", timestamp, serverRecoveryResult.Message)
				// Log the successful server recovery
				event := events.Event{
					Type:      "daemon.server_recovery",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"orphaned": serverRecoveryResult.OrphanedCount,
						"resumed":  serverRecoveryResult.ResumedCount,
						"skipped":  serverRecoveryResult.SkippedCount,
						"message":  serverRecoveryResult.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log server recovery event: %v\n", err)
				}
			} else if daemonVerbose {
				fmt.Printf("[%s] Server recovery: %s\n", timestamp, serverRecoveryResult.Message)
			}
		}

		// Run periodic dead session detection if due
		if result := d.RunPeriodicDeadSessionDetection(); result != nil {
			if result.Error != nil {
				fmt.Fprintf(os.Stderr, "[%s] Dead session detection error: %v\n", timestamp, result.Error)
				// Log the error
				event := events.Event{
					Type:      "daemon.dead_session_detection",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"detected": 0,
						"marked":   0,
						"skipped":  result.SkippedCount,
						"error":    result.Error.Error(),
						"message":  result.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log dead session detection error event: %v\n", err)
				}
			} else if result.MarkedCount > 0 {
				fmt.Printf("[%s] Dead session detection: %s\n", timestamp, result.Message)
				// Log the successful detection
				event := events.Event{
					Type:      "daemon.dead_session_detection",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"detected": result.DetectedCount,
						"marked":   result.MarkedCount,
						"skipped":  result.SkippedCount,
						"message":  result.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log dead session detection event: %v\n", err)
				}
			} else if daemonVerbose {
				fmt.Printf("[%s] Dead session detection: no dead sessions found\n", timestamp)
			}
		}

		// Process completions: auto-close agents that report Phase: Complete
		// This frees capacity slots for new work. Uses the escalation model:
		// - None/Info/Review: Auto-complete (closes issue)
		// - Block/Failed: Requires human review (issue stays open)
		completionConfig := daemon.CompletionConfig{
			ProjectDir: projectDir,
			ServerURL:  serverURL,
			DryRun:     false,
			Verbose:    daemonVerbose,
		}
		completionResult, err := d.CompletionOnce(completionConfig)
		if err != nil && daemonVerbose {
			fmt.Fprintf(os.Stderr, "[%s] Completion processing error: %v\n", timestamp, err)
		} else if completionResult != nil {
			completedThisCycle := 0
			for _, cr := range completionResult.Processed {
				if cr.Processed {
					completedThisCycle++
					completed++
					lastCompletion = time.Now()
					fmt.Printf("[%s] Auto-completed: %s (escalation=%s)\n",
						timestamp, cr.BeadsID, cr.Escalation)

					// Release the pool slot immediately on completion.
					// This provides active slot release without waiting for reconciliation,
					// fixing the capacity leak where beads lookup errors cause stuck counters.
					if d.Pool != nil && d.Pool.ReleaseByBeadsID(cr.BeadsID) {
						if daemonVerbose {
							fmt.Printf("[%s] Released pool slot for %s\n", timestamp, cr.BeadsID)
						}
					}

					// Log the completion
					event := events.Event{
						Type:      "daemon.complete",
						Timestamp: time.Now().Unix(),
						Data: map[string]interface{}{
							"beads_id":   cr.BeadsID,
							"reason":     cr.CloseReason,
							"escalation": cr.Escalation.String(),
							"source":     "daemon_auto_complete",
						},
					}
					if err := logger.Log(event); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to log completion event: %v\n", err)
					}
				} else if cr.Error != nil && daemonVerbose {
					fmt.Printf("[%s] Completion blocked: %s - %v (escalation=%s)\n",
						timestamp, cr.BeadsID, cr.Error, cr.Escalation)
				}
			}
			if completedThisCycle > 0 && daemonVerbose {
				fmt.Printf("[%s] Auto-completed %d agent(s) this cycle\n", timestamp, completedThisCycle)
			}
		}

		// Process factual questions if enabled
		// This happens after completions free up capacity but before regular issue polling
		if config.SpawnFactualQuestions {
			factualSpawned := d.ProcessFactualQuestions()
			if factualSpawned > 0 {
				processed += factualSpawned
				lastSpawn = time.Now()
				fmt.Printf("[%s] Spawned %d investigation(s) for factual questions\n", timestamp, factualSpawned)
			} else if daemonVerbose && !d.AtCapacity() {
				fmt.Printf("[%s] No factual questions to spawn\n", timestamp)
			}
		}

		// Get ready issues count for status (filtered by configured label)
		readyIssues, _ := daemon.ListReadyIssuesWithLabel(config.Label)
		readyCount := len(readyIssues)

		// Write daemon status file AFTER reconciliation and completions so counts are accurate
		status := daemon.DaemonStatus{
			Capacity: daemon.CapacityStatus{
				Max:       config.MaxAgents,
				Active:    d.ActiveCount(),
				Available: d.AvailableSlots(),
			},
			LastPoll:       pollTime,
			LastSpawn:      lastSpawn,
			LastCompletion: lastCompletion,
			ReadyCount:     readyCount,
			Status:         daemon.DetermineStatus(pollTime, config.PollInterval),
		}
		if err := daemon.WriteStatusFile(status); err != nil && daemonVerbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to write status file: %v\n", err)
		}

		// Check capacity before polling
		if d.AtCapacity() {
			activeCount := d.ActiveCount()
			if daemonVerbose {
				fmt.Printf("[%s] At capacity (%d/%d agents active), waiting...\n",
					timestamp, activeCount, daemonMaxAgents)
			}
			// Wait for poll interval before checking again
			select {
			case <-ctx.Done():
				fmt.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
				return nil
			case <-time.After(config.PollInterval):
				continue
			}
		}

		if daemonVerbose {
			fmt.Printf("[%s] Polling for issues...\n", timestamp)
		}

		// Process issues until queue is empty or at capacity
		// Track issues that failed to spawn this cycle (e.g., failure report gate)
		// to skip them and continue with other issues.
		spawnedThisCycle := 0
		skippedThisCycle := make(map[string]bool)
		for {
			// Check for interrupt
			select {
			case <-ctx.Done():
				fmt.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
				return nil
			default:
			}

			// Check capacity
			if d.AtCapacity() {
				if daemonVerbose {
					fmt.Printf("[%s] At capacity, stopping this cycle\n", timestamp)
				}
				break
			}

			// Use cross-project or single-project polling based on config
			if config.CrossProject {
				// Cross-project polling: iterate over all kb-registered projects
				cpResult, err := d.CrossProjectOnceExcluding(skippedThisCycle)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					break
				}

				if !cpResult.Processed {
					// Check if this is a spawn failure
					if cpResult.Issue != nil && cpResult.Error != nil {
						// For cross-project, skip key is "projectPath:issueID"
						skipKey := fmt.Sprintf("%s:%s", cpResult.Project.Path, cpResult.Issue.ID)
						skippedThisCycle[skipKey] = true
						fmt.Fprintf(os.Stderr, "[%s] [%s] Skipping %s: %v\n",
							timestamp, cpResult.ProjectName, cpResult.Issue.ID, cpResult.Error)
						continue
					}

					// No more issues across all projects
					if daemonVerbose && spawnedThisCycle == 0 {
						fmt.Printf("[%s] %s\n", timestamp, cpResult.Message)
					}
					break
				}

				processed++
				spawnedThisCycle++
				lastSpawn = time.Now()
				fmt.Printf("[%s] [%s] Spawned: %s (%s) - %s\n",
					timestamp,
					cpResult.ProjectName,
					cpResult.Issue.ID,
					cpResult.Skill,
					cpResult.Issue.Title,
				)

				// Log the spawn with project context
				event := events.Event{
					Type:      "daemon.spawn",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"beads_id": cpResult.Issue.ID,
						"skill":    cpResult.Skill,
						"title":    cpResult.Issue.Title,
						"project":  cpResult.ProjectName,
						"count":    processed,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
				}
			} else {
				// Single-project polling (original behavior)
				result, err := d.OnceExcluding(skippedThisCycle)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					break
				}

				if !result.Processed {
					// Check if this is a spawn failure (not queue empty or capacity)
					if result.Issue != nil && result.Error != nil {
						skippedThisCycle[result.Issue.ID] = true
						fmt.Fprintf(os.Stderr, "[%s] Skipping %s: %v\n",
							timestamp, result.Issue.ID, result.Error)
						continue
					}

					// No more issues or non-issue-specific error
					if daemonVerbose && spawnedThisCycle == 0 {
						fmt.Printf("[%s] %s\n", timestamp, result.Message)
					}
					break
				}

				processed++
				spawnedThisCycle++
				lastSpawn = time.Now()
				fmt.Printf("[%s] Spawned: %s (%s) - %s\n",
					timestamp,
					result.Issue.ID,
					result.Skill,
					result.Issue.Title,
				)

				// Log the spawn
				event := events.Event{
					Type:      "daemon.spawn",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"beads_id": result.Issue.ID,
						"skill":    result.Skill,
						"title":    result.Issue.Title,
						"count":    processed,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
				}
			}

			// Delay before next spawn to avoid rate limits
			select {
			case <-ctx.Done():
				fmt.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
				return nil
			case <-time.After(config.SpawnDelay):
			}
		}

		// If poll interval is 0, run once and exit
		if config.PollInterval == 0 {
			fmt.Printf("Run-once mode. Spawned %d, completed %d.\n", processed, completed)
			return nil
		}

		// Wait for next poll cycle
		if daemonVerbose {
			fmt.Printf("[%s] Spawned %d this cycle, waiting %s before next poll...\n",
				timestamp, spawnedThisCycle, formatDaemonDuration(config.PollInterval))
		}
		select {
		case <-ctx.Done():
			fmt.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
			return nil
		case <-time.After(config.PollInterval):
		}
	}
}

// formatDaemonDuration formats a duration for daemon display.
func formatDaemonDuration(d time.Duration) string {
	if d == 0 {
		return "0 (run once)"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return d.String()
}

func runDaemonDryRun() error {
	config := daemon.Config{
		Label:        daemonLabel,
		CrossProject: daemonCrossProject,
	}
	d := daemon.NewWithConfig(config)

	// Inject event logger for dedup telemetry
	logger := events.NewLogger(events.DefaultLogPath())
	d.SetEventLogger(logger)

	// Configure hotspot checking for dry-run
	d.HotspotChecker = daemon.NewGitHotspotChecker()

	// Use cross-project preview if enabled
	if config.CrossProject {
		cpResult, err := d.CrossProjectPreview()
		if err != nil {
			return fmt.Errorf("preview error: %w", err)
		}

		fmt.Println("[DRY-RUN] Would process the following issue:")
		fmt.Println()
		fmt.Print(daemon.FormatCrossProjectPreview(cpResult))
		fmt.Println("\nNo agents were spawned (dry-run mode).")
		return nil
	}

	// Single-project preview (original behavior)
	result, err := d.Preview()
	if err != nil {
		return fmt.Errorf("preview error: %w", err)
	}

	fmt.Println("[DRY-RUN] Would process the following issue:")
	fmt.Println()

	// Get current directory for context
	projectDir, _ := os.Getwd()
	projectName := filepath.Base(projectDir)

	if result.Issue != nil {
		fmt.Printf("  Project:  %s\n", projectName)
		fmt.Println(daemon.FormatPreview(result.Issue))
		fmt.Printf("\nInferred skill: %s\n", result.Skill)

		// Display hotspot warnings if any
		if result.HasHotspotWarnings() {
			fmt.Print(daemon.FormatHotspotWarnings(result.HotspotWarnings))
		}
	} else {
		fmt.Println("No spawnable issues in queue")
	}

	// Display rejected issues with reasons
	if len(result.RejectedIssues) > 0 {
		fmt.Print(daemon.FormatRejectedIssues(result.RejectedIssues))
	}

	fmt.Println("\nNo agents were spawned (dry-run mode).")

	return nil
}

func runDaemonOnce() error {
	config := daemon.Config{
		Label:        daemonLabel,
		CrossProject: daemonCrossProject,
	}
	d := daemon.NewWithConfig(config)

	// Inject event logger for dedup telemetry
	logger := events.NewLogger(events.DefaultLogPath())
	d.SetEventLogger(logger)

	// Use cross-project version if enabled
	if config.CrossProject {
		cpResult, err := d.CrossProjectOnce()
		if err != nil {
			return fmt.Errorf("daemon error: %w", err)
		}

		if !cpResult.Processed {
			fmt.Println(cpResult.Message)
			return nil
		}

		fmt.Printf("[%s] Spawned: %s\n", cpResult.ProjectName, cpResult.Issue.ID)
		fmt.Printf("  Title:  %s\n", cpResult.Issue.Title)
		fmt.Printf("  Type:   %s\n", cpResult.Issue.IssueType)
		fmt.Printf("  Skill:  %s\n", cpResult.Skill)

		// Log the spawn
		event := events.Event{
			Type:      "daemon.once",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"beads_id": cpResult.Issue.ID,
				"skill":    cpResult.Skill,
				"title":    cpResult.Issue.Title,
				"project":  cpResult.ProjectName,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
		}
		return nil
	}

	// Single-project version (original behavior)
	result, err := d.Once()
	if err != nil {
		return fmt.Errorf("daemon error: %w", err)
	}

	if !result.Processed {
		fmt.Println(result.Message)
		return nil
	}

	fmt.Printf("Spawned: %s\n", result.Issue.ID)
	fmt.Printf("  Title:  %s\n", result.Issue.Title)
	fmt.Printf("  Type:   %s\n", result.Issue.IssueType)
	fmt.Printf("  Skill:  %s\n", result.Skill)

	// Log the spawn
	event := events.Event{
		Type:      "daemon.once",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id": result.Issue.ID,
			"skill":    result.Skill,
			"title":    result.Issue.Title,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	return nil
}

func runDaemonPreview() error {
	config := daemon.Config{
		Label:        daemonLabel,
		CrossProject: daemonCrossProject,
	}
	d := daemon.NewWithConfig(config)

	// Inject event logger for dedup telemetry
	logger := events.NewLogger(events.DefaultLogPath())
	d.SetEventLogger(logger)

	// Configure hotspot checking for preview
	d.HotspotChecker = daemon.NewGitHotspotChecker()

	// Use cross-project preview if enabled
	if config.CrossProject {
		cpResult, err := d.CrossProjectPreview()
		if err != nil {
			return fmt.Errorf("preview error: %w", err)
		}

		fmt.Print(daemon.FormatCrossProjectPreview(cpResult))

		if cpResult.NextIssue != nil {
			fmt.Println("\nRun 'orch daemon once --cross-project' to process this issue.")
		}
		return nil
	}

	// Single-project preview (original behavior)
	result, err := d.Preview()
	if err != nil {
		return fmt.Errorf("preview error: %w", err)
	}

	// Get current directory for context
	projectDir, _ := os.Getwd()
	projectName := filepath.Base(projectDir)

	// Display spawnable issue if available
	if result.Issue != nil {
		fmt.Println("Spawnable issues:")
		fmt.Printf("  Project:  %s\n", projectName)
		fmt.Println(daemon.FormatPreview(result.Issue))
		fmt.Printf("\nInferred skill: %s\n", result.Skill)

		// Display hotspot warnings if any
		if result.HasHotspotWarnings() {
			fmt.Print(daemon.FormatHotspotWarnings(result.HotspotWarnings))
		}
	} else {
		fmt.Println(result.Message)
	}

	// Display rejected issues with reasons
	if len(result.RejectedIssues) > 0 {
		fmt.Print(daemon.FormatRejectedIssues(result.RejectedIssues))
	}

	if result.Issue != nil {
		fmt.Println("\nRun 'orch-go daemon once' to process this issue.")
	}

	return nil
}

func runDaemonReflect() error {
	fmt.Println("Running knowledge reflection analysis...")

	result := daemon.RunAndSaveReflection()
	if result.Error != nil {
		return fmt.Errorf("reflection error: %w", result.Error)
	}

	if result.Suggestions == nil || !result.Suggestions.HasSuggestions() {
		fmt.Println("No reflection suggestions found.")
		return nil
	}

	// Print summary
	fmt.Printf("\n%s\n", result.Suggestions.Summary())

	// Print details by category
	if len(result.Suggestions.Synthesis) > 0 {
		fmt.Printf("\nSynthesis Opportunities (%d):\n", len(result.Suggestions.Synthesis))
		for _, s := range result.Suggestions.Synthesis[:min(5, len(result.Suggestions.Synthesis))] {
			fmt.Printf("  - %s: %d investigations\n", s.Topic, s.Count)
		}
		if len(result.Suggestions.Synthesis) > 5 {
			fmt.Printf("  ... and %d more\n", len(result.Suggestions.Synthesis)-5)
		}
	}

	if len(result.Suggestions.Promote) > 0 {
		fmt.Printf("\nPromotion Candidates (%d):\n", len(result.Suggestions.Promote))
		for _, p := range result.Suggestions.Promote[:min(5, len(result.Suggestions.Promote))] {
			fmt.Printf("  - %s\n", truncateDaemonString(p.Content, 60))
		}
	}

	if len(result.Suggestions.Stale) > 0 {
		fmt.Printf("\nStale Decisions (%d):\n", len(result.Suggestions.Stale))
		for _, s := range result.Suggestions.Stale[:min(5, len(result.Suggestions.Stale))] {
			fmt.Printf("  - %s (%d days old)\n", filepath.Base(s.Path), s.Age)
		}
	}

	if len(result.Suggestions.Drift) > 0 {
		fmt.Printf("\nPotential Drifts (%d):\n", len(result.Suggestions.Drift))
		for _, d := range result.Suggestions.Drift[:min(5, len(result.Suggestions.Drift))] {
			fmt.Printf("  - %s\n", truncateDaemonString(d.Content, 60))
		}
	}

	if result.Saved {
		fmt.Printf("\nSuggestions saved to: %s\n", daemon.SuggestionsPath())
		fmt.Println("They will be surfaced at next session start.")
	}

	return nil
}

func truncateDaemonString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// runReflectionAnalysis runs kb reflect and saves suggestions.
// Called at the end of daemon processing to update reflection suggestions.
func runReflectionAnalysis(verbose bool) {
	if verbose {
		fmt.Println("Running reflection analysis...")
	}

	result := daemon.RunAndSaveReflection()
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Warning: reflection analysis failed: %v\n", result.Error)
		return
	}

	if result.Suggestions == nil || !result.Suggestions.HasSuggestions() {
		if verbose {
			fmt.Println("No reflection suggestions found.")
		}
		return
	}

	fmt.Printf("Reflection: %s\n", result.Suggestions.Summary())
	if result.Saved {
		if verbose {
			fmt.Printf("Suggestions saved to: %s\n", daemon.SuggestionsPath())
		}
	}
}
