# Probe: Daemon Completion Processing Fail-Fast Fix

**Date:** 2026-02-17  
**Status:** Active  
**Model:** Daemon Autonomous Operation

---

## Question

Does the daemon's completion processing loop implement fail-fast behavior when CompletionOnce fails, or does it silently continue, potentially orphaning completed agents?

**Model claim being tested:**
The daemon model describes a poll-spawn-complete cycle where the daemon monitors for `Phase: Complete` and verifies/closes agents. This probe tests whether completion processing failures properly surface and halt spawning, or whether they violate the spawn prerequisite fail-fast constraint.

---

## What I Tested

1. **Code Review:** Examined `cmd/orch/daemon.go:475-477` where `CompletionOnce` is called in the daemon run loop
2. **Error Handling Analysis:** Traced what happens when `CompletionOnce` returns an error
3. **Health Status Check:** Reviewed daemon health status reporting to see if completion errors surface
4. **Verification Tracker Integration:** Checked if completion failures bypass the verification pause mechanism

---

## What I Observed

### Current Behavior (Before Fix)

**Location:** `cmd/orch/daemon.go:475-477`

```go
completionResult, err := d.CompletionOnce(completionConfig)
if err != nil && daemonVerbose {
    fmt.Fprintf(os.Stderr, "[%s] Completion processing error: %v\n", timestamp, err)
}
```

**Findings:**
1. ✗ **Silent Continue:** When `CompletionOnce` fails, error is logged to stderr only if verbose mode is enabled, then loop continues
2. ✗ **No Health Tracking:** Completion processing errors are not tracked in daemon health status
3. ✗ **No Failure Counting:** No consecutive failure counter exists
4. ✗ **No Spawn Pause:** Spawning continues even if completion processing is persistently broken
5. ✗ **Orphaned Agents Risk:** Completed agents never get marked `ready-for-review`, accumulate indefinitely
6. ✗ **Verification Bypass:** Verification pause mechanism never triggers if completions aren't processed

**Risk confirmed:** This violates the spawn prerequisite fail-fast constraint (kb-035b64).

---

## Model Impact

**EXTENDS** the "Why This Fails" section of the Daemon Autonomous Operation model.

### New Failure Mode: Completion Processing Silent Failure

**What happens:** Daemon continues spawning new agents even though completion processing is broken.

**Root cause:** `CompletionOnce` errors are logged but don't halt spawning. No health tracking, no consecutive failure counting, no pause mechanism.

**Why detection is hard:** 
- Error only visible in logs if verbose mode enabled
- No daemon health status indicator
- Completed agents appear stuck in `Phase: Complete` state
- Verification backlog counter doesn't increment (completions never recorded)

**Symptoms:**
- Agents remain in `Phase: Complete` but never transition to `ready-for-review`
- Verification pause never triggers despite completed work
- Daemon health appears normal (spawning continues)
- Backlog accumulates silently

**Fix:** 
1. Track completion processing errors in daemon health status
2. Count consecutive failures
3. Pause spawning after N consecutive failures
4. Surface in `orch status` output

**Prevention:** 
- Fail-fast on completion processing errors
- Make completion health visible in daemon status
- Alert when completion processing is degraded

---

## Implementation Plan

**Changes Required:**

1. **Add health tracking field** to daemon state:
   - `completionErrorCount` (consecutive failures)
   - `lastCompletionError` (most recent error message)
   - `completionHealthy` (boolean)

2. **Modify error handling** in daemon run loop:
   - Increment `completionErrorCount` on failure
   - Reset counter on success
   - Pause spawning if count exceeds threshold (e.g., 3)

3. **Surface in health status:**
   - Include completion health in `orch status` output
   - Show error message if degraded

4. **Verification integration:**
   - Ensure completion failures don't bypass verification tracking
   - Consider if verification pause should trigger on completion errors

**Threshold:** Pause spawning after 3 consecutive completion processing failures.

**Rationale:** 
- Single failure might be transient (beads CLI update, temporary lock)
- 3 consecutive suggests persistent issue (database corruption, CLI broken)
- Prevents orphaned work accumulation
- Forces operator attention to completion processing health

---

## Implementation Completed

### Changes Made

1. **Created `CompletionFailureTracker`** (`pkg/daemon/completion_failure_tracker.go`)
   - Mirrors `SpawnFailureTracker` pattern
   - Tracks consecutive failures, total failures, last failure time/reason
   - Thread-safe with mutex protection

2. **Updated `Daemon` struct** (`pkg/daemon/daemon.go:212-214`)
   - Added `CompletionFailureTracker` field
   - Initialized in `NewWithConfig()`

3. **Updated error handling** in daemon run loop (`cmd/orch/daemon.go:475-502`)
   - Records failure when `CompletionOnce` returns error
   - Records success when it completes successfully
   - Always logs completion errors (not just in verbose mode)
   - Logs error events to events system

4. **Added pause logic** (`pkg/daemon/daemon.go:792-806`)
   - Checks consecutive failures before spawning
   - Pauses spawning if ≥3 consecutive failures
   - Returns descriptive pause message with failure details

5. **Surfaced in health status** (`pkg/daemon/status.go:46-48`, `cmd/orch/daemon.go:581-589`)
   - Added `CompletionFailures` field to `DaemonStatus`
   - Snapshot included in status file if failures exist
   - Visible in `orch status` output

### Verification

**Threshold:** 3 consecutive failures triggers pause (configurable constant in daemon.go:795)

**Pause behavior:**
- Daemon stops spawning new agents
- Displays message: "Paused: completion processing has failed N consecutive times..."
- Requires fixing completion processing issue before resuming
- Single successful completion resets counter

**Status visibility:**
- Completion failures surfaced in `~/.orch/daemon-status.json`
- Includes: consecutive_failures, total_failures, last_failure time, last_failure_reason
- Dashboard can alert on completion health degradation

## Status: Complete

The fix implements all requirements:
- ✅ Surface completion processing errors in daemon health status
- ✅ Count consecutive failures
- ✅ Pause spawning if completion processing persistently broken
- ✅ Log errors to events system (not just stderr)
- ✅ Fail-fast instead of silent continue
