## Summary (D.E.K.N.)

**Delta:** The daemon's three structural problems (6-layer dedup gauntlet, 625-line loop, operational unreliability) share a common root: internal complexity makes failure modes invisible and reasoning impossible. The fix is a 3-phase inside-out simplification: (1) collapse dedup to CAS-like gate + advisory checks, (2) extract scheduler from loop, (3) supervision gaps close naturally when the simplified daemon is launchd-managed.

**Evidence:** Code trace of spawnIssue() (daemon.go:672-928, 245 lines, 6 dedup layers); runDaemonLoop (cmd/orch/daemon.go:380-1077, 697 lines, 12 periodic subsystems); beads UpdateArgs has no ExpectedStatus field (no CAS); `orch daemon install` already exists for launchd plist generation; daemonConfigFromFlags() unified config decision (2026-02-15) partially implemented; periodic tasks already extracted to daemon_periodic.go.

**Knowledge:** Beads lacks native CAS but we can simulate CAS semantics in Go: fresh-check + update as a single atomic function (read-then-write behind a local mutex, with the existing fail-fast on update error). The dedup layers are heuristic-first because they predate the beads status update (L6). Inverting to structural-first eliminates 4 of 6 layers from the critical path. The loop extraction is partially done (periodic tasks extracted) but the main loop body still has reconciliation, verification, completion, invariants, circuit breaker, status writing, and spawn loop all inline.

**Next:** Create implementation issues for the 3 phases. Phase 1 (dedup pipeline) is the highest-value change — it reduces spawnIssue from 245 lines to ~60 and makes the dedup invariant explicit and testable.

**Authority:** architectural - Cross-component redesign affecting daemon, spawn pipeline, beads integration, and operational workflow.

---

# Investigation: Unified Reliability Design for Orch Daemon

**Question:** How should the daemon's three connected structural problems (dedup gauntlet, loop extraction, operational reliability) be unified into a single coherent design, and in what order should they be implemented?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect (orch-go-pkm65)
**Phase:** Complete
**Next Step:** Create implementation issues from phased plan
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-02-15-daemon-unified-config-construction.md` (extends Part 1)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-03-01-inv-structural-review-daemon-dedup-after.md` | deepens | Yes — 6-layer mapping confirmed; all findings verified in code | None |
| `2026-03-02-design-daemon-launchd-supervision.md` | extends | Yes — launchd readiness confirmed, `orch daemon install` exists | None |
| `2026-02-15-design-daemon-unified-config-persistent-tracker.md` | extends | Yes — `daemonConfigFromFlags()` exists at daemon.go:284; Part 1 done | None |
| `2026-02-27-audit-daemon-code-health-complexity.md` | confirms | Yes — Daemon struct has 30+ fields, pkg/daemon is 99 files | None |
| `2026-02-19-design-extract-daemon-config-package.md` | confirms | Yes — `pkg/daemonconfig/` exists with config.go and plist.go | None |

---

## Problem Framing

### Design Question

Three connected structural problems prevent the daemon from running reliably without manual intervention:

1. **Dedup Gauntlet** — 6 independent layers in spawnIssue() (245 lines), accumulated via 9 tactical fixes. Only L6 (UpdateStatus fail-fast) is structural. Others are heuristic/fail-open.
2. **Loop Extraction** — 697 lines in runDaemonLoop orchestrating 12 subsystems. Periodic tasks partially extracted to daemon_periodic.go, but reconciliation/verification/completion/invariants/circuit-breaker/status-writing/spawn-loop remain inline.
3. **Operational Reliability** — JSONL lock pileup, stale status files showing phantom agents, orphan detector retaining spawn cache for 6h blocking retries, foreground daemon dying with shell session.

### Success Criteria

1. Single positive invariant for "no duplicate spawns" that can be stated in one sentence
2. spawnIssue() under 80 lines (dedup extracted to composable pipeline)
3. runDaemonLoop under 200 lines (subsystems extracted to phases)
4. Daemon runs unattended for 1 week via launchd with automatic crash recovery
5. No fail-open checks in the dedup critical path

### Key Insight

**These aren't independent problems.** The 6-layer dedup fails subtly BECAUSE its complexity makes reasoning impossible. Scattered config silently disables recovery features. The 697-line loop makes failure modes invisible. **Internal simplification IS the reliability fix.**

---

## Findings

