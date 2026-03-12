## Summary (D.E.K.N.)

**Delta:** The spawn dedup system has a TOCTOU race in the pipeline gates (check-then-mark is not atomic), but production risk is low because the daemon calls Once() sequentially with PID lock preventing multiple instances. The real race window is daemon + manual spawn, mitigated by triage label removal. A data race was found and fixed in session_dedup.go's lazy initialization.

**Evidence:** 8 race condition tests written: concurrent spawnIssue() without status reflection allowed 10/10 goroutines through; WITH status reflection (simulating beads) only 1/10 succeeded. SpawnTrackerGate TOCTOU: 2/100 goroutines passed before first mark. Go race detector caught data race in initDefaultSessionDedupChecker (fixed with sync.Once).

**Knowledge:** The beads status update (FreshStatusGate) is the real structural protection, not the in-memory spawn tracker. The spawn tracker is defense-in-depth for the sequential daemon loop. Under concurrent stress, only beads status serializes access.

**Next:** No immediate action needed — production path is sequential and protected. Long-term: beads CAS (compare-and-swap) would close the manual-spawn TOCTOU structurally.

**Authority:** implementation - Race testing within existing architecture, no cross-boundary changes

---

# Investigation: Test Duplicate Spawn Race Condition

**Question:** Does the orch spawn system have a duplicate spawn race condition? If two spawns target the same issue simultaneously, does only one agent get created?

**Started:** 2026-03-12
**Updated:** 2026-03-12
**Owner:** investigation (orch-go-84uad)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** agent-lifecycle-state-model

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-01-inv-structural-review-daemon-dedup-after.md | extends | Yes — 5-layer pipeline confirmed (was 6, now extracted to SpawnGate interface) | None |
| .kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-ttl-fragility.md | extends | Yes — TTL is 6h as documented | None |
| .kb/investigations/archived/2026-01-15-inv-investigate-daemon-duplicate-spawn-issue.md | extends | Yes — SpawnedIssueTracker exists and functions as designed | None |

---

## Findings

### Finding 1: SpawnTrackerGate Has a TOCTOU Race (Check-Then-Mark Not Atomic)

The spawn pipeline gates check `IsSpawned()` and the daemon later calls `MarkSpawnedWithTitle()` — these are separate operations with a window between them where concurrent callers can all pass.

**Evidence:** `TestSpawnTrackerGate_TOCTOU_Window` — 2/100 goroutines passed the gate before the first one marked the issue. `TestConcurrentSpawnIssue_DirectCall` — all 10 goroutines spawned when mocked status always returns "open".

**Source:** `pkg/daemon/spawn_gate.go:161-173` (Check), `pkg/daemon/spawn_execution.go:98-100` (MarkSpawnedWithTitle)

**Significance:** Under concurrent stress, the spawn tracker alone cannot prevent duplicate spawns. But in production, the daemon calls Once() sequentially (daemon_loop.go:355), so this window never opens within a single daemon instance.

---

### Finding 2: Beads Status Update (FreshStatusGate) IS the Real Protection

When the FreshStatusGate correctly reflects actual state changes (status updated to "in_progress"), it serializes concurrent access effectively.

**Evidence:** `TestConcurrentSpawnIssue_FreshStatusSerializes` — with status reflection enabled, only 1/10 goroutines succeeded in spawning. This is because: goroutine A sets status to "in_progress" via StatusUpdater, goroutine B's FreshStatusGate sees "in_progress" and rejects.

**Source:** `pkg/daemon/spawn_gate.go:257-288` (FreshStatusGate), `pkg/daemon/spawn_execution.go:80` (StatusUpdater.UpdateStatus)

**Significance:** The beads status update in spawnIssue() is the structural backstop. The 5-gate pipeline is defense-in-depth for the sequential daemon loop; beads is what actually prevents duplicates when multiple callers race.

---

### Finding 3: Data Race in session_dedup.go (Fixed)

