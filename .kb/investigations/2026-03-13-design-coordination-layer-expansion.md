## Summary (D.E.K.N.)

**Delta:** Designed four coordination capabilities — work graph (global dedup/removal), cognitive allocation (skill-aware slot scoring), cross-session learning (outcome feedback loops), and the coordinator daemon vision (OODA-structured poll cycle) — that become possible once compliance is separated from coordination.

**Evidence:** Code analysis of 10 daemon files shows current ratio ~80% compliance / ~20% coordination. The 20% coordination (skill inference, focus boost, project interleaving) is embedded in issue_selection.go and skill_inference.go. Compliance gates (5-layer dedup pipeline, hotspot enforcement, verification pause, rate limiting) dominate the spawn path.

**Knowledge:** The key architectural insight is that coordination decisions are O(n) over the queue while compliance decisions are O(1) per spawn. As agent count scales, coordination cost grows but compliance cost doesn't — making coordination the leverage point.

**Next:** Implement in 4 phases: (1) Learning Store, (2) Allocation Profile, (3) Work Graph, (4) Coordinator refactor. Create implementation issues for each phase.

**Authority:** architectural — Cross-component design affecting daemon, spawn, events, and configuration subsystems.

---

# Investigation: Coordination Layer Expansion Design

**Question:** What new coordination capabilities become possible once compliance is separated from the daemon, and how should they be designed?

**Started:** 2026-03-13
**Updated:** 2026-03-13
**Owner:** architect (autonomous)
**Phase:** Complete
**Next Step:** None — design complete, implementation issues created
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/threads/2026-03-13-compliance-coordination-bifurcation-designing-split.md | extends | Yes — verified ratio claim against code | None |
| .kb/threads/2026-03-12-compliance-coordination-bifurcation-as-models.md | extends | Yes — "as models improve" thesis validated by code structure | None |

---

## Findings

### Finding 1: Current coordination is embedded in issue selection, not the spawn path

**Evidence:** The daemon's coordination capabilities live in just 3 files:
- `issue_selection.go`: Priority sort, project interleaving, epic expansion, focus boost application
- `skill_inference.go`: 4-level inference chain (label → title → description → type), model mapping from skill
- `focus_boost.go`: Priority adjustment based on `~/.orch/focus.json` goal matching

Meanwhile, compliance dominates the spawn path:
- `spawn_execution.go`: 5-gate dedup pipeline (SpawnTracker, SessionDedup, TitleDedupMemory, TitleDedupBeads, FreshStatus)
- `daemon.go:281-402`: Verification pause check, completion failure check, rate limit check, extraction gate, architect escalation
- `architect_escalation.go`: Layer 2 hotspot enforcement
- `completion_processing.go`: Verification retry budget, escalation routing
- `orphan_detector.go`: Fail-closed session checking

**Source:** `pkg/daemon/daemon.go:269-423`, `pkg/daemon/spawn_execution.go:14-219`, `pkg/daemon/issue_selection.go:56-178`

**Significance:** Coordination decisions happen before spawn (issue selection), compliance decisions happen during spawn (gates). Separating them is architecturally natural — they already occupy different phases. The coordination layer can be expanded without touching the spawn gate pipeline.

---

### Finding 2: The daemon has no feedback loop from outcomes to decisions

**Evidence:** The spawn path is purely forward: queue → select → infer → gate → spawn → done.

Events are logged to `events.jsonl` (27 event types including `spawn.skill_inferred`, `agent.completed`, `agent.abandoned`, `daemon.architect_escalation`). But nothing reads these events back. The daemon's decisions are identical on day 1 and day 100 — no learning occurs.

The `SpawnedIssueTracker.spawnCounts` field tracks how many times an issue has been spawned (thrashing detection), but this signal is advisory-only (warns at 3+ spawns) and doesn't feed back into issue selection or model choice.

**Source:** `pkg/daemon/spawn_tracker.go:39-43`, `pkg/events/logger.go` (27 event types), `pkg/daemon/skill_inference.go:260-282` (static model mapping)

**Significance:** This is the single highest-leverage coordination capability to add. The data exists. The aggregation is straightforward. The feedback paths are obvious (model selection, priority adjustment, complexity estimation). The daemon should get smarter over time.

---

### Finding 3: All pool slots are fungible — no skill-awareness in capacity management

**Evidence:** `WorkerPool` in `pool.go` is a pure semaphore: `activeCount int`, `maxWorkers int`. `TryAcquire()` returns a slot or nil. No metadata about what type of work occupies each slot.