### Finding 1: Beads Lacks CAS But We Can Simulate It

**Evidence:** `UpdateArgs` (pkg/beads/types.go:294-310) has no `ExpectedStatus` field. `UpdateBeadsStatus()` (issue_adapter.go:528) unconditionally sets status — if two processes race, both succeed ("idempotent" per comment at daemon.go:759). The structural review (2026-03-01) correctly identified this TOCTOU gap.

However, we don't need to fork beads. The daemon is a single process (PID-locked). We can implement CAS-like semantics in Go:

```go
// pkg/daemon/dedup.go
func (g *SpawnGate) TryClaimIssue(id string) (bool, error) {
    // 1. Fresh check: read current status from beads
    status, err := g.issues.GetIssueStatus(id)
    if err != nil {
        return false, err  // fail-fast: can't verify = can't spawn
    }
    if status != "open" {
        return false, nil  // already claimed
    }
    // 2. Atomic claim: set to in_progress
    if err := g.updater.UpdateStatus(id, "in_progress"); err != nil {
        return false, err  // fail-fast: can't claim = can't spawn
    }
    return true, nil
}
```

This collapses L5 (fresh status check) + L6 (update status) into a single fail-fast function. The PID lock guarantees single daemon instance, eliminating the multi-process TOCTOU race that CAS was meant to solve.

**Source:** `pkg/beads/types.go:294-310`, `pkg/daemon/daemon.go:748-832`, `pkg/daemon/pidlock.go`

**Significance:** The CAS gap identified in the structural review is real for multi-process scenarios, but the PID lock makes it moot for the daemon. We can get structural-first dedup without touching beads. The only remaining multi-process risk is `orch spawn --bypass-triage` (manual spawn), which should use the same gate.

---

### Finding 2: The Dedup Pipeline Has a Clean 2-Tier Decomposition

**Evidence:** Tracing the 6 layers reveals two distinct concerns:

**Tier 1: Structural Gate (fail-fast, authoritative)**
- L5+L6 collapsed → `TryClaimIssue()` (see Finding 1)
- Invariant: "If TryClaimIssue succeeds, this daemon process has exclusive authority to spawn this issue"
- On failure: abort, no spawn

**Tier 2: Advisory Checks (warn-only, non-blocking)**
- L1 (SpawnedIssueTracker): thrash detection — "was this issue spawned 3+ times?" → log warning
- L2 (Session/Tmux check): stale session detection — "is there already an agent for this?" → log warning
- L3/L4 (Title dedup): content duplicate detection — "is there another issue with the same title?" → log warning

The advisory tier provides operational visibility but does NOT block spawns. Only the structural gate blocks.

**Source:** `pkg/daemon/daemon.go:672-928`, `pkg/daemon/spawn_tracker.go`, `pkg/daemon/session_dedup.go`

**Significance:** This decomposition eliminates the fail-open compound gap (Finding 3 of the structural review). When beads is unavailable, TryClaimIssue fails-fast and blocks the spawn. Advisory checks can independently degrade without affecting correctness.

---

### Finding 3: The Daemon Struct Has Become a God Object (30+ Fields, 16 Interfaces)

**Evidence:** `Daemon` struct (daemon.go:47-183) has:
- 12 `last*` timestamp fields (one per periodic task)
- 16 interface fields (Issues, Spawner, Completions, Reflector, ModelDrift, KnowledgeHealth, AgreementCheck, Cleaner, ActiveCounter, Agents, StatusUpdater, BeadsHealth, FrictionAccumulator, AutoCompleter, BeadsCircuitBreaker, InvariantChecker)
- 5 tracker/state fields (SpawnedIssues, VerificationTracker, CompletionFailureTracker, VerificationRetryTracker, questionNotified)
- 4 focus fields

This makes the daemon untestable as a unit — tests must wire up 16 mock interfaces. It also means every new periodic task adds 2-3 fields (interface + lastRun + config).

The periodic task pattern is already repeating: each task has `ShouldRun*()`, `RunPeriodic*()`, `Last*Time()`, `Next*Time()`. This is a generic scheduler begging to be extracted.

**Source:** `pkg/daemon/daemon.go:47-183`, `pkg/daemon/periodic.go`, `cmd/orch/daemon_periodic.go`

