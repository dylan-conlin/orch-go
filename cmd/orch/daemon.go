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

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/notify"
	"github.com/dylan-conlin/orch-go/pkg/verify"
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
  status   Show daemon running state with PID liveness check
  stop     Stop the running daemon
  restart  Stop and restart the daemon
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
  orch-go daemon run --replace              # Stop existing daemon first, then start
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

var daemonCleanStaleCmd = &cobra.Command{
	Use:   "clean-stale",
	Short: "Close orphaned cross-project completions that block verification pause",
	Long: `Find and close beads issues from other projects that are stuck in
Phase: Complete with daemon:ready-review label but will never be resolved
through normal orch complete flow.

This happens when projects are merged/archived but their beads databases
still contain open issues. The daemon's cross-project completion scanner
detects them every cycle, inflating the verification pause counter.

Examples:
  orch daemon clean-stale           # Show stale completions (dry run)
  orch daemon clean-stale --close   # Close stale issues`,
	RunE: func(cmd *cobra.Command, args []string) error {
		closeStale, _ := cmd.Flags().GetBool("close")
		return runDaemonCleanStale(closeStale)
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon running state with PID liveness check",
	Long: `Show the current daemon status including PID liveness validation.

Reads the daemon status file (~/.orch/daemon-status.json) and validates
that the daemon process is actually alive. Detects stale status files
from crashed daemons that would otherwise report false "running" state.

Examples:
  orch daemon status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonStatus()
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	Long: `Stop the running daemon by sending SIGTERM and waiting for graceful shutdown.

The daemon will finish any in-progress spawn cycle before exiting.
If the daemon doesn't stop within 10 seconds, an error is returned.

Examples:
  orch daemon stop`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonStop()
	},
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Stop and restart the daemon",
	Long: `Stop the running daemon and start a new one with the current flags.

Equivalent to running 'orch daemon stop' followed by 'orch daemon run'.
All flags from 'daemon run' are available (--concurrency, --poll-interval, etc.).

If no daemon is currently running, starts a new one directly.

Examples:
  orch daemon restart                        # Restart with default flags
  orch daemon restart --concurrency 5        # Restart with new concurrency
  orch daemon restart --poll-interval 30     # Restart with new poll interval`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonRestart()
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
	daemonKnowledgeHealthInterval int  // Knowledge health check interval in minutes (0 = disabled)
	daemonCleanupEnabled         bool // Enable periodic session cleanup
	daemonCleanupInterval        int  // Session cleanup interval in minutes (0 = disabled)
	daemonCleanupAge             int  // Session age threshold in days for cleanup
	daemonCleanupPreserveOrch    bool // Preserve orchestrator sessions during cleanup
	daemonOrphanDetectionInterval int // Orphan detection interval in minutes (0 = disabled)
	daemonOrphanAgeThreshold      int // Orphan age threshold in minutes
	daemonPhaseTimeoutInterval    int // Phase timeout check interval in minutes (0 = disabled)
	daemonPhaseTimeoutThreshold   int // Phase timeout threshold in minutes
	daemonAgreementCheckInterval  int // Agreement check interval in minutes (0 = disabled)
	daemonReplace                 bool // Stop existing daemon before starting (graceful takeover)
)

func init() {
	daemonCmd.AddCommand(daemonRunCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
	daemonCmd.AddCommand(daemonOnceCmd)
	daemonCmd.AddCommand(daemonPreviewCmd)
	daemonCmd.AddCommand(daemonReflectCmd)
	daemonCmd.AddCommand(daemonResumeCmd)
	daemonCmd.AddCommand(daemonCleanStaleCmd)

	daemonCleanStaleCmd.Flags().Bool("close", false, "Actually close stale issues (default: dry run)")

	// --replace is only on daemon run (daemon restart already has this behavior)
	daemonRunCmd.Flags().BoolVar(&daemonReplace, "replace", false, "Stop existing daemon before starting (graceful takeover)")

	// Register daemon run flags on both run and restart commands
	for _, cmd := range []*cobra.Command{daemonRunCmd, daemonRestartCmd} {
		cmd.Flags().IntVar(&daemonDelay, "delay", 3, "Delay between spawns in seconds")
		cmd.Flags().BoolVar(&daemonDryRun, "dry-run", false, "Preview mode - show what would be processed without spawning")
		cmd.Flags().IntVar(&daemonPollInterval, "poll-interval", 15, "Poll interval in seconds (0 = run once and exit)")
		cmd.Flags().IntVarP(&daemonMaxAgents, "concurrency", "c", 3, "Maximum concurrent agents (0 = no limit)")
		cmd.Flags().IntVar(&daemonMaxAgents, "max-agents", 3, "Maximum concurrent agents (alias for --concurrency)")
		cmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
		cmd.Flags().BoolVarP(&daemonVerbose, "verbose", "v", false, "Enable verbose output")
		cmd.Flags().BoolVar(&daemonReflect, "reflect", true, "Run kb reflect analysis on exit (default: true)")
		cmd.Flags().IntVar(&daemonReflectInterval, "reflect-interval", 60, "Periodic reflection interval in minutes (0 = disabled, default: 60)")
		cmd.Flags().BoolVar(&daemonReflectIssues, "reflect-issues", true, "Create beads issues for synthesis opportunities (default: true)")
		cmd.Flags().BoolVar(&daemonReflectOpen, "reflect-open", true, "Create beads issues for open investigation actions (default: true)")
		cmd.Flags().IntVar(&daemonModelDriftInterval, "reflect-model-drift-interval", 240, "Model drift reflection interval in minutes (0 = disabled, default: 240 = 4 hours)")
		cmd.Flags().IntVar(&daemonKnowledgeHealthInterval, "knowledge-health-interval", 120, "Knowledge health check interval in minutes (0 = disabled, default: 120 = 2 hours)")
		cmd.Flags().BoolVar(&daemonCleanupEnabled, "cleanup-enabled", true, "Enable periodic session cleanup (default: true)")
		cmd.Flags().IntVar(&daemonCleanupInterval, "cleanup-interval", 360, "Session cleanup interval in minutes (0 = disabled, default: 360 = 6 hours)")
		cmd.Flags().IntVar(&daemonCleanupAge, "cleanup-age", 7, "Session age threshold in days for cleanup (default: 7)")
		cmd.Flags().BoolVar(&daemonCleanupPreserveOrch, "cleanup-preserve-orchestrator", true, "Preserve orchestrator sessions during cleanup (default: true)")
		cmd.Flags().IntVar(&daemonOrphanDetectionInterval, "orphan-detection-interval", 30, "Orphan detection interval in minutes (0 = disabled, default: 30)")
		cmd.Flags().IntVar(&daemonOrphanAgeThreshold, "orphan-age-threshold", 60, "How long (minutes) before issue is considered orphaned (default: 60)")
		cmd.Flags().IntVar(&daemonPhaseTimeoutInterval, "phase-timeout-interval", 5, "Phase timeout check interval in minutes (0 = disabled, default: 5)")
		cmd.Flags().IntVar(&daemonPhaseTimeoutThreshold, "phase-timeout-threshold", 30, "Minutes without phase update before flagging as unresponsive (default: 30)")
		cmd.Flags().IntVar(&daemonAgreementCheckInterval, "agreement-check-interval", 30, "Agreement check interval in minutes (0 = disabled, default: 30)")
		cmd.Flags().MarkHidden("max-agents")
	}

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
	config.KnowledgeHealthEnabled = daemonKnowledgeHealthInterval > 0
	config.KnowledgeHealthInterval = time.Duration(daemonKnowledgeHealthInterval) * time.Minute
	config.CleanupEnabled = daemonCleanupEnabled && daemonCleanupInterval > 0
	config.CleanupInterval = time.Duration(daemonCleanupInterval) * time.Minute
	config.CleanupAgeDays = daemonCleanupAge
	config.CleanupPreserveOrchestrator = daemonCleanupPreserveOrch
	config.CleanupServerURL = serverURL
	config.OrphanDetectionEnabled = daemonOrphanDetectionInterval > 0
	config.OrphanDetectionInterval = time.Duration(daemonOrphanDetectionInterval) * time.Minute
	config.OrphanAgeThreshold = time.Duration(daemonOrphanAgeThreshold) * time.Minute
	config.PhaseTimeoutEnabled = daemonPhaseTimeoutInterval > 0
	config.PhaseTimeoutInterval = time.Duration(daemonPhaseTimeoutInterval) * time.Minute
	config.PhaseTimeoutThreshold = time.Duration(daemonPhaseTimeoutThreshold) * time.Minute
	config.AgreementCheckEnabled = daemonAgreementCheckInterval > 0
	config.AgreementCheckInterval = time.Duration(daemonAgreementCheckInterval) * time.Minute

	return config
}

// verificationBreakdown returns a per-project breakdown string for verification messages.
// Best-effort: returns empty string on error so the primary count always displays.
func verificationBreakdown() string {
	items, err := verify.ListUnverifiedWork()
	if err != nil || len(items) == 0 {
		return ""
	}
	return verify.FormatProjectBreakdown(items)
}

// seedVerificationTracker seeds the tracker with the backlog IDs.
// Called after daemon construction, before entering the main loop.
func seedVerificationTracker(d *daemon.Daemon) {
	if d.VerificationTracker == nil {
		return
	}

	items, err := verify.ListUnverifiedWork()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not seed verification tracker: %v\n", err)
		return
	}

	if len(items) > 0 {
		ids := make([]string, len(items))
		for i, item := range items {
			ids[i] = item.BeadsID
		}
		d.VerificationTracker.SeedFromBacklog(ids)
		breakdown := verify.FormatProjectBreakdown(items)
		fmt.Printf("  Verification backlog: %d unverified completions from previous sessions%s\n", len(items), breakdown)

		if d.VerificationTracker.IsPaused() {
			status := d.VerificationTracker.Status()
			fmt.Printf("  Warning: Verification pause: backlog exceeds threshold (%d/%d)%s\n",
				len(items), status.Threshold, breakdown)
			fmt.Println("  Run 'orch daemon resume' after reviewing completed work")
		}
	}
}

func runDaemonLoop() error {
	// Handle dry-run mode
	if daemonDryRun {
		return runDaemonDryRun()
	}

	// If --replace, stop existing daemon before acquiring lock
	if daemonReplace {
		pid := daemon.ReadPIDFromLockFile()
		if pid > 0 && daemon.IsProcessAlive(pid) {
			fmt.Printf("Replacing existing daemon (PID %d)...\n", pid)
			if err := daemon.StopDaemon(daemon.StopOptions{}); err != nil && err != daemon.ErrNoDaemonRunning {
				return fmt.Errorf("failed to stop existing daemon: %w", err)
			}
			fmt.Println("Previous daemon stopped.")
		}
	}

	// Acquire PID lock to ensure single daemon instance.
	// This prevents multiple daemon processes from accumulating silently
	// and fighting over the status file and spawns.
	pidLock, err := daemon.AcquirePIDLock()
	if err != nil {
		return fmt.Errorf("cannot start daemon: %w", err)
	}
	defer pidLock.Release()

	// Auto-lock control plane at daemon launch to ensure agents can't
	// modify settings.json or enforcement hooks during autonomous operation.
	if n, err := control.EnsureLocked(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to lock control plane: %v\n", err)
	} else if n > 0 {
		fmt.Fprintf(os.Stderr, "Control plane: locked %d unlocked files\n", n)
	}

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

	// Wire auto-completer for auto-tier agents.
	// When review tier is "auto", daemon runs `orch complete` directly
	// instead of labeling for orchestrator review.
	d.AutoCompleter = &daemon.OrcCompleter{}

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
	var lastSpawn time.Time              // Track last successful spawn
	var lastCompletion time.Time         // Track last auto-completion
	var lastStuckNotification time.Time  // Cooldown for stuck notifications
	stuckNotifier := notify.Default()

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
	if config.KnowledgeHealthEnabled {
		fmt.Printf("  Knowledge health:  %s (threshold: %d entries)\n", formatDaemonDuration(config.KnowledgeHealthInterval), config.KnowledgeHealthThreshold)
	} else {
		fmt.Println("  Knowledge health:  disabled")
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
	if config.OrphanDetectionEnabled {
		fmt.Printf("  Orphan detection:  %s (age threshold: %s)\n", formatDaemonDuration(config.OrphanDetectionInterval), formatDaemonDuration(config.OrphanAgeThreshold))
	} else {
		fmt.Println("  Orphan detection:  disabled")
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

		// Reconcile pool with actual running agents FIRST.
		// This handles two cases:
		// 1. Agents completed without daemon knowing → free stale slots
		// 2. After daemon restart, agents from prior run still active → add synthetic
		//    slots to prevent over-spawning past the concurrency cap
		// Must happen before status write so status shows accurate counts.
		reconcileResult := d.ReconcileWithOpenCode()
		if reconcileResult.Freed > 0 && daemonVerbose {
			fmt.Printf("[%s] Reconciled: freed %d stale slots\n", timestamp, reconcileResult.Freed)
		}
		if reconcileResult.Added > 0 {
			fmt.Printf("[%s] Reconciled: seeded %d agents from prior run (pool was under-counting)\n", timestamp, reconcileResult.Added)
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
				breakdown := verificationBreakdown()
				fmt.Printf("[%s] Verification pause: %d unverified completions, threshold is %d%s\n",
					timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
				fmt.Printf("[%s]    Run 'orch daemon resume' after reviewing completed work to continue\n", timestamp)

				// Write status file during pause so last_poll stays fresh
				// and status correctly shows "paused" instead of going stale.
				pauseStatus := daemon.DaemonStatus{
					PID: os.Getpid(),
					Capacity: daemon.CapacityStatus{
						Max:       config.MaxAgents,
						Active:    d.ActiveCount(),
						Available: d.AvailableSlots(),
					},
					LastPoll:       time.Now(),
					LastSpawn:      lastSpawn,
					LastCompletion: lastCompletion,
					Status:         "paused",
					Verification: &daemon.VerificationStatusSnapshot{
						IsPaused:                     true,
						CompletionsSinceVerification: verifyStatus.CompletionsSinceVerification,
						Threshold:                    verifyStatus.Threshold,
						LastVerification:             verifyStatus.LastVerification,
						RemainingBeforePause:         verifyStatus.RemainingBeforePause(),
					},
				}
				if err := daemon.WriteStatusFile(pauseStatus); err != nil && daemonVerbose {
					fmt.Fprintf(os.Stderr, "Warning: failed to write status file: %v\n", err)
				}

				time.Sleep(config.PollInterval)
				continue
			}
			if verifyStatus.IsEnabled() {
				fmt.Printf("[%s] Verification check: %d/%d unverified completions, proceeding\n",
					timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
			}
		}

		// Run all periodic maintenance tasks (reflection, cleanup, recovery, etc.)
		periodicResult := runPeriodicTasks(d, timestamp, daemonVerbose, logger)
		knowledgeHealthSnapshot := periodicResult.KnowledgeHealthSnapshot
		phaseTimeoutSnapshot := periodicResult.PhaseTimeoutSnapshot
		questionDetectionSnapshot := periodicResult.QuestionDetectionSnapshot
		agreementCheckSnapshot := periodicResult.AgreementCheckSnapshot

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
					if cr.AutoCompleted {
						fmt.Printf("[%s] Auto-completed: %s (tier=auto)\n",
							timestamp, cr.BeadsID)
					} else {
						fmt.Printf("[%s] Ready for review: %s (escalation=%s)\n",
							timestamp, cr.BeadsID, cr.Escalation)
					}

					// NOTE: RecordCompletion() is called inside ProcessCompletion()
					// (completion_processing.go). Do NOT call it again here — that
					// caused a double-counting bug where each completion incremented
					// the counter by 2, making the daemon pause at half the expected
					// number of completions.

					// Check if verification tracker was paused by ProcessCompletion
					if d.VerificationTracker != nil && d.VerificationTracker.IsPaused() {
						verifyStatus := d.VerificationTracker.Status()
						breakdown := verificationBreakdown()
						fmt.Printf("[%s] ⚠️  Verification threshold reached: %d/%d agents ready for review%s\n",
							timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
						fmt.Printf("[%s]    Daemon will pause spawning on next cycle\n", timestamp)
						fmt.Printf("[%s]    Run 'orch daemon resume' after reviewing completed work\n", timestamp)
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

		// Check beads circuit breaker — if beads is unhealthy, skip polling and back off.
		// This prevents the lock cascade where daemon keeps spawning bd processes
		// that pile up behind a stuck JSONL lock.
		if d.BeadsCircuitBreaker != nil && d.BeadsCircuitBreaker.IsOpen() {
			backoff := d.BeadsCircuitBreaker.BackoffDuration()
			failures := d.BeadsCircuitBreaker.ConsecutiveFailures()
			fmt.Printf("[%s] ⚠️  Beads circuit breaker open: %d consecutive failures, backing off %s\n",
				timestamp, failures, formatDaemonDuration(backoff))
			select {
			case <-ctx.Done():
				fmt.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
				return nil
			case <-time.After(backoff):
				continue
			}
		}

		// Get ready issues count for status (multi-project when registry available)
		readyIssues, readyErr := daemon.ListReadyIssuesMultiProject(d.ProjectRegistry)
		if readyErr != nil {
			if d.BeadsCircuitBreaker != nil {
				d.BeadsCircuitBreaker.RecordFailure()
			}
			fmt.Fprintf(os.Stderr, "[%s] ⚠️  Failed to list ready issues: %v\n", timestamp, readyErr)
		} else {
			if d.BeadsCircuitBreaker != nil {
				d.BeadsCircuitBreaker.RecordSuccess()
			}
		}
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

		// Capture completion failure snapshot for health card visibility
		var completionFailureSnapshot *daemon.CompletionFailureSnapshot
		if d.CompletionFailureTracker != nil {
			snapshot := d.CompletionFailureTracker.Snapshot()
			// Only include if there have been failures
			if snapshot.TotalFailures > 0 {
				completionFailureSnapshot = &snapshot
			}
		}

		// Refresh pollTime to reflect actual status write time.
		// Processing (reconciliation, periodic tasks, completions, ready count)
		// between cycle start and here can take significant time, causing
		// DetermineStatus to see a stale pollTime and return "stalled" incorrectly.
		pollTime = time.Now()

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
			CompletionFailures: completionFailureSnapshot,
			KnowledgeHealth:    knowledgeHealthSnapshot,
			PhaseTimeout:       phaseTimeoutSnapshot,
			QuestionDetection:  questionDetectionSnapshot,
			AgreementCheck:     agreementCheckSnapshot,
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

			// Stuck detection: all slots full with no activity for 10+ min
			stuckThreshold := 10 * time.Minute
			stuckCooldown := 30 * time.Minute
			if checkDaemonStuck(lastSpawn, lastCompletion, lastStuckNotification, stuckThreshold, stuckCooldown) {
				fmt.Printf("[%s] ⚠ Daemon stuck: capacity full, no spawns or completions in %s\n",
					timestamp, stuckThreshold)
				if err := stuckNotifier.DaemonStuck(activeCount, daemonMaxAgents); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: stuck notification failed: %v\n", err)
				}
				lastStuckNotification = time.Now()
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
				// If result identifies a specific issue, skip it and try the next one.
				// This handles both error cases (spawn failure, status update failure)
				// and non-error skip cases (existing session, title dedup, status mismatch).
				// Without this, a single high-priority dedup'd issue would break the
				// inner loop and block all lower-priority issues from being tried.
				if result.Issue != nil {
					skippedThisCycle[result.Issue.ID] = true
					if result.Error != nil {
						fmt.Fprintf(os.Stderr, "[%s] Skipping %s: %v\n",
							timestamp, result.Issue.ID, result.Error)
					} else if daemonVerbose {
						fmt.Printf("[%s] Skipping %s: %s\n",
							timestamp, result.Issue.ID, result.Message)
					}
					// Continue to try the next issue
					continue
				}

				// No more issues or non-issue-specific condition (rate limit, paused, etc.)
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

// checkDaemonStuck returns true if the daemon appears stuck:
// all slots full with no spawns or completions for stuckThreshold,
// and last notification was more than cooldown ago.
func checkDaemonStuck(lastSpawn, lastCompletion, lastNotification time.Time, stuckThreshold, cooldown time.Duration) bool {
	if lastSpawn.IsZero() || lastCompletion.IsZero() {
		return false
	}
	return time.Since(lastSpawn) > stuckThreshold &&
		time.Since(lastCompletion) > stuckThreshold &&
		time.Since(lastNotification) > cooldown
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

	// Initialize project registry for cross-project issue visibility
	if registry, err := daemon.NewProjectRegistry(); err == nil {
		d.ProjectRegistry = registry
	}

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
			breakdown := verificationBreakdown()
			fmt.Printf("[DRY-RUN] Verification pause: %d unverified completions, threshold is %d%s\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
		} else if verifyStatus.IsEnabled() {
			fmt.Printf("[DRY-RUN] Verification check: %d/%d unverified completions\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
		}
	}

	// Get current directory for context
	projectDir, _ := os.Getwd()
	projectName := filepath.Base(projectDir)

	// Show queue summary: spawnable vs rejected counts
	spawnableCount := 0
	if result.Issue != nil {
		spawnableCount = 1
	}
	rejectedCount := len(result.RejectedIssues)
	fmt.Printf("[DRY-RUN] Queue: %d spawnable, %d rejected\n\n", spawnableCount, rejectedCount)

	if result.Issue != nil {
		fmt.Println("Next spawn:")
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

	// Display rejected issues grouped by reason
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
			breakdown := verificationBreakdown()
			fmt.Printf("Verification pause: %d unverified completions, threshold is %d%s\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
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

	// Initialize project registry for cross-project issue visibility
	if registry, err := daemon.NewProjectRegistry(); err == nil {
		d.ProjectRegistry = registry
	}

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

	// Show queue summary: spawnable vs rejected counts
	spawnableCount := 0
	if result.Issue != nil {
		spawnableCount = 1
	}
	rejectedCount := len(result.RejectedIssues)
	fmt.Printf("Queue: %d spawnable, %d rejected\n\n", spawnableCount, rejectedCount)

	// Display spawnable issue if available
	if result.Issue != nil {
		fmt.Println("Next spawn:")
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

	// Display rejected issues grouped by reason
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

func runDaemonStatus() error {
	info := daemon.GetStatusInfo()

	// Clean up stale status file if detected
	if info.StaleFile {
		daemon.RemoveStatusFile()
	}

	fmt.Print(daemon.FormatStatusInfo(info))
	return nil
}

func runDaemonStop() error {
	pid := daemon.ReadPIDFromLockFile()
	if pid > 0 {
		fmt.Printf("Stopping daemon (PID %d)...\n", pid)
	} else {
		fmt.Println("Stopping daemon...")
	}

	err := daemon.StopDaemon(daemon.StopOptions{})
	if err == daemon.ErrNoDaemonRunning {
		fmt.Println("No daemon is currently running.")
		return nil
	}
	if err == daemon.ErrStopTimeout {
		return fmt.Errorf("daemon (PID %d) did not stop within timeout - it may need to be killed manually", pid)
	}
	if err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	fmt.Println("Daemon stopped.")
	return nil
}

func runDaemonRestart() error {
	// Try to stop existing daemon first (ignore "not running" error)
	pid := daemon.ReadPIDFromLockFile()
	if pid > 0 && daemon.IsProcessAlive(pid) {
		fmt.Printf("Stopping existing daemon (PID %d)...\n", pid)
		err := daemon.StopDaemon(daemon.StopOptions{})
		if err != nil && err != daemon.ErrNoDaemonRunning {
			return fmt.Errorf("failed to stop existing daemon: %w", err)
		}
		fmt.Println("Daemon stopped.")
	}

	fmt.Println("Starting new daemon...")
	return runDaemonLoop()
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

// runDaemonCleanStale finds and optionally closes orphaned cross-project completions.
// These are issues from other projects that have Phase: Complete + daemon:ready-review
// but will never be resolved via orch complete (project merged/archived).
func runDaemonCleanStale(closeStale bool) error {
	registry, err := daemon.NewProjectRegistry()
	if err != nil {
		return fmt.Errorf("failed to load project registry: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	type staleIssue struct {
		ID         string
		Title      string
		ProjectDir string
		Project    string
	}

	var stale []staleIssue

	for _, proj := range registry.Projects() {
		issues, err := daemon.ListIssuesWithLabelForProject("daemon:ready-review", proj.Dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan %s: %v\n", proj.Dir, err)
			continue
		}

		for _, issue := range issues {
			// Only flag cross-project issues (not the current project)
			if proj.Dir == cwd {
				continue
			}
			stale = append(stale, staleIssue{
				ID:         issue.ID,
				Title:      issue.Title,
				ProjectDir: proj.Dir,
				Project:    proj.Prefix,
			})
		}
	}

	if len(stale) == 0 {
		fmt.Println("No stale cross-project completions found.")
		return nil
	}

	fmt.Printf("Found %d stale cross-project completion(s):\n\n", len(stale))
	for _, s := range stale {
		fmt.Printf("  %s: %s\n    Project: %s (%s)\n", s.ID, s.Title, s.Project, s.ProjectDir)
	}
	fmt.Println()

	if !closeStale {
		fmt.Println("Run with --close to close these issues.")
		return nil
	}

	closed := 0
	for _, s := range stale {
		if err := daemon.CloseIssueForProject(s.ID, s.ProjectDir, "Closed by orch daemon clean-stale: orphaned cross-project completion"); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to close %s: %v\n", s.ID, err)
			continue
		}
		fmt.Printf("  Closed: %s\n", s.ID)
		closed++
	}

	fmt.Printf("\nClosed %d/%d stale completions.\n", closed, len(stale))

	// Also send resume signal to unblock daemon
	if closed > 0 {
		if err := daemon.WriteResumeSignal(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to send resume signal: %v\n", err)
		} else {
			fmt.Println("Resume signal sent to daemon.")
		}
	}

	return nil
}
