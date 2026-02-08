// Package main provides the CLI entry point for orch-go.
package main

import (
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
	daemonMaxDeadSessionRetries        int    // Max dead session retries before escalation
	daemonOrphanReapEnabled            bool   // Enable periodic orphan process reaping
	daemonOrphanReapInterval           int    // Orphan reap interval in minutes (0 = disabled)
	daemonSortMode                     string // Sort strategy for issue prioritization
	daemonDashboardWatchdog            bool   // Enable dashboard health watchdog
	daemonDashboardWatchdogInterval    int    // Dashboard watchdog check interval in seconds
	daemonAllowFeatureWork             bool   // Override investigation circuit breaker and allow feature issues
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
	daemonRunCmd.Flags().IntVar(&daemonMaxDeadSessionRetries, "max-dead-session-retries", 2, "Max times a dead session is retried before escalating to needs:human (default: 2)")

	// Orphan process reaping
	daemonRunCmd.Flags().BoolVar(&daemonOrphanReapEnabled, "orphan-reap", true, "Enable periodic orphan process reaping (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonOrphanReapInterval, "orphan-reap-interval", 5, "Orphan reap interval in minutes (0 = disabled, default: 5)")

	// Sort mode for issue prioritization
	daemonRunCmd.Flags().StringVar(&daemonSortMode, "sort-mode", "priority", "Sort strategy for issue prioritization (priority, unblock)")
	daemonRunCmd.Flags().BoolVar(&daemonAllowFeatureWork, "allow-feature-work", false, "Override investigation circuit breaker and include feature issues in ready queue")

	// Dashboard health watchdog
	daemonRunCmd.Flags().BoolVar(&daemonDashboardWatchdog, "dashboard-watchdog", true, "Enable dashboard health monitoring and auto-restart (default: true)")
	daemonRunCmd.Flags().IntVar(&daemonDashboardWatchdogInterval, "dashboard-watchdog-interval", 30, "Dashboard health check interval in seconds (default: 30)")

	// Add label filter to preview and once commands (share the same variable)
	daemonPreviewCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
	daemonPreviewCmd.Flags().BoolVar(&daemonCrossProject, "cross-project", false, "Preview issues from all kb-registered projects")
	daemonPreviewCmd.Flags().StringVar(&daemonSortMode, "sort-mode", "priority", "Sort strategy for issue prioritization (priority, unblock)")
	daemonPreviewCmd.Flags().BoolVar(&daemonAllowFeatureWork, "allow-feature-work", false, "Override investigation circuit breaker and include feature issues in ready queue")
	daemonOnceCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
	daemonOnceCmd.Flags().BoolVar(&daemonCrossProject, "cross-project", false, "Process one issue from all kb-registered projects")
	daemonOnceCmd.Flags().StringVar(&daemonSortMode, "sort-mode", "priority", "Sort strategy for issue prioritization (priority, unblock)")
	daemonOnceCmd.Flags().BoolVar(&daemonAllowFeatureWork, "allow-feature-work", false, "Override investigation circuit breaker and include feature issues in ready queue")
}

func runDaemonLoop() error {
	// Handle dry-run mode
	if daemonDryRun {
		return runDaemonDryRun()
	}

	// Phase 1: Build configuration from CLI flags
	config, err := buildDaemonConfig()
	if err != nil {
		return err
	}

	// Phase 2: Initialize runtime (daemon, cache, logger, signal handler)
	rt, err := initDaemonRuntime(config)
	if err != nil {
		return err
	}
	defer rt.cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, stopping daemon...")
		rt.cancel()
	}()

	// Ensure reflection runs on exit if enabled
	if daemonReflect {
		defer runReflectionAnalysis(daemonVerbose)
	}

	// Clean up status file on shutdown
	defer daemon.RemoveStatusFile()

	// Phase 3: Print startup banner
	printDaemonBanner(config)

	// Phase 4: Main polling loop
	for {
		select {
		case <-rt.ctx.Done():
			fmt.Printf("\n%s\n", rt.stopMessage())
			return nil
		default:
		}

		rt.cycles++
		timestamp := time.Now().Format("15:04:05")
		pollTime := time.Now()

		// Check server health and update recovery state FIRST.
		serverAvailable := rt.d.CheckServerHealth()
		if daemonVerbose {
			fmt.Printf("[%s] Server health: available=%v\n", timestamp, serverAvailable)
		}

		// Reconcile pool with actual OpenCode sessions.
		if freed := rt.d.ReconcileWithOpenCode(); freed > 0 && daemonVerbose {
			fmt.Printf("[%s] Reconciled: freed %d stale slots\n", timestamp, freed)
		}

		// Run periodic subsystems (reflection, cleanup, recovery, dead sessions)
		rt.runSubsystems(timestamp)

		// Process completions: auto-close agents that report Phase: Complete
		rt.processCompletions(timestamp)

		// Process factual questions if enabled
		rt.processFactualQuestions(timestamp)

		// Write daemon status file AFTER reconciliation and completions
		rt.writeStatus(timestamp, pollTime)

		// Check capacity before polling
		if rt.d.AtCapacity() {
			activeCount := rt.d.ActiveCount()
			if daemonVerbose {
				fmt.Printf("[%s] At capacity (%d/%d agents active), waiting...\n",
					timestamp, activeCount, daemonMaxAgents)
			}
			select {
			case <-rt.ctx.Done():
				fmt.Printf("\n%s\n", rt.stopMessage())
				return nil
			case <-time.After(config.PollInterval):
				continue
			}
		}

		// Refresh frontier cache for sort strategies that need leverage data.
		// Runs once per poll cycle (~60s), which is acceptable staleness for batch daemon.
		if rt.config.SortMode == "unblock" {
			rt.d.RefreshFrontierCache()
			if daemonVerbose {
				if rt.d.CachedFrontier != nil {
					fmt.Printf("[%s] Frontier cache refreshed: %d ready, %d blocked\n",
						timestamp, len(rt.d.CachedFrontier.Ready), len(rt.d.CachedFrontier.Blocked))
				} else {
					fmt.Printf("[%s] Frontier cache: unavailable (sort will use priority fallback)\n", timestamp)
				}
			}
		}

		if daemonVerbose {
			fmt.Printf("[%s] Polling for issues...\n", timestamp)
		}

		// Process issues until queue is empty or at capacity
		spawnedThisCycle := rt.processSpawns(timestamp)

		// If poll interval is 0, run once and exit
		if config.PollInterval == 0 {
			fmt.Printf("Run-once mode. Spawned %d, completed %d.\n", rt.processed, rt.completed)
			return nil
		}

		// Wait for next poll cycle
		if daemonVerbose {
			fmt.Printf("[%s] Spawned %d this cycle, waiting %s before next poll...\n",
				timestamp, spawnedThisCycle, formatDaemonDuration(config.PollInterval))
		}
		select {
		case <-rt.ctx.Done():
			fmt.Printf("\n%s\n", rt.stopMessage())
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
		Label:                    daemonLabel,
		CrossProject:             daemonCrossProject,
		SortMode:                 daemonSortMode,
		AllowFeatureWorkOverride: daemonAllowFeatureWork,
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
	projectDir, _ := currentProjectDir()
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
		Label:                    daemonLabel,
		CrossProject:             daemonCrossProject,
		SortMode:                 daemonSortMode,
		AllowFeatureWorkOverride: daemonAllowFeatureWork,
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
		logDaemonEvent(logger, "daemon.once", map[string]interface{}{
			"beads_id": cpResult.Issue.ID,
			"skill":    cpResult.Skill,
			"title":    cpResult.Issue.Title,
			"project":  cpResult.ProjectName,
		})
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
	logDaemonEvent(logger, "daemon.once", map[string]interface{}{
		"beads_id": result.Issue.ID,
		"skill":    result.Skill,
		"title":    result.Issue.Title,
	})

	return nil
}

func runDaemonPreview() error {
	config := daemon.Config{
		Label:                    daemonLabel,
		CrossProject:             daemonCrossProject,
		SortMode:                 daemonSortMode,
		AllowFeatureWorkOverride: daemonAllowFeatureWork,
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
	projectDir, _ := currentProjectDir()
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