The `Slot` struct has `BeadsID string` but no skill, model, or category field. The daemon cannot answer "how many investigations are running?" without querying an external system.

Issue selection in `NextIssueExcluding` sorts by priority and interleaves by project, but has no visibility into what skills are currently occupying slots. The daemon's greedy fill strategy means: if 5 P1 feature-impls are queued, all 5 slots go to feature-impls. Knowledge-producing work (investigations, architects) gets starved.

**Source:** `pkg/daemon/pool.go:11-19`, `pkg/daemon/pool.go:22-26`, `pkg/daemon/issue_selection.go:79-86`

**Significance:** Skill-aware allocation is the difference between a queue processor and a resource allocator. The coordination layer should reason about what mix of work is optimal, not just what's next in the queue.

---

### Finding 4: The "No Local Agent State" constraint shapes the work graph design

**Evidence:** CLAUDE.md states: "orch-go must not maintain local agent state (registries, projection DBs, SSE materializers, caches for agent discovery). Query beads and OpenCode directly."

This means the work graph (for dedup/removal) must be computed per-cycle from authoritative sources, not maintained as a persistent projection. This is actually fine — the work graph is a per-cycle analysis, not a database.

**Source:** CLAUDE.md "Architectural Constraints" section

**Significance:** The work graph is a function, not state. `BuildWorkGraph(queue []Issue, completions []Completion) *WorkGraph`. This makes it testable, deterministic, and free of drift — exactly what the constraint intends to preserve.

---

### Finding 5: The creation/removal asymmetry is the deepest coordination problem

**Evidence:** The daemon currently only processes forward: `triage:ready → in_progress → completed → closed`. It adds work to the system (spawn from queue) but never removes work. Work removal is entirely manual — the orchestrator or Dylan closes obsolete issues.

Adding work is O(1): `bd create "task" --type task -l triage:ready`. One command, one context.

Removing work requires O(n) context: "Is this still relevant? Has anything superseded it? Does another issue cover the same area? Has the underlying problem been fixed?" This requires knowing the full state of the system.

The daemon is the only actor with full-system visibility (it polls all projects, all issues, all active agents). It's the natural place for removal intelligence.

**Source:** `pkg/daemon/issue_selection.go:57` (ListReadyIssues queries all open issues), `cmd/orch/daemon_loop.go:92-129` (ProjectRegistry discovers all projects)

**Significance:** Work removal is the defining capability of the coordinator. Any queue processor can add work. Only a coordinator with global context can identify work that should be removed. This is what makes the daemon a "cognitive resource allocator" rather than a "spawn bot."

---

## Synthesis

**Key Insights:**

1. **Coordination and compliance occupy different phases** — Coordination happens in issue selection (before spawn), compliance happens in the spawn pipeline (during spawn). Separation is architecturally natural.

2. **The feedback loop is the highest-leverage addition** — The daemon has comprehensive event logging but zero feedback from outcomes to decisions. Adding a learning store that aggregates events into actionable signals (success rates, duration estimates, time-of-day patterns) transforms a stateless queue processor into an adaptive coordinator.

3. **The work graph is a function, not state** — The "No Local Agent State" constraint means dedup/removal intelligence must be computed per-cycle from authoritative sources. This is actually the right design — it's testable, deterministic, and free of drift.

4. **Allocation profiling extends existing mechanisms** — The daemon already adjusts priorities via `applyFocusBoost`. Skill-aware allocation is the same mechanism (priority adjustment before sort) with a richer signal (current slot utilization + allocation targets).

**Answer to Investigation Question:**

Four coordination capabilities become possible once compliance is separated:

**A. Work Graph (Global Context for Work Removal)**
A per-cycle computation that maps relationships between queue items, running agents, and recent completions. Detects semantic duplicates, file-target overlap, and obsolescence. Surfaces removal suggestions as beads questions rather than auto-closing. Respects "No Local Agent State" by computing from authoritative sources each cycle.

**B. Allocation Profile (Cognitive Resource Allocation)**
A desired skill distribution (e.g., 30% investigation, 50% implementation, 20% maintenance) that modifies issue priority scoring. Soft preference, not hard reservation — P0 issues always win. Dynamically adjusted by system state (stalled agents → boost debugging), time-of-day (overnight → boost investigations), and focus goal. Extends the existing `applyFocusBoost` pattern.