**Significance:** The scheduler extraction would move 12 lastX fields + their ShouldRun/RunPeriodic methods to a generic `Scheduler` type, reducing the Daemon struct to its core: config + pool + spawn gate + scheduler. Each periodic task becomes a registered `Task` with interval + handler, not a method on the god object.

---

### Finding 4: Operational Reliability Gaps Are Symptoms, Not Root Causes

**Evidence:** The task identifies specific operational failures:

| Failure | Root Cause | Fixed By |
|---------|-----------|----------|
| JSONL lock pileup | Daemon polls faster than hung bd processes clear | BeadsCircuitBreaker (already implemented, daemon.go:823-835) |
| Stale status files | Status file written but daemon dies before next cycle | Already addressed: `defer daemon.RemoveStatusFile()` + PID liveness check |
| Orphan detector 6h block | Spawn cache TTL prevents retries after orphan detection | Dedup redesign: structural gate replaces TTL-based cache |
| Foreground daemon dying | No process supervision | `orch daemon install` for launchd (already exists, needs activation) |

Three of four are already addressed or will be fixed by the dedup redesign. The remaining gap (launchd activation) is configuration, not code.

**Source:** `pkg/daemon/beads_circuit_breaker.go`, `cmd/orch/daemon.go:486`, `cmd/orch/daemon_launchd.go`

**Significance:** This validates the key insight: internal simplification IS the reliability fix. The dedup redesign eliminates the orphan-cache contradiction. The circuit breaker handles beads lock pileup. Launchd handles process supervision. No new reliability mechanisms needed — just activate and simplify existing ones.

---

### Finding 5: runDaemonLoop Has a Clear Phase Structure Hidden in Its Linearity

**Evidence:** The 697-line loop body has a natural phase structure that's currently expressed as inline sequential code:

```
Phase 1: Housekeeping (lines 554-639)
  - Reconcile pool with OpenCode
  - Check verification signal
  - Check resume signal
  - Verification pause check

Phase 2: Maintenance (line 642)
  - runPeriodicTasks() [already extracted]

Phase 3: Completions (lines 650-742)
  - CompletionOnce + failure tracking + event logging

Phase 4: Health Checks (lines 744-835)
  - Invariant checker
  - Circuit breaker

Phase 5: Status (lines 837-907)
  - Ready issues count
  - Write status file

Phase 6: Spawn (lines 909-1057)
  - Capacity check
  - Stuck detection
  - Spawn inner loop
```

Each phase could be a method on the Daemon struct, making the loop body ~30 lines of phase calls.

**Source:** `cmd/orch/daemon.go:542-1077`

**Significance:** The extraction path is clear and low-risk: turn each phase into a method, call them sequentially. No behavioral change needed — pure structural extraction. The periodic tasks extraction (daemon_periodic.go) already proves this pattern works.

---

## Synthesis

**Key Insights:**

1. **CAS without forking beads.** The PID lock guarantees single daemon process. Fresh-check + update as a single Go function provides CAS-like semantics without beads changes. This was the main blocker in the structural review ("requires beads to support CAS") — it's not a blocker.

2. **Structural-first eliminates compound failure.** The 6-layer dedup fails together because 4 layers depend on the same infrastructure (beads). Making the structural gate the ONLY gate, and demoting heuristics to advisory, means infrastructure failure = no spawn (correct) instead of infrastructure failure = bypass all dedup (incorrect).

3. **The reliability fix IS the simplification.** Each operational failure maps to either (a) an already-implemented fix (circuit breaker, PID liveness), (b) a side effect of dedup redesign (orphan 6h block), or (c) launchd activation (already coded). No new reliability mechanisms needed.

4. **Phase ordering matters: dedup first.** The dedup pipeline is the highest-complexity, highest-risk code. Simplifying it first reduces the cognitive load for subsequent extractions, and eliminates the orphan-cache contradiction that is the most subtle operational reliability failure.

**Answer to Investigation Question:**

The three problems should be unified as a 3-phase inside-out simplification:

**Phase 1: Dedup Pipeline** (highest value, ~2 days) — Extract `pkg/daemon/dedup.go` with `SpawnGate` struct. Replace 6-layer gauntlet with 2-tier pipeline (structural gate + advisory checks). Eliminate fail-open compound gap. Fix orphan-cache contradiction by removing TTL-as-policy.

**Phase 2: Scheduler Extraction** (~1 day) — Extract `pkg/daemon/scheduler.go` with generic `Scheduler` type. Move 12 periodic tasks out of Daemon struct. Reduce runDaemonLoop to 6 phase calls.

