<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Prevent orphaned bun/orch accumulation with a two-tier process lifecycle design: deterministic startup sweep plus periodic daemon reaping, backed by persistent child-process lineage.

**Evidence:** The daemon already runs periodic orphan reaping and server-restart recovery, but spawn paths still create detached child processes that can outlive parents, and `orch serve` startup currently has no stale-process sweep.

**Knowledge:** The failure is not missing one cleanup call; it is a cross-layer lifecycle gap between process creation, process ownership, and restart-time reconciliation.

**Next:** Implement shared startup sweep + persistent process ledger + stricter orphan qualification, then validate with crash/restart reliability tests.

**Authority:** architectural - The fix spans `orch serve`, daemon lifecycle, spawn process management, and OpenCode restart behavior.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Design Process Lifecycle Cleanup Prevent

**Question:** How should orch and OpenCode enforce process lifecycle cleanup so bun/orch agent processes cannot accumulate after crashes/restarts, including startup strategy, periodic reaping strategy, and `run --attach` parent-death handling?

**Started:** 2026-02-08
**Updated:** 2026-02-08
**Owner:** architect worker (orch-go-21504)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->

**Patches-Decision:** `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md`
**Extracted-From:** N/A

## Prior Work

| Investigation                                                                               | Relationship | Verified                                                 | Conflicts    |
| ------------------------------------------------------------------------------------------- | ------------ | -------------------------------------------------------- | ------------ |
| `.kb/investigations/archived/2026-01-11-design-opencode-session-cleanup-mechanism.md`       | extends      | Yes - verified against current daemon and process code   | No conflicts |
| `.kb/investigations/archived/2026-02-07-inv-process-census-false-positives.md`              | deepens      | Yes - verified orphan classification logic               | No conflicts |
| `.kb/investigations/archived/2026-02-07-inv-system-reliability-crisis-diagnosis-and-fix.md` | confirms     | Yes - verified C2/process-leak mechanism in current code | No conflicts |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: The system already has partial two-tier lifecycle recovery, but not at all process entry points

**Evidence:** `runDaemonLoop()` executes subsystem checks every poll cycle and invokes `ReapOrphanProcesses()` and `RunServerRecovery()`; orphan reaping is enabled by default and runs on first eligible cycle because `lastOrphanReap` starts zero; server recovery also handles server down->up restart detection and resumed sessions.

**Source:** `cmd/orch/daemon.go:299`, `cmd/orch/daemon.go:338`, `cmd/orch/daemon_loop.go:303`, `cmd/orch/daemon_loop.go:389`, `pkg/daemon/orphan_reaper.go:35`, `pkg/daemon/recovery.go:346`, `pkg/daemon/recovery.go:389`, `pkg/daemon/daemon_lifecycle.go:271`.

**Significance:** This confirms periodic cleanup exists and should be kept, but recurrence means this tier alone is insufficient when startup gaps, classification gaps, or ownership gaps leave processes unclaimed long enough to accumulate.

---

### Finding 2: Orphan qualification currently relies on weak identity matching (title parsing) instead of durable lineage

**Evidence:** The daemon orphan reaper calls `getActiveSessionTitles()` then `FindOrphanProcesses(activeTitles)`; process matching is based on command containing `bun` and `run --attach`, then title fragments from `--title` and bracketed beads IDs. Any process lacking parseable title linkage, or using changed command format, can be misclassified or missed.

**Source:** `pkg/daemon/orphan_reaper.go:47`, `pkg/daemon/orphan_reaper.go:57`, `pkg/daemon/orphan_reaper.go:97`, `pkg/process/orphans.go:24`, `pkg/process/orphans.go:45`, `pkg/process/orphans.go:67`, `pkg/process/orphans.go:91`.

**Significance:** Cleanup reliability is bounded by naming conventions rather than authoritative ownership records; this is fragile under process format changes and restart race windows.

---

### Finding 3: Spawn paths create child processes without durable parent-death semantics

**Evidence:** Headless spawn starts an OpenCode subprocess, writes `.process_id`, and launches an in-process goroutine to drain stdout/wait. If parent `orch` process dies/restarts, that goroutine disappears while the child process can continue as orphan. Cleanup support exists in workspace metadata but not as guaranteed parent-death coupling.

**Source:** `cmd/orch/spawn_cmd.go:523`, `cmd/orch/spawn_cmd.go:531`, `cmd/orch/spawn_cmd.go:629`, `cmd/orch/spawn_cmd.go:654`, `cmd/orch/spawn_cmd.go:677`, `pkg/process/terminate.go:12`, `.kb/decisions/2026-02-07-unbounded-resource-consumption-constraints.md:26`.

**Significance:** This is the C2 defect class directly: process creation is not tied to guaranteed cleanup on parent death, so orphan growth remains possible even with periodic janitors.