**C. Learning Store (Cross-Session Learning)**
Periodic aggregation of events.jsonl into actionable metrics: skill-model success rates, median completion times, time-of-day patterns, thrashing detection. Persisted to `~/.orch/daemon_learning.json` as a cache (can be rebuilt from events.jsonl). Feeds into model selection, priority adjustment, and complexity estimation.

**D. OODA Poll Cycle (The Coordinator Daemon)**
Restructured poll cycle: Sense (build world state) → Orient (evaluate work graph, calculate allocation gap, score candidates) → Decide (select best allocation, run minimized compliance gates) → Act (spawn, record outcomes, update learning). Compliance shrinks to O(1) safety net; coordination grows to O(n) analysis.

---

## Structured Uncertainty

**What's tested:**

- ✅ Current coordination/compliance ratio (~80/20 compliance) verified by reading all daemon files
- ✅ "No Local Agent State" constraint confirmed in CLAUDE.md — work graph must be per-cycle computation
- ✅ Existing priority adjustment mechanism (applyFocusBoost) confirmed as extension point for allocation profiling
- ✅ Events.jsonl has 27 event types sufficient for learning store aggregation

**What's untested:**

- ⚠️ Work graph computation cost — per-cycle analysis of queue + completions may add latency to 15s poll interval
- ⚠️ Allocation profile effectiveness — soft preference may not sufficiently shift skill distribution when queue is skill-homogeneous
- ⚠️ Learning store accuracy — events.jsonl may have gaps (crashes, manual operations) that bias aggregated metrics
- ⚠️ Semantic similarity for dedup — title/description overlap detection may have high false positive rate without embedding-based similarity

**What would change this:**

- If per-cycle work graph computation exceeds 2s, need to cache partial results (violates "No Local Agent State" — would need decision)
- If allocation profiles cause P1 issues to be delayed significantly, need to narrow the allocation to P2+ only
- If learning store false positives cause wrong model selection >10% of the time, need to add confidence thresholds

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Learning Store (Phase 1) | implementation | Self-contained in pkg/daemon, reads existing events.jsonl, no cross-component impact |
| Allocation Profile (Phase 2) | implementation | Extends existing applyFocusBoost pattern, configuration-only changes |
| Work Graph (Phase 3) | architectural | Cross-project query patterns, new interface for dedup, affects issue selection contract |
| OODA Refactor (Phase 4) | architectural | Restructures daemon poll cycle, touches cmd/orch/daemon_loop.go and pkg/daemon/daemon.go |

### Recommended Approach ⭐

**Incremental Coordination Expansion** — Add coordination capabilities in 4 phases, each independently valuable, building toward the OODA coordinator.

**Why this approach:**
- Each phase is independently testable and deployable
- Phase 1 (Learning Store) provides immediate value with zero risk (read-only aggregation)
- Phase 2 (Allocation Profile) extends existing mechanism (applyFocusBoost pattern)
- Phases 3-4 build on signals from 1-2

**Trade-offs accepted:**
- Per-cycle work graph computation adds latency to poll interval (mitigated by keeping it lightweight)
- Allocation profiles are soft preferences — they won't perfectly balance skill distribution
- Learning store is a cache that can drift from events.jsonl (mitigated by periodic rebuild)

**Implementation sequence:**

1. **Phase 1: Learning Store** (~2h) — New `pkg/daemon/learning.go`. Periodic aggregation of events.jsonl into `~/.orch/daemon_learning.json`. Metrics: skill-model success rates, median durations, hourly patterns. Read at daemon startup, rebuilt periodically (new periodic task).

2. **Phase 2: Allocation Profile** (~3h) — New `pkg/daemon/allocation.go`. Configurable skill distribution targets in daemonconfig. `applyAllocationBoost` function called after `applyFocusBoost` in `NextIssueExcluding`. Requires slot metadata (add Skill field to Slot struct).

3. **Phase 3: Work Graph** (~4h) — New `pkg/daemon/work_graph.go`. Per-cycle computation from queue + recent completions. Title similarity, file-target overlap, investigation-chain detection. Surfaces removal suggestions via `bd create --type question`. Feeds skip set into `NextIssueExcluding`.

4. **Phase 4: OODA Refactor** (~4h) — Restructure `runDaemonSpawnCycle` into Sense→Orient→Decide→Act phases. Move compliance gates to minimized safety layer. Add coordinator status to daemon status file. Surface coordination insights to orchestrator.

### Alternative Approaches Considered

