## Summary (D.E.K.N.)

**Delta:** The daemon's 6 symptom cluster (crashes, duplicate work, frozen observability, double logging, redundant scans, stuck counters) traces to 3 structural roots: (1) no shutdown budget (unbounded defer chains), (2) conflated concerns in the OODA loop (13+ periodic tasks, spawn pipeline, completion polling all in one process), and (3) dual-write observability (launchd + DaemonLogger both writing to daemon.log).

**Evidence:** Code trace of cmd/orch/daemon.go defer chain, launchd plist StandardOutPath pointing to same file as DaemonLogger, 13 periodic tasks sharing a single scheduler, and prior structural dedup review (2026-03-01) recommending CAS-based redesign.

**Knowledge:** The daemon grew from 5 operations to 30+ subsystems via tactical fixes. Each fix was locally correct but the composite has emergent failures: correlated fail-open degradation in dedup, double logging from dual write paths, and unbounded shutdown work. The cycle cache (cachedAgentDiscoverer) already solved the worst shared-scan problem; further consolidation has diminishing returns.

**Next:** 5 implementation issues created for phased rollout. Double logging fix (Fork 5) is immediate. Shutdown budget (Fork 1) prevents recurring SIGKILL. Dedup consolidation and observability decoupling are Phase 2.

**Authority:** architectural - Cross-subsystem redesign affecting daemon, observability, and dedup layers; requires orchestrator synthesis across 5 decision forks.

---

# Investigation: Daemon Reliability Architecture — Crash Recovery, Dedup, Observability, Shared Scan

**Question:** What structural changes resolve the daemon's recurring symptom cluster (SIGKILL crashes, duplicate spawns, frozen observability, double logging, redundant scans)?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** architect (orch-go-5b9st)
**Phase:** Complete
**Next Step:** None — 5 implementation issues created
**Status:** Complete
**Model:** daemon-autonomous-operation

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-01-inv-structural-review-daemon-dedup-after.md | extends | Yes — dedup pipeline now extracted into SpawnPipeline, L6/L7 gates added | TTL documented as 5min in guide is now fixed to 6h |
| 2026-03-02-design-daemon-launchd-supervision.md | extends | Yes — launchd plist confirmed with StandardOutPath | None |
| 2026-03-04-inv-audit-daemon-code-code-health.md | deepens | Yes — daemon package now 100+ files, Daemon struct ~40 fields | None |
| 2026-03-26-inv-daemon-behaviors-substrate-machinery-vs.md | deepens | Yes — substrate/product distinction applied to periodic tasks | None |

---

## Findings

### Finding 1: Shutdown Path Has No Budget — Defers Are Unbounded

The daemon's `runDaemonLoop` (cmd/orch/daemon.go:13-187) defers 5 operations in order:

```go
defer s.pidLock.Release()      // instant (file unlock)
defer s.cancel()               // instant (context cancel)
defer s.dlog.Close()           // instant (file close)
defer daemon.RemoveStatusFile() // instant (file remove)
defer runReflectionAnalysis()  // UNBOUNDED (shells to kb reflect)
```

Go defers run LIFO, so `runReflectionAnalysis` runs first. orch-go-e0h25 added a 3s context timeout, but:

1. The 3s timeout for reflection leaves only 2s for all other shutdown work within launchd's 5s ExitTimeOut
2. No explicit budget enforces the 5s total
3. If a new defer is added (e.g., spawn cache flush, event log sync), it erodes the margin silently
4. Context cancellation (`s.cancel()`) fires AFTER reflection analysis — child processes spawned during reflection may not get cleaned up

**Evidence:** `cmd/orch/daemon.go:23-29` (defer chain), `cmd/orch/daemon_handlers.go:285-316` (reflection with 3s timeout), `pkg/daemon/reflect.go:19` (ShutdownReflectTimeout=3s), launchd plist ExitTimeOut defaults to 5s.

**Source:** `cmd/orch/daemon.go:23-29`, `cmd/orch/daemon_handlers.go:285-316`, `pkg/daemon/reflect.go:19`

**Significance:** The pattern of "unbounded work in shutdown path" was identified in the symptom cluster. The 3s fix addressed the immediate SIGKILL but doesn't prevent recurrence. Any subsystem that adds a deferred cleanup action can re-introduce the same crash. **Defect class exposure: Class 3 (Stale Artifact Accumulation) — shutdown creates stale artifacts (status file, PID lock) when it exceeds the budget.**

