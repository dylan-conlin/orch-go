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

Examples:
  orch-go daemon run                        # Continuous polling (default 60s)
  orch-go daemon run --poll-interval 30     # Poll every 30 seconds
  orch-go daemon run --poll-interval 0      # Run once and exit
  orch-go daemon run --concurrency 5        # Allow up to 5 concurrent agents
  orch-go daemon run --max-agents 5         # Same as --concurrency (alias)
  orch-go daemon run --label triage:ready   # Only process issues with this label
  orch-go daemon run --dry-run              # Preview without spawning`,
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
  orch-go daemon preview`,
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

var daemonResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume daemon after verification pause",
	Long: `Resume the daemon after reviewing completed work.

When the daemon marks N issues as ready-for-review without human verification (manual orch complete),
it pauses spawning to enforce the verifiability-first constraint. After reviewing the
completed work, run this command to reset the completion counter and resume operation.

The daemon checks for the resume signal on each poll cycle and automatically resumes.

Examples:
  orch daemon resume          # Resume after reviewing completed work`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonResume()
	},
}

var (
	// Daemon flags
	daemonDelay               int    // Delay between spawns in seconds
	daemonDryRun              bool   // Preview mode - show what would be processed without spawning
	daemonPollInterval        int    // Poll interval in seconds (0 = run once)
	daemonMaxAgents           int    // Maximum concurrent agents (0 = no limit)
	daemonLabel               string // Filter issues by label
	daemonVerbose             bool   // Enable verbose output
	daemonReflect             bool   // Run reflection analysis after processing (on exit)
	daemonReflectInterval     int    // Periodic reflection interval in minutes (0 = disabled)
	daemonReflectIssues       bool   // Create beads issues for synthesis opportunities
	daemonReflectOpen         bool   // Create beads issues for open investigation actions
	daemonModelDriftInterval  int    // Periodic model drift reflection interval in minutes (0 = disabled)
	daemonCleanupEnabled      bool   // Enable periodic session cleanup
	daemonCleanupInterval     int    // Session cleanup interval in minutes (0 = disabled)
	daemonCleanupAge          int    // Session age threshold in days for cleanup
	daemonCleanupPreserveOrch bool   // Preserve orchestrator sessions during cleanup
)