**Phase 3: Operational Hardening** (~0.5 day) — Activate launchd via `orch daemon install`. Add log rotation. Verify 1-week unattended operation.

The phases build on each other: Phase 1 makes spawnIssue testable, Phase 2 makes the loop testable, Phase 3 activates supervision for the simplified daemon.

---

## Structured Uncertainty

**What's tested:**

- ✅ Beads UpdateArgs has no ExpectedStatus field (verified: read pkg/beads/types.go:294-310)
- ✅ PID lock exists and works (verified: daemon.go:401-405, pidlock.go)
- ✅ `orch daemon install` exists (verified: cmd/orch/daemon_launchd.go:16-66)
- ✅ daemonConfigFromFlags() exists (verified: cmd/orch/daemon.go:284-321)
- ✅ Periodic tasks already extracted to daemon_periodic.go (verified: 348 lines, 11 task handlers)
- ✅ BeadsCircuitBreaker implemented (verified: beads_circuit_breaker.go, wired in loop at line 823)
- ✅ 6 dedup layers mapped with exact line numbers (verified: structural review + code trace)

**What's untested:**

- ⚠️ Whether the fresh-check + update Go-level CAS is sufficient without true database-level CAS (the PID lock covers single daemon, but `orch spawn --bypass-triage` could still race)
- ⚠️ Whether removing spawn cache TTL-as-policy actually eliminates thrash loops in practice (the 6h TTL was an empirical fix for overnight runs)
- ⚠️ Whether launchd KeepAlive interacts poorly with daemon's own crash recovery logic
- ⚠️ Whether the advisory checks produce enough signal to be worth keeping vs. just removing them entirely

**What would change this:**

- If `orch spawn --bypass-triage` is used frequently by the orchestrator, the Go-level CAS isn't sufficient and we'd need beads-level CAS or a shared lock file
- If removing the 6h TTL causes thrash loops in production overnight, we'd need an explicit cooldown mechanism (separate from dedup) with semantic meaning ("cooling down after N failures" vs "blocked for arbitrary duration")
- If the daemon turns out to need multi-instance support (e.g., VPS deployment), the PID lock assumption breaks and we'd need real distributed CAS

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Dedup pipeline redesign (Phase 1) | architectural | Cross-component: dedup pipeline, beads integration, spawn path, orphan detector |
| Scheduler extraction (Phase 2) | implementation | Pure structural refactor, no behavioral change, stays within daemon package |
| Operational hardening (Phase 3) | implementation | Configuration activation + log rotation, no cross-boundary impact |

### Recommended Approach ⭐

**3-Phase Inside-Out Simplification** — Simplify internal structure first (dedup, then scheduler), then activate operational reliability mechanisms that already exist.

**Why this approach:**
- Phase 1 (dedup) eliminates the highest-complexity code and the compound fail-open gap
- Phase 2 (scheduler) reduces the loop from 697 to ~150 lines, making all failure modes visible
- Phase 3 (operational) is mostly activation of existing code, not new development
- Each phase is independently testable and deployable

**Trade-offs accepted:**
- Go-level CAS doesn't protect against `orch spawn --bypass-triage` races (acceptable: bypass-triage is rare, used only by orchestrator)
- Advisory checks are demoted from blocking to warning (acceptable: structural gate is authoritative)
- Removing 6h TTL-as-policy requires explicit cooldown for thrash prevention (small additional work in Phase 1)

---

## Phased Implementation Plan

### Phase 1: Dedup Pipeline Extraction (~2 days)

**Goal:** Replace 6-layer gauntlet with 2-tier pipeline. Make dedup invariant explicit and testable.

#### Step 1.1: Create `pkg/daemon/spawn_gate.go`

```go
// SpawnGate provides the structural gate for spawn authorization.
// Invariant: TryClaimIssue succeeds → this process has exclusive spawn authority.
type SpawnGate struct {
    issues  IssueQuerier    // for fresh status check
    updater IssueUpdater    // for status update
    pool    *WorkerPool     // for capacity
}

// TryClaimIssue atomically checks status and claims the issue.
// Returns (true, slot) on success, (false, nil) on already-claimed.
// Fail-fast: returns error if infrastructure is unavailable.
func (g *SpawnGate) TryClaimIssue(issue *Issue) (bool, *Slot, error)

// RollbackClaim reverses a claim if spawn fails.
func (g *SpawnGate) RollbackClaim(issue *Issue, slot *Slot) error
```