---

## Synthesis

**Key Insights:**

1. **Startup + periodic is required (not either/or)** - Existing periodic reaping proves useful but cannot guarantee bounded orphan count during restart windows; startup sweep is required to close the "before first periodic pass" gap.

2. **Identity by title is insufficient** - Process ownership must move from heuristic string parsing to a persisted lineage contract keyed by PID/process-group/workspace/session/beads.

3. **Cleanup must follow the four-layer model** - Process cleanup must reconcile OpenCode memory, OpenCode disk, workspace artifacts, and runtime processes together; fixing only one layer recreates ghosts.

**Answer to Investigation Question:**

Use a two-tier process lifecycle architecture with stronger ownership semantics: (1) run a deterministic stale-process sweep at `orch serve` startup using a shared reaper contract, (2) keep daemon periodic orphan reaping as the safety net, and (3) add persistent child-process lineage so restart logic can distinguish "owned active child" from "orphaned leftover" without relying on title parsing. For `run --attach` children, enforce parent-death semantics (process-group tracking + restart-time reaping of stale groups) so child lifetimes are bounded even when parent dies abruptly. This answer is supported by Findings 1-3 and aligns with existing two-tier cleanup and C2 reliability constraints; open implementation details remain around exact lineage schema and cross-platform signaling behavior.

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon runs orphan reaping periodically and on first eligible cycle (verified by reading active `runDaemonLoop()` + `runSubsystems()` + `ReapOrphanProcesses()` code paths).
- ✅ Server-restart recovery already exists and is triggered on down->up transitions (verified by reading `ServerRecoveryState` restart detection and recovery trigger logic).
- ✅ Spawn path persists PID metadata but relies on in-process cleanup goroutine (verified by reading `spawn_cmd.go` headless spawn and `StartBackgroundCleanup()`).

**What's untested:**

- ⚠️ Exact false-positive/false-negative rate of current orphan matching under production process-name variance (not stress-tested in this session).
- ⚠️ Whether OpenCode itself can expose child lineage hooks to avoid external process introspection (not validated against OpenCode internals in this repo session).
- ⚠️ End-to-end restart storm behavior with proposed startup sweep + periodic reaper under concurrent spawns (requires reliability testing harness).

**What would change this:**

- If runtime testing shows current title-based matching catches all orphan cases with zero drift, lineage hardening priority could drop.
- If startup sweep introduces measurable startup latency or kills legitimate processes, sweep scope must narrow to process groups with verified ownership markers.
- If OpenCode introduces authoritative child-process ownership APIs, orch-side lineage persistence should be replaced by direct API-backed reconciliation.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation                                                                                                     | Authority     | Rationale                                                                              |
| ------------------------------------------------------------------------------------------------------------------ | ------------- | -------------------------------------------------------------------------------------- |
| Add startup stale-process sweep in `orch serve` using shared orphan detector with strict ownership markers         | architectural | Crosses serve startup, daemon ownership model, and process package behavior.           |
| Keep daemon periodic orphan reaper as tier-2 safety net, but tighten orphan qualification to lineage-backed checks | architectural | Changes daemon policy and process identity contracts across components.                |
| Add persistent child-process ledger for spawned agents and reconcile it on restart                                 | architectural | Requires shared schema used by spawn, cleanup, and reaper paths.                       |
| Add parent-death semantics for attach-mode child processes (process-group ownership + forced reap policy)          | architectural | Affects spawn mechanics and restart-time cleanup behavior across lifecycle boundaries. |

**Authority Levels:**

- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"

- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Lineage-backed two-tier process cleanup** - Pair deterministic startup sweep with periodic reaping and persistent child ownership records so orphaned agent processes cannot survive restart boundaries.

**Why this approach:**

- Eliminates the immediate restart window where stale processes can run before periodic cleanup catches up.
- Replaces fragile string-based process identification with durable ownership lineage.
- Preserves existing safety-net behavior (daemon periodic reaper) while making it precise instead of heuristic.

**Trade-offs accepted:**

- More implementation complexity (startup reconciliation + ledger schema + migration paths).
- Slight startup overhead for `orch serve` and daemon due to one-time process census.

**Implementation sequence:**

1. Add shared ownership marker + ledger write path at spawn time (foundation for deterministic classification).
2. Add startup sweep in `orch serve` and daemon boot path using the same classifier (close immediate restart gap).
3. Tighten periodic reaper to require lineage mismatch before kill, then run reliability soak against restart/crash scenarios.

### Alternative Approaches Considered

**Option B: Periodic-only reaping (status quo with interval tuning)**

- **Pros:** Minimal implementation work; no new persistence schema.
- **Cons:** Leaves startup/restart accumulation windows and keeps heuristic matching fragility.
- **When to use instead:** Temporary emergency mitigation while implementing full lineage-backed model.

