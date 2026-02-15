<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Synthesized 9 serve investigations into `.kb/guides/background-services-performance.md` - a comprehensive guide covering CPU anti-patterns, caching strategies, service architecture, and debugging checklists.

**Evidence:** Read and analyzed all 9 serve investigations from Dec 2025 - Jan 2026; identified 5 major pattern clusters (CPU performance, caching, process management, status determination, port architecture); created 300+ line guide with concrete code patterns and decision history.

**Knowledge:** Background services face unique performance challenges: O(n*m) complexity compounds at scale, SSE+polling feedback loops burn CPU, and cache invalidation must be event-driven not just TTL-based. The three-tier port architecture (4096/3348/5188) is often misunderstood.

**Next:** Close - guide created and committed. Future serve investigations should reference and update this guide.

**Promote to Decision:** recommend-no - This is a synthesis artifact (guide), not an architectural decision.

---

# Investigation: Synthesize Serve Investigation Cluster

**Question:** What patterns emerge from the serve investigation cluster, and how can they be externalized into a reusable guide for background service performance?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Three CPU Performance Anti-Patterns Recur

**Evidence:** Three distinct anti-patterns appeared across multiple investigations:

1. **SSE + Polling Feedback Loop (Dec 25, 2025):** SSE events trigger API refetches that fetch the same state SSE provides. Result: 125% CPU with 3 agents.

2. **O(n*m) File Operations (Dec 25, 2025):** Per-agent operations scanning all workspaces creates multiplicative complexity. With 10 agents and 466 workspaces: 4,660 file operations per request.

3. **Unbounded Process Spawning (Jan 3, 2026):** Spawning bd processes for all 618 workspaces caused 90+ second response times and 20+ concurrent processes.

**Source:**
- `2025-12-25-inv-orch-serve-hit-125-cpu.md`
- `2025-12-25-inv-orch-serve-cpu-runaway-recurring.md`
- `2026-01-03-inv-orch-serve-causes-cpu-spike.md`

**Significance:** These patterns are generalizable - any long-running service with similar characteristics will face them. The guide now documents solutions for each.

---

### Finding 2: Caching Requires Both TTL and Event-Driven Invalidation

**Evidence:** Jan 4, 2026 investigation showed dashboard displaying stale "active" status 30 seconds after orch complete ran. Root cause: TTL cache only, no invalidation mechanism when external CLI updated state.

**Solution implemented:** Explicit invalidation API (`POST /api/cache/invalidate`) called from CLI after state changes, with silent failure (TTL handles eventual consistency).

**Source:** `2026-01-04-inv-orch-serve-cache-not-invalidated.md`

**Significance:** This is a general principle: TTL is for reducing load, event-driven invalidation is for freshness. Both are needed.

---

### Finding 3: Three-Tier Port Architecture Creates Confusion

**Evidence:** Jan 3, 2026 investigation traced "port confusion" to conflating three distinct services:
- OpenCode (4096) - Claude sessions
- orch serve (3348) - API aggregator
- Vite dev (5188) - Frontend

**Source:** `2026-01-03-inv-dashboard-port-confusion-orch-serve.md`

**Significance:** This architecture is correct but undocumented. The guide now explicitly documents the data flow: Browser → Vite (5188) → orch serve (3348) → OpenCode (4096).

---

### Finding 4: launchd Environment Requires Startup Path Resolution

**Evidence:** Jan 7, 2026 investigation found `bd` executable not found when orch serve runs under launchd. Root cause: launchd provides minimal PATH (`/usr/bin:/bin:/usr/sbin:/sbin`).

**Solution:** Resolve absolute paths at startup by searching common locations, store in module-level variable.

**Source:** `2026-01-07-inv-orch-serve-path-issue-server.md`

**Significance:** This is a pattern for any Go service that shells out to user-installed tools - resolve at startup, not at call time.

---

### Finding 5: Agent Status Requires Priority Cascade Model

**Evidence:** Jan 4, 2026 investigation found closed agents showing as "active" because status determination missed beads issue status check.

**Solution:** Check signals in priority order: beads issue closed > Phase: Complete comment > SYNTHESIS.md exists > session activity.

**Source:** `2026-01-04-inv-orch-serve-shows-closed-agents.md`

**Significance:** Multiple completion signals exist; the guide documents the authoritative ordering (beads is source of truth).

---

## Synthesis

**Key Insights:**

