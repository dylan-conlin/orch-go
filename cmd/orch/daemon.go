// Package main provides the CLI entry point for orch-go.
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

func runDaemonLoop() error {
	// Handle dry-run mode
	if daemonDryRun {
		return runDaemonDryRun()
	}

	s, err := daemonSetup()
	if err != nil {
		return err
	}
	defer s.pidLock.Release()
	defer s.cancel()
	defer s.dlog.Close()
	defer daemon.RemoveStatusFile()
	if daemonReflect {
		defer runReflectionAnalysis(daemonVerbose)
	}

	s.logDaemonConfig()

	// Main polling loop
	for {
		select {
		case <-s.ctx.Done():
			s.dlog.Printf("%s", s.stopMessage())
			return nil
		default:
		}

		s.cycles++
		timestamp := time.Now().Format("15:04:05")

		// Reconcile pool with actual running agents FIRST.
		// This handles two cases:
		// 1. Agents completed without daemon knowing → free stale slots
		// 2. After daemon restart, agents from prior run still active → add synthetic
		//    slots to prevent over-spawning past the concurrency cap
		// Must happen before status write so status shows accurate counts.
		reconcileResult := s.d.ReconcileWithOpenCode()
		if reconcileResult.Freed > 0 {
			s.dlog.Printf("[%s] Reconciled: freed %d stale slots\n", timestamp, reconcileResult.Freed)
		}
		if reconcileResult.Added > 0 {
			s.dlog.Printf("[%s] Reconciled: seeded %d agents from prior run (pool was under-counting)\n", timestamp, reconcileResult.Added)
		}

		s.checkDaemonSignals(timestamp)

		// Check verification pause BEFORE spawning
		if s.checkVerificationPause(timestamp) {
			continue
		}

		// Run all periodic maintenance tasks (reflection, cleanup, recovery, etc.)
		periodicResult := runPeriodicTasks(s.d, timestamp, daemonVerbose, s.logger)

		// Process completions
		completionResult := s.processDaemonCompletions(timestamp)

		// Run self-check invariants
		if s.checkInvariants(timestamp, completionResult) {
			continue
		}

		// Check beads circuit breaker — if beads is unhealthy, skip polling and back off.
		// This prevents the lock cascade where daemon keeps spawning bd processes
		// that pile up behind a stuck JSONL lock.
		if s.d.BeadsCircuitBreaker != nil && s.d.BeadsCircuitBreaker.IsOpen() {
			backoff := s.d.BeadsCircuitBreaker.BackoffDuration()
			failures := s.d.BeadsCircuitBreaker.ConsecutiveFailures()
			s.dlog.Printf("[%s] Beads circuit breaker open: %d consecutive failures, backing off %s\n",
				timestamp, failures, formatDaemonDuration(backoff))
			select {
			case <-s.ctx.Done():
				s.dlog.Printf("%s", s.stopMessage())
				return nil
			case <-time.After(backoff):
				continue
			}
		}

		// Get ready issues count for status (multi-project when registry available)
		readyIssues, readyErr := daemon.ListReadyIssuesMultiProject(s.d.ProjectRegistry)
		if readyErr != nil {
			if s.d.BeadsCircuitBreaker != nil {
				s.d.BeadsCircuitBreaker.RecordFailure()
			}
			s.dlog.Errorf("[%s] Failed to list ready issues: %v\n", timestamp, readyErr)
		} else {
			if s.d.BeadsCircuitBreaker != nil {
				s.d.BeadsCircuitBreaker.RecordSuccess()
			}
		}
		readyCount := s.d.CountSpawnable(readyIssues)

		// Work Graph: per-cycle Orient phase — detect duplicates, overlaps, and chains.
		// Computed fresh each cycle (no local state). Surfaces removal candidates as questions.
		if readyErr == nil && len(readyIssues) > 1 {
			s.runWorkGraphAnalysis(readyIssues, timestamp)
		}

		// Write daemon status file AFTER reconciliation and completions so counts are accurate
		s.writeDaemonStatusFile(readyCount, periodicResult)

		// Check capacity before polling
		if s.d.AtCapacity() {
			activeCount := s.d.ActiveCount()
			if daemonVerbose {
				s.dlog.Printf("[%s] At capacity (%d/%d agents active), waiting...\n",
					timestamp, activeCount, daemonMaxAgents)
			}

			// Stuck detection: all slots full with no activity for 10+ min
			stuckThreshold := 10 * time.Minute
			stuckCooldown := 30 * time.Minute
			if checkDaemonStuck(s.lastSpawn, s.lastCompletion, s.lastStuckNotification, stuckThreshold, stuckCooldown) {
				s.dlog.Printf("[%s] Daemon stuck: capacity full, no spawns or completions in %s\n",
					timestamp, stuckThreshold)
				if err := s.stuckNotifier.DaemonStuck(activeCount, daemonMaxAgents); err != nil {
					s.dlog.Errorf("Warning: stuck notification failed: %v\n", err)
				}
				s.lastStuckNotification = time.Now()
			}

			// Wait for poll interval before checking again
			select {
			case <-s.ctx.Done():
				s.dlog.Printf("%s", s.stopMessage())
				return nil
			case <-time.After(s.config.PollInterval):
				continue
			}
		}

		if daemonVerbose {
			s.dlog.Printf("[%s] Polling for issues...\n", timestamp)
		}

		// Run the inner spawn cycle
		cycleResult := s.runDaemonSpawnCycle(timestamp)
		if cycleResult.cancelled {
			s.dlog.Printf("%s", s.stopMessage())
			return nil
		}

		// If poll interval is 0, run once and exit
		if s.config.PollInterval == 0 {
			s.dlog.Printf("Run-once mode. Spawned %d, completed %d.\n", s.processed, s.completed)
			return nil
		}

		// Wait for next poll cycle
		if daemonVerbose {
			s.dlog.Printf("[%s] Spawned %d this cycle, waiting %s before next poll...\n",
				timestamp, cycleResult.spawned, formatDaemonDuration(s.config.PollInterval))
		}
		select {
		case <-s.ctx.Done():
			s.dlog.Printf("%s", s.stopMessage())
			return nil
		case <-time.After(s.config.PollInterval):
		}
	}
}

// notifyDashboardCompletion sends a fire-and-forget POST to the serve API
// to push completion events to connected dashboard clients in real-time.
// This eliminates dashboard polling latency for completion surfacing.
func notifyDashboardCompletion(beadsID, reason, escalation string) {
	go func() {
		event := CompletionEvent{
			BeadsID:    beadsID,
			Reason:     reason,
			Escalation: escalation,
			Source:     "daemon",
			Timestamp:  time.Now().Unix(),
		}
		data, err := json.Marshal(event)
		if err != nil {
			return
		}

		// Best-effort POST to serve API (may not be running)
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Post("http://localhost:3348/api/notify/completion", "application/json", bytes.NewReader(data))
		if err != nil {
			return // Serve not running or unreachable - that's OK
		}
		resp.Body.Close()
	}()
}