---

### Finding 2: Double Logging Is a Dual-Write to Same File

The daemon's log output appears twice because two independent write paths target `~/.orch/daemon.log`:

**Path 1 — DaemonLogger:** `NewDaemonLogger()` (pkg/daemon/log.go:49-72) creates an `io.MultiWriter(os.Stdout, file)` where `file` is `~/.orch/daemon.log`. Every `dlog.Printf()` writes to both stdout and the log file directly.

**Path 2 — launchd plist:** The generated plist (pkg/daemonconfig/plist.go:64-68) includes:
```xml
<key>StandardOutPath</key>
<string>/Users/dylanconlin/.orch/daemon.log</string>
<key>StandardErrorPath</key>
<string>/Users/dylanconlin/.orch/daemon.log</string>
```

launchd redirects stdout/stderr to daemon.log. Since DaemonLogger also writes to stdout AND directly to daemon.log, every log line is written twice: once by the MultiWriter's stdout arm (captured by launchd) and once by the MultiWriter's file arm (direct write).

Additionally, `Errorf()` (log.go:80-86) writes to stderr AND the file separately — under launchd, stderr also goes to daemon.log, creating a third write for error messages.

**Evidence:** `pkg/daemon/log.go:70` (MultiWriter construction), `pkg/daemonconfig/plist.go:64-68` (plist template), confirmed launchd plist at `~/Library/LaunchAgents/com.orch.daemon.plist` has StandardOutPath pointing to same file.

**Source:** `pkg/daemon/log.go:49-72`, `pkg/daemonconfig/plist.go:64-68`

**Significance:** This is **Defect Class 5 (Contradictory Authority Signals)** — two mechanisms both claim authority over daemon log output. The fix is simple: detect when running under launchd (stdout is already captured) and skip the direct file write. Or: use a different file path.

---

### Finding 3: Dedup Architecture Has Matured — Prior Art Gates Are the Right Layer

The dedup pipeline has been extracted from the 245-line `spawnIssue()` gauntlet (per 2026-03-01 investigation recommendation) into a composable `SpawnPipeline` with named gates (spawn_execution.go:247-300). The 7 gates now run as a pipeline:

| Gate | Layer | Type | State |
|------|-------|------|-------|
| SpawnTrackerGate | L1 | Heuristic (6h TTL) | Working |
| SessionDedupGate | L2 | Infrastructure (session/tmux) | Working |
| TitleDedupMemoryGate | L3 | Heuristic (in-memory) | Working |
| TitleDedupBeadsGate | L4 | Structural (beads DB) | Fail-open |
| FreshStatusGate | L5 | Structural (beads query) | Fail-open |
| CommitDedupGate | L6 | Prior-art (git log) | Fail-open |
| KeywordDedupGate | L7 | Prior-art (title keywords) | Fail-open |

The spawn orientation question asked: "Should dedup move to issue-creation time, spawn time, or both?"

**Answer: Both, but they solve different problems.**

- **Spawn-time dedup** (current 7-gate pipeline) prevents executing the same work twice. This is infrastructure-level and already well-structured.
- **Issue-creation-time dedup** prevents creating duplicate issues. This is a skill-level concern — architects creating follow-up issues should check if work is already committed. The CommitDedupGate (L6) is a spawn-time bandaid for this upstream problem.

The CAS-based redesign recommended in the 2026-03-01 review (beads atomic status transitions) would eliminate the correlated fail-open risk but requires beads fork changes. This should be a Phase 2 enhancement — the extracted pipeline is a good intermediate state.

**Evidence:** `pkg/daemon/spawn_execution.go:247-300` (pipeline construction), `pkg/daemon/prior_art_dedup.go` (CommitDedupGate, KeywordDedupGate), `pkg/daemon/spawn_gate.go` (SpawnPipeline interface)

**Source:** `pkg/daemon/spawn_execution.go:247-300`, `pkg/daemon/prior_art_dedup.go:1-261`

**Significance:** The spawn pipeline extraction was the right structural move. The remaining gap is upstream (issue-creation-time) not downstream (spawn-time). Keyword dedup threshold (50%, 3 common) needs measurement data before tuning.

---

### Finding 4: Cycle Cache Already Solves the Shared Scan Problem

