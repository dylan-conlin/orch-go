package main

import (
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
  orch daemon run                    # Use defaults (3 concurrent agents, 15s poll)
  orch daemon run --concurrency 5   # Allow 5 concurrent agents
  orch daemon run --delay 5          # 5 second delay between spawns
  orch daemon run --dry-run          # Preview mode, no actual spawning
  orch daemon run --replace          # Stop existing daemon first, then start
  orch daemon run --group orch       # Only process issues from orch group projects`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonLoop()
	},
}

var daemonOnceCmd = &cobra.Command{
	Use:   "once",
	Short: "Process a single issue and exit",
	Long: `Process the next ready issue from the beads queue and exit.

Useful for testing daemon behavior without running the full polling loop.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonOnce()
	},
}

var daemonPreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Show what would be processed next",
	Long: `Show the next issue that would be processed without actually processing it.

Displays the issue details, inferred skill, and model selection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonPreview()
	},
}

var daemonReflectCmd = &cobra.Command{
	Use:   "reflect",
	Short: "Run kb reflect analysis and store suggestions",
	Long: `Run knowledge base reflection analysis to identify synthesis opportunities,
promotion candidates, stale decisions, and potential drifts.

Results are saved and will be surfaced at next session start.

Examples:
  orch daemon reflect                # Run reflection analysis`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonReflect()
	},
}

var daemonResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a paused daemon",
	Long: `Send a resume signal to the daemon.

The daemon pauses spawning after a configurable number of agents are marked
ready-for-review without human verification. This command resets the counter
and allows the daemon to continue spawning.

Also clears invariant checker pause state if the daemon was paused due to
invariant violations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonResume()
	},
}

var daemonCleanStaleCmd = &cobra.Command{
	Use:   "clean-stale",
	Short: "Find and clean stale cross-project completions",
	Long: `Find orphaned cross-project completions that have Phase: Complete
and daemon:ready-review labels but will never be resolved because
the source project was merged or archived.

By default, shows stale issues without closing them (dry-run).
Use --close to actually close the stale issues.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		closeStale, _ := cmd.Flags().GetBool("close")
		return runDaemonCleanStale(closeStale)
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon running state",
	Long: `Show whether the daemon is running, its PID, capacity, and recent activity.

Performs PID liveness check to detect stale status files from crashed daemons.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonStatus()
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	Long: `Send SIGTERM to the running daemon process. The daemon will finish
its current work and shut down gracefully.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemonStop()
	},
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Stop and restart the daemon",
	Long: `Stop the existing daemon (if running) and start a new one.

Equivalent to running 'daemon stop' followed by 'daemon run'.
All flags for 'daemon run' are also available on restart.`,
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
	daemonAgreementCheckInterval     int // Agreement check interval in minutes (0 = disabled)
	daemonBeadsHealthInterval        int // Beads health snapshot interval in minutes (0 = disabled)
	daemonFrictionAccumInterval      int // Friction accumulation interval in minutes (0 = disabled)
	daemonReplace                 bool   // Stop existing daemon before starting (graceful takeover)
	daemonGroup                   string // Filter to projects in this group (from groups.yaml)
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
		cmd.Flags().IntVar(&daemonBeadsHealthInterval, "beads-health-interval", 60, "Beads health snapshot interval in minutes (0 = disabled, default: 60)")
		cmd.Flags().IntVar(&daemonFrictionAccumInterval, "friction-accumulation-interval", 60, "Friction accumulation interval in minutes (0 = disabled, default: 60)")
		cmd.Flags().StringVar(&daemonGroup, "group", "", "Filter to projects in this group (from groups.yaml)")
		cmd.Flags().MarkHidden("max-agents")
	}

	// Add label filter and group to preview and once commands (share the same variable)
	daemonPreviewCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
	daemonPreviewCmd.Flags().StringVar(&daemonGroup, "group", "", "Filter to projects in this group (from groups.yaml)")
	daemonOnceCmd.Flags().StringVar(&daemonLabel, "label", "triage:ready", "Filter issues by label (empty = no filter)")
	daemonOnceCmd.Flags().StringVar(&daemonGroup, "group", "", "Filter to projects in this group (from groups.yaml)")
}
