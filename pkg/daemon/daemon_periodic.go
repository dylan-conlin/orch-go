// daemon_periodic.go contains periodic maintenance operations:
// reflection, cleanup, recovery, dead session detection, and server recovery.
package daemon

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// ShouldRunReflection returns true if periodic reflection should run.
// This checks if reflection is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunReflection() bool {
	if !d.Config.ReflectEnabled || d.Config.ReflectInterval <= 0 {
		return false
	}
	// Run immediately if we've never run before
	if d.lastReflect.IsZero() {
		return true
	}
	return time.Since(d.lastReflect) >= d.Config.ReflectInterval
}

// RunPeriodicReflection runs the periodic reflection analysis if due.
// Returns the result if reflection was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicReflection() *ReflectResult {
	if !d.ShouldRunReflection() {
		return nil
	}

	result, err := d.reflectFunc(d.Config.ReflectCreateIssues)
	if err != nil {
		return &ReflectResult{
			Error:   err,
			Message: fmt.Sprintf("Reflection failed: %v", err),
		}
	}

	// Update last reflect time on success
	d.lastReflect = time.Now()

	return result
}

// LastReflectTime returns when reflection was last run.
// Returns zero time if reflection has never run.
func (d *Daemon) LastReflectTime() time.Time {
	return d.lastReflect
}

// NextReflectTime returns when the next reflection is scheduled.
// Returns zero time if reflection is disabled.
func (d *Daemon) NextReflectTime() time.Time {
	if !d.Config.ReflectEnabled || d.Config.ReflectInterval <= 0 {
		return time.Time{}
	}
	if d.lastReflect.IsZero() {
		return time.Now() // Due immediately
	}
	return d.lastReflect.Add(d.Config.ReflectInterval)
}

// ShouldRunCleanup returns true if periodic session cleanup should run.
// This checks if cleanup is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunCleanup() bool {
	if !d.Config.CleanupEnabled || d.Config.CleanupInterval <= 0 {
		return false
	}
	// Run immediately if we've never run before
	if d.lastCleanup.IsZero() {
		return true
	}
	return time.Since(d.lastCleanup) >= d.Config.CleanupInterval
}

// CleanupResult contains the result of a cleanup operation.
type CleanupResult struct {
	SessionsDeleted        int
	WorkspacesArchived     int
	InvestigationsArchived int
	Error                  error
	Message                string
}

// RunPeriodicCleanup runs the periodic cleanup operations if due.
// This includes: sessions, workspaces, and investigations (based on config).
// Returns the result if cleanup was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicCleanup() *CleanupResult {
	if !d.ShouldRunCleanup() {
		return nil
	}

	result := &CleanupResult{}
	var messages []string

	// Get project directory for workspace/investigation cleanup
	projectDir := getProjectDir()

	// Run session cleanup if enabled
	if d.Config.CleanupSessions {
		deleted, err := runSessionCleanup(d.Config.CleanupServerURL, d.Config.CleanupSessionsAgeDays, d.Config.CleanupPreserveOrchestrator)
		if err != nil {
			return &CleanupResult{
				Error:   err,
				Message: fmt.Sprintf("Session cleanup failed: %v", err),
			}
		}
		result.SessionsDeleted = deleted
		if deleted > 0 {
			messages = append(messages, fmt.Sprintf("%d sessions", deleted))
		}
	}

	// Run workspace cleanup if enabled
	if d.Config.CleanupWorkspaces && projectDir != "" {
		archived, err := runWorkspaceCleanup(projectDir, d.Config.CleanupWorkspacesAgeDays, d.Config.CleanupPreserveOrchestrator)
		if err != nil {
			return &CleanupResult{
				SessionsDeleted: result.SessionsDeleted,
				Error:           err,
				Message:         fmt.Sprintf("Workspace cleanup failed: %v", err),
			}
		}
		result.WorkspacesArchived = archived
		if archived > 0 {
			messages = append(messages, fmt.Sprintf("%d workspaces", archived))
		}
	}

	// Run investigation cleanup if enabled
	if d.Config.CleanupInvestigations && projectDir != "" {
		archived, err := runInvestigationCleanup(projectDir)
		if err != nil {
			return &CleanupResult{
				SessionsDeleted:    result.SessionsDeleted,
				WorkspacesArchived: result.WorkspacesArchived,
				Error:              err,
				Message:            fmt.Sprintf("Investigation cleanup failed: %v", err),
			}
		}
		result.InvestigationsArchived = archived
		if archived > 0 {
			messages = append(messages, fmt.Sprintf("%d investigations", archived))
		}
	}

	// Update last cleanup time on success
	d.lastCleanup = time.Now()

	// Build summary message
	if len(messages) == 0 {
		result.Message = "No stale items found"
	} else {
		result.Message = fmt.Sprintf("Cleaned: %s", strings.Join(messages, ", "))
	}

	return result
}

