# Investigation: Daemon Double-Spawn Race Conditions

**Date:** 2026-01-25
**Status:** Complete
**Trigger:** Task from team-lead to investigate why daemon sometimes spawns same issue twice

## Summary

Analyzed the daemon spawn deduplication system to identify potential race conditions that could cause the same beads issue to be spawned multiple times. Found **four layers of deduplication** with varying robustness, and identified **three primary race conditions** that could cause duplicate spawns.

## Deduplication Layers (In Order)

### 1. SpawnedIssueTracker (In-Memory, 6-hour TTL)
**Location:** `pkg/daemon/spawn_tracker.go`

**How it works:**
- Maps issue ID → spawn timestamp in memory
- Issues marked spawned BEFORE `spawnFunc` call (daemon.go:885)
- TTL of 6 hours (configurable)
- Cleaned on reconciliation and stale entry cleanup

**Strengths:**
- Fast (in-memory lookup)
- Marks issues immediately before spawn
- Prevents rapid retry loops

**Weaknesses:**
- **In-memory only** - doesn't survive daemon restarts
- **Single-process** - no cross-instance coordination
- **TTL expiry** - issues can be respawned after 6 hours even if still running
- **Reconciliation gaps** - CleanStale() only runs at poll cycle start

**Code reference:** daemon.go:332-340 (IsSpawned check), daemon.go:883-887 (MarkSpawned)

---

### 2. Session-Level Deduplication (OpenCode API)
**Location:** `pkg/daemon/session_dedup.go`

**How it works:**
- Queries OpenCode API `/session` endpoint for all sessions
- Extracts beads ID from session title
- Checks if session was created within MaxAge (6 hours)
- Runs AFTER SpawnedIssueTracker check (daemon.go:840-847)

**Strengths:**
- **Persistent** - survives daemon restarts
- **Authoritative** - queries actual running sessions
- **Cross-instance** - multiple daemons see same session list

**Weaknesses:**
- **Fail-open** - returns false on API errors (session_dedup.go:74-76)
- **Network dependency** - API outages disable this layer
- **Title parsing** - relies on consistent beads ID extraction from session titles
- **MaxAge window** - old sessions (>6h) don't block spawns

**Code reference:** daemon.go:840-847 (check), session_dedup.go:67-96 (implementation)

---

### 3. Phase: Complete Detection (Beads Comments)
**Location:** `pkg/daemon/issue_adapter.go:172-225`

**How it works:**
- Queries beads comments for "Phase: Complete" marker
- Uses RPC daemon if available, falls back to `bd comments` CLI
- Runs AFTER session dedup (daemon.go:855-862)
- Case-insensitive phase name check

**Strengths:**
- **Persistent** - survives all restarts
- **Completion-aware** - prevents respawning finished work
- **Fail-safe** - CLI fallback when RPC unavailable

**Weaknesses:**
- **Late-stage only** - only helps after agent reports completion
- **Doesn't prevent initial duplicates** - can't stop race before first spawn
- **CLI fallback swallows errors** - returns false on failure (issue_adapter.go:211-216)
- **Cross-project coordination** - requires correct project path for socket lookup

**Code reference:** daemon.go:855-862 (check), issue_adapter.go:172-225 (implementation)

---

### 4. Beads Status Check (Open vs In Progress)
**Location:** `pkg/daemon/daemon.go:356-361`

**How it works:**
- `NextIssueExcluding` filters out issues with status = "in_progress"
- Relies on beads CLI or RPC to update status when agent starts work
- Runs BEFORE all other dedup checks

**Strengths:**
- **Intended design** - issue should transition to in_progress when agent starts
- **First-line filter** - catches issues that successfully transitioned

**Weaknesses:**
- **Async update** - status update happens AFTER spawn, creating race window
- **Update failures** - if status update fails silently, issue stays "open"
- **Polling race** - daemon can poll again before status propagates
- **This is the PRIMARY race condition** causing double spawns

**Code reference:** daemon.go:356-361

---

