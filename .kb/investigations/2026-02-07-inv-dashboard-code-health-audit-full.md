<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The 30s work-graph symptom is primarily backend data-collection debt (CLI fanout and collector recursion), not raw Svelte render cost.

**Evidence:** Cold timings were `graph_open=1.199s`, `agents=1.401s`, `attention=4.994s` after cache invalidation, and the graph handler shells out to `bd` repeatedly (`list` + per-issue `dep list`).

**Knowledge:** The current architecture composes many small polls and collectors that are individually reasonable but cumulatively expensive, while key UI entry files have become large coordination monoliths.

**Next:** Prioritize backend query-plan reduction for `/api/beads/graph` and `/api/attention`, then split dashboard route/store orchestration into focused modules.

**Authority:** architectural - fixes cross API boundaries (attention, graph, agents, store orchestration) and require coordinated design choices.

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

# Investigation: Dashboard Code Health Audit Full

**Question:** Where is `/work-graph` latency actually spent, how healthy is dashboard architecture (API + Svelte), what neglected risks are accumulating, and what should be fixed first?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** OpenCode worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-01-14-inv-orch-doctor-verify-dashboard-fetch.md` | deepens | yes | none |
| `.kb/investigations/2026-01-28-inv-design-unified-strategic-center-dashboard.md` | deepens | yes | none |
| `.kb/investigations/2026-01-10-inv-p0-implement-orch-doctor-health.md` | confirms | yes | none |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: `/work-graph` API path is CLI-backed and scales with dependency fanout

**Evidence:** `handleBeadsGraph` delegates to `buildFullGraph` for `scope=open`, which runs `bd list --status open`, `bd list --status in_progress`, then `bd dep list <id>` for every issue with dependencies. Parent filtering is applied only after full graph construction, so `parent=` does not reduce backend work.

**Source:** `cmd/orch/serve_beads_graph.go:168`, `cmd/orch/serve_beads_graph.go:345`, `cmd/orch/serve_beads_graph.go:367`, `cmd/orch/serve_beads_graph.go:389`, `cmd/orch/serve_beads_graph.go:409`, `cmd/orch/serve_beads_graph.go:493`; command: benchmark script for `bd list/dep`.

**Significance:** This creates a query plan that is effectively `2 + N` subprocess calls per cold graph build (N = issues with dependencies). It is manageable at current scale but vulnerable to 30s+ tails when CLI stalls or N grows.

---

### Finding 2: Initial load latency is dominated by attention aggregation, not graph rendering

**Evidence:** Cold endpoint timings (post cache invalidation) were `focus 0.008s`, `graph_open 1.199s`, `agents 1.401s`, `attention 4.994s`, `daemon 0.088s`. Warm timings improved but still show attention as the heaviest call (`1.351s`). `/work-graph` mounts with `Promise.all([workGraph.fetch, agents.fetch, attention.fetch])`, so one slow endpoint delays visible completion.

**Source:** `web/src/routes/work-graph/+page.svelte:106`, `web/src/routes/work-graph/+page.svelte:109`; command: API cold/warm benchmark against `https://localhost:3348`.

**Significance:** Users experience aggregate page latency; optimizing only `/api/beads/graph` will not eliminate slow first paint if `/api/attention` remains serial and expensive.

---

### Finding 3: Attention pipeline introduces recursive and serial cost amplification

**Evidence:** `handleAttention` instantiates 11 collectors and executes them serially. Two collectors (`AgentCollector`, `StuckCollector`) each call `/api/agents?since=all`, creating nested expensive requests. This duplicates agent fetch work inside a single attention request.

**Source:** `cmd/orch/serve_attention.go:176`, `cmd/orch/serve_attention.go:206`, `cmd/orch/serve_attention.go:227`, `cmd/orch/serve_attention.go:258`; `pkg/attention/agent_collector.go:53`; `pkg/attention/stuck_collector.go:54`.

