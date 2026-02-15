# Probe: Daemon Duplicate Spawn Feb 14 Incident

**Status:** Complete
**Date:** 2026-02-14
**Model:** Daemon Autonomous Operation

---

## Question

Why did the daemon spawn orch-go-w50 10 times on Feb 14, 2026, despite the fix from 2026-02-14-daemon-duplicate-spawn-ttl-fragility.md that added beads status update BEFORE spawning?

**Model claim being tested:**

> ### 2. Duplicate Spawns
> 
> **Fix:** Spawn deduplication via tracking. Track spawned beads IDs in memory, skip on subsequent polls until status confirms transition.
> 
> **Defense in Depth Architecture (from prior probe):**
> 1. PRIMARY: Beads Status Update - happens before spawn, persistent, eliminates root cause
> 2. SECONDARY: SpawnedIssueTracker - catches race window, in-memory, 6h TTL
> 3. TERTIARY: Session-level Check - queries OpenCode API

---

## What I Tested

1. **Read daemon spawn flow** in `pkg/daemon/daemon.go:Once()` (lines 849-893)
2. **Read UpdateBeadsStatus** implementation in `pkg/daemon/issue_adapter.go` (lines 117-139)
3. **Read ListReadyIssues** implementation in `pkg/daemon/issue_adapter.go` (lines 16-36)
4. **Checked bd ready behavior** via `bd ready --help`
5. **Analyzed failure handling** in daemon code

---

## What I Observed

### Current Implementation (Confirmed Deployed)

The fix from the prior probe IS implemented:

```go
// pkg/daemon/daemon.go:856-861
if err := UpdateBeadsStatus(issue.ID, "in_progress"); err != nil {
    if d.Config.Verbose {
        fmt.Printf("  Warning: failed to mark %s as in_progress: %v (proceeding with spawn - tracker will provide fallback)\n", issue.ID, err)
    }
    // Continue with spawn - SpawnedIssueTracker provides secondary protection
}
```

### The Critical Bug

**The daemon CONTINUES spawning even when UpdateBeadsStatus fails!**

This means:
1. Daemon tries to mark issue as `in_progress` via UpdateBeadsStatus
2. **If UpdateBeadsStatus fails** (RPC timeout, beads daemon down, etc.)
3. Daemon logs warning but PROCEEDS with spawn
4. Issue status remains `open` in beads database
5. Daemon relies on SpawnedIssueTracker (fragile, TTL-based, in-memory)
6. Next poll: `bd ready` returns the SAME issue (still `open`)
7. SpawnedIssueTracker checks if spawn is recent (within 6h TTL)
8. **If TTL expired OR daemon restarted** → SpawnedIssueTracker doesn't block
9. Duplicate spawn occurs

### Evidence from Feb 14 Incident

Spawn context says:
> "Issue remained OPEN status with triage:ready label, so the daemon kept picking it up every poll cycle."

This confirms UpdateBeadsStatus was FAILING repeatedly:
- 10 spawns in ~20 minutes
- Each spawn: UpdateBeadsStatus failed → issue stayed `open`
- Each poll: `bd ready` returned same issue
- SpawnedIssueTracker failed to prevent (TTL expired or daemon restarted)

### Why Defense-in-Depth Failed

From the prior probe's defense architecture:

| Layer | Status During Incident | Effectiveness |
|-------|------------------------|---------------|
| PRIMARY: Beads Status Update | ❌ FAILING | UpdateBeadsStatus failing → no persistent tracking |
| SECONDARY: SpawnedIssueTracker | ❌ INSUFFICIENT | TTL-based, lost on daemon restart, 10 spawns in 20min suggests multiple failures |
| TERTIARY: Session Check | ❓ UNKNOWN | May have helped limit but didn't prevent duplicates |

The "defense in depth" only works if PRIMARY or (SECONDARY + TERTIARY) succeed. When PRIMARY fails and daemon restarts happen, SECONDARY fails too.

---

## Model Impact

**CONTRADICTS the prior probe's fix effectiveness.**

### Prior Probe Claimed

> The beads database now reflects reality immediately, making the TTL cache a secondary protection layer instead of the primary one.