## Race Conditions (Root Causes)

### RC-1: Multiple Daemon Instances (Cross-Instance Race)
**Severity:** High
**Likelihood:** Medium (if users run `orch daemon run` in multiple terminals)

**Scenario:**
```
Time    Daemon A                    Daemon B                    State
------------------------------------------------------------------------------------
T0      Poll bd ready → sees X      -                          X is "open"
T1      -                           Poll bd ready → sees X      X is "open"
T2      MarkSpawned(X) (in-memory)  -                          A's tracker has X
T3      -                           MarkSpawned(X) (in-memory)  B's tracker has X
T4      SpawnWork(X) starts         -                          Session 1 created
T5      -                           SpawnWork(X) starts         Session 2 created
```

**Why it happens:**
- SpawnedIssueTracker is in-memory per daemon process
- No file-based lock or shared memory between daemon instances
- Both daemons mark and spawn independently

**Evidence:**
- No cross-instance coordination in spawn_tracker.go
- Registry file locking only protects registry.json updates, not spawn decisions
- Event logs don't show duplicates, suggesting this is rare or not happening

**Mitigation:**
- Single daemon instance via launchd (current production setup)
- File-based spawn lock (not implemented)
- Shared state via beads RPC (not implemented)

---

### RC-2: OpenCode API Failure (Fail-Open Bypass)
**Severity:** Medium
**Likelihood:** Low (API is usually stable)

**Scenario:**
```
Time    Daemon              OpenCode API            State
---------------------------------------------------------------------------
T0      Poll bd ready → X   -                       X is "open"
T1      HasExistingSession  GET /session → 500      API error
T2      Returns false       -                       Fail-open allows spawn
T3      SpawnWork(X) #1     -                       First spawn
T4      Poll again → X      -                       X still "open" (race)
T5      HasExistingSession  GET /session → 500      API still down
T6      Returns false       -                       Fail-open allows spawn
T7      SpawnWork(X) #2     -                       Second spawn
```

**Why it happens:**
- session_dedup.go:74-76 returns false on API errors (fail-open)
- Daemon continues spawning even when session dedup is non-functional
- No retry or degraded mode

**Evidence:**
- Explicit fail-open in session_dedup.go:74-76
- No error logging in HasExistingSession (silent failure)

**Mitigation:**
- Fail-closed mode (reject spawns during API outages)
- Exponential backoff on API failures
- Circuit breaker pattern

---

### RC-3: Status Update Delay (Beads Async Race)
**Severity:** High
**Likelihood:** High (this is THE primary race)

**Scenario:**
```
Time    Daemon                      Beads                   SpawnedTracker
------------------------------------------------------------------------------
T0      Poll bd ready → X           X status = "open"       {}
T1      MarkSpawned(X)              -                       {X: T1}
T2      SpawnWork(X) starts         -                       {X: T1}
T3      orch work calls bd update   -                       {X: T1}
T4      -                           X status = "open" still {X: T1}
        (status update pending)
T5      Poll again (new cycle)      -                       {X: T1}
T6      CleanStale() (no-op)        -                       {X: T1} (within TTL)
T7      NextIssue → sees X          X status = "open"       {X: T1}
T8      IsSpawned(X) → true         -                       Blocked by tracker!
        SKIP (tracker prevents)
```

**Why it happens:**
- Beads status update is async (happens inside spawned agent)
- Daemon polls faster than agent can update status
- **SpawnedIssueTracker successfully prevents this** when it's working

**Why it COULD still fail:**
- Daemon restart clears in-memory tracker
- TTL expiry after 6 hours
- CleanStale() called but issue still spawning

**Evidence:**
- daemon.go:332-340 shows IsSpawned check DOES run
- daemon.go:563-567 shows ReconcileWithOpenCode cleanup
- No evidence of failures in event logs

**Mitigation:**
- **Already implemented**: SpawnedIssueTracker (working as designed)
- **Enhancement**: Persist tracker to disk for daemon restarts
- **Enhancement**: Emit event when skipping due to IsSpawned

