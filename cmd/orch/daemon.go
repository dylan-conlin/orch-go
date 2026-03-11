// Package main provides the CLI entry point for orch-go.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/control"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/notify"
)

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

	// Wire beads health service (reuses collectHealthSnapshot from doctor_health.go)
	d.BeadsHealth = daemon.NewDefaultBeadsHealthService(collectHealthSnapshot, getHealthStore())

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
	defer cancel()

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
	defer dlog.Close()

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

	dlog.Printf("Starting daemon...\n")
	dlog.Printf("  Poll interval:    %s\n", formatDaemonDuration(config.PollInterval))
	dlog.Printf("  Concurrency:      %d (worker pool)\n", config.MaxAgents)
	dlog.Printf("  Required label:   %s\n", config.Label)
	dlog.Printf("  Spawn delay:      %s\n", formatDaemonDuration(config.SpawnDelay))
	if config.ReflectEnabled {
		dlog.Printf("  Reflect interval:  %s\n", formatDaemonDuration(config.ReflectInterval))
		dlog.Printf("  Reflect issues:    %v\n", config.ReflectCreateIssues)
		dlog.Printf("  Reflect open:      %v\n", config.ReflectOpenEnabled)
	} else {
		dlog.Printf("  Reflect interval:  disabled\n")
	}
	if config.ReflectModelDriftEnabled {
		dlog.Printf("  Model drift:       %s\n", formatDaemonDuration(config.ReflectModelDriftInterval))
	} else {
		dlog.Printf("  Model drift:       disabled\n")
	}
	if config.KnowledgeHealthEnabled {
		dlog.Printf("  Knowledge health:  %s (threshold: %d entries)\n", formatDaemonDuration(config.KnowledgeHealthInterval), config.KnowledgeHealthThreshold)
	} else {
		dlog.Printf("  Knowledge health:  disabled\n")
	}
	if config.CleanupEnabled {
		dlog.Printf("  Cleanup interval:  %s\n", formatDaemonDuration(config.CleanupInterval))
		dlog.Printf("  Cleanup age:       %d days\n", config.CleanupAgeDays)
		dlog.Printf("  Cleanup preserve:  %v (orchestrator sessions)\n", config.CleanupPreserveOrchestrator)
	} else {
		dlog.Printf("  Cleanup interval:  disabled\n")
	}
	if config.RecoveryEnabled {
		dlog.Printf("  Recovery interval: %s\n", formatDaemonDuration(config.RecoveryInterval))
		dlog.Printf("  Recovery idle:     %s\n", formatDaemonDuration(config.RecoveryIdleThreshold))
		dlog.Printf("  Recovery rate:     %s (per agent)\n", formatDaemonDuration(config.RecoveryRateLimit))
	} else {
		dlog.Printf("  Recovery interval: disabled\n")
	}
	if config.OrphanDetectionEnabled {
		dlog.Printf("  Orphan detection:  %s (age threshold: %s)\n", formatDaemonDuration(config.OrphanDetectionInterval), formatDaemonDuration(config.OrphanAgeThreshold))
	} else {
		dlog.Printf("  Orphan detection:  disabled\n")
	}
	if config.VerificationPauseThreshold > 0 {
		dlog.Printf("  Verify threshold:  %d (pause after N unverified completions)\n", config.VerificationPauseThreshold)
	} else {
		dlog.Printf("  Verify threshold:  disabled\n")
	}
	if config.InvariantCheckEnabled && config.InvariantViolationThreshold > 0 {
		dlog.Printf("  Invariant check:   enabled (pause after %d consecutive violation cycles)\n", config.InvariantViolationThreshold)
	} else {
		dlog.Printf("  Invariant check:   disabled\n")
	}
	dlog.Printf("\n")

	// Emit accretion snapshot at startup if last snapshot >6 days old
	if emitDaemonSnapshot(logger, projectDir) {
		dlog.Printf("Emitted accretion snapshot (>6d since last)\n")
	}

	// Main polling loop
	for {
		select {
		case <-ctx.Done():
			dlog.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
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
		if reconcileResult.Freed > 0 {
			dlog.Printf("[%s] Reconciled: freed %d stale slots\n", timestamp, reconcileResult.Freed)
		}
		if reconcileResult.Added > 0 {
			dlog.Printf("[%s] Reconciled: seeded %d agents from prior run (pool was under-counting)\n", timestamp, reconcileResult.Added)
		}

		// Check for verification signal (human ran `orch complete`)
		// This resets the counter and unpauses the daemon.
		if d.VerificationTracker != nil {
			if verified, err := daemon.CheckAndClearVerificationSignal(); err != nil {
				dlog.Errorf("[%s] Warning: failed to check verification signal: %v\n", timestamp, err)
			} else if verified {
				d.VerificationTracker.RecordHumanVerification()
				dlog.Printf("[%s] Human verification detected - verification counter reset\n", timestamp)
			}
		}

		// Check for resume signal (manual resume command)
		// This allows Dylan to resume the daemon without running orch complete.
		// Also clears invariant checker pause state.
		{
			if resumed, err := daemon.CheckAndClearResumeSignal(); err != nil {
				dlog.Errorf("[%s] Warning: failed to check resume signal: %v\n", timestamp, err)
			} else if resumed {
				if d.VerificationTracker != nil {
					d.VerificationTracker.Resume()
				}
				if d.InvariantChecker != nil {
					d.InvariantChecker.Resume()
				}
				dlog.Printf("[%s] Daemon resumed manually - verification counter and invariant checker reset\n", timestamp)
			}
		}

		// Check verification pause BEFORE spawning
		// This enforces the verifiability-first constraint by pausing after N agents
		// are marked ready-for-review without human verification (manual orch complete).
		if d.VerificationTracker != nil {
			verifyStatus := d.VerificationTracker.Status()
			if d.VerificationTracker.IsPaused() {
				breakdown := verificationBreakdown()
				dlog.Printf("[%s] Verification pause: %d unverified completions, threshold is %d%s\n",
					timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
				dlog.Printf("[%s]    Run 'orch daemon resume' after reviewing completed work to continue\n", timestamp)

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
					dlog.Errorf("Warning: failed to write status file: %v\n", err)
				}

				time.Sleep(config.PollInterval)
				continue
			}
			if verifyStatus.IsEnabled() {
				dlog.Printf("[%s] Verification check: %d/%d unverified completions, proceeding\n",
					timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
			}
		}

		// Run all periodic maintenance tasks (reflection, cleanup, recovery, etc.)
		periodicResult := runPeriodicTasks(d, timestamp, daemonVerbose, logger)
		knowledgeHealthSnapshot := periodicResult.KnowledgeHealthSnapshot
		phaseTimeoutSnapshot := periodicResult.PhaseTimeoutSnapshot
		questionDetectionSnapshot := periodicResult.QuestionDetectionSnapshot
		agreementCheckSnapshot := periodicResult.AgreementCheckSnapshot
		beadsHealthSnapshot := periodicResult.BeadsHealthSnapshot
		frictionAccumulationSnapshot := periodicResult.FrictionAccumulationSnapshot

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
			dlog.Errorf("[%s] Completion processing error: %v\n", timestamp, err)

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
				dlog.Errorf("Warning: failed to log completion error event: %v\n", logErr)
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
						dlog.Printf("[%s] Auto-completed: %s (tier=auto)\n",
							timestamp, cr.BeadsID)
					} else {
						dlog.Printf("[%s] Ready for review: %s (escalation=%s)\n",
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
						dlog.Printf("[%s] Verification threshold reached: %d/%d agents ready for review%s\n",
							timestamp, verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
						dlog.Printf("[%s]    Daemon will pause spawning on next cycle\n", timestamp)
						dlog.Printf("[%s]    Run 'orch daemon resume' after reviewing completed work\n", timestamp)
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
						dlog.Errorf("Warning: failed to log completion event: %v\n", err)
					}
				} else if cr.Error != nil && daemonVerbose {
					dlog.Printf("[%s] Review required: %s - %v (escalation=%s)\n",
						timestamp, cr.BeadsID, cr.Error, cr.Escalation)
				}
			}
			if completedThisCycle > 0 && daemonVerbose {
				dlog.Printf("[%s] Marked %d agent(s) ready for review this cycle\n", timestamp, completedThisCycle)
			}
		}

		// Run self-check invariants to catch scope-expansion bugs at runtime.
		// Checks: active count range, verification counter bounds, completion agent validity.
		// Pauses daemon after configurable threshold of consecutive violation cycles.
		if d.InvariantChecker != nil {
			var completedAgents []daemon.CompletedAgent
			if completionResult != nil {
				for _, cr := range completionResult.Processed {
					if cr.Processed {
						// Build a CompletedAgent from the completion result for checking.
						// The actual CompletedAgent list comes from the completion finder.
						completedAgents = append(completedAgents, daemon.CompletedAgent{
							BeadsID: cr.BeadsID,
						})
					}
				}
			}
			// Also get full completed agents list if available (has WorkspacePath/ProjectDir)
			if d.Completions != nil {
				fullAgents, err := d.Completions.ListCompletedAgents(completionConfig)
				if err == nil {
					completedAgents = fullAgents
				}
				// Fail-open: if listing fails, use the partial list from completion results
			}

			verifyStatus := daemon.VerificationStatus{}
			if d.VerificationTracker != nil {
				verifyStatus = d.VerificationTracker.Status()
			}

			invariantInput := &daemon.InvariantInput{
				ActiveCount:           d.ActiveCount(),
				MaxAgents:             config.MaxAgents,
				VerificationCount:     verifyStatus.CompletionsSinceVerification,
				VerificationThreshold: verifyStatus.Threshold,
				CompletedAgents:       completedAgents,
			}

			checkResult := d.InvariantChecker.Check(invariantInput)

			if checkResult.Error != nil {
				dlog.Errorf("[%s] Invariant check error (fail-open): %v\n", timestamp, checkResult.Error)
			} else if checkResult.HasViolations() {
				for _, v := range checkResult.Violations {
					dlog.Errorf("[%s] INVARIANT VIOLATION [%s/%s]: %s\n", timestamp, v.Severity, v.Name, v.Message)
				}
				dlog.Printf("[%s] Invariant violations: %d this cycle, %d consecutive cycles (threshold: %d)\n",
					timestamp, len(checkResult.Violations), d.InvariantChecker.ViolationCount(), config.InvariantViolationThreshold)
			}

			if d.InvariantChecker.IsPaused() {
				dlog.Printf("[%s] DAEMON PAUSED: invariant violations exceeded threshold (%d consecutive cycles)\n",
					timestamp, d.InvariantChecker.ViolationCount())
				dlog.Printf("[%s]    Run 'orch daemon resume' to clear and continue\n", timestamp)

				// Write paused status file
				pauseStatus := daemon.DaemonStatus{
					PID: os.Getpid(),
					Capacity: daemon.CapacityStatus{
						Max:       config.MaxAgents,
						Active:    d.ActiveCount(),
						Available: d.AvailableSlots(),
					},
					LastPoll:  time.Now(),
					LastSpawn: lastSpawn,
					Status:    "paused",
				}
				if err := daemon.WriteStatusFile(pauseStatus); err != nil && daemonVerbose {
					dlog.Errorf("Warning: failed to write status file: %v\n", err)
				}

				time.Sleep(config.PollInterval)
				continue
			}
		}

		// Check beads circuit breaker — if beads is unhealthy, skip polling and back off.
		// This prevents the lock cascade where daemon keeps spawning bd processes
		// that pile up behind a stuck JSONL lock.
		if d.BeadsCircuitBreaker != nil && d.BeadsCircuitBreaker.IsOpen() {
			backoff := d.BeadsCircuitBreaker.BackoffDuration()
			failures := d.BeadsCircuitBreaker.ConsecutiveFailures()
			dlog.Printf("[%s] Beads circuit breaker open: %d consecutive failures, backing off %s\n",
				timestamp, failures, formatDaemonDuration(backoff))
			select {
			case <-ctx.Done():
				dlog.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
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
			dlog.Errorf("[%s] Failed to list ready issues: %v\n", timestamp, readyErr)
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
			KnowledgeHealth:      knowledgeHealthSnapshot,
			PhaseTimeout:         phaseTimeoutSnapshot,
			QuestionDetection:    questionDetectionSnapshot,
			AgreementCheck:       agreementCheckSnapshot,
			BeadsHealth:          beadsHealthSnapshot,
			FrictionAccumulation: frictionAccumulationSnapshot,
		}
		if err := daemon.WriteStatusFile(status); err != nil && daemonVerbose {
			dlog.Errorf("Warning: failed to write status file: %v\n", err)
		}

		// Check capacity before polling
		if d.AtCapacity() {
			activeCount := d.ActiveCount()
			if daemonVerbose {
				dlog.Printf("[%s] At capacity (%d/%d agents active), waiting...\n",
					timestamp, activeCount, daemonMaxAgents)
			}

			// Stuck detection: all slots full with no activity for 10+ min
			stuckThreshold := 10 * time.Minute
			stuckCooldown := 30 * time.Minute
			if checkDaemonStuck(lastSpawn, lastCompletion, lastStuckNotification, stuckThreshold, stuckCooldown) {
				dlog.Printf("[%s] Daemon stuck: capacity full, no spawns or completions in %s\n",
					timestamp, stuckThreshold)
				if err := stuckNotifier.DaemonStuck(activeCount, daemonMaxAgents); err != nil {
					dlog.Errorf("Warning: stuck notification failed: %v\n", err)
				}
				lastStuckNotification = time.Now()
			}

			// Wait for poll interval before checking again
			select {
			case <-ctx.Done():
				dlog.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
				return nil
			case <-time.After(config.PollInterval):
				continue
			}
		}

		if daemonVerbose {
			dlog.Printf("[%s] Polling for issues...\n", timestamp)
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
				dlog.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
				return nil
			default:
			}

			// Check capacity
			if d.AtCapacity() {
				if daemonVerbose {
					dlog.Printf("[%s] At capacity, stopping this cycle\n", timestamp)
				}
				break
			}

			result, err := d.OnceExcluding(skippedThisCycle)
			if err != nil {
				dlog.Errorf("Error: %v\n", err)
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
						dlog.Errorf("[%s] Skipping %s: %v\n",
							timestamp, result.Issue.ID, result.Error)
					} else if daemonVerbose {
						dlog.Printf("[%s] Skipping %s: %s\n",
							timestamp, result.Issue.ID, result.Message)
					}
					// Continue to try the next issue
					continue
				}

				// No more issues or non-issue-specific condition (rate limit, paused, etc.)
				if daemonVerbose && spawnedThisCycle == 0 {
					// Use the message from Once() which indicates why processing stopped
					dlog.Printf("[%s] %s\n", timestamp, result.Message)
				}
				break
			}

			processed++
			spawnedThisCycle++
			lastSpawn = time.Now()
			if result.ExtractionSpawned {
				dlog.Printf("[%s] Auto-extraction: %s (blocking %s) - %s\n",
					timestamp,
					result.Issue.ID,
					result.OriginalIssueID,
					result.Issue.Title,
				)
			} else if result.ArchitectEscalated {
				dlog.Printf("[%s] Architect escalation: %s (%s, %s) - %s\n",
					timestamp,
					result.Issue.ID,
					result.Skill,
					result.Model,
					result.Issue.Title,
				)
			} else {
				dlog.Printf("[%s] Spawned: %s (%s, %s) - %s\n",
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
				dlog.Errorf("Warning: failed to log event: %v\n", err)
			}

			// Log architect escalation decision when a hotspot match was evaluated
			if result.ArchitectEscalationDetail != nil {
				if err := logger.LogArchitectEscalation(events.ArchitectEscalationData{
					IssueID:           result.Issue.ID,
					HotspotFile:       result.ArchitectEscalationDetail.HotspotFile,
					HotspotType:       result.ArchitectEscalationDetail.HotspotType,
					Escalated:         result.ArchitectEscalationDetail.Escalated,
					PriorArchitectRef: result.ArchitectEscalationDetail.PriorArchitectRef,
				}); err != nil {
					dlog.Errorf("Warning: failed to log architect escalation event: %v\n", err)
				}
			}

			// Delay before next spawn to avoid rate limits
			select {
			case <-ctx.Done():
				dlog.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
				return nil
			case <-time.After(config.SpawnDelay):
			}
		}

		// If poll interval is 0, run once and exit
		if config.PollInterval == 0 {
			dlog.Printf("Run-once mode. Spawned %d, completed %d.\n", processed, completed)
			return nil
		}

		// Wait for next poll cycle
		if daemonVerbose {
			dlog.Printf("[%s] Spawned %d this cycle, waiting %s before next poll...\n",
				timestamp, spawnedThisCycle, formatDaemonDuration(config.PollInterval))
		}
		select {
		case <-ctx.Done():
			dlog.Printf("\nDaemon stopped. Spawned %d, completed %d, cycles %d.\n", processed, completed, cycles)
			return nil
		case <-time.After(config.PollInterval):
		}
	}
}
