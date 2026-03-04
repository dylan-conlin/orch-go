## Summary (D.E.K.N.)

**Delta:** The daemon code is well-architected with thorough deduplication layers and safety mechanisms, but contains 7 concrete defects including a constructor divergence bug, race condition in lazy initialization, direct function call bypassing an interface, unbounded map growth, and dead code.

**Evidence:** Full audit of cmd/orch/daemon.go (1371 lines), cmd/orch/daemon_periodic.go (287 lines), pkg/daemon/daemon.go (892 lines), and 25+ supporting files in pkg/daemon/. Findings verified against source code with line numbers.

**Knowledge:** The daemon's complexity is intentional and well-documented (prior investigations/probes explain every dedup layer). The remaining defects are latent (most won't trigger in practice) but the constructor divergence is a real correctness issue.

**Next:** Create fix issues for HIGH severity findings (constructor divergence, recovery interface bypass, lazy init race). The rest are LOW priority.

**Authority:** architectural - Fixes span constructor patterns, interface contracts, and multiple daemon subsystems

---

# Investigation: Audit Daemon Code Health

**Question:** What code health issues, bugs, latent defects, and maintainability concerns exist in the daemon codebase?

**Started:** 2026-03-04
**Updated:** 2026-03-04
**Owner:** Agent (orch-go-ei9r)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Constructor Divergence Between NewWithConfig and NewWithPool

**Severity: HIGH - Correctness Bug**

**Evidence:** `NewWithPool()` (pkg/daemon/daemon.go:199-227) is missing several fields that `NewWithConfig()` (pkg/daemon/daemon.go:159-195) initializes:
- Missing: `VerificationTracker`
- Missing: `CompletionFailureTracker`
- Missing: `VerificationRetryTracker`
- Missing: `BeadsCircuitBreaker`

Any daemon created via `NewWithPool()` will have nil trackers. When the daemon loop checks `d.VerificationTracker != nil` or `d.CompletionFailureTracker != nil`, these checks will silently skip verification enforcement and failure tracking. This means a daemon created via `NewWithPool()` runs without the verifiability-first safety constraint.

**Source:** pkg/daemon/daemon.go:199-227 vs pkg/daemon/daemon.go:159-195

**Significance:** Defect Class 6 (Duplicate Action) - two constructors that should produce equivalent objects but diverge. If `NewWithPool` is used in production or tests, verification enforcement is silently disabled.

---

### Finding 2: RunPeriodicRecovery Bypasses Injected AgentDiscoverer Interface

**Severity: HIGH - Interface Contract Violation**

**Evidence:** `RunPeriodicRecovery()` at pkg/daemon/periodic.go:173 calls `GetActiveAgents()` (the package-level function) directly instead of using `d.Agents.GetActiveAgents()` like every other periodic method does (orphan detection at orphan_detector.go:82, phase timeout at phase_timeout.go:86, question detection at question_detector.go:76).

This means:
1. In production: Works identically (default implementation calls the same function)
2. In tests: The mock `AgentDiscoverer` is bypassed for recovery — tests can't control what agents the recovery subsystem sees
3. Pattern violation: Every other periodic method follows `agentDiscoverer := d.Agents; if nil { use default }` but recovery directly calls the package function

**Source:** pkg/daemon/periodic.go:173 vs pkg/daemon/orphan_detector.go:77-80

**Significance:** Makes recovery untestable via the standard mock pattern. A test that sets `d.Agents = mockDiscoverer` would control orphan detection, phase timeout, and question detection but NOT recovery.

---

### Finding 3: Race Condition in initDefaultSessionDedupChecker

**Severity: MEDIUM - Latent Race**

**Evidence:** The lazy singleton initialization at pkg/daemon/session_dedup.go:121-126 has a classic check-then-act race:

```go
func initDefaultSessionDedupChecker() *SessionDedupChecker {
    if defaultSessionDedupChecker == nil {
        defaultSessionDedupChecker = NewSessionDedupChecker(DefaultSessionDedupConfig())
    }
    return defaultSessionDedupChecker
}
```

The package-level `defaultSessionDedupChecker` variable (line 118) is read and written without synchronization. If multiple goroutines call `HasExistingSessionForBeadsID()` concurrently, two checkers could be created.

**Source:** pkg/daemon/session_dedup.go:118-126

**Significance:** In practice, the daemon loop is single-threaded, so this race doesn't trigger during normal operation. But it's a data race detectable by `go test -race` if tests parallelize this path. The fix is trivial: use `sync.Once`.

---

### Finding 4: OnceWithSlot Lacks Several Safety Checks Present in OnceExcluding

