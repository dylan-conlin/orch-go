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
	"github.com/dylan-conlin/orch-go/pkg/group"
	"github.com/dylan-conlin/orch-go/pkg/notify"
)

// daemonLoopState holds shared state for the daemon main loop.
// Extracted to allow runDaemonLoop to be composed from smaller functions.
type daemonLoopState struct {
	config     daemon.Config
	d          *daemon.Daemon
	dlog       *daemon.DaemonLogger
	logger     *events.Logger
	projectDir string
	ctx        context.Context
	cancel     context.CancelFunc

	// Counters
	processed int
	completed int
	cycles    int

	// Timing
	lastSpawn             time.Time
	lastCompletion        time.Time
	lastStuckNotification time.Time

	stuckNotifier *notify.Notifier

	// Resources to clean up (managed by caller via defer)
	pidLock *daemon.PIDLock
}

// daemonSetup initializes the daemon: acquires PID lock, builds config,
// wires subsystems, sets up signal handling and logging.
func daemonSetup() (*daemonLoopState, error) {
	// If --replace, stop existing daemon before acquiring lock
	if daemonReplace {
		pid := daemon.ReadPIDFromLockFile()
		if pid > 0 && daemon.IsProcessAlive(pid) {
			fmt.Printf("Replacing existing daemon (PID %d)...\n", pid)
			if err := daemon.StopDaemon(daemon.StopOptions{}); err != nil && err != daemon.ErrNoDaemonRunning {
				return nil, fmt.Errorf("failed to stop existing daemon: %w", err)
			}
			fmt.Println("Previous daemon stopped.")
		}
	}

	// Acquire PID lock to ensure single daemon instance.
	// This prevents multiple daemon processes from accumulating silently
	// and fighting over the status file and spawns.
	pidLock, err := daemon.AcquirePIDLock()
	if err != nil {
		return nil, fmt.Errorf("cannot start daemon: %w", err)
	}

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
		pidLock.Release()
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	config := daemonConfigFromFlags()
	d := daemon.NewWithConfig(config)

	// Initialize project registry for cross-project issue resolution.
	// Uses groups-aware discovery: merges kb projects list with groups.yaml members,
	// auto-discovering projects from filesystem heuristics.
	// Falls back to kb-only if groups.yaml is missing.
	// If both fail (kb not installed, no projects), daemon still works
	// but spawns everything into the current directory.
	registry, err := daemon.NewProjectRegistryWithGroups()
	if err != nil {
		// Fall back to kb-only registry
		registry, err = daemon.NewProjectRegistry()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: project registry unavailable: %v\n", err)
	} else {
		// Wire group-based account routing: load groups.yaml and build
		// the kbProjects map so the daemon can resolve account per project.
		if groupCfg, err := group.Load(); err == nil {
			d.GroupConfig = groupCfg
			d.KBProjects = daemon.BuildKBProjectsMap(registry)

			// Filter registry to only projects in the specified group
			if daemonGroup != "" {
				members := groupCfg.ResolveGroupMembers(daemonGroup, d.KBProjects)
				if len(members) == 0 {
					pidLock.Release()
					return nil, fmt.Errorf("group %q not found or has no members in groups.yaml", daemonGroup)
				}
				// Build allowed dirs set from group member names
				allowedDirs := make(map[string]bool, len(members))
				for _, name := range members {
					if path, ok := d.KBProjects[name]; ok {
						allowedDirs[path] = true
					}
				}
				registry = registry.FilterByDirs(allowedDirs)
				fmt.Printf("Group %q: scoped to %d projects\n", daemonGroup, len(registry.Projects()))
			}
		} else if daemonGroup != "" {
			pidLock.Release()
			return nil, fmt.Errorf("--group requires groups.yaml: %w", err)
		}

		d.ProjectRegistry = registry
	}

	// NOTE: Extraction system disabled. HotspotChecker is not set, so the
	// extraction gate in Once() and hotspot warnings in Preview() are skipped.
	// The daemon goes straight from polling bd ready to spawning issues.
	// To re-enable, uncomment: d.HotspotChecker = daemon.NewGitHotspotChecker()

	// Wire beads health service (reuses collectHealthSnapshot from doctor_health.go)
	d.BeadsHealth = daemon.NewDefaultBeadsHealthService(collectHealthSnapshot, getHealthStore())

	// Wire proactive extraction service (creates architect issues at 1200 lines)
	d.ProactiveExtraction = daemon.NewDefaultProactiveExtractionService()

	// Wire digest producer (scans .kb/ artifacts, produces thinking products)
	{
		projectDir, _ := os.Getwd()
		d.Digest = daemon.NewDefaultDigestService(projectDir)
		homeDir, _ := os.UserHomeDir()
		d.DigestDir = filepath.Join(homeDir, ".orch", "digest")
		d.DigestStatePath = filepath.Join(homeDir, ".orch", "digest-state.json")
	}

	// Wire focus-aware priority boost
	wireFocusBoost(d)

	// Wire auto-completer for auto-tier agents.
	// When review tier is "auto", daemon runs `orch complete` directly
	// instead of labeling for orchestrator review.
	d.AutoCompleter = &daemon.OrcCompleter{}

	// Seed verification tracker with unverified backlog from previous sessions
	seedVerificationTracker(d)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, stopping daemon...")
		cancel()
	}()

	// Initialize daemon logger — writes to both stdout and ~/.orch/daemon.log.
	// The log file survives process detachment and orphaned stdout file descriptors.
	dlog := daemon.NewDaemonLogger()

	logger := events.NewLogger(events.DefaultLogPath())

	return &daemonLoopState{
		config:        config,
		d:             d,
		dlog:          dlog,
		logger:        logger,
		projectDir:    projectDir,
		ctx:           ctx,
		cancel:        cancel,
		pidLock:       pidLock,
		stuckNotifier: notify.Default(),
	}, nil
}