**Files created:** `pkg/daemon/spawn_gate.go`, `pkg/daemon/spawn_gate_test.go`
**Files modified:** `pkg/daemon/daemon.go` (spawnIssue calls SpawnGate)

#### Step 1.2: Create `pkg/daemon/advisory_checks.go`

```go
// AdvisoryResult contains non-blocking warnings from heuristic checks.
type AdvisoryResult struct {
    Warnings []string
    ThrashCount int  // how many times this issue has been spawned
}

// RunAdvisoryChecks runs non-blocking heuristic checks after structural gate.
// These checks NEVER block a spawn — they only produce warnings.
func RunAdvisoryChecks(issue *Issue, tracker *SpawnedIssueTracker) AdvisoryResult
```

Move L1 (spawn tracker), L2 (session check), L3/L4 (title dedup) into advisory checks.

**Files created:** `pkg/daemon/advisory_checks.go`, `pkg/daemon/advisory_checks_test.go`

#### Step 1.3: Refactor `spawnIssue()` to use pipeline

The new spawnIssue() becomes:

```go
func (d *Daemon) spawnIssue(issue *Issue, skill, model string) (*OnceResult, *Slot, error) {
    // Phase 1: Structural Gate (fail-fast)
    claimed, slot, err := d.spawnGate.TryClaimIssue(issue)
    if err != nil { return errorResult(issue, err) }
    if !claimed { return skippedResult(issue, "already claimed") }

    // Phase 2: Advisory Checks (warn-only)
    advisory := RunAdvisoryChecks(issue, d.SpawnedIssues)
    for _, w := range advisory.Warnings {
        d.log.Warn(w)
    }

    // Phase 3: Spawn
    if err := d.spawner.SpawnWork(issue.ID, model, issue.ProjectDir); err != nil {
        d.spawnGate.RollbackClaim(issue, slot)
        return spawnErrorResult(issue, err)
    }

    // Phase 4: Post-spawn tracking
    d.SpawnedIssues.MarkSpawnedWithTitle(issue.ID, issue.Title)
    d.rateLimiter.RecordSpawn()
    return successResult(issue, skill, model), slot, nil
}
```

~60 lines. Down from 245.

**Files modified:** `pkg/daemon/daemon.go` (replace spawnIssue body)

#### Step 1.4: Add explicit cooldown to replace TTL-as-policy

```go
// CooldownTracker prevents thrash loops with semantic cooldowns.
// Unlike the 6h TTL cache, cooldowns have explicit reasons and can be cleared.
type CooldownTracker struct {
    cooldowns map[string]Cooldown
}

type Cooldown struct {
    IssueID   string
    Reason    string    // "orphan_detected", "spawn_failed_3x"
    Until     time.Time
    Clearable bool      // true = orphan detector can clear; false = requires manual
}
```

The orphan detector sets a cooldown when it resets an issue. The cooldown has a configurable duration (default 30min, not 6h) and a reason. Cooldowns can be explicitly cleared (`orch daemon clear-cooldown <id>`).

**Files created:** `pkg/daemon/cooldown.go`, `pkg/daemon/cooldown_test.go`
**Files modified:** `pkg/daemon/orphan_detector.go` (use cooldown instead of spawn cache retention)

#### Step 1.5: Clean up dead code

- Remove `ReconcileWithIssues()` from spawn_tracker.go (dead code, never called)
- Clean unbounded `spawnCounts` map in `CleanStale()`
- Update daemon guide (.kb/guides/daemon.md) with new dedup architecture

**Files modified:** `pkg/daemon/spawn_tracker.go`, `.kb/guides/daemon.md`

---

### Phase 2: Scheduler Extraction (~1 day)

**Goal:** Extract periodic task scheduling from the Daemon struct. Reduce runDaemonLoop to phase calls.

#### Step 2.1: Create `pkg/daemon/scheduler.go`

```go
// Task represents a periodic maintenance task.
type Task struct {
    Name     string
    Interval time.Duration
    Enabled  bool
    Handler  func() interface{}  // returns result or nil
}

// Scheduler manages periodic task execution.
type Scheduler struct {
    tasks   []Task
    lastRun map[string]time.Time
}

// RunDue executes all tasks that are due and returns results.
func (s *Scheduler) RunDue() []TaskResult
```

