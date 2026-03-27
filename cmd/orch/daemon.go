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

	// Main polling loop — structured as OODA: Sense → Orient → Decide → Act.
	// Each cycle feeds back into the next via the Act phase's spawn outcomes.
	for {
		select {
		case <-s.ctx.Done():
			s.dlog.Printf("%s", s.stopMessage())
			return nil
		default:
		}

		s.cycles++
		timestamp := time.Now().Format("15:04:05")

		// ── SENSE: gather signals from the environment ───────────────
		// Reconcile pool state, process completions, check health signals.

		reconcileResult := s.d.ReconcileWithOpenCode()
		if reconcileResult.Freed > 0 {
			s.dlog.Printf("[%s] Reconciled: %d agents completed (capacity freed)\n", timestamp, reconcileResult.Freed)
		}
		if reconcileResult.Added > 0 {
			s.dlog.Printf("[%s] Reconciled: %d new agents detected (external spawns or prior run)\n", timestamp, reconcileResult.Added)
		}

		s.checkDaemonSignals(timestamp)

		// Run periodic maintenance BEFORE verification pause check.
		// Cleanup, health monitoring, and orphan detection must run even when
		// the daemon is paused for verification — pause prevents new spawns,
		// not maintenance. Without this, stale tmux windows accumulate during
		// verification pause because cleanStaleTmuxWindows never executes.
		periodicResult := runPeriodicTasks(s.d, timestamp, daemonVerbose, s.logger)

		completionResult := s.processDaemonCompletions(timestamp)

		if s.checkInvariants(timestamp, completionResult) {
			continue
		}

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

		// ── ORIENT: analyze and contextualize ────────────────────────
		// Work graph dedup/overlap detection, spawnable count.

		readyCount := s.d.CountSpawnable(readyIssues)

		if readyErr == nil && len(readyIssues) > 1 {
			s.runWorkGraphAnalysis(readyIssues, timestamp)
		}

		// ── DECIDE: determine what to do this cycle ──────────────────
		// Write status, check capacity, decide whether to enter spawn cycle.

		s.writeDaemonStatusFile(readyCount, periodicResult)

		if s.d.AtCapacity() {
			activeCount := s.d.ActiveCount()
			if daemonVerbose {
				s.dlog.Printf("[%s] At capacity (%d/%d agents active), waiting...\n",
					timestamp, activeCount, daemonMaxAgents)
			}

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

			select {
			case <-s.ctx.Done():
				s.dlog.Printf("%s", s.stopMessage())
				return nil
			case <-time.After(s.config.PollInterval):
				continue
			}
		}

		// ── ACT: execute spawn cycle ─────────────────────────────────
		// Inner loop calls Daemon.OnceExcluding (which itself runs
		// Sense → Orient → Decide → Act per-issue via ooda.go).

		if daemonVerbose {
			s.dlog.Printf("[%s] Polling for issues...\n", timestamp)
		}

		cycleResult := s.runDaemonSpawnCycle(timestamp)
		if cycleResult.cancelled {
			s.dlog.Printf("%s", s.stopMessage())
			return nil
		}

		// ── FEEDBACK: Act results feed into next Sense cycle ─────────

		if s.config.PollInterval == 0 {
			s.dlog.Printf("Run-once mode. Spawned %d, completed %d.\n", s.processed, s.completed)
			return nil
		}

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
