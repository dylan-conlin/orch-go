## Summary (D.E.K.N.)

**Delta:** Daemon dedup has 6 independent layers accumulated via 9 tactical fixes, but they lack a coherent invariant model — the "no duplicates" guarantee depends on no layer having a gap, yet layers contradict each other (orphan detector retains cache to block respawns, but ReconcileWithIssues would clear those entries if called).

**Evidence:** Code trace of `spawnIssue()` (pkg/daemon/daemon.go:620-864) shows 6 sequential dedup checks with overlapping but inconsistent TTLs, fail-open/fail-fast inconsistencies, and one dead-code method (ReconcileWithIssues). Each layer was added to patch a gap in the previous one.

**Knowledge:** The dedup system is heuristic-first (TTL cache, session polling, title matching) rather than structural (single source of truth with atomic state transitions). This produces the "accumulation problem" documented in the heuristic-vs-structural guide — each fix is locally correct but the composite system has emergent gaps.

**Next:** Architect follow-up to redesign dedup around a single structural invariant: beads status as sole source of truth with atomic CAS (compare-and-swap) semantics. Fallback layers become purely advisory (defense-in-depth monitoring, not enforcement).

**Authority:** architectural - Cross-component redesign affecting daemon, beads, and spawn paths; requires orchestrator synthesis.

---

# Investigation: Structural Review of Daemon Dedup After 7+ Tactical Fixes

**Question:** What dedup invariants must hold for no duplicates, how do the 6 layers interact, what race windows remain, and should the next fix be another patch or a structural redesign?

**Defect-Class:** race-condition

**Started:** 2026-03-01
**Updated:** 2026-03-01
**Owner:** architect (orch-go-n77b)
**Phase:** Complete
**Next Step:** None — architect follow-up recommended
**Status:** Complete