1. **Complexity compounds at scale** - O(1) operations that seem fast become O(n*m) disasters with 500+ workspaces. Always ask: "What happens at 10x scale?"

2. **Caching is not one thing** - TTL caching reduces load, event-driven invalidation ensures freshness. Both mechanisms serve different purposes and are both required.

3. **Architecture documentation prevents investigation loops** - Port confusion investigation (Jan 3) could have been prevented by documenting the three-tier architecture.

**Answer to Investigation Question:**

Nine investigations revealed five major pattern clusters for background service performance:

| Cluster | Key Pattern | Guide Section |
|---------|-------------|---------------|
| CPU Performance | SSE+polling, O(n*m), process spawning | CPU Performance Anti-Patterns |
| Caching | TTL + event-driven invalidation | Caching Patterns |
| Service Architecture | Three-tier ports, launchd PATH | Service Architecture |
| Status Determination | Priority cascade model | Agent Status Determination |
| Code Organization | Extract by domain at 1000+ lines | Code Organization |

All patterns have been externalized into `.kb/guides/background-services-performance.md` with concrete code patterns, decision history, and debugging checklists.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 9 serve investigations read and analyzed
- ✅ Guide created with sections covering all pattern clusters
- ✅ Code patterns verified against original investigation evidence
- ✅ Decision history populated from investigation dates

**What's untested:**

- ⚠️ Guide hasn't been used to prevent a new investigation yet (requires future validation)
- ⚠️ Some code patterns are excerpts, not full implementations

**What would change this:**

- If a new serve investigation discovers a pattern not in this guide → update guide
- If guide sections prove unhelpful → refine structure

---

## Implementation Recommendations

### Recommended Approach ⭐

**Guide-based knowledge externalization** - Create comprehensive guide rather than individual kb quick entries.

**Why this approach:**
- Nine investigations is too many for individual constraints
- Patterns are interconnected (caching affects CPU, architecture affects debugging)
- Guide format supports code examples and decision history

**Trade-offs accepted:**
- Guide is long (~300 lines) but comprehensive
- May become stale if not updated alongside new investigations

**Implementation sequence:**
1. ✅ Analyze all investigations
2. ✅ Identify pattern clusters
3. ✅ Create guide with sections per cluster
4. ✅ Include code patterns and debugging checklist

### Alternative Approaches Considered

**Option B: Individual kb quick entries**
- **Pros:** More discoverable via kb context
- **Cons:** Loses interconnections between patterns; 9+ entries to manage
- **When to use:** Single isolated learnings

**Option C: Update existing daemon.md**
- **Pros:** Consolidates related content
- **Cons:** daemon.md is already 500+ lines; serve concerns are distinct
- **When to use:** If daemon and serve converge architecturally

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-25-inv-orch-serve-hit-125-cpu.md`
- `.kb/investigations/2025-12-25-inv-orch-serve-cpu-runaway-recurring.md`
- `.kb/investigations/2025-12-25-inv-separate-orch-serve-status-orch.md`
- `.kb/investigations/2026-01-03-inv-orch-serve-causes-cpu-spike.md`
- `.kb/investigations/2026-01-03-inv-dashboard-port-confusion-orch-serve.md`
- `.kb/investigations/2026-01-04-inv-orch-serve-cache-not-invalidated.md`
- `.kb/investigations/2026-01-04-inv-orch-serve-shows-closed-agents.md`
- `.kb/investigations/2026-01-04-inv-analyze-serve-agents-go-1399.md`
- `.kb/investigations/2026-01-07-inv-orch-serve-path-issue-server.md`

**Files Created:**
- `.kb/guides/background-services-performance.md` - Synthesized guide

**Related Artifacts:**
- **Guide:** `.kb/guides/daemon.md` - Complementary guide for daemon-specific patterns
- **Guide:** `.kb/guides/resilient-infrastructure-patterns.md` - Crash recovery patterns

---

## Investigation History

**2026-01-17 [start]:** Investigation started
- Initial question: Synthesize serve cluster into performance guide
- Context: kb reflect identified cluster of 8+ serve investigations needing consolidation

**2026-01-17 [analysis]:** Analyzed all 9 investigations
- Identified 5 pattern clusters
- Found recurring CPU anti-patterns across 3 investigations
- Mapped caching, architecture, and status determination patterns

**2026-01-17 [complete]:** Investigation completed
- Status: Complete
- Key outcome: Created comprehensive guide at `.kb/guides/background-services-performance.md`