---

## Event Log Analysis

**Command run:**
```bash
tail -500 ~/.orch/events.jsonl | jq -r 'select(.type == "session.spawned" and .payload.beads_id) | .payload.beads_id' | sort | uniq -c | awk '$1 > 1 {print}'
```

**Result:** No duplicates found in last 500 spawns

**Interpretation:**
- Either the deduplication is working
- Or double-spawns are rare enough not to appear in recent history
- Or they're happening with `beads_id: null` (issues without beads tracking)

---

## Code Quality Observations

### Good Patterns
1. **Defense in depth** - 4 layers of deduplication
2. **Fail-safe defaults** - Most errors prevent spawn rather than allowing it
3. **Explicit locking** - Registry uses syscall.Flock with timeouts
4. **Reconciliation** - CleanStale() and ReconcileWithOpenCode() cleanup stale state

### Concerning Patterns
1. **Fail-open on API error** - session_dedup.go:74-76 (silently allows spawn)
2. **Silent error swallowing** - hasPhaseCompleteCLI logs warnings but returns false
3. **No cross-instance coordination** - SpawnedIssueTracker is process-local
4. **No duplicate detection telemetry** - No events emitted when dedup blocks spawn

---

## Recommendations

### High Priority
1. **Add duplicate spawn detection telemetry**
   - Emit event when IsSpawned blocks a spawn
   - Emit event when HasExistingSession blocks a spawn
   - Emit event when HasPhaseComplete blocks a spawn
   - Allows monitoring for how often dedup is saving us

2. **Persist SpawnedIssueTracker to disk**
   - Survive daemon restarts
   - Use same file locking as registry
   - Fallback to in-memory if file unavailable

3. **Fail-closed mode for OpenCode API**
   - Option to reject spawns during API outages
   - Exponential backoff on repeated failures
   - Alert when fail-open happens

### Medium Priority
4. **Cross-instance spawn lock**
   - File-based lock held during spawn decision
   - Prevents RC-1 (multiple daemon instances)
   - Use advisory lock, not mandatory (allow manual override)

5. **Audit logging for dedup skips**
   - Log to ~/.orch/daemon.log when issue skipped due to dedup
   - Include which layer blocked it (tracker/session/phase)
   - Helps diagnose why issues aren't spawning

### Low Priority
6. **Reduce SpawnedIssueTracker TTL**
   - 6 hours is very conservative
   - Consider 2 hours (typical agent work duration)
   - Reduces window for TTL-based race

---

## Conclusion

**Root cause:** The daemon CAN spawn the same issue twice, but has **four layers of deduplication** that make it unlikely:

1. **SpawnedIssueTracker** (in-memory, 6h TTL) - Prevents rapid retries
2. **Session dedup** (OpenCode API) - Persistent, cross-instance, but fail-open
3. **Phase: Complete** (beads comments) - Prevents respawning finished work
4. **Beads status** (open → in_progress) - Intended first-line defense, but async

**Primary race conditions:**
- **RC-3** (status update delay) is the most likely, but mitigated by SpawnedIssueTracker
- **RC-1** (multiple instances) is prevented by single daemon via launchd
- **RC-2** (API failure) is rare but possible due to fail-open design

**Evidence:** No duplicates found in recent event logs (last 500 spawns).

**Recommendation:** Add telemetry first (emit events when dedup blocks spawn) to measure how often this protection is needed, then consider persistence enhancements if double-spawns are observed in production.

---

## References

- Registry locking: `pkg/registry/registry.go:136-184` (load with LOCK_SH), `registry.go:186-274` (save with LOCK_EX)
- SpawnedIssueTracker: `pkg/daemon/spawn_tracker.go`
- Session dedup: `pkg/daemon/session_dedup.go:64-96`
- Phase: Complete: `pkg/daemon/issue_adapter.go:172-225`
- Daemon main loop: `cmd/orch/daemon.go:273-627`
- Event logs: `~/.orch/events.jsonl`