#### Step 2.2: Register all periodic tasks

Move the 12 `ShouldRun*/RunPeriodic*` pairs from daemon.go/periodic.go into registered tasks:

| Task | Current Location | Interval |
|------|-----------------|----------|
| Reflection | periodic.go:12-57 | 1h |
| ModelDrift | model_drift_reflection.go | 4h |
| KnowledgeHealth | knowledge_health.go | 2h |
| Cleanup | periodic.go:77-124 | 6h |
| Recovery | periodic.go:144-255 | 5m |
| OrphanDetection | orphan_detector.go | 30m |
| PhaseTimeout | phase_timeout.go | 5m |
| QuestionDetection | question_detector.go | 5m |
| AgreementCheck | agreement_check.go | 30m |
| BeadsHealth | beads_health.go | 1h |
| FrictionAccumulation | friction_accumulator.go | 1h |

This removes 12 `last*` fields from Daemon struct and 12 `ShouldRun*/Last*Time/Next*Time` method sets.

**Files created:** `pkg/daemon/scheduler.go`, `pkg/daemon/scheduler_test.go`
**Files modified:** `pkg/daemon/daemon.go` (Daemon struct loses 12 fields, gains 1 Scheduler), all periodic task files (remove ShouldRun boilerplate)

#### Step 2.3: Extract loop phases

Refactor runDaemonLoop into phase methods:

```go
for {
    if ctx.Err() != nil { return }
    cycles++

    d.reconcile()           // Phase 1: pool + signals
    d.runMaintenance()      // Phase 2: periodic tasks
    d.processCompletions()  // Phase 3: completion pipeline
    d.checkHealth()         // Phase 4: invariants + circuit breaker
    d.writeStatus()         // Phase 5: status file
    d.spawnAgents(ctx)      // Phase 6: spawn loop

    d.sleepOrExit(ctx)
}
```

~100 lines for the loop, down from 697.

**Files created:** `cmd/orch/daemon_phases.go` (phase methods)
**Files modified:** `cmd/orch/daemon.go` (loop body replaced with phase calls)

---

### Phase 3: Operational Hardening (~0.5 day)

**Goal:** Activate launchd supervision and verify 1-week unattended operation.

#### Step 3.1: Activate launchd

```bash
orch daemon install  # already exists
```

Verify:
- `launchctl list | grep orch.daemon` shows PID
- `kill -9 <pid>` → launchd restarts within seconds
- `tail -f ~/.orch/daemon.log` shows continuous operation

#### Step 3.2: Add log rotation

The daemon log (`~/.orch/daemon.log`) grows indefinitely. Add `newsyslog` configuration or in-process rotation.

```bash
# /etc/newsyslog.d/orch-daemon.conf
/Users/dylanconlin/.orch/daemon.log 644 5 1024 * J
```

Or: in-process rotation via `lumberjack` library (more Go-idiomatic, no system config).

#### Step 3.3: Monitoring checkpoint

After Phase 1+2 deployed and launchd activated:
- [ ] Daemon runs for 24h without manual intervention
- [ ] No duplicate spawns (check events.jsonl)
- [ ] Completion processing works (check daemon.log)
- [ ] Orphan detection resets issues correctly (not blocked by 6h TTL)
- [ ] Circuit breaker activates and recovers during beads lock events
- Extend to 1 week once 24h passes

---

## Defect Class Exposure

| Phase | Defect Classes | Mitigation |
|-------|---------------|------------|
| Phase 1 (dedup) | Class 5 (Contradictory Authority), Class 6 (Duplicate Action) | Single structural gate eliminates contradictory layers; advisory checks can't cause duplicates |
| Phase 1 (cooldown) | Class 3 (Stale Artifact Accumulation) | Cooldowns have expiry AND reason; explicit cleanup instead of TTL-only |
| Phase 2 (scheduler) | Class 0 (Scope Expansion) | Scheduler is pure — tasks register themselves, scheduler doesn't know task internals |
| Phase 3 (launchd) | Class 3 (Stale Artifact) | Status file cleanup on shutdown; PID liveness check on status read |

---

## Blocking Questions

### Q1: Should advisory checks be removed entirely, or kept as warnings?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If removed, spawn_tracker.go and session_dedup.go can be deleted entirely (~500 lines). If kept as warnings, they become purely observational with no blocking behavior.
- **Recommendation:** Keep as warnings for the first release, measure signal-to-noise ratio over 2 weeks, then decide whether to remove.