**Option C: Startup-only cleanup (no periodic reaper)**

- **Pros:** Reduces daemon runtime process scans.
- **Cons:** Fails two-tier cleanup principle; misses orphans introduced between restarts.
- **When to use instead:** Only in constrained environments where periodic scans are forbidden and external supervisor guarantees cleanup.

**Rationale for recommendation:** Option A is the only design that satisfies both proven two-tier cleanup requirements and C2 process-lifecycle enforcement under crash/restart conditions.

---

### Implementation Details

**What to implement first:**

- Introduce a shared process ownership descriptor (workspace, beads_id, session_id, spawn_pid, child_pid, pgid, started_at, last_seen).
- Refactor orphan detection to prefer ownership descriptor reconciliation over title parsing.
- Add startup reconciliation entrypoint invoked by `orch serve` and daemon initialization.

**Things to watch out for:**

- ⚠️ Do not kill launchd-managed or intentionally long-running processes (keep existing whitelist behavior and require ownership evidence).
- ⚠️ Ensure no race between new spawn writing ownership record and startup sweep reading it.
- ⚠️ Avoid cross-project contamination in process matching; ownership markers must include project path normalization.

**Areas needing further investigation:**

- Evaluate whether OpenCode can expose authoritative child-process metadata to avoid external `ps` parsing.
- Validate parent-death signaling strategy per platform (darwin/Linux) for attach-mode children.
- Add reliability benchmark for restart storms and measure orphan count convergence over time.

**Success criteria:**

- ✅ Restarting `orch serve` or OpenCode leaves zero eligible stale agent processes after startup sweep.
- ✅ Periodic reaper reports stable near-zero orphan count over 7-day run under mixed crash/restart workload.
- ✅ Operator health process census and daemon orphan metrics agree within expected tolerance (no contradictory orphan visibility).

---

## References

**Files Examined:**

- `cmd/orch/daemon.go` - Verified main loop order and subsystem invocation timing.
- `cmd/orch/daemon_loop.go` - Verified periodic reaper and server-recovery execution paths.
- `pkg/daemon/orphan_reaper.go` - Verified orphan detection/reaping flow and current active-session matching approach.
- `pkg/process/orphans.go` - Verified process discovery and title-based orphan classification heuristics.
- `cmd/orch/spawn_cmd.go` - Verified process creation and current cleanup mechanics for spawned children.
- `pkg/daemon/recovery.go` - Verified server restart detection and orphaned-session recovery behavior.
- `.kb/models/system-reliability-feb2026.md` - Verified C2 reliability context and recurrence framing.
- `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md` - Verified required cleanup architecture precedent.
- `.kb/decisions/2026-02-07-unbounded-resource-consumption-constraints.md` - Verified C2 enforcement intent.

**Commands Run:**

```bash
# Report phase and recovery progress
orch phase orch-go-21504 Planning "Framing lifecycle cleanup options and locating relevant process-management code"
orch phase orch-go-21504 Planning "Recovered after server restart; resumed analysis and validating prior findings before synthesis"
orch phase orch-go-21504 Implementing "Documenting lifecycle fork analysis and drafting architecture recommendations"

# Investigation artifact + context loading
kb create investigation design-process-lifecycle-cleanup-prevent
kb context "process lifecycle cleanup orphan reaper opencode"
```

**External Documentation:**

- N/A (analysis based on in-repo primary sources and existing knowledge artifacts).

**Related Artifacts:**

- **Decision:** `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md` - Establishes startup+periodic cleanup architecture baseline.
- **Decision:** `.kb/decisions/2026-02-07-unbounded-resource-consumption-constraints.md` - Defines C2 process lifecycle enforcement requirement.
- **Investigation:** `.kb/investigations/archived/2026-01-11-design-opencode-session-cleanup-mechanism.md` - Prior cleanup mechanism exploration extended by this design.
- **Workspace:** `.orch/workspace/og-arch-design-process-lifecycle-08feb-3cf2/` - Session workspace for this architecture task.

---

## Investigation History

**[2026-02-08 20:16]:** Investigation started

- Initial question: Design process lifecycle cleanup to prevent orphaned bun/orch processes under restart/crash conditions.
- Context: Third recurrence of orphan accumulation with high CPU impact; scope includes startup cleanup, periodic cleanup, restart reaping, and attach child-parent death semantics.

**[2026-02-08 20:33]:** Core lifecycle paths validated

- Confirmed existing daemon periodic orphan reaper + server restart recovery paths and identified remaining ownership/qualification gaps.

**[2026-02-08 20:46]:** Investigation completed

- Status: Complete
- Key outcome: Recommended lineage-backed two-tier cleanup design with startup sweep + periodic reaper + parent-death process ownership hardening.