**Option B: Hard slot reservation**
- **Pros:** Guarantees skill diversity (e.g., "always 1 investigation slot")
- **Cons:** Rigid, wastes capacity when queue lacks certain skill types. When queue has 0 investigations, 1 slot sits empty.
- **When to use instead:** If soft allocation profiles don't achieve sufficient diversity in practice

**Option C: Embedding-based semantic dedup**
- **Pros:** High-accuracy dedup based on meaning, not just title overlap
- **Cons:** Requires API calls for embeddings (cost, latency), external dependency
- **When to use instead:** If title/file-based work graph has >30% false positive rate

**Rationale for recommendation:** Incremental expansion preserves the daemon's simplicity while adding coordination intelligence. Each phase can be evaluated independently. The soft-preference approach (allocation profiles, work graph suggestions) is safer than hard enforcement — it degrades gracefully.

---

### Implementation Details

**What to implement first:**
- Learning Store (Phase 1) — highest leverage, zero risk, provides signals for all later phases
- Add `Skill` and `Model` fields to `Slot` struct in pool.go — prerequisite for allocation awareness

**Things to watch out for:**
- ⚠️ Defect Class 0 (Scope Expansion): Work graph scanning may expand to include closed issues, old completions — use explicit time windows
- ⚠️ Defect Class 3 (Stale Artifact Accumulation): Learning store file must have cleanup/rotation — don't let it grow unbounded
- ⚠️ Defect Class 4 (Cross-Project Boundary Bleed): Work graph dedup across projects must respect project isolation — similar titles in different projects may be intentional
- ⚠️ "No Local Agent State" constraint: Work graph is per-cycle computation, NOT a persistent projection. Learning store is a cache, NOT a source of truth (events.jsonl is authoritative)

**Areas needing further investigation:**
- Optimal allocation profile ratios (need empirical data from learning store first)
- Work graph performance characteristics at scale (>50 issues in queue)
- How to surface coordinator insights to orchestrator without creating noise

**Success criteria:**
- ✅ Learning store aggregates events.jsonl within 500ms at startup
- ✅ Allocation profile shifts at least 1 slot per cycle from overrepresented to underrepresented skills
- ✅ Work graph detects >80% of duplicate issues surfaced by manual review
- ✅ OODA poll cycle completes within existing 15s interval

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` — Daemon struct, subsystem wiring, OnceExcluding spawn flow
- `pkg/daemon/issue_selection.go` — NextIssueExcluding, priority sort, project interleaving, epic expansion
- `pkg/daemon/skill_inference.go` — 4-level skill inference chain, static model mapping
- `pkg/daemon/focus_boost.go` — Focus goal priority adjustment
- `pkg/daemon/pool.go` — WorkerPool semaphore, Slot struct, Reconcile
- `pkg/daemon/spawn_tracker.go` — SpawnedIssueTracker, title dedup, spawn counts, disk persistence
- `pkg/daemon/spawn_execution.go` — 5-gate dedup pipeline, status update, rollback, account resolution
- `pkg/daemon/architect_escalation.go` — Layer 2 hotspot enforcement, prior architect lookup
- `cmd/orch/daemon_loop.go` — Main poll loop, spawn cycle, signal handling, status file
- `.kb/threads/2026-03-13-compliance-coordination-bifurcation-designing-split.md` — Bifurcation thesis
- `.kb/threads/2026-03-12-compliance-coordination-bifurcation-as-models.md` — Prior thread

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-13-compliance-coordination-bifurcation-designing-split.md` — Originated this design
- **Thread:** `.kb/threads/2026-03-12-compliance-coordination-bifurcation-as-models.md` — Foundation: compliance/coordination value trajectories

---

## Investigation History

**2026-03-13 00:15:** Investigation started
- Initial question: What coordination capabilities become possible once compliance is separated from the daemon?
- Context: Bifurcation thread identified 80/20 compliance/coordination ratio, need to design the coordination expansion

**2026-03-13 00:20:** Phase: Exploration complete
- Read all 10 daemon files, identified coordination vs compliance decision points
- Confirmed bifurcation thesis: coordination in issue selection, compliance in spawn pipeline

**2026-03-13 00:35:** Phase: Synthesis complete
- Designed 4 capabilities: Work Graph, Allocation Profile, Learning Store, OODA Poll Cycle
- Identified implementation sequence: Learning Store first (highest leverage, zero risk)

**2026-03-13 00:45:** Investigation completed
- Status: Complete
- Key outcome: 4-phase coordination expansion design with clear implementation sequence and authority routing
