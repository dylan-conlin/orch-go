## Summary (D.E.K.N.)

**Delta:** Ghost slots keep recurring because capacity is derived from infrastructure scanning (tmux windows + OpenCode sessions) rather than from authoritative business state (beads `in_progress` issues) — each new tmux window shape requires a new filter, and the filter set is never complete.

**Evidence:** Three ghost slot incidents (46cl, hb4g, yhzf) each add a filter to `CombinedActiveCount()` for a new window type; the system is exposed to 4/7 defect classes (0, 1, 3, 4) that all vanish under beads-based counting.

**Knowledge:** The daemon already maintains beads as the authoritative lifecycle state (`spawnIssue` sets `in_progress` before spawning, completion closes issues, orphan detector resets to `open`). Tmux scanning is a redundant, fragile projection of this same state.

**Next:** Implement beads-based capacity counting (replace `CombinedActiveCount` with beads `in_progress` query). Demote tmux scanning to liveness-only role (orphan detection, not capacity).

**Authority:** architectural — Cross-component redesign affecting daemon capacity, active counting, pool reconciliation, and tmux scanning roles. Multiple valid approaches evaluated.

---

# Investigation: Design Review — Daemon Capacity Tracking

**Question:** Why do ghost slots keep recurring in daemon capacity tracking, and what capacity model would structurally prevent them?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect (orch-go-yhzf)
**Phase:** Complete
**Next Step:** None — implementation issues created
**Status:** Complete