// LastCleanupTime returns when cleanup was last run.
// Returns zero time if cleanup has never run.
func (d *Daemon) LastCleanupTime() time.Time {
	return d.lastCleanup
}

// NextCleanupTime returns when the next cleanup is scheduled.
// Returns zero time if cleanup is disabled.
func (d *Daemon) NextCleanupTime() time.Time {
	if !d.Config.CleanupEnabled || d.Config.CleanupInterval <= 0 {
		return time.Time{}
	}
	if d.lastCleanup.IsZero() {
		return time.Now() // Due immediately
	}
	return d.lastCleanup.Add(d.Config.CleanupInterval)
}

// ShouldRunRecovery returns true if periodic recovery should run.
// This checks if recovery is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunRecovery() bool {
	if !d.Config.RecoveryEnabled || d.Config.RecoveryInterval <= 0 {
		return false
	}
	// Run immediately if we've never run before
	if d.lastRecovery.IsZero() {
		return true
	}
	return time.Since(d.lastRecovery) >= d.Config.RecoveryInterval
}

// RecoveryResult contains the result of a recovery operation.
type RecoveryResult struct {
	ResumedCount   int
	SkippedCount   int
	EscalatedCount int // Agents escalated to needs human decision
	AbandonedCount int // Agents auto-abandoned after timeout
	Error          error
	Message        string
}