func init() {
	daemonCmd.AddCommand(daemonRunCmd)
	daemonCmd.AddCommand(daemonOnceCmd)
	daemonCmd.AddCommand(daemonPreviewCmd)
	daemonCmd.AddCommand(daemonReflectCmd)
	daemonCmd.AddCommand(daemonResumeCmd)

	// Spawn delay between issues
	daemonRunCmd.Flags().IntVar(&daemonDelay, "delay", 3, "Delay between spawns in seconds")
	daemonRunCmd.Flags().BoolVar(&daemonDryRun, "dry-run", false, "Preview mode - show what would be processed without spawning")

	// New flags for continuous polling
	daemonRunCmd.Flags().IntVar(&daemonPollInterval, "poll-interval", 15, "Poll interval in seconds (0 = run once and exit)")
	daemonRunCmd.Flags().IntVarP(&daemonMaxAgents, "concurrency", "c", 3, "Maximum concurrent agents (0 = no limit)")
	daemonRunCmd.Flags().IntVar(&daemonMaxAgents, "max-agents", 3, "Maximum concurrent agents (alias for --concurrency)")
	daemonRunCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
	daemonRunCmd.Flags().BoolVarP(&daemonVerbose, "verbose", "v", false, "Enable verbose output")
	daemonRunCmd.Flags().BoolVar(&daemonReflect, "reflect", true, "Run kb reflect analysis on exit (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonReflectInterval, "reflect-interval", 60, "Periodic reflection interval in minutes (0 = disabled, default: 60)")
	daemonRunCmd.Flags().BoolVar(&daemonReflectIssues, "reflect-issues", true, "Create beads issues for synthesis opportunities (default: true)")
	daemonRunCmd.Flags().BoolVar(&daemonReflectOpen, "reflect-open", true, "Create beads issues for open investigation actions (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonModelDriftInterval, "reflect-model-drift-interval", 240, "Model drift reflection interval in minutes (0 = disabled, default: 240 = 4 hours)")
	daemonRunCmd.Flags().BoolVar(&daemonCleanupEnabled, "cleanup-enabled", true, "Enable periodic session cleanup (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonCleanupInterval, "cleanup-interval", 360, "Session cleanup interval in minutes (0 = disabled, default: 360 = 6 hours)")
	daemonRunCmd.Flags().IntVar(&daemonCleanupAge, "cleanup-age", 7, "Session age threshold in days for cleanup (default: 7)")
	daemonRunCmd.Flags().BoolVar(&daemonCleanupPreserveOrch, "cleanup-preserve-orchestrator", true, "Preserve orchestrator sessions during cleanup (default: true)")
	// Mark max-agents as hidden since --concurrency is the preferred name
	daemonRunCmd.Flags().MarkHidden("max-agents")

	// Add label filter to preview and once commands (share the same variable)
	daemonPreviewCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
	daemonOnceCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
}

// daemonConfigFromFlags builds a Config starting from DefaultConfig(),
// overriding with CLI flag values. All daemon paths (run, once, dry-run,
// preview) MUST use this function instead of constructing Config directly.
func daemonConfigFromFlags() daemon.Config {
	config := daemon.DefaultConfig()

	// Override with CLI flags
	config.PollInterval = time.Duration(daemonPollInterval) * time.Second
	config.MaxAgents = daemonMaxAgents
	config.Label = daemonLabel
	config.SpawnDelay = time.Duration(daemonDelay) * time.Second
	config.DryRun = daemonDryRun
	config.Verbose = daemonVerbose
	config.ReflectEnabled = daemonReflectInterval > 0
	config.ReflectInterval = time.Duration(daemonReflectInterval) * time.Minute
	config.ReflectCreateIssues = daemonReflectIssues
	config.ReflectOpenEnabled = daemonReflectOpen
	config.ReflectModelDriftEnabled = daemonModelDriftInterval > 0
	config.ReflectModelDriftInterval = time.Duration(daemonModelDriftInterval) * time.Minute
	config.CleanupEnabled = daemonCleanupEnabled && daemonCleanupInterval > 0
	config.CleanupInterval = time.Duration(daemonCleanupInterval) * time.Minute
	config.CleanupAgeDays = daemonCleanupAge
	config.CleanupPreserveOrchestrator = daemonCleanupPreserveOrch
	config.CleanupServerURL = serverURL

	return config
}

// seedVerificationTracker seeds the tracker with the backlog count.
// Called after daemon construction, before entering the main loop.
func seedVerificationTracker(d *daemon.Daemon) {
	if d.VerificationTracker == nil {
		return
	}

	count, err := daemon.CountUnverifiedCompletions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not seed verification tracker: %v\n", err)
		return
	}

	if count > 0 {
		d.VerificationTracker.SeedFromBacklog(count)
		fmt.Printf("  Verification backlog: %d unverified completions from previous sessions\n", count)

		if d.VerificationTracker.IsPaused() {
			status := d.VerificationTracker.Status()
			fmt.Printf("  Warning: Verification pause: backlog exceeds threshold (%d/%d)\n",
				count, status.Threshold)
			fmt.Println("  Run 'orch daemon resume' after reviewing completed work")
		}
	}
}

func runDaemonLoop() error {
	// Handle dry-run mode
	if daemonDryRun {
		return runDaemonDryRun()
	}

	// Acquire PID lock to ensure single daemon instance.
	// This prevents multiple daemon processes from accumulating silently
	// and fighting over the status file and spawns.
	pidLock, err := daemon.AcquirePIDLock()
	if err != nil {
		return fmt.Errorf("cannot start daemon: %w", err)
	}
	defer pidLock.Release()

	// Get current directory for completion processing
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	config := daemonConfigFromFlags()
	d := daemon.NewWithConfig(config)

	// Initialize project registry for cross-project issue resolution.
	// If kb projects list fails (kb not installed, no projects), daemon still works
	// but spawns everything into the current directory.
	registry, err := daemon.NewProjectRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: project registry unavailable: %v\n", err)
	} else {
		d.ProjectRegistry = registry
	}

	// NOTE: Extraction system disabled. HotspotChecker is not set, so the
	// extraction gate in Once() and hotspot warnings in Preview() are skipped.
	// The daemon goes straight from polling bd ready to spawning issues.
	// To re-enable, uncomment: d.HotspotChecker = daemon.NewGitHotspotChecker()

	// Seed verification tracker with unverified backlog from previous sessions
	seedVerificationTracker(d)

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
	processed := 0
	completed := 0 // Track agents marked ready-for-review
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
	fmt.Printf("  Required label:   %s\n", config.Label)
	fmt.Printf("  Spawn delay:      %s\n", formatDaemonDuration(config.SpawnDelay))
	if config.ReflectEnabled {
		fmt.Printf("  Reflect interval:  %s\n", formatDaemonDuration(config.ReflectInterval))
		fmt.Printf("  Reflect issues:    %v\n", config.ReflectCreateIssues)
		fmt.Printf("  Reflect open:      %v\n", config.ReflectOpenEnabled)
	} else {
		fmt.Println("  Reflect interval:  disabled")
	}
	if config.ReflectModelDriftEnabled {
		fmt.Printf("  Model drift:       %s\n", formatDaemonDuration(config.ReflectModelDriftInterval))
	} else {
		fmt.Println("  Model drift:       disabled")
	}
	if config.CleanupEnabled {
		fmt.Printf("  Cleanup interval:  %s\n", formatDaemonDuration(config.CleanupInterval))
		fmt.Printf("  Cleanup age:       %d days\n", config.CleanupAgeDays)
		fmt.Printf("  Cleanup preserve:  %v (orchestrator sessions)\n", config.CleanupPreserveOrchestrator)
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
	if config.VerificationPauseThreshold > 0 {
		fmt.Printf("  Verify threshold:  %d (pause after N unverified completions)\n", config.VerificationPauseThreshold)
	} else {
		fmt.Println("  Verify threshold:  disabled")
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

		// Reconcile pool with actual OpenCode sessions FIRST.
		// This prevents stale capacity counts when agents complete without
		// the daemon knowing (overnight runs, crashes, manual kills).
		// Must happen before status write so status shows accurate counts.
		if freed := d.ReconcileWithOpenCode(); freed > 0 && daemonVerbose {
			fmt.Printf("[%s] Reconciled: freed %d stale slots\n", timestamp, freed)
		}

		// Check for verification signal (human ran `orch complete`)
		// This resets the counter and unpauses the daemon.
		if d.VerificationTracker != nil {
			if verified, err := daemon.CheckAndClearVerificationSignal(); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Warning: failed to check verification signal: %v\n", timestamp, err)
			} else if verified {
				d.VerificationTracker.RecordHumanVerification()
				fmt.Printf("[%s] ✅ Human verification detected - verification counter reset\n", timestamp)
			}
		}

		// Check for resume signal (manual resume command)
		// This allows Dylan to resume the daemon without running orch complete.
		if d.VerificationTracker != nil {
			if resumed, err := daemon.CheckAndClearResumeSignal(); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Warning: failed to check resume signal: %v\n", timestamp, err)
			} else if resumed {
				d.VerificationTracker.Resume()
				fmt.Printf("[%s] ✅ Daemon resumed manually - verification counter reset\n", timestamp)
			}
		}

		// Check verification pause BEFORE spawning
		// This enforces the verifiability-first constraint by pausing after N agents
		// are marked ready-for-review without human verification (manual orch complete).
		if d.VerificationTracker != nil {
			verifyStatus := d.VerificationTracker.Status()
			if d.VerificationTracker.IsPaused() {
				fmt.Printf("[%s] Verification pause: %d unverified completions, threshold is %d\n",
					timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
				fmt.Printf("[%s]    Run 'orch daemon resume' after reviewing completed work to continue\n", timestamp)
				time.Sleep(config.PollInterval)
				continue
			}
			if verifyStatus.IsEnabled() {
				fmt.Printf("[%s] Verification check: %d/%d unverified completions, proceeding\n",
					timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
			}
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

		// Run periodic model drift reflection if due
		if result := d.RunPeriodicModelDriftReflection(); result != nil {
			if result.Error != nil {
				fmt.Fprintf(os.Stderr, "[%s] Model drift error: %v\n", timestamp, result.Error)
			} else if result.Message != "" {
				fmt.Printf("[%s] Model drift: %s\n", timestamp, result.Message)
			} else if daemonVerbose {
				fmt.Printf("[%s] Model drift: no updates\n", timestamp)
			}
		}

		// Run periodic session cleanup if due
		if result := d.RunPeriodicCleanup(); result != nil {
			if result.Error != nil {
				fmt.Fprintf(os.Stderr, "[%s] Cleanup error: %v\n", timestamp, result.Error)
				// Log the cleanup error
				event := events.Event{
					Type:      "daemon.cleanup",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"deleted": 0,
						"error":   result.Error.Error(),
						"message": result.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log cleanup error event: %v\n", err)
				}
			} else if result.Deleted > 0 {
				fmt.Printf("[%s] Cleanup: %s\n", timestamp, result.Message)
				// Log the successful cleanup
				event := events.Event{
					Type:      "daemon.cleanup",
					Timestamp: time.Now().Unix(),
					Data: map[string]interface{}{
						"deleted": result.Deleted,
						"message": result.Message,
					},
				}
				if err := logger.Log(event); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log cleanup event: %v\n", err)
				}
			} else if daemonVerbose {
				fmt.Printf("[%s] Cleanup: no stale sessions found\n", timestamp)
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

		// Process completions: mark Phase: Complete agents as ready-for-review
		// This signals they're waiting for orchestrator review. Uses the escalation model:
		// - None/Info/Review: Mark ready-for-review (labeled, not closed)
		// - Block/Failed: Requires human review (no label, remains in_progress)
		completionConfig := daemon.CompletionConfig{
			ProjectDir: projectDir,
			ServerURL:  serverURL,
			DryRun:     false,
			Verbose:    daemonVerbose,
		}
		completionResult, err := d.CompletionOnce(completionConfig)
		if err != nil {
			// Record completion failure for health tracking
			if d.CompletionFailureTracker != nil {
				d.CompletionFailureTracker.RecordFailure(err.Error())
			}

			// Always log completion errors (not just in verbose mode)
			fmt.Fprintf(os.Stderr, "[%s] ⚠️  Completion processing error: %v\n", timestamp, err)

			// Log the error event
			event := events.Event{
				Type:      "daemon.completion_error",
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"error":   err.Error(),
					"message": "Completion processing failed",
				},
			}
			if logErr := logger.Log(event); logErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log completion error event: %v\n", logErr)
			}
		} else {
			// Record successful completion processing
			if d.CompletionFailureTracker != nil {
				d.CompletionFailureTracker.RecordSuccess()
			}
		}

		if completionResult != nil {
			completedThisCycle := 0
			for _, cr := range completionResult.Processed {
				if cr.Processed {
					completedThisCycle++
					completed++
					lastCompletion = time.Now()
					fmt.Printf("[%s] Ready for review: %s (escalation=%s)\n",
						timestamp, cr.BeadsID, cr.Escalation)

					// Record completion for verification tracking
					// This increments the counter and may pause daemon if threshold reached
					if d.VerificationTracker != nil {
						if shouldPause := d.VerificationTracker.RecordCompletion(); shouldPause {
							verifyStatus := d.VerificationTracker.Status()
							fmt.Printf("[%s] ⚠️  Verification threshold reached: %d/%d agents ready for review\n",
								timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
							fmt.Printf("[%s]    Daemon will pause spawning on next cycle\n", timestamp)
							fmt.Printf("[%s]    Run 'orch daemon resume' after reviewing completed work\n", timestamp)
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
							"source":     "daemon_ready_for_review",
						},
					}
					if err := logger.Log(event); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to log completion event: %v\n", err)
					}
				} else if cr.Error != nil && daemonVerbose {
					fmt.Printf("[%s] Review required: %s - %v (escalation=%s)\n",
						timestamp, cr.BeadsID, cr.Error, cr.Escalation)
				}
			}
			if completedThisCycle > 0 && daemonVerbose {
				fmt.Printf("[%s] Marked %d agent(s) ready for review this cycle\n", timestamp, completedThisCycle)
			}
		}

		// Get ready issues count for status (multi-project when registry available)
		readyIssues, _ := daemon.ListReadyIssuesMultiProject(d.ProjectRegistry)
		readyCount := len(readyIssues)

		// Write daemon status file AFTER reconciliation and completions so counts are accurate
		var verificationSnapshot *daemon.VerificationStatusSnapshot
		isPaused := false
		if d.VerificationTracker != nil {
			verifyStatus := d.VerificationTracker.Status()
			isPaused = verifyStatus.IsPaused
			if verifyStatus.IsEnabled() {
				verificationSnapshot = &daemon.VerificationStatusSnapshot{
					IsPaused:                     verifyStatus.IsPaused,
					CompletionsSinceVerification: verifyStatus.CompletionsSinceVerification,
					Threshold:                    verifyStatus.Threshold,
					LastVerification:             verifyStatus.LastVerification,
					RemainingBeforePause:         verifyStatus.RemainingBeforePause(),
				}
			}
		}

		// Capture spawn failure snapshot for health card visibility
		var spawnFailureSnapshot *daemon.SpawnFailureSnapshot
		if d.SpawnFailureTracker != nil {
			snapshot := d.SpawnFailureTracker.Snapshot()
			// Only include if there have been failures
			if snapshot.TotalFailures > 0 {
				spawnFailureSnapshot = &snapshot
			}
		}

		// Capture completion failure snapshot for health card visibility
		var completionFailureSnapshot *daemon.CompletionFailureSnapshot
		if d.CompletionFailureTracker != nil {
			snapshot := d.CompletionFailureTracker.Snapshot()
			// Only include if there have been failures
			if snapshot.TotalFailures > 0 {
				completionFailureSnapshot = &snapshot
			}
		}

		status := daemon.DaemonStatus{
			PID: os.Getpid(),
			Capacity: daemon.CapacityStatus{
				Max:       config.MaxAgents,
				Active:    d.ActiveCount(),
				Available: d.AvailableSlots(),
			},
			LastPoll:           pollTime,
			LastSpawn:          lastSpawn,
			LastCompletion:     lastCompletion,
			ReadyCount:         readyCount,
			Status:             daemon.DetermineStatus(pollTime, config.PollInterval, isPaused),
			Verification:       verificationSnapshot,
			SpawnFailures:      spawnFailureSnapshot,
			CompletionFailures: completionFailureSnapshot,
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

			result, err := d.OnceExcluding(skippedThisCycle)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				break
			}

			if !result.Processed {
				// Check if this is a spawn failure (not queue empty or capacity)
				// If so, skip this issue and try the next one.
				if result.Issue != nil && result.Error != nil {
					skippedThisCycle[result.Issue.ID] = true
					fmt.Fprintf(os.Stderr, "[%s] Skipping %s: %v\n",
						timestamp, result.Issue.ID, result.Error)
					// Continue to try the next issue
					continue
				}

				// No more issues or non-issue-specific error
				if daemonVerbose && spawnedThisCycle == 0 {
					// Use the message from Once() which indicates why processing stopped
					fmt.Printf("[%s] %s\n", timestamp, result.Message)
				}
				break
			}

			processed++
			spawnedThisCycle++
			lastSpawn = time.Now()
			if result.ExtractionSpawned {
				fmt.Printf("[%s] Auto-extraction: %s (blocking %s) - %s\n",
					timestamp,
					result.Issue.ID,
					result.OriginalIssueID,
					result.Issue.Title,
				)
			} else if result.ArchitectEscalated {
				fmt.Printf("[%s] Architect escalation: %s (%s, %s) - %s\n",
					timestamp,
					result.Issue.ID,
					result.Skill,
					result.Model,
					result.Issue.Title,
				)
			} else {
				fmt.Printf("[%s] Spawned: %s (%s, %s) - %s\n",
					timestamp,
					result.Issue.ID,
					result.Skill,
					result.Model,
					result.Issue.Title,
				)
			}

			// Log the spawn
			eventData := map[string]interface{}{
				"beads_id": result.Issue.ID,
				"skill":    result.Skill,
				"model":    result.Model,
				"title":    result.Issue.Title,
				"count":    processed,
			}
			if result.ExtractionSpawned {
				eventData["extraction"] = true
				eventData["original_issue"] = result.OriginalIssueID
			}
			if result.ArchitectEscalated {
				eventData["architect_escalated"] = true
			}
			event := events.Event{
				Type:      "daemon.spawn",
				Timestamp: time.Now().Unix(),
				Data:      eventData,
			}
			if err := logger.Log(event); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
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
	config := daemonConfigFromFlags()
	d := daemon.NewWithConfig(config)

	// NOTE: Extraction system disabled. Hotspot checking not configured.
	// To re-enable, uncomment: d.HotspotChecker = daemon.NewGitHotspotChecker()

	// Seed verification tracker with unverified backlog
	seedVerificationTracker(d)

	result, err := d.Preview()
	if err != nil {
		return fmt.Errorf("preview error: %w", err)
	}

	// Show verification status in dry-run output
	if d.VerificationTracker != nil {
		verifyStatus := d.VerificationTracker.Status()
		if d.VerificationTracker.IsPaused() {
			fmt.Printf("[DRY-RUN] Verification pause: %d unverified completions, threshold is %d\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
		} else if verifyStatus.IsEnabled() {
			fmt.Printf("[DRY-RUN] Verification check: %d/%d unverified completions\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
		}
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
		fmt.Printf("Inferred model: %s\n", result.Model)
		if result.ArchitectEscalated {
			fmt.Println("⚠️  Architect escalation: implementation skill escalated to architect (hotspot area)")
		}

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
	config := daemonConfigFromFlags()
	d := daemon.NewWithConfig(config)

	// Initialize project registry for cross-project issue resolution
	registry, err := daemon.NewProjectRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: project registry unavailable: %v\n", err)
	} else {
		d.ProjectRegistry = registry
	}

	// Seed verification tracker with unverified backlog
	seedVerificationTracker(d)

	// Show verification status before spawning
	if d.VerificationTracker != nil {
		verifyStatus := d.VerificationTracker.Status()
		if d.VerificationTracker.IsPaused() {
			fmt.Printf("Verification pause: %d unverified completions, threshold is %d\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
			fmt.Println("  Run 'orch daemon resume' after reviewing completed work to continue")
		} else if verifyStatus.IsEnabled() {
			fmt.Printf("Verification check: %d/%d unverified completions, proceeding\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
		}
	}

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
	fmt.Printf("  Model:  %s\n", result.Model)

	// Log the spawn
	logger := events.NewLogger(events.DefaultLogPath())
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
	config := daemonConfigFromFlags()
	d := daemon.NewWithConfig(config)

	// NOTE: Extraction system disabled. Hotspot checking not configured.
	// To re-enable, uncomment: d.HotspotChecker = daemon.NewGitHotspotChecker()

	// Seed verification tracker with unverified backlog
	seedVerificationTracker(d)

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
		fmt.Printf("Inferred model: %s\n", result.Model)

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

func runDaemonResume() error {
	fmt.Println("Sending resume signal to daemon...")

	if err := daemon.WriteResumeSignal(); err != nil {
		return fmt.Errorf("failed to write resume signal: %w", err)
	}

	fmt.Println("✅ Resume signal sent")
	fmt.Println()
	fmt.Println("The daemon will detect the signal on the next poll cycle and resume operation.")
	fmt.Println("The verification counter will be reset, allowing the daemon to continue spawning.")

	return nil
}