**Significance:** Serial collector execution plus duplicate `/api/agents` calls creates avoidable latency and can produce cascading slowness under load.

---

### Finding 4: Frontend route/store orchestration has monolith pressure and overlapping refresh loops

**Evidence:** File sizes are high: `web/src/routes/+page.svelte` 1018 lines, `web/src/routes/work-graph/+page.svelte` 638 lines, `web/src/lib/stores/agents.ts` 900 lines, `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` 893 lines. `+page.svelte` coordinates many stores and polling loops; `work-graph/+page.svelte` polls 4 endpoints every 5s while also using SSE.

**Source:** `web/src/routes/+page.svelte:1`, `web/src/routes/+page.svelte:162`, `web/src/routes/+page.svelte:198`, `web/src/routes/work-graph/+page.svelte:141`, `web/src/routes/work-graph/+page.svelte:143`; command: `wc -l` on key files.

**Significance:** Current decomposition exists but orchestration logic is still centralized in route files/stores, raising change risk and making perf debugging harder.

---

### Finding 5: Build and quality health is mixed: frontend builds with a11y debt; Go build currently broken

**Evidence:** `go build ./...` fails with duplicate function declarations across `cmd/orch/complete_*` files. Frontend `bun run build` succeeds but emits multiple accessibility warnings, including click handlers on non-interactive elements and dialog/label issues.

**Source:** `cmd/orch/complete_export.go:38`, `cmd/orch/complete_pipeline.go:220`, `cmd/orch/complete_gates.go:20`; `web/src/lib/components/work-graph-tree/work-graph-tree.svelte:538`; `web/src/lib/components/close-issue-modal/close-issue-modal.svelte:40`; command outputs from `go build ./...` and `bun run build`.

**Significance:** Production confidence is constrained by backend compile breakage and unaddressed a11y warnings in high-traffic UI surfaces.

---

## Synthesis

**Key Insights:**

1. **Backend composition is the primary latency driver** - `/work-graph` waits on a combined load path (`graph + agents + attention`), and the slowest piece is currently attention collection plus nested API calls.

2. **Current graph data flow is functionally correct but cost-inefficient** - building full open graph before parent filtering and executing per-issue dependency CLI calls creates structural overhead that scales poorly.

3. **Architecture debt is now operational debt** - oversized route/store files and overlapping poll/SSE flows increase incident surface area, debugging time, and regression risk.

4. **Neglected quality dimensions are visible and actionable** - accessibility warnings, missing small-unit tests around store algorithms, and a broken Go build are concrete backlog items, not abstract concerns.

5. **The system is close to recoverable with targeted work** - endpoint latency is seconds (not irreducible 30s) in local measurement, so focused query-plan and orchestration cleanup should produce large wins quickly.

**Answer to Investigation Question:**

The current code health risk is concentrated in backend data assembly and orchestration complexity, not a single rendering bottleneck. `/work-graph` itself is built from CLI-backed graph queries (`bd list` + per-issue `bd dep list`), while first-load UX is gated by the combined `workGraph + agents + attention` calls where attention is currently the slowest contributor. Architecture is partially decomposed into components/stores, but core route/store files remain monolithic enough to impede maintainability. Build health is currently red for Go due to duplicate declarations, while frontend build is green with notable a11y warnings. Limitations: measurements were taken on this local dataset and may understate worst-case tails during known `bd` hangs.

---

## Structured Uncertainty

**What's tested:**

- ✅ `/api/beads/graph`, `/api/agents`, `/api/attention` latency profile was measured cold and warm against running server.
- ✅ CLI fanout behavior was verified in source and correlated with measured `bd list`/`bd dep list` timings.
- ✅ Build health was tested with `go build ./...` and frontend health with `bun run build`.

**What's untested:**

