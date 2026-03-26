## Summary (D.E.K.N.)

**Delta:** The current web UI contradicts the thread-comprehension product decision: root route is a 32KB execution dashboard, threads have zero API endpoints, and the nav hierarchy puts operational views first.

**Evidence:** Audited all 6 Svelte routes, 50+ serve endpoints, pkg/thread data structures, and the orient CLI — threads are well-modeled in Go but completely absent from the HTTP API and web UI.

**Knowledge:** The minimum viable thread-first home surface requires 2 new API endpoints (thread list + thread detail), a new Svelte store, a redesigned root route with 3 comprehension sections above a condensed operational summary, and a nav reorder. No new backend data primitives needed — pkg/thread already has everything.

**Next:** Implement in 2 phases: Phase 1 (thread API + redesigned root route), Phase 2 (thread graph visualization + brief-in-thread rendering).

**Authority:** architectural — Cross-component (Go backend + Svelte frontend + nav hierarchy), multiple valid approaches, affects product surface identity.

---

# Investigation: Design Thread-First Home Surface Above Work Graph

**Question:** What should the primary dashboard/home surface look like when orch-go's identity is thread/comprehension first, and how do we get there without a rewrite?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-q6ykb
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` | extends | yes | - |
| `.kb/plans/2026-03-26-thread-comprehension-consolidation.md` | extends (Phase 3 deliverable) | yes | - |
| `.kb/investigations/2026-03-26-inv-inventory-current-system-into-core.md` | extends | yes | confirms UI mismatch |
| `.kb/threads/2026-03-24-threads-as-primary-artifact-thinking.md` | extends | yes | - |

---

## Findings

### Finding 1: Thread Data Is Rich in Go, Absent from API

**Evidence:** `pkg/thread/` (2,337 lines) has well-structured domain types:
- `Thread` struct with Title, Status (forming/active/converged/subsumed/resolved), Created, Updated, SpawnedFrom, Spawned, ActiveWork, ResolvedBy, Entries (dated sections)
- `ThreadSummary` for listing: Name, Title, Status, Created, Updated, LatestEntry (truncated), EntryCount
- Full lifecycle management: Create, Append, Resolve, LinkWork, CreateWithParent
- Relationship tracking: parent-child threads, thread-to-work (beads ID) linking

CLI commands (`thread_cmd.go`) expose all of this: `orch thread list`, `show`, `new`, `append`, `resolve`, `link`.

However, `serve.go` has zero thread endpoints. The web UI has no thread store. Threads are CLI-only.

**Source:** `pkg/thread/thread.go`, `pkg/thread/lifecycle.go`, `pkg/thread/relations.go`, `cmd/orch/thread_cmd.go`, `cmd/orch/serve.go` (50+ HandleFunc registrations, none for threads)

**Significance:** The hardest part — data modeling — is done. Exposing threads via API is straightforward HTTP handler wrapping of existing Go functions. This is the single highest-leverage v1 task.

---

### Finding 2: Orient Command Already Models the Ideal Home Surface

**Evidence:** `orch orient` produces exactly the session-start orientation the home surface needs:
1. Active threads with latest entry previews
2. Changelog (recent commits since last session)
3. Relevant models
4. Completion/abandonment stats
5. Metric divergence alerts
6. Last session insights
7. Ready-to-work queue

This is the CLI equivalent of "what am I thinking about, what changed, what remains open." The web home surface should render the same information architecture, not invent a different one.

**Source:** `orch orient` output (verified by running it)

**Significance:** The design question is not "what should the home surface show?" — orient already answered that. The question is "how should orient's information architecture translate to a dashboard layout?"

---

### Finding 3: Current Root Route Is Execution-Only (32KB)

**Evidence:** `web/src/routes/+page.svelte` imports 18 stores and 12 components, all execution-centric:
- AgentCard, AgentDetailPanel (agent lifecycle)
- ReviewQueueSection, ReadyQueueSection, UpNextSection (work queue management)
- RecentWins, NeedsAttention (execution outcomes)
- StatsBar, ServicesSection (operational health)
- QuestionsSection (blocking work, not thinking questions)

There is zero thread content, zero brief preview, zero knowledge-change notification on the root route.

**Source:** `web/src/routes/+page.svelte:1-84`, `web/src/routes/+layout.svelte:60-85`

**Significance:** A user's first impression is "execution monitoring tool." The product decision says it should be "comprehension layer." The root route directly contradicts the product identity.

---

### Finding 4: Nav Hierarchy Reinforces Execution Identity

**Evidence:** Current nav order: Dashboard | Work Graph | Knowledge Tree | Briefs | Harness

- Positions 1-2: execution-centric (Dashboard, Work Graph)
- Positions 3-4: comprehension-centric (Knowledge Tree, Briefs)
- Position 5: adjacent (Harness)

The `/thinking` route exists but is not in the nav at all — it renders digest products (thread progressions, model updates, probes, decisions) but the backend API (`/api/digest`) has no Go implementation. It appears to be a UI prototype without a working backend.

**Source:** `web/src/routes/+layout.svelte:60-85`, `web/src/routes/thinking/+page.svelte`, `web/src/lib/stores/digest.ts` (points to `/api/digest` which has no handler in serve.go)

**Significance:** The nav tells the story of what the product is. Right now it says "execution first, comprehension second." Reordering costs nothing and immediately signals the shift.

---

### Finding 5: Briefs and Knowledge Tree Are Ready as Subordinate Surfaces

**Evidence:** Both have full API coverage:
- Briefs: `GET/POST /api/briefs`, `GET/POST /api/briefs/{id}` — list, read, mark-as-read
- Knowledge Tree: `GET /api/tree` with node types (investigation, decision, model, probe, guide), SSE stream via `/api/events/tree`
- Review Queue: `GET /api/beads/review-queue` — issues awaiting review with tier, gates, has_brief

These don't need to become the home surface — they're already good as secondary routes. The home surface should show summaries of these (unread brief count, recent KB changes) with links to dive deeper.

**Source:** `cmd/orch/serve_briefs.go`, `cmd/orch/serve.go:413-437`

**Significance:** v1 doesn't need to rebuild briefs or knowledge tree. It needs to compose summaries from existing APIs into the home surface.

---

## Synthesis

**Key Insights:**

1. **Orient is the design spec** — The CLI orient command already answers "what should Dylan see first?" The home surface is orient rendered as a dashboard, not a new information architecture.

2. **Two new endpoints unlock everything** — Thread list and thread detail are the only missing API primitives. Everything else (briefs, review queue, knowledge tree, questions) already has HTTP endpoints. The home surface is a composition problem, not a data problem.

3. **The /thinking route was a false start** — It has the right idea (comprehension products) but no working backend. Rather than fix it, the design should fold its intent into the redesigned root route.

**Answer to Investigation Question:**

The thread-first home surface should mirror `orch orient`'s information architecture as a web dashboard:

**Above the fold (comprehension context):**
1. **Active Threads** — the thinking spine. Shows active+forming threads sorted by updated, with title, status, latest entry preview, entry count, linked work count.
2. **New Evidence** — what changed. Unread briefs count + top 3 previews, recent knowledge artifacts (from existing `/api/tree` data filtered to recent changes).
3. **Open Tensions** — what's unresolved. Blocking questions (from `/api/questions`), forming threads with no recent entries.

**Below the fold (operational context):**
4. **Work Summary** — condensed to counts + one-line per active agent. Links to full Work page.
5. **Comprehension Queue** — completions awaiting review (from existing `/api/beads/review-queue`).

**Nav reorder:** Threads | Briefs | Knowledge | Work | Harness

---

## Structured Uncertainty

**What's tested:**

- ✅ Thread data structures have full domain model (verified: read pkg/thread/*.go)
- ✅ Zero thread API endpoints exist (verified: searched all HandleFunc registrations in serve.go)
- ✅ Briefs and knowledge tree have full API coverage (verified: read serve_briefs.go, serve.go endpoints)
- ✅ Orient command produces the target information architecture (verified: ran `orch orient`)
- ✅ Digest/thinking backend does not exist (verified: grepped for /api/digest across entire codebase)

**What's untested:**

- ⚠️ Whether thread list API will be fast enough for polling (unknown thread count at scale)
- ⚠️ Whether condensed operational section will satisfy daily monitoring needs
- ⚠️ Whether the /thinking page's digest product concept should be preserved or abandoned
- ⚠️ Whether thread-to-brief linking displays well in the thread card UX

**What would change this:**

- If Dylan finds he still starts from execution state, the above/below fold split should be adjustable (toggle or localStorage preference)
- If thread count grows large (50+), the thread list needs filtering/search beyond status
- If the digest product API gets implemented, it could replace the "New Evidence" section with richer data

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Thread API endpoints | implementation | Wrapping existing Go code in HTTP handlers |
| Root route redesign | architectural | Cross-component (backend + frontend), changes product surface |
| Nav hierarchy reorder | architectural | Affects product identity and user mental model |
| Phase 2: thread graph visualization | strategic | Significant new frontend capability, resource commitment |

### Recommended Approach: Staged Root Route Redesign

**Phase 1: Thread-First Home (ships value immediately)**

Create the minimum new infrastructure needed to make the root route thread-first:

**Why this approach:**
- Reuses existing pkg/thread domain code (no new data primitives)
- Composes existing API data (briefs, review queue, questions) into the home surface
- Preserves all execution data on the same page, just repositioned below
- No route deletions, no bookmark breakage

**Trade-offs accepted:**
- Thread cards won't show full graph relationships in v1 (just parent name + child count)
- "New Evidence" section composites from multiple endpoints rather than a single optimized one
- The /thinking route becomes orphaned (can be removed or redirected later)

**Implementation sequence:**

1. **Thread API endpoints** (Go backend, ~200 lines)
   - `GET /api/threads` — returns `[]ThreadSummary` (status filter param, sorted by updated desc)
   - `GET /api/threads/{slug}` — returns full `Thread` with entries
   - Handler in new file `cmd/orch/serve_threads.go`
   - Register in `serve.go`

2. **Thread Svelte store** (frontend, ~60 lines)
   - `web/src/lib/stores/threads.ts`
   - ThreadSummary type, fetch with polling (30s), status filter

3. **Redesigned root route** (frontend, replace +page.svelte)
   - Top: ActiveThreads section (thread cards with expand/collapse)
   - Middle: NewEvidence section (briefs summary + KB recent)
   - Middle: OpenTensions section (questions + stalled threads)
   - Bottom: WorkSummary (condensed agent counts, link to /work)
   - Bottom: ComprehensionQueue (review-queue section, already exists)

4. **Nav reorder** (layout.svelte, ~10 line change)
   - Rename "Dashboard" → "Threads" (pointing to /)
   - Reorder: Threads | Briefs | Knowledge | Work | Harness
   - "Work" links to current work-graph route (or a merged operational view)

### Alternative Approaches Considered

**Option B: Promote /thinking to root**
- **Pros:** Already has the right intent (comprehension products)
- **Cons:** Backend doesn't exist (/api/digest has no Go handlers), would need both backend + frontend work, and the digest product model is different from thread-centric orientation
- **When to use instead:** If the digest product concept proves more valuable than thread-centric orientation after v1 ships

**Option C: New /threads route, redirect root**
- **Pros:** Clean separation, old dashboard still accessible at /dashboard
- **Cons:** Two routes showing overlapping data, redirect feels indecisive, old URL ("/") stops being canonical
- **When to use instead:** If the execution dashboard has other users who would be disrupted

**Rationale for recommendation:** The root route IS the product identity. Making it thread-first is the clearest expression of the decision. And the staged approach (add sections above, condense below) is safer than a parallel route.

---

### Implementation Details

**What to implement first:**
- Thread API endpoints (Finding 1 — everything else depends on thread data being available via HTTP)
- Thread Svelte store (needed before UI work)
- Nav reorder (Finding 4 — zero risk, immediate signal)

**Things to watch out for:**
- ⚠️ Thread files are read from `.kb/threads/` — ensure serve_threads.go uses the same directory resolution as pkg/thread (not hardcoded paths)
- ⚠️ Root route currently uses 18 stores; condensing execution sections means some stores may no longer be needed on the root route — don't remove them, they're still used by /work-graph
- ⚠️ The briefs store already polls; adding thread polling adds another 30s interval. Consider a composite `/api/home` endpoint that returns threads + briefs summary + questions in one call to reduce HTTP overhead
- ⚠️ Defect class exposure: Class 1 (Filter Amnesia) — if thread status filtering is implemented in the API but a different filter set is used in the UI, threads could be invisible. Use the same status enum in both layers.

**Areas needing further investigation:**
- Whether a composite `/api/home` endpoint is worth building vs. parallel fetches from existing endpoints
- How thread entries should render in the card (full markdown? plain text preview? first N words?)
- Whether stale forming threads (no entry in 3+ days) is a useful "tension" signal or just noise

**Success criteria:**
- ✅ Opening localhost:5188 shows active threads above operational data
- ✅ Nav reads "Threads | Briefs | Knowledge | Work | Harness"
- ✅ Thread cards show title, status, latest entry preview, entry count
- ✅ Clicking a thread card expands to show recent entries
- ✅ Unread briefs count is visible on the home surface
- ✅ Active agent count is visible but subordinate (below fold)
- ✅ All existing execution data is still accessible (via Work route)

---

### Phase 2: Follow-On Work (Not v1)

| Feature | Description | Prerequisite |
|---------|-------------|--------------|
| Thread graph visualization | Threads as nodes, spawned_from/spawned as edges, work hanging off threads | v1 thread API |
| Brief-in-thread rendering | Briefs display within thread context, not flat list | Thread-to-brief linking data |
| Thread CRUD in web UI | Create, append, resolve threads from dashboard | v1 thread API |
| Digest product API | Implement `/api/digest` backend, replace /thinking with integrated home section | Design decision on digest vs thread-first |
| Composite home endpoint | Single `/api/home` returning all home surface data | v1 performance measurement |
| Thread search/filter | Filter by status, keyword, date range | Thread count growth |

---

## References

**Files Examined:**
- `pkg/thread/thread.go` — Thread domain types and CRUD
- `pkg/thread/lifecycle.go` — Thread status states
- `pkg/thread/relations.go` — Thread-to-thread and thread-to-work linking
- `cmd/orch/thread_cmd.go` — CLI thread commands
- `cmd/orch/serve.go` — All HTTP endpoint registrations (50+ handlers)
- `cmd/orch/serve_briefs.go` — Briefs API implementation
- `web/src/routes/+page.svelte` — Current root route (execution dashboard)
- `web/src/routes/+layout.svelte` — Nav hierarchy
- `web/src/routes/briefs/+page.svelte` — Briefs reading surface
- `web/src/routes/thinking/+page.svelte` — Digest product surface (broken backend)
- `web/src/routes/knowledge-tree/+page.svelte` — Knowledge tree UI
- `web/src/lib/stores/digest.ts` — Digest store (points to non-existent API)

**Commands Run:**
```bash
# Current thread list output
go run ./cmd/orch thread list

# Current orient command (session start orientation)
go run ./cmd/orch orient
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — Product boundary this implements
- **Plan:** `.kb/plans/2026-03-26-thread-comprehension-consolidation.md` — This is Phase 3 deliverable
- **Investigation:** `.kb/investigations/2026-03-26-inv-inventory-current-system-into-core.md` — System inventory confirming UI mismatch

---

## Investigation History

**2026-03-26:** Investigation started
- Initial question: What should the primary home surface look like when thread/comprehension is the product identity?
- Context: Product boundary decision accepted, consolidation plan Phase 3 calls for thread-first UI

**2026-03-26:** Exploration complete
- 4 forks identified: root route content, thread data source, nav hierarchy, migration path
- Key finding: pkg/thread has full domain model but zero API endpoints
- Key finding: orient command already models the ideal home surface
- Key finding: /thinking route is a broken prototype (no backend)

**2026-03-26:** Investigation completed
- Status: Complete
- Key outcome: Staged root route redesign — 2 new API endpoints, redesigned root route with thread sections above condensed operational view, nav reorder. Phase 1 ships value without new data primitives.