The parked issue (orch-go-6tym2) identified "23+ periodic tasks scanning same resources independently." The daemon actually has 13 named periodic tasks (scheduler.go:7-19):

| Task | Interval | Expensive Query |
|------|----------|-----------------|
| cleanup | per config | Session listing |
| recovery | per config | GetActiveAgents() |
| orphan_detection | per config | GetActiveAgents() |
| phase_timeout | per config | GetActiveAgents() |
| question_detection | per config | GetActiveAgents() |
| agreement_check | per config | beads queries |
| beads_health | per config | beads health snapshot |
| artifact_sync | per config | file system scan |
| registry_refresh | per config | kb projects list |
| verification_failed_escalation | per config | beads queries |
| lightweight_cleanup | per config | beads queries |
| capacity_poll | per config | account capacity |
| audit_select | per config | beads + events queries |

The `cachedAgentDiscoverer` (cycle_cache.go) wraps `d.Agents` during each OODA cycle via `BeginCycle()`/`EndCycle()`, sharing a single `GetActiveAgents()` call across recovery, orphan detection, phase timeout, and question detection. This is the right abstraction — it avoids 4 redundant beads round-trips per cycle.

The remaining queries (beads_health, agreement_check, audit_select, etc.) each query different data — they're not redundant, they're independent concerns that happen to run in the same loop.

**Evidence:** `pkg/daemon/cycle_cache.go:1-61` (cachedAgentDiscoverer), `pkg/daemon/scheduler.go:7-19` (13 task names), `cmd/orch/daemon_periodic.go:33-34` (BeginCycle/EndCycle wrapping)

**Source:** `pkg/daemon/cycle_cache.go:1-61`, `cmd/orch/daemon_periodic.go:29-142`

**Significance:** The shared scan problem is already solved for the most expensive query. Further consolidation would conflate different concerns for marginal gains. The real opportunity is categorizing tasks into tiers for scheduling priority, not consolidating their queries.

---

### Finding 5: Observability Is Appropriately Decoupled (With One Gap)

The daemon's observability stack:

1. **Status file** (`daemon-status.json`) — written atomically each cycle (status.go:100-130)
2. **PID lock** (`~/.orch/daemon.pid`) — process liveness check (pidlock.go)
3. **Health signals** — computed from status file (health_signals.go)
4. **Event log** (`events.jsonl`) — append-only event stream (events package)
5. **DaemonLogger** — dual-write stdout + file (log.go)

`ReadValidatedStatusFile()` (status.go:155-182) already validates PID liveness before trusting the status file, and falls back to PID lock when the status file is missing (SIGKILL restart window). This addresses the "frozen status when daemon dies" symptom.

**The remaining gap:** The sketchybar widget reads `daemon-status.json` and interprets staleness based on `last_poll` timestamp. If the daemon dies, `ReadValidatedStatusFile()` detects the stale file — but the widget may not call this function (it may just parse the JSON directly). The widget's reliability should depend on file mtime (filesystem-level liveness), not file content (application-level liveness).

**Evidence:** `pkg/daemon/status.go:155-182` (ReadValidatedStatusFile), `pkg/daemon/health_signals.go:42-58` (liveness based on LastPoll age), `pkg/daemon/status.go:100-130` (atomic write)

**Source:** `pkg/daemon/status.go:99-205`, `pkg/daemon/health_signals.go:1-207`

**Significance:** The observability architecture is mostly right. The PID validation fix addressed the core issue. The widget should use file mtime as a liveness signal — simpler than JSON parsing and inherently reflects actual daemon activity.

---

## Synthesis

**Key Insights:**

1. **The daemon's reliability problems are concentrated in three root causes, not six symptoms.** The 6 symptoms (crashes, duplicates, frozen observability, double logging, redundant scans, stuck counters) map to: (a) unbounded shutdown (crashes), (b) dual-write logging (double logs), and (c) organic complexity from 30+ subsystems growing independently (the rest). The cycle cache, pipeline extraction, and PID validation already addressed the worst manifestations.

2. **The dedup architecture is at a stable intermediate state.** The SpawnPipeline extraction (from the 245-line gauntlet to composable gates) was the right structural move. The CAS-based redesign (from the 2026-03-01 review) remains desirable but is Phase 2. The immediate gap is upstream: architects should check committed work before creating follow-up issues, which is a skill-level concern not a daemon-level concern.