- ⚠️ Browser main-thread render cost (FPS/flamegraph) under very large graphs was not profiled with DevTools.
- ⚠️ Peak-load behavior with many simultaneous agents/workspaces was not replayed in a synthetic stress run.
- ⚠️ Whether 30s symptom is primarily due to `bd` hang in other environments was inferred from architecture, not reproduced here.

**What would change this:**

- If end-to-end trace shows frontend render > backend response time on representative slow sessions, performance priority should shift to UI virtualization/render strategy.
- If attention collector parallelization does not reduce p95 materially, deeper collector-level algorithmic work (or cache strategy) is needed.
- If `bd` RPC path is stable and CLI fallback rarely used in production, graph optimization priority may drop behind attention and agents APIs.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Rework `/api/beads/graph` query plan (single dependency fetch path + early parent pruning) | architectural | Cross-cutting API and data-model behavior change with backend/frontend implications |
| Refactor `/api/attention` to avoid duplicate `/api/agents` calls and support parallel collector execution | architectural | Requires collector contract changes and response-shaping decisions |
| Decompose route/store monoliths into orchestration + domain modules | architectural | Spans multiple files/components and affects team conventions |
| Fix duplicate `complete_*` declarations to restore Go build | implementation | Tactical compile fix within existing command structure |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Staged Backend-First Stabilization** - Reduce request cost and recursion first, then split frontend orchestration files once latency and build health are stable.

**Why this approach:**
- It attacks the highest observed latency sources (`attention`, graph CLI fanout) before cosmetic refactors.
- It lowers blast radius by restoring backend build correctness and improving endpoint predictability.
- It creates cleaner boundaries that make subsequent UI decomposition straightforward and measurable.

**Trade-offs accepted:**
- Route/store decomposition and a11y polish are partially deferred while backend hotspots are reduced.
- This is acceptable because current user pain is load latency and build breakage, not missing features.

**Implementation sequence:**
1. Restore `go build` by resolving duplicate `complete_*` declarations (unblocks all validation and deployment workflows).
2. Optimize `/api/attention` (dedupe `/api/agents` requests, run independent collectors concurrently, add per-collector timing logs).
3. Optimize `/api/beads/graph` (avoid post-hoc parent filtering, reduce per-issue CLI calls, preserve cache semantics).
4. Split `work-graph/+page.svelte` orchestration into composable hooks/modules and reduce polling overlap with SSE events.
5. Triage and fix high-signal accessibility warnings in work-graph and modal surfaces.

### Alternative Approaches Considered

**Option B: Frontend-first optimization and component split**
- **Pros:** Improves maintainability and readability quickly; likely lowers some render-time overhead.
- **Cons:** Does not address serial backend collectors or CLI graph fanout that currently dominate cold load path.
- **When to use instead:** If profiling proves browser render dominates p95 latency.

**Option C: Add heavier caching only (without query-plan changes)**
- **Pros:** Fastest implementation path; can mask latency spikes in steady-state usage.
- **Cons:** Preserves expensive cold paths and recursive collector behavior; risks stale-data complexity.
- **When to use instead:** As temporary mitigation while refactor work is scheduled.

**Rationale for recommendation:** Backend-first stabilization directly addresses measured bottlenecks and system fragility, while still leaving room for UI decomposition as a second-stage quality improvement.

---

### Implementation Details

**What to implement first:**
- Remove duplicate function declarations across `cmd/orch/complete_*` to get CI/local build green.
- Add timing instrumentation to `/api/attention` and each collector to establish p50/p95 baseline.
- Replace duplicate `/api/agents` collector fetches with a shared in-request snapshot.
- Move parent filtering earlier in graph build and short-circuit unnecessary dependency lookups.

**Things to watch out for:**
- ⚠️ Collector parallelism can amplify `bd` pressure unless bounded and cache-aware.
- ⚠️ Changing graph semantics (focus/open/parent) can break existing tree expectations if not covered by tests.
- ⚠️ Decomposition of large Svelte files can accidentally alter keyboard navigation behavior.

