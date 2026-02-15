# Design: VerificationTracker Wiring into Daemon Run Loop

**Date:** 2026-02-15
**Type:** Architecture Design
**Issue:** orch-go-ydzu
**Phase:** Complete

## Design Question

How should we wire the VerificationTracker into the daemon run loop to enforce the verifiability-first constraint (Constraint 3: Mechanical Enforcement from `.kb/decisions/2026-02-14-verifiability-first-hard-constraint.md`)?

The VerificationTracker was implemented by orch-go-7jl (commit 85a0e021) with working tests, but it's never called in the daemon's actual run loop. This creates the exact failure mode the verifiability-first decision was created to prevent: daemon spawning indefinitely without human verification (26-day spiral, 1163 commits, zero human involvement).

## Success Criteria

**Good answer looks like:**
1. Clear wiring points identified for all three VerificationTracker methods
2. Resume mechanism integrated into daemon loop
3. Status visibility exposed via daemon status API and dashboard
4. Testing strategy defined for integration validation

**Constraints:**
- Must not break existing daemon functionality
- Must provide clear feedback when paused (dashboard visibility)
- Must support manual resume without requiring orch complete
- Must be testable without requiring actual OpenCode sessions

**Scope:**
- **In scope:** Wiring VerificationTracker methods, resume command, status visibility
- **Out of scope:** Changing VerificationTracker implementation, modifying verification gates, dashboard UI changes (just expose data)

## Problem Framing

### The Three Integration Points

Based on code analysis in the probe, three methods need wiring:

1. **RecordCompletion()** - Call after daemon marks issue as ready-for-review
   - Location: `pkg/daemon/completion_processing.go:ProcessCompletion()`
   - Current line 261-264: Adds "daemon:ready-review" label
   - Wiring needed: Call `d.VerificationTracker.RecordCompletion()` after successful label add
   - Return value: `true` if threshold reached (daemon should pause)

2. **IsPaused()** - Check before daemon spawns new agents
   - Location: `pkg/daemon/daemon.go:OnceExcluding()`
   - Current line 752-777: Checks rate limit before fetching issues
   - Wiring needed: Check `d.VerificationTracker.IsPaused()` before proceeding with spawn
   - Return value: `true` if paused (skip spawn, return paused message)

3. **RecordHumanVerification()** - Call when `orch complete` runs
   - Location: `cmd/orch/complete_cmd.go`
   - Needs to be called after successful completion
   - Resets counter and unpauses daemon

### Additional Requirements

**Resume signal mechanism:**
- File-based signal already implemented: `~/.orch/daemon-resume.signal`
- Helper functions exist: `WriteResumeSignal()`, `CheckAndClearResumeSignal()`
- Need `orch daemon resume` command to write signal
- Need daemon loop to check signal periodically

**Status visibility:**
- VerificationTracker has `Status()` method returning VerificationStatus
- Need to expose in daemon status API
- Need to surface in dashboard health cards

## Exploration (Fork Navigation)

### Fork 1: Where to call RecordCompletion()?

**Decision:** Where exactly should we call RecordCompletion() in the auto-completion flow?

**Options:**
- **A**: After AddLabel() succeeds (line 264 in ProcessCompletion)
- **B**: After all processing complete (end of ProcessCompletion)
- **C**: In CompletionOnce() after ProcessCompletion returns

**Substrate says:**
- Principle: **Fail closed** - Record completion only after successful label add, not before
- Model: Completion Verification Architecture - verification happens before marking complete
- Decision: N/A (no prior decision on this)

**Recommendation:** **Option A** - After AddLabel() succeeds

**Reasoning:**
- Label add is the point of no return - issue is now marked for review
- If we record before label add, and label add fails, we've incremented counter without actually completing anything
- If we record at end of function, we might not reach it on error paths
- Fail-closed: Only record when we've actually marked something ready-for-review

**Trade-off accepted:** If label add succeeds but logging fails, we still increment counter. This is correct - the issue IS marked ready-for-review even if logging failed.

### Fork 2: Where to check IsPaused()?

**Decision:** Where in the spawn flow should we check if the daemon is paused?

**Options:**
- **A**: Early in OnceExcluding(), before rate limit check (line 754)
- **B**: After rate limit check, before fetching issues (line 767)
- **C**: After fetching issues, before spawning (line 778)