3. **Double logging has a trivially simple root cause: two writers, one file.** This is Defect Class 5 (Contradictory Authority Signals) and the fix is deterministic: DaemonLogger should write to stdout only when running under launchd, letting launchd handle file persistence.

4. **The shared scan problem is already solved.** The cachedAgentDiscoverer shares the most expensive query. Further consolidation would conflate different concerns for marginal gains.

5. **The shutdown budget needs explicit enforcement, not just timeout values.** The current approach (3s timeout on reflection) works but is fragile — any new defer erodes the margin. An explicit budget enforcer would prevent recurrence.

**Answer to Investigation Question:**

The daemon's structural reliability requires 5 targeted interventions, not a comprehensive rewrite:

1. **Shutdown budget** (Fork 1): Explicit per-subsystem timeouts summing to <4s of launchd's 5s ExitTimeOut, with a budget enforcer that logs warnings when exceeded.
2. **Double logging fix** (Fork 5): DaemonLogger detects launchd context and writes to stdout only, eliminating dual-write.
3. **Issue-creation dedup** (Fork 2): Architect skill should check git history for committed work before creating follow-up issues. Spawn-time dedup is already adequate.
4. **Widget mtime liveness** (Fork 3): sketchybar widget uses file mtime for daemon liveness instead of parsing JSON content.
5. **Periodic task tiering** (Fork 4): Categorize tasks into required/valuable/deferrable tiers and adjust scheduling intervals accordingly, but no structural changes needed.

---

## Structured Uncertainty

**What's tested:**

- ✅ Double logging is caused by dual-write to daemon.log (verified: launchd plist StandardOutPath and DaemonLogger both target ~/.orch/daemon.log)
- ✅ Shutdown defer chain has reflection as the only unbounded operation (verified: code trace of cmd/orch/daemon.go:23-29)
- ✅ Cycle cache already shares GetActiveAgents() across 4 periodic tasks (verified: cycle_cache.go + daemon_periodic.go BeginCycle/EndCycle)
- ✅ SpawnPipeline extraction is complete (verified: spawn_execution.go:247-300 builds pipeline from composable gates)
- ✅ PID validation fallback exists in ReadValidatedStatusFile (verified: status.go:155-182)

**What's untested:**

- ⚠️ Whether the sketchybar widget calls ReadValidatedStatusFile or reads JSON directly (would change the observability gap assessment)
- ⚠️ Whether keyword dedup threshold (50%, 3 keywords) produces false positives in production (no measurement data available)
- ⚠️ Whether beads supports CAS semantics (determines Phase 2 dedup feasibility)
- ⚠️ Actual frequency of SIGKILL events under launchd (determines urgency of shutdown budget)

**What would change this:**

- If beads already supports CAS, the dedup CAS redesign moves from Phase 2 to Phase 1
- If the sketchybar widget already uses ReadValidatedStatusFile, Fork 3 is already solved
- If keyword dedup false-positive rate is >20%, the gate should be demoted to advisory

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Double logging fix (Fork 5) | implementation | Single-scope, deterministic fix, no cross-boundary impact |
| Shutdown budget (Fork 1) | implementation | Within daemon scope, clear criteria, reversible |
| Issue-creation dedup (Fork 2) | architectural | Crosses daemon-skill boundary, requires architect skill changes |
| Widget mtime liveness (Fork 3) | implementation | Single component change (widget script) |
| Periodic task tiering (Fork 4) | implementation | Config tuning, no structural changes |

### Recommended Approach: Phased Interventions

**Phase 1 (immediate, high-value):**

1. **Fix double logging** — Detect launchd context in NewDaemonLogger (check if stdout fd is a file matching DaemonLogPath) and skip direct file write. If running foreground (stdout is tty), keep MultiWriter behavior. Estimated: <30 lines changed.

2. **Explicit shutdown budget** — Replace ad-hoc defers with a `shutdownBudget` struct that tracks remaining time and logs warnings when individual subsystems exceed their allotment:
   ```
   Total budget: 4s (launchd 5s - 1s safety)
   Reflection: 2.5s (was 3s, tightened)
   Status cleanup: 0.5s
   Log flush: 0.5s
   Safety margin: 0.5s
   ```
   Move `s.cancel()` to run BEFORE reflection (so child processes get SIGTERM early).