// RunPeriodicRecovery runs the periodic stuck agent recovery if due.
// Returns the result if recovery was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicRecovery() *RecoveryResult {
	if !d.ShouldRunRecovery() {
		return nil
	}

	// Get list of active agents via live discovery
	agents, err := GetActiveAgents()
	if err != nil {
		return &RecoveryResult{
			ResumedCount:   0,
			SkippedCount:   0,
			EscalatedCount: 0,
			AbandonedCount: 0,
			Error:          err,
			Message:        fmt.Sprintf("Recovery failed to list agents: %v", err),
		}
	}

	resumed := 0
	skipped := 0
	escalated := 0
	abandoned := 0
	now := time.Now()

	for _, agent := range agents {
		// Skip agents without beads ID (can't resume without ID)
		if agent.BeadsID == "" {
			skipped++
			continue
		}

		// Skip agents that already reported Phase: Complete
		// (they're waiting for orchestrator review, not stuck)
		if strings.EqualFold(agent.Phase, "complete") {
			skipped++
			continue
		}

		// Check if agent is idle long enough to trigger recovery
		idleTime := now.Sub(agent.UpdatedAt)
		if idleTime < d.Config.RecoveryIdleThreshold {
			skipped++
			continue
		}

		// Auto-abandon: If agent has been dead for X hours with no progress, auto-abandon
		if d.Config.AutoAbandonAfterHours > 0 {
			abandonThreshold := time.Duration(d.Config.AutoAbandonAfterHours) * time.Hour
			if idleTime >= abandonThreshold {
				if d.Config.Verbose {
					fmt.Printf("  Auto-abandoning %s (dead for %v, threshold: %v)\n",
						agent.BeadsID, idleTime.Round(time.Minute), abandonThreshold)
				}
				// Close the issue with auto-abandon reason, using force to bypass
				// bd's Phase: Complete gate since abandoned agents won't have one
				reason := fmt.Sprintf("Auto-abandoned: No progress for %v (threshold: %v)",
					idleTime.Round(time.Minute), abandonThreshold)
				if err := verify.CloseIssueForce(agent.BeadsID, reason, true); err != nil {
					if d.Config.Verbose {
						fmt.Printf("  Failed to auto-abandon %s: %v\n", agent.BeadsID, err)
					}
				} else {
					abandoned++
					// Remove triage:ready label to prevent respawning (matches orch abandon behavior)
					if err := verify.RemoveTriageReadyLabel(agent.BeadsID); err != nil {
						if d.Config.Verbose {
							fmt.Printf("  Note: could not remove triage:ready label from %s: %v\n", agent.BeadsID, err)
						}
					}
					// Clear tracking for this agent
					delete(d.resumeAttempts, agent.BeadsID)
					delete(d.resumeAttemptCounts, agent.BeadsID)
				}
				continue
			}
		}

		// Escalation: If agent has failed resume N times, escalate to needs human decision
		attemptCount := d.resumeAttemptCounts[agent.BeadsID]
		if d.Config.MaxResumeAttempts > 0 && attemptCount >= d.Config.MaxResumeAttempts {
			if d.Config.Verbose {
				fmt.Printf("  Escalating %s to 'Needs Human Decision' (attempts: %d, threshold: %d)\n",
					agent.BeadsID, attemptCount, d.Config.MaxResumeAttempts)
			}
			// Add needs:human label for escalation
			if err := addNeedsHumanLabel(agent.BeadsID); err != nil {
				if d.Config.Verbose {
					fmt.Printf("  Failed to escalate %s: %v\n", agent.BeadsID, err)
				}
			} else {
				escalated++
				// Reset attempt count after escalation
				delete(d.resumeAttemptCounts, agent.BeadsID)
			}
			// Skip resume attempt after escalation
			skipped++
			continue
		}

		// Check if we've attempted resume recently (rate limiting)
		if lastAttempt, exists := d.resumeAttempts[agent.BeadsID]; exists {
			timeSinceLastAttempt := now.Sub(lastAttempt)
			if timeSinceLastAttempt < d.Config.RecoveryRateLimit {
				skipped++
				if d.Config.Verbose {
					fmt.Printf("  Skipping %s: resumed %v ago (rate limit: %v)\n",
						agent.BeadsID, timeSinceLastAttempt.Round(time.Minute), d.Config.RecoveryRateLimit)
				}
				continue
			}
		}

		// Attempt to resume the agent
		if d.Config.Verbose {
			fmt.Printf("  Attempting recovery for %s (idle for %v, attempt %d)\n",
				agent.BeadsID, idleTime.Round(time.Minute), attemptCount+1)
		}

		// Increment attempt count BEFORE attempting resume
		d.resumeAttemptCounts[agent.BeadsID] = attemptCount + 1

		if err := ResumeAgentByBeadsID(agent.BeadsID); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to resume %s: %v\n", agent.BeadsID, err)
			}
			// Record failed attempt time (for rate limiting)
			d.resumeAttempts[agent.BeadsID] = now
			skipped++
			continue
		}

		// Record successful resume attempt
		d.resumeAttempts[agent.BeadsID] = now
		resumed++

		if d.Config.Verbose {
			fmt.Printf("  Resumed %s successfully\n", agent.BeadsID)
		}
	}

	// Update last recovery time on success
	d.lastRecovery = time.Now()

	message := fmt.Sprintf("Recovery attempted: %d resumed, %d skipped", resumed, skipped)
	if escalated > 0 {
		message += fmt.Sprintf(", %d escalated", escalated)
	}
	if abandoned > 0 {
		message += fmt.Sprintf(", %d abandoned", abandoned)
	}

	return &RecoveryResult{
		ResumedCount:   resumed,
		SkippedCount:   skipped,
		EscalatedCount: escalated,
		AbandonedCount: abandoned,
		Error:          nil,
		Message:        message,
	}
}

// LastRecoveryTime returns when recovery was last run.
// Returns zero time if recovery has never run.
func (d *Daemon) LastRecoveryTime() time.Time {
	return d.lastRecovery
}

// NextRecoveryTime returns when the next recovery is scheduled.
// Returns zero time if recovery is disabled.
func (d *Daemon) NextRecoveryTime() time.Time {
	if !d.Config.RecoveryEnabled || d.Config.RecoveryInterval <= 0 {
		return time.Time{}
	}
	if d.lastRecovery.IsZero() {
		return time.Now() // Due immediately
	}
	return d.lastRecovery.Add(d.Config.RecoveryInterval)
}