**Substrate says:**
- Principle: **Premise before solution** - Check if we SHOULD spawn before checking WHAT to spawn
- Model: Daemon Autonomous Operation - checks happen in priority order
- Decision: N/A

**Recommendation:** **Option A** - Early in OnceExcluding(), before rate limit check

**Reasoning:**
- Verification pause is a higher-priority constraint than rate limiting
- No point checking rate limit if we're paused anyway
- No point fetching issues if we're paused
- Consistent with existing pattern: rate limit checked before fetching issues

**Trade-off accepted:** Adds one extra check to the hot path, but it's a simple RLock read (fast).

### Fork 3: How to call RecordHumanVerification()?

**Decision:** How should orch complete call RecordHumanVerification()?

**Options:**
- **A**: Direct method call via daemon package
- **B**: Via daemon client (if daemon is running)
- **C**: Via file-based signal (write ~/.orch/verification-reset.signal)

**Substrate says:**
- Principle: **Coherence over patches** - Reuse existing patterns (daemon already has client)
- Model: Beads Integration Architecture - Uses RPC-first with CLI fallback
- Decision: N/A

**Recommendation:** **Option A** - Direct method call via daemon package

**Reasoning:**
- orch complete runs in same process space as daemon (when daemon is running)
- No need for IPC when we can call directly
- Simpler than file-based signaling
- Consistent with how other daemon state is accessed

**Unknown:** Does orch complete have access to the running daemon instance? Need to check if there's a global daemon reference or if we need to create one.

**Spike needed:** Check how orch complete currently accesses daemon state (if at all).

### Fork 4: Resume signal integration

**Decision:** How should the daemon check for resume signals?

**Options:**
- **A**: Check in main Run() loop alongside reflection/cleanup checks
- **B**: Check in OnceExcluding() before spawning
- **C**: Dedicated goroutine polling for signal file

**Substrate says:**
- Principle: **Evolve by distinction** - Resume signal is different from periodic tasks (reflection/cleanup)
- Model: Daemon Autonomous Operation - Periodic tasks have interval-based scheduling
- Decision: N/A

**Recommendation:** **Option A** - Check in main Run() loop

**Reasoning:**
- Resume is a state transition, not a periodic task
- Checking in Run() loop means it's checked every poll cycle (every 15s by default)
- Consistent with existing pattern for periodic tasks
- No need for dedicated goroutine (adds complexity)

**Trade-off accepted:** Resume detection has up to 15s latency. This is acceptable - resume is a manual operation, not time-critical.

### Fork 5: orch daemon resume command

**Decision:** Should we add `orch daemon resume` command or reuse existing mechanism?

**Options:**
- **A**: New command: `orch daemon resume`
- **B**: Reuse existing: `orch complete` auto-calls RecordHumanVerification
- **C**: Manual signal: Dylan writes signal file directly

**Substrate says:**
- Principle: **User mental model** - Separate actions for separate intents
- Model: N/A
- Decision: N/A

**Recommendation:** **Option A** - New command: `orch daemon resume`

**Reasoning:**
- Explicit resume command matches user mental model: "I want to release more work"
- orch complete is about finishing specific work, not releasing the daemon
- Resume might be needed without orch complete (e.g., after reviewing dashboard)
- Command is more discoverable than manual signal file writing

**Trade-off accepted:** Adds one more command to maintain. But it's a simple command (just calls WriteResumeSignal()).

## Synthesis (Design Decisions)

Based on fork navigation, here's the recommended design:

### 1. RecordCompletion() wiring

**Location:** `pkg/daemon/completion_processing.go:ProcessCompletion()`

```go
// After line 264 (after AddLabel succeeds):
if !config.DryRun {
    if err := verify.AddLabel(agent.BeadsID, "daemon:ready-review"); err != nil {
        result.Error = fmt.Errorf("failed to mark ready for review: %w", err)
        return result
    }
    
    // NEW: Record auto-completion for verification tracking
    if d.VerificationTracker != nil {
        shouldPause := d.VerificationTracker.RecordCompletion()
        if shouldPause && config.Verbose {
            status := d.VerificationTracker.Status()
            fmt.Printf("    Verification pause triggered: %d/%d auto-completions\n",
                status.CompletionsSinceVerification, status.Threshold)
        }
    }
}
```

### 2. IsPaused() check

**Location:** `pkg/daemon/daemon.go:OnceExcluding()` - Early, before rate limit check