The `initDefaultSessionDedupChecker()` function used a check-then-set pattern on a package-level variable without synchronization. Go's race detector flagged this under concurrent goroutine access.

**Evidence:** `go test -race` output: `DATA RACE: Read at session_dedup.go:51 ... Write at session_dedup.go:123`. Fixed by replacing manual check-then-set with `sync.Once`.

**Source:** `pkg/daemon/session_dedup.go:121-126` (before fix)

**Significance:** Real bug. Although the daemon calls Once() sequentially, other code paths (tests, potential future concurrent usage) could trigger this. The fix is minimal and correct.

---

### Finding 4: Manual Spawn Path Has TOCTOU (Low Risk)

`SetupBeadsTracking` (spawn_beads.go:38-56) checks issue status then updates it — non-atomic. Two concurrent `orch spawn --issue X` calls could both see "open" before either sets "in_progress".

**Evidence:** Code analysis of spawn_beads.go:38-61. No test for this (shells out to `bd` CLI, not mockable at unit level). The triage label removal at spawn_cmd.go:311 narrows the daemon race window but doesn't eliminate the manual-manual TOCTOU.

**Source:** `pkg/orch/spawn_beads.go:38-61`, `cmd/orch/spawn_cmd.go:306-313`

**Significance:** Low production risk — manual spawns are human-driven (1-2/day), and the workspace random suffix means both agents would get different workspaces (visible, not silently colliding). Still, the structural gap exists.

---

### Finding 5: Sequential Daemon Path Is Correctly Protected

The production daemon loop (daemon_loop.go:334-386) calls `OnceExcluding()` in a sequential `for` loop. PID lock (daemon_loop.go:63) prevents multiple instances. The spawn tracker provides effective dedup for this sequential pattern.

**Evidence:** `TestSequentialOnce_PreventsDuplicate` — second sequential call correctly blocked. `TestSpawnIssue_SequentialBlocksSecondCall` — same result for spawnIssue.

**Source:** `cmd/orch/daemon_loop.go:334-386` (sequential loop), `cmd/orch/daemon_loop.go:63` (PID lock)

**Significance:** The production daemon path has NO duplicate spawn race condition. The TOCTOU exists only under concurrent stress testing.

---

## Synthesis

**Key Insights:**

1. **Sequential daemon path is safe** — PID lock + sequential Once() + spawn tracker = no duplicates. The 5-gate pipeline is defense-in-depth that works correctly for this pattern.

2. **Beads status is the structural serializer** — When multiple callers race, only the FreshStatusGate (backed by actual beads status) prevents duplicates. The in-memory spawn tracker has a TOCTOU that is benign for sequential access but would allow duplicates under concurrent stress.

3. **Manual spawn TOCTOU is the real risk** — Two `orch spawn --issue X` calls or daemon + manual spawn can race. Mitigated by: low frequency, label removal, workspace randomization. Structural fix: beads CAS.

**Answer to Investigation Question:**

The orch spawn system does NOT have a practical duplicate spawn race condition in its primary path (daemon sequential polling). However, it has a theoretical TOCTOU in the pipeline gates that manifests under concurrent stress testing, and a low-risk TOCTOU in the manual spawn path. The beads status update is the structural protection that prevents actual duplicates in production. A real data race bug was found and fixed in session_dedup.go's lazy initialization.

---

## Structured Uncertainty

**What's tested:**

- ✅ Concurrent Once() TOCTOU exists: 10/10 goroutines spawned without status reflection
- ✅ FreshStatusGate serializes: only 1/10 succeeded with status reflection
- ✅ SpawnTrackerGate TOCTOU: 2/100 passed before first mark
- ✅ Sequential path correctly blocks duplicates (two test cases)
- ✅ session_dedup.go data race fixed (go test -race passes cleanly)
- ✅ SpawnTracker concurrent operations are thread-safe

**What's untested:**

- ⚠️ End-to-end daemon + manual spawn race (would require running actual `orch` processes)
- ⚠️ Whether beads UpdateStatus is truly atomic under concurrent access (shells out to bd CLI)
- ⚠️ Whether two tmux windows can be created for the same beads ID in practice