// logDaemonConfig prints the daemon configuration at startup.
func (s *daemonLoopState) logDaemonConfig() {
	s.dlog.Printf("Starting daemon...\n")
	s.dlog.Printf("  Poll interval:    %s\n", formatDaemonDuration(s.config.PollInterval))
	s.dlog.Printf("  Concurrency:      %d (worker pool)\n", s.config.MaxAgents)
	s.dlog.Printf("  Required label:   %s\n", s.config.Label)
	s.dlog.Printf("  Spawn delay:      %s\n", formatDaemonDuration(s.config.SpawnDelay))
	if s.config.ReflectEnabled {
		s.dlog.Printf("  Reflect interval:  %s\n", formatDaemonDuration(s.config.ReflectInterval))
		s.dlog.Printf("  Reflect issues:    %v\n", s.config.ReflectCreateIssues)
		s.dlog.Printf("  Reflect open:      %v\n", s.config.ReflectOpenEnabled)
	} else {
		s.dlog.Printf("  Reflect interval:  disabled\n")
	}
	if s.config.ReflectModelDriftEnabled {
		s.dlog.Printf("  Model drift:       %s\n", formatDaemonDuration(s.config.ReflectModelDriftInterval))
	} else {
		s.dlog.Printf("  Model drift:       disabled\n")
	}
	if s.config.KnowledgeHealthEnabled {
		s.dlog.Printf("  Knowledge health:  %s (threshold: %d entries)\n", formatDaemonDuration(s.config.KnowledgeHealthInterval), s.config.KnowledgeHealthThreshold)
	} else {
		s.dlog.Printf("  Knowledge health:  disabled\n")
	}
	if s.config.CleanupEnabled {
		s.dlog.Printf("  Cleanup interval:  %s\n", formatDaemonDuration(s.config.CleanupInterval))
		s.dlog.Printf("  Cleanup age:       %d days\n", s.config.CleanupAgeDays)
		s.dlog.Printf("  Cleanup preserve:  %v (orchestrator sessions)\n", s.config.CleanupPreserveOrchestrator)
	} else {
		s.dlog.Printf("  Cleanup interval:  disabled\n")
	}
	if s.config.RecoveryEnabled {
		s.dlog.Printf("  Recovery interval: %s\n", formatDaemonDuration(s.config.RecoveryInterval))
		s.dlog.Printf("  Recovery idle:     %s\n", formatDaemonDuration(s.config.RecoveryIdleThreshold))
		s.dlog.Printf("  Recovery rate:     %s (per agent)\n", formatDaemonDuration(s.config.RecoveryRateLimit))
	} else {
		s.dlog.Printf("  Recovery interval: disabled\n")
	}
	if s.config.OrphanDetectionEnabled {
		s.dlog.Printf("  Orphan detection:  %s (age threshold: %s)\n", formatDaemonDuration(s.config.OrphanDetectionInterval), formatDaemonDuration(s.config.OrphanAgeThreshold))
	} else {
		s.dlog.Printf("  Orphan detection:  disabled\n")
	}
	if s.config.VerificationPauseThreshold > 0 {
		s.dlog.Printf("  Verify threshold:  %d (pause after N unverified completions)\n", s.config.VerificationPauseThreshold)
	} else {
		s.dlog.Printf("  Verify threshold:  disabled\n")
	}
	if s.config.InvariantCheckEnabled && s.config.InvariantViolationThreshold > 0 {
		s.dlog.Printf("  Invariant check:   enabled (pause after %d consecutive violation cycles)\n", s.config.InvariantViolationThreshold)
	} else {
		s.dlog.Printf("  Invariant check:   disabled\n")
	}
	if s.config.RegistryRefreshEnabled {
		s.dlog.Printf("  Registry refresh:  %s\n", formatDaemonDuration(s.config.RegistryRefreshInterval))
	} else {
		s.dlog.Printf("  Registry refresh:  disabled\n")
	}
	if s.d.ProjectRegistry != nil {
		s.dlog.Printf("  Projects:          %d registered\n", len(s.d.ProjectRegistry.Projects()))
	}
	if daemonGroup != "" {
		s.dlog.Printf("  Group filter:      %s\n", daemonGroup)
	}
	s.dlog.Printf("\n")

	// Emit accretion snapshot at startup if last snapshot >6 days old
	if emitDaemonSnapshot(s.logger, s.projectDir) {
		s.dlog.Printf("Emitted accretion snapshot (>6d since last)\n")
	}
}

