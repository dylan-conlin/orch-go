# Design Investigation: Issue Structure Visibility

**Date:** 2026-02-16
**Status:** Complete
**Issue:** orch-go-979

## Question

How should the system make issue dependency structure (priority inheritance, critical path, topological ordering) visible and actionable across the bd → orch-go serve → dashboard pipeline?

## Problem Statement

Three signals point to the same gap:

1. **Orchestrators asking "should we bump P2 to P1?"** when the dependency graph makes the answer obvious — a P2 that blocks a P1 is effectively P1
2. **Knowledge-tree sorting alphabetically** when topological/priority sort would reveal critical path
3. **"Something structural is opaque"** — the dependency information EXISTS in bd but is lost as it flows through the rendering pipeline

## What We Know (Evidence)

### The Pipeline Leaks Structural Information at Each Layer

| Layer | What Exists | What's Missing |
|-------|------------|----------------|
| **bd (beads core)** | `computeLayout()` with topological layer assignment; `blocked_issues_cache`; dependency types with blocking semantics | Layer info not included in `--all --json` export; no effective priority computation; no topological sort option for `bd ready` |
| **orch-go serve** | `/api/beads/graph` returns nodes + edges; 15s cache | No layer enrichment; no effective priority; `GraphNode` struct has no layer field; passthrough of bd output |
| **web (dashboard)** | `GraphNode.layer?: number` field defined; `buildTree()` + `groupTreeNodes()`; blocked_by/blocks tracked | `layer` never populated; sort by `localeCompare` (alphabetical) or declared priority; no critical path view |

### bd Already Has the Algorithm

`beads/cmd/bd/graph.go:322-418` (`computeLayout()`) implements longest-path topological layer assignment:
- Layer 0 = no dependencies (ready to work)
- Layer N = max(dependency layers) + 1
- Handles cycles (assigns layer 0)

But `loadFullGraph()` (the `--all --json` export) returns `FullGraphNode` **without** layer info. The per-issue `bd graph <id> --json` does include layout with layers, but the full export used by the dashboard does not.

### Blocking Semantics Are Rich