**What would change this:**

- If beads supported CAS semantics, the manual spawn TOCTOU could be closed structurally
- If daemon were changed to call Once() concurrently (e.g., parallel spawning), the TOCTOU would become a production issue
- If multiple daemon instances could run (PID lock failure), duplicates would occur

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| session_dedup.go sync.Once fix | implementation | Single-file bug fix, no behavioral change |
| Race condition test suite | implementation | Tests within existing architecture |
| Beads CAS for manual spawn path | architectural | Cross-component (beads + orch), design choice |

### Recommended Approach ⭐

**Keep current architecture + sync.Once fix + race tests** — The session_dedup.go fix (already applied) closes the real data race. The race condition tests document the TOCTOU boundaries and validate sequential dedup. No structural changes needed.

**Why this approach:**
- Production path (daemon sequential) is already correctly protected
- The TOCTOU in pipeline gates is benign for sequential access
- Manual spawn TOCTOU risk is low (1-2 spawns/day, human-driven)

**Trade-offs accepted:**
- Manual spawn TOCTOU remains open (acceptable given low frequency)
- No beads CAS (would require beads fork changes)

### Alternative Approaches Considered

**Option B: Add mutex around spawnIssue pipeline**
- **Pros:** Eliminates TOCTOU in spawnIssue()
- **Cons:** Adds contention for a sequential path, no production benefit
- **When to use instead:** If daemon evolves to parallel spawning

**Option C: Implement beads CAS**
- **Pros:** Structural fix for all race windows including manual spawn
- **Cons:** Requires beads fork changes, higher complexity
- **When to use instead:** If manual spawn duplicates become a production issue

---

## References

**Files Examined:**
- `pkg/daemon/spawn_tracker.go` — SpawnedIssueTracker: ID-based, title-based, and count-based dedup
- `pkg/daemon/spawn_execution.go` — spawnIssue(): 5-gate pipeline + status update + spawn call
- `pkg/daemon/spawn_gate.go` — SpawnPipeline: SpawnTrackerGate, SessionDedupGate, TitleDedupMemory/Beads, FreshStatusGate
- `pkg/daemon/session_dedup.go` — Session dedup checker with data race fix
- `pkg/orch/spawn_beads.go` — SetupBeadsTracking: manual spawn path with TOCTOU
- `cmd/orch/spawn_cmd.go` — Spawn command with triage label removal
- `cmd/orch/daemon_loop.go` — Sequential daemon loop with PID lock

**Commands Run:**
```bash
# Race condition tests with Go race detector
go test -race -run "TestConcurrent|TestSpawnTracker|TestSequential|TestSpawnIssue_Sequential|TestManualSpawn" -v ./pkg/daemon/

# Full daemon package test suite
go test ./pkg/daemon/
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-01-inv-structural-review-daemon-dedup-after.md` — Prior structural review of 6-layer dedup
- **Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-ttl-fragility.md` — TTL fragility analysis

---

## Investigation History

**2026-03-12:** Investigation started
- Initial question: Does the orch spawn system have a duplicate spawn race condition?
- Context: Orchestrator task to test concurrent spawn dedup

**2026-03-12:** Code analysis complete
- Traced 5-gate pipeline in spawnIssue(), identified TOCTOU between Check and Mark
- Identified data race in session_dedup.go initDefaultSessionDedupChecker

**2026-03-12:** Race tests written and executed
- 8 test cases covering concurrent, sequential, and TOCTOU scenarios
- Key finding: beads status is the real serializer, spawn tracker is defense-in-depth

**2026-03-12:** session_dedup.go fix applied
- Replaced manual check-then-set with sync.Once
- All tests pass with -race flag, no data races

**2026-03-12:** Investigation completed
- Status: Complete
- Key outcome: No production duplicate spawn risk (sequential daemon + PID lock). TOCTOU exists under concurrent stress but beads status prevents actual duplicates.