**Phase 2 (near-term, medium-value):**

3. **Issue-creation dedup in architect skill** — Add a pre-creation check to the architect skill: before `bd create`, search recent git commits for the proposed beads ID or similar title. This addresses the upstream gap that CommitDedupGate bandaids at spawn-time.

4. **Widget mtime liveness** — Update sketchybar widget script to check `daemon-status.json` mtime. If mtime > 2 * poll_interval ago, show daemon as dead regardless of file content.

**Phase 3 (deferred, low-urgency):**

5. **Periodic task tiering** — Adjust scheduler intervals: Tier 3 tasks (artifact_sync, audit_select, agreement_check) run at 2-4x their current interval. No code changes needed, just config defaults.

6. **Beads CAS investigation** — Determine if beads can support atomic status transitions for structural dedup primary gate.

**Why this approach:**
- Addresses all 5 forks in the orientation frame
- Phase 1 fixes are deterministic, low-risk, high-value
- No architectural overhaul required — the intermediate state is stable
- Preserves the good structural work already done (pipeline extraction, cycle cache, PID validation)

**Trade-offs accepted:**
- Deferring CAS-based dedup redesign — the 7-gate pipeline is adequate for now
- Not consolidating periodic tasks — the cycle cache handles the expensive case
- Not building a "minimal daemon" — removing subsystems would reduce observability without proven benefit

### Alternative Approaches Considered

**Option B: Full Daemon Rewrite with Subsystem Process Model**
- **Pros:** Clean separation of concerns, each subsystem is independently restartable
- **Cons:** Massive scope, violates "no local agent state" constraint (would need IPC), the current code works
- **When to use instead:** If the daemon grows beyond 150+ files or if multiple daemon instances need to coordinate

**Option C: Extract Periodic Tasks to Cron Jobs**
- **Pros:** Simplifies daemon to pure spawn/complete loop, each task independently scheduled
- **Cons:** Loses cycle cache benefit, 13 separate cron entries to manage, loses shared daemon state (spawn tracker, rate limiter)
- **When to use instead:** If periodic tasks need project-independent scheduling or if daemon restarts frequently break periodic task state

**Rationale for recommendation:** The daemon's problems are specific and fixable without wholesale restructuring. The dual-write logging fix alone eliminates 50% of the reported noise. The shutdown budget prevents the most severe failure mode (SIGKILL data loss). These are the highest-value interventions per line of code changed.

---

### Implementation Details

**What to implement first:**
- Double logging fix (immediate, ~30 lines, zero risk)
- Shutdown budget enforcement (same session, ~50 lines, prevents SIGKILL recurrence)