// ShouldRunDeadSessionDetection returns true if dead session detection should run.
// This checks if detection is enabled and enough time has elapsed since the last run.
func (d *Daemon) ShouldRunDeadSessionDetection() bool {
	if !d.Config.DeadSessionDetectionEnabled || d.Config.DeadSessionDetectionInterval <= 0 {
		return false
	}
	// Run immediately if we've never run before
	if d.lastDeadSessionDetection.IsZero() {
		return true
	}
	return time.Since(d.lastDeadSessionDetection) >= d.Config.DeadSessionDetectionInterval
}

// DeadSessionDetectionResult contains the result of a dead session detection operation.
type DeadSessionDetectionResult struct {
	DetectedCount  int
	MarkedCount    int
	SkippedCount   int
	EscalatedCount int
	Error          error
	Message        string
}

// RunPeriodicDeadSessionDetection runs dead session detection if due.
// For each dead session, checks the retry count (number of prior DEAD SESSION comments).
// If the retry count exceeds the threshold, escalates to needs:human instead of resetting.
func (d *Daemon) RunPeriodicDeadSessionDetection() *DeadSessionDetectionResult {
	if !d.ShouldRunDeadSessionDetection() {
		return nil
	}

	config := DeadSessionDetectionConfig{
		Verbose:    d.Config.Verbose,
		MaxRetries: d.Config.MaxDeadSessionRetries,
	}

	deadSessions, err := FindDeadSessions(config)
	if err != nil {
		return &DeadSessionDetectionResult{
			Error:   err,
			Message: fmt.Sprintf("Dead session detection failed: %v", err),
		}
	}

	detected := len(deadSessions)
	marked := 0
	skipped := 0
	escalated := 0

	for _, dead := range deadSessions {
		retryCount, err := CountDeadSessionComments(dead.BeadsID)
		if err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to count retries for %s: %v (treating as 0)\n", dead.BeadsID, err)
			}
			retryCount = 0
		}

		maxRetries := config.maxRetries()

		if retryCount >= maxRetries {
			if d.Config.Verbose {
				fmt.Printf("  Escalating %s: %d dead sessions (threshold: %d)\n",
					dead.BeadsID, retryCount, maxRetries)
			}
			if err := EscalateDeadSession(dead.BeadsID, retryCount, dead.Reason); err != nil {
				if d.Config.Verbose {
					fmt.Printf("  Failed to escalate %s: %v\n", dead.BeadsID, err)
				}
				skipped++
				continue
			}
			escalated++
			continue
		}

		if d.Config.Verbose {
			fmt.Printf("  Marking %s as dead (attempt %d/%d)\n",
				dead.BeadsID, retryCount+1, maxRetries)
		}
		if err := MarkSessionAsDead(dead.BeadsID, dead.Reason); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to mark %s as dead: %v\n", dead.BeadsID, err)
			}
			skipped++
			continue
		}
		marked++
	}

	d.lastDeadSessionDetection = time.Now()

	message := fmt.Sprintf("Dead session detection: %d detected, %d marked, %d escalated, %d skipped",
		detected, marked, escalated, skipped)
	return &DeadSessionDetectionResult{
		DetectedCount:  detected,
		MarkedCount:    marked,
		EscalatedCount: escalated,
		SkippedCount:   skipped,
		Message:        message,
	}
}

// LastDeadSessionDetectionTime returns when dead session detection was last run.
// Returns zero time if detection has never run.
func (d *Daemon) LastDeadSessionDetectionTime() time.Time {
	return d.lastDeadSessionDetection
}

// NextDeadSessionDetectionTime returns when the next dead session detection is scheduled.
// Returns zero time if detection is disabled.
func (d *Daemon) NextDeadSessionDetectionTime() time.Time {
	if !d.Config.DeadSessionDetectionEnabled || d.Config.DeadSessionDetectionInterval <= 0 {
		return time.Time{}
	}
	if d.lastDeadSessionDetection.IsZero() {
		return time.Now() // Due immediately
	}
	return d.lastDeadSessionDetection.Add(d.Config.DeadSessionDetectionInterval)
}

// addNeedsHumanLabel adds the needs:human label to a beads issue.
// This label indicates that the agent requires human intervention.
// Uses the beads RPC client with auto-reconnect when available, falling back to CLI.
func addNeedsHumanLabel(beadsID string) error {
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()
		return client.AddLabel(beadsID, "needs:human")
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return nil
	}

	// Fallback to CLI
	return beads.FallbackAddLabel(beadsID, "needs:human")
}