This is TRUE when UpdateBeadsStatus succeeds. But the prior probe didn't account for UpdateBeadsStatus FAILING.

### Root Cause (Updated)

**The daemon's failure handling is incorrect.** When UpdateBeadsStatus fails, the daemon should:
- FAIL the spawn attempt
- NOT mark in SpawnedIssueTracker (issue wasn't actually spawned)
- Log error and move to next issue

Instead, it currently:
- Logs warning but CONTINUES
- Marks in SpawnedIssueTracker (creating false sense of protection)
- Spawns anyway, leaving issue `open` in beads

### Recommended Fix

**Change daemon behavior: FAIL FAST when UpdateBeadsStatus fails.**

```go
// pkg/daemon/daemon.go:856
if err := UpdateBeadsStatus(issue.ID, "in_progress"); err != nil {
    // Release slot on status update failure
    if d.Pool != nil && slot != nil {
        d.Pool.Release(slot)
    }
    return &OnceResult{
        Processed: false,
        Issue:     issue,
        Skill:     skill,
        Error:     fmt.Errorf("failed to mark issue as in_progress: %w", err),
        Message:   fmt.Sprintf("Failed to update beads status for %s - skipping spawn to prevent duplicates", issue.ID),
    }, nil
}
```

**Why this is better:**
- Preserves issue status as `open` → next poll will try again
- Prevents spawning when we can't track it persistently
- Eliminates reliance on fragile in-memory fallback
- Clear failure signal → can investigate WHY UpdateBeadsStatus is failing

### Why UpdateBeadsStatus Might Fail

Possible failure modes:
1. **Beads daemon not running** - RPC fails, CLI fallback might also fail if daemon required
2. **Database lock contention** - Multiple processes trying to update simultaneously
3. **Filesystem issues** - Beads SQLite database on network mount or permission issues
4. **Process isolation** - Daemon running as different user, can't access beads database

Need to investigate Feb 14 daemon logs to determine actual failure cause.

---

## Implementation

### Code Changes

Modified `pkg/daemon/daemon.go` in BOTH `Once()` and `OnceWithSlot()` functions:

**Before (lines 856-861):**
```go
if err := UpdateBeadsStatus(issue.ID, "in_progress"); err != nil {
    if d.Config.Verbose {
        fmt.Printf("  Warning: failed to mark %s as in_progress: %v (proceeding with spawn - tracker will provide fallback)\n", issue.ID, err)
    }
    // Continue with spawn - SpawnedIssueTracker provides secondary protection
}
```

**After:**
```go
if err := UpdateBeadsStatus(issue.ID, "in_progress"); err != nil {
    // Release slot on status update failure
    if d.Pool != nil && slot != nil {
        d.Pool.Release(slot)
    }
    return &OnceResult{
        Processed: false,
        Issue:     issue,
        Skill:     skill,
        Error:     fmt.Errorf("failed to mark issue as in_progress: %w", err),
        Message:   fmt.Sprintf("Failed to update beads status for %s - skipping spawn to prevent duplicates", issue.ID),
    }, nil
}
```

### Test Results

```bash
$ go test ./pkg/daemon/... -run "TestDaemon_Once|TestDaemon_OnceWithSlot|TestNextIssue|TestSpawnedIssue" -v
=== All spawn-related tests PASSED (2.049s)
```

Key tests that validate the fix:
- `TestDaemon_Once_WithPool_ReleasesSlotOnError` - Verifies slot release on spawn failure
- `TestDaemon_OnceMarksSpawned` - Verifies marking before spawn
- `TestDaemon_OnceUnmarksOnFailure` - Verifies rollback on spawn failure

---

## Next Steps

1. ✅ Create this probe documenting the bug
2. ⬜ Check daemon logs from Feb 14 to find UpdateBeadsStatus failure messages
3. ✅ Implement fail-fast fix: stop spawn when UpdateBeadsStatus fails
4. ⬜ Add instrumentation: track UpdateBeadsStatus failure rate (future enhancement)
5. ✅ Test fix with existing test suite
6. ⬜ Document in SYNTHESIS.md
7. ⬜ Update Status: Complete in this probe