```go
// After line 752 (at start of OnceExcluding):
func (d *Daemon) OnceExcluding(skip map[string]bool) (*OnceResult, error) {
    // NEW: Check verification pause BEFORE any other checks
    if d.VerificationTracker != nil && d.VerificationTracker.IsPaused() {
        status := d.VerificationTracker.Status()
        return &OnceResult{
            Processed: false,
            Message: fmt.Sprintf("Paused for human verification (%d/%d auto-completions). Resume with: orch daemon resume",
                status.CompletionsSinceVerification, status.Threshold),
        }, nil
    }
    
    // Existing rate limit check follows...
    if d.RateLimiter != nil {
        // ...
    }
}
```

### 3. RecordHumanVerification() call

**Spike finding:** Need to check how orch complete accesses daemon state.

**Initial approach:** Add to complete command after successful verification:

**Location:** `cmd/orch/complete_cmd.go` - After successful completion

```go
// After closing beads issue successfully:
// NEW: Record human verification to reset daemon pause
if err := recordHumanVerification(); err != nil {
    // Log but don't fail completion
    if verbose {
        fmt.Printf("Warning: failed to record human verification: %v\n", err)
    }
}

// Helper function:
func recordHumanVerification() error {
    // Option 1: Via daemon package (if accessible)
    // Option 2: Via file signal (like resume)
    // TBD based on spike findings
}
```

**Unknown resolved via spike:** `grep -n "daemon" cmd/orch/complete_cmd.go` shows no daemon package usage. Will use file-based signal (consistent with resume mechanism).

**Revised approach:** Use file-based signal for RecordHumanVerification():

```go
// In complete_cmd.go, after successful completion:
if err := daemon.WriteVerificationSignal(); err != nil {
    // Log but don't fail completion
    if verbose {
        fmt.Printf("Warning: failed to write verification signal: %v\n", err)
    }
}
```

Need to add `WriteVerificationSignal()` similar to `WriteResumeSignal()` in verification_tracker.go.

### 4. Resume signal integration

**Location:** `cmd/orch/daemon.go` - Main Run() loop

```go
// In Run() loop, check for resume signal each iteration:
for {
    // NEW: Check for resume signal
    if d.VerificationTracker != nil {
        if resumed, err := daemon.CheckAndClearResumeSignal(); err != nil {
            fmt.Printf("Error checking resume signal: %v\n", err)
        } else if resumed {
            d.VerificationTracker.Resume()
            fmt.Println("Daemon resumed by signal")
        }
    }
    
    // Existing once() call follows...
    result, err := d.once()
    // ...
}
```

### 5. orch daemon resume command

**Location:** `cmd/orch/daemon.go` - New subcommand

```go
var daemonResumeCmd = &cobra.Command{
    Use:   "resume",
    Short: "Resume daemon after verification pause",
    Long: `Resume the daemon after it has paused for human verification.

The daemon pauses after N auto-completions without human review. Use this
command to release the daemon after reviewing completed work.

Example:
  orch daemon resume
`,
    RunE: func(cmd *cobra.Command, args []string) error {
        if err := daemon.WriteResumeSignal(); err != nil {
            return fmt.Errorf("failed to write resume signal: %w", err)
        }
        fmt.Println("Resume signal sent to daemon")
        return nil
    },
}

// Add to daemon command:
func init() {
    daemonCmd.AddCommand(daemonResumeCmd)
}
```

### 6. Status visibility

**Location:** `pkg/daemon/status.go` - Add to HealthStatus

```go
// Add to HealthStatus struct:
type HealthStatus struct {
    // ... existing fields ...
    
    // NEW: Verification tracking status
    VerificationStatus VerificationStatus `json:"verification_status"`
}

// In GetHealthStatus():
func GetHealthStatus() (*HealthStatus, error) {
    // ... existing code ...
    
    // NEW: Add verification status
    status.VerificationStatus = d.VerificationTracker.Status()
    
    return status, nil
}
```

**Dashboard exposure:** The status API already returns HealthStatus as JSON. No dashboard code changes needed - it will automatically pick up the new field.

## Testing Strategy

### Unit Tests

**Already exist:** `pkg/daemon/verification_tracker_test.go` covers:
- RecordCompletion() behavior
- IsPaused() behavior
- RecordHumanVerification() reset
- Resume signal file operations

**Need to add:** Integration tests for wiring:

1. **Test: RecordCompletion wiring**
   ```go
   func TestDaemon_ProcessCompletion_RecordsCompletion(t *testing.T) {
       // Setup daemon with mock tracker
       // Process completion successfully
       // Verify RecordCompletion() was called
       // Verify counter incremented
   }
   ```

2. **Test: IsPaused check**
   ```go
   func TestDaemon_OnceExcluding_RespectsPause(t *testing.T) {
       // Setup daemon with paused tracker
       // Call OnceExcluding()
       // Verify no spawn occurred
       // Verify paused message returned
   }
   ```

3. **Test: Resume signal integration**
   ```go
   func TestDaemon_Run_DetectsResumeSignal(t *testing.T) {
       // Start daemon in paused state
       // Write resume signal file
       // Verify daemon detects signal
       // Verify tracker.Resume() called
       // Verify daemon unpaused
   }
   ```

### Integration Test

**Full flow test:**
```go
func TestVerificationTracker_FullFlow(t *testing.T) {
    // 1. Daemon spawns and auto-completes 3 issues
    // 2. Verify daemon paused after 3rd completion
    // 3. Verify spawn attempt returns paused message
    // 4. Call RecordHumanVerification()
    // 5. Verify daemon unpaused
    // 6. Verify spawn works again
}
```

## File Targets

Files to create:
- `cmd/orch/daemon_resume.go` - New resume subcommand

Files to modify:
- `pkg/daemon/completion_processing.go` - Add RecordCompletion() call
- `pkg/daemon/daemon.go` - Add IsPaused() check in OnceExcluding(), resume signal check in Run()
- `cmd/orch/complete_cmd.go` - Add RecordHumanVerification() call
- `pkg/daemon/status.go` - Add VerificationStatus to HealthStatus
- `pkg/daemon/completion_processing_test.go` - Add wiring tests (or create new test file)

## Out of Scope

- Changing VerificationTracker threshold defaults
- Modifying verification gate logic
- Dashboard UI changes (status is exposed, UI changes happen separately)
- Changing how auto-completion works (just adding tracking)

## Acceptance Criteria

**Testable conditions for done:**

1. ✅ Daemon increments completion counter after each auto-completion
2. ✅ Daemon pauses after N auto-completions (default 3)
3. ✅ Paused daemon returns clear message explaining how to resume
4. ✅ `orch daemon resume` command unpauses daemon
5. ✅ `orch complete` resets counter and unpauses daemon
6. ✅ Verification status visible in daemon health API
7. ✅ All integration tests pass

**Manual verification:**
1. Run daemon, let it auto-complete 3 issues
2. Verify daemon shows paused status in dashboard
3. Run `orch daemon resume`
4. Verify daemon resumes spawning

## Trade-offs Considered

**Alternative: Event-driven approach**
- Emit events for auto-completion, subscribe to reset on orch complete
- **Rejected:** Adds complexity, direct method calls are simpler

**Alternative: Database-backed counter**
- Store counter in beads database or separate state file
- **Rejected:** In-memory is sufficient, daemon restart resets counter (acceptable)

**Alternative: Pause via configuration**
- Disable spawning via config flag instead of tracker
- **Rejected:** Doesn't track why paused, no automatic reset on human verification

## Dependencies/Blockers

**Blocker:** Need to spike how orch complete accesses daemon state
- If no access, need file-based signal like resume
- If has access, can call RecordHumanVerification() directly

**Recommendation:** Use file-based signal for consistency with resume mechanism. Simpler than trying to access daemon instance from complete command.

## Implementation Notes

**Phase 1: Core wiring**
1. Add RecordCompletion() call in ProcessCompletion()
2. Add IsPaused() check in OnceExcluding()
3. Add resume signal check in Run() loop

**Phase 2: Commands**
4. Implement `orch daemon resume` command
5. Add RecordHumanVerification() call in complete command (via signal)

**Phase 3: Visibility**
6. Add VerificationStatus to health API
7. Verify dashboard picks up status

**Phase 4: Testing**
8. Add integration tests
9. Manual validation

## Decision Gate Guidance

**This investigation should be promoted to a decision when:**
- Dylan accepts the wiring design
- Implementation is complete and tested
- The solution prevents the 26-day spiral failure mode

**Add blocks: frontmatter when promoting:**
```yaml
blocks:
  - verificationtracker
  - daemon pause
  - autonomous spawning
  - human verification
```

These keywords help future spawns find this decision when working on related daemon behavior.

**Phase:** Active