// ShouldRunServerRecovery returns true if server restart recovery should run.
// This runs once after daemon startup, after the stabilization delay has passed.
func (d *Daemon) ShouldRunServerRecovery() bool {
	if d.Config.Verbose {
		fmt.Printf("[DEBUG] ShouldRunServerRecovery: ServerRecoveryEnabled=%v\n", d.Config.ServerRecoveryEnabled)
	}
	if !d.Config.ServerRecoveryEnabled {
		if d.Config.Verbose {
			fmt.Printf("[DEBUG] ShouldRunServerRecovery: returning false - ServerRecoveryEnabled is false\n")
		}
		return false
	}
	if d.serverRecoveryState == nil {
		if d.Config.Verbose {
			fmt.Printf("[DEBUG] ShouldRunServerRecovery: returning false - serverRecoveryState is nil\n")
		}
		return false
	}
	result := d.serverRecoveryState.ShouldRunServerRecovery(d.Config.ServerRecoveryStabilizationDelay)
	if d.Config.Verbose {
		fmt.Printf("[DEBUG] ShouldRunServerRecovery: stabilizationDelay=%v, result=%v\n",
			d.Config.ServerRecoveryStabilizationDelay, result)
	}
	return result
}

// RunServerRecovery runs server restart recovery if due.
// This detects orphaned sessions (sessions that exist on disk but aren't in OpenCode's
// in-memory state) and resumes them with recovery-specific context.
//
// Unlike RunPeriodicRecovery which handles individual stuck agents, this handles
// the bulk recovery scenario after a server restart where ALL in-memory sessions
// are lost simultaneously.
//
// Returns the result if recovery was run, or nil if it wasn't due.
func (d *Daemon) RunServerRecovery() *ServerRecoveryResult {
	if !d.ShouldRunServerRecovery() {
		return nil
	}

	// Mark that we've run recovery (regardless of outcome)
	d.serverRecoveryState.MarkRecoveryRun()

	serverURL := d.Config.CleanupServerURL
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}

	// Find orphaned sessions
	orphaned, err := FindOrphanedSessions(serverURL)
	if err != nil {
		return &ServerRecoveryResult{
			Error:   err,
			Message: fmt.Sprintf("Server recovery failed to find orphaned sessions: %v", err),
		}
	}

	if len(orphaned) == 0 {
		return &ServerRecoveryResult{
			OrphanedCount: 0,
			Message:       "Server recovery: no orphaned sessions found",
		}
	}

	// Resume orphaned sessions with staggered delay
	resumed := 0
	skipped := 0

	for i, orphan := range orphaned {
		// Check rate limit for this specific agent
		if d.serverRecoveryState.WasRecentlyRecovered(orphan.BeadsID, d.Config.ServerRecoveryRateLimit) {
			if d.Config.Verbose {
				fmt.Printf("  Skipping %s: already recovered recently (rate limit)\n", orphan.BeadsID)
			}
			skipped++
			continue
		}

		// Add delay between resumes (except for the first one)
		if i > 0 && d.Config.ServerRecoveryResumeDelay > 0 {
			time.Sleep(d.Config.ServerRecoveryResumeDelay)
		}

		if d.Config.Verbose {
			fmt.Printf("  Resuming orphaned session %s (phase=%s)\n", orphan.BeadsID, orphan.Phase)
		}

		if err := ResumeOrphanedAgent(orphan, serverURL); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to resume %s: %v\n", orphan.BeadsID, err)
			}
			// Still mark as attempted to avoid retry storm
			d.serverRecoveryState.MarkRecovered(orphan.BeadsID)
			skipped++
			continue
		}

		// Mark as successfully recovered
		d.serverRecoveryState.MarkRecovered(orphan.BeadsID)
		resumed++

		if d.Config.Verbose {
			fmt.Printf("  Resumed %s successfully\n", orphan.BeadsID)
		}
	}

	return &ServerRecoveryResult{
		ResumedCount:  resumed,
		SkippedCount:  skipped,
		OrphanedCount: len(orphaned),
		Message:       fmt.Sprintf("Server recovery: %d orphaned found, %d resumed, %d skipped", len(orphaned), resumed, skipped),
	}
}