**Patches-Decision:** N/A (no prior decision document for dedup — that's part of the problem)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-27-audit-daemon-code-health-complexity.md | extends | Yes - confirmed Daemon struct at 93 fields, pkg/daemon 30 files | None |
| 2026-01-06-inv-synthesize-daemon-investigations.md | deepens | Yes - confirmed SpawnedIssueTracker origin from Jan 2026 | Daemon guide still documents 5-min TTL (line 168) but code uses 6h |
| Probe: Phantom Agent Spawns (2026-02-27) | extends | Yes - stale file bug is orthogonal to this analysis | None |

---

## Findings

### Finding 1: Six Independent Dedup Layers With No Unifying Invariant

The `spawnIssue()` function (daemon.go:620-864) executes 6 sequential dedup checks, each added by a different fix. Here is the complete dedup pipeline, in execution order:

| Layer | Check | Location | Added By | Fail Mode | Nature |
|-------|-------|----------|----------|-----------|--------|
| L1 | SpawnedIssueTracker (ID) | daemon.go:621-636 | 48b850cca (Jan) + 142c23ae2 (Mar) | Blocks spawn | Heuristic (6h TTL) |
| L2 | Session/Tmux existence | daemon.go:638-655 | 674912984 (Jan) + ba6b612fd (Feb) | Blocks spawn | Heuristic (API poll) |
| L3 | Title dedup (in-memory) | daemon.go:657-677 | d29a3c8af (Feb) | Blocks spawn | Heuristic (TTL-coupled) |
| L4 | Title dedup (beads DB) | daemon.go:678-691 | d29a3c8af (Feb) | Blocks spawn (fail-open) | Structural-ish |
| L5 | Fresh status re-check | daemon.go:693-724 | 4e6609909 (Feb) | Blocks spawn (fail-open) | Structural |
| L6 | UpdateStatus("in_progress") | daemon.go:742-784 | c96d78f96 (Feb) | **Fail-fast** | Structural (PRIMARY) |

Additionally, `NextIssueExcluding()` (daemon.go:258-358) has its own L1 check at line 291, creating a **redundant L1 check** (explicitly called defense-in-depth).

**Evidence:** Each layer has its own implicit invariant:
- L1: "If spawned within 6h, don't respawn" (TTL-based, no semantic meaning)
- L2: "If a session/window exists, don't spawn" (infrastructure-coupled)
- L3: "If same title spawned recently, don't spawn" (heuristic, false positives possible)
- L4: "If same title is in_progress in beads, don't spawn" (DB-backed but fail-open)
- L5: "If status != open, don't spawn" (structural, but TOCTOU race remains)
- L6: "If can't set in_progress, don't spawn" (structural, fail-fast)

**Source:** `pkg/daemon/daemon.go:620-864`

**Significance:** No single invariant guarantees no duplicates. The guarantee emerges from the *composition* of 6 layers, each added to patch a gap in the previous layer. This is the "Coherence Over Patches" anti-pattern — the 9th fix to the same area.

---

### Finding 2: Orphan Detection and Spawn Cache Have Contradictory Lifecycle Semantics

The orphan detector (`orphan_detector.go:71-164`) resets issues from `in_progress` → `open` but **intentionally does NOT** clear the spawn cache entry (lines 141-146, added by 142c23ae2). This means:

1. Issue is spawned → marked in spawn cache (6h TTL)
2. Agent dies → no session/window exists
3. Orphan detector runs after 1h → resets status to `open`
4. Issue is now `open` in beads but **blocked** in spawn cache for up to 5 more hours

This was the intentional fix for thrash loops. But it creates a new problem: legitimate retries are blocked for the remainder of the 6h TTL. The only escape valves are:
- TTL expiry (up to 6h wait)
- Manual `Unmark()` (no UI for this)
- Daemon restart with stale cache cleared

Meanwhile, `ReconcileWithIssues()` (spawn_tracker.go:307-332) would clear entries for issues that transitioned to `in_progress` — but **this method is never called from production code**. It exists only in tests. If it were called, it would counteract the orphan detector fix.

**Evidence:** `ReconcileWithIssues` defined at spawn_tracker.go:307, called only in spawn_tracker_test.go:85,97. `ReconcileActiveAgents` (capacity.go:87-106) calls `CleanStale()` but not `ReconcileWithIssues`.

**Source:** `pkg/daemon/orphan_detector.go:141-146`, `pkg/daemon/spawn_tracker.go:307-332`, `pkg/daemon/capacity.go:87-106`

**Significance:** The spawn cache lifecycle has no clear semantic — it's neither "block until agent finishes" nor "block until status updates." It's "block for a fixed duration regardless of what actually happened." This is a TTL-as-policy anti-pattern.

---

### Finding 3: Multiple Fail-Open Checks Create Compound Gap Risk

Four of the six layers are fail-open (allow spawn on error):

| Layer | Failure Behavior | Risk |
|-------|-----------------|------|
| L2 (session check) | OpenCode API down → allow spawn | If both API + tmux fail, all session dedup bypassed |
| L3 (title in-memory) | N/A (always available) | Only same-process, lost on restart |
| L4 (title beads DB) | beads socket/CLI fail → allow spawn | Silent duplicate when beads unavailable |
| L5 (fresh status) | beads query fail → allow spawn | TOCTOU: another process can win the race |

L6 (UpdateStatus) is the only fail-fast layer. But even L6 has a gap: `UpdateBeadsStatus(id, "in_progress")` is **idempotent** — if two processes race, both succeed. The fresh status check (L5) was added specifically for this, but L5 is fail-open.

The compound risk: if beads is unavailable (e.g., beads daemon not running, which is the configured default per `BEADS_NO_DAEMON=1` in the launchd plist), then L4, L5, and L6 all degrade:
- L4 fails open (no DB to query)
- L5 fails open (no DB to query)
- L6: unclear — does UpdateBeadsStatus work without beads daemon?

**Evidence:** Code trace of error handling in daemon.go:699-723 (L5 fail-open), daemon.go:767-784 (L6 fail-fast), session_dedup.go:75-77 (L2 fail-open)

**Source:** `pkg/daemon/daemon.go:620-864`, `pkg/daemon/session_dedup.go:69-98`

**Significance:** The system has defense-in-depth but the failure modes are positively correlated — beads unavailability degrades multiple layers simultaneously, which is exactly when dedup matters most (post-restart, infrastructure instability).

---

### Finding 4: SpawnCounts Grow Unboundedly

The `spawnCounts` map (spawn_tracker.go:42) tracks how many times each issue has been spawned. Unlike `spawned` entries (cleaned by `CleanStale()` when TTL expires), spawn counts are **never cleaned**. They accumulate forever and are persisted to disk.

**Evidence:** `CleanStale()` (spawn_tracker.go:280-301) iterates `spawned` and deletes stale entries, plus cleans `spawnedTitles` for orphaned entries. It does NOT clean `spawnCounts`. No other code cleans them.

**Source:** `pkg/daemon/spawn_tracker.go:280-301`

**Significance:** Minor — the count map grows slowly (one entry per unique issue ID). But it's symptomatic of the broader pattern: features are added without lifecycle management. The spawn cache file will grow indefinitely.

---

### Finding 5: Daemon Guide Is Stale on TTL Value

The daemon guide (`.kb/guides/daemon.md:168`) documents the SpawnedIssueTracker TTL as 5 minutes. The actual TTL has been 6 hours since commit 674912984 (January 2026). The guide's code example at line 173-179 also still shows the `Mark()`/`Unmark()` pattern, which no longer reflects current behavior (orphan detector no longer calls `Unmark()`).

**Evidence:** daemon.md line 168: "5-minute TTL allows entries to expire naturally" vs spawn_tracker.go line 65: `TTL: 6 * time.Hour`

**Source:** `.kb/guides/daemon.md:159-179`, `pkg/daemon/spawn_tracker.go:61-72`

**Significance:** Stale documentation is a session-amnesia violation — the next agent investigating dedup bugs will be misled by the guide. The guide should be authoritative but it lags the code by 2 months.

---

### Finding 6: The `spawnIssue()` Function Is a 245-Line Gauntlet

`spawnIssue()` (daemon.go:620-864) runs 245 lines of sequential dedup checks, pool management, status updates, spawn execution, and rollback handling. It has:
- 6 dedup layers
- 3 early returns for dedup rejection
- Pool slot acquisition/release
- Beads status update with rollback
- Spawner invocation with failure tracking
- Rate limiter recording
- Title tracking for content dedup

This violates the accretion boundary constraint (CLAUDE.md: "Files >1,500 lines require extraction"). While daemon.go overall is under the limit, this single function concentrates the entire dedup logic with no separation of concerns.

**Evidence:** Line count: daemon.go has 903 lines total; `spawnIssue` is 245 lines (27% of the file). Prior audit (orch-go-ajay) identified this as a risk area.

**Source:** `pkg/daemon/daemon.go:620-864`

**Significance:** The function is difficult to reason about because dedup, capacity, spawning, and rollback are all interleaved. Each tactical fix adds to this function. Extracting dedup into a composable pipeline would make invariants explicit and testable.

---

## Synthesis

**Key Insights:**

1. **The system is heuristic-first, not structural-first.** The primary dedup mechanism evolved from a TTL cache (heuristic) with layers of heuristic checks bolted on. The structural check (beads status) was added 4th, but in a fail-open mode. The system should be inverted: structural first (beads status as authoritative), heuristic as monitoring/alerting.

2. **Fail-open failures are positively correlated.** When the infrastructure that dedup depends on is degraded (beads unavailable, OpenCode down), multiple layers fail simultaneously. This is when duplicates are most likely. The defense-in-depth architecture assumes independent failure modes, but the dependencies are shared.

3. **The coherence-over-patches principle applies.** 9 commits modified dedup logic over 2 months. Each was locally correct. The composite system has emergent gaps (orphan-cache contradiction, compound fail-open, unbounded counts). The next tactical fix will add a 7th layer without resolving the structural issue.

**Answer to Investigation Question:**

The dedup system's invariant for "no duplicates" is: **at least one of 6 layers must reject the spawn for every previously-spawned issue.** This is an implicit, negative invariant — it's defined by what must NOT happen rather than what MUST be true. The correct invariant should be positive: **an issue can only be spawned if its beads status is `open` AND no active agent (session/window) exists for it, verified atomically.**

The race handling vs redesign boundary should be:
- **Race handling (tactical):** Anything that closes a gap within the existing 6-layer architecture (adding another check, fixing a TTL)
- **Redesign (structural):** Replacing the 6-layer gauntlet with a pipeline that enforces the positive invariant through atomic state transitions

The system has hit the heuristic-vs-structural tipping point described in the guide: 9+ fixes in the same area, with each fix adding complexity rather than removing it.

---

## Structured Uncertainty

**What's tested:**

- ✅ 6 dedup layers exist in spawnIssue(), verified by code trace of daemon.go:620-864
- ✅ ReconcileWithIssues() is dead code — grep confirms no production callers
- ✅ spawnCounts map has no cleanup — traced CleanStale() which only cleans `spawned` and `spawnedTitles`
- ✅ Daemon guide TTL value is stale — daemon.md says 5min, code says 6h
- ✅ Orphan detector intentionally does NOT call Unmark() — explicit comment at orphan_detector.go:141-146

**What's untested:**

- ⚠️ Whether beads UpdateStatus works without beads daemon running (BEADS_NO_DAEMON=1 env var set in launchd plist)
- ⚠️ Whether compound fail-open has actually caused a production duplicate (the individual layer gaps are real, but the compound scenario may be unlikely in practice)
- ⚠️ Whether beads supports CAS semantics or could be extended to support them

**What would change this:**

- If beads UpdateStatus uses file-locking that provides atomic CAS, then L6 alone might be sufficient and the redesign scope shrinks significantly
- If duplicate spawns have stopped occurring since 142c23ae2, the urgency of structural redesign decreases (though the maintenance burden remains)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Structural dedup redesign | architectural | Cross-component (daemon, beads, spawn), multiple valid approaches, requires synthesis |
| Daemon guide update | implementation | Factual error in documentation, no judgment needed |
| spawnCounts cleanup | implementation | Minor leak, single-scope fix |

### Recommended Approach ⭐

**Dedup Pipeline with Structural Primary + Heuristic Advisory** — Replace the 6-layer gauntlet in `spawnIssue()` with a 2-phase pipeline:

**Phase 1: Structural Gate (fail-fast, authoritative)**
- Atomic beads status transition: `open` → `in_progress` with CAS semantics
- If CAS fails (another process won), abort — no spawn
- This single check replaces L5 (fresh status) and L6 (update status), and makes them atomic

**Phase 2: Advisory Checks (warn-only, non-blocking after Phase 1)**
- Session/window existence check (L2) — warn if session found but CAS succeeded (stale session?)
- Title dedup (L3/L4) — warn if title match found (content duplicate worth human review?)
- Spawn count (L1 enhanced) — warn if issue spawned 3+ times (thrashing?)

**Why this approach:**
- Single invariant: "CAS succeeded" = spawn is authorized. All other checks are informational.
- Eliminates fail-open compound gaps — the structural gate is authoritative.
- Makes the spawnIssue function testable: Phase 1 is a single mock point, Phase 2 is pure logging.
- Aligns with heuristic-vs-structural guide's recommendation: structural primary, heuristic fallback.

**Trade-offs accepted:**
- Requires beads to support CAS or equivalent (may need fork enhancement)
- Removes the redundant defense-in-depth checks that currently mask bugs (makes bugs more visible, which is good long-term)
- Orphan respawn timing changes: currently blocked for 6h TTL; with CAS, orphan detection can immediately re-authorize spawn by resetting status to `open` (since CAS is the gate, not a TTL)

**Implementation sequence:**
1. **Investigate beads CAS support** — Can beads do `update status where current_status = 'open'`? If not, what's the minimal fork change?
2. **Extract dedup pipeline** — Move dedup logic from `spawnIssue()` into `pkg/daemon/dedup.go` as composable checks with clear types
3. **Implement CAS gate** — Replace L5+L6 with atomic CAS. Fail-fast.
4. **Demote remaining layers to advisory** — L1-L4 become warning logs, not spawn blockers
5. **Update daemon guide** — Fix stale TTL, document new invariant model
6. **Add spawnCounts cleanup** — Clean counts older than 30 days during `CleanStale()`

### Alternative Approaches Considered

**Option B: Add CAS Without Restructuring**
- **Pros:** Minimal code change, no extraction needed
- **Cons:** Adds 7th layer to the gauntlet, doesn't reduce complexity, coherence-over-patches still applies
- **When to use instead:** If beads CAS is trivially available and urgency is high (active duplicate production incidents)

**Option C: Remove Heuristic Layers Entirely, Trust Beads Only**
- **Pros:** Maximum simplicity — one check, one gate
- **Cons:** Beads unavailability would remove ALL dedup protection; need beads to be very reliable
- **When to use instead:** When beads has proven 99.9%+ availability and atomic operations

**Rationale for recommendation:** Option A gives the structural guarantee (no more heuristic accumulation) while keeping advisory visibility (spawn counts, title matches) that has operational value. It's the "structural with fallback" pattern from the heuristic-vs-structural guide.

---

### Implementation Details

**What to implement first:**
- Daemon guide update (fix stale TTL, document current 6-layer architecture) — quick win, prevents misleading future agents
- SpawnCounts cleanup in CleanStale() — prevents unbounded growth
- Beads CAS investigation — determines feasibility of the structural redesign

**Things to watch out for:**
- ⚠️ The orphan detector's "retain cache entry" behavior (142c23ae2) was a critical fix for overnight thrash loops. Any redesign must preserve the cooldown semantics, even if implemented differently (e.g., explicit cooldown flag vs TTL)
- ⚠️ ReconcileWithIssues() is dead code but tested — if removed, understand why it was built and whether the design it represents was intentionally abandoned or forgotten
- ⚠️ Cross-project dedup: `GetBeadsIssueStatusForProject()` path (daemon.go:702-706) adds project-dir awareness to L5. CAS gate must handle this.

**Areas needing further investigation:**
- Beads CAS semantics: Does beads support conditional updates? If not, what's the minimal change?
- Production duplicate frequency: Are duplicates still occurring post-142c23ae2? If not, the urgency shifts to maintenance burden vs active bugs.
- Spawn cache file growth: How large is `~/.orch/spawn_cache.json` in practice? Is unbounded spawnCounts actually a problem or theoretical?

**Success criteria:**
- ✅ Single invariant for "no duplicates" that can be stated in one sentence
- ✅ No fail-open checks in the critical path (all fail-open checks are advisory only)
- ✅ `spawnIssue()` function under 100 lines (dedup extracted to pipeline)
- ✅ Daemon guide accurate for current behavior
- ✅ No new duplicate spawn incidents for 2 weeks after deployment

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` — Main spawn pipeline with 6 dedup layers (620-864)
- `pkg/daemon/spawn_tracker.go` — SpawnedIssueTracker with TTL, title, and count tracking
- `pkg/daemon/orphan_detector.go` — Orphan detection with intentional cache retention
- `pkg/daemon/session_dedup.go` — OpenCode + tmux session existence checking
- `pkg/daemon/capacity.go` — ReconcileActiveAgents and ReconcileWithOpenCode
- `pkg/daemon/issue_adapter.go` — FindInProgressByTitle for content-aware dedup
- `cmd/orch/daemon.go` — Poll loop, config flags, periodic task orchestration
- `.kb/guides/daemon.md` — Daemon reference guide (stale on TTL)
- `~/.kb/guides/heuristic-vs-structural.md` — Framework for when to go structural
- `~/.kb/principles.md` — Coherence Over Patches, Evolve by Distinction

**Commands Run:**
```bash
# Trace dedup fix history (9 commits)
git log --oneline 48b850cca..142c23ae2 -- pkg/daemon/spawn_tracker.go pkg/daemon/orphan_detector.go pkg/daemon/daemon.go

# Find all dedup-related commits
git log --oneline --all --grep="dedup\|duplicate\|spawn.*cache\|spawn.*track\|orphan" | head -30

# Check for dead code (ReconcileWithIssues callers)
grep -rn ReconcileWithIssues pkg/daemon/ --include="*.go"

# Check Unmark callers
grep -rn "Unmark(" pkg/daemon/ --include="*.go"
```

**Related Artifacts:**
- **Principle:** Coherence Over Patches (`~/.kb/principles.md:461-466`) — direct trigger for this review
- **Principle:** Evolve by Distinction (`~/.kb/principles.md:808-810`) — dedup conflates "prevent duplicate" with "prevent thrash" with "monitor health"
- **Guide:** Heuristic vs Structural (`~/.kb/guides/heuristic-vs-structural.md`) — framework for recommended redesign
- **Constraint:** `cmd/orch/daemon.go runDaemonLoop must be extracted before adding new daemon subsystems` — from prior knowledge

---

## Investigation History

**2026-03-01 02:30:** Investigation started
- Initial question: Structural review of daemon dedup after 7+ tactical fixes
- Context: 9 dedup-related commits (48b850cca through 142c23ae2) over 2 months

**2026-03-01 03:15:** Code trace complete — 6 layers mapped
- All dedup checks in spawnIssue() traced with fail modes and origins
- Key finding: ReconcileWithIssues is dead code, orphan/cache lifecycle contradictory

**2026-03-01 03:45:** Investigation completed
- Status: Complete
- Key outcome: System hit the coherence-over-patches threshold; recommend structural redesign with CAS-based primary gate and advisory-only heuristic layers
