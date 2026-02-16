# Session Synthesis

**Agent:** og-work-dylan-issue-backlog-16feb-b154
**Issue:** orch-go-979
**Duration:** 2026-02-16
**Outcome:** success

---

## Plain-Language Summary

The issue dependency graph already contains the structural information Dylan needs (which blockers are urgent, what's on the critical path, which P2 is effectively P0 because it blocks P0 work) — but that information gets lost as data flows from beads through the orch-go API to the dashboard. The bd tool has a topological layer algorithm (`computeLayout()` in graph.go) but doesn't include it in its JSON export. The orch-go API passes node/edge data through without enrichment. The frontend has a `layer` field defined in its TypeScript interface but never receives data. The fix is plumbing: port the 50-line layer algorithm to orch-go, add effective priority computation (transitive closure of downstream priorities), enrich the API response, and render it in the dashboard. No new capability needed — just connecting what exists.

## Verification Contract

See: `VERIFICATION_SPEC.yaml` — all acceptance criteria passed.

---

## TLDR

Explored the full pipeline (beads → orch-go serve → dashboard) for how issue dependency structure flows (or fails to flow). Produced an investigation with 4 navigated design forks and a concrete epic shape: Phase 1 ports graph enrichment to orch-go API (no cross-repo changes), Phase 2 renders effective priority + layer ordering in dashboard, Phase 3 (deferred) adds `--sort=effective` to `bd ready`.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-16-inv-design-issue-structure-visibility.md` - Full investigation with 4 navigated design forks and epic shape
- `.orch/workspace/og-work-dylan-issue-backlog-16feb-b154/SYNTHESIS.md` - This file
- `.orch/workspace/og-work-dylan-issue-backlog-16feb-b154/VERIFICATION_SPEC.yaml` - Verification spec

### Files Modified
- None (design exploration only)

---

## Evidence (What Was Observed)

- `beads/cmd/bd/graph.go:322-418` — `computeLayout()` already implements topological layer assignment via longest-path algorithm. Works. Tested. But only used for per-issue graph display, not full export.
- `beads/cmd/bd/graph.go:24-46` — `FullGraphNode` struct has no `Layer` or `EffectivePriority` field. The `--all --json` export is a flat passthrough.
- `cmd/orch/serve_beads.go:596-606` — Go `GraphNode` struct has no layer or effective_priority fields.
- `web/src/lib/stores/work-graph.ts:19` — Frontend `GraphNode.layer?: number` field exists but is never populated by API.
- `web/src/lib/stores/work-graph.ts:258-261` — Children sorted by `localeCompare(id)`, roots by priority then ID. No dependency-aware ordering.
- `beads/internal/storage/sqlite/blocked_cache.go` — Materialized blocked_issues_cache with transitive parent-child propagation (50 levels deep). Proves transitive computation at scale is feasible.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-16-inv-design-issue-structure-visibility.md` - Complete design investigation

### Decisions Made (Recommendations — orchestrator to decide)
1. **Effective priority computation in orch-go serve** (not bd, not frontend) — because it's policy, not data; no cross-repo change needed; composable
2. **Transitive closure algorithm** for effective priority — because P3 blocking P2 blocking P0 should surface as eff:P0
3. **Show both declared + effective priority** with badge when they differ — preserves mental model, adds information
4. **Phase bd integration (cross-repo)** as Phase 3 — prove concept in dashboard first

### Constraints Discovered
- `bd graph --all --json` deliberately omits layer info — only includes id/title/type/status/priority/source. Enrichment must happen downstream.
- `GraphNode.layer` field exists in frontend TypeScript but has no backend counterpart in Go struct — additive API change needed.
- beads blocked_issues_cache handles transitive blocking but is binary (blocked/not). No "degree of urgency" concept exists anywhere in the stack today.

---

## Next (What Should Happen)

**Recommendation:** close — then create epic from the investigation's proposed implementation shape.

### Follow-up Epic (for orchestrator to create)

**Title:** "Make issue dependency structure visible in dashboard (effective priority + topological ordering)"

**Children from investigation:**
1. `pkg/graph/` — Port layer computation + effective priority algorithm (orch-go)
2. Enrich `/api/beads/graph` response with layer + effective_priority (orch-go)
3. Frontend: consume enriched data, default sort by effective priority (web)
4. Frontend: visual badge for effective priority when it differs (web)
5. Integration test with real issue data
6. (Phase 3, deferred) bd graph export enrichment + `bd ready --sort=effective`

**Skill:** feature-impl for children 1-4, investigation for child 5

---

## Unexplored Questions

- **Should effective priority persist in beads?** Currently proposed as computed on-the-fly at API level. If concept proves valuable, could add to `blocked_issues_cache` or create `effective_priority_cache` table in beads.
- **Cross-project effective priority** — beads supports `external:project:capability` deps. Does a P2 in orch-go blocking a P0 in price-watch surface? Dashboard currently shows one project at a time.
- **Daemon spawn ordering** — If daemon uses effective priority for spawn order, highest-priority blockers get worked first (probably good). But could starve independent low-priority work. Worth observing.
- **Knowledge tree topological ordering** — The task mentioned knowledge-tree alphabetical sort. The work-graph tree uses the same `buildTree()` pattern. Enriching the graph data would fix both, but the knowledge tree (`/api/tree`) has a separate code path. May need parallel treatment.

---

## Session Metadata

**Skill:** design-session
**Model:** opus
**Workspace:** `.orch/workspace/og-work-dylan-issue-backlog-16feb-b154/`
**Investigation:** `.kb/investigations/2026-02-16-inv-design-issue-structure-visibility.md`
**Beads:** `bd show orch-go-979`