### Q2: What cooldown duration should replace the 6h TTL for orphan respawns?

- **Authority:** implementation
- **Subtype:** judgment
- **What changes based on answer:** Shorter cooldown (15-30min) = faster recovery from agent death. Longer cooldown (2-4h) = more conservative against thrash loops.
- **Recommendation:** 30 minutes, matching the orphan detection interval. If an orphan is detected and reset, the cooldown means it won't respawn until the next orphan detection cycle, providing a natural pacing mechanism.

### Q3: Should `orch spawn --bypass-triage` also use the SpawnGate?

- **Authority:** architectural
- **Subtype:** factual
- **What changes based on answer:** If yes, bypass-triage spawns get dedup protection (prevents manual+daemon race). If no, the existing behavior continues (manual spawns are trusted).
- **Recommendation:** Yes — bypass-triage should use SpawnGate.TryClaimIssue() for consistency. The "bypass" refers to triage routing, not dedup protection.

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` — Main daemon struct, spawnIssue (245 lines), dedup layers
- `pkg/daemon/spawn_tracker.go` — SpawnedIssueTracker, 6h TTL, dead ReconcileWithIssues
- `pkg/daemon/orphan_detector.go` — Orphan detection, intentional cache retention
- `pkg/daemon/session_dedup.go` — Session/tmux existence checks
- `pkg/daemon/beads_circuit_breaker.go` — Lock pileup protection
- `pkg/daemon/periodic.go` — Periodic task pattern (ShouldRun/RunPeriodic)
- `pkg/daemon/interfaces.go` — 16 interface definitions
- `pkg/daemonconfig/config.go` — Config struct (175 lines, 30+ fields)
- `cmd/orch/daemon.go` — runDaemonLoop (697 lines), daemonConfigFromFlags
- `cmd/orch/daemon_periodic.go` — Extracted periodic task handlers
- `cmd/orch/daemon_launchd.go` — Install/uninstall commands
- `pkg/beads/types.go` — UpdateArgs (no CAS support)
- `.kb/investigations/2026-03-01-inv-structural-review-daemon-dedup-after.md` — Structural review
- `.kb/investigations/2026-03-02-design-daemon-launchd-supervision.md` — launchd investigation
- `.kb/decisions/2026-02-15-daemon-unified-config-construction.md` — Config unification

**Related Artifacts:**
- **Principle:** Coherence Over Patches (`~/.kb/principles.md:461-466`) — primary trigger for this design
- **Principle:** Evolve by Distinction (`~/.kb/principles.md:808-810`) — dedup conflates "prevent duplicate" + "prevent thrash" + "monitor health"
- **Constraint:** JSONL lock pileup cascade (from kb context)
- **Constraint:** runDaemonLoop extraction required before adding new subsystems (from kb context)
- **Model:** Daemon Autonomous Operation (`~/.kb/models/daemon-autonomous-operation/model.md`)

---

## Investigation History

**2026-03-05 20:00:** Investigation started
- Initial question: Architect unified reliability design for daemon's three connected structural problems
- Context: Orchestrator identified dedup gauntlet, loop extraction, and operational reliability as connected problems

**2026-03-05 20:30:** Code read complete — 5 major forks identified
- Mapped full daemon structure: 99 files, 30+ struct fields, 16 interfaces
- Confirmed daemonConfigFromFlags and periodic extraction already exist
- Key finding: beads lacks CAS but PID lock makes Go-level CAS sufficient

**2026-03-05 21:00:** All forks navigated
- CAS without beads fork: Go-level CAS + PID lock (no beads changes needed)
- Advisory vs remove: keep as warnings, measure signal
- Cooldown duration: 30min matching orphan interval
- Scheduler pattern: generic Task/Scheduler (proven by existing periodic extraction)
- Supervision: launchd via existing `orch daemon install`

**2026-03-05 21:30:** Investigation completed
- Status: Complete
- Key outcome: 3-phase inside-out simplification. Phase 1 (dedup) highest priority — reduces 245 lines to 60, eliminates compound fail-open gap, fixes orphan-cache contradiction. Phase 2 (scheduler) reduces loop from 697 to ~100 lines. Phase 3 (operational) activates existing launchd infrastructure.