From `beads/internal/types/types.go`:
- **`blocks`**: Standard blocking (B waits for A)
- **`parent-child`**: Transitive blockage propagation (50 levels deep)
- **`conditional-blocks`**: Failure-driven (B runs only if A fails)
- **`waits-for`**: Fanout gates (B waits for A's children)

The blocked_issues_cache already computes transitive blocking. But blocking is binary (blocked/not). There's no **degree** of urgency — how important is it that this blocker gets resolved?

### Current Dashboard Rendering

- `buildTree()` sorts children by `localeCompare(id)` (creation order)
- Roots sorted by `priority` then `localeCompare(id)`
- `groupTreeNodes()` groups by priority/area/effort labels
- No concept of dependency-aware ordering
- blocked_by/blocks arrays populated but used only for display badges, not sorting

## Design Forks

### Fork 1: Where Does Effective Priority Computation Live?

**Options:**
- **A: bd** computes it (data layer) — `effective_priority` field on issue
- **B: orch-go serve** computes it (API enrichment layer)
- **C: web** computes it (display layer)
- **D: bd provides primitives, orch-go composes** (split responsibility)

**Substrate says:**
- Principle "Surfacing Over Browsing": Computation should happen server-side, surface to consumer
- Principle "Compose Over Monolith": Small focused tools that combine
- Principle "Share Patterns Not Tools": bd defines dependency semantics, orch-go can compute derived properties independently
- Current architecture: bd shells out via CLI, orch-go enriches for API consumers

**Recommendation: D** — bd already provides graph primitives (`bd list` + dependencies in issue data). orch-go computes effective priority and layer assignment at the API level before returning to frontend.

**Reasoning:**
1. No cross-repo change needed for initial implementation
2. Layer computation is ~50 lines of Go (algorithm exists in graph.go to port)
3. Effective priority is a policy decision (how to weight urgency) — better in the orchestration layer than beads core
4. Dashboard gets enriched data without frontend graph computation
5. Future: bd can add `--sort=effective` opt-in as Phase 3 if concept proves useful

### Fork 2: What Algorithm for Effective Priority?

**Options:**
- **A: Direct dependents only** — `min(own_priority, min(priority of issues I directly block))`
- **B: Transitive closure** — Walk full downstream chain, take highest priority (lowest number)
- **C: Depth-weighted** — Factor in dependency depth (closer dependents weight more)

**Recommendation: B** — transitive closure.

**Reasoning:**
- A P3 that blocks a P2 that blocks a P0 should surface as effectively P0
- Dependency chains can be deep in this system (epics → tasks → subtasks)
- Simple BFS from each node, collect min priority of all downstream nodes
- Option A misses transitive urgency
- Option C adds complexity without clear benefit (urgency doesn't decay with distance)

**Algorithm:**
```
for each node N:
  effective_priority(N) = min(
    N.priority,
    min(effective_priority(M) for M in all_nodes_transitively_blocked_by(N))
  )
```

**Performance:** With current issue counts (10K max in beads), BFS per node is negligible. Can memoize with dynamic programming (compute in reverse topological order).

### Fork 3: How Should This Render in the Dashboard?

**Options:**
- **A: Sort by effective priority** (replace current sort)
- **B: Show both** — declared + effective priority badge when they differ
- **C: Critical path view** — separate view showing only the critical chain
- **D: Layer-based columns** — dependency depth as visual columns

**Recommendation: B + A** — default sort by effective priority, show badge when effective differs from declared.

**Reasoning:**
- Preserves existing mental model (you still see P2)
- Adds new information (P2 → eff:P0) without breaking expectations
- The sort surfaces urgent blockers naturally
- Critical path view (C) is valuable but can be Phase 2 UX work
- Layer columns (D) duplicates bd graph ASCII output, less useful in tree view

**Visual concept:**
```
P0 🔴 Fix API authentication          ← declared P0, no badge
P2 🟡 Update DB schema  [eff:P0]      ← declared P2, but blocks the P0 above
P1 🟠 Add rate limiting               ← declared P1, no badge (eff:P1)
P3 🔵 Refactor utils  [eff:P1]        ← declared P3, but blocks the P1
```

### Fork 4: Should `bd ready` Sort by Effective Priority?

**Options:**
- **A: Yes** — change bd ready default sort
- **B: No** — keep bd ready as-is, orch-go enrichment only
- **C: Add `--sort=effective` option** — opt-in

**Recommendation: C** — opt-in flag in bd, but defer to Phase 3. Focus on orch-go API enrichment first.

**Reasoning:**
- Cross-repo change to beads (higher coordination cost)
- bd ready is used by daemon for auto-spawning — changing sort could affect spawn order
- The dashboard path (orch-go serve) is where Dylan sees the data
- If effective priority proves valuable in dashboard, then port to bd

## Proposed Implementation (Epic Shape)

### Phase 1: API Enrichment (orch-go only, no cross-repo changes)

**Estimated: 2-3 child issues**

1. **Create `pkg/graph/` package** — Port layer computation from bd's `computeLayout()` + add effective priority algorithm
   - `ComputeLayers(nodes, edges) → map[id]int` — topological layer assignment
   - `ComputeEffectivePriority(nodes, edges) → map[id]int` — transitive closure min-priority
   - Unit tests with cycle handling, disconnected components

2. **Enrich `/api/beads/graph` response** — Add `layer` and `effective_priority` to `GraphNode` struct in serve_beads.go
   - Call pkg/graph functions after building nodes+edges
   - Add new JSON fields: `"layer": 0, "effective_priority": 1`
   - Backward compatible (additive fields)

### Phase 2: Frontend Rendering (web/ changes)

**Estimated: 2-3 child issues**

3. **Consume enriched data** — Update work-graph.ts to use `layer` and `effective_priority`
   - Add `effective_priority` to `GraphNode` and `TreeNode` interfaces
   - Default sort by effective_priority (instead of declared priority)
   - Add "effective" grouping mode to `groupTreeNodes()`

4. **Visual badges** — Show effective priority badge when it differs from declared
   - Render `[eff:P0]` or equivalent when effective_priority < priority
   - Color coding: use effective_priority color for node border/indicator
   - Tooltip: "This P2 blocks P0 work through: [chain description]"

5. **Optional: Layer-aware rendering** — Use `layer` for secondary ordering within groups
   - Within same effective priority, layer 0 (actionable) before layer 1 (needs prereqs)
   - Visual indicator: lighter opacity for higher layers (less immediately actionable)

### Phase 3: bd Integration (cross-repo, future)

**Estimated: 2 child issues (deferred)**

6. **Add effective_priority to bd graph export** — Extend `FullGraphNode` with `EffectivePriority int`
7. **Add `--sort=effective` to bd ready** — Opt-in topological priority sort

### Integration Issue

8. **End-to-end validation** — Verify enriched graph renders correctly with real issue data
   - Test with current orch-go issue set (100+ issues)
   - Verify performance (enrichment should add <10ms to graph response)
   - Verify critical path visibility: can Dylan see which P2 is blocking P0 work?

## What Remains Unclear

1. **Should effective priority persist?** Currently proposed as computed on-the-fly. If it proves useful, should bd cache it (like blocked_issues_cache)? Pro: `bd ready --sort=effective` would be fast. Con: another cache to invalidate.

2. **Cross-project dependencies** — beads supports `external:project:capability` deps. Should effective priority cross project boundaries? Likely yes, but needs investigation into how the dashboard handles multi-project views.

3. **Daemon impact** — If daemon uses effective priority for spawn ordering, the highest-priority blockers get worked first. This is probably good but could starve low-priority independent work. Worth observing before changing daemon behavior.

## Recommendation

**Output type:** Epic with children (Phase 1 + Phase 2 + integration).

**Clarity level:** High — all forks navigated, algorithm defined, implementation path clear.

**Next step:** Create the epic with Phase 1-2 children. Phase 3 deferred to separate epic after validation.

**The key insight for Dylan:** The structural information already exists in beads. This isn't building new capability — it's plumbing existing graph data through the pipeline so it reaches the dashboard. The `computeLayout()` algorithm is already written and tested in bd. We're porting ~50 lines of graph code and adding ~30 lines of priority propagation.