**Areas needing further investigation:**
- Capture browser Performance trace for slow `/work-graph` sessions to separate paint/layout cost from network wait.
- Measure endpoint latencies under simulated many-agent many-workspace load.
- Evaluate whether a backend graph endpoint can consume a single bulk dependency dataset instead of N per-issue calls.

**Success criteria:**
- ✅ `go build ./...` succeeds.
- ✅ `/api/attention` and `/api/beads/graph` p95 latencies decrease measurably on cold requests.
- ✅ Work-graph first-contentful-data time drops and remains stable under polling/SSE activity.
- ✅ Accessibility warnings in touched work-graph UI surfaces are reduced with no keyboard regression.

---

## References

**Files Examined:**
- `cmd/orch/serve_beads_graph.go` - Graph endpoint query plan and CLI subprocess fanout.
- `cmd/orch/serve_beads_cache.go` - Graph cache TTL and invalidation behavior.
- `cmd/orch/serve_bd_limiter.go` - Global bd subprocess limiter and timeout model.
- `cmd/orch/serve_attention.go` - Attention collector composition and serial execution path.
- `pkg/attention/agent_collector.go` - Nested `/api/agents` collector call.
- `pkg/attention/stuck_collector.go` - Second nested `/api/agents` call.
- `pkg/verify/beads_api.go` - Comment batch strategy and 30s fallback timeout behavior.
- `web/src/routes/work-graph/+page.svelte` - Initial load, polling, and reactive rebuild workflow.
- `web/src/lib/stores/work-graph.ts` - Graph fetch behavior and tree construction.
- `web/src/routes/+page.svelte` - Dashboard-wide orchestration and multi-store fetch patterns.
- `web/src/lib/stores/agents.ts` - Agent fetch/debounce/SSE orchestration complexity.
- `cmd/orch/complete_export.go`, `cmd/orch/complete_pipeline.go`, `cmd/orch/complete_gates.go` - Duplicate declaration build failures.

**Commands Run:**
```bash
# Verify project location
pwd

# Create investigation artifact
kb create investigation dashboard-code-health-audit-full

# Locate work-graph references
rg -n "work-graph|work graph|workgraph" web/src pkg cmd

# Count store files
ls -1 web/src/lib/stores | wc -l

# Measure key file sizes
wc -l web/src/routes/+page.svelte web/src/routes/work-graph/+page.svelte web/src/lib/stores/work-graph.ts web/src/lib/stores/agents.ts web/src/lib/services/sse-connection.ts

# Check backend build health
go build ./...

# Benchmark graph/agent/attention endpoints (cold/warm)
python3 <benchmark_script>

# Build frontend and collect warnings
bun run build
```

**External Documentation:**
- None consulted; conclusions are from repository code and local endpoint measurements.

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md` - Existing reliability direction this audit stress-tests.
- **Investigation:** `.kb/investigations/2026-01-14-inv-orch-doctor-verify-dashboard-fetch.md` - Prior dashboard fetch reliability context.
- **Workspace:** `.orch/workspace/og-audit-dashboard-code-health-07feb-46c6/` - Spawn workspace for this audit.

---

## Investigation History

**[2026-02-07 11:22]:** Investigation started
- Initial question: Audit dashboard code health and explain observed `/work-graph` slowness.
- Context: Spawned full-stack audit covering performance, architecture, neglected areas, data flow, and build health.

**[2026-02-07 11:30]:** Core bottleneck path validated
- Measured cold API timings and confirmed attention endpoint dominates initial load in current environment.

**[2026-02-07 11:42]:** Data flow and architecture findings consolidated
- Verified graph endpoint subprocess fanout, post-hoc parent filtering, and monolith pressure in route/store files.

**[2026-02-07 11:50]:** Investigation completed
- Status: Complete
- Key outcome: Produced prioritized backend-first remediation plan with architecture score and concrete next actions.