// processDaemonCompletions marks Phase: Complete agents as ready-for-review
// and logs completion events. Returns the completion result for invariant checking.
func (s *daemonLoopState) processDaemonCompletions(timestamp string) *daemon.CompletionLoopResult {
	completionConfig := daemon.CompletionConfig{
		ProjectDir: s.projectDir,
		ServerURL:  serverURL,
		DryRun:     false,
		Verbose:    daemonVerbose,
	}
	completionResult, err := s.d.CompletionOnce(completionConfig)
	if err != nil {
		// Record completion failure for health tracking
		if s.d.CompletionFailureTracker != nil {
			s.d.CompletionFailureTracker.RecordFailure(err.Error())
		}

		// Always log completion errors (not just in verbose mode)
		s.dlog.Errorf("[%s] Completion processing error: %v\n", timestamp, err)

		// Log the error event
		event := events.Event{
			Type:      "daemon.completion_error",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"error":   err.Error(),
				"message": "Completion processing failed",
			},
		}
		if logErr := s.logger.Log(event); logErr != nil {
			s.dlog.Errorf("Warning: failed to log completion error event: %v\n", logErr)
		}
	} else {
		// Record successful completion processing
		if s.d.CompletionFailureTracker != nil {
			s.d.CompletionFailureTracker.RecordSuccess()
		}
	}

	if completionResult != nil {
		completedThisCycle := 0
		for _, cr := range completionResult.Processed {
			if cr.Processed {
				completedThisCycle++
				s.completed++
				s.lastCompletion = time.Now()
				if cr.AutoCompleted {
					s.dlog.Printf("[%s] Auto-completed: %s (tier=auto)\n",
						timestamp, cr.BeadsID)
				} else {
					s.dlog.Printf("[%s] Ready for review: %s (escalation=%s)\n",
						timestamp, cr.BeadsID, cr.Escalation)
				}

				// NOTE: RecordCompletion() is called inside ProcessCompletion()
				// (completion_processing.go). Do NOT call it again here — that
				// caused a double-counting bug where each completion incremented
				// the counter by 2, making the daemon pause at half the expected
				// number of completions.

				// Check if verification tracker was paused by ProcessCompletion
				if s.d.VerificationTracker != nil && s.d.VerificationTracker.IsPaused() {
					verifyStatus := s.d.VerificationTracker.Status()
					breakdown := verificationBreakdown()
					s.dlog.Printf("[%s] Verification threshold reached: %d/%d agents ready for review%s\n",
						timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
					s.dlog.Printf("[%s]    Daemon will pause spawning on next cycle\n", timestamp)
					s.dlog.Printf("[%s]    Run 'orch daemon resume' after reviewing completed work\n", timestamp)
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
				if err := s.logger.Log(event); err != nil {
					s.dlog.Errorf("Warning: failed to log completion event: %v\n", err)
				}

				// Push completion notification to dashboard (fire-and-forget)
				notifyDashboardCompletion(cr.BeadsID, cr.CloseReason, cr.Escalation.String())
			} else if cr.Error != nil && daemonVerbose {
				s.dlog.Printf("[%s] Review required: %s - %v (escalation=%s)\n",
					timestamp, cr.BeadsID, cr.Error, cr.Escalation)
			}
		}
		if completedThisCycle > 0 && daemonVerbose {
			s.dlog.Printf("[%s] Marked %d agent(s) ready for review this cycle\n", timestamp, completedThisCycle)
		}
	}

	return completionResult
}

// spawnCycleResult reports the outcome of a single spawn cycle.
type spawnCycleResult struct {
	spawned   int
	cancelled bool // ctx was cancelled during spawn cycle
}

// runDaemonSpawnCycle processes the inner spawn loop: polls for ready issues
// and spawns agents until at capacity or queue is empty.
func (s *daemonLoopState) runDaemonSpawnCycle(timestamp string) spawnCycleResult {
	spawnedThisCycle := 0
	skippedThisCycle := make(map[string]bool)
	for {
		// Check for interrupt
		select {
		case <-s.ctx.Done():
			return spawnCycleResult{spawned: spawnedThisCycle, cancelled: true}
		default:
		}

		// Check capacity
		if s.d.AtCapacity() {
			if daemonVerbose {
				s.dlog.Printf("[%s] At capacity, stopping this cycle\n", timestamp)
			}
			break
		}

		result, err := s.d.OnceExcluding(skippedThisCycle)
		if err != nil {
			s.dlog.Errorf("Error: %v\n", err)
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
					s.dlog.Errorf("[%s] Skipping %s: %v\n",
						timestamp, result.Issue.ID, result.Error)
				} else if daemonVerbose {
					s.dlog.Printf("[%s] Skipping %s: %s\n",
						timestamp, result.Issue.ID, result.Message)
				}
				// Continue to try the next issue
				continue
			}

			// No more issues or non-issue-specific condition (rate limit, paused, etc.)
			if daemonVerbose && spawnedThisCycle == 0 {
				// Use the message from Once() which indicates why processing stopped
				s.dlog.Printf("[%s] %s\n", timestamp, result.Message)
			}
			break
		}

		s.processed++
		spawnedThisCycle++
		s.lastSpawn = time.Now()
		if result.ExtractionSpawned {
			s.dlog.Printf("[%s] Auto-extraction: %s (blocking %s) - %s\n",
				timestamp,
				result.Issue.ID,
				result.OriginalIssueID,
				result.Issue.Title,
			)
		} else if result.ArchitectEscalated {
			s.dlog.Printf("[%s] Architect escalation: %s (%s, %s) - %s\n",
				timestamp,
				result.Issue.ID,
				result.Skill,
				result.Model,
				result.Issue.Title,
			)
		} else {
			s.dlog.Printf("[%s] Spawned: %s (%s, %s) - %s\n",
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
			"count":    s.processed,
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
		if err := s.logger.Log(event); err != nil {
			s.dlog.Errorf("Warning: failed to log event: %v\n", err)
		}

		// Log architect escalation decision when a hotspot match was evaluated
		if result.ArchitectEscalationDetail != nil {
			if err := s.logger.LogArchitectEscalation(events.ArchitectEscalationData{
				IssueID:           result.Issue.ID,
				HotspotFile:       result.ArchitectEscalationDetail.HotspotFile,
				HotspotType:       result.ArchitectEscalationDetail.HotspotType,
				Escalated:         result.ArchitectEscalationDetail.Escalated,
				PriorArchitectRef: result.ArchitectEscalationDetail.PriorArchitectRef,
			}); err != nil {
				s.dlog.Errorf("Warning: failed to log architect escalation event: %v\n", err)
			}
		}

		// Delay before next spawn to avoid rate limits
		select {
		case <-s.ctx.Done():
			return spawnCycleResult{spawned: spawnedThisCycle, cancelled: true}
		case <-time.After(s.config.SpawnDelay):
		}
	}

	return spawnCycleResult{spawned: spawnedThisCycle}
}

// checkDaemonSignals checks for verification and resume signals between cycles.
func (s *daemonLoopState) checkDaemonSignals(timestamp string) {
	// Check for verification signal (human ran `orch complete`)
	if s.d.VerificationTracker != nil {
		if verified, err := daemon.CheckAndClearVerificationSignal(); err != nil {
			s.dlog.Errorf("[%s] Warning: failed to check verification signal: %v\n", timestamp, err)
		} else if verified {
			s.d.VerificationTracker.RecordHumanVerification()
			s.dlog.Printf("[%s] Human verification detected - verification counter reset\n", timestamp)
		}
	}

	// Check for resume signal (manual resume command)
	// This allows Dylan to resume the daemon without running orch complete.
	// Also clears invariant checker pause state.
	if resumed, err := daemon.CheckAndClearResumeSignal(); err != nil {
		s.dlog.Errorf("[%s] Warning: failed to check resume signal: %v\n", timestamp, err)
	} else if resumed {
		if s.d.VerificationTracker != nil {
			s.d.VerificationTracker.Resume()
		}
		if s.d.InvariantChecker != nil {
			s.d.InvariantChecker.Resume()
		}
		s.dlog.Printf("[%s] Daemon resumed manually - verification counter and invariant checker reset\n", timestamp)
	}
}

// checkVerificationPause checks if the daemon should pause for human verification.
// Returns true if the cycle should be skipped (daemon is paused).
func (s *daemonLoopState) checkVerificationPause(timestamp string) bool {
	if s.d.VerificationTracker == nil {
		return false
	}
	verifyStatus := s.d.VerificationTracker.Status()
	if s.d.VerificationTracker.IsPaused() {
		breakdown := verificationBreakdown()
		s.dlog.Printf("[%s] Verification pause: %d unverified completions, threshold is %d%s\n",
			timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
		s.dlog.Printf("[%s]    Run 'orch daemon resume' after reviewing completed work to continue\n", timestamp)

		// Write status file during pause so last_poll stays fresh
		// and status correctly shows "paused" instead of going stale.
		pauseStatus := daemon.DaemonStatus{
			PID: os.Getpid(),
			Capacity: daemon.CapacityStatus{
				Max:       s.config.MaxAgents,
				Active:    s.d.ActiveCount(),
				Available: s.d.AvailableSlots(),
			},
			LastPoll:       time.Now(),
			LastSpawn:      s.lastSpawn,
			LastCompletion: s.lastCompletion,
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
			s.dlog.Errorf("Warning: failed to write status file: %v\n", err)
		}

		time.Sleep(s.config.PollInterval)
		return true
	}
	if verifyStatus.IsEnabled() {
		s.dlog.Printf("[%s] Verification check: %d/%d unverified completions, proceeding\n",
			timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
	}
	return false
}

// checkInvariants runs self-check invariants to catch scope-expansion bugs at runtime.
// Returns true if the cycle should be skipped (daemon is paused due to violations).
func (s *daemonLoopState) checkInvariants(timestamp string, completionResult *daemon.CompletionLoopResult) bool {
	if s.d.InvariantChecker == nil {
		return false
	}

	completionConfig := daemon.CompletionConfig{
		ProjectDir: s.projectDir,
		ServerURL:  serverURL,
		DryRun:     false,
		Verbose:    daemonVerbose,
	}

	var completedAgents []daemon.CompletedAgent
	if completionResult != nil {
		for _, cr := range completionResult.Processed {
			if cr.Processed {
				completedAgents = append(completedAgents, daemon.CompletedAgent{
					BeadsID: cr.BeadsID,
				})
			}
		}
	}
	// Also get full completed agents list if available (has WorkspacePath/ProjectDir)
	if s.d.Completions != nil {
		fullAgents, err := s.d.Completions.ListCompletedAgents(completionConfig)
		if err == nil {
			completedAgents = fullAgents
		}
		// Fail-open: if listing fails, use the partial list from completion results
	}

	verifyStatus := daemon.VerificationStatus{}
	if s.d.VerificationTracker != nil {
		verifyStatus = s.d.VerificationTracker.Status()
	}

	invariantInput := &daemon.InvariantInput{
		ActiveCount:           s.d.ActiveCount(),
		MaxAgents:             s.config.MaxAgents,
		VerificationCount:     verifyStatus.CompletionsSinceVerification,
		VerificationThreshold: verifyStatus.Threshold,
		CompletedAgents:       completedAgents,
	}

	checkResult := s.d.InvariantChecker.Check(invariantInput)

	if checkResult.Error != nil {
		s.dlog.Errorf("[%s] Invariant check error (fail-open): %v\n", timestamp, checkResult.Error)
	} else if checkResult.HasViolations() {
		for _, v := range checkResult.Violations {
			s.dlog.Errorf("[%s] INVARIANT VIOLATION [%s/%s]: %s\n", timestamp, v.Severity, v.Name, v.Message)
		}
		s.dlog.Printf("[%s] Invariant violations: %d this cycle, %d consecutive cycles (threshold: %d)\n",
			timestamp, len(checkResult.Violations), s.d.InvariantChecker.ViolationCount(), s.config.InvariantViolationThreshold)
	}

	if s.d.InvariantChecker.IsPaused() {
		s.dlog.Printf("[%s] DAEMON PAUSED: invariant violations exceeded threshold (%d consecutive cycles)\n",
			timestamp, s.d.InvariantChecker.ViolationCount())
		s.dlog.Printf("[%s]    Run 'orch daemon resume' to clear and continue\n", timestamp)

		// Write paused status file
		pauseStatus := daemon.DaemonStatus{
			PID: os.Getpid(),
			Capacity: daemon.CapacityStatus{
				Max:       s.config.MaxAgents,
				Active:    s.d.ActiveCount(),
				Available: s.d.AvailableSlots(),
			},
			LastPoll:  time.Now(),
			LastSpawn: s.lastSpawn,
			Status:    "paused",
		}
		if err := daemon.WriteStatusFile(pauseStatus); err != nil && daemonVerbose {
			s.dlog.Errorf("Warning: failed to write status file: %v\n", err)
		}

		time.Sleep(s.config.PollInterval)
		return true
	}

	return false
}

// writeDaemonStatusFile writes the daemon status file with current state and snapshots.
func (s *daemonLoopState) writeDaemonStatusFile(readyCount int, periodicResult periodicTasksResult) {
	var verificationSnapshot *daemon.VerificationStatusSnapshot
	isPaused := false
	if s.d.VerificationTracker != nil {
		verifyStatus := s.d.VerificationTracker.Status()
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

	var completionFailureSnapshot *daemon.CompletionFailureSnapshot
	if s.d.CompletionFailureTracker != nil {
		snapshot := s.d.CompletionFailureTracker.Snapshot()
		if snapshot.TotalFailures > 0 {
			completionFailureSnapshot = &snapshot
		}
	}

	// Refresh pollTime to reflect actual status write time.
	// Processing (reconciliation, periodic tasks, completions, ready count)
	// between cycle start and here can take significant time, causing
	// DetermineStatus to see a stale pollTime and return "stalled" incorrectly.
	pollTime := time.Now()

	status := daemon.DaemonStatus{
		PID: os.Getpid(),
		Capacity: daemon.CapacityStatus{
			Max:       s.config.MaxAgents,
			Active:    s.d.ActiveCount(),
			Available: s.d.AvailableSlots(),
		},
		LastPoll:             pollTime,
		LastSpawn:            s.lastSpawn,
		LastCompletion:       s.lastCompletion,
		ReadyCount:           readyCount,
		Status:               daemon.DetermineStatus(pollTime, s.config.PollInterval, isPaused),
		Verification:         verificationSnapshot,
		CompletionFailures:   completionFailureSnapshot,
		KnowledgeHealth:      periodicResult.KnowledgeHealthSnapshot,
		PhaseTimeout:         periodicResult.PhaseTimeoutSnapshot,
		QuestionDetection:    periodicResult.QuestionDetectionSnapshot,
		AgreementCheck:       periodicResult.AgreementCheckSnapshot,
		BeadsHealth:          periodicResult.BeadsHealthSnapshot,
		FrictionAccumulation: periodicResult.FrictionAccumulationSnapshot,
	}
	if err := daemon.WriteStatusFile(status); err != nil && daemonVerbose {
		s.dlog.Errorf("Warning: failed to write status file: %v\n", err)
	}
}

// runWorkGraphAnalysis computes the work graph and surfaces removal candidates.
// Runs each cycle with the ready queue (no local state — fresh computation).
func (s *daemonLoopState) runWorkGraphAnalysis(readyIssues []daemon.Issue, timestamp string) {
	graph := daemon.ComputeWorkGraph(readyIssues, nil)

	candidates := graph.RemovalCandidates()
	if len(candidates) == 0 {
		return
	}

	// Log detected signals
	for _, c := range candidates {
		s.dlog.Printf("[%s] Work graph: %s\n", timestamp, c.Detail)
	}

	// Surface as a question issue (only title duplicates — high confidence).
	// Rate-limit to avoid creating duplicate question issues on every cycle.
	for _, dup := range graph.TitleDuplicates {
		if dup.Similarity >= 0.80 {
			s.dlog.Printf("[%s] Work graph: high-confidence duplicate detected (%s <-> %s, %.0f%%), consider dedup\n",
				timestamp, dup.IssueA, dup.IssueB, dup.Similarity*100)
		}
	}
}

// stopMessage returns the standard daemon shutdown summary.
func (s *daemonLoopState) stopMessage() string {
	return fmt.Sprintf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", s.processed, s.completed, s.cycles)
}