**Severity: MEDIUM - Asymmetric Code Paths**

**Evidence:** `OnceWithSlot()` (pkg/daemon/daemon.go:588-624) is a separate code path from `OnceExcluding()` (daemon.go:444-583) that shares `spawnIssue()` but misses several pre-spawn checks:

1. **No verification pause check** — `OnceExcluding` checks `VerificationTracker.IsPaused()` at line 448; `OnceWithSlot` skips it entirely
2. **No completion failure check** — `OnceExcluding` checks `CompletionFailureTracker` at line 461; `OnceWithSlot` skips it
3. **No skip set support** — `OnceWithSlot` calls `d.NextIssue()` (no skip), while the main loop uses `OnceExcluding(skip)`
4. **No extraction/architect escalation** — `OnceWithSlot` goes straight to `spawnIssue()` without hotspot or architect checks

`OnceWithSlot` appears to be an older API that wasn't updated as safety checks were added to `OnceExcluding`.

**Source:** pkg/daemon/daemon.go:588-624 vs 444-583

**Significance:** If `OnceWithSlot` is called (it's exported and available), it bypasses all the safety mechanisms that were added to `OnceExcluding` over time. The main `runDaemonLoop` uses `OnceExcluding`, so this is only an issue for external callers or future code that uses `OnceWithSlot`.

---

### Finding 5: resumeAttempts Map Grows Unbounded

**Severity: LOW - Latent Resource Leak**

**Evidence:** The `d.resumeAttempts` map (pkg/daemon/daemon.go:103-104) records agent resume timestamps but is never cleaned. New entries are added at periodic.go:232/238 for every recovery attempt, but no code removes entries for agents that have been closed, completed, or become irrelevant.

Over long daemon runs (days/weeks), this map accumulates entries for every agent that was ever attempted for recovery. Each entry is small (~80 bytes: string key + time.Time value), so this won't cause OOM, but it violates the "no local state" architectural constraint by accumulating unbounded state.

**Source:** pkg/daemon/daemon.go:103-104, pkg/daemon/periodic.go:209-238

**Significance:** Practically harmless (map would need thousands of entries to matter), but represents a pattern violation. A simple TTL cleanup similar to `SpawnedIssueTracker.CleanStale()` would fix it.

---

### Finding 6: Dead Code — daemon_test.go.bak (2832 lines)

**Severity: LOW - Dead Code**

**Evidence:** File `pkg/daemon/daemon_test.go.bak` is a 2832-line backup of old tests. It's untracked by git (.gitignore or similar) and not compiled, but it clutters the directory listing and may confuse future developers or tools.

**Source:** pkg/daemon/daemon_test.go.bak

**Significance:** Stale Artifact Accumulation (Defect Class 3). Should be deleted — the content is preserved in git history.

---

### Finding 7: BeadsCircuitBreaker.Status() Calls BackoffDurationLocked() While Holding Lock

**Severity: LOW - Correct but Fragile**

**Evidence:** `BeadsCircuitBreaker.Status()` at beads_circuit_breaker.go:96-105 holds the mutex and calls `BackoffDurationLocked()`. The `BackoffDurationLocked()` method exists specifically for this purpose (it's the internal version without mutex). However, the public `BackoffDuration()` at line 76-93 and `BackoffDurationLocked()` at line 108-121 have identical logic duplicated. If one is updated without the other, they'll diverge.

**Source:** pkg/daemon/beads_circuit_breaker.go:76-121

**Significance:** Code duplication creating a maintenance risk. The standard pattern is to have the locked version contain the logic and the public version acquire lock + delegate. Here it's inverted — both contain the logic.

---

## Synthesis

**Key Insights:**

1. **Defense-in-depth is well-implemented** — The daemon has 6 deduplication layers for spawn prevention (spawn tracker, session dedup, content dedup, fresh status check, beads status update, pool slot). Each layer has clear documentation explaining why it exists and what incident it prevents. This is good engineering.

2. **Asymmetric code paths are the primary risk** — Findings 1 and 4 both follow the same pattern: an alternate code path (NewWithPool, OnceWithSlot) that was created early and not updated as safety mechanisms were added to the primary path. This is the most likely source of future production bugs.

3. **The daemon is single-threaded by design** — While the daemon has concurrent-safe primitives (sync.Mutex in trackers, pool, circuit breaker), the main daemon loop is single-threaded. This makes most concurrency concerns theoretical rather than practical (Finding 3).

**Answer to Investigation Question:**

The daemon codebase is in good health overall. It has extensive test coverage (25+ test files), well-documented design decisions (referenced incident IDs in comments), and a clean interface-based architecture. The 7 findings are split between 2 HIGH severity (constructor divergence, interface bypass), 2 MEDIUM (race condition, asymmetric paths), and 3 LOW (unbounded map, dead code, code duplication). None are actively causing production issues, but the constructor divergence (Finding 1) is a correctness bug that could silently disable safety mechanisms.

---

## Structured Uncertainty

**What's tested:**

- ✅ Constructor divergence confirmed by diff of NewWithConfig vs NewWithPool field lists
- ✅ RunPeriodicRecovery bypass confirmed by comparing all 4 periodic methods' agent discovery patterns
- ✅ Lazy init race pattern confirmed by reading session_dedup.go source
- ✅ OnceWithSlot asymmetry confirmed by comparing pre-spawn checks in both code paths

**What's untested:**

- ⚠️ Whether NewWithPool is actually called in production (may be test-only)
- ⚠️ Whether the session dedup race is detectable by `go test -race` (not executed)
- ⚠️ Whether resumeAttempts map size matters in practice (no daemon run duration data)

**What would change this:**

- If NewWithPool is only used in tests → Finding 1 severity drops to LOW
- If daemon is restarted frequently (daily) → Finding 5 becomes irrelevant
- If OnceWithSlot is never called → Finding 4 becomes dead code, severity drops to LOW

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Fix constructor divergence | implementation | Mechanical fix, no design decision needed |
| Fix recovery interface bypass | implementation | Align with existing pattern, no design decision |
| Fix lazy init race | implementation | Standard sync.Once pattern, no design decision |
| Consolidate OnceWithSlot | architectural | Affects public API surface, may have external callers |
| Clean resumeAttempts | implementation | Follow existing TTL pattern from SpawnedIssueTracker |

### Recommended Approach ⭐

**Fix HIGH findings first, then address MEDIUM** - Prioritize the constructor divergence and recovery interface bypass as they affect correctness. The lazy init and OnceWithSlot are safety improvements.

**Implementation sequence:**
1. **Fix NewWithPool** — Add missing fields (VerificationTracker, CompletionFailureTracker, VerificationRetryTracker, BeadsCircuitBreaker) to match NewWithConfig
2. **Fix RunPeriodicRecovery** — Change `GetActiveAgents()` to `d.Agents.GetActiveAgents()` (with nil fallback pattern matching other periodic methods)
3. **Fix lazy init** — Replace check-then-act with `sync.Once` in initDefaultSessionDedupChecker
4. **Delete daemon_test.go.bak** — Remove dead backup file
5. **Evaluate OnceWithSlot** — Determine if it's called anywhere; if not, deprecate/remove

---

## References

**Files Examined:**
- `cmd/orch/daemon.go` (1371 lines) - Main daemon loop, CLI commands, flag handling
- `cmd/orch/daemon_periodic.go` (287 lines) - Periodic task runner, result handlers
- `cmd/orch/doctor_daemon.go` (436 lines) - Doctor self-healing daemon
- `pkg/daemon/daemon.go` (892 lines) - Core daemon struct, constructors, spawn pipeline
- `pkg/daemon/interfaces.go` (190 lines) - Dependency injection interfaces
- `pkg/daemon/periodic.go` (274 lines) - Periodic tasks (reflection, cleanup, recovery)
- `pkg/daemon/completion_processing.go` (603 lines) - Phase: Complete detection and processing
- `pkg/daemon/spawn_tracker.go` (367 lines) - Spawn dedup cache
- `pkg/daemon/pool.go` (254 lines) - Worker pool concurrency control
- `pkg/daemon/orphan_detector.go` (199 lines) - Orphan issue detection
- `pkg/daemon/recovery.go` (177 lines) - Stuck agent recovery
- `pkg/daemon/verification_tracker.go` (313 lines) - Verification pause enforcement
- `pkg/daemon/beads_circuit_breaker.go` (122 lines) - Beads health circuit breaker
- `pkg/daemon/active_count.go` (283 lines) - Multi-backend agent counting
- `pkg/daemon/completion_failure_tracker.go` (97 lines) - Completion failure health tracking
- `pkg/daemon/session_dedup.go` (216 lines) - Session-level deduplication
- `pkg/daemon/cleanup.go` (224 lines) - Stale tmux window cleanup
- `pkg/daemon/capacity.go` (114 lines) - Pool capacity helpers
- `pkg/daemon/question_detector.go` (168 lines) - QUESTION phase detection
- `pkg/daemon/phase_timeout.go` (167 lines) - Phase timeout detection

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-27-audit-daemon-code-health-complexity.md` - Prior daemon health audit
- **Guide:** `.kb/guides/daemon.md` - Daemon operational guide