**Things to watch out for:**
- ⚠️ The DaemonLogger must still work when running foreground (not under launchd). Detect launchd context by checking if stdout fd matches daemon.log, not by environment variable
- ⚠️ Shutdown budget must not block the PID lock release — this should always be the last operation (currently it runs last due to LIFO, but budget enforcement shouldn't change this)
- ⚠️ The architect skill's commit check must handle cross-project git histories — `git log --grep` only searches the current repo
- ⚠️ Keyword dedup gate false-positive rate should be measured before tuning thresholds

**Areas needing further investigation:**
- Beads CAS support for structural dedup primary gate
- Keyword dedup false-positive rate in production
- Whether sketchybar widget already handles stale status files

**Success criteria:**
- ✅ No double log lines when running under launchd
- ✅ Daemon survives launchd stop/restart without SIGKILL (measured over 1 week)
- ✅ Shutdown completes within 4s budget (logged if exceeded)
- ✅ No new duplicate spawn incidents for 2 weeks after Phase 1 deployment

### Composition Claims

| ID | Claim | Components Involved | How to Verify |
|----|-------|--------------------|----|
| CC-1 | "Daemon logs each event exactly once under launchd" | DaemonLogger + launchd plist | Run under launchd, count lines in daemon.log matching a unique timestamp |
| CC-2 | "Daemon shuts down within 4s of SIGTERM" | shutdown budget + reflection timeout + defer chain | Send SIGTERM, measure time to PID disappearance |
| CC-3 | "Widget shows daemon as dead within 2 poll intervals of crash" | mtime check + status file write | Kill daemon with SIGKILL, verify widget shows red within 2 minutes |

---

## Defect Class Exposure

| Finding | Defect Class | Mitigation |
|---------|-------------|------------|
| Double logging | Class 5 (Contradictory Authority Signals) | Single write path based on execution context |
| Unbounded shutdown | Class 3 (Stale Artifact Accumulation) | Explicit budget prevents stale PID/status files |
| Correlated fail-open dedup | Class 6 (Duplicate Action) | CAS-based primary gate (Phase 2) |
| Frozen observability | Class 3 (Stale Artifact Accumulation) | mtime-based liveness (Phase 2) |

---

## References

**Files Examined:**
- `cmd/orch/daemon.go` — Main daemon loop, defer chain, shutdown check gates
- `cmd/orch/daemon_loop.go` — daemonSetup, daemonLoopState, spawn cycle, status file writing
- `cmd/orch/daemon_handlers.go` — runReflectionAnalysis with shutdown timeout
- `cmd/orch/daemon_periodic.go` — runPeriodicTasks, handler functions for 13 periodic tasks
- `pkg/daemon/daemon.go` — Daemon struct (40+ fields), New/NewWithConfig constructors
- `pkg/daemon/ooda.go` — OODA phase implementations (Sense, Orient, Decide, Act)
- `pkg/daemon/spawn_execution.go` — spawnIssue with SpawnPipeline, buildSpawnPipeline
- `pkg/daemon/prior_art_dedup.go` — CommitDedupGate, KeywordDedupGate
- `pkg/daemon/scheduler.go` — PeriodicScheduler with 13 named tasks
- `pkg/daemon/cycle_cache.go` — cachedAgentDiscoverer for shared GetActiveAgents
- `pkg/daemon/log.go` — DaemonLogger with MultiWriter to stdout + file
- `pkg/daemon/status.go` — DaemonStatus, WriteStatusFile, ReadValidatedStatusFile
- `pkg/daemon/health_signals.go` — Traffic-light health signal computation
- `pkg/daemon/reflect.go` — RunReflection*, ShutdownReflectTimeout
- `pkg/daemon/completion_processing.go` — CompletionConfig, CompletionOnce
- `pkg/daemon/periodic.go` — RunPeriodicCleanup, RunPeriodicRecovery, RunPeriodicRegistryRefresh
- `pkg/daemonconfig/plist.go` — Plist template with StandardOutPath/StandardErrorPath
- `.kb/guides/daemon.md` — Authoritative daemon reference guide
- `.kb/investigations/2026-03-01-inv-structural-review-daemon-dedup-after.md` — Prior dedup structural review
- `.kb/models/defect-class-taxonomy/model.md` — 7-class defect taxonomy

**Commands Run:**
```bash
# Confirm launchd plist writes to same file as DaemonLogger
cat ~/Library/LaunchAgents/com.orch.daemon.plist | grep -A1 "StandardOut\|StandardErr"

# Count daemon package files
ls pkg/daemon/

# Check daemon.go line count
wc -l pkg/daemon/daemon.go
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-01-inv-structural-review-daemon-dedup-after.md` — Prior dedup structural review (recommended CAS redesign)
- **Investigation:** `.kb/investigations/2026-03-02-design-daemon-launchd-supervision.md` — launchd architecture
- **Investigation:** `.kb/investigations/2026-03-26-inv-daemon-behaviors-substrate-machinery-vs.md` — Substrate vs product classification
- **Model:** `.kb/models/daemon-autonomous-operation/model.md` — Daemon operational model
- **Model:** `.kb/models/defect-class-taxonomy/model.md` — Defect class taxonomy

---

## Investigation History

**2026-03-27 11:42:** Investigation started
- Initial question: What structural changes resolve the daemon's recurring symptom cluster?
- Context: 312-issue hotspot, 4+ independent daemon bugs fixed in recent sessions with same shape

**2026-03-27 12:00:** Exploration phase — 15+ source files read
- Mapped defer chain, periodic task structure, dedup pipeline, logging architecture
- Identified 5 decision forks from orientation frame

**2026-03-27 12:30:** Root cause identified — double logging
- Confirmed: launchd StandardOutPath + DaemonLogger MultiWriter both target daemon.log
- Defect Class 5 (Contradictory Authority Signals)

**2026-03-27 13:00:** Investigation completed
- Status: Complete
- Key outcome: 3 structural roots, 5 phased interventions, double logging fix is immediate win