**Patches-Decision:** N/A (no prior capacity decision exists — that's part of the problem)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-01-inv-structural-review-daemon-dedup-after.md | extends | Yes — confirmed 6-layer dedup with same "heuristic accumulation" pattern | None — that investigation covers spawn dedup; this covers capacity tracking (related but distinct) |
| 2026-03-04-inv-audit-daemon-code-code-health.md | extends | Yes — confirmed NewWithPool() constructor divergence, OnceWithSlot asymmetry | None |

---

## Findings

### Finding 1: Capacity Is Derived From Infrastructure, Not Business State

The daemon's capacity system uses `CombinedActiveCount()` (active_count.go:255) to determine how many agents are running. This function scans:

1. **OpenCode API** — queries active sessions, extracts beads IDs from titles
2. **Tmux windows** — scans project worker session, checks pane liveness, extracts beads IDs
3. **Merges + deduplicates** by beads ID
4. **Filters out closed** issues via `GetClosedIssuesBatch()`

The result feeds into `Pool.Reconcile(actualCount)` which adjusts the semaphore.

**Evidence:** `CombinedActiveCount()` at active_count.go:255-324 builds a `map[string]bool` from two infrastructure sources. `ReconcileActiveAgents()` at capacity.go:87-106 passes this count to `Pool.Reconcile()`.

**Source:** `pkg/daemon/active_count.go:255-324`, `pkg/daemon/capacity.go:87-106`, `pkg/daemon/pool.go:232-274`

**Significance:** This is the root cause of recurring ghost slots. Each infrastructure source has its own failure modes (dead panes, cross-project windows, child agent windows), and each failure mode requires a new filter. The filter set grows accretively but is never complete because new tmux window shapes keep appearing.

---

### Finding 2: Three Ghost Slot Incidents Follow the Same Pattern

| Incident | Window Shape | Filter Added | Defect Class |
|----------|-------------|-------------|-------------|
| **46cl** | Dead tmux panes (process exited, shell remains) | `IsPaneActive()` — checks pane_current_command + child processes | Class 3 (Stale Artifact Accumulation) |
| **hb4g** | Cross-project tmux windows (skillc agents in `workers-skillc`) | Project-scoped scanning via `projectName` parameter | Class 4 (Cross-Project Boundary Bleed) |
| **yhzf** | Child agent windows (parent completed, children alive) | *No fix yet* | Class 0 (Scope Expansion) + Class 1 (Filter Amnesia) |

Each incident:
1. A new kind of tmux window appears that the scanner doesn't expect
2. `CombinedActiveCount()` inflates
3. `Pool.Reconcile()` creates/retains slots for phantom agents
4. Daemon hits capacity ceiling, stops spawning
5. A new filter is added to `CountActiveTmuxAgents()`

**Evidence:** Git history for `pkg/daemon/active_count.go` shows filter additions: `IsPaneActive()` for 46cl, `projectName` scoping for hb4g. Current issue yhzf describes child windows as the latest shape.

**Source:** `pkg/daemon/active_count.go:190-242` (CountActiveTmuxAgents), `pkg/tmux/tmux.go:1042-1062` (IsPaneActive)

**Significance:** This is the "Coherence Over Patches" anti-pattern from principles.md — 3+ fixes to the same area, each locally correct, but the composite system has emergent gaps. The next fix (child window filter) will be the 4th patch without resolving the structural issue.

---

### Finding 3: Beads Already Tracks the Authoritative Lifecycle State

The daemon already maintains beads as the source of truth for agent lifecycle:

1. **Spawn:** `spawnIssue()` sets beads status to `in_progress` BEFORE spawning (daemon.go:762-800). Fail-fast: if status update fails, spawn is aborted.
2. **Completion:** `CompletionOnce()` detects Phase: Complete comments, closes beads issues, releases pool slots.
3. **Orphan detection:** `OrphanDetector` resets dead agents from `in_progress` → `open` (orphan_detector.go).
4. **Abandonment:** `Abandon()` resets beads status to `open` (lifecycle_impl.go).

This means `count of beads issues with status=in_progress` already represents the correct active agent count, maintained by the existing lifecycle machinery.

**Evidence:**
- L6 in spawn dedup pipeline (daemon.go:762-784): status updated to `in_progress` before spawn, rolled back on failure
- Orphan detector (orphan_detector.go): resets orphaned issues to `open`
- Completion (lifecycle_impl.go:157-170): closes beads issue on completion

**Source:** `pkg/daemon/daemon.go:762-800`, `pkg/daemon/orphan_detector.go`, `pkg/agent/lifecycle_impl.go:157-170`

**Significance:** The infrastructure scanning in `CombinedActiveCount()` is a redundant, fragile projection of state that beads already tracks authoritatively. Every ghost slot incident is caused by the projection diverging from the authoritative source.

---

### Finding 4: Child Agent Windows Are a Lifecycle Problem, Not a Cleanup Problem

The current issue (yhzf) frames child windows as a cleanup problem: "orch complete should discover and kill all tmux windows associated with a beads ID." But child agents spawned via `orch spawn` from within an agent have their own beads issues with their own lifecycle.

The real problem is:
- Parent agent completes → `orch complete` kills parent window → parent beads issue closed
- Child agent's beads issue remains `in_progress` → this is CORRECT if child is still running
- But if child also completed and daemon hasn't run `CompletionOnce()` yet, child window persists
- `CombinedActiveCount()` counts the child window → inflated capacity

With beads-based capacity counting, this resolves naturally:
- If child beads issue is `in_progress`, it counts (correct — child is still working)
- If child beads issue is closed, it doesn't count (correct — child is done)
- The tmux window's existence is irrelevant to capacity

**Evidence:** `cleanInfrastructure()` in lifecycle_impl.go:215-243 only kills the agent's own window by workspace name. No parent-child window relationship is tracked. But `AgentManifest` (spawn/session.go:164-212) stores `BeadsID` per agent, and children have their own beads IDs.

**Source:** `pkg/agent/lifecycle_impl.go:215-243`, `pkg/spawn/session.go:164-212`, `cmd/orch/complete_cleanup.go:10-43`

**Significance:** Beads-based capacity makes the child window problem disappear for capacity tracking. Tmux window cleanup remains a resource hygiene concern (preventing window accumulation) but stops being a capacity correctness problem.

---

### Finding 5: Defect Class Exposure Analysis

Current system (infrastructure-based capacity) is exposed to **4 of 7 defect classes**:

| Defect Class | Exposure | Example |
|-------------|----------|---------|
| Class 0: Scope Expansion | **HIGH** — tmux scanner widens with new window types, consumer assumptions break | Child agent windows are a new "scope" the scanner didn't anticipate |
| Class 1: Filter Amnesia | **HIGH** — each new window type needs a filter in `CountActiveTmuxAgents` | 46cl added liveness filter, hb4g added project scoping, yhzf needs child filter |
| Class 3: Stale Artifact Accumulation | **HIGH** — dead/orphan windows accumulate and inflate count | Dead panes, completed agent shells, abandoned sessions |
| Class 4: Cross-Project Boundary Bleed | **MEDIUM** — fixed for tmux (hb4g), but OpenCode sessions still scan globally | skillc agents were counted toward orch-go capacity |

Proposed system (beads-based capacity) exposure:

| Defect Class | Exposure | Mitigation |
|-------------|----------|------------|
| Class 0 | **NONE** — beads schema is stable, no scanning | N/A |
| Class 1 | **NONE** — single query, no filters | N/A |
| Class 3 | **LOW** — only if beads issues stuck at `in_progress` | Orphan detector already handles this |
| Class 4 | **LOW** — scope by beads ID prefix or project registry | Already handled in cross-project daemon |

**Source:** `.kb/models/defect-class-taxonomy/model.md`, analysis of three ghost slot incidents

**Significance:** Beads-based capacity eliminates the defect classes that produce ghost slots. The only remaining risk (stuck `in_progress`) is already mitigated by the orphan detector.

---

## Synthesis

**Key Insights:**

1. **Infrastructure scanning is the wrong abstraction for capacity.** The daemon conflates "tmux window exists with active process" with "orch-managed agent is active." These are different concepts with different failure modes. Tmux windows are infrastructure artifacts; beads issues are business state. Capacity should be derived from business state.

2. **The accretive fix pattern is a symptom, not the disease.** Each ghost slot fix adds a filter to `CountActiveTmuxAgents()` — liveness checks, project scoping, etc. But the underlying problem is that scanning infrastructure for capacity creates an open-ended filter surface. No matter how many filters are added, new window shapes will appear (child agents today, worktree agents tomorrow, tmuxinator windows next week).

3. **The fix already exists in the codebase.** Beads lifecycle tracking (spawn → in_progress, complete → closed, orphan → open) is already the authoritative state machine. `CombinedActiveCount()` is a redundant, fragile projection of this same information. The solution is to stop projecting and query the source directly.

4. **Tmux scanning retains value for a different purpose.** Tmux scanning should NOT determine capacity (that's beads' job). But it SHOULD be used for:
   - **Orphan detection:** "This beads issue is `in_progress` but no tmux window/session exists" → orphan
   - **Resource cleanup:** "This tmux window has no corresponding `in_progress` beads issue" → cleanup candidate
   - **Liveness signals:** Status display, `orch status` output

**Answer to Investigation Question:**

Ghost slots keep recurring because capacity is derived from infrastructure scanning (tmux windows), which has an open-ended failure surface (new window types → new ghost slot shapes → new filters needed). The capacity model that structurally prevents ghost slots is: **count beads `in_progress` issues, not tmux windows.** This eliminates the scan-filter-patch cycle entirely because beads is the authoritative lifecycle state machine, maintained by spawn, completion, and orphan detection that already exist. Tmux scanning should be demoted to a liveness-checking role (used by orphan detection and resource cleanup) rather than a capacity-enforcement role.

**Principle alignment:**
- **Coherence Over Patches** (principles.md): 3+ fixes to the same area → redesign, not another patch
- **Evolve by Distinction** (principles.md): The system conflates "capacity tracking" with "infrastructure scanning" — these are distinct concerns that should be separated
- **No Local Agent State** (CLAUDE.md): "Query beads and OpenCode directly" — beads IS the authoritative source; tmux scanning is a local projection

---

## Structured Uncertainty

**What's tested:**

- ✅ Three ghost slot incidents all trace to `CombinedActiveCount()` inflating due to unexpected tmux window types (code-traced through 46cl, hb4g, yhzf)
- ✅ Beads lifecycle already tracks in_progress → closed → open transitions correctly (verified: spawnIssue L6 at daemon.go:762, orphan_detector.go, lifecycle_impl.go:157)
- ✅ `Pool.Reconcile()` trusts `CombinedActiveCount()` as sole input (verified: capacity.go:87-106)
- ✅ Defect class exposure reduces from 4/7 to 1/7 under beads-based model (analyzed against taxonomy)

**What's untested:**

- ⚠️ Beads query performance for `in_progress` count — if beads CLI/RPC is slow, poll cycle latency increases. Current tmux scanning is ~50ms; beads query timing unknown but `bd list --status in_progress` should be fast on small datasets.
- ⚠️ Cross-project beads counting — daemon uses `ProjectRegistry` and `BEADS_DIR` for cross-project polling, but counting `in_progress` across all registered projects needs verification.
- ⚠️ Edge case: agent spawned via manual `orch spawn` without `--issue` flag — these don't have beads issues. Need to handle untracked agents (likely by counting tmux windows without beads IDs as a fallback).

**What would change this:**

- If beads `in_progress` count is unreliable (e.g., status updates fail silently, leaving stale `in_progress` entries with no orphan detection), then beads-based capacity would also accumulate ghost slots. Mitigation: orphan detector already handles this case.
- If untracked agents (no beads issue) are common, beads-only counting would undercount capacity. Mitigation: `--no-track` spawns are rare in daemon context; daemon always creates beads issues.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Switch capacity source to beads-based counting | architectural | Cross-component: changes daemon, active_count, capacity, pool reconciliation |
| Demote tmux scanning to liveness-only role | architectural | Changes the contract between tmux package and daemon |
| Add periodic tmux window cleanup (GC) | implementation | Single-scope addition to existing cleanup cycle |
| Fix child window cleanup in orch complete | implementation | Single-scope change to completion cleanup |

### Recommended Approach ⭐

**Beads-Based Capacity with Tmux Liveness** — Replace infrastructure scanning as capacity source with beads `in_progress` query. Retain tmux scanning for orphan detection and resource cleanup only.

**Why this approach:**
- Eliminates the open-ended filter surface that produces ghost slots (Finding 1, 2)
- Leverages lifecycle machinery that already exists and is authoritative (Finding 3)
- Reduces defect class exposure from 4/7 to 1/7 (Finding 5)
- Makes child agent capacity tracking automatic — no special-case handling needed (Finding 4)
- Aligns with "No Local Agent State" constraint and "Coherence Over Patches" principle

**Trade-offs accepted:**
- Beads must be available for capacity checks — but this is already true (spawn fails at L6 if beads is unavailable)
- Untracked agents (`--no-track` spawns) won't count toward capacity — acceptable since daemon never creates untracked spawns
- Slightly higher latency per poll cycle if beads query is slower than tmux scan — mitigated by caching (beads already cached in daemon with 15s TTL)

**Implementation sequence:**

1. **Phase 1: `BeadsActiveCount()` function** — New function in `active_count.go` that queries beads for `in_progress` issues (filtered to orch-spawned work, scoped to project). Wire into `ReconcileActiveAgents()` as `ActiveCounter`.
   - ~60 lines: query beads, filter to in_progress + orch-managed, count by project
   - Test: unit test with mock beads returning various issue states

2. **Phase 2: Demote `CombinedActiveCount()` to liveness role** — Move tmux scanning out of capacity path. Rename/repurpose for orphan detection only.
   - `ReconcileActiveAgents()` uses `BeadsActiveCount()` instead of `CombinedActiveCount()`
   - `CombinedActiveCount()` becomes `DiscoverLiveAgents()` — used by orphan detector to find running infrastructure
   - Inverted responsibility: instead of "scan infra → derive capacity", it's "check capacity (beads) → verify infra matches (tmux)"

3. **Phase 3: Periodic tmux window GC** — Add a cleanup step to daemon's periodic tasks that kills tmux windows whose corresponding beads issues are closed.
   - Scan project worker session for windows with beads IDs
   - For each, check if beads issue is closed → kill window
   - Handles child windows, crashed agents, manual kills — anything that leaves orphan windows
   - Run every 5 minutes (matches orphan detection interval)

4. **Phase 4: Fix `cleanupTmuxWindow()` for multiple matches** — Enhance `complete_cleanup.go` to find and kill ALL windows matching a beads ID across all sessions (not just the first match). This is a resource hygiene improvement independent of capacity tracking.

### Alternative Approaches Considered

**Option B: Keep tmux scanning, add child window filter**
- **Pros:** Minimal change, fixes current incident
- **Cons:** 4th patch to same area (coherence-over-patches). Next window type (worktree agents? tmuxinator windows?) creates 5th ghost slot class. Open-ended filter surface remains.
- **When to use instead:** If beads proves unreliable as capacity source (stale `in_progress` not caught by orphan detector)

**Option C: Hybrid — tmux primary for Claude backend, beads primary for OpenCode backend**
- **Pros:** Uses the most direct signal per backend
- **Cons:** Two counting strategies means two failure surfaces. More complex, harder to reason about. Cross-backend dedup still needed.
- **When to use instead:** Never — this increases complexity without eliminating the fundamental problem

**Option D: Event-driven capacity (increment on spawn, decrement on complete)**
- **Pros:** O(1) per operation, no scanning
- **Cons:** Violates "No Local Agent State" constraint. Doesn't survive daemon restarts without persistence. Accumulated state drifts if events are lost.
- **When to use instead:** Never under current architectural constraints

**Rationale for recommendation:** Option A (beads-based) is the only approach that eliminates the scan-filter-patch cycle. It leverages infrastructure that already works (beads lifecycle), removes a fragile intermediary (tmux scanning for capacity), and aligns with the project's architectural constraint (no local agent state — query authoritative sources directly).

---

### Implementation Details

**What to implement first:**
- Phase 1 (`BeadsActiveCount`) is foundational — all other phases depend on it
- Phase 3 (tmux GC) can be implemented independently as a quick win for resource hygiene
- Phase 4 (multi-window cleanup) is a small, low-risk improvement

**Things to watch out for:**
- ⚠️ **Beads query for in_progress must include cross-project issues** — daemon polls multiple projects. The count should include all orch-managed `in_progress` issues across all registered projects, not just the current working directory.
- ⚠️ **Non-orch in_progress issues** — manually-created beads issues that happen to be `in_progress` should not count toward daemon capacity. Filter by `orch:agent` label OR by presence in spawn cache.
- ⚠️ **Orphan detector interaction** — orphan detector currently uses `HasExistingSessionOrError()` (tmux + OpenCode check) to determine if an agent is alive. This is CORRECT for orphan detection (liveness signal). Don't break this when demoting tmux from capacity role.
- ⚠️ **Reconciliation direction change** — currently, pool reconciles DOWN (free ghost slots). With beads-based counting, it should also reconcile UP (seed slots for agents discovered post-restart). Current `Reconcile()` already handles both directions — verify this path works.

**Areas needing further investigation:**
- How to filter beads `in_progress` to only orch-managed work (label-based vs spawn-cache-based)
- Whether `bd list --status in_progress --json` is fast enough for per-cycle polling (current tmux scan is ~50ms)
- Whether the `orch:agent` label issue (1096: label never removed on close) affects the recommended approach — answer: NO, because we filter by `status=in_progress`, not by label

**Success criteria:**
- ✅ No ghost slot incidents for 2 weeks after deployment
- ✅ Daemon correctly counts capacity with: dead panes, cross-project windows, child agent windows all present
- ✅ `CombinedActiveCount()` removed from capacity path (only used for liveness/orphan detection)
- ✅ Periodic tmux GC cleans up orphan windows within 5 minutes of beads issue closure
- ✅ Existing tests pass; new tests cover beads-based counting with various issue states

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves recurring ghost slot issues (3 incidents)
- This decision establishes the capacity source-of-truth pattern

**Suggested blocks keywords:**
- capacity tracking
- ghost slots
- daemon capacity
- active agent count
- tmux scanning

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` — Main daemon struct, spawnIssue with 6-layer dedup pipeline
- `pkg/daemon/active_count.go` — CombinedActiveCount, CountActiveTmuxAgents, DefaultActiveCount
- `pkg/daemon/capacity.go` — ReconcileActiveAgents, pool reconciliation
- `pkg/daemon/pool.go` — WorkerPool semaphore, Reconcile logic
- `pkg/daemon/session_dedup.go` — HasExistingSessionForBeadsID, dual-layer dedup
- `pkg/daemon/orphan_detector.go` — Orphan detection, fail-closed liveness checking
- `pkg/daemon/interfaces.go` — ActiveCounter interface, defaultActiveCounter
- `pkg/tmux/tmux.go` — CountActiveTmuxAgents, IsPaneActive, FindWindowByBeadsIDAllSessions
- `pkg/agent/lifecycle_impl.go` — cleanInfrastructure, completion/abandonment cleanup
- `pkg/spawn/session.go` — AgentManifest structure
- `cmd/orch/complete_cleanup.go` — cleanupTmuxWindow single-window cleanup
- `.kb/guides/daemon.md` — Daemon reference (capacity section, reconciliation)
- `.kb/investigations/2026-03-01-inv-structural-review-daemon-dedup-after.md` — Prior dedup review
- `.kb/investigations/2026-03-04-inv-audit-daemon-code-code-health.md` — Code health audit
- `.kb/models/defect-class-taxonomy/model.md` — 7 defect classes
- `~/.kb/principles.md` — Coherence Over Patches, Evolve by Distinction

**Related Artifacts:**
- **Investigation:** `2026-03-01-inv-structural-review-daemon-dedup-after.md` — Prior review of spawn dedup (related: same "heuristic accumulation" pattern, complementary scope)
- **Constraint:** `CLAUDE.md` "No Local Agent State" — "Query beads and OpenCode directly; do not build a projection"
- **Principle:** Coherence Over Patches — 3+ fixes to same area → redesign
- **Principle:** Evolve by Distinction — separate "capacity tracking" from "infrastructure scanning"
- **Issue 1096:** `orch:agent` label never removed on close (related but orthogonal — beads-based counting uses `status`, not label)

---

## Investigation History

**2026-03-05:** Investigation started
- Initial question: Why do ghost slots keep recurring in daemon capacity tracking?
- Context: Three incidents (46cl, hb4g, yhzf) each added filters to tmux scanning; pattern suggests structural fragility

**2026-03-05:** Code exploration complete — 4 design forks identified
- Mapped full capacity path: CombinedActiveCount → Pool.Reconcile → capacity decision
- Identified beads lifecycle as existing authoritative source
- Analyzed defect class exposure (4/7 current → 1/7 proposed)

**2026-03-05:** Investigation completed
- Status: Complete
- Key outcome: Replace infrastructure-based capacity (tmux scanning) with beads-based capacity (in_progress query). Demote tmux scanning to liveness-only role. Creates 4-phase implementation plan.
